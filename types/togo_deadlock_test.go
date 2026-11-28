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

package types_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/xgo-dev/llgo/types"
)

type togoTestImporter map[string]*types.Package

func (m togoTestImporter) Import(path string) (*types.Package, error) {
	if p, ok := m[path]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("package %q not found", path)
}

// TestToGoPackageNoDeadlockOnSynthInstantiation verifies that converting an
// imported synthetic package (such as one decoded from exportdata/.llgotypes)
// containing a synthetic generic instantiation whose origin is a lazily
// unresolved Named type in the same Context does not self-deadlock on Context.mu.
func TestToGoPackageNoDeadlockOnSynthInstantiation(t *testing.T) {
	ctx := types.NewContext()
	fset := token.NewFileSet()

	// Package A defines a generic Named type checked in ctx. Its Named shell
	// remains lazily unresolved until touched.
	fileA, err := parser.ParseFile(fset, "a.go", `package a
type Box[T any] struct { V T }
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	confA := &types.Config{Context: ctx, Sizes: ctx}
	pkgA, err := confA.Check("example.com/a", fset, []*ast.File{fileA}, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Package B simulates a package decoded by exportdata (goPkg == nil) that
	// references a synthetic instantiation of a.Box[int].
	boxNamed := pkgA.Scope().Lookup("Box").Type().(*types.Named)
	boxInt, err := types.Instantiate(nil, boxNamed, []types.Type{types.Typ[types.Int]}, false)
	if err != nil {
		t.Fatal(err)
	}
	pkgB := types.NewPackage("example.com/b", "b")
	pkgB.SetImports([]*types.Package{pkgA})
	sig := types.NewSignatureType(nil, nil, nil, nil,
		types.NewTuple(types.NewParam(token.NoPos, pkgB, "", boxInt)), false)
	pkgB.Scope().Insert(types.NewFunc(token.NoPos, pkgB, "Make", sig))
	pkgB.MarkComplete()

	// Package C imports B and typechecks in the SAME ctx. Converting B to
	// go/types holds ctx.mu.Lock() and must not re-enter ctx.resolveNamed.
	fileC, err := parser.ParseFile(fset, "c.go", `package c
import "example.com/b"
func Use() int { return b.Make().V }
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	confC := &types.Config{
		Context:  ctx,
		Sizes:    ctx,
		Importer: togoTestImporter{"example.com/a": pkgA, "example.com/b": pkgB},
	}
	if _, err := confC.Check("example.com/c", fset, []*ast.File{fileC}, nil); err != nil {
		t.Fatalf("check C: %v", err)
	}
}
