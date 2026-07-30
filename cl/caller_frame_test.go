//go:build !llgo
// +build !llgo

package cl

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/goplus/gogen/packages"
	llssa "github.com/goplus/llgo/ssa"
	gossa "golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func parseCallerFrameFile(t *testing.T, src string) *ast.File {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "caller_frame.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	return file
}

func TestFilesUseRuntimeCaller(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want bool
	}{
		{
			name: "runtime selector",
			src: `package foo
import "runtime"
func f() { runtime.Caller(0) }
`,
			want: true,
		},
		{
			name: "runtime alias",
			src: `package foo
import rt "runtime"
func f() { rt.Callers(0, nil) }
`,
			want: true,
		},
		{
			name: "runtime debug stack",
			src: `package foo
import dbg "runtime/debug"
func f() { _ = dbg.Stack() }
`,
			want: true,
		},
		{
			name: "dot import",
			src: `package foo
import . "runtime"
func f() { Caller(0) }
`,
			want: true,
		},
		{
			name: "runtime FuncForPC only",
			src: `package foo
import "runtime"
func f() { _ = runtime.FuncForPC(0) }
`,
			want: false,
		},
		{
			name: "blank import",
			src: `package foo
import _ "runtime"
func f() {}
`,
			want: false,
		},
		{
			name: "non caller runtime selector",
			src: `package foo
import "runtime"
func f() { _ = runtime.GOOS }
`,
			want: false,
		},
		{
			name: "caller name without runtime import",
			src: `package foo
func f() { Caller(0) }
`,
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filesUseRuntimeCaller([]*ast.File{parseCallerFrameFile(t, tt.src)}); got != tt.want {
				t.Fatalf("filesUseRuntimeCaller() = %v, want %v", got, tt.want)
			}
		})
	}

	badImport := &ast.File{
		Imports: []*ast.ImportSpec{{
			Path: &ast.BasicLit{Kind: token.STRING, Value: "runtime"},
		}},
	}
	if filesUseRuntimeCaller([]*ast.File{badImport}) {
		t.Fatal("invalid import literal should not enable caller frame tracking")
	}
}

func buildCallerFrameSSAPackage(t *testing.T, pkgPath, src string) (*gossa.Package, []*ast.File) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "caller_frame_compile.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	files := []*ast.File{file}
	imp := packages.NewImporter(fset)
	mode := gossa.SanityCheckFunctions | gossa.InstantiateGenerics
	ssapkg, _, err := ssautil.BuildPackage(
		&types.Config{Importer: imp},
		fset,
		types.NewPackage(pkgPath, file.Name.Name),
		files,
		mode,
	)
	if err != nil {
		t.Fatal(err)
	}
	return ssapkg, files
}

func newLLSSAProgForTarget(t *testing.T, target *llssa.Target) llssa.Program {
	t.Helper()
	prog := llssa.NewProgram(target)
	prog.SetRuntime(func() *types.Package {
		rt, err := importer.For("source", nil).Import(llssa.PkgRuntime)
		if err != nil {
			t.Fatal("load runtime failed:", err)
		}
		return rt
	})
	if target != nil && target.GOARCH != "" {
		prog.TypeSizes(types.SizesFor("gc", target.GOARCH))
	}
	return prog
}

func newRuntimeCallerAnalysis(pkg *gossa.Package) *runtimeCallerAnalysis {
	funcs, trackable := collectRuntimeCallerFunctions(pkg)
	return &runtimeCallerAnalysis{
		pkg:       pkg,
		funcs:     funcs,
		trackable: trackable,
		callsites: collectRuntimeCallerCallsites(funcs),
		memo:      make(map[*gossa.Function]bool),
		visiting:  make(map[*gossa.Function]bool),
	}
}

