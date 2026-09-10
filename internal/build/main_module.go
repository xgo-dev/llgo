//go:build !llgo
// +build !llgo

/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package build contains the llgo compiler build orchestration logic.
//
// The main_module.go file generates the entry point module for llgo programs,
// which contains the main() function, initialization sequence, and global
// variables like argc/argv. This module is generated differently depending on
// BuildMode (exe, c-archive, c-shared).

package build

import (
	"go/token"
	"go/types"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	llvm "github.com/xgo-dev/llvm"
)

type genConfig struct {
	rtInit        bool
	pyInit        bool
	abiInit       int
	packageInits  []string
	methodByIndex map[int]none
	methodByName  map[string]none
	abiSymbols    map[string]none
	abiTypes      []llssa.AbiTypeInfo
	funcInfo      []funcInfoRecord
	pcLineInfo    []pcLineRecord
	cExports      []cExport
}

type cExport struct {
	goName string
	cName  string
	sig    *types.Signature
}

const (
	processEntrySymbol = "main"
	runtimeMainSymbol  = "runtime.main"
	runtimeGoexitName  = "runtime.goexit"
)

func needsRuntimeMainFrame(ctx *context) bool {
	conf := ctx.buildConf
	return conf.BuildMode == BuildModeExe && !isWasmTarget(conf.Goos) && conf.PCLNMode != PCLNNone
}

func needsWasmRuntimeScheduler(ctx *context) bool {
	if ctx.buildConf.BuildMode != BuildModeExe {
		return false
	}
	if ctx.crossCompile.WasmPostLink.Asyncify {
		return true
	}
	return ctx.crossCompile.WasmProvider == crosscompile.WasmProviderEmscripten
}

