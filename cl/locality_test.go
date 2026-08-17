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

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/ssa/abi"
	"github.com/xgo-dev/llgo/ssa/ssatest"
	"golang.org/x/tools/go/ssa"
)

func compileLocalitySource(t *testing.T, src string) (llssa.Program, string) {
	return compileLocalitySourceWithOptions(t, src, Options{})
}

func compileLocalitySourceWithOptions(t *testing.T, src string, options Options) (llssa.Program, string) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "locality.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	files := []*ast.File{file}
	info := newLocalityTypeInfo()
	imp := importer.Default()
	pkg, err := (&types.Config{Importer: imp}).Check("example.com/locality", fset, files, info)
	if err != nil {
		t.Fatal(err)
	}
	prog := ssatest.NewProgramEx(t, nil, imp)
	prog.TypeSizes(types.SizesFor("gc", runtime.GOARCH))
	prog.SetRuntime(localityRuntimePackage())
	if err := ParsePkgSyntaxWithOptions(prog, fset, pkg, files, options); err != nil {
		t.Fatal(err)
	}
	if err := PrepareLocalVariables(prog, fset, pkg, info, files); err != nil {
		t.Fatal(err)
	}
	goProg := ssa.NewProgram(fset, ssa.SanityCheckFunctions)
	ssaPkg := goProg.CreatePackage(pkg, files, info, true)
	ssaPkg.Build()
	compiled, _, err := NewPackageExWithEmbedMetaOptions(
		prog, nil, nil, nil, ssaPkg, files, nil, false, options,
	)
	if err != nil {
		t.Fatal(err)
	}
	return prog, compiled.String()
}

func newLocalityTypeInfo() *types.Info {
	return &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Scopes:     make(map[ast.Node]*types.Scope),
		Instances:  make(map[*ast.Ident]types.Instance),
	}
}

func localityRuntimePackage() *types.Package {
	pkg := types.NewPackage(llssa.PkgRuntime, "runtime")
	unsafePointer := types.Typ[types.UnsafePointer]
	blocks := types.NewField(token.NoPos, pkg, "blocks", unsafePointer, false)
	localContextName := types.NewTypeName(token.NoPos, pkg, "LocalContext", nil)
	localContext := types.NewNamed(localContextName, types.NewStruct([]*types.Var{blocks}, nil), nil)
	pkg.Scope().Insert(localContextName)

	localPackageParams := types.NewTuple(
		types.NewParam(token.NoPos, pkg, "cache", types.NewPointer(types.Typ[types.Uintptr])),
		types.NewParam(token.NoPos, pkg, "size", types.Typ[types.Uintptr]),
		types.NewParam(token.NoPos, pkg, "align", types.Typ[types.Uintptr]),
	)
	localPackageResults := types.NewTuple(types.NewParam(token.NoPos, pkg, "", types.Typ[types.UnsafePointer]))
	pkg.Scope().Insert(types.NewFunc(token.NoPos, pkg, "LocalPackage", types.NewSignatureType(nil, nil, nil, localPackageParams, localPackageResults, false)))

	callback := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	ensureParams := types.NewTuple(
		types.NewParam(token.NoPos, pkg, "state", types.NewPointer(types.Typ[types.Uint8])),
		types.NewParam(token.NoPos, pkg, "failureCache", types.NewPointer(types.Typ[types.Uintptr])),
		types.NewParam(token.NoPos, pkg, "initialize", callback),
	)
	pkg.Scope().Insert(types.NewFunc(token.NoPos, pkg, "EnsureLocalInitializer", types.NewSignatureType(nil, nil, nil, ensureParams, nil, false)))

	contextPointer := types.NewPointer(localContext)
	enterParams := types.NewTuple(types.NewParam(token.NoPos, pkg, "ctx", contextPointer))
	enterResults := types.NewTuple(types.NewParam(token.NoPos, pkg, "previous", types.Typ[types.Uintptr]))
	pkg.Scope().Insert(types.NewFunc(token.NoPos, pkg, "EnterLocalContext", types.NewSignatureType(nil, nil, nil, enterParams, enterResults, false)))
	leaveParams := types.NewTuple(
		types.NewParam(token.NoPos, pkg, "ctx", contextPointer),
		types.NewParam(token.NoPos, pkg, "previous", types.Typ[types.Uintptr]),
	)
	pkg.Scope().Insert(types.NewFunc(token.NoPos, pkg, "LeaveLocalContext", types.NewSignatureType(nil, nil, nil, leaveParams, nil, false)))
	return pkg
}

