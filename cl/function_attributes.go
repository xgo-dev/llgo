package cl

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/xgo-dev/llgo/internal/directive"
	"github.com/xgo-dev/llgo/internal/funcattrs"
)

// The syntax preload can use an export-data-only package, where private
// functions are absent. Their contracts are still parsed and are type checked
// when a definition or caller-side declaration is lowered.
func sourceAttributeFunction(pkg *types.Package, decl *ast.FuncDecl) *types.Func {
	if decl.Recv == nil {
		fn, _ := pkg.Scope().Lookup(decl.Name.Name).(*types.Func)
		return fn
	}
	if len(decl.Recv.List) != 1 {
		return nil
	}
	recv := decl.Recv.List[0].Type
	if p, ok := recv.(*ast.StarExpr); ok {
		recv = p.X
	}
	obj, ok := pkg.Scope().Lookup(recvTypeName(recv)).(*types.TypeName)
	if !ok {
		return nil
	}
	named, ok := types.Unalias(obj.Type()).(*types.Named)
	if !ok {
		return nil
	}
	for i := 0; i < named.NumMethods(); i++ {
		if method := named.Method(i); method.Name() == decl.Name.Name {
			return method
		}
	}
	return nil
}

func validateAttributePlacement(fset *token.FileSet, file *ast.File) error {
	allowed := make(map[token.Pos]bool)
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Doc != nil {
			for _, c := range fn.Doc.List {
				allowed[c.Pos()] = true
			}
		}
	}
	for _, group := range file.Comments {
		for _, d := range directive.ParseGroup(group) {
			if d.Name == "llgo:attribute" {
				return (funcattrs.Attribute{Position: fset.Position(d.Pos)}).Error("use //llgo:attr")
			}
			if d.Name == "llgo:attr" && !allowed[d.Pos] {
				return (funcattrs.Attribute{Position: fset.Position(d.Pos)}).Error("requires a named function or method declaration")
			}
		}
	}
	return nil
}
