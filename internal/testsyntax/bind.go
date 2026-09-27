package testsyntax

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/xgo-dev/llgo/internal/directive"
)

// BindScope binds records to a test importer's checked objects. Export data
// omits columns, so those positions are matched by file and line. Methods use
// the receiver's named origin, including receiver aliases.
func BindScope(p *directive.Package, fset *token.FileSet, pkg *types.Package) {
	if pkg == nil {
		return
	}
	samePosition := func(a, b token.Pos) bool {
		if a == token.NoPos || b == token.NoPos {
			return false
		}
		if a == b {
			return true
		}
		x, y := fset.Position(a), fset.Position(b)
		return x.IsValid() && y.IsValid() && x.Filename == y.Filename && x.Line == y.Line && (x.Column <= 1 || x.Column == y.Column)
	}
	for d, r := range p.Functions {
		if p.Names[r.Name] != r {
			continue
		}
		var obj types.Object
		if d.Recv == nil {
			obj = pkg.Scope().Lookup(d.Name.Name)
		} else if len(d.Recv.List) == 1 {
			t := ast.Unparen(d.Recv.List[0].Type)
			if ptr, ok := t.(*ast.StarExpr); ok {
				t = ptr.X
			}
			if recv := pkg.Scope().Lookup(directive.ReceiverName(t)); recv != nil {
				typ := types.Unalias(recv.Type())
				if ptr, ok := typ.(*types.Pointer); ok {
					typ = types.Unalias(ptr.Elem())
				}
				if named, ok := typ.(*types.Named); ok {
					for i := 0; i < named.NumMethods(); i++ {
						m := named.Method(i)
						if m.Name() == d.Name.Name {
							obj = m
							break
						}
					}
				}
			}
		}
		if obj != nil && samePosition(obj.Pos(), d.Name.Pos()) {
			p.Objects[obj] = r
		}
	}
	for id, r := range p.Variables {
		if p.Names[r.Name] != r {
			continue
		}
		if o := pkg.Scope().Lookup(id.Name); o != nil && samePosition(o.Pos(), id.Pos()) {
			p.Objects[o] = r
		}
	}
	for d, r := range p.Types {
		if p.Names[r.Name] != r {
			continue
		}
		if o := pkg.Scope().Lookup(d.Name.Name); o != nil && samePosition(o.Pos(), d.Name.Pos()) {
			p.Objects[o] = r
		}
	}
}
