package cl

import (
	"go/types"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func collectDebugAllocVariables(fn *ssa.Function) map[*ssa.Alloc]*types.Var {
	variables := make(map[*ssa.Alloc]*types.Var)
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			ref, ok := instr.(*ssa.DebugRef)
			if !ok || !ref.IsAddr {
				continue
			}
			alloc, ok := ref.X.(*ssa.Alloc)
			if !ok || alloc.Parent() != fn {
				continue
			}
			variable, ok := ref.Object().(*types.Var)
			if ok && !variable.IsField() {
				variables[alloc] = variable
			}
		}
	}
	return variables
}

func hasDebugAlloc(variables map[*ssa.Alloc]*types.Var, variable *types.Var) bool {
	for _, candidate := range variables {
		if candidate == variable {
			return true
		}
	}
	return false
}

func (p *context) debugAlloc(b llssa.Builder, alloc *ssa.Alloc, addr llssa.Expr) {
	variable := p.debugAllocVars[alloc]
	if variable == nil {
		return
	}
	pos := p.goProg.Fset.Position(variable.Pos())
	var dbgVar llssa.DIVar
	if argNo := debugParameterArgNo(p.goFn, variable); argNo != 0 {
		dbgVar = b.DIVarParam(p.fn, pos, variable.Name(), p.type_(variable.Type(), llssa.InGo), argNo)
		p.debugDIVars[variable] = dbgVar
	} else {
		dbgVar = p.getLocalVariable(b, p.goFn, variable)
	}
	diScope := b.DIScope(p.fn, variable.Parent())
	b.DIDeclare(variable, addr, dbgVar, diScope, pos, b.Func.Block(alloc.Block().Index))
}

func debugParameterArgNo(fn *ssa.Function, variable *types.Var) int {
	for i, param := range fn.Params {
		if param.Object() == variable {
			return i + 1
		}
	}
	return 0
}