// genMainModule generates the main entry module for an llgo program.
//
// The module contains argc/argv globals and, for executable build modes,
// the entry function that wires initialization and main. C archive and shared
// library modes also arrange runtime initialization before exported Go code.
func genMainModule(ctx *context, rtPkgPath string, pkg *packages.Package, cfg *genConfig) Package {
	prog := ctx.prog
	mainPkg := prog.NewPackage("", pkg.ID+".main")

	argcVar := mainPkg.NewVarEx("__llgo_argc", prog.Pointer(prog.Int32()))
	argcVar.Init(prog.Zero(prog.Int32()))

	argvValueType := prog.Pointer(prog.CStr())
	argvVar := mainPkg.NewVarEx("__llgo_argv", prog.Pointer(argvValueType))
	argvVar.InitNil()
	funcInfo := cfg.funcInfo
	if needsRuntimeMainFrame(ctx) {
		// The native process entry is the physical frame below runtime.main,
		// which is runtime.goexit in a Go traceback. Keep the required C symbol
		// while giving both generated frames their logical Go names.
		tailRecords := []funcInfoRecord{
			{symbol: runtimeMainSymbol, name: runtimeMainSymbol},
			{symbol: processEntrySymbol, name: runtimeGoexitName},
		}
		funcInfo = append(funcInfo, tailRecords...)
		// Entry sites are emitted after these generated functions have bodies.
		// Keep matching metadata in this module so the generic site emitter can
		// attach the symbol IDs without adding a special PCLN path.
		for _, rec := range tailRecords {
			mainPkg.EmitFuncInfo(rec.symbol, rec.name, "", 0, 0)
		}
	}
	emitFuncInfoTable(ctx, mainPkg, funcInfo, cfg.pcLineInfo)

	exportFile := pkg.ExportFile
	if exportFile == "" {
		exportFile = pkg.PkgPath
	}
	mainAPkg := &aPackage{
		Package: &packages.Package{
			PkgPath:    pkg.PkgPath + ".main",
			ExportFile: exportFile + "-main",
		},
		LPkg: mainPkg,
	}

	runtimeStub := defineWeakNoArgStub(mainPkg, "runtime.init")
	// TODO(lijie): workaround for syscall patch
	defineWeakNoArgStub(mainPkg, "syscall.init")

	var pyInit llssa.Function
	var pyFinalize llssa.Function
	if cfg.pyInit {
		pyInit = declareNoArgFunc(mainPkg, "Py_Initialize")
		pyFinalize = declareNoArgFunc(mainPkg, "Py_Finalize")
	}

	wasmRuntimeScheduler := needsWasmRuntimeScheduler(ctx)
	var rtInit llssa.Function
	if cfg.rtInit || wasmRuntimeScheduler {
		rtInit = declareNoArgFunc(mainPkg, rtPkgPath+".init")
	}
	var processExit llssa.Function
	if ctx.buildConf.Goos == "windows" {
		processExit = declareRuntimeExit(mainPkg, "runtime.exit")
	}

	var abiInit llssa.Function
	if cfg.abiInit != 0 {
		mainPkg.RegisterAbiTypes(cfg.abiTypes)
		abiInit = mainPkg.InitAbiTypesFor("init$abitypes", func(sym *llssa.AbiSymbol) bool {
			if _, ok := cfg.abiSymbols[sym.Name]; !ok {
				return false
			}
			return filterAbiSymbol(cfg.abiInit, sym)
		})
	}

	var pkgPath string
	if pkg.Types != nil && pkg.Types.Name() == "main" {
		pkgPath = "main"
	} else {
		pkgPath = pkg.PkgPath
	}

	mainInit := declareNoArgFunc(mainPkg, pkgPath+".init")
	mainMain := declareNoArgFunc(mainPkg, pkgPath+".main")
	packageInits := make([]llssa.Function, len(cfg.packageInits))
	for i, name := range cfg.packageInits {
		packageInits[i] = declareNoArgFunc(mainPkg, name)
	}
	defineRootInitTask(mainPkg, pkgPath)

	if ctx.buildConf.BuildMode != BuildModeExe {
		initArraySection := ""
		if ctx.buildConf.BuildMode == BuildModeCShared && ctx.buildConf.Goos == "linux" {
			initArraySection = ".init_array"
		}
		inits := []llssa.Function{pyInit, rtInit, abiInit, runtimeStub}
		// Windows already enables this during runtime initialization because
		// callbacks in ordinary executables also enter from foreign threads.
		// Other platforms enable it only for libraries with C exports.
		if len(cfg.cExports) != 0 && ctx.buildConf.Goos != "windows" {
			inits = append(inits, declareNoArgFunc(
				mainPkg, llssa.PkgRuntime+".EnableForeignThreadRegistration",
			))
		}
		// The C test runner supplies argc/argv before calling the generated
		// test main package's init and main functions.
		if ctx.mode != ModeTest {
			inits = append(inits, packageInits...)
			inits = append(inits, mainInit)
		}
		var ensureInit llssa.Function
		if ctx.buildConf.BuildMode == BuildModeCShared && ctx.buildConf.Goos == "windows" {
			ensureInit = defineWindowsSharedRuntimeInit(mainPkg, ctx.buildConf.Goarch, inits...)
		} else {
			defineLibraryRuntimeInit(
				mainPkg, initArraySection, argcVar, argvVar, argvValueType,
				libraryConstructorReceivesProcessArgs(ctx.buildConf.Goos), inits...,
			)
		}
		defineCExportWrappers(mainPkg, cfg.cExports, ensureInit)
		return mainAPkg
	}

	var wasmRunMain llssa.Function
	if wasmRuntimeScheduler {
		defineWasmMainTask(mainPkg, packageInits, mainInit, mainMain)
		wasmRunMain = declareNoArgFunc(mainPkg, rtPkgPath+".RunWasmMain")
	}
	entryFn := defineEntryFunction(ctx, mainPkg, argcVar, argvVar, argvValueType, entryFunctions{
		runtimeStub:  runtimeStub,
		mainInit:     mainInit,
		mainMain:     mainMain,
		wasmRunMain:  wasmRunMain,
		pyInit:       pyInit,
		pyFinalize:   pyFinalize,
		rtInit:       rtInit,
		processExit:  processExit,
		abiInit:      abiInit,
		packageInits: packageInits,
	})

	if needStart(ctx) {
		defineStart(mainPkg, entryFn, argvValueType)
	}
	emitFuncInfoEntrySites(ctx, mainPkg)

	return mainAPkg
}

