package llvmattr_test

import (
	"os"
	"testing"

	"github.com/xgo-dev/llgo/internal/llvmattr"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
	"golang.org/x/tools/go/packages"
)

// Load the 32-bit runtime from source to include private helpers and NewChan64.
func TestRuntimeAttributeModelSignatures(t *testing.T) {
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.LoadSyntax,
		Env:  append(os.Environ(), "GOOS=linux", "GOARCH=386", "CGO_ENABLED=0"),
	}, ssa.PkgRuntime)
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("loaded %d runtime packages, want 1", len(pkgs))
	}
	if len(pkgs[0].Errors) != 0 {
		t.Fatalf("loading runtime: %v", pkgs[0].Errors)
	}
	rt := pkgs[0].Types
	ssa.Initialize(ssa.InitAllTargets | ssa.InitAllTargetInfos | ssa.InitAllTargetMCs | ssa.InitAllAsmPrinters)
	for _, roots := range []bool{false, true} {
		prog := ssa.NewProgram(&ssa.Target{GOOS: "linux", GOARCH: "386"})
		defer prog.Dispose()
		prog.SetRuntime(rt)
		prog.EnableGCRoots(roots)
		pkg := prog.NewPackage("p", "example.com/modeltest")
		for _, name := range llvmattr.RuntimeModelNames {
			if rt.Scope().Lookup(name) == nil {
				t.Fatalf("missing runtime declaration %s", name)
			}
			pkg.RuntimeFunc(name)
		}
		if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
			t.Fatalf("gcRoots=%t: %v", roots, err)
		}
	}
}
