//go:build !llgo
// +build !llgo

/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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
	"fmt"
	"go/constant"
	"go/importer"
	"go/token"
	"go/types"
	"os"
	"runtime"
	"sort"
	"strings"
	"testing"
	"unsafe"

	"github.com/goplus/gogen/packages"
	"github.com/goplus/llgo/internal/semmeta"
	"github.com/goplus/llvm"
)

func TestEndDefer(t *testing.T) {
	prog := NewProgram(nil)
	pkg := prog.NewPackage("foo", "foo")
	fn := pkg.NewFunc("main", NoArgsNoRet, InC)
	b := fn.MakeBody(1)
	fn.defer_ = &aDefer{}
	fn.endDefer(b)
}

func TestDeferToPendingLoopCasesAndFallback(t *testing.T) {
	prog := NewProgram(nil)
	prog.SetRuntime(func() *types.Package {
		fset := token.NewFileSet()
		imp := packages.NewImporter(fset)
		pkg, _ := imp.Import(PkgRuntime)
		return pkg
	})
	pkg := prog.NewPackage("foo", "foo")

	callee := pkg.NewFunc("callee", NoArgsNoRet, InGo)
	cb := callee.MakeBody(1)
	cb.Return()
	cb.EndBuild()

	fn := pkg.NewFunc("main", NoArgsNoRet, InGo)
	fn.SetRecover(fn.MakeBlock())
	b := fn.MakeBody(1)

	stack := b.DeferStack()
	if stack.IsNil() {
		t.Fatal("defer stack should be available with recover")
	}
	fn.defer_ = nil
	fn.pendingLoopCases = nil
	fn.nextDeferID = 0

	b.DeferTo(fn, stack, callee.Expr, Builder.Call)
	if got := len(fn.pendingLoopCases); got != 1 {
		t.Fatalf("pendingLoopCases len = %d, want 1", got)
	}
	if fn.nextDeferID != 1 {
		t.Fatalf("nextDeferID = %d, want 1", fn.nextDeferID)
	}

	self := b.getDeferInCurrentBlock()
	if self == nil {
		t.Fatal("expected current-block defer stack")
	}
	if len(fn.pendingLoopCases) != 0 {
		t.Fatal("pendingLoopCases should be consumed once defer stack is materialized")
	}
	if got := len(self.loopCases); got != 1 {
		t.Fatalf("loopCases len = %d, want 1", got)
	}

	b.DeferTo(fn, stack, callee.Expr, Builder.Call)
	if got := len(self.loopCases); got != 2 {
		t.Fatalf("loopCases len after direct append = %d, want 2", got)
	}

	fallback := pkg.NewFunc("fallback", NoArgsNoRet, InGo)
	fallback.SetRecover(fallback.MakeBlock())
	fb := fallback.MakeBody(1)
	fb.Return()
	fb.SetBlockEx(fallback.Block(0), BeforeLast, true)
	fb.DeferTo(nil, prog.Nil(prog.VoidPtr()), callee.Expr, Builder.Call)
	if fallback.defer_ == nil || len(fallback.defer_.loopCases) != 1 {
		t.Fatal("owner=nil should fall back to normal loop defer")
	}
}

func TestUnsafeString(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	os.Chdir("../../runtime")
	defer os.Chdir(wd)
	prog := NewProgram(nil)
	prog.SetRuntime(func() *types.Package {
		fset := token.NewFileSet()
		imp := packages.NewImporter(fset)
		pkg, _ := imp.Import(PkgRuntime)
		return pkg
	})
	pkg := prog.NewPackage("foo", "foo")
	b := pkg.NewFunc("main", NoArgsNoRet, InC).MakeBody(1)
	b.Println(b.BuiltinCall("String", b.CStr("hello"), prog.Val(5)))
	b.Return()
}

func TestTooManyConditionalDefers(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	os.Chdir("../../runtime")
	defer os.Chdir(wd)

	prog := NewProgram(nil)
	prog.SetRuntime(func() *types.Package {
		fset := token.NewFileSet()
		imp := packages.NewImporter(fset)
		pkg, _ := imp.Import(PkgRuntime)
		return pkg
	})

	pkg := prog.NewPackage("foo", "foo")
	target := pkg.NewFunc("f", NoArgsNoRet, InGo)
	fn := pkg.NewFunc("main", NoArgsNoRet, InGo)
	fn.SetRecover(fn.MakeBlock())
	b := fn.MakeBody(1)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic: too many conditional defers")
		} else if r != "too many conditional defers" {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()

	b.Return()
	for i := 0; i < 65; i++ {
		b.Defer(DeferInCond, target.Expr, Builder.Call)
	}
}

func TestPointerSize(t *testing.T) {
	expected := unsafe.Sizeof(uintptr(0))
	if size := NewProgram(nil).PointerSize(); size != int(expected) {
		t.Fatal("bad PointerSize:", size)
	}
}

func TestNewFuncExLLVMUsed(t *testing.T) {
	prog := NewProgram(nil)
	pkg := prog.NewPackage("main", "main")
	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)

	// Mark the exported name before function creation so NewFuncEx can protect it via llvm.compiler.used.
	pkg.SetExport("main.Foo", "Foo")
	pkg.SetExport("main.Bar", "Bar")
	pkg.NewFunc("Foo", sig, InGo)
	pkg.NewFunc("Bar", sig, InGo)
	pkg.NewFunc("Baz", sig, InGo)
	pkg.MaterializePreserveSyms()

	used := pkg.Module().NamedGlobal("llvm.compiler.used")
	if used.IsNil() {
		t.Fatal("missing llvm.compiler.used")
	}
	if got := used.Linkage(); got != llvm.AppendingLinkage {
		t.Fatalf("llvm.compiler.used linkage = %v, want %v", got, llvm.AppendingLinkage)
	}
	if got := used.Section(); got != "llvm.metadata" {
		t.Fatalf("llvm.compiler.used section = %q, want %q", got, "llvm.metadata")
	}
	if got := pkg.String(); !strings.Contains(got, `@llvm.compiler.used = appending global [2 x ptr] [ptr @Foo, ptr @Bar], section "llvm.metadata"`) {
		t.Fatalf("module missing llvm.compiler.used entry:\n%s", got)
	}
}

func TestSetBlock(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Log("SetBlock: no error?")
		}
	}()
	fn := &aFunction{}
	b := &aBuilder{Func: fn}
	b.SetBlock(&aBasicBlock{})
}

func TestSetBlockEx(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Log("SetBlockEx: no error?")
		}
	}()
	fn := &aFunction{}
	b := &aBuilder{Func: fn}
	b.SetBlockEx(&aBasicBlock{fn: fn}, -1, false)
}

func TestSetPython(t *testing.T) {
	prog := NewProgram(nil)
	typ := types.NewPackage("foo", "foo")
	prog.SetPython(typ)
}

func TestClosureCtx(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Log("closureCtx: no error?")
		}
	}()
	var f aFunction
	f.closureCtx(nil)
}

func TestClosureNoCtxValue(t *testing.T) {
	prog := NewProgram(nil)
	pkg := prog.NewPackage("bar", "foo/bar")
	params := types.NewTuple(types.NewVar(0, nil, "x", types.Typ[types.Int]))
	rets := types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Int]))
	sig := types.NewSignatureType(nil, nil, nil, params, rets, false)
	fn := pkg.NewFunc("fn", sig, InGo)
	b := fn.MakeBody(1)
	b.Return(fn.Param(0))

	holderSig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	holder := pkg.NewFunc("holder", holderSig, InGo)
	hb := holder.MakeBody(1)
	closureT := prog.Closure(sig)
	ptr := hb.AllocaT(closureT)
	hb.Store(ptr, fn.Expr)
	hb.Return()

	assertPkg(t, pkg, `; ModuleID = 'foo/bar'
source_filename = "foo/bar"

; Function Attrs: null_pointer_is_valid
define i64 @fn(i64 %0) #0 {
_llgo_0:
  ret i64 %0
}

; Function Attrs: null_pointer_is_valid
define void @holder() #0 {
_llgo_0:
  %0 = alloca { ptr, ptr }, align 8
  store { ptr, ptr } { ptr @__llgo_stub.fn, ptr null }, ptr %0, align 8
  ret void
}

define linkonce i64 @__llgo_stub.fn(ptr %0, i64 %1) {
_llgo_0:
  %2 = tail call i64 @fn(i64 %1)
  ret i64 %2
}

attributes #0 = { null_pointer_is_valid }
`)
}

func TestClosureFuncPtrValue(t *testing.T) {
	prog := NewProgram(nil)
	prog.SetRuntime(func() *types.Package {
		fset := token.NewFileSet()
		imp := packages.NewImporter(fset)
		pkg, _ := imp.Import(PkgRuntime)
		return pkg
	})
	pkg := prog.NewPackage("bar", "foo/bar")

	params := types.NewTuple(types.NewVar(0, nil, "x", types.Typ[types.Int]))
	rets := types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Int]))
	sig := types.NewSignatureType(nil, nil, nil, params, rets, false)
	fn := pkg.NewFunc("fn", sig, InGo)
	b := fn.MakeBody(1)
	b.Return(fn.Param(0))

	holderSig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	holder := pkg.NewFunc("holder", holderSig, InGo)
	hb := holder.MakeBody(1)
	closureT := prog.Closure(sig)
	ptr := hb.AllocaT(closureT)
	fnPtrType := prog.rawType(sig)
	fnPtr := hb.ChangeType(fnPtrType, fn.Expr)
	hb.Store(ptr, fnPtr)
	hb.Return()

	wrapName := "__llgo_stub." + prog.abi.FuncName(sig)
	wrapRef := wrapName
	if strings.Contains(wrapName, "$") {
		wrapRef = fmt.Sprintf("\"%s\"", wrapName)
	}
	expected := fmt.Sprintf(`; ModuleID = 'foo/bar'
source_filename = "foo/bar"

; Function Attrs: null_pointer_is_valid
define i64 @fn(i64 %%0) #0 {
_llgo_0:
  ret i64 %%0
}

; Function Attrs: null_pointer_is_valid
define void @holder() #0 {
_llgo_0:
  %%0 = alloca { ptr, ptr }, align 8
  %%1 = call ptr @"github.com/goplus/llgo/runtime/internal/runtime.AllocU"(i64 8)
  store ptr @fn, ptr %%1, align 8
  %%2 = insertvalue { ptr, ptr } { ptr @%s, ptr undef }, ptr %%1, 1
  store { ptr, ptr } %%2, ptr %%0, align 8
  ret void
}

define linkonce i64 @%s(ptr %%0, i64 %%1) {
_llgo_0:
  %%2 = load ptr, ptr %%0, align 8
  %%3 = tail call i64 %%2(i64 %%1)
  ret i64 %%3
}

; Function Attrs: null_pointer_is_valid
declare ptr @"github.com/goplus/llgo/runtime/internal/runtime.AllocU"(i64) #0

attributes #0 = { null_pointer_is_valid }
`, wrapRef, wrapRef)
	assertPkg(t, pkg, expected)
}