func libraryConstructorReceivesProcessArgs(goos string) bool {
	return goos == "linux" || goos == "darwin"
}

// defineLibraryRuntimeInit arranges for the LLGo runtime to be initialized
// before a C program calls an exported Go function. Linux and Darwin c-shared
// and c-archive constructors receive the host process argc/argv; these values
// must be stored before package initialization so os.Args sees them. Linux
// c-shared libraries use .init_array explicitly because the raw LLVM target
// machine lowers llvm.global_ctors to legacy .ctors. Linux c-archive and Darwin
// library modes use LLVM's platform-specific global constructor lowering.
func defineLibraryRuntimeInit(
	pkg llssa.Package, initArraySection string,
	argcVar, argvVar llssa.Global, argvType llssa.Type,
	receivesProcessArgs bool, inits ...llssa.Function,
) {
	const ctorName = "__llgo_runtime_ctor"
	ctorSig := llssa.NoArgsNoRet
	if receivesProcessArgs {
		ctorSig = newSignature(
			[]types.Type{types.Typ[types.Int32], argvType.RawType()},
			nil,
		)
	}
	ctor := pkg.NewFunc(ctorName, ctorSig, llssa.InC)
	ctorValue := pkg.Module().NamedFunction(ctorName)
	ctorValue.SetLinkage(llvm.InternalLinkage)
	b := ctor.MakeBody(1)
	if receivesProcessArgs {
		b.Store(argcVar.Expr, ctor.Param(0))
		b.Store(argvVar.Expr, ctor.Param(1))
	}
	for _, init := range inits {
		if init != nil {
			b.Call(init.Expr)
		}
	}
	b.Return()

	mod := pkg.Module()
	if initArraySection != "" {
		initEntry := llvm.AddGlobal(mod, ctorValue.Type(), "__llgo_runtime_ctor_init")
		initEntry.SetInitializer(ctorValue)
		initEntry.SetGlobalConstant(true)
		initEntry.SetSection(initArraySection)
		initEntry.SetVisibility(llvm.HiddenVisibility)
		return
	}
	llvmCtx := mod.Context()
	priority := llvm.ConstInt(llvmCtx.Int32Type(), 65535, false)
	entry := llvmCtx.ConstStruct([]llvm.Value{
		priority,
		ctorValue,
		llvm.ConstNull(ctorValue.Type()),
	}, false)
	init := llvm.ConstArray(entry.Type(), []llvm.Value{entry})
	ctors := llvm.AddGlobal(mod, init.Type(), "llvm.global_ctors")
	ctors.SetInitializer(init)
	ctors.SetLinkage(llvm.AppendingLinkage)
}

