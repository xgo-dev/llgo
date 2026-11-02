//go:build !llgo

package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestSIMDFMVAggregatesLLVM(t *testing.T) {
	for _, goos := range []string{"linux", "darwin", "windows"} {
		t.Run(goos, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.Mkdir(filepath.Join(dir, "kernel"), 0700); err != nil {
				t.Fatal(err)
			}
			for name, source := range map[string]string{
				"go.mod": "module fmvaggregate\n\ngo 1.27\n",
				"kernel/kernel.go": `package kernel
import "simd/archsimd"
type Block struct { Value archsimd.Float32x4; Count int; Label string }
//go:noinline
func Compute(x Block) (Block, int) { x.Value = x.Value.Add(x.Value); x.Count++; return x, x.Count }
//go:noinline
func Array(x [2]archsimd.Float32x4) [2]archsimd.Float32x4 { x[0] = x[0].Add(x[1]); return x }
`,
				"main.go": `package main
import "simd/archsimd"
import "fmvaggregate/kernel"
//go:noinline
func guarded(x []float32, label string, b kernel.Block) (kernel.Block, int) {
 if archsimd.X86.AVX2() { b.Label = label; b.Count += len(x); return kernel.Compute(b) }
 return b, -1
}
//go:noinline
func array(x [2]archsimd.Float32x4) [2]archsimd.Float32x4 {
 if archsimd.X86.AVX2() { return kernel.Array(x) }; return x
}
//go:noinline
func pair(x string) (float64, int) {
 if archsimd.X86.AVX2() { return 1.5, len(x) }; return 2.5, -1
}
type One struct { Value archsimd.Float32x4 }
//go:noinline
func one(x One) One {
 if archsimd.X86.AVX2() { x.Value = x.Value.Add(x.Value) }; return x
}
//go:noinline
func large(x [8193]uint64) [8193]uint64 {
 if archsimd.X86.AVX2() { x[0]++ }; return x
}
func main() {}
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
			// These are FMV roots because they call the CPU query; pair and
			// large have no SIMD values in their signatures.
			for _, name := range []string{"main.guarded", "main.array", "main.pair", "main.one", "main.large"} {
				fast := mod.NamedFunction(name + ".__llgo_fmv_avx2")
				if fast.IsNil() {
					t.Fatalf("function %s has no AVX2 implementation", name)
				}
				if strings.Contains(fast.String(), "X86Features.AVX2") {
					t.Fatalf("unfolded query:\n%s", fast.String())
				}
				assertAggregateTailForwarding(t, mod.NamedFunction(name))
				assertAggregateTailForwarding(t, mod.NamedFunction(name+".__llgo_fmv_resolve"))
			}
			for _, name := range []string{"Compute", "Array"} {
				if mod.NamedFunction("fmvaggregate/kernel." + name + ".__llgo_fmv_avx2").IsNil() {
					t.Fatalf("missing aggregate cross-package variant %s", name)
				}
			}
			if !strings.Contains(mod.NamedFunction("main.guarded.__llgo_fmv_avx2").String(), "kernel.Compute.__llgo_fmv_avx2") {
				t.Fatal("aggregate direct call did not specialize")
			}
			kernels, err := Build(Invocation{Args: []string{"./kernel"}, Config: conf, Dir: dir})
			if err != nil {
				t.Fatal(err)
			}
			defer kernels[0].LPkg.Prog.Dispose()
			for _, name := range []string{"Compute", "Array"} {
				fn := kernels[0].LPkg.Module().NamedFunction("fmvaggregate/kernel." + name + ".__llgo_fmv_avx2")
				if fn.IsNil() || fn.IsDeclaration() {
					t.Fatalf("missing aggregate cross-package definition %s", name)
				}
			}
			simdCodegenVariants(t, []string{"O0", "O2"}, pkgs, func(t *testing.T, asm string) {
				if goos == "linux" {
					assertSIMDBaselineAssembly(t, asm)
				}
			})
		})
	}
}

func assertAggregateTailForwarding(t *testing.T, fn llvm.Value) {
	t.Helper()
	calls := 0
	for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
		for inst := bb.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
			if !inst.IsAAllocaInst().IsNil() {
				t.Fatalf("dispatcher owns forwarding storage:\n%s", fn.String())
			}
			if inst.IsACallInst().IsNil() {
				continue
			}
			calls++
			if inst.TailCallKind() != llvm.TailCallKindMustTail {
				t.Fatalf("lost tail forwarding:\n%s", fn.String())
			}
			if inst.CalledFunctionType() != fn.GlobalValueType() {
				t.Fatalf("mismatched forwarding signature:\n%s", fn.String())
			}
			for i, p := range fn.Params() {
				if inst.Operand(i) != p {
					t.Fatalf("does not forward parameter %d:\n%s", i, fn.String())
				}
			}
			for i := 0; i <= fn.ParamsCount(); i++ {
				for _, name := range []string{"sret", "byval"} {
					attr := fn.GetEnumAttributeAtIndex(i, llvm.AttributeKindID(name))
					if !attr.IsNil() {
						got := inst.GetCallSiteEnumAttribute(i, uint(attr.GetEnumKind()))
						if got.IsNil() || got.GetTypeValue() != attr.GetTypeValue() {
							t.Fatalf("lost ABI attribute at %d:\n%s", i, fn.String())
						}
					}
				}
			}
		}
	}
	if calls == 0 {
		t.Fatalf("missing forwarding call: %s", fn.Name())
	}
}