func TestConvertNamedStructValue(t *testing.T) {
	prog := NewProgram(nil)
	pkg := prog.NewPackage("bar", "foo/bar")
	tpkg := types.NewPackage("foo/bar", "bar")

	st := types.NewStruct([]*types.Var{
		types.NewField(0, tpkg, "sec", types.Typ[types.Int64], false),
		types.NewField(0, tpkg, "nsec", types.Typ[types.Int64], false),
		types.NewField(0, tpkg, "loc", types.NewPointer(types.Typ[types.Int8]), false),
	}, nil)
	srcNamed := types.NewNamed(types.NewTypeName(0, tpkg, "SrcTime", nil), st, nil)
	dstNamed := types.NewNamed(types.NewTypeName(0, tpkg, "DstTime", nil), st, nil)

	sig := types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(types.NewParam(0, tpkg, "v", srcNamed)),
		types.NewTuple(types.NewParam(0, tpkg, "", dstNamed)),
		false,
	)
	fn := pkg.NewFunc("convertNamed", sig, InGo)
	b := fn.MakeBody(1)
	src := fn.Param(0)
	dst := b.Convert(prog.Type(dstNamed, InGo), src)
	b.Return(dst)

	ir := fn.impl.String()
	if strings.Contains(ir, `ret %"foo/bar.SrcTime"`) {
		t.Fatalf("named struct convert returned source type:\n%s", ir)
	}
	if !strings.Contains(ir, `ret %"foo/bar.DstTime"`) {
		t.Fatalf("named struct convert did not return destination type:\n%s", ir)
	}
}

func TestConvertStringFromWideIntegers(t *testing.T) {
	prog := NewProgram(nil)
	prog.SetRuntime(func() *types.Package {
		fset := token.NewFileSet()
		imp := packages.NewImporter(fset)
		pkg, _ := imp.Import(PkgRuntime)
		return pkg
	})
	pkg := prog.NewPackage("bar", "foo/bar")

	params := types.NewTuple(
		types.NewVar(0, nil, "i64", types.Typ[types.Int64]),
		types.NewVar(0, nil, "u64", types.Typ[types.Uint64]),
		types.NewVar(0, nil, "i32", types.Typ[types.Int32]),
		types.NewVar(0, nil, "u32", types.Typ[types.Uint32]),
	)
	rets := types.NewTuple(
		types.NewVar(0, nil, "", types.Typ[types.String]),
		types.NewVar(0, nil, "", types.Typ[types.String]),
		types.NewVar(0, nil, "", types.Typ[types.String]),
		types.NewVar(0, nil, "", types.Typ[types.String]),
	)
	sig := types.NewSignatureType(nil, nil, nil, params, rets, false)
	fn := pkg.NewFunc("convertStrings", sig, InGo)
	b := fn.MakeBody(1)
	str := prog.Type(types.Typ[types.String], InGo)
	b.Return(
		b.Convert(str, fn.Param(0)),
		b.Convert(str, fn.Param(1)),
		b.Convert(str, fn.Param(2)),
		b.Convert(str, fn.Param(3)),
	)

	ir := fn.impl.String()
	for _, want := range []string{
		`StringFromInt64"(i64 %0)`,
		`StringFromUint64"(i64 %1)`,
		`sext i32 %2 to i64`,
		`StringFromInt64"(i64 %6)`,
		`zext i32 %3 to i64`,
		`StringFromUint64"(i64 %8)`,
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("missing %q in IR:\n%s", want, ir)
		}
	}
}

func TestCallClosureDynamic(t *testing.T) {
	prog := NewProgram(nil)
	pkg := prog.NewPackage("bar", "foo/bar")

	params := types.NewTuple(types.NewVar(0, nil, "x", types.Typ[types.Int]))
	rets := types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Int]))
	sig := types.NewSignatureType(nil, nil, nil, params, rets, false)
	callerParams := types.NewTuple(
		types.NewVar(0, nil, "f", sig),
		types.NewVar(0, nil, "x", types.Typ[types.Int]),
	)
	callerSig := types.NewSignatureType(nil, nil, nil, callerParams, rets, false)
	caller := pkg.NewFunc("caller", callerSig, InGo)
	b := caller.MakeBody(1)
	b.Return(b.Call(caller.Param(0), caller.Param(1)))

	assertPkg(t, pkg, `; ModuleID = 'foo/bar'
source_filename = "foo/bar"

; Function Attrs: null_pointer_is_valid
define i64 @caller({ ptr, ptr } %0, i64 %1) #0 {
_llgo_0:
  %2 = extractvalue { ptr, ptr } %0, 1
  %3 = extractvalue { ptr, ptr } %0, 0
  %4 = call i64 %3(ptr %2, i64 %1)
  ret i64 %4
}

attributes #0 = { null_pointer_is_valid }
`)
}

func TestMakeClosureWithCtx(t *testing.T) {
	prog := NewProgram(nil)
	prog.SetRuntime(func() *types.Package {
		fset := token.NewFileSet()
		imp := packages.NewImporter(fset)
		pkg, _ := imp.Import(PkgRuntime)
		return pkg
	})
	pkg := prog.NewPackage("bar", "foo/bar")
	ctxFields := []*types.Var{types.NewField(0, nil, "x", types.Typ[types.Int], false)}
	ctxStruct := types.NewStruct(ctxFields, nil)
	ctxPtr := types.NewPointer(ctxStruct)
	ctxParam := types.NewParam(0, nil, "__llgo_ctx", ctxPtr)
	innerParams := types.NewTuple(ctxParam, types.NewVar(0, nil, "y", types.Typ[types.Int]))
	innerRets := types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Int]))
	innerSig := types.NewSignatureType(nil, nil, nil, innerParams, innerRets, false)
	inner := pkg.NewFunc("inner", innerSig, InGo)
	ib := inner.MakeBody(1)
	ib.Return(inner.Param(1))

	outerParams := types.NewTuple(types.NewVar(0, nil, "x", types.Typ[types.Int]))
	outerRetSig := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(types.NewVar(0, nil, "y", types.Typ[types.Int])),
		innerRets, false)
	outerSig := types.NewSignatureType(nil, nil, nil, outerParams,
		types.NewTuple(types.NewVar(0, nil, "", outerRetSig)), false)
	outer := pkg.NewFunc("outer", outerSig, InGo)
	ob := outer.MakeBody(1)
	closure := ob.MakeClosure(inner.Expr, []Expr{outer.Param(0)})
	ob.Return(closure)

	assertPkg(t, pkg, `; ModuleID = 'foo/bar'
source_filename = "foo/bar"

; Function Attrs: null_pointer_is_valid
define i64 @inner(ptr %0, i64 %1) #0 {
_llgo_0:
  ret i64 %1
}

; Function Attrs: null_pointer_is_valid
define { ptr, ptr } @outer(i64 %0) #0 {
_llgo_0:
  %1 = call ptr @"github.com/goplus/llgo/runtime/internal/runtime.AllocU"(i64 8)
  %2 = getelementptr inbounds { i64 }, ptr %1, i32 0, i32 0
  store i64 %0, ptr %2, align 8
  %3 = insertvalue { ptr, ptr } { ptr @inner, ptr undef }, ptr %1, 1
  ret { ptr, ptr } %3
}

; Function Attrs: null_pointer_is_valid
declare ptr @"github.com/goplus/llgo/runtime/internal/runtime.AllocU"(i64) #0

attributes #0 = { null_pointer_is_valid }
`)
}

func TestCvtClosureDropsRecv(t *testing.T) {
	prog := NewProgram(nil)
	pkg := types.NewPackage("foo", "foo")
	iface := types.NewInterfaceType(nil, nil)
	namedIface := types.NewNamed(types.NewTypeName(0, pkg, "IFmt", nil), iface, nil)
	recv := types.NewVar(0, pkg, "recv", namedIface)
	params := types.NewTuple(VArg())
	rets := types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Int]))
	sig := types.NewSignatureType(recv, nil, nil, params, rets, true)

	st := prog.gocvt.cvtClosure(sig)
	fnSig, ok := st.Field(0).Type().(*types.Signature)
	if !ok {
		t.Fatalf("closure field[0] not signature: %T", st.Field(0).Type())
	}
	if fnSig.Recv() != nil {
		t.Fatalf("closure signature should not keep recv: %v", fnSig.Recv())
	}
	if !fnSig.Variadic() {
		t.Fatal("closure signature should be variadic")
	}
	if fnSig.Params().Len() != 1 {
		t.Fatalf("closure signature should have 1 param for variadic, got %d", fnSig.Params().Len())
	}
	if fnSig.Params().At(0).Name() != NameValist {
		t.Fatalf("closure signature param name mismatch: got %q, want %q",
			fnSig.Params().At(0).Name(), NameValist)
	}
}