func TestRuntimeCallerPackageDetection(t *testing.T) {
	ssapkg, _ := buildCallerFrameSSAPackage(t, "example.com/foo", `package foo
import "runtime"
import "runtime/debug"

type callerIface interface { Call() }
type callerImpl struct{}
type workerIface interface { Work() }
type workerImpl struct{}

func direct() { runtime.Caller(0) }
func indirect() { direct() }
func dynamic(f func()) { f() }
func dynamicCaller() { dynamic(direct) }
func (callerImpl) Call() { direct() }
func interfaceDispatch(c callerIface) { c.Call() }
func interfaceCaller(c callerIface) { interfaceDispatch(c) }
func closureLayer(next func()) func() { return func() { next() } }
func closureCaller() { closureLayer(closureLayer(direct))() }
func stack() { _ = debug.Stack() }
func anonOnly() { func() { runtime.Caller(0) }() }
func funcForPCOnly() { _ = runtime.FuncForPC(0) }
func leaf() {}
func callFunc(f func()) { f() }
func callFuncHot() { callFunc(leaf) }
func (workerImpl) Work() {}
func callWorker(w workerIface) { w.Work() }
func workerHot() { var w workerIface = workerImpl{}; callWorker(w) }
func plain() {}
`)
	callerCaches := NewCallerTracking()
	if !packageUsesRuntimeCaller(callerCaches, ssapkg) {
		t.Fatal("package should report runtime caller usage")
	}
	if !fnUsesRuntimeCaller(callerCaches, ssapkg.Func("direct")) {
		t.Fatal("direct runtime.Caller use should be detected")
	}
	if !fnUsesRuntimeCaller(callerCaches, ssapkg.Func("indirect")) {
		t.Fatal("transitive runtime.Caller use should be detected")
	}
	if !fnUsesRuntimeCaller(callerCaches, ssapkg.Func("stack")) {
		t.Fatal("runtime/debug.Stack use should be detected")
	}
	if !fnUsesRuntimeCaller(callerCaches, ssapkg.Func("anonOnly")) {
		t.Fatal("runtime caller use in anonymous functions should be detected")
	}
	if fnUsesRuntimeCaller(callerCaches, ssapkg.Func("plain")) {
		t.Fatal("plain function should not report runtime caller usage")
	}
	runtimeCallerFuncs := runtimeCallerFuncSet(callerCaches, ssapkg)
	for _, name := range []string{"dynamic", "dynamicCaller", "interfaceDispatch", "interfaceCaller", "closureLayer", "closureCaller"} {
		if !runtimeCallerFuncs[ssapkg.Func(name)] {
			t.Fatalf("%s should be tracked because dynamic calls may reach runtime stack APIs", name)
		}
	}
	for _, name := range []string{"leaf", "callFunc", "callFuncHot", "callWorker", "workerHot"} {
		if runtimeCallerFuncs[ssapkg.Func(name)] {
			t.Fatalf("%s should not be tracked when resolved dynamic targets do not reach runtime stack APIs", name)
		}
	}
	if runtimeCallerFuncs[ssapkg.Func("funcForPCOnly")] {
		t.Fatal("FuncForPC-only function should not need caller frame tracking")
	}
	if runtimeCallerFuncs[ssapkg.Func("plain")] {
		t.Fatal("plain function should not be tracked")
	}

	for _, name := range []string{"Caller", "Callers", "CallersFrames", "FuncForPC", "Stack"} {
		if !isRuntimeCallerName(name) {
			t.Fatalf("%s should be a runtime caller metadata function", name)
		}
	}
	if isRuntimeCallerFrameName("FuncForPC") {
		t.Fatal("FuncForPC should not require caller frame tracking")
	}
	if !isRuntimeCallerFrameName("Caller") {
		t.Fatal("Caller should require caller frame tracking")
	}
	if isRuntimeCallerName("Version") {
		t.Fatal("Version should not be a runtime caller metadata function")
	}

	rtpkg, _ := buildCallerFrameSSAPackage(t, "github.com/goplus/llgo/runtime/internal/lib/runtime", `package runtime
func Caller(skip int) (uintptr, string, int, bool) { return 0, "", 0, false }
func FuncForPC(pc uintptr) uintptr { return 0 }
`)
	if !isRuntimeCallerFunc(rtpkg.Func("Caller")) || !isRuntimeCallerLookupFunc(rtpkg.Func("Caller")) {
		t.Fatal("LLGo runtime lib Caller should be treated as runtime.Caller")
	}
	if !isRuntimeCallerFunc(rtpkg.Func("FuncForPC")) {
		t.Fatal("LLGo runtime lib FuncForPC should be treated as runtime metadata use")
	}
	if isRuntimeCallerFrameFunc(rtpkg.Func("FuncForPC")) {
		t.Fatal("FuncForPC should not require caller frame tracking")
	}
	if isRuntimeCallerLookupFunc(rtpkg.Func("FuncForPC")) {
		t.Fatal("FuncForPC should not consume caller lookup tokens")
	}
}

