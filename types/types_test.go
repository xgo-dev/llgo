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
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	gotypes "go/types"
	"hash/maphash"
	"strings"
	"sync"
	"testing"

	"github.com/xgo-dev/llgo/target"
	"github.com/xgo-dev/llgo/types"
)

type mapImporter struct {
	pkgs      map[string]*types.Package
	fromCalls int
}

func (m *mapImporter) Import(path string) (*types.Package, error) {
	if p, ok := m.pkgs[path]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("unknown package %q", path)
}

func (m *mapImporter) ImportFrom(path, dir string, mode types.ImportMode) (*types.Package, error) {
	m.fromCalls++
	return m.Import(path)
}

func fullInfo() *types.Info {
	return &types.Info{
		Types:        make(map[ast.Expr]types.TypeAndValue),
		Instances:    make(map[*ast.Ident]types.Instance),
		Defs:         make(map[*ast.Ident]types.Object),
		Uses:         make(map[*ast.Ident]types.Object),
		Implicits:    make(map[ast.Node]types.Object),
		Selections:   make(map[*ast.SelectorExpr]*types.Selection),
		Scopes:       make(map[ast.Node]*types.Scope),
		FileVersions: make(map[*ast.File]string),
	}
}

func newContextForSpec(spec target.Spec) *types.Context {
	return types.NewContextWithConfig(types.ContextConfig{Spec: spec})
}

func TestConfigCheckEndToEnd(t *testing.T) {
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

var (
	boxInt    Box[int]
	boxString Box[string]
)

func Use(nArg int) (int, string, *Node) {
	n := &Node{Val: int64(nArg)}
	return boxInt.Get(), boxString.Get(), n
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "demo.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	spec := target.ResolveSpec(target.Spec{GOOS: "linux", GOARCH: "amd64"})
	ctx := newContextForSpec(spec)
	info := fullInfo()
	cfg := &types.Config{
		Context:   ctx,
		GoVersion: "go1.20",
		Importer:  &mapImporter{pkgs: map[string]*types.Package{"unsafe": types.Unsafe}},
	}
	pkg, err := cfg.Check("example.com/demo", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatalf("Config.Check failed: %v", err)
	}

	if pkg.Path() != "example.com/demo" || pkg.Name() != "demo" || !pkg.Complete() || pkg.GoVersion() != "go1.20" {
		t.Fatalf("unexpected pkg metadata: %s complete=%v ver=%q", pkg, pkg.Complete(), pkg.GoVersion())
	}
	if len(pkg.Imports()) != 1 || pkg.Imports()[0] != types.Unsafe {
		t.Fatalf("unexpected imports: %v", pkg.Imports())
	}
	if info.FileVersions[file] != "go1.20" {
		t.Fatalf("FileVersions = %q, want go1.20", info.FileVersions[file])
	}

	// Verify compile-time unsafe.Sizeof constants folded by go/types using ctx.GoSizes()!
	checkConst := func(name string, want int64) {
		t.Helper()
		obj, ok := pkg.Scope().Lookup(name).(*types.Const)
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

	// Verify Signature CallConv and Variadic semantics.
	cCb := pkg.Scope().Lookup("CCallback").(*types.TypeName).Type().(*types.Named)
	cSig := cCb.Underlying().(*types.Signature)
	if cSig.CallConv() != types.CDecl || !cSig.IsNative() || cSig.IsClosure() || !cSig.Variadic() {
		t.Fatalf("CCallback signature = %s (cc=%v variadic=%v)", cSig, cSig.CallConv(), cSig.Variadic())
	}
	if ctx.Layout(cCb).Size != 8 || cCb.HasPtrData() {
		t.Fatalf("CCallback layout=%+v hasPtr=%v", ctx.Layout(cCb), cCb.HasPtrData())
	}

	winCb := pkg.Scope().Lookup("WinCallback").(*types.TypeName).Type().(*types.Named)
	if winCb.CallConv() != types.Stdcall || winCb.Underlying().(*types.Signature).CallConv() != types.Stdcall {
		t.Fatalf("WinCallback callconv = %v", winCb.CallConv())
	}

	goFn := pkg.Scope().Lookup("GoFn").(*types.TypeName).Type().(*types.Named)
	goSig := goFn.Underlying().(*types.Signature)
	if !goSig.IsClosure() || goSig.IsNative() || !goSig.HasPtrData() || ctx.Layout(goFn).Size != 16 {
		t.Fatalf("GoFn sig=%s layout=%+v", goSig, ctx.Layout(goFn))
	}

	// Verify Func.Type() and Go function value types are Identical (matching go/types!).
	useFn := pkg.Scope().Lookup("Use").(*types.Func)
	if !useFn.Signature().IsClosure() {
		t.Fatalf("Use func signature should be Go closure")
	}
	if useFn.Signature().Params().At(0).Kind() != types.ParamVar || useFn.Signature().Results().At(0).Kind() != types.ResultVar {
		t.Fatalf("VarKind mismatch: param=%v result=%v", useFn.Signature().Params().At(0).Kind(), useFn.Signature().Results().At(0).Kind())
	}
	wantUseSig := types.NewSignature(
		nil,
		types.NewTuple(types.NewParam(0, nil, "", types.Typ[types.Int])),
		types.NewTuple(
			types.NewParam(0, nil, "", types.Typ[types.Int]),
			types.NewParam(0, nil, "", types.Typ[types.String]),
			types.NewParam(0, nil, "", types.NewPointer(pkg.Scope().Lookup("Node").Type())),
		),
		false,
	)
	if !types.Identical(useFn.Type(), wantUseSig) {
		t.Fatalf("Func.Type() not Identical to equivalent Go Signature: %s vs %s", useFn.Type(), wantUseSig)
	}

	// Verify generic instantiations are distinct *Named types sharing the same Origin!
	bi := pkg.Scope().Lookup("boxInt").(*types.Var).Type().(*types.Named)
	bs := pkg.Scope().Lookup("boxString").(*types.Var).Type().(*types.Named)
	if pkg.Scope().Lookup("boxInt").(*types.Var).Kind() != types.PackageVar {
		t.Fatalf("boxInt.Kind() = %v, want PackageVar", pkg.Scope().Lookup("boxInt").(*types.Var).Kind())
	}
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
	node := pkg.Scope().Lookup("Node").(*types.TypeName).Type().(*types.Named)
	if node.Underlying().(*types.Struct).Field(0).Kind() != types.FieldVar {
		t.Fatalf("Node.Val.Kind() = %v, want FieldVar", node.Underlying().(*types.Struct).Field(0).Kind())
	}
	nodeLay := ctx.Layout(node)
	if nodeLay.Size != 16 || nodeLay.Align != 8 || nodeLay.PtrBytes != 16 || len(nodeLay.PtrOffsets) != 1 || nodeLay.PtrOffsets[0] != 8 {
		t.Fatalf("Node layout = %+v", nodeLay)
	}
	if len(nodeLay.GCBitmap) != 1 || nodeLay.GCBitmap[0] != 0b10 {
		t.Fatalf("Node GCBitmap = %08b, want 0b10", nodeLay.GCBitmap)
	}

	// Verify NewMethodSet & Info.Selections speak native llgo/types!
	ms := types.NewMethodSet(types.NewPointer(bi))
	sel := ms.Lookup(pkg, "Get")
	if sel == nil || sel.Kind() != types.MethodVal {
		t.Fatalf("NewMethodSet(*Box[int]).Lookup(Get) = %+v", sel)
	}
	if sel.Obj().(*types.Func).Signature().Recv().Kind() != types.RecvVar {
		t.Fatalf("Get recv VarKind = %v, want RecvVar", sel.Obj().(*types.Func).Signature().Recv().Kind())
	}
	if len(info.Selections) == 0 || len(info.Instances) == 0 || len(info.Types) == 0 {
		t.Fatalf("expected non-empty Info maps: selections=%d instances=%d types=%d", len(info.Selections), len(info.Instances), len(info.Types))
	}
	for expr, tv := range info.Types {
		if tv.Type != nil && info.TypeOf(expr) != tv.Type {
			t.Fatalf("Info.TypeOf mismatch for %T", expr)
		}
	}

	// Eval, CheckExpr & Instantiate on checked package!
	tv, err := types.Eval(fset, pkg, token.NoPos, "SizeGoFn + SizeCFn")
	if err != nil {
		t.Fatalf("Eval failed: %v", err)
	}
	if v, ok := constant.Int64Val(tv.Value); !ok || v != 24 {
		t.Fatalf("Eval(SizeGoFn + SizeCFn) = %v", tv.Value)
	}
	exprAST, err := parser.ParseExpr("boxInt.Item")
	if err != nil {
		t.Fatal(err)
	}
	checkInfo := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue)}
	if err := types.CheckExpr(fset, pkg, token.NoPos, exprAST, checkInfo); err != nil {
		t.Fatalf("CheckExpr failed: %v", err)
	}
	if got := checkInfo.TypeOf(exprAST); !types.Identical(got, types.Typ[types.Int]) {
		t.Fatalf("CheckExpr(boxInt.Item) type = %v, want int", got)
	}
	boxOrig := pkg.Scope().Lookup("Box").Type()
	instFloat, err := types.Instantiate(ctx, boxOrig, []types.Type{types.Typ[types.Float64]}, true)
	if err != nil {
		t.Fatalf("Instantiate(Box, [float64]) failed: %v", err)
	}
	if instNamed, ok := instFloat.(*types.Named); !ok || instNamed.TypeArgs().Len() != 1 || instNamed.TypeArgs().At(0) != types.Typ[types.Float64] {
		t.Fatalf("Instantiate(Box, [float64]) = %T %v", instFloat, instFloat)
	}
}

