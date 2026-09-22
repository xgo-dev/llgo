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
	target.WasmReflectBridges = wasiProvider && programUseFor(ctx).usesWasmReflectBridges()
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
	target.WasmFuncInfoEntries = programUseFor(ctx).usesRuntimeFuncForPC()
}

// analyzeWasmInitialUse keeps feature analysis local to one initial package.
// Executables use RTA from init/main. Entry-less packages have no whole-program
// roots, so start RTA from their own functions rather than the union SSA program.
func analyzeWasmInitialUse(prog *ssa.Program, pkg *types.Package) *programUse {
	use := &programUse{all: make(map[*ssa.Function]bool)}
	if prog == nil || pkg == nil {
		return use
	}
	ssaPkg := prog.Package(pkg)
	if ssaPkg == nil {
		return use
	}
	if main := ssaPkg.Func("main"); main != nil {
		roots := []*ssa.Function{main}
		if init := ssaPkg.Func("init"); init != nil {
			roots = append(roots, init)
		}
		return analyzeProgramUse(prog, roots)
	}
	var roots []*ssa.Function
	for fn := range ssautil.AllFunctions(prog) {
		if fnPkg := fn.Package(); fnPkg != nil && fnPkg.Pkg == pkg {
			roots = append(roots, fn)
		}
	}
	if len(roots) != 0 {
		return analyzeProgramUse(prog, roots)
	}
	return use
}

func programUsesRuntimeFuncForPC(prog *ssa.Program, roots []*ssa.Function) bool {
	return analyzeProgramUse(prog, roots).usesRuntimeFuncForPC()
}

func (use *programUse) usesRuntimeFuncForPC() bool {
	if use == nil {
		return false
	}
	found := false
	use.eachFunction(func(fn *ssa.Function) {
		if !found && isRuntimeFuncForPC(fn) {
			found = true
		}
	})
	return found
}

func isRuntimeFuncForPC(fn *ssa.Function) bool {
	return fn != nil && fn.Pkg != nil && fn.Pkg.Pkg != nil &&
		fn.Pkg.Pkg.Path() == "runtime" && fn.Name() == "FuncForPC"
}

func programUsesWasmReflectBridges(prog *ssa.Program, roots []*ssa.Function) bool {
	return analyzeProgramUse(prog, roots).usesWasmReflectBridges()
}

func (use *programUse) usesWasmReflectBridges() bool {
	if use == nil {
		return false
	}
	found := false
	use.eachFunction(func(fn *ssa.Function) {
		if !found && functionCallsWasmReflectBridge(fn) {
			found = true
		}
	})
	if found {
		return true
	}
	if use.rooted {
		// RTA conservatively retains wrapper methods for address-taken
		// reflect.Value values. Require an actual direct call above, or a
		// plausible indirect function/interface call here, so ordinary value
		// inspection (notably fmt) does not turn on every typed bridge.
		return programMayCallWasmReflectBridgeIndirectly(use)
	}
	return false
}

func programMayCallWasmReflectBridgeIndirectly(use *programUse) bool {
	if use == nil {
		return false
	}
	var functionSignatures typeutil.Map
	found := false
	use.eachFunction(func(fn *ssa.Function) {
		if found || fn == nil || ssaFunctionPackagePath(fn) != reflectPackagePath {
			return
		}
		// RTA creates thunks and bound wrappers only when a method is used as
		// a function value. Plain pointer wrappers retain the original name.
		if name, suffix, ok := strings.Cut(fn.Name(), "$"); ok {
			if fn.Parent() == nil && suffix != "" && isWasmReflectBridgeName(name) {
				found = true
			}
			return
		}
		if isWasmReflectBridgeFunction(fn) && fn.Signature.Recv() == nil {
			functionSignatures.Set(fn.Signature, struct{}{})
		}
	})
	if found {
		return true
	}
	use.eachFunction(func(fn *ssa.Function) {
		if found || fn == nil || ssaFunctionPackagePath(fn) == reflectPackagePath {
			return
		}
		for _, block := range fn.Blocks {
			for _, instruction := range block.Instrs {
				call, ok := instruction.(ssa.CallInstruction)
				if !ok || call.Common() == nil || call.Common().StaticCallee() != nil {
					continue
				}
				common := call.Common()
				if common.IsInvoke() && common.Method != nil && isWasmReflectBridgeName(common.Method.Name()) {
					found = true
					return
				}
				if !common.IsInvoke() && functionSignatures.At(common.Signature()) != nil {
					found = true
					return
				}
			}
		}
	})
	return found
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
