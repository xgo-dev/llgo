package cl

import (
	"go/token"
	"go/types"

	"golang.org/x/tools/go/ssa"
)

func debugFunctionScope(fn *ssa.Function) *types.Scope {
	if fn == nil {
		return nil
	}
	if object, ok := fn.Object().(*types.Func); ok && object.Scope() != nil {
		return object.Scope()
	}
	for _, param := range fn.Params {
		if object := param.Object(); object != nil && object.Parent() != nil {
			return object.Parent()
		}
	}

	syntax := fn.Syntax()
	if syntax == nil {
		return nil
	}
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			ref, ok := instr.(*ssa.DebugRef)
			if !ok {
				continue
			}
			variable, ok := ref.Object().(*types.Var)
			if !ok || variable.Parent() == nil {
				continue
			}
			return outermostScopeWithin(variable.Parent(), syntax.Pos(), syntax.End())
		}
	}
	return nil
}

func outermostScopeWithin(scope *types.Scope, start, end token.Pos) *types.Scope {
	result := scope
	for parent := scope.Parent(); parent != nil; parent = parent.Parent() {
		if parent.Pos() == token.NoPos || parent.End() == token.NoPos ||
			parent.Pos() < start || parent.End() > end {
			break
		}
		result = parent
	}
	return result
}
