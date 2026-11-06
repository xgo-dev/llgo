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

// Atomic accesses prevent O2 from scalarizing the loop-local allocation.
// Promoted slots must be hoisted without moving per-iteration initialization.
const loopAllocSource = `package main

import "sync/atomic"

//go:noinline
func work(n uint64) uint64 {
	var sum uint64
	for i := uint64(0); i < n; i++ {
		p := new(uint64)
		atomic.StoreUint64(p, i)
		sum += atomic.LoadUint64(p)
	}
	return sum
}
//go:noinline
func workZeroed(n uint64) uint64 {
 var sum uint64
 for i := uint64(0); i < n; i++ {
  p := new(uint64)
  sum += atomic.LoadUint64(p)
  atomic.StoreUint64(p, 99)
 }
 return sum
}

//go:noinline
func workCarried(n uint64) uint64 {
 previous := new(uint64)
 var sum uint64
 for i := uint64(0); i < n; i++ {
  p := new(uint64)
  *p = i + 1
  old := *previous
  q := new(uint64)
  atomic.StoreUint64(q, old)
  sum += atomic.LoadUint64(q)
  previous = p
 }
 return sum
}
func main() {
	const n = 2000000
	done := make(chan uint64)
	go func() {
  if workZeroed(n) != 0 { panic("zero initialization") }
  if workCarried(n) != n*(n-1)/2 { panic("overlapping objects") }
  done <- work(n)
 }()
	if <-done != n*(n-1)/2 {
		panic("checksum")
	}
	println("loop-ok")
}
`

func TestLoopAllocationBoundedStack(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs the loop allocation regression")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	dir := t.TempDir()
	for _, file := range []struct{ name, source string }{
		{"go.mod", "module loopalloc\n\ngo 1.27.0\n"}, {"main.go", loopAllocSource},
	} {
		if err := os.WriteFile(filepath.Join(dir, file.name), []byte(file.source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		name string
		opt  optlevel.Level
		lto  lto.Mode
	}{
		{"default", optlevel.Unset, lto.Off},
		{"O2", optlevel.O2, lto.Off},
		{"O2-full", optlevel.O2, lto.Full},
	} {
		t.Run(tc.name, func(t *testing.T) {
			conf := NewDefaultConf(ModeBuild)
			conf.OptLevel, conf.LTO = tc.opt, tc.lto
			// The worker uses a fixed 1 MiB native thread stack. Two million
			// surviving 16-byte loop allocas would exhaust it many times over.
			conf.PthreadStackSize, conf.PthreadStackSizeSet = 1<<20, true
			conf.OutFile = filepath.Join(dir, tc.name)
			if runtime.GOOS == "windows" {
				conf.OutFile += ".exe"
			}
			if _, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: dir}); err != nil {
				t.Fatal(err)
			}
			out, err := exec.CommandContext(t.Context(), conf.OutFile).CombinedOutput()
			if err != nil || strings.TrimSpace(string(out)) != "loop-ok" {
				t.Fatalf("loop allocation exceeded bounded stack or changed results: %v\n%s", err, out)
			}
		})
	}
}

// Check optimization as well as bounded-stack execution: keeping every loop
// allocation on the heap would pass the runtime check alone.
func TestLoopAllocationPlacement(t *testing.T) {
	dir := t.TempDir()
	for _, file := range []struct{ name, source string }{
		{"go.mod", "module loopalloc\n\ngo 1.27.0\n"}, {"main.go", loopAllocSource},
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
	ctx := &context{prog: pkg.Prog, buildConf: conf, passOpt: true}
	if err := optimizeLLVMModule(ctx, pkg.Path(), mod); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"work", "workZeroed"} {
		if body := mod.NamedFunction("main." + name).String(); strings.Contains(body, "runtime.Alloc") {
			t.Fatalf("%s retains iteration-local heap allocation:\n%s", name, body)
		}
	}
	if body := mod.NamedFunction("main.workCarried").String(); strings.Count(body, "runtime.AllocZ") != 1 {
		t.Fatalf("expected only the loop-carried object on heap:\n%s", body)
	}
}
