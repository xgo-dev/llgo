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
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/ssa/ssatest"
	"github.com/xgo-dev/llvm"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func init() {
	llssa.Initialize(llssa.InitAll | llssa.InitNative)
}

func compileWithRewrites(t *testing.T, src string, rewrites map[string]string) string {
	return compileWithRewritesTarget(t, src, rewrites, nil)
}

func compileWithRewritesTarget(t *testing.T, src string, rewrites map[string]string, target *llssa.Target) string {
	return compileWithRewritesModeTarget(t, src, rewrites,
		ssa.SanityCheckFunctions|ssa.InstantiateGenerics, target)
}

func compileWithRewritesMode(t *testing.T, src string, rewrites map[string]string, mode ssa.BuilderMode) string {
	return compileWithRewritesModeTarget(t, src, rewrites, mode, nil)
}

func compileWithRewritesModeTarget(t *testing.T, src string, rewrites map[string]string, mode ssa.BuilderMode, target *llssa.Target) string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "rewrite.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	importer := gpackages.NewImporter(fset)
	pkg, _, err := ssautil.BuildPackage(&types.Config{Importer: importer}, fset,
		types.NewPackage(file.Name.Name, file.Name.Name), []*ast.File{file}, mode)
	if err != nil {
		t.Fatalf("build package failed: %v", err)
	}
	prog := ssatest.NewProgramEx(t, target, importer)
	goarch := runtime.GOARCH
	if target != nil && target.GOARCH != "" {
		goarch = target.GOARCH
	}
	prog.TypeSizes(types.SizesFor("gc", goarch))
	ret, _, err := NewPackageEx(prog, nil, rewrites, pkg, []*ast.File{file})
	if err != nil {
		t.Fatalf("NewPackageEx failed: %v", err)
	}
	return ret.String()
}

func TestClosureEnvIntrinsicRequiresEnvBearingEntry(t *testing.T) {
	valid := `package closureenv

import "unsafe"

//go:linkname closureEnv llgo.closureEnv
func closureEnv() unsafe.Pointer

DIRECTIVE
func use() unsafe.Pointer { return closureEnv() }
`
	for _, spelling := range []string{"//llgo:env", "// llgo:env"} {
		t.Run(spelling, func(t *testing.T) {
			ir := compileWithRewrites(t, strings.Replace(valid, "DIRECTIVE", spelling, 1), nil)
			if !strings.Contains(ir, `define ptr @closureenv.use(ptr `) ||
				!strings.Contains(ir, `ret ptr %0`) ||
				(!strings.Contains(ir, `ptr nest %0`) && !strings.Contains(ir, `ptr swiftself %0`)) {
				t.Fatalf("closureEnv intrinsic did not return the physical environment:\n%s", ir)
			}
		})
	}

	for _, test := range []struct {
		name string
		src  string
	}{
		{
			name: "plain entry",
			src: `package closureenv
import "unsafe"
//go:linkname closureEnv llgo.closureEnv
func closureEnv() unsafe.Pointer
func use() unsafe.Pointer { return closureEnv() }
`,
		},
		{
			name: "arguments",
			src: `package closureenv
import "unsafe"
//go:linkname closureEnv llgo.closureEnv
func closureEnv(int) unsafe.Pointer
//llgo:env
func use() unsafe.Pointer { return closureEnv(1) }
`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			mustPanic(t, "invalid closureEnv intrinsic", func() {
				compileWithRewrites(t, test.src, nil)
			})
		})
	}
}

func TestClosureEnvRejectsConflictingEntryABI(t *testing.T) {
	const src = `package closureenv

import _ "unsafe"

//go:linkname plain closureenv.entry
func plain() {}

//go:linkname withEnv closureenv.entry
//llgo:env
func withEnv() {}
`
	mustPanic(t, "conflicting closure environment ABI", func() {
		compileWithRewrites(t, src, nil)
	})
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
	if !strings.Contains(ir, `[%"github.com/xgo-dev/llgo/runtime/internal/runtime.String" { ptr @0, i64 9 }, %"github.com/xgo-dev/llgo/runtime/internal/runtime.String" { ptr @1, i64 12 }]`) {
		t.Fatalf("unexpected MethodNames.Value initializer:\n%s", ir)
	}
	if !strings.Contains(ir, `%staticinit.Nested { [2 x %"github.com/xgo-dev/llgo/runtime/internal/runtime.String"] [%"github.com/xgo-dev/llgo/runtime/internal/runtime.String" { ptr @2, i64 8 }, %"github.com/xgo-dev/llgo/runtime/internal/runtime.String" { ptr @3, i64 11 }] }`) {
		t.Fatalf("unexpected MethodNames.Nested initializer:\n%s", ir)
	}
	for _, want := range []string{`c"KeepValue"`, `c"KeepValueAlt"`, `c"KeepType"`, `c"KeepTypeAlt"`} {
		if !strings.Contains(ir, want) {
			t.Fatalf("missing %s in IR:\n%s", want, ir)
		}
	}
	assertNoStoreToGlobal(t, ir, "@staticinit.MethodNames")
}

func TestStaticGlobalBlankFieldInit(t *testing.T) {
	const src = `package staticinit

type Nested struct {
	Left int
	_ int
	Right int
}

type Outer struct {
	Left int
	_ Nested
	Right int
}

var Flat = Nested{1, 2, 3}
var Deep = Outer{4, Nested{5, 6, 7}, 8}

func Use() int {
	return Flat.Left + Flat.Right + Deep.Left + Deep.Right
}
`
	ir := compileWithRewrites(t, src, nil)
	for _, want := range []string{
		"@staticinit.Flat = global %staticinit.Nested { i64 1, i64 0, i64 3 }",
		"@staticinit.Deep = global %staticinit.Outer { i64 4, %staticinit.Nested zeroinitializer, i64 8 }",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("blank field was not zeroed in static initializer %q:\n%s", want, ir)
		}
	}
}

func TestBlankFieldStoreRejectsInvalidAddresses(t *testing.T) {
	for name, addr := range map[string]ssa.Value{
		"nil":           nil,
		"invalid field": &ssa.FieldAddr{},
		"invalid index": &ssa.IndexAddr{},
	} {
		t.Run(name, func(t *testing.T) {
			if isBlankFieldStore(addr) {
				t.Fatal("invalid address was classified as a blank field store")
			}
		})
	}
}

