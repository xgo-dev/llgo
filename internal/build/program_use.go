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
	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// programUse is a whole-program view of Go SSA. Executable roots use RTA from
// init and main; library and test graphs scan every function.
type programUse struct {
	rooted    bool
	reachable map[*ssa.Function]struct{ AddrTaken bool }
	all       map[*ssa.Function]bool
}

func programUseFor(ctx *context) *programUse {
	if ctx == nil {
		return nil
	}
	ctx.programUseOnce.Do(func() {
		ctx.programUse = analyzeProgramUse(ctx.progSSA, programUseRoots(ctx))
	})
	return ctx.programUse
}

func analyzeProgramUse(prog *ssa.Program, roots []*ssa.Function) *programUse {
	use := &programUse{rooted: len(roots) != 0}
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

func programUseRoots(ctx *context) (roots []*ssa.Function) {
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

func (use *programUse) eachFunction(visit func(*ssa.Function)) {
	if use == nil || visit == nil {
		return
	}
	if use.rooted {
		for fn := range use.reachable {
			visit(fn)
		}
		return
	}
	for fn := range use.all {
		visit(fn)
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