func TestRuntimeCallerAnalysisEdgeCases(t *testing.T) {
	callerCaches := NewCallerTracking()
	if fnUsesRuntimeCaller(callerCaches, nil) {
		t.Fatal("nil function should not use runtime caller metadata")
	}
	if fnUsesRuntimeCaller(callerCaches, &gossa.Function{}) {
		t.Fatal("function without a package should not use runtime caller metadata")
	}
	if runtimeCallerFuncSet(callerCaches, nil) != nil {
		t.Fatal("nil package should have no runtime caller set")
	}
	if fnHasDirectRuntimeCaller(nil) {
		t.Fatal("nil function should not have direct runtime caller use")
	}
	if functionBelongsToPackage(nil, nil) {
		t.Fatal("nil function/package should not belong to a package")
	}
	if typeBelongsToPackage(types.Typ[types.Int], nil) {
		t.Fatal("types should not belong to a nil package")
	}
	if isRuntimeCallerLookupFunc(nil) {
		t.Fatal("nil function should not be a runtime caller lookup")
	}
	called := false
	forEachCall(nil, func(*gossa.CallCommon) {
		called = true
	})
	if called {
		t.Fatal("forEachCall should ignore nil functions")
	}

	ssapkg, _ := buildCallerFrameSSAPackage(t, "example.com/foo", `package foo
import "runtime"

type I interface { Call() }
type J interface { Call() }
type T struct{}

func target() { runtime.Caller(0) }
func plain() {}
func call(fn func()) { fn() }
func callRuntime() { call(target) }
func (T) Call() { runtime.Caller(0) }
func viaStatic() { var i I = T{}; i.Call() }
func viaChange(j J) { var i I = j; i.Call() }
func viaParam(i I) { i.Call() }
func passInterface() { var i I = T{}; viaParam(i) }
`)
	analysis := newRuntimeCallerAnalysis(ssapkg)
	if analysis.fnMayReachRuntimeCaller(nil) {
		t.Fatal("nil function should not reach runtime caller metadata")
	}
	if targets, ok := analysis.functionValueTargets(ssapkg.Func("callRuntime"), ssapkg.Func("target")); !ok || !targets[ssapkg.Func("target")] {
		t.Fatal("static function value should resolve to its target")
	}
	if _, ok := analysis.functionValueTargets(ssapkg.Func("target"), nil); ok {
		t.Fatal("nil function value should be unresolved")
	}
	if _, ok := analysis.functionParamTargets(ssapkg.Func("call"), 99); ok {
		t.Fatal("out-of-range function argument should be unresolved")
	}
	callFn := ssapkg.Func("call")
	callParam := callFn.Params[0]
	callParams := callFn.Params
	callFn.Params = nil
	if _, ok := analysis.functionValueTargets(callFn, callParam); ok {
		t.Fatal("function parameter missing from Params should be unresolved")
	}
	callFn.Params = callParams

	iface := ssapkg.Pkg.Scope().Lookup("I").Type().Underlying().(*types.Interface)
	method := iface.Method(0)
	if !analysis.fnMayReachRuntimeCaller(ssapkg.Func("viaStatic")) {
		t.Fatal("static interface dispatch should reach runtime caller metadata")
	}
	if !analysis.fnMayReachRuntimeCaller(ssapkg.Func("viaChange")) {
		t.Fatal("changed interface dispatch should conservatively reach runtime caller metadata")
	}
	if targets, ok := analysis.interfaceMethodTargets(ssapkg.Func("viaParam"), ssapkg.Func("viaParam").Params[0], method); !ok || len(targets) == 0 {
		t.Fatal("interface parameter callsites should resolve concrete method targets")
	}
	analysis.callsites[ssapkg.Func("viaParam")] = []*gossa.CallCommon{{}}
	if _, ok := analysis.interfaceMethodTargets(ssapkg.Func("viaParam"), ssapkg.Func("viaParam").Params[0], method); ok {
		t.Fatal("out-of-range interface argument should be unresolved")
	}
	if _, ok := analysis.interfaceMethodTargets(ssapkg.Func("viaStatic"), nil, method); ok {
		t.Fatal("nil interface receiver should be unresolved")
	}
	if _, ok := analysis.staticInterfaceMethodTargets(&gossa.ChangeInterface{}, method); ok {
		t.Fatal("empty interface conversion should be unresolved")
	}
	viaParam := ssapkg.Func("viaParam")
	interfaceParam := viaParam.Params[0]
	interfaceParams := viaParam.Params
	viaParam.Params = nil
	if _, ok := analysis.interfaceMethodTargets(viaParam, interfaceParam, method); ok {
		t.Fatal("interface parameter missing from Params should be unresolved")
	}
	viaParam.Params = interfaceParams
	if _, ok := analysis.methodTargetsForType(nil, nil); ok {
		t.Fatal("nil method lookup should be unresolved")
	}
	other := types.NewFunc(token.NoPos, ssapkg.Pkg, "Other", nil)
	if _, ok := analysis.methodTargetsForType(ssapkg.Type("T").Type(), other); ok {
		t.Fatal("missing interface method should be unresolved")
	}
	if idx, ok := parameterIndex(ssapkg.Func("target"), nil); ok || idx != 0 {
		t.Fatal("nil parameter should not be found")
	}

	methodOnlyPkg, _ := buildCallerFrameSSAPackage(t, "example.com/methodonly", `package methodonly
import "runtime"

type T struct{}
func (T) Call() { runtime.Caller(0) }
var _ = T{}
`)
	methodOnlySet := runtimeCallerFuncSet(NewCallerTracking(), methodOnlyPkg)
	if methodOnlySet == nil {
		t.Fatal("a method calling runtime.Caller must be tracked (slog.(*Logger).Info escaped exactly this way)")
	}
	foundMethod := false
	for fn := range methodOnlySet {
		if fn.Name() == "Call" {
			foundMethod = true
		}
	}
	if !foundMethod {
		t.Fatal("method-only runtime caller set should contain the method itself")
	}
}

