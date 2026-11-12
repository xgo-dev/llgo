package build

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/xgo-dev/llgo/internal/lto"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

// Entry overrides create anonymous private constants after package optimization.
// Both initial emission and the PCLN relink must name them for ThinLTO's index.
func TestThinLTONamesLateEntryGlobals(t *testing.T) {
	llssa.Initialize(llssa.InitAll)
	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	pkg := prog.NewPackage("entry", "entry")
	mod := pkg.Module()
	private := llvm.AddGlobal(mod, mod.Context().Int32Type(), "")
	private.SetInitializer(llvm.ConstInt(mod.Context().Int32Type(), 42, false))
	private.SetLinkage(llvm.PrivateLinkage)
	private.SetGlobalConstant(true)
	root := llvm.AddGlobal(mod, private.Type(), "root")
	root.SetInitializer(private)
	conf := &Config{Goos: runtime.GOOS, Goarch: runtime.GOARCH, LTO: lto.Thin}
	ctx := &context{buildConf: conf, prog: prog}
	dir := t.TempDir()
	ir := filepath.Join(dir, "entry.ll")
	// buildMainLink snapshots IR before initial object generation.
	if err := os.WriteFile(ir, []byte(mod.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	first, _, err := emitObjectToMemoryBuffer(ctx, pkg)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Dispose()
	if private.Name() == "" {
		t.Fatal("initial ThinLTO emission left an anonymous private global")
	}
	second := filepath.Join(dir, "entry.o")
	if err := compileEmbeddedEntry(ctx, &mainLinkPlan{entryTarget: *prog.Target()}, ir, second); err != nil {
		t.Fatal(err)
	}
	lc := llvm.NewContext()
	defer lc.Dispose()
	relinked, err := lc.ParseBitcodeFile(second)
	if err != nil {
		t.Fatal(err)
	}
	defer relinked.Dispose()
	for g := relinked.FirstGlobal(); !g.IsNil(); g = llvm.NextGlobal(g) {
		if g.Linkage() == llvm.PrivateLinkage && g.Name() == "" {
			t.Fatal("PCLN relink left an anonymous private global")
		}
	}
	if err := llvm.VerifyModule(relinked, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
}
