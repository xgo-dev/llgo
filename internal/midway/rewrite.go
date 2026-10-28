// Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by an Apache 2.0 license.

package midway

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/xgo-dev/llgo/internal/directive"

	"golang.org/x/tools/go/ast/astutil"
)

// Rewrite replaces portable operations in dependent declarations with the
// official bridge's emulated and SIMD128 representations. Original types and
// signatures remain available to the initial check of importing packages.
// Imports reports additional dependencies that the loader must make available
// to the second type check; their packages already occur below simd in the graph.
// Rewriting mutates files in place. On error, callers must treat both syntax
// and type information as invalid; the loader marks the package IllTyped and
// excludes it from its cache before any downstream compilation.
func Rewrite(fset *token.FileSet, pkg *types.Package, info *types.Info, files []*ast.File, index *directive.Index) (changed bool, imports []string, err error) {
	if !importsSIMD(pkg, make(map[*types.Package]bool)) || pkg.Path() == bridgePath || pkg.Path() == "simd/archsimd" {
		return false, nil, nil
	}
	// Separate RHS expressions are separate initialization units. Preserve
	// them before recording dependency reads; a multi-result RHS stays grouped.
	for _, file := range files {
		for _, decl := range file.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.VAR {
				continue
			}
			var specs []ast.Spec
			for _, spec := range gd.Specs {
				vs := spec.(*ast.ValueSpec)
				if len(vs.Names) > 1 && len(vs.Values) == len(vs.Names) {
					for i, name := range vs.Names {
						copy := *vs
						copy.Names = []*ast.Ident{name}
						copy.Values = []ast.Expr{vs.Values[i]}
						specs = append(specs, &copy)
					}
				} else {
					specs = append(specs, spec)
				}
			}
			gd.Specs = specs
		}
	}
	a := analyze(pkg, info, files)
	if err := a.validateGlobalBindings(fset, index.Files(files)); err != nil {
		return false, nil, err
	}
	r := &rewriter{directives: index, fset: fset, a: a, imports: make(map[string]bool), dependencies: make(map[ast.Node][]*types.Var)}
	// Record initialization edges before any original body is replaced.
	for _, file := range files {
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Name.Name != "_" && a.dependent(info.Defs[d.Name]) && d.Body != nil {
					r.dependencies[d] = a.globalDependencies(d.Body)
				}
			case *ast.GenDecl:
				if d.Tok != token.VAR {
					continue
				}
				for _, spec := range d.Specs {
					if vs, ok := spec.(*ast.ValueSpec); ok {
						r.dependencies[vs] = a.globalDependencies(vs)
					}
				}
			}
		}
	}
	for _, file := range files {
		fr := r.file(file)
		fileChanged := false
		var decls []ast.Decl
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				obj, _ := info.Defs[d.Name].(*types.Func)
				if d.Name.Name == "_" || !a.dependent(obj) {
					decls = append(decls, d)
					continue
				}
				if pkg.Path() == simdPath {
					// Base methods are supplied by bridge. Only the public
					// FromArch/ToArch generic functions need specialized Go bodies.
					if d.Recv == nil && strings.HasPrefix(filepath.Base(fset.Position(d.Pos()).Filename), "tofrom_") {
						for _, width := range widths {
							decls = append(decls, fr.specialize(d, width).(*ast.FuncDecl))
						}
					}
					d.Body = fr.panicBody(d.Pos())
					decls = append(decls, d)
					changed = true
					fileChanged = true
					continue
				}
				if d.Body == nil {
					return false, nil, fmt.Errorf("%s: cannot specialize bodyless SIMD-dependent function %s", fset.Position(d.Pos()), d.Name.Name)
				}
				for _, width := range widths {
					decls = append(decls, fr.specialize(d, width).(*ast.FuncDecl))
				}
				if a.types[obj.Type()] {
					d.Body = fr.panicBody(d.Pos())
				} else {
					d.Body = fr.dispatch(d)
				}
				if vars := r.dependencies[d]; len(vars) != 0 {
					// A separate declaration avoids parameter/local shadowing
					// of the original global names. A function-value reference
					// preserves go/types' init dependency without a runtime call.
					name := "@midwayDeps" + strconv.Itoa(int(d.Pos()))
					decls = append(decls, &ast.FuncDecl{Name: ident(name, d.Pos()), Type: &ast.FuncType{
						Func: d.Pos(), Params: &ast.FieldList{Opening: d.Pos(), Closing: d.Pos()},
					}, Body: &ast.BlockStmt{Lbrace: d.Pos(), Rbrace: d.Pos(), List: dependencyReads(vars, d.Pos())}})
					d.Body.List = append([]ast.Stmt{&ast.AssignStmt{TokPos: d.Pos(), Tok: token.ASSIGN,
						Lhs: []ast.Expr{ident("_", d.Pos())}, Rhs: []ast.Expr{ident(name, d.Pos())}}}, d.Body.List...)
				}
				decls = append(decls, d)
				changed = true
				fileChanged = true
			case *ast.GenDecl:
				decls = append(decls, d)
				if pkg.Path() == simdPath {
					continue
				}
				if d.Tok == token.CONST && a.bodyDependent(d) {
					for _, width := range widths {
						decls = append(decls, fr.specialize(d, width).(*ast.GenDecl))
					}
					changed, fileChanged = true, true
					continue
				}
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if !a.dependent(info.Defs[s.Name]) {
							continue
						}
						for _, width := range widths {
							copy := fr.specialize(s, width).(*ast.TypeSpec)
							decls = append(decls, &ast.GenDecl{TokPos: d.TokPos, Tok: token.TYPE, Specs: []ast.Spec{copy}, Doc: d.Doc})
						}
						changed = true
						fileChanged = true
					case *ast.ValueSpec:
						if d.Tok == token.VAR {
							extra, rewritten := fr.global(d, s)
							decls = append(decls, extra...)
							changed = changed || rewritten
							fileChanged = fileChanged || rewritten
						}
					}
				}
			default:
				decls = append(decls, d)
			}
			if r.err != nil {
				return false, nil, r.err
			}
		}
		file.Decls = decls
		if fileChanged {
			fr.finish()
		}
	}
	for path := range r.imports {
		imports = append(imports, path)
	}
	slices.Sort(imports)
	return changed, imports, nil
}

