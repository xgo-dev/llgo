package directive

import (
	"go/ast"
	"go/token"
	"go/types"
)

// Declaration is identified by its source node and package instance, never by
// its final linker symbol. Link/Export are declaration properties.
type Declaration struct {
	Err         error
	Name        string
	Pos         token.Pos
	Linkname    string
	HasLinkname bool
	ExportName  string
}
type FunctionDecl struct {
	Declaration
	Source *ast.FuncDecl
	Function
}
type VariableDecl struct {
	Declaration
	Source *ast.Ident
}
type TypeDecl struct {
	Declaration
	Source     *ast.TypeSpec
	Background string
}

// Package is the authoritative source metadata for one types.Package. Object
// bindings are added after checking, before concurrent lowering starts.
type Package struct {
	Names     map[string]any
	Functions map[*ast.FuncDecl]*FunctionDecl
	Variables map[*ast.Ident]*VariableDecl
	Types     map[*ast.TypeSpec]*TypeDecl
	Objects   map[types.Object]any
	Files     []*File
	Skip      Skip
}

func Collect(files []*File, cPackage, exportRename bool) *Package {
	p := &Package{Names: make(map[string]any), Functions: make(map[*ast.FuncDecl]*FunctionDecl), Variables: make(map[*ast.Ident]*VariableDecl), Types: make(map[*ast.TypeSpec]*TypeDecl), Objects: make(map[types.Object]any), Files: files}
	syms := make(map[string]*Declaration)
	for _, file := range files {
		for _, decl := range file.Syntax.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				name := FuncName(d)
				rec := &FunctionDecl{Declaration: Declaration{Name: name, Pos: d.Pos()}, Source: d, Function: file.Functions[d]}
				rec.Err = associateLink(&rec.Declaration, file.Group(d.Doc), exportRename)
				if !rec.HasLinkname && cPackage && d.Recv == nil && token.IsExported(name) {
					rec.HasLinkname = true
					rec.Linkname = name
					if name[0] == 'X' {
						rec.Linkname = name[1:]
					}
					rec.ExportName = rec.Linkname
				}
				p.Functions[d] = rec
				p.Names[name] = rec
				syms[name] = &rec.Declaration
			case *ast.GenDecl:
				switch d.Tok {
				case token.VAR:
					for _, spec := range d.Specs {
						for _, id := range spec.(*ast.ValueSpec).Names {
							rec := &VariableDecl{Declaration: Declaration{Name: id.Name, Pos: id.Pos()}, Source: id}
							if len(d.Specs) == 1 && len(spec.(*ast.ValueSpec).Names) == 1 {
								rec.Err = associateLink(&rec.Declaration, file.Group(d.Doc), exportRename)
							}
							p.Variables[id] = rec
							p.Names[id.Name] = rec
							syms[id.Name] = &rec.Declaration
						}
					}
				case token.TYPE, token.CONST:
					skip := file.Group(d.Doc).Skip
					p.Skip.All = p.Skip.All || skip.All
					p.Skip.Names = append(p.Skip.Names, skip.Names...)
					if d.Tok == token.TYPE {
						for _, spec := range d.Specs {
							spec := spec.(*ast.TypeSpec)
							bg := ""
							if len(d.Specs) == 1 {
								bg = file.Group(d.Doc).TypeBackground
							}
							p.Types[spec] = &TypeDecl{Declaration: Declaration{Name: spec.Name.Name, Pos: spec.Pos()}, Source: spec, Background: bg}
							p.Names[spec.Name.Name] = p.Types[spec]
						}
					}
				case token.IMPORT:
					g := file.Group(d.Doc)
					// The deprecated import spelling consumes only the last comment.
					if g.LastSkip {
						all, names, _ := LegacySkip(g.LastLine)
						p.Skip.All = p.Skip.All || all
						p.Skip.Names = append(p.Skip.Names, names...)
					}
				}
			}
		}
	}
	// The existing package-wide link pass runs after attached declarations, and
	// only scans files importing unsafe. Keep that ordering and scope exactly.
	for _, file := range files {
		unsafe := false
		for _, imp := range file.Syntax.Imports {
			if imp.Path.Value == `"unsafe"` {
				unsafe = true
				break
			}
		}
		if unsafe {
			for _, l := range file.GoLinks {
				if d := syms[l.Local]; d != nil {
					d.Linkname = l.Target
					d.HasLinkname = true
				}
			}
		}
	}
	return p
}
func associateLink(d *Declaration, g *Group, rename bool) error {
	l, ok, err := g.DeclarationLink(d.Name, rename)
	if err != nil {
		return err
	}
	if ok {
		d.Linkname = l.Target
		d.HasLinkname = true
		if l.Export {
			d.ExportName = l.Target
		}
	}
	return nil
}