func TestCheckerImporterErrorsAndInstantiate(t *testing.T) {
	fset := token.NewFileSet()
	depFile, err := parser.ParseFile(fset, "dep.go", `package dep
type Pair[K comparable, V any] struct {
	Key K
	Val V
}
const Answer = 42
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	ctx := types.NewContext()
	depPkg := types.NewPackage("example.com/dep", "dep")
	depInfo := fullInfo()
	checker := types.NewChecker(&types.Config{
		Context: ctx,
		Sizes:   &types.StdSizes{WordSize: 8, MaxAlign: 8},
	}, fset, depPkg, depInfo)
	if err := checker.Files([]*ast.File{depFile}); err != nil {
		t.Fatalf("Checker.Files(dep) failed: %v", err)
	}
	if !depPkg.Complete() || depPkg.Scope().Lookup("Answer") == nil {
		t.Fatal("Checker.Files did not populate depPkg")
	}

	// Import depPkg into a second package via ImporterFrom!
	imp := &mapImporter{pkgs: map[string]*types.Package{"example.com/dep": depPkg}}
	appFile, err := parser.ParseFile(fset, "app.go", `package app
import "example.com/dep"
var P dep.Pair[string, int]
var X = dep.Answer
`, 0)
	if err != nil {
		t.Fatal(err)
	}
	appPkg, err := (&types.Config{
		Context:  ctx,
		Importer: imp,
	}).Check("example.com/app", fset, []*ast.File{appFile}, nil)
	if err != nil {
		t.Fatalf("Config.Check(app) failed: %v", err)
	}
	if imp.fromCalls == 0 || appPkg.Scope().Lookup("X") == nil {
		t.Fatalf("ImporterFrom not invoked or missing X: calls=%d", imp.fromCalls)
	}

	// Instantiate constraint violation returns *types.ArgumentError.
	pairType := depPkg.Scope().Lookup("Pair").Type()
	_, err = types.Instantiate(ctx, pairType, []types.Type{types.NewSlice(types.Typ[types.Int]), types.Typ[types.Int]}, true)
	var argErr *types.ArgumentError
	if !errors.As(err, &argErr) || argErr.Index != 0 || argErr.Error() == "" || argErr.Unwrap() == nil {
		t.Fatalf("expected *ArgumentError at index 0, got %T: %v", err, err)
	}

	// Config.Error receives llgo/types.Error.
	badFile, err := parser.ParseFile(fset, "bad.go", `package bad; var x int = "oops"`, 0)
	if err != nil {
		t.Fatal(err)
	}
	var gotErrs []types.Error
	_, checkErr := (&types.Config{
		Error: func(e error) {
			if te, ok := e.(types.Error); ok {
				gotErrs = append(gotErrs, te)
			}
		},
	}).Check("example.com/bad", fset, []*ast.File{badFile}, nil)
	if checkErr == nil || len(gotErrs) == 0 || !strings.Contains(gotErrs[0].Error(), "bad.go:") {
		t.Fatalf("expected types.Error, got checkErr=%v gotErrs=%v", checkErr, gotErrs)
	}

	// Synthetic pure-Go Instantiate (*Named, *Alias, *Signature) + validation.
	synPkg := types.NewPackage("example.com/syn", "syn")
	compConstraint := types.Universe.Lookup("comparable").Type()
	tp := types.NewTypeParam(types.NewTypeName(0, synPkg, "T", nil), compConstraint)
	synSt := types.NewStruct([]*types.Var{types.NewField(0, synPkg, "V", tp, false)}, nil)
	synNamed := types.NewNamed(types.NewTypeName(0, synPkg, "Cell", nil), synSt, nil)
	synNamed.SetTypeParams([]*types.TypeParam{tp})
	synMethodSig := types.NewSignature(types.NewParam(0, synPkg, "c", synNamed), nil, types.NewTuple(types.NewParam(0, synPkg, "", tp)), false)
	synNamed.AddMethod(types.NewFunc(0, synPkg, "Val", synMethodSig))

	instCell, err := types.Instantiate(nil, synNamed, []types.Type{types.Typ[types.Int]}, true)
	if err != nil {
		t.Fatalf("synthetic Instantiate(Cell, [int]) failed: %v", err)
	}
	cn := instCell.(*types.Named)
	if cn.Origin() != synNamed || cn.Underlying().(*types.Struct).Field(0).Type() != types.Typ[types.Int] ||
		cn.Method(0).Signature().Results().At(0).Type() != types.Typ[types.Int] {
		t.Fatalf("synthetic instantiated Cell mismatch: %v", cn)
	}
	if _, err := types.Instantiate(nil, synNamed, []types.Type{types.NewSlice(types.Typ[types.Int])}, true); !errors.As(err, &argErr) {
		t.Fatalf("synthetic Instantiate constraint error = %T %v, want *ArgumentError", err, err)
	}
	synAlias := types.NewAlias(types.NewTypeName(0, synPkg, "CellA", nil), types.NewSlice(tp))
	synAlias.SetTypeParams([]*types.TypeParam{tp})
	instAlias, err := types.Instantiate(nil, synAlias, []types.Type{types.Typ[types.String]}, true)
	if err != nil || !types.Identical(instAlias.(*types.Alias).Rhs(), types.NewSlice(types.Typ[types.String])) {
		t.Fatalf("synthetic Instantiate(Alias) = %v, err=%v", instAlias, err)
	}
	synSig := types.NewSignatureType(nil, nil, []*types.TypeParam{tp}, types.NewTuple(types.NewParam(0, synPkg, "x", tp)), nil, false)
	instSig, err := types.Instantiate(nil, synSig, []types.Type{types.Typ[types.Int64]}, true)
	if err != nil || instSig.(*types.Signature).Params().At(0).Type() != types.Typ[types.Int64] {
		t.Fatalf("synthetic Instantiate(Signature) = %v, err=%v", instSig, err)
	}
}

func TestPointerKindsAndIdentical(t *testing.T) {
	ctx := newContextForSpec(target.Spec{GOOS: "linux", GOARCH: "amd64"})
	i32 := types.Typ[types.Int32]

	goP := types.NewPointerWithKind(i32, types.GoPtr)
	unsafeP := types.NewPointerWithKind(i32, types.UnsafePtr)
	cP := types.NewPointerWithKind(i32, types.CPtr)

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
	ctx := newContextForSpec(target.Spec{GOOS: "linux", GOARCH: "amd64"})
	pkg, err := (&types.Config{
		Context:  ctx,
		Importer: &mapImporter{pkgs: map[string]*types.Package{"unsafe": types.Unsafe}},
	}).Check("rtlike", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatalf("Check failed on mutually-recursive runtime-like types: %v", err)
	}
	cObj := pkg.Scope().Lookup("hchanSize").(*types.Const)
	got, ok := constant.Int64Val(cObj.Val())
	if !ok || got != 48 {
		t.Fatalf("hchanSize = %v, want 48", cObj.Val())
	}
}

func TestSingleEngineLayoutsHostWasmSIMDAndStdSizes(t *testing.T) {
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

		armCtx := newContextForSpec(target.Spec{GOOS: "linux", GOARCH: "arm"})
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

		i386Ctx := newContextForSpec(target.Spec{GOOS: "linux", GOARCH: "386"})
		if i386Ctx.HostUsesNativeGoLayout() {
			t.Fatal("expected linux/386 HostUsesNativeGoLayout = false")
		}
		if got := i386Ctx.GoSizes().Alignof(goSt); got != 4 {
			t.Fatalf("linux/386 HostLayout Alignof = %d, want 4", got)
		}
	})

	t.Run("Wasm32 wide Go vs native C layout", func(t *testing.T) {
		spec := target.ResolveSpec(target.Spec{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32", WasmProvider: "wasi"})
		ctx := newContextForSpec(spec)
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

		amd64Ctx := newContextForSpec(target.Spec{GOOS: "linux", GOARCH: "amd64"})
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

		arm64Ctx := newContextForSpec(target.Spec{GOOS: "darwin", GOARCH: "arm64"})
		armLay := arm64Ctx.Layout(arm64Ctx.FromGo(i32x8))
		if armLay.Size != 32 || armLay.Align != 16 || !armLay.HasSIMDStorage {
			t.Fatalf("arm64 Int32x8 layout = %+v", armLay)
		}
	})

	t.Run("Legacy closure struct and StdSizes", func(t *testing.T) {
		ctx := newContextForSpec(target.Spec{GOOS: "linux", GOARCH: "amd64"})
		sig := gotypes.NewSignatureType(nil, nil, nil, nil, nil, false)
		cloSt := gotypes.NewStruct([]*gotypes.Var{
			gotypes.NewField(0, nil, "$f", sig, false),
			gotypes.NewField(0, nil, "$data", gotypes.Typ[gotypes.UnsafePointer], false),
		}, nil)
		lay := ctx.Layout(ctx.FromGo(cloSt))
		if lay.Size != 16 || len(lay.Fields) != 2 || lay.Fields[0].Offset != 0 || lay.Fields[1].Offset != 8 {
			t.Fatalf("legacy closure struct layout = %+v", lay)
		}

		std := &types.StdSizes{WordSize: 8, MaxAlign: 8}
		f0 := types.NewField(0, nil, "a", types.Typ[types.Int8], false)
		f1 := types.NewField(0, nil, "b", types.Typ[types.Int64], false)
		st := types.NewStruct([]*types.Var{f0, f1}, nil)
		if std.Alignof(st) != 8 || std.Sizeof(st) != 16 {
			t.Fatalf("StdSizes(st) = align %d size %d", std.Alignof(st), std.Sizeof(st))
		}
		offs := std.Offsetsof([]*types.Var{f0, f1})
		if len(offs) != 2 || offs[0] != 0 || offs[1] != 8 {
			t.Fatalf("StdSizes.Offsetsof = %v", offs)
		}
		if std.Sizeof(types.NewSlice(types.Typ[types.Int])) != 24 || std.Sizeof(types.Typ[types.String]) != 16 {
			t.Fatal("StdSizes slice/string mismatch")
		}
	})
}

func TestNativeIntExtDataKindAndEqualName(t *testing.T) {
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

	ctxArm64 := newContextForSpec(target.ResolveSpec(target.Spec{GOOS: "darwin", GOARCH: "arm64"}))
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
	ctx := newContextForSpec(target.Spec{GOOS: "linux", GOARCH: "amd64"})
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

	pkg := types.NewPackage("example.com/p", "p")
	v := types.NewVar(0, pkg, "x", types.Typ[types.Int])
	pkg.Scope().Insert(v)
	if _, found := pkg.Scope().LookupParent("x", token.NoPos); found != v {
		t.Fatalf("Scope.LookupParent(x) = %v", found)
	}
	if !strings.Contains(v.String(), "var x int") {
		t.Fatalf("Var.String() = %q", v.String())
	}
}

func TestGoTypesAlignmentUnionPredicatesHasherAndFormatting(t *testing.T) {
	var typSlice []*types.Basic = types.Typ
	if len(typSlice) <= int(types.UntypedNil) || types.Unsafe == nil || types.Unsafe.Scope().Lookup("Pointer") == nil {
		t.Fatal("Typ slice or Unsafe package mismatch")
	}

	src := `package aligned

import u "unsafe"

type Signed interface {
	~int | ~int64
}

type MyInt int

type Reader interface {
	Read(p []byte) int
}

type ReadCloser interface {
	Reader
	Close()
}

type Base struct{ X int }
func (b Base) Read(p []byte) int { return b.X }
func (b *Base) Close() {}

type Derived struct {
	Base
	Y int
}

func Sum[T Signed](a, b T) T { return a + b }
func Noop() {}

var (
	_ = u.Sizeof(MyInt(0))
	V = Sum(MyInt(1), MyInt(2))
)

func Use() { Noop() }
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "aligned.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	ctx := newContextForSpec(target.Spec{GOOS: "linux", GOARCH: "amd64"})
	info := fullInfo()
	pkg, err := (&types.Config{
		Context:  ctx,
		Importer: &mapImporter{pkgs: map[string]*types.Package{"unsafe": types.Unsafe}},
	}).Check("example.com/aligned", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if pkg.Imports()[0] != types.Unsafe {
		t.Fatalf("unsafe import did not resolve to types.Unsafe singleton: %v", pkg.Imports()[0])
	}
	if pn := info.PkgNameOf(file.Imports[0]); pn == nil || pn.Name() != "u" || pn.Imported() != types.Unsafe {
		t.Fatalf("Info.PkgNameOf = %v", pn)
	}

	signedIface := pkg.Scope().Lookup("Signed").Type().Underlying().(*types.Interface)
	if !signedIface.IsComparable() || signedIface.IsMethodSet() || signedIface.Empty() || signedIface.NumEmbeddeds() != 1 {
		t.Fatalf("Signed interface flags: comp=%v mset=%v empty=%v embs=%d", signedIface.IsComparable(), signedIface.IsMethodSet(), signedIface.Empty(), signedIface.NumEmbeddeds())
	}
	union, ok := signedIface.EmbeddedType(0).(*types.Union)
	if !ok || union.Len() != 2 || !union.Term(0).Tilde() || union.Term(0).Type() != types.Typ[types.Int] || union.String() != "~int | ~int64" {
		t.Fatalf("Signed union = %T %v", signedIface.EmbeddedType(0), signedIface.EmbeddedType(0))
	}
	myInt := pkg.Scope().Lookup("MyInt").Type()
	if !types.Satisfies(myInt, signedIface) || !types.Satisfies(types.Typ[types.Int64], signedIface) || types.Satisfies(types.Typ[types.String], signedIface) {
		t.Fatal("Satisfies(Signed) mismatch")
	}
	if !types.Comparable(myInt) || types.Comparable(types.NewSlice(types.Typ[types.Int])) || !types.IsInterface(signedIface) || types.IsInterface(myInt) {
		t.Fatal("Comparable/IsInterface mismatch")
	}
	if types.Default(types.Typ[types.UntypedInt]) != types.Typ[types.Int] || types.Default(types.Typ[types.UntypedNil]) != types.Typ[types.UntypedNil] {
		t.Fatal("Default mismatch")
	}

	rc := pkg.Scope().Lookup("ReadCloser").Type().Underlying().(*types.Interface)
	if rc.NumExplicitMethods() != 1 || rc.ExplicitMethod(0).Name() != "Close" || rc.NumMethods() != 2 || rc.IsComparable() || !rc.IsMethodSet() {
		t.Fatalf("ReadCloser interface: explicit=%d total=%d comp=%v mset=%v", rc.NumExplicitMethods(), rc.NumMethods(), rc.IsComparable(), rc.IsMethodSet())
	}
	reader := pkg.Scope().Lookup("Reader").Type().Underlying().(*types.Interface)
	derived := pkg.Scope().Lookup("Derived").Type()
	ptrDerived := types.NewPointer(derived)
	if !types.Implements(derived, reader) || types.Implements(derived, rc) || !types.Implements(ptrDerived, rc) {
		t.Fatal("Implements mismatch on Derived/*Derived")
	}
	if m, wrong := types.MissingMethod(derived, rc, true); m == nil || m.Name() != "Close" || !wrong {
		t.Fatalf("MissingMethod(Derived, ReadCloser) = (%v, %v), want (Close, true)", m, wrong)
	}
	if !types.AssertableTo(reader, ptrDerived) || types.AssertableTo(rc, derived) {
		t.Fatal("AssertableTo mismatch")
	}

	// LookupFieldOrMethod & LookupSelection.
	if obj, idx, ind := types.LookupFieldOrMethod(derived, false, pkg, "X"); obj == nil || obj.Name() != "X" || len(idx) != 2 || ind {
		t.Fatalf("LookupFieldOrMethod(Derived, X) = (%v, %v, %v)", obj, idx, ind)
	}
	if sel, ok := types.LookupSelection(derived, false, pkg, "X"); !ok || sel.Kind() != types.FieldVal || sel.Obj().Name() != "X" {
		t.Fatalf("LookupSelection(Derived, X) = (%v, %v)", sel, ok)
	}
	if sel, ok := types.LookupSelection(derived, true, pkg, "Close"); !ok || sel.Kind() != types.MethodVal || sel.Obj().Name() != "Close" {
		t.Fatalf("LookupSelection(addr Derived, Close) = (%v, %v)", sel, ok)
	}
	if _, ok := types.LookupSelection(derived, false, pkg, "Missing"); ok {
		t.Fatal("LookupSelection(Missing) should be false")
	}
	if obj, _, ind := types.LookupFieldOrMethod(derived, false, pkg, "Close"); obj != nil || !ind {
		t.Fatalf("LookupFieldOrMethod(non-addr Derived, Close) = (%v, %v), want (nil, true)", obj, ind)
	}
	if ms := types.NewMethodSet(ptrDerived); ms.Len() != 2 || !strings.Contains(ms.String(), "method") {
		t.Fatalf("NewMethodSet(*Derived) = len %d %s", ms.Len(), ms.String())
	}

	// Hasher & HasherIgnoreTags consistency with Identical / IdenticalIgnoreTags.
	s1 := types.NewStruct([]*types.Var{types.NewField(0, pkg, "A", types.Typ[types.Int], false)}, []string{`json:"a"`})
	s2 := types.NewStruct([]*types.Var{types.NewField(0, pkg, "A", types.Typ[types.Int], false)}, []string{`json:"b"`})
	if types.Identical(s1, s2) || !types.IdenticalIgnoreTags(s1, s2) || !types.ConvertibleTo(s1, s2) {
		t.Fatal("IdenticalIgnoreTags/ConvertibleTo mismatch on tagged structs")
	}
	seed := maphash.MakeSeed()
	hashOf := func(fn func(*maphash.Hash, types.Type), typ types.Type) uint64 {
		var h maphash.Hash
		h.SetSeed(seed)
		fn(&h, typ)
		return h.Sum64()
	}
	var hasher types.Hasher
	var hasherNoTags types.HasherIgnoreTags
	if hasher.Equal(s1, s2) || !hasherNoTags.Equal(s1, s2) {
		t.Fatal("Hasher.Equal / HasherIgnoreTags.Equal mismatch")
	}
	if hashOf(hasherNoTags.Hash, s1) != hashOf(hasherNoTags.Hash, s2) {
		t.Fatal("HasherIgnoreTags produced different hashes for tag-only diff")
	}
	aliasInt := types.NewAlias(types.NewTypeName(0, pkg, "IntAlias", nil), types.Typ[types.Int])
	if !hasher.Equal(aliasInt, types.Typ[types.Int]) || hashOf(hasher.Hash, aliasInt) != hashOf(hasher.Hash, types.Typ[types.Int]) {
		t.Fatal("Hasher alias hash mismatch")
	}
	for _, sample := range []types.Type{
		nil, types.Typ[types.Int], types.NewPointer(types.Typ[types.Int]),
		types.NewSlice(types.Typ[types.Int]), types.NewArray(types.Typ[types.Int], 3),
		types.NewMap(types.Typ[types.String], types.Typ[types.Int]),
		types.NewChan(types.SendRecv, types.Typ[types.Int]),
		types.NewTuple(types.NewParam(0, nil, "a", types.Typ[types.Int])),
		types.NewSignature(nil, nil, nil, false),
		signedIface, rc, union, derived,
	} {
		if hashOf(hasher.Hash, sample) != hashOf(hasher.Hash, sample) {
			t.Fatalf("non-deterministic Hasher for %v", sample)
		}
	}

	goSig := types.NewSignature(nil, nil, nil, false)
	cSig := types.NewSignatureWithCallConv(nil, nil, nil, false, types.CDecl)
	if types.AssignableTo(goSig, cSig) || !types.AssignableTo(types.Typ[types.UntypedNil], cSig) || !types.AssignableTo(ptrDerived, rc) {
		t.Fatal("AssignableTo CallConv/interface mismatch")
	}
	if !types.ConvertibleTo(types.Typ[types.Int32], types.Typ[types.Float64]) || !types.ConvertibleTo(types.NewSlice(types.Typ[types.Int]), types.NewArray(types.Typ[types.Int], 2)) {
		t.Fatal("ConvertibleTo numeric/slice-to-array mismatch")
	}

	sc := types.NewScope(pkg.Scope(), token.Pos(10), token.Pos(50), "fn")
	inner := types.NewScope(sc, token.Pos(20), token.Pos(30), "block")
	if !sc.Contains(token.Pos(15)) || sc.Contains(token.Pos(50)) || sc.Innermost(token.Pos(25)) != inner || sc.Innermost(token.Pos(15)) != sc {
		t.Fatal("Scope.Contains/Innermost mismatch")
	}
	if !strings.Contains(sc.String(), "fn scope") {
		t.Fatalf("Scope.String() = %q", sc.String())
	}
	var buf bytes.Buffer
	sc.WriteTo(&buf, 0, true)
	if !strings.Contains(buf.String(), "block scope") {
		t.Fatalf("Scope.WriteTo recurse = %q", buf.String())
	}

	tmpPkg := types.NewPackage("example.com/tmp", "old")
	tmpPkg.SetName("renamed")
	if tmpPkg.Name() != "renamed" {
		t.Fatal("Package.SetName failed")
	}

	tp := types.NewTypeParam(types.NewTypeName(0, tmpPkg, "T", nil), nil)
	tp.SetConstraint(types.NewUnion([]*types.Term{types.NewTerm(true, types.Typ[types.Int])}))
	genNamed := types.NewNamed(types.NewTypeName(0, tmpPkg, "Gen", nil), types.Typ[types.Int], nil)
	genNamed.SetTypeParams([]*types.TypeParam{tp})
	genAlias := types.NewAlias(types.NewTypeName(0, tmpPkg, "GenAlias", nil), genNamed)
	genAlias.SetTypeParams([]*types.TypeParam{tp})
	if genNamed.TypeParams().String() != "[T ~int]" || genAlias.TypeParams().Len() != 1 {
		t.Fatalf("TypeParams mismatch: %s", genNamed.TypeParams().String())
	}
	impIface := types.NewInterfaceType(nil, []types.Type{types.NewUnion([]*types.Term{types.NewTerm(false, types.Typ[types.Int])})}).Complete()
	impIface.MarkImplicit()
	if !impIface.IsImplicit() || !impIface.IsComparable() || impIface.IsMethodSet() {
		t.Fatal("synthetic constraint Interface flags mismatch")
	}

	if len(info.InitOrder) == 0 || !strings.Contains(info.InitOrder[0].String(), "=") {
		t.Fatalf("Initializer.String = %v", info.InitOrder)
	}
	for _, inst := range info.Instances {
		if !strings.Contains(inst.String(), "->") || inst.TypeArgs.String() == "" {
			t.Fatalf("Instance.String = %q", inst.String())
		}
	}
	sawVoid := false
	for _, tv := range info.Types {
		if tv.IsVoid() {
			sawVoid = true
		}
	}
	if !sawVoid {
		t.Fatal("expected at least one IsVoid() expression for Noop()")
	}
	buf.Reset()
	types.WriteSignature(&buf, cSig, nil)
	if buf.String() != "<cdecl>()" {
		t.Fatalf("WriteSignature(cSig) = %q", buf.String())
	}
	if sel := types.NewMethodSet(derived).Lookup(pkg, "Read"); sel == nil || !strings.Contains(types.SelectionString(sel, types.RelativeTo(pkg)), "method (Derived) Read") {
		t.Fatalf("SelectionString = %v", sel)
	}

	for _, tc := range []struct {
		in, want types.Type
	}{
		{types.Typ[types.UntypedBool], types.Typ[types.Bool]},
		{types.Typ[types.UntypedRune], types.Typ[types.Rune]},
		{types.Typ[types.UntypedFloat], types.Typ[types.Float64]},
		{types.Typ[types.UntypedComplex], types.Typ[types.Complex128]},
		{types.Typ[types.UntypedString], types.Typ[types.String]},
	} {
		if got := types.Default(tc.in); got != tc.want {
			t.Fatalf("Default(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
	if !types.Comparable(tp) || !types.Comparable(types.NewArray(types.Typ[types.Int], 2)) || !types.Comparable(s1) {
		t.Fatal("Comparable on TypeParam/Array/Struct failed")
	}
	e1 := types.NewNamed(types.NewTypeName(0, tmpPkg, "E1", nil), types.NewStruct([]*types.Var{types.NewField(0, tmpPkg, "F", types.Typ[types.Int], false)}, nil), nil)
	e2 := types.NewNamed(types.NewTypeName(0, tmpPkg, "E2", nil), types.NewStruct([]*types.Var{types.NewField(0, tmpPkg, "F", types.Typ[types.Int], false)}, nil), nil)
	ambig := types.NewStruct([]*types.Var{
		types.NewField(0, tmpPkg, "E1", e1, true),
		types.NewField(0, tmpPkg, "E2", e2, true),
	}, nil)
	if obj, idx, _ := types.LookupFieldOrMethod(ambig, false, tmpPkg, "F"); obj != nil || idx == nil {
		t.Fatalf("ambiguous LookupFieldOrMethod = (%v, %v), want (nil, non-nil)", obj, idx)
	}
	if obj, _, _ := types.LookupFieldOrMethod(reader, false, pkg, "Read"); obj == nil || obj.Name() != "Read" {
		t.Fatalf("LookupFieldOrMethod(Reader, Read) = %v", obj)
	}
	if m, _ := types.MissingMethod(reader, rc, true); m == nil || m.Name() != "Close" {
		t.Fatalf("MissingMethod(Reader, ReadCloser, static=true) = %v", m)
	}
	if m, _ := types.MissingMethod(reader, rc, false); m != nil {
		t.Fatalf("MissingMethod(Reader, ReadCloser, static=false) = %v, want nil", m)
	}
	bidir := types.NewChan(types.SendRecv, types.Typ[types.Int])
	sendOnly := types.NewChan(types.SendOnly, types.Typ[types.Int])
	if !types.AssignableTo(bidir, sendOnly) || types.AssignableTo(sendOnly, bidir) {
		t.Fatal("directional chan AssignableTo mismatch")
	}
	anyT := types.NewInterfaceType(nil, nil).Complete()
	if !types.AssignableTo(types.Typ[types.UntypedBool], types.Typ[types.Bool]) ||
		!types.AssignableTo(types.Typ[types.UntypedInt], types.Typ[types.Float64]) ||
		!types.AssignableTo(types.Typ[types.UntypedFloat], types.Typ[types.Complex128]) ||
		!types.AssignableTo(types.Typ[types.UntypedComplex], types.Typ[types.Complex64]) ||
		!types.AssignableTo(types.Typ[types.UntypedString], types.Typ[types.String]) ||
		!types.AssignableTo(types.Typ[types.UntypedInt], anyT) {
		t.Fatal("untyped constant AssignableTo mismatch")
	}
	if !types.ConvertibleTo(types.NewPointer(s1), types.NewPointer(s2)) ||
		!types.ConvertibleTo(types.NewPointer(types.Typ[types.Int]), types.Typ[types.UnsafePointer]) ||
		!types.ConvertibleTo(types.Typ[types.UnsafePointer], types.NewPointer(types.Typ[types.Int])) ||
		!types.ConvertibleTo(types.Typ[types.UnsafePointer], types.Typ[types.Uintptr]) ||
		!types.ConvertibleTo(types.Typ[types.Uintptr], types.Typ[types.UnsafePointer]) ||
		!types.ConvertibleTo(types.Typ[types.Complex64], types.Typ[types.Complex128]) ||
		!types.ConvertibleTo(types.Typ[types.Int], types.Typ[types.String]) ||
		!types.ConvertibleTo(types.Typ[types.String], types.NewSlice(types.Typ[types.Byte])) ||
		!types.ConvertibleTo(types.NewSlice(types.Typ[types.Rune]), types.Typ[types.String]) ||
		!types.ConvertibleTo(types.NewSlice(types.Typ[types.Int]), types.NewPointer(types.NewArray(types.Typ[types.Int], 2))) {
		t.Fatal("ConvertibleTo edge cases failed")
	}
	if types.SelectionString(nil, nil) != "<nil>" || types.ExprString(nil) != "" {
		t.Fatal("nil formatting helpers failed")
	}
}

// TestPredicatesMatchStdlibGoTypes cross-checks llgo/types predicates against
// stdlib go/types across a rich matrix of Go types (generics, union constraints,
// recursive interfaces, aliases, tagged structs, directional channels, value vs
// pointer methods) AND verifies LLGo-specific CallConv/PtrKind overrides are
// preserved without being collapsed by go/types delegation.
func TestPredicatesMatchStdlibGoTypes(t *testing.T) {
	src := `package xcheck

type Signed interface {
	~int | ~int32 | ~int64
}

type Ordered interface {
	Signed | ~string
}

type MyInt int
type IntAlias = int
type TaggedA struct { X int "a" }
type TaggedB struct { X int "b" }
type NonComp struct { S []int }

type Reader interface { Read() int }
type ReadCloser interface { Reader; Close() }
type RecIface interface { Self() RecIface }

type Impl struct{ V int }
func (i Impl) Read() int { return i.V }
func (i *Impl) Close() {}
func (i Impl) Self() RecIface { return i }

type ChBidir chan int
type ChSend chan<- int
type IntSlice []int
type IntArr [2]int
type IntArrPtr *[2]int

//llgo:cdecl
type CFn func(int32) int32
type GoFn func(int32) int32
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "xcheck.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	goPkg, err := (&gotypes.Config{}).Check("example.com/xcheck", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatalf("go/types.Check: %v", err)
	}
	ctx := newContextForSpec(target.Spec{GOOS: "linux", GOARCH: "amd64"})
	llPkg, err := (&types.Config{Context: ctx}).Check("example.com/xcheck", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatalf("llgo/types.Check: %v", err)
	}

	names := []string{
		"Signed", "Ordered", "MyInt", "IntAlias", "TaggedA", "TaggedB",
		"NonComp", "Reader", "ReadCloser", "RecIface", "Impl",
		"ChBidir", "ChSend", "IntSlice", "IntArr", "IntArrPtr", "GoFn",
	}
	goTypes := make([]gotypes.Type, 0, len(names)+2)
	llTypes := make([]types.Type, 0, len(names)+2)
	for _, n := range names {
		goTypes = append(goTypes, goPkg.Scope().Lookup(n).Type())
		llTypes = append(llTypes, llPkg.Scope().Lookup(n).Type())
	}
	// Also include *Impl and builtin int.
	goTypes = append(goTypes, gotypes.NewPointer(goPkg.Scope().Lookup("Impl").Type()), gotypes.Typ[gotypes.Int])
	llTypes = append(llTypes, types.NewPointer(llPkg.Scope().Lookup("Impl").Type()), types.Typ[types.Int])

	for i := range goTypes {
		if got, want := types.Comparable(llTypes[i]), gotypes.Comparable(goTypes[i]); got != want {
			t.Errorf("Comparable(%s): llgo=%v go=%v", llTypes[i], got, want)
		}
		for j := range goTypes {
			if got, want := types.Identical(llTypes[i], llTypes[j]), gotypes.Identical(goTypes[i], goTypes[j]); got != want {
				t.Errorf("Identical(%s, %s): llgo=%v go=%v", llTypes[i], llTypes[j], got, want)
			}
			if got, want := types.IdenticalIgnoreTags(llTypes[i], llTypes[j]), gotypes.IdenticalIgnoreTags(goTypes[i], goTypes[j]); got != want {
				t.Errorf("IdenticalIgnoreTags(%s, %s): llgo=%v go=%v", llTypes[i], llTypes[j], got, want)
			}
			if got, want := types.AssignableTo(llTypes[i], llTypes[j]), gotypes.AssignableTo(goTypes[i], goTypes[j]); got != want {
				t.Errorf("AssignableTo(%s, %s): llgo=%v go=%v", llTypes[i], llTypes[j], got, want)
			}
			if got, want := types.ConvertibleTo(llTypes[i], llTypes[j]), gotypes.ConvertibleTo(goTypes[i], goTypes[j]); got != want {
				t.Errorf("ConvertibleTo(%s, %s): llgo=%v go=%v", llTypes[i], llTypes[j], got, want)
			}
			if gIface, ok := goTypes[j].Underlying().(*gotypes.Interface); ok {
				lIface := llTypes[j].Underlying().(*types.Interface)
				if got, want := types.Implements(llTypes[i], lIface), gotypes.Implements(goTypes[i], gIface); got != want {
					t.Errorf("Implements(%s, %s): llgo=%v go=%v", llTypes[i], llTypes[j], got, want)
				}
				if got, want := types.Satisfies(llTypes[i], lIface), gotypes.Satisfies(goTypes[i], gIface); got != want {
					t.Errorf("Satisfies(%s, %s): llgo=%v go=%v", llTypes[i], llTypes[j], got, want)
				}
				if gIface.IsMethodSet() {
					if got, want := types.AssertableTo(lIface, llTypes[i]), gotypes.AssertableTo(gIface, goTypes[i]); got != want {
						t.Errorf("AssertableTo(%s, %s): llgo=%v go=%v", llTypes[j], llTypes[i], got, want)
					}
					lm, lwrong := types.MissingMethod(llTypes[i], lIface, true)
					gm, gwrong := gotypes.MissingMethod(goTypes[i], gIface, true)
					if (lm == nil) != (gm == nil) || lwrong != gwrong || (lm != nil && lm.Name() != gm.Name()) {
						t.Errorf("MissingMethod(%s, %s): llgo=(%v,%v) go=(%v,%v)", llTypes[i], llTypes[j], lm, lwrong, gm, gwrong)
					}
				}
			}
		}
	}

	// Verify LLGo extensions (//llgo:cdecl CFn vs GoFn, CPtr vs GoPtr) are NOT collapsed by go/types delegation!
	cFn := llPkg.Scope().Lookup("CFn").Type()
	goFn := llPkg.Scope().Lookup("GoFn").Type()
	if types.Identical(cFn.Underlying(), goFn.Underlying()) {
		t.Fatal("CDecl signature should NOT be Identical to Go signature")
	}
	if types.AssignableTo(cFn.Underlying(), goFn.Underlying()) {
		t.Fatal("CDecl signature should NOT be AssignableTo Go signature")
	}
	goPtr := types.NewPointerWithKind(types.Typ[types.Int], types.GoPtr)
	cPtr := types.NewPointerWithKind(types.Typ[types.Int], types.CPtr)
	if types.Identical(goPtr, cPtr) || types.AssignableTo(goPtr, cPtr) {
		t.Fatal("GoPtr and CPtr should NOT be Identical or AssignableTo each other")
	}

	// Nil *Tuple (empty parameter/result list) must not panic in Identical or predicates.
	emptyTuple := types.NewTuple() // (*types.Tuple)(nil)
	oneInt := types.NewTuple(types.NewParam(0, nil, "", types.Typ[types.Int64]))
	sigNoParams := types.NewSignatureType(nil, nil, nil, emptyTuple, oneInt, false)
	sigOneParam := types.NewSignatureType(nil, nil, nil, oneInt, oneInt, false)
	if types.Identical(emptyTuple, oneInt) || types.Identical(oneInt, emptyTuple) {
		t.Fatal("nil *Tuple should not be Identical to non-empty *Tuple")
	}
	if !types.Identical(emptyTuple, emptyTuple) {
		t.Fatal("nil *Tuple should be Identical to itself")
	}
	if types.Identical(sigNoParams, sigOneParam) || types.Identical(sigOneParam, sigNoParams) {
		t.Fatal("signatures with nil vs non-nil params should not be Identical")
	}

	// go/types parity for UnsafePointer Name/String, empty Interface, embedded Struct fields with qf,
	// deterministic Scope child ordering, and generic TypeName / Signature Instantiate context identity.
	if got := types.Typ[types.UnsafePointer].Name(); got != "Pointer" {
		t.Fatalf("UnsafePointer.Name() = %q, want %q", got, "Pointer")
	}
	if got := types.Typ[types.UnsafePointer].String(); got != "unsafe.Pointer" {
		t.Fatalf("UnsafePointer.String() = %q, want %q", got, "unsafe.Pointer")
	}
	if got := types.NewInterfaceType(nil, nil).Complete().String(); got != "interface{}" {
		t.Fatalf("empty Interface.String() = %q, want %q", got, "interface{}")
	}
	embedSrc := `package main
type T struct{}
type W struct{ m int; *T }
type Set[E comparable] map[E]struct{}
func New[E comparable]() Set[E] { return nil }
func (s Set[E]) Insert(E) Set[E] { return s }
func (s Set[E]) Diff() Set[E] { r := New[E](); return r.Insert(*new(E)) }
func Scope1() int { type L int; return int(L(1)) }
func Scope2() int { type L int; return int(L(2)) }
type Future[A any] interface{ Then(func(A)) }
func Async[A any]() Future[A] { return nil }
func UseAsync() Future[T] { return Async[T]() }
`
	fset2 := token.NewFileSet()
	f2, err := parser.ParseFile(fset2, "parity.go", embedSrc, 0)
	if err != nil {
		t.Fatal(err)
	}
	info2 := &types.Info{
		Types:     make(map[ast.Expr]types.TypeAndValue),
		Defs:      make(map[*ast.Ident]types.Object),
		Uses:      make(map[*ast.Ident]types.Object),
		Instances: make(map[*ast.Ident]types.Instance),
		Scopes:    make(map[ast.Node]*types.Scope),
	}
	pkg2, err := (&types.Config{}).Check("example.com/parity", fset2, []*ast.File{f2}, info2)
	if err != nil {
		t.Fatal(err)
	}
	wUnder := pkg2.Scope().Lookup("W").Type().Underlying()
	qfMain := func(p *types.Package) string { return p.Name() }
	if got := types.TypeString(wUnder, qfMain); got != "struct{m int; *main.T}" {
		t.Fatalf("TypeString(W.Underlying, qf) = %q, want %q", got, "struct{m int; *main.T}")
	}
	// Generic TypeName must retain origin *Named (not an instantiation Set[E]).
	setNamed := pkg2.Scope().Lookup("Set").Type().(*types.Named)
	if setNamed.TypeArgs().Len() != 0 || setNamed.Origin() != setNamed {
		t.Fatalf("Set TypeName.Type() hijacked by instantiation: args=%v origin=%v", setNamed.TypeArgs(), setNamed.Origin())
	}
	setAny, err := types.Instantiate(nil, setNamed, []types.Type{types.Universe.Lookup("any").Type()}, true)
	if err != nil || types.NewMethodSet(setAny).Lookup(nil, "Insert") == nil {
		t.Fatalf("Instantiate(Set[any]) missing Insert method: %v err=%v", setAny, err)
	}
	// Instantiating a generic function signature with a separate Context must preserve package/object identity.
	tNamed := pkg2.Scope().Lookup("T").Type().(*types.Named)
	asyncSig := pkg2.Scope().Lookup("Async").(*types.Func).Signature()
	instSig, err := types.Instantiate(types.NewContext(), asyncSig, []types.Type{tNamed}, false)
	if err != nil {
		t.Fatal(err)
	}
	instRet := instSig.(*types.Signature).Results().At(0).Type().(*types.Named)
	wantRet := pkg2.Scope().Lookup("UseAsync").(*types.Func).Signature().Results().At(0).Type().(*types.Named)
	if instRet.Obj() != wantRet.Obj() || instRet.Obj().Pkg() != pkg2 || !types.Identical(instRet, wantRet) {
		t.Fatalf("Instantiate(Signature) lost object/package identity: inst=%p/%p want=%p/%p identical=%v",
			instRet.Obj(), instRet.Obj().Pkg(), wantRet.Obj(), wantRet.Obj().Pkg(), types.Identical(instRet, wantRet))
	}
	// Scope child order must match go/types deterministically (Scope1 before Scope2 inside file scope).
	s1Scope := pkg2.Scope().Lookup("Scope1").(*types.Func).Scope()
	s2Scope := pkg2.Scope().Lookup("Scope2").(*types.Func).Scope()
	fileScope := pkg2.Scope().Child(0)
	if s1Scope.Parent() != fileScope || s2Scope.Parent() != fileScope {
		t.Fatalf("unexpected parent scope for Scope1/Scope2: %p / %p vs fileScope %p", s1Scope.Parent(), s2Scope.Parent(), fileScope)
	}
	idx1, idx2 := -1, -1
	for i := 0; i < fileScope.NumChildren(); i++ {
		switch fileScope.Child(i) {
		case s1Scope:
			idx1 = i
		case s2Scope:
			idx2 = i
		}
	}
	if idx1 < 0 || idx2 < 0 || idx1 >= idx2 {
		t.Fatalf("non-deterministic Scope child ordering: Scope1=%d Scope2=%d", idx1, idx2)
	}
}

func TestConstructorsPreserveBoundTypeName(t *testing.T) {
	pkg := types.NewPackage("example.com/p", "p")
	tnNamed := types.NewTypeName(0, pkg, "N", nil)
	origNamed := types.NewNamed(tnNamed, types.Typ[types.String], nil)
	altNamed := types.NewNamed(tnNamed, types.Typ[types.Int], nil)
	if tnNamed.Type() != origNamed || altNamed.Obj() != tnNamed {
		t.Fatalf("NewNamed overwrote bound TypeName: got %v, want %v", tnNamed.Type(), origNamed)
	}

	tnAlias := types.NewTypeName(0, pkg, "A", nil)
	origAlias := types.NewAlias(tnAlias, types.Typ[types.String])
	altAlias := types.NewAlias(tnAlias, types.Typ[types.Int])
	if tnAlias.Type() != origAlias || altAlias.Obj() != tnAlias {
		t.Fatalf("NewAlias overwrote bound TypeName: got %v, want %v", tnAlias.Type(), origAlias)
	}

	tnParam := types.NewTypeName(0, pkg, "T", nil)
	origParam := types.NewTypeParam(tnParam, types.Universe.Lookup("any").Type())
	altParam := types.NewTypeParam(tnParam, types.Universe.Lookup("comparable").Type())
	if tnParam.Type() != origParam || altParam.Obj() != tnParam {
		t.Fatalf("NewTypeParam overwrote bound TypeName: got %v, want %v", tnParam.Type(), origParam)
	}
}

func TestSyncAtomicAlign64On386(t *testing.T) {
	for _, pkgPath := range []string{"sync/atomic", "internal/runtime/atomic"} {
		t.Run(pkgPath, func(t *testing.T) {
			pkg := types.NewPackage(pkgPath, "atomic")
			align64 := types.NewNamed(types.NewTypeName(0, pkg, "align64", nil), types.NewStruct(nil, nil), nil)
			noCopy := types.NewNamed(types.NewTypeName(0, pkg, "noCopy", nil), types.NewStruct(nil, nil), nil)
			u64Struct := types.NewStruct([]*types.Var{
				types.NewField(0, pkg, "noCopy", noCopy, false),
				types.NewField(0, pkg, "_", align64, false),
				types.NewField(0, pkg, "v", types.Typ[types.Uint64], false),
			}, nil)
			u64 := types.NewNamed(types.NewTypeName(0, pkg, "Uint64", nil), u64Struct, nil)
			bucket := types.NewStruct([]*types.Var{
				types.NewField(0, pkg, "_", types.NewArray(u64, 0), false),
				types.NewField(0, pkg, "head", types.Typ[types.Uint32], false),
				types.NewField(0, pkg, "count", u64, false),
			}, nil)

			sizesList := []struct {
				name  string
				sizes types.Sizes
			}{
				{"SizesFor(gc,386)", types.SizesFor("gc", "386")},
				{"SizesFor(llgo,386)", types.SizesFor("llgo", "386")},
				{"Context(windows/386)", newContextForSpec(target.Spec{GOOS: "windows", GOARCH: "386"})},
				{"Context(linux/386)", newContextForSpec(target.Spec{GOOS: "linux", GOARCH: "386"})},
				{"StdSizes(4,4)", &types.StdSizes{WordSize: 4, MaxAlign: 4}},
			}
			for _, tc := range sizesList {
				if got := tc.sizes.Alignof(align64); got != 8 {
					t.Errorf("%s Alignof(align64) = %d, want 8", tc.name, got)
				}
				if got := tc.sizes.Sizeof(align64); got != 0 {
					t.Errorf("%s Sizeof(align64) = %d, want 0", tc.name, got)
				}
				if got := tc.sizes.Alignof(u64); got != 8 {
					t.Errorf("%s Alignof(Uint64) = %d, want 8", tc.name, got)
				}
				if got := tc.sizes.Sizeof(u64); got != 8 {
					t.Errorf("%s Sizeof(Uint64) = %d, want 8", tc.name, got)
				}
				if got := tc.sizes.Alignof(bucket); got != 8 {
					t.Errorf("%s Alignof(bucket) = %d, want 8", tc.name, got)
				}
				offs := tc.sizes.Offsetsof([]*types.Var{
					bucket.Field(0),
					bucket.Field(1),
					bucket.Field(2),
				})
				if len(offs) != 3 || offs[0] != 0 || offs[1] != 0 || offs[2] != 8 {
					t.Errorf("%s Offsetsof(bucket) = %v, want [0 0 8]", tc.name, offs)
				}
			}
		})
	}
}

func TestSizesForIsStateless(t *testing.T) {
	s1 := types.SizesFor("gc", "amd64")
	s2 := types.SizesFor("gc", "amd64")
	if s1 == nil || s1 != s2 {
		t.Fatalf("SizesFor(gc, amd64) must preserve pointer identity: %p vs %p", s1, s2)
	}
	if _, isCtx := s1.(*types.Context); isCtx {
		t.Fatal("SizesFor must not return a globally retained *types.Context")
	}
	pkg := types.NewPackage("example.com/stateless", "stateless")
	integer := types.NewNamed(types.NewTypeName(0, pkg, "Integer", nil), types.Typ[types.Int64], nil)
	record := types.NewNamed(types.NewTypeName(0, pkg, "Record", nil), types.NewStruct([]*types.Var{
		types.NewField(0, pkg, "A", types.Typ[types.Byte], false),
		types.NewField(0, pkg, "X", integer, false),
	}, nil), nil)
	nested := types.NewNamed(types.NewTypeName(0, pkg, "Nested", nil), types.NewArray(record, 8), nil)
	for _, typ := range []types.Type{integer, record, nested} {
		if allocs := testing.AllocsPerRun(100, func() { _ = s1.Alignof(typ) }); allocs != 0 {
			t.Fatalf("SizesFor.Alignof(%s) allocs = %g, want 0", typ, allocs)
		}
		if allocs := testing.AllocsPerRun(100, func() { _ = s1.Sizeof(typ) }); allocs != 0 {
			t.Fatalf("SizesFor.Sizeof(%s) allocs = %g, want 0", typ, allocs)
		}
	}
}

func TestEvalAndCheckExprOnPubliclyConstructedPackage(t *testing.T) {
	fset := token.NewFileSet()
	pkg := types.NewPackage("example.com/manual", "manual")
	kConst := types.NewConst(token.NoPos, pkg, "K", types.Typ[types.UntypedInt], constant.MakeInt64(42))
	pkg.Scope().Insert(kConst)
	xVar := types.NewVar(token.NoPos, pkg, "X", types.Typ[types.Int])
	pkg.Scope().Insert(xVar)
	tObj := types.NewTypeName(token.NoPos, pkg, "MyInt", nil)
	tNamed := types.NewNamed(tObj, types.Typ[types.Int], nil)
	pkg.Scope().Insert(tObj)

	tv, err := types.Eval(fset, pkg, token.NoPos, "K + 8")
	if err != nil {
		t.Fatalf("Eval(K + 8) failed: %v", err)
	}
	if got, ok := constant.Int64Val(tv.Value); !ok || got != 50 {
		t.Fatalf("Eval(K + 8) value = %v, want 50", tv.Value)
	}

	expr, err := parser.ParseExpr("MyInt(X + K)")
	if err != nil {
		t.Fatalf("ParseExpr: %v", err)
	}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	if err := types.CheckExpr(fset, pkg, token.NoPos, expr, info); err != nil {
		t.Fatalf("CheckExpr(MyInt(X + K)) failed: %v", err)
	}
	if got := info.Types[expr].Type; got != tNamed {
		t.Fatalf("CheckExpr result type = %v (%p), want MyInt (%p)", got, got, tNamed)
	}
	foundK, foundX, foundT := false, false, false
	for _, obj := range info.Uses {
		switch obj {
		case kConst:
			foundK = true
		case xVar:
			foundX = true
		case tObj:
			foundT = true
		}
	}
	if !foundK || !foundX || !foundT {
		t.Fatalf("CheckExpr info.Uses did not preserve object identities: K=%v X=%v T=%v uses=%v", foundK, foundX, foundT, info.Uses)
	}

	// Verify subsequent scope insertions remain visible to Eval.
	pkg.Scope().Insert(types.NewConst(token.NoPos, pkg, "K2", types.Typ[types.UntypedInt], constant.MakeInt64(7)))
	tv2, err := types.Eval(fset, pkg, token.NoPos, "K + K2")
	if err != nil {
		t.Fatalf("Eval(K + K2) failed: %v", err)
	}
	if got, ok := constant.Int64Val(tv2.Value); !ok || got != 49 {
		t.Fatalf("Eval(K + K2) value = %v, want 49", tv2.Value)
	}

	// Also verify post-Check scope mutations on a Config.Check-produced package.
	checkedFile, err := parser.ParseFile(fset, "checked.go", "package checked\nconst Base = 10\n", 0)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	var conf types.Config
	checkedPkg, err := conf.Check("example.com/checked", fset, []*ast.File{checkedFile}, nil)
	if err != nil {
		t.Fatalf("Config.Check: %v", err)
	}
	extraTN := types.NewTypeName(token.NoPos, checkedPkg, "ExtraInt", nil)
	_ = types.NewNamed(extraTN, types.Typ[types.Int], nil)
	checkedPkg.Scope().Insert(extraTN)
	checkedPkg.Scope().Insert(types.NewConst(token.NoPos, checkedPkg, "Extra", types.Typ[types.UntypedInt], constant.MakeInt64(5)))
	tv3, err := types.Eval(fset, checkedPkg, token.NoPos, "ExtraInt(Base + Extra)")
	if err != nil {
		t.Fatalf("Eval on mutated checked package failed: %v", err)
	}
	if got, ok := constant.Int64Val(tv3.Value); !ok || got != 15 {
		t.Fatalf("Eval(ExtraInt(Base + Extra)) = %v, want 15", tv3.Value)
	}
	// Nil package should evaluate universe expressions cleanly.
	if tvNil, err := types.Eval(fset, nil, token.NoPos, "1 + 2"); err != nil {
		t.Fatalf("Eval with nil pkg failed: %v", err)
	} else if got, ok := constant.Int64Val(tvNil.Value); !ok || got != 3 {
		t.Fatalf("Eval(1 + 2) with nil pkg = %v, want 3", tvNil.Value)
	}
}

func TestAliasedFuncCallConvPreservesPlainGoFuncLayout(t *testing.T) {
	src := `package aliasabi

import "unsafe"

type GoFn = func(x int32)

//llgo:cdecl
type CFn = GoFn

type CFn2 = CFn

//llgo:stdcall
type StdFn = CFn

const (
	SizeGoFn  = unsafe.Sizeof(GoFn(nil))
	SizeCFn   = unsafe.Sizeof(CFn(nil))
	SizeCFn2  = unsafe.Sizeof(CFn2(nil))
	SizeStdFn = unsafe.Sizeof(StdFn(nil))
)

var (
	g GoFn
	c CFn = g
)
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "aliasabi.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	ctx := types.NewContextWithConfig(types.ContextConfig{
		Spec: target.ResolveSpec(target.Spec{GOOS: "linux", GOARCH: "amd64"}),
	})
	info := &types.Info{
		Defs: make(map[*ast.Ident]types.Object),
		Uses: make(map[*ast.Ident]types.Object),
	}
	conf := types.Config{
		Context:  ctx,
		Importer: &mapImporter{pkgs: map[string]*types.Package{"unsafe": types.Unsafe}},
	}
	pkg, err := conf.Check("example.com/aliasabi", fset, []*ast.File{f}, info)
	if err != nil {
		t.Fatalf("Config.Check: %v", err)
	}
	wantSizes := map[string]int64{
		"SizeGoFn":  16,
		"SizeCFn":   8,
		"SizeCFn2":  8,
		"SizeStdFn": 8,
	}
	for name, want := range wantSizes {
		cObj, ok := pkg.Scope().Lookup(name).(*types.Const)
		if !ok {
			t.Fatalf("missing const %s", name)
		}
		got, ok := constant.Int64Val(cObj.Val())
		if !ok || got != want {
			t.Fatalf("%s = %v, want %d", name, cObj.Val(), want)
		}
	}
	goFnSig := types.Unalias(pkg.Scope().Lookup("GoFn").Type()).(*types.Signature)
	cFnSig := types.Unalias(pkg.Scope().Lookup("CFn").Type()).(*types.Signature)
	stdFnSig := types.Unalias(pkg.Scope().Lookup("StdFn").Type()).(*types.Signature)
	if goFnSig.CallConv() != types.Go {
		t.Fatalf("GoFn CallConv = %v, want Go", goFnSig.CallConv())
	}
	if cFnSig.CallConv() != types.CDecl {
		t.Fatalf("CFn CallConv = %v, want CDecl", cFnSig.CallConv())
	}
	if stdFnSig.CallConv() != types.Stdcall {
		t.Fatalf("StdFn CallConv = %v, want Stdcall", stdFnSig.CallConv())
	}
}

func TestCrossFileAliasedFuncCallConvPreservesFileImports(t *testing.T) {
	fset := token.NewFileSet()
	// Also test conflicting import aliases across files: a.go binds wire to
	// unsafe (and dep1), while b.go binds wire to a different package.
	dep1 := types.NewPackage("example.com/dep1", "dep1")
	dep1T := types.NewNamed(types.NewTypeName(token.NoPos, dep1, "T", nil), types.Typ[types.Int32], nil)
	dep1.Scope().Insert(dep1T.Obj())
	dep1.MarkComplete()

	dep2 := types.NewPackage("example.com/dep2", "dep2")
	dep2T := types.NewNamed(types.NewTypeName(token.NoPos, dep2, "T", nil), types.Typ[types.String], nil)
	dep2.Scope().Insert(dep2T.Obj())
	dep2.MarkComplete()

	srcA := `package xfile

import (
	wire "unsafe"
	dep "example.com/dep1"
)

type GoFn = func(wire.Pointer, dep.T)

func fA(n int) int {
	return n
}
`
	srcB := `package xfile

import (
	"unsafe"
	dep "example.com/dep2"
)

//llgo:cdecl
type CFn = GoFn

const (
	SizeGoFn = unsafe.Sizeof(GoFn(nil))
	SizeCFn  = unsafe.Sizeof(CFn(nil))
)

var (
	_ dep.T
	g GoFn
	c CFn = g
)

func fB(m int) int {
	return m
}
`
	fa, err := parser.ParseFile(fset, "a.go", srcA, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile a.go: %v", err)
	}
	fb, err := parser.ParseFile(fset, "b.go", srcB, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile b.go: %v", err)
	}

	ctx := types.NewContextWithConfig(types.ContextConfig{
		Spec: target.ResolveSpec(target.Spec{GOOS: "linux", GOARCH: "amd64"}),
	})
	info := &types.Info{
		Defs:   make(map[*ast.Ident]types.Object),
		Uses:   make(map[*ast.Ident]types.Object),
		Scopes: make(map[ast.Node]*types.Scope),
	}
	conf := types.Config{
		Context: ctx,
		Importer: &mapImporter{pkgs: map[string]*types.Package{
			"unsafe":           types.Unsafe,
			"example.com/dep1": dep1,
			"example.com/dep2": dep2,
		}},
	}
	pkg, err := conf.Check("example.com/xfile", fset, []*ast.File{fa, fb}, info)
	if err != nil {
		t.Fatalf("Config.Check cross-file alias failed: %v", err)
	}
	for _, name := range pkg.Scope().Names() {
		if strings.HasPrefix(name, "__llgo_ccalias_") || strings.HasPrefix(name, "$llgo_ccalias_") {
			t.Fatalf("synthetic alias %q leaked into package scope", name)
		}
	}
	// Synthetic aliases must never be visible to Eval or CheckExpr, while real
	// declarations remain evaluable.
	if _, err := types.Eval(fset, pkg, token.NoPos, "__llgo_ccalias_0"); err == nil {
		t.Fatal("Eval unexpectedly resolved undeclared __llgo_ccalias_0")
	}
	if _, err := types.Eval(fset, pkg, token.NoPos, "$llgo_ccalias_0"); err == nil {
		t.Fatal("Eval unexpectedly accepted synthetic $llgo_ccalias_0")
	}
	synthID := ast.NewIdent("$llgo_ccalias_0")
	synthInfo := &types.Info{
		Uses:  make(map[*ast.Ident]types.Object),
		Types: make(map[ast.Expr]types.TypeAndValue),
	}
	if err := types.CheckExpr(fset, pkg, token.NoPos, synthID, synthInfo); err == nil {
		if synthInfo.Uses[synthID] != nil || (synthInfo.Types[synthID].Type != nil && synthInfo.Types[synthID].Type != types.Typ[types.Invalid]) {
			t.Fatalf("CheckExpr unexpectedly resolved synthetic $llgo_ccalias_0: use=%v tv=%v", synthInfo.Uses[synthID], synthInfo.Types[synthID])
		}
	}
	if tv, err := types.Eval(fset, pkg, token.NoPos, "SizeCFn"); err != nil {
		t.Fatalf("Eval(SizeCFn) failed: %v", err)
	} else if got, ok := constant.Int64Val(tv.Value); !ok || got != 8 {
		t.Fatalf("Eval(SizeCFn) = %v, want 8", tv.Value)
	}
	// Lexical scopes in both files must remain intact for Eval and CheckExpr
	// with real source positions (non-token.NoPos), including after post-Check inserts.
	pkg.Scope().Insert(types.NewConst(token.NoPos, pkg, "PostExtra", types.Typ[types.Int], constant.MakeInt64(3)))
	for _, tc := range []struct {
		file  *ast.File
		param string
	}{
		{file: fa, param: "n"},
		{file: fb, param: "m"},
	} {
		var retPos token.Pos
		ast.Inspect(tc.file, func(n ast.Node) bool {
			if rs, ok := n.(*ast.ReturnStmt); ok {
				retPos = rs.Pos()
				return false
			}
			return true
		})
		if !retPos.IsValid() {
			t.Fatalf("failed to find return statement position for param %s", tc.param)
		}
		tv, err := types.Eval(fset, pkg, retPos, tc.param+" + PostExtra")
		if err != nil {
			t.Fatalf("Eval(%s + PostExtra) at return pos failed: %v", tc.param, err)
		}
		if tv.Type != types.Typ[types.Int] {
			t.Fatalf("Eval(%s + PostExtra) type = %v, want int", tc.param, tv.Type)
		}
		expr, err := parser.ParseExpr(tc.param + " + PostExtra")
		if err != nil {
			t.Fatalf("ParseExpr: %v", err)
		}
		exprInfo := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue)}
		if err := types.CheckExpr(fset, pkg, retPos, expr, exprInfo); err != nil {
			t.Fatalf("CheckExpr(%s + PostExtra) at return pos failed: %v", tc.param, err)
		}
		if got := exprInfo.Types[expr].Type; got != types.Typ[types.Int] {
			t.Fatalf("CheckExpr(%s + PostExtra) type = %v, want int", tc.param, got)
		}
	}
	// Source referencing an undeclared __llgo_ccalias_0 must still fail Check.
	badB, err := parser.ParseFile(fset, "bad_b.go", "package xfile\n//llgo:cdecl\ntype CFn = GoFn\nvar _ __llgo_ccalias_0\n", parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile bad_b.go: %v", err)
	}
	badConf := types.Config{
		Importer: &mapImporter{pkgs: map[string]*types.Package{
			"unsafe":           types.Unsafe,
			"example.com/dep1": dep1,
		}},
	}
	if _, err := badConf.Check("example.com/xfilebad", fset, []*ast.File{fa, badB}, nil); err == nil {
		t.Fatal("Config.Check unexpectedly accepted undeclared __llgo_ccalias_0")
	}
	for name, want := range map[string]int64{"SizeGoFn": 16, "SizeCFn": 8} {
		cObj, ok := pkg.Scope().Lookup(name).(*types.Const)
		if !ok {
			t.Fatalf("missing const %s", name)
		}
		if got, ok := constant.Int64Val(cObj.Val()); !ok || got != want {
			t.Fatalf("%s = %v, want %d", name, cObj.Val(), want)
		}
	}
	cFnSig := types.Unalias(pkg.Scope().Lookup("CFn").Type()).(*types.Signature)
	if cFnSig.CallConv() != types.CDecl {
		t.Fatalf("CFn CallConv = %v, want CDecl", cFnSig.CallConv())
	}
	if param1 := cFnSig.Params().At(1).Type(); param1 != dep1T {
		t.Fatalf("CFn param 1 = %v, want dep1.T (%v)", param1, dep1T)
	}
}

func TestIncrementalCheckerFilesErasedAliases(t *testing.T) {
	assertNoSynthInScope := func(t *testing.T, pkg *types.Package) {
		t.Helper()
		for _, name := range pkg.Scope().Names() {
			if strings.HasPrefix(name, "$llgo_ccalias_") || strings.HasPrefix(name, "__llgo_ccalias_") {
				t.Fatalf("synthetic alias %q leaked into package scope: %v", name, pkg.Scope().Names())
			}
		}
	}
	parse := func(t *testing.T, fset *token.FileSet, name, src string) *ast.File {
		t.Helper()
		f, err := parser.ParseFile(fset, name, src, parser.ParseComments)
		if err != nil {
			t.Fatalf("ParseFile %s: %v", name, err)
		}
		return f
	}
	newChecker := func(fset *token.FileSet, pkgPath string, imports map[string]*types.Package) (*types.Checker, *types.Package) {
		ctx := types.NewContextWithConfig(types.ContextConfig{
			Spec: target.ResolveSpec(target.Spec{GOOS: "linux", GOARCH: "amd64"}),
		})
		pkg := types.NewPackage(pkgPath, "inc")
		conf := &types.Config{
			Context:  ctx,
			Importer: &mapImporter{pkgs: imports},
		}
		info := &types.Info{
			Defs:   make(map[*ast.Ident]types.Object),
			Uses:   make(map[*ast.Ident]types.Object),
			Scopes: make(map[ast.Node]*types.Scope),
		}
		return types.NewChecker(conf, fset, pkg, info), pkg
	}

	// Case 1: Batch 1 generates a cross-file synthetic alias; Batch 2 adds only
	// `var New int`. The synthetic alias must stay excluded from pkg.Scope().
	t.Run("Batch2VarDoesNotResurfaceSynth", func(t *testing.T) {
		fset := token.NewFileSet()
		chk, pkg := newChecker(fset, "example.com/inc1", map[string]*types.Package{"unsafe": types.Unsafe})
		fa := parse(t, fset, "a.go", "package inc\nimport wire \"unsafe\"\ntype GoFn = func(wire.Pointer)\nfunc f(n int) int { return n }\n")
		fb := parse(t, fset, "b.go", "package inc\n//llgo:cdecl\ntype CFn = GoFn\n")
		fc := parse(t, fset, "c.go", "package inc\nvar New int\n")

		if err := chk.Files([]*ast.File{fa, fb}); err != nil {
			t.Fatalf("batch 1 Files: %v", err)
		}
		assertNoSynthInScope(t, pkg)
		if err := chk.Files([]*ast.File{fc}); err != nil {
			t.Fatalf("batch 2 Files: %v", err)
		}
		assertNoSynthInScope(t, pkg)
		if pkg.Scope().Lookup("New") == nil {
			t.Fatal("batch 2 var New missing from scope")
		}
		var retPos token.Pos
		ast.Inspect(fa, func(n ast.Node) bool {
			if rs, ok := n.(*ast.ReturnStmt); ok {
				retPos = rs.Pos()
				return false
			}
			return true
		})
		if tv, err := types.Eval(fset, pkg, retPos, "n + New"); err != nil || tv.Type != types.Typ[types.Int] {
			t.Fatalf("Eval(n + New) at return pos = %v, %v; want int", tv, err)
		}
	})

	// Case 2: Two separate batches each declare a distinct cross-file alias pair;
	// synthetic names must not collide across batches.
	t.Run("TwoBatchesDistinctCrossFileAliases", func(t *testing.T) {
		fset := token.NewFileSet()
		chk, pkg := newChecker(fset, "example.com/inc2", map[string]*types.Package{"unsafe": types.Unsafe})
		a1 := parse(t, fset, "a1.go", "package inc\ntype GoFn1 = func(int32)\n")
		b1 := parse(t, fset, "b1.go", "package inc\nimport \"unsafe\"\n//llgo:cdecl\ntype CFn1 = GoFn1\nconst S1 = unsafe.Sizeof(CFn1(nil))\n")
		a2 := parse(t, fset, "a2.go", "package inc\ntype GoFn2 = func(int64)\n")
		b2 := parse(t, fset, "b2.go", "package inc\nimport \"unsafe\"\n//llgo:cdecl\ntype CFn2 = GoFn2\nconst S2 = unsafe.Sizeof(CFn2(nil))\n")

		if err := chk.Files([]*ast.File{a1, b1}); err != nil {
			t.Fatalf("batch 1 Files: %v", err)
		}
		if err := chk.Files([]*ast.File{a2, b2}); err != nil {
			t.Fatalf("batch 2 Files: %v", err)
		}
		assertNoSynthInScope(t, pkg)
		for _, tc := range []struct {
			name string
			cc   types.CallConv
		}{
			{"GoFn1", types.Go},
			{"CFn1", types.CDecl},
			{"GoFn2", types.Go},
			{"CFn2", types.CDecl},
		} {
			sig := types.Unalias(pkg.Scope().Lookup(tc.name).Type()).(*types.Signature)
			if sig.CallConv() != tc.cc {
				t.Fatalf("%s CallConv = %v, want %v", tc.name, sig.CallConv(), tc.cc)
			}
		}
		for _, name := range []string{"S1", "S2"} {
			got, ok := constant.Int64Val(pkg.Scope().Lookup(name).(*types.Const).Val())
			if !ok || got != 8 {
				t.Fatalf("%s = %v, want 8", name, got)
			}
		}
	})

	// Case 3: Batch 1 declares GoFn using directory-sensitive ImporterFrom AND a
	// dot-imported package exporting Value (while GoFn has struct{ Value int });
	// Batch 2 declares //llgo:cdecl CFn = GoFn and verifies zero synthetic Info nodes.
	t.Run("Batch1GoFnBatch2CFn", func(t *testing.T) {
		fset := token.NewFileSet()
		relDep := types.NewPackage("example.com/reldep", "reldep")
		relT := types.NewNamed(types.NewTypeName(token.NoPos, relDep, "T", nil), types.Typ[types.Int32], nil)
		relDep.Scope().Insert(relT.Obj())
		relDep.MarkComplete()

		dotDep := types.NewPackage("example.com/dotdep", "dotdep")
		dotDep.Scope().Insert(types.NewConst(token.NoPos, dotDep, "Value", types.Typ[types.Int], constant.MakeInt64(7)))
		dotDep.MarkComplete()

		strictImp := &dirStrictImporter{
			wantDir: "/src/p",
			pkgs: map[string]*types.Package{
				"unsafe":             types.Unsafe,
				"example.com/reldep": relDep,
				"example.com/dotdep": dotDep,
			},
		}
		ctx := types.NewContextWithConfig(types.ContextConfig{
			Spec: target.ResolveSpec(target.Spec{GOOS: "linux", GOARCH: "amd64"}),
		})
		pkg := types.NewPackage("example.com/inc3", "inc")
		info := &types.Info{
			Types:        make(map[ast.Expr]types.TypeAndValue),
			Defs:         make(map[*ast.Ident]types.Object),
			Uses:         make(map[*ast.Ident]types.Object),
			Implicits:    make(map[ast.Node]types.Object),
			Selections:   make(map[*ast.SelectorExpr]*types.Selection),
			Scopes:       make(map[ast.Node]*types.Scope),
			Instances:    make(map[*ast.Ident]types.Instance),
			FileVersions: make(map[*ast.File]string),
		}
		chk := types.NewChecker(&types.Config{
			Context:   ctx,
			GoVersion: "go1.20",
			Importer:  strictImp,
		}, fset, pkg, info)

		fa := parse(t, fset, "/src/p/a.go", `//go:build go1.22

package inc
import (
	wire "example.com/reldep"
	. "example.com/dotdep"
)
type GoFn = func(wire.T, struct{ Value int })
var UsedDot = Value
func helper(k int) int { return k }
`)
		fb := parse(t, fset, "/src/p/b.go", `//go:build go1.22

package inc
import "unsafe"
//llgo:cdecl
type CFn = GoFn
const (
	SizeGoFn = unsafe.Sizeof(GoFn(nil))
	SizeCFn  = unsafe.Sizeof(CFn(nil))
)
`)
		if err := chk.Files([]*ast.File{fa}); err != nil {
			t.Fatalf("batch 1 Files: %v", err)
		}
		if err := chk.Files([]*ast.File{fb}); err != nil {
			t.Fatalf("batch 2 Files: %v", err)
		}
		assertNoSynthInScope(t, pkg)
		for name, want := range map[string]int64{"SizeGoFn": 16, "SizeCFn": 8} {
			got, ok := constant.Int64Val(pkg.Scope().Lookup(name).(*types.Const).Val())
			if !ok || got != want {
				t.Fatalf("%s = %v, want %d", name, got, want)
			}
		}
		for name, wantCC := range map[string]types.CallConv{"GoFn": types.Go, "CFn": types.CDecl} {
			sig := types.Unalias(pkg.Scope().Lookup(name).Type()).(*types.Signature)
			if sig.CallConv() != wantCC {
				t.Fatalf("%s CallConv = %v, want %v", name, sig.CallConv(), wantCC)
			}
		}
		// Every node recorded in Info must come from the caller's original ASTs.
		origNodes := make(map[ast.Node]bool)
		for _, rf := range []*ast.File{fa, fb} {
			ast.Inspect(rf, func(n ast.Node) bool {
				if n != nil {
					origNodes[n] = true
				}
				return true
			})
		}
		for id := range info.Defs {
			if !origNodes[id] {
				t.Fatalf("synthetic ident %q leaked into Info.Defs", id.Name)
			}
		}
		for id := range info.Uses {
			if !origNodes[id] {
				t.Fatalf("synthetic ident %q leaked into Info.Uses", id.Name)
			}
		}
		for expr := range info.Types {
			if !origNodes[expr] {
				t.Fatalf("synthetic expr %T leaked into Info.Types", expr)
			}
		}
		for node := range info.Scopes {
			if !origNodes[node] {
				t.Fatalf("synthetic node %T leaked into Info.Scopes", node)
			}
		}
		for node := range info.Implicits {
			if !origNodes[node] {
				t.Fatalf("synthetic node %T leaked into Info.Implicits", node)
			}
		}
		for f := range info.FileVersions {
			if !origNodes[f] {
				t.Fatalf("synthetic file leaked into Info.FileVersions")
			}
		}
		// Cloned file fb must preserve the exact same per-file language version
		// as unmodified control file fa (both //go:build go1.22 under GoVersion=go1.20).
		if info.FileVersions[fa] == "" {
			t.Fatal("missing FileVersions for control file fa")
		}
		if info.FileVersions[fb] != info.FileVersions[fa] {
			t.Fatalf("FileVersions[fb] = %q, want %q (matching unmodified control fa)", info.FileVersions[fb], info.FileVersions[fa])
		}
		// Also verify same-batch cross-file rewriting preserves //go:build go1.22
		// FileVersions and leaves zero synthetic nodes in Info.
		fc := parse(t, fset, "/src/p/c.go", "//go:build go1.22\n\npackage inc\ntype GoFnBatch3 = func(x int)\n")
		fd := parse(t, fset, "/src/p/d.go", "//go:build go1.22\n\npackage inc\n//llgo:cdecl\ntype CFnBatch3 = GoFnBatch3\n")
		if err := chk.Files([]*ast.File{fc, fd}); err != nil {
			t.Fatalf("batch 3 Files: %v", err)
		}
		assertNoSynthInScope(t, pkg)
		for _, rf := range []*ast.File{fc, fd} {
			ast.Inspect(rf, func(n ast.Node) bool {
				if n != nil {
					origNodes[n] = true
				}
				return true
			})
			if info.FileVersions[rf] != info.FileVersions[fa] {
				t.Fatalf("FileVersions[%s] = %q, want %q", fset.Position(rf.Pos()).Filename, info.FileVersions[rf], info.FileVersions[fa])
			}
		}
		for id := range info.Defs {
			if !origNodes[id] {
				t.Fatalf("batch 3 synthetic ident %q leaked into Info.Defs", id.Name)
			}
		}
		for id := range info.Uses {
			if !origNodes[id] {
				t.Fatalf("batch 3 synthetic ident %q leaked into Info.Uses", id.Name)
			}
		}
		for expr := range info.Types {
			if !origNodes[expr] {
				t.Fatalf("batch 3 synthetic expr %T leaked into Info.Types", expr)
			}
		}
		for node := range info.Scopes {
			if !origNodes[node] {
				t.Fatalf("batch 3 synthetic node %T leaked into Info.Scopes", node)
			}
		}
		for node := range info.Implicits {
			if !origNodes[node] {
				t.Fatalf("batch 3 synthetic node %T leaked into Info.Implicits", node)
			}
		}
		for f := range info.FileVersions {
			if !origNodes[f] {
				t.Fatalf("batch 3 synthetic file leaked into Info.FileVersions")
			}
		}
		var retPos token.Pos
		ast.Inspect(fa, func(n ast.Node) bool {
			if rs, ok := n.(*ast.ReturnStmt); ok {
				retPos = rs.Pos()
				return false
			}
			return true
		})
		if tv, err := types.Eval(fset, pkg, retPos, "k + 1"); err != nil || tv.Type != types.Typ[types.Int] {
			t.Fatalf("Eval(k + 1) at batch 1 return pos = %v, %v; want int", tv, err)
		}
	})
}

type dirStrictImporter struct {
	wantDir string
	pkgs    map[string]*types.Package
}

func (d *dirStrictImporter) Import(path string) (*types.Package, error) {
	return nil, fmt.Errorf("unexpected Import(%q) without dir", path)
}

func (d *dirStrictImporter) ImportFrom(path, dir string, _ types.ImportMode) (*types.Package, error) {
	if path != "unsafe" && dir != d.wantDir {
		return nil, fmt.Errorf("ImportFrom(%q, dir=%q): want dir %q", path, dir, d.wantDir)
	}
	if p, ok := d.pkgs[path]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("package %q not found", path)
}

func TestImportedAliasedFuncCallConvBatchingParity(t *testing.T) {
	assertInfoClean := func(t *testing.T, info *types.Info, files ...*ast.File) {
		t.Helper()
		origNodes := make(map[ast.Node]bool)
		for _, rf := range files {
			ast.Inspect(rf, func(n ast.Node) bool {
				if n != nil {
					origNodes[n] = true
				}
				return true
			})
		}
		for id := range info.Defs {
			if !origNodes[id] {
				t.Fatalf("synthetic ident %q leaked into Info.Defs", id.Name)
			}
		}
		for id := range info.Uses {
			if !origNodes[id] {
				t.Fatalf("synthetic ident %q leaked into Info.Uses", id.Name)
			}
		}
		for expr := range info.Types {
			if !origNodes[expr] {
				t.Fatalf("synthetic expr %T leaked into Info.Types", expr)
			}
		}
		for node := range info.Scopes {
			if !origNodes[node] {
				t.Fatalf("synthetic node %T leaked into Info.Scopes", node)
			}
		}
		for node := range info.Implicits {
			if !origNodes[node] {
				t.Fatalf("synthetic node %T leaked into Info.Implicits", node)
			}
		}
		for sel := range info.Selections {
			if !origNodes[sel] {
				t.Fatalf("synthetic selector leaked into Info.Selections")
			}
		}
		for id := range info.Instances {
			if !origNodes[id] {
				t.Fatalf("synthetic ident %q leaked into Info.Instances", id.Name)
			}
		}
		for f := range info.FileVersions {
			if !origNodes[f] {
				t.Fatalf("synthetic file leaked into Info.FileVersions")
			}
		}
	}

	makeCheckedDep := func(t *testing.T, path, name, paramType string) *types.Package {
		t.Helper()
		fset := token.NewFileSet()
		src := fmt.Sprintf("package %s\ntype Fn = func(%s)\ntype DefFn func(%s)\n", name, paramType, paramType)
		f, err := parser.ParseFile(fset, name+".go", src, 0)
		if err != nil {
			t.Fatalf("ParseFile %s: %v", name, err)
		}
		var cfg types.Config
		p, err := cfg.Check(path, fset, []*ast.File{f}, nil)
		if err != nil {
			t.Fatalf("Check dep %s: %v", path, err)
		}
		return p
	}

	makeManualDep := func(path, name string) *types.Package {
		dep := types.NewPackage(path, name)
		sig := types.NewSignatureType(
			nil, nil, nil,
			types.NewTuple(types.NewParam(token.NoPos, dep, "", types.Typ[types.Int32])),
			nil, false,
		)
		fnTN := types.NewTypeName(token.NoPos, dep, "Fn", nil)
		_ = types.NewAlias(fnTN, sig)
		dep.Scope().Insert(fnTN)
		defTN := types.NewTypeName(token.NoPos, dep, "DefFn", nil)
		_ = types.NewNamed(defTN, sig, nil)
		dep.Scope().Insert(defTN)
		dep.MarkComplete()
		return dep
	}

	for _, depKind := range []string{"CheckedDep", "ManualAliasDep"} {
		for _, splitBatches := range []bool{false, true} {
			modeName := "SingleBatch"
			if splitBatches {
				modeName = "TwoBatches"
			}
			t.Run(depKind+"/"+modeName, func(t *testing.T) {
				var dep *types.Package
				if depKind == "CheckedDep" {
					dep = makeCheckedDep(t, "example.com/dep", "dep", "int32")
				} else {
					dep = makeManualDep("example.com/dep", "dep")
				}
				dep2 := makeCheckedDep(t, "example.com/dep2", "dep2", "string")

				fset := token.NewFileSet()
				fa, err := parser.ParseFile(fset, "/src/p/a.go", `package p
import (
	"example.com/dep"
	d "example.com/dep"
)
type GoFn = dep.Fn
type GoDef = dep.DefFn
type AliasedDepFn = d.Fn
`, parser.ParseComments)
				if err != nil {
					t.Fatalf("ParseFile a.go: %v", err)
				}
				fb, err := parser.ParseFile(fset, "/src/p/b.go", `package p
import (
	"unsafe"
	d "example.com/dep2"
)
var _ d.Fn
//llgo:cdecl
type CFn = (GoFn)
//llgo:cdecl
type CDef = GoDef
//llgo:cdecl
type CAliasedDep = AliasedDepFn
const (
	SizeGoFn = unsafe.Sizeof(GoFn(nil))
	SizeCFn  = unsafe.Sizeof(CFn(nil))
	SizeCA   = unsafe.Sizeof(CAliasedDep(nil))
)
var (
	g GoFn
	c CFn = g
)
`, parser.ParseComments)
				if err != nil {
					t.Fatalf("ParseFile b.go: %v", err)
				}
				fc, err := parser.ParseFile(fset, "/src/p/c.go", `package p
import (
	"unsafe"
	"example.com/dep"
)
//llgo:cdecl
type DirectDepCFn = (dep.Fn)
const SizeDirect = unsafe.Sizeof(DirectDepCFn(nil))
`, parser.ParseComments)
				if err != nil {
					t.Fatalf("ParseFile c.go: %v", err)
				}
				fd, err := parser.ParseFile(fset, "/src/p/d.go", `package p
import . "example.com/dep"
type DotGoFn = Fn
`, parser.ParseComments)
				if err != nil {
					t.Fatalf("ParseFile d.go: %v", err)
				}
				fe, err := parser.ParseFile(fset, "/src/p/e.go", `package p
import "unsafe"
//llgo:cdecl
type DotCFn = DotGoFn
const (
	SizeDotGo = unsafe.Sizeof(DotGoFn(nil))
	SizeDotC  = unsafe.Sizeof(DotCFn(nil))
)
`, parser.ParseComments)
				if err != nil {
					t.Fatalf("ParseFile e.go: %v", err)
				}

				ctx := types.NewContextWithConfig(types.ContextConfig{
					Spec: target.ResolveSpec(target.Spec{GOOS: "linux", GOARCH: "amd64"}),
				})
				pkg := types.NewPackage("example.com/p", "p")
				info := &types.Info{
					Types:        make(map[ast.Expr]types.TypeAndValue),
					Defs:         make(map[*ast.Ident]types.Object),
					Uses:         make(map[*ast.Ident]types.Object),
					Implicits:    make(map[ast.Node]types.Object),
					Selections:   make(map[*ast.SelectorExpr]*types.Selection),
					Scopes:       make(map[ast.Node]*types.Scope),
					Instances:    make(map[*ast.Ident]types.Instance),
					FileVersions: make(map[*ast.File]string),
				}
				chk := types.NewChecker(&types.Config{
					Context: ctx,
					Importer: &dirStrictImporter{
						wantDir: "/src/p",
						pkgs: map[string]*types.Package{
							"unsafe":           types.Unsafe,
							"example.com/dep":  dep,
							"example.com/dep2": dep2,
						},
					},
				}, fset, pkg, info)

				if splitBatches {
					if err := chk.Files([]*ast.File{fa, fd}); err != nil {
						t.Fatalf("batch 1 Files: %v", err)
					}
					if err := chk.Files([]*ast.File{fb, fc, fe}); err != nil {
						t.Fatalf("batch 2 Files: %v", err)
					}
				} else {
					if err := chk.Files([]*ast.File{fa, fb, fc, fd, fe}); err != nil {
						t.Fatalf("single-batch Files: %v", err)
					}
				}
				assertInfoClean(t, info, fa, fb, fc, fd, fe)

				for name, want := range map[string]int64{
					"SizeGoFn":   16,
					"SizeCFn":    8,
					"SizeCA":     8,
					"SizeDirect": 8,
					"SizeDotGo":  16,
					"SizeDotC":   8,
				} {
					got, ok := constant.Int64Val(pkg.Scope().Lookup(name).(*types.Const).Val())
					if !ok || got != want {
						t.Fatalf("%s = %v, want %d", name, got, want)
					}
				}
				for name, wantCC := range map[string]types.CallConv{
					"GoFn":         types.Go,
					"CFn":          types.CDecl,
					"CAliasedDep":  types.CDecl,
					"DirectDepCFn": types.CDecl,
					"DotGoFn":      types.Go,
					"DotCFn":       types.CDecl,
				} {
					sig := types.Unalias(pkg.Scope().Lookup(name).Type()).(*types.Signature)
					if sig.CallConv() != wantCC {
						t.Fatalf("%s CallConv = %v, want %v", name, sig.CallConv(), wantCC)
					}
				}
				// dep.Fn itself must remain Go ABI.
				if depSig := types.Unalias(dep.Scope().Lookup("Fn").Type()).(*types.Signature); depSig.CallConv() != types.Go {
					t.Fatalf("dep.Fn CallConv polluted: got %v, want Go", depSig.CallConv())
				}
				// CAliasedDep must resolve a.go's `d` (dep: int32), NOT b.go's `d` (dep2: string).
				caSig := types.Unalias(pkg.Scope().Lookup("CAliasedDep").Type()).(*types.Signature)
				if caSig.Params().At(0).Type() != types.Typ[types.Int32] {
					t.Fatalf("CAliasedDep param 0 = %v, want int32", caSig.Params().At(0).Type())
				}
				// CDef must preserve defined type dep.DefFn rather than unwrapping to Signature.
				if _, ok := types.Unalias(pkg.Scope().Lookup("CDef").Type()).(*types.Named); !ok {
					t.Fatalf("CDef type = %T, want *types.Named (dep.DefFn)", types.Unalias(pkg.Scope().Lookup("CDef").Type()))
				}
				// Info.TypeOf on parenthesized (GoFn) and (dep.Fn) must be non-nil.
				cfnSpec := fb.Decls[2].(*ast.GenDecl).Specs[0].(*ast.TypeSpec)
				if got := info.TypeOf(cfnSpec.Type); got == nil {
					t.Fatal("info.TypeOf((GoFn)) = nil, want non-nil type")
				}
				directSpec := fc.Decls[1].(*ast.GenDecl).Specs[0].(*ast.TypeSpec)
				if got := info.TypeOf(directSpec.Type); got == nil {
					t.Fatal("info.TypeOf((dep.Fn)) = nil, want non-nil type")
				}
				sel := directSpec.Type.(*ast.ParenExpr).X.(*ast.SelectorExpr)
				if info.Uses[sel.X.(*ast.Ident)] == nil || info.Uses[sel.Sel] == nil {
					t.Fatalf("info.Uses missing dep or Fn for DirectDepCFn: dep=%v Fn=%v", info.Uses[sel.X.(*ast.Ident)], info.Uses[sel.Sel])
				}
			})
		}
	}
}

func TestErasedAliasPreservesOriginalASTInfo(t *testing.T) {
	for _, layout := range []string{"SameFile", "SameBatchCrossFile", "TwoBatches"} {
		t.Run(layout, func(t *testing.T) {
			fset := token.NewFileSet()
			ctx := types.NewContextWithConfig(types.ContextConfig{
				Spec: target.ResolveSpec(target.Spec{GOOS: "linux", GOARCH: "amd64"}),
			})
			pkg := types.NewPackage("example.com/astinfo", "astinfo")
			info := &types.Info{
				Types:        make(map[ast.Expr]types.TypeAndValue),
				Defs:         make(map[*ast.Ident]types.Object),
				Uses:         make(map[*ast.Ident]types.Object),
				Implicits:    make(map[ast.Node]types.Object),
				Selections:   make(map[*ast.SelectorExpr]*types.Selection),
				Scopes:       make(map[ast.Node]*types.Scope),
				Instances:    make(map[*ast.Ident]types.Instance),
				FileVersions: make(map[*ast.File]string),
			}
			chk := types.NewChecker(&types.Config{
				Context:  ctx,
				Importer: &mapImporter{pkgs: map[string]*types.Package{"unsafe": types.Unsafe}},
			}, fset, pkg, info)

			var (
				files []*ast.File
				fa    *ast.File
				fb    *ast.File
				err   error
			)
			if layout == "SameFile" {
				fa, err = parser.ParseFile(fset, "all.go", `package astinfo
import "unsafe"
type GoFn = func(struct{ Value int }, func(x int))
//llgo:cdecl
type CFn = (GoFn)
const (
	SizeGoFn = unsafe.Sizeof(GoFn(nil))
	SizeCFn  = unsafe.Sizeof(CFn(nil))
)
func fn(k int) int { return k }
`, parser.ParseComments)
				if err != nil {
					t.Fatalf("ParseFile: %v", err)
				}
				fb = fa
				files = []*ast.File{fa}
				if err := chk.Files(files); err != nil {
					t.Fatalf("Files: %v", err)
				}
			} else {
				fa, err = parser.ParseFile(fset, "a.go", `package astinfo
type GoFn = func(struct{ Value int }, func(x int))
func fn(k int) int { return k }
`, parser.ParseComments)
				if err != nil {
					t.Fatalf("ParseFile a.go: %v", err)
				}
				fb, err = parser.ParseFile(fset, "b.go", `package astinfo
import "unsafe"
//llgo:cdecl
type CFn = (GoFn)
const (
	SizeGoFn = unsafe.Sizeof(GoFn(nil))
	SizeCFn  = unsafe.Sizeof(CFn(nil))
)
`, parser.ParseComments)
				if err != nil {
					t.Fatalf("ParseFile b.go: %v", err)
				}
				files = []*ast.File{fa, fb}
				if layout == "SameBatchCrossFile" {
					if err := chk.Files(files); err != nil {
						t.Fatalf("Files: %v", err)
					}
				} else {
					if err := chk.Files([]*ast.File{fa}); err != nil {
						t.Fatalf("batch 1 Files: %v", err)
					}
					if err := chk.Files([]*ast.File{fb}); err != nil {
						t.Fatalf("batch 2 Files: %v", err)
					}
				}
			}

			for name, want := range map[string]int64{"SizeGoFn": 16, "SizeCFn": 8} {
				got, ok := constant.Int64Val(pkg.Scope().Lookup(name).(*types.Const).Val())
				if !ok || got != want {
					t.Fatalf("%s = %v, want %d", name, got, want)
				}
			}

			goFnSig := types.Unalias(pkg.Scope().Lookup("GoFn").Type()).(*types.Signature)
			cFnSig := types.Unalias(pkg.Scope().Lookup("CFn").Type()).(*types.Signature)
			if goFnSig.CallConv() != types.Go || cFnSig.CallConv() != types.CDecl {
				t.Fatalf("callconvs: GoFn=%v CFn=%v, want Go/CDecl", goFnSig.CallConv(), cFnSig.CallConv())
			}

			// Locate original Value, x, nested FuncType, and (GoFn) ParenExpr AST nodes.
			var (
				valueIdent   *ast.Ident
				xIdent       *ast.Ident
				innerFT      *ast.FuncType
				cfnParenExpr *ast.ParenExpr
				retPos       token.Pos
			)
			ast.Inspect(fa, func(n ast.Node) bool {
				switch node := n.(type) {
				case *ast.StructType:
					valueIdent = node.Fields.List[0].Names[0]
				case *ast.FuncType:
					if len(node.Params.List) == 1 && len(node.Params.List[0].Names) == 1 && node.Params.List[0].Names[0].Name == "x" {
						innerFT = node
						xIdent = node.Params.List[0].Names[0]
					}
				case *ast.ReturnStmt:
					retPos = node.Pos()
				}
				return true
			})
			ast.Inspect(fb, func(n ast.Node) bool {
				if ts, ok := n.(*ast.TypeSpec); ok && ts.Name.Name == "CFn" {
					cfnParenExpr = ts.Type.(*ast.ParenExpr)
					return false
				}
				return true
			})
			if valueIdent == nil || xIdent == nil || innerFT == nil || cfnParenExpr == nil {
				t.Fatalf("failed to locate AST nodes: Value=%v x=%v innerFT=%v paren=%v", valueIdent, xIdent, innerFT, cfnParenExpr)
			}

			// Original AST Info.Defs for Value and x must point to GoFn's objects,
			// never overwritten by CFn's cloned signature objects.
			goFnField := goFnSig.Params().At(0).Type().(*types.Struct).Field(0)
			goFnX := goFnSig.Params().At(1).Type().(*types.Signature).Params().At(0)
			if got := info.Defs[valueIdent]; got != goFnField {
				t.Fatalf("Info.Defs[Value] = %p (%v), want GoFn field %p", got, got, goFnField)
			}
			if got := info.Defs[xIdent]; got != goFnX {
				t.Fatalf("Info.Defs[x] = %p (%v), want GoFn param %p", got, got, goFnX)
			}
			if sc := info.Scopes[innerFT]; sc == nil || sc.Lookup("x") != goFnX {
				t.Fatalf("Info.Scopes[innerFuncType] missing or mismatched x: sc=%v", sc)
			}

			// Original parenthesized RHS `(GoFn)` and inner `GoFn` ident must retain
			// Info.TypeOf and Info.Uses.
			if got := info.TypeOf(cfnParenExpr); got == nil || !types.Identical(got, goFnSig) {
				t.Fatalf("Info.TypeOf((GoFn)) = %v, want identical to GoFn", got)
			}
			goFnRef := cfnParenExpr.X.(*ast.Ident)
			if got := info.TypeOf(goFnRef); got == nil || !types.Identical(got, goFnSig) {
				t.Fatalf("Info.TypeOf(GoFn) = %v, want identical to GoFn", got)
			}
			if got := info.Uses[goFnRef]; got != pkg.Scope().Lookup("GoFn") {
				t.Fatalf("Info.Uses[GoFn] = %v, want GoFn object", got)
			}

			// Every node recorded in Info must belong to the caller's original ASTs.
			origNodes := make(map[ast.Node]bool)
			for _, rf := range files {
				ast.Inspect(rf, func(n ast.Node) bool {
					if n != nil {
						origNodes[n] = true
					}
					return true
				})
			}
			for id := range info.Defs {
				if !origNodes[id] {
					t.Fatalf("synthetic ident %q leaked into Info.Defs", id.Name)
				}
			}
			for id := range info.Uses {
				if !origNodes[id] {
					t.Fatalf("synthetic ident %q leaked into Info.Uses", id.Name)
				}
			}
			for expr := range info.Types {
				if !origNodes[expr] {
					t.Fatalf("synthetic expr %T leaked into Info.Types", expr)
				}
			}
			for node := range info.Scopes {
				if !origNodes[node] {
					t.Fatalf("synthetic node %T leaked into Info.Scopes", node)
				}
			}
			for node := range info.Implicits {
				if !origNodes[node] {
					t.Fatalf("synthetic node %T leaked into Info.Implicits", node)
				}
			}

			// Positional Eval inside fa must remain intact.
			if tv, err := types.Eval(fset, pkg, retPos, "k + 1"); err != nil || tv.Type != types.Typ[types.Int] {
				t.Fatalf("Eval(k + 1) at return pos = %v, %v; want int", tv, err)
			}
		})
	}
}

func TestErasedAliasFuncLitConstArrayLenBatchingParity(t *testing.T) {
	for _, splitBatches := range []bool{false, true} {
		name := "SingleBatch"
		if splitBatches {
			name = "TwoBatches"
		}
		t.Run(name, func(t *testing.T) {
			fset := token.NewFileSet()
			fa, err := parser.ParseFile(fset, "a.go", `package p
import "unsafe"
type GoFn = func([unsafe.Sizeof(func() { var v = 1; _ = v })]byte)
`, parser.ParseComments)
			if err != nil {
				t.Fatalf("ParseFile a.go: %v", err)
			}
			fb, err := parser.ParseFile(fset, "b.go", `package p
import "unsafe"
//llgo:cdecl
type CFn = GoFn
const (
	SizeGoFn = unsafe.Sizeof(GoFn(nil))
	SizeCFn  = unsafe.Sizeof(CFn(nil))
)
`, parser.ParseComments)
			if err != nil {
				t.Fatalf("ParseFile b.go: %v", err)
			}
			ctx := types.NewContextWithConfig(types.ContextConfig{
				Spec: target.ResolveSpec(target.Spec{GOOS: "linux", GOARCH: "amd64"}),
			})
			pkg := types.NewPackage("example.com/funclit", "p")
			info := &types.Info{
				Types:     make(map[ast.Expr]types.TypeAndValue),
				Defs:      make(map[*ast.Ident]types.Object),
				Uses:      make(map[*ast.Ident]types.Object),
				Implicits: make(map[ast.Node]types.Object),
				Scopes:    make(map[ast.Node]*types.Scope),
			}
			chk := types.NewChecker(&types.Config{
				Context:  ctx,
				Importer: &mapImporter{pkgs: map[string]*types.Package{"unsafe": types.Unsafe}},
			}, fset, pkg, info)

			if splitBatches {
				if err := chk.Files([]*ast.File{fa}); err != nil {
					t.Fatalf("batch 1 Files: %v", err)
				}
				if err := chk.Files([]*ast.File{fb}); err != nil {
					t.Fatalf("batch 2 Files: %v", err)
				}
			} else {
				if err := chk.Files([]*ast.File{fa, fb}); err != nil {
					t.Fatalf("single-batch Files: %v", err)
				}
			}

			for cname, want := range map[string]int64{"SizeGoFn": 16, "SizeCFn": 8} {
				got, ok := constant.Int64Val(pkg.Scope().Lookup(cname).(*types.Const).Val())
				if !ok || got != want {
					t.Fatalf("%s = %v, want %d", cname, got, want)
				}
			}
			goFnSig := types.Unalias(pkg.Scope().Lookup("GoFn").Type()).(*types.Signature)
			cFnSig := types.Unalias(pkg.Scope().Lookup("CFn").Type()).(*types.Signature)
			if goFnSig.CallConv() != types.Go || cFnSig.CallConv() != types.CDecl {
				t.Fatalf("CallConv: GoFn=%v CFn=%v, want Go/CDecl", goFnSig.CallConv(), cFnSig.CallConv())
			}

			// Verify zero synthetic nodes leaked from the cloned FuncLit body.
			origNodes := make(map[ast.Node]bool)
			for _, rf := range []*ast.File{fa, fb} {
				ast.Inspect(rf, func(n ast.Node) bool {
					if n != nil {
						origNodes[n] = true
					}
					return true
				})
			}
			for id := range info.Defs {
				if !origNodes[id] {
					t.Fatalf("synthetic ident %q leaked into Info.Defs", id.Name)
				}
			}
			for id := range info.Uses {
				if !origNodes[id] {
					t.Fatalf("synthetic ident %q leaked into Info.Uses", id.Name)
				}
			}
			for expr := range info.Types {
				if !origNodes[expr] {
					t.Fatalf("synthetic expr %T leaked into Info.Types", expr)
				}
			}
			for node := range info.Scopes {
				if !origNodes[node] {
					t.Fatalf("synthetic node %T leaked into Info.Scopes", node)
				}
			}
			for node := range info.Implicits {
				if !origNodes[node] {
					t.Fatalf("synthetic node %T leaked into Info.Implicits", node)
				}
			}
		})
	}
}

func TestSecondCheckerOnSamePackageDoesNotCollideSynthNames(t *testing.T) {
	fset := token.NewFileSet()
	ctx := types.NewContextWithConfig(types.ContextConfig{
		Spec: target.ResolveSpec(target.Spec{GOOS: "linux", GOARCH: "amd64"}),
	})
	pkg := types.NewPackage("example.com/twochk", "twochk")
	conf := &types.Config{
		Context:  ctx,
		Importer: &mapImporter{pkgs: map[string]*types.Package{"unsafe": types.Unsafe}},
	}

	fa, err := parser.ParseFile(fset, "a.go", "package twochk\ntype GoFn1 = func(int)\n", parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile a.go: %v", err)
	}
	fb, err := parser.ParseFile(fset, "b.go", "package twochk\nimport \"unsafe\"\n//llgo:cdecl\ntype CFn1 = GoFn1\nconst S1 = unsafe.Sizeof(CFn1(nil))\n", parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile b.go: %v", err)
	}
	fc, err := parser.ParseFile(fset, "c.go", "package twochk\ntype GoFn2 = func(string)\n", parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile c.go: %v", err)
	}
	fd, err := parser.ParseFile(fset, "d.go", "package twochk\nimport \"unsafe\"\n//llgo:cdecl\ntype CFn2 = GoFn2\nconst S2 = unsafe.Sizeof(CFn2(nil))\n", parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile d.go: %v", err)
	}

	chk1 := types.NewChecker(conf, fset, pkg, nil)
	if err := chk1.Files([]*ast.File{fa}); err != nil {
		t.Fatalf("chk1 batch 1: %v", err)
	}
	if err := chk1.Files([]*ast.File{fb}); err != nil {
		t.Fatalf("chk1 batch 2: %v", err)
	}

	// Second Checker on the exact same Package (also with nil Context so it must
	// inherit pkg.ctx and avoid colliding with $llgo_ccalias_0 in pkg.goPkg).
	chk2 := types.NewChecker(&types.Config{
		Importer: &mapImporter{pkgs: map[string]*types.Package{"unsafe": types.Unsafe}},
	}, fset, pkg, nil)
	if err := chk2.Files([]*ast.File{fc}); err != nil {
		t.Fatalf("chk2 batch 1: %v", err)
	}
	if err := chk2.Files([]*ast.File{fd}); err != nil {
		t.Fatalf("chk2 batch 2: %v", err)
	}

	for _, name := range pkg.Scope().Names() {
		if strings.HasPrefix(name, "$llgo_ccalias_") {
			t.Fatalf("synthetic alias %q leaked into pkg.Scope()", name)
		}
	}
	cFn1 := types.Unalias(pkg.Scope().Lookup("CFn1").Type()).(*types.Signature)
	cFn2 := types.Unalias(pkg.Scope().Lookup("CFn2").Type()).(*types.Signature)
	if cFn1.CallConv() != types.CDecl || cFn1.Params().At(0).Type() != types.Typ[types.Int] {
		t.Fatalf("CFn1 = %v (cc=%v), want func<cdecl>(int)", cFn1, cFn1.CallConv())
	}
	if cFn2.CallConv() != types.CDecl || cFn2.Params().At(0).Type() != types.Typ[types.String] {
		t.Fatalf("CFn2 = %v (cc=%v), want func<cdecl>(string)", cFn2, cFn2.CallConv())
	}
	for _, cname := range []string{"S1", "S2"} {
		got, ok := constant.Int64Val(pkg.Scope().Lookup(cname).(*types.Const).Val())
		if !ok || got != 8 {
			t.Fatalf("%s = %v, want 8", cname, got)
		}
	}
}

func TestDotImportDoesNotMatchUnexportedAlias(t *testing.T) {
	fset := token.NewFileSet()
	depSrc := `package dep
type error = func(int)
const Exported = 1
`
	depFile, err := parser.ParseFile(fset, "dep.go", depSrc, 0)
	if err != nil {
		t.Fatalf("ParseFile dep.go: %v", err)
	}
	var depCfg types.Config
	depPkg, err := depCfg.Check("example.com/dep", fset, []*ast.File{depFile}, nil)
	if err != nil {
		t.Fatalf("Check dep: %v", err)
	}

	mainSrc := `package main
import . "example.com/dep"
var _ = Exported
//llgo:cdecl
type A = error
var _ error = A(nil)
`
	mainFile, err := parser.ParseFile(fset, "main.go", mainSrc, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile main.go: %v", err)
	}
	conf := types.Config{
		Importer: &mapImporter{pkgs: map[string]*types.Package{"example.com/dep": depPkg}},
	}
	pkg, err := conf.Check("example.com/main", fset, []*ast.File{mainFile}, nil)
	if err != nil {
		t.Fatalf("Config.Check main: %v", err)
	}
	aTyp := types.Unalias(pkg.Scope().Lookup("A").Type())
	if _, isSig := aTyp.(*types.Signature); isSig {
		t.Fatalf("A bound to unexported dot-imported func alias %v instead of predeclared error", aTyp)
	}
	if !types.Identical(aTyp, types.Universe.Lookup("error").Type()) {
		t.Fatalf("A type = %v, want predeclared error", aTyp)
	}
}

func TestCheckExprReusingInfoOverwritesPreviousEntries(t *testing.T) {
	fset := token.NewFileSet()
	pkg1 := types.NewPackage("example.com/p1", "p1")
	x1 := types.NewConst(token.NoPos, pkg1, "X", types.Typ[types.Int], constant.MakeInt64(1))
	pkg1.Scope().Insert(x1)

	pkg2 := types.NewPackage("example.com/p2", "p2")
	x2 := types.NewConst(token.NoPos, pkg2, "X", types.Typ[types.String], constant.MakeString("second"))
	pkg2.Scope().Insert(x2)

	ident := ast.NewIdent("X")
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Uses:  make(map[*ast.Ident]types.Object),
	}

	if err := types.CheckExpr(fset, pkg1, token.NoPos, ident, info); err != nil {
		t.Fatalf("first CheckExpr(X) failed: %v", err)
	}
	if info.Uses[ident] != x1 {
		t.Fatalf("first CheckExpr info.Uses[X] = %v, want x1", info.Uses[ident])
	}
	if tv := info.Types[ident]; tv.Type != types.Typ[types.Int] || tv.Value.String() != "1" {
		t.Fatalf("first CheckExpr info.Types[X] = %v (%v), want int (1)", tv.Type, tv.Value)
	}

	if err := types.CheckExpr(fset, pkg2, token.NoPos, ident, info); err != nil {
		t.Fatalf("second CheckExpr(X) failed: %v", err)
	}
	if info.Uses[ident] != x2 {
		t.Fatalf("second CheckExpr info.Uses[X] = %v, want x2", info.Uses[ident])
	}
	if tv := info.Types[ident]; tv.Type != types.Typ[types.String] || constant.StringVal(tv.Value) != "second" {
		t.Fatalf("second CheckExpr info.Types[X] = %v (%v), want string (\"second\")", tv.Type, tv.Value)
	}
}