func importsSIMD(pkg *types.Package, seen map[*types.Package]bool) bool {
	if pkg == nil || seen[pkg] {
		return false
	}
	seen[pkg] = true
	if pkg.Path() == simdPath {
		return true
	}
	for _, imp := range pkg.Imports() {
		if importsSIMD(imp, seen) {
			return true
		}
	}
	return false
}

type rewriter struct {
	directives   *directive.Index
	fset         *token.FileSet
	a            *analysis
	imports      map[string]bool
	dependencies map[ast.Node][]*types.Var
	err          error
}

type fileRewriter struct {
	*rewriter
	file    *ast.File
	aliases map[string]string
	added   []*ast.ImportSpec
	used    map[string]bool
}

func (r *rewriter) file(file *ast.File) *fileRewriter {
	f := &fileRewriter{rewriter: r, file: file, aliases: make(map[string]string), used: make(map[string]bool)}
	// Index definitions while visiting this file once, instead of scanning
	// package-wide Defs again for every import. PkgNames are the imports
	// themselves, not declarations that shadow them.
	defined := make(map[string]bool)
	ast.Inspect(file, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			f.used[id.Name] = true
			if obj, ok := r.a.info.Defs[id]; ok {
				if _, imported := obj.(*types.PkgName); !imported {
					defined[id.Name] = true
				}
			}
		}
		return true
	})
	for _, imp := range file.Imports {
		obj, _ := r.a.info.Implicits[imp].(*types.PkgName)
		if imp.Name != nil {
			obj, _ = r.a.info.Defs[imp.Name].(*types.PkgName)
		}
		if obj == nil || obj.Name() == "_" || obj.Name() == "." {
			continue
		}
		if !defined[obj.Name()] {
			f.aliases[obj.Imported().Path()] = obj.Name()
		}
	}
	return f
}

func (f *fileRewriter) qualify(path, name string, pos token.Pos) ast.Expr {
	if path == f.a.pkg.Path() {
		return ident(name, pos)
	}
	alias := f.aliases[path]
	if alias == "" {
		alias = "llgoMidway" + strconv.Itoa(len(f.aliases))
		for f.used[alias] || f.a.pkg.Scope().Lookup(alias) != nil {
			alias += "_"
		}
		f.used[alias] = true
		f.aliases[path] = alias
		f.added = append(f.added, &ast.ImportSpec{Name: ident(alias, pos), Path: &ast.BasicLit{ValuePos: pos, Kind: token.STRING, Value: strconv.Quote(path)}})
		f.imports[path] = true
	}
	return &ast.SelectorExpr{X: ident(alias, pos), Sel: ident(name, pos)}
}

