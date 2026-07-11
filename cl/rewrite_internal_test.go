//go:build !llgo
// +build !llgo

package cl

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"runtime"
	"strings"
	"testing"

	gpackages "github.com/goplus/gogen/packages"
	llssa "github.com/goplus/llgo/ssa"
	"github.com/goplus/llgo/ssa/ssatest"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func init() {
	llssa.Initialize(llssa.InitAll | llssa.InitNative)
}

func compileWithRewrites(t *testing.T, src string, rewrites map[string]string) string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "rewrite.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	importer := gpackages.NewImporter(fset)
	mode := ssa.SanityCheckFunctions | ssa.InstantiateGenerics
	pkg, _, err := ssautil.BuildPackage(&types.Config{Importer: importer}, fset,
		types.NewPackage(file.Name.Name, file.Name.Name), []*ast.File{file}, mode)
	if err != nil {
		t.Fatalf("build package failed: %v", err)
	}
	prog := ssatest.NewProgramEx(t, nil, importer)
	prog.TypeSizes(types.SizesFor("gc", runtime.GOARCH))
	ret, _, err := NewPackageEx(prog, nil, rewrites, pkg, []*ast.File{file})
	if err != nil {
		t.Fatalf("NewPackageEx failed: %v", err)
	}
	return ret.String()
}

func assertNoStoreToGlobal(t *testing.T, ir, global string) {
	t.Helper()
	for _, line := range strings.Split(ir, "\n") {
		if strings.Contains(line, "store ") && strings.Contains(line, global) {
			t.Fatalf("%s initializer store was not folded: %s\n%s", global, line, ir)
		}
	}
}

func assertStoreToGlobal(t *testing.T, ir, global string) {
	t.Helper()
	for _, line := range strings.Split(ir, "\n") {
		if strings.Contains(line, "store ") && strings.Contains(line, global) {
			return
		}
	}
	t.Fatalf("expected store to %s in IR:\n%s", global, ir)
}

func TestRewriteGlobalStrings(t *testing.T) {
	const src = `package rewritepkg
var VarInit = "original_value"
var VarPlain string
func Use() string { return VarInit + VarPlain }
`
	ir := compileWithRewrites(t, src, map[string]string{
		"VarInit":  "rewrite_init",
		"VarPlain": "rewrite_plain",
	})
	if strings.Contains(ir, "original_value") {
		t.Fatalf("original initializer still present:\n%s", ir)
	}
	for _, want := range []string{`c"rewrite_init"`, `c"rewrite_plain"`} {
		if !strings.Contains(ir, want) {
			t.Fatalf("missing %s in IR:\n%s", want, ir)
		}
	}
}

func TestStaticGlobalLiteralInit(t *testing.T) {
	const src = `package staticinit

type Names struct {
	Value [2]string
	Nested Nested
}

type Nested struct {
	Type [2]string
}

var MethodNames = Names{
	Value: [2]string{"KeepValue", "KeepValueAlt"},
	Nested: Nested{
		Type: [2]string{"KeepType", "KeepTypeAlt"},
	},
}

func Use() string {
	return MethodNames.Value[0] + MethodNames.Nested.Type[1]
}
`
	ir := compileWithRewrites(t, src, nil)
	if !strings.Contains(ir, "@staticinit.MethodNames = global %staticinit.Names") {
		t.Fatalf("missing MethodNames global initializer:\n%s", ir)
	}
	if strings.Contains(ir, "@staticinit.MethodNames = global %staticinit.Names zeroinitializer") {
		t.Fatalf("MethodNames still uses a zero initializer:\n%s", ir)
	}
	for _, want := range []string{`c"KeepValue"`, `c"KeepValueAlt"`, `c"KeepType"`, `c"KeepTypeAlt"`} {
		if !strings.Contains(ir, want) {
			t.Fatalf("missing %s in IR:\n%s", want, ir)
		}
	}
	assertNoStoreToGlobal(t, ir, "@staticinit.MethodNames")
}

