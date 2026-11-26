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
 if archsimd.X86.AVX2() { return assembly(local(x.Add(y), kernel.Compute(x, y))) }
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
		t.Run(goos+"/v1", func(t *testing.T) {
			testSIMDFMVLLVM(t, goos, "v1", "O0", "O2")
		})
	}
	t.Run("linux/v3", func(t *testing.T) {
		testSIMDFMVLLVM(t, "linux", "v3", "O2")
	})
}

// Independent guards decide how many times the same baseline kernel runs.
func TestSIMDGuardedNarrowHelperSharesCode(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module fmvtest\n\ngo 1.27\n")
	var src strings.Builder
	src.WriteString(`package main
import "simd/archsimd"
//go:noinline
func kernel(p *[4]int32) {
 x := archsimd.LoadInt32x4Array(p)
 x.Add(x).StoreArray(p)
}
//go:noinline
func root(p *[4]int32) {
 if !archsimd.X86.AVX() { return }
`)
	guards := []string{"FMA", "SHA", "AVXAES", "VAES", "AVXVNNI", "AVX512", "AVX512VBMI2", "AVX512GFNI", "AVX512VNNI", "AVX512BITALG"}
	for _, guard := range guards {
		src.WriteString("if archsimd.X86." + guard + "() { kernel(p) }\n")
	}
	src.WriteString("}\nfunc main() { var p [4]int32; root(&p) }\n")
	writeFile(t, filepath.Join(dir, "main.go"), src.String())
	conf := NewDefaultConf(ModeGen)
	conf.Goos, conf.Goarch, conf.GOAMD64, conf.GOEXPERIMENT = "linux", "amd64", "v1", "simd"
	pkgs, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	defer pkgs[0].LPkg.Prog.Dispose()
	mod := pkgs[0].LPkg.Module()
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		if strings.HasPrefix(fn.Name(), "main.kernel.__llgo_fmv_") || strings.HasPrefix(fn.Name(), "main.root.__llgo_fmv_") {
			t.Fatalf("unexpected specialization %s", fn.Name())
		}
	}
	root := mod.NamedFunction("main.root").String()
	if got := strings.Count(root, "call void @main.kernel("); got != len(guards) {
		t.Fatalf("expected %d conditional calls to the shared kernel, got %d:\n%s", len(guards), got, root)
	}
}

func testSIMDFMVLLVM(t *testing.T, goos, baseline string, levels ...string) {
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
		suffix = ".__llgo_fmv_guard_avx2"
		if !strings.Contains(mod.String(), `linkageName: "main.guarded.__llgo_fmv_guard_avx2"`) {
			t.Fatal("specialized debug subprogram lost its linker identity")
		}
		baselineBody := mod.NamedFunction("main.guarded.__llgo_fmv_guard_base").String()
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
			strings.Contains(guard, "X86Features.AVX2") || !strings.Contains(baselineBody, "fsub") {
			t.Fatalf("invalid dispatch or baseline:\n%s\n%s", guard, baselineBody)
		}
	}
	if strings.Contains(fast, "X86Features.AVX2") || strings.Contains(fast, "fsub") ||
		!strings.Contains(fast, "@main.local(") || !strings.Contains(fast, "fmvtest/kernel.Compute\"(") {
		t.Fatalf("invalid specialization:\n%s", fast)
	}
	if !strings.Contains(fast, "@fmv.externalAssembly(") || !mod.NamedFunction("fmv.externalAssembly.__llgo_fmv_guard_avx2").IsNil() {
		t.Fatal("bodyless assembly declaration must keep its baseline entry")
	}
	kernels, err := Build(Invocation{Args: []string{"./kernel"}, Config: conf, Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	defer kernels[0].LPkg.Prog.Dispose()
	kernel := kernels[0].LPkg.Module().NamedFunction("fmvtest/kernel.Compute")
	if kernel.IsNil() || kernel.BasicBlocksCount() == 0 {
		t.Fatal("ordinary cross-package definition missing")
	}
	simdCodegenVariants(t, levels, pkgs, func(t *testing.T, mod llvm.Module, asm string) {
		if goos == "linux" && baseline == "v1" {
			assertSIMDBaselineAssembly(t, mod, asm)
		}
		if !strings.Contains(asm, "vaddps") {
			t.Fatal("AVX2 version did not select AVX instructions")
		}
	})
}

// The level selects only the LLVM pass pipeline. Generate and check the shared
// input IR once, then read an independent bitcode copy for each pipeline so
// code generation cannot mutate the input of the next case.
func simdCodegenVariants(t *testing.T, levels []string, pkgs []*aPackage, check func(*testing.T, llvm.Module, string)) {
	t.Helper()
	paths := make([]string, len(pkgs))
	for i, pkg := range pkgs {
		prog, mod := pkg.LPkg.Prog, pkg.LPkg.Module()
		mod.SetDataLayout(prog.DataLayout())
		mod.SetTarget(prog.Target().Spec().Triple)
		paths[i] = filepath.Join(t.TempDir(), "input.bc")
		bitcode := llvm.WriteBitcodeToMemoryBuffer(mod)
		err := os.WriteFile(paths[i], bitcode.Bytes(), 0600)
		bitcode.Dispose()
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			for i, pkg := range pkgs {
				ctx := llvm.NewContext()
				defer ctx.Dispose()
				mod, err := ctx.ParseBitcodeFile(paths[i])
				if err != nil {
					t.Fatal(err)
				}
				defer mod.Dispose()
				machine := pkg.LPkg.Prog.TargetMachine()
				if level == "O2" {
					opts := llvm.NewPassBuilderOptions()
					defer opts.Dispose()
					opts.SetVerifyEach(true)
					if err := mod.RunPasses("default<O2>", machine, opts); err != nil {
						t.Fatal(err)
					}
				}
				asm, err := machine.EmitToMemoryBuffer(mod, llvm.AssemblyFile)
				if err != nil {
					t.Fatal(err)
				}
				defer asm.Dispose()
				check(t, mod, string(asm.Bytes()))
			}
		})
	}
}

// Exercise the real archsimd lowerings as well as the version planner. Adding
// an independent extension must not duplicate the kernel for each subset.
func TestSIMDIndependentInstructionFamilies(t *testing.T) {
	dir, err := filepath.Abs("testdata/fmv-independent")
	if err != nil {
		t.Fatal(err)
	}
	conf := NewDefaultConf(ModeGen)
	conf.Goos, conf.Goarch, conf.GOAMD64, conf.GOEXPERIMENT = "linux", "amd64", "v1", "simd"
	pkgs, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	defer pkgs[0].LPkg.Prog.Dispose()
	mod := pkgs[0].LPkg.Module()
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	var bodies int
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		if strings.HasPrefix(fn.Name(), "main.kernel.__llgo_fmv_guard_") {
			bodies++
		}
	}
	// Of the common ISA levels, only FMA is used behind its own guard.
	// At most three register widths can pair with its two states.
	if bodies > 6 || mod.NamedFunction("main.kernel.__llgo_fmv_guard_fma").IsNil() ||
		mod.NamedFunction("main.kernel.__llgo_fmv_guard_zmm").IsNil() {
		t.Fatalf("expected FMA and ZMM implementations within six bodies, got %d", bodies)
	}
}