func TestIfaceMethodClosureCallIR(t *testing.T) {
	prog := NewProgram(nil)
	prog.SetRuntime(func() *types.Package {
		fset := token.NewFileSet()
		imp := packages.NewImporter(fset)
		pkg, _ := imp.Import(PkgRuntime)
		return pkg
	})
	pkgTypes := types.NewPackage("foo/bar", "bar")
	rawSig := types.NewSignatureType(nil, nil, nil, types.NewTuple(VArg()),
		types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Int])), true)
	rawMeth := types.NewFunc(0, pkgTypes, "Printf", rawSig)
	rawIface := types.NewInterfaceType([]*types.Func{rawMeth}, nil)
	rawIface.Complete()
	namedIface := types.NewNamed(types.NewTypeName(0, pkgTypes, "IFmt", nil), rawIface, nil)
	pkgTypes.Scope().Insert(namedIface.Obj())
	recv := types.NewVar(0, pkgTypes, "recv", namedIface)
	recvSig := types.NewSignatureType(recv, nil, nil, types.NewTuple(VArg()),
		types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Int])), true)
	recvMeth := types.NewFunc(0, pkgTypes, "Printf", recvSig)

	pkg := prog.NewPackage("bar", "foo/bar")
	callerSig := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(types.NewVar(0, pkgTypes, "i", namedIface)),
		types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Int])), false)
	caller := pkg.NewFunc("caller", callerSig, InGo)
	b := caller.MakeBody(1)
	closure := b.Imethod(caller.Param(0), recvMeth)
	ret := b.Call(closure, prog.Val(100), prog.Val(200))
	b.Return(ret)

	assertPkg(t, pkg, `; ModuleID = 'foo/bar'
source_filename = "foo/bar"

%"github.com/goplus/llgo/runtime/internal/runtime.iface" = type { ptr, ptr }

; Function Attrs: null_pointer_is_valid
define i64 @caller(%"github.com/goplus/llgo/runtime/internal/runtime.iface" %0) #0 {
_llgo_0:
  %1 = call ptr @"github.com/goplus/llgo/runtime/internal/runtime.IfacePtrData"(%"github.com/goplus/llgo/runtime/internal/runtime.iface" %0)
  %2 = extractvalue %"github.com/goplus/llgo/runtime/internal/runtime.iface" %0, 0
  %3 = getelementptr ptr, ptr %2, i64 3
  %4 = load ptr, ptr %3, align 8
  %5 = insertvalue { ptr, ptr } undef, ptr %4, 0
  %6 = insertvalue { ptr, ptr } %5, ptr %1, 1
  %7 = extractvalue { ptr, ptr } %6, 1
  %8 = extractvalue { ptr, ptr } %6, 0
  %9 = call i64 (ptr, ...) %8(ptr %7, i64 100, i64 200)
  ret i64 %9
}

; Function Attrs: null_pointer_is_valid
declare ptr @"github.com/goplus/llgo/runtime/internal/runtime.IfacePtrData"(%"github.com/goplus/llgo/runtime/internal/runtime.iface") #0

attributes #0 = { null_pointer_is_valid }

!llgo.useifacemethod = !{!0}
!llgo.interfaceinfo = !{!1}

!0 = !{!"caller", !"_llgo_foo/bar.IFmt", !"Printf", !"_llgo_func$_RYiBYcSxJjuvzYmA4xYm18hT18pH0_ng6z76aK77Bk"}
!1 = !{!"_llgo_foo/bar.IFmt", !"Printf", !"_llgo_func$_RYiBYcSxJjuvzYmA4xYm18hT18pH0_ng6z76aK77Bk"}
`)

	useIfaceMethodRows := pkg.Module().NamedMetadataOperands(semmeta.UseIfaceMethodMetadata)
	requireMetadataRows(t, semmeta.UseIfaceMethodMetadata, useIfaceMethodRows, 1)
	useIfaceMethodFields := requireMetadataFields(t, semmeta.UseIfaceMethodMetadata, useIfaceMethodRows[0], 4)
	requireMDString(t, semmeta.UseIfaceMethodMetadata, useIfaceMethodFields, 0, "caller")
	requireMDString(t, semmeta.UseIfaceMethodMetadata, useIfaceMethodFields, 1, "_llgo_foo/bar.IFmt")
	requireMDString(t, semmeta.UseIfaceMethodMetadata, useIfaceMethodFields, 2, "Printf")
	requireMDString(t, semmeta.UseIfaceMethodMetadata, useIfaceMethodFields, 3, "_llgo_func$_RYiBYcSxJjuvzYmA4xYm18hT18pH0_ng6z76aK77Bk")

	interfaceInfoRows := pkg.Module().NamedMetadataOperands(semmeta.InterfaceInfoMetadata)
	requireMetadataRows(t, semmeta.InterfaceInfoMetadata, interfaceInfoRows, 1)
	interfaceInfoFields := requireMetadataFields(t, semmeta.InterfaceInfoMetadata, interfaceInfoRows[0], 3)
	requireMDString(t, semmeta.InterfaceInfoMetadata, interfaceInfoFields, 0, "_llgo_foo/bar.IFmt")
	requireMDString(t, semmeta.InterfaceInfoMetadata, interfaceInfoFields, 1, "Printf")
	requireMDString(t, semmeta.InterfaceInfoMetadata, interfaceInfoFields, 2, "_llgo_func$_RYiBYcSxJjuvzYmA4xYm18hT18pH0_ng6z76aK77Bk")
}

func TestNamedMetadataReadback(t *testing.T) {
	prog := NewProgram(nil)
	prog.sizes = types.SizesFor("gc", runtime.GOARCH)
	prog.SetRuntime(func() *types.Package {
		pkg, err := importer.For("source", nil).Import(PkgRuntime)
		if err != nil {
			t.Fatal(err)
		}
		return pkg
	})

	pkgTypes := types.NewPackage("foo/bar", "bar")

	rawIfaceSig := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(types.NewVar(0, nil, "x", types.Typ[types.Int])),
		types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Int])),
		false)
	rawIfaceMeth := types.NewFunc(0, pkgTypes, "Add", rawIfaceSig)
	rawIface := types.NewInterfaceType([]*types.Func{rawIfaceMeth}, nil)
	rawIface.Complete()

	namedIface := types.NewNamed(types.NewTypeName(0, pkgTypes, "IFmt", nil), rawIface, nil)
	pkgTypes.Scope().Insert(namedIface.Obj())

	ifaceRecv := types.NewVar(0, pkgTypes, "recv", namedIface)
	ifaceMethSig := types.NewSignatureType(ifaceRecv, nil, nil,
		types.NewTuple(types.NewVar(0, nil, "x", types.Typ[types.Int])),
		types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Int])),
		false)
	ifaceMeth := types.NewFunc(0, pkgTypes, "Add", ifaceMethSig)

	namedStruct := types.NewNamed(types.NewTypeName(0, pkgTypes, "S", nil), types.NewStruct(nil, nil), nil)
	pkgTypes.Scope().Insert(namedStruct.Obj())
	structRecv := types.NewVar(0, pkgTypes, "recv", types.NewPointer(namedStruct))
	structMethSig := types.NewSignatureType(structRecv, nil, nil,
		types.NewTuple(types.NewVar(0, nil, "x", types.Typ[types.Int])),
		types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Int])),
		false)
	structMeth := types.NewFunc(0, pkgTypes, "Add", structMethSig)
	namedStruct.AddMethod(structMeth)

	pkg := prog.NewPackage("bar", "foo/bar")
	caller := pkg.NewFunc("caller", types.NewSignatureType(nil, nil, nil, nil,
		types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Int])), false), InGo)
	b := caller.MakeBody(1)
	ptrType := prog.Type(types.NewPointer(namedStruct), InGo)
	ifaceType := prog.Type(namedIface, InGo)
	x := prog.Zero(ptrType)
	intf := b.MakeInterface(ifaceType, x)
	closure := b.Imethod(intf, ifaceMeth)
	b.Return(b.Call(closure, prog.Val(1)))

	mtypName, _ := prog.abi.TypeName(funcType(prog, structMeth.Type()))
	structTypeName, _ := prog.abi.TypeName(types.NewPointer(namedStruct))
	ifaceTypeName, _ := prog.abi.TypeName(namedIface)

	useIfaceRows := pkg.Module().NamedMetadataOperands(semmeta.UseIfaceMetadata)
	requireMetadataRows(t, semmeta.UseIfaceMetadata, useIfaceRows, 1)
	useIfaceFields := requireMetadataFields(t, semmeta.UseIfaceMetadata, useIfaceRows[0], 2)
	requireMDString(t, semmeta.UseIfaceMetadata, useIfaceFields, 0, "caller")
	requireMDString(t, semmeta.UseIfaceMetadata, useIfaceFields, 1, structTypeName)

	methodInfoRows := pkg.Module().NamedMetadataOperands(semmeta.MethodInfoMetadata)
	if len(methodInfoRows) == 0 {
		t.Fatalf("%s rows len = 0, want at least 1", semmeta.MethodInfoMetadata)
	}
	methodInfoFields := requireMetadataFields(t, semmeta.MethodInfoMetadata, findMetadataRowByFirstString(t, semmeta.MethodInfoMetadata, methodInfoRows, structTypeName), 6)
	requireMDString(t, semmeta.MethodInfoMetadata, methodInfoFields, 0, structTypeName)
	requireMDUint(t, semmeta.MethodInfoMetadata, methodInfoFields, 1, 0)
	requireMDString(t, semmeta.MethodInfoMetadata, methodInfoFields, 2, "Add")
	requireMDString(t, semmeta.MethodInfoMetadata, methodInfoFields, 3, mtypName)

	useIfaceMethodRows := pkg.Module().NamedMetadataOperands(semmeta.UseIfaceMethodMetadata)
	requireMetadataRows(t, semmeta.UseIfaceMethodMetadata, useIfaceMethodRows, 1)
	useIfaceMethodFields := requireMetadataFields(t, semmeta.UseIfaceMethodMetadata, useIfaceMethodRows[0], 4)
	requireMDString(t, semmeta.UseIfaceMethodMetadata, useIfaceMethodFields, 0, "caller")
	requireMDString(t, semmeta.UseIfaceMethodMetadata, useIfaceMethodFields, 1, ifaceTypeName)
	requireMDString(t, semmeta.UseIfaceMethodMetadata, useIfaceMethodFields, 2, "Add")
	requireMDString(t, semmeta.UseIfaceMethodMetadata, useIfaceMethodFields, 3, mtypName)

	interfaceInfoRows := pkg.Module().NamedMetadataOperands(semmeta.InterfaceInfoMetadata)
	if len(interfaceInfoRows) == 0 {
		t.Fatalf("%s rows len = 0, want at least 1", semmeta.InterfaceInfoMetadata)
	}
	interfaceInfoFields := requireMetadataFields(t, semmeta.InterfaceInfoMetadata, findMetadataRowByFirstString(t, semmeta.InterfaceInfoMetadata, interfaceInfoRows, ifaceTypeName), 3)
	requireMDString(t, semmeta.InterfaceInfoMetadata, interfaceInfoFields, 0, ifaceTypeName)
	requireMDString(t, semmeta.InterfaceInfoMetadata, interfaceInfoFields, 1, "Add")
	requireMDString(t, semmeta.InterfaceInfoMetadata, interfaceInfoFields, 2, mtypName)
}