func TestStaticGlobalScalarAndSparseArrayInit(t *testing.T) {
	const src = `package staticinit

var StaticBool = true
var StaticInt int8 = -7
var StaticUint uint16 = 42
var StaticFloat = 1.5
var StaticComplex = complex(2, -3)
var StaticSparse = [4]int{1: 7, 3: 9}

func Use() (bool, int8, uint16, float64, complex128, int) {
	return StaticBool, StaticInt, StaticUint, StaticFloat, StaticComplex, StaticSparse[3]
}
`
	ir := compileWithRewrites(t, src, nil)
	for _, want := range []string{
		"@staticinit.StaticBool = global i1 true",
		"@staticinit.StaticInt = global i8 -7",
		"@staticinit.StaticUint = global i16 42",
		"@staticinit.StaticFloat = global double 1.500000e+00",
		"@staticinit.StaticComplex = global { double, double } { double 2.000000e+00, double -3.000000e+00 }",
		"@staticinit.StaticSparse = global [4 x i64] [i64 0, i64 7, i64 0, i64 9]",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("missing static initializer %q in IR:\n%s", want, ir)
		}
	}
	for _, global := range []string{
		"@staticinit.StaticBool",
		"@staticinit.StaticInt",
		"@staticinit.StaticUint",
		"@staticinit.StaticFloat",
		"@staticinit.StaticComplex",
		"@staticinit.StaticSparse",
	} {
		assertNoStoreToGlobal(t, ir, global)
	}
}

func TestStaticGlobalInitSkipsDynamicGlobal(t *testing.T) {
	const src = `package staticinit

func value() int { return 3 }

var Dynamic = [2]int{value(), 9}

func Use() int { return Dynamic[0] + Dynamic[1] }
`
	ir := compileWithRewrites(t, src, nil)
	if !strings.Contains(ir, "@staticinit.Dynamic = global [2 x i64] zeroinitializer") {
		t.Fatalf("dynamic global should keep zero initializer:\n%s", ir)
	}
	assertStoreToGlobal(t, ir, "@staticinit.Dynamic")
}

func TestStaticGlobalInitSkipsExternalGlobal(t *testing.T) {
	const src = `package staticinit

import _ "unsafe"

//go:linkname External C.external
var External = 7

func Use() int { return External }
`
	ir := compileWithRewrites(t, src, nil)
	if !strings.Contains(ir, "@C.external = external global i64") {
		t.Fatalf("linknamed global should remain an external declaration:\n%s", ir)
	}
	assertStoreToGlobal(t, ir, "@C.external")
}

func TestStaticGlobalInitDeterministicOrder(t *testing.T) {
	const src = `package staticinit

var Zulu = "zulu-static-init"
var Alpha = "alpha-static-init"

func Use() string { return Zulu + Alpha }
`
	ir := compileWithRewrites(t, src, nil)
	alpha := strings.Index(ir, `c"alpha-static-init"`)
	zulu := strings.Index(ir, `c"zulu-static-init"`)
	if alpha < 0 || zulu < 0 {
		t.Fatalf("missing static string data:\n%s", ir)
	}
	if alpha > zulu {
		t.Fatalf("static initializers were not built in global-name order:\n%s", ir)
	}
}

func TestStaticGlobalInitSkipsLargeArray(t *testing.T) {
	length := maxStaticInitArrayElements + 1
	src := fmt.Sprintf(`package staticinit

var Large = [%d]byte{%d: 1}

func Use() byte { return Large[%d] }
`, length, length-1, length-1)
	ir := compileWithRewrites(t, src, nil)
	want := fmt.Sprintf("@staticinit.Large = global [%d x i8] zeroinitializer", length)
	if !strings.Contains(ir, want) {
		t.Fatalf("large array should keep a zero initializer:\n%s", ir)
	}
	assertStoreToGlobal(t, ir, "@staticinit.Large")
}

