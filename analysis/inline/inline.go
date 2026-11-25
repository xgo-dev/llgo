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

// Package inline computes pure-Go per-package inlining cost and eligibility
// summaries over llgo/ssa IR for cross-package incremental builds.
package inline

import (
	"sort"

	"github.com/xgo-dev/llgo/exportdata"
	"github.com/xgo-dev/llgo/ssa"
)

// DefaultBudget is the standard instruction-cost budget mirroring Go's inliner.
const DefaultBudget = 80

// AnalyzePackage evaluates inlining cost and eligibility for every function in
// pkg using budget (defaults to DefaultBudget if budget <= 0).
func AnalyzePackage(pkg *ssa.Package, budget int) *exportdata.InlineSummary {
	if budget <= 0 {
		budget = DefaultBudget
	}
	if pkg == nil {
		return &exportdata.InlineSummary{Budget: budget}
	}
	res := &exportdata.InlineSummary{
		PkgPath: pkg.Path,
		Budget:  budget,
		Funcs:   make([]exportdata.FuncInline, 0, len(pkg.Funcs)),
	}
	for _, fn := range pkg.Funcs {
		res.Funcs = append(res.Funcs, evaluateFunc(fn, budget))
	}
	sort.Slice(res.Funcs, func(i, j int) bool { return res.Funcs[i].Name < res.Funcs[j].Name })
	return res
}

func evaluateFunc(fn *ssa.Function, budget int) exportdata.FuncInline {
	fi := exportdata.FuncInline{Name: fn.Name}
	if fn.IsDeclaration() {
		fi.Reason = "declaration"
		return fi
	}
	if fn.NoInline {
		fi.Reason = "noinline"
		return fi
	}

	cost := 0
	callees := make(map[string]bool)
	disqualify := ""

	for _, blk := range fn.Blocks {
		for _, ins := range blk.Instrs {
			switch ins := ins.(type) {
			case *ssa.DebugRef, *ssa.KeepAlive:
				// Zero runtime cost.
			case *ssa.Jump, *ssa.Return, *ssa.Unreachable, *ssa.Extract:
				cost += 1
			case *ssa.BinOp, *ssa.UnOp, *ssa.Convert, *ssa.PtrCast, *ssa.FieldAddr, *ssa.IndexAddr, *ssa.Load, *ssa.Store, *ssa.Phi:
				cost += 1
			case *ssa.Branch:
				cost += 2
			case *ssa.Alloc:
				if ins.Heap {
					cost += 12
				} else {
					cost += 2
				}
			case *ssa.WriteBarrier, *ssa.Safepoint, *ssa.GCRoot:
				cost += 4
			case *ssa.MakeClosure:
				cost += 8
				callees[ins.FnName] = true
			case *ssa.CallDirect:
				callees[ins.Callee] = true
				if ins.Callee == fn.Name {
					disqualify = "recursive"
				}
				cost += 15
			case *ssa.CallCFunc:
				cost += 20
			case *ssa.CallClosure, *ssa.CallInterface:
				cost += 25
			case *ssa.Defer, *ssa.RunDefers:
				disqualify = "defer"
			case *ssa.Recover:
				disqualify = "recover"
			case *ssa.Go:
				disqualify = "go"
			case *ssa.Panic:
				cost += 10
			default:
				cost += 2
			}
		}
	}

	for c := range callees {
		fi.Callees = append(fi.Callees, c)
	}
	sort.Strings(fi.Callees)
	fi.Cost = cost

	if disqualify != "" {
		fi.Reason = disqualify
		return fi
	}
	if cost > budget {
		fi.Reason = "over-budget"
		return fi
	}
	fi.Inlineable = true
	return fi
}
