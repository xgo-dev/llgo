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
	"go/parser"
	"go/token"
	gotypes "go/types"
	"testing"

	"github.com/xgo-dev/llgo/target"
	"github.com/xgo-dev/llgo/types"
)

func TestCoverageAllTypesObjectsAndContext(t *testing.T) {
	spec := target.ResolveSpec(target.Spec{GOOS: "linux", GOARCH: "amd64"})
	ctx := types.NewContextWithConfig(types.ContextConfig{Spec: spec})

	// Constructors & basic types.
	i8 := types.Typ[types.Int8]
	i16 := types.Typ[types.Int16]
	i32 := types.Typ[types.Int32]
	i64 := types.Typ[types.Int64]
	u8 := types.Typ[types.Uint8]
	u16 := types.Typ[types.Uint16]
	u32 := types.Typ[types.Uint32]
	u64 := types.Typ[types.Uint64]
	f32 := types.Typ[types.Float32]
	f64 := types.Typ[types.Float64]
	c64 := types.Typ[types.Complex64]
	c128 := types.Typ[types.Complex128]
	str := types.Typ[types.String]
	up := types.Typ[types.UnsafePointer]

	anyT := types.NewInterfaceType(nil, nil).Complete()
	if !anyT.Empty() || anyT.IsComparable() || !anyT.IsMethodSet() || anyT.IsImplicit() || anyT.String() != "interface{}" || !anyT.HasPtrData() {
		t.Fatalf("empty interface = %+v", anyT)
	}
	sl := types.NewSlice(i32)
	if sl.Elem() != i32 || !sl.HasPtrData() || sl.String() != "[]int32" {
		t.Fatalf("Slice() = %v", sl)
	}
	mp := types.NewMap(str, i64)
	if mp.Key() != str || mp.Elem() != i64 || !mp.HasPtrData() || mp.String() != "map[string]int64" {
		t.Fatalf("Map() = %v", mp)
	}
	ch1 := types.NewChan(types.SendRecv, i32)
	ch2 := types.NewChan(types.SendOnly, i32)
	ch3 := types.NewChan(types.RecvOnly, i32)
	if ch1.Dir() != types.SendRecv || ch1.Elem() != i32 ||
		ch1.String() != "chan int32" || ch2.String() != "chan<- int32" || ch3.String() != "<-chan int32" {
		t.Fatalf("Chan() = %v, %v, %v", ch1, ch2, ch3)
	}
	arr := types.NewArray(i32, 4)
	if arr.Elem() != i32 || arr.Len() != 4 || arr.HasPtrData() || arr.String() != "[4]int32" {
		t.Fatalf("Array() = %v", arr)
	}
	ptr := types.NewPointer(i32)
	if ptr.Elem() != i32 || ptr.Kind() != types.GoPtr || ptr.String() != "*int32" {
		t.Fatalf("Pointer() = %v", ptr)
	}
	cptr := types.NewPointerWithKind(i32, types.CPtr)
	if cptr.Kind() != types.CPtr || cptr.String() != "*c(int32)" || types.GoPtr.String() != "go" || types.UnsafePtr.String() != "unsafe" || types.CPtr.String() != "c" || types.PtrKind(99).String() == "" {
		t.Fatalf("PtrKind.String() failed: %s", cptr.String())
	}

	// CallConv & VarKind String().
	for _, cc := range []types.CallConv{types.Go, types.CDecl, types.Stdcall, types.Fastcall, types.Py, types.Wasm, types.CallConv(99)} {
		if cc.String() == "" {
			t.Fatalf("empty CallConv.String() for %d", cc)
		}
	}
	for _, vk := range []types.VarKind{types.PackageVar, types.LocalVar, types.RecvVar, types.ParamVar, types.ResultVar, types.FieldVar, types.VarKind(99)} {
		if vk.String() == "" {
			t.Fatalf("empty VarKind.String() for %d", vk)
		}
	}

	// Tuple, Interface, Struct, Named, Alias, TypeParam, MethodSet, Identical.
	pkg := types.NewPackage("example.com/cov", "cov")
	pkg.MarkComplete()
	pkg.SetImports([]*types.Package{types.Unsafe})
	if pkg.Scope().Lookup("missing") != nil {
		t.Fatal("unexpected Package defaults")
	}

	p0 := types.NewParam(0, pkg, "x", i64)
	p1 := types.NewParam(0, pkg, "y", str)
	tup := types.NewTuple(p0, p1)
	if tup.Len() != 2 || tup.At(0) != p0 || !tup.HasPtrData() || tup.String() == "" || types.NewTuple() != nil {
		t.Fatalf("Tuple = %v", tup)
	}
	if (*types.Tuple)(nil).Len() != 0 || (*types.Tuple)(nil).HasPtrData() || (*types.Tuple)(nil).String() != "()" {
		t.Fatal("nil Tuple methods failed")
	}

	tpObj := types.NewTypeName(0, pkg, "T", nil)
	tp := types.NewTypeParam(tpObj, anyT)
	if tp.Obj() != tpObj || tp.Index() != 0 || tp.Constraint() != anyT || tp.Underlying() != anyT || !tp.HasPtrData() || tp.String() != "T" {
		t.Fatalf("TypeParam = %v", tp)
	}

	namedObj := types.NewTypeName(0, pkg, "Rec", nil)
	st := types.NewStruct([]*types.Var{
		types.NewField(0, pkg, "A", i32, false),
		types.NewField(0, pkg, "B", ptr, true),
	}, []string{"json:\"a\"", ""})
	if st.NumFields() != 2 || st.Field(0).Name() != "A" || st.Tag(0) != "json:\"a\"" || st.Tag(5) != "" || st.HasHostLayout() || st.ContainsHostLayout() || st.SIMDTagIndex() != -1 || !st.HasPtrData() || st.String() == "" {
		t.Fatalf("Struct = %v", st)
	}
	if types.NewStruct(nil, nil).String() != "struct{}" {
		t.Fatal("empty struct string")
	}

	named := types.NewNamed(namedObj, st, nil)
	named.SetUnderlying(st)
	mSig := types.NewSignatureType(
		types.NewParam(0, pkg, "r", named),
		[]*types.TypeParam{tp},
		[]*types.TypeParam{tp},
		types.NewTuple(types.NewParam(0, pkg, "v", sl)),
		types.NewTuple(types.NewParam(0, pkg, "", i64), types.NewParam(0, pkg, "", str)),
		true,
	)
	if !mSig.Variadic() || !mSig.IsClosure() || mSig.IsNative() || mSig.TypeParams().Len() != 1 || mSig.RecvTypeParams().Len() != 1 || mSig.String() == "" {
		t.Fatalf("Signature = %v", mSig)
	}
	mFn := types.NewFunc(0, pkg, "Do", mSig)
	named.AddMethod(mFn)
	if named.Obj() != namedObj || named.Origin() != named || named.NumMethods() != 1 || named.Method(0) != mFn || named.CallConv() != types.Go || named.HasHostLayout() || named.ContainsHostLayout() || named.SIMDTagIndex() != -1 || named.SIMDVec() != nil || !named.HasPtrData() || named.String() == "" {
		t.Fatalf("Named = %v", named)
	}

	aliasObj := types.NewTypeName(0, pkg, "RecAlias", nil)
	alias := types.NewAlias(aliasObj, named)
	if !aliasObj.IsAlias() || alias.Obj() != aliasObj || alias.Rhs() != named || alias.Origin() != alias || alias.Underlying() != st || alias.CallConv() != types.Go || !alias.HasPtrData() || types.Unalias(alias) != named || alias.String() == "" {
		t.Fatalf("Alias = %v", alias)
	}
	if !types.Identical(alias, named) {
		t.Fatal("alias should be identical to target")
	}
	qf := types.RelativeTo(pkg)
	if got := types.TypeString(named, qf); got != "Rec" {
		t.Fatalf("TypeString(named, RelativeTo(pkg)) = %q, want Rec", got)
	}
	if got := types.TypeString(types.NewPointer(named), qf); got != "*Rec" {
		t.Fatalf("TypeString(*named, qf) = %q", got)
	}
	for _, sample := range []types.Type{alias, sl, arr, mp, ch1, ch2, ch3, cptr} {
		if types.TypeString(sample, qf) == "" {
			t.Fatalf("empty TypeString for %T", sample)
		}
	}
	_ = types.ObjectString(mFn, qf)
	_ = types.ObjectString(namedObj, qf)
	_ = types.ObjectString(aliasObj, qf)
	_ = types.ObjectString(p0, qf)
	_ = types.ObjectString(nil, qf)

	if sz := types.SizesFor("llgo", "amd64"); sz == nil || sz.Sizeof(i64) != 8 || sz.Alignof(i64) != 8 || len(sz.Offsetsof([]*types.Var{p0, p1})) != 2 {
		t.Fatalf("SizesFor(llgo, amd64) failed")
	}
	if types.SizesFor("unknown", "amd64") != nil {
		t.Fatal("SizesFor(unknown) should be nil")
	}
	if types.Universe.Lookup("error") == nil || types.Universe.Lookup("any") == nil || types.Universe.Lookup("byte") == nil || types.Universe.Lookup("len") == nil {
		t.Fatal("Universe missing predeclared objects")
	}

	iface := types.NewInterfaceType([]*types.Func{mFn}, []types.Type{anyT}).Complete()
	_ = types.NewInterface([]*types.Func{mFn}, []*types.Named{named}).Complete()
	if iface.NumMethods() != 1 || iface.Method(0) != mFn || iface.NumEmbeddeds() != 1 || iface.EmbeddedType(0) != anyT || iface.Empty() || iface.String() == "" {
		t.Fatalf("Interface = %v", iface)
	}
	for _, recv := range []types.Type{named, types.NewPointer(named), iface} {
		ms := types.NewMethodSet(recv)
		if ms.Len() != 1 || ms.At(0).Obj() != mFn || ms.Lookup(pkg, "Do") == nil || ms.Lookup(pkg, "Nope") != nil {
			t.Fatalf("NewMethodSet(%s) = len %d", recv, ms.Len())
		}
		sel := ms.At(0)
		_ = sel.Kind()
		_ = sel.Recv()
		_ = sel.Type()
		_ = sel.Index()
		_ = sel.Indirect()
	}

	// Objects, Scope, Const, PkgName, Label, Var.SetKind.
	sc := types.NewScope(pkg.Scope(), token.NoPos, token.NoPos, "block")
	cObj := types.NewConst(0, pkg, "K", i64, constant.MakeInt64(7))
	vObj := types.NewVar(0, pkg, "v", i64)
	vObj.SetKind(types.LocalVar)
	if vObj.Kind() != types.LocalVar {
		t.Fatalf("Var.SetKind failed: %v", vObj.Kind())
	}
	pnObj := types.NewPkgName(0, pkg, "u", pkg.Imports()[0])
	lblObj := types.NewLabel(0, pkg, "L")
	for _, o := range []types.Object{mFn, cObj, vObj, namedObj, pnObj, lblObj} {
		sc.Insert(o)
		_ = o.Parent()
		_ = o.Pos()
		_ = o.Pkg()
		_ = o.Exported()
		_ = o.Id()
		_ = o.String()
	}
	if sc.Insert(cObj) != cObj || sc.Len() != 6 || len(sc.Names()) != 6 || pkg.Scope().NumChildren() != 1 || pkg.Scope().Child(0) != sc {
		t.Fatalf("Scope operations failed: len=%d", sc.Len())
	}
	if vObj.Origin() != vObj || vObj.Anonymous() || vObj.Embedded() || vObj.IsField() || mFn.Origin() != mFn || mFn.Scope() != nil || cObj.Val() == nil || pnObj.Imported() == nil {
		t.Fatal("Object accessors failed")
	}

	// EqualName & DataKindOf across all basic & composite types.
	for _, typ := range []types.Type{
		types.Typ[types.Bool], i8, i16, i32, i64, u8, u16, u32, u64,
		types.Typ[types.Int], types.Typ[types.Uintptr],
		f32, f64, c64, c128, str, up,
		ptr, ch1, sl, mp, mSig, anyT, iface,
		types.NewStruct(nil, nil),
		types.NewStruct([]*types.Var{types.NewField(0, nil, "a", i32, false)}, nil),
		types.NewStruct([]*types.Var{types.NewField(0, nil, "a", sl, false)}, nil),
		types.NewArray(i32, 3),
		types.NewArray(sl, 3),
		nil,
	} {
		_ = ctx.EqualName(typ)
		_ = ctx.Layout(typ)
		_, _, _ = types.DataKindOf(typ, false)
		_, _, _ = types.DataKindOf(typ, true)
	}
	ctx32 := types.NewContextWithConfig(types.ContextConfig{Spec: target.Spec{GOOS: "linux", GOARCH: "386"}})
	if ctx32.EqualName(types.Typ[types.Int]) != "memequal32" {
		t.Fatal("32-bit int EqualName != memequal32")
	}
	if dk, _, depth := types.DataKindOf(types.NewArray(f32, 1), false); dk != types.DataBitCast || depth != 1 {
		t.Fatalf("DataKindOf([1]float32) = (%v, %d)", dk, depth)
	}
	if dk, _, _ := types.DataKindOf(f64, true); dk != types.DataIndirect {
		t.Fatalf("DataKindOf(float64, 32bit) = %v", dk)
	}
	if dk, _, _ := types.DataKindOf(up, false); dk != types.DataPointer {
		t.Fatalf("DataKindOf(unsafe.Pointer) = %v", dk)
	}

	// Identical across composite kinds.
	if !types.Identical(sl, types.NewSlice(i32)) ||
		!types.Identical(arr, types.NewArray(i32, 4)) ||
		!types.Identical(mp, types.NewMap(str, i64)) ||
		!types.Identical(ch1, types.NewChan(types.SendRecv, i32)) ||
		!types.Identical(tup, types.NewTuple(p0, p1)) ||
		!types.Identical(iface, types.NewInterfaceType([]*types.Func{mFn}, []types.Type{anyT})) ||
		!types.Identical(tp, tp) ||
		types.Identical(i32, nil) ||
		types.Identical(i32, i64) {
		t.Fatal("Identical checks failed")
	}

	// Context mutators, native metadata & HostFieldLayout & FromGoObject & FromGoWithCallConv.
	ctx.SetSizes(8, 8)
	_ = ctx.HostFieldLayout(i64)
	_ = ctx.HostFieldLayout(arr)
	_ = ctx.HostFieldLayout(mSig)
	_ = ctx.HostFieldLayout(types.NewSignatureWithCallConv(nil, nil, nil, false, types.CDecl))
	_ = ctx.HostFieldLayout(named)
	_ = ctx.HostFieldLayout(nil)
	_ = ctx.HostBasicLayout(types.BasicKind(999))
	_ = ctx.SIMDAlignment(99)

	goSig := gotypes.NewSignatureType(nil, nil, nil, nil, nil, false)
	if sig := ctx.FromGoWithCallConv(goSig, types.Stdcall).(*types.Signature); sig.CallConv() != types.Stdcall {
		t.Fatalf("FromGoWithCallConv = %v", sig)
	}
	if ctx.FromGo(nil) != nil || ctx.FromGoWithCallConv(nil, types.CDecl) != nil || ctx.FromGoObject(nil) != nil {
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
	ctx.SetTypeCallConv("example.com/cov.Rec", types.CDecl)
	if !ctx.HasNativeMetadata() {
		t.Fatal("HasNativeMetadata = false after SetTypeCallConv")
	}
	if cc, ok := ctx.TypeCallConv("example.com/cov.Rec"); !ok || cc != types.CDecl {
		t.Fatalf("TypeCallConv = (%v, %v)", cc, ok)
	}
	if cc, ok := ctx.NativeTypeCallConv(alias); !ok || cc != types.CDecl {
		t.Fatalf("NativeTypeCallConv(alias) = (%v, %v)", cc, ok)
	}
	fnField := types.NewField(0, pkg, "fn", types.NewSignature(nil, nil, nil, false), false)
	ctx.MarkNativeField(fnField)
	ctx.SetNativeStructFields([]token.Pos{token.Pos(42)})
	if sf := ctx.FromGoWithCallConv(gotypes.NewStruct([]*gotypes.Var{
		gotypes.NewField(0, goPkg, "fn", goSig, false),
	}, nil), types.CDecl).(*types.Struct); sf.Field(0).Type().(*types.Signature).CallConv() != types.CDecl {
		t.Fatalf("FromGoWithCallConv native struct = %v", sf)
	}
}

func TestCoverageSIMDShapesAndLazyInfo(t *testing.T) {
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
	simdCtx := types.NewContextWithConfig(types.ContextConfig{Spec: target.Spec{GOOS: "linux", GOARCH: "amd64"}})
	llNum := simdCtx.FromGo(numVec)
	llMask := simdCtx.FromGo(maskVec)
	if v := llNum.SIMDVec(); v == nil || v.IsMask() || v.BitWidth() != 128 || v.SIMDVec() != v || v.SIMDTagIndex() != -1 || !v.ContainsSIMDStorage() || v.String() != "vec<4xfloat32>" {
		t.Fatalf("Float32x4 SIMDVec = %+v", v)
	}
	if v := llMask.SIMDVec(); v == nil || !v.IsMask() || v.BitWidth() != 512 || v.String() != "mask<8xint64>" {
		t.Fatalf("Mask64x8 SIMDVec = %+v", v)
	}
	if !simdCtx.FromGo(gotypes.NewArray(numVec, 2)).ContainsSIMDStorage() {
		t.Fatal("[2]Float32x4.ContainsSIMDStorage() = false")
	}
	if simdCtx.FromGo(v128).SIMDTagIndex() != 0 || simdCtx.FromGo(v512).SIMDTagIndex() != 2 {
		t.Fatal("v128/v512 SIMDTagIndex mismatch")
	}
	for _, typ := range []types.Type{
		types.Typ[types.Int], types.NewPointer(types.Typ[types.Int]),
		types.NewTuple(types.NewParam(0, nil, "a", types.Typ[types.Int])),
		types.NewSignature(nil, nil, nil, false),
		types.NewSlice(types.Typ[types.Int]),
		types.NewArray(types.Typ[types.Int], 2),
		types.NewMap(types.Typ[types.String], types.Typ[types.Int]),
		types.NewChan(types.SendRecv, types.Typ[types.Int]),
		types.NewInterfaceType(nil, nil),
		types.NewStruct(nil, nil),
		types.NewTypeParam(types.NewTypeName(0, nil, "T", nil), nil),
		types.NewAlias(types.NewTypeName(0, nil, "A", nil), types.Typ[types.Int]),
	} {
		if typ.IsHostLayoutMarker() || typ.HasHostLayout() || typ.ContainsHostLayout() || typ.ContainsSIMDStorage() || typ.SIMDTagIndex() != -1 || typ.SIMDVec() != nil {
			t.Fatalf("unexpected layout/SIMD predicate on %T", typ)
		}
	}

	// Custom Sizes adapter via Config.Check (Alignof/Offsetsof/Sizeof) + FromGo Map/Chan + synthetic Instantiate + NativeIntExtForSpec matrix.
	src := `package lazy
import "unsafe"
type S struct { A int8; B int64 }
const (
	sz  = unsafe.Sizeof(S{})
	al  = unsafe.Alignof(S{})
	off = unsafe.Offsetof(S{}.B)
)
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
	info := fullInfo()
	_, err = (&types.Config{
		Sizes:    customSizes{&types.StdSizes{WordSize: 8, MaxAlign: 8}},
		Importer: &mapImporter{pkgs: map[string]*types.Package{"unsafe": types.Unsafe}},
	}).Check("lazy", fset, []*ast.File{f}, info)
	if err != nil {
		t.Fatal(err)
	}
	if len(info.InitOrder) != 1 {
		t.Fatalf("unexpected Info: initOrder=%d", len(info.InitOrder))
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
	if (*types.Info)(nil).TypeOf(nil) != nil || (*types.Info)(nil).ObjectOf(nil) != nil {
		t.Fatal("nil Info methods failed")
	}

	// FromGo Map & Chan lowering + NativeIntExtForSpec matrix.
	_ = simdCtx.FromGo(gotypes.NewMap(gotypes.Typ[gotypes.String], gotypes.Typ[gotypes.Int]))
	_ = simdCtx.FromGo(gotypes.NewChan(gotypes.SendRecv, gotypes.Typ[gotypes.Int]))
	for _, spec := range []target.Spec{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "linux", GOARCH: "arm64"},
		{GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "windows", GOARCH: "amd64"},
		{GOOS: "linux", GOARCH: "386"},
		{GOOS: "linux", GOARCH: "arm"},
		{GOOS: "wasip1", GOARCH: "wasm"},
		{GOOS: "linux", GOARCH: "s390x"},
		{GOOS: "linux", GOARCH: "ppc64le"},
		{GOOS: "linux", GOARCH: "loong64"},
		{GOOS: "linux", GOARCH: "mips64"},
	} {
		_ = types.NativeIntExtForSpec(target.ResolveSpec(spec))
	}

	// Synthetic Instantiate covering Pointer, Array, Map, Chan, Interface, Union substitution.
	tp := types.NewTypeParam(types.NewTypeName(0, nil, "T", nil), nil)
	genIface := types.NewInterfaceType(
		[]*types.Func{types.NewFunc(0, nil, "Get", types.NewSignature(nil, nil, types.NewTuple(types.NewParam(0, nil, "", tp)), false))},
		[]types.Type{types.NewUnion([]*types.Term{types.NewTerm(true, tp)})},
	).Complete()
	richSt := types.NewStruct([]*types.Var{
		types.NewField(0, nil, "P", types.NewPointer(tp), false),
		types.NewField(0, nil, "A", types.NewArray(tp, 2), false),
		types.NewField(0, nil, "M", types.NewMap(types.Typ[types.String], tp), false),
		types.NewField(0, nil, "C", types.NewChan(types.SendRecv, tp), false),
		types.NewField(0, nil, "I", genIface, false),
	}, nil)
	richNamed := types.NewNamed(types.NewTypeName(0, nil, "Rich", nil), richSt, nil)
	richNamed.SetTypeParams([]*types.TypeParam{tp})
	if inst, err := types.Instantiate(nil, richNamed, []types.Type{types.Typ[types.Int32]}, true); err != nil {
		t.Fatalf("rich synthetic Instantiate failed: %v", err)
	} else if st := inst.Underlying().(*types.Struct); st.Field(0).Type().(*types.Pointer).Elem() != types.Typ[types.Int32] {
		t.Fatalf("rich instantiated field 0 = %v", st.Field(0).Type())
	}
	if _, err := types.Instantiate(nil, nil, nil, true); err == nil {
		t.Fatal("Instantiate(nil, validate=true) should fail")
	}
	if _, err := types.Instantiate(nil, types.Typ[types.Int], nil, true); err == nil {
		t.Fatal("Instantiate(int, validate=true) should fail")
	}
	if _, err := types.Instantiate(nil, richNamed, nil, true); err == nil {
		t.Fatal("Instantiate wrong arg count should fail")
	}
}

type customSizes struct {
	types.Sizes
}
