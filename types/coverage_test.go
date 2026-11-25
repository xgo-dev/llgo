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
	"go/constant"
	"go/importer"
	"go/parser"
	"go/token"
	gotypes "go/types"
	"math"
	"testing"

	"github.com/xgo-dev/llgo/target"
	lltypes "github.com/xgo-dev/llgo/types"
)

func TestCoverageAllTypesObjectsAndContext(t *testing.T) {
	spec := target.ResolveSpec(target.Spec{GOOS: "linux", GOARCH: "amd64"})
	ctx := lltypes.NewContext(spec)
	if ctx.Spec().GOARCH != "amd64" {
		t.Fatalf("Spec() = %+v", ctx.Spec())
	}

	// Interning & constructors.
	i8 := ctx.Basic(lltypes.Int8)
	i16 := ctx.Basic(lltypes.Int16)
	i32 := ctx.Basic(lltypes.Int32)
	i64 := ctx.Basic(lltypes.Int64)
	u8 := ctx.Basic(lltypes.Uint8)
	u16 := ctx.Basic(lltypes.Uint16)
	u32 := ctx.Basic(lltypes.Uint32)
	u64 := ctx.Basic(lltypes.Uint64)
	f32 := ctx.Basic(lltypes.Float32)
	f64 := ctx.Basic(lltypes.Float64)
	c64 := ctx.Basic(lltypes.Complex64)
	c128 := ctx.Basic(lltypes.Complex128)
	str := ctx.Basic(lltypes.String)
	up := ctx.Basic(lltypes.UnsafePointer)
	_ = ctx.Basic(lltypes.BasicKind(999))

	anyT := ctx.Any()
	if !anyT.Empty() || !anyT.Comparable() || anyT.String() != "any" || !anyT.HasPtrData() {
		t.Fatalf("Any() = %+v", anyT)
	}
	sl := ctx.Slice(i32)
	if ctx.Slice(i32) != sl || sl.Elem() != i32 || !sl.HasPtrData() || sl.String() != "[]int32" {
		t.Fatalf("Slice() = %v", sl)
	}
	mp := ctx.Map(str, i64)
	if ctx.Map(str, i64) != mp || mp.Key() != str || mp.Elem() != i64 || !mp.HasPtrData() || mp.String() != "map[string]int64" {
		t.Fatalf("Map() = %v", mp)
	}
	ch1 := ctx.Chan(lltypes.SendRecv, i32)
	ch2 := ctx.Chan(lltypes.SendOnly, i32)
	ch3 := ctx.Chan(lltypes.RecvOnly, i32)
	if ctx.Chan(lltypes.SendRecv, i32) != ch1 || ch1.Dir() != lltypes.SendRecv || ch1.Elem() != i32 ||
		ch1.String() != "chan int32" || ch2.String() != "chan<- int32" || ch3.String() != "<-chan int32" {
		t.Fatalf("Chan() = %v, %v, %v", ch1, ch2, ch3)
	}
	arr := ctx.Array(i32, 4, false)
	if ctx.Array(i32, 4, false) != arr || arr.Elem() != i32 || arr.Len() != 4 || arr.NativeStorage() || arr.HasPtrData() || arr.String() != "[4]int32" {
		t.Fatalf("Array() = %v", arr)
	}
	ptr := ctx.Pointer(i32, lltypes.GoPtr)
	if ctx.Pointer(i32, lltypes.GoPtr) != ptr || ptr.Elem() != i32 || ptr.String() != "*int32" {
		t.Fatalf("Pointer() = %v", ptr)
	}
	cptr := ctx.Pointer(i32, lltypes.CPtr)
	if cptr.String() != "*c(int32)" || lltypes.GoPtr.String() != "go" || lltypes.UnsafePtr.String() != "unsafe" || lltypes.CPtr.String() != "c" || lltypes.PtrKind(99).String() == "" {
		t.Fatalf("PtrKind.String() failed: %s", cptr.String())
	}

	// CallConv String() and predicates.
	for _, cc := range []lltypes.CallConv{lltypes.GoClosure, lltypes.GoDirect, lltypes.CDecl, lltypes.Stdcall, lltypes.Fastcall, lltypes.PyFunc, lltypes.WasmFunc, lltypes.CallConv(99)} {
		if cc.String() == "" {
			t.Fatalf("empty CallConv.String() for %d", cc)
		}
	}

	// Tuple, Interface, Struct, Named, Alias, TypeParam, MethodSet, Identical.
	pkg := lltypes.NewPackage("example.com/cov", "cov")
	pkg.MarkComplete()
	pkg.SetImports([]*lltypes.Package{lltypes.NewPackage("unsafe", "unsafe")})
	if pkg.GoPackage() != nil || pkg.Lookup("missing") != nil {
		t.Fatal("unexpected Package defaults")
	}

	p0 := lltypes.NewParam(0, pkg, "x", i64)
	p1 := lltypes.NewParam(0, pkg, "y", str)
	tup := lltypes.NewTuple(p0, p1)
	if tup.Len() != 2 || tup.At(0) != p0 || !tup.HasPtrData() || tup.String() == "" || lltypes.NewTuple() != nil {
		t.Fatalf("Tuple = %v", tup)
	}
	if (*lltypes.Tuple)(nil).Len() != 0 || (*lltypes.Tuple)(nil).HasPtrData() || (*lltypes.Tuple)(nil).String() != "()" {
		t.Fatal("nil Tuple methods failed")
	}

	tpObj := lltypes.NewTypeName(0, pkg, "T", nil)
	tp := lltypes.NewTypeParam(tpObj, anyT)
	if tp.Obj() != tpObj || tp.Index() != 0 || tp.Constraint() != anyT || tp.Underlying() != anyT || !tp.HasPtrData() || tp.String() != "T" {
		t.Fatalf("TypeParam = %v", tp)
	}

	namedObj := lltypes.NewTypeName(0, pkg, "Rec", nil)
	st := lltypes.NewStruct([]*lltypes.Var{
		lltypes.NewField(0, pkg, "A", i32, false),
		lltypes.NewField(0, pkg, "B", ptr, true),
	}, []string{"json:\"a\"", ""})
	st.SetHostLayout(true)
	st.SetNativeLayout(false)
	if st.NumFields() != 2 || st.Field(0).Name() != "A" || st.Tag(0) != "json:\"a\"" || st.Tag(5) != "" || !st.HostLayout() || st.NativeLayout() || st.IsClosureStruct() || st.SIMDTagIndex() != -1 || !st.HasPtrData() || st.String() == "" {
		t.Fatalf("Struct = %v", st)
	}
	if lltypes.NewStruct(nil, nil).String() != "struct{}" {
		t.Fatal("empty struct string")
	}

	named := lltypes.NewNamed(namedObj, st, nil)
	named.SetUnderlying(st)
	mSig := lltypes.NewSignatureType(
		lltypes.NewParam(0, pkg, "r", named),
		[]*lltypes.TypeParam{tp},
		[]*lltypes.TypeParam{tp},
		lltypes.NewTuple(lltypes.NewParam(0, pkg, "v", sl)),
		lltypes.NewTuple(lltypes.NewParam(0, pkg, "", i64), lltypes.NewParam(0, pkg, "", str)),
		true,
		lltypes.GoDirect,
	)
	if !mSig.IsGoVariadic() || mSig.IsCVariadic() || mSig.IsPyVariadic() || len(mSig.TypeParams()) != 1 || len(mSig.RecvTypeParams()) != 1 || mSig.String() == "" {
		t.Fatalf("Signature = %v", mSig)
	}
	mFn := lltypes.NewFunc(0, pkg, "Do", mSig)
	mFn.SetLinkName("cov.Do")
	mFn.SetCallConv(lltypes.GoDirect)
	mFn.SetExternal(false)
	mFn.SetSkip(false)
	mFn.SetPyList(false)
	named.AddMethod(mFn)
	if named.Obj() != namedObj || named.Origin() != named || named.NumMethods() != 1 || named.Method(0) != mFn || named.HasExplicitCallConv() || named.HostLayout() || named.NativeLayout() || named.SIMDTagIndex() != -1 || named.SIMDVec() != nil || !named.HasPtrData() || named.String() == "" {
		t.Fatalf("Named = %v", named)
	}

	// Alias & Unalias.
	aliasObj := lltypes.NewTypeName(0, pkg, "RecAlias", nil)
	alias := lltypes.NewAlias(aliasObj, named)
	if !aliasObj.IsAlias() || alias.Obj() != aliasObj || alias.Rhs() != named || alias.Origin() != alias || alias.Underlying() != st || !alias.HasPtrData() || lltypes.Unalias(alias) != named || alias.String() == "" {
		t.Fatalf("Alias = %v", alias)
	}
	if !lltypes.Identical(alias, named) {
		t.Fatal("alias should be identical to target")
	}

	// Synthetic MethodSet on Named, *Named, Interface.
	iface := lltypes.NewInterfaceType([]*lltypes.Func{mFn}, []lltypes.Type{anyT})
	if iface.NumMethods() != 1 || iface.Method(0) != mFn || iface.NumEmbeddeds() != 1 || iface.EmbeddedType(0) != anyT || iface.Empty() || iface.String() == "" {
		t.Fatalf("Interface = %v", iface)
	}
	for _, recv := range []lltypes.Type{named, lltypes.NewPointer(named), iface} {
		ms := ctx.MethodSet(recv)
		if ms.Len() != 1 || ms.At(0).Obj() != mFn || ms.Lookup(pkg, "Do") == nil || ms.Lookup(pkg, "Nope") != nil {
			t.Fatalf("MethodSet(%s) = len %d", recv, ms.Len())
		}
		sel := ms.At(0)
		_ = sel.Kind()
		_ = sel.Recv()
		_ = sel.Type()
		_ = sel.Index()
		_ = sel.Indirect()
	}

	// Objects, Scope, Const, PkgName, Label.
	sc := lltypes.NewScope(pkg.Scope(), token.NoPos, token.NoPos, "block")
	cObj := lltypes.NewConst(0, pkg, "K", i64, constant.MakeInt64(7))
	vObj := lltypes.NewVar(0, pkg, "v", i64)
	vObj.SetLinkName("C.v")
	vObj.SetExternal(true)
	pnObj := lltypes.NewPkgName(0, pkg, "u", pkg.Imports()[0])
	lblObj := lltypes.NewLabel(0, pkg, "L")
	for _, o := range []lltypes.Object{mFn, cObj, vObj, namedObj, pnObj, lblObj} {
		sc.Insert(o)
		_ = o.Parent()
		_ = o.Pos()
		_ = o.Pkg()
		_ = o.Exported()
		_ = o.Id()
		_ = o.String()
		_ = o.GoObject()
	}
	if sc.Insert(cObj) != cObj || sc.Len() != 6 || len(sc.Names()) != 6 || pkg.Scope().NumChildren() != 1 || pkg.Scope().Child(0) != sc {
		t.Fatalf("Scope operations failed: len=%d", sc.Len())
	}
	if vObj.Origin() != vObj || vObj.Anonymous() || vObj.Embedded() || vObj.IsField() || mFn.Origin() != mFn || mFn.Scope() != nil || cObj.Val() == nil || pnObj.Imported() == nil {
		t.Fatal("Object accessors failed")
	}

	// EqualName across all basic & composite types.
	for _, typ := range []lltypes.Type{
		ctx.Basic(lltypes.Bool), i8, i16, i32, i64, u8, u16, u32, u64,
		ctx.Basic(lltypes.Int), ctx.Basic(lltypes.Uintptr),
		f32, f64, c64, c128, str, up,
		ptr, ch1, sl, mp, mSig, anyT, iface,
		lltypes.NewStruct(nil, nil),
		lltypes.NewStruct([]*lltypes.Var{lltypes.NewField(0, nil, "a", i32, false)}, nil),
		lltypes.NewStruct([]*lltypes.Var{lltypes.NewField(0, nil, "a", sl, false)}, nil),
		lltypes.NewArray(i32, 3),
		lltypes.NewArray(sl, 3),
		ctx.SIMDVec(lltypes.Int32, 4, false),
		nil,
	} {
		_ = ctx.EqualName(typ)
		_ = ctx.Layout(typ)
		_, _, _ = lltypes.DataKindOf(typ, false)
		_, _, _ = lltypes.DataKindOf(typ, true)
	}
	// 32-bit EqualName for int.
	ctx32 := lltypes.NewContext(target.Spec{GOOS: "linux", GOARCH: "386"})
	if ctx32.EqualName(ctx32.Basic(lltypes.Int)) != "memequal32" {
		t.Fatal("32-bit int EqualName != memequal32")
	}
	if dk, _, depth := lltypes.DataKindOf(lltypes.NewArray(f32, 1), false); dk != lltypes.DataBitCast || depth != 1 {
		t.Fatalf("DataKindOf([1]float32) = (%v, %d)", dk, depth)
	}
	if dk, _, _ := lltypes.DataKindOf(f64, true); dk != lltypes.DataIndirect {
		t.Fatalf("DataKindOf(float64, 32bit) = %v", dk)
	}
	if dk, _, _ := lltypes.DataKindOf(up, false); dk != lltypes.DataPointer {
		t.Fatalf("DataKindOf(unsafe.Pointer) = %v", dk)
	}

	// Identical across composite kinds.
	if !lltypes.Identical(sl, lltypes.NewSlice(i32)) ||
		!lltypes.Identical(arr, lltypes.NewArray(i32, 4)) ||
		!lltypes.Identical(mp, lltypes.NewMap(str, i64)) ||
		!lltypes.Identical(ch1, lltypes.NewChan(lltypes.SendRecv, i32)) ||
		!lltypes.Identical(tup, lltypes.NewTuple(p0, p1)) ||
		!lltypes.Identical(iface, lltypes.NewInterfaceType([]*lltypes.Func{mFn}, []lltypes.Type{anyT})) ||
		!lltypes.Identical(ctx.SIMDVec(lltypes.Int32, 4, false), lltypes.NewSIMDVec(i32, 4, false)) ||
		!lltypes.Identical(tp, tp) ||
		lltypes.Identical(i32, nil) ||
		lltypes.Identical(i32, i64) {
		t.Fatal("Identical checks failed")
	}

	// Context mutators & HostFieldLayout & FromGoObject & FromGoWithCallConv & StructFromGoFields.
	ctx.SetSizes(8, 8)
	ctx.InvalidateGoCache()
	_ = ctx.HostFieldLayout(i64)
	_ = ctx.HostFieldLayout(arr)
	_ = ctx.HostFieldLayout(mSig)
	_ = ctx.HostFieldLayout(lltypes.NewSignature(nil, nil, nil, false, lltypes.CDecl))
	_ = ctx.HostFieldLayout(named)
	_ = ctx.HostFieldLayout(nil)
	_ = ctx.HostBasicLayout(lltypes.BasicKind(999))
	_ = ctx.SIMDAlignment(99)

	goSig := gotypes.NewSignatureType(nil, nil, nil, nil, nil, false)
	if sig := ctx.FromGoWithCallConv(goSig, lltypes.Stdcall).(*lltypes.Signature); sig.CallConv() != lltypes.Stdcall {
		t.Fatalf("FromGoWithCallConv = %v", sig)
	}
	if ctx.FromGo(nil) != nil || ctx.FromGoWithCallConv(nil, lltypes.CDecl) != nil || ctx.FromGoObject(nil) != nil {
		t.Fatal("nil FromGo should return nil")
	}
	goPkg := gotypes.NewPackage("p", "p")
	for _, goObj := range []gotypes.Object{
		gotypes.NewVar(0, goPkg, "x", gotypes.Typ[gotypes.Int]),
		gotypes.NewConst(0, goPkg, "C", gotypes.Typ[gotypes.Int], constant.MakeInt64(1)),
		gotypes.NewFunc(0, goPkg, "F", goSig),
		gotypes.NewTypeName(0, goPkg, "T", gotypes.Typ[gotypes.Int]),
		gotypes.NewPkgName(0, goPkg, "imp", goPkg),
		gotypes.NewLabel(0, goPkg, "L"),
		gotypes.Universe.Lookup("len"),
		gotypes.Universe.Lookup("nil"),
	} {
		if ctx.FromGoObject(goObj) == nil {
			t.Fatalf("FromGoObject(%T) returned nil", goObj)
		}
	}
	sf := ctx.StructFromGoFields([]*gotypes.Var{
		gotypes.NewField(0, goPkg, "fn", goSig, false),
	}, true)
	if !sf.NativeLayout() || sf.Field(0).Type().(*lltypes.Signature).CallConv() != lltypes.CDecl {
		t.Fatalf("StructFromGoFields native = %v", sf)
	}

	// Overflow edge cases in AggregateLayout & GCBitmap.
	if p, _ := lltypes.AggregateLayout(2, func(i int) lltypes.LayoutPair {
		if i == 0 {
			return lltypes.LayoutPair{Size: math.MaxInt64, Align: 1}
		}
		return lltypes.LayoutPair{Size: 0, Align: 1}
	}, true); p.Size != -1 {
		t.Fatalf("AggregateLayout tail overflow = %+v", p)
	}
	if lltypes.GCBitmap(nil, 8, 8) != nil || lltypes.GCBitmap([]int64{-1, 100}, 8, 8) == nil {
		t.Fatal("GCBitmap edge case failed")
	}
}

