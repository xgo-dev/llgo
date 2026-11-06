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

// A source-level tail-position call with a fresh object and a callee that
// overwrites stack storage. It must work whether or not a tail marker is
// present when heap-to-stack runs.
const tailAllocSource = `package main
import "sync/atomic"

//go:noinline
func take(p *uint64) uint64 {
	var scratch [128]uint64
	for i := range scratch {
		atomic.StoreUint64(&scratch[i], 99)
	}
	return atomic.LoadUint64(p)
}

//go:noinline
func caller() uint64 {
	p := new(uint64)
	atomic.StoreUint64(p, 42)
	return take(p)
}

func main() {
	if caller() != 42 { panic("tail allocation lifetime") }
	println("tail-ok")
}
`

func TestTailAllocationGo(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	t.Setenv(llgoBuildCache, "off")
	dir := t.TempDir()
	for _, file := range []struct{ name, source string }{
		{"go.mod", "module tailalloc\n\ngo 1.27.0\n"}, {"main.go", tailAllocSource},
	} {
		if err := os.WriteFile(filepath.Join(dir, file.name), []byte(file.source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	conf := NewDefaultConf(ModeGen)
	conf.OptLevel = optlevel.O2
	pkgs, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	pkg := pkgs[0].LPkg
	defer pkg.Prog.Dispose()
	mod := pkg.Module()
	t.Logf("caller before heap-to-stack:\n%s", mod.NamedFunction("main.caller").String())
	ctx := &context{prog: pkg.Prog, buildConf: conf, passOpt: true}
	if err := optimizeLLVMModule(ctx, pkg.Path(), mod); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	t.Logf("caller after optimization:\n%s", mod.NamedFunction("main.caller").String())
	for _, mode := range []lto.Mode{lto.Off, lto.Full} {
		conf := NewDefaultConf(ModeBuild)
		conf.OptLevel, conf.LTO = optlevel.O2, mode
		conf.OutFile = filepath.Join(dir, "tail")
		if runtime.GOOS == "windows" {
			conf.OutFile += ".exe"
		}
		if _, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: dir}); err != nil {
			t.Fatal(err)
		}
		out, err := exec.CommandContext(t.Context(), conf.OutFile).CombinedOutput()
		if err != nil || strings.TrimSpace(string(out)) != "tail-ok" {
			t.Fatalf("LTO %v: %v\n%s", mode, err, out)
		}
	}
}
