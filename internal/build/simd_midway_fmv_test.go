//go:build !llgo

package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestSIMDMidwayFMVLLVM(t *testing.T) {
	for _, goos := range []string{"linux", "darwin", "windows"} {
		for _, level := range []string{"O0", "O2"} {
			t.Run(goos+"/"+level, func(t *testing.T) {
				dir := t.TempDir()
				if err := os.Mkdir(filepath.Join(dir, "kernel"), 0700); err != nil {
					t.Fatal(err)
				}
				for name, source := range map[string]string{
					"go.mod": "module midwayfmv\n\ngo 1.27\n",
					"kernel/kernel.go": `package kernel
import "simd"
type Block struct { Value [2]simd.Uint32s; Count int; Label string }
//go:noinline
func Compute(x Block) (Block, int) { x.Value[0] = x.Value[0].Add(x.Value[1]); x.Count++; return x, x.Count }
//go:noinline
func Array(x [2]simd.Uint32s) [2]simd.Uint32s { x[0] = x[0].Add(x[1]); return x }
`,
					"main.go": `package main
import "simd"
import "simd/archsimd"
import "midwayfmv/kernel"
//go:noinline
func guarded(x []uint32, label string, b kernel.Block) (kernel.Block, int) {
 if archsimd.X86.AVX2() { b.Label = label; b.Count += len(x); return kernel.Compute(b) }
 return b, -1
}
//go:noinline
func array(x [2]simd.Uint32s) [2]simd.Uint32s {
 if archsimd.X86.AVX2() { return kernel.Array(x) }; return x
}
//go:noinline
func sum(x uint32) int {
 b:=kernel.Block{Value:[2]simd.Uint32s{simd.BroadcastUint32s(x),simd.BroadcastUint32s(1)}}
 _,n:=guarded(nil,"label",b);return n
}
func main() { println(sum(3)) }
`,
				} {
					if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0600); err != nil {
						t.Fatal(err)
					}
				}
				conf := NewDefaultConf(ModeGen)
				conf.Goos, conf.Goarch, conf.GOAMD64, conf.GOEXPERIMENT = goos, "amd64", "v1", "simd"
				pkgs, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: dir})
				if err != nil {
					t.Fatal(err)
				}
				defer pkgs[0].LPkg.Prog.Dispose()
				mod := pkgs[0].LPkg.Module()
				if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(mod.NamedFunction("main.sum").String(), "simd.Emulated") {
					t.Fatal("lost Midway scalar dispatch")
				}
				for _, name := range []string{"main.guarded@simd0", "main.guarded@simd128", "main.array@simd0", "main.array@simd128"} {
					fast := mod.NamedFunction(name + ".__llgo_fmv_avx2")
					if fast.IsNil() || fast.IsDeclaration() {
						t.Fatalf("missing composed variant %s", name)
					}
					if strings.Contains(fast.String(), "X86Features.AVX2") {
						t.Fatalf("unfolded AVX2 query: %s", fast.String())
					}
					assertAggregateTailForwarding(t, mod.NamedFunction(name))
					assertAggregateTailForwarding(t, mod.NamedFunction(name+".__llgo_fmv_resolve"))
				}
				for _, name := range []string{"Compute", "Array"} {
					if mod.NamedFunction("midwayfmv/kernel." + name + "@simd128.__llgo_fmv_avx2").IsNil() {
						t.Fatalf("missing aggregate cross-package variant %s", name)
					}
					if !mod.NamedFunction("midwayfmv/kernel." + name + "@simd0.__llgo_fmv_avx2").IsNil() {
						t.Fatalf("emulated leaf unexpectedly eligible for FMV: %s", name)
					}
				}
				if !strings.Contains(mod.NamedFunction("main.guarded@simd128.__llgo_fmv_avx2").String(), "kernel.Compute@simd128.__llgo_fmv_avx2") {
					t.Fatal("aggregate direct call did not specialize")
				}
				kernels, err := Build(Invocation{Args: []string{"./kernel"}, Config: conf, Dir: dir})
				if err != nil {
					t.Fatal(err)
				}
				defer kernels[0].LPkg.Prog.Dispose()
				if err := llvm.VerifyModule(kernels[0].LPkg.Module(), llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
				for _, name := range []string{"Compute", "Array"} {
					fn := kernels[0].LPkg.Module().NamedFunction("midwayfmv/kernel." + name + "@simd128.__llgo_fmv_avx2")
					if fn.IsNil() || fn.IsDeclaration() {
						t.Fatalf("missing aggregate cross-package definition %s", name)
					}
				}
				for _, compiled := range append(pkgs, kernels...) {
					prog := compiled.LPkg.Prog
					mod := compiled.LPkg.Module()
					mod.SetDataLayout(prog.DataLayout())
					mod.SetTarget(prog.Target().Spec().Triple)
					if level == "O2" {
						opts := llvm.NewPassBuilderOptions()
						defer opts.Dispose()
						opts.SetVerifyEach(true)
						if err := mod.RunPasses("default<O2>", prog.TargetMachine(), opts); err != nil {
							t.Fatal(err)
						}
					}
					asm, err := prog.TargetMachine().EmitToMemoryBuffer(mod, llvm.AssemblyFile)
					if err != nil {
						t.Fatal(err)
					}
					defer asm.Dispose()
					if goos == "linux" {
						assertSIMDBaselineAssembly(t, string(asm.Bytes()))
					}
				}
			})
		}
	}
}
