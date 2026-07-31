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

package cl

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/goplus/llgo/internal/typepatch"
)

func TestPreparePatchBuildsImmutableMergedTypes(t *testing.T) {
	original := types.NewPackage("example.com/p", "p")
	original.Scope().Insert(types.NewVar(token.NoPos, original, "Keep", types.Typ[types.Int]))
	original.Scope().Insert(types.NewVar(token.NoPos, original, "Skip", types.Typ[types.Int]))
	alternate := types.NewPackage("example.com/p", "p")
	alternate.Scope().Insert(types.NewVar(token.NoPos, alternate, "Alt", types.Typ[types.Int]))

	file, err := parser.ParseFile(token.NewFileSet(), "p.go", `package p
//llgo:skip Skip
type T int
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	prepared := PreparePatch(Patch{Types: typepatch.Clone(alternate)}, original, []*ast.File{file})
	if !prepared.prepared {
		t.Fatal("patch was not marked prepared")
	}
	for _, name := range []string{"Alt", "Keep"} {
		if prepared.Types.Scope().Lookup(name) == nil {
			t.Fatalf("prepared patch is missing %s", name)
		}
	}
	if prepared.Types.Scope().Lookup("Skip") != nil {
		t.Fatal("prepared patch retained skipped declaration")
	}
	if typepatch.IsPatched(original) {
		t.Fatal("preparing a patch modified the original types package")
	}
	if original.Scope().Lookup("Keep") == nil || original.Scope().Lookup("Skip") == nil {
		t.Fatal("preparing a patch modified the original package scope")
	}
}

func TestPreparePatchHandlesSkipAllAndNoopInputs(t *testing.T) {
	original := types.NewPackage("example.com/p", "p")
	original.Scope().Insert(types.NewVar(token.NoPos, original, "Keep", types.Typ[types.Int]))
	alternate := types.NewPackage("example.com/p", "p")
	alternate.Scope().Insert(types.NewVar(token.NoPos, alternate, "Alt", types.Typ[types.Int]))

	file, err := parser.ParseFile(token.NewFileSet(), "p.go", `package p
//llgo:skipall
import "unsafe"
var V int
func F() {}
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	prepared := PreparePatch(Patch{Types: typepatch.Clone(alternate)}, original, []*ast.File{file})
	if !prepared.prepared || !prepared.skipall {
		t.Fatalf("prepared patch flags = (prepared=%v, skipall=%v), want true, true", prepared.prepared, prepared.skipall)
	}
	if prepared.Types.Scope().Lookup("Keep") != nil {
		t.Fatal("skipall merged an original declaration")
	}

	alreadyPrepared := Patch{Types: prepared.Types, prepared: true}
	if got := PreparePatch(alreadyPrepared, original, nil); !got.prepared || got.Types != alreadyPrepared.Types {
		t.Fatal("PreparePatch changed an already prepared patch")
	}
	if got := PreparePatch(Patch{}, original, nil); got.prepared {
		t.Fatal("PreparePatch prepared a patch without a type package")
	}
	if got := PreparePatch(Patch{Types: prepared.Types}, nil, nil); got.prepared {
		t.Fatal("PreparePatch prepared a patch without an original package")
	}
}
