package cl

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"runtime"
	"strings"
	"testing"

	llssa "github.com/goplus/llgo/ssa"
	"github.com/goplus/llgo/ssa/ssatest"
	"golang.org/x/tools/go/ssa"
)

func compileLocalitySource(t *testing.T, src string) (llssa.Program, string) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "locality.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	files := []*ast.File{file}
	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Scopes:     make(map[ast.Node]*types.Scope),
		Instances:  make(map[*ast.Ident]types.Instance),
	}
	imp := importer.Default()
	pkg, err := (&types.Config{Importer: imp}).Check("example.com/locality", fset, files, info)
	if err != nil {
		t.Fatal(err)
	}
	prog := ssatest.NewProgramEx(t, nil, imp)
	prog.TypeSizes(types.SizesFor("gc", runtime.GOARCH))
	runtimePkg := types.NewPackage(llssa.PkgRuntime, "runtime")
	params := types.NewTuple(
		types.NewParam(token.NoPos, runtimePkg, "start", types.Typ[types.UnsafePointer]),
		types.NewParam(token.NoPos, runtimePkg, "size", types.Typ[types.Uintptr]),
	)
	runtimePkg.Scope().Insert(types.NewFunc(token.NoPos, runtimePkg, "RegisterLocalRoot", types.NewSignatureType(nil, nil, nil, params, nil, false)))
	prog.SetRuntime(runtimePkg)
	if err := ParsePkgSyntax(prog, fset, pkg, files); err != nil {
		t.Fatal(err)
	}
	if err := PrepareLocalVariables(prog, fset, pkg, info, files); err != nil {
		t.Fatal(err)
	}
	goProg := ssa.NewProgram(fset, ssa.SanityCheckFunctions)
	ssaPkg := goProg.CreatePackage(pkg, files, info, true)
	ssaPkg.Build()
	compiled, err := NewPackage(prog, ssaPkg, files)
	if err != nil {
		t.Fatal(err)
	}
	return prog, compiled.String()
}

func prepareLocalitySource(t *testing.T, src string) error {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "locality.go", src, parser.ParseComments)
	if err != nil {
		return err
	}
	files := []*ast.File{file}
	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Scopes:     make(map[ast.Node]*types.Scope),
		Instances:  make(map[*ast.Ident]types.Instance),
	}
	pkg, err := (&types.Config{Importer: importer.Default()}).Check("example.com/locality", fset, files, info)
	if err != nil {
		return err
	}
	prog := ssatest.NewProgram(t, nil)
	if err := ParsePkgSyntax(prog, fset, pkg, files); err != nil {
		return err
	}
	return PrepareLocalVariables(prog, fset, pkg, info, files)
}

func TestLocalityDirectivesLowerToTLS(t *testing.T) {
	prog, ir := compileLocalitySource(t, `package locality

func makeValue() int { return 42 }

var ordinaryValue int

//llgo:tls
var threadValue = makeValue()

//llgo:gls
var goroutineValue *int

// llgo:tls
var zeroValue int

func values() (int, *int, int) { return threadValue, goroutineValue, zeroValue }
`)
	thread, ok := prog.DeclInfo("example.com/locality.threadValue")
	if !ok || thread.Locality != llssa.ThreadLocal || thread.InitFunc == "" {
		t.Fatalf("threadValue metadata = %+v, %v", thread, ok)
	}
	goroutine, ok := prog.DeclInfo("example.com/locality.goroutineValue")
	if !ok || goroutine.Locality != llssa.GoroutineLocal || goroutine.InitFunc != "" || goroutine.EnsureFunc == "" {
		t.Fatalf("goroutineValue metadata = %+v, %v", goroutine, ok)
	}
	zero, ok := prog.DeclInfo("example.com/locality.zeroValue")
	if !ok || zero.Locality != llssa.ThreadLocal || zero.InitFunc != "" || zero.EnsureFunc != "" {
		t.Fatalf("zeroValue metadata = %+v, %v", zero, ok)
	}
	for _, want := range []string{
		`@"example.com/locality.threadValue" = thread_local global i64`,
		`@"example.com/locality.goroutineValue" = thread_local global ptr`,
		`@"example.com/locality.zeroValue" = thread_local global i64`,
		`@"example.com/locality.__llgo_local_init_0$guard" = thread_local global i8 0`,
		`@"example.com/locality.goroutineValue$guard" = thread_local global i8 0`,
		`define void @"example.com/locality.__llgo_local_init_0$ensure"()`,
		`define void @"example.com/locality.goroutineValue$ensure"()`,
		`call void @"example.com/locality.__llgo_local_init_0$ensure"()`,
		`call void @"github.com/goplus/llgo/runtime/internal/runtime.RegisterLocalRoot"`,
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("IR missing %q:\n%s", want, ir)
		}
	}
	guardDone := `store i8 2, ptr @"example.com/locality.__llgo_local_init_0$guard"`
	if got := strings.Count(ir, guardDone); got != 2 {
		t.Fatalf("initializer guard done stores = %d, want 2 (ensure and package init):\n%s", got, ir)
	}
}

