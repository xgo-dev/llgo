package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

// Check the complete source-patch and package-loading path. A local-only
// frontend test cannot detect a missing math/bits source-patch registration.
func TestLLVMIntrinsicWrapperAcrossPackages(t *testing.T) {
	file := filepath.Join(t.TempDir(), "adapter.go")
	if err := os.WriteFile(file, []byte(`package adapter
import "math/bits"
func Rotate(x uint8, k int) uint8 { return bits.RotateLeft8(x, k) }
var RotateValue = bits.RotateLeft8
`), 0644); err != nil {
		t.Fatal(err)
	}
	pkgs, err := Do([]string{file}, NewDefaultConf(ModeGen))
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 || pkgs[0].LPkg == nil {
		t.Fatalf("expected one compiled package, got %v", pkgs)
	}
	pkg := pkgs[0].LPkg
	defer pkg.Prog.Dispose()
	mod := pkg.Module()
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	ir := mod.String()
	if !strings.Contains(ir, "@llvm.fshl.i8(") || strings.Contains(ir, "call i8 @\"math/bits.RotateLeft8\"(") {
		t.Fatalf("cross-package adapter did not expand without optimization:\n%s", ir)
	}
	if !strings.Contains(ir, "@\"math/bits.RotateLeft8\"") {
		t.Fatalf("ordinary Go function-value entry disappeared:\n%s", ir)
	}
}
