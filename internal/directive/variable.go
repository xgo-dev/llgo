// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package directive

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"
)

// VariableKind describes a variable's compiler-provided value. It does not
// describe a particular read: source locations belong to the use, not the declaration.
type VariableKind uint8

const (
	OrdinaryVariable VariableKind = iota
	FileVariable
	LineVariable
)

// Variable is a source-independent declaration record, scoped to one checked
// package instance. No AST or LLVM objects escape the directive collector.
type Variable struct {
	Name        string
	Pos         token.Pos
	Linkname    string
	ExportName  string
	Kind        VariableKind
	Initialized bool
}

// Variables includes ordinary declarations too: absence of a link in this
// source view must never inherit a link from a different package instance.
type Variables map[string]Variable

// CollectVariables records variable links using the same precedence as the
// declaration collector: package-wide go:linkname overrides attached directives.
func CollectVariables(fset *token.FileSet, files []*ast.File, exportRename bool) (Variables, error) {
	vars := make(Variables)
	var order []string
	for _, file := range files {
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				continue
			}
			for _, spec := range gen.Specs {
				spec := spec.(*ast.ValueSpec)
				for _, id := range spec.Names {
					rec := Variable{Name: id.Name, Pos: id.Pos(), Initialized: len(spec.Values) != 0}
					if len(gen.Specs) == 1 && len(spec.Names) == 1 {
						if err := collectVariableLink(fset, gen.Doc, &rec, exportRename); err != nil {
							return nil, err
						}
					}

					if _, exists := vars[id.Name]; !exists {
						order = append(order, id.Name)
					}
					vars[id.Name] = rec
				}
			}
		}
	}
	for _, file := range files {
		unsafe := false
		for _, imp := range file.Imports {
			unsafe = unsafe || imp.Path.Value == `"unsafe"`
		}
		if !unsafe {
			continue
		}
		for _, group := range file.Comments {
			for _, item := range ParseGroup(group) {
				if item.Name != "go:linkname" {
					continue
				}
				local, target := ParseLinknameArgs(item.Args)
				if rec, ok := vars[local]; ok && target != "" {
					rec.Linkname = target
					vars[local] = rec
				}
			}
		}
	}
	// Keep source order for deterministic diagnostics.
	for _, name := range order {
		rec := vars[name]
		switch rec.Linkname {
		case "llgo.file":
			rec.Kind = FileVariable
		case "llgo.line":
			rec.Kind = LineVariable
		default:
			if strings.HasPrefix(rec.Linkname, "llgo.") {
				return nil, fmt.Errorf("%s: unknown compiler variable %q", fset.Position(rec.Pos), rec.Linkname)
			}
		}
		if rec.Kind != OrdinaryVariable && rec.Initialized {
			return nil, fmt.Errorf("%s: compiler variable %s cannot have an initializer", fset.Position(rec.Pos), rec.Name)
		}
		vars[name] = rec
	}

	return vars, nil
}

// collectVariableLink keeps declaration attachment and validation in the
// directive layer. Package-wide Go links are applied after attached links.
func collectVariableLink(fset *token.FileSet, doc *ast.CommentGroup, rec *Variable, exportRename bool) error {
	items := ParseGroup(doc)
	for i := len(items) - 1; i >= 0; i-- {
		item := items[i]
		switch item.Name {
		case "go:linkname", "llgo:link":
			local, target := ParseLinknameArgs(item.Args)
			if item.Name == "llgo:link" {
				if target == "" {
					return fmt.Errorf("%s: //llgo:link requires a local name and a target", fset.Position(item.Pos))
				}
				if local != rec.Name {
					return fmt.Errorf("%s: //llgo:link local name %q does not match declaration %q", fset.Position(item.Pos), local, rec.Name)
				}
			}
			if rec.Linkname == "" && local == rec.Name {
				rec.Linkname = target
			}
		case "export":
			if rec.Linkname != "" || item.Args == "" {
				continue
			}
			if item.Args != rec.Name && !exportRename {
				return fmt.Errorf("export comment has wrong name %q", item.Args)
			}
			rec.Linkname, rec.ExportName = item.Args, item.Args
		}
	}
	return nil
}

// Validate checks types after checking (or after restoring checked exports).
func (vars Variables) Validate(fset *token.FileSet, pkg *types.Package) error {
	for _, name := range pkg.Scope().Names() {
		rec, ok := vars[name]
		if !ok || rec.Kind == OrdinaryVariable {
			continue
		}
		obj, ok := pkg.Scope().Lookup(name).(*types.Var)
		if !ok {
			return fmt.Errorf("%s: compiler variable %s is not a variable", fset.Position(rec.Pos), name)
		}
		valid := false
		want := "int32"
		switch rec.Kind {
		case LineVariable:
			valid = types.Identical(obj.Type().Underlying(), types.Typ[types.Int32])
		case FileVariable:
			want = "*int8"
			if ptr, ok := obj.Type().Underlying().(*types.Pointer); ok {
				valid = types.Identical(ptr.Elem().Underlying(), types.Typ[types.Int8])
			}
		}
		if !valid {
			return fmt.Errorf("%s: compiler variable %s (%s) must have type %s", fset.Position(rec.Pos), name, rec.Linkname, want)
		}
	}
	return nil
}

// ValidateVariableUses must run before source transformations or SSA building:
// those can erase address-taking, dead writes, and *(&LINE). lookup resolves
// declaration identity, never an unqualified name or a final linker symbol.
// info may be nil for direct compiler clients; checked lexical scopes then
// provide the same binding for the identifier/qualified-identifier operands.
func ValidateVariableUses(fset *token.FileSet, pkg *types.Package, info *types.Info, files []*ast.File, lookup func(*types.Var) Variable) error {
	var object func(ast.Expr) types.Object
	object = func(expr ast.Expr) types.Object {
		expr = ast.Unparen(expr)
		switch x := expr.(type) {
		case *ast.Ident:
			if info != nil {
				return info.ObjectOf(x)
			}
			scope := pkg.Scope().Innermost(x.Pos())
			if scope == nil {
				return nil
			}
			_, obj := scope.LookupParent(x.Name, x.Pos())
			return obj
		case *ast.SelectorExpr:
			if info != nil {
				return info.ObjectOf(x.Sel)
			}
			if imp, ok := object(x.X).(*types.PkgName); ok {
				return imp.Imported().Scope().Lookup(x.Sel.Name)
			}
		}
		return nil
	}
	var err error
	check := func(expr ast.Expr, action string) {
		if expr == nil || err != nil {
			return
		}
		if v, ok := object(expr).(*types.Var); ok {
			if rec := lookup(v); rec.Kind != OrdinaryVariable {
				err = types.Error{Fset: fset, Pos: expr.Pos(), Msg: fmt.Sprintf("cannot %s compiler variable %s", action, rec.Name)}
			}
		}
	}
	for _, file := range files {
		ast.Inspect(file, func(node ast.Node) bool {
			if err != nil {
				return false
			}
			switch n := node.(type) {
			case *ast.UnaryExpr:
				if n.Op == token.AND {
					check(n.X, "take address of")
				}
			case *ast.AssignStmt:
				if n.Tok == token.DEFINE {
					break
				}
				for _, lhs := range n.Lhs {
					check(lhs, "assign to")
				}
			case *ast.IncDecStmt:
				check(n.X, "assign to")
			case *ast.RangeStmt:
				if n.Tok == token.ASSIGN {
					check(n.Key, "assign to")
					check(n.Value, "assign to")
				}
			}
			return true
		})
		if err != nil {
			return err
		}
	}
	return nil
}
