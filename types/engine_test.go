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

package types

import (
	"go/token"
	gotypes "go/types"
	"math"
	"reflect"
	"testing"

	"github.com/xgo-dev/llgo/target"
)

func syntheticSIMDTypes() (*gotypes.Named, *gotypes.Named, *gotypes.Named, *gotypes.Named) {
	pkg := gotypes.NewPackage("simd/archsimd", "archsimd")
	makeTag := func(name string) *gotypes.Named {
		tagStruct := gotypes.NewStruct([]*gotypes.Var{
			gotypes.NewField(token.NoPos, pkg, "_", gotypes.NewArray(gotypes.Typ[gotypes.Int8], 0), false),
		}, nil)
		return gotypes.NewNamed(gotypes.NewTypeName(token.NoPos, pkg, name, nil), tagStruct, nil)
	}
	v128 := makeTag("v128")
	v256 := makeTag("v256")
	f32x4 := gotypes.NewNamed(
		gotypes.NewTypeName(token.NoPos, pkg, "Float32x4", nil),
		gotypes.NewStruct([]*gotypes.Var{
			gotypes.NewField(token.NoPos, pkg, "_", v128, false),
			gotypes.NewField(token.NoPos, pkg, "_", gotypes.NewArray(gotypes.Typ[gotypes.Float32], 4), false),
		}, nil),
		nil,
	)
	mask32x8 := gotypes.NewNamed(
		gotypes.NewTypeName(token.NoPos, pkg, "Mask32x8", nil),
		gotypes.NewStruct([]*gotypes.Var{
			gotypes.NewField(token.NoPos, pkg, "_", v256, false),
			gotypes.NewField(token.NoPos, pkg, "_", gotypes.NewArray(gotypes.Typ[gotypes.Int32], 8), false),
		}, nil),
		nil,
	)
	return v128, v256, f32x4, mask32x8
}

