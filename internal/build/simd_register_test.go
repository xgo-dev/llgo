//go:build !llgo

package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestSIMD512RegisterCalls(t *testing.T) {
	for _, goos := range []string{"linux", "windows"} {
		t.Run(goos, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.Mkdir(filepath.Join(dir, "kernel"), 0700); err != nil {
				t.Fatal(err)
			}
			for name, source := range map[string]string{
				"go.mod": "module widecalls\n\ngo 1.27\n",
				"kernel/kernel.go": `package kernel
import "simd/archsimd"
//go:noinline
func Twice(x archsimd.Uint32x16) archsimd.Uint32x16 { return x.Add(x) }`,
				"main.go": `package main
import ("simd/archsimd"; "widecalls/kernel")
//go:noinline
func twice(x archsimd.Uint32x16) archsimd.Uint32x16 { return x.Add(x) }
//go:noinline
func chain(p *[16]uint32) {
 if archsimd.X86.AVX512() { z:=archsimd.LoadUint32x16Array(p); kernel.Twice(twice(twice(z))).StoreArray(p) }
}
func main() { var p [16]uint32; chain(&p) }`,
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
			pkg := pkgs[0].LPkg
			mod := pkg.Module()
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			fn := mod.NamedFunction("main.chain.__llgo_fmv_guard_avx512")
			if fn.IsNil() || !strings.Contains(fn.String(), "kernel.Twice.__llgo_fmv_reg512_go") || strings.Count(fn.String(), ".__llgo_fmv_reg512(") != 2 {
				t.Fatalf("missing local or cross-package ZMM calls: %v", fn)
			}
			mod.SetTarget(pkg.Prog.Target().Spec().Triple)
			mod.SetDataLayout(pkg.Prog.DataLayout())
			opts := llvm.NewPassBuilderOptions()
			defer opts.Dispose()
			if err := mod.RunPasses("default<O2>", pkg.Prog.TargetMachine(), opts); err != nil {
				t.Fatal(err)
			}
			buf, err := pkg.Prog.TargetMachine().EmitToMemoryBuffer(mod, llvm.AssemblyFile)
			if err != nil {
				t.Fatal(err)
			}
			defer buf.Dispose()
			asm := string(buf.Bytes())
			if !strings.Contains(asm, "%zmm0") || !strings.Contains(asm, "vpaddd") {
				t.Fatal("missing ZMM integer addition")
			}
		})
	}
}
