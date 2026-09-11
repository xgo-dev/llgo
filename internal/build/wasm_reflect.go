/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
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

package build

import (
	"go/types"

	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

const reflectPackagePath = "reflect"

// configureWasmReflectBridges enables the typed fallback only for WASI
// programs that can use Go's dynamic reflection entry points.
// Executables are restricted to functions reachable from init and main so
// dead reflection helpers in the standard library do not affect every build.
// Library builds, which have no executable roots, remain conservative.
func configureWasmReflectBridges(ctx *context) {
	if ctx == nil || ctx.prog == nil || ctx.prog.Target() == nil {
		return
	}
	target := ctx.prog.Target()
	wasiProvider := target.GOARCH == "wasm" && target.WasmProvider == "wasi"
	target.WasmReflectBridges = wasiProvider && programUsesWasmReflectBridges(ctx.progSSA, wasmReflectRoots(ctx))
}

func wasmReflectRoots(ctx *context) (roots []*ssa.Function) {
	if ctx == nil || ctx.progSSA == nil {
		return nil
	}
	for _, pkg := range ctx.initial {
		if pkg == nil || pkg.Types == nil {
			continue
		}
		ssaPkg := ctx.progSSA.Package(pkg.Types)
		if ssaPkg == nil {
			continue
		}
		for _, name := range []string{"init", "main"} {
			if fn := ssaPkg.Func(name); fn != nil {
				roots = append(roots, fn)
			}
		}
	}
	return roots
}

func programUsesWasmReflectBridges(prog *ssa.Program, roots []*ssa.Function) bool {
	if prog == nil {
		return false
	}
	if len(roots) != 0 {
		for fn := range rta.Analyze(roots, false).Reachable {
			if isWasmReflectBridgeFunction(fn) || functionCallsWasmReflectBridge(fn) {
				return true
			}
		}
		return false
	}
	for fn := range ssautil.AllFunctions(prog) {
		if functionCallsWasmReflectBridge(fn) {
			return true
		}
	}
	return false
}

func functionCallsWasmReflectBridge(fn *ssa.Function) bool {
	if fn == nil || ssaFunctionPackagePath(fn) == reflectPackagePath {
		return false
	}
	for _, block := range fn.Blocks {
		for _, instruction := range block.Instrs {
			call, ok := instruction.(ssa.CallInstruction)
			if ok && isWasmReflectBridgeCall(call.Common()) {
				return true
			}
		}
	}
	return false
}

func isWasmReflectBridgeCall(call *ssa.CallCommon) bool {
	if call == nil {
		return false
	}
	return isWasmReflectBridgeFunction(call.StaticCallee())
}

func isWasmReflectBridgeFunction(fn *ssa.Function) bool {
	if fn == nil || ssaFunctionPackagePath(fn) != reflectPackagePath {
		return false
	}
	switch fn.Name() {
	case "Call", "CallSlice", "MakeFunc", "Seq", "Seq2":
		return true
	default:
		return false
	}
}

func ssaFunctionPackagePath(fn *ssa.Function) string {
	if fn == nil {
		return ""
	}
	if pkg := fn.Package(); pkg != nil && pkg.Pkg != nil {
		return pkg.Pkg.Path()
	}
	if obj := fn.Object(); obj != nil {
		if pkg := obj.Pkg(); pkg != nil {
			return pkg.Path()
		}
	}
	if signature, ok := fn.Type().(*types.Signature); ok && signature.Recv() != nil {
		typ := signature.Recv().Type()
		if pointer, ok := typ.(*types.Pointer); ok {
			typ = pointer.Elem()
		}
		if named, ok := typ.(*types.Named); ok && named.Obj().Pkg() != nil {
			return named.Obj().Pkg().Path()
		}
	}
	return ""
}