func TestStaticInitHelperRejectsUnsupportedPaths(t *testing.T) {
	if _, ok := staticInitConstIndex(nil); ok {
		t.Fatal("nil index should not be accepted")
	}
	if _, ok := staticInitConstIndex(ssa.NewConst(constant.MakeInt64(-1), types.Typ[types.Int])); ok {
		t.Fatal("negative index should not be accepted")
	}
	if _, ok := staticInitStorePath(ssa.NewConst(constant.MakeInt64(0), types.Typ[types.Int])); ok {
		t.Fatal("non-address value should not produce a static init path")
	}
	if global := staticInitRootGlobal(ssa.NewConst(constant.MakeInt64(0), types.Typ[types.Int])); global != nil {
		t.Fatalf("non-address value should not have root global: %v", global)
	}

	ssapkg := buildSSAPackage(t, `package foo
var Array [2]int
`)
	arrayGlobal, ok := ssapkg.Members["Array"].(*ssa.Global)
	if !ok {
		t.Fatalf("missing Array global: %T", ssapkg.Members["Array"])
	}
	if _, ok := staticInitStorePath(&ssa.IndexAddr{X: arrayGlobal, Index: arrayGlobal}); ok {
		t.Fatal("dynamic index should not produce a static init path")
	}

	c := ssa.NewConst(constant.MakeInt64(1), types.Typ[types.Int])
	root := new(staticInitNode)
	if !root.add(nil, c) {
		t.Fatal("first root value should be accepted")
	}
	if root.add([]staticInitPathElem{{index: 0}}, c) {
		t.Fatal("child path under scalar value should be rejected")
	}

	parent := new(staticInitNode)
	if !parent.add([]staticInitPathElem{{index: 0}}, c) {
		t.Fatal("first child value should be accepted")
	}
	if parent.add(nil, c) {
		t.Fatal("root value after child path should be rejected")
	}
}

func TestStaticInitHelperBuildFailuresAndZeroes(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	pkg := prog.NewPackage("staticinit", "staticinit")
	ctx := &context{prog: prog, pkg: pkg}

	if _, ok := ctx.buildStaticInitExpr(types.Typ[types.Int], nil); !ok {
		t.Fatal("nil node should build a zero initializer")
	}
	if _, ok := ctx.staticConstExpr(ssa.NewConst(nil, types.Typ[types.Int]), prog.Int()); !ok {
		t.Fatal("nil const should build a zero initializer")
	}

	c := ssa.NewConst(constant.MakeInt64(1), types.Typ[types.Int])
	scalarWithChild := &staticInitNode{children: map[int]*staticInitNode{
		0: {value: c},
	}}
	if _, ok := ctx.buildStaticInitExpr(types.Typ[types.Int], scalarWithChild); ok {
		t.Fatal("scalar initializer with children should be rejected")
	}

	arrayWithOutOfRangeChild := &staticInitNode{children: map[int]*staticInitNode{
		1: {value: c},
	}}
	if _, ok := ctx.buildStaticInitExpr(types.NewArray(types.Typ[types.Int], 1), arrayWithOutOfRangeChild); ok {
		t.Fatal("array initializer with out-of-range child should be rejected")
	}
}

func TestStaticInitCollectEarlyExitsAndFilters(t *testing.T) {
	noInit := buildSSAPackage(t, `package foo
const A = 1
`)
	if initFn := noInit.Func("init"); initFn != nil {
		initFn.Synthetic = ""
	}
	(&context{}).collectStaticGlobalInits(noInit)

	skipped := buildSSAPackage(t, `package foo
var Skip = 1
func Use() int { return Skip }
`)
	ctx := &context{skips: map[string]none{"Skip": {}}}
	ctx.collectStaticGlobalInits(skipped)
	if ctx.staticGlobalInits != nil {
		t.Fatalf("skipped global should not produce static initializers: %+v", ctx.staticGlobalInits)
	}

	cgoFuncPtr := buildSSAPackage(t, `package foo
var __cgo_callback = 1
func Use() int { return __cgo_callback }
`)
	ctx = &context{}
	ctx.collectStaticGlobalInits(cgoFuncPtr)
	if ctx.staticGlobalInits != nil {
		t.Fatalf("__cgo function pointer globals should be ignored: %+v", ctx.staticGlobalInits)
	}

	nonGlobalStore := buildSSAPackage(t, `package foo
var G int
`)
	initFn := nonGlobalStore.Func("init")
	if initFn == nil || len(initFn.Blocks) == 0 {
		t.Fatal("expected package initializer for nonGlobalStore")
	}
	initFn.Blocks[0].Instrs = append([]ssa.Instruction{
		&ssa.Store{
			Addr: ssa.NewConst(constant.MakeInt64(0), types.Typ[types.Int]),
			Val:  ssa.NewConst(constant.MakeInt64(1), types.Typ[types.Int]),
		},
	}, initFn.Blocks[0].Instrs...)
	ctx = &context{prog: ssatest.NewProgram(t, nil)}
	ctx.collectStaticGlobalInits(nonGlobalStore)
}

