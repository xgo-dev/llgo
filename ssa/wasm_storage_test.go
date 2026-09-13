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
	"go/importer"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func newJ32Program(t *testing.T) Program {
	t.Helper()
	prog := NewProgram(&Target{
		GOOS:        "js",
		GOARCH:      "wasm",
		LLVMTarget:  "wasm32-unknown-emscripten",
		WasmProfile: "j32",
	})
	prog.TypeSizes(&types.StdSizes{WordSize: 8, MaxAlign: 8})
	t.Cleanup(prog.Dispose)
	return prog
}

func TestWasm32UsesGo64WordStorage(t *testing.T) {
	prog := newJ32Program(t)
	if got := prog.PointerSize(); got != 4 {
		t.Fatalf("physical pointer size = %d, want 4", got)
	}
	if got := prog.GoWordSize(); got != 8 {
		t.Fatalf("Go word size = %d, want 8", got)
	}
	ptr := prog.Pointer(prog.Byte())
	if got := prog.SizeOf(prog.Int()); got != 8 {
		t.Fatalf("int size = %d, want 8", got)
	}
	if got := prog.SizeOf(ptr); got != 8 {
		t.Fatalf("pointer storage size = %d, want 8", got)
	}
	if got := prog.AlignOf(ptr); got != 8 {
		t.Fatalf("pointer storage alignment = %d, want 8", got)
	}
	array := prog.rawType(types.NewArray(ptr.RawType(), 2))
	if got := prog.SizeOf(array); got != 16 {
		t.Fatalf("[2]*byte size = %d, want 16", got)
	}
	if got := prog.AlignOf(array); got != 8 {
		t.Fatalf("[2]*byte alignment = %d, want 8", got)
	}
	box := prog.rawType(types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "P", ptr.RawType(), false),
		types.NewField(token.NoPos, nil, "B", types.Typ[types.Byte], false),
	}, nil))
	if got := prog.OffsetOf(box, 1); got != 8 {
		t.Fatalf("pointer-following field offset = %d, want 8", got)
	}
	if got := prog.SizeOf(box); got != 16 {
		t.Fatalf("pointer box size = %d, want 16", got)
	}
}

func TestNativeStorageViewsAreCachedAndWasmOnly(t *testing.T) {
	prog := newJ32Program(t)
	word := prog.Uintptr()
	first := prog.withNativeStorage(word)
	if first == word {
		t.Fatal("J32 native uintptr storage reused the Go64 view")
	}
	if second := prog.withNativeStorage(word); second != first {
		t.Fatal("J32 native uintptr storage was not cached")
	}
	if got := len(prog.nativeStorage); got != 2 {
		t.Fatalf("J32 native storage cache contains %d entries, want 2", got)
	}
	words := prog.rawType(types.NewArray(types.Typ[types.Uintptr], 2))
	nativeWords := prog.withNativeStorage(words)
	if got := nativeWords.ll.String(); got != "[2 x i32]" {
		t.Fatalf("J32 native uintptr array type = %s, want [2 x i32]", got)
	}

	native := NewProgram(nil)
	t.Cleanup(native.Dispose)
	nativeWord := native.Uintptr()
	if got := native.withNativeStorage(nativeWord); got != nativeWord {
		t.Fatal("native target created an unnecessary storage view")
	}
	if native.nativeStorage != nil {
		t.Fatal("native target allocated the wasm storage-view cache")
	}
	if got := native.nativeStorageLLVMType(types.Typ[types.Uintptr], nativeWord.ll); got != nativeWord.ll {
		t.Fatalf("native uintptr storage type = %s, want %s", got, nativeWord.ll)
	}
}