// Bind uses checker object identity, including receiver aliases and generic
// origins. Call only during package preparation, before publishing to workers.
func (p *Package) Bind(info *types.Info) {
	if info == nil {
		return
	}
	for d, r := range p.Functions {
		if p.Names[r.Name] != r {
			continue
		}
		if o := info.Defs[d.Name]; o != nil {
			p.Objects[o] = r
		}
	}
	for id, r := range p.Variables {
		if p.Names[r.Name] != r {
			continue
		}
		if o := info.Defs[id]; o != nil {
			p.Objects[o] = r
		}
	}
	for d, r := range p.Types {
		if p.Names[r.Name] != r {
			continue
		}
		if o := info.Defs[d.Name]; o != nil {
			p.Objects[o] = r
		}
	}
}

// FuncName is a source selector, not a linker name. It is also used by patches.
func FuncName(fn *ast.FuncDecl) string {
	name := fn.Name.Name
	if fn.Recv != nil && len(fn.Recv.List) == 1 {
		t := fn.Recv.List[0].Type
		if p, ok := t.(*ast.StarExpr); ok {
			return "(*" + ReceiverName(p.X) + ")." + name
		}
		return ReceiverName(t) + "." + name
	}
	return name
}
func ReceiverName(t ast.Expr) string {
	switch t := t.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		return ReceiverName(t.X)
	case *ast.IndexListExpr:
		return ReceiverName(t.X)
	case *ast.ParenExpr:
		return ReceiverName(t.X)
	}
	panic("unreachable")
}

// BindScope is for standalone compiler entrypoints that receive checked SSA
// without types.Info. Checked objects are matched by declaration position;
// methods use the receiver's named origin, including receiver aliases.
func (p *Package) BindScope(pkg *types.Package) {
	if pkg == nil {
		return
	}
	for d, r := range p.Functions {
		if p.Names[r.Name] != r {
			continue
		}
		var obj types.Object
		if d.Recv == nil {
			obj = pkg.Scope().Lookup(d.Name.Name)
		} else if len(d.Recv.List) == 1 {
			t := d.Recv.List[0].Type
			if ptr, ok := t.(*ast.StarExpr); ok {
				t = ptr.X
			}
			if recv := pkg.Scope().Lookup(ReceiverName(t)); recv != nil {
				receiver := types.Unalias(recv.Type())
				if ptr, ok := receiver.(*types.Pointer); ok {
					receiver = types.Unalias(ptr.Elem())
				}
				if named, ok := receiver.(*types.Named); ok {
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
		if obj != nil && obj.Pos() == d.Name.Pos() {
			p.Objects[obj] = r
		}
	}
	for id, r := range p.Variables {
		if p.Names[r.Name] != r {
			continue
		}
		if o := pkg.Scope().Lookup(id.Name); o != nil && o.Pos() == id.Pos() {
			p.Objects[o] = r
		}
	}
	for d, r := range p.Types {
		if p.Names[r.Name] != r {
			continue
		}
		if o := pkg.Scope().Lookup(d.Name.Name); o != nil && o.Pos() == d.Name.Pos() {
			p.Objects[o] = r
		}
	}
}
