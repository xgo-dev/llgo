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

	"github.com/goplus/llgo/internal/packages"
	llssa "github.com/goplus/llgo/ssa"
	llvm "github.com/xgo-dev/llvm"
)

type genConfig struct {
	rtInit        bool
	pyInit        bool
	abiInit       int
	methodByIndex map[int]none
	methodByName  map[string]none
	abiSymbols    map[string]none
	funcInfo      []funcInfoRecord
	pcLineInfo    []pcLineRecord
	funcInfoStubs []funcInfoStubRecord
}

// genMainModule generates the main entry module for an llgo program.
//
// The module contains argc/argv globals and, for executable build modes,
// the entry function that wires initialization and main. C archive and shared
// library modes also get a constructor when the LLGo runtime is linked.
func genMainModule(ctx *context, rtPkgPath string, pkg *packages.Package, cfg *genConfig) Package {
	prog := ctx.prog
	mainPkg := prog.NewPackage("", pkg.ID+".main")

	argcVar := mainPkg.NewVarEx("__llgo_argc", prog.Pointer(prog.Int32()))
	argcVar.Init(prog.Zero(prog.Int32()))

	argvValueType := prog.Pointer(prog.CStr())
	argvVar := mainPkg.NewVarEx("__llgo_argv", prog.Pointer(argvValueType))
	argvVar.InitNil()
	emitFuncInfoTable(ctx, mainPkg, cfg.funcInfo, cfg.pcLineInfo, cfg.funcInfoStubs)

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

	var rtInit llssa.Function
	if cfg.rtInit {
		rtInit = declareNoArgFunc(mainPkg, rtPkgPath+".init")
	}

	var abiInit llssa.Function
	if cfg.abiInit != 0 {
		abiInit = mainPkg.InitAbiTypesFor("init$abitypes", func(sym *llssa.AbiSymbol) bool {
			if _, ok := cfg.abiSymbols[sym.Name]; !ok {
				return false
			}
			return filterAbiSymbol(cfg.abiInit, sym)
		})
	}

	var pkgPath string
	if ctx.buildConf.RewriteMainPrefix && pkg.Types.Name() == "main" {
		pkgPath = "main"
	} else {
		pkgPath = pkg.PkgPath
	}

	mainInit := declareNoArgFunc(mainPkg, pkgPath+".init")
	mainMain := declareNoArgFunc(mainPkg, pkgPath+".main")

	if ctx.buildConf.BuildMode != BuildModeExe {
		initArraySection := ""
		if ctx.buildConf.BuildMode == BuildModeCShared && ctx.buildConf.Goos == "linux" {
			initArraySection = ".init_array"
		}
		inits := []llssa.Function{pyInit, rtInit, abiInit, runtimeStub}
		// The C test runner supplies argc/argv before calling the generated
		// test main package's init and main functions.
		if ctx.mode != ModeTest {
			inits = append(inits, mainInit)
		}
		defineLibraryRuntimeInit(mainPkg, initArraySection, inits...)
		return mainAPkg
	}

	entryFn := defineEntryFunction(ctx, mainPkg, argcVar, argvVar, argvValueType, entryFunctions{
		runtimeStub: runtimeStub,
		mainInit:    mainInit,
		mainMain:    mainMain,
		pyInit:      pyInit,
		pyFinalize:  pyFinalize,
		rtInit:      rtInit,
		abiInit:     abiInit,
	})

	if needStart(ctx) {
		defineStart(mainPkg, entryFn, argvValueType)
	}

	return mainAPkg
}

// defineLibraryRuntimeInit arranges for the LLGo runtime to be initialized
// before a C program calls an exported Go function. Linux shared libraries use
// .init_array explicitly because the raw LLVM target machine lowers
// llvm.global_ctors to legacy .ctors; other library formats use LLVM's
// platform-specific constructor lowering.
func defineLibraryRuntimeInit(pkg llssa.Package, initArraySection string, inits ...llssa.Function) {
	const ctorName = "__llgo_runtime_ctor"
	ctor := pkg.NewFunc(ctorName, llssa.NoArgsNoRet, llssa.InC)
	ctorValue := pkg.Module().NamedFunction(ctorName)
	ctorValue.SetLinkage(llvm.InternalLinkage)
	b := ctor.MakeBody(1)
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
	runtimeStub llssa.Function
	mainInit    llssa.Function
	mainMain    llssa.Function
	pyInit      llssa.Function
	pyFinalize  llssa.Function
	rtInit      llssa.Function
	abiInit     llssa.Function
}

