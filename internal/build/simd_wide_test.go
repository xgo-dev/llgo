//go:build !llgo

package build

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/lto"
	llssa "github.com/xgo-dev/llgo/ssa"
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
 if archsimd.X86.AVX2() { return arithmetic(x, y) }
 return x.Sub(y)
}
func indirect(f func(archsimd.Float32x8, archsimd.Float32x8) archsimd.Float32x8, x, y archsimd.Float32x8) archsimd.Float32x8 { return f(x, y) }
//go:noinline
func arithmetic512(x, y archsimd.Float32x16) archsimd.Float32x16 { return x.Add(y) }
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
				if ft.ReturnType().TypeKind() != llvm.VectorTypeKind || ft.ReturnType().VectorSize() != 8 {
					t.Fatalf("missing wide vector return in %s:\n%s", name, fn)
				}
				params := ft.ParamTypes()
				if len(params) < 2 || params[len(params)-1] != ft.ReturnType() || params[len(params)-2] != ft.ReturnType() {
					t.Fatalf("missing wide vector arguments in %s:\n%s", name, fn)
				}
				if goos == "windows" && fn.FunctionCallConv() != llvm.CallConv(80) {
					t.Fatalf("expected vectorcall for %s", name)
				}
			}
			for _, name := range []string{"main.guarded", "main.guarded.__llgo_fmv_resolve"} {
				body := mod.NamedFunction(name).String()
				if strings.Contains(body, "alloca") || strings.Contains(body, "load volatile") || !strings.Contains(body, "musttail call") {
					t.Fatalf("dispatcher must tail-forward the vector arguments:\n%s", body)
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
					if strings.Contains(body, "load volatile") || strings.Contains(body, "store volatile") {
						t.Fatalf("%s introduced a memory carrier:\n%s", level, body)
					}
				}
				asm, err := pkg.Prog.TargetMachine().EmitToMemoryBuffer(mod, llvm.AssemblyFile)
				if err != nil {
					t.Fatal(err)
				}
				if goos == "linux" {
					assertSIMDBaselineAssembly(t, mod, string(asm.Bytes()))
				}
				for _, reg := range []string{"ymm", "zmm"} {
					if !regexp.MustCompile("vaddps[^\\n]*%" + reg).Match(asm.Bytes()) {
						t.Fatalf("%s missing vaddps on %s registers", level, reg)
					}
				}
				asm.Dispose()
			}
		})
	}
}

// LTO's target machine can default to x86-64 even when the frontend used v4.
func TestWideBaselineInLTOBitcode(t *testing.T) {
	llssa.Initialize(llssa.InitAll)
	generic := llssa.NewProgram(&llssa.Target{GOOS: "linux", GOARCH: "amd64", GOAMD64: "v1"})
	defer generic.Dispose()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module wide\n\ngo 1.27\n"), 0600); err != nil {
		t.Fatal(err)
	}
	const source = `package wide
import "simd/archsimd"
func Add(x, y archsimd.Float32x16) archsimd.Float32x16 { return x.Add(y) }
`
	if err := os.WriteFile(filepath.Join(dir, "wide.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	conf := NewDefaultConf(ModeGen)
	conf.Goos, conf.Goarch, conf.GOAMD64, conf.GOEXPERIMENT = "linux", "amd64", "v4", "simd"
	pkgs, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	pkg := pkgs[0].LPkg
	defer pkg.Prog.Dispose()
	// The textual IR handed to Clang must carry the same baseline as bitcode.
	add := pkg.Module().NamedFunction("wide.Add")
	if add.IsNil() {
		t.Fatal("missing wide.Add")
	}
	cpu := add.GetStringAttributeAtIndex(-1, "target-cpu")
	if cpu.IsNil() || cpu.GetStringValue() != "x86-64-v4" {
		t.Fatalf("missing v4 baseline in package IR:\n%s", add)
	}
	for _, mode := range []lto.Mode{lto.Thin, lto.Full} {
		t.Run(mode.String(), func(t *testing.T) {
			conf.LTO = mode
			buf, _, err := emitObjectToMemoryBuffer(conf, pkg.Prog, pkg.Module())
			if err != nil {
				t.Fatal(err)
			}
			defer buf.Dispose()
			path := filepath.Join(t.TempDir(), "wide.bc")
			if err := os.WriteFile(path, buf.Bytes(), 0600); err != nil {
				t.Fatal(err)
			}
			linked, err := pkg.Module().Context().ParseBitcodeFile(path)
			if err != nil {
				t.Fatal(err)
			}
			defer linked.Dispose()
			asm, err := generic.TargetMachine().EmitToMemoryBuffer(linked, llvm.AssemblyFile)
			if err != nil {
				t.Fatal(err)
			}
			defer asm.Dispose()
			if !regexp.MustCompile(`vaddps[^\n]*%zmm`).Match(asm.Bytes()) {
				t.Fatalf("lost the v4 register ABI:\n%s", asm.Bytes())
			}
		})
	}
}