func TestPositionalCheckExprPreservesLocalObjectIdentityAndScopes(t *testing.T) {
	fset := token.NewFileSet()
	src := `package posid

func f(x int) int {
	type localT struct{ V int }
	const localK = 2
	y := x + localK
	_ = localT{V: y}
	return y
}
`
	f, err := parser.ParseFile(fset, "posid.go", src, 0)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	origInfo := &types.Info{
		Defs:   make(map[*ast.Ident]types.Object),
		Uses:   make(map[*ast.Ident]types.Object),
		Types:  make(map[ast.Expr]types.TypeAndValue),
		Scopes: make(map[ast.Node]*types.Scope),
	}
	var conf types.Config
	pkg, err := conf.Check("example.com/posid", fset, []*ast.File{f}, origInfo)
	if err != nil {
		t.Fatalf("Config.Check: %v", err)
	}
	if got := pkg.Scope().NumChildren(); got != 1 {
		t.Fatalf("initial pkg.Scope().NumChildren() = %d, want 1", got)
	}

	var (
		retPos    token.Pos
		origX     types.Object
		origY     types.Object
		origK     types.Object
		origT     types.Object
		origField types.Object
	)
	ast.Inspect(f, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.ReturnStmt:
			retPos = node.Pos()
		case *ast.Ident:
			if obj := origInfo.Defs[node]; obj != nil {
				switch node.Name {
				case "x":
					origX = obj
				case "y":
					origY = obj
				case "localK":
					origK = obj
				case "localT":
					origT = obj
				case "V":
					origField = obj
				}
			}
		}
		return true
	})
	if !retPos.IsValid() || origX == nil || origY == nil || origK == nil || origT == nil || origField == nil {
		t.Fatalf("missing orig objects: retPos=%v x=%v y=%v k=%v t=%v field=%v", retPos, origX, origY, origK, origT, origField)
	}

	for _, tc := range []struct {
		expr string
		want map[string]types.Object
	}{
		{expr: "y", want: map[string]types.Object{"y": origY}},
		{expr: "x + y + localK", want: map[string]types.Object{"x": origX, "y": origY, "localK": origK}},
		{expr: "localT{V: y}", want: map[string]types.Object{"localT": origT, "V": origField, "y": origY}},
		{expr: "func(z int) int { return y + z }", want: map[string]types.Object{"y": origY}},
	} {
		expr, err := parser.ParseExpr(tc.expr)
		if err != nil {
			t.Fatalf("ParseExpr(%q): %v", tc.expr, err)
		}
		checkInfo := &types.Info{
			Defs:   make(map[*ast.Ident]types.Object),
			Uses:   make(map[*ast.Ident]types.Object),
			Types:  make(map[ast.Expr]types.TypeAndValue),
			Scopes: make(map[ast.Node]*types.Scope),
		}
		if err := types.CheckExpr(fset, pkg, retPos, expr, checkInfo); err != nil {
			t.Fatalf("CheckExpr(%q) failed: %v", tc.expr, err)
		}
		for name, wantObj := range tc.want {
			found := false
			for id, gotObj := range checkInfo.Uses {
				if id.Name == name {
					found = true
					if gotObj != wantObj {
						t.Fatalf("CheckExpr(%q) Info.Uses[%s] = %p (%v), want %p (%v)", tc.expr, name, gotObj, gotObj, wantObj, wantObj)
					}
				}
			}
			if !found {
				t.Fatalf("CheckExpr(%q) did not record %s in Info.Uses", tc.expr, name)
			}
		}
		if got := pkg.Scope().NumChildren(); got != 1 {
			t.Fatalf("after CheckExpr(%q): pkg.Scope().NumChildren() = %d, want 1", tc.expr, got)
		}
	}
}

