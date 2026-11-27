package cl

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/llvmgen"
	"github.com/xgo-dev/llgo/ssa"
)

func TestTypeAssertEmptyInterfaceSkipsImplements(t *testing.T) {
	const source = `package emptyassert

type Runner interface {
	Run()
}

type Extra interface {
	Run()
	Stop()
}

func toEmpty(r Runner) (any, bool) {
	v, ok := r.(any)
	return v, ok
}

func toNonEmpty(r Runner) (Extra, bool) {
	v, ok := r.(Extra)
	return v, ok
}
`
	for _, target := range []llvmgen.Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "wasip1", GOARCH: "wasm"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			ir := compileInterfaceBoxIR(t, source, ssa.InstantiateGenerics, &target, target.GOARCH == "wasm")
			emptyFn := llvmFunction(t, ir, "emptyassert.toEmpty")
			if strings.Contains(emptyFn, "runtime.Implements") {
				t.Fatalf("empty-interface TypeAssert still called runtime.Implements:\n%s", emptyFn)
			}
			nonEmptyFn := llvmFunction(t, ir, "emptyassert.toNonEmpty")
			if !strings.Contains(nonEmptyFn, "runtime.Implements") {
				t.Fatalf("non-empty-interface TypeAssert missing runtime.Implements:\n%s", nonEmptyFn)
			}
		})
	}
}
