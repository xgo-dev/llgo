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

func TestRuntimeCallerPackageDetection(t *testing.T) {
	ssapkg, _, _ := buildGoSSAPkg(t, `package foo
import "runtime"

func direct() { runtime.Caller(0) }

func anonOnly() {
	func() { runtime.FuncForPC(0) }()
}

func plain() {}
`)
	if !packageUsesRuntimeCaller(ssapkg) {
		t.Fatal("package should report runtime caller usage")
	}
	if packageUsesRuntimeCaller(nil) {
		t.Fatal("nil package should not report runtime caller usage")
	}
	if !fnUsesRuntimeCaller(ssapkg.Func("direct")) {
		t.Fatal("direct runtime.Caller use should be detected")
	}
	if !fnUsesRuntimeCaller(ssapkg.Func("anonOnly")) {
		t.Fatal("runtime caller use in anonymous functions should be detected")
	}
	if fnUsesRuntimeCaller(ssapkg.Func("plain")) {
		t.Fatal("plain function should not report runtime caller usage")
	}
	if fnUsesRuntimeCaller(nil) {
		t.Fatal("nil function should not report runtime caller usage")
	}

	directCall := findStaticCallByName(t, ssapkg.Func("direct"), "Caller")
	if !isRuntimeCallerFunc(directCall.Common().StaticCallee()) {
		t.Fatal("runtime.Caller static callee should be recognized")
	}
	if isRuntimeCallerFunc(ssapkg.Func("plain")) {
		t.Fatal("non-runtime function should not be recognized as runtime caller")
	}
	if isRuntimeCallerFunc(nil) {
		t.Fatal("nil function should not be recognized as runtime caller")
	}

	for _, name := range []string{"Caller", "Callers", "CallersFrames", "FuncForPC"} {
		if !isRuntimeCallerName(name) {
			t.Fatalf("%s should be a runtime caller metadata function", name)
		}
	}
	if isRuntimeCallerName("Version") {
		t.Fatal("Version should not be a runtime caller metadata function")
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
		{name: "llgo runtime internal", pkgPath: "github.com/goplus/llgo/runtime/internal/libc", track: true, want: false},
		{name: "command line package", pkgPath: "command-line-arguments", track: true, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog := llssa.NewProgram(nil)
			if tt.targetName != "" {
				prog.Target().Target = tt.targetName
			}
			if tt.goarch != "" {
				prog.Target().GOARCH = tt.goarch
			}
			pkg := prog.NewPackage("foo", tt.pkgPath)
			fn := pkg.NewFunc("f", llssa.NoArgsNoRet, llssa.InGo)
			ctx := &context{
				prog:              prog,
				pkg:               pkg,
				fn:                fn,
				trackCallerFrames: tt.track,
			}
			if got := ctx.shouldTrackCallerFrames(); got != tt.want {
				t.Fatalf("shouldTrackCallerFrames() = %v, want %v", got, tt.want)
			}
			if got := canTrackCallerFramesForPackage(tt.pkgPath); got != (tt.want || tt.targetName != "" || tt.goarch == "wasm" || !tt.track) {
				t.Fatalf("canTrackCallerFramesForPackage(%q) = %v", tt.pkgPath, got)
			}
		})
	}

	if !isStandardLibraryPackage("fmt") {
		t.Fatal("fmt should be treated as a standard library package")
	}
	if isStandardLibraryPackage("command-line-arguments") {
		t.Fatal("command-line-arguments should be eligible for caller frame tracking")
	}
	if isStandardLibraryPackage("example.com/foo") {
		t.Fatal("package paths containing dots should not be treated as standard library packages")
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

func TestCompileRuntimeCallerFrameInstrumentation(t *testing.T) {
	ssapkg, files := buildCallerFrameSSAPackage(t, "example.com/foo", `package foo
import "runtime"

func f() {
	runtime.Caller(0)
}
`)
	prog := newLLSSAProg(t)
	pkg, err := NewPackage(prog, ssapkg, files)
	if err != nil {
		t.Fatal(err)
	}
	ir := pkg.Module().String()
	for _, want := range []string{
		"PushCallerFrame",
		"SetCallerLine",
		"PopCallerFrame",
		`c"example.com/foo.f`,
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("compiled caller-frame IR missing %q:\n%s", want, ir)
		}
	}
}

func TestCompileRuntimeCallerFrameInstrumentationSkippedForTarget(t *testing.T) {
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
	if ir := pkg.Module().String(); strings.Contains(ir, "PushCallerFrame") {
		t.Fatalf("target builds should not emit caller-frame tracking:\n%s", ir)
	}
}

func TestPushSetPopCallerFrameEdgeCases(t *testing.T) {
	prog := newLLSSAProg(t)
	pkg := prog.NewPackage("foo", "example.com/foo")
	fn := pkg.NewFunc("f", llssa.NoArgsNoRet, llssa.InGo)
	b := fn.MakeBody(1)

	ctx := &context{prog: prog, pkg: pkg, fn: fn, fset: token.NewFileSet(), trackCallerFrames: true}
	ctx.pushCallerFrame(b, nil)
	ctx.setCallerLine(b, token.NoPos)
	ctx.popCallerFrame(b)
	b.Return()
	b.EndBuild()
}

func TestCallerFrameCompileDoesNotRunOnStdlibLikePackage(t *testing.T) {
	if canTrackCallerFramesForPackage("net/http") {
		t.Fatal("stdlib paths without dots should not track caller frames")
	}
}
