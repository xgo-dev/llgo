//go:build !llgo

package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestSIMDPortableBridgeLLVM(t *testing.T) {
	for _, target := range []struct{ os, arch string }{
		{"linux", "amd64"}, {"windows", "amd64"}, {"darwin", "arm64"}, {"wasip1", "wasm"},
	} {
		t.Run(target.os+"/"+target.arch, func(t *testing.T) {
			dir := t.TempDir()
			for name, source := range map[string]string{
				"go.mod": "module simdbridge\n\ngo 1.27\n",
				"main.go": `package main
import "simd/archsimd"
func average(x,y archsimd.Uint8x16) archsimd.Uint8x16 { return x.Average(y) }
func madd(x,y,z archsimd.Float32x4) archsimd.Float32x4 { return x.MulAdd(y,z) }
func rotate(x archsimd.Uint32x4, n uint64) archsimd.Uint32x4 { return x.RotateAllLeft(n) }
func main() {}
`,
				"masked_amd64.go": `package main
import "simd/archsimd"
func masked(x archsimd.Uint32x4, p *[4]uint32, mask archsimd.Mask32x4, n int) int {
 for i:=0; i<n; i++ { x.StoreArrayMasked(p,mask) }; return n+1
}
`,
				"widen_wasm.go": `package main
import "simd/archsimd"
func widen(x,y archsimd.Uint32x4) archsimd.Uint64x2 { return x.MulWidenLo(y) }
`,
			} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0600); err != nil {
					t.Fatal(err)
				}
			}
			conf := NewDefaultConf(ModeGen)
			conf.Goos, conf.Goarch, conf.GOEXPERIMENT = target.os, target.arch, "simd"
			if target.arch == "amd64" {
				conf.GOAMD64 = "v1"
			}
			pkgs, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: dir})
			if err != nil {
				t.Fatal(err)
			}
			archPkgs, err := Build(Invocation{Args: []string{"simd/archsimd"}, Config: conf, Dir: dir})
			if err != nil {
				t.Fatal(err)
			}
			pkgs = append(pkgs, archPkgs...)
			mainIndex := -1
			for i, pkg := range pkgs {
				defer pkg.LPkg.Prog.Dispose()
				if !pkg.LPkg.Module().NamedFunction("main.average").IsNil() {
					mainIndex = i
				}
				if err := llvm.VerifyModule(pkg.LPkg.Module(), llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
				if pkg.PkgPath == "simd/archsimd" && target.arch != "wasm" {
					name := "simd/archsimd.Uint64x2.carrylessMultiply"
					if target.arch == "arm64" {
						name += "WidenLo"
					}
					body := pkg.LPkg.Module().NamedFunction(name).String()
					if !strings.Contains(body, "@llvm.clmul.i128") {
						t.Fatalf("carryless multiplication lost its full-width portable operation:\n%s", body)
					}
				}
			}
			if mainIndex < 0 {
				t.Fatal("main package was not compiled")
			}
			pkg := pkgs[mainIndex].LPkg
			mod := pkg.Module()
			fma := "@llvm.fma.v4f32"
			if target.os == "windows" {
				fma = "SIMDFMA32"
			}
			if target.arch != "wasm" && !strings.Contains(mod.NamedFunction("main.madd").String(), fma) {
				t.Fatal("native MulAdd lost fused rounding")
			}
			if target.arch == "amd64" && !strings.Contains(mod.NamedFunction("main.masked").String(), "@llvm.masked.store") {
				t.Fatal("masked store lost inactive-lane memory protection")
			}
			if target.arch == "wasm" && !strings.Contains(mod.NamedFunction("main.rotate").String(), "@llvm.fshl.v4i32") {
				t.Fatal("rotate lost its modulo-count operation")
			}
			for _, compiled := range pkgs {
				pkg := compiled.LPkg
				mod := pkg.Module()
				if compiled.PkgPath == "simd/archsimd" {
					if target.arch == "wasm" {
						continue
					}
					// Keep just the private polynomial primitive for baseline
					// code generation; the main module covers direct operations.
					for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
						if fn.BasicBlocksCount() != 0 && !strings.HasPrefix(fn.Name(), "simd/archsimd.Uint64x2.carrylessMultiply") {
							fn.SetLinkage(llvm.InternalLinkage)
						}
					}
					opts := llvm.NewPassBuilderOptions()
					err := mod.RunPasses("globaldce", pkg.Prog.TargetMachine(), opts)
					opts.Dispose()
					if err != nil {
						t.Fatal(err)
					}
				}
				mod.SetTarget(pkg.Prog.Target().Spec().Triple)
				mod.SetDataLayout(pkg.Prog.DataLayout())
				for _, level := range []string{"O0", "O2"} {
					if level == "O2" {
						opts := llvm.NewPassBuilderOptions()
						opts.SetVerifyEach(true)
						err := mod.RunPasses("default<O2>", pkg.Prog.TargetMachine(), opts)
						opts.Dispose()
						if err != nil {
							t.Fatal(err)
						}
					}
					asm, err := pkg.Prog.TargetMachine().EmitToMemoryBuffer(mod, llvm.AssemblyFile)
					if err != nil {
						t.Fatal(err)
					}
					if target.os == "linux" {
						assertSIMDBaselineAssembly(t, mod, string(asm.Bytes()))
					}
					asm.Dispose()
				}
			}
		})
	}
}
