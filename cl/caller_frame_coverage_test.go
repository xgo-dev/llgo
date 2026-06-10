//go:build !llgo
// +build !llgo

package cl

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"runtime"
	"strings"
	"testing"

	"github.com/goplus/gogen/packages"
	llssa "github.com/goplus/llgo/ssa"
	gossa "golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestCallerFrameTrackingHelpers(t *testing.T) {
	if packageUsesRuntimeCaller(nil) {
		t.Fatal("nil package should not use runtime caller")
	}
	plainPkg, _, _ := buildGoSSAPkg(t, `package foo
func plain() {}
`)
	if packageUsesRuntimeCaller(plainPkg) {
		t.Fatal("package without runtime caller should not use runtime caller")
	}
	callerPkg, _, _ := buildGoSSAPkg(t, `package foo
import "runtime"
func direct() { runtime.Caller(0) }
func indirect() { func() { runtime.FuncForPC(0) }() }
`)
	if !packageUsesRuntimeCaller(callerPkg) {
		t.Fatal("package with runtime.Caller should use runtime caller")
	}
	if fnUsesRuntimeCaller(nil) {
		t.Fatal("nil function should not use runtime caller")
	}
	if !fnUsesRuntimeCaller(callerPkg.Func("direct")) {
		t.Fatal("direct runtime.Caller call was not detected")
	}
	if !fnUsesRuntimeCaller(callerPkg.Func("indirect")) {
		t.Fatal("runtime caller use in anonymous function was not detected")
	}
	call := findStaticCallByName(t, callerPkg.Func("direct"), "Caller")
	if !isRuntimeCallerFunc(call.Common().StaticCallee()) {
		t.Fatal("runtime.Caller static callee was not detected")
	}
	if isRuntimeCallerFunc(nil) {
		t.Fatal("nil callee should not be a runtime caller")
	}

	trackable := []string{
		"command-line-arguments",
		"example.com/app",
	}
	for _, path := range trackable {
		if !canTrackCallerFramesForPackage(path) {
			t.Fatalf("canTrackCallerFramesForPackage(%q) = false, want true", path)
		}
		if isStandardLibraryPackage(path) {
			t.Fatalf("isStandardLibraryPackage(%q) = true, want false", path)
		}
	}

	blocked := []string{
		llssa.PkgRuntime,
		"runtime",
		"fmt",
		"github.com/goplus/llgo/runtime/internal/runtime",
	}
	for _, path := range blocked {
		if canTrackCallerFramesForPackage(path) {
			t.Fatalf("canTrackCallerFramesForPackage(%q) = true, want false", path)
		}
	}
	if !isStandardLibraryPackage("fmt") {
		t.Fatal("fmt should be treated as a standard library package")
	}

	for _, name := range []string{"Caller", "Callers", "CallersFrames", "FuncForPC"} {
		if !isRuntimeCallerName(name) {
			t.Fatalf("isRuntimeCallerName(%q) = false, want true", name)
		}
	}
	if isRuntimeCallerName("NumGoroutine") {
		t.Fatal("NumGoroutine should not enable caller-frame tracking")
	}

	nameCases := map[string]string{
		"command-line-arguments.main$1": "main.main.func1",
		"foo.bar$12":                    "foo.bar.func12",
		"foo.bar":                       "foo.bar",
		"foo.bar$":                      "foo.bar$",
		"foo.bar$x":                     "foo.bar$x",
	}
	for in, want := range nameCases {
		if got := runtimeFrameName(in); got != want {
			t.Fatalf("runtimeFrameName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFilesUseRuntimeCallerImportForms(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want bool
	}{
		{
			name: "alias",
			src: `package foo
import rt "runtime"
func f() { rt.Callers(0, nil) }
`,
			want: true,
		},
		{
			name: "dot",
			src: `package foo
import . "runtime"
func f() { FuncForPC(0) }
`,
			want: true,
		},
		{
			name: "blank",
			src: `package foo
import _ "runtime"
func f() {}
`,
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := parseCallerFrameFile(t, tt.src)
			if got := filesUseRuntimeCaller([]*ast.File{file}); got != tt.want {
				t.Fatalf("filesUseRuntimeCaller() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldTrackCallerFrames(t *testing.T) {
	if (&context{}).shouldTrackCallerFrames() {
		t.Fatal("empty context should not track caller frames")
	}

	prog := newLLSSAProg(t)
	pkg := prog.NewPackage("foo", "example.com/foo")
	fn := pkg.NewFunc("foo.f", llssa.NoArgsNoRet, llssa.InGo)
	ctx := &context{prog: prog, pkg: pkg, fn: fn, trackCallerFrames: true}
	if !ctx.shouldTrackCallerFrames() {
		t.Fatal("native non-runtime package should track caller frames")
	}

	ctx.trackCallerFrames = false
	if ctx.shouldTrackCallerFrames() {
		t.Fatal("trackCallerFrames=false should disable caller-frame tracking")
	}
	ctx.trackCallerFrames = true

	prog.Target().Target = "esp32"
	if ctx.shouldTrackCallerFrames() {
		t.Fatal("explicit non-native target should disable caller-frame tracking")
	}
	prog.Target().Target = ""
	prog.Target().GOARCH = "wasm"
	if ctx.shouldTrackCallerFrames() {
		t.Fatal("wasm target should disable caller-frame tracking")
	}
	prog.Target().GOARCH = runtime.GOARCH

	runtimePkg := prog.NewPackage("runtime", "runtime")
	ctx.pkg = runtimePkg
	if ctx.shouldTrackCallerFrames() {
		t.Fatal("runtime package should not track caller frames")
	}

	ctx.pkg = pkg
	ctx.trackCallerFrames = false
	ctx.setCallerLineNumber(nil, 123)
	ctx.trackCallerFrames = true
	ctx.setCallerLineNumber(nil, 0)
	ctx.callerFrameMark = llssa.Nil
	ctx.popCallerFrame(nil)
	ctx.pushCallerFrame(nil, nil)
}

func TestCompileCallerFrameInstrumentationForPanicLineOps(t *testing.T) {
	ir := compileCallerFramePackage(t, `package foo

import "runtime"

type box struct {
	value int
}

var (
	b     *box
	slice = []int{1, 2}
	array = [2]int{1, 2}
	sink  int
)

func callerLineOps(i int) {
	_, _, _, _ = runtime.Caller(0)
	_ = -i
	_ = b.value
	sink = array[i]
	sink = slice[i]
	sink = slice[:i][0]
	panic("boom")
}

func deferLine() {
	defer func() {
		_, _, _, _ = runtime.Caller(1)
	}()
}
`)
	for _, want := range []string{"PushCallerFrame", "SetCallerLine", "PopCallerFrame"} {
		if !strings.Contains(ir, want) {
			t.Fatalf("compiled IR missing %s:\n%s", want, ir)
		}
	}
	if got := strings.Count(ir, "SetCallerLine"); got < 6 {
		t.Fatalf("compiled IR has %d SetCallerLine calls, want at least 6:\n%s", got, ir)
	}
}

func compileCallerFramePackage(t *testing.T, src string) string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "foo.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	files := []*ast.File{file}
	pkg := types.NewPackage("example.com/foo", file.Name.Name)
	imp := packages.NewImporter(fset)
	mode := gossa.SanityCheckFunctions | gossa.InstantiateGenerics
	ssaPkg, _, err := ssautil.BuildPackage(&types.Config{Importer: imp}, fset, pkg, files, mode)
	if err != nil {
		t.Fatal(err)
	}
	prog := newLLSSAProg(t)
	ret, err := NewPackage(prog, ssaPkg, files)
	if err != nil {
		t.Fatal(err)
	}
	return ret.String()
}

func parseCallerFrameFile(t *testing.T, src string) *ast.File {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "foo.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	return file
}