func TestWasm32StorageIntegerConversions(t *testing.T) {
	prog := newJ32Program(t)
	setTestRuntime(t, prog)
	pkg := prog.NewPackage("p", "example.com/p")
	word := types.Typ[types.Int]
	param := types.NewParam(token.NoPos, nil, "word", word)
	result := types.NewParam(token.NoPos, nil, "", word)
	sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(param), types.NewTuple(result), false)

	fn := pkg.NewFunc("example.com/p.narrowInt", sig, InGo)
	b := fn.MakeBody(1)
	b.fitLLVMValue(b.Param(0).impl, prog.Int(), prog.Int32().ll)
	b.Return(b.Param(0))
	b.EndBuild()

	extended := prog.fitLLVMConstant(
		llvm.ConstInt(prog.Int32().ll, ^uint64(0), true), prog.Int32(), prog.Int().ll,
	)
	if got := extended.Type().IntTypeWidth(); got != 64 {
		t.Fatalf("extended signed native int width = %d, want 64", got)
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("out-of-range signed native integer constant did not fail")
			}
		}()
		prog.fitLLVMConstant(prog.IntVal(uint64(1)<<31, prog.Int()).impl, prog.Int(), prog.Int32().ll)
	}()

	ir := pkg.Module().String()
	for _, want := range []string{"sext i32", "trunc i64", "icmp ne i64", "AssertRuntimeError"} {
		if !strings.Contains(ir, want) {
			t.Fatalf("J32 signed native boundary IR does not contain %q:\n%s", want, ir)
		}
	}
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatalf("invalid J32 signed native-boundary module: %v\n%s", err, ir)
	}
}

func TestWasm32PhysicalPointerIndexBounds(t *testing.T) {
	prog := newJ32Program(t)
	pkg := prog.NewPackage("p", "example.com/p")
	fn := pkg.NewFunc("example.com/p.index", NoArgsNoRet, InGo)
	b := fn.MakeBody(1)

	i32 := prog.Int32()
	alreadyPhysical := Expr{llvm.ConstInt(i32.ll, 0, false), i32}
	if got := b.physicalPointerIndex(alreadyPhysical); got.impl != alreadyPhysical.impl {
		t.Fatal("physical i32 pointer index was unnecessarily converted")
	}
	i16 := prog.Type(types.Typ[types.Int16], InGo)
	narrow := Expr{llvm.ConstInt(i16.ll, 0, false), i16}
	if got := b.physicalPointerIndex(narrow); got.impl != narrow.impl {
		t.Fatal("narrow pointer index was unnecessarily converted")
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("non-integer pointer index did not fail")
			}
		}()
		b.physicalPointerIndex(prog.FloatVal(0, prog.Float32()))
	}()
	b.Return()
	b.EndBuild()
}

func TestWasm32PointerSlotsConvertAtMemoryBoundary(t *testing.T) {
	prog := newJ32Program(t)
	pkg := prog.NewPackage("p", "example.com/p")
	fn := pkg.NewFunc("example.com/p.storeLoad", NoArgsNoRet, InC)
	b := fn.MakeBody(1)
	ptr := prog.Pointer(prog.Byte())
	value := pkg.NewVarEx("example.com/p.target", ptr)
	global := pkg.NewVarEx("example.com/p.value", prog.Pointer(ptr))
	global.Init(value.Expr)
	slot := b.Alloc(ptr, false)
	b.Store(slot, value.Expr)
	b.Load(slot)
	b.Return()

	ir := pkg.Module().String()
	for _, want := range []string{
		"alloca { ptr, i32 }, align 8",
		"global { ptr, i32 }",
		"ptr @\"example.com/p.target\", i32 0",
		"ptrtoint (ptr",
		"to i32",
		"zext i32",
		"load i64",
		"trunc i64",
		"inttoptr i32",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("J32 pointer slot IR does not contain %q:\n%s", want, ir)
		}
	}
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatalf("invalid J32 pointer-storage module: %v\n%s", err, ir)
	}
}