func TestBlankFieldStoreClassification(t *testing.T) {
	const src = `package blankstore

type Leaf struct {
	Value int
}

type Outer struct {
	_ Leaf
	Keep Leaf
	_ [2]Leaf
	_ []Leaf
}

func next() int { return 1 }

var Value = Outer{
	Leaf{next()},
	Leaf{next()},
	[2]Leaf{{next()}, {next()}},
	[]Leaf{{next()}},
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "blankstore.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	importer := gpackages.NewImporter(fset)
	pkg, _, err := ssautil.BuildPackage(
		&types.Config{Importer: importer},
		fset,
		types.NewPackage("blankstore", "blankstore"),
		[]*ast.File{file},
		ssa.SanityCheckFunctions,
	)
	if err != nil {
		t.Fatal(err)
	}

	addressPath := func(addr ssa.Value) (fields []string, indexed bool) {
		for {
			switch current := addr.(type) {
			case *ssa.FieldAddr:
				_, st, ok := fieldAddrStruct(current)
				if !ok {
					return fields, indexed
				}
				fields = append(fields, st.Field(current.Field).Name())
				addr = current.X
			case *ssa.IndexAddr:
				indexed = true
				addr = current.X
			default:
				return fields, indexed
			}
		}
	}

	var (
		blankSliceField                    *ssa.FieldAddr
		sawDirectBlank, sawNonBlankSibling bool
	)
	initFn := pkg.Func("init")
	for _, block := range initFn.Blocks {
		for _, instr := range block.Instrs {
			if field, ok := instr.(*ssa.FieldAddr); ok {
				_, st, valid := fieldAddrStruct(field)
				if valid && st.Field(field.Field).Name() == "_" {
					if _, ok := st.Field(field.Field).Type().Underlying().(*types.Slice); ok {
						blankSliceField = field
					}
				}
			}
			store, ok := instr.(*ssa.Store)
			if !ok {
				continue
			}
			fields, indexed := addressPath(store.Addr)
			if len(fields) == 0 {
				continue
			}
			want := false
			for _, field := range fields {
				want = want || field == "_"
			}
			if got := isBlankFieldStore(store.Addr); got != want {
				t.Fatalf("isBlankFieldStore(%s) = %v, want %v", store.Addr, got, want)
			}
			switch {
			case len(fields) == 1 && fields[0] == "_":
				sawDirectBlank = true
			case !want && !indexed:
				sawNonBlankSibling = true
			}
		}
	}
	if !sawDirectBlank || !sawNonBlankSibling {
		t.Fatalf(
			"missing SSA classification coverage: direct=%v sibling=%v",
			sawDirectBlank,
			sawNonBlankSibling,
		)
	}
	if blankSliceField == nil {
		t.Fatal("missing blank slice field address")
	}
	// This shape is not currently emitted by go/ssa: it defensively verifies
	// that a future IndexAddr form cannot cross a blank slice header.
	if isBlankFieldStore(&ssa.IndexAddr{X: blankSliceField}) {
		t.Fatal("slice backing-array address crossed the blank slice header")
	}
}

func TestStaticGlobalSliceLiteralInit(t *testing.T) {
	const src = `package staticinit

type callbackType string

var CallbackTypes = []callbackType{
	"BeforeCreate",
	"AfterCreate",
}

func Use() callbackType { return CallbackTypes[1] }
`
	ir := compileWithRewrites(t, src, nil)
	for _, want := range []string{
		`@"staticinit.CallbackTypes$data" = global [2 x %"github.com/xgo-dev/llgo/runtime/internal/runtime.String"]`,
		`@staticinit.CallbackTypes = global %"github.com/xgo-dev/llgo/runtime/internal/runtime.Slice" { ptr @"staticinit.CallbackTypes$data", i64 2, i64 2 }`,
		`c"BeforeCreate"`,
		`c"AfterCreate"`,
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("missing static slice initializer %q in IR:\n%s", want, ir)
		}
	}
	assertNoStoreToGlobal(t, ir, "@staticinit.CallbackTypes")
	if strings.Contains(ir, "runtime.AllocZ") {
		t.Fatalf("static slice initializer still allocates at runtime:\n%s", ir)
	}
}

func TestStaticGlobalSliceLiteralInitWithDebugRefs(t *testing.T) {
	const src = `package staticinit

var CallbackTypes = []string{"BeforeCreate", "AfterCreate"}

func Use() string { return CallbackTypes[1] }
`
	ir := compileWithRewritesMode(t, src, nil,
		ssa.SanityCheckFunctions|ssa.InstantiateGenerics|ssa.GlobalDebug)
	for _, want := range []string{
		`@"staticinit.CallbackTypes$data" = global [2 x %"github.com/xgo-dev/llgo/runtime/internal/runtime.String"]`,
		`@staticinit.CallbackTypes = global %"github.com/xgo-dev/llgo/runtime/internal/runtime.Slice" { ptr @"staticinit.CallbackTypes$data", i64 2, i64 2 }`,
		`c"BeforeCreate"`,
		`c"AfterCreate"`,
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("missing static slice initializer %q with debug refs:\n%s", want, ir)
		}
	}
	assertNoStoreToGlobal(t, ir, "@staticinit.CallbackTypes")
	if strings.Contains(ir, "runtime.AllocZ") {
		t.Fatalf("static slice initializer allocates at runtime with debug refs:\n%s", ir)
	}
}

func TestStaticGlobalStructSliceLiteralInit(t *testing.T) {
	const src = `package staticinit

type Info struct {
	Name    string
	Package string
	Changed int
}

var All = []Info{
	{"godebug1", "runtime", 1},
	{"godebug2", "internal/poll", 2},
}

func Use() Info { return All[0] }
`
	ir := compileWithRewrites(t, src, nil)
	for _, want := range []string{
		`@"staticinit.All$data" = global [2 x %staticinit.Info]`,
		`@staticinit.All = global %"github.com/xgo-dev/llgo/runtime/internal/runtime.Slice" { ptr @"staticinit.All$data", i64 2, i64 2 }`,
		`c"godebug1"`,
		`c"runtime"`,
		`c"godebug2"`,
		`c"internal/poll"`,
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("missing static struct slice initializer %q in IR:\n%s", want, ir)
		}
	}
	assertNoStoreToGlobal(t, ir, "@staticinit.All")
	if strings.Contains(ir, "runtime.AllocZ") {
		t.Fatalf("static struct slice initializer still allocates at runtime:\n%s", ir)
	}
}

func TestStaticGlobalNestedStructSliceLiteralInit(t *testing.T) {
	const src = `package staticinit

type Point struct {
	X, Y int
}

type Entry struct {
	Points [2]Point
	Label  string
}

var Entries = []Entry{
	{Points: [2]Point{{X: 1, Y: 2}, {X: 3}}, Label: "first"},
	{},
}

func Use() int { return Entries[0].Points[1].X + len(Entries[0].Label) }
`
	ir := compileWithRewrites(t, src, nil)
	var dataInit string
	for _, line := range strings.Split(ir, "\n") {
		if strings.Contains(line, `@"staticinit.Entries$data" = global`) {
			dataInit = line
			break
		}
	}
	if dataInit == "" {
		t.Fatalf("missing nested struct slice backing global:\n%s", ir)
	}
	for _, want := range []string{"i64 1", "i64 2", "i64 3", "%staticinit.Entry zeroinitializer"} {
		if !strings.Contains(dataInit, want) {
			t.Fatalf("nested struct slice initializer missing %q:\n%s", want, ir)
		}
	}
	if !strings.Contains(ir, `c"first"`) {
		t.Fatalf("nested struct slice initializer missing string data:\n%s", ir)
	}
	assertNoStoreToGlobal(t, ir, "@staticinit.Entries")
	if strings.Contains(ir, "runtime.AllocZ") {
		t.Fatalf("nested struct slice initializer still allocates at runtime:\n%s", ir)
	}
}

func TestStaticGlobalNestedSliceLiteralInit(t *testing.T) {
	const src = `package staticinit

type EventSpec struct {
	Name     string
	Args     []string
	StackIDs []int
}

var specs = [2]EventSpec{
	{Name: "first", Args: []string{"time", "stack"}, StackIDs: []int{1}},
	{Name: "second", Args: []string{"id"}},
}

func Use() int { return len(specs[0].Args) + specs[0].StackIDs[0] }
`
	ir := compileWithRewrites(t, src, nil)
	for _, want := range []string{
		`@staticinit.specs = global [2 x %staticinit.EventSpec]`,
		`@"staticinit.specs$data$0$1" = global [2 x %"github.com/xgo-dev/llgo/runtime/internal/runtime.String"]`,
		`@"staticinit.specs$data$0$2" = global [1 x i64]`,
		`@"staticinit.specs$data$1$1" = global [1 x %"github.com/xgo-dev/llgo/runtime/internal/runtime.String"]`,
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("missing nested slice initializer %q in IR:\n%s", want, ir)
		}
	}
	if strings.Contains(ir, `@staticinit.specs = global [2 x %staticinit.EventSpec] zeroinitializer`) {
		t.Fatalf("nested slices left the root global zero-initialized:\n%s", ir)
	}
	assertNoStoreToGlobal(t, ir, "@staticinit.specs")
	if strings.Contains(ir, "runtime.AllocZ") {
		t.Fatalf("nested slice initializer still allocates at runtime:\n%s", ir)
	}
}

func TestStaticGlobalFunctionTableInit(t *testing.T) {
	const src = `package staticinit

type Handler struct {
	Name string
	Call func(int) int
}

func plusOne(v int) int { return v + 1 }
func timesTwo(v int) int { return v * 2 }

var Handlers = []*Handler{
	{Name: "plus", Call: plusOne},
	{Name: "times", Call: timesTwo},
}

func Use(v int) int { return Handlers[0].Call(v) }
`
	ir := compileWithRewrites(t, src, nil)
	for _, want := range []string{
		`@"staticinit.Handlers$data" = global [2 x ptr] [ptr @"staticinit.Handlers$data$0", ptr @"staticinit.Handlers$data$1"]`,
		`@"staticinit.Handlers$data$0" = global %staticinit.Handler`,
		`@"staticinit.Handlers$data$1" = global %staticinit.Handler`,
		`{ ptr @staticinit.plusOne, ptr null }`,
		`{ ptr @staticinit.timesTwo, ptr null }`,
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("missing static function table initializer %q in IR:\n%s", want, ir)
		}
	}
	assertNoStoreToGlobal(t, ir, "@staticinit.Handlers")
	if strings.Contains(ir, "runtime.AllocZ") {
		t.Fatalf("static function table still allocates at runtime:\n%s", ir)
	}
}

func TestStaticGlobalPointerArrayInit(t *testing.T) {
	const src = `package staticinit

var Values = []*[2]int{{1, 2}}

func Use() int { return Values[0][1] }
`
	ir := compileWithRewrites(t, src, nil)
	for _, want := range []string{
		`@"staticinit.Values$data" = global [1 x ptr] [ptr @"staticinit.Values$data$0"]`,
		`@"staticinit.Values$data$0" = global [2 x i64] [i64 1, i64 2]`,
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("missing static pointer-array initializer %q in IR:\n%s", want, ir)
		}
	}
	assertNoStoreToGlobal(t, ir, "@staticinit.Values")
}

func TestStaticGlobalExplicitEnvFunctionFallsBack(t *testing.T) {
	const src = `package staticinit

//llgo:env
func withEnv(v int) int { return v }

var Handlers = []func(int) int{withEnv}

func Use(v int) int { return Handlers[0](v) }
`
	ir := compileWithRewrites(t, src, nil)
	if strings.Contains(ir, `@"staticinit.Handlers$data"`) {
		t.Fatalf("explicit-env function unexpectedly used a static function table:\n%s", ir)
	}
	assertStoreToGlobal(t, ir, "@staticinit.Handlers")
}

func TestStaticGlobalZeroSizedPointerLiteralFallsBack(t *testing.T) {
	const src = `package foo

var Values = []*struct{}{{}}
`
	ssapkg := buildSSAPackage(t, src)
	global := ssapkg.Members["Values"].(*ssa.Global)
	initFn := ssapkg.Func("init")
	var globalStore *ssa.Store
	for _, block := range initFn.Blocks {
		for _, instr := range block.Instrs {
			if store, ok := instr.(*ssa.Store); ok && store.Addr == global {
				globalStore = store
			}
		}
	}
	if globalStore == nil {
		t.Fatal("store to Values not found")
	}
	if _, ok := staticSliceInitOf(globalStore); ok {
		t.Fatal("zero-sized pointer values unexpectedly accepted by static slice folding")
	}
}

func TestStaticGlobalInterfaceLiteralFallsBack(t *testing.T) {
	const src = `package foo

var Values = []any{1, "two"}
`
	ssapkg := buildSSAPackage(t, src)
	global := ssapkg.Members["Values"].(*ssa.Global)
	initFn := ssapkg.Func("init")
	var globalStore *ssa.Store
	for _, block := range initFn.Blocks {
		for _, instr := range block.Instrs {
			if store, ok := instr.(*ssa.Store); ok && store.Addr == global {
				globalStore = store
			}
		}
	}
	if globalStore == nil {
		t.Fatal("store to Values not found")
	}
	if _, ok := staticSliceInitOf(globalStore); ok {
		t.Fatal("interface values unexpectedly accepted by static slice folding")
	}
}

func TestStaticGlobalStructSliceDynamicFieldFallsBack(t *testing.T) {
	const src = `package staticinit

type Entry struct {
	Value int
}

func next() int { return 1 }

var Entries = []Entry{{Value: next()}}

func Use() int { return Entries[0].Value }
`
	ir := compileWithRewrites(t, src, nil)
	if strings.Contains(ir, `@"staticinit.Entries$data"`) {
		t.Fatalf("dynamic struct slice unexpectedly used static backing storage:\n%s", ir)
	}
	assertStoreToGlobal(t, ir, "@staticinit.Entries")
}

func TestStaticSliceInitRejectsExecutableReferrers(t *testing.T) {
	const src = `package foo

var Values, Other []int

func useSlice([]int) {}
func usePointer(*int) {}

func sliceUser() {
	backing := [2]int{1, 2}
	values := backing[:]
	Values = values
	useSlice(values)
}

func elementUser() {
	var backing [2]int
	elem := &backing[0]
	*elem = 1
	usePointer(elem)
	Values = backing[:]
}

func sharedBackingStores() {
	backing := [2]int{1, 2}
	Values = backing[:]
	Other = backing[:]
}

func boundedSlice() {
	backing := [2]int{1, 2}
	Values = backing[1:]
}
`
	ssapkg := buildSSAPackage(t, src)
	global := ssapkg.Members["Values"].(*ssa.Global)
	for _, name := range []string{"sliceUser", "elementUser", "sharedBackingStores", "boundedSlice"} {
		fn := ssapkg.Func(name)
		var globalStore *ssa.Store
		for _, block := range fn.Blocks {
			for _, instr := range block.Instrs {
				if store, ok := instr.(*ssa.Store); ok && store.Addr == global {
					globalStore = store
				}
			}
		}
		if globalStore == nil {
			t.Fatalf("%s: store to Values not found", name)
		}
		if _, ok := staticSliceInitOf(globalStore); ok {
			t.Fatalf("%s: static slice init accepted an executable referrer", name)
		}
	}
}

func TestCollectAllocStoresRejectsExtraLoad(t *testing.T) {
	const src = `package foo

type Point struct {
	X, Y int
}

var First, Second Point

func sharedLoad() {
	point := Point{X: 1, Y: 2}
	First = point
	Second = point
}
`
	ssapkg := buildSSAPackage(t, src)
	first := ssapkg.Members["First"].(*ssa.Global)
	fn := ssapkg.Func("sharedLoad")
	var resultStore *ssa.Store
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if store, ok := instr.(*ssa.Store); ok && store.Addr == first {
				resultStore = store
			}
		}
	}
	if resultStore == nil {
		t.Fatal("store to First not found")
	}
	resultLoad, ok := resultStore.Val.(*ssa.UnOp)
	if !ok || resultLoad.Op != token.MUL {
		t.Fatalf("store to First does not load a composite alloc: %T", resultStore.Val)
	}
	alloc, ok := resultLoad.X.(*ssa.Alloc)
	if !ok {
		t.Fatalf("First load does not read an alloc: %T", resultLoad.X)
	}
	var stores []staticInitStore
	var instrs []ssa.Instruction
	if collectAllocStores(alloc, resultLoad, resultStore, nil, &stores, &instrs, make(map[*ssa.Alloc]bool)) {
		t.Fatal("alloc with an additional load was accepted for static folding")
	}
}

func TestBuildStaticSliceInitRejectsMalformedStores(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	ctx := &context{prog: prog, pkg: prog.NewPackage("staticinit", "staticinit")}
	value := ssa.NewConst(constant.MakeInt64(1), types.Typ[types.Int])
	store := func(path ...int) staticInitStore {
		ret := staticInitStore{value: value}
		for _, index := range path {
			ret.path = append(ret.path, staticInitPathElem{index: index})
		}
		return ret
	}

	for _, test := range []struct {
		name   string
		stores []staticInitStore
	}{
		{name: "missing element index", stores: []staticInitStore{store()}},
		{name: "element index out of range", stores: []staticInitStore{store(1)}},
		{name: "duplicate element store", stores: []staticInitStore{store(0), store(0)}},
		{name: "path below scalar element", stores: []staticInitStore{store(0, 0)}},
	} {
		t.Run(test.name, func(t *testing.T) {
			init := &staticSliceInit{
				array:  types.NewArray(types.Typ[types.Int], 1),
				stores: test.stores,
			}
			if _, ok := ctx.buildStaticSliceInit(nil, init); ok {
				t.Fatal("malformed static slice stores were accepted")
			}
		})
	}
}

func TestStaticGlobalZeroSizedSliceLiteralFallsBack(t *testing.T) {
	const src = `package staticinit

var Values = []struct{}{{}, {}}

func Use() int { return len(Values) }
`
	ir := compileWithRewrites(t, src, nil)
	if strings.Contains(ir, "staticinit.Values$data") {
		t.Fatalf("zero-sized slice literal unexpectedly uses static backing storage:\n%s", ir)
	}
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

func TestStaticGlobalInitFoldsLargeByteArray(t *testing.T) {
	length := maxStaticInitArrayElements + 1
	src := fmt.Sprintf(`package staticinit

var Large = [%d]byte{%d: 1}

func Use() byte { return Large[%d] }
`, length, length-1, length-1)
	ir := compileWithRewrites(t, src, nil)
	want := fmt.Sprintf("@staticinit.Large = global [%d x i8] c\"", length)
	if !strings.Contains(ir, want) {
		t.Fatalf("large byte array should use a compact static initializer")
	}
	if strings.Contains(ir, fmt.Sprintf("@staticinit.Large = global [%d x i8] zeroinitializer", length)) {
		t.Fatal("large byte array still uses a zero initializer")
	}
	assertNoStoreToGlobal(t, ir, "@staticinit.Large")
}

func TestStaticGlobalInitSkipsLargeByteSlice(t *testing.T) {
	length := maxStaticInitArrayElements + 1
	src := fmt.Sprintf(`package staticinit

var Large = []byte{%d: 1}

func Use() byte { return Large[%d] }
`, length-1, length-1)
	ir := compileWithRewrites(t, src, nil)
	if strings.Contains(ir, `@"staticinit.Large$data"`) {
		t.Fatal("large byte slice should retain runtime backing allocation")
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

	allocBacked := buildSSAPackage(t, `package foo
var G = 1
`)
	allocGlobal, ok := allocBacked.Members["G"].(*ssa.Global)
	if !ok {
		t.Fatalf("missing G global: %T", allocBacked.Members["G"])
	}
	var allocGlobalStore *ssa.Store
	for _, block := range allocBacked.Func("init").Blocks {
		for _, instr := range block.Instrs {
			if store, ok := instr.(*ssa.Store); ok && store.Addr == allocGlobal {
				allocGlobalStore = store
			}
		}
	}
	if allocGlobalStore == nil {
		t.Fatal("missing store to alloc-backed G")
	}
	value, ok := allocGlobalStore.Val.(*ssa.Const)
	if !ok {
		t.Fatalf("G initializer = %T, want *ssa.Const", allocGlobalStore.Val)
	}
	alloc := new(ssa.Alloc)
	allocStore := &ssa.Store{Addr: alloc, Val: value}
	resultLoad := &ssa.UnOp{Op: token.MUL, X: alloc}
	allocGlobalStore.Val = resultLoad
	allocRefs := alloc.Referrers()
	loadRefs := resultLoad.Referrers()
	if allocRefs == nil || loadRefs == nil {
		t.Fatal("expected alloc-backed initializer values to track referrers")
	}
	*allocRefs = []ssa.Instruction{allocStore, resultLoad}
	*loadRefs = []ssa.Instruction{allocGlobalStore}
	loweringProg := ssatest.NewProgram(t, nil)
	ctx = &context{prog: loweringProg, pkg: loweringProg.NewPackage("foo", "foo")}
	ctx.collectStaticGlobalInits(allocBacked)
	if _, ok := ctx.staticGlobalInits[allocGlobal]; !ok {
		t.Fatal("alloc-backed constant initializer was not folded")
	}
	for _, instr := range []ssa.Instruction{alloc, allocStore, resultLoad, allocGlobalStore} {
		if _, ok := ctx.staticInitInstrs[instr]; !ok {
			t.Fatalf("alloc-backed initializer did not suppress %T", instr)
		}
	}
	if _, ok := ctx.staticInitStores[allocStore]; !ok {
		t.Fatal("alloc-backed initializer store was not recorded")
	}

	// A second executable load makes ownership of the temporary alloc
	// ambiguous. Verify the package-level collector preserves the dynamic
	// initializer instead of only testing the helper's rejection in isolation.
	*allocRefs = append(*allocRefs, &ssa.UnOp{Op: token.MUL, X: alloc})
	fallbackCtx := &context{prog: loweringProg, pkg: loweringProg.NewPackage("foo-fallback", "foo-fallback")}
	fallbackCtx.collectStaticGlobalInits(allocBacked)
	if fallbackCtx.staticGlobalInits != nil {
		t.Fatalf("alloc with an extra load produced static initializers: %+v", fallbackCtx.staticGlobalInits)
	}
	if fallbackCtx.staticInitInstrs != nil || fallbackCtx.staticInitStores != nil {
		t.Fatal("rejected alloc-backed initializer suppressed executable instructions")
	}

	invalidPath := buildSSAPackage(t, `package foo
var G = 1
`)
	global, ok := invalidPath.Members["G"].(*ssa.Global)
	if !ok {
		t.Fatalf("missing G global: %T", invalidPath.Members["G"])
	}
	var globalStore *ssa.Store
	for _, block := range invalidPath.Func("init").Blocks {
		for _, instr := range block.Instrs {
			if store, ok := instr.(*ssa.Store); ok && store.Addr == global {
				globalStore = store
			}
		}
	}
	if globalStore == nil {
		t.Fatal("missing store to G")
	}
	// Keep a recognizable root global but make its index non-constant. A malformed
	// or future SSA address shape must make the whole candidate fall back, rather
	// than treating the failed path as a store to the root scalar.
	globalStore.Addr = &ssa.IndexAddr{X: global, Index: global}
	ctx = &context{prog: ssatest.NewProgram(t, nil)}
	ctx.collectStaticGlobalInits(invalidPath)
	if ctx.staticGlobalInits != nil {
		t.Fatalf("unsupported store path produced static initializers: %+v", ctx.staticGlobalInits)
	}
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

	target := new(ssa.Alloc)
	foreign := new(ssa.Alloc)
	if _, ok := staticInitStorePathToAlloc(&ssa.FieldAddr{X: foreign, Field: 0}, target); ok {
		t.Fatal("field path rooted at a different alloc should be rejected")
	}
	if _, ok := staticInitStorePathToAlloc(&ssa.IndexAddr{
		X:     foreign,
		Index: ssa.NewConst(constant.MakeInt64(0), types.Typ[types.Int]),
	}, target); ok {
		t.Fatal("index path rooted at a different alloc should be rejected")
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

	if new(staticInitNode).addStore(staticInitStore{}) {
		t.Fatal("store without a static leaf should be rejected")
	}

	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	if _, ok := ctx.buildStaticInitExpr(types.Typ[types.Int], &staticInitNode{
		function: &ssa.Function{Signature: sig},
	}); ok {
		t.Fatal("function leaf for a non-signature type should be rejected")
	}
	if _, ok := ctx.buildStaticInitExpr(
		types.NewArray(types.Typ[types.Int], maxStaticInitArrayElements+1),
		new(staticInitNode),
	); ok {
		t.Fatal("large non-byte array should be rejected")
	}

	array := types.NewArray(types.Typ[types.Int], 1)
	if _, ok := ctx.buildStaticSliceValue("data", types.Typ[types.Int], &staticSliceInit{array: array}); ok {
		t.Fatal("non-slice type should be rejected by static slice builder")
	}
	if _, ok := ctx.buildStaticSliceValue("data", types.NewSlice(types.Typ[types.String]), &staticSliceInit{array: array}); ok {
		t.Fatal("mismatched slice element type should be rejected")
	}
	if _, ok := ctx.buildStaticSliceValue("data", types.NewSlice(types.Typ[types.Int]), &staticSliceInit{
		array:  array,
		stores: []staticInitStore{{value: c}},
	}); ok {
		t.Fatal("slice element store without an index path should be rejected")
	}
	if _, ok := ctx.buildStaticSliceValue("data", types.NewSlice(types.Typ[types.Int]), &staticSliceInit{
		array: array,
		stores: []staticInitStore{{
			path:  []staticInitPathElem{{index: 1}},
			value: c,
		}},
	}); ok {
		t.Fatal("out-of-range slice element store should be rejected")
	}
	duplicate := staticInitStore{path: []staticInitPathElem{{index: 0}}, value: c}
	if _, ok := ctx.buildStaticSliceValue("data", types.NewSlice(types.Typ[types.Int]), &staticSliceInit{
		array:  array,
		stores: []staticInitStore{duplicate, duplicate},
	}); ok {
		t.Fatal("duplicate slice element stores should be rejected")
	}
	if _, ok := ctx.buildStaticPointerValue("data", types.Typ[types.Int], nil); ok {
		t.Fatal("non-pointer type should be rejected by static pointer builder")
	}

	pointerPkg := buildSSAPackage(t, `package foo
var P = &[1]int{1}
`)
	pointerGlobal, ok := pointerPkg.Members["P"].(*ssa.Global)
	if !ok {
		t.Fatalf("missing P global: %T", pointerPkg.Members["P"])
	}
	var pointerStore *ssa.Store
	for _, block := range pointerPkg.Func("init").Blocks {
		for _, instr := range block.Instrs {
			if store, ok := instr.(*ssa.Store); ok && store.Addr == pointerGlobal {
				pointerStore = store
			}
		}
	}
	if pointerStore == nil {
		t.Fatal("missing store to P")
	}
	pointerInit, ok := staticPointerInitOfVisited(pointerStore, nil)
	if !ok {
		t.Fatal("valid pointer initializer was rejected")
	}
	if _, ok := staticPointerInitOfVisited(pointerStore, map[*ssa.Alloc]bool{pointerInit.alloc: true}); ok {
		t.Fatal("cyclic pointer initializer should be rejected")
	}
	if _, ok := ctx.buildStaticPointerValue("data", types.NewPointer(types.Typ[types.Int]), pointerInit); ok {
		t.Fatal("mismatched pointer element type should be rejected")
	}
	if len(pointerInit.stores) == 0 {
		t.Fatal("pointer initializer has no element stores")
	}
	duplicatePointer := *pointerInit
	duplicatePointer.stores = append(append([]staticInitStore(nil), pointerInit.stores...), pointerInit.stores[0])
	pointerType := pointerGlobal.Type().Underlying().(*types.Pointer).Elem()
	if _, ok := ctx.buildStaticPointerValue("data", pointerType, &duplicatePointer); ok {
		t.Fatal("duplicate pointer element stores should be rejected")
	}
	invalidPointer := *pointerInit
	invalidPointer.stores = []staticInitStore{{
		path:  []staticInitPathElem{{index: 1}},
		value: c,
	}}
	if _, ok := ctx.buildStaticPointerValue("data", pointerType, &invalidPointer); ok {
		t.Fatal("invalid pointer element initializer should be rejected")
	}

	byteArray := types.NewArray(types.Typ[types.Uint8], 1)
	if _, ok := staticInitByteArray(&staticInitNode{children: map[int]*staticInitNode{
		0: {value: ssa.NewConst(constant.MakeString("x"), types.Typ[types.String])},
	}}, byteArray); ok {
		t.Fatal("non-integer byte array element should be rejected")
	}
	if _, ok := staticInitByteArray(&staticInitNode{children: map[int]*staticInitNode{
		0: {value: ssa.NewConst(constant.MakeInt64(256), types.Typ[types.Uint8])},
	}}, byteArray); ok {
		t.Fatal("out-of-range byte array element should be rejected")
	}
}

func TestStaticInitFunctionLeafVariants(t *testing.T) {
	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	function := &ssa.Function{Signature: sig}
	terminal := new(ssa.Store)
	closure := &ssa.MakeClosure{Fn: function}
	terminal.Val = closure
	refs := closure.Referrers()
	if refs == nil {
		t.Fatal("closure referrers unavailable")
	}
	*refs = []ssa.Instruction{terminal}

	if got, ok := staticInitFunctionOf(function, terminal); !ok || got != function {
		t.Fatalf("direct function leaf = (%v, %v), want (%v, true)", got, ok, function)
	}
	if got, ok := staticInitFunctionOf(closure, terminal); !ok || got != function {
		t.Fatalf("closure function leaf = (%v, %v), want (%v, true)", got, ok, function)
	}
	var stores []staticInitStore
	var instrs []ssa.Instruction
	if !handleStoreVal(terminal, nil, &stores, &instrs, nil) {
		t.Fatal("valid closure leaf was rejected")
	}
	if len(stores) != 1 || stores[0].function != function || len(instrs) != 1 || instrs[0] != closure {
		t.Fatalf("closure leaf collection = (%+v, %+v)", stores, instrs)
	}

	bound := &ssa.MakeClosure{Fn: function, Bindings: []ssa.Value{
		ssa.NewConst(constant.MakeInt64(1), types.Typ[types.Int]),
	}}
	if _, ok := staticInitFunctionOf(bound, terminal); ok {
		t.Fatal("closure with bindings should be rejected")
	}
	if _, ok := staticInitFunctionOf(&ssa.MakeClosure{
		Fn: ssa.NewConst(constant.MakeInt64(1), types.Typ[types.Int]),
	}, terminal); ok {
		t.Fatal("closure with non-function code should be rejected")
	}
	if _, ok := staticInitFunctionOf(&ssa.MakeClosure{
		Fn: &ssa.Function{Signature: sig, FreeVars: []*ssa.FreeVar{new(ssa.FreeVar)}},
	}, terminal); ok {
		t.Fatal("closure with free variables should be rejected")
	}
	*refs = append(*refs, new(ssa.Store))
	if _, ok := staticInitFunctionOf(closure, terminal); ok {
		t.Fatal("closure with an additional executable referrer should be rejected")
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

func namedResult(stop bool) (result *int) {
	if stop {
		return nil
	}
	for v := range seq {
		defer func() { result = &v }()
	}
	return nil
}

func unnamedResult(stop bool) int {
	if stop {
		return 1
	}
	for v := range seq {
		defer func() { _ = v }()
	}
	return 2
}
`)

	ir := m.String()
	if !strings.Contains(ir, "FreeDeferNode") {
		t.Fatalf("expected rangefunc defer node cleanup in module, got:\n%s", ir)
	}
	if !strings.Contains(ir, "sigsetjmp") && !strings.Contains(ir, "setjmp") {
		t.Fatalf("expected rangefunc defer stack setup in module, got:\n%s", ir)
	}
	if err := llvm.VerifyModule(m, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("rangefunc defer module is invalid: %v\n%s", err, ir)
	}
}

