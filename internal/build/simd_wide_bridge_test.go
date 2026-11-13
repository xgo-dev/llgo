//go:build !llgo

package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestSIMD256BridgeLLVM(t *testing.T) {
	for _, goos := range []string{"linux", "windows"} {
		t.Run(goos, func(t *testing.T) {
			dir := t.TempDir()
			for name, source := range map[string]string{
				"go.mod": "module widebridge\n\ngo 1.27\n",
				"main.go": `package main
import "simd/archsimd"
func minimum(x,y archsimd.Float32x8) archsimd.Float32x8 { return x.Min(y) }
func maximum(x,y archsimd.Uint32x8) archsimd.Uint32x8 { return x.Max(y) }
func squareRoot(x archsimd.Float64x4) archsimd.Float64x4 { return x.Sqrt() }
func fused(x,y,z archsimd.Float32x8) archsimd.Float32x8 { return x.MulAdd(y,z) }
func convert(x archsimd.Float32x8) archsimd.Int32x8 { return x.ConvertToInt32() }
func average(x,y archsimd.Uint16x16) archsimd.Uint16x16 { return x.Average(y) }
func saturated(x,y archsimd.Int8x32) archsimd.Int8x32 { return x.AddSaturated(y).SubSaturated(y) }
func polynomial(x,y archsimd.Uint64x4) archsimd.Uint64x4 { return x.CarrylessMultiplyOddEven(y) }
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
			pkg := pkgs[0].LPkg
			defer pkg.Prog.Dispose()
			mod := pkg.Module()
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			checks := map[string][]string{
				"minimum":    {"fcmp olt <8 x float>", "select <8 x i1>"},
				"maximum":    {"llvm.umax.v8i32"},
				"squareRoot": {"llvm.sqrt.v4f64"},
				"convert":    {"llvm.fptosi.sat.v8i32.v8f32", "fcmp uge <8 x float>"},
				"average":    {"lshr <16 x i16>", "add <16 x i16>"},
				"saturated":  {"llvm.sadd.sat.v32i8", "llvm.ssub.sat.v32i8"},
			}
			if goos == "windows" {
				checks["fused"] = []string{"SIMDFMA32"}
			} else {
				checks["fused"] = []string{"llvm.fma.v8f32"}
			}
			for name, wants := range checks {
				fn := mod.NamedFunction("main." + name)
				if fn.IsNil() {
					t.Fatalf("missing %s", name)
				}
				for _, want := range wants {
					if !strings.Contains(fn.String(), want) {
						t.Fatalf("%s missing %s:\n%s", name, want, fn.String())
					}
				}
			}
			mod.SetTarget(pkg.Prog.Target().Spec().Triple)
			mod.SetDataLayout(pkg.Prog.DataLayout())
			for _, level := range []string{"O0", "O2", "O3"} {
				if level != "O0" {
					opts := llvm.NewPassBuilderOptions()
					opts.SetVerifyEach(true)
					err := mod.RunPasses("default<"+level+">", pkg.Prog.TargetMachine(), opts)
					opts.Dispose()
					if err != nil {
						t.Fatal(err)
					}
				}
				asm, err := pkg.Prog.TargetMachine().EmitToMemoryBuffer(mod, llvm.AssemblyFile)
				if err != nil {
					t.Fatal(err)
				}
				if goos == "linux" {
					assertSIMDBaselineAssembly(t, string(asm.Bytes()))
				}
				asm.Dispose()
			}
		})
	}
}