func TestLocalPackageAccessorUsesDirectTLSCache(t *testing.T) {
	const src = `package locality

//llgo:gls
var pointer *int

func value() *int { return pointer }
`
	_, ir := compileLocalitySource(t, src)
	if !strings.Contains(ir, `@"example.com/locality.__llgo_local_cache" = thread_local global i64 0`) {
		t.Fatalf("package direct cache not found:\n%s", ir)
	}
	accessor := llvmFunction(t, ir, "example.com/locality.__llgo_local_block")
	cacheLoad := `load i64, ptr @"example.com/locality.__llgo_local_cache"`
	slow := `call ptr @"` + llssa.PkgRuntime + `.LocalPackage"(ptr @"example.com/locality.__llgo_local_cache"`
	if loadAt, slowAt := strings.Index(accessor, cacheLoad), strings.Index(accessor, slow); loadAt < 0 || slowAt < 0 || loadAt >= slowAt {
		t.Fatalf("accessor does not load its direct cache before the cold path:\n%s", accessor)
	}
	if got := strings.Count(accessor, "icmp "); got != 1 {
		t.Fatalf("accessor comparisons = %d, want one cache check:\n%s", got, accessor)
	}
	if strings.Contains(accessor, "currentLocalContext") {
		t.Fatalf("accessor still depends on runtime context layout:\n%s", accessor)
	}
}

func llvmFunction(t *testing.T, ir, name string) string {
	t.Helper()
	markerAt := strings.Index(ir, `@"`+name+`"(`)
	if markerAt < 0 {
		markerAt = strings.Index(ir, `@`+name+`(`)
	}
	if markerAt < 0 {
		t.Fatalf("function %s not found:\n%s", name, ir)
	}
	start := strings.LastIndex(ir[:markerAt], "define ")
	if start < 0 {
		t.Fatalf("definition for %s not found", name)
	}
	end := strings.Index(ir[markerAt:], "\n}")
	if end < 0 {
		t.Fatalf("end of %s not found", name)
	}
	return ir[start : markerAt+end+2]
}

func TestLocalityPlansNativeTLSAndSharedPointerBlock(t *testing.T) {
	prog, ir := compileLocalitySource(t, `package locality

var backing int
func scalar() int { return 42 }
func pointer() *int { return &backing }

//llgo:tls
var tlsScalar = scalar()
//llgo:tls
var tlsPointer = pointer()
//llgo:gls
var glsScalar = scalar()
//llgo:gls
var glsPointer = pointer()

func values() (int, *int, int, *int) {
	return tlsScalar, tlsPointer, glsScalar, glsPointer
}
`)

	checks := map[string]struct {
		kind    llssa.Locality
		storage llssa.LocalStorage
	}{
		"tlsScalar":  {llssa.ThreadLocal, llssa.LocalStorageNativeTLS},
		"tlsPointer": {llssa.ThreadLocal, llssa.LocalStoragePackage},
		"glsScalar":  {llssa.GoroutineLocal, llssa.LocalStorageNativeTLS},
		"glsPointer": {llssa.GoroutineLocal, llssa.LocalStoragePackage},
	}
	for name, want := range checks {
		got, ok := prog.VariableLocality("example.com/locality." + name)
		if !ok || got.Locality != want.kind || got.LocalStorage != want.storage {
			t.Fatalf("%s metadata = %+v, %v", name, got, ok)
		}
	}
	for _, name := range []string{"tlsScalar", "glsScalar"} {
		if !strings.Contains(ir, `@"example.com/locality.`+name+`" = thread_local global i64`) {
			t.Fatalf("%s is not native TLS:\n%s", name, ir)
		}
	}
	for _, name := range []string{"tlsPointer", "glsPointer"} {
		if strings.Contains(ir, `@"example.com/locality.`+name+`" = thread_local`) {
			t.Fatalf("%s retained a pointer-bearing TLS global:\n%s", name, ir)
		}
	}
	if got := strings.Count(ir, `@"example.com/locality.__llgo_local_cache" = thread_local global i64 0`); got != 1 {
		t.Fatalf("package block caches = %d, want 1:\n%s", got, ir)
	}
	if got := strings.Count(ir, `call ptr @"github.com/xgo-dev/llgo/runtime/internal/runtime.LocalPackage"`); got != 1 {
		t.Fatalf("LocalPackage calls = %d, want one accessor definition:\n%s", got, ir)
	}
	values := llvmFunction(t, ir, "example.com/locality.values")
	if got := strings.Count(values, `call ptr @"example.com/locality.__llgo_local_block"()`); got != 1 {
		t.Fatalf("values package-base calls = %d, want 1:\n%s", got, values)
	}
	if got := strings.Count(values, `call void @"example.com/locality.__llgo_tls_init$ensure"()`); got != 1 {
		t.Fatalf("values TLS ensure calls = %d, want 1:\n%s", got, values)
	}
	if got := strings.Count(values, `call void @"example.com/locality.__llgo_gls_init$ensure"()`); got != 1 {
		t.Fatalf("values GLS ensure calls = %d, want 1:\n%s", got, values)
	}
	if !prog.NeedsLocalContext() {
		t.Fatal("pointer-bearing local variables did not enable a local context")
	}
}

