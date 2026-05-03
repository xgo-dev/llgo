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
	llvm "github.com/xgo-dev/llvm"

	llssa "github.com/goplus/llgo/ssa"
)

type genConfig struct {
	rtInit     bool
	pyInit     bool
	abiInit    int
	abiSymbols map[string]none
}

// genMainModule generates the main entry module for an llgo program.
//
// The module contains argc/argv globals and, for executable build modes,
// the entry function that wires initialization and main. For C archive or
// shared library modes, only the globals are emitted.
func genMainModule(ctx *context, rtPkgPath string, pkg *packages.Package, cfg *genConfig) Package {
	prog := ctx.prog
	mainPkg := prog.NewPackage("", pkg.ID+".main")

	argcVar := mainPkg.NewVarEx("__llgo_argc", prog.Pointer(prog.Int32()))
	argcVar.Init(prog.Zero(prog.Int32()))

	argvValueType := prog.Pointer(prog.CStr())
	argvVar := mainPkg.NewVarEx("__llgo_argv", prog.Pointer(argvValueType))
	argvVar.InitNil()

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

	if ctx.buildConf.BuildMode != BuildModeExe {
		return mainAPkg
	}

	runtimeStub := defineWeakNoArgStub(mainPkg, "runtime.init")
	// TODO(lijie): workaround for syscall patch
	defineWeakNoArgStub(mainPkg, "syscall.init")

	var pyInit llssa.Function
	if cfg.pyInit {
		pyInit = declareNoArgFunc(mainPkg, "Py_Initialize")
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

	mainInit := declareNoArgFunc(mainPkg, pkg.PkgPath+".init")
	mainMain := declareNoArgFunc(mainPkg, pkg.PkgPath+".main")

	entryFn := defineEntryFunction(ctx, mainPkg, argcVar, argvVar, argvValueType, runtimeStub, mainInit, mainMain, pyInit, rtInit, abiInit)

	if needStart(ctx) {
		defineStart(mainPkg, entryFn, argvValueType)
	}

	return mainAPkg
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

// defineEntryFunction creates the program's entry function. The name is
// "main" for standard targets, or "__main_argc_argv" with hidden visibility
// for WASM targets that don't require _start.
//
// The entry stores argc/argv, optionally disables stdio buffering, runs
// initialization hooks (Python, runtime, package init), and finally calls
// main.main before returning 0.
func defineEntryFunction(ctx *context, pkg llssa.Package, argcVar, argvVar llssa.Global, argvType llssa.Type, runtimeStub, mainInit, mainMain llssa.Function, pyInit, rtInit, abiInit llssa.Function) llssa.Function {
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
	b.Store(argcVar.Expr, fn.Param(0))
	b.Store(argvVar.Expr, fn.Param(1))
	if IsStdioNobuf() {
		emitStdioNobuf(b, pkg, ctx.buildConf.Goos)
	}
	if pyInit != nil {
		b.Call(pyInit.Expr)
	}
	if rtInit != nil {
		b.Call(rtInit.Expr)
	}
	if abiInit != nil {
		b.Call(abiInit.Expr)
	}
	b.Call(runtimeStub.Expr)
	b.Call(mainInit.Expr)
	b.Call(mainMain.Expr)
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
