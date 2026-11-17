package cl

import (
	"testing"

	gossa "golang.org/x/tools/go/ssa"
)

func TestHeapProfileAllocationFramesAcrossPackages(t *testing.T) {
	for _, observed := range []bool{false, true} {
		rootSource := `package root; import "example.com/dep"; var Sink *int; func Run(){Sink=dep.Wrap()}`
		if observed {
			rootSource = `package root; import("example.com/dep";"runtime"); var Sink *int; func Run(){Sink=dep.Wrap()};func Snapshot(){runtime.MemProfile(nil,true)}`
		}
		dep, root := buildCallerFrameSSAProgram(t, "example.com/dep", `package dep
func Allocate()*int { return new(int) }
func Wrap()*int { return Allocate() }
func Pure(x int)int { return x+1 }
`, "example.com/root", rootSource)
		tracking := NewCallerTracking()
		tracking.Precompute([]*gossa.Package{dep, root})
		for _, entry := range []struct {
			pkg  *gossa.Package
			name string
		}{{dep, "Allocate"}, {dep, "Wrap"}, {root, "Run"}} {
			if got := runtimeCallerFuncSet(tracking, entry.pkg)[entry.pkg.Func(entry.name)]; got != observed {
				t.Errorf("%s retained=%v want=%v", entry.name, got, observed)
			}
		}
		if runtimeCallerFuncSet(tracking, dep)[dep.Func("Pure")] {
			t.Error("nonallocating leaf lost inlining")
		}
	}
}