func TestInterfaceInfoMetadataSingleRow(t *testing.T) {
	prog := NewProgram(nil)
	pkg := prog.NewPackage("bar", "foo/bar")

	pkg.semMetaEmitter.AddInterfaceInfo("_llgo_foo/bar.IFmt", []semmeta.MethodSig{
		{Name: "Read", MType: "_llgo_func$read"},
		{Name: "Write", MType: "_llgo_func$write"},
	})

	rows := pkg.Module().NamedMetadataOperands(semmeta.InterfaceInfoMetadata)
	requireMetadataRows(t, semmeta.InterfaceInfoMetadata, rows, 2)
	fields := requireMetadataFields(t, semmeta.InterfaceInfoMetadata, rows[0], 3)
	requireMDString(t, semmeta.InterfaceInfoMetadata, fields, 0, "_llgo_foo/bar.IFmt")
	requireMDString(t, semmeta.InterfaceInfoMetadata, fields, 1, "Read")
	requireMDString(t, semmeta.InterfaceInfoMetadata, fields, 2, "_llgo_func$read")
	fields = requireMetadataFields(t, semmeta.InterfaceInfoMetadata, rows[1], 3)
	requireMDString(t, semmeta.InterfaceInfoMetadata, fields, 0, "_llgo_foo/bar.IFmt")
	requireMDString(t, semmeta.InterfaceInfoMetadata, fields, 1, "Write")
	requireMDString(t, semmeta.InterfaceInfoMetadata, fields, 2, "_llgo_func$write")
}

func requireMetadataRows(t *testing.T, table string, rows []llvm.Value, want int) {
	t.Helper()
	if got := len(rows); got != want {
		t.Fatalf("%s rows len = %d, want %d", table, got, want)
	}
}

func requireMetadataFields(t *testing.T, table string, row llvm.Value, want int) []llvm.Value {
	t.Helper()
	fields := row.MDNodeOperands()
	if got := len(fields); got != want {
		t.Fatalf("%s field len = %d, want %d", table, got, want)
	}
	return fields
}

func findMetadataRowByFirstString(t *testing.T, table string, rows []llvm.Value, want string) llvm.Value {
	t.Helper()
	for _, row := range rows {
		fields := row.MDNodeOperands()
		if len(fields) == 0 || !fields[0].IsAMDString() {
			continue
		}
		if fields[0].MDString() == want {
			return row
		}
	}
	t.Fatalf("%s missing row for %q", table, want)
	return llvm.Value{}
}

func requireMDString(t *testing.T, table string, fields []llvm.Value, index int, want string) {
	t.Helper()
	if index >= len(fields) {
		t.Fatalf("%s field index %d out of range", table, index)
	}
	if !fields[index].IsAMDString() {
		t.Fatalf("%s field[%d] should be MDString", table, index)
	}
	got := fields[index].MDString()
	if got != want {
		t.Fatalf("%s field[%d] string = %q, want %q", table, index, got, want)
	}
}

func requireMDUint(t *testing.T, table string, fields []llvm.Value, index int, want uint64) {
	t.Helper()
	if index >= len(fields) {
		t.Fatalf("%s field index %d out of range", table, index)
	}
	got := fields[index].ZExtValue()
	if got != want {
		t.Fatalf("%s field[%d] uint = %d, want %d", table, index, got, want)
	}
}

func TestClosureCtxHelpers(t *testing.T) {
	if closureCtxParam(nil) != nil {
		t.Fatal("closureCtxParam should be nil for nil signature")
	}
	params := types.NewTuple()
	rets := types.NewTuple()
	sig := types.NewSignatureType(nil, nil, nil, params, rets, false)
	if closureCtxParam(sig) != nil {
		t.Fatal("closureCtxParam should be nil for empty params")
	}
	if removeCtx(sig) != sig {
		t.Fatal("removeCtx should return original signature when no ctx param")
	}

	badCtx := types.NewParam(0, nil, closureCtx, types.Typ[types.Int])
	badSig := types.NewSignatureType(nil, nil, nil, types.NewTuple(badCtx), rets, false)
	if closureCtxParam(badSig) != nil {
		t.Fatal("closureCtxParam should ignore non-pointer ctx param")
	}

	ctxStruct := types.NewStruct([]*types.Var{
		types.NewVar(0, nil, "v", types.Typ[types.Int]),
	}, nil)
	goodCtx := types.NewParam(0, nil, closureCtx, types.NewPointer(ctxStruct))
	arg := types.NewParam(0, nil, "x", types.Typ[types.Int])
	goodSig := types.NewSignatureType(nil, nil, nil, types.NewTuple(goodCtx, arg), rets, false)
	if closureCtxParam(goodSig) == nil {
		t.Fatal("closureCtxParam should detect ctx param")
	}
	noCtx := removeCtx(goodSig)
	if noCtx.Params().Len() != 1 || noCtx.Params().At(0).Name() != "x" {
		t.Fatalf("removeCtx result mismatch: params=%v", noCtx.Params().Len())
	}
}

func TestClosureWrapHelpers(t *testing.T) {
	prog := NewProgram(nil)
	pkg := prog.NewPackage("bar", "foo/bar")
	ctx := types.NewParam(0, nil, closureCtx, types.Typ[types.UnsafePointer])
	sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(), types.NewTuple(), false)
	sigCtx := FuncAddCtx(ctx, sig)
	wrap := pkg.NewFunc("wrap", sigCtx, InGo)
	b := wrap.MakeBody(1)
	if args := closureWrapArgs(wrap); len(args) != 0 {
		t.Fatalf("closureWrapArgs should return 0 args, got %d", len(args))
	}
	closureWrapReturn(b, sig, Expr{})
}

func TestClosureWrapCache(t *testing.T) {
	prog := NewProgram(nil)
	pkg := prog.NewPackage("bar", "foo/bar")

	params := types.NewTuple(types.NewVar(0, nil, "x", types.Typ[types.Int]))
	rets := types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Int]))
	sig := types.NewSignatureType(nil, nil, nil, params, rets, false)
	fn := pkg.NewFunc("fn", sig, InGo)
	b := fn.MakeBody(1)
	b.Return(fn.Param(0))

	w1 := pkg.closureWrapDecl(fn.Expr, sig)
	w2 := pkg.closureWrapDecl(fn.Expr, sig)
	if w1 != w2 {
		t.Fatal("closureWrapDecl should reuse existing wrapper")
	}

	p1 := pkg.closureWrapPtr(sig)
	p2 := pkg.closureWrapPtr(sig)
	if p1 != p2 {
		t.Fatal("closureWrapPtr should reuse existing wrapper")
	}
}

func TestMakeInterfaceKinds(t *testing.T) {
	prog := NewProgram(nil)
	prog.sizes = types.SizesFor("gc", runtime.GOARCH)
	prog.SetRuntime(func() *types.Package {
		pkg, err := importer.For("source", nil).Import(PkgRuntime)
		if err != nil {
			t.Fatal(err)
		}
		return pkg
	})
	pkg := prog.NewPackage("bar", "foo/bar")

	emptyIface := types.NewInterfaceType(nil, nil)
	emptyIface.Complete()
	emptyType := prog.Type(emptyIface, InGo)

	makeFn := func(name string, x Expr) {
		sig := types.NewSignatureType(nil, nil, nil, nil, types.NewTuple(types.NewVar(0, nil, "", emptyIface)), false)
		fn := pkg.NewFunc(name, sig, InGo)
		b := fn.MakeBody(1)
		iface := b.MakeInterface(emptyType, x)
		b.Return(iface)
	}

	makeFn("intIface", prog.Val(1))
	makeFn("ptrIface", prog.Nil(prog.VoidPtr()))
	makeFn("floatIface", prog.FloatVal(3.5, prog.Float32()))

	st := types.NewStruct([]*types.Var{
		types.NewVar(0, nil, "a", types.Typ[types.Int]),
		types.NewVar(0, nil, "b", types.Typ[types.Int]),
	}, nil)
	makeFn("structIface", prog.Zero(prog.Type(st, InGo)))

	single := types.NewStruct([]*types.Var{
		types.NewVar(0, nil, "v", types.Typ[types.Int]),
	}, nil)
	makeFn("singleFieldIface", prog.Zero(prog.Type(single, InGo)))

	pkgTypes := types.NewPackage("foo/bar", "bar")
	rawSig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	rawMeth := types.NewFunc(0, pkgTypes, "M", rawSig)
	nonEmpty := types.NewInterfaceType([]*types.Func{rawMeth}, nil)
	nonEmpty.Complete()
	nonEmptyType := prog.Type(nonEmpty, InGo)
	sigNE := types.NewSignatureType(nil, nil, nil, nil, types.NewTuple(types.NewVar(0, nil, "", nonEmpty)), false)
	fnNE := pkg.NewFunc("nonEmptyIface", sigNE, InGo)
	bNE := fnNE.MakeBody(1)
	bNE.Return(bNE.MakeInterface(nonEmptyType, prog.Val(7)))
}

func TestCheckExprAssignmentConversions(t *testing.T) {
	prog := NewProgram(nil)
	prog.sizes = types.SizesFor("gc", runtime.GOARCH)
	prog.SetRuntime(func() *types.Package {
		pkg, err := importer.For("source", nil).Import(PkgRuntime)
		if err != nil {
			t.Fatal(err)
		}
		return pkg
	})
	pkg := prog.NewPackage("bar", "foo/bar")
	fn := pkg.NewFunc("checkExprAssignmentConversions", NoArgsNoRet, InGo)
	b := fn.MakeBody(1)

	pkgTypes := types.NewPackage("foo/bar", "bar")
	intSlice := types.NewSlice(types.Typ[types.Int])
	namedSlice := types.NewNamed(types.NewTypeName(token.NoPos, pkgTypes, "checkExprSlice", nil), intSlice, nil)
	concrete := checkExpr(prog.Zero(prog.Type(namedSlice, InGo)), intSlice, b)
	if !types.Identical(concrete.RawType(), intSlice) {
		t.Fatalf("concrete retag type = %v, want %v", concrete.RawType(), intSlice)
	}

	emptyIface := types.NewInterfaceType(nil, nil)
	emptyIface.Complete()
	toIface := checkExpr(prog.Val(7), emptyIface, b)
	if !types.Identical(toIface.RawType(), emptyIface) {
		t.Fatalf("concrete-to-interface type = %v, want %v", toIface.RawType(), emptyIface)
	}

	namedIface := types.NewNamed(types.NewTypeName(token.NoPos, pkgTypes, "checkExprIface", nil), emptyIface, nil)
	fromIface := b.MakeInterface(prog.Type(namedIface, InGo), prog.Val(7))
	changedIface := checkExpr(fromIface, emptyIface, b)
	if !types.Identical(changedIface.RawType(), emptyIface) {
		t.Fatalf("interface-to-interface type = %v, want %v", changedIface.RawType(), emptyIface)
	}

	b.Return()
}

