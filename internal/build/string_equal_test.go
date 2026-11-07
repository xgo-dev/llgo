package build

import (
	"path/filepath"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestStringEqualWordUsesStackMemory(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	pkgs, err := Build(Invocation{Args: []string{llssa.PkgRuntime}, Config: NewDefaultConf(ModeGen)})
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 || pkgs[0].LPkg == nil {
		t.Fatal("missing runtime package")
	}
	defer pkgs[0].LPkg.Prog.Dispose()
	fn := pkgs[0].LPkg.Module().NamedFunction(llssa.PkgRuntime + ".loadStringWord64")
	if fn.IsNil() || fn.FirstBasicBlock().IsNil() {
		t.Fatal("missing string equality word reader")
	}
	body := fn.String()
	// Check before LLVM optimization: debug builds must not rely on heap-to-stack
	// promotion to avoid introducing allocations into string comparisons.
	if !strings.Contains(body, "alloca i8") || !strings.Contains(body, "@llvm.memcpy.inline.") || strings.Contains(body, "runtime.Alloc") {
		t.Fatalf("word reader lost its fixed stack copy:\n%s", body)
	}
}
