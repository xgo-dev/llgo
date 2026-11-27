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

	"github.com/xgo-dev/llgo/directive"
	"github.com/xgo-dev/llgo/target"
	"github.com/xgo-dev/llgo/types"
)

type testImporter func(string) (*types.Package, error)

func (f testImporter) Import(path string) (*types.Package, error) { return f(path) }

func checkSource(t *testing.T, path, src string, ctx *types.Context) (*types.Package, *types.Info) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
	}
	cfg := &types.Config{
		Context: ctx,
		Importer: testImporter(func(impPath string) (*types.Package, error) {
			if impPath == "unsafe" {
				return types.Unsafe, nil
			}
			return nil, types.Error{Msg: "unexpected import " + impPath}
		}),
	}
	if ctx != nil {
		cfg.Sizes = ctx
	}
	pkg, err := cfg.Check(path, fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatalf("check %s: %v", path, err)
	}
	return pkg, info
}

func TestTypesDirectivesAndPackageMetadata(t *testing.T) {
	src := `package demo

import _ "unsafe"

const LLGoPackage = "py.math"

//llgo:skip Foo Bar
type marker int

//llgo:link cSin C.sin
//go:noinline
//go:noescape
//go:uintptrescapes
//go:nosplit
func cSin(x float64) float64

//llgo:link winCb stdcall.WinProc
func winCb(x int32) int32

//llgo:link pySqrt py.sqrt
func pySqrt(x float64) float64

//llgo:link myAlloca llgo.alloca
func myAlloca[T any](n uintptr) *T { return nil }

//llgo:link llvmTrap llvm.trap
func llvmTrap()

//go:wasmimport wasi_snapshot_preview1 fd_write
func fdWrite(fd int32) int32

//go:wasmexport on_tick
func onTick() {}

//llgo:env
func envEntry() {}

//export exported_add
func exported_add(a, b int32) int32 { return a + b }

//go:linkname pkgScoped runtime.secret
func pkgScoped()

type Box struct{}
type BoxAlias = Box

//go:nointerface
//llgo:link (*Box).Touch C.box_touch
func (*BoxAlias) Touch()

var (
	//llgointernal:tls
	//llgo:link tlsSlot C.tls_slot
	tlsSlot uintptr

	//llgointernal:gls
	glsSlot uintptr
)
`
	pkg, _ := checkSource(t, "example.com/demo", src, nil)

	if pkg.Kind() != directive.PkgPyModule || pkg.KindParam() != "math" || pkg.PyModule() != "math" {
		t.Fatalf("pkg kind = (%v, %q, py=%q), want (PkgPyModule, math)", pkg.Kind(), pkg.KindParam(), pkg.PyModule())
	}
	if pkg.SkipsInit() || pkg.IsCPackage() || pkg.SkipAll() {
		t.Fatalf("unexpected pkg flags: SkipsInit=%v IsCPackage=%v SkipAll=%v", pkg.SkipsInit(), pkg.IsCPackage(), pkg.SkipAll())
	}
	if !pkg.Skipped("Foo") || !pkg.Skipped("Bar") || pkg.Skipped("Baz") {
		t.Fatalf("unexpected skip set: %v", pkg.SkipNames())
	}

	cSin := pkg.Scope().Lookup("cSin").(*types.Func)
	if ns, sym := cSin.LinkTarget(); ns != directive.LinkC || sym != "sin" {
		t.Fatalf("cSin.LinkTarget() = (%v, %q), want (LinkC, sin)", ns, sym)
	}
	if !cSin.NoInline() || !cSin.NoEscape() || !cSin.UintptrEscapes() || !cSin.NoSplit() || cSin.CallConv() != types.CDecl {
		t.Fatalf("cSin metadata = (NoInline=%v NoEscape=%v UintptrEscapes=%v NoSplit=%v CallConv=%v)",
			cSin.NoInline(), cSin.NoEscape(), cSin.UintptrEscapes(), cSin.NoSplit(), cSin.CallConv())
	}

	winCb := pkg.Scope().Lookup("winCb").(*types.Func)
	if winCb.CallConv() != types.Stdcall {
		t.Fatalf("winCb.CallConv() = %v, want Stdcall", winCb.CallConv())
	}

	pySqrt := pkg.Scope().Lookup("pySqrt").(*types.Func)
	if pySqrt.CallConv() != types.Py {
		t.Fatalf("pySqrt.CallConv() = %v, want Py", pySqrt.CallConv())
	}

	myAlloca := pkg.Scope().Lookup("myAlloca").(*types.Func)
	if ns, sym := myAlloca.LinkTarget(); ns != directive.LinkLLGo || sym != "alloca" {
		t.Fatalf("myAlloca.LinkTarget() = (%v, %q), want (LinkLLGo, alloca)", ns, sym)
	}

	llvmTrap := pkg.Scope().Lookup("llvmTrap").(*types.Func)
	if ns, sym := llvmTrap.LinkTarget(); ns != directive.LinkLLVM || sym != "llvm.trap" {
		t.Fatalf("llvmTrap.LinkTarget() = (%v, %q), want (LinkLLVM, llvm.trap)", ns, sym)
	}

	fdWrite := pkg.Scope().Lookup("fdWrite").(*types.Func)
	wi := fdWrite.WasmImport()
	if wi == nil || wi.Module != "wasi_snapshot_preview1" || wi.Name != "fd_write" || fdWrite.CallConv() != types.Wasm {
		t.Fatalf("fdWrite wasmimport = %+v, CallConv=%v", wi, fdWrite.CallConv())
	}

	onTick := pkg.Scope().Lookup("onTick").(*types.Func)
	if onTick.WasmExport() != "on_tick" {
		t.Fatalf("onTick.WasmExport() = %q, want on_tick", onTick.WasmExport())
	}

	envEntry := pkg.Scope().Lookup("envEntry").(*types.Func)
	if !envEntry.HasClosureEnv() {
		t.Fatal("envEntry.HasClosureEnv() = false, want true")
	}

	exp := pkg.Scope().Lookup("exported_add").(*types.Func)
	if exp.ExportName() != "exported_add" || exp.LinkName() != "exported_add" {
		t.Fatalf("exported_add export/link = (%q, %q)", exp.ExportName(), exp.LinkName())
	}

	scoped := pkg.Scope().Lookup("pkgScoped").(*types.Func)
	if scoped.LinkName() != "runtime.secret" {
		t.Fatalf("pkgScoped.LinkName() = %q, want runtime.secret", scoped.LinkName())
	}

	box := pkg.Scope().Lookup("Box").(*types.TypeName).Type().(*types.Named)
	if box.NumMethods() != 1 {
		t.Fatalf("Box.NumMethods() = %d, want 1", box.NumMethods())
	}
	touch := box.Method(0)
	if !touch.NoInterface() || touch.LinkName() != "C.box_touch" {
		t.Fatalf("Box.Touch = (NoInterface=%v, LinkName=%q)", touch.NoInterface(), touch.LinkName())
	}

	tlsSlot := pkg.Scope().Lookup("tlsSlot").(*types.Var)
	if tlsSlot.Locality() != directive.LocalityThread || tlsSlot.LinkName() != "C.tls_slot" {
		t.Fatalf("tlsSlot = (Locality=%v, LinkName=%q)", tlsSlot.Locality(), tlsSlot.LinkName())
	}
	glsSlot := pkg.Scope().Lookup("glsSlot").(*types.Var)
	if glsSlot.Locality() != directive.LocalityGoroutine {
		t.Fatalf("glsSlot.Locality() = %v, want LocalityGoroutine", glsSlot.Locality())
	}
}