// defineWindowsSharedRuntimeInit defers hosted runtime and package
// initialization until a C export is called. Running it from a PE global
// constructor would hold the Windows loader lock while Go loads DLLs or starts
// threads. The generated public wrappers live in this uncached link module and
// call the returned InitOnce-backed guard before entering Go.
func defineWindowsSharedRuntimeInit(pkg llssa.Package, goarch string, inits ...llssa.Function) llssa.Function {
	const (
		initializeName = "__llgo_runtime_initialize"
		ensureName     = "__llgo_runtime_ensure_initialized"
		callbackName   = "__llgo_runtime_init_once_callback"
	)

	initialize := pkg.NewFunc(initializeName, llssa.NoArgsNoRet, llssa.InC)
	initializeValue := pkg.Module().NamedFunction(initializeName)
	initializeValue.SetLinkage(llvm.InternalLinkage)
	b := initialize.MakeBody(1)
	for _, init := range inits {
		if init != nil {
			b.Call(init.Expr)
		}
	}
	b.Return()

	mod := pkg.Module()
	llvmCtx := mod.Context()
	voidType := llvmCtx.VoidType()
	ptrType := llvm.PointerType(voidType, 0)
	i32Type := llvmCtx.Int32Type()
	once := llvm.AddGlobal(mod, ptrType, "__llgo_runtime_init_once")
	once.SetInitializer(llvm.ConstNull(ptrType))
	once.SetLinkage(llvm.InternalLinkage)

	callbackType := llvm.FunctionType(i32Type, []llvm.Type{ptrType, ptrType, ptrType}, false)
	callback := llvm.AddFunction(mod, callbackName, callbackType)
	callback.SetLinkage(llvm.InternalLinkage)
	initOnceType := llvm.FunctionType(i32Type, []llvm.Type{ptrType, ptrType, ptrType, ptrType}, false)
	initOnce := llvm.AddFunction(mod, "InitOnceExecuteOnce", initOnceType)
	initOnce.SetDLLStorageClass(llvm.DLLImportStorageClass)
	if goarch == "386" {
		callback.SetFunctionCallConv(llvm.X86StdcallCallConv)
		initOnce.SetFunctionCallConv(llvm.X86StdcallCallConv)
	}

	builder := llvmCtx.NewBuilder()
	defer builder.Dispose()
	callbackBlock := llvmCtx.AddBasicBlock(callback, "entry")
	builder.SetInsertPointAtEnd(callbackBlock)
	builder.CreateCall(llvm.FunctionType(voidType, nil, false), initializeValue, nil, "")
	builder.CreateRet(llvm.ConstInt(i32Type, 1, false))

	ensure := pkg.NewFunc(ensureName, llssa.NoArgsNoRet, llssa.InC)
	ensureValue := mod.NamedFunction(ensureName)
	ensureValue.SetVisibility(llvm.HiddenVisibility)
	block := llvmCtx.AddBasicBlock(ensureValue, "entry")
	builder.SetInsertPointAtEnd(block)
	call := builder.CreateCall(initOnceType, initOnce, []llvm.Value{
		once, callback, llvm.ConstNull(ptrType), llvm.ConstNull(ptrType),
	}, "")
	if goarch == "386" {
		call.SetInstructionCallConv(llvm.X86StdcallCallConv)
	}
	builder.CreateRetVoid()
	return ensure
}

func defineCExportWrappers(pkg llssa.Package, exports []cExport, ensureInit llssa.Function) {
	if len(exports) == 0 {
		return
	}
	enterForeignThread := pkg.NewFunc(
		llssa.PkgRuntime+".EnterForeignThread",
		newSignature(nil, []types.Type{types.Typ[types.Bool]}), llssa.InGo,
	)
	exitForeignThread := pkg.NewFunc(
		llssa.PkgRuntime+".ExitForeignThread",
		newSignature([]types.Type{types.Typ[types.Bool]}, nil), llssa.InGo,
	)
	for _, export := range exports {
		implementation := pkg.NewFunc(export.goName, export.sig, llssa.InGo)
		wrapper := pkg.NewFunc(export.cName, export.sig, llssa.InGo)
		b := wrapper.MakeBody(1)
		if ensureInit != nil {
			b.Call(ensureInit.Expr)
		}
		registered := b.Call(enterForeignThread.Expr)
		args := make([]llssa.Expr, export.sig.Params().Len())
		for i := range args {
			args[i] = wrapper.Param(i)
		}
		result := b.Call(implementation.Expr, args...)
		b.Call(exitForeignThread.Expr, registered)
		if export.sig.Results().Len() == 0 {
			b.Return()
		} else {
			b.Return(result)
		}
	}
}