func TestLocalityDirectiveConflict(t *testing.T) {
	err := prepareLocalitySource(t, `package locality

//llgo:tls
//llgo:gls
var value int
`)
	if err == nil || !strings.Contains(err.Error(), "cannot apply to the same variable declaration") {
		t.Fatalf("PrepareLocalVariables error = %v", err)
	}
}

func TestLocalityDirectiveRejectsBlankIdentifier(t *testing.T) {
	err := prepareLocalitySource(t, `package locality

//llgo:tls
var _ int
`)
	if err == nil || !strings.Contains(err.Error(), "blank identifier") {
		t.Fatalf("PrepareLocalVariables error = %v", err)
	}
}

func TestLocalityDirectiveDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "legacy tls spelling",
			src: `package locality

//llgo:threadlocal
var value int
`,
			want: "use //llgo:tls",
		},
		{
			name: "legacy gls spelling",
			src: `package locality

//llgo:goroutinelocal
var value int
`,
			want: "use //llgo:gls",
		},
		{
			name: "arguments",
			src: `package locality

//llgo:tls extra
var value int
`,
			want: "does not accept arguments",
		},
		{
			name: "function",
			src: `package locality

//llgo:gls
func value() {}
`,
			want: "applies only to package-level var declarations",
		},
		{
			name: "type",
			src: `package locality

//llgo:tls
type value int
`,
			want: "applies only to package-level var declarations",
		},
		{
			name: "constant",
			src: `package locality

//llgo:tls
const value = 1
`,
			want: "applies only to package-level var declarations",
		},
		{
			name: "embed",
			src: `package locality

import _ "embed"

//go:embed locality.go
//llgo:tls
var value string
`,
			want: "cannot apply to the same variable declaration",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := prepareLocalitySource(t, test.src)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("PrepareLocalVariables error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestTypeHasPointers(t *testing.T) {
	pkg := types.NewPackage("example.com/types", "types")
	namedInt := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Int", nil), types.Typ[types.Int], nil)
	typeParam := types.NewTypeParam(types.NewTypeName(token.NoPos, pkg, "T", nil), types.NewInterfaceType(nil, nil).Complete())
	tests := []struct {
		name string
		typ  types.Type
		want bool
	}{
		{name: "int", typ: types.Typ[types.Int]},
		{name: "string", typ: types.Typ[types.String], want: true},
		{name: "unsafe pointer", typ: types.Typ[types.UnsafePointer], want: true},
		{name: "pointer", typ: types.NewPointer(types.Typ[types.Int]), want: true},
		{name: "slice", typ: types.NewSlice(types.Typ[types.Int]), want: true},
		{name: "map", typ: types.NewMap(types.Typ[types.Int], types.Typ[types.Int]), want: true},
		{name: "channel", typ: types.NewChan(types.SendRecv, types.Typ[types.Int]), want: true},
		{name: "signature", typ: types.NewSignatureType(nil, nil, nil, nil, nil, false), want: true},
		{name: "interface", typ: types.NewInterfaceType(nil, nil).Complete(), want: true},
		{name: "array", typ: types.NewArray(types.Typ[types.Int], 2)},
		{name: "pointer array", typ: types.NewArray(types.NewPointer(types.Typ[types.Int]), 2), want: true},
		{name: "struct", typ: types.NewStruct([]*types.Var{types.NewVar(token.NoPos, pkg, "n", types.Typ[types.Int])}, nil)},
		{name: "pointer struct", typ: types.NewStruct([]*types.Var{
			types.NewVar(token.NoPos, pkg, "n", types.Typ[types.Int]),
			types.NewVar(token.NoPos, pkg, "p", types.NewPointer(types.Typ[types.Int])),
		}, nil), want: true},
		{name: "named", typ: namedInt},
		{name: "type parameter", typ: typeParam, want: true},
		{name: "tuple", typ: types.NewTuple(types.NewVar(token.NoPos, pkg, "n", types.Typ[types.Int]))},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := typeHasPointers(test.typ); got != test.want {
				t.Fatalf("typeHasPointers(%v) = %v, want %v", test.typ, got, test.want)
			}
		})
	}
}

