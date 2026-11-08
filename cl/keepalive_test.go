package cl

import (
	"fmt"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestCompileKeepAliveBarrier(t *testing.T) {
	_, mod := mustCompileLLPkgFromSrc(t, `package foo
import _ "unsafe"
//go:linkname keepAlive llgo.keepAlive
func keepAlive(any)
func keep(x any) { keepAlive(x) }
func keepNil() { keepAlive(nil) }
`)
	options := llvm.NewPassBuilderOptions()
	defer options.Dispose()
	if err := mod.RunPasses("default<O2>", llvm.TargetMachine{}, options); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"foo.keep", "foo.keepNil"} {
		body := mustNamedFunction(t, mod, name).String()
		if !strings.Contains(body, `asm sideeffect "", "X,X,~{memory}"`) {
			t.Fatalf("%s lost its value and memory liveness barrier:\n%s", name, body)
		}
	}
}

func TestCompileKeepAliveDiagnostics(t *testing.T) {
	for _, tc := range []struct{ name, declaration, use, want string }{
		{"arity", "func keepAlive()", "func keep() { keepAlive() }", "keepAlive(any): invalid arguments"},
		{"result", "func keepAlive(any) int", "func keep(x any) int { return keepAlive(x) }", "keepAlive(any): invalid arguments"},
		{"type", "func keepAlive(int)", "func keep(x int) { keepAlive(x) }", "keepAlive(any): invalid argument type"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := "package foo\nimport _ \"unsafe\"\n//go:linkname keepAlive llgo.keepAlive\n" + tc.declaration + "\n" + tc.use
			pkg, _, files := buildGoSSAPkg(t, src)
			prog := newLLSSAProg(t)
			defer prog.Dispose()
			defer func() {
				if got := fmt.Sprint(recover()); !strings.Contains(got, tc.want) {
					t.Fatalf("diagnostic = %q, want %q", got, tc.want)
				}
			}()
			if _, err := NewPackage(prog, pkg, files); err != nil {
				panic(err)
			}
		})
	}
}

func TestCompileKeepAliveCallForms(t *testing.T) {
	_, mod := mustCompileLLPkgFromSrc(t, `package foo
import _ "unsafe"
//go:linkname KeepAlive llgo.keepAlive
func KeepAlive(any)
var Saved = KeepAlive
func direct(x any) { KeepAlive(x) }
//go:noinline
func after() {}
func deferred(x any) { defer KeepAlive(x); after() }
func launched(x any) { go KeepAlive(x) }
func indirect(x any) { Saved(x) }
`)
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	const barrier = `asm sideeffect "", "X,X,~{memory}"`
	direct := mustNamedFunction(t, mod, "foo.direct").String()
	if !strings.Contains(direct, barrier) || strings.Contains(direct, "@foo.KeepAlive(") {
		t.Fatalf("direct KeepAlive must lower without a forwarding call:\n%s", direct)
	}
	wrapper := mustNamedFunction(t, mod, "foo.KeepAlive").String()
	if !strings.Contains(wrapper, barrier) {
		t.Fatal("function value lost its callable intrinsic wrapper")
	}
	deferred := mustNamedFunction(t, mod, "foo.deferred").String()
	after := strings.Index(deferred, "@foo.after(")
	if after < 0 {
		t.Fatalf("missing call after defer registration:\n%s", deferred)
	}
	if use := strings.Index(deferred, barrier); use >= 0 && use < after {
		t.Fatalf("deferred KeepAlive executed at registration:\n%s", deferred)
	}
	launched := mustNamedFunction(t, mod, "foo.launched").String()
	if strings.Contains(launched, barrier) || !strings.Contains(launched, ".NewProc") {
		t.Fatalf("go KeepAlive must execute in the new goroutine:\n%s", launched)
	}
}

func TestCompileKeepAliveTypeMetadata(t *testing.T) {
	const src = `package foo
import "unsafe"
//go:linkname KeepAlive llgo.keepAlive
func KeepAlive(any)
type Item struct { P *int; N int }
func (*Item) Method() {}
var Saved = KeepAlive
func pointer(p *Item) { KeepAlive(p) }
func raw(p unsafe.Pointer) { KeepAlive(p) }
func aggregate(p Item) { KeepAlive(p) }
func repeated(p *Item) { v := any(p); KeepAlive(v); KeepAlive(v) }
func deferred(p *Item) { defer KeepAlive(p) }
func launched(p *Item) { go KeepAlive(p) }
func shared(p *Item) any { v := any(p); KeepAlive(v); return v }
func indirect(p *Item) { Saved(p) }
func dynamic(p any) { KeepAlive(p) }
`
	for _, target := range []*llssa.Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "windows", GOARCH: "386"},
		{GOOS: "js", GOARCH: "wasm", WasmProfile: "j32"},
		{GOOS: "js", GOARCH: "wasm", WasmProfile: "j64"},
		{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH+"/"+target.WasmProfile, func(t *testing.T) {
			ssapkg, _, files := buildGoSSAPkg(t, src)
			prog := newLLSSAProgForTarget(t, target)
			defer prog.Dispose()
			prog.EnableGCRoots(target.GOARCH == "wasm")
			pkg, err := NewPackage(prog, ssapkg, files)
			if err != nil {
				t.Fatal(err)
			}
			mod := pkg.Module()
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"pointer", "raw", "aggregate", "repeated", "deferred", "launched"} {
				ir := mustNamedFunction(t, mod, "foo."+name).String()
				if strings.Contains(ir, "_llgo_foo.Item") || strings.Contains(ir, "_llgo_itab$") {
					t.Errorf("%s unnecessarily retains static type metadata:\n%s", name, ir)
				}
			}
			for _, name := range []string{"shared", "indirect"} {
				ir := mustNamedFunction(t, mod, "foo."+name).String()
				if !strings.Contains(ir, "_llgo_foo.Item") {
					t.Errorf("%s lost observable type metadata:\n%s", name, ir)
				}
			}
			dynamic := mustNamedFunction(t, mod, "foo.dynamic").String()
			if !strings.Contains(dynamic, `asm sideeffect "", "X,X,~{memory}"`) {
				t.Fatalf("dynamic interface lost its two-word barrier:\n%s", dynamic)
			}
		})
	}
}
