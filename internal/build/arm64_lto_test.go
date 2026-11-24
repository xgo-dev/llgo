//go:build !llgo

package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestDarwinARM64AtomicLTOBitcode(t *testing.T) {
	llssa.Initialize(llssa.InitAll)
	linkTarget := llssa.NewProgram(&llssa.Target{GOOS: "darwin", GOARCH: "arm64", GOARM64: "v8.0"})
	defer linkTarget.Dispose()

	dir := t.TempDir()
	for name, source := range map[string]string{
		"go.mod": "module atomiclto\n\ngo 1.26\n",
		"atomic.go": `package atomiclto
import "sync/atomic"
func Add(p *uint64, v uint64) uint64 { return atomic.AddUint64(p, v) }
func SwapIf(p *uint64, old, next uint64) bool { return atomic.CompareAndSwapUint64(p, old, next) }
`,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, isa := range []struct {
		level string
		lse   bool
	}{
		{"v8.0", false},
		{"v8.0,lse", true},
		{"v8.1", true},
	} {
		t.Run(isa.level, func(t *testing.T) {
			conf := NewDefaultConf(ModeGen)
			conf.Goos, conf.Goarch, conf.GOARM64 = "darwin", "arm64", isa.level
			conf.OptLevel = optlevel.O3
			pkgs, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: dir})
			if err != nil {
				t.Fatal(err)
			}
			pkg := pkgs[0].LPkg
			defer pkg.Prog.Dispose()
			mod := pkg.Module()

			// Read bitcode with a generic link target, as the Darwin linker
			// can create a target machine without the source GOARM64 setting.
			for _, mode := range []lto.Mode{lto.Full, lto.Thin} {
				t.Run(mode.String(), func(t *testing.T) {
					conf.LTO = mode
					buf, _, err := emitObjectToMemoryBuffer(conf, pkg.Prog, mod)
					if err != nil {
						t.Fatal(err)
					}
					defer buf.Dispose()
					path := filepath.Join(t.TempDir(), "atomic.bc")
					if err := os.WriteFile(path, buf.Bytes(), 0600); err != nil {
						t.Fatal(err)
					}
					linked, err := mod.Context().ParseBitcodeFile(path)
					if err != nil {
						t.Fatal(err)
					}
					defer linked.Dispose()
					asm, err := linkTarget.TargetMachine().EmitToMemoryBuffer(linked, llvm.AssemblyFile)
					if err != nil {
						t.Fatal(err)
					}
					defer asm.Dispose()
					code := string(asm.Bytes())
					for _, op := range []string{"ldaddal", "casal"} {
						if strings.Contains(code, "\t"+op+"\t") != isa.lse {
							t.Fatalf("unexpected %s for %s:\n%s", op, isa.level, code)
						}
					}
					for _, op := range []string{"ldaxr", "stlxr"} {
						if strings.Contains(code, "\t"+op+"\t") == isa.lse {
							t.Fatalf("unexpected %s for %s:\n%s", op, isa.level, code)
						}
					}
				})
			}
		})
	}
}
