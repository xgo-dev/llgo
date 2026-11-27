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

func TestGenericSignatureIdentical(t *testing.T) {
	pkg := types.NewPackage("example.com/gen", "gen")
	tildeConstraint := func(k types.BasicKind) *types.Interface {
		iface := types.NewInterfaceType(nil, []types.Type{
			types.NewUnion([]*types.Term{types.NewTerm(true, types.Typ[k])}),
		}).Complete()
		iface.MarkImplicit()
		return iface
	}
	anyType := types.Universe.Lookup("any").Type()
	comparableType := types.Universe.Lookup("comparable").Type()

	// 1. Different constraints, no params: func[T ~int]() vs func[S ~string]()
	tpInt1 := types.NewTypeParam(types.NewTypeName(token.NoPos, pkg, "T", nil), tildeConstraint(types.Int))
	tpStr1 := types.NewTypeParam(types.NewTypeName(token.NoPos, pkg, "S", nil), tildeConstraint(types.String))
	sigNoParamInt := types.NewSignatureType(nil, nil, []*types.TypeParam{tpInt1}, nil, nil, false)
	sigNoParamStr := types.NewSignatureType(nil, nil, []*types.TypeParam{tpStr1}, nil, nil, false)
	if types.Identical(sigNoParamInt, sigNoParamStr) {
		t.Fatalf("func[T ~int]() and func[S ~string]() reported identical")
	}

	// 2. Alpha-equivalent type params with same constraint: func[T ~int](T) vs func[S ~int](S)
	tpInt2 := types.NewTypeParam(types.NewTypeName(token.NoPos, pkg, "S", nil), tildeConstraint(types.Int))
	sigParamIntT := types.NewSignatureType(nil, nil, []*types.TypeParam{tpInt1},
		types.NewTuple(types.NewParam(token.NoPos, pkg, "x", tpInt1)), nil, false)
	sigParamIntS := types.NewSignatureType(nil, nil, []*types.TypeParam{tpInt2},
		types.NewTuple(types.NewParam(token.NoPos, pkg, "y", tpInt2)), nil, false)
	if !types.Identical(sigParamIntT, sigParamIntS) {
		t.Fatalf("func[T ~int](T) and func[S ~int](S) reported not identical")
	}
	sigParamStrS := types.NewSignatureType(nil, nil, []*types.TypeParam{tpStr1},
		types.NewTuple(types.NewParam(token.NoPos, pkg, "y", tpStr1)), nil, false)
	if types.Identical(sigParamIntT, sigParamStrS) {
		t.Fatalf("func[T ~int](T) and func[S ~string](S) reported identical")
	}

	// 3. Multi-param positional correspondence & swapped order
	tpA := types.NewTypeParam(types.NewTypeName(token.NoPos, pkg, "T", nil), anyType)
	tpB := types.NewTypeParam(types.NewTypeName(token.NoPos, pkg, "U", nil), anyType)
	tpX := types.NewTypeParam(types.NewTypeName(token.NoPos, pkg, "A", nil), anyType)
	tpY := types.NewTypeParam(types.NewTypeName(token.NoPos, pkg, "B", nil), anyType)
	sigTU := types.NewSignatureType(nil, nil, []*types.TypeParam{tpA, tpB},
		types.NewTuple(types.NewParam(token.NoPos, pkg, "", tpA), types.NewParam(token.NoPos, pkg, "", tpB)), nil, false)
	sigAB := types.NewSignatureType(nil, nil, []*types.TypeParam{tpX, tpY},
		types.NewTuple(types.NewParam(token.NoPos, pkg, "", tpX), types.NewParam(token.NoPos, pkg, "", tpY)), nil, false)
	sigBA := types.NewSignatureType(nil, nil, []*types.TypeParam{tpX, tpY},
		types.NewTuple(types.NewParam(token.NoPos, pkg, "", tpY), types.NewParam(token.NoPos, pkg, "", tpX)), nil, false)
	if !types.Identical(sigTU, sigAB) {
		t.Fatal("func[T, U any](T, U) and func[A, B any](A, B) reported not identical")
	}
	if types.Identical(sigTU, sigBA) {
		t.Fatal("func[T, U any](T, U) and func[A, B any](B, A) reported identical")
	}
	// Reused TypeParam pointer at different positions must not short-circuit via x == y.
	sigReuse1 := types.NewSignatureType(nil, nil, []*types.TypeParam{tpA, tpB},
		types.NewTuple(types.NewParam(token.NoPos, pkg, "", tpA)), nil, false)
	sigReuse2 := types.NewSignatureType(nil, nil, []*types.TypeParam{tpB, tpA},
		types.NewTuple(types.NewParam(token.NoPos, pkg, "", tpA)), nil, false)
	if types.Identical(sigReuse1, sigReuse2) {
		t.Fatal("func[T, U any](T) and func[U, T any](T) reported identical")
	}

	// 4. Dependent constraints: func[T any, S ~[]T](S) T vs func[A any, B ~[]A](B) A
	depT := types.NewTypeParam(types.NewTypeName(token.NoPos, pkg, "T", nil), anyType)
	depS := types.NewTypeParam(types.NewTypeName(token.NoPos, pkg, "S", nil),
		types.NewInterfaceType(nil, []types.Type{types.NewUnion([]*types.Term{types.NewTerm(true, types.NewSlice(depT))})}).Complete())
	depA := types.NewTypeParam(types.NewTypeName(token.NoPos, pkg, "A", nil), anyType)
	depB := types.NewTypeParam(types.NewTypeName(token.NoPos, pkg, "B", nil),
		types.NewInterfaceType(nil, []types.Type{types.NewUnion([]*types.Term{types.NewTerm(true, types.NewSlice(depA))})}).Complete())
	sigDep1 := types.NewSignatureType(nil, nil, []*types.TypeParam{depT, depS},
		types.NewTuple(types.NewParam(token.NoPos, pkg, "", depS)),
		types.NewTuple(types.NewParam(token.NoPos, pkg, "", depT)), false)
	sigDep2 := types.NewSignatureType(nil, nil, []*types.TypeParam{depA, depB},
		types.NewTuple(types.NewParam(token.NoPos, pkg, "", depB)),
		types.NewTuple(types.NewParam(token.NoPos, pkg, "", depA)), false)
	if !types.Identical(sigDep1, sigDep2) {
		t.Fatal("dependent-constraint generic signatures func[T any, S ~[]T](S) T and func[A any, B ~[]A](B) A reported not identical")
	}

	// 5. F-bounded constraint: func[T interface{ Less(T) bool }](T, T) bool
	fT := types.NewTypeParam(types.NewTypeName(token.NoPos, pkg, "T", nil), nil)
	fT.SetConstraint(types.NewInterfaceType([]*types.Func{
		types.NewFunc(token.NoPos, pkg, "Less", types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewParam(token.NoPos, pkg, "", fT)),
			types.NewTuple(types.NewParam(token.NoPos, pkg, "", types.Typ[types.Bool])), false)),
	}, nil).Complete())
	fS := types.NewTypeParam(types.NewTypeName(token.NoPos, pkg, "S", nil), nil)
	fS.SetConstraint(types.NewInterfaceType([]*types.Func{
		types.NewFunc(token.NoPos, pkg, "Less", types.NewSignatureType(nil, nil, nil,
			types.NewTuple(types.NewParam(token.NoPos, pkg, "", fS)),
			types.NewTuple(types.NewParam(token.NoPos, pkg, "", types.Typ[types.Bool])), false)),
	}, nil).Complete())
	sigFB1 := types.NewSignatureType(nil, nil, []*types.TypeParam{fT},
		types.NewTuple(types.NewParam(token.NoPos, pkg, "", fT), types.NewParam(token.NoPos, pkg, "", fT)),
		types.NewTuple(types.NewParam(token.NoPos, pkg, "", types.Typ[types.Bool])), false)
	sigFB2 := types.NewSignatureType(nil, nil, []*types.TypeParam{fS},
		types.NewTuple(types.NewParam(token.NoPos, pkg, "", fS), types.NewParam(token.NoPos, pkg, "", fS)),
		types.NewTuple(types.NewParam(token.NoPos, pkg, "", types.Typ[types.Bool])), false)
	if !types.Identical(sigFB1, sigFB2) {
		t.Fatal("F-bounded generic signatures reported not identical")
	}

	// 6. Outer free type parameter vs bound type parameter, and comparable vs any
	outerT := types.NewTypeParam(types.NewTypeName(token.NoPos, pkg, "Outer", nil), anyType)
	sigBound := types.NewSignatureType(nil, nil, []*types.TypeParam{tpA},
		types.NewTuple(types.NewParam(token.NoPos, pkg, "", tpA)), nil, false)
	sigFree := types.NewSignatureType(nil, nil, []*types.TypeParam{tpX},
		types.NewTuple(types.NewParam(token.NoPos, pkg, "", outerT)), nil, false)
	if types.Identical(sigBound, sigFree) {
		t.Fatal("bound type parameter matched unrelated outer type parameter")
	}
	tpComp := types.NewTypeParam(types.NewTypeName(token.NoPos, pkg, "C", nil), comparableType)
	sigComp := types.NewSignatureType(nil, nil, []*types.TypeParam{tpComp},
		types.NewTuple(types.NewParam(token.NoPos, pkg, "", tpComp)), nil, false)
	if types.Identical(sigBound, sigComp) {
		t.Fatal("func[T any](T) and func[C comparable](C) reported identical")
	}

	// 7. Checked go/types-backed generic signature vs synthetic generic signature
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "g.go", `package g
func H[T ~int](x T) {}
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	checkedPkg, err := (&types.Config{}).Check("example.com/g", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	checkedH := checkedPkg.Scope().Lookup("H").Type()
	if !types.Identical(checkedH, sigParamIntS) {
		t.Fatalf("checked %v and synthetic %v reported not identical", checkedH, sigParamIntS)
	}
	if types.Identical(checkedH, sigParamStrS) {
		t.Fatalf("checked %v and synthetic %v reported identical", checkedH, sigParamStrS)
	}
}