func TestNamedResultSlot(t *testing.T) {
	ssaPkg, _, _ := buildGoSSAPkg(t, `
package foo

func seq(yield func(int) bool) { _ = yield(1) }

func f() (result *int) {
	for v := range seq {
		defer func() { result = &v }()
	}
	return nil
}

func unnamed() string { return "" }

func lifted() (result int) { return }
`)

	fn := ssaPkg.Func("f")
	ctx := &context{goFn: fn}
	if got := (&context{}).namedResultSlot(0); got != nil {
		t.Fatalf("nil function result slot = %v, want nil", got)
	}
	for _, index := range []int{-1, 1} {
		if got := ctx.namedResultSlot(index); got != nil {
			t.Fatalf("result slot at index %d = %v, want nil", index, got)
		}
	}
	slot := ctx.namedResultSlot(0)
	if slot == nil || slot.Comment != "result" {
		t.Fatalf("named result slot = %v, want result allocation", slot)
	}
	ctx.goFn = ssaPkg.Func("unnamed")
	if got := ctx.namedResultSlot(0); got != nil {
		t.Fatalf("unnamed result slot = %v, want nil", got)
	}
	ctx.goFn = ssaPkg.Func("lifted")
	if got := ctx.namedResultSlot(0); got != nil {
		t.Fatalf("lifted named result slot = %v, want nil", got)
	}
}