func TestCallerFrameTrackingEligibility(t *testing.T) {
	if (&context{}).shouldTrackCallerFrames() {
		t.Fatal("missing compiler state should not track caller frames")
	}
	var nilContext *context
	if nilContext.shouldTrackCallerFrames() {
		t.Fatal("nil context should not track caller frames")
	}

	tests := []struct {
		name       string
		pkgPath    string
		track      bool
		targetName string
		goarch     string
		want       bool
	}{
		{name: "enabled user package", pkgPath: "example.com/foo", track: true, want: true},
		{name: "disabled flag", pkgPath: "example.com/foo", want: false},
		{name: "named target", pkgPath: "example.com/foo", track: true, targetName: "esp32", want: false},
		{name: "wasm", pkgPath: "example.com/foo", track: true, goarch: "wasm", want: false},
		{name: "stdlib", pkgPath: "fmt", track: true, want: true},
		{name: "runtime", pkgPath: "runtime", track: true, want: false},
		{name: "llgo runtime", pkgPath: llssa.PkgRuntime, track: true, want: false},
		{name: "llgo runtime internal", pkgPath: "github.com/goplus/llgo/runtime/internal/foo", track: true, want: false},
		{name: "command line package", pkgPath: "command-line-arguments", track: true, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ssapkg, _ := buildCallerFrameSSAPackage(t, tt.pkgPath, `package foo
import "runtime"
func f() { runtime.Caller(0) }
`)
			prog := llssa.NewProgram(nil)
			if tt.targetName != "" {
				prog.Target().Target = tt.targetName
			}
			if tt.goarch != "" {
				prog.Target().GOARCH = tt.goarch
			}
			pkg := prog.NewPackage("foo", tt.pkgPath)
			fn := pkg.NewFunc("f", llssa.NoArgsNoRet, llssa.InGo)
			goFn := ssapkg.Func("f")
			ctx := &context{
				prog:               prog,
				pkg:                pkg,
				fn:                 fn,
				goFn:               goFn,
				trackCallerFrames:  tt.track,
				runtimeCallerFuncs: runtimeCallerFuncSet(NewCallerTracking(), ssapkg),
			}
			if got := ctx.shouldTrackCallerFrames(); got != tt.want {
				t.Fatalf("shouldTrackCallerFrames() = %v, want %v", got, tt.want)
			}
		})
	}

	// Tracking applies uniformly; only the runtime core is excluded.
	if !canTrackCallerFramesForPackage("net/http") {
		t.Fatal("stdlib packages must be trackable like any other code")
	}
	if canTrackCallerFramesForPackage("runtime") ||
		canTrackCallerFramesForPackage(llssa.PkgRuntime) ||
		canTrackCallerFramesForPackage("github.com/goplus/llgo/runtime/internal/lib") {
		t.Fatal("runtime core must stay untracked")
	}
}

func TestRuntimeFrameNameNormalization(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "command-line-arguments.main", want: "main.main"},
		{in: "example.com/foo.f$1", want: "example.com/foo.f.func1"},
		{in: "example.com/foo.f", want: "example.com/foo.f"},
		{in: "example.com/foo.f$", want: "example.com/foo.f$"},
		{in: "example.com/foo.f$inner", want: "example.com/foo.f$inner"},
	}
	for _, tt := range tests {
		if got := runtimeFrameName(tt.in); got != tt.want {
			t.Fatalf("runtimeFrameName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}

	if got := (*context)(nil).runtimeCallerFrameName(); got != "" {
		t.Fatalf("nil context runtimeCallerFrameName() = %q, want empty", got)
	}
	if got := (&context{}).runtimeCallerFrameName(); got != "" {
		t.Fatalf("empty context runtimeCallerFrameName() = %q, want empty", got)
	}
	prog := newLLSSAProg(t)
	pkg := prog.NewPackage("main", "command-line-arguments")
	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	ctx := &context{fn: pkg.NewFuncEx("command-line-arguments.f$1", sig, llssa.InGo, false, false)}
	if got, want := ctx.runtimeCallerFrameName(), "main.f.func1"; got != want {
		t.Fatalf("fallback runtimeCallerFrameName() = %q, want %q", got, want)
	}
}

func TestCompileRuntimeCallerFrameInstrumentation(t *testing.T) {
	t.Setenv("LLGO_SHADOW_STACK", "1")
	ssapkg, files := buildCallerFrameSSAPackage(t, "example.com/foo", `package foo
import "runtime/debug"

func f() {
	_ = debug.Stack()
}
`)
	prog := newLLSSAProg(t)
	pkg, err := NewPackage(prog, ssapkg, files)
	if err != nil {
		t.Fatal(err)
	}
	ir := pkg.Module().String()
	for _, want := range []string{
		"RecordCallerLocation",
		`c"example.com/foo.f`,
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("compiled caller-frame IR missing %q:\n%s", want, ir)
		}
	}
	for _, old := range []string{"PushCallerFrame", "SetCallerLine", "PopCallerFrame"} {
		if strings.Contains(ir, old) {
			t.Fatalf("compiled caller-frame IR still contains old %q instrumentation:\n%s", old, ir)
		}
	}
}

func TestCompileRuntimeCallerPCLineMetadata(t *testing.T) {
	ssapkg, files := buildCallerFrameSSAPackage(t, "example.com/foo", `package foo
import "runtime"

func top() {
	runtime.Caller(0)
	leaf()
}

func leaf() {}
`)
	prog := newLLSSAProg(t)
	prog.Target().GOOS = "linux"
	prog.Target().GOARCH = "amd64"
	prog.EnableFuncInfoMetadata(true)
	prog.EnableFuncInfoSites(true)
	pkg, err := NewPackage(prog, ssapkg, files)
	if err != nil {
		t.Fatal(err)
	}
	ir := pkg.Module().String()
	for _, want := range []string{
		`!llgo.pcline = !{!`,
		`!"example.com/foo.top"`,
		`!"caller_frame_compile.go"`,
		"__llgo_pcsite_",
		"${:uid}",
		`.pushsection llgo_pcline,\22awo\22,@progbits`,
		`.quad __llgo_pcsite_`,
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("missing pcline metadata %s:\n%s", want, ir)
		}
	}
	for _, line := range strings.Split(ir, "\n") {
		if strings.Contains(line, "!llgo.pcline") || strings.Contains(line, `!"example.com/foo.top"`) {
			if strings.Contains(line, `ptr @`) {
				t.Fatalf("pcline metadata should use symbol strings, not function pointers:\n%s", line)
			}
		}
	}
}