func TestDotImportPrunePreservesDependencyObjectIdentityInSharedContext(t *testing.T) {
	for _, sameCtxDep := range []bool{true, false} {
		name := "DepInSharedCtx"
		if !sameCtxDep {
			name = "DepInSeparateCtx"
		}
		t.Run(name, func(t *testing.T) {
			fset := token.NewFileSet()
			sharedCtx := types.NewContext()

			depSrc := `package dep
type T struct{}
const K = 7
var V int
func F() {}
`
			depFile, err := parser.ParseFile(fset, "dep.go", depSrc, 0)
			if err != nil {
				t.Fatalf("ParseFile dep.go: %v", err)
			}
			depConf := types.Config{}
			if sameCtxDep {
				depConf.Context = sharedCtx
			}
			depPkg, err := depConf.Check("example.com/dep", fset, []*ast.File{depFile}, nil)
			if err != nil {
				t.Fatalf("Check dep: %v", err)
			}

			imp := &mapImporter{pkgs: map[string]*types.Package{"example.com/dep": depPkg}}

			p1Src := `package p1
import . "example.com/dep"
var _ T
var _ = K + V
var _ = F
`
			p1File, err := parser.ParseFile(fset, "p1.go", p1Src, 0)
			if err != nil {
				t.Fatalf("ParseFile p1.go: %v", err)
			}
			p1Info := &types.Info{Uses: make(map[*ast.Ident]types.Object)}
			conf := types.Config{Context: sharedCtx, Importer: imp}
			if _, err := conf.Check("example.com/p1", fset, []*ast.File{p1File}, p1Info); err != nil {
				t.Fatalf("Check p1: %v", err)
			}

			p2Src := `package p2
import "example.com/dep"
var _ dep.T
var _ = dep.K + dep.V
var _ = dep.F
`
			p2File, err := parser.ParseFile(fset, "p2.go", p2Src, 0)
			if err != nil {
				t.Fatalf("ParseFile p2.go: %v", err)
			}
			p2Info := &types.Info{Uses: make(map[*ast.Ident]types.Object)}
			if _, err := conf.Check("example.com/p2", fset, []*ast.File{p2File}, p2Info); err != nil {
				t.Fatalf("Check p2: %v", err)
			}

			for _, pkgName := range []struct {
				label string
				info  *types.Info
			}{
				{"p1", p1Info},
				{"p2", p2Info},
			} {
				for _, sym := range []string{"T", "K", "V", "F"} {
					want := depPkg.Scope().Lookup(sym)
					found := false
					for id, got := range pkgName.info.Uses {
						if id.Name == sym {
							found = true
							if got != want {
								t.Fatalf("%s Info.Uses[%s] = %p (%v), want dep.Scope().Lookup(%q) = %p", pkgName.label, sym, got, got, sym, want)
							}
						}
					}
					if !found {
						t.Fatalf("%s did not record %s in Info.Uses", pkgName.label, sym)
					}
				}
			}
		})
	}
}

