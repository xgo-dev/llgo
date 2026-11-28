package build

import (
	"runtime"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/filecheck"
	"github.com/xgo-dev/llgo/cmd/compiler/littest"
	"github.com/xgo-dev/llgo/cmd/compiler/llvmgen"
	"github.com/xgo-dev/llvm"
)

func TestCgoDeferExportEntries(t *testing.T) {
	testCExportEntries(t, "../cl/_testgo/cgodefer", []string{"go_callback", "go_callback_c_only"})
}

func TestCExportCallbackEntries(t *testing.T) {
	testCExportEntries(t, "../cl/_testdrop/c_export_callback", []string{"Callback"})
}

func testCExportEntries(t *testing.T, fixture string, names []string) {
	t.Helper()
	switch runtime.GOOS {
	case "darwin", "linux", "windows":
	default:
		t.Skip("hosted native C callbacks")
	}
	spec, err := littest.LoadSpec(fixture)
	if err != nil {
		t.Fatal(err)
	}
	conf := NewDefaultConf(ModeGen)
	pkgs, err := Do([]string{fixture}, conf)
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("got %d packages for %s, want one command package", len(pkgs), fixture)
	}
	pkg := pkgs[0]
	defer pkg.LPkg.Prog.Dispose()
	module := pkg.LPkg.Module()
	for _, name := range names {
		goName := "main." + name
		if fn := module.NamedFunction(goName); fn.IsNil() || fn.IsDeclaration() {
			t.Fatalf("package module is missing Go implementation %q:\n%s", goName, module.String())
		}
		if fn := module.NamedFunction(name); !fn.IsNil() && !fn.IsDeclaration() {
			t.Fatalf("package module defines C entry %q before final-link wrapping:\n%s", name, fn.String())
		}
	}

	// The package checks above and the final-link checks below must agree on
	// both symbols: the public C name guards the qualified Go implementation.
	for _, goos := range []string{"darwin", "linux", "windows"} {
		for _, mode := range []BuildMode{BuildModeExe, BuildModeCShared, BuildModeCArchive} {
			t.Run(goos+"/"+string(mode), func(t *testing.T) {
				ctx := &context{
					prog: llvmgen.NewProgram(nil),
					buildConf: &Config{
						BuildMode: mode, Goos: goos, Goarch: runtime.GOARCH,
					},
				}
				defer ctx.prog.Dispose()
				exports, err := linkedCExports(ctx, pkgs)
				if err != nil {
					t.Fatal(err)
				}
				if len(exports) != len(names) {
					t.Fatalf("got %d C exports, want %d", len(exports), len(names))
				}
				entry := genMainModule(ctx, llvmgen.PkgRuntime, pkg.Package, &genConfig{
					rtInit: true, cExports: exports,
				}).LPkg.Module()
				if err := llvm.VerifyModule(entry, llvm.ReturnStatusAction); err != nil {
					t.Fatalf("invalid entry module: %v\n%s", err, entry.String())
				}
				if err := filecheck.MatchWithPrefixes(spec.Path, entry.String(), "ENTRY"); err != nil {
					t.Fatalf("public C entry checks: %v\n%s", err, entry.String())
				}
				for _, name := range names {
					goName := "main." + name
					wrapper := entry.NamedFunction(name)
					if wrapper.IsNil() || wrapper.IsDeclaration() {
						t.Fatalf("final-link module is missing public C entry %q:\n%s", name, entry.String())
					}
					if fn := entry.NamedFunction(goName); fn.IsNil() || !fn.IsDeclaration() {
						t.Fatalf("final-link module must reference the package's Go implementation %q:\n%s", goName, entry.String())
					}
					if goos == "windows" && mode == BuildModeCShared {
						assertInOrder(t, wrapper.String(),
							"call void @__llgo_runtime_ensure_initialized()",
							`call i1 @"github.com/xgo-dev/llgo/runtime/internal/runtime.EnterForeignThread"()`,
						)
					}
				}
			})
		}
	}
}