func filterAbiSymbol(abiInit int, sym *llssa.AbiSymbol) bool {
	switch sym.Raw.(type) {
	case *types.Array:
		if abiInit&llssa.ReflectArrayOf != 0 {
			return true
		}
	case *types.Chan:
		if abiInit&llssa.ReflectChanOf != 0 {
			return true
		}
	case *types.Signature:
		if abiInit&llssa.ReflectFuncOf != 0 {
			return true
		}
		if abiInit&llssa.ReflectMethodMask != 0 {
			return true
		}
	case *types.Map:
		if abiInit&llssa.ReflectMapOf != 0 {
			return true
		}
	case *types.Pointer:
		if abiInit&llssa.ReflectPointerTo != 0 {
			return true
		}
	case *types.Slice:
		if abiInit&llssa.ReflectSliceOf != 0 {
			return true
		}
	case *types.Struct:
		if abiInit&llssa.ReflectStructOf != 0 {
			return true
		}
	}
	return false
}

type entryFunctions struct {
	runtimeStub  llssa.Function
	mainInit     llssa.Function
	mainMain     llssa.Function
	wasmRunMain  llssa.Function
	pyInit       llssa.Function
	pyFinalize   llssa.Function
	rtInit       llssa.Function
	processExit  llssa.Function
	abiInit      llssa.Function
	packageInits []llssa.Function
}

// defineEntryFunction creates the program's entry function. The name is
// "main" for standard targets, or "__main_argc_argv" with hidden visibility
// for WASM targets that don't require _start.
//
// The entry stores argc/argv, optionally disables stdio buffering, and manages
// the local context. Native PCLN builds run the common startup sequence through
// runtime.main; other builds run it inline. That sequence initializes Python,
// the runtime stub/package, ABI types, and packages, calls main.main, and then
// finalizes Python. The entry leaves the local context and returns 0.
func defineEntryFunction(ctx *context, pkg llssa.Package, argcVar, argvVar llssa.Global, argvType llssa.Type, fns entryFunctions) llssa.Function {
	prog := pkg.Prog
	entryName := processEntrySymbol
	if !needStart(ctx) && isWasmTarget(ctx.buildConf.Goos) {
		entryName = "__main_argc_argv"
	}
	sig := newEntrySignature(argvType.RawType())
	fn := pkg.NewFunc(entryName, sig, llssa.InC)
	fnVal := pkg.Module().NamedFunction(entryName)
	if entryName != processEntrySymbol {
		fnVal.SetVisibility(llvm.HiddenVisibility)
		fnVal.SetUnnamedAddr(true)
	}
	b := fn.MakeBody(1)
	var localCtx, previousLocalCtx llssa.Expr
	// The single-worker runtime itself uses owner-local state even when the
	// user program has no TLS/GLS declarations. Root that state on the host
	// entry stack before runtime.init and keep it installed while logical Go
	// stacks are dispatched by RunWasmMain.
	hasLocalContext := prog.NeedsLocalContext() || fns.wasmRunMain != nil
	if hasLocalContext {
		localCtx, previousLocalCtx = b.EnterLocalContext()
	}
	b.Store(argcVar.Expr, fn.Param(0))
	b.Store(argvVar.Expr, fn.Param(1))
	if IsStdioNobuf() {
		emitStdioNobuf(b, pkg, ctx.buildConf.Goos)
	}
	if needsRuntimeMainFrame(ctx) {
		runtimeMain := defineRuntimeMainFunction(pkg, fns)
		b.Call(runtimeMain.Expr)
	} else {
		emitRuntimeMainBody(b, fns)
	}
	if hasLocalContext {
		b.LeaveLocalContext(localCtx, previousLocalCtx)
	}
	b.Return(prog.IntVal(0, prog.Int32()))
	return fn
}

// defineRuntimeMainFunction gives native executables the same logical tail as
// gc: user code is called by runtime.main, while the C entry below it is named
// runtime.goexit in funcinfo metadata. The wrapper is a startup-only call.
func defineRuntimeMainFunction(pkg llssa.Package, fns entryFunctions) llssa.Function {
	fn := pkg.NewFunc(runtimeMainSymbol, llssa.NoArgsNoRet, llssa.InGo)
	fn.Inline(llssa.NoInline)
	fn.DisableTailCalls()
	b := fn.MakeBody(1)
	emitRuntimeMainBody(b, fns)
	b.Return()
	return fn
}