func TestCoverageFuncABITargetsAndSIMDShapes(t *testing.T) {
	// PassMode.String
	for _, m := range []lltypes.PassMode{lltypes.PassIgnore, lltypes.PassDirect, lltypes.PassIndirect, lltypes.PassCoerce, lltypes.PassExpand} {
		if m.String() == "" {
			t.Fatalf("empty PassMode.String(%d)", m)
		}
	}

	smallSt := lltypes.NewStruct([]*lltypes.Var{
		lltypes.NewField(0, nil, "a", lltypes.Typ[lltypes.Int32], false),
		lltypes.NewField(0, nil, "b", lltypes.Typ[lltypes.Int32], false),
	}, nil)
	bigSt := lltypes.NewStruct([]*lltypes.Var{
		lltypes.NewField(0, nil, "a", lltypes.NewArray(lltypes.Typ[lltypes.Int64], 4), false),
	}, nil)
	hfaArr := lltypes.NewArray(lltypes.Typ[lltypes.Float64], 3)
	emptySt := lltypes.NewStruct(nil, nil)

	for _, spec := range []target.Spec{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "linux", GOARCH: "arm64"},
		{GOOS: "windows", GOARCH: "amd64"},
		{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32"},
		{GOOS: "linux", GOARCH: "386"},
		{GOOS: "linux", GOARCH: "riscv64"},
		{GOOS: "linux", GOARCH: "riscv32"},
	} {
		ctx := lltypes.NewContext(spec)
		for _, cc := range []lltypes.CallConv{lltypes.GoClosure, lltypes.GoDirect, lltypes.CDecl} {
			sig := lltypes.NewSignature(
				lltypes.NewParam(0, nil, "r", lltypes.NewPointer(smallSt)),
				lltypes.NewTuple(
					lltypes.NewParam(0, nil, "z", emptySt),
					lltypes.NewParam(0, nil, "s", smallSt),
					lltypes.NewParam(0, nil, "b", bigSt),
					lltypes.NewParam(0, nil, "h", hfaArr),
					lltypes.NewParam(0, nil, "i8", lltypes.Typ[lltypes.Int8]),
					lltypes.NewParam(0, nil, "u8", lltypes.Typ[lltypes.Uint8]),
					lltypes.NewParam(0, nil, "i32", lltypes.Typ[lltypes.Int32]),
					lltypes.NewParam(0, nil, "p", ctx.Pointer(lltypes.Typ[lltypes.Int8], lltypes.CPtr)),
				),
				lltypes.NewTuple(
					lltypes.NewParam(0, nil, "", smallSt),
					lltypes.NewParam(0, nil, "", bigSt),
				),
				false,
				cc,
			)
			abi := ctx.FuncABI(sig)
			if abi == nil || ctx.FuncABI(sig) != abi {
				t.Fatalf("FuncABI(%s, %s) failed", spec.GOARCH, cc)
			}
			// Zero-return & single-return variants.
			_ = ctx.FuncABI(lltypes.NewSignature(nil, nil, nil, false, cc))
			_ = ctx.FuncABI(lltypes.NewSignature(nil, nil, lltypes.NewTuple(lltypes.NewParam(0, nil, "", smallSt)), false, cc))
		}
	}
	if lltypes.ClassifyFuncABI(target.Spec{}, nil, nil) != nil {
		t.Fatal("ClassifyFuncABI(nil) != nil")
	}

	// SIMDNumericShape & SIMDMaskShape.
	spkg := gotypes.NewPackage("simd/archsimd", "archsimd")
	v128 := gotypes.NewNamed(gotypes.NewTypeName(0, spkg, "v128", nil), gotypes.NewStruct([]*gotypes.Var{
		gotypes.NewField(0, spkg, "_", gotypes.NewArray(gotypes.Typ[gotypes.Int8], 0), false),
	}, nil), nil)
	v512 := gotypes.NewNamed(gotypes.NewTypeName(0, spkg, "v512", nil), gotypes.NewStruct([]*gotypes.Var{
		gotypes.NewField(0, spkg, "_", gotypes.NewArray(gotypes.Typ[gotypes.Int8], 0), false),
	}, nil), nil)
	numVec := gotypes.NewNamed(gotypes.NewTypeName(0, spkg, "Float32x4", nil), gotypes.NewStruct([]*gotypes.Var{
		gotypes.NewField(0, spkg, "_", v128, false),
		gotypes.NewField(0, spkg, "x", gotypes.NewArray(gotypes.Typ[gotypes.Float32], 4), false),
	}, nil), nil)
	maskVec := gotypes.NewNamed(gotypes.NewTypeName(0, spkg, "Mask64x8", nil), gotypes.NewStruct([]*gotypes.Var{
		gotypes.NewField(0, spkg, "_", v512, false),
		gotypes.NewField(0, spkg, "x", gotypes.NewArray(gotypes.Typ[gotypes.Int64], 8), false),
	}, nil), nil)
	if _, ok := lltypes.SIMDNumericShape(numVec); !ok {
		t.Fatal("SIMDNumericShape(Float32x4) = false")
	}
	if _, ok := lltypes.SIMDMaskShape(maskVec); !ok {
		t.Fatal("SIMDMaskShape(Mask64x8) = false")
	}
	if _, ok := lltypes.SIMDMaskShape(numVec); ok {
		t.Fatal("SIMDMaskShape accepted Float32x4")
	}
	if _, ok := lltypes.SIMDNumericShape(maskVec); ok {
		t.Fatal("SIMDNumericShape accepted Mask64x8")
	}
	if !lltypes.ContainsVectorStorage(gotypes.NewArray(numVec, 2)) || lltypes.ContainsVectorStorage(nil) {
		t.Fatal("ContainsVectorStorage failed")
	}
	if lltypes.TypeContainsSIMDStorage(nil) || lltypes.TypeContainsHostLayout(nil) || lltypes.TypeIsHostLayoutMarker(nil) {
		t.Fatal("nil TypeContains helpers should be false")
	}

	// Info lazy fallback & TypeAndValue methods.
	src := `package lazy
var X = 1 + 2
func F() int {
	a := []int{X}
	return a[0]
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "lazy.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	ctx := lltypes.NewContext(target.Spec{GOOS: "linux", GOARCH: "amd64"})
	_, info, err := ctx.Check("lazy", fset, []*ast.File{f}, &gotypes.Config{Importer: importer.Default()})
	if err != nil {
		t.Fatal(err)
	}
	if info.GoInfo() == nil || len(info.InitOrder) != 1 {
		t.Fatalf("unexpected Info: goInfo=%v initOrder=%d", info.GoInfo() != nil, len(info.InitOrder))
	}
	for _, tv := range info.Types {
		_ = tv.IsType()
		_ = tv.IsValue()
		_ = tv.IsNil()
		_ = tv.Addressable()
		_ = tv.Assignable()
		_ = tv.HasOk()
		_ = tv.IsBuiltin()
	}
	if (*lltypes.Info)(nil).TypeOf(nil) != nil || (*lltypes.Info)(nil).ObjectOf(nil) != nil || (*lltypes.Info)(nil).GoInfo() != nil {
		t.Fatal("nil Info methods failed")
	}
}
