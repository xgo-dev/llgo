package cl

import (
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestWasmValueMethodNilWrapper(t *testing.T) {
	goPkg, files := buildCallerFrameSSAPackage(t, "command-line-arguments", `package main

type value[T any] struct{}
type arg struct{}

func (value[T]) Method() {}

var x interface{ Method() } = (*value[arg])(nil)
func main() { x.Method() }
`)
	prog := newLLSSAProgForTarget(t, &llssa.Target{GOOS: "wasip1", GOARCH: "wasm"})
	pkg, err := NewPackage(prog, goPkg, files)
	if err != nil {
		t.Fatal(err)
	}
	ir := pkg.Module().String()
	const message = "value method main.value[command-line-arguments.arg].Method called using nil *value[command-line-arguments.arg] pointer"
	if !strings.Contains(ir, "PanicWrapNilPointerText") || !strings.Contains(ir, message) {
		t.Fatalf("Wasm wrapper did not pass a constant panic message:\n%s", ir)
	}
	if strings.Contains(ir, `PanicWrapNilPointer"`) {
		t.Fatalf("Wasm wrapper still calls the dynamic panic formatter:\n%s", ir)
	}
}
