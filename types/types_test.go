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
	"strings"
	"sync"
	"testing"

	"github.com/xgo-dev/llgo/directive"
	"github.com/xgo-dev/llgo/target"
	"github.com/xgo-dev/llgo/types"
)

func TestContextCheckEndToEnd(t *testing.T) {
	src := `package demo

import "unsafe"

//llgo:cdecl
type CCallback func(a int32, args ...any) int32

//llgo:stdcall
type WinCallback func(a int32) int32

//llgo:type C
type LegacyC func()

//llgo:cdecl
type AliasC = func(int32)

type GoFn func(int32) int32

const (
	SizeCFn      = unsafe.Sizeof(CCallback(nil))
	SizeWinFn    = unsafe.Sizeof(WinCallback(nil))
	SizeLegacyC  = unsafe.Sizeof(LegacyC(nil))
	SizeAliasC   = unsafe.Sizeof(AliasC(nil))
	SizeGoFn     = unsafe.Sizeof(GoFn(nil))
	SizeRawGoFn  = unsafe.Sizeof((func(int32) int32)(nil))
)

type Node struct {
	Val  int64
	Next *Node
}

type Box[T any] struct {
	Item T
}

func (b *Box[T]) Get() T { return b.Item }

//llgo:link printf C.printf
func printf(fmt *int8, args ...any) int32

//llgo:link pyMax py.max
//llgo:pylist
func pyMax(args ...any) unsafe.Pointer

//llgo:skip
func skipped() {}

//go:wasmimport env log
func hostLog(x int32)

//go:wasmexport run
func exportedRun() {}

//llgo:link extVar C.errno
var extVar int32

var (
	boxInt    Box[int]
	boxString Box[string]
)

func Use() (int, string, *Node) {
	n := &Node{Val: 42}
	return boxInt.Get(), boxString.Get(), n
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "demo.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	spec := target.ResolveSpec(target.Spec{GOOS: "linux", GOARCH: "amd64"})
	ctx := types.NewContext(spec)
	pkg, info, err := ctx.Check("example.com/demo", fset, []*ast.File{file}, &gotypes.Config{
		Importer: importer.Default(),
	})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if pkg.Path() != "example.com/demo" || pkg.Name() != "demo" || !pkg.Complete() {
		t.Fatalf("unexpected pkg metadata: %s complete=%v", pkg, pkg.Complete())
	}
	if len(pkg.Imports()) != 1 || pkg.Imports()[0].Path() != "unsafe" {
		t.Fatalf("unexpected imports: %v", pkg.Imports())
	}

	// Verify compile-time unsafe.Sizeof constants folded by go/types using ctx.GoSizes()!
	checkConst := func(name string, want int64) {
		t.Helper()
		obj, ok := pkg.Lookup(name).(*types.Const)
		if !ok || obj == nil {
			t.Fatalf("const %s not found", name)
		}
		got, exact := constant.Int64Val(obj.Val())
		if !exact || got != want {
			t.Fatalf("const %s = %v, want %d", name, obj.Val(), want)
		}
	}
	checkConst("SizeCFn", 8)
	checkConst("SizeWinFn", 8)
	checkConst("SizeLegacyC", 8)
	checkConst("SizeAliasC", 8)
	checkConst("SizeGoFn", 16)
	checkConst("SizeRawGoFn", 16)

	// Verify Signature CallConv and single Variadic bool semantics.
	cCb := pkg.Lookup("CCallback").(*types.TypeName).Type().(*types.Named)
	cSig := cCb.Underlying().(*types.Signature)
	if cSig.CallConv() != types.CDecl || !cSig.IsNative() || cSig.IsClosure() || !cSig.Variadic() || !cSig.IsCVariadic() || cSig.IsGoVariadic() {
		t.Fatalf("CCallback signature = %s (cc=%v variadic=%v)", cSig, cSig.CallConv(), cSig.Variadic())
	}
	if ctx.Layout(cCb).Size != 8 || cCb.HasPtrData() {
		t.Fatalf("CCallback layout=%+v hasPtr=%v", ctx.Layout(cCb), cCb.HasPtrData())
	}

	winCb := pkg.Lookup("WinCallback").(*types.TypeName).Type().(*types.Named)
	if winCb.CallConv() != types.Stdcall || winCb.Underlying().(*types.Signature).CallConv() != types.Stdcall {
		t.Fatalf("WinCallback callconv = %v", winCb.CallConv())
	}

	goFn := pkg.Lookup("GoFn").(*types.TypeName).Type().(*types.Named)
	goSig := goFn.Underlying().(*types.Signature)
	if !goSig.IsClosure() || goSig.IsNative() || !goSig.HasPtrData() || ctx.Layout(goFn).Size != 16 {
		t.Fatalf("GoFn sig=%s layout=%+v", goSig, ctx.Layout(goFn))
	}

	// Verify //llgo:* and //go:wasm* directive attributes directly on Objects!
	printfFn := pkg.Lookup("printf").(*types.Func)
	if printfFn.LinkName() != "C.printf" || printfFn.CallConv() != types.CDecl || !printfFn.IsExternal() || !printfFn.Signature().IsCVariadic() {
		t.Fatalf("printf func metadata = link=%q cc=%v ext=%v sig=%s", printfFn.LinkName(), printfFn.CallConv(), printfFn.IsExternal(), printfFn.Signature())
	}

	pyFn := pkg.Lookup("pyMax").(*types.Func)
	if pyFn.LinkName() != "py.max" || pyFn.CallConv() != types.Py || !pyFn.IsPyList() || !pyFn.Signature().IsPyVariadic() {
		t.Fatalf("pyMax func metadata = link=%q cc=%v pylist=%v", pyFn.LinkName(), pyFn.CallConv(), pyFn.IsPyList())
	}

	skipFn := pkg.Lookup("skipped").(*types.Func)
	if !skipFn.IsSkip() {
		t.Fatalf("skipped.IsSkip() = false")
	}

	wasmImp := pkg.Lookup("hostLog").(*types.Func)
	if mod, name, ok := wasmImp.WasmImport(); !ok || mod != "env" || name != "log" || wasmImp.CallConv() != types.Wasm {
		t.Fatalf("hostLog wasmimport = (%q, %q, %v) cc=%v", mod, name, ok, wasmImp.CallConv())
	}

	wasmExp := pkg.Lookup("exportedRun").(*types.Func)
	if wasmExp.WasmExport() != "run" {
		t.Fatalf("exportedRun.WasmExport() = %q", wasmExp.WasmExport())
	}

	// Verify Foo.Type() and Go function value types are Identical (matching go/types!).
	useFn := pkg.Lookup("Use").(*types.Func)
	if useFn.CallConv() != types.Go || !useFn.Signature().IsClosure() {
		t.Fatalf("Use func CallConv = %v, want Go", useFn.CallConv())
	}
	wantUseSig := types.NewSignature(nil, nil, types.NewTuple(
		types.NewParam(0, nil, "", types.Typ[types.Int]),
		types.NewParam(0, nil, "", types.Typ[types.String]),
		types.NewParam(0, nil, "", types.NewPointer(pkg.Lookup("Node").Type())),
	), false)
	if !types.Identical(useFn.Type(), wantUseSig) {
		t.Fatalf("Func.Type() not Identical to equivalent Go Signature: %s vs %s", useFn.Type(), wantUseSig)
	}

	extV := pkg.Lookup("extVar").(*types.Var)
	if extV.LinkName() != "C.errno" || !extV.IsExternal() {
		t.Fatalf("extVar metadata = link=%q ext=%v", extV.LinkName(), extV.IsExternal())
	}

	// Verify generic instantiations are distinct *Named types sharing the same Origin!
	bi := pkg.Lookup("boxInt").(*types.Var).Type().(*types.Named)
	bs := pkg.Lookup("boxString").(*types.Var).Type().(*types.Named)
	if bi == bs || types.Identical(bi, bs) {
		t.Fatalf("Box[int] and Box[string] collided: %p vs %p", bi, bs)
	}
	if bi.Origin() != bs.Origin() {
		t.Fatalf("Box[int].Origin() != Box[string].Origin(): %p vs %p", bi.Origin(), bs.Origin())
	}
	if bi.TypeArgs().Len() != 1 || !types.Identical(bi.TypeArgs().At(0), types.Typ[types.Int]) {
		t.Fatalf("Box[int].TypeArgs() = %v", bi.TypeArgs())
	}
	if bs.TypeArgs().Len() != 1 || !types.Identical(bs.TypeArgs().At(0), types.Typ[types.String]) {
		t.Fatalf("Box[string].TypeArgs() = %v", bs.TypeArgs())
	}
	if ctx.Layout(bi).Size != 8 || ctx.Layout(bi).PtrBytes != 0 {
		t.Fatalf("Box[int] layout = %+v", ctx.Layout(bi))
	}
	if ctx.Layout(bs).Size != 16 || ctx.Layout(bs).PtrBytes != 8 {
		t.Fatalf("Box[string] layout = %+v", ctx.Layout(bs))
	}

	// Verify cyclic Node struct layout & GC pointer tracking.
	node := pkg.Lookup("Node").(*types.TypeName).Type().(*types.Named)
	nodeLay := ctx.Layout(node)
	if nodeLay.Size != 16 || nodeLay.Align != 8 || nodeLay.PtrBytes != 16 || len(nodeLay.PtrOffsets) != 1 || nodeLay.PtrOffsets[0] != 8 {
		t.Fatalf("Node layout = %+v", nodeLay)
	}
	if len(nodeLay.GCBitmap) != 1 || nodeLay.GCBitmap[0] != 0b10 {
		t.Fatalf("Node GCBitmap = %08b, want 0b10", nodeLay.GCBitmap)
	}

	// Verify MethodSet & Info.Selections speak native llgo/types!
	ms := ctx.MethodSet(ctx.Pointer(bi, types.GoPtr))
	sel := ms.Lookup(pkg, "Get")
	if sel == nil || sel.Kind() != types.MethodVal {
		t.Fatalf("MethodSet(*Box[int]).Lookup(Get) = %+v", sel)
	}
	if len(info.Selections) == 0 || len(info.Instances) == 0 || len(info.Types) == 0 {
		t.Fatalf("expected non-empty Info maps: selections=%d instances=%d types=%d", len(info.Selections), len(info.Instances), len(info.Types))
	}
	for expr, tv := range info.Types {
		if tv.Type != nil && info.TypeOf(expr) != tv.Type {
			t.Fatalf("Info.TypeOf mismatch for %T", expr)
		}
	}
}

func TestPointerKindsAndIdentical(t *testing.T) {
	ctx := types.NewContext(target.Spec{GOOS: "linux", GOARCH: "amd64"})
	i32 := ctx.Basic(types.Int32)

	goP := ctx.Pointer(i32, types.GoPtr)
	unsafeP := ctx.Pointer(i32, types.UnsafePtr)
	cP := ctx.Pointer(i32, types.CPtr)

	if !goP.HasPtrData() || !unsafeP.HasPtrData() || cP.HasPtrData() {
		t.Fatalf("HasPtrData mismatch: go=%v unsafe=%v c=%v", goP.HasPtrData(), unsafeP.HasPtrData(), cP.HasPtrData())
	}
	if types.Identical(goP, cP) || types.Identical(goP, unsafeP) {
		t.Fatal("different PtrKinds treated as Identical")
	}
	if ctx.Layout(goP).PtrBytes != 8 || ctx.Layout(unsafeP).PtrBytes != 8 || ctx.Layout(cP).PtrBytes != 0 {
		t.Fatalf("PtrBytes mismatch: go=%d unsafe=%d c=%d", ctx.Layout(goP).PtrBytes, ctx.Layout(unsafeP).PtrBytes, ctx.Layout(cP).PtrBytes)
	}

	// Signature identity distinguishes CallConv and Variadic.
	params := types.NewTuple(types.NewParam(0, nil, "x", i32))
	sGo := types.NewSignature(nil, params, nil, false)
	sC := types.NewSignatureWithCallConv(nil, params, nil, false, types.CDecl)
	sStd := sGo.WithCallConv(types.Stdcall)
	sCVar := types.NewSignatureWithCallConv(nil, params, nil, true, types.CDecl)
	if types.Identical(sGo, sC) || types.Identical(sC, sStd) || types.Identical(sC, sCVar) {
		t.Fatal("distinct Signatures treated as Identical")
	}
	if !types.Identical(sC, types.NewSignatureWithCallConv(nil, params, nil, false, types.CDecl)) {
		t.Fatal("identical CDecl Signatures not Identical")
	}
}

// TestGoSizesMidCheckUnresolvedPointerTargets reproduces go/types checking
// runtime/chan.go where unsafe.Sizeof(hchan{}) runs while *sudog / *g / *m
// point to not-yet-checked Named types (preventing go/types/named.go:608 panic).
func TestGoSizesMidCheckUnresolvedPointerTargets(t *testing.T) {
	src := `package rtlike

import "unsafe"

const hchanSize = unsafe.Sizeof(hchan{})

type hchan struct {
	qcount uint
	sendq  waitq
	recvq  waitq
	lock   mutex
}

type waitq struct {
	first *sudog
	last  *sudog
}

type sudog struct {
	g    *g
	next *sudog
	prev *sudog
	elem unsafe.Pointer
}

type g struct {
	m      *m
	waiter *sudog
	iface  any
	custom CustomIface
}

type CustomIface interface {
	Next() *m
}

type m struct {
	curg *g
	c    *hchan
}

type mutex struct {
	key uintptr
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "rtlike.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	ctx := types.NewContext(target.Spec{GOOS: "linux", GOARCH: "amd64"})
	pkg, _, err := ctx.Check("rtlike", fset, []*ast.File{file}, &gotypes.Config{
		Importer: importer.Default(),
	})
	if err != nil {
		t.Fatalf("Check failed on mutually-recursive runtime-like types: %v", err)
	}
	cObj := pkg.Lookup("hchanSize").(*types.Const)
	got, ok := constant.Int64Val(cObj.Val())
	if !ok || got != 48 {
		t.Fatalf("hchanSize = %v, want 48", cObj.Val())
	}
}

func TestSingleEngineLayoutsHostWasmSIMDAndOverflow(t *testing.T) {
	t.Run("HostLayout on 32-bit ARM vs 386", func(t *testing.T) {
		structsPkg := gotypes.NewPackage("structs", "structs")
		hostMarker := gotypes.NewNamed(gotypes.NewTypeName(0, structsPkg, "HostLayout", nil), gotypes.NewStruct(nil, nil), nil)
		goSt := gotypes.NewStruct([]*gotypes.Var{
			gotypes.NewField(0, nil, "_", hostMarker, false),
			gotypes.NewField(0, nil, "b", gotypes.Typ[gotypes.Byte], false),
			gotypes.NewField(0, nil, "v", gotypes.Typ[gotypes.Int64], false),
		}, nil)
		outerSt := gotypes.NewStruct([]*gotypes.Var{
			gotypes.NewField(0, nil, "pad", gotypes.Typ[gotypes.Byte], false),
			gotypes.NewField(0, nil, "inner", goSt, false),
		}, nil)

		armCtx := types.NewContext(target.Spec{GOOS: "linux", GOARCH: "arm"})
		if !armCtx.HostUsesNativeGoLayout() {
			t.Fatal("expected arm HostUsesNativeGoLayout = true")
		}
		llHostMarker := armCtx.FromGo(hostMarker)
		llGoSt := armCtx.FromGo(goSt)
		llOuterSt := armCtx.FromGo(outerSt)
		if !llHostMarker.IsHostLayoutMarker() || llGoSt.IsHostLayoutMarker() {
			t.Fatalf("IsHostLayoutMarker mismatch: marker=%v st=%v", llHostMarker.IsHostLayoutMarker(), llGoSt.IsHostLayoutMarker())
		}
		if !llGoSt.HasHostLayout() || !llGoSt.ContainsHostLayout() || llOuterSt.HasHostLayout() || !llOuterSt.ContainsHostLayout() {
			t.Fatalf("HostLayout methods mismatch: st=(%v,%v) outer=(%v,%v)", llGoSt.HasHostLayout(), llGoSt.ContainsHostLayout(), llOuterSt.HasHostLayout(), llOuterSt.ContainsHostLayout())
		}
		// Real stdlib shape: struct { _ structs.hostLayout } and defined types of it.
		innerHost := gotypes.NewNamed(gotypes.NewTypeName(0, structsPkg, "hostLayout", nil), gotypes.NewStruct(nil, nil), nil)
		realMarkerBody := gotypes.NewStruct([]*gotypes.Var{gotypes.NewField(0, structsPkg, "_", innerHost, false)}, nil)
		myHost := gotypes.NewNamed(gotypes.NewTypeName(0, nil, "MyHost", nil), realMarkerBody, nil)
		if !armCtx.FromGo(myHost).IsHostLayoutMarker() {
			t.Fatal("defined type of real structs.HostLayout shape should be recognized as IsHostLayoutMarker")
		}
		armSizes := armCtx.GoSizes()
		if got := armSizes.Alignof(goSt); got != 8 {
			t.Fatalf("arm HostLayout Alignof = %d, want 8", got)
		}
		if got := armSizes.Sizeof(goSt); got != 16 {
			t.Fatalf("arm HostLayout Sizeof = %d, want 16", got)
		}
		if got := armSizes.Sizeof(outerSt); got != 24 {
			t.Fatalf("arm outer HostLayout Sizeof = %d, want 24", got)
		}

		i386Ctx := types.NewContext(target.Spec{GOOS: "linux", GOARCH: "386"})
		if i386Ctx.HostUsesNativeGoLayout() {
			t.Fatal("expected linux/386 HostUsesNativeGoLayout = false")
		}
		if got := i386Ctx.GoSizes().Alignof(goSt); got != 4 {
			t.Fatalf("linux/386 HostLayout Alignof = %d, want 4", got)
		}
	})

	t.Run("Wasm32 wide Go vs native C layout", func(t *testing.T) {
		spec := target.ResolveSpec(target.Spec{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32", WasmProvider: "wasi"})
		ctx := types.NewContext(spec)
		ctx.SetTypeCallConv("native.CRecord", types.CDecl)

		npkg := gotypes.NewPackage("native", "native")
		cRec := gotypes.NewNamed(
			gotypes.NewTypeName(0, npkg, "CRecord", nil),
			gotypes.NewStruct([]*gotypes.Var{
				gotypes.NewField(0, npkg, "p", gotypes.NewPointer(gotypes.Typ[gotypes.Byte]), false),
				gotypes.NewField(0, npkg, "u", gotypes.Typ[gotypes.Uintptr], false),
			}, nil),
			nil,
		)
		goRec := gotypes.NewStruct([]*gotypes.Var{
			gotypes.NewField(0, nil, "p", gotypes.NewPointer(gotypes.Typ[gotypes.Byte]), false),
			gotypes.NewField(0, nil, "c", cRec, false),
		}, nil)

		sizes := ctx.GoSizes()
		if got := sizes.Sizeof(cRec); got != 8 {
			t.Fatalf("Wasm32 native CRecord Sizeof = %d, want 8", got)
		}
		if got := sizes.Alignof(cRec); got != 4 {
			t.Fatalf("Wasm32 native CRecord Alignof = %d, want 4", got)
		}
		goLay := ctx.Layout(ctx.FromGo(goRec))
		if goLay.Size != 16 || !goLay.HasWasmWideSlots || !goLay.Fields[0].WasmWideSlot || goLay.Fields[1].Offset != 8 {
			t.Fatalf("Wasm32 mixed Go/C layout = %+v", goLay)
		}
	})

	t.Run("SIMD vector tags and AArch64 vs AMD64 alignment", func(t *testing.T) {
		spkg := gotypes.NewPackage("simd/archsimd", "archsimd")
		v256 := gotypes.NewNamed(
			gotypes.NewTypeName(0, spkg, "v256", nil),
			gotypes.NewStruct([]*gotypes.Var{
				gotypes.NewField(0, spkg, "_", gotypes.NewArray(gotypes.Typ[gotypes.Int8], 0), false),
			}, nil),
			nil,
		)
		i32x8 := gotypes.NewNamed(
			gotypes.NewTypeName(0, spkg, "Int32x8", nil),
			gotypes.NewStruct([]*gotypes.Var{
				gotypes.NewField(0, spkg, "_", v256, false),
				gotypes.NewField(0, spkg, "x", gotypes.NewArray(gotypes.Typ[gotypes.Int32], 8), false),
			}, nil),
			nil,
		)

		amd64Ctx := types.NewContext(target.Spec{GOOS: "linux", GOARCH: "amd64"})
		llV256 := amd64Ctx.FromGo(v256)
		llI32x8 := amd64Ctx.FromGo(i32x8)
		if llV256.SIMDTagIndex() != 1 || llV256.SIMDVec() != nil || !llV256.ContainsSIMDStorage() {
			t.Fatalf("v256 SIMD methods mismatch: tag=%d vec=%v contains=%v", llV256.SIMDTagIndex(), llV256.SIMDVec(), llV256.ContainsSIMDStorage())
		}
		if llI32x8.SIMDTagIndex() != -1 || !llI32x8.ContainsSIMDStorage() {
			t.Fatalf("Int32x8 SIMD methods mismatch: tag=%d contains=%v", llI32x8.SIMDTagIndex(), llI32x8.ContainsSIMDStorage())
		}
		amdLay := amd64Ctx.Layout(llI32x8)
		if amdLay.Size != 32 || amdLay.Align != 32 || !amdLay.HasSIMDStorage {
			t.Fatalf("amd64 Int32x8 layout = %+v", amdLay)
		}
		if vec := llI32x8.SIMDVec(); vec == nil || vec.Len() != 8 || vec.BitWidth() != 256 || vec.IsMask() {
			t.Fatalf("Int32x8 SIMDVec = %+v", vec)
		}

		arm64Ctx := types.NewContext(target.Spec{GOOS: "darwin", GOARCH: "arm64"})
		armLay := arm64Ctx.Layout(arm64Ctx.FromGo(i32x8))
		if armLay.Size != 32 || armLay.Align != 16 || !armLay.HasSIMDStorage {
			t.Fatalf("arm64 Int32x8 layout = %+v", armLay)
		}
	})

	t.Run("Legacy closure struct and overflow protection", func(t *testing.T) {
		ctx := types.NewContext(target.Spec{GOOS: "linux", GOARCH: "amd64"})
		sig := gotypes.NewSignatureType(nil, nil, nil, nil, nil, false)
		cloSt := gotypes.NewStruct([]*gotypes.Var{
			gotypes.NewField(0, nil, "$f", sig, false),
			gotypes.NewField(0, nil, "$data", gotypes.Typ[gotypes.UnsafePointer], false),
		}, nil)
		lay := ctx.Layout(ctx.FromGo(cloSt))
		if lay.Size != 16 || len(lay.Fields) != 2 || lay.Fields[0].Offset != 0 || lay.Fields[1].Offset != 8 {
			t.Fatalf("legacy closure struct layout = %+v", lay)
		}
		if types.AlignSize(math.MaxInt64, 8) != -1 {
			t.Fatal("AlignSize overflow did not return -1")
		}
		if p := types.ArrayLayout(types.LayoutPair{Size: math.MaxInt64 / 2, Align: 8}, 4); p.Size != -1 {
			t.Fatalf("ArrayLayout overflow = %+v", p)
		}
	})
}

func TestFuncABIAndRuntimeMetadata(t *testing.T) {
	// AAPCS64 HFA + Darwin arm64 narrow int extension.
	darwinArm64 := target.ResolveSpec(target.Spec{GOOS: "darwin", GOARCH: "arm64"})
	ctxArm64 := types.NewContext(darwinArm64)
	vec2 := types.NewStruct([]*types.Var{
		types.NewField(0, nil, "x", types.Typ[types.Float32], false),
		types.NewField(0, nil, "y", types.Typ[types.Float32], false),
	}, nil)
	big := types.NewStruct([]*types.Var{
		types.NewField(0, nil, "a", types.Typ[types.Int64], false),
		types.NewField(0, nil, "b", types.Typ[types.Int64], false),
		types.NewField(0, nil, "c", types.Typ[types.Int64], false),
	}, nil)
	sig := types.NewSignatureWithCallConv(
		nil,
		types.NewTuple(
			types.NewParam(0, nil, "b", types.Typ[types.Int8]),
			types.NewParam(0, nil, "u", types.Typ[types.Uint16]),
			types.NewParam(0, nil, "v", vec2),
		),
		types.NewTuple(types.NewParam(0, nil, "", big)),
		false,
		types.CDecl,
	)
	abi := ctxArm64.FuncABI(sig)
	if !abi.Params[0].SExt || !abi.Params[1].ZExt {
		t.Fatalf("darwin/arm64 narrow int ext = %+v, %+v", abi.Params[0], abi.Params[1])
	}
	if abi.Params[2].HFACount != 2 || abi.Params[2].HFAElem != types.Float32 {
		t.Fatalf("arm64 HFA = %+v", abi.Params[2])
	}
	if !abi.HasSRet || !abi.Return.SRet || abi.Return.Mode != types.PassIndirect {
		t.Fatalf("arm64 big return = %+v", abi.Return)
	}

	// Win64 bool-only ZExt + 8B aggregate Coerce vs 24B SRet.
	win64 := target.ResolveSpec(target.Spec{GOOS: "windows", GOARCH: "amd64"})
	ctxWin := types.NewContext(win64)
	winSig := types.NewSignatureWithCallConv(
		nil,
		types.NewTuple(
			types.NewParam(0, nil, "flag", types.Typ[types.Bool]),
			types.NewParam(0, nil, "small", types.Typ[types.Int8]),
			types.NewParam(0, nil, "v", vec2),
		),
		types.NewTuple(types.NewParam(0, nil, "", big)),
		false,
		types.CDecl,
	)
	winABI := ctxWin.FuncABI(winSig)
	if !winABI.Params[0].ZExt || winABI.Params[1].SExt || winABI.Params[2].Mode != types.PassCoerce || !winABI.HasSRet {
		t.Fatalf("win64 ABI = %+v", winABI)
	}

	// RV64 sign-extends both int32 and uint32; embedded physical triples override GOARCH.
	rv64 := target.ResolveSpec(target.Spec{GOOS: "linux", GOARCH: "riscv64"})
	if types.NativeIntExtForSpec(rv64) != types.NativeIntRV64Ext {
		t.Fatal("riscv64 NativeIntExt mismatch")
	}
	esp32c3 := target.ResolveSpec(target.Spec{GOOS: "linux", GOARCH: "arm", LLVMTarget: "riscv32-esp-elf"})
	if types.NativeIntExtForSpec(esp32c3) != types.NativeIntNarrowExt {
		t.Fatal("riscv32 physical triple NativeIntExt mismatch")
	}
	winMinGW := target.ResolveSpec(target.Spec{GOOS: "windows", GOARCH: "amd64", LLVMTarget: "x86_64-w64-mingw32"})
	if types.NativeIntExtForSpec(winMinGW) != types.NativeIntBoolExt {
		t.Fatal("mingw64 physical triple NativeIntExt mismatch")
	}
	darwinTriple := target.ResolveSpec(target.Spec{GOOS: "linux", GOARCH: "arm64", LLVMTarget: "arm64-apple-macosx14.0.0"})
	if types.NativeIntExtForSpec(darwinTriple) != types.NativeIntNarrowExt {
		t.Fatal("darwin arm64 physical triple NativeIntExt mismatch")
	}

	// Package.PackageKind and Package.LLGoFiles speak native llgo/types.
	cpkg := types.NewPackage("example.com/clib", "clib")
	cpkg.Scope().Insert(types.NewConst(token.NoPos, cpkg, "LLGoPackage", types.Typ[types.UntypedString], constant.MakeString("link: -lm")))
	cpkg.Scope().Insert(types.NewConst(token.NoPos, cpkg, "LLGoFiles", types.Typ[types.UntypedString], constant.MakeString("a.c; b.c")))
	if kind, param := cpkg.PackageKind(); kind != directive.PkgLinkExtern || param != " -lm" {
		t.Fatalf("Package.PackageKind = (%v, %q)", kind, param)
	}
	if files, ok := cpkg.LLGoFiles(); !ok || files != "a.c; b.c" {
		t.Fatalf("Package.LLGoFiles = (%q, %v)", files, ok)
	}
	if kind, param := (*types.Package)(nil).PackageKind(); kind != directive.PkgNormal || param != "" {
		t.Fatalf("nil Package.PackageKind = (%v, %q)", kind, param)
	}
	if files, ok := (*types.Package)(nil).LLGoFiles(); ok || files != "" {
		t.Fatalf("nil Package.LLGoFiles = (%q, %v)", files, ok)
	}

	// DataKindOf & EqualName & IsRegularMemory.
	wrap := types.NewStruct([]*types.Var{
		types.NewField(0, nil, "v", types.Typ[types.Int64], false),
	}, nil)
	dk, inner, depth := types.DataKindOf(wrap, false)
	if dk != types.DataInteger || inner != types.Typ[types.Int64] || depth != 1 {
		t.Fatalf("DataKindOf(wrap, 64bit) = (%v, %v, %d)", dk, inner, depth)
	}
	dk32, _, _ := types.DataKindOf(wrap, true)
	if dk32 != types.DataIndirect {
		t.Fatalf("DataKindOf(wrap, 32bit) = %v, want DataIndirect", dk32)
	}
	if ctxArm64.EqualName(types.NewArray(types.NewStruct(nil, nil), 4)) != "memequal0" {
		t.Fatal("EqualName([4]struct{}) != memequal0")
	}
	if ctxArm64.EqualName(types.NewSlice(types.Typ[types.Int])) != "" {
		t.Fatal("EqualName([]int) should be empty")
	}
	padded := types.NewStruct([]*types.Var{
		types.NewField(0, nil, "a", types.Typ[types.Byte], false),
		types.NewField(0, nil, "b", types.Typ[types.Int64], false),
	}, nil)
	if ctxArm64.Layout(padded).IsRegularMemory {
		t.Fatal("padded struct should not be regular memory")
	}
	packed := types.NewStruct([]*types.Var{
		types.NewField(0, nil, "a", types.Typ[types.Int64], false),
		types.NewField(0, nil, "b", types.Typ[types.Int64], false),
	}, nil)
	if !ctxArm64.Layout(packed).IsRegularMemory {
		t.Fatal("packed int64 struct should be regular memory")
	}
}

func TestContextConcurrencyAndFormatting(t *testing.T) {
	ctx := types.NewContext(target.Spec{GOOS: "linux", GOARCH: "amd64"})
	goSt := gotypes.NewStruct([]*gotypes.Var{
		gotypes.NewField(0, nil, "a", gotypes.Typ[gotypes.Int64], false),
		gotypes.NewField(0, nil, "p", gotypes.NewPointer(gotypes.Typ[gotypes.String]), false),
	}, nil)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			llT := ctx.FromGo(goSt)
			lay := ctx.Layout(llT)
			if lay.Size != 16 || lay.PtrBytes != 16 {
				t.Errorf("concurrent layout = %+v", lay)
			}
			_ = ctx.EqualName(llT)
		}()
	}
	wg.Wait()

	// Exercise String() and helper coverage on objects/scopes/types.
	pkg := types.NewPackage("example.com/p", "p")
	v := types.NewVar(0, pkg, "x", types.Typ[types.Int])
	pkg.Scope().Insert(v)
	if _, found := pkg.Scope().LookupParent("x", token.NoPos); found != v {
		t.Fatalf("Scope.LookupParent(x) = %v", found)
	}
	if !strings.Contains(v.String(), "var x int") {
		t.Fatalf("Var.String() = %q", v.String())
	}
	vec := ctx.SIMDVec(types.Int32, 4, true)
	if vec.String() != "mask<4xint32>" {
		t.Fatalf("SIMDVec.String() = %q", vec.String())
	}
}
