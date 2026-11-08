package cl

import (
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