func TestLocalityDebugInfoOnlyUsesFixedGlobals(t *testing.T) {
	_, ir := compileLocalitySourceWithOptions(t, `package locality

//llgo:tls
var direct int

//llgo:gls
var pointer *int

func values() (int, *int) { return direct, pointer }
`, Options{Debug: true, DebugSymbols: true})

	direct := `@"example.com/locality.direct" = thread_local global i64`
	start := strings.Index(ir, direct)
	if start < 0 {
		t.Fatalf("native TLS global not found:\n%s", ir)
	}
	end := strings.IndexByte(ir[start:], '\n')
	if end < 0 || !strings.Contains(ir[start:start+end], "!dbg") {
		t.Fatalf("native TLS global has no debug metadata:\n%s", ir[start:])
	}
	if strings.Contains(ir, `@"example.com/locality.pointer" =`) {
		t.Fatalf("package-local pointer was emitted as a fixed debug global:\n%s", ir)
	}
}

func TestLocalityInitializersPreserveGoOrderPerKind(t *testing.T) {
	_, ir := compileLocalitySource(t, `package locality

func mark(value int) int { return value }
//llgo:tls
var t0 = mark(0)
//llgo:gls
var g0 = mark(1)
//llgo:tls
var t1 = mark(2)
func values() (int, int, int) { return t0, t1, g0 }
`)
	tls := llvmFunction(t, ir, "example.com/locality.__llgo_tls_init")
	gls := llvmFunction(t, ir, "example.com/locality.__llgo_gls_init")
	first := strings.Index(tls, `__llgo_local_init_0`)
	second := strings.Index(tls, `__llgo_local_init_2`)
	if first < 0 || second < first || strings.Contains(tls, `__llgo_local_init_1`) {
		t.Fatalf("TLS dispatcher order is wrong:\n%s", tls)
	}
	if !strings.Contains(gls, `__llgo_local_init_1`) || strings.Contains(gls, `__llgo_local_init_0`) || strings.Contains(gls, `__llgo_local_init_2`) {
		t.Fatalf("GLS dispatcher contains the wrong helpers:\n%s", gls)
	}
	initBody := llvmFunction(t, ir, "example.com/locality.init")
	for _, guard := range []string{"__llgo_tls_init$guard", "__llgo_gls_init$guard"} {
		if !strings.Contains(initBody, `store i8 2, ptr @"example.com/locality.`+guard+`"`) {
			t.Fatalf("package init does not mark %s ready:\n%s", guard, initBody)
		}
	}
}

func TestDirectInitializerStillRequiresFailureContext(t *testing.T) {
	prog, ir := compileLocalitySource(t, `package locality
func value() int { return 1 }
//llgo:tls
var localValue = value()
func get() int { return localValue }
`)
	if !prog.NeedsLocalContext() {
		t.Fatal("initializer failure storage did not enable a local context")
	}
	if strings.Contains(ir, `__llgo_local_block`) {
		t.Fatalf("pointer-free package unexpectedly has a value block:\n%s", ir)
	}
	if !strings.Contains(ir, `@"example.com/locality.localValue" = thread_local global i64`) || !strings.Contains(ir, `EnsureLocalInitializer`) {
		t.Fatalf("direct initializer lowering is incomplete:\n%s", ir)
	}
}

