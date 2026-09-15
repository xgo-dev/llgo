//go:build !llgo

package cl

import (
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestWasmNilChecksDoNotRelyOnTrappingMemory(t *testing.T) {
	const source = `package wasmnil
type T struct { pad [4096]byte; value int }
func Field(p *T) int { return p.value }
func Aggregate(p *T) T { return *p }
func Store(p *int) { *p = 7 }
func Array(p *[2]int, i int) int { return p[i] }
func Indirect(f func() int) int { return f() }
func Selected(p, q *int, b bool) int { if b { p = q }; return *p }
`
	for _, goos := range []string{"js", "wasip1"} {
		t.Run(goos, func(t *testing.T) {
			ir := compileWithRewritesTarget(t, source, nil, &llssa.Target{GOOS: goos, GOARCH: "wasm"})
			for _, name := range []string{"Field", "Aggregate", "Store", "Array", "Indirect", "Selected"} {
				body := llvmFunction(t, ir, "wasmnil."+name)
				if !strings.Contains(body, "AssertNilDeref") {
					t.Errorf("%s relies on a wasm memory/table trap:\n%s", name, body)
				}
			}
		})
	}
}

func TestWasmNilChecksExcludeRuntimeSupportPackages(t *testing.T) {
	for _, test := range []struct {
		path string
		want bool
	}{
		{path: "example.com/app", want: true},
		{path: "runtime"},
		{path: "runtime/internal/sys"},
		{path: "internal/runtime/atomic"},
		{path: llssa.PkgRuntime},
		{path: strings.TrimSuffix(llssa.PkgRuntime, "/internal/runtime") + "/internal/lib/runtime"},
	} {
		if got := !llssa.IsRuntimeSupportPackage(test.path); got != test.want {
			t.Errorf("recoverable nil checks for %q = %v, want %v", test.path, got, test.want)
		}
	}
}