func TestMakeInterfaceFromPtrKinds(t *testing.T) {
	prog := NewProgram(nil)
	prog.sizes = types.SizesFor("gc", runtime.GOARCH)
	prog.SetRuntime(func() *types.Package {
		pkg, err := importer.For("source", nil).Import(PkgRuntime)
		if err != nil {
			t.Fatal(err)
		}
		return pkg
	})
	pkg := prog.NewPackage("bar", "foo/bar")

	emptyIface := types.NewInterfaceType(nil, nil)
	emptyIface.Complete()
	emptyType := prog.Type(emptyIface, InGo)
	returns := types.NewTuple(types.NewVar(0, nil, "", emptyIface))

	makePtrIface := func(name string, elem types.Type) {
		ptrTyp := types.NewPointer(elem)
		params := types.NewTuple(types.NewVar(0, nil, "p", ptrTyp))
		sig := types.NewSignatureType(nil, nil, nil, params, returns, false)
		fn := pkg.NewFunc(name, sig, InGo)
		b := fn.MakeBody(1)
		b.Return(b.MakeInterfaceFromPtr(emptyType, fn.Param(0)))
		b.EndBuild()
	}

	makePtrIface("smallPtrIface", types.Typ[types.Int])
	makePtrIface("largePtrIface", types.NewArray(types.Typ[types.Byte], 1<<21))

	ir := pkg.Module().String()
	if !strings.Contains(ir, "AssertNilDeref") {
		t.Fatalf("MakeInterfaceFromPtr should emit nil-deref guard, got:\n%s", ir)
	}
	if !strings.Contains(ir, "Typedmemmove") {
		t.Fatalf("large MakeInterfaceFromPtr should copy via Typedmemmove, got:\n%s", ir)
	}
}

func TestTypeAssertSingleElemArrayUsesInsertValue(t *testing.T) {
	prog := NewProgram(nil)
	prog.sizes = types.SizesFor("gc", runtime.GOARCH)
	prog.SetRuntime(func() *types.Package {
		pkg, err := importer.For("source", nil).Import(PkgRuntime)
		if err != nil {
			t.Fatal(err)
		}
		return pkg
	})
	pkg := prog.NewPackage("bar", "foo/bar")

	emptyIface := types.NewInterfaceType(nil, nil)
	emptyIface.Complete()
	arrayRaw := types.NewArray(types.Typ[types.Int], 1)
	arrayType := prog.Type(arrayRaw, InGo)

	params := types.NewTuple(types.NewVar(0, nil, "x", emptyIface))
	results := types.NewTuple(
		types.NewVar(0, nil, "", arrayRaw),
		types.NewVar(0, nil, "", types.Typ[types.Bool]),
	)
	sig := types.NewSignatureType(nil, nil, nil, params, results, false)
	fn := pkg.NewFunc("assertArray", sig, InGo)
	b := fn.MakeBody(1)
	ret := b.TypeAssert(fn.Param(0), arrayType, true)
	b.Return(b.Extract(ret, 0), b.Extract(ret, 1))

	ir := fn.impl.String()
	if !strings.Contains(ir, "insertvalue { [1 x i") {
		t.Fatalf("single element array type assert should rebuild via insertvalue:\n%s", ir)
	}
	if strings.Contains(ir, "alloca [1 x i") {
		t.Fatalf("single element array type assert should not rebuild via alloca:\n%s", ir)
	}
}

func TestInterfaceHelpers(t *testing.T) {
	rawSig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	rawMeth := types.NewFunc(0, nil, "M", rawSig)
	rawIface := types.NewInterfaceType([]*types.Func{rawMeth}, nil)
	rawIface.Complete()

	if got := iMethodOf(rawIface, "missing"); got != -1 {
		t.Fatalf("iMethodOf missing: got %d", got)
	}

	prog := NewProgram(nil)
	prog.SetRuntime(func() *types.Package {
		fset := token.NewFileSet()
		imp := packages.NewImporter(fset)
		pkg, _ := imp.Import(PkgRuntime)
		return pkg
	})
	pkg := prog.NewPackage("bar", "foo/bar")
	intfType := prog.Type(rawIface, InGo)
	fn := pkg.NewFunc("call", types.NewSignatureType(nil, nil, nil,
		types.NewTuple(types.NewVar(0, nil, "i", rawIface)), nil, false), InGo)
	b := fn.MakeBody(1)

	// Method signature with first param being the interface itself.
	params := types.NewTuple(types.NewVar(0, nil, "self", rawIface),
		types.NewVar(0, nil, "x", types.Typ[types.Int]))
	sig := types.NewSignatureType(nil, nil, nil, params,
		types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Int])), false)
	method := types.NewFunc(0, nil, "M", sig)
	closure := b.Imethod(fn.Param(0), method)
	if got := closure.raw.Type.(*types.Struct).Field(0).Type().(*types.Signature).Params().Len(); got != 1 {
		t.Fatalf("Imethod should drop interface param: got %d params", got)
	}
	_ = intfType
}

func TestValFromDataKinds(t *testing.T) {
	prog := NewProgram(nil)
	pkg := prog.NewPackage("bar", "foo/bar")
	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	fn := pkg.NewFunc("caller", sig, InGo)
	b := fn.MakeBody(1)
	data := prog.Nil(prog.VoidPtr()).impl

	b.valFromData(prog.Int(), data)
	b.valFromData(prog.Float32(), data)
	b.valFromData(prog.Type(types.NewPointer(types.Typ[types.Int]), InGo), data)

	st := types.NewStruct([]*types.Var{
		types.NewVar(0, nil, "a", types.Typ[types.Int]),
		types.NewVar(0, nil, "b", types.Typ[types.Int]),
	}, nil)
	b.valFromData(prog.Type(st, InGo), data)

	single := types.NewStruct([]*types.Var{
		types.NewVar(0, nil, "v", types.Typ[types.Int]),
	}, nil)
	b.valFromData(prog.Type(single, InGo), data)

	arr := types.NewArray(types.Typ[types.Int], 1)
	b.valFromData(prog.Type(arr, InGo), data)

	b.Return()
}

func TestPackageCoverageHelpers(t *testing.T) {
	if !is32Bits("386") {
		t.Fatal("is32Bits should return true for 386")
	}
	if is32Bits("amd64") {
		t.Fatal("is32Bits should return false for amd64")
	}
	prog := NewProgram(nil)
	_ = prog.CIntPtr()
	pkg := prog.NewPackage("bar", "foo/bar")
	if len(pkg.ExportFuncs()) != 0 {
		t.Fatal("ExportFuncs should be empty for new package")
	}

	// cover closureStub default branch
	fn := pkg.NewFunc("noop", NoArgsNoRet, InGo)
	b := fn.MakeBody(1)
	expr := prog.Val(1)
	got, data := pkg.closureStub(b, expr, nil, vkString)
	if got.impl.IsNil() || !data.impl.IsNull() {
		t.Fatal("closureStub default branch should return expr and nil data")
	}
	b.Return()
}

func TestExprCoverageHelpers(t *testing.T) {
	prog := NewProgram(nil)
	pkg := prog.NewPackage("bar", "foo/bar")
	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	fn := pkg.NewFunc("fn", sig, InGo)
	b := fn.MakeBody(1)

	// SetName coverage
	tmp := b.AllocaT(prog.Int())
	tmp.SetName("tmp0")

	// Printf / tyPrintf coverage
	b.Printf("value=%d", prog.Val(1))
	b.Return()
}

func TestTypes(t *testing.T) {
	ctx := llvm.NewContext()
	llvmIntType(ctx, 4)

	intT := types.NewVar(0, nil, "", types.Typ[types.Int])
	ret := types.NewTuple(intT, intT)
	sig := types.NewSignatureType(nil, nil, nil, nil, ret, false)
	prog := NewProgram(nil)
	prog.retType(sig)
}

func TestIndexType(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Log("indexType: no error?")
		}
	}()
	indexType(types.Typ[types.Int])
}

func TestCvtType(t *testing.T) {
	gt := newGoTypes()
	params := types.NewTuple(types.NewParam(0, nil, "", NoArgsNoRet))
	sig := types.NewSignatureType(nil, nil, nil, params, nil, false)
	ret1 := gt.cvtFunc(sig, nil)
	if ret1 == sig {
		t.Fatal("cvtFunc failed")
	}
	defer func() {
		if r := recover(); r == nil {
			t.Log("cvtType: no error?")
		}
	}()
	gt.cvtType(nil)
}

func TestUserdefExpr(t *testing.T) {
	c := &pyVarTy{}
	b := &builtinTy{}
	_ = c.String()
	_ = b.String()
	test := func(a types.Type) {
		defer func() {
			if r := recover(); r == nil {
				t.Log("TestUserdefExpr: no error?")
			}
		}()
		a.Underlying()
	}
	test(c)
	test(b)
}

func TestAny(t *testing.T) {
	prog := NewProgram(nil)
	prog.SetRuntime(func() *types.Package {
		ret := types.NewPackage("runtime", "runtime")
		scope := ret.Scope()
		name := types.NewTypeName(0, ret, "Eface", nil)
		types.NewNamed(name, types.NewStruct(nil, nil), nil)
		scope.Insert(name)
		return ret
	})
	prog.Any()
}

func assertPkg(t *testing.T, p Package, expected string) {
	t.Helper()
	got := StripModuleTarget(p.String())
	want := StripModuleTarget(expected)
	if got != want {
		t.Fatalf("\n==> got:\n%s\n==> expected:\n%s\n", got, want)
	}
}

