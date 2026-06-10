//go:build !llgo
// +build !llgo

package cl

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"
	"testing"

	gpackages "github.com/goplus/gogen/packages"
	llssa "github.com/goplus/llgo/ssa"
	gossa "golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func parseRuntimeCallerAST(t *testing.T, name, src string) *ast.File {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), name, src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	return file
}

func compileLLPkgFromSrcPathIR(t *testing.T, pkgPath, src string) string {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "foo.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	files := []*ast.File{file}
	imp := gpackages.NewImporter(fset)
	mode := gossa.SanityCheckFunctions | gossa.InstantiateGenerics
	ssaPkg, _, err := ssautil.BuildPackage(&types.Config{Importer: imp}, fset,
		types.NewPackage(pkgPath, file.Name.Name), files, mode)
	if err != nil {
		t.Fatal(err)
	}
	prog := newLLSSAProg(t)
	pkg, err := NewPackage(prog, ssaPkg, files)
	if err != nil {
		t.Fatal(err)
	}
	return pkg.Module().String()
}

func TestFilesUseRuntimeCallerImportForms(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want bool
	}{
		{
			name: "no runtime import",
			src:  `package foo; func f() {}`,
		},
		{
			name: "default import",
			src: `package foo
import "runtime"
func f() { _, _, _, _ = runtime.Caller(0) }
`,
			want: true,
		},
		{
			name: "alias import",
			src: `package foo
import rt "runtime"
func f() { _ = rt.FuncForPC(0) }
`,
			want: true,
		},
		{
			name: "dot import",
			src: `package foo
import . "runtime"
func f() { _ = Callers(0, nil) }
`,
			want: true,
		},
		{
			name: "blank import",
			src: `package foo
import _ "runtime"
func f() {}
`,
		},
		{
			name: "non caller selector",
			src: `package foo
import "runtime"
func f() { runtime.GC() }
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := parseRuntimeCallerAST(t, "runtime_caller.go", tt.src)
			if got := filesUseRuntimeCaller([]*ast.File{file}); got != tt.want {
				t.Fatalf("filesUseRuntimeCaller() = %v, want %v", got, tt.want)
			}
		})
	}

	badImport := &ast.File{Imports: []*ast.ImportSpec{{
		Path: &ast.BasicLit{Kind: token.STRING, Value: "runtime"},
	}}}
	if filesUseRuntimeCaller([]*ast.File{badImport}) {
		t.Fatal("malformed import literal should not enable runtime caller tracking")
	}
}

func TestRuntimeCallerSSADetection(t *testing.T) {
	if packageUsesRuntimeCaller(nil) {
		t.Fatal("nil package should not use runtime caller")
	}
	if fnUsesRuntimeCaller(nil) {
		t.Fatal("nil function should not use runtime caller")
	}
	if isRuntimeCallerFunc(nil) {
		t.Fatal("nil runtime callee should not be treated as runtime caller")
	}

	plain, _, _ := buildSSAPackageWithFiles(t, `package foo
func plain() {}
`)
	if packageUsesRuntimeCaller(plain) {
		t.Fatal("plain package should not use runtime caller")
	}

	ssapkg, _, _ := buildSSAPackageWithFiles(t, `package foo
import "runtime"

func direct() {
	_, _, _, _ = runtime.Caller(0)
}

func nested() {
	func() { _ = runtime.FuncForPC(0) }()
}

func nonCaller() {
	runtime.GC()
}
`)
	if !packageUsesRuntimeCaller(ssapkg) {
		t.Fatal("package with runtime.Caller should use runtime caller")
	}
	if !fnUsesRuntimeCaller(ssapkg.Func("direct")) {
		t.Fatal("direct runtime.Caller should be detected")
	}
	if !fnUsesRuntimeCaller(ssapkg.Func("nested")) {
		t.Fatal("runtime caller in anon function should be detected")
	}
	if fnUsesRuntimeCaller(ssapkg.Func("nonCaller")) {
		t.Fatal("runtime.GC should not enable runtime caller tracking")
	}
}

func TestCallerFrameTrackingPredicates(t *testing.T) {
	var nilCtx *context
	if nilCtx.shouldTrackCallerFrames() {
		t.Fatal("nil context should not track caller frames")
	}

	for _, pkgPath := range []string{
		llssa.PkgRuntime,
		"runtime",
		"fmt",
		"github.com/goplus/llgo/runtime/internal/runtime",
		"github.com/goplus/llgo/runtime/internal/clite",
	} {
		if canTrackCallerFramesForPackage(pkgPath) {
			t.Fatalf("canTrackCallerFramesForPackage(%q) = true, want false", pkgPath)
		}
	}
	for _, pkgPath := range []string{"example.com/mod/pkg", "command-line-arguments"} {
		if !canTrackCallerFramesForPackage(pkgPath) {
			t.Fatalf("canTrackCallerFramesForPackage(%q) = false, want true", pkgPath)
		}
	}
	if !isStandardLibraryPackage("fmt") {
		t.Fatal("fmt should be recognized as standard library")
	}
	if isStandardLibraryPackage("command-line-arguments") {
		t.Fatal("command-line-arguments should not be treated as standard library")
	}

	prog := newLLSSAProg(t)
	pkg := prog.NewPackage("foo", "example.com/foo")
	fn := pkg.NewFunc("f", llssa.NoArgsNoRet, llssa.InGo)
	ctx := &context{prog: prog, pkg: pkg, fn: fn, trackCallerFrames: true}
	if !ctx.shouldTrackCallerFrames() {
		t.Fatal("user package with tracking enabled should track caller frames")
	}

	ctx.trackCallerFrames = false
	if ctx.shouldTrackCallerFrames() {
		t.Fatal("trackCallerFrames=false should disable tracking")
	}
	ctx.trackCallerFrames = true

	ctx.pkg = prog.NewPackage("fmt", "fmt")
	if ctx.shouldTrackCallerFrames() {
		t.Fatal("standard library package should not track caller frames")
	}
	ctx.pkg = pkg

	target := prog.Target()
	oldTarget, oldGOARCH := target.Target, target.GOARCH
	target.Target = "esp32"
	if ctx.shouldTrackCallerFrames() {
		t.Fatal("-target builds should not track caller frames")
	}
	target.Target = ""
	target.GOARCH = "wasm"
	if ctx.shouldTrackCallerFrames() {
		t.Fatal("wasm builds should not track caller frames")
	}
	target.Target, target.GOARCH = oldTarget, oldGOARCH
}

func TestCallerFrameFileHelpers(t *testing.T) {
	root := t.TempDir()
	original := filepath.Join(root, "src", "main.go")
	fset := token.NewFileSet()
	file := fset.AddFile(original, -1, 64)
	pos := file.Pos(1)
	ctx := &context{fset: fset}

	if got := runtimeFrameName("command-line-arguments.main"); got != "main.main" {
		t.Fatalf("runtimeFrameName command-line-arguments = %q", got)
	}
	if got := runtimeFrameName("example.com/foo.f"); got != "example.com/foo.f" {
		t.Fatalf("runtimeFrameName ordinary package = %q", got)
	}
	ctx.pushCallerFrame(nil, nil)
	if got := ctx.callerFrameFile(pos, ""); got != "??" {
		t.Fatalf("empty caller frame filename = %q", got)
	}
	if got := ctx.callerFrameFile(pos, original); got != original {
		t.Fatalf("same caller frame filename = %q, want %q", got, original)
	}

	inside := filepath.Join(root, "src", "generated", "caller.go")
	if got := ctx.callerFrameFile(pos, inside); got != "generated/caller.go" {
		t.Fatalf("relative caller frame filename = %q", got)
	}
	outside := filepath.Join(root, "other", "caller.go")
	if got := ctx.callerFrameFile(pos, outside); got != outside {
		t.Fatalf("outside caller frame filename = %q, want %q", got, outside)
	}

	if _, ok := relativeLineDirectiveFile(original, filepath.Dir(original)); ok {
		t.Fatal("relativeLineDirectiveFile should reject same directory")
	}
	if _, ok := relativeLineDirectiveFile(filepath.Join(root, "src", "main.go"), root); ok {
		t.Fatal("relativeLineDirectiveFile should reject parent directory")
	}
	if _, ok := relativeLineDirectiveFile(original, outside); ok {
		t.Fatal("relativeLineDirectiveFile should reject paths outside the original directory")
	}
}

func TestCompileRuntimeCallerFrameInstrumentation(t *testing.T) {
	ir := compileLLPkgFromSrcPathIR(t, "example.com/foo", `package foo
import "runtime"

func marker() {}

func f() {
//line generated/caller_frame.go:42
	_, _, _, _ = runtime.Caller(0)
	marker()
}
`)
	for _, want := range []string{
		"PushCallerFrame",
		"SetCallerLocation",
		"PopCallerFrame",
		"generated/caller_frame.go",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("instrumented IR missing %q:\n%s", want, ir)
		}
	}

	plainIR := compileLLPkgFromSrcPathIR(t, "example.com/plain", `package plain
func marker() {}
func f() { marker() }
`)
	for _, unwanted := range []string{"PushCallerFrame", "SetCallerLocation", "PopCallerFrame"} {
		if strings.Contains(plainIR, unwanted) {
			t.Fatalf("plain package IR unexpectedly contains %q:\n%s", unwanted, plainIR)
		}
	}
}
