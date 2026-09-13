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
	"strings"

	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
	"golang.org/x/tools/go/types/typeutil"
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
	target.WasmReflectBridges = wasiProvider && wasmProgramUseFor(ctx).usesWasmReflectBridges()
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
	target.WasmFuncInfoEntries = wasmProgramUseFor(ctx).usesRuntimeFuncForPC()
}

type wasmProgramUse struct {
	rooted    bool
	reachable map[*ssa.Function]struct{ AddrTaken bool }
	all       map[*ssa.Function]bool
}

func wasmProgramUseFor(ctx *context) *wasmProgramUse {
	if ctx == nil {
		return nil
	}
	ctx.wasmProgramUseOnce.Do(func() {
		ctx.wasmProgramUse = analyzeWasmProgramUse(ctx.progSSA, wasmReflectRoots(ctx))
	})
	return ctx.wasmProgramUse
}

func analyzeWasmProgramUse(prog *ssa.Program, roots []*ssa.Function) *wasmProgramUse {
	use := &wasmProgramUse{rooted: len(roots) != 0}
	if prog == nil {
		return use
	}
	if use.rooted {
		use.reachable = rta.Analyze(roots, false).Reachable
	} else {
		use.all = ssautil.AllFunctions(prog)
	}
	return use
}

func programUsesRuntimeFuncForPC(prog *ssa.Program, roots []*ssa.Function) bool {
	return analyzeWasmProgramUse(prog, roots).usesRuntimeFuncForPC()
}

func (use *wasmProgramUse) usesRuntimeFuncForPC() bool {
	if use == nil {
		return false
	}
	if use.rooted {
		for fn := range use.reachable {
			if isRuntimeFuncForPC(fn) {
				return true
			}
		}
		return false
	}
	for fn := range use.all {
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
	return analyzeWasmProgramUse(prog, roots).usesWasmReflectBridges()
}

func (use *wasmProgramUse) usesWasmReflectBridges() bool {
	if use == nil {
		return false
	}
	if use.rooted {
		for fn := range use.reachable {
			if functionCallsWasmReflectBridge(fn) {
				return true
			}
		}
		// RTA conservatively retains wrapper methods for address-taken
		// reflect.Value values. Require an actual direct call above, or a
		// plausible indirect function/interface call here, so ordinary value
		// inspection (notably fmt) does not turn on every typed bridge.
		return programMayCallWasmReflectBridgeIndirectly(use.reachable)
	}
	for fn := range use.all {
		if functionCallsWasmReflectBridge(fn) {
			return true
		}
	}
	return false
}

func programMayCallWasmReflectBridgeIndirectly(reachable map[*ssa.Function]struct{ AddrTaken bool }) bool {
	var functionSignatures typeutil.Map
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
			functionSignatures.Set(fn.Signature, struct{}{})
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
				if !common.IsInvoke() && functionSignatures.At(common.Signature()) != nil {
					return true
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