func TestZeroValueDirectLocalsNeedNoContext(t *testing.T) {
	prog, ir := compileLocalitySource(t, `package locality
//llgo:tls
var threadValue int
//llgo:gls
var goroutineValue uintptr
func values() (int, uintptr) { return threadValue, goroutineValue }
`)
	if prog.NeedsLocalContext() {
		t.Fatal("pointer-free zero-value locals enabled a local context")
	}
	if strings.Contains(ir, `LocalPackage`) || strings.Contains(ir, `EnsureLocalInitializer`) {
		t.Fatalf("zero-value direct locals emitted cold-path support:\n%s", ir)
	}
}

func TestExportedFunctionInstallsLocalContext(t *testing.T) {
	_, ir := compileLocalitySource(t, `package locality
//llgo:gls
var pointer *int
//export Exported
func Exported(useLocal bool) *int {
	if useLocal { return pointer }
	return nil
}
`)
	exported := llvmFunction(t, ir, "Exported")
	if got := strings.Count(exported, "EnterLocalContext"); got != 1 {
		t.Fatalf("exported function context entries = %d, want 1:\n%s", got, exported)
	}
	if got := strings.Count(exported, "LeaveLocalContext"); got != 2 {
		t.Fatalf("exported function context leaves = %d, want 2:\n%s", got, exported)
	}
	assertTextOrder(t, exported,
		"EnterLocalContext",
		"__llgo_local_block",
		"LeaveLocalContext",
		"ret ptr",
	)
}

func TestExportedNativeTLSNeedsNoLocalContext(t *testing.T) {
	prog, ir := compileLocalitySource(t, `package locality
//llgo:tls
var scalar int
//export Exported
func Exported() int { return scalar }
`)
	if prog.NeedsLocalContext() {
		t.Fatal("zero-value native TLS enabled a local context")
	}
	exported := llvmFunction(t, ir, "Exported")
	if strings.Contains(exported, "LocalContext") {
		t.Fatalf("native-TLS-only export installed a local context:\n%s", exported)
	}
}

func assertTextOrder(t *testing.T, text string, wants ...string) {
	t.Helper()
	offset := 0
	for _, want := range wants {
		index := strings.Index(text[offset:], want)
		if index < 0 {
			t.Fatalf("%q not found after offset %d:\n%s", want, offset, text)
		}
		offset += index + len(want)
	}
}