func TestStaticInitHelperRejectsNestedUnsupportedPaths(t *testing.T) {
	badBase := ssa.NewConst(constant.MakeInt64(0), types.Typ[types.Int])
	if _, ok := staticInitStorePath(&ssa.FieldAddr{X: badBase, Field: 0}); ok {
		t.Fatal("field path with unsupported base should be rejected")
	}
	if _, ok := staticInitStorePath(&ssa.IndexAddr{
		X:     badBase,
		Index: ssa.NewConst(constant.MakeInt64(0), types.Typ[types.Int]),
	}); ok {
		t.Fatal("index path with unsupported base should be rejected")
	}
}

func TestStaticInitHelperBuildAdditionalFailures(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	pkg := prog.NewPackage("staticinit", "staticinit")
	ctx := &context{prog: prog, pkg: pkg}
	c := ssa.NewConst(constant.MakeInt64(1), types.Typ[types.Int])

	if _, ok := ctx.buildStaticGlobalInit(&ssa.Global{}, []staticInitStore{{value: c}}); ok {
		t.Fatal("global with non-pointer type should be rejected")
	}

	ssapkg := buildSSAPackage(t, `package foo
var G int
`)
	global, ok := ssapkg.Members["G"].(*ssa.Global)
	if !ok {
		t.Fatalf("missing G global: %T", ssapkg.Members["G"])
	}
	if _, ok := ctx.buildStaticGlobalInit(global, []staticInitStore{
		{value: c},
		{value: c},
	}); ok {
		t.Fatal("conflicting root stores should be rejected")
	}

	scalarWithChild := &staticInitNode{children: map[int]*staticInitNode{
		0: {value: c},
	}}
	structType := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "A", types.Typ[types.Int], false),
	}, nil)
	if _, ok := ctx.buildStaticInitExpr(structType, &staticInitNode{
		children: map[int]*staticInitNode{0: scalarWithChild},
	}); ok {
		t.Fatal("struct initializer with invalid field child should be rejected")
	}
	if _, ok := ctx.buildStaticInitExpr(structType, &staticInitNode{
		children: map[int]*staticInitNode{1: {value: c}},
	}); ok {
		t.Fatal("struct initializer with out-of-range child should be rejected")
	}
	if _, ok := ctx.buildStaticInitExpr(types.NewArray(types.Typ[types.Int], 1), &staticInitNode{
		children: map[int]*staticInitNode{0: scalarWithChild},
	}); ok {
		t.Fatal("array initializer with invalid element child should be rejected")
	}
	if _, ok := ctx.buildStaticInitExpr(types.NewPointer(types.Typ[types.Int]), new(staticInitNode)); !ok {
		t.Fatal("empty unsupported scalar node should build a zero initializer")
	}
}

func TestStaticConstExprRejectsUnsupportedConstants(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	pkg := prog.NewPackage("staticinit", "staticinit")
	ctx := &context{prog: prog, pkg: pkg}

	if _, ok := ctx.staticConstExpr(&ssa.Const{}, prog.Int()); !ok {
		t.Fatal("zero ssa.Const should build a zero initializer")
	}
	if _, ok := ctx.staticConstExpr(ssa.NewConst(constant.MakeInt64(1), types.Typ[types.Int]), prog.Pointer(prog.Int())); ok {
		t.Fatal("const for non-basic LLVM type should be rejected")
	}
	if _, ok := ctx.staticConstExpr(ssa.NewConst(constant.MakeFloat64(1.25), types.Typ[types.Float64]), prog.Int()); ok {
		t.Fatal("inexact signed integer constant should be rejected")
	}
	if _, ok := ctx.staticConstExpr(ssa.NewConst(constant.MakeInt64(-1), types.Typ[types.Int]), prog.Uint()); ok {
		t.Fatal("negative unsigned integer constant should be rejected")
	}
	if _, ok := ctx.staticConstExpr(
		ssa.NewConst(constant.MakeInt64(0), types.Typ[types.Int]),
		ctx.type_(types.Typ[types.UnsafePointer], llssa.InGo),
	); ok {
		t.Fatal("unsupported unsafe.Pointer constants should be rejected")
	}
}

func TestRewriteSkipsNonConstStores(t *testing.T) {
	const src = `package rewritepkg
import "strings"
var VarInit = strings.ToUpper("original_value")
var VarPlain string
func Use() string { return VarInit + VarPlain }
`
	ir := compileWithRewrites(t, src, map[string]string{
		"VarInit":  "rewrite_init",
		"VarPlain": "rewrite_plain",
	})
	if !strings.Contains(ir, `c"rewrite_init"`) {
		t.Fatalf("expected rewrite_init constant to remain:\n%s", ir)
	}
	if !strings.Contains(ir, "strings.ToUpper") {
		t.Fatalf("expected call to strings.ToUpper in IR:\n%s", ir)
	}
}

