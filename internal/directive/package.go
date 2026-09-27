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

package directive

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"
)

// Declaration is identified by its source node and package instance, never by
// its final linker symbol. Link/Export are declaration properties.
// Name uses the canonical receiver; Source retains aliases for directive matching.
type Declaration struct {
	Err         error
	Name        string
	Pos         token.Pos
	Linkname    string
	HasLinkname bool
	ExportName  string
}

// FunctionDecl collected for a package has a non-empty Declaration.Name.
// Standalone snapshots from Index contain only Source and Function, leaving
// Declaration zero-valued; their link names come from the legacy import index.
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
	aliases := receiverAliases(files)
	for _, file := range files {
		for _, decl := range file.Syntax.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				sourceName := FuncName(d)
				name := canonicalFuncName(d, aliases)
				rec := &FunctionDecl{Declaration: Declaration{Name: name, Pos: d.Pos()}, Source: d, Function: file.Functions[d]}
				rec.Err = associateLink(&rec.Declaration, file.Group(d.Doc), sourceName, exportRename)
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
				syms[sourceName] = &rec.Declaration
				if alias := ParenthesizedMethodName(sourceName); alias != "" {
					syms[alias] = &rec.Declaration
				}
			case *ast.GenDecl:
				switch d.Tok {
				case token.VAR:
					for _, spec := range d.Specs {
						for _, id := range spec.(*ast.ValueSpec).Names {
							rec := &VariableDecl{Declaration: Declaration{Name: id.Name, Pos: id.Pos()}, Source: id}
							if len(d.Specs) == 1 && len(spec.(*ast.ValueSpec).Names) == 1 {
								rec.Err = associateLink(&rec.Declaration, file.Group(d.Doc), id.Name, exportRename)
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
func associateLink(d *Declaration, g *Group, sourceName string, rename bool) error {
	l, ok, err := g.DeclarationLink(sourceName, rename)
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
		t := ast.Unparen(fn.Recv.List[0].Type)
		if p, ok := t.(*ast.StarExpr); ok {
			return "(*" + ReceiverName(p.X) + ")." + name
		}
		return ReceiverName(t) + "." + name
	}
	return name
}

// canonicalFuncName resolves local receiver aliases before type checking, while
// FuncName retains the spelling used to match a declaration's directives.
func canonicalFuncName(fn *ast.FuncDecl, aliases map[string]ast.Expr) string {
	if fn.Recv == nil || len(fn.Recv.List) != 1 {
		return fn.Name.Name
	}
	t := ast.Unparen(fn.Recv.List[0].Type)
	pointer := false
	if ptr, ok := t.(*ast.StarExpr); ok {
		t, pointer = ptr.X, true
	}
	name := ReceiverName(t)
	// Bound the walk so invalid alias cycles remain the type checker's concern.
	for n := 0; n < len(aliases); n++ {
		rhs, ok := aliases[name]
		if !ok {
			break
		}
		rhs = ast.Unparen(rhs)
		ptr, isPtr := rhs.(*ast.StarExpr)
		if isPtr {
			rhs = ast.Unparen(ptr.X)
		}
		ident, ok := rhs.(*ast.Ident)
		if !ok {
			break
		}
		name, pointer = ident.Name, pointer || isPtr
	}
	if pointer {
		name = "(*" + name + ")"
	}
	return name + "." + fn.Name.Name
}

func receiverAliases(files []*File) map[string]ast.Expr {
	aliases := make(map[string]ast.Expr)
	for _, file := range files {
		for _, decl := range file.Syntax.Decls {
			if decl, ok := decl.(*ast.GenDecl); ok && decl.Tok == token.TYPE {
				for _, spec := range decl.Specs {
					if spec := spec.(*ast.TypeSpec); spec.Assign.IsValid() {
						aliases[spec.Name.Name] = spec.Type
					}
				}
			}
		}
	}
	return aliases
}

// ValidateLinks diagnoses malformed or detached LLGo links and unresolved Go
// method links using prepared groups. Declaration names retain source spelling.
func (p *Package) ValidateLinks(fset *token.FileSet) error {
	syms := make(map[string]bool)
	attached := make(map[*ast.CommentGroup]bool)
	validate := func(doc *ast.CommentGroup, g *Group, name string) error {
		syms[name] = true
		if alias := ParenthesizedMethodName(name); alias != "" {
			syms[alias] = true
		}
		attached[doc] = true
		for i := len(g.Items) - 1; i >= 0; i-- {
			d := g.Items[i]
			if d.Name != "llgo:link" {
				continue
			}
			fields := strings.Fields(d.Args)
			if len(fields) < 2 {
				return fmt.Errorf("%s: //llgo:link requires a local name and a target", fset.Position(d.Pos))
			}
			if fields[0] != name && fields[0] != ParenthesizedMethodName(name) {
				return fmt.Errorf("%s: //llgo:link local name %q does not match declaration %q", fset.Position(d.Pos), fields[0], name)
			}
		}
		return nil
	}
	for _, file := range p.Files {
		for _, node := range file.Syntax.Decls {
			switch d := node.(type) {
			case *ast.FuncDecl:
				if err := validate(d.Doc, file.Group(d.Doc), FuncName(d)); err != nil {
					return err
				}
			case *ast.GenDecl:
				if d.Tok != token.VAR {
					continue
				}
				for _, spec := range d.Specs {
					for _, name := range spec.(*ast.ValueSpec).Names {
						syms[name.Name] = true
					}
				}
				if len(d.Specs) == 1 {
					if names := d.Specs[0].(*ast.ValueSpec).Names; len(names) == 1 {
						if err := validate(d.Doc, file.Group(d.Doc), names[0].Name); err != nil {
							return err
						}
					}
				}
			}
		}
	}
	for _, file := range p.Files {
		unsafe := false
		for _, imp := range file.Syntax.Imports {
			unsafe = unsafe || imp.Path.Value == `"unsafe"`
		}
		for _, doc := range file.Syntax.Comments {
			for _, d := range file.Group(doc).Items {
				if d.Name != "llgo:link" && (d.Name != "go:linkname" || !unsafe) {
					continue
				}
				fields := strings.Fields(d.Args)
				if d.Name == "llgo:link" {
					if len(fields) < 2 {
						return fmt.Errorf("%s: //llgo:link requires a local name and a target", fset.Position(d.Pos))
					}
					if !attached[doc] {
						return fmt.Errorf("%s: //llgo:link local name %q is not attached to a declaration", fset.Position(d.Pos), fields[0])
					}
				} else if len(fields) >= 2 && !syms[fields[0]] && strings.Contains(fields[0], ".") {
					return fmt.Errorf("%s: //go:linkname local method %q not found", fset.Position(d.Pos), fields[0])
				}
			}
		}
	}
	return nil
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
			t := ast.Unparen(d.Recv.List[0].Type)
			if ptr, ok := t.(*ast.StarExpr); ok {
				t = ptr.X
			}
			if recv := pkg.Scope().Lookup(ReceiverName(t)); recv != nil {
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
