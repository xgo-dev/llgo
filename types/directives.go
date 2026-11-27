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

package types

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/xgo-dev/llgo/directive"
)

// applyDeclarationDirectives attaches LLGo and Go declaration directives from
// files directly onto pkg, its package-level *Func/*Var objects, and methods
// on declared *Named types.
func applyDeclarationDirectives(pkg *Package, info *Info, files []*ast.File) {
	if pkg == nil || len(files) == 0 {
		return
	}
	cPkg := pkg.IsCPackage()
	aliases := collectLocalTypeAliases(files)
	syms := make(map[string]Object)

	for _, file := range files {
		if file == nil {
			continue
		}
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Name == nil || d.Name.Name == "_" {
					continue
				}
				fn := resolveDeclFunc(pkg, info, d, aliases)
				if fn == nil {
					continue
				}
				inPkgName := declFuncInPkgName(d, aliases)
				if inPkgName != "" {
					syms[inPkgName] = fn
					if alt := parenthesizedMethodName(inPkgName); alt != "" {
						syms[alt] = fn
					}
				}
				rawInPkgName := declFuncInPkgName(d, nil)
				if rawInPkgName != "" && rawInPkgName != inPkgName {
					syms[rawInPkgName] = fn
					if alt := parenthesizedMethodName(rawInPkgName); alt != "" {
						syms[alt] = fn
					}
				}
				applyFuncDirectives(fn, d.Doc, inPkgName, rawInPkgName, d.Recv == nil, cPkg)

			case *ast.GenDecl:
				switch d.Tok {
				case token.VAR:
					for _, spec := range d.Specs {
						vs, ok := spec.(*ast.ValueSpec)
						if !ok {
							continue
						}
						loc := parseVarLocality(d.Doc, vs.Doc, len(d.Specs) == 1)
						for _, id := range vs.Names {
							if id == nil || id.Name == "_" {
								continue
							}
							v := resolveDeclVar(pkg, info, id)
							if v == nil {
								continue
							}
							syms[id.Name] = v
							if loc != directive.LocalityNone {
								v.SetLocality(loc)
							}
							if len(vs.Names) == 1 && vs.Doc != nil {
								applyVarLinkDirectives(v, vs.Doc, id.Name)
							}
						}
					}
					if len(d.Specs) == 1 {
						if vs, ok := d.Specs[0].(*ast.ValueSpec); ok && len(vs.Names) == 1 && vs.Names[0] != nil {
							if v := resolveDeclVar(pkg, info, vs.Names[0]); v != nil && v.LinkName() == "" {
								applyVarLinkDirectives(v, d.Doc, vs.Names[0].Name)
							}
						}
					}
				case token.CONST, token.TYPE:
					applySkipDirectives(pkg, d.Doc)
				case token.IMPORT:
					if d.Doc != nil && len(d.Doc.List) > 0 {
						applySkipLine(pkg, d.Doc.List[len(d.Doc.List)-1].Text)
					}
				}
			}
		}
	}

	// Package-scoped //go:linkname directives in files that import "unsafe".
	for _, file := range files {
		if file == nil {
			continue
		}
		importsUnsafe := false
		for _, imp := range file.Imports {
			if imp != nil && imp.Path != nil && imp.Path.Value == `"unsafe"` {
				importsUnsafe = true
				break
			}
		}
		if !importsUnsafe {
			continue
		}
		for _, group := range file.Comments {
			for _, item := range directive.ParseGroup(group) {
				if item.Name != "go:linkname" {
					continue
				}
				local, target := directive.ParseLinknameArgs(item.Args)
				if target == "" {
					continue
				}
				switch obj := syms[local].(type) {
				case *Func:
					obj.SetLinkName(target)
				case *Var:
					obj.SetLinkName(target)
				}
			}
		}
	}
}