func TestImplicitDeferResultSlotRejectsMissingSlot(t *testing.T) {
	for _, ctx := range []*context{
		{},
		{implicitDeferResults: make([]llssa.Expr, 1)},
	} {
		func() {
			defer func() {
				if got := recover(); got != "missing implicit defer result slot 0" {
					t.Fatalf("implicitDeferResultSlot panic = %v", got)
				}
			}()
			ctx.implicitDeferResultSlot(0)
		}()
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

func TestCollectAllocStoresFromSSA(t *testing.T) {
	const src = `package allocstore

type Point struct {
	X, Y int
}

type Nested struct {
	P   Point
	Arr [2]int
	Tag string
}

func testConstNested() Nested {
	return Nested{
		P:   Point{10, 20},
		Arr: [2]int{30, 40},
		Tag: "hello",
	}
}

func testConstPoint() Point {
	return Point{100, 200}
}

func testDynamic() Point {
	return Point{next(), 200}
}

func testCall(p Point) {}

func testEscape() {
	var p = Point{1, 2}
	testCall(p)
	var n = Nested{P: Point{3, 4}}
	pRef := &n.P
	pRef.X = 99
}

func testArrayInit() [2]int {
	var a [2]int
	a[0] = 10
	a[1] = 20
	return a
}

func testDirectStore() int {
	var x int
	x = 42
	return x
}

func testArrayDynamic() [2]int {
	var a [2]int
	a[0] = next()
	return a
}

func next() int { return 1 }
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "allocstore.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	importer := gpackages.NewImporter(fset)
	pkg, _, err := ssautil.BuildPackage(
		&types.Config{Importer: importer},
		fset,
		types.NewPackage("allocstore", "allocstore"),
		[]*ast.File{file},
		ssa.SanityCheckFunctions,
	)
	if err != nil {
		t.Fatal(err)
	}

	var foundAllocs []*ssa.Alloc
	var foundStores []*ssa.Store
	var foundFields []*ssa.FieldAddr
	var foundIndices []*ssa.IndexAddr

	for _, member := range pkg.Members {
		fn, ok := member.(*ssa.Function)
		if !ok {
			continue
		}
		for _, block := range fn.Blocks {
			for _, instr := range block.Instrs {
				switch instr := instr.(type) {
				case *ssa.Alloc:
					if !instr.Heap {
						foundAllocs = append(foundAllocs, instr)
					}
				case *ssa.Store:
					foundStores = append(foundStores, instr)
				case *ssa.FieldAddr:
					foundFields = append(foundFields, instr)
				case *ssa.IndexAddr:
					foundIndices = append(foundIndices, instr)
				}
			}
		}
	}

	if len(foundAllocs) == 0 {
		t.Fatal("expected to find local allocs in SSA")
	}

	targetAlloc := foundAllocs[0]

	// 1. Cycle detection
	var stores []staticInitStore
	var instrs []ssa.Instruction
	visited := map[*ssa.Alloc]bool{targetAlloc: true}
	if collectAllocStores(targetAlloc, nil, nil, nil, &stores, &instrs, visited) {
		t.Fatal("expected cycle protection to return false")
	}

	// 2. appendStaticInitPath
	p1 := []staticInitPathElem{{index: 1}, {index: 2}}
	p2 := []staticInitPathElem{{index: 3}}
	merged := appendStaticInitPath(p1, p2)
	if len(merged) != 3 || merged[0].index != 1 || merged[1].index != 2 || merged[2].index != 3 {
		t.Fatalf("unexpected appendStaticInitPath result: %+v", merged)
	}

	// 3. handleStoreVal branches
	if len(foundStores) > 0 {
		store := foundStores[0]
		// Test Const store
		cStore := &ssa.Store{Addr: store.Addr, Val: ssa.NewConst(constant.MakeInt64(1), types.Typ[types.Int])}
		stores = nil
		if !handleStoreVal(cStore, p1, &stores, &instrs, make(map[*ssa.Alloc]bool)) {
			t.Fatal("handleStoreVal failed on const")
		}
		if len(stores) != 1 || len(stores[0].path) != 2 {
			t.Fatalf("unexpected handleStoreVal result: %+v", stores)
		}

		// Test non-const non-unop store
		badStore := &ssa.Store{Addr: store.Addr, Val: store.Addr}
		if handleStoreVal(badStore, p1, &stores, &instrs, make(map[*ssa.Alloc]bool)) {
			t.Fatal("expected handleStoreVal to fail on non-const, non-alloc Val")
		}

		// Test Heap alloc
		heapAlloc := &ssa.Alloc{Heap: true, Comment: "heap"}
		heapUnOp := &ssa.UnOp{Op: token.MUL, X: heapAlloc}
		heapStore := &ssa.Store{Addr: store.Addr, Val: heapUnOp}
		if handleStoreVal(heapStore, p1, &stores, &instrs, make(map[*ssa.Alloc]bool)) {
			t.Fatal("expected handleStoreVal to fail on heap alloc")
		}
	}

	// 4. staticInitStorePathToAlloc edge cases
	if _, ok := staticInitStorePathToAlloc(nil, targetAlloc); ok {
		t.Fatal("expected nil addr to fail")
	}
	if path, ok := staticInitStorePathToAlloc(targetAlloc, targetAlloc); !ok || len(path) != 0 {
		t.Fatalf("expected exact alloc to return empty path, got %+v, %v", path, ok)
	}
	if len(foundAllocs) > 1 {
		if _, ok := staticInitStorePathToAlloc(foundAllocs[1], targetAlloc); ok {
			t.Fatal("expected different alloc to fail")
		}
	}
	if len(foundFields) > 0 {
		field := foundFields[0]
		_, _ = staticInitStorePathToAlloc(field, targetAlloc)
	}
	if len(foundIndices) > 0 {
		index := foundIndices[0]
		_, _ = staticInitStorePathToAlloc(index, targetAlloc)
	}
}

func TestStaticGlobalPointerIndirectionLiteralInit(t *testing.T) {
	const src = `package staticinit

type Inner struct {
	A [2]int
	B string
}

type Outer struct {
	I Inner
	Val int
}

var G = Outer{
	I: Inner{
		A: [2]int{10, 20},
		B: "hello",
	},
	Val: 99,
}

func Use() int {
	return G.I.A[0] + G.I.A[1] + len(G.I.B) + G.Val
}
`
	ir := compileWithRewrites(t, src, nil)
	if strings.Contains(ir, "@staticinit.G = global %staticinit.Outer zeroinitializer") {
		t.Fatalf("G still uses a zero initializer:\n%s", ir)
	}
	if !strings.Contains(ir, `c"hello"`) {
		t.Fatalf("missing hello in IR:\n%s", ir)
	}
}

func TestStaticGlobalZeroCompositeLiteralInit(t *testing.T) {
	const src = `package staticinit

type Zero struct {
	Value int
	Array [2]int
}

var G = Zero{}

func Use() int { return G.Value + G.Array[1] }
`
	ir := compileWithRewrites(t, src, nil)
	if !strings.Contains(ir, "@staticinit.G = global %staticinit.Zero zeroinitializer") {
		t.Fatalf("missing zero static initializer:\n%s", ir)
	}
	assertNoStoreToGlobal(t, ir, "@staticinit.G")
}

func TestStaticInitNodeAddEdgeCases(t *testing.T) {
	c1 := &ssa.Const{Value: constant.MakeInt64(1)}
	c2 := &ssa.Const{Value: constant.MakeInt64(2)}

	// 1. Setting value on empty path
	root := new(staticInitNode)
	if !root.add(nil, c1) {
		t.Fatal("expected add to succeed")
	}
	// 2. Overwriting existing leaf value should fail
	if root.add(nil, c2) {
		t.Fatal("expected duplicate leaf add to fail")
	}
	// 3. Adding child path when node already has a leaf value should fail
	if root.add([]staticInitPathElem{{index: 0}}, c2) {
		t.Fatal("expected adding child to leaf to fail")
	}

	// 4. Adding leaf to a node with children should fail
	root2 := new(staticInitNode)
	if !root2.add([]staticInitPathElem{{index: 0}}, c1) {
		t.Fatal("expected child add to succeed")
	}
	if root2.add(nil, c2) {
		t.Fatal("expected leaf add to branch node to fail")
	}
}

func TestStaticInitPathHelperEdgeCases(t *testing.T) {
	// staticInitRootGlobal
	if g := staticInitRootGlobal(nil); g != nil {
		t.Fatalf("expected nil root global, got %v", g)
	}

	// staticInitStorePath
	if _, ok := staticInitStorePath(nil); ok {
		t.Fatal("expected nil addr to fail store path")
	}

	// staticInitConstIndex
	if _, ok := staticInitConstIndex(nil); ok {
		t.Fatal("expected nil index to fail")
	}
	negConst := &ssa.Const{Value: constant.MakeInt64(-1)}
	if _, ok := staticInitConstIndex(negConst); ok {
		t.Fatal("expected negative index to fail")
	}
	strConst := &ssa.Const{Value: constant.MakeString("not an int")}
	if _, ok := staticInitConstIndex(strConst); ok {
		t.Fatal("expected string index to fail")
	}
	validConst := &ssa.Const{Value: constant.MakeInt64(5)}
	if idx, ok := staticInitConstIndex(validConst); !ok || idx != 5 {
		t.Fatalf("expected index 5, got %d, %v", idx, ok)
	}

	// staticInitStorePathToAlloc
	targetAlloc := new(ssa.Alloc)
	otherAlloc := new(ssa.Alloc)
	if _, ok := staticInitStorePathToAlloc(nil, targetAlloc); ok {
		t.Fatal("expected nil to fail")
	}
	if _, ok := staticInitStorePathToAlloc(otherAlloc, targetAlloc); ok {
		t.Fatal("expected different alloc to fail")
	}
	if path, ok := staticInitStorePathToAlloc(targetAlloc, targetAlloc); !ok || len(path) != 0 {
		t.Fatalf("expected empty path for matching alloc, got %v, %v", path, ok)
	}
}

func TestStaticInitZeroSizedPredicates(t *testing.T) {
	// Zero-sized array
	arrZero := types.NewArray(types.Typ[types.Int], 0)
	if !staticInitZeroSized(arrZero) {
		t.Fatal("expected [0]int to be zero sized")
	}

	// Non-zero sized array
	arrNonZero := types.NewArray(types.Typ[types.Int], 5)
	if staticInitZeroSized(arrNonZero) {
		t.Fatal("expected [5]int to not be zero sized")
	}

	// Zero-sized struct (empty struct)
	structEmpty := types.NewStruct(nil, nil)
	if !staticInitZeroSized(structEmpty) {
		t.Fatal("expected empty struct to be zero sized")
	}

	// Array of zero-sized struct
	arrEmptyStruct := types.NewArray(structEmpty, 10)
	if !staticInitZeroSized(arrEmptyStruct) {
		t.Fatal("expected [10]struct{} to be zero sized")
	}

	// Non-zero sized struct
	field := types.NewField(0, nil, "X", types.Typ[types.Int], false)
	structNonEmpty := types.NewStruct([]*types.Var{field}, nil)
	if staticInitZeroSized(structNonEmpty) {
		t.Fatal("expected struct with int field to not be zero sized")
	}
}

func TestStaticInitChildrenInRange(t *testing.T) {
	node := &staticInitNode{
		children: map[int]*staticInitNode{
			0: {},
			1: {},
		},
	}
	if !staticInitChildrenInRange(node, 2) {
		t.Fatal("expected 2 children in range 2 to return true")
	}
	if staticInitChildrenInRange(node, 1) {
		t.Fatal("expected child at index 1 to be out of range 1")
	}
	negNode := &staticInitNode{
		children: map[int]*staticInitNode{-1: {}},
	}
	if staticInitChildrenInRange(negNode, 5) {
		t.Fatal("expected negative index child to fail")
	}
}

func TestStaticInitCycleDetection(t *testing.T) {
	alloc := new(ssa.Alloc)
	visited := map[*ssa.Alloc]bool{
		alloc: true,
	}
	var stores []staticInitStore
	var instrs []ssa.Instruction
	if collectAllocStores(alloc, nil, nil, nil, &stores, &instrs, visited) {
		t.Fatal("expected visited alloc to be rejected")
	}
}

func TestStaticGlobalSliceWithBoundsRejects(t *testing.T) {
	const src = `package staticinit
var backing = [4]int{1, 2, 3, 4}
var SliceSub = backing[1:3]
func Use() int { return SliceSub[0] }
`
	ir := compileWithRewrites(t, src, nil)
	assertStoreToGlobal(t, ir, "@staticinit.SliceSub")
}

func TestStaticGlobalScalarNumericTypes(t *testing.T) {
	const src = `package staticinit
type FloatStruct struct {
	F32 float32
	F64 float64
	U8  uint8
	U64 uint64
	B   bool
}
var GFloat = FloatStruct{
	F32: 1.5,
	F64: 3.14159,
	U8:  255,
	U64: 18446744073709551615,
	B:   true,
}
func Use() float64 { return float64(GFloat.F32) + GFloat.F64 }
`
	ir := compileWithRewrites(t, src, nil)
	assertNoStoreToGlobal(t, ir, "@staticinit.GFloat")
}

func TestStaticSliceInitOfEdgeCases(t *testing.T) {
	// 1. Non-slice store value
	intStore := &ssa.Store{
		Addr: new(ssa.Global),
		Val:  ssa.NewConst(constant.MakeInt64(1), types.Typ[types.Int]),
	}
	if _, ok := staticSliceInitOf(intStore); ok {
		t.Fatal("expected non-slice store to fail staticSliceInitOf")
	}

	// 2. Slice with Low / High / Max bounds
	c0 := ssa.NewConst(constant.MakeInt64(0), types.Typ[types.Int])
	alloc := &ssa.Alloc{Comment: "arr"}
	boundedSlice := &ssa.Slice{
		X:    alloc,
		Low:  c0,
		High: c0,
	}
	boundedStore := &ssa.Store{
		Addr: new(ssa.Global),
		Val:  boundedSlice,
	}
	if _, ok := staticSliceInitOf(boundedStore); ok {
		t.Fatal("expected bounded slice to fail staticSliceInitOf")
	}

	// 3. Slice whose X is not an Alloc
	nonAllocSlice := &ssa.Slice{
		X: new(ssa.Global),
	}
	nonAllocStore := &ssa.Store{
		Addr: new(ssa.Global),
		Val:  nonAllocSlice,
	}
	if _, ok := staticSliceInitOf(nonAllocStore); ok {
		t.Fatal("expected non-alloc slice source to fail staticSliceInitOf")
	}
}

func TestCollectAllocStoresBranchEdgeCases(t *testing.T) {
	alloc := new(ssa.Alloc)
	var stores []staticInitStore
	var instrs []ssa.Instruction

	// 1. Cycle detection
	visited := map[*ssa.Alloc]bool{alloc: true}
	if collectAllocStores(alloc, nil, nil, nil, &stores, &instrs, visited) {
		t.Fatal("expected visited alloc to fail collectAllocStores")
	}

	// 2. handleStoreVal on UnOp with heap alloc
	heapAlloc := &ssa.Alloc{Heap: true}
	unopHeap := &ssa.UnOp{Op: token.MUL, X: heapAlloc}
	storeHeap := &ssa.Store{Addr: alloc, Val: unopHeap}
	if handleStoreVal(storeHeap, nil, &stores, &instrs, make(map[*ssa.Alloc]bool)) {
		t.Fatal("expected heap alloc unop to fail handleStoreVal")
	}

	// 3. handleStoreVal on non-MUL UnOp
	unopNotMul := &ssa.UnOp{Op: token.NOT, X: alloc}
	storeNotMul := &ssa.Store{Addr: alloc, Val: unopNotMul}
	if handleStoreVal(storeNotMul, nil, &stores, &instrs, make(map[*ssa.Alloc]bool)) {
		t.Fatal("expected non-MUL unop to fail handleStoreVal")
	}
}

func TestCollectAllocStoresRejectsInvalidTerminalContract(t *testing.T) {
	validLoad := func(alloc *ssa.Alloc) *ssa.UnOp {
		return &ssa.UnOp{Op: token.MUL, X: alloc}
	}
	constValue := func() *ssa.Const {
		return ssa.NewConst(constant.MakeInt64(0), types.Typ[types.Int])
	}
	tests := []struct {
		name  string
		build func(*ssa.Alloc) (ssa.Value, *ssa.Store)
	}{
		{
			name: "load with wrong operation",
			build: func(alloc *ssa.Alloc) (ssa.Value, *ssa.Store) {
				terminal := &ssa.UnOp{Op: token.NOT, X: alloc}
				return terminal, &ssa.Store{Val: terminal}
			},
		},
		{
			name: "load from foreign alloc",
			build: func(_ *ssa.Alloc) (ssa.Value, *ssa.Store) {
				terminal := validLoad(new(ssa.Alloc))
				return terminal, &ssa.Store{Val: terminal}
			},
		},
		{
			name: "slice from foreign alloc",
			build: func(_ *ssa.Alloc) (ssa.Value, *ssa.Store) {
				terminal := &ssa.Slice{X: new(ssa.Alloc)}
				return terminal, &ssa.Store{Val: terminal}
			},
		},
		{
			name: "bounded slice",
			build: func(alloc *ssa.Alloc) (ssa.Value, *ssa.Store) {
				terminal := &ssa.Slice{X: alloc, Low: constValue()}
				return terminal, &ssa.Store{Val: terminal}
			},
		},
		{
			name: "unsupported terminal",
			build: func(_ *ssa.Alloc) (ssa.Value, *ssa.Store) {
				terminal := constValue()
				return terminal, &ssa.Store{Val: terminal}
			},
		},
		{
			name: "missing destination store",
			build: func(alloc *ssa.Alloc) (ssa.Value, *ssa.Store) {
				return validLoad(alloc), nil
			},
		},
		{
			name: "destination store has another value",
			build: func(alloc *ssa.Alloc) (ssa.Value, *ssa.Store) {
				return validLoad(alloc), &ssa.Store{Val: constValue()}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			alloc := new(ssa.Alloc)
			terminal, terminalStore := test.build(alloc)
			var stores []staticInitStore
			var instrs []ssa.Instruction
			if collectAllocStores(
				alloc, terminal, terminalStore, nil,
				&stores, &instrs, make(map[*ssa.Alloc]bool),
			) {
				t.Fatal("accepted an invalid alloc terminal contract")
			}
			if len(stores) != 0 || len(instrs) != 0 {
				t.Fatalf("invalid terminal collected stores or instructions: stores=%v instrs=%v", stores, instrs)
			}
		})
	}
}

func TestCollectAllocStoresRejectsNestedDynamicIndex(t *testing.T) {
	setRefs := func(value ssa.Value, refs ...ssa.Instruction) {
		t.Helper()
		referrers := value.Referrers()
		if referrers == nil {
			t.Fatalf("%T does not track referrers", value)
		}
		*referrers = refs
	}

	alloc := new(ssa.Alloc)
	field := &ssa.FieldAddr{X: alloc}
	dynamicIndex := &ssa.IndexAddr{X: field, Index: alloc}
	resultLoad := &ssa.UnOp{Op: token.MUL, X: alloc}
	resultStore := &ssa.Store{Val: resultLoad}
	setRefs(alloc, field, resultLoad)
	setRefs(field, dynamicIndex)
	setRefs(resultLoad, resultStore)

	var stores []staticInitStore
	var instrs []ssa.Instruction
	if collectAllocStores(
		alloc, resultLoad, resultStore, nil,
		&stores, &instrs, make(map[*ssa.Alloc]bool),
	) {
		t.Fatal("accepted a nested address with a dynamic index")
	}
}

func TestStaticInitStorePathToAllocRejectsDynamicIndex(t *testing.T) {
	target := new(ssa.Alloc)
	if _, ok := staticInitStorePathToAlloc(&ssa.IndexAddr{X: target, Index: target}, target); ok {
		t.Fatal("dynamic index produced a static alloc path")
	}
}

func TestCollectAllocStoresNestedPaths(t *testing.T) {
	setRefs := func(value ssa.Value, refs ...ssa.Instruction) {
		t.Helper()
		referrers := value.Referrers()
		if referrers == nil {
			t.Fatalf("%T does not track referrers", value)
		}
		*referrers = refs
	}

	leafValue := ssa.NewConst(constant.MakeInt64(42), types.Typ[types.Int])
	outerAlloc := new(ssa.Alloc)
	innerAlloc := new(ssa.Alloc)
	outerField := &ssa.FieldAddr{X: outerAlloc, Field: 1}
	innerIndex := &ssa.IndexAddr{
		X:     innerAlloc,
		Index: ssa.NewConst(constant.MakeInt64(0), types.Typ[types.Int]),
	}
	leafStore := &ssa.Store{Addr: innerIndex, Val: leafValue}
	innerLoad := &ssa.UnOp{Op: token.MUL, X: innerAlloc}
	nestedStore := &ssa.Store{Addr: outerField, Val: innerLoad}
	resultLoad := &ssa.UnOp{Op: token.MUL, X: outerAlloc}
	resultStore := &ssa.Store{Val: resultLoad}

	setRefs(outerAlloc, outerField, resultLoad)
	setRefs(outerField, nestedStore)
	setRefs(innerAlloc, innerIndex, innerLoad)
	setRefs(innerIndex, leafStore)
	setRefs(innerLoad, nestedStore)
	setRefs(resultLoad, resultStore)

	var stores []staticInitStore
	var instrs []ssa.Instruction
	if !collectAllocStores(
		outerAlloc, resultLoad, resultStore, []staticInitPathElem{{index: 3}},
		&stores, &instrs, make(map[*ssa.Alloc]bool),
	) {
		t.Fatal("rejected a fully owned nested aggregate")
	}
	if len(stores) != 1 || stores[0].store != leafStore || stores[0].value != leafValue {
		t.Fatalf("unexpected collected stores: %+v", stores)
	}
	wantPath := []int{3, 1, 0}
	if len(stores[0].path) != len(wantPath) {
		t.Fatalf("collected path = %+v, want %v", stores[0].path, wantPath)
	}
	for i, want := range wantPath {
		if stores[0].path[i].index != want {
			t.Fatalf("collected path = %+v, want %v", stores[0].path, wantPath)
		}
	}
	for _, want := range []ssa.Instruction{
		outerAlloc, outerField, nestedStore, innerAlloc, innerIndex, leafStore, innerLoad, resultLoad,
	} {
		found := false
		for _, instr := range instrs {
			found = found || instr == want
		}
		if !found {
			t.Fatalf("missing suppressed instruction %T", want)
		}
	}
}

func TestCollectAllocStoresRejectsUnrelatedLoadConsumer(t *testing.T) {
	alloc := new(ssa.Alloc)
	resultLoad := &ssa.UnOp{Op: token.MUL, X: alloc}
	resultStore := &ssa.Store{Val: resultLoad}
	unrelatedStore := &ssa.Store{Val: resultLoad}
	allocRefs := alloc.Referrers()
	loadRefs := resultLoad.Referrers()
	if allocRefs == nil || loadRefs == nil {
		t.Fatal("expected alloc and load to track referrers")
	}
	*allocRefs = []ssa.Instruction{resultLoad}
	*loadRefs = []ssa.Instruction{unrelatedStore}

	var stores []staticInitStore
	var instrs []ssa.Instruction
	if collectAllocStores(
		alloc, resultLoad, resultStore, nil,
		&stores, &instrs, make(map[*ssa.Alloc]bool),
	) {
		t.Fatal("accepted a result load consumed by a store outside the fold")
	}
}

func TestCollectAllocStoresRejectsUnsafeReferrers(t *testing.T) {
	setRefs := func(t *testing.T, value ssa.Value, refs ...ssa.Instruction) {
		t.Helper()
		referrers := value.Referrers()
		if referrers == nil {
			t.Fatalf("%T does not track referrers", value)
		}
		*referrers = refs
	}
	constValue := func() *ssa.Const {
		return ssa.NewConst(constant.MakeInt64(1), types.Typ[types.Int])
	}
	constIndex := func() *ssa.Const {
		return ssa.NewConst(constant.MakeInt64(0), types.Typ[types.Int])
	}

	tests := []struct {
		name string
		bad  func(*testing.T, *ssa.Alloc) ssa.Instruction
	}{
		{
			name: "extra load",
			bad: func(t *testing.T, alloc *ssa.Alloc) ssa.Instruction {
				load := &ssa.UnOp{Op: token.MUL, X: alloc}
				setRefs(t, load, &ssa.Store{Val: load})
				return load
			},
		},
		{
			name: "field with multiple consumers",
			bad: func(t *testing.T, alloc *ssa.Alloc) ssa.Instruction {
				field := &ssa.FieldAddr{X: alloc}
				setRefs(t, field,
					&ssa.Store{Addr: field, Val: constValue()},
					&ssa.Store{Addr: field, Val: constValue()},
				)
				return field
			},
		},
		{
			name: "field with dynamic value",
			bad: func(t *testing.T, alloc *ssa.Alloc) ssa.Instruction {
				field := &ssa.FieldAddr{X: alloc}
				setRefs(t, field, &ssa.Store{Addr: field, Val: field})
				return field
			},
		},
		{
			name: "index with dynamic subscript",
			bad: func(t *testing.T, alloc *ssa.Alloc) ssa.Instruction {
				return &ssa.IndexAddr{X: alloc, Index: alloc}
			},
		},
		{
			name: "index with foreign consumer",
			bad: func(t *testing.T, alloc *ssa.Alloc) ssa.Instruction {
				index := &ssa.IndexAddr{X: alloc, Index: constIndex()}
				setRefs(t, index, &ssa.Store{Addr: new(ssa.Alloc), Val: constValue()})
				return index
			},
		},
		{
			name: "direct store to foreign address",
			bad: func(_ *testing.T, _ *ssa.Alloc) ssa.Instruction {
				return &ssa.Store{Addr: new(ssa.Alloc), Val: constValue()}
			},
		},
		{
			name: "direct store with dynamic value",
			bad: func(_ *testing.T, alloc *ssa.Alloc) ssa.Instruction {
				return &ssa.Store{Addr: alloc, Val: alloc}
			},
		},
		{
			name: "unsupported referrer",
			bad: func(_ *testing.T, _ *ssa.Alloc) ssa.Instruction {
				return new(ssa.Call)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			alloc := new(ssa.Alloc)
			resultLoad := &ssa.UnOp{Op: token.MUL, X: alloc}
			resultStore := &ssa.Store{Val: resultLoad}
			setRefs(t, resultLoad, resultStore)
			setRefs(t, alloc, test.bad(t, alloc), resultLoad)

			var stores []staticInitStore
			var instrs []ssa.Instruction
			if collectAllocStores(
				alloc, resultLoad, resultStore, nil,
				&stores, &instrs, make(map[*ssa.Alloc]bool),
			) {
				t.Fatal("accepted an unsafe alloc referrer")
			}
		})
	}
}
