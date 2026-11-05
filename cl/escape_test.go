package cl

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/goescape"
	"golang.org/x/tools/go/ssa"
)

func TestEscapeResolvedInstructionContracts(t *testing.T) {
	const src = `package effects
import "unsafe"
var sink unsafe.Pointer
//go:linkname load llgo.atomicLoad
func load(p *unsafe.Pointer) unsafe.Pointer { panic("not the called body") }
//go:linkname store llgo.atomicStore
func store(p *unsafe.Pointer, v unsafe.Pointer) {}
//go:linkname unordered llgo.atomicStoreUnordered
func unordered(p *unsafe.Pointer, v unsafe.Pointer) {}
//go:linkname exchange llgo.atomicXchg
func exchange(p *unsafe.Pointer, v unsafe.Pointer) unsafe.Pointer { return nil }
//go:linkname compare llgo.atomicCmpXchg
func compare(p *unsafe.Pointer, old, new unsafe.Pointer) (unsafe.Pointer,bool) { return nil,false }
//go:linkname compareOK llgo.atomicCmpXchgOK
func compareOK(p *unsafe.Pointer, old, new unsafe.Pointer) bool {return false}
//go:linkname add llgo.atomicAddReturnNew
func add(p *int64, n int64) int64 {return 0}
//go:linkname ignore llgo.skip
func ignore(any) {panic("not the called body")}
//go:noescape
func borrow(*int)
//go:noescape
func callback(func())
func reader(p unsafe.Pointer) unsafe.Pointer {return load(&p)}
func writer() {x:=1;store(&sink,unsafe.Pointer(&x))}
func unorderedWriter() {x:=1;unordered(&sink,unsafe.Pointer(&x))}
func exchangeWriter() {x:=1;exchange(&sink,unsafe.Pointer(&x))}
func compareWriter() {x:=1;compare(&sink,nil,unsafe.Pointer(&x))}
func compareOKWriter() {x:=1;compareOK(&sink,nil,unsafe.Pointer(&x))}
func adder(x int64)int64 {return add(&x,1)}
func discarded(x int) {ignore(&x)}
func borrowed(x int) {borrow(&x)}
func called(x int) {callback(func(){sink=unsafe.Pointer(&x)})}
`
	ir := compileInterfaceBoxIR(t, src, ssa.InstantiateGenerics, nil, false)
	for _, name := range []string{"reader", "discarded", "borrowed", "adder"} {
		if body := llvmFunction(t, ir, "effects."+name); strings.Contains(body, "runtime.Alloc") {
			t.Errorf("%s retains borrowed storage:\n%s", name, body)
		}
	}
	for _, name := range []string{"writer", "unorderedWriter", "exchangeWriter", "compareWriter", "compareOKWriter", "called"} {
		if body := llvmFunction(t, ir, "effects."+name); !strings.Contains(body, "runtime.AllocZ") {
			t.Errorf("%s lost published pointer:\n%s", name, body)
		}
	}
}

func TestMissingEscapeFactsAreConservative(t *testing.T) {
	var result *goescape.Result
	if decision := result.Decision(&ssa.Alloc{}); !decision.Heap || decision.Reason == "" {
		t.Fatalf("missing facts accepted: %+v", decision)
	}
}
