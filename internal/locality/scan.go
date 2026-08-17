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

package locality

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/xgo-dev/llgo/internal/directive"
)

const (
	legacyThread    = "llgo:threadlocal"
	legacyGoroutine = "llgo:goroutinelocal"
)

// Variable records locality metadata collected for one package variable.
type Variable struct {
	Name string
	Info Info
}

// ScanPackageVar validates and collects locality directives on a package-level
// var declaration.
func ScanPackageVar(fset *token.FileSet, decl *ast.GenDecl) ([]Variable, error) {
	declKind, declPos, err := FromDoc(fset, decl.Doc)
	if err != nil {
		return nil, err
	}
	var ret []Variable
	for _, node := range decl.Specs {
		spec, ok := node.(*ast.ValueSpec)
		if !ok {
			continue
		}
		specKind, specPos, err := FromDoc(fset, spec.Doc)
		if err != nil {
			return nil, err
		}
		kind, _, err := mergeAt(fset, declKind, declPos, specKind, specPos)
		if err != nil {
			return nil, err
		}
		if kind == None {
			continue
		}
		if hasDirective(decl.Doc, "go:embed") || hasDirective(spec.Doc, "go:embed") {
			return nil, errorAt(fset, spec.Pos(), "%s and //go:embed cannot apply to the same variable declaration", Directive(kind))
		}
		if hasDirective(decl.Doc, "go:linkname") || hasDirective(spec.Doc, "go:linkname") {
			return nil, errorAt(fset, spec.Pos(), "%s cannot apply to a //go:linkname variable", Directive(kind))
		}
		for _, ident := range spec.Names {
			if ident.Name == "_" {
				return nil, errorAt(fset, ident.Pos(), "locality directive cannot apply to the blank identifier")
			}
			if ast.IsExported(ident.Name) {
				return nil, errorAt(fset, ident.Pos(), "%s requires an unexported package variable", Directive(kind))
			}
			ret = append(ret, Variable{
				Name: ident.Name,
				Info: Info{Locality: kind, HasInitializer: len(spec.Values) != 0},
			})
		}
	}
	return ret, nil
}

// ValidateNonPackageVar rejects locality directives on declarations other
// than package-level vars, including grouped import/type/const specs.
func ValidateNonPackageVar(fset *token.FileSet, decl *ast.GenDecl) error {
	if err := ValidateDoc(fset, decl.Doc); err != nil {
		return err
	}
	for _, node := range decl.Specs {
		var doc *ast.CommentGroup
		switch spec := node.(type) {
		case *ast.ImportSpec:
			doc = spec.Doc
		case *ast.TypeSpec:
			doc = spec.Doc
		case *ast.ValueSpec:
			doc = spec.Doc
		}
		if err := ValidateDoc(fset, doc); err != nil {
			return err
		}
	}
	return nil
}

// ValidateFuncBody rejects locality directives on declarations nested inside
// a function or function literal.
func ValidateFuncBody(fset *token.FileSet, body *ast.BlockStmt) error {
	if body == nil {
		return nil
	}
	var firstErr error
	ast.Inspect(body, func(node ast.Node) bool {
		if firstErr != nil {
			return false
		}
		stmt, ok := node.(*ast.DeclStmt)
		if !ok {
			return true
		}
		decl, ok := stmt.Decl.(*ast.GenDecl)
		if ok {
			firstErr = ValidateNonPackageVar(fset, decl)
		}
		return firstErr == nil
	})
	return firstErr
}

// FromDoc returns the locality directive attached to doc.
func FromDoc(fset *token.FileSet, doc *ast.CommentGroup) (Kind, token.Pos, error) {
	var kind Kind
	var pos token.Pos
	for _, directive := range directive.ParseGroup(doc) {
		var next Kind
		switch directive.Name {
		case "llgo:tls":
			next = Thread
		case "llgo:gls":
			next = Goroutine
		case legacyThread:
			return None, token.NoPos, errorAt(fset, directive.Pos, "//%s is not supported; use %s", legacyThread, ThreadDirective)
		case legacyGoroutine:
			return None, token.NoPos, errorAt(fset, directive.Pos, "//%s is not supported; use %s", legacyGoroutine, GoroutineDirective)
		default:
			continue
		}
		if directive.Args != "" {
			return None, token.NoPos, errorAt(fset, directive.Pos, "//%s does not accept arguments", directive.Name)
		}
		var err error
		kind, pos, err = mergeAt(fset, kind, pos, next, directive.Pos)
		if err != nil {
			return None, token.NoPos, err
		}
	}
	return kind, pos, nil
}

// ValidateDoc rejects a locality directive outside a package-level var.
func ValidateDoc(fset *token.FileSet, doc *ast.CommentGroup) error {
	kind, pos, err := FromDoc(fset, doc)
	if err != nil {
		return err
	}
	if kind != None {
		return errorAt(fset, pos, "%s applies only to package-level var declarations", Directive(kind))
	}
	return nil
}

func mergeAt(fset *token.FileSet, a Kind, apos token.Pos, b Kind, bpos token.Pos) (Kind, token.Pos, error) {
	merged, ok := Merge(a, b)
	if !ok {
		return None, token.NoPos, errorAt(fset, bpos, "%s and %s cannot apply to the same variable declaration", ThreadDirective, GoroutineDirective)
	}
	if b != None {
		return merged, bpos, nil
	}
	return merged, apos, nil
}

func hasDirective(doc *ast.CommentGroup, name string) bool {
	for _, directive := range directive.ParseGroup(doc) {
		if directive.Name == name {
			return true
		}
	}
	return false
}

func errorAt(fset *token.FileSet, pos token.Pos, format string, args ...any) error {
	msg := fmt.Sprintf(format, args...)
	if fset == nil || pos == token.NoPos {
		return fmt.Errorf("%s", msg)
	}
	return fmt.Errorf("%s: %s", fset.Position(pos), msg)
}