func TestRewriteValueNoDot(t *testing.T) {
	ctx := &context{rewrites: map[string]string{"VarInit": "rewrite_init"}}
	if _, ok := ctx.rewriteValue("VarInit"); ok {
		t.Fatalf("rewriteValue should skip names without package prefix")
	}
	if _, ok := ctx.rewriteValue("pkg."); ok {
		t.Fatalf("rewriteValue should skip trailing dot names")
	}
}

func TestIsStringPtrTypeDefault(t *testing.T) {
	ctx := &context{}
	if ctx.isStringPtrType(types.NewPointer(types.Typ[types.Int])) {
		t.Fatalf("expected non-string pointer to return false")
	}
}

func TestIsStringPtrTypeBranches(t *testing.T) {
	ctx := &context{}
	if ctx.isStringPtrType(types.NewSlice(types.Typ[types.String])) {
		t.Fatalf("slice should trigger default branch and return false")
	}
	if ctx.isStringPtrType(nil) {
		t.Fatalf("nil type should return false")
	}
	if !ctx.isStringPtrType(types.NewPointer(types.Typ[types.String])) {
		t.Fatalf("*string should return true")
	}
}

func TestRewriteIgnoredInNonInitStore(t *testing.T) {
	const src = `package rewritepkg
var VarInit = "original_value"
func Override() { VarInit = "override_value" }
`
	ir := compileWithRewrites(t, src, map[string]string{"VarInit": "rewrite_init"})
	if !strings.Contains(ir, `c"override_value"`) {
		t.Fatalf("override store should retain original literal:\n%s", ir)
	}
	if !strings.Contains(ir, `c"rewrite_init"`) {
		t.Fatalf("global initializer should still be rewritten:\n%s", ir)
	}
}

func TestRewriteMissingEntry(t *testing.T) {
	const src = `package rewritepkg
var VarInit = "original_value"
var VarOther = "other_value"
`
	ir := compileWithRewrites(t, src, map[string]string{"VarInit": "rewrite_init"})
	if !strings.Contains(ir, `c"other_value"`) {
		t.Fatalf("VarOther should keep original initializer:\n%s", ir)
	}
	if !strings.Contains(ir, `c"rewrite_init"`) {
		t.Fatalf("VarInit should still be rewritten:\n%s", ir)
	}
}

func TestRewriteIgnoresNonStringVar(t *testing.T) {
	const src = `package rewritepkg
type wrapper struct{ v int }
var VarStruct = wrapper{v: 1}
`
	ir := compileWithRewrites(t, src, map[string]string{"VarStruct": "rewrite_struct"})
	if strings.Contains(ir, `c"rewrite_struct"`) {
		t.Fatalf("non-string variables must not be rewritten:\n%s", ir)
	}
}

func TestRewriteIgnoresStringAlias(t *testing.T) {
	const src = `package rewritepkg
type T string
var VarAlias T = "original_value"
`
	ir := compileWithRewrites(t, src, map[string]string{"VarAlias": "rewrite_alias"})
	if strings.Contains(ir, `c"rewrite_alias"`) {
		t.Fatalf("string alias types must not be rewritten:\n%s", ir)
	}
	if !strings.Contains(ir, `c"original_value"`) {
		t.Fatalf("original value should remain for alias type:\n%s", ir)
	}
}

func buildSSAPackageWithFiles(t *testing.T, src string) (*ssa.Package, []*ast.File, *gpackages.Importer) {
	t.Helper()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "foo.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	files := []*ast.File{f}
	pkg := types.NewPackage("foo", "foo")
	imp := gpackages.NewImporter(fset)
	mode := ssa.SanityCheckFunctions | ssa.InstantiateGenerics
	ssapkg, _, err := ssautil.BuildPackage(&types.Config{Importer: imp}, fset, pkg, files, mode)
	if err != nil {
		t.Fatal(err)
	}
	return ssapkg, files, imp
}

func findStaticCallByName(t *testing.T, fn *ssa.Function, callee string) *ssa.Call {
	t.Helper()
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			call, ok := instr.(*ssa.Call)
			if !ok {
				continue
			}
			target := call.Common().StaticCallee()
			if target != nil && target.Name() == callee {
				return call
			}
		}
	}
	t.Fatalf("call to %q not found in %s", callee, fn.Name())
	return nil
}

