package cl

import (
	"go/types"

	"github.com/xgo-dev/llgo/cmd/compiler/llvmgen"
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

func collectDebugAllocObjects(variables map[*ssa.Alloc]*types.Var) map[*types.Var]bool {
	objects := make(map[*types.Var]bool, len(variables))
	for _, variable := range variables {
		objects[variable] = true
	}
	return objects
}

func (p *context) debugAlloc(b llvmgen.Builder, alloc *ssa.Alloc, addr llvmgen.Expr) {
	variable := p.debugAllocVars[alloc]
	if variable == nil {
		return
	}
	pos := p.goProg.Fset.Position(variable.Pos())
	var dbgVar llvmgen.DIVar
	if argNo := debugParameterArgNo(p.goFn, variable); argNo != 0 {
		dbgVar = b.DIVarParam(p.fn, pos, variable.Name(), p.type_(variable.Type(), llvmgen.GoDecl), argNo)
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