func TestWasm32GEPIndexesUsePhysicalAddressWidth(t *testing.T) {
	prog := newJ32Program(t)
	setTestRuntime(t, prog)
	prog.disableBoundsChecks = true
	pkg := prog.NewPackage("p", "example.com/p")
	index := types.NewParam(token.NoPos, nil, "index", types.Typ[types.Int])

	str := types.NewParam(token.NoPos, nil, "value", types.Typ[types.String])
	byteResult := types.NewParam(token.NoPos, nil, "", types.Typ[types.Byte])
	indexSig := types.NewSignatureType(nil, nil, nil, types.NewTuple(str, index), types.NewTuple(byteResult), false)
	indexFn := pkg.NewFunc("example.com/p.index", indexSig, InGo)
	ib := indexFn.MakeBody(1)
	ib.Return(ib.Index(ib.Param(0), ib.Param(1), func() (Expr, bool) { return Nil, false }))
	ib.EndBuild()

	ptrType := types.NewPointer(types.Typ[types.Byte])
	ptr := types.NewParam(token.NoPos, nil, "ptr", ptrType)
	ptrResult := types.NewParam(token.NoPos, nil, "", ptrType)
	advanceSig := types.NewSignatureType(nil, nil, nil, types.NewTuple(ptr, index), types.NewTuple(ptrResult), false)
	advanceFn := pkg.NewFunc("example.com/p.advance", advanceSig, InGo)
	ab := advanceFn.MakeBody(1)
	ab.Return(ab.Advance(ab.Param(0), ab.Param(1)))
	ab.EndBuild()

	arrayType := types.NewArray(types.Typ[types.Byte], 16)
	array := types.NewParam(token.NoPos, nil, "array", types.NewPointer(arrayType))
	addrResult := types.NewParam(token.NoPos, nil, "", types.NewPointer(types.Typ[types.Byte]))
	addrSig := types.NewSignatureType(nil, nil, nil, types.NewTuple(array, index), types.NewTuple(addrResult), false)
	addrFn := pkg.NewFunc("example.com/p.addr", addrSig, InGo)
	adb := addrFn.MakeBody(1)
	adb.Return(adb.IndexAddr(adb.Param(0), adb.Param(1)))
	adb.EndBuild()

	ir := pkg.Module().String()
	if got := strings.Count(ir, "trunc i64"); got < 3 {
		t.Fatalf("J32 GEP paths contain %d physical-index truncations, want at least 3:\n%s", got, ir)
	}
	var physicalGEPs int
	for _, line := range strings.Split(ir, "\n") {
		if !strings.Contains(line, "getelementptr") || !strings.Contains(line, "i8, ptr") {
			continue
		}
		if strings.Contains(line, ", i64 ") {
			t.Fatalf("J32 GEP retains an i64 index: %s\n%s", line, ir)
		}
		if strings.Contains(line, ", i32 ") {
			physicalGEPs++
		}
	}
	if physicalGEPs < 3 {
		t.Fatalf("J32 GEP paths contain %d i32 indexes, want at least 3:\n%s", physicalGEPs, ir)
	}
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatalf("invalid J32 GEP module: %v\n%s", err, ir)
	}
}

func TestWasm32AdvanceRespectsGoAndNativePointerSlots(t *testing.T) {
	prog := newJ32Program(t)
	pkg := prog.NewPackage("p", "example.com/p")
	elem := types.NewPointer(types.Typ[types.Byte])
	param := types.NewParam(token.NoPos, nil, "base", types.NewPointer(elem))
	sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(param), nil, false)

	for _, tc := range []struct {
		name string
		bg   Background
	}{
		{name: "goAdvance", bg: InGo},
		{name: "cAdvance", bg: InC},
	} {
		fn := pkg.NewFunc("example.com/p."+tc.name, sig, tc.bg)
		b := fn.MakeBody(1)
		base := b.Param(0)
		if tc.bg == InC {
			base.Type = prog.Type(param.Type(), InC)
		}
		b.Advance(base, prog.IntVal(1, prog.Int()))
		b.Return()
		b.EndBuild()
	}

	ir := pkg.Module().String()
	if !strings.Contains(ir, "getelementptr { ptr, i32 }") {
		t.Fatalf("Go pointer arithmetic must use eight-byte pointer slots:\n%s", ir)
	}
	if !strings.Contains(ir, "getelementptr ptr") {
		t.Fatalf("C pointer arithmetic must use physical four-byte pointer slots:\n%s", ir)
	}
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatalf("invalid J32 pointer-arithmetic module: %v\n%s", err, ir)
	}
}