func TestInitializerLocalityDiagnostics(t *testing.T) {
	pkg := types.NewPackage("example.com/locality", "locality")
	thread := types.NewVar(token.NoPos, pkg, "thread", types.Typ[types.Int])
	goroutine := types.NewVar(token.NoPos, pkg, "goroutine", types.Typ[types.Int])
	ordinary := types.NewVar(token.NoPos, pkg, "ordinary", types.Typ[types.Int])
	prog := ssatest.NewProgram(t, nil)
	prog.SetVarLocality(llssa.FullName(pkg, thread.Name()), llssa.ThreadLocal, true)
	prog.SetVarLocality(llssa.FullName(pkg, goroutine.Name()), llssa.GoroutineLocal, true)
	rhs := ast.NewIdent("rhs")
	tests := []struct {
		name string
		lhs  []*types.Var
		want string
	}{
		{name: "local then ordinary", lhs: []*types.Var{thread, ordinary}, want: "mix local and ordinary"},
		{name: "ordinary then local", lhs: []*types.Var{ordinary, thread}, want: "mix local and ordinary"},
		{name: "tls then gls", lhs: []*types.Var{thread, goroutine}, want: "mix thread-local and goroutine-local"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := initializerLocality(prog, nil, pkg, &types.Initializer{Lhs: test.lhs, Rhs: rhs})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("initializerLocality error = %v, want %q", err, test.want)
			}
		})
	}
	if locality, found, err := initializerLocality(prog, nil, pkg, &types.Initializer{Lhs: []*types.Var{ordinary}, Rhs: rhs}); err != nil || found || locality != llssa.LocalityNone {
		t.Fatalf("ordinary initializer = %v, %v, %v", locality, found, err)
	}
}

func TestValidateLocalInitializers(t *testing.T) {
	pkg := types.NewPackage("example.com/locality", "locality")
	prog := ssatest.NewProgram(t, nil)
	name := llssa.FullName(pkg, "value")
	prog.SetVarLocality(name, llssa.ThreadLocal, true)
	if err := validateLocalInitializers(prog, pkg); err == nil || !strings.Contains(err.Error(), "initializer prepared") {
		t.Fatalf("validateLocalInitializers error = %v", err)
	}
	prog.SetLocalInitFunc(name, "example.com/locality.init")
	prog.SetLocalEnsureFunc(name, "example.com/locality.init$ensure")
	if err := validateLocalInitializers(prog, pkg); err != nil {
		t.Fatal(err)
	}
}

