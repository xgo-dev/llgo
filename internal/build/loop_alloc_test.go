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
)

// Atomic accesses prevent O2 from scalarizing the loop-local allocation.
// Without the cycle guard, Attributor creates an alloca on every iteration.
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
func main() {
	const n = 2000000
	done := make(chan uint64)
	go func() { done <- work(n) }()
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