func TestWasm32NativeBoundaryKeepsPhysicalWordWidth(t *testing.T) {
	prog := newJ32Program(t)
	setTestRuntime(t, prog)
	pkg := prog.NewPackage("p", "example.com/p")
	word := types.Typ[types.Uintptr]
	param := types.NewParam(token.NoPos, nil, "word", word)
	result := types.NewParam(token.NoPos, nil, "", word)
	sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(param), types.NewTuple(result), false)

	goFn := pkg.NewFunc("example.com/p.goWord", sig, InGo)
	gb := goFn.MakeBody(1)
	gb.Return(gb.Param(0))
	gb.EndBuild()

	cFn := pkg.NewFunc("example.com/p.cWord", sig, InC)
	cb := cFn.MakeBody(1)
	cb.Return(cb.Param(0))
	cb.EndBuild()

	caller := pkg.NewFunc("example.com/p.callCWord", sig, InGo)
	callb := caller.MakeBody(1)
	callb.Return(callb.Call(cFn.Expr, callb.Param(0)))
	callb.EndBuild()

	results := types.NewTuple(result, result)
	multiSig := types.NewSignatureType(nil, nil, nil, types.NewTuple(param), results, false)
	multiCallee := pkg.NewFunc("example.com/p.cWords", multiSig, InC)
	mcb := multiCallee.MakeBody(1)
	mcb.Return(mcb.Param(0), mcb.Param(0))
	mcb.EndBuild()
	multiCaller := pkg.NewFunc("example.com/p.callCWords", multiSig, InGo)
	mb := multiCaller.MakeBody(1)
	multiResult := mb.Call(multiCallee.Expr, mb.Param(0))
	mb.Return(mb.Extract(multiResult, 0), mb.Extract(multiResult, 1))
	mb.EndBuild()

	if got := goFn.impl.Param(0).Type().IntTypeWidth(); got != 64 {
		t.Fatalf("Go uintptr parameter width = %d, want 64", got)
	}
	if got := cFn.impl.Param(0).Type().IntTypeWidth(); got != 32 {
		t.Fatalf("C uintptr parameter width = %d, want 32", got)
	}
	ir := pkg.Module().String()
	for _, want := range []string{
		"zext i32",
		"trunc i64",
		"zext i32",
		"icmp ne i64",
		"AssertRuntimeError",
		"call i32 @\"example.com/p.cWord\"(i32",
		"call { i32, i32 } @\"example.com/p.cWords\"(i32",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("J32 native boundary IR does not contain %q:\n%s", want, ir)
		}
	}
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatalf("invalid J32 native-boundary module: %v\n%s", err, ir)
	}
}

