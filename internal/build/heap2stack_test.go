//go:build !llgo

package build

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llvm"
)

// ModeGen keeps LLVM passes disabled so IR fixtures can inspect their output.
// Run allocation optimization at its production boundary, then let Build
// perform ABI lowering before the test runs the ordinary LLVM pipeline.
func optimizeTestAllocationsBeforeABI(t *testing.T, conf *Config, before func(llvm.Module)) {
	t.Helper()
	conf.ModuleHook = func(pkg Package) {
		if pkg.Name != "main" {
			return
		}
		mod := pkg.LPkg.Module()
		if before != nil {
			before(mod)
		}
		ctx := &context{prog: pkg.LPkg.Prog, buildConf: conf, passOpt: true}
		lowerLargeAggregates(ctx.prog, mod)
		if err := optimizeHeapAllocations(ctx, pkg.PkgPath, mod); err != nil {
			t.Fatal(err)
		}
	}
}

func TestHeapToStackABIExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs aggregate ABI allocation regression")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	dir := t.TempDir()
	const source = `package main
type triple struct { a, b, c uint64 }
//go:noinline
func read(p *uint64) triple { return triple{*p, *p + 1, *p + 2} }
//go:noinline
func sum(v triple) uint64 { return v.a + v.b + v.c }
//go:noinline
func local(n uint64) triple {
 p := new(uint64)
 *p = n
 return read(p)
}
func main() {
 for n := uint64(1); n < 100; n++ {
  v := local(n)
  if v != (triple{n, n+1, n+2}) || sum(v) != 3*n+3 { panic("aggregate ABI") }
 }
 println("abi-ok")
}
`
	for name, text := range map[string]string{
		"go.mod": "module allocabi\n\ngo 1.27.0\n", "main.go": source,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, mode := range []lto.Mode{lto.Off, lto.Full} {
		t.Run(mode.String(), func(t *testing.T) {
			conf := NewDefaultConf(ModeBuild)
			conf.OptLevel, conf.LTO = optlevel.O2, mode
			conf.OutFile = filepath.Join(dir, "abi-"+mode.String())
			if runtime.GOOS == "windows" {
				conf.OutFile += ".exe"
			}
			if _, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: dir}); err != nil {
				t.Fatal(err)
			}
			out, err := exec.CommandContext(t.Context(), conf.OutFile).CombinedOutput()
			if err != nil || strings.TrimSpace(string(out)) != "abi-ok" {
				t.Fatalf("aggregate ABI execution: %v\n%s", err, out)
			}
		})
	}
}
