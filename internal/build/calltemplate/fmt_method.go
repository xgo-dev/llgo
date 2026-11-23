package calltemplate

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/xgo-dev/llgo/internal/packages"
)

// Record methods whose return expression only reads values or combines
// strings. Calls, pointer dereferences and indexing keep fmt's panic handler.
func valueMethods(p *packages.Package) map[*types.Func]bool {
	safe := make(map[*types.Func]bool)
	for _, file := range p.Syntax {
		for _, decl := range file.Decls {
			method, ok := decl.(*ast.FuncDecl)
			if !ok || method.Recv == nil || method.Body == nil || len(method.Body.List) != 1 ||
				(method.Name.Name != "String" && method.Name.Name != "Error") {
				continue
			}
			fn, _ := p.TypesInfo.Defs[method.Name].(*types.Func)
			ret, returns := method.Body.List[0].(*ast.ReturnStmt)
			if fn == nil || !returns || len(ret.Results) != 1 {
				continue
			}
			sig := fn.Signature()
			if sig.Params().Len() == 0 && sig.Results().Len() == 1 && types.Identical(sig.Results().At(0).Type(), types.Typ[types.String]) &&
				safeStringExpression(p.TypesInfo, ret.Results[0]) {
				safe[fn] = true
			}
		}
	}
	return safe
}

func safeStringExpression(info *types.Info, expr ast.Expr) bool {
	if info.Types[expr].Value != nil {
		return true
	}
	switch expr := ast.Unparen(expr).(type) {
	case *ast.Ident:
		_, variable := info.Uses[expr].(*types.Var)
		return variable
	case *ast.SelectorExpr:
		if selection := info.Selections[expr]; selection != nil {
			return selection.Kind() == types.FieldVal && !selection.Indirect() && safeStringExpression(info, expr.X)
		}
		// A qualified package variable does not dereference a receiver.
		pkg, _ := expr.X.(*ast.Ident)
		_, imported := info.Uses[pkg].(*types.PkgName)
		_, variable := info.Uses[expr.Sel].(*types.Var)
		return imported && variable
	case *ast.BinaryExpr:
		if typ := info.TypeOf(expr); expr.Op == token.ADD && typ != nil {
			basic, _ := typ.Underlying().(*types.Basic)
			return basic != nil && basic.Kind() == types.String &&
				safeStringExpression(info, expr.X) && safeStringExpression(info, expr.Y)
		}
	}
	return false
}

// defaultValueMethod implements the exact %v method precedence. A promoted
// method or pointer receiver may dereference nil before its body starts.
func defaultValueMethod(t types.Type, safe map[*types.Func]bool) (string, bool) {
	methods := types.NewMethodSet(t)
	var selected *types.Selection
	for i := 0; i < methods.Len(); i++ {
		m := methods.At(i)
		switch m.Obj().Name() {
		case "Format":
			return "", false
		case "Error":
			selected = m
		case "String":
			if selected == nil {
				selected = m
			}
		}
	}
	if selected == nil {
		return "", true
	}
	_, pointer := t.Underlying().(*types.Pointer)
	fn, _ := selected.Obj().(*types.Func)
	if pointer || len(selected.Index()) != 1 || selected.Indirect() || fn == nil || !safe[fn.Origin()] {
		return "", false
	}
	return selected.Obj().Name(), true
}
