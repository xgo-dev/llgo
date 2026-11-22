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
	"slices"
	"strings"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
	"golang.org/x/tools/go/types/typeutil"
)

const reflectPackagePath = "reflect"
const wasmReflectLibffiTag = "llgo.wasm.reflect.libffi"

// configureWasmReflectBridges enables typed Wasm entries for programs that
// can use Go's dynamic reflection entry points. Emscripten workers have
// separate function tables, so dynamically installed libffi entries cannot
// be called by a different worker.
// Executables are restricted to functions reachable from init and main so
// dead reflection helpers in the standard library do not affect every build.
// Library builds, which have no executable roots, remain conservative.
func configureWasmReflectBridges(ctx *context) {
	if ctx == nil || ctx.prog == nil || ctx.prog.Target() == nil {
		return
	}
	target := ctx.prog.Target()
	if ctx.frontend != nil && ctx.frontend.program != nil {
		target.WasmReflectBridges = ctx.frontend.program.Features.ReflectBridges
	} else {
		target.WasmReflectBridges = wasmReflectBridgeProvider(ctx) && wasmProgramUseFor(ctx).usesWasmReflectBridges(ctx.prog)
	}
}

func wasmReflectBridgeProvider(ctx *context) bool {
	target := ctx.prog.Target()
	libffi := ctx.buildConf != nil && slices.Contains(splitSourcePatchBuildTags(ctx.buildConf.Tags), wasmReflectLibffiTag)
	return target.GOARCH == "wasm" && (target.WasmProvider == "wasi" || target.GOOS == "js" && !libffi)
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
	if ctx.frontend != nil && ctx.frontend.program != nil {
		target.WasmFuncInfoEntries = ctx.frontend.program.Features.FuncInfoEntries
	} else {
		target.WasmFuncInfoEntries = wasmProgramUseFor(ctx).usesRuntimeFuncForPC()
	}
}

type wasmProgramUse struct {
	rooted       bool
	runtimeFuncs map[*types.TypeName]bool
	reachable    map[*ssa.Function]struct{ AddrTaken bool }
	all          map[*ssa.Function]bool
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
		result := rta.Analyze(roots, false)
		use.reachable = result.Reachable
		use.runtimeFuncs = make(map[*types.TypeName]bool)
		result.RuntimeTypes.Iterate(func(t types.Type, _ any) {
			if named, ok := types.Unalias(t).(*types.Named); ok {
				if _, ok := named.Underlying().(*types.Signature); ok {
					use.runtimeFuncs[named.Origin().Obj()] = true
				}
			}
		})
	} else {
		use.all = ssautil.AllFunctions(prog)
	}
	return use
}

// analyzeWasmInitialUse keeps feature analysis local to one initial package.
// Executables use RTA from init/main. Entry-less packages have no whole-program
// roots, so start RTA from their own functions rather than the union SSA program.
func analyzeWasmInitialUse(prog *ssa.Program, pkg *types.Package) *wasmProgramUse {
	use := &wasmProgramUse{all: make(map[*ssa.Function]bool)}
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
		return analyzeWasmProgramUse(prog, roots)
	}
	var roots []*ssa.Function
	for fn := range ssautil.AllFunctions(prog) {
		if fnPkg := fn.Package(); fnPkg != nil && fnPkg.Pkg == pkg {
			roots = append(roots, fn)
		}
	}
	if len(roots) != 0 {
		use = analyzeWasmProgramUse(prog, roots)
		// A library's callers may supply additional reflected types.
		use.runtimeFuncs = nil
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
	return analyzeWasmProgramUse(prog, roots).usesWasmReflectBridges(nil)
}

func (use *wasmProgramUse) usesWasmReflectBridges(prog llssa.Program) bool {
	if use == nil {
		return false
	}
	nativeCallbacks := prog != nil && prog.HasNativeFuncTypes(use.runtimeFuncs)
	if use.rooted {
		for fn := range use.reachable {
			if functionCallsWasmReflectBridge(fn, nativeCallbacks) {
				return true
			}
		}
		// RTA conservatively retains wrapper methods for address-taken
		// reflect.Value values. Require an actual direct call above, or a
		// plausible indirect function/interface call here, so ordinary value
		// inspection (notably fmt) does not turn on every typed bridge.
		return programMayCallWasmReflectBridgeIndirectly(use.reachable, nativeCallbacks)
	}
	for fn := range use.all {
		if functionCallsWasmReflectBridge(fn, nativeCallbacks) {
			return true
		}
	}
	return false
}