func TestPyFunc(t *testing.T) {
	prog := NewProgram(nil)
	py := types.NewPackage("foo", "foo")
	o := types.NewTypeName(0, py, "Object", nil)
	types.NewNamed(o, types.Typ[types.Int], nil)
	py.Scope().Insert(o)
	prog.SetPython(py)
	pkg := prog.NewPackage("bar", "foo/bar")
	a := pkg.PyNewFunc("a", NoArgsNoRet, false)
	if pkg.PyNewFunc("a", NoArgsNoRet, false) != a {
		t.Fatal("NewPyFunc(a) failed")
	}
	foo := pkg.PyNewModVar("foo", false)
	if pkg.PyNewModVar("foo", false) != foo {
		t.Fatal("NewPyModVar(foo) failed")
	}
}

func TestVar(t *testing.T) {
	prog := NewProgram(nil)
	pkg := prog.NewPackage("bar", "foo/bar")
	typ := types.NewPointer(types.Typ[types.Int])
	a := pkg.NewVar("a", typ, InGo)
	if pkg.NewVar("a", typ, InGo) != a {
		t.Fatal("NewVar(a) failed")
	}
	pkg.NewVarEx("a", prog.Type(typ, InGo))
	a.Init(prog.Val(100))
	b := pkg.NewVar("b", typ, InGo)
	b.Init(Expr{llvm.ConstPtrToInt(a.impl, prog.Int().ll), prog.Int()})
	assertPkg(t, pkg, `; ModuleID = 'foo/bar'
source_filename = "foo/bar"

@a = global i64 100, align 8
@b = global i64 ptrtoint (ptr @a to i64), align 8
`)
}

func TestConst(t *testing.T) {
	prog := NewProgram(nil)
	pkg := prog.NewPackage("bar", "foo/bar")
	rets := types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Bool]))
	sig := types.NewSignatureType(nil, nil, nil, nil, rets, false)
	b := pkg.NewFunc("fn", sig, InGo).MakeBody(1)
	b.Return(b.Const(constant.MakeBool(true), prog.Bool()))
	assertPkg(t, pkg, `; ModuleID = 'foo/bar'
source_filename = "foo/bar"

; Function Attrs: null_pointer_is_valid
define i1 @fn() #0 {
_llgo_0:
  ret i1 true
}

attributes #0 = { null_pointer_is_valid }
`)
}

func TestStruct(t *testing.T) {
	empty := types.NewStruct(nil, nil)

	prog := NewProgram(nil)
	pkg := prog.NewPackage("bar", "foo/bar")
	pkg.NewVar("a", types.NewPointer(empty), InGo)
	assertPkg(t, pkg, `; ModuleID = 'foo/bar'
source_filename = "foo/bar"

@a = external global {}, align 1
`)
	if pkg.NeedRuntime {
		t.Fatal("NeedRuntime?")
	}
}

func TestNamedStruct(t *testing.T) {
	src := types.NewPackage("bar", "foo/bar")
	empty := types.NewNamed(types.NewTypeName(0, src, "Empty", nil), types.NewStruct(nil, nil), nil)

	prog := NewProgram(nil)
	pkg := prog.NewPackage("bar", "foo/bar")
	pkg.NewVar("a", types.NewPointer(empty), InGo)
	if pkg.VarOf("a") == nil {
		t.Fatal("VarOf failed")
	}
	assertPkg(t, pkg, `; ModuleID = 'foo/bar'
source_filename = "foo/bar"

%bar.Empty = type {}

@a = external global %bar.Empty, align 1
`)
}

func TestDeclFunc(t *testing.T) {
	prog := NewProgram(nil)
	pkg := prog.NewPackage("bar", "foo/bar")
	params := types.NewTuple(types.NewVar(0, nil, "a", types.Typ[types.Int]))
	sig := types.NewSignatureType(nil, nil, nil, params, nil, false)
	pkg.NewFunc("fn", sig, InGo)
	if pkg.FuncOf("fn") == nil {
		t.Fatal("FuncOf failed")
	}
	if prog.retType(sig) != prog.Void() {
		t.Fatal("retType failed")
	}
	assertPkg(t, pkg, `; ModuleID = 'foo/bar'
source_filename = "foo/bar"

; Function Attrs: null_pointer_is_valid
declare void @fn(i64) #0

attributes #0 = { null_pointer_is_valid }
`)
}

func TestBasicFunc(t *testing.T) {
	prog := NewProgram(nil)
	pkg := prog.NewPackage("bar", "foo/bar")
	params := types.NewTuple(
		types.NewVar(0, nil, "a", types.Typ[types.Int]),
		types.NewVar(0, nil, "b", types.Typ[types.Float64]))
	rets := types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Int]))
	sig := types.NewSignatureType(nil, nil, nil, params, rets, false)
	pkg.NewFunc("fn", sig, InGo).MakeBody(1).
		Return(prog.Val(1))
	assertPkg(t, pkg, `; ModuleID = 'foo/bar'
source_filename = "foo/bar"

; Function Attrs: null_pointer_is_valid
define i64 @fn(i64 %0, double %1) #0 {
_llgo_0:
  ret i64 1
}

attributes #0 = { null_pointer_is_valid }
`)
}

func TestFuncParam(t *testing.T) {
	prog := NewProgram(nil)
	pkg := prog.NewPackage("bar", "foo/bar")
	params := types.NewTuple(
		types.NewVar(0, nil, "a", types.Typ[types.Int]),
		types.NewVar(0, nil, "b", types.Typ[types.Float64]))
	rets := types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Int]))
	sig := types.NewSignatureType(nil, nil, nil, params, rets, false)
	fn := pkg.NewFunc("fn", sig, InGo)
	fn.MakeBody(1).Return(fn.Param(0))
	assertPkg(t, pkg, `; ModuleID = 'foo/bar'
source_filename = "foo/bar"

; Function Attrs: null_pointer_is_valid
define i64 @fn(i64 %0, double %1) #0 {
_llgo_0:
  ret i64 %0
}

attributes #0 = { null_pointer_is_valid }
`)
}

func TestFuncCall(t *testing.T) {
	prog := NewProgram(nil)
	pkg := prog.NewPackage("bar", "foo/bar")

	params := types.NewTuple(
		types.NewVar(0, nil, "a", types.Typ[types.Int]),
		types.NewVar(0, nil, "b", types.Typ[types.Float64]))
	rets := types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Int]))
	sig := types.NewSignatureType(nil, nil, nil, params, rets, false)
	fn := pkg.NewFunc("fn", sig, InGo)
	fn.MakeBody(1).
		Return(prog.Val(1))

	b := pkg.NewFunc("main", NoArgsNoRet, InGo).MakeBody(1)
	b.Call(fn.Expr, prog.Val(1), prog.Val(1.2))
	b.Return()

	assertPkg(t, pkg, `; ModuleID = 'foo/bar'
source_filename = "foo/bar"

; Function Attrs: null_pointer_is_valid
define i64 @fn(i64 %0, double %1) #0 {
_llgo_0:
  ret i64 1
}

; Function Attrs: null_pointer_is_valid
define void @main() #0 {
_llgo_0:
  %0 = call i64 @fn(i64 1, double 1.200000e+00)
  ret void
}

attributes #0 = { null_pointer_is_valid }
`)
}

func TestFuncMultiRet(t *testing.T) {
	prog := NewProgram(nil)
	pkg := prog.NewPackage("bar", "foo/bar")
	params := types.NewTuple(
		types.NewVar(0, nil, "b", types.Typ[types.Float64]))
	rets := types.NewTuple(
		types.NewVar(0, nil, "c", types.Typ[types.Int]),
		types.NewVar(0, nil, "d", types.Typ[types.Float64]))
	sig := types.NewSignatureType(nil, nil, nil, params, rets, false)
	a := pkg.NewVar("a", types.NewPointer(types.Typ[types.Int]), InGo)
	fn := pkg.NewFunc("fn", sig, InGo)
	b := fn.MakeBody(1)
	aInt := Expr{llvm.ConstPtrToInt(a.impl, prog.Int().ll), prog.Int()}
	b.Return(aInt, fn.Param(0))
	assertPkg(t, pkg, `; ModuleID = 'foo/bar'
source_filename = "foo/bar"

@a = external global i64, align 8

; Function Attrs: null_pointer_is_valid
define { i64, double } @fn(double %0) #0 {
_llgo_0:
  %1 = insertvalue { i64, double } { i64 ptrtoint (ptr @a to i64), double undef }, double %0, 1
  ret { i64, double } %1
}

attributes #0 = { null_pointer_is_valid }
`)
}

func TestJump(t *testing.T) {
	prog := NewProgram(nil)
	pkg := prog.NewPackage("bar", "foo/bar")
	fn := pkg.NewFunc("loop", NoArgsNoRet, InGo)
	b := fn.MakeBody(1)
	b.Jump(fn.Block(0))
	assertPkg(t, pkg, `; ModuleID = 'foo/bar'
source_filename = "foo/bar"

; Function Attrs: null_pointer_is_valid
define void @loop() #0 {
_llgo_0:
  br label %_llgo_0
}

attributes #0 = { null_pointer_is_valid }
`)
}

func TestIf(t *testing.T) {
	prog := NewProgram(nil)
	pkg := prog.NewPackage("bar", "foo/bar")
	params := types.NewTuple(types.NewVar(0, nil, "a", types.Typ[types.Int]))
	rets := types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Int]))
	sig := types.NewSignatureType(nil, nil, nil, params, rets, false)
	fn := pkg.NewFunc("fn", sig, InGo)
	b := fn.MakeBody(3)
	iftrue := fn.Block(1)
	iffalse := fn.Block(2)
	if iftrue.Index() != 1 || iftrue.Parent() != fn {
		t.Fatal("iftrue")
	}
	cond := b.BinOp(token.GTR, fn.Param(0), prog.Val(0))
	b.If(cond, iftrue, iffalse)
	b.SetBlock(iftrue).Return(prog.Val(1))
	b.SetBlock(iffalse).Return(prog.Val(0))
	assertPkg(t, pkg, `; ModuleID = 'foo/bar'
source_filename = "foo/bar"

; Function Attrs: null_pointer_is_valid
define i64 @fn(i64 %0) #0 {
_llgo_0:
  %1 = icmp sgt i64 %0, 0
  br i1 %1, label %_llgo_1, label %_llgo_2

_llgo_1:                                          ; preds = %_llgo_0
  ret i64 1

_llgo_2:                                          ; preds = %_llgo_0
  ret i64 0
}

attributes #0 = { null_pointer_is_valid }
`)
}