func TestCompileRuntimeCallerPCLineMetadata32Bit(t *testing.T) {
	ssapkg, files := buildCallerFrameSSAPackage(t, "example.com/foo", `package foo
import "runtime"

func top() {
	runtime.Caller(0)
}
`)
	prog := newLLSSAProgForTarget(t, &llssa.Target{GOOS: "linux", GOARCH: "386"})
	prog.EnableFuncInfoMetadata(true)
	prog.EnableFuncInfoSites(true)
	pkg, err := NewPackage(prog, ssapkg, files)
	if err != nil {
		t.Fatal(err)
	}
	ir := pkg.Module().String()
	for _, want := range []string{
		`.p2align 2`,
		`.long __llgo_pcsite_`,
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("missing 32-bit pcline asm %q:\n%s", want, ir)
		}
	}
}

func TestCompileRuntimeCallerPCLineUsesLocalAnchor(t *testing.T) {
	ssapkg, files := buildCallerFrameSSAPackage(t, "example.com/foo", `package foo
import "runtime"

func top() {
	func() {
		runtime.Caller(0)
	}()
}
`)
	prog := newLLSSAProg(t)
	prog.Target().GOOS = "linux"
	prog.Target().GOARCH = "amd64"
	prog.EnableFuncInfoMetadata(true)
	prog.EnableFuncInfoSites(true)
	pkg, err := NewPackage(prog, ssapkg, files)
	if err != nil {
		t.Fatal(err)
	}
	ir := pkg.Module().String()
	if !strings.Contains(ir, `!"example.com/foo.top$1"`) {
		t.Fatalf("metadata should keep the original Go symbol name:\n%s", ir)
	}
	found := false
	for _, line := range strings.Split(ir, "\n") {
		if strings.Contains(line, `.pushsection llgo_pcline`) {
			found = true
			if !strings.Contains(line, `__llgo_pcsite_`) || strings.Contains(line, `example.com/foo.top`) {
				t.Fatalf("pcline section should use its local PC label as the SHF_LINK_ORDER anchor:\n%s", line)
			}
		}
	}
	if !found {
		t.Fatalf("missing pcline section:\n%s", ir)
	}
}

func TestRuntimeCallerInstrumentationEdgeCases(t *testing.T) {
	ssapkg, _ := buildCallerFrameSSAPackage(t, "example.com/foo", `package foo
import "runtime"

func top() {
	runtime.Caller(0)
}
`)
	prog := newLLSSAProgForTarget(t, &llssa.Target{GOOS: "linux", GOARCH: "amd64"})
	prog.EnableFuncInfoMetadata(true)
	prog.EnableFuncInfoSites(true)
	pkg := prog.NewPackage("foo", "example.com/foo")
	fn := pkg.NewFunc("example.com/foo.top", llssa.NoArgsNoRet, llssa.InGo)
	ctx := &context{
		prog:               prog,
		pkg:                pkg,
		fn:                 fn,
		goFn:               ssapkg.Func("top"),
		fset:               token.NewFileSet(),
		trackCallerFrames:  true,
		runtimeCallerFuncs: runtimeCallerFuncSet(NewCallerTracking(), ssapkg),
	}
	var b llssa.Builder
	ctx.pushCallerLocationFrame(b, nil)
	ctx.recordRuntimeLocation(b, token.NoPos, "RecordCallerLocation")
	ctx.emitPCLineLabel(b, token.NoPos)
	ctx.popCallerLocationFrame(b)

	if pos := (&context{}).funcInfoPosition(nil); pos.IsValid() {
		t.Fatal("nil function should have no funcinfo position")
	}
	if canEmitPCLineLabelsForTarget(nil) {
		t.Fatal("nil target should not emit pc-line labels")
	}
	if canEmitPCLineLabelsForTarget(&llssa.Target{GOOS: "linux", GOARCH: "wasm"}) {
		t.Fatal("wasm target should not emit pc-line labels")
	}
	if canEmitPCLineLabelsForTarget(&llssa.Target{GOOS: "linux", GOARCH: "amd64", Target: "esp32"}) {
		t.Fatal("named target should not emit pc-line labels")
	}
	if got, want := asmQuoteSymbol(`a\b"c$d`), `"a\\b\"c$$d"`; got != want {
		t.Fatalf("asmQuoteSymbol() = %q, want %q", got, want)
	}
}