func TestMethodSetAndLookupShadowingMatchesGoTypes(t *testing.T) {
	src := `package shadow

type Iface interface{ Kind() int }
type Inner struct{}
func (*Inner) Type() Iface { return nil }
func (*Inner) Kept() int   { return 1 }
func (*Inner) Fielded()    {}

type Wrap struct{ Iface }
type Outer struct {
	*Inner
	Fielded int
}
func (*Outer) Type() Wrap { return Wrap{} }

type WithField struct{ M int }
type WithMethod struct{}
func (WithMethod) M() {}
type CollideFM struct {
	WithField
	WithMethod
}

type Leaf struct{}
func (Leaf) L() {}
type Mid1 struct{ Leaf }
type Mid2 struct{ Leaf }
type TopAmbig struct {
	Mid1
	Mid2
}

type NamedPtr *Outer
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "shadow.go", src, 0)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	var conf types.Config
	pkg, err := conf.Check("example.com/shadow", fset, []*ast.File{f}, nil)
	if err != nil {
		t.Fatalf("Config.Check: %v", err)
	}

	outer := pkg.Scope().Lookup("Outer").Type()
	ptrOuter := types.NewPointer(outer)

	// Value method set of Outer must NOT promote (*Inner).Type() because
	// (*Outer).Type() at depth 0 shadows it; Fielded field also hides (*Inner).Fielded().
	msVal := types.NewMethodSet(outer)
	if sel := msVal.Lookup(pkg, "Type"); sel != nil {
		t.Fatalf("NewMethodSet(Outer).Lookup(Type) = %v, want nil (shadowed by (*Outer).Type)", sel)
	}
	if sel := msVal.Lookup(pkg, "Fielded"); sel != nil {
		t.Fatalf("NewMethodSet(Outer).Lookup(Fielded) = %v, want nil (shadowed by Outer.Fielded field)", sel)
	}
	if sel := msVal.Lookup(pkg, "Kept"); sel == nil {
		t.Fatal("NewMethodSet(Outer).Lookup(Kept) = nil, want promoted (*Inner).Kept")
	}

	// Pointer method set of *Outer must resolve Type to (*Outer).Type returning Wrap,
	// and must still hide (*Inner).Fielded behind Outer.Fielded.
	msPtr := types.NewMethodSet(ptrOuter)
	selType := msPtr.Lookup(pkg, "Type")
	if selType == nil {
		t.Fatal("NewMethodSet(*Outer).Lookup(Type) = nil")
	}
	retType := selType.Obj().(*types.Func).Signature().Results().At(0).Type()
	if retType != pkg.Scope().Lookup("Wrap").Type() {
		t.Fatalf("(*Outer).Type() returns %v, want Wrap", retType)
	}
	if sel := msPtr.Lookup(pkg, "Fielded"); sel != nil {
		t.Fatalf("NewMethodSet(*Outer).Lookup(Fielded) = %v, want nil", sel)
	}

	// LookupFieldOrMethod on non-addressable Outer for "Type" must signal
	// pointer-receiver required (nil, nil, true), NOT promote (*Inner).Type().
	if obj, idx, ind := types.LookupFieldOrMethod(outer, false, pkg, "Type"); obj != nil || idx != nil || !ind {
		t.Fatalf("LookupFieldOrMethod(Outer, false, Type) = (%v, %v, %v), want (nil, nil, true)", obj, idx, ind)
	}
	// Addressable Outer resolves to (*Outer).Type().
	if obj, _, _ := types.LookupFieldOrMethod(outer, true, pkg, "Type"); obj == nil || obj.Type().(*types.Signature).Results().At(0).Type() != pkg.Scope().Lookup("Wrap").Type() {
		t.Fatalf("LookupFieldOrMethod(Outer, true, Type) = %v, want (*Outer).Type() -> Wrap", obj)
	}

	// Same-depth field vs method collision must be ambiguous.
	collideFM := pkg.Scope().Lookup("CollideFM").Type()
	if sel := types.NewMethodSet(collideFM).Lookup(pkg, "M"); sel != nil {
		t.Fatalf("NewMethodSet(CollideFM).Lookup(M) = %v, want nil (field/method collision)", sel)
	}
	if obj, idx, _ := types.LookupFieldOrMethod(collideFM, true, pkg, "M"); obj != nil || idx == nil {
		t.Fatalf("LookupFieldOrMethod(CollideFM, M) = (%v, %v), want (nil, non-nil ambiguous index)", obj, idx)
	}

	// Multi-level duplicate embedded type must propagate multiples and stay ambiguous.
	topAmbig := pkg.Scope().Lookup("TopAmbig").Type()
	if sel := types.NewMethodSet(topAmbig).Lookup(pkg, "L"); sel != nil {
		t.Fatalf("NewMethodSet(TopAmbig).Lookup(L) = %v, want nil (ambiguous)", sel)
	}
	if obj, idx, _ := types.LookupFieldOrMethod(topAmbig, true, pkg, "L"); obj != nil || idx == nil {
		t.Fatalf("LookupFieldOrMethod(TopAmbig, L) = (%v, %v), want (nil, non-nil ambiguous index)", obj, idx)
	}

	// Named pointer type has empty method set, exposes fields, but never methods.
	namedPtr := pkg.Scope().Lookup("NamedPtr").Type()
	if got := types.NewMethodSet(namedPtr).Len(); got != 0 {
		t.Fatalf("NewMethodSet(NamedPtr).Len() = %d, want 0", got)
	}
	if obj, _, _ := types.LookupFieldOrMethod(namedPtr, false, pkg, "Fielded"); obj == nil {
		t.Fatal("LookupFieldOrMethod(NamedPtr, Fielded) = nil, want field")
	}
	if obj, _, _ := types.LookupFieldOrMethod(namedPtr, false, pkg, "Type"); obj != nil {
		t.Fatalf("LookupFieldOrMethod(NamedPtr, Type) = %v, want nil", obj)
	}
}

func TestLookupAndMethodSetParityWithGoTypes(t *testing.T) {
	src := `package parity

type S struct {
	hidden int
}
func (S) priv() {}
func (*S) ptrPriv() {}

type I interface {
	M()
	priv()
}

type EmbedI struct {
	I
}

var AnonI interface {
	M()
	priv()
}

func UseAnon[T interface{ M(); priv() }](x T) {}
func UseNamed[U I](x U) {}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "parity.go", src, 0)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	const pkgPath = "example.com/parity"

	var llConf types.Config
	llPkg, err := llConf.Check(pkgPath, fset, []*ast.File{f}, nil)
	if err != nil {
		t.Fatalf("llgo/types Check: %v", err)
	}
	var goConf gotypes.Config
	goPkg, err := goConf.Check(pkgPath, fset, []*ast.File{f}, nil)
	if err != nil {
		t.Fatalf("go/types Check: %v", err)
	}

	// Distinct Package instances with the same import path must match unexported
	// fields and methods identically to go/types.
	llSamePath := types.NewPackage(pkgPath, "othername")
	goSamePath := gotypes.NewPackage(pkgPath, "othername")
	llDiffPath := types.NewPackage("example.com/other", "parity")
	goDiffPath := gotypes.NewPackage("example.com/other", "parity")

	llS := llPkg.Scope().Lookup("S").Type()
	goS := goPkg.Scope().Lookup("S").Type()
	llI := llPkg.Scope().Lookup("I").Type()
	goI := goPkg.Scope().Lookup("I").Type()
	llAnonI := llPkg.Scope().Lookup("AnonI").Type()
	goAnonI := goPkg.Scope().Lookup("AnonI").Type()
	llEmbedI := llPkg.Scope().Lookup("EmbedI").Type()
	goEmbedI := goPkg.Scope().Lookup("EmbedI").Type()
	llTPAnon := llPkg.Scope().Lookup("UseAnon").(*types.Func).Signature().TypeParams().At(0)
	goTPAnon := goPkg.Scope().Lookup("UseAnon").(*gotypes.Func).Type().(*gotypes.Signature).TypeParams().At(0)
	llTPNamed := llPkg.Scope().Lookup("UseNamed").(*types.Func).Signature().TypeParams().At(0)
	goTPNamed := goPkg.Scope().Lookup("UseNamed").(*gotypes.Func).Type().(*gotypes.Signature).TypeParams().At(0)

	type pkgCase struct {
		label string
		llPkg *types.Package
		goPkg *gotypes.Package
	}
	pkgCases := []pkgCase{
		{"orig", llPkg, goPkg},
		{"same-path", llSamePath, goSamePath},
		{"diff-path", llDiffPath, goDiffPath},
		{"nil", nil, nil},
	}

	type typeCase struct {
		label string
		llT   types.Type
		goT   gotypes.Type
		names []string
	}
	typeCases := []typeCase{
		{"S", llS, goS, []string{"hidden", "priv", "ptrPriv"}},
		{"*S", types.NewPointer(llS), gotypes.NewPointer(goS), []string{"hidden", "priv", "ptrPriv"}},
		{"I", llI, goI, []string{"M", "priv"}},
		{"*I", types.NewPointer(llI), gotypes.NewPointer(goI), []string{"M", "priv"}},
		{"AnonI", llAnonI, goAnonI, []string{"M", "priv"}},
		{"EmbedI", llEmbedI, goEmbedI, []string{"M", "priv"}},
		{"*EmbedI", types.NewPointer(llEmbedI), gotypes.NewPointer(goEmbedI), []string{"M", "priv"}},
		{"TPAnon", llTPAnon, goTPAnon, []string{"M", "priv"}},
		{"*TPAnon", types.NewPointer(llTPAnon), gotypes.NewPointer(goTPAnon), []string{"M", "priv"}},
		{"TPNamed", llTPNamed, goTPNamed, []string{"M", "priv"}},
	}

	for _, tc := range typeCases {
		llMS := types.NewMethodSet(tc.llT)
		goMS := gotypes.NewMethodSet(tc.goT)
		if llMS.Len() != goMS.Len() {
			t.Fatalf("%s: MethodSet.Len = %d, go/types = %d", tc.label, llMS.Len(), goMS.Len())
		}
		for i := 0; i < llMS.Len(); i++ {
			ls, gs := llMS.At(i), goMS.At(i)
			if ls.Obj().Id() != gs.Obj().Id() || ls.Indirect() != gs.Indirect() || !intsEqual(ls.Index(), gs.Index()) {
				t.Fatalf("%s: MethodSet.At(%d) = (%s, idx=%v, ind=%v), go/types = (%s, idx=%v, ind=%v)",
					tc.label, i, ls.Obj().Id(), ls.Index(), ls.Indirect(), gs.Obj().Id(), gs.Index(), gs.Indirect())
			}
		}

		for _, pc := range pkgCases {
			for _, name := range tc.names {
				ls := llMS.Lookup(pc.llPkg, name)
				gs := goMS.Lookup(pc.goPkg, name)
				if (ls == nil) != (gs == nil) {
					t.Fatalf("%s[%s].MethodSet.Lookup(%s) nil=%v, go/types nil=%v", tc.label, pc.label, name, ls == nil, gs == nil)
				}
				if ls != nil && (ls.Indirect() != gs.Indirect() || !intsEqual(ls.Index(), gs.Index())) {
					t.Fatalf("%s[%s].MethodSet.Lookup(%s) = (idx=%v, ind=%v), go/types = (idx=%v, ind=%v)",
						tc.label, pc.label, name, ls.Index(), ls.Indirect(), gs.Index(), gs.Indirect())
				}

				for _, addr := range []bool{false, true} {
					lo, li, lind := types.LookupFieldOrMethod(tc.llT, addr, pc.llPkg, name)
					goObj, gi, gind := gotypes.LookupFieldOrMethod(tc.goT, addr, pc.goPkg, name)
					if (lo == nil) != (goObj == nil) || lind != gind || !intsEqual(li, gi) {
						t.Fatalf("LookupFieldOrMethod(%s, addr=%v, %s, %s) = (nil=%v, idx=%v, ind=%v), go/types = (nil=%v, idx=%v, ind=%v)",
							tc.label, addr, pc.label, name, lo == nil, li, lind, goObj == nil, gi, gind)
					}
					if lo != nil && lo.Id() != goObj.Id() {
						t.Fatalf("LookupFieldOrMethod(%s, addr=%v, %s, %s) id = %s, go/types = %s",
							tc.label, addr, pc.label, name, lo.Id(), goObj.Id())
					}
				}
			}
		}
	}
}

func intsEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestConcurrentEvalAndCheckExprAfterPostCheckScopeInsert(t *testing.T) {
	fset := token.NewFileSet()
	checkedFile, err := parser.ParseFile(fset, "checked.go", "package checked\nconst Base = 10\n", 0)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	var conf types.Config
	checkedPkg, err := conf.Check("example.com/checkedrace", fset, []*ast.File{checkedFile}, nil)
	if err != nil {
		t.Fatalf("Config.Check: %v", err)
	}
	extraTN := types.NewTypeName(token.NoPos, checkedPkg, "ExtraInt", nil)
	_ = types.NewNamed(extraTN, types.Typ[types.Int], nil)
	checkedPkg.Scope().Insert(extraTN)
	checkedPkg.Scope().Insert(types.NewConst(token.NoPos, checkedPkg, "Extra", types.Typ[types.UntypedInt], constant.MakeInt64(5)))
	checkedPkg.Scope().Insert(types.NewVar(token.NoPos, checkedPkg, "V", types.Typ[types.Int]))

	manualPkg := types.NewPackage("example.com/manualrace", "manualrace")
	manualPkg.Scope().Insert(types.NewConst(token.NoPos, manualPkg, "K", types.Typ[types.UntypedInt], constant.MakeInt64(42)))

	const workers = 16
	start := make(chan struct{})
	errCh := make(chan error, workers*3)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			tv, err := types.Eval(fset, checkedPkg, token.NoPos, "ExtraInt(Base + Extra)")
			if err != nil {
				errCh <- fmt.Errorf("Eval checkedPkg: %w", err)
				return
			}
			if got, ok := constant.Int64Val(tv.Value); !ok || got != 15 {
				errCh <- fmt.Errorf("Eval checkedPkg = %v, want 15", tv.Value)
				return
			}
			expr, err := parser.ParseExpr("ExtraInt(V + Extra)")
			if err != nil {
				errCh <- fmt.Errorf("ParseExpr: %w", err)
				return
			}
			info := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue)}
			if err := types.CheckExpr(fset, checkedPkg, token.NoPos, expr, info); err != nil {
				errCh <- fmt.Errorf("CheckExpr checkedPkg: %w", err)
				return
			}
			tvM, err := types.Eval(fset, manualPkg, token.NoPos, "K + 1")
			if err != nil {
				errCh <- fmt.Errorf("Eval manualPkg: %w", err)
				return
			}
			if got, ok := constant.Int64Val(tvM.Value); !ok || got != 43 {
				errCh <- fmt.Errorf("Eval manualPkg = %v, want 43", tvM.Value)
				return
			}
		}()
	}
	close(start)
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatal(err)
	}
}