func TestPrintf(t *testing.T) {
	prog := NewProgram(nil)
	pkg := prog.NewPackage("bar", "foo/bar")
	pchar := types.NewPointer(types.Typ[types.Int8])
	params := types.NewTuple(types.NewVar(0, nil, "format", pchar), VArg())
	rets := types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Int32]))
	sig := types.NewSignatureType(nil, nil, nil, params, rets, true)
	pkg.NewFunc("printf", sig, InC)
	assertPkg(t, pkg, `; ModuleID = 'foo/bar'
source_filename = "foo/bar"

declare i32 @printf(ptr, ...)
`)
}

func TestBinOp(t *testing.T) {
	prog := NewProgram(nil)
	pkg := prog.NewPackage("bar", "foo/bar")
	params := types.NewTuple(
		types.NewVar(0, nil, "a", types.Typ[types.Int]),
		types.NewVar(0, nil, "b", types.Typ[types.Float64]))
	rets := types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Int]))
	sig := types.NewSignatureType(nil, nil, nil, params, rets, false)
	fn := pkg.NewFunc("fn", sig, InGo)
	b := fn.MakeBody(1)
	ret := b.BinOp(token.ADD, fn.Param(0), prog.Val(1))
	b.Return(ret)
	assertPkg(t, pkg, `; ModuleID = 'foo/bar'
source_filename = "foo/bar"

; Function Attrs: null_pointer_is_valid
define i64 @fn(i64 %0, double %1) #0 {
_llgo_0:
  %2 = add i64 %0, 1
  ret i64 %2
}

attributes #0 = { null_pointer_is_valid }
`)
}

func TestUnOp(t *testing.T) {
	prog := NewProgram(nil)
	pkg := prog.NewPackage("bar", "foo/bar")
	params := types.NewTuple(
		types.NewVar(0, nil, "p", types.NewPointer(types.Typ[types.Int])),
	)
	rets := types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Int]))
	sig := types.NewSignatureType(nil, nil, nil, params, rets, false)
	fn := pkg.NewFunc("fn", sig, InGo)
	b := fn.MakeBody(1)
	ptr := fn.Param(0)
	val := b.UnOp(token.MUL, ptr)
	val2 := b.BinOp(token.XOR, val, prog.Val(1))
	b.Store(ptr, val2)
	b.Return(val2)
	assertPkg(t, pkg, `; ModuleID = 'foo/bar'
source_filename = "foo/bar"

; Function Attrs: null_pointer_is_valid
define i64 @fn(ptr %0) #0 {
_llgo_0:
  %1 = load i64, ptr %0, align 8
  %2 = xor i64 %1, 1
  store i64 %2, ptr %0, align 8
  ret i64 %2
}

attributes #0 = { null_pointer_is_valid }
`)
}

func TestBasicType(t *testing.T) {
	type typeInfo struct {
		typ  Type
		kind types.BasicKind
	}
	prog := NewProgram(nil)
	infos := []*typeInfo{
		{prog.Bool(), types.Bool},
		{prog.Byte(), types.Byte},
		{prog.Int(), types.Int},
		{prog.Uint(), types.Uint},
		{prog.Int32(), types.Int32},
		{prog.Int64(), types.Int64},
		{prog.Uint32(), types.Uint32},
		{prog.Uint64(), types.Uint64},
		{prog.Uintptr(), types.Uintptr},
		{prog.VoidPtr(), types.UnsafePointer},
	}
	for _, info := range infos {
		if info.typ.RawType() != types.Typ[info.kind] {
			t.Fatal("bad type", info)
		}
	}
}

func TestCompareSelect(t *testing.T) {
	prog := NewProgram(nil)
	pkg := prog.NewPackage("bar", "foo/bar")

	params := types.NewTuple(
		types.NewVar(0, nil, "a", types.Typ[types.Int]),
		types.NewVar(0, nil, "b", types.Typ[types.Int]),
		types.NewVar(0, nil, "c", types.Typ[types.Int]),
	)
	rets := types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Int]))
	sig := types.NewSignatureType(nil, nil, nil, params, rets, false)
	fn := pkg.NewFunc("fn", sig, InGo)

	b := fn.MakeBody(1)
	result := b.compareSelect(token.GTR, fn.Param(0), fn.Param(1), fn.Param(2))
	b.Return(result)

	assertPkg(t, pkg, `; ModuleID = 'foo/bar'
source_filename = "foo/bar"

; Function Attrs: null_pointer_is_valid
define i64 @fn(i64 %0, i64 %1, i64 %2) #0 {
_llgo_0:
  %3 = icmp sgt i64 %0, %1
  %4 = select i1 %3, i64 %0, i64 %1
  %5 = icmp sgt i64 %4, %2
  %6 = select i1 %5, i64 %4, i64 %2
  ret i64 %6
}

attributes #0 = { null_pointer_is_valid }
`)
}

func TestGlobalStrings(t *testing.T) {
	prog := NewProgram(nil)
	prog.SetRuntime(func() *types.Package {
		fset := token.NewFileSet()
		imp := packages.NewImporter(fset)
		pkg, _ := imp.Import(PkgRuntime)
		return pkg
	})
	pkg := prog.NewPackage("bar", "foo/bar")
	typ := types.NewPointer(types.Typ[types.String])
	a := pkg.NewVar("foo/bar.a", typ, InGo)
	if pkg.NewVar("foo/bar.a", typ, InGo) != a {
		t.Fatal("NewVar(a) failed")
	}
	a.InitNil()
	pkg.NewVarEx("foo/bar.a", prog.Type(typ, InGo))
	b := pkg.NewVar("foo/bar.b", typ, InGo)
	b.InitNil()
	c := pkg.NewVar("foo/bar.c", types.NewPointer(types.Typ[types.Int]), InGo)
	c.Init(prog.Val(100))
	assertPkg(t, pkg, `; ModuleID = 'foo/bar'
source_filename = "foo/bar"

%"github.com/goplus/llgo/runtime/internal/runtime.String" = type { ptr, i64 }

@"foo/bar.a" = global %"github.com/goplus/llgo/runtime/internal/runtime.String" zeroinitializer, align 8
@"foo/bar.b" = global %"github.com/goplus/llgo/runtime/internal/runtime.String" zeroinitializer, align 8
@"foo/bar.c" = global i64 100, align 8
`)
	err := pkg.Undefined("foo/bar.a", "foo/bar.b")
	if err != nil {
		t.Fatal(err)
	}
	pkg.Undefined("foo.bar.d")
	err = pkg.Undefined("foo/bar.c")
	if err == nil {
		t.Fatal("must err")
	}
	assertPkg(t, pkg, `; ModuleID = 'foo/bar'
source_filename = "foo/bar"

%"github.com/goplus/llgo/runtime/internal/runtime.String" = type { ptr, i64 }

@"foo/bar.c" = global i64 100, align 8
@"foo/bar.a" = external global %"github.com/goplus/llgo/runtime/internal/runtime.String"
@"foo/bar.b" = external global %"github.com/goplus/llgo/runtime/internal/runtime.String"
`)
	global := prog.NewPackage("", "global")
	global.AddGlobalString("foo/bar.a", "1.0")
	global.AddGlobalString("foo/bar.b", "info")
	assertPkg(t, global, `; ModuleID = 'global'
source_filename = "global"

%"github.com/goplus/llgo/runtime/internal/runtime.String" = type { ptr, i64 }

@0 = private unnamed_addr constant [3 x i8] c"1.0", align 1
@"foo/bar.a" = global %"github.com/goplus/llgo/runtime/internal/runtime.String" { ptr @0, i64 3 }, align 8
@1 = private unnamed_addr constant [4 x i8] c"info", align 1
@"foo/bar.b" = global %"github.com/goplus/llgo/runtime/internal/runtime.String" { ptr @1, i64 4 }, align 8
`)
}

func TestZeroSizedGlobalEmitsAliasSymbol(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	os.Chdir("../../runtime")
	defer os.Chdir(wd)

	prog := NewProgram(nil)
	prog.SetRuntime(func() *types.Package {
		fset := token.NewFileSet()
		imp := packages.NewImporter(fset)
		pkg, _ := imp.Import(PkgRuntime)
		return pkg
	})
	pkg := prog.NewPackage("bar", "foo/bar")
	typ := types.NewPointer(types.NewArray(types.Typ[types.Int], 0))
	a := pkg.NewVar("foo/bar.a", typ, InGo)
	a.InitNil()
	pkg.NewVar("other/pkg.a", typ, InGo)
	assertPkg(t, pkg, `; ModuleID = 'foo/bar'
source_filename = "foo/bar"

@"__llgo.moduleZeroSizedAlloc$" = linkonce_odr unnamed_addr global i8 0
@"other/pkg.a" = external global [0 x i64], align 8

@"foo/bar.a" = alias [0 x i64], ptr @"__llgo.moduleZeroSizedAlloc$"
`)
}

func TestGlobalConstLiterals(t *testing.T) {
	prog := NewProgram(nil)
	prog.SetRuntime(func() *types.Package {
		fset := token.NewFileSet()
		imp := packages.NewImporter(fset)
		pkg, _ := imp.Import(PkgRuntime)
		return pkg
	})
	pkg := prog.NewPackage("bar", "foo/bar")

	_ = pkg.ConstString("hello")
	_ = pkg.ConstString("hello")
	ir := pkg.String()
	if strings.Count(ir, `c"hello"`) != 1 {
		t.Fatalf("ConstString should reuse backing global, got:\n%s", ir)
	}

	before := pkg.String()
	_ = pkg.ConstBytes(nil)
	_ = pkg.ConstBytes([]byte{})
	_ = pkg.createGlobalBytes(nil)
	afterEmpty := pkg.String()
	if afterEmpty != before {
		t.Fatalf("ConstBytes(empty) should not emit globals:\n%s", afterEmpty)
	}

	_ = pkg.ConstBytes([]byte("hi"))
	_ = pkg.ConstBytes([]byte("hi"))
	ir = pkg.String()
	if strings.Count(ir, `c"hi"`) != 2 {
		t.Fatalf("ConstBytes should allocate writable backing each call, got:\n%s", ir)
	}
}