func TestCompileRuntimeCallerPCLineMetadataSitesDisabled(t *testing.T) {
	ssapkg, files := buildCallerFrameSSAPackage(t, "example.com/foo", `package foo
import "runtime"

func top() {
	runtime.Caller(0)
}
`)
	prog := newLLSSAProg(t)
	prog.Target().GOOS = "linux"
	prog.Target().GOARCH = "amd64"
	prog.EnableFuncInfoMetadata(true)
	prog.EnableFuncInfoSites(false)
	pkg, err := NewPackage(prog, ssapkg, files)
	if err != nil {
		t.Fatal(err)
	}
	ir := pkg.Module().String()
	// Funcinfo metadata still flows...
	if !strings.Contains(ir, llssa.FuncInfoMetadataName) {
		t.Fatalf("sites disabled should keep funcinfo metadata:\n%s", ir)
	}
	// ...but no pc-line site labels are emitted.
	for _, bad := range []string{"__llgo_pcsite_", ".pushsection llgo_pcline", "!llgo.pcline"} {
		if strings.Contains(ir, bad) {
			t.Fatalf("sites disabled should not emit pc-line sites, found %q:\n%s", bad, ir)
		}
	}
}

func TestCompileRuntimeCallerPCLineMetadataOnDarwin(t *testing.T) {
	ssapkg, files := buildCallerFrameSSAPackage(t, "example.com/foo", `package foo
import "runtime"

func top() {
	runtime.Caller(0)
}
`)
	prog := newLLSSAProg(t)
	prog.Target().GOOS = "darwin"
	prog.Target().GOARCH = "arm64"
	prog.EnableFuncInfoMetadata(true)
	prog.EnableFuncInfoSites(true)
	pkg, err := NewPackage(prog, ssapkg, files)
	if err != nil {
		t.Fatal(err)
	}
	ir := pkg.Module().String()
	for _, want := range []string{
		`!llgo.pcline`,
		"__llgo_pcsite_",
		`.pushsection __DATA,__llgo_pcl`,
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("darwin should emit Mach-O pc-site labels, missing %q:\n%s", want, ir)
		}
	}
	if strings.Contains(ir, `.pushsection llgo_pcline`) {
		t.Fatalf("darwin must not use the ELF pcline section syntax:\n%s", ir)
	}
}

func TestCompileRuntimeCallerFrameUsesGoNameForLinkname(t *testing.T) {
	t.Setenv("LLGO_SHADOW_STACK", "1")
	ssapkg, files := buildCallerFrameSSAPackage(t, "command-line-arguments", `package main
import "runtime"

func renamedPC() uintptr {
	pc, _, _, _ := runtime.Caller(0)
	return pc
}
`)
	prog := newLLSSAProg(t)
	prog.SetLinkname("command-line-arguments.renamedPC", "main.renamedPCSymbol")
	pkg, err := NewPackage(prog, ssapkg, files)
	if err != nil {
		t.Fatal(err)
	}
	ir := pkg.Module().String()
	if !strings.Contains(ir, `c"main.renamedPC"`) {
		t.Fatalf("compiled caller-frame IR missing source function name:\n%s", ir)
	}
	if strings.Contains(ir, `c"main.renamedPCSymbol"`) {
		t.Fatalf("compiled caller-frame IR used linkname target as runtime function name:\n%s", ir)
	}
}

