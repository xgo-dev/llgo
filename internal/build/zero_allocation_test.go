package build

import (
	"path/filepath"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
)

// Compile the real implementations: zero-value tests also pass when AllocZ
// redundantly clears GC_malloc's already-zeroed memory.
func TestNativeZeroAllocationIR(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	for _, tags := range []string{"", "nogc"} {
		name := "gc"
		if tags != "" {
			name = tags
		}
		t.Run(name, func(t *testing.T) {
			conf := NewDefaultConf(ModeGen)
			conf.Tags = tags
			pkgs, err := Build(Invocation{Args: []string{llssa.PkgRuntime}, Config: conf})
			if err != nil {
				t.Fatal(err)
			}
			if len(pkgs) != 1 || pkgs[0].LPkg == nil {
				t.Fatal("missing compiled runtime package")
			}
			defer pkgs[0].LPkg.Prog.Dispose()
			mod := pkgs[0].LPkg.Module()
			for _, name := range []string{"AllocU", "AllocZ"} {
				fn := mod.NamedFunction(llssa.PkgRuntime + "." + name)
				if fn.IsNil() || fn.FirstBasicBlock().IsNil() {
					t.Fatalf("missing %s definition", name)
				}
				body := fn.String()
				clears := strings.Contains(body, "memset") || strings.Contains(body, "bzero")
				wantClear := tags == "nogc" && name == "AllocZ"
				if clears != wantClear {
					t.Errorf("%s clearing = %t, want %t:\n%s", name, clears, wantClear, body)
				}
				if name == "AllocZ" && !strings.Contains(body, "runtime.AllocU") {
					t.Errorf("AllocZ no longer calls the allocator:\n%s", body)
				}
				callsGC := strings.Contains(body, "@GC_malloc(") || strings.Contains(body, `@"GC_malloc"(`)
				if tags == "" && name == "AllocU" && !callsGC {
					t.Errorf("native allocation lost GC_malloc's clearing contract:\n%s", body)
				}
			}
		})
	}
}
