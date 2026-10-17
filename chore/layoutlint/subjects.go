package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"
)

// discoverSubjects keeps concrete uses, including instances without a source
// alias, separate from the declarations that merely define generic templates.
func discoverSubjects(pkg *packages.Package) ([]metadataCheck, error) {
	var subjects []metadataCheck
	sourceFiles := make(map[string]bool, len(pkg.GoFiles))
	for _, file := range pkg.GoFiles {
		sourceFiles[file] = true
	}
	excluded := make(map[*types.TypeName]bool)
	add := func(typ types.Type, name, identifier string, pos token.Pos) error {
		if _, ok := types.Unalias(typ).Underlying().(*types.Struct); !ok || hasFreeTypeParameters(typ) {
			return nil
		}
		if slices.ContainsFunc(subjects, func(previous metadataCheck) bool { return types.Identical(previous.typ, typ) }) {
			return nil
		}
		if identifier == "" {
			identifier = fmt.Sprintf("layoutlintSubject%d", len(subjects))
			if pkg.Types.Scope().Lookup(identifier) != nil {
				return fmt.Errorf("%s: generated type name %s already exists", pkg.PkgPath, identifier)
			}
		}
		position := pkg.Fset.PositionFor(pos, true)
		subjects = append(subjects, metadataCheck{typ: typ, name: name, identifier: identifier, file: position.Filename, line: position.Line})
		return nil
	}
	for _, file := range pkg.Syntax {
		for _, declaration := range file.Decls {
			group, ok := declaration.(*ast.GenDecl)
			if !ok || group.Tok != token.TYPE {
				continue
			}
			for _, spec := range group.Specs {
				spec := spec.(*ast.TypeSpec)
				object := pkg.TypesInfo.Defs[spec.Name].(*types.TypeName)
				if _, ok := object.Type().Underlying().(*types.Struct); !ok || object.Name() == "_" {
					continue
				}
				physical := pkg.Fset.PositionFor(spec.Pos(), false)
				position := pkg.Fset.PositionFor(spec.Pos(), true)
				if !sourceFiles[physical.Filename] && !sourceFiles[position.Filename] {
					excluded[object] = true
					fmt.Printf("INFO generated C type %s excluded (%q, line %d)\n", object.Name(), position.Filename, position.Line)
					continue
				}
				for _, comments := range []*ast.CommentGroup{group.Doc, spec.Doc} {
					if comments == nil {
						continue
					}
					for _, comment := range comments.List {
						text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
						if text == "llgo:cdecl" || text == "llgo:stdcall" || strings.HasPrefix(text, "llgo:type C") {
							excluded[object] = true
						}
					}
				}
				if excluded[object] {
					fmt.Printf("INFO special ABI type %s excluded (%q, line %d)\n", object.Name(), position.Filename, position.Line)
					continue
				}
				if spec.TypeParams.NumFields() != 0 {
					fmt.Printf("INFO generic template %s; checking discovered concrete instances (%q, line %d)\n", object.Name(), position.Filename, position.Line)
					continue
				}
				if err := add(object.Type(), object.Name(), object.Name(), spec.Pos()); err != nil {
					return nil, err
				}
			}
		}
	}
	var uses []*ast.Ident
	for use := range pkg.TypesInfo.Instances {
		uses = append(uses, use)
	}
	slices.SortFunc(uses, func(a, b *ast.Ident) int { return int(a.Pos() - b.Pos()) })
	seen := make(map[types.Type]bool)
	var visit func(types.Type) error
	visit = func(typ types.Type) error {
		if seen[typ] || hasFreeTypeParameters(typ) {
			return nil
		}
		seen[typ] = true
		typ = types.Unalias(typ)
		if named, ok := typ.(*types.Named); ok && named.TypeArgs().Len() != 0 {
			object := named.Obj()
			if object.Pkg() == pkg.Types && object.Parent() == pkg.Types.Scope() && !excluded[object] {
				name := types.TypeString(named, func(owner *types.Package) string {
					if owner == pkg.Types {
						return ""
					}
					return owner.Path()
				})
				if err := add(named, name, "", object.Pos()); err != nil {
					return err
				}
			}
		}
		var children []types.Type
		switch typ := typ.Underlying().(type) {
		case *types.Struct:
			for i := 0; i < typ.NumFields(); i++ {
				children = append(children, typ.Field(i).Type())
			}
		case *types.Array:
			children = append(children, typ.Elem())
		case *types.Slice:
			children = append(children, typ.Elem())
		case *types.Pointer:
			children = append(children, typ.Elem())
		case *types.Map:
			children = append(children, typ.Key(), typ.Elem())
		case *types.Chan:
			children = append(children, typ.Elem())
		case *types.Signature:
			children = append(children, typ.Params(), typ.Results())
		case *types.Tuple:
			for i := 0; i < typ.Len(); i++ {
				children = append(children, typ.At(i).Type())
			}
		}
		for _, child := range children {
			if err := visit(child); err != nil {
				return err
			}
		}
		return nil
	}
	for _, use := range uses {
		if err := visit(pkg.TypesInfo.Instances[use].Type); err != nil {
			return nil, err
		}
	}
	for i := 0; i < len(subjects); i++ {
		if err := visit(subjects[i].typ); err != nil {
			return nil, err
		}
	}
	return subjects, nil
}

func hasFreeTypeParameters(typ types.Type) bool {
	seen := make(map[types.Type]bool)
	var visit func(types.Type) bool
	visit = func(typ types.Type) bool {
		if typ == nil || seen[typ] {
			return false
		}
		seen[typ] = true
		switch typ := typ.(type) {
		case *types.TypeParam:
			return true
		case *types.Alias:
			if typ.TypeParams().Len() > typ.TypeArgs().Len() {
				return true
			}
			for i := 0; i < typ.TypeArgs().Len(); i++ {
				if visit(typ.TypeArgs().At(i)) {
					return true
				}
			}
			return visit(typ.Rhs())
		case *types.Named:
			if typ.TypeParams().Len() > typ.TypeArgs().Len() {
				return true
			}
			for i := 0; i < typ.TypeArgs().Len(); i++ {
				if visit(typ.TypeArgs().At(i)) {
					return true
				}
			}
			return visit(typ.Underlying())
		case *types.Struct:
			for i := 0; i < typ.NumFields(); i++ {
				if visit(typ.Field(i).Type()) {
					return true
				}
			}
		case *types.Array:
			return visit(typ.Elem())
		case *types.Slice:
			return visit(typ.Elem())
		case *types.Pointer:
			return visit(typ.Elem())
		case *types.Map:
			return visit(typ.Key()) || visit(typ.Elem())
		case *types.Chan:
			return visit(typ.Elem())
		case *types.Signature:
			return typ.TypeParams().Len() != 0 || visit(typ.Params()) || visit(typ.Results())
		case *types.Tuple:
			for i := 0; i < typ.Len(); i++ {
				if visit(typ.At(i).Type()) {
					return true
				}
			}
		}
		return false
	}
	return visit(typ)
}