func TestParseSourceDirective(t *testing.T) {
	tests := []struct {
		text string
		name string
		args string
		ok   bool
	}{
		{text: "// ordinary"},
		{text: "//go:"},
		{text: "//go:noinline", name: "go:noinline", ok: true},
		{text: "//llgo:tls", name: "llgo:tls", ok: true},
		{text: "// llgo:type C", name: "llgo:type", args: "C", ok: true},
		{text: "//llgo:link\tF C.f", name: "llgo:link", args: "F C.f", ok: true},
		{text: "//export F", name: "export", args: "F", ok: true},
	}
	if _, ok := parseSourceDirective(nil); ok {
		t.Fatal("nil comment parsed as a directive")
	}
	for _, test := range tests {
		t.Run(test.text, func(t *testing.T) {
			got, ok := parseSourceDirective(&ast.Comment{Text: test.text})
			if ok != test.ok || got.Name != test.name || got.Args != test.args {
				t.Fatalf("parseSourceDirective(%q) = %+v, %v", test.text, got, ok)
			}
		})
	}
}

func TestPrepareLocalVariablesEarlyReturns(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	pkg := types.NewPackage("example.com/locality", "locality")
	info := &types.Info{}
	if err := PrepareLocalVariables(prog, nil, nil, info, nil); err != nil {
		t.Fatal(err)
	}
	if err := PrepareLocalVariables(prog, nil, pkg, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := PrepareLocalVariables(prog, nil, pkg, info, nil); err != nil {
		t.Fatal(err)
	}
	value := types.NewVar(token.NoPos, pkg, "value", types.Typ[types.Int])
	pkg.Scope().Insert(value)
	prog.SetVarLocality(llssa.FullName(pkg, value.Name()), llssa.ThreadLocal, true)
	info.InitOrder = []*types.Initializer{{Lhs: []*types.Var{value}, Rhs: ast.NewIdent("rhs")}}
	if err := PrepareLocalVariables(prog, nil, pkg, info, nil); err == nil || !strings.Contains(err.Error(), "without syntax files") {
		t.Fatalf("PrepareLocalVariables without files error = %v", err)
	}
	if err := rejectNonVarLocality(nil, nil); err != nil {
		t.Fatal(err)
	}
	(&context{}).initializeLocalGuards(nil)
}

func TestLocalInitializerNameCollision(t *testing.T) {
	prog, _ := compileLocalitySource(t, `package locality

func __llgo_local_init_0() {}

//llgo:tls
var value = 1
`)
	info, ok := prog.DeclInfo("example.com/locality.value")
	if !ok || !strings.HasSuffix(info.InitFunc, ".__llgo_local_init_1") {
		t.Fatalf("value metadata = %+v, %v", info, ok)
	}
}

func TestMergeLocality(t *testing.T) {
	tests := []struct {
		a, b llssa.Locality
		want llssa.Locality
		err  bool
	}{
		{want: llssa.LocalityNone},
		{a: llssa.ThreadLocal, want: llssa.ThreadLocal},
		{b: llssa.GoroutineLocal, want: llssa.GoroutineLocal},
		{a: llssa.ThreadLocal, b: llssa.ThreadLocal, want: llssa.ThreadLocal},
		{a: llssa.ThreadLocal, b: llssa.GoroutineLocal, err: true},
	}
	for _, test := range tests {
		got, _, err := mergeLocality(nil, test.a, token.NoPos, test.b, token.NoPos)
		if (err != nil) != test.err || got != test.want {
			t.Fatalf("mergeLocality(%v, %v) = %v, %v", test.a, test.b, got, err)
		}
	}
}

func TestLocalityDirectiveRejectsEmbeddedTarget(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "locality.go", `package locality

//llgo:tls
var value int
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := (&types.Config{}).Check("example.com/locality", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	prog := ssatest.NewProgram(t, nil)
	prog.Target().Target = "rp2040"
	err = ParsePkgSyntax(prog, fset, pkg, []*ast.File{file})
	if err == nil || !strings.Contains(err.Error(), `not supported for target "rp2040"`) {
		t.Fatalf("ParsePkgSyntax error = %v", err)
	}
}