func TestLocalityRejectsLinknameAlias(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "locality.go", `package locality
//llgo:gls
var target *int
//go:linkname alias example.com/locality.target
var alias *int
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	info := newLocalityTypeInfo()
	pkg, err := (&types.Config{}).Check("example.com/locality", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	prog := ssatest.NewProgram(t, nil)
	if err := ParsePkgSyntax(prog, fset, pkg, []*ast.File{file}); err != nil {
		t.Fatal(err)
	}
	if err := prog.ValidateLocalities(pkg.Path()); err == nil || !strings.Contains(err.Error(), "cannot reference local variable") {
		t.Fatalf("ValidateLocalities error = %v", err)
	}
}

func TestLocalityCrossPackageAccessUsesFunctions(t *testing.T) {
	fset := token.NewFileSet()
	parse := func(name, source string) *ast.File {
		file, err := parser.ParseFile(fset, name, source, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		return file
	}
	depFile := parse("dep.go", `package dep
func initialScalar() int { return 1 }
//llgo:tls
var scalar = initialScalar()
//llgo:gls
var pointer *int
func Values() (int, *int) { return scalar, pointer }
`)
	rootFile := parse("root.go", `package root
import "example.com/dep"
func Values() (int, *int) { return dep.Values() }
`)
	check := func(path string, files []*ast.File, imp types.Importer) (*types.Package, *types.Info) {
		info := newLocalityTypeInfo()
		pkg, err := (&types.Config{Importer: imp}).Check(path, fset, files, info)
		if err != nil {
			t.Fatal(err)
		}
		return pkg, info
	}
	depPkg, depInfo := check("example.com/dep", []*ast.File{depFile}, nil)
	rootPkg, rootInfo := check("example.com/root", []*ast.File{rootFile}, importerFunc(func(path string) (*types.Package, error) {
		if path == depPkg.Path() {
			return depPkg, nil
		}
		return nil, types.Error{Msg: "unexpected import " + path}
	}))

	prog := ssatest.NewProgram(t, nil)
	prog.TypeSizes(types.SizesFor("gc", runtime.GOARCH))
	prog.SetRuntime(localityRuntimePackage())
	for _, input := range []struct {
		pkg   *types.Package
		info  *types.Info
		files []*ast.File
	}{
		{depPkg, depInfo, []*ast.File{depFile}},
		{rootPkg, rootInfo, []*ast.File{rootFile}},
	} {
		if err := ParsePkgSyntax(prog, fset, input.pkg, input.files); err != nil {
			t.Fatal(err)
		}
		if err := PrepareLocalVariables(prog, fset, input.pkg, input.info, input.files); err != nil {
			t.Fatal(err)
		}
	}

	goProg := ssa.NewProgram(fset, ssa.SanityCheckFunctions)
	depSSA := goProg.CreatePackage(depPkg, []*ast.File{depFile}, depInfo, true)
	rootSSA := goProg.CreatePackage(rootPkg, []*ast.File{rootFile}, rootInfo, true)
	goProg.Build()
	if _, err := NewPackage(prog, depSSA, []*ast.File{depFile}); err != nil {
		t.Fatal(err)
	}
	root, err := NewPackage(prog, rootSSA, []*ast.File{rootFile})
	if err != nil {
		t.Fatal(err)
	}
	ir := root.String()
	if !strings.Contains(ir, `@"example.com/dep.Values"`) {
		t.Fatalf("root package did not call the dependency function:\n%s", ir)
	}
	for _, symbol := range []string{"example.com/dep.scalar", "example.com/dep.pointer", "example.com/dep.__llgo_"} {
		if strings.Contains(ir, symbol) {
			t.Fatalf("root package referenced dependency locality storage %q:\n%s", symbol, ir)
		}
	}
}

func TestParseRejectsLocalAliasInitializer(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "locality.go", `package locality
//llgo:tls
var target int
//go:linkname alias example.com/locality.target
//llgo:tls
var alias = 1
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	files := []*ast.File{file}
	pkg, err := (&types.Config{}).Check("example.com/locality", fset, files, newLocalityTypeInfo())
	if err != nil {
		t.Fatal(err)
	}
	prog := llssa.NewProgram(nil)
	if err := ParsePkgSyntax(prog, fset, pkg, files); err == nil || !strings.Contains(err.Error(), "cannot apply to a //go:linkname variable") {
		t.Fatalf("ParsePkgSyntax error = %v", err)
	}
}

func TestValidateLocalInitializers(t *testing.T) {
	pkg := types.NewPackage("example.com/locality", "locality")
	prog := ssatest.NewProgram(t, nil)
	name := llssa.FullName(pkg, "value")
	prog.SetLocalityInfo(name, llssa.LocalityInfo{Locality: llssa.ThreadLocal, HasInitializer: true})
	if err := validateLocalInitializers(prog, pkg); err == nil || !strings.Contains(err.Error(), "inconsistent initializer metadata") {
		t.Fatalf("validateLocalInitializers error = %v", err)
	}
	prog.SetLocalityInfo(name, llssa.LocalityInfo{Locality: llssa.ThreadLocal, HasInitializer: true, InitFunc: "example.com/locality.init", InitOrder: 1})
	if err := validateLocalInitializers(prog, pkg); err != nil {
		t.Fatal(err)
	}
}