func (f *fileRewriter) finish() {
	if len(f.added) != 0 {
		specs := make([]ast.Spec, len(f.added))
		for i, spec := range f.added {
			specs[i] = spec
		}
		f.file.Decls = append([]ast.Decl{&ast.GenDecl{TokPos: f.file.Package, Tok: token.IMPORT, Specs: specs}}, f.file.Decls...)
		f.file.Imports = append(f.file.Imports, f.added...)
	}
	// Specialization can remove the only use of the original simd import.
	// Preserve the import and its initialization edge without calling anything.
	for _, imp := range f.file.Imports {
		path, _ := strconv.Unquote(imp.Path.Value)
		if path != simdPath || imp.Name != nil && imp.Name.Name == "_" {
			continue
		}
		alias := "simd"
		if imp.Name != nil {
			alias = imp.Name.Name
		}
		var expr ast.Expr = ident("VectorBitSize", imp.Pos())
		if alias != "." {
			expr = &ast.SelectorExpr{X: ident(alias, imp.Pos()), Sel: ident("VectorBitSize", imp.Pos())}
		}
		f.file.Decls = append(f.file.Decls, &ast.GenDecl{TokPos: imp.Pos(), Tok: token.VAR, Specs: []ast.Spec{
			&ast.ValueSpec{Names: []*ast.Ident{ident("_", imp.Pos())}, Values: []ast.Expr{expr}},
		}})
	}
}

func bridgeName(name string, width int) (string, bool) {
	prefix, tail := "", ""
	if strings.HasPrefix(name, "Load") {
		prefix, name = "Load", strings.TrimPrefix(name, "Load")
		if strings.HasSuffix(name, "Part") {
			tail, name = "Part", strings.TrimSuffix(name, "Part")
		}
	} else if strings.HasPrefix(name, "Broadcast") {
		prefix, name = "Broadcast", strings.TrimPrefix(name, "Broadcast")
	}
	bits := elementBits(name)
	if bits == 0 {
		return "", false
	}
	if width != 0 {
		name = strings.TrimSuffix(name, "s") + "x" + strconv.Itoa(width/bits)
	}
	return prefix + name + tail, true
}

func (f *fileRewriter) specialize(node ast.Node, width int) ast.Node {
	copy, originals := cloneSyntax(node)
	result := astutil.Apply(copy, func(c *astutil.Cursor) bool {
		old := originals[c.Node()]
		switch n := c.Node().(type) {
		case *ast.SelectorExpr:
			old, ok := old.(*ast.SelectorExpr)
			if !ok {
				break
			}
			if obj := f.a.info.Uses[old.Sel]; obj != nil && obj.Pkg() != nil && obj.Pkg().Path() == simdPath && packageObject(obj) {
				if name, ok := bridgeName(obj.Name(), width); ok {
					c.Replace(f.qualify(bridgePath, name, n.Pos()))
					return false
				}
			}
		case *ast.Ident:
			old, ok := old.(*ast.Ident)
			if !ok {
				break
			}
			// An embedded type identifier both defines a field and uses a
			// type. Rewrite the type use here; field selections and literal
			// keys below have only a field use.
			obj := f.a.info.Uses[old]
			if obj == nil {
				obj = f.a.info.Defs[old]
			}
			if obj == nil {
				break
			}
			if obj.Pkg() != nil && obj.Pkg().Path() == simdPath && packageObject(obj) {
				if name, ok := bridgeName(obj.Name(), width); ok {
					c.Replace(f.qualify(bridgePath, name, n.Pos()))
					return false
				}
			}
			if fn, ok := obj.(*types.Func); ok && fn.Signature().Recv() != nil {
				break // method names preserve interface satisfaction
			}
			if field, ok := obj.(*types.Var); ok && field.Embedded() {
				// An embedded alias names the field, even when its RHS is
				// a portable base type. Do not erase that spelling here.
				t := field.Type()
				if ptr, ok := t.(*types.Pointer); ok {
					t = ptr.Elem()
				}
				var name *types.TypeName
				switch t := t.(type) {
				case *types.Named:
					name = t.Obj()
				case *types.Alias:
					name = t.Obj()
				}
				if name != nil {
					if baseType(name) {
						n.Name, _ = bridgeName(name.Name(), width)
					} else if f.a.dependent(name) {
						n.Name += suffix(width)
					}
				}
			}
			if packageObject(obj) && f.a.dependent(obj) {
				n.Name += suffix(width)
			}
		}
		return true
	}, nil)
	if fn, ok := result.(*ast.FuncDecl); ok && fn.Recv == nil {
		// init declarations are deliberately absent from the package scope.
		fn.Name.Name = functionVariant(node.(*ast.FuncDecl), width)
	}
	if fn, ok := result.(*ast.FuncDecl); ok && fn.Doc != nil {
		// Export/link bindings belong to the retained entry declaration.
		// Variants are implementation details and have different names.
		bindings := make(map[token.Pos]bool)
		for _, d := range f.directives.Group(node.(*ast.FuncDecl).Doc).Bindings {
			if d.Name == "export" || d.Name == "go:linkname" || d.Name == "llgo:link" {
				bindings[d.Pos] = true
			}
		}
		comments := fn.Doc.List[:0]
		for _, comment := range fn.Doc.List {
			if !bindings[comment.Pos()] {
				comments = append(comments, comment)
			}
		}
		fn.Doc.List = comments
	}
	return result
}