// defineEntryFunction creates the program's entry function. The name is
// "main" for standard targets, or "__main_argc_argv" with hidden visibility
// for WASM targets that don't require _start.
//
// The entry stores argc/argv, optionally disables stdio buffering, runs
// initialization hooks (Python, runtime, package init), calls main.main,
// finalizes Python if it was initialized, and returns 0.
func defineEntryFunction(ctx *context, pkg llssa.Package, argcVar, argvVar llssa.Global, argvType llssa.Type, fns entryFunctions) llssa.Function {
	prog := pkg.Prog
	entryName := "main"
	if !needStart(ctx) && isWasmTarget(ctx.buildConf.Goos) {
		entryName = "__main_argc_argv"
	}
	sig := newEntrySignature(argvType.RawType())
	fn := pkg.NewFunc(entryName, sig, llssa.InC)
	fnVal := pkg.Module().NamedFunction(entryName)
	if entryName != "main" {
		fnVal.SetVisibility(llvm.HiddenVisibility)
		fnVal.SetUnnamedAddr(true)
	}
	b := fn.MakeBody(1)
	var localCtx, previousLocalCtx llssa.Expr
	hasLocalContext := prog.NeedsLocalContext()
	if hasLocalContext {
		localCtx, previousLocalCtx = b.EnterLocalContext()
	}
	b.Store(argcVar.Expr, fn.Param(0))
	b.Store(argvVar.Expr, fn.Param(1))
	if isEnvOnConfig(ctx.buildConf, llgoStdioNobuf, false) {
		emitStdioNobuf(b, pkg, ctx.buildConf.Goos)
	}
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
	b.Call(fns.mainInit.Expr)
	b.Call(fns.mainMain.Expr)
	if fns.pyFinalize != nil {
		b.Call(fns.pyFinalize.Expr)
	}
	if hasLocalContext {
		b.LeaveLocalContext(localCtx, previousLocalCtx)
	}
	b.Return(prog.IntVal(0, prog.Int32()))
	return fn
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

func defineWeakNoArgStub(pkg llssa.Package, name string) llssa.Function {
	fn := pkg.NewFunc(name, llssa.NoArgsNoRet, llssa.InC)
	pkg.Module().NamedFunction(name).SetLinkage(llvm.WeakAnyLinkage)
	b := fn.MakeBody(1)
	b.Return()
	return fn
}

const (
	// ioNoBuf represents the _IONBF flag for setvbuf (no buffering)
	ioNoBuf = 2
)

// emitStdioNobuf generates code to disable buffering on stdout and stderr
// when the LLGO_STDIO_NOBUF environment variable is set. Only Darwin uses
// the alternate `__stdoutp`/`__stderrp` symbols; other targets rely on the
// standard `stdout`/`stderr` globals.
func emitStdioNobuf(b llssa.Builder, pkg llssa.Package, goos string) {
	prog := pkg.Prog
	streamType := prog.VoidPtr()
	streamPtrType := prog.Pointer(streamType)

	stdoutName := "stdout"
	stderrName := "stderr"
	if goos == "darwin" {
		stdoutName = "__stdoutp"
		stderrName = "__stderrp"
	}
	stdout := declareExternalPtrGlobal(pkg, stdoutName, streamPtrType)
	stderr := declareExternalPtrGlobal(pkg, stderrName, streamPtrType)
	stdoutPtr := b.Load(stdout)
	stderrPtr := b.Load(stderr)
	sizeType := prog.Uintptr()
	setvbuf := declareSetvbuf(pkg, streamPtrType, prog.CStr(), prog.Int32(), sizeType)

	noBufMode := prog.IntVal(ioNoBuf, prog.Int32())
	zeroSize := prog.Zero(sizeType)
	nullBuf := prog.Nil(prog.CStr())

	b.Call(setvbuf.Expr, stdoutPtr, nullBuf, noBufMode, zeroSize)
	b.Call(setvbuf.Expr, stderrPtr, nullBuf, noBufMode, zeroSize)
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
