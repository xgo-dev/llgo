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
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/xgo-dev/llgo/types"
)

func TestSyntheticInstantiatedSignatureIdentity(t *testing.T) {
	const src = `package p

func F[T any](x T) T { return x }
func G[T any]() int { return 0 }

type Iface[T any] interface {
	M(x T) T
}

type Box[T any] struct {
	V T
}

func (b Box[T]) Get() T { return b.V }
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := (&types.Config{}).Check("example.com/p", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Synthetic struct types (goType == nil) force instantiateSynthetic even though
	// the checked generic declarations have goType != nil.
	sInt := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "X", types.Typ[types.Int], false),
	}, nil)
	sStr := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, pkg, "X", types.Typ[types.String], false),
	}, nil)

	// 1. Generic function signature F[T any](T) T
	fOrig := pkg.Scope().Lookup("F").Type()
	fInt, err := types.Instantiate(nil, fOrig, []types.Type{sInt}, false)
	if err != nil {
		t.Fatal(err)
	}
	fStr, err := types.Instantiate(nil, fOrig, []types.Type{sStr}, false)
	if err != nil {
		t.Fatal(err)
	}
	if types.Identical(fInt, fStr) {
		t.Fatalf("F[struct{X int}] and F[struct{X string}] reported identical: %v vs %v", fInt, fStr)
	}
	if types.Identical(fInt, fOrig) {
		t.Fatalf("instantiated F[struct{X int}] reported identical to uninstantiated %v", fOrig)
	}
	wantFInt := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(types.NewParam(token.NoPos, pkg, "x", sInt)),
		types.NewTuple(types.NewParam(token.NoPos, pkg, "", sInt)), false)
	if !types.Identical(fInt, wantFInt) {
		t.Fatalf("F[struct{X int}] not identical to expected %v", wantFInt)
	}

	// 2. Generic function whose params/results do not mention T: G[T any]() int
	gOrig := pkg.Scope().Lookup("G").Type()
	gInt, err := types.Instantiate(nil, gOrig, []types.Type{sInt}, false)
	if err != nil {
		t.Fatal(err)
	}
	wantG := types.NewSignatureType(nil, nil, nil, nil,
		types.NewTuple(types.NewParam(token.NoPos, pkg, "", types.Typ[types.Int])), false)
	if !types.Identical(gInt, wantG) {
		t.Fatalf("G[struct{X int}] not identical to func() int")
	}
	if types.Identical(gInt, gOrig) {
		t.Fatalf("instantiated G[struct{X int}] reported identical to generic %v", gOrig)
	}

	// 3. Generic interface method signature Iface[T any].M(T) T
	ifaceOrig := pkg.Scope().Lookup("Iface").Type()
	ifaceInt, err := types.Instantiate(nil, ifaceOrig, []types.Type{sInt}, false)
	if err != nil {
		t.Fatal(err)
	}
	ifaceStr, err := types.Instantiate(nil, ifaceOrig, []types.Type{sStr}, false)
	if err != nil {
		t.Fatal(err)
	}
	mInt := ifaceInt.Underlying().(*types.Interface).Method(0).Type()
	mStr := ifaceStr.Underlying().(*types.Interface).Method(0).Type()
	if types.Identical(mInt, mStr) {
		t.Fatalf("Iface[struct{X int}].M and Iface[struct{X string}].M reported identical")
	}
	if !types.Identical(mInt, wantFInt) {
		t.Fatalf("Iface[struct{X int}].M not identical to expected %v", wantFInt)
	}

	// 4. Generic Named method signature Box[T any].Get() T
	boxOrig := pkg.Scope().Lookup("Box").Type()
	boxInt, err := types.Instantiate(nil, boxOrig, []types.Type{sInt}, false)
	if err != nil {
		t.Fatal(err)
	}
	boxStr, err := types.Instantiate(nil, boxOrig, []types.Type{sStr}, false)
	if err != nil {
		t.Fatal(err)
	}
	getInt := boxInt.(*types.Named).Method(0).Type()
	getStr := boxStr.(*types.Named).Method(0).Type()
	if types.Identical(getInt, getStr) {
		t.Fatalf("Box[struct{X int}].Get and Box[struct{X string}].Get reported identical")
	}
	wantGetInt := types.NewSignatureType(nil, nil, nil, nil,
		types.NewTuple(types.NewParam(token.NoPos, pkg, "", sInt)), false)
	if !types.Identical(getInt, wantGetInt) {
		t.Fatalf("Box[struct{X int}].Get not identical to expected %v", wantGetInt)
	}
}