func programMayCallWasmReflectBridgeIndirectly(reachable map[*ssa.Function]struct{ AddrTaken bool }, nativeCallbacks bool) bool {
	var functionSignatures typeutil.Map
	for fn := range reachable {
		if fn == nil || ssaFunctionPackagePath(fn) != reflectPackagePath {
			continue
		}
		// RTA creates thunks and bound wrappers only when a method is used as
		// a function value. Plain pointer wrappers retain the original name.
		if name, suffix, ok := strings.Cut(fn.Name(), "$"); ok {
			if fn.Parent() == nil && suffix != "" && isWasmReflectBridgeSignature(name, fn.Signature, nativeCallbacks) {
				return true
			}
			continue
		}
		if isWasmReflectBridgeFunction(fn, nativeCallbacks) && fn.Signature.Recv() == nil {
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
				if common.IsInvoke() && common.Method != nil && isWasmReflectBridgeSignature(common.Method.Name(), common.Signature(), nativeCallbacks) {
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

func functionCallsWasmReflectBridge(fn *ssa.Function, nativeCallbacks bool) bool {
	if fn == nil || ssaFunctionPackagePath(fn) == reflectPackagePath {
		return false
	}
	for _, block := range fn.Blocks {
		for _, instruction := range block.Instrs {
			call, ok := instruction.(ssa.CallInstruction)
			if ok && isWasmReflectBridgeCall(call.Common(), nativeCallbacks) {
				return true
			}
		}
	}
	return false
}

func isWasmReflectBridgeCall(call *ssa.CallCommon, nativeCallbacks bool) bool {
	if call == nil {
		return false
	}
	return isWasmReflectBridgeFunction(call.StaticCallee(), nativeCallbacks)
}

func isWasmReflectBridgeFunction(fn *ssa.Function, nativeCallbacks bool) bool {
	if fn == nil || ssaFunctionPackagePath(fn) != reflectPackagePath {
		return false
	}
	name := fn.Name()
	if obj := fn.Object(); obj != nil && obj.Name() == "TypeAssert" {
		name = obj.Name() // Instantiated SSA functions include type arguments in their name.
	}
	return isWasmReflectBridgeSignature(name, fn.Signature, nativeCallbacks)
}

func isWasmReflectBridgeSignature(name string, signature *types.Signature, nativeCallbacks bool) bool {
	switch name {
	case "Set", "Convert", "SetMapIndex", "SetIterKey", "SetIterValue", "Send", "TrySend", "Select", "Append", "AppendSlice", "Copy":
		// Each can adapt a native callback to a callable Go frame.
		return nativeCallbacks
	case "TypeAssert":
		if nativeCallbacks && signature != nil && signature.Results().Len() == 2 {
			_, ok := signature.Results().At(0).Type().Underlying().(*types.Signature)
			return ok
		}
		return false
	case "Call", "CallSlice", "MakeFunc", "Seq", "Seq2":
		// Indirect calls retain the conservative name-only match: an
		// unrelated interface method can enable extra bridges, but cannot
		// cause a required reflection bridge to be omitted.
		return true
	case "Method", "MethodByName":
		// A bound Value method can escape through Interface and be invoked as
		// an ordinary Go function, without any explicit Call or MakeFunc. Its
		// runtime trampoline still needs typed bridges. Type.Method returns
		// metadata instead and must not enable bridges in metadata-only users.
		if signature == nil || signature.Results().Len() != 1 {
			return false
		}
		typ, ok := types.Unalias(signature.Results().At(0).Type()).(*types.Named)
		return ok && typ.Obj().Pkg() != nil && typ.Obj().Pkg().Path() == reflectPackagePath && typ.Obj().Name() == "Value"
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
	if parent := fn.Parent(); parent != nil {
		// A closure in an instantiated generic can lack its own package and object.
		return ssaFunctionPackagePath(parent)
	}
	return ""
}
