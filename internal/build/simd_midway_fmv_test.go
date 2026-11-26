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
		t.Run(goos, func(t *testing.T) {
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
//go:noinline
func optional(p *[4]float32) {
 var portable simd.Uint32s
 _ = portable
 if archsimd.X86.FMA() { x:=archsimd.LoadFloat32x4Array(p); x.MulAdd(x,x).StoreArray(p) }
}
//go:noinline
func avx2(p *[8]int32) {
 var portable simd.Uint32s
 _ = portable
 if archsimd.X86.AVX2() { x:=archsimd.LoadInt32x8Array(p); x.Add(x).StoreArray(p) }
}
//go:noinline
func ordinary(p *[4]float32) { optional(p) }
//go:noinline
func gated(p *[4]float32) { if archsimd.X86.FMA() { optional(p) } }
//go:noinline
func sameWidth(p *[4]float32) { var portable simd.Uint32s; _ = portable; optional(p) }
type box struct{ V simd.Uint32s }
//go:noinline
func (b *box) marker() int { return 7 }
//go:noinline
func plain[T any](x T) int { return 17 }
//go:noinline
func width[T ~int](x T) T { var v simd.Uint32s; _=v; return x+1 }
func identity() func() int { var b box; v:=b.marker()+plain(&b)+int(width(1)); return func() int { return v } }
func main() { println(identity()()); var p [4]float32; ordinary(&p); gated(&p); sameWidth(&p); var ints [8]int32; avx2(&ints); println(sum(3)) }
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
			for _, caller := range []string{"main.ordinary", "main.gated"} {
				assertDirectSIMDCall(t, mod.NamedFunction(caller), "main.optional")
			}
			for _, tc := range []struct{ suffix, floor string }{{"0", ""}, {"128", "+avx"}, {"256", "+avx2"}, {"512", "+avx512f"}} {
				width := "@simd" + tc.suffix
				entry := mod.NamedFunction("main.optional" + width)
				if entry.IsNil() {
					t.Fatalf("missing Midway implementation for %s", width)
				}
				assertDirectSIMDCall(t, mod.NamedFunction("main.optional"), entry.Name())
				assertDirectSIMDCall(t, mod.NamedFunction("main.sameWidth"+width), entry.Name())
				bodies := []llvm.Value{entry}
				if goos == "windows" {
					// Win64 lowers FMA to the scalar runtime helper; the original
					// conditional call must still observe the source CPU query.
					if !strings.Contains(entry.String(), "runtime.SIMDFMA32") || !strings.Contains(entry.String(), "X86Features.FMA") {
						t.Fatalf("missing guarded runtime FMA in %s", entry.Name())
					}
				} else {
					assertAggregateTailForwarding(t, entry)
					fast := mod.NamedFunction(entry.Name() + ".__llgo_fmv_guard_fma")
					if fast.IsNil() || !strings.Contains(fast.String(), "llvm.fma.") {
						t.Fatalf("%s lost its independent FMA implementation", entry.Name())
					}
					bodies = append(bodies, fast)
				}
				for _, fn := range bodies {
					attr := fn.GetStringAttributeAtIndex(-1, "target-features")
					if tc.floor != "" && (attr.IsNil() || !strings.Contains(attr.GetStringValue(), tc.floor)) {
						t.Fatalf("%s lacks %s", fn.Name(), tc.floor)
					}
				}
				for _, tc := range []struct {
					name           string
					implementation bool
				}{
					{"main.(*box" + width + ").marker", true}, {"main.width" + width + "[int]", true},
					{"main.plain[*main.box" + width + "]", false}, {"main.identity" + width + "$1", false},
				} {
					fn := mod.NamedFunction(tc.name)
					if fn.IsNil() {
						t.Fatalf("missing %s", tc.name)
					}
					floor := fn.GetStringAttributeAtIndex(-1, "llgo.cpu.midway-floor")
					if got, want := !floor.IsNil(), tc.implementation && width != "@simd0"; got != want {
						t.Fatalf("%s has Midway floor = %v, want %v", tc.name, got, want)
					}
				}
				avx2 := mod.NamedFunction("main.avx2" + width)
				if tc.suffix == "256" || tc.suffix == "512" {
					if !mod.NamedFunction(avx2.Name()+".__llgo_fmv_resolve").IsNil() || !strings.Contains(avx2.String(), "X86Features.AVX2") {
						t.Fatalf("%s must preserve the source query without another ISA dispatch", avx2.Name())
					}
				} else if mod.NamedFunction(avx2.Name() + ".__llgo_fmv_guard_avx2").IsNil() {
					t.Fatalf("%s needs a local AVX2 implementation", avx2.Name())
				}
				for _, pair := range [][2]string{{"guarded", "Compute"}, {"array", "Array"}} {
					caller := mod.NamedFunction("main." + pair[0] + width)
					target := "midwayfmv/kernel." + pair[1] + width
					if fast := mod.NamedFunction(caller.Name() + ".__llgo_fmv_guard_avx2"); !fast.IsNil() {
						caller = fast
					}
					assertDirectSIMDCall(t, caller, target)
				}
			}
			kernels, err := Build(Invocation{Args: []string{"./kernel"}, Config: conf, Dir: dir})
			if err != nil {
				t.Fatal(err)
			}
			defer kernels[0].LPkg.Prog.Dispose()
			if err := llvm.VerifyModule(kernels[0].LPkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			for _, width := range []string{"0", "128", "256", "512"} {
				for _, name := range []string{"Compute", "Array"} {
					sym := "midwayfmv/kernel." + name + "@simd" + width
					fn := kernels[0].LPkg.Module().NamedFunction(sym)
					if fn.IsNil() || fn.IsDeclaration() {
						t.Fatalf("missing cross-package definition %s", sym)
					}
				}
			}
			simdCodegenVariants(t, []string{"O0", "O2"}, append(pkgs, kernels...), func(t *testing.T, mod llvm.Module, asm string) {
				if goos == "linux" {
					assertSIMDBaselineAssembly(t, mod, asm)
				}
			})
		})
	}
}

func assertDirectSIMDCall(t *testing.T, fn llvm.Value, target string) {
	t.Helper()
	if !fn.IsNil() {
		for b := fn.FirstBasicBlock(); !b.IsNil(); b = llvm.NextBasicBlock(b) {
			for i := b.FirstInstruction(); !i.IsNil(); i = llvm.NextInstruction(i) {
				if (!i.IsACallInst().IsNil() || !i.IsAInvokeInst().IsNil()) && i.CalledValue().Name() == target {
					return
				}
			}
		}
	}
	t.Fatalf("missing direct call to %s in %v", target, fn)
}
