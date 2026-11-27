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

package llvmgen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"runtime"
	"testing"

	"github.com/xgo-dev/llgo/types"
)

func TestNamedTypeLoweringPreservesIdentityAndOrderIndependence(t *testing.T) {
	pkg := types.NewPackage("example.com/cycle", "cycle")
	a := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "A", nil), types.Typ[types.Int], nil)
	b := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "B", nil), types.Typ[types.Int], nil)
	a.SetUnderlying(types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "B", types.NewPointer(b), false),
	}, nil))
	b.SetUnderlying(types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "A", types.NewPointer(a), false),
		types.NewField(token.NoPos, pkg, "F", types.NewSignatureType(nil, nil, nil, nil, nil, false), false),
	}, nil))

	lower := func(first, second *types.Named) (Type, Type) {
		prog := NewProgram(nil)
		t.Cleanup(prog.Dispose)
		tFirst := prog.Type(first, GoDecl)
		tSecond := prog.Type(second, GoDecl)
		if first == a {
			return tFirst, tSecond
		}
		return tSecond, tFirst
	}
	aFromA, bFromA := lower(a, b)
	aFromB, bFromB := lower(b, a)

	if aFromA.RawType() != a || aFromB.RawType() != a {
		t.Fatalf("lowering mutated recursive type A identity")
	}
	if bFromA.RawType() != b || bFromB.RawType() != b {
		t.Fatalf("lowering mutated recursive type B identity")
	}
	if aFromA.ll.String() != aFromB.ll.String() {
		t.Fatalf("A LLVM type depends on traversal order: %s vs %s", aFromA.ll.String(), aFromB.ll.String())
	}
	if bFromA.ll.String() != bFromB.ll.String() {
		t.Fatalf("B LLVM type depends on traversal order: %s vs %s", bFromA.ll.String(), bFromB.ll.String())
	}
}

func TestRecursiveGenericNamedTypeLoweringPreservesIdentity(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "generic.go", `package generic
type My[T any] struct {
	F func(T)
	Next *My[T]
}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := (&types.Config{}).Check("example.com/generic", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	origin := pkg.Scope().Lookup("My").Type()
	instantiated, err := types.Instantiate(nil, origin, []types.Type{types.Typ[types.Int]}, false)
	if err != nil {
		t.Fatal(err)
	}
	original := instantiated.(*types.Named)

	prog := NewProgram(nil)
	defer prog.Dispose()
	got := prog.Type(original, GoDecl)
	if got.RawType() != original {
		t.Fatalf("recursive generic RawType = %v, want original %v", got.RawType(), original)
	}
	st := original.Underlying().(*types.Struct)
	if !prog.IsClosureType(st.Field(0).Type()) {
		t.Fatalf("expected My[int].F to lower as closure, got %v", st.Field(0).Type())
	}
	if gotSize, wantSize := prog.SizeOf(got), uint64(3*prog.PointerSize()); gotSize != wantSize {
		t.Fatalf("sizeof(My[int]) = %d, want %d", gotSize, wantSize)
	}
}

func TestCFunctionAliasKeepsNativeRepresentation(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	pkg := types.NewPackage("example.com/callback", "callback")
	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	callback := types.NewAlias(types.NewTypeName(token.NoPos, pkg, "Callback", nil), sig)
	goCallback := types.NewAlias(types.NewTypeName(token.NoPos, pkg, "GoCallback", nil), sig)
	outer := types.NewAlias(types.NewTypeName(token.NoPos, pkg, "Outer", nil), callback)
	goOuter := types.NewAlias(types.NewTypeName(token.NoPos, pkg, "GoOuter", nil), goCallback)
	prog.SetTypeCallConv("example.com/callback.Callback", CDecl)

	for _, alias := range []*types.Alias{callback, outer} {
		if prog.IsClosureType(alias) {
			t.Fatalf("C callback %s should not be treated as a Go closure", alias.Obj().Name())
		}
		got := prog.Type(alias, GoDecl)
		if got.RawType() != alias {
			t.Fatalf("C callback %s RawType = %v, want original alias", alias.Obj().Name(), got.RawType())
		}
		if got.kind != vkFuncPtr || prog.SizeOf(got) != uint64(prog.PointerSize()) {
			t.Fatalf("C callback %s representation = (%v, %d bytes), want pointer", alias.Obj().Name(), got.kind, prog.SizeOf(got))
		}
	}
	for _, alias := range []*types.Alias{goCallback, goOuter} {
		if !prog.IsClosureType(alias) {
			t.Fatalf("Go callback %s should be treated as a Go closure", alias.Obj().Name())
		}
		got := prog.Type(alias, GoDecl)
		if !types.Identical(got.RawType(), alias) {
			t.Fatalf("Go callback %s RawType = %v, want identical to %v", alias.Obj().Name(), got.RawType(), alias)
		}
		if got.kind != vkClosure || prog.SizeOf(got) != uint64(2*prog.PointerSize()) {
			t.Fatalf("Go callback %s representation = (%v, %d bytes), want two-word closure", alias.Obj().Name(), got.kind, prog.SizeOf(got))
		}
	}
	sizes := prog.TypeSizes(types.SizesFor("gc", runtime.GOARCH))
	for _, alias := range []*types.Alias{callback, outer} {
		if got := sizes.Sizeof(alias); got != int64(prog.PointerSize()) {
			t.Fatalf("C callback %s Go size = %d, want pointer size", alias.Obj().Name(), got)
		}
	}
	for _, alias := range []*types.Alias{goCallback, goOuter} {
		if got := sizes.Sizeof(alias); got != int64(2*prog.PointerSize()) {
			t.Fatalf("Go callback %s Go size = %d, want two pointers", alias.Obj().Name(), got)
		}
	}
}

func TestSharedNamedTypeLoweringGraph(t *testing.T) {
	pkg := types.NewPackage("example.com/sharedgraph", "sharedgraph")
	var typ types.Type = types.Typ[types.Int]
	for i := 0; i < 48; i++ {
		typ = types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Node", nil),
			types.NewStruct([]*types.Var{
				types.NewField(token.NoPos, pkg, "Left", types.NewPointer(typ), false),
				types.NewField(token.NoPos, pkg, "Right", types.NewPointer(typ), false),
			}, nil), nil)
	}
	prog := NewProgram(nil)
	defer prog.Dispose()
	got := prog.Type(typ, GoDecl)
	if got.RawType() != typ {
		t.Fatalf("shared graph RawType = %v, want original type", got.RawType())
	}
}