func functionVariant(decl *ast.FuncDecl, width int) string {
	name := decl.Name.Name
	if name == "init" {
		name += "@" + strconv.Itoa(int(decl.Pos()))
	}
	return name + suffix(width)
}

func ident(name string, pos token.Pos) *ast.Ident {
	return &ast.Ident{NamePos: pos, Name: name}
}

func (f *fileRewriter) panicBody(pos token.Pos) *ast.BlockStmt {
	// The user's package or parameters can shadow panic. Use a bridge-owned
	// trap and a terminating loop so the retained signature type-checks even
	// though go/types cannot infer that the helper never returns.
	return &ast.BlockStmt{Lbrace: pos, Rbrace: pos, List: []ast.Stmt{
		&ast.ExprStmt{X: &ast.CallExpr{Fun: f.qualify(bridgePath, "LLGoMidwayUnreachable", pos), Lparen: pos, Rparen: pos}},
		&ast.ForStmt{For: pos, Body: &ast.BlockStmt{Lbrace: pos, Rbrace: pos}},
	}}
}

func (f *fileRewriter) dispatch(decl *ast.FuncDecl) *ast.BlockStmt {
	pos := decl.Pos()
	var args []ast.Expr
	for _, field := range decl.Type.Params.List {
		if len(field.Names) == 0 {
			field.Names = []*ast.Ident{ident("@arg"+strconv.Itoa(len(args)), pos)}
		}
		for i, name := range field.Names {
			if name.Name == "_" {
				name = ident("@arg"+strconv.Itoa(len(args)), pos)
				field.Names[i] = name
			}
			args = append(args, ident(name.Name, pos))
		}
	}
	// Blank type parameters cannot be used as explicit type arguments.
	if decl.Type.TypeParams != nil {
		n := 0
		for _, field := range decl.Type.TypeParams.List {
			for i, name := range field.Names {
				if name.Name == "_" {
					field.Names[i] = ident("@typearg"+strconv.Itoa(n), name.Pos())
				}
				n++
			}
		}
	}
	invoke := func(width int) []ast.Stmt {
		var fun ast.Expr = ident(functionVariant(decl, width), pos)
		if decl.Type.TypeParams != nil {
			var params []ast.Expr
			for _, field := range decl.Type.TypeParams.List {
				for _, name := range field.Names {
					params = append(params, ident(name.Name, pos))
				}
			}
			fun = &ast.IndexListExpr{X: fun, Lbrack: pos, Rbrack: pos, Indices: params}
		}
		call := &ast.CallExpr{Fun: fun, Lparen: pos, Rparen: pos, Args: args}
		if n := len(decl.Type.Params.List); n != 0 {
			if _, ok := decl.Type.Params.List[n-1].Type.(*ast.Ellipsis); ok {
				call.Ellipsis = pos
			}
		}
		if decl.Type.Results != nil && len(decl.Type.Results.List) != 0 {
			return []ast.Stmt{&ast.ReturnStmt{Return: pos, Results: []ast.Expr{call}}}
		}
		return []ast.Stmt{&ast.ExprStmt{X: call}, &ast.ReturnStmt{Return: pos}}
	}
	return &ast.BlockStmt{Lbrace: pos, Rbrace: pos, List: []ast.Stmt{
		&ast.IfStmt{If: pos, Cond: &ast.CallExpr{Fun: f.qualify(simdPath, "Emulated", pos), Lparen: pos, Rparen: pos}, Body: &ast.BlockStmt{Lbrace: pos, Rbrace: pos, List: invoke(0)}},
		invoke(128)[0],
	}}
}
