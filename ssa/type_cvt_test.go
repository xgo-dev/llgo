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

package ssa

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestNamedTypeConversionIsIndependentOfTraversalOrder(t *testing.T) {
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

	convert := func(first, second *types.Named) (*types.Named, *types.Named) {
		cvt := newGoTypes()
		cvt.cvtNamed(first)
		rawSecond, _ := cvt.cvtNamed(second)
		rawFirst, _ := cvt.cvtNamed(first)
		if first == a {
			return rawFirst, rawSecond
		}
		return rawSecond, rawFirst
	}
	aFromA, bFromA := convert(a, b)
	aFromB, bFromB := convert(b, a)

	for name, got := range map[string]*types.Named{
		"A after A-first conversion": aFromA,
		"B after A-first conversion": bFromA,
		"A after B-first conversion": aFromB,
		"B after B-first conversion": bFromB,
	} {
		original := a
		if name[0] == 'B' {
			original = b
		}
		if got == original {
			t.Errorf("%s retained the unconverted recursive type", name)
		}
	}
	if got, want := types.TypeString(aFromA, nil), types.TypeString(aFromB, nil); got != want {
		t.Errorf("A conversion depends on traversal order:\nA-first: %s\nB-first: %s", got, want)
	}
	if got, want := types.TypeString(bFromA, nil), types.TypeString(bFromB, nil); got != want {
		t.Errorf("B conversion depends on traversal order:\nA-first: %s\nB-first: %s", got, want)
	}
	assertCycle := func(name string, rawA, rawB *types.Named) {
		t.Helper()
		aStruct := rawA.Underlying().(*types.Struct)
		if got := aStruct.Field(0).Type().(*types.Pointer).Elem(); got != rawB {
			t.Errorf("%s: converted A points to %v, want converted B", name, got)
		}
		bStruct := rawB.Underlying().(*types.Struct)
		if got := bStruct.Field(0).Type().(*types.Pointer).Elem(); got != rawA {
			t.Errorf("%s: converted B points to %v, want converted A", name, got)
		}
		if got, ok := bStruct.Field(1).Type().(*types.Struct); !ok || !IsClosure(got) {
			t.Errorf("%s: converted B.F type = %v, want closure", name, got)
		}
	}
	assertCycle("A-first", aFromA, bFromA)
	assertCycle("B-first", aFromB, bFromB)
}

func TestRecursiveGenericNamedTypeConversion(t *testing.T) {
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

	cvt := newGoTypes()
	raw, changed := cvt.cvtNamed(original)
	if !changed || raw == original {
		t.Fatal("recursive generic type was not converted")
	}
	underlying, ok := raw.Underlying().(*types.Struct)
	if !ok {
		t.Fatalf("converted generic underlying = %T, want *types.Struct", raw.Underlying())
	}
	if closure, ok := underlying.Field(0).Type().(*types.Struct); !ok || !IsClosure(closure) {
		t.Fatalf("converted generic F = %v, want closure", underlying.Field(0).Type())
	}
	next := underlying.Field(1).Type().(*types.Pointer).Elem()
	if next != raw {
		t.Fatalf("converted generic Next points to %v, want converted instance %v", next, raw)
	}
}

func TestTypeConversionRequirementShapes(t *testing.T) {
	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	sigParam := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(types.NewVar(token.NoPos, nil, "f", sig)), nil, false)
	method := types.NewFunc(token.NoPos, nil, "M", sigParam)
	methodInterface := types.NewInterfaceType([]*types.Func{method}, nil)
	methodInterface.Complete()
	embeddedInterface := types.NewInterfaceType(nil, []types.Type{methodInterface})
	embeddedInterface.Complete()
	typeParam := types.NewTypeParam(
		types.NewTypeName(token.NoPos, nil, "T", nil), types.Universe.Lookup("any").Type())
	alias := types.NewAlias(types.NewTypeName(token.NoPos, nil, "Alias", nil), sig)
	union := types.NewUnion([]*types.Term{types.NewTerm(false, types.Typ[types.Int])})

	tests := []struct {
		name string
		typ  types.Type
		want bool
	}{
		{name: "basic", typ: types.Typ[types.Int]},
		{name: "pointer", typ: types.NewPointer(sig), want: true},
		{name: "interface method", typ: methodInterface, want: true},
		{name: "embedded interface", typ: embeddedInterface, want: true},
		{name: "slice", typ: types.NewSlice(sig), want: true},
		{name: "map key", typ: types.NewMap(sig, types.Typ[types.Int]), want: true},
		{name: "map value", typ: types.NewMap(types.Typ[types.Int], sig), want: true},
		{name: "closure", typ: newGoTypes().cvtClosure(sig)},
		{name: "struct", typ: types.NewStruct([]*types.Var{types.NewField(token.NoPos, nil, "F", sig, false)}, nil), want: true},
		{name: "signature", typ: sig, want: true},
		{name: "array", typ: types.NewArray(sig, 1), want: true},
		{name: "channel", typ: types.NewChan(types.SendRecv, sig), want: true},
		{name: "tuple", typ: types.NewTuple(types.NewVar(token.NoPos, nil, "F", sig)), want: true},
		{name: "type parameter", typ: typeParam},
		{name: "alias", typ: alias, want: true},
		{name: "union", typ: union, want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cvt := newGoTypes()
			query := make(conversionNeedQuery)
			if got := cvt.needsTypeConversion(test.typ, query); got != test.want {
				t.Fatalf("needsTypeConversion(%v) = %v, want %v", test.typ, got, test.want)
			}
			if _, got := newGoTypes().cvtType(test.typ); got != test.want {
				t.Fatalf("cvtType(%v) changed = %v, want %v", test.typ, got, test.want)
			}
		})
	}
}

func TestRecursiveNamedTypesWithoutConversionKeepTheirIdentity(t *testing.T) {
	pkg := types.NewPackage("example.com/plaincycle", "plaincycle")
	a := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "A", nil), types.Typ[types.Int], nil)
	b := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "B", nil), types.Typ[types.Int], nil)
	a.SetUnderlying(types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "B", types.NewPointer(b), false),
	}, nil))
	b.SetUnderlying(types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "A", types.NewPointer(a), false),
	}, nil))

	cvt := newGoTypes()
	if got, changed := cvt.cvtNamed(a); changed || got != a {
		t.Fatalf("plain recursive A conversion = (%v, %v), want original type", got, changed)
	}
	if got, changed := cvt.cvtNamed(b); changed || got != b {
		t.Fatalf("plain recursive B conversion = (%v, %v), want original type", got, changed)
	}
}
