//go:build !llgo

package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestSIMD256LLVM(t *testing.T) {
	for _, goos := range []string{"linux", "windows"} {
		t.Run(goos, func(t *testing.T) {
			dir := t.TempDir()
			for name, source := range map[string]string{
				"go.mod": "module simdwide\n\ngo 1.27\n",
				"main.go": `package main
import "simd/archsimd"
//go:noinline
func arithmetic(x, y archsimd.Float32x8) archsimd.Float32x8 { return x.Add(y).Mul(x) }
//go:noinline
func guarded(x, y archsimd.Float32x8) archsimd.Float32x8 {
 if archsimd.X86.AVX2() { z := arithmetic(x, y); return arithmetic(z, z) }
 return x.Sub(y)
}
func indirect(f func(archsimd.Float32x8, archsimd.Float32x8) archsimd.Float32x8, x, y archsimd.Float32x8) archsimd.Float32x8 { return f(x, y) }
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
			fn := mod.NamedFunction("main.arithmetic")
			if fn.IsNil() || !strings.Contains(fn.String(), "fadd <8 x float>") || !strings.Contains(fn.String(), "fmul <8 x float>") {
				t.Fatalf("missing 256-bit computation:\n%s", fn.String())
			}
			for _, name := range []string{"main.arithmetic", "main.guarded", "main.indirect", "main.arithmetic.__llgo_fmv_avx2", "main.guarded.__llgo_fmv_avx2"} {
				fn := mod.NamedFunction(name)
				if fn.IsNil() {
					t.Fatalf("missing %s", name)
				}
				ft := fn.GlobalValueType()
				if ft.ReturnType().TypeKind() != llvm.VoidTypeKind || ft.ParamTypes()[0].TypeKind() != llvm.PointerTypeKind {
					t.Fatalf("wide return is not an indirect carrier:\n%s", fn.String())
				}
				for _, typ := range ft.ParamTypes() {
					if typ.TypeKind() == llvm.VectorTypeKind {
						t.Fatalf("vector call carrier in %s", name)
					}
				}
			}
			regName := "main.arithmetic.__llgo_fmv_avx2.__llgo_fmv_reg256"
			reg := mod.NamedFunction(regName)
			if reg.IsNil() || reg.GlobalValueType().ReturnType().TypeKind() != llvm.VectorTypeKind || !strings.Contains(mod.NamedFunction("main.guarded.__llgo_fmv_avx2").String(), regName) {
				t.Fatalf("missing private register call: %s", mod.NamedFunction("main.guarded.__llgo_fmv_avx2").String())
			}
			for _, name := range []string{"main.guarded", "main.guarded.__llgo_fmv_baseline"} {
				if strings.Contains(mod.NamedFunction(name).String(), regName) {
					t.Fatalf("unproven call from %s", name)
				}
			}
			if goos == "windows" && (reg.FunctionCallConv() != llvm.CallConv(80) || !strings.Contains(mod.NamedFunction("main.guarded.__llgo_fmv_avx2").String(), "x86_vectorcallcc")) {
				t.Fatal("expected vectorcall on both sides")
			}
			if strings.Contains(reg.String(), "load volatile") || strings.Contains(reg.String(), "store volatile") {
				t.Fatalf("register entry still transfers values through pointers: %s", reg.String())
			}
			for _, name := range []string{"main.guarded", "main.guarded.__llgo_fmv_resolve"} {
				body := mod.NamedFunction(name).String()
				if strings.Contains(body, "alloca") || strings.Contains(body, "load volatile") || !strings.Contains(body, "musttail call") {
					t.Fatalf("dispatcher must forward caller-owned storage:\n%s", body)
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
				for _, name := range []string{"main.arithmetic", "main.arithmetic.__llgo_fmv_avx2"} {
					body := mod.NamedFunction(name).String()
					if !strings.Contains(body, "load volatile <8 x float>") || !strings.Contains(body, "store volatile <8 x float>") {
						t.Fatalf("%s changed the wide memory boundary:\n%s", level, body)
					}
				}
				asm, err := pkg.Prog.TargetMachine().EmitToMemoryBuffer(mod, llvm.AssemblyFile)
				if err != nil {
					t.Fatal(err)
				}
				if goos == "linux" {
					assertSIMDBaselineAssembly(t, string(asm.Bytes()))
				}
				if !strings.Contains(string(asm.Bytes()), "vaddps") || !strings.Contains(string(asm.Bytes()), "%ymm") {
					t.Fatal("AVX2 body did not select 256-bit computation")
				}
				asm.Dispose()
			}
		})
	}
}
