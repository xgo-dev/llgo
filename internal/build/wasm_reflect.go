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
	"strings"

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

// configureWasmFuncInfoEntries keeps table-index metadata out of ordinary
// Wasm programs. The records deliberately retain address-taken functions, so
// emitting them unconditionally would turn dead function values into link
// roots. Executables need the records only when runtime.FuncForPC is reachable;
// library builds remain conservative because their external call roots are not
// represented by main/init reachability.
func configureWasmFuncInfoEntries(ctx *context) {
	if ctx == nil || ctx.prog == nil || ctx.prog.Target() == nil {
		return
	}
	target := ctx.prog.Target()
	if target.GOARCH != "wasm" {
		return
	}
	if ctx.buildConf == nil || ctx.buildConf.BuildMode != BuildModeExe {
		target.WasmFuncInfoEntries = true
		return
	}
	target.WasmFuncInfoEntries = programUsesRuntimeFuncForPC(ctx.progSSA, wasmReflectRoots(ctx))
}

func programUsesRuntimeFuncForPC(prog *ssa.Program, roots []*ssa.Function) bool {
	if prog == nil {
		return false
	}
	if len(roots) != 0 {
		for fn := range rta.Analyze(roots, false).Reachable {
			if isRuntimeFuncForPC(fn) {
				return true
			}
		}
		return false
	}
	for fn := range ssautil.AllFunctions(prog) {
		if isRuntimeFuncForPC(fn) {
			return true
		}
	}
	return false
}

func isRuntimeFuncForPC(fn *ssa.Function) bool {
	return fn != nil && fn.Pkg != nil && fn.Pkg.Pkg != nil &&
		fn.Pkg.Pkg.Path() == "runtime" && fn.Name() == "FuncForPC"
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
		result := rta.Analyze(roots, false)
		for fn := range result.Reachable {
			if functionCallsWasmReflectBridge(fn) {
				return true
			}
		}
		// RTA conservatively retains wrapper methods for address-taken
		// reflect.Value values. Require an actual direct call above, or a
		// plausible indirect function/interface call here, so ordinary value
		// inspection (notably fmt) does not turn on every typed bridge.
		return programMayCallWasmReflectBridgeIndirectly(result.Reachable)
	}
	for fn := range ssautil.AllFunctions(prog) {
		if functionCallsWasmReflectBridge(fn) {
			return true
		}
	}
	return false
}

func programMayCallWasmReflectBridgeIndirectly(reachable map[*ssa.Function]struct{ AddrTaken bool }) bool {
	var functionSignatures []*types.Signature
	for fn := range reachable {
		if fn == nil || ssaFunctionPackagePath(fn) != reflectPackagePath {
			continue
		}
		// RTA creates thunks and bound wrappers only when a method is used as
		// a function value. Plain pointer wrappers retain the original name.
		if name, suffix, ok := strings.Cut(fn.Name(), "$"); ok {
			if fn.Parent() == nil && suffix != "" && isWasmReflectBridgeName(name) {
				return true
			}
			continue
		}
		if isWasmReflectBridgeFunction(fn) && fn.Signature.Recv() == nil {
			functionSignatures = append(functionSignatures, fn.Signature)
		}
	}
	for fn := range reachable {
		if fn == nil || ssaFunctionPackagePath(fn) == reflectPackagePath {
			continue
		}
		for _, block := range fn.Blocks {
			for _, instruction := range block.Instrs {
				call, ok := instruction.(ssa.CallInstruction)
				if !ok || call.Common() == nil || call.Common().StaticCallee() != nil {
					continue
				}
				common := call.Common()
				if common.IsInvoke() && common.Method != nil && isWasmReflectBridgeName(common.Method.Name()) {
					return true
				}
				if !common.IsInvoke() {
					for _, signature := range functionSignatures {
						if types.Identical(common.Signature(), signature) {
							return true
						}
					}
				}
			}
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
	return isWasmReflectBridgeName(fn.Name())
}

func isWasmReflectBridgeName(name string) bool {
	switch name {
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
	return ""
}