func TestCPackageAndSkipAllDirectives(t *testing.T) {
	src := `package C

//llgo:skipall
const LLGoPackage = "decl"

func XAdd(a, b int32) int32 { return a + b }
func Sub(a, b int32) int32 { return a - b }
func helper() {}
`
	pkg, _ := checkSource(t, "example.com/clib", src, nil)
	if !pkg.IsCPackage() || pkg.Kind() != directive.PkgDeclOnly || !pkg.SkipsInit() || !pkg.SkipAll() {
		t.Fatalf("unexpected C pkg metadata: IsCPackage=%v Kind=%v SkipsInit=%v SkipAll=%v",
			pkg.IsCPackage(), pkg.Kind(), pkg.SkipsInit(), pkg.SkipAll())
	}
	xadd := pkg.Scope().Lookup("XAdd").(*types.Func)
	if xadd.ExportName() != "Add" || xadd.LinkName() != "Add" {
		t.Fatalf("XAdd export/link = (%q, %q), want (Add, Add)", xadd.ExportName(), xadd.LinkName())
	}
	sub := pkg.Scope().Lookup("Sub").(*types.Func)
	if sub.ExportName() != "Sub" || sub.LinkName() != "Sub" {
		t.Fatalf("Sub export/link = (%q, %q), want (Sub, Sub)", sub.ExportName(), sub.LinkName())
	}
	helper := pkg.Scope().Lookup("helper").(*types.Func)
	if helper.ExportName() != "" || helper.LinkName() != "" {
		t.Fatalf("unexported helper unexpectedly exported: (%q, %q)", helper.ExportName(), helper.LinkName())
	}
}