func TestSetjmpReturnsTwice(t *testing.T) {
	prog := NewProgram(nil)
	pkg := prog.NewPackage("bar", "foo/bar")

	// func test(jmpbuf unsafe.Pointer) int32
	params := types.NewTuple(
		types.NewVar(0, nil, "jmpbuf", types.Typ[types.UnsafePointer]))
	rets := types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Int32]))
	sig := types.NewSignatureType(nil, nil, nil, params, rets, false)

	fn := pkg.NewFunc("test", sig, InGo)
	b := fn.MakeBody(1)
	ret := b.Setjmp(fn.Param(0))
	b.Return(ret)

	assertPkg(t, pkg, `; ModuleID = 'foo/bar'
source_filename = "foo/bar"

; Function Attrs: null_pointer_is_valid
define i32 @test(ptr %0) #0 {
_llgo_0:
  %1 = call i32 @setjmp(ptr %0)
  ret i32 %1
}

; Function Attrs: returns_twice
declare i32 @setjmp(ptr) #1

attributes #0 = { null_pointer_is_valid }
attributes #1 = { returns_twice }
`)
}

func TestTargetMachineAndDataLayout(t *testing.T) {
	tests := []struct {
		goos       string
		goarch     string
		dataLayout string
		triple     string
	}{
		{"linux", "amd64", "e-m:e-p270:32:32-p271:32:32-p272:64:64-i64:64-i128:128-f80:128-n8:16:32:64-S128", "x86_64-unknown-linux"},
		{"linux", "arm64", "e-m:e-i8:8:32-i16:16:32-i64:64-i128:128-n32:64-S128-Fn32", "aarch64-unknown-linux"},
		{"darwin", "amd64", "e-m:o-p270:32:32-p271:32:32-p272:64:64-i64:64-i128:128-f80:128-n8:16:32:64-S128", "x86_64-apple-macosx"},
		{"darwin", "arm64", "e-m:o-i64:64-i128:128-n32:64-S128-Fn32", "arm64-apple-macosx"},
	}
	for _, tt := range tests {
		prog := NewProgram(&Target{GOOS: tt.goos, GOARCH: tt.goarch})

		// Test TargetMachine() returns a valid target machine
		tm := prog.TargetMachine()
		if tm.C == nil {
			t.Fatalf("%s/%s TargetMachine() returned nil", tt.goos, tt.goarch)
		}

		// Test TargetData() returns a valid target data
		td := prog.TargetData()
		if td.C == nil {
			t.Fatalf("%s/%s TargetData() returned nil", tt.goos, tt.goarch)
		}

		// Test DataLayout() returns the expected data layout string
		if dl := prog.DataLayout(); dl != tt.dataLayout {
			t.Fatalf("%s/%s DataLayout mismatch: got %q, want %q", tt.goos, tt.goarch, dl, tt.dataLayout)
		}

		pkg := prog.NewPackage("foo", "foo/bar")
		if dl := pkg.Module().DataLayout(); dl != tt.dataLayout {
			t.Fatalf("%s/%s module DataLayout mismatch: got %q, want %q", tt.goos, tt.goarch, dl, tt.dataLayout)
		}

		// Test Target().Spec().Triple returns the expected triple
		if triple := prog.Target().Spec().Triple; triple != tt.triple {
			t.Fatalf("%s/%s Triple mismatch: got %q, want %q", tt.goos, tt.goarch, triple, tt.triple)
		}
	}
}

func TestAbiTables(t *testing.T) {
	prog := NewProgram(nil)
	prog.sizes = types.SizesFor("gc", runtime.GOARCH)
	prog.SetRuntime(func() *types.Package {
		pkg, err := importer.For("source", nil).Import(PkgRuntime)
		if err != nil {
			t.Fatal(err)
		}
		return pkg
	})
	pkg := prog.NewPackage("bar", "foo/bar")

	emptyIface := types.NewInterfaceType(nil, nil)
	emptyIface.Complete()
	emptyType := prog.Type(emptyIface, InGo)

	makeFn := func(name string, x Expr) {
		sig := types.NewSignatureType(nil, nil, nil, nil, types.NewTuple(types.NewVar(0, nil, "", emptyIface)), false)
		fn := pkg.NewFunc(name, sig, InGo)
		b := fn.MakeBody(1)
		iface := b.MakeInterface(emptyType, x)
		b.Return(iface)
	}

	makeFn("intIface", prog.Val(1))
	makeFn("ptrIface", prog.Nil(prog.VoidPtr()))
	makeFn("floatIface", prog.FloatVal(3.5, prog.Float32()))

	st := types.NewStruct([]*types.Var{
		types.NewVar(0, nil, "a", types.Typ[types.Int]),
		types.NewVar(0, nil, "b", types.Typ[types.Int]),
	}, nil)
	makeFn("structIface", prog.Zero(prog.Type(st, InGo)))

	single := types.NewStruct([]*types.Var{
		types.NewVar(0, nil, "v", types.Typ[types.Int]),
	}, nil)
	makeFn("singleFieldIface", prog.Zero(prog.Type(single, InGo)))

	pkgTypes := types.NewPackage("foo/bar", "bar")
	rawSig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	rawMeth := types.NewFunc(0, pkgTypes, "M", rawSig)
	nonEmpty := types.NewInterfaceType([]*types.Func{rawMeth}, nil)
	nonEmpty.Complete()
	nonEmptyType := prog.Type(nonEmpty, InGo)
	sigNE := types.NewSignatureType(nil, nil, nil, nil, types.NewTuple(types.NewVar(0, nil, "", nonEmpty)), false)
	fnNE := pkg.NewFunc("nonEmptyIface", sigNE, InGo)
	bNE := fnNE.MakeBody(1)
	bNE.Return(bNE.MakeInterface(nonEmptyType, prog.Val(7)))

	fn := pkg.InitAbiTypes(pkg.Path() + ".init$abitables")
	s := fn.impl.String()
	if !strings.Contains(s, `define void @"foo/bar.init$abitables"()`) ||
		!strings.Contains(s, `@"foo/bar.init$abitables$slice"`) ||
		!strings.Contains(s, `@"github.com/goplus/llgo/runtime/internal/runtime.typelist"`) {
		t.Fatal("error abi tables", s)
	}
}

func TestInitAbiTypesForSubset(t *testing.T) {
	prog := NewProgram(nil)
	prog.sizes = types.SizesFor("gc", runtime.GOARCH)
	prog.SetRuntime(func() *types.Package {
		pkg, err := importer.For("source", nil).Import(PkgRuntime)
		if err != nil {
			t.Fatal(err)
		}
		return pkg
	})
	pkg := prog.NewPackage("bar", "foo/bar")

	emptyIface := types.NewInterfaceType(nil, nil)
	emptyIface.Complete()
	emptyType := prog.Type(emptyIface, InGo)
	makeFn := func(name string, x Expr) {
		sig := types.NewSignatureType(nil, nil, nil, nil, types.NewTuple(types.NewVar(0, nil, "", emptyIface)), false)
		fn := pkg.NewFunc(name, sig, InGo)
		b := fn.MakeBody(1)
		b.Return(b.MakeInterface(emptyType, x))
	}

	makeFn("intIface", prog.Val(1))
	makeFn("floatIface", prog.FloatVal(3.5, prog.Float32()))

	if len(prog.abiSymbol) < 2 {
		t.Fatalf("expected multiple abi symbols, got %d", len(prog.abiSymbol))
	}
	names := make([]string, 0, len(prog.abiSymbol))
	for name := range prog.abiSymbol {
		names = append(names, name)
	}
	sort.Strings(names)

	pkg.getAbiTypesFor("subset", func(sym *AbiSymbol) bool {
		if sym.Name == names[0] || sym.Name == "missing.symbol" {
			return true
		}
		return false
	})
	subsetArray := pkg.Module().NamedGlobal("subset$array")
	if subsetArray.IsNil() {
		t.Fatal("missing subset abi array global")
	}
	if got := subsetArray.GlobalValueType().ArrayLength(); got != 1 {
		t.Fatalf("subset abi array length = %d, want 1", got)
	}

	pkg.getAbiTypes("all")
	allArray := pkg.Module().NamedGlobal("all$array")
	if allArray.IsNil() {
		t.Fatal("missing full abi array global")
	}
	if got := allArray.GlobalValueType().ArrayLength(); got != len(prog.abiSymbol) {
		t.Fatalf("full abi array length = %d, want %d", got, len(prog.abiSymbol))
	}
}

func TestInitAbiTypesForEmptySelection(t *testing.T) {
	prog := NewProgram(nil)
	pkg := prog.NewPackage("bar", "foo/bar")

	if fn := pkg.InitAbiTypes("empty"); fn != nil {
		t.Fatalf("InitAbiTypes on empty abi symbol set = %v, want nil", fn)
	}
	if fn := pkg.InitAbiTypesFor("subset", nil); fn != nil {
		t.Fatalf("InitAbiTypesFor with empty selection = %v, want nil", fn)
	}
}

func TestRtFuncResolvesLinkname(t *testing.T) {
	prog := NewProgram(nil)
	rt := types.NewPackage(PkgRuntime, PkgRuntime)
	sig := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(
			types.NewVar(0, nil, "env", types.NewPointer(types.NewStruct(nil, nil))),
			types.NewVar(0, nil, "savemask", types.Typ[types.Int32]),
		),
		types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Int32])),
		false,
	)
	if err := rt.Scope().Insert(types.NewFunc(token.NoPos, rt, "Sigsetjmp", sig)); err != nil {
		t.Fatal(err)
	}
	prog.SetRuntime(rt)
	prog.SetLinkname(PkgRuntime+".Sigsetjmp", "C.sigsetjmp")

	pkg := prog.NewPackage("foo", "foo")
	pkg.SetResolveLinkname(func(name string) string {
		if link, ok := prog.Linkname(name); ok {
			prefix, target, _ := strings.Cut(link, ".")
			if prefix == "C" {
				return target
			}
		}
		return name
	})

	if got := pkg.rtFunc("Sigsetjmp").impl.Name(); got != "sigsetjmp" {
		t.Fatalf("rtFunc linkname = %q, want %q", got, "sigsetjmp")
	}
}