func TestCompileRuntimeCallerFrameInstrumentationSkipped(t *testing.T) {
	ssapkg, files := buildCallerFrameSSAPackage(t, "example.com/foo", `package foo
import "runtime"

func f() {
	runtime.Caller(0)
}
`)
	prog := newLLSSAProg(t)
	prog.Target().Target = "esp32"
	pkg, err := NewPackage(prog, ssapkg, files)
	if err != nil {
		t.Fatal(err)
	}
	if ir := pkg.Module().String(); strings.Contains(ir, "RecordCallerLocation") || strings.Contains(ir, "RecordPanicLocation") {
		t.Fatalf("target builds should not emit caller location tracking:\n%s", ir)
	}

	ssapkg, files = buildCallerFrameSSAPackage(t, "example.com/foo", `package foo
func f() {}
`)
	prog = newLLSSAProg(t)
	pkg, err = NewPackage(prog, ssapkg, files)
	if err != nil {
		t.Fatal(err)
	}
	if ir := pkg.Module().String(); strings.Contains(ir, "RecordCallerLocation") || strings.Contains(ir, "RecordPanicLocation") {
		t.Fatalf("packages without runtime stack APIs should not emit caller location tracking:\n%s", ir)
	}

	ssapkg, files = buildCallerFrameSSAPackage(t, "example.com/foo", `package foo
import "runtime"
func f() { _ = runtime.FuncForPC(0) }
`)
	prog = newLLSSAProg(t)
	prog.Target().GOOS = "linux"
	prog.Target().GOARCH = "amd64"
	prog.EnableFuncInfoMetadata(true)
	prog.EnableFuncInfoSites(true)
	pkg, err = NewPackage(prog, ssapkg, files)
	if err != nil {
		t.Fatal(err)
	}
	ir := pkg.Module().String()
	for _, bad := range []string{"RecordCallerLocation", "RecordPanicLocation", "PushCallerLocationFrame", `!llgo.pcline`} {
		if strings.Contains(ir, bad) {
			t.Fatalf("FuncForPC-only packages should not emit caller frame tracking %q:\n%s", bad, ir)
		}
	}
	if !strings.Contains(ir, `!llgo.funcinfo = !{!`) {
		t.Fatalf("FuncForPC-only packages should still emit funcinfo metadata:\n%s", ir)
	}
}

func TestCompileRuntimeCallerLocationOnlyForRuntimePaths(t *testing.T) {
	t.Setenv("LLGO_SHADOW_STACK", "1")
	ssapkg, files := buildCallerFrameSSAPackage(t, "example.com/foo", `package foo
import "runtime"

func helper() {}

func f() {
	helper()
	runtime.Caller(0)
}
`)
	prog := newLLSSAProg(t)
	pkg, err := NewPackage(prog, ssapkg, files)
	if err != nil {
		t.Fatal(err)
	}
	ir := pkg.Module().String()
	if !strings.Contains(ir, "RecordCallerLocation") {
		t.Fatalf("runtime.Caller should record caller location:\n%s", ir)
	}
	if strings.Contains(ir, "SetCallerLine") || strings.Contains(ir, "PushCallerFrame") {
		t.Fatalf("caller location tracking should not emit old TLS instrumentation:\n%s", ir)
	}
}

// importerFunc adapts a lookup function to types.Importer.
type importerFunc func(string) (*types.Package, error)

func (f importerFunc) Import(path string) (*types.Package, error) { return f(path) }

