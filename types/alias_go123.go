//go:build go1.23

package types

import (
	"go/ast"
	"go/parser"
	"go/token"
	gotypes "go/types"
	"sync"
)

func unaliasType(t gotypes.Type) gotypes.Type {
	return gotypes.Unalias(t)
}

func aliasObjAndRhs(t gotypes.Type) (*gotypes.TypeName, gotypes.Type, bool) {
	if a, ok := t.(*gotypes.Alias); ok {
		return a.Obj(), a.Rhs(), true
	}
	return nil, nil, false
}

func aliasGenericInfo(t gotypes.Type) (origin gotypes.Type, tparams *gotypes.TypeParamList, targs *gotypes.TypeList) {
	if a, ok := t.(*gotypes.Alias); ok {
		if orig := a.Origin(); orig != nil && orig != a {
			origin = orig
		}
		return origin, a.TypeParams(), a.TypeArgs()
	}
	return nil, nil, nil
}

func newGoAlias(obj *gotypes.TypeName, rhs gotypes.Type, tps []*gotypes.TypeParam) gotypes.Type {
	a := gotypes.NewAlias(obj, rhs)
	if len(tps) > 0 {
		a.SetTypeParams(tps)
	}
	return a
}

var (
	goTypesErasesAliasesOnce sync.Once
	goTypesErasesAliasesVal  bool
)

func goTypesErasesAliases() bool {
	goTypesErasesAliasesOnce.Do(func() {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "aliasprobe.go", "package p\ntype a = int\n", 0)
		if err != nil {
			return
		}
		var cfg gotypes.Config
		pkg, err := cfg.Check("p", fset, []*ast.File{f}, nil)
		if err != nil || pkg == nil {
			return
		}
		if obj := pkg.Scope().Lookup("a"); obj != nil {
			_, isAlias := obj.Type().(*gotypes.Alias)
			goTypesErasesAliasesVal = !isAlias
		}
	})
	return goTypesErasesAliasesVal
}