func TestWasm32NativeStructAndGCRootStorage(t *testing.T) {
	prog := newJ32Program(t)
	setTestRuntime(t, prog)
	typesPkg := types.NewPackage("example.com/p", "p")
	name := types.NewTypeName(token.NoPos, typesPkg, "CIovec", nil)
	fields := []*types.Var{
		types.NewField(token.NoPos, typesPkg, "data", types.Typ[types.UnsafePointer], false),
		types.NewField(token.NoPos, typesPkg, "size", types.Typ[types.Uintptr], false),
	}
	cIovec := types.NewNamed(name, types.NewStruct(fields, nil), nil)
	prog.SetTypeBackground("example.com/p.CIovec", InC)
	typ := prog.Type(cIovec, InC)
	if got := prog.SizeOf(typ); got != 8 {
		t.Fatalf("native wasm32 iovec size = %d, want 8", got)
	}
	if got := prog.OffsetOf(typ, 1); got != 4 {
		t.Fatalf("native wasm32 iovec size field offset = %d, want 4", got)
	}
	nativeWord := prog.Type(types.Typ[types.Uintptr], InC)
	if got := prog.Zero(nativeWord).impl.Type(); got != nativeWord.ll {
		t.Fatalf("native wasm32 zero word type = %s, want %s", got, nativeWord.ll)
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("out-of-range native wasm32 integer constant did not fail")
			}
		}()
		prog.toStorageConstant(nativeWord, prog.IntVal(uint64(1)<<32, prog.Uintptr()).impl)
	}()
	if got := prog.Zero(typ).impl.Type(); got != typ.ll {
		t.Fatalf("native wasm32 zero type = %s, want %s", got, typ.ll)
	}

	pkg := prog.NewPackage("p", "example.com/p")
	fn := pkg.NewFunc("example.com/p.root", NoArgsNoRet, InGo)
	b := fn.MakeBody(1)
	iovec := b.Alloc(typ, false)
	size := b.FieldAddr(iovec, 1)
	b.Store(size, prog.IntVal(7, prog.Uintptr()))
	b.Load(size).SetVolatile(true)
	roots := fn.NewGCRoots(2)
	b.SetGCRoot(roots[0], prog.Nil(prog.VoidPtr()))
	b.Return()
	b.EndBuild()

	ir := pkg.Module().String()
	if !strings.Contains(ir, "[2 x { ptr, i32 }]") {
		t.Fatalf("GC root frame must match the Go64 runtime layout:\n%s", ir)
	}
	if !strings.Contains(ir, "@llvm_gc_root_chain = linkonce global { ptr, i32 }") {
		t.Fatalf("GC root-chain storage must match the Go64 runtime global:\n%s", ir)
	}
	if !strings.Contains(ir, "store i32 7") || !strings.Contains(ir, "load volatile i32") {
		t.Fatalf("native wasm32 uintptr fields must use i32 memory operations:\n%s", ir)
	}
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatalf("invalid J32 native-structure/root module: %v\n%s", err, ir)
	}
}

func TestWasm32DirectInterfaceUnwrapsPointerStorage(t *testing.T) {
	prog := newJ32Program(t)
	prog.SetRuntime(func() *types.Package {
		pkg, err := importer.For("source", nil).Import(PkgRuntime)
		if err != nil {
			t.Fatal(err)
		}
		return pkg
	})
	pkg := prog.NewPackage("p", "example.com/p")
	pointer := types.NewPointer(types.Typ[types.Byte])
	structWrapper := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "P", pointer, false),
	}, nil)
	arrayWrapper := types.NewArray(pointer, 1)
	empty := types.NewInterfaceType(nil, nil)
	empty.Complete()
	for name, wrapper := range map[string]types.Type{
		"boxStruct": structWrapper,
		"boxArray":  arrayWrapper,
	} {
		param := types.NewParam(token.NoPos, nil, "value", wrapper)
		result := types.NewParam(token.NoPos, nil, "", empty)
		sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(param), types.NewTuple(result), false)
		fn := pkg.NewFunc("example.com/p."+name, sig, InGo)
		b := fn.MakeBody(1)
		b.Return(b.MakeInterface(prog.Type(empty, InGo), b.Param(0)))
		b.EndBuild()
	}

	ir := pkg.Module().String()
	if !strings.Contains(ir, "extractvalue") || !strings.Contains(ir, "extractvalue { ptr, i32 }") {
		t.Fatalf("direct interface conversion did not unwrap wide pointer storage:\n%s", ir)
	}
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatalf("invalid J32 direct-interface module: %v\n%s", err, ir)
	}
}
