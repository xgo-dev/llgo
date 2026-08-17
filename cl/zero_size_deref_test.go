//go:build !llgo
// +build !llgo

package cl

import (
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestZeroSizedFieldDerefEmitsBaseNilGuard(t *testing.T) {
	const src = `package zeroderef
type T struct {
	n int
	z struct{}
}
func Eq(p, q *T) bool {
	return p.z == q.z
}
`
	ir := compileWithRewrites(t, src, nil)
	if got := strings.Count(ir, "AssertNilDeref"); got < 4 {
		t.Fatalf("zero-sized field comparison should guard field bases and loads, got %d guards:\n%s", got, ir)
	}
}

func TestZeroSizedSourceClosureElidesEnvironment(t *testing.T) {
	const src = `package zeroclosure

func makeValue() (func() *struct{}, *struct{}) {
	value := struct{}{}
	return func() *struct{} { return &value }, &value
}

func compareLocal() bool {
	value := struct{}{}
	closure := func() *struct{} { return &value }
	return closure() == &value
}

func keepPointer(pointer *struct{}) func() bool {
	return func() bool { return pointer == nil }
}
`
	targets := []struct {
		name   string
		target *llssa.Target
	}{
		{name: "native"},
		{name: "wasm-explicit", target: &llssa.Target{GOOS: "wasip1", GOARCH: "wasm"}},
	}
	for _, target := range targets {
		t.Run(target.name, func(t *testing.T) {
			ir := compileWithRewritesTarget(t, src, nil, target.target)
			for _, want := range []string{
				`{ ptr @"zeroclosure.makeValue$1", ptr null }`,
				`define ptr @"zeroclosure.makeValue$1"()`,
				`define ptr @"zeroclosure.compareLocal$1"()`,
				`ret ptr @"__llgo.moduleZeroSizedAlloc$"`,
			} {
				if !strings.Contains(ir, want) {
					t.Fatalf("zero-sized source closure did not elide its environment; missing %q:\n%s", want, ir)
				}
			}
			if !strings.Contains(ir, `define i1 @"zeroclosure.keepPointer$1"(ptr `) ||
				!strings.Contains(ir, `@"zeroclosure.keepPointer$1", ptr undef`) {
				t.Fatalf("captured pointer value incorrectly lost its environment:\n%s", ir)
			}
		})
	}
}

func TestUnusedDerefEmitsNilGuard(t *testing.T) {
	const src = `package unusedderef
func LoadArrayElement() {
	var values [2]*int
	_ = *values[1]
}
func LoadPointer(p *int) {
	_ = *p
}
`
	ir := compileWithRewrites(t, src, nil)
	arrayLoad := llvmFunction(t, ir, "unusedderef.LoadArrayElement")
	if !strings.Contains(arrayLoad, "AssertNilDeref") {
		t.Fatalf("unused array-element dereference should retain a nil guard:\n%s", arrayLoad)
	}
	directLoad := llvmFunction(t, ir, "unusedderef.LoadPointer")
	if !strings.Contains(directLoad, "AssertNilDeref") {
		t.Fatalf("unused direct dereference should retain a nil guard:\n%s", directLoad)
	}
}
