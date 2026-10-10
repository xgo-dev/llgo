//go:build !llgo

package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

const simdFMVSource = `package main
import "simd/archsimd"
import "fmvtest/kernel"
import _ "unsafe"

//go:noinline
func local(x, y archsimd.Float32x4) archsimd.Float32x4 { return x.Add(y).Mul(x) }

//go:linkname assembly fmv.externalAssembly
func assembly(x archsimd.Float32x4) archsimd.Float32x4

//go:noinline
func guarded(x, y archsimd.Float32x4) archsimd.Float32x4 {
 if archsimd.X86.AVX2() { return assembly(local(x, kernel.Compute(x, y))) }
 return x.Sub(y)
}
func main() {}
`

func simdFMVTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "kernel"), 0700); err != nil {
		t.Fatal(err)
	}
	for name, source := range map[string]string{
		"go.mod":  "module fmvtest\n\ngo 1.27\n",
		"main.go": simdFMVSource,
		"kernel/kernel.go": `package kernel
import "simd/archsimd"
//go:noinline
func Compute(x, y archsimd.Float32x4) archsimd.Float32x4 { return x.Add(y) }
`,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestSIMDFMVLLVM(t *testing.T) {
	for _, goos := range []string{"linux", "darwin", "windows"} {
		for _, level := range []string{"O0", "O2"} {
			t.Run(goos+"/v1/"+level, func(t *testing.T) {
				testSIMDFMVLLVM(t, goos, "v1", level)
			})
		}
	}
	t.Run("linux/v3/O2", func(t *testing.T) {
		testSIMDFMVLLVM(t, "linux", "v3", "O2")
	})
}

func testSIMDFMVLLVM(t *testing.T, goos, baseline, level string) {
	conf := NewDefaultConf(ModeGen)
	conf.Goos, conf.Goarch, conf.GOAMD64, conf.GOEXPERIMENT = goos, "amd64", baseline, "simd"
	conf.LinkOptions.DWARF = DWARFPreserve
	dir := simdFMVTestDir(t)
	pkgs, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	defer pkgs[0].LPkg.Prog.Dispose()
	mod := pkgs[0].LPkg.Module()
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	guard := mod.NamedFunction("main.guarded").String()
	fast := guard
	suffix := ""
	if baseline == "v3" || baseline == "v4" {
		if strings.Contains(mod.String(), "__llgo_fmv") {
			t.Fatalf("baseline feature generated redundant versions:\n%s", mod.String())
		}
	} else {
		suffix = ".__llgo_fmv_avx2"
		if !strings.Contains(mod.String(), `linkageName: "main.guarded.__llgo_fmv_avx2"`) {
			t.Fatal("specialized debug subprogram lost its linker identity")
		}
		baselineBody := mod.NamedFunction("main.guarded.__llgo_fmv_baseline").String()
		if goos == "windows" {
			for _, name := range []string{"main.guarded", "main.guarded.__llgo_fmv_resolve"} {
				fn := mod.NamedFunction(name)
				if fn.Param(0).Type().TypeKind() != llvm.PointerTypeKind || strings.Contains(fn.String(), "alloca") || strings.Contains(fn.String(), "load <4 x float>") {
					t.Fatalf("Win64 tail transfer must forward caller-owned vector storage:\n%s", fn.String())
				}
			}
		}
		fast = mod.NamedFunction("main.guarded" + suffix).String()
		if !strings.Contains(guard, "musttail call") || !strings.Contains(guard, "load atomic ptr") ||
			strings.Contains(guard, "X86Features.AVX2") || strings.Contains(baselineBody, "X86Features.AVX2") || !strings.Contains(baselineBody, "fsub") {
			t.Fatalf("invalid dispatch or baseline:\n%s\n%s", guard, baselineBody)
		}
	}
	if strings.Contains(fast, "X86Features.AVX2") || strings.Contains(fast, "fsub") ||
		!strings.Contains(fast, "main.local"+suffix) || !strings.Contains(fast, "fmvtest/kernel.Compute"+suffix) {
		t.Fatalf("invalid specialization:\n%s", fast)
	}
	if !strings.Contains(fast, "@fmv.externalAssembly(") || !mod.NamedFunction("fmv.externalAssembly.__llgo_fmv_avx2").IsNil() {
		t.Fatal("bodyless assembly declaration must keep its baseline entry")
	}
	kernels, err := Build(Invocation{Args: []string{"./kernel"}, Config: conf, Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	defer kernels[0].LPkg.Prog.Dispose()
	kernel := kernels[0].LPkg.Module().NamedFunction("fmvtest/kernel.Compute" + suffix)
	if kernel.IsNil() || kernel.BasicBlocksCount() == 0 {
		t.Fatal("cross-package specialized definition missing")
	}
	prog := pkgs[0].LPkg.Prog
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
	if goos == "linux" && baseline == "v1" {
		assertSIMDBaselineAssembly(t, string(asm.Bytes()))
	}
	if !strings.Contains(string(asm.Bytes()), "vaddps") {
		t.Fatal("AVX2 version did not select AVX instructions")
	}
}