func lastNonRecoverReturn(t *testing.T, fn *ssa.Function) *ssa.Return {
	t.Helper()
	for i := len(fn.Blocks) - 1; i >= 0; i-- {
		block := fn.Blocks[i]
		if block == fn.Recover {
			continue
		}
		for j := len(block.Instrs) - 1; j >= 0; j-- {
			if ret, ok := block.Instrs[j].(*ssa.Return); ok {
				return ret
			}
		}
	}
	t.Fatalf("return not found in %s", fn.Name())
	return nil
}

func TestRangeFuncDeferAnalysisHelpers(t *testing.T) {
	const src = `package foo

func seq(yield func(int) bool) {
	_ = yield(1)
	_ = yield(2)
}

func withRangeDefer() {
	for v := range seq {
		defer func() { println(v) }()
	}
}

func withRangeNoDefer() {
	for range seq {
	}
}

func directDefer() {
	defer println(1)
}
`

	ssapkg := buildSSAPackage(t, src)
	ctx := &context{}

	withRangeDefer := ssapkg.Func("withRangeDefer")
	if withRangeDefer == nil || len(withRangeDefer.AnonFuncs) != 1 {
		t.Fatalf("expected one yield closure for withRangeDefer, got %v", withRangeDefer)
	}
	yieldFn := withRangeDefer.AnonFuncs[0]
	if yieldFn.Synthetic != "range-over-func yield" {
		t.Fatalf("unexpected synthetic kind: %q", yieldFn.Synthetic)
	}
	if !ctx.functionHasExplicitStackDefer(yieldFn) {
		t.Fatal("yield closure should have explicit defer stack")
	}
	if ctx.returnNeedsImplicitRunDefers(lastNonRecoverReturn(t, yieldFn)) {
		t.Fatal("synthetic yield closure should not insert implicit RunDefers")
	}
	if !ctx.functionHasExplicitStackDeferInAnon(withRangeDefer) {
		t.Fatal("outer function should detect explicit defer stack in anon funcs")
	}
	if !ctx.functionHasExplicitStackDefer(yieldFn) {
		t.Fatal("cached explicit defer stack result should remain true")
	}
	if !ctx.functionHasExplicitStackDeferInAnon(withRangeDefer) {
		t.Fatal("cached anon explicit defer stack result should remain true")
	}

	withRangeCall := findStaticCallByName(t, withRangeDefer, "seq")
	if !ctx.rangeFuncCallNeedsDeferDrain(withRangeCall.Common()) {
		t.Fatal("range-over-func call with defering yield should require drain")
	}
	withRangeRet := lastNonRecoverReturn(t, withRangeDefer)
	if !ctx.returnNeedsImplicitRunDefers(withRangeRet) {
		t.Fatal("outer function should insert implicit RunDefers")
	}

	withRangeNoDefer := ssapkg.Func("withRangeNoDefer")
	if withRangeNoDefer == nil {
		t.Fatal("missing withRangeNoDefer")
	}
	if ctx.functionHasExplicitStackDeferInAnon(withRangeNoDefer) {
		t.Fatal("range-over-func without defer should not report explicit stack defer")
	}
	noDeferCall := findStaticCallByName(t, withRangeNoDefer, "seq")
	if ctx.rangeFuncCallNeedsDeferDrain(noDeferCall.Common()) {
		t.Fatal("range-over-func call without defering yield should not require drain")
	}
	noDeferRet := lastNonRecoverReturn(t, withRangeNoDefer)
	if ctx.returnNeedsImplicitRunDefers(noDeferRet) {
		t.Fatal("range-over-func without explicit defer stack should not add RunDefers")
	}

	directDefer := ssapkg.Func("directDefer")
	if directDefer == nil {
		t.Fatal("missing directDefer")
	}
	directRet := lastNonRecoverReturn(t, directDefer)
	if !previousNonDebugInstrIsRunDefers(directRet) {
		t.Fatal("direct defer return should already be preceded by RunDefers")
	}
	if ctx.returnNeedsImplicitRunDefers(directRet) {
		t.Fatal("existing RunDefers should suppress implicit insertion")
	}
}