func TestNativeStructDirectivesAutoComputeWasm32Offsets(t *testing.T) {
	ctx := types.NewContextWithConfig(types.ContextConfig{
		Spec: target.ResolveSpec(target.Spec{
			GOOS:        "js",
			GOARCH:      "wasm",
			WasmProfile: "j32",
		}),
	})
	src := `package native

//llgo:cdecl
type CStruct struct {
	A uintptr
	B uint32
}

type GoStruct struct {
	A uintptr
	B uint32
}
`
	pkg, _ := checkSource(t, "example.com/native", src, ctx)
	cNamed := pkg.Scope().Lookup("CStruct").(*types.TypeName).Type().(*types.Named)
	goNamed := pkg.Scope().Lookup("GoStruct").(*types.TypeName).Type().(*types.Named)

	cSt := cNamed.Underlying().(*types.Struct)
	goSt := goNamed.Underlying().(*types.Struct)
	cOffs := ctx.Offsetsof([]*types.Var{cSt.Field(0), cSt.Field(1)})
	goOffs := ctx.Offsetsof([]*types.Var{goSt.Field(0), goSt.Field(1)})
	if cOffs[1] != 4 {
		t.Fatalf("CStruct field B offset on wasm32/j32 = %d, want 4", cOffs[1])
	}
	if goOffs[1] != 8 {
		t.Fatalf("GoStruct field B offset on wasm32/j32 = %d, want 8", goOffs[1])
	}
}

func TestCheckerPreservesPackageIdentityDuringUnsafeSizeof(t *testing.T) {
	ctx := types.NewContextWithConfig(types.ContextConfig{
		Spec: target.ResolveSpec(target.Spec{GOOS: "linux", GOARCH: "amd64"}),
	})
	const src = `package p

import "unsafe"

type Inner struct {
	X int
	F func()
}

type Outer struct {
	Inner
	Y int
}

const (
	_ = unsafe.Sizeof(Inner{})
	_ = unsafe.Offsetof(Outer{}.Y)
)
`
	pkg, _ := checkSource(t, "example.com/identity", src, ctx)
	for _, name := range []string{"Inner", "Outer"} {
		obj := pkg.Scope().Lookup(name)
		if obj == nil {
			t.Fatalf("missing %s", name)
		}
		if obj.Pkg() != pkg {
			t.Fatalf("%s.Pkg() = %p, want %p (orphan duplicate package created during Sizes query)", name, obj.Pkg(), pkg)
		}
	}
	outer := pkg.Scope().Lookup("Outer").Type().(*types.Named).Underlying().(*types.Struct)
	embedded := outer.Field(0).Type().(*types.Named)
	if embedded.Obj().Pkg() != pkg {
		t.Fatalf("Outer.Inner Obj().Pkg() = %p, want %p", embedded.Obj().Pkg(), pkg)
	}
}

func TestContextSetTypeCallConvPreservesGenericNamedIdentity(t *testing.T) {
	ctx := types.NewContext()
	const src = `package p

type Box[T any] struct {
	V T
}

var IntBox Box[int]
`
	pkg, _ := checkSource(t, "example.com/genericid", src, ctx)
	originBefore := pkg.Scope().Lookup("Box").Type().(*types.Named)
	instBefore := pkg.Scope().Lookup("IntBox").Type().(*types.Named)
	if instBefore.Origin() != originBefore {
		t.Fatalf("IntBox.Origin() = %p, want %p", instBefore.Origin(), originBefore)
	}
	// Registering an unrelated native calling convention must not invalidate
	// existing canonical *Named identities in the shared Context.
	ctx.SetTypeCallConv("example.com/other.Callback", types.CDecl)
	if got := pkg.Scope().Lookup("Box").Type().(*types.Named); got != originBefore {
		t.Fatalf("Box identity changed after unrelated SetTypeCallConv")
	}
	if got := pkg.Scope().Lookup("IntBox").Type().(*types.Named).Origin(); got != originBefore {
		t.Fatalf("IntBox.Origin() changed after unrelated SetTypeCallConv")
	}
}
