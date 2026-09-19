//go:build !llgo

package cl

import (
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestGenericRangeDeferStackOwner(t *testing.T) {
	ssapkg := buildSSAPackage(t, `package foo
func seq(yield func(int) bool) { yield(1) }
func f[T any](x T) { for range seq { defer func() { _ = x }() } }
type Set[T any] struct { value T }
func (s *Set[T]) Visit() { for range seq { defer func() { _ = s.value }() } }
func use() { f(1); new(Set[string]).Visit() }
`)
	prog := llssa.NewProgram(nil)
	t.Cleanup(prog.Dispose)
	pkg := prog.NewPackage("foo", "foo")
	count := 0
	for root := range ssautil.AllFunctions(ssapkg.Prog) {
		if root.Parent() != nil || len(root.TypeArgs()) == 0 || len(root.AnonFuncs) == 0 {
			continue
		}
		owner := pkg.NewFunc(root.String(), llssa.NoArgsNoRet, llssa.InGo)
		ctx := &context{funcs: map[*ssa.Function]llssa.Function{root: owner}}
		if got := ctx.deferStackOwner(root); got != owner {
			t.Fatalf("%s: generic frame lost its defer ownership", root)
		}
		if got := ctx.deferStackOwner(root.AnonFuncs[0]); got != owner {
			t.Fatalf("%s: yield has wrong defer owner", root)
		}
		if !ctx.returnNeedsImplicitRunDefers(lastNonRecoverReturn(t, root)) {
			t.Fatalf("%s: generic return does not drain the defer stack", root)
		}
		if ctx.returnNeedsImplicitRunDefers(lastNonRecoverReturn(t, root.AnonFuncs[0])) {
			t.Fatalf("%s: yield must not drain its enclosing frame", root)
		}
		count++
	}
	if count != 2 {
		t.Fatalf("checked %d generic instances, want 2", count)
	}
}