func TestEmitDoWithExplicitDeferStack(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	pkg := prog.NewPackage("foo", "foo")

	callee := pkg.NewFunc("callee", llssa.NoArgsNoRet, llssa.InGo)
	cb := callee.MakeBody(1)
	cb.Return()
	cb.EndBuild()

	owner := pkg.NewFunc("owner", llssa.NoArgsNoRet, llssa.InGo)
	b := owner.MakeBody(1)
	owner.SetRecover(owner.MakeBlock())
	stack := b.BuiltinCall("ssa:deferstack")
	b.Return()
	b.SetBlockEx(owner.Block(0), llssa.BeforeLast, true)

	ctx := &context{}
	ctx.emitDo(b, llssa.DeferInLoop, &explicitDeferStack{stack: stack, owner: owner}, false, callee.Expr, llssa.Builder.Call)
	ctx.emitDo(b, llssa.DeferAlways, nil, false, callee.Expr, llssa.Builder.Call)
	b.DeferStackDrain()
	b.RunDefers()
	b.Return()
	b.EndBuild()

	ir := pkg.String()
	if !strings.Contains(ir, "FreeDeferNode") {
		t.Fatalf("expected explicit defer stack drain in IR, got:\n%s", ir)
	}
	if !strings.Contains(ir, "sigsetjmp") && !strings.Contains(ir, "setjmp") {
		t.Fatalf("expected defer stack setup in IR, got:\n%s", ir)
	}
}

func TestCompileRangeFuncDeferModule(t *testing.T) {
	_, m := mustCompileLLPkgFromSrc(t, `
package foo

func seq(yield func(int) bool) {
	_ = yield(1)
	_ = yield(2)
}

func f() {
	for v := range seq {
		defer func() { _ = v }()
	}
}
`)

	ir := m.String()
	if !strings.Contains(ir, "FreeDeferNode") {
		t.Fatalf("expected rangefunc defer node cleanup in module, got:\n%s", ir)
	}
	if !strings.Contains(ir, "sigsetjmp") && !strings.Contains(ir, "setjmp") {
		t.Fatalf("expected rangefunc defer stack setup in module, got:\n%s", ir)
	}
}

func TestDeferStackOwnerUsesEnclosingSourceFunction(t *testing.T) {
	const src = `package foo

func seq(yield func(int) bool) { _ = yield(1) }

func f() {
	for v := range seq {
		defer func() { println(v) }()
	}
}
`

	ssapkg := buildSSAPackage(t, src)
	root := ssapkg.Func("f")
	if root == nil || len(root.AnonFuncs) != 1 {
		t.Fatalf("expected one yield closure for f, got %v", root)
	}
	yieldFn := root.AnonFuncs[0]

	prog := llssa.NewProgram(nil)
	pkg := prog.NewPackage("foo", "foo")
	owner := pkg.NewFunc("f", llssa.NoArgsNoRet, llssa.InGo)

	ctx := &context{funcs: map[*ssa.Function]llssa.Function{root: owner}}
	if got := ctx.deferStackOwner(root); got != owner {
		t.Fatalf("deferStackOwner(root) = %v, want %v", got, owner)
	}
	if got := ctx.deferStackOwner(yieldFn); got != owner {
		t.Fatalf("deferStackOwner(yield) = %v, want %v", got, owner)
	}
}

func TestRangeFuncDeferHelperNilAndCachePaths(t *testing.T) {
	ctx := &context{}
	if ctx.functionHasExplicitStackDefer(nil) {
		t.Fatal("nil function should not report explicit stack defer")
	}
	if ctx.functionHasExplicitStackDeferInAnon(nil) {
		t.Fatal("nil function should not report anon explicit stack defer")
	}

	const src = `package foo

func plain() {}

func wrap() {
	plain()
}
`
	ssapkg := buildSSAPackage(t, src)
	plain := ssapkg.Func("plain")
	wrap := ssapkg.Func("wrap")
	if plain == nil || wrap == nil {
		t.Fatal("missing test functions")
	}
	if ctx.functionHasExplicitStackDefer(plain) {
		t.Fatal("plain should not report explicit stack defer")
	}
	if ctx.functionHasExplicitStackDeferInAnon(wrap) {
		t.Fatal("wrap should not report anon explicit stack defer")
	}
	if ctx.functionHasExplicitStackDeferSeen(plain, map[*ssa.Function]bool{plain: true}) {
		t.Fatal("seen function should short-circuit to false")
	}
	if ctx.functionHasExplicitStackDeferInAnonSeen(wrap, map[*ssa.Function]bool{wrap: true}) {
		t.Fatal("seen function should short-circuit anon scan")
	}
	call := findStaticCallByName(t, wrap, "plain")
	if ctx.rangeFuncCallNeedsDeferDrain(call.Common()) {
		t.Fatal("plain call should not require defer drain")
	}
	if ctx.deferStackOwner(nil) != nil {
		t.Fatal("nil deferStackOwner should return nil")
	}

	if previousNonDebugInstrIsRunDefers(&ssa.Return{}) {
		t.Fatal("return without block should not see RunDefers")
	}
}