func TestConcretePackageLocalityMetadata(t *testing.T) {
	std := types.NewPackage("runtime", "runtime")
	alt := types.NewPackage(abi.PatchPathPrefix+"runtime", "runtime")
	for _, pkg := range []*types.Package{std, alt} {
		pkg.Scope().Insert(types.NewVar(token.NoPos, pkg, "state", types.Typ[types.Uintptr]))
	}

	prog := ssatest.NewProgram(t, nil)
	name := "runtime.state"
	prog.DeclareLocality(std, "state", llssa.LocalityInfo{
		Locality:       llssa.ThreadLocal,
		HasInitializer: true,
	})
	prog.DeclareLocality(alt, "state", llssa.LocalityInfo{
		Locality: llssa.GoroutineLocal,
	})
	prog.SetLocalStorageFor(std, name, llssa.LocalStorageNativeTLS)
	prog.SetLocalStorageFor(alt, name, llssa.LocalStoragePackage)

	if err := validateLocalInitializers(prog, std); err == nil || !strings.Contains(err.Error(), "inconsistent initializer metadata") {
		t.Fatalf("standard validation error = %v", err)
	}
	if err := validateLocalInitializers(prog, alt); err != nil {
		t.Fatalf("alternate validation used standard metadata: %v", err)
	}
	plan, err := planLocalPackage(prog, alt)
	if err != nil {
		t.Fatal(err)
	}
	state, ok := plan.Lookup(name)
	if !ok || state.Info.Locality != llssa.GoroutineLocal {
		t.Fatalf("alternate plan state = %+v, %v", state, ok)
	}
}

func TestNewPackageReportsLocalityPreparationErrors(t *testing.T) {
	tests := []struct {
		name        string
		src         string
		parseSyntax bool
		wantError   string
	}{
		{
			name: "invalid directive",
			src: `package locality
//llgo:tls
func invalid() {}
`,
			wantError: "applies only to package-level var declarations",
		},
		{
			name: "unprepared initializer",
			src: `package locality
func initialValue() int { return 1 }
//llgo:tls
var value = initialValue()
`,
			parseSyntax: true,
			wantError:   "inconsistent initializer metadata",
		},
		{
			name: "linkname locality",
			src: `package locality
//llgo:tls
var target int
//go:linkname alias example.com/locality.target
//llgo:gls
var alias int
`,
			wantError: "cannot apply to a //go:linkname variable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "locality.go", tt.src, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			files := []*ast.File{file}
			info := newLocalityTypeInfo()
			pkg, err := (&types.Config{}).Check("example.com/locality", fset, files, info)
			if err != nil {
				t.Fatal(err)
			}
			goProg := ssa.NewProgram(fset, ssa.SanityCheckFunctions)
			ssaPkg := goProg.CreatePackage(pkg, files, info, true)
			ssaPkg.Build()
			prog := ssatest.NewProgram(t, nil)
			if tt.parseSyntax {
				if err := ParsePkgSyntax(prog, fset, pkg, files); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := NewPackage(prog, ssaPkg, files); err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("NewPackage error = %v, want %q", err, tt.wantError)
			}
		})
	}
}

