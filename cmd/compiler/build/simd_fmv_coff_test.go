//go:build !llgo

package build

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/xgo-dev/llvm"
)

// Compile the same generic instantiation in two packages and link the actual
// COFF objects. LLVM linkage alone does not deduplicate definitions on Windows.
func TestSIMDFMVWindowsGenericLink(t *testing.T) {
	linker, err := exec.LookPath("lld-link")
	if err != nil {
		t.Fatal("lld-link is required for the Windows FMV link regression")
	}
	dir := t.TempDir()
	for name, source := range map[string]string{
		"go.mod": "module fmvcomdat\n\ngo 1.27\n",
		"shared/shared.go": `package shared
import "simd/archsimd"
//go:noinline
func Echo[T ~int](x archsimd.Float32x4, n T) archsimd.Float32x4 {
 if archsimd.X86.AVX2() { return x.Add(x) }
 return x
}`,
		"a/a.go": `package a
import "simd/archsimd"
import "fmvcomdat/shared"
//go:noinline
func A(x archsimd.Float32x4) archsimd.Float32x4 { return shared.Echo(x, 1) }`,
		"b/b.go": `package b
import "simd/archsimd"
import "fmvcomdat/shared"
//go:noinline
func B(x archsimd.Float32x4) archsimd.Float32x4 { return shared.Echo(x, 1) }`,
	} {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, level := range []string{"O0", "O2"} {
		t.Run(level, func(t *testing.T) {
			outDir := t.TempDir()
			var objects []string
			for _, name := range []string{"a", "b"} {
				conf := NewDefaultConf(ModeGen)
				conf.Goos, conf.Goarch, conf.GOAMD64, conf.GOEXPERIMENT = "windows", "amd64", "v1", "simd"
				pkgs, err := Build(Invocation{Args: []string{"./" + name}, Config: conf, Dir: dir})
				if err != nil {
					t.Fatal(err)
				}
				prog := pkgs[0].LPkg.Prog
				defer prog.Dispose()
				mod := pkgs[0].LPkg.Module()
				variant := mod.NamedFunction("fmvcomdat/shared.Echo[int].__llgo_fmv_avx2")
				if variant.IsNil() || variant.IsDeclaration() {
					t.Fatal("generic AVX2 definition missing from package", name)
				}
				writeObject := func(name string, mod llvm.Module) {
					t.Helper()
					mod.SetTarget(prog.Target().Spec().Triple)
					mod.SetDataLayout(prog.DataLayout())
					if level == "O2" {
						opts := llvm.NewPassBuilderOptions()
						defer opts.Dispose()
						if err := mod.RunPasses("default<O2>", prog.TargetMachine(), opts); err != nil {
							t.Fatal(err)
						}
					}
					buf, err := prog.TargetMachine().EmitToMemoryBuffer(mod, llvm.ObjectFile)
					if err != nil {
						t.Fatal(err)
					}
					defer buf.Dispose()
					path := filepath.Join(outDir, name+".obj")
					if err := os.WriteFile(path, buf.Bytes(), 0600); err != nil {
						t.Fatal(err)
					}
					objects = append(objects, path)
				}
				writeObject(name, mod)
				if name == "a" {
					// Supply the fixture's runtime dependencies without linking the
					// whole runtime. Unresolved symbols remain fatal to this test.
					ctx := mod.Context()
					support := ctx.NewModule("support")
					defer support.Dispose()
					mask := llvm.AddGlobal(support, ctx.Int64Type(), "github.com/xgo-dev/llgo/runtime/internal/runtime.CPUFeatures")
					mask.SetInitializer(llvm.ConstNull(ctx.Int64Type()))
					b := ctx.NewBuilder()
					defer b.Dispose()
					for _, init := range []string{"simd/archsimd.init", "fmvcomdat/shared.init"} {
						fn := llvm.AddFunction(support, init, llvm.FunctionType(ctx.VoidType(), nil, false))
						b.SetInsertPointAtEnd(ctx.AddBasicBlock(fn, "entry"))
						b.CreateRetVoid()
					}
					writeObject("support", support)
				}
			}
			args := []string{"/dll", "/noentry", "/nodefaultlib", "/opt:noref", "/out:" + filepath.Join(outDir, "generic.dll")}
			if out, err := exec.Command(linker, append(args, objects...)...).CombinedOutput(); err != nil {
				t.Fatalf("link generic SIMD instantiations: %v\n%s", err, out)
			}
		})
	}
}
