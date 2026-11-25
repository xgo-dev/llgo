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

// Package deadcode implements pure-Go reachability and dead-code elimination
// analysis over llgo/ssa packages, exportdata.PackageMeta summaries, and
// exportdata.LinkSnapshot metadata.
package deadcode

import (
	"sort"

	"github.com/xgo-dev/llgo/analysis/callgraph"
	"github.com/xgo-dev/llgo/exportdata"
	"github.com/xgo-dev/llgo/ssa"
)

// Result holds deterministic reachability sets computed by Analyze.
type Result struct {
	ReachableFuncs   []string
	DeadFuncs        []string
	ReachableGlobals []string
	DeadGlobals      []string
	ReachableMethods []string
}

// Analyze computes reachable and dead functions, globals, and interface methods
// from entry roots plus exported C/Wasm/ABI symbols found in packages and
// LinkSnapshots.
func Analyze(pkgs []*ssa.Package, metas []*exportdata.PackageMeta, snapshots []*exportdata.LinkSnapshot, roots []string) *Result {
	allRoots := append([]string(nil), roots...)
	allFuncs := make(map[string]bool)
	allGlobals := make(map[string]bool)
	funcGlobals := make(map[string][]string)

	for _, pkg := range pkgs {
		if pkg == nil {
			continue
		}
		for goSym := range pkg.Exports {
			allRoots = append(allRoots, goSym)
		}
		for _, g := range pkg.Globals {
			allGlobals[g.Name] = true
		}
		for _, fn := range pkg.Funcs {
			allFuncs[fn.Name] = true
			if fn.ExportName != "" {
				allRoots = append(allRoots, fn.Name)
			}
			for _, blk := range fn.Blocks {
				for _, ins := range blk.Instrs {
					for _, op := range ins.Operands() {
						if gr, ok := op.(*ssa.GlobalRef); ok && gr.Global != nil {
							funcGlobals[fn.Name] = append(funcGlobals[fn.Name], gr.Global.Name)
						}
					}
				}
			}
		}
	}

	for _, meta := range metas {
		if meta == nil {
			continue
		}
		for _, fn := range meta.Summary.DefinedFuncs {
			allFuncs[fn] = true
		}
		for _, g := range meta.DefinedGlobals {
			allGlobals[g] = true
		}
		allRoots = append(allRoots, meta.ExportFuncs...)
	}

	liveMethods := make(map[string]bool)
	for _, snap := range snapshots {
		if snap == nil {
			continue
		}
		allRoots = append(allRoots, snap.CExports...)
		for _, m := range snap.MethodByName {
			liveMethods[m] = true
		}
	}

	cg := callgraph.Analyze(pkgs, metas, allRoots)
	for m := range cg.InterfaceCalls {
		liveMethods[m] = true
	}

	reachableGlobals := make(map[string]bool)
	for fn := range cg.Reachable {
		for _, g := range funcGlobals[fn] {
			reachableGlobals[g] = true
		}
	}

	res := &Result{}
	for fn := range allFuncs {
		if cg.Reachable[fn] {
			res.ReachableFuncs = append(res.ReachableFuncs, fn)
		} else {
			res.DeadFuncs = append(res.DeadFuncs, fn)
		}
	}
	for g := range allGlobals {
		if reachableGlobals[g] {
			res.ReachableGlobals = append(res.ReachableGlobals, g)
		} else {
			res.DeadGlobals = append(res.DeadGlobals, g)
		}
	}
	for m := range liveMethods {
		res.ReachableMethods = append(res.ReachableMethods, m)
	}

	sort.Strings(res.ReachableFuncs)
	sort.Strings(res.DeadFuncs)
	sort.Strings(res.ReachableGlobals)
	sort.Strings(res.DeadGlobals)
	sort.Strings(res.ReachableMethods)
	return res
}