// buildCallerFrameSSAProgram builds dep and root packages with bodies in a
// single SSA program, so cross-package analysis (runtimeCallerFuncSet
// criterion 2) can consult the callee package's own base set.
func buildCallerFrameSSAProgram(t *testing.T, depPath, depSrc, rootPath, rootSrc string) (dep, root *gossa.Package) {
	t.Helper()
	fset := token.NewFileSet()
	parse := func(name, src string) *ast.File {
		file, err := parser.ParseFile(fset, name, src, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		return file
	}
	depFile := parse("dep.go", depSrc)
	rootFile := parse("root.go", rootSrc)
	base := packages.NewImporter(fset)
	prog := gossa.NewProgram(fset, gossa.SanityCheckFunctions|gossa.InstantiateGenerics)
	created := map[*types.Package]bool{}
	var createDeps func(p *types.Package)
	createDeps = func(p *types.Package) {
		if created[p] {
			return
		}
		created[p] = true
		for _, imp := range p.Imports() {
			createDeps(imp)
		}
		prog.CreatePackage(p, nil, nil, true)
	}
	check := func(path string, file *ast.File, imp types.Importer) (*types.Package, *types.Info) {
		info := &types.Info{
			Types:      map[ast.Expr]types.TypeAndValue{},
			Defs:       map[*ast.Ident]types.Object{},
			Uses:       map[*ast.Ident]types.Object{},
			Implicits:  map[ast.Node]types.Object{},
			Scopes:     map[ast.Node]*types.Scope{},
			Selections: map[*ast.SelectorExpr]*types.Selection{},
			Instances:  map[*ast.Ident]types.Instance{},
		}
		pkg := types.NewPackage(path, file.Name.Name)
		if err := types.NewChecker(&types.Config{Importer: imp}, fset, pkg, info).Files([]*ast.File{file}); err != nil {
			t.Fatal(err)
		}
		return pkg, info
	}
	depPkgT, depInfo := check(depPath, depFile, base)
	for _, p := range depPkgT.Imports() {
		createDeps(p)
	}
	depSSA := prog.CreatePackage(depPkgT, []*ast.File{depFile}, depInfo, true)
	created[depPkgT] = true
	rootImp := importerFunc(func(path string) (*types.Package, error) {
		if path == depPath {
			return depPkgT, nil
		}
		return base.Import(path)
	})
	rootPkgT, rootInfo := check(rootPath, rootFile, rootImp)
	for _, p := range rootPkgT.Imports() {
		createDeps(p)
	}
	rootSSA := prog.CreatePackage(rootPkgT, []*ast.File{rootFile}, rootInfo, true)
	prog.Build()
	return depSSA, rootSSA
}

// Criterion 2: calling another package's pc-consuming function makes the
// caller trackable — decided by that package's analysis, not by name.
func TestRuntimeCallerFuncSetCrossPackage(t *testing.T) {
	depSSA, rootSSA := buildCallerFrameSSAProgram(t,
		"example.com/dep", `package dep

import "runtime"

func Where() bool {
	_, _, _, ok := runtime.Caller(0)
	return ok
}

func Quiet() int { return 1 }
`,
		"example.com/root", `package root

import "example.com/dep"

func Logs() bool { return dep.Where() }

func Plain() int { return dep.Quiet() }
`)
	crossCaches := NewCallerTracking()
	if !runtimeCallerBaseSet(crossCaches, depSSA)[depSSA.Func("Where")] {
		t.Fatal("dep.Where must be in its own package's base set")
	}
	set := runtimeCallerFuncSet(crossCaches, rootSSA)
	if !set[rootSSA.Func("Logs")] {
		t.Fatal("caller of a pc-consuming function must be tracked (criterion 2)")
	}
	if set[rootSSA.Func("Plain")] {
		t.Fatal("caller of a quiet function must not be tracked")
	}
}

// Criteria 3 and 4: program-unique frames (main.main, init) and
// //go:noinline functions are pinned; plain helpers are not.
func TestRuntimeCallerFuncSetPinnedFrames(t *testing.T) {
	ssapkg, _ := buildCallerFrameSSAPackage(t, "example.com/cmd", `package main

func main() { helper(); pinned() }

func init() { helper() }

//go:noinline
func pinned() {}

func helper() {}
`)
	set := runtimeCallerFuncSet(NewCallerTracking(), ssapkg)
	for _, name := range []string{"main", "init", "pinned"} {
		if !set[ssapkg.Func(name)] {
			t.Fatalf("%s must be pinned in the tracking set", name)
		}
	}
	if set[ssapkg.Func("helper")] {
		t.Fatal("plain helper must not be tracked")
	}
}

// Display names follow gc's reporting conventions regardless of the module
// path; linker symbols are untouched.
func TestFuncInfoDisplayName(t *testing.T) {
	mainPkg := types.NewPackage("example.com/cmd", "main")
	libPkg := types.NewPackage("example.com/lib", "lib")
	cases := []struct {
		pkg      *types.Package
		in, want string
	}{
		{mainPkg, "example.com/cmd.main", "main.main"},
		{mainPkg, "example.com/cmd.main$2", "main.main.func2"},
		{mainPkg, "other/path.f", "other/path.f"},
		{libPkg, "example.com/lib.f$1", "example.com/lib.f.func1"},
		{nil, "plain.f$x", "plain.f$x"},
	}
	for _, c := range cases {
		if got := funcInfoDisplayName(c.pkg, c.in); got != c.want {
			t.Fatalf("funcInfoDisplayName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// //line directive filenames follow gc's spelling: relative directives
// stay relative to the declaring file's directory, empty directives report
// "??", absolute ones and directive-free positions pass through.
func TestDirectiveFilename(t *testing.T) {
	fset := token.NewFileSet()
	src := "package p\nfunc D() {}\n//line /abs/dir/renamed.go:10\nfunc A() {}\n//line rel.go:20\nfunc B() {}\n//line :30\nfunc C() {}\n"
	file, err := parser.ParseFile(fset, "/work/pkg/orig.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	pos := func(i int) token.Pos { return file.Decls[i].Pos() }
	cases := []struct {
		pos  token.Pos
		want string
	}{
		{pos(0), "/work/pkg/orig.go"},
		{pos(1), "/abs/dir/renamed.go"},
		{pos(2), "rel.go"},
		{pos(3), "??"},
	}
	for i, c := range cases {
		adjusted := fset.Position(c.pos).Filename
		if c.want == "rel.go" {
			// The package loader hands cl an absolute expansion; emulate it.
			adjusted = "/work/pkg/rel.go"
		}
		if got := directiveFilename(fset, c.pos, adjusted); got != c.want {
			t.Fatalf("case %d: directiveFilename(%q) = %q, want %q", i, adjusted, got, c.want)
		}
	}
	if got := directiveFilename(fset, token.NoPos, "x.go"); got != "x.go" {
		t.Fatal("NoPos must pass through")
	}
	if got := directiveFilename(nil, pos(0), "x.go"); got != "x.go" {
		t.Fatal("nil fset must pass through")
	}
}
