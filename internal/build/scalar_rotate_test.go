package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestMathBitsRotateSourcePatch(t *testing.T) {
	dir := t.TempDir()
	for name, source := range map[string]string{
		"go.mod": "module scalarrotate\n\ngo 1.27\n",
		"main.go": `package main
import "math/bits"
func r8(x uint8, k int) uint8 { return bits.RotateLeft8(x, k) }
func r16(x uint16, k int) uint16 { return bits.RotateLeft16(x, k) }
func r32(x uint32, k int) uint32 { return bits.RotateLeft32(x, k) }
func r64(x uint64, k int) uint64 { return bits.RotateLeft64(x, k) }
func word(x uint, k int) uint { return bits.RotateLeft(x, k) }
func main() {}
`,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	pkgs, err := Build(Invocation{Args: []string{"."}, Dir: dir, Config: NewDefaultConf(ModeGen)})
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("got %d packages, want 1", len(pkgs))
	}
	pkg := pkgs[0].LPkg
	defer pkg.Prog.Dispose()
	mod := pkg.Module()
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		width int
	}{{"r8", 8}, {"r16", 16}, {"r32", 32}, {"r64", 64}, {"word", int(pkg.Prog.SizeOf(pkg.Prog.Uintptr())) * 8}} {
		fn := mod.NamedFunction("main." + tc.name)
		if fn.IsNil() || !strings.Contains(fn.String(), fmt.Sprintf("@llvm.fshl.i%d(", tc.width)) {
			t.Fatalf("math/bits source patch did not lower %s to a funnel shift", tc.name)
		}
	}
}