func TestNewPackageValidatesPreloadedLocalityMetadata(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "locality.go", `package locality
var value int
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	files := []*ast.File{file}
	info := newLocalityTypeInfo()
	pkg, err := (&types.Config{}).Check("example.com/locality", fset, files, info)
	if err != nil {
		t.Fatal(err)
	}
	goProg := ssa.NewProgram(fset, ssa.SanityCheckFunctions)
	ssaPkg := goProg.CreatePackage(pkg, files, info, true)
	ssaPkg.Build()

	prog := ssatest.NewProgram(t, nil)
	name := llssa.FullName(pkg, "value")
	prog.SetLocalityInfo(name, llssa.LocalityInfo{Locality: llssa.ThreadLocal})
	prog.SetLinkname(name, name+"Alias")
	if _, err := NewPackage(prog, ssaPkg, files); err == nil || !strings.Contains(err.Error(), "cannot use go:linkname") {
		t.Fatalf("NewPackage locality validation error = %v", err)
	}
}

func TestParseRejectsExportedLocalVariable(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "locality.go", `package locality
//llgo:tls
var Value int
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	files := []*ast.File{file}
	pkg, err := (&types.Config{}).Check("example.com/locality", fset, files, newLocalityTypeInfo())
	if err != nil {
		t.Fatal(err)
	}
	prog := ssatest.NewProgram(t, nil)
	if err := ParsePkgSyntax(prog, fset, pkg, files); err == nil || !strings.Contains(err.Error(), "requires an unexported package variable") {
		t.Fatalf("ParsePkgSyntax error = %v", err)
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
	prog.SetLocalityInfo(llssa.FullName(pkg, value.Name()), llssa.LocalityInfo{Locality: llssa.ThreadLocal, HasInitializer: true})
	info.InitOrder = []*types.Initializer{{Lhs: []*types.Var{value}, Rhs: ast.NewIdent("rhs")}}
	if err := PrepareLocalVariables(prog, nil, pkg, info, nil); err == nil || !strings.Contains(err.Error(), "without syntax files") {
		t.Fatalf("PrepareLocalVariables without files error = %v", err)
	}
	(&context{}).initializeLocalGuards(nil)
}

func TestPrepareLocalVariablesRejectsInvalidMetadata(t *testing.T) {
	t.Run("missing object", func(t *testing.T) {
		prog := ssatest.NewProgram(t, nil)
		pkg := types.NewPackage("example.com/missing", "missing")
		prog.SetLocalityInfo(llssa.FullName(pkg, "Value"), llssa.LocalityInfo{Locality: llssa.ThreadLocal})
		if err := PrepareLocalVariables(prog, nil, pkg, &types.Info{}, nil); err == nil || !strings.Contains(err.Error(), "has no variable") {
			t.Fatalf("PrepareLocalVariables error = %v", err)
		}
	})
	t.Run("local linkname", func(t *testing.T) {
		prog := ssatest.NewProgram(t, nil)
		pkg := types.NewPackage("example.com/cycle", "cycle")
		first := llssa.FullName(pkg, "First")
		second := llssa.FullName(pkg, "Second")
		pkg.Scope().Insert(types.NewVar(token.NoPos, pkg, "First", types.Typ[types.Int]))
		prog.SetLocalityInfo(first, llssa.LocalityInfo{Locality: llssa.ThreadLocal})
		prog.SetLinkname(first, second)
		prog.SetLinkname(second, first)
		if err := PrepareLocalVariables(prog, nil, pkg, &types.Info{}, nil); err == nil || !strings.Contains(err.Error(), "cannot use go:linkname") {
			t.Fatalf("PrepareLocalVariables error = %v", err)
		}
	})
}

func TestPlanLocalPackageDiagnostics(t *testing.T) {
	prog := ssatest.NewProgram(t, nil)
	if plan, err := planLocalPackage(prog, nil); err != nil || len(plan.Variables) != 0 {
		t.Fatalf("nil package plan = %+v, %v", plan, err)
	}

	pkg := types.NewPackage("example.com/plan", "plan")
	ordinary := llssa.FullName(pkg, "Ordinary")
	pkg.Scope().Insert(types.NewVar(token.NoPos, pkg, "Ordinary", types.Typ[types.Int]))
	prog.SetLocalStorage(ordinary, llssa.LocalStorageNativeTLS)
	if plan, err := planLocalPackage(prog, pkg); err != nil || len(plan.Variables) != 0 {
		t.Fatalf("non-local metadata plan = %+v, %v", plan, err)
	}

	missing := llssa.FullName(pkg, "Missing")
	prog.SetLocalityInfo(missing, llssa.LocalityInfo{Locality: llssa.ThreadLocal})
	if _, err := planLocalPackage(prog, pkg); err == nil || !strings.Contains(err.Error(), "has no variable") {
		t.Fatalf("missing object plan error = %v", err)
	}
}

func TestLocalInitializerNameCollision(t *testing.T) {
	prog, _ := compileLocalitySource(t, `package locality
func __llgo_local_init_0() {}
//llgo:tls
var value = 1
`)
	info, ok := prog.VariableLocality("example.com/locality.value")
	if !ok || !strings.HasSuffix(info.InitFunc, ".__llgo_local_init_1") || info.InitOrder != 1 {
		t.Fatalf("value metadata = %+v, %v", info, ok)
	}
}

func TestNamedPointerLocalUsesPackageStorage(t *testing.T) {
	prog, ir := compileLocalitySource(t, `package locality
type Handle struct { Pointer *int }
func makeHandle() Handle { return Handle{} }
//llgo:tls
var value = makeHandle()
func get() Handle { return value }
`)
	info, ok := prog.VariableLocality("example.com/locality.value")
	if !ok || info.LocalStorage != llssa.LocalStoragePackage {
		t.Fatalf("named pointer metadata = %+v, %v", info, ok)
	}
	if !strings.Contains(ir, `call ptr @"example.com/locality.__llgo_local_block"()`) {
		t.Fatalf("named pointer did not use package storage:\n%s", ir)
	}
}
