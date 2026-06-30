//go:build !llgo
// +build !llgo

package cl

import (
	"go/ast"
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
func f() { _ = FuncForPC(0) }
`,
			want: true,
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
func anonOnly() { func() { runtime.FuncForPC(0) }() }
func leaf() {}
func callFunc(f func()) { f() }
func callFuncHot() { callFunc(leaf) }
func (workerImpl) Work() {}
func callWorker(w workerIface) { w.Work() }
func workerHot() { var w workerIface = workerImpl{}; callWorker(w) }
func plain() {}
`)
	if !packageUsesRuntimeCaller(ssapkg) {
		t.Fatal("package should report runtime caller usage")
	}
	if !fnUsesRuntimeCaller(ssapkg.Func("direct")) {
		t.Fatal("direct runtime.Caller use should be detected")
	}
	if !fnUsesRuntimeCaller(ssapkg.Func("indirect")) {
		t.Fatal("transitive runtime.Caller use should be detected")
	}
	if !fnUsesRuntimeCaller(ssapkg.Func("stack")) {
		t.Fatal("runtime/debug.Stack use should be detected")
	}
	if !fnUsesRuntimeCaller(ssapkg.Func("anonOnly")) {
		t.Fatal("runtime caller use in anonymous functions should be detected")
	}
	if fnUsesRuntimeCaller(ssapkg.Func("plain")) {
		t.Fatal("plain function should not report runtime caller usage")
	}
	runtimeCallerFuncs := runtimeCallerFuncSet(ssapkg)
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
	if runtimeCallerFuncs[ssapkg.Func("plain")] {
		t.Fatal("plain function should not be tracked")
	}

	for _, name := range []string{"Caller", "Callers", "CallersFrames", "FuncForPC", "Stack"} {
		if !isRuntimeCallerName(name) {
			t.Fatalf("%s should be a runtime caller metadata function", name)
		}
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
	if isRuntimeCallerLookupFunc(rtpkg.Func("FuncForPC")) {
		t.Fatal("FuncForPC should not consume caller lookup tokens")
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
		{name: "stdlib", pkgPath: "fmt", track: true, want: false},
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
				runtimeCallerFuncs: runtimeCallerFuncSet(ssapkg),
			}
			if got := ctx.shouldTrackCallerFrames(); got != tt.want {
				t.Fatalf("shouldTrackCallerFrames() = %v, want %v", got, tt.want)
			}
		})
	}

	if canTrackCallerFramesForPackage("net/http") {
		t.Fatal("stdlib paths without dots should not track caller frames")
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

func TestCompileRuntimeCallerFrameUsesGoNameForLinkname(t *testing.T) {
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
}

func TestCompileRuntimeCallerLocationOnlyForRuntimePaths(t *testing.T) {
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