func emitRuntimeMainBody(b llssa.Builder, fns entryFunctions) {
	if fns.pyInit != nil {
		b.Call(fns.pyInit.Expr)
	}
	if fns.rtInit != nil {
		b.Call(fns.rtInit.Expr)
	}
	if fns.abiInit != nil {
		b.Call(fns.abiInit.Expr)
	}
	b.Call(fns.runtimeStub.Expr)
	if fns.wasmRunMain != nil {
		b.Call(fns.wasmRunMain.Expr)
	} else {
		for _, init := range fns.packageInits {
			b.Call(init.Expr)
		}
		b.Call(fns.mainInit.Expr)
		b.Call(fns.mainMain.Expr)
	}
	if fns.pyFinalize != nil {
		b.Call(fns.pyFinalize.Expr)
	}
	if fns.processExit != nil {
		// Go terminates the process as soon as main returns, regardless of
		// other goroutines. On Windows, call runtime.exit (ExitProcess) rather
		// than returning through CRT teardown, which may wait on runtime-owned
		// threads and violates that guarantee.
		b.Call(fns.processExit.Expr, b.Prog.IntVal(0, b.Prog.Int32()))
	}
}

func defineWasmMainTask(pkg llssa.Package, packageInits []llssa.Function, mainInit, mainMain llssa.Function) {
	prog := pkg.Prog
	sig := newSignature(
		[]types.Type{types.Typ[types.UnsafePointer]},
		[]types.Type{types.Typ[types.UnsafePointer]},
	)
	fn := pkg.NewFunc("__llgo_wasm_main", sig, llssa.InC)
	fnVal := pkg.Module().NamedFunction("__llgo_wasm_main")
	fnVal.SetVisibility(llvm.HiddenVisibility)
	b := fn.MakeBody(1)
	for _, init := range packageInits {
		b.Call(init.Expr)
	}
	b.Call(mainInit.Expr)
	b.Call(mainMain.Expr)
	b.Return(prog.Nil(prog.VoidPtr()))
}

func defineStart(pkg llssa.Package, entry llssa.Function, argvType llssa.Type) {
	fn := pkg.NewFunc("_start", llssa.NoArgsNoRet, llssa.InC)
	pkg.Module().NamedFunction("_start").SetLinkage(llvm.WeakAnyLinkage)
	b := fn.MakeBody(1)
	prog := pkg.Prog
	b.Call(entry.Expr, prog.IntVal(0, prog.Int32()), prog.Nil(argvType))
	b.Return()
}

func declareNoArgFunc(pkg llssa.Package, name string) llssa.Function {
	return pkg.NewFunc(name, llssa.NoArgsNoRet, llssa.InC)
}

func declareRuntimeExit(pkg llssa.Package, name string) llssa.Function {
	sig := newSignature([]types.Type{types.Typ[types.Int32]}, nil)
	return pkg.NewFunc(name, sig, llssa.InGo)
}

// defineRootInitTask exposes the header of the compiler-generated Go init task.
// LLGo runs package initialization through guarded package init functions, so
// there are no runtime-dispatched function entries in this compatibility task.
func defineRootInitTask(pkg llssa.Package, pkgPath string) {
	prog := pkg.Prog
	taskType := prog.Struct(prog.Uint32(), prog.Uint32())
	pkg.NewVarEx(pkgPath+"..inittask", prog.Pointer(taskType)).InitNil()
}

func defineWeakNoArgStub(pkg llssa.Package, name string) llssa.Function {
	fn := pkg.NewFunc(name, llssa.NoArgsNoRet, llssa.InC)
	pkg.Module().NamedFunction(name).SetLinkage(llvm.WeakAnyLinkage)
	b := fn.MakeBody(1)
	b.Return()
	return fn
}

