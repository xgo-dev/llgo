package build

import (
	"path/filepath"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestRuntimeEscapeContractsAvoidObserverAndMapKeyAllocation(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	for _, test := range []struct {
		path  string
		names []string
	}{
		{"runtime", []string{"runtime.ReadMemStats"}},
		{llssa.PkgRuntime, []string{llssa.PkgRuntime + ".mapaccess1_faststr", llssa.PkgRuntime + ".mapaccess2_faststr"}},
	} {
		t.Run(test.path, func(t *testing.T) {
			conf := NewDefaultConf(ModeGen)
			pkgs, err := Build(Invocation{Args: []string{test.path}, Config: conf})
			if err != nil {
				t.Fatal(err)
			}
			if len(pkgs) != 1 || pkgs[0].LPkg == nil {
				t.Fatal("missing compiled runtime package")
			}
			defer pkgs[0].LPkg.Prog.Dispose()
			for _, name := range test.names {
				fn := pkgs[0].LPkg.Module().NamedFunction(name)
				if fn.IsNil() || fn.FirstBasicBlock().IsNil() {
					t.Fatalf("missing %s definition", name)
				}
				if body := fn.String(); strings.Contains(body, "runtime.Alloc") {
					t.Errorf("%s retains temporary heap storage:\n%s", name, body)
				}
			}
		})
	}
}
