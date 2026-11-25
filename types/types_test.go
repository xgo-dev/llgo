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