const (
	// The Universal CRT assigns _IONBF a different value from the Unix
	// runtimes. Passing the Unix value invokes UCRT's invalid-parameter
	// handler instead of disabling buffering.
	ioNoBufUnix    = 2
	ioNoBufWindows = 4
)

// emitStdioNobuf generates code to disable buffering on stdout and stderr
// when the LLGO_STDIO_NOBUF environment variable is set. Darwin exposes
// pointer globals with alternate names, while the Universal CRT exposes
// standard streams only through __acrt_iob_func.
func emitStdioNobuf(b llssa.Builder, pkg llssa.Package, goos string) {
	prog := pkg.Prog
	streamType := prog.VoidPtr()
	streamPtrType := prog.Pointer(streamType)

	var stdoutPtr, stderrPtr llssa.Expr
	if goos == "windows" {
		indexType := prog.Uint32()
		iob := declareAcrtIobFunc(pkg, streamPtrType, indexType)
		stdoutPtr = b.Call(iob.Expr, prog.IntVal(1, indexType))
		stderrPtr = b.Call(iob.Expr, prog.IntVal(2, indexType))
	} else {
		stdoutName := "stdout"
		stderrName := "stderr"
		if goos == "darwin" {
			stdoutName = "__stdoutp"
			stderrName = "__stderrp"
		}
		stdout := declareExternalPtrGlobal(pkg, stdoutName, streamPtrType)
		stderr := declareExternalPtrGlobal(pkg, stderrName, streamPtrType)
		stdoutPtr = b.Load(stdout)
		stderrPtr = b.Load(stderr)
	}
	sizeType := prog.Uintptr()
	setvbuf := declareSetvbuf(pkg, streamPtrType, prog.CStr(), prog.Int32(), sizeType)

	noBufModeValue := uint64(ioNoBufUnix)
	if goos == "windows" {
		noBufModeValue = ioNoBufWindows
	}
	noBufMode := prog.IntVal(noBufModeValue, prog.Int32())
	zeroSize := prog.Zero(sizeType)
	nullBuf := prog.Nil(prog.CStr())

	b.Call(setvbuf.Expr, stdoutPtr, nullBuf, noBufMode, zeroSize)
	b.Call(setvbuf.Expr, stderrPtr, nullBuf, noBufMode, zeroSize)
}

func declareAcrtIobFunc(pkg llssa.Package, streamPtrType, indexType llssa.Type) llssa.Function {
	sig := newSignature(
		[]types.Type{indexType.RawType()},
		[]types.Type{streamPtrType.RawType()},
	)
	return pkg.NewFunc("__acrt_iob_func", sig, llssa.InC)
}

func declareExternalPtrGlobal(pkg llssa.Package, name string, valueType llssa.Type) llssa.Expr {
	global := pkg.NewVarEx(name, valueType)
	pkg.Module().NamedGlobal(name).SetLinkage(llvm.ExternalLinkage)
	return global.Expr
}

func declareSetvbuf(pkg llssa.Package, streamPtrType, bufPtrType, intType, sizeType llssa.Type) llssa.Function {
	sig := newSignature(
		[]types.Type{
			streamPtrType.RawType(),
			bufPtrType.RawType(),
			intType.RawType(),
			sizeType.RawType(),
		},
		[]types.Type{intType.RawType()},
	)
	return pkg.NewFunc("setvbuf", sig, llssa.InC)
}

func tupleOf(tys ...types.Type) *types.Tuple {
	if len(tys) == 0 {
		return types.NewTuple()
	}
	vars := make([]*types.Var, len(tys))
	for i, t := range tys {
		vars[i] = types.NewParam(token.NoPos, nil, "", t)
	}
	return types.NewTuple(vars...)
}

func newSignature(params []types.Type, results []types.Type) *types.Signature {
	return types.NewSignatureType(nil, nil, nil, tupleOf(params...), tupleOf(results...), false)
}

func newEntrySignature(argvType types.Type) *types.Signature {
	return newSignature(
		[]types.Type{types.Typ[types.Int32], argvType},
		[]types.Type{types.Typ[types.Int32]},
	)
}
