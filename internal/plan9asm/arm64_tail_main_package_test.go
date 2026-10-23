package plan9asm

import (
	"fmt"
	"path/filepath"
	"testing"

	llssaabi "github.com/xgo-dev/llgo/ssa/abi"
	"github.com/xgo-dev/llvm"
)

func TestTranslateARM64ABI0TailMainPackageIdentity(t *testing.T) {
	for _, owner := range []struct {
		name string
		path string
	}{
		{"canonical", "main"},
		{"module", "example.com/cmd/abizero"},
		{"command-line", "command-line-arguments"},
	} {
		for _, overlay := range []bool{false, true} {
			for _, tuple := range []bool{false, true} {
				for _, preprocessed := range []bool{false, true} {
					name := fmt.Sprintf("%s/overlay=%t/tuple=%t/preprocessed=%t", owner.name, overlay, tuple, preprocessed)
					t.Run(name, func(t *testing.T) {
						// Parse and type-check the complete actual main package;
						// never rename or replace its types.Package after checking.
						pkg, opt, source := arm64TailSelectionPackageFixture(t, overlay, tuple, owner.path, "main")
						originalTypes := pkg.Types
						originalDeclaration := pkg.Types.Scope().Lookup("Slots")
						if pkg.Types.Path() != owner.path || llssaabi.PathOf(pkg.Types) != "main" {
							t.Fatal("fixture lost the distinct source owner and executable symbol namespace")
						}
						opt.SourcePreprocessed = preprocessed
						file := filepath.Join(filepath.Dir(pkg.CompiledGoFiles[0]), "slots_arm64.s")
						tr, err := TranslateSourceModuleForPkgWithOptions(pkg, file, []byte(source), "linux", "arm64", opt)
						if err != nil {
							t.Fatal("actual main-package ABI0 tail lost its selected source identity:", err)
						}
						defer tr.Module.Dispose()
						if err := llvm.VerifyModule(tr.Module, llvm.ReturnStatusAction); err != nil {
							t.Fatal(err)
						}
						entry := tr.Module.NamedFunction("main.Slots")
						if entry.IsNil() || entry.BasicBlocksCount() == 0 {
							t.Fatal("translated main-package ABI0 entry has no body")
						}
						if owner.path != "main" && !tr.Module.NamedFunction(owner.path+".Slots").IsNil() {
							t.Fatal("source owner escaped into the executable symbol namespace")
						}
						if pkg.Types != originalTypes || pkg.Types.Path() != owner.path ||
							pkg.Types.Scope().Lookup("Slots") != originalDeclaration {
							t.Fatal("translation changed the original type-checked source owner")
						}
					})
				}
			}
		}
	}
}
