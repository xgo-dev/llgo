//go:build !llgo

package build

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/xgo-dev/llgo/internal/optlevel"
)

// The archive contains physical FMV variants that do not exist in the
// reconstructed frontend module. Both isolated and coordinator links must use
// the archive's source metadata when reusing that machine code.
func TestSIMDFMVArchiveCache(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("FMV execution requires an amd64 host")
	}
	isolateEmbedBuildCache(t)
	t.Setenv(llgoEscapeCache, "0") // Exercise archive reuse without frontend exports.
	root := writeMultiBuildModule(t, map[string]string{
		"main.go": `package main
import (
 "runtime"
 "strings"
 "simd/archsimd"
 "example.com/multibuild/kernel"
)
func check(pcs []uintptr, names ...string) {
 frames := runtime.CallersFrames(pcs)
 counts := make([]int, len(names))
 for {
  frame, more := frames.Next()
  if strings.Contains(frame.Function, "__llgo_fmv") { panic("physical variant leaked into traceback") }
  for i, name := range names {
   if frame.Function == "example.com/multibuild/kernel." + name {
    counts[i]++
    if !strings.HasSuffix(strings.ReplaceAll(frame.File, "\\", "/"), "/kernel.go") || frame.Line == 0 { panic("missing source position") }
   }
  }
  if !more { break }
 }
 for i, count := range counts { if count != 1 { panic("missing or duplicate frame: " + names[i]) } }
}
func main() {
 var pcs [20]uintptr
 var n int
 x := archsimd.BroadcastFloat32x4(5)
 kernel.Trace(x, &pcs, &n)
 check(pcs[:n], "Trace")
 kernel.InlineCaller(x, &pcs, &n)
 check(pcs[:n], "InlineTrace", "InlineCaller")
 kernel.Ordinary(&pcs, &n)
 check(pcs[:n], "Ordinary")
 println("ok")
}
`,
		"kernel/kernel.go": `package kernel
import (
 "runtime"
 "simd/archsimd"
)
//go:noinline
func Trace(x archsimd.Float32x4, pcs *[20]uintptr, n *int) archsimd.Float32x4 {
 if archsimd.X86.AVX2() {
  *n = runtime.Callers(0, pcs[:])
  return x.Add(x)
 }
 *n = runtime.Callers(0, pcs[:])
 return x
}
func InlineTrace(x archsimd.Float32x4, pcs *[20]uintptr, n *int) archsimd.Float32x4 {
 if archsimd.X86.AVX2() { x = x.Add(x) } else { x = x.Sub(x) }
 *n = runtime.Callers(0, pcs[:])
 return x
}
//go:noinline
func InlineCaller(x archsimd.Float32x4, pcs *[20]uintptr, n *int) archsimd.Float32x4 {
 return InlineTrace(x, pcs, n).Mul(x)
}
//go:noinline
func Ordinary(pcs *[20]uintptr, n *int) { *n = runtime.Callers(0, pcs[:]) }
`,
	})
	// simd/archsimd is part of the Go 1.27 experiment.
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/multibuild\n\ngo 1.27\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, phase := range []string{"cold", "warm-isolated", "warm-coordinator"} {
		t.Run(phase, func(t *testing.T) {
			conf := NewDefaultConf(ModeBuild)
			conf.GOEXPERIMENT, conf.GOAMD64 = "simd", "v1"
			conf.OptLevel = optlevel.O2
			conf.BuildParallelism = 2
			conf.OutFile = filepath.Join(t.TempDir(), "fmvcache")
			if runtime.GOOS == "windows" {
				conf.OutFile += ".exe"
			}
			if phase == "warm-coordinator" {
				conf.ModuleHook = func(Package) {}
			}
			pkgs, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: root})
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, pkg := range pkgs {
				if pkg.PkgPath == "example.com/multibuild/kernel" {
					found = true
					if want := phase != "cold"; pkg.CacheHit != want {
						t.Fatalf("kernel cache hit = %v, want %v", pkg.CacheHit, want)
					}
				}
			}
			if !found {
				t.Fatal("missing kernel package")
			}
			for _, cpu := range []string{"", "cpu.avx2=off", "cpu.all=off"} {
				cmd := exec.Command(conf.OutFile)
				cmd.Env = append(os.Environ(), "GODEBUG="+cpu)
				out, err := cmd.CombinedOutput()
				if err != nil || string(out) != "ok\n" {
					t.Fatalf("GODEBUG=%s: %v\n%s", cpu, err, out)
				}
			}
		})
	}
}
