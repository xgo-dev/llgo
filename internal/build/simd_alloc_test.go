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

const simdArrayAllocSource = `package main

import "simd/archsimd"

//go:noinline
func arrayAdd(a, b uint64) uint64 {
	x := archsimd.LoadUint64x2Array(&[2]uint64{a, 0})
	y := archsimd.LoadUint64x2Array(&[2]uint64{b, 0})
	var out [2]uint64
	x.Add(y).StoreArray(&out)
	return out[0]
}

func zeroed() uint64 {
	var out [2]uint64
	return archsimd.LoadUint64x2Array(&out).GetElem(0)
}

func escaped(a uint64) *[2]uint64 {
	var out [2]uint64
	archsimd.LoadUint64x2Array(&[2]uint64{a, 0}).StoreArray(&out)
	return &out
}

var saved *[2]uint64

func published(a uint64) { saved = escaped(a) }

func main() {}
`

// Exercise heap-to-stack before ABI lowering and the ordinary pipeline: merely
// tagging the source declarations, or running O2 without Attributor, leaves
// the local buffers on the heap (issue #2983).
func TestSIMDArrayAllocationsO2(t *testing.T) {
	dir := t.TempDir()
	for _, file := range []struct{ name, source string }{
		{"go.mod", "module simdalloc\n\ngo 1.27\n"}, {"main.go", simdArrayAllocSource},
	} {
		if err := os.WriteFile(filepath.Join(dir, file.name), []byte(file.source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, target := range []struct{ os, arch string }{
		{"linux", "amd64"}, {"darwin", "arm64"}, {"windows", "amd64"},
	} {
		for _, mode := range []lto.Mode{lto.Off, lto.Full, lto.Thin} {
			t.Run(target.os+"/"+target.arch+"/"+mode.String(), func(t *testing.T) {
				conf := NewDefaultConf(ModeGen)
				conf.Goos, conf.Goarch, conf.GOEXPERIMENT = target.os, target.arch, "simd"
				conf.OptLevel, conf.LTO = optlevel.O2, mode
				if target.arch == "amd64" {
					conf.GOAMD64 = "v1"
				}
				optimizeTestAllocationsBeforeABI(t, conf, func(mod llvm.Module) {
					if body := mod.NamedFunction("main.arrayAdd").String(); strings.Count(body, "runtime.AllocAtomicU") != 3 {
						t.Fatalf("fixture does not exercise three local buffers:\n%s", body)
					}
				})
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
				for _, name := range []string{"arrayAdd", "zeroed"} {
					body := mod.NamedFunction("main." + name).String()
					if strings.Contains(body, "runtime.Alloc") || strings.Contains(body, "alloca ") {
						t.Fatalf("%s retained a buffer:\n%s", name, body)
					}
					if name == "zeroed" && !strings.Contains(body, "ret i64 0") {
						t.Fatalf("zero initialization lost:\n%s", body)
					}
				}
				for _, name := range []string{"escaped", "published"} {
					body := mod.NamedFunction("main." + name).String()
					if !strings.Contains(body, "runtime.AllocAtomicU") {
						t.Fatalf("%s lost its escaping allocation:\n%s", name, body)
					}
				}
			})
		}
	}
}

func TestSIMDArrayAllocationsExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs the SIMD allocation regression")
	}
	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		t.Skip("requires a native archsimd target")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	dir := t.TempDir()
	source := strings.Replace(simdArrayAllocSource, `import "simd/archsimd"`, `import ("simd/archsimd"; "runtime"; "fmt")`, 1)
	source = strings.Replace(source, "func main() {}", `func main() {
	const n uint64 = 100000
	for round := 0; round < 2; round++ {
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		var sum uint64
		for i := uint64(0); i < n; i++ {
			sum += arrayAdd(i, 17)
		}
		runtime.ReadMemStats(&after)
		if sum != n*(n-1)/2+17*n {
			panic("wrong SIMD sum")
		}
		// TotalAlloc is approximate under BDWGC. Allow incidental runtime work,
		// while rejecting the old roughly 9.6 MB per round of local buffers.
		if bytes := after.TotalAlloc - before.TotalAlloc; bytes > 256*1024 {
			panic(fmt.Sprintf("local SIMD buffers allocated %d bytes", bytes))
		}
	}
	p := escaped(42)
	runtime.GC()
	if p[0] != 42 || p[1] != 0 {
		panic("escaped buffer lost")
	}
	fmt.Println("ok")
}
`, 1)
	for _, file := range []struct{ name, contents string }{
		{"go.mod", "module simdalloc\n\ngo 1.27\n"}, {"main.go", source},
	} {
		if err := os.WriteFile(filepath.Join(dir, file.name), []byte(file.contents), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, mode := range []lto.Mode{lto.Off, lto.Full} {
		t.Run(mode.String(), func(t *testing.T) {
			conf := NewDefaultConf(ModeBuild)
			conf.GOEXPERIMENT, conf.OptLevel, conf.LTO = "simd", optlevel.O2, mode
			conf.OutFile = filepath.Join(dir, "probe-"+mode.String())
			if runtime.GOOS == "windows" {
				conf.OutFile += ".exe"
			}
			if _, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: dir}); err != nil {
				t.Fatal(err)
			}
			out, err := exec.CommandContext(t.Context(), conf.OutFile).CombinedOutput()
			if err != nil || strings.TrimSpace(string(out)) != "ok" {
				t.Fatalf("SIMD allocation regression: %v\n%s", err, out)
			}
		})
	}
}
