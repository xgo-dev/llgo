package cl

import (
	"fmt"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestAtomicStoreUnorderedOrdering(t *testing.T) {
	const source = `package atomicordering
import _ "unsafe"

//go:linkname store llgo.atomicStore
func store(*uintptr, uintptr)

//go:linkname storeUnordered llgo.atomicStoreUnordered
func storeUnordered(*uintptr, uintptr)

func Ordered(p *uintptr, v uintptr) { store(p, v) }
func Unordered(p *uintptr, v uintptr) { storeUnordered(p, v) }
func Deferred(p *uintptr, v uintptr) { defer storeUnordered(p, v) }
func Asynchronous(p *uintptr, v uintptr) { go storeUnordered(p, v) }
`
	for _, target := range []llssa.Target{
		{GOOS: "linux", GOARCH: "386"},
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "wasip1", GOARCH: "wasm"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			ssaPkg, _, files := buildGoSSAPkg(t, source)
			prog := newLLSSAProgForTarget(t, &target)
			defer prog.Dispose()
			pkg, err := NewPackage(prog, ssaPkg, files)
			if err != nil {
				t.Fatal(err)
			}
			defer pkg.Module().Dispose()
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatalf("invalid atomic-store IR: %v\n%s", err, pkg.String())
			}
			for _, test := range []struct{ name, ordering string }{
				{"Ordered", "seq_cst"},
				{"Unordered", "unordered"},
				{"Deferred", "unordered"},
				{"_llgo_routine$1", "unordered"},
			} {
				body := mustNamedFunction(t, pkg.Module(), "atomicordering."+test.name).String()
				stores := 0
				for _, line := range strings.Split(body, "\n") {
					if strings.Contains(line, "store atomic ") {
						stores++
						if !strings.Contains(line, " "+test.ordering+",") {
							t.Errorf("%s store must use %s ordering:\n%s", test.name, test.ordering, body)
						}
					}
				}
				if stores != 1 {
					t.Errorf("%s has %d atomic stores, want one:\n%s", test.name, stores, body)
				}
			}
			// A go statement must put the atomic store in the worker instead of
			// executing it synchronously while constructing the startup record.
			body := mustNamedFunction(t, pkg.Module(), "atomicordering.Asynchronous").String()
			if strings.Contains(body, "store atomic ") || !strings.Contains(body, llssa.PkgRuntime+".NewProc") {
				t.Errorf("go statement did not defer the store to its worker:\n%s", body)
			}
		})
	}
}

func TestAtomicStoreUnorderedInvalidArity(t *testing.T) {
	for _, count := range []int{0, 1, 3} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			defer func() {
				if got := fmt.Sprint(recover()); got != "atomicStoreUnordered(addr *T, val T): invalid arguments" {
					t.Errorf("invalid-arity panic = %q", got)
				}
			}()
			var ctx context
			ctx.atomicStoreUnordered(nil, make([]llssa.Expr, count))
		})
	}
}
