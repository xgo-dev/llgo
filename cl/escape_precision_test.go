package cl

import (
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func TestEscapePrecisionLowering(t *testing.T) {
	const src = `package precision
func read(p *int) int {return *p}
func apply(fn func(*int)int, p *int) int {return fn(p)}
func borrow(x int)int{return apply(read,&x)}
func immutable(x int)func()int{return func()int{return x}}
func mutable(x int)func()int{return func()int{x++;return x}}
func equal(a,b string)bool{return a==b}
func compare(b []byte,a,c string)bool{s:=string(b);return equal(s,a)||s==c}
func snapshot(b []byte,a string)bool{s:=string(b);b[0]++;return s==a}
`
	for _, target := range []llssa.Target{
		{GOOS: "darwin", GOARCH: "arm64"}, {GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "windows", GOARCH: "386"}, {GOOS: "wasip1", GOARCH: "wasm"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			t.Setenv("GOOS", target.GOOS)
			t.Setenv("GOARCH", target.GOARCH)
			ir := compileInterfaceBoxIR(t, src, ssa.InstantiateGenerics, &target, target.GOARCH == "wasm")
			if body := llvmFunction(t, ir, "precision.borrow"); strings.Contains(body, "runtime.Alloc") {
				t.Errorf("callback argument still allocates:\n%s", body)
			}
			if body := llvmFunction(t, ir, "precision.immutable"); strings.Contains(body, "runtime.AllocZ") || strings.Count(body, "call ptr @\"github.com/xgo-dev/llgo/runtime/internal/runtime.AllocU\"") > 1 {
				t.Errorf("immutable variable has separate heap storage:\n%s", body)
			}
			if body := llvmFunction(t, ir, "precision.mutable"); !strings.Contains(body, "runtime.AllocZ") {
				t.Errorf("mutable capture lost shared storage:\n%s", body)
			}
			if body := llvmFunction(t, ir, "precision.compare"); !strings.Contains(body, "StringFromBytesReadOnly") {
				t.Errorf("read region still copies:\n%s", body)
			}
			if body := llvmFunction(t, ir, "precision.snapshot"); strings.Contains(body, "StringFromBytesReadOnly") {
				t.Errorf("mutable source lost snapshot:\n%s", body)
			}
		})
	}
}