func resolveDeclFunc(pkg *Package, info *Info, decl *ast.FuncDecl, aliases map[string]ast.Expr) *Func {
	if info != nil && info.Defs != nil {
		if fn, ok := info.Defs[decl.Name].(*Func); ok {
			return fn
		}
	}
	if decl.Recv == nil {
		if pkg.scope != nil {
			if fn, ok := pkg.scope.Lookup(decl.Name.Name).(*Func); ok {
				return fn
			}
		}
		return nil
	}
	if len(decl.Recv.List) != 1 || pkg.scope == nil {
		return nil
	}
	recvName, _, ok := recvBaseTypeName(decl.Recv.List[0].Type)
	if !ok {
		return nil
	}
	canonical := recvName
	for n := 0; n < len(aliases); n++ {
		rhs, found := aliases[canonical]
		if !found {
			break
		}
		rhs = unparenExpr(rhs)
		if star, isStar := rhs.(*ast.StarExpr); isStar {
			rhs = unparenExpr(star.X)
		}
		id, ok := rhs.(*ast.Ident)
		if !ok {
			break
		}
		canonical = id.Name
	}
	tn, ok := pkg.scope.Lookup(canonical).(*TypeName)
	if !ok || tn == nil {
		return nil
	}
	named, ok := Unalias(tn.Type()).(*Named)
	if !ok || named == nil {
		return nil
	}
	for i := 0; i < named.NumMethods(); i++ {
		m := named.Method(i)
		if m != nil && m.Name() == decl.Name.Name {
			return m
		}
	}
	return nil
}

func resolveDeclVar(pkg *Package, info *Info, id *ast.Ident) *Var {
	if info != nil && info.Defs != nil {
		if v, ok := info.Defs[id].(*Var); ok {
			return v
		}
	}
	if pkg.scope != nil {
		if v, ok := pkg.scope.Lookup(id.Name).(*Var); ok {
			return v
		}
	}
	return nil
}

func applyFuncDirectives(fn *Func, doc *ast.CommentGroup, inPkgName, rawInPkgName string, isTopLevel, cPkg bool) {
	if doc != nil {
		linkCollected := false
		directives := directive.ParseGroup(doc)
		for i := len(directives) - 1; i >= 0; i-- {
			item := directives[i]
			switch item.Name {
			case "go:linkname", "llgo:link":
				if linkCollected {
					continue
				}
				local, target := directive.ParseLinknameArgs(item.Args)
				if target != "" && (matchesFuncLocalName(local, inPkgName) || matchesFuncLocalName(local, rawInPkgName)) {
					fn.SetLinkName(target)
					linkCollected = true
				}
			case "export":
				if linkCollected || item.Args == "" {
					continue
				}
				fn.SetLinkName(item.Args)
				fn.SetExportName(item.Args)
				linkCollected = true
			}
		}
		if directive.HasClosureEnv(doc) {
			fn.SetHasClosureEnv(true)
		}
		if directive.HasNoInline(doc) {
			fn.SetNoInline(true)
		}
		if directive.HasNoInterface(doc) {
			fn.SetNoInterface(true)
		}
		if directive.HasNoEscape(doc) {
			fn.SetNoEscape(true)
		}
		if wi, ok, err := directive.ParseWasmImport(doc); ok && err == nil {
			fn.SetWasmImport(&wi)
		}
		if name, _, ok := directive.ParseWasmExport(doc); ok {
			fn.SetWasmExport(name)
		}
	}
	if fn.LinkName() == "" && cPkg && isTopLevel && token.IsExported(inPkgName) {
		exportName := strings.TrimPrefix(inPkgName, "X")
		fn.SetLinkName(exportName)
		fn.SetExportName(exportName)
	}
}

func applyVarLinkDirectives(v *Var, doc *ast.CommentGroup, inPkgName string) {
	if doc == nil {
		return
	}
	directives := directive.ParseGroup(doc)
	for i := len(directives) - 1; i >= 0; i-- {
		item := directives[i]
		switch item.Name {
		case "go:linkname", "llgo:link":
			local, target := directive.ParseLinknameArgs(item.Args)
			if target != "" && local == inPkgName {
				v.SetLinkName(target)
				return
			}
		case "export":
			if item.Args != "" {
				v.SetLinkName(item.Args)
				v.SetExportName(item.Args)
				return
			}
		}
	}
}

