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
	"runtime"
	"strings"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/types/typeutil"
)

type libffiOptions struct {
	windows         bool
	setFinalizerPtr bool
}

func libffiOptionsFor(ctx *context) libffiOptions {
	var opts libffiOptions
	if ctx == nil {
		return opts
	}
	if ctx.prog != nil && ctx.prog.Target() != nil {
		goos := ctx.prog.Target().GOOS
		if goos == "" {
			goos = runtime.GOOS
		}
		opts.windows = goos == "windows"
	}
	opts.setFinalizerPtr = programHasSetFinalizerPtr(ctx.progSSA)
	return opts
}

func programHasSetFinalizerPtr(prog *ssa.Program) bool {
	if prog == nil {
		return false
	}
	for _, pkg := range prog.AllPackages() {
		if pkg == nil || pkg.Pkg == nil || pkg.Func("SetFinalizerPtr") == nil {
			continue
		}
		if llssa.IsRuntimeSupportPackage(pkg.Pkg.Path()) {
			return true
		}
	}
	return false
}

// programUsesLibffi reports whether the already-built Go SSA program can
// reach a libffi entry. Executables use the shared init/main RTA result;
// libraries, tests, and package generation scan every function.
func programUsesLibffi(ctx *context) bool {
	if ctx == nil {
		return false
	}
	return libffiProgramUse(ctx).usesLibffi(libffiOptionsFor(ctx))
}

func libffiProgramUse(ctx *context) *programUse {
	if ctx == nil {
		return analyzeProgramUse(nil, nil)
	}
	if libffiUseExecutableRoots(ctx) {
		return programUseFor(ctx)
	}
	return analyzeProgramUse(ctx.progSSA, nil)
}

func libffiUseExecutableRoots(ctx *context) bool {
	if ctx == nil || ctx.buildConf == nil || ctx.progSSA == nil {
		return false
	}
	switch ctx.mode {
	case ModeBuild, ModeRun, ModeInstall:
	default:
		return false
	}
	if ctx.buildConf.BuildMode != BuildModeExe {
		return false
	}
	for _, pkg := range ctx.initial {
		if pkg == nil || pkg.Types == nil || pkg.Types.Name() != "main" {
			continue
		}
		ssaPkg := ctx.progSSA.Package(pkg.Types)
		if ssaPkg != nil && ssaPkg.Func("main") != nil {
			return true
		}
	}
	return false
}

func (use *programUse) usesLibffi(opts libffiOptions) bool {
	if use == nil {
		return false
	}
	found := false
	use.eachFunction(func(fn *ssa.Function) {
		if !found && functionCallsLibffi(fn, opts) {
			found = true
		}
	})
	return found || programMayCallLibffiIndirectly(use, opts)
}

func functionCallsLibffi(fn *ssa.Function, opts libffiOptions) bool {
	if fn == nil || skipLibffiCallSite(ssaFunctionPackagePath(fn)) {
		return false
	}
	for _, block := range fn.Blocks {
		for _, instruction := range block.Instrs {
			call, ok := instruction.(ssa.CallInstruction)
			if !ok || call.Common() == nil {
				continue
			}
			if isLibffiCall(call.Common(), opts) {
				return true
			}
		}
	}
	return false
}

func isLibffiCall(call *ssa.CallCommon, opts libffiOptions) bool {
	if call == nil {
		return false
	}
	callee := call.StaticCallee()
	if callee == nil {
		return false
	}
	path := ssaFunctionPackagePath(callee)
	name := callee.Name()
	if path == reflectPackagePath && isLibffiReflectName(name) {
		return true
	}
	if opts.setFinalizerPtr && isRuntimeSetFinalizer(callee) && setFinalizerCallNeedsFFI(call) {
		return true
	}
	return opts.windows && isSyscallNewCallback(callee)
}

func isLibffiReflectName(name string) bool {
	switch name {
	case "Call", "CallSlice", "MakeFunc":
		return true
	default:
		return false
	}
}

func isRuntimeSetFinalizer(fn *ssa.Function) bool {
	return fn != nil && fn.Name() == "SetFinalizer" && llssa.IsRuntimeSupportPackage(ssaFunctionPackagePath(fn))
}

func isSyscallNewCallback(fn *ssa.Function) bool {
	if fn == nil || ssaFunctionPackagePath(fn) != "syscall" {
		return false
	}
	switch fn.Name() {
	case "NewCallback", "NewCallbackCDecl":
		return true
	default:
		return false
	}
}

func skipLibffiCallSite(path string) bool {
	return path == reflectPackagePath || llssa.IsRuntimeSupportPackage(path)
}

func setFinalizerCallNeedsFFI(call *ssa.CallCommon) bool {
	if call == nil || len(call.Args) != 2 {
		return true
	}
	fn := unwrapSSAValue(call.Args[1])
	if c, ok := fn.(*ssa.Const); ok && (c == nil || c.Value == nil) {
		return false
	}
	if _, ok := fn.(*ssa.Function); ok {
		return false
	}
	return true
}

func unwrapSSAValue(v ssa.Value) ssa.Value {
	for v != nil {
		switch x := v.(type) {
		case *ssa.MakeInterface:
			v = x.X
		case *ssa.ChangeInterface:
			v = x.X
		case *ssa.ChangeType:
			v = x.X
		case *ssa.Convert:
			v = x.X
		default:
			return v
		}
	}
	return v
}

func programMayCallLibffiIndirectly(use *programUse, opts libffiOptions) bool {
	if use == nil {
		return false
	}
	var functionSignatures typeutil.Map
	found := false
	use.eachFunction(func(fn *ssa.Function) {
		if found || fn == nil {
			return
		}
		path := ssaFunctionPackagePath(fn)
		if name, suffix, ok := strings.Cut(fn.Name(), "$"); ok {
			if fn.Parent() == nil && suffix != "" && isLibffiIndirectName(path, name, opts) {
				found = true
			}
			return
		}
		if isLibffiIndirectFunction(fn, opts) && fn.Signature.Recv() == nil {
			functionSignatures.Set(fn.Signature, struct{}{})
		}
	})
	if found {
		return true
	}
	use.eachFunction(func(fn *ssa.Function) {
		if found || fn == nil || skipLibffiCallSite(ssaFunctionPackagePath(fn)) {
			return
		}
		for _, block := range fn.Blocks {
			for _, instruction := range block.Instrs {
				call, ok := instruction.(ssa.CallInstruction)
				if !ok || call.Common() == nil || call.Common().StaticCallee() != nil {
					continue
				}
				common := call.Common()
				if common.IsInvoke() && common.Method != nil && isLibffiReflectName(common.Method.Name()) {
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

func isLibffiIndirectName(path, name string, opts libffiOptions) bool {
	if path == reflectPackagePath && isLibffiReflectName(name) {
		return true
	}
	if opts.setFinalizerPtr && llssa.IsRuntimeSupportPackage(path) && name == "SetFinalizer" {
		return true
	}
	if opts.windows && path == "syscall" {
		switch name {
		case "NewCallback", "NewCallbackCDecl":
			return true
		}
	}
	return false
}

func isLibffiIndirectFunction(fn *ssa.Function, opts libffiOptions) bool {
	if fn == nil {
		return false
	}
	return isLibffiIndirectName(ssaFunctionPackagePath(fn), fn.Name(), opts)
}
