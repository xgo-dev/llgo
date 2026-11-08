package cl

import (
	"fmt"
	"strings"
	"testing"

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