func parseVarLocality(declDoc, specDoc *ast.CommentGroup, singleSpec bool) directive.Locality {
	if loc, _, err := directive.ParseLocality(specDoc); err == nil && loc != directive.LocalityNone {
		return loc
	}
	if singleSpec {
		if loc, _, err := directive.ParseLocality(declDoc); err == nil && loc != directive.LocalityNone {
			return loc
		}
	}
	return directive.LocalityNone
}

func applySkipDirectives(pkg *Package, doc *ast.CommentGroup) {
	if doc == nil {
		return
	}
	for i := len(doc.List) - 1; i >= 0; i-- {
		if !applySkipLine(pkg, doc.List[i].Text) {
			break
		}
	}
}

func applySkipLine(pkg *Package, line string) bool {
	d, ok := directive.Parse(&ast.Comment{Text: line})
	if !ok || (!strings.HasPrefix(d.Name, "go:") && !strings.HasPrefix(d.Name, "llgo:")) {
		return false
	}
	switch d.Name {
	case "llgo:skipall":
		if d.Args == "" {
			pkg.SetSkipAll(true)
		}
	case "llgo:skip":
		pkg.AddSkip(strings.Fields(d.Args)...)
	}
	return true
}

func collectLocalTypeAliases(files []*ast.File) map[string]ast.Expr {
	aliases := make(map[string]ast.Expr)
	for _, file := range files {
		if file == nil {
			continue
		}
		for _, decl := range file.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if ok && ts.Assign.IsValid() && ts.Name != nil {
					aliases[ts.Name.Name] = ts.Type
				}
			}
		}
	}
	return aliases
}

func recvBaseTypeName(typ ast.Expr) (name string, pointer, ok bool) {
	switch t := typ.(type) {
	case *ast.Ident:
		return t.Name, false, true
	case *ast.IndexExpr:
		return recvBaseTypeName(t.X)
	case *ast.IndexListExpr:
		return recvBaseTypeName(t.X)
	case *ast.ParenExpr:
		return recvBaseTypeName(t.X)
	case *ast.StarExpr:
		name, nestedPointer, ok := recvBaseTypeName(t.X)
		if !ok || nestedPointer {
			return "", false, false
		}
		return name, true, true
	}
	return "", false, false
}

func matchesFuncLocalName(local, inPkgName string) bool {
	if inPkgName == "" {
		return false
	}
	return local == inPkgName || local == parenthesizedMethodName(inPkgName)
}

func declFuncInPkgName(decl *ast.FuncDecl, aliases map[string]ast.Expr) string {
	if decl == nil || decl.Name == nil {
		return ""
	}
	name := decl.Name.Name
	if decl.Recv == nil || len(decl.Recv.List) != 1 {
		return name
	}
	recvName, pointer, ok := recvBaseTypeName(decl.Recv.List[0].Type)
	if !ok {
		return ""
	}
	if len(aliases) > 0 {
		canonical := recvName
		for n := 0; n < len(aliases); n++ {
			rhs, found := aliases[canonical]
			if !found {
				break
			}
			rhs = unparenExpr(rhs)
			if star, isStar := rhs.(*ast.StarExpr); isStar {
				pointer = true
				rhs = unparenExpr(star.X)
			}
			id, ok := rhs.(*ast.Ident)
			if !ok {
				break
			}
			canonical = id.Name
		}
		recvName = canonical
	}
	if pointer {
		return "(*" + recvName + ")." + name
	}
	return recvName + "." + name
}

func parenthesizedMethodName(name string) string {
	if i := strings.IndexByte(name, '.'); i > 0 && name[0] != '(' {
		return "(" + name[:i] + ")" + name[i:]
	}
	return ""
}

func unparenExpr(e ast.Expr) ast.Expr {
	for {
		p, ok := e.(*ast.ParenExpr)
		if !ok {
			return e
		}
		e = p.X
	}
}

func embeddedStructFieldPos(expr ast.Expr) token.Pos {
	switch expr := expr.(type) {
	case *ast.StarExpr:
		return embeddedStructFieldPos(expr.X)
	case *ast.SelectorExpr:
		return expr.Sel.Pos()
	case *ast.IndexExpr:
		return embeddedStructFieldPos(expr.X)
	case *ast.IndexListExpr:
		return embeddedStructFieldPos(expr.X)
	}
	return expr.Pos()
}