func TestContextInterningLayoutAndSIMD(t *testing.T) {
	v128, v256, f32x4, mask32x8 := syntheticSIMDTypes()
	if idx, ok := SIMDTagIndex(v128); !ok || idx != 0 {
		t.Fatalf("SIMDTagIndex(v128) = (%d, %v), want (0, true)", idx, ok)
	}
	if idx, ok := SIMDTagIndex(v256); !ok || idx != 1 {
		t.Fatalf("SIMDTagIndex(v256) = (%d, %v), want (1, true)", idx, ok)
	}
	if _, ok := SIMDTagIndex(nil); ok {
		t.Fatal("SIMDTagIndex(nil) = true")
	}
	if !ContainsVectorStorage(f32x4) || !ContainsVectorStorage(gotypes.NewArray(f32x4, 2)) || ContainsVectorStorage(gotypes.Typ[gotypes.Int64]) {
		t.Fatal("ContainsVectorStorage classification mismatch")
	}
	if lanes, ok := SIMDNumericShape(f32x4); !ok || lanes.Len() != 4 {
		t.Fatalf("SIMDNumericShape(Float32x4) = (%v, %v)", lanes, ok)
	}
	if _, ok := SIMDMaskShape(f32x4); ok {
		t.Fatal("SIMDMaskShape(Float32x4) = true, want false")
	}
	if lanes, ok := SIMDMaskShape(mask32x8); !ok || lanes.Len() != 8 {
		t.Fatalf("SIMDMaskShape(Mask32x8) = (%v, %v)", lanes, ok)
	}
	if _, ok := SIMDNumericShape(mask32x8); ok {
		t.Fatal("SIMDNumericShape(Mask32x8) = true, want false")
	}

	ctx := NewContext(target.Spec{GOOS: "linux", GOARCH: "amd64"}, nil)
	if ctx.Spec().GOARCH != "amd64" || ctx.Sizes() == nil || ctx.SourceSizes() == nil {
		t.Fatal("NewContext did not initialize Spec/Sizes/SourceSizes")
	}
	// Canonical interning: structurally identical compound types share pointer identity.
	i64a := ctx.Basic(gotypes.Int64, GoPtr)
	i64b := ctx.Type(gotypes.Typ[gotypes.Int64], GoDecl)
	if i64a != i64b {
		t.Fatal("Basic int64 was not interned")
	}
	p1 := ctx.Pointer(i64a, GoPtr)
	p2 := ctx.Type(gotypes.NewPointer(gotypes.Typ[gotypes.Int64]), GoDecl)
	if p1 != p2 {
		t.Fatal("Pointer *int64 was not interned")
	}
	if ctx.Pointer(i64a, NativePtr) == p1 {
		t.Fatal("NativePtr collided with GoPtr in interning table")
	}
	if ctx.Slice(i64a) != ctx.Type(gotypes.NewSlice(gotypes.Typ[gotypes.Int64]), GoDecl) {
		t.Fatal("Slice []int64 was not interned")
	}
	if ctx.Map(ctx.String(), i64a) != ctx.Type(gotypes.NewMap(gotypes.Typ[gotypes.String], gotypes.Typ[gotypes.Int64]), GoDecl) {
		t.Fatal("Map was not interned")
	}
	if ctx.Chan(gotypes.SendRecv, i64a) != ctx.Type(gotypes.NewChan(gotypes.SendRecv, gotypes.Typ[gotypes.Int64]), GoDecl) {
		t.Fatal("Chan was not interned")
	}
	if ctx.Array(i64a, 4, false) != ctx.Type(gotypes.NewArray(gotypes.Typ[gotypes.Int64], 4), GoDecl) {
		t.Fatal("Array was not interned")
	}
	if ctx.Any() != ctx.Type(gotypes.NewInterfaceType(nil, nil), GoDecl) {
		t.Fatal("empty interface was not interned")
	}

	// simd/archsimd types map directly to interned *SIMDVec with Layout.
	vecT := ctx.Type(f32x4, GoDecl)
	sv, ok := vecT.(*SIMDVec)
	if !ok || sv.Len != 4 || sv.Mask || sv != ctx.SIMDVec(gotypes.Float32, 4, false) {
		t.Fatalf("mapped Float32x4 = %#v", vecT)
	}
	vl := ctx.Layout(vecT)
	if vl.Size != 16 || vl.Align != 16 || !vl.HasSIMDStorage || vl.IsRegularMemory {
		t.Fatalf("amd64 Float32x4 Layout = %+v", vl)
	}
	maskT := ctx.Type(mask32x8, GoDecl)
	ml := ctx.Layout(maskT)
	if ml.Size != 32 || ml.Align != 32 || !ml.HasSIMDStorage || !ml.IsRegularMemory {
		t.Fatalf("amd64 Mask32x8 Layout = %+v", ml)
	}

	// ARM64 caps fixed-length SIMD alignment at 16 bytes (matching Clang).
	arm64Ctx := NewContext(target.Spec{GOOS: "linux", GOARCH: "arm64"}, nil)
	armMaskL := arm64Ctx.GoLayout(mask32x8, GoDecl)
	if armMaskL.Size != 32 || armMaskL.Align != 16 {
		t.Fatalf("arm64 Mask32x8 Layout = %+v, want Size=32 Align=16", armMaskL)
	}

	// Struct Layout: Fields, PaddingAfter, GCBitmap, IsRegularMemory.
	pkg := checkPkg(t, `package sample
type Padded struct {
	a byte
	p *int64
	b int64
	q *int64
}
type Regular struct {
	x int64
	y int64
}
type BlankField struct {
	_ int64
	x int64
}
`)
	paddedL := ctx.GoLayout(pkg.Scope().Lookup("Padded").Type(), GoDecl)
	if paddedL.Size != 32 || paddedL.Align != 8 || paddedL.PtrBytes != 32 || paddedL.IsRegularMemory {
		t.Fatalf("Padded Layout = %+v", paddedL)
	}
	if len(paddedL.Fields) != 4 || paddedL.Fields[0].PaddingAfter != 7 || paddedL.Fields[1].Offset != 8 || paddedL.Fields[3].Offset != 24 {
		t.Fatalf("Padded Fields = %+v", paddedL.Fields)
	}
	// Words 0..3: word 1 (p at offset 8) and word 3 (q at offset 24) => bits 1 and 3 => 0b1010 = 0x0a.
	if !reflect.DeepEqual(paddedL.GCBitmap, []byte{0x0a}) {
		t.Fatalf("Padded GCBitmap = %x, want [0a]", paddedL.GCBitmap)
	}
	regL := ctx.GoLayout(pkg.Scope().Lookup("Regular").Type(), GoDecl)
	if !regL.IsRegularMemory || regL.PtrBytes != 0 || regL.GCBitmap != nil {
		t.Fatalf("Regular Layout = %+v", regL)
	}
	if ctx.GoLayout(pkg.Scope().Lookup("BlankField").Type(), GoDecl).IsRegularMemory {
		t.Fatal("BlankField must not be RegularMemory")
	}

	// Wasm32 wide Go slots tracking.
	w32Ctx := NewContext(target.Spec{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32", WasmProvider: "wasi"}, nil)
	if !w32Ctx.Layout(w32Ctx.String()).HasWasmWideSlots || !w32Ctx.Layout(w32Ctx.Pointer(w32Ctx.Basic(gotypes.Int32, GoPtr), GoPtr)).HasWasmWideSlots {
		t.Fatal("w32 Go string/pointer must report HasWasmWideSlots=true")
	}
	if w32Ctx.Layout(w32Ctx.Pointer(w32Ctx.Basic(gotypes.Int32, GoPtr), NativePtr)).HasWasmWideSlots {
		t.Fatal("w32 NativePtr must report HasWasmWideSlots=false")
	}
}

func TestSourceSizesOverflowAndUnifiedPolicies(t *testing.T) {
	// Overflow safety matching go/types.Sizes (-1 on int64 overflow).
	if AlignSize(math.MaxInt64, 8) != -1 {
		t.Fatal("AlignSize overflow did not return -1")
	}
	if ArrayLayout(LayoutPair{Size: math.MaxInt64 / 2, Align: 8}, 4).Size != -1 {
		t.Fatal("ArrayLayout overflow did not return -1")
	}
	ov, _ := AggregateLayout(2, func(i int) LayoutPair {
		return LayoutPair{Size: math.MaxInt64 - 4, Align: 1}
	}, false)
	if ov.Size != -1 {
		t.Fatalf("AggregateLayout overflow = %+v, want Size=-1", ov)
	}

	// SourceSizes with CallConvLookup covering Go closures, C callbacks, HostLayout, and SIMD.
	_, _, f32x4, _ := syntheticSIMDTypes()
	ss := NewSourceSizes(SourceSizesConfig{
		Spec: target.Spec{GOOS: "linux", GOARCH: "amd64"},
	})
	if ss.Alignof(nil) != 1 || ss.Sizeof(nil) != 0 {
		t.Fatal("SourceSizes nil guards failed")
	}
	if ss.SIMDAlignment(0) != 16 || ss.SIMDAlignment(1) != 32 || ss.SIMDAlignment(2) != 64 || ss.SIMDAlignment(9) != 1 {
		t.Fatal("SIMDAlignment values unexpected")
	}
	if ss.Sizeof(f32x4) != 16 || ss.Alignof(f32x4) != 16 {
		t.Fatalf("SourceSizes(Float32x4) = (%d, %d), want (16, 16)", ss.Sizeof(f32x4), ss.Alignof(f32x4))
	}
	if ss.HostBasicLayout(gotypes.Int64).Align != 8 || ss.HostBasicLayout(-1).Size != 0 {
		t.Fatal("HostBasicLayout unexpected")
	}

	// 32-bit ARM host layout policy (HostUsesNativeGoLayout == true).
	armSS := NewSourceSizes(SourceSizesConfig{
		Spec: target.Spec{GOOS: "linux", GOARCH: "arm"},
	})
	if !armSS.HostUsesNativeGoLayout() {
		t.Fatal("linux/arm should report HostUsesNativeGoLayout=true")
	}
}

func TestABIHelpersAndClassifyFuncABI(t *testing.T) {
	ctx := NewContext(target.Spec{GOOS: "linux", GOARCH: "arm64"}, nil)
	i8 := ctx.Basic(gotypes.Int8, GoPtr)
	u16 := ctx.Basic(gotypes.Uint16, GoPtr)
	f32 := ctx.Basic(gotypes.Float32, GoPtr)
	i64 := ctx.Basic(gotypes.Int64, GoPtr)
	empty := &Struct{}
	hfa := &Struct{Fields: []*Field{{Name: "x", Type: f32}, {Name: "y", Type: f32}, {Name: "z", Type: f32}}}
	smallAgg := &Struct{Fields: []*Field{{Name: "a", Type: i64}, {Name: "b", Type: i64}}}
	bigAgg := &Struct{Fields: []*Field{{Name: "a", Type: i64}, {Name: "b", Type: i64}, {Name: "c", Type: i64}}}

	// DataKindOf
	if DataKindOf(i64, false) != DataInteger || DataKindOf(i64, true) != DataIndirect {
		t.Fatal("DataKindOf(int64) mismatch")
	}
	if DataKindOf(f32, true) != DataBitCast || DataKindOf(ctx.Basic(gotypes.Float64, GoPtr), true) != DataIndirect {
		t.Fatal("DataKindOf(float) mismatch")
	}
	if DataKindOf(ctx.Pointer(i64, GoPtr), false) != DataPointer || DataKindOf(nil, false) != DataInvalid {
		t.Fatal("DataKindOf(ptr/nil) mismatch")
	}
	if DataKindOf(&Struct{Fields: []*Field{{Name: "p", Type: ctx.Pointer(i64, GoPtr)}}}, false) != DataPointer {
		t.Fatal("DataKindOf(single-field ptr struct) mismatch")
	}
	if DataKindOf(&Array{Elem: i8, Len: 1}, false) != DataInteger {
		t.Fatal("DataKindOf([1]int8) mismatch")
	}

	// AAPCS64 C ABI: HFA, small <=16B coerce, big >16B sret/byval, small ints sext/zext, zero-sized ignore.
	cSig := &Func{
		CallConv: CDecl,
		Params: []*Var{
			{Name: "a", Type: i8},
			{Name: "b", Type: u16},
			{Name: "z", Type: empty},
			{Name: "v", Type: hfa},
			{Name: "s", Type: smallAgg},
			{Name: "big", Type: bigAgg},
		},
		Results: []*Var{{Type: bigAgg}},
	}
	abi64 := ctx.FuncABI(cSig)
	if abi64 == nil || !abi64.HasSRet || abi64.Return.Mode != PassIndirect || !abi64.Return.SRet {
		t.Fatalf("arm64 big return FuncABI = %+v", abi64)
	}
	if !abi64.Params[0].SExt || !abi64.Params[1].ZExt || abi64.Params[2].Mode != PassIgnore {
		t.Fatalf("arm64 scalar/empty params = %+v", abi64.Params[:3])
	}
	if abi64.Params[3].Mode != PassDirect || abi64.Params[3].HFACount != 3 || abi64.Params[3].HFAElem != gotypes.Float32 {
		t.Fatalf("arm64 HFA param = %+v", abi64.Params[3])
	}
	if abi64.Params[4].Mode != PassCoerce || abi64.Params[5].Mode != PassIndirect || !abi64.Params[5].ByVal {
		t.Fatalf("arm64 aggregate params = %+v", abi64.Params[4:])
	}

	// Windows amd64: 16B aggregate is indirect (Win64 only passes 1/2/4/8B directly), no small-int ext.
	winCtx := NewContext(target.Spec{GOOS: "windows", GOARCH: "amd64"}, nil)
	winABI := winCtx.FuncABI(cSig)
	if winABI.Params[0].SExt || winABI.Params[1].ZExt || winABI.Params[4].Mode != PassIndirect {
		t.Fatalf("windows/amd64 FuncABI = %+v", winABI.Params)
	}

	// Linux amd64 SysV + Wasm + GoDecl multi-return.
	amd64Ctx := NewContext(target.Spec{GOOS: "linux", GOARCH: "amd64"}, nil)
	sysvABI := amd64Ctx.FuncABI(cSig)
	if sysvABI.Params[4].Mode != PassCoerce || sysvABI.Params[5].Mode != PassIndirect {
		t.Fatalf("linux/amd64 SysV FuncABI = %+v", sysvABI.Params[4:])
	}
	goSig := &Func{
		CallConv: GoDecl,
		Recv:     &Var{Name: "r", Type: ctx.Pointer(i64, GoPtr)},
		Params:   []*Var{{Name: "x", Type: i64}},
		Results:  []*Var{{Type: i64}, {Type: i8}},
		HasEnv:   true,
	}
	goABI := amd64Ctx.FuncABI(goSig)
	if !goABI.HasEnv || goABI.Return.Mode != PassDirect || len(goABI.Params) != 2 {
		t.Fatalf("GoDecl FuncABI = %+v", goABI)
	}
	wasmCtx := NewContext(target.Spec{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32", WasmProvider: "wasi"}, nil)
	wasmABI := wasmCtx.FuncABI(cSig)
	if !wasmABI.HasSRet || wasmABI.Return.Mode != PassIndirect {
		t.Fatalf("wasm FuncABI = %+v", wasmABI.Return)
	}
	for _, m := range []PassMode{PassIgnore, PassDirect, PassIndirect, PassCoerce, PassExpand} {
		if m.String() == "" {
			t.Fatalf("empty PassMode(%d).String()", m)
		}
	}
	if ClassifyFuncABI(target.Spec{}, nil, nil) != nil {
		t.Fatal("ClassifyFuncABI(nil) != nil")
	}

	// GoEqualName, GoHasPointers, GoPtrBytes, GoIsRegularMemory.
	amd64Sizes := amd64Ctx.SourceSizes()
	if !GoHasPointers(gotypes.Typ[gotypes.String]) || GoHasPointers(gotypes.Typ[gotypes.Int64]) {
		t.Fatal("GoHasPointers mismatch")
	}
	if GoPtrBytes(gotypes.NewInterfaceType(nil, nil), 8, amd64Sizes) != 16 {
		t.Fatal("GoPtrBytes(any) != 16")
	}
	if !GoIsRegularMemory(gotypes.NewStruct(nil, nil), 8, amd64Sizes) || GoIsRegularMemory(gotypes.Typ[gotypes.Float64], 8, amd64Sizes) {
		t.Fatal("GoIsRegularMemory mismatch")
	}
	if GoEqualName(gotypes.Typ[gotypes.String], 8, amd64Sizes) != "strequal" ||
		GoEqualName(gotypes.NewSlice(gotypes.Typ[gotypes.Byte]), 8, amd64Sizes) != "" ||
		GoEqualName(gotypes.NewInterfaceType(nil, nil), 8, amd64Sizes) != "nilinterequal" {
		t.Fatal("GoEqualName mismatch")
	}
}

func TestContextLookupSignaturesTuplesAndEdgeABIs(t *testing.T) {
	pkg := checkPkg(t, `package sample
import (
	"structs"
	"unsafe"
)
type CRec struct {
	P unsafe.Pointer
	F func(int) int
	Inner struct{ X int }
}
type CRecArr [2]CRec
type HostWrap struct {
	_ structs.HostLayout
	A byte
	B int64
}
type Node struct {
	Next *Node
}
type Reader interface {
	Read([]byte) (int, error)
}
type R struct{}
func (r *R) Do(a int, b ...string) (int, bool) { return a, true }
`)
	lookup := func(name string) (CallConv, bool) {
		switch name {
		case "example.com/sample.CRec", "example.com/sample.CRecArr":
			return CDecl, true
		}
		return CallConvUnknown, false
	}
	w32Spec := target.Spec{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32", WasmProvider: "wasi"}
	ctx := NewContext(w32Spec, lookup)

	// Map CRec, CRecArr, HostWrap, recursive Node, non-empty Interface, method Signature, Tuple, nil.
	if ctx.Type(nil, GoDecl) != nil {
		t.Fatal("ctx.Type(nil) != nil")
	}
	if ctx.Layout(nil).Align != 1 {
		t.Fatal("ctx.Layout(nil).Align != 1")
	}
	crec := ctx.Type(pkg.Scope().Lookup("CRec").Type(), CallConvUnknown)
	crecL := ctx.Layout(crec)
	if crecL.Size <= 0 {
		t.Fatalf("CRec Layout = %+v", crecL)
	}
	if !ctx.GoLayout(pkg.Scope().Lookup("HostWrap").Type(), GoDecl).HasHostLayout {
		t.Fatal("HostWrap Layout.HasHostLayout = false")
	}
	if !ctx.Layout(ctx.Array(ctx.Type(pkg.Scope().Lookup("HostWrap").Type(), GoDecl), 2, false)).HasHostLayout {
		t.Fatal("[2]HostWrap Layout.HasHostLayout = false")
	}
	node := ctx.Type(pkg.Scope().Lookup("Node").Type(), GoDecl)
	if node == nil || !ctx.Layout(node).HasWasmWideSlots {
		t.Fatal("Node on w32 should have WasmWideSlots")
	}
	if !ctx.GoLayout(pkg.Scope().Lookup("Reader").Type(), GoDecl).HasWasmWideSlots {
		t.Fatal("Reader interface on w32 should have WasmWideSlots")
	}
	// Method signature with receiver, variadic params, multi-results -> exercises mapSignatureLocked + Tuple Layout.
	rObj := pkg.Scope().Lookup("R").Type().(*gotypes.Named)
	doSig := rObj.Method(0).Type().(*gotypes.Signature)
	goClo := ctx.Type(doSig, GoDecl)
	if _, ok := goClo.(*Closure); !ok {
		t.Fatalf("Do mapped as %T, want *Closure", goClo)
	}
	cFn := ctx.Type(doSig, CDecl)
	if fn, ok := cFn.(*Func); !ok || fn.Recv == nil || !fn.Variadic || len(fn.Results) != 2 {
		t.Fatalf("Do CDecl mapped as %#v", cFn)
	}
	tupT := ctx.Type(doSig.Results(), GoDecl)
	tupL := ctx.Layout(tupT)
	if len(tupL.Fields) != 2 || tupL.IsRegularMemory {
		t.Fatalf("Tuple(int, bool) Layout = %+v", tupL)
	}
	if !ctx.Layout(&Tuple{Elems: []*Var{{Type: ctx.Basic(gotypes.Int64, GoPtr)}}}).IsRegularMemory {
		t.Fatal("single int64 Tuple should be RegularMemory")
	}
	if !ctx.Layout(&Struct{}).IsRegularMemory || !ctx.Layout(&Array{Elem: ctx.Basic(gotypes.Float64, GoPtr), Len: 0}).IsRegularMemory {
		t.Fatal("empty struct / zero-len float array should be RegularMemory")
	}
	if !ctx.Layout(&Struct{Fields: []*Field{{Name: "a", Type: ctx.Basic(gotypes.Int64, GoPtr)}}}).IsRegularMemory {
		t.Fatal("single-field int64 struct should be RegularMemory")
	}
	if ctx.Layout(&Struct{Fields: []*Field{{Name: "_", Type: ctx.Basic(gotypes.Int64, GoPtr)}}}).IsRegularMemory {
		t.Fatal("single blank-field struct must not be RegularMemory")
	}
	// Wasm wide slots for Func, Array, Tuple, UnsafePointer.
	if !ctx.Layout(&Func{CallConv: GoDecl}).HasWasmWideSlots ||
		ctx.Layout(&Func{CallConv: CDecl}).HasWasmWideSlots ||
		!ctx.Layout(ctx.Basic(gotypes.UnsafePointer, GoPtr)).HasWasmWideSlots ||
		!ctx.Layout(&Array{Elem: ctx.String(), Len: 2}).HasWasmWideSlots ||
		!ctx.Layout(&Tuple{Elems: []*Var{{Type: ctx.String()}}}).HasWasmWideSlots {
		t.Fatal("WasmWideSlots classification mismatch")
	}

	// SourceSizes with CallConvLookup on w32 exercises lookupNativeCallConv + hasLegacyNativeLayoutByLookup + WasmNativeLayout.
	ss := ctx.SourceSizes()
	if ss.Spec().GOARCH != "wasm" {
		t.Fatalf("SourceSizes.Spec() = %+v", ss.Spec())
	}
	if sz := ss.Sizeof(pkg.Scope().Lookup("CRecArr").Type()); sz <= 0 {
		t.Fatalf("w32 CRecArr Sizeof = %d", sz)
	}

	// Additional ClassifyFuncABI branches: 32-bit linux/386, windows/386, Win64 8B struct, array HFA, void return, pointer arg.
	f64 := ctx.Basic(gotypes.Float64, GoPtr)
	i32 := ctx.Basic(gotypes.Int32, GoPtr)
	hfaArr := &Array{Elem: f64, Len: 2}
	mixedStruct := &Struct{Fields: []*Field{{Name: "a", Type: f64}, {Name: "b", Type: i32}}}
	eightByteStruct := &Struct{Fields: []*Field{{Name: "a", Type: i32}, {Name: "b", Type: i32}}}
	sigExtra := &Func{
		CallConv: CDecl,
		Params: []*Var{
			{Type: ctx.Pointer(i32, NativePtr)},
			{Type: hfaArr},
			{Type: mixedStruct},
			{Type: eightByteStruct},
		},
	}
	armABI := NewContext(target.Spec{GOOS: "linux", GOARCH: "arm64"}, nil).FuncABI(sigExtra)
	if armABI.Return.Mode != PassIgnore || armABI.Params[0].Mode != PassDirect || armABI.Params[1].HFACount != 2 || armABI.Params[2].HFACount != 0 {
		t.Fatalf("arm64 extra FuncABI = %+v", armABI)
	}
	win64ABI := NewContext(target.Spec{GOOS: "windows", GOARCH: "amd64"}, nil).FuncABI(sigExtra)
	if win64ABI.Params[3].Mode != PassCoerce {
		t.Fatalf("win64 8B struct = %+v, want PassCoerce", win64ABI.Params[3])
	}
	for _, sp := range []target.Spec{
		{GOOS: "linux", GOARCH: "386"},
		{GOOS: "windows", GOARCH: "386"},
		{GOOS: "windows", GOARCH: "arm64"},
	} {
		ab := NewContext(sp, nil).FuncABI(sigExtra)
		if ab == nil || len(ab.Params) != 4 {
			t.Fatalf("FuncABI(%+v) = %+v", sp, ab)
		}
	}
}
