package cl

import (
	"go/constant"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/ssa"
)

func rangeSliceIndexInBounds(access *ssa.IndexAddr) bool {
	if _, slice := access.X.Type().Underlying().(*types.Slice); !slice {
		return false
	}
	body := access.Block()
	header := body.Idom()
	if header == nil || len(body.Preds) != 1 || len(header.Succs) != 2 || header.Succs[0] != body {
		return false
	}
	branch, ok := header.Instrs[len(header.Instrs)-1].(*ssa.If)
	if !ok {
		return false
	}
	test, ok := branch.Cond.(*ssa.BinOp)
	if !ok || test.Op != token.LSS || test.X != access.Index {
		return false
	}
	limit, ok := test.Y.(*ssa.Call)
	if !ok || len(limit.Call.Args) != 1 || limit.Call.Args[0] != access.X {
		return false
	}
	builtin, ok := limit.Call.Value.(*ssa.Builtin)
	if !ok || builtin.Name() != "len" {
		return false
	}
	step, ok := access.Index.(*ssa.BinOp)
	if !ok || step.Op != token.ADD || step.Block() != header || !types.Identical(step.Type(), types.Typ[types.Int]) {
		return false
	}
	one, ok := step.Y.(*ssa.Const)
	if !ok || !constant.Compare(one.Value, token.EQL, constant.MakeInt64(1)) {
		return false
	}
	prev, ok := step.X.(*ssa.Phi)
	if !ok || prev.Block() != header {
		return false
	}
	for i, edge := range prev.Edges {
		if body.Dominates(header.Preds[i]) {
			if edge != step {
				return false
			}
		} else if start, ok := edge.(*ssa.Const); !ok ||
			!constant.Compare(start.Value, token.EQL, constant.MakeInt64(-1)) {
			return false
		}
	}
	return true
}