func TestEmitDoWithoutExplicitDeferStack(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	pkg := prog.NewPackage("foo", "foo")
	sig := types.NewSignatureType(nil, nil, nil, nil,
		types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.Int])),
		false)

	callee := pkg.NewFunc("callee", sig, llssa.InGo)
	cb := callee.MakeBody(1)
	cb.Return(prog.IntVal(7, prog.Int()))
	cb.EndBuild()

	fn := pkg.NewFunc("main", llssa.NoArgsNoRet, llssa.InGo)
	b := fn.MakeBody(1)

	ctx := &context{}
	got := ctx.emitDo(b, llssa.Call, nil, false, callee.Expr, llssa.Builder.Call)
	if got.IsNil() {
		t.Fatal("emitDo without explicit defer stack should return direct call result")
	}
	if got.Type == nil || got.Type.RawType() != types.Typ[types.Int] {
		t.Fatalf("emitDo returned unexpected type: %v", got.Type)
	}
}

func TestNestedRangeFuncDeferAnalysisCombinations(t *testing.T) {
	const src = `package foo

func outerSeq(yield func(int) bool) {
	_ = yield(1)
}

func innerSeq(base int) func(func(int) bool) {
	return func(yield func(int) bool) {
		_ = yield(base*10 + 1)
		_ = yield(base*10 + 2)
	}
}

func nestedReturn() {
	for i := range outerSeq {
		defer func() { println(i) }()
		for j := range innerSeq(i) {
			defer func() { println(j) }()
			return
		}
	}
}
`

	ssapkg := buildSSAPackage(t, src)
	ctx := &context{}

	root := ssapkg.Func("nestedReturn")
	if root == nil || len(root.AnonFuncs) == 0 {
		t.Fatalf("expected nestedReturn with yield closures, got %v", root)
	}

	outerCall := findStaticCallByName(t, root, "outerSeq")
	if !ctx.rangeFuncCallNeedsDeferDrain(outerCall.Common()) {
		t.Fatal("outer range-over-func call should require defer drain")
	}
	if !ctx.returnNeedsImplicitRunDefers(lastNonRecoverReturn(t, root)) {
		t.Fatal("nested return should still require implicit RunDefers")
	}

	var outerYield *ssa.Function
	for _, child := range root.AnonFuncs {
		if child.Synthetic == "range-over-func yield" {
			outerYield = child
			break
		}
	}
	if outerYield == nil {
		t.Fatal("missing outer yield closure")
	}
	if !ctx.functionHasExplicitStackDefer(outerYield) {
		t.Fatal("outer yield closure should report explicit stack defer")
	}
	if ctx.returnNeedsImplicitRunDefers(lastNonRecoverReturn(t, outerYield)) {
		t.Fatal("synthetic outer yield closure must not insert implicit RunDefers")
	}
}

func TestDeferStackOwnerCompilesMissingOwner(t *testing.T) {
	ssapkg, fset, files := buildGoSSAPkg(t, `package foo

func f() {}
`)
	prog := newLLSSAProg(t)
	pkg := prog.NewPackage("foo", "foo")
	ctx := &context{
		prog:   prog,
		pkg:    pkg,
		fset:   fset,
		goProg: ssapkg.Prog,
		goTyps: ssapkg.Pkg,
		goPkg:  ssapkg,
		funcs:  map[*ssa.Function]llssa.Function{},
	}
	_ = files

	root := ssapkg.Func("f")
	if root == nil {
		t.Fatal("missing source function f")
	}
	owner := ctx.deferStackOwner(root)
	if owner == nil {
		t.Fatal("deferStackOwner should lazily compile missing owner")
	}
	if ctx.funcs[root] != owner {
		t.Fatal("compiled owner should be cached")
	}
}
