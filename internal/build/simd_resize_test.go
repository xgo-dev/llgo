//go:build !llgo

package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestSIMDIntegerResizeLLVM(t *testing.T) {
	for _, target := range []struct{ os, arch string }{
		{"linux", "amd64"}, {"windows", "amd64"}, {"darwin", "arm64"}, {"wasip1", "wasm"},
	} {
		t.Run(target.os+"/"+target.arch, func(t *testing.T) {
			dir := t.TempDir()
			for name, source := range map[string]string{
				"go.mod": "module simdresize\n\ngo 1.27\n",
				"main.go": `package main
import "simd/archsimd"
func extendSigned(x archsimd.Int8x16) archsimd.Int16x8 { return x.ExtendLo8ToInt16() }
func extendUnsigned(x archsimd.Uint32x4) archsimd.Uint64x2 { return x.ExtendLo2ToUint64() }
func main() {}
`,
				"narrow.go": `//go:build amd64 || arm64
package main
import "simd/archsimd"
func narrowSigned(x archsimd.Int16x8) archsimd.Int8x16 { return x.SaturateToInt8() }
func narrowUnsigned(x archsimd.Uint64x2) archsimd.Uint32x4 { return x.SaturateToUint32() }
`,
				"resize_amd64.go": `package main
import "simd/archsimd"
func concat(x,y archsimd.Int32x4) archsimd.Uint16x8 { return x.SaturateToUint16Concat(y) }
func extendQuarter(x archsimd.Uint8x16) archsimd.Uint64x2 { return x.ExtendLo2ToUint64() }
func narrowQuarter(x archsimd.Int64x2) archsimd.Int8x16 { return x.SaturateToInt8() }
`,
				"resize_arm64.go": `package main
import "simd/archsimd"
func narrowMixed(x archsimd.Int16x8) archsimd.Uint8x16 { return x.SaturateToUint8() }
`,
				"resize_wasm.go": `package main
import "simd/archsimd"
func extendHighSigned(x archsimd.Int16x8) archsimd.Int32x4 { return x.ExtendHi4ToInt32() }
func extendHighUnsigned(x archsimd.Uint16x8) archsimd.Uint32x4 { return x.ExtendHi4ToUint32() }
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
			pkg := pkgs[0].LPkg
			defer pkg.Prog.Dispose()
			mod := pkg.Module()
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			checks := map[string][]string{
				"extendSigned":   {"shufflevector <16 x i8>", "sext <8 x i8>"},
				"extendUnsigned": {"shufflevector <4 x i32>", "zext <2 x i32>"},
			}
			if target.arch != "wasm" {
				checks["narrowSigned"] = []string{"icmp slt <8 x i16>", "icmp sgt <8 x i16>", "trunc <8 x i16>", "shufflevector <8 x i8>"}
				checks["narrowUnsigned"] = []string{"icmp ugt <2 x i64>", "trunc <2 x i64>"}
			}
			switch target.arch {
			case "amd64":
				checks["concat"] = []string{"icmp slt <4 x i32>", "icmp sgt <4 x i32>", "shufflevector <4 x i16>"}
				checks["extendQuarter"] = []string{"zext <2 x i8>"}
				checks["narrowQuarter"] = []string{"trunc <2 x i64>", "to <2 x i8>", "shufflevector <2 x i8>"}
			case "arm64":
				checks["narrowMixed"] = []string{"icmp slt <8 x i16>", "icmp sgt <8 x i16>", "trunc <8 x i16>"}
			case "wasm":
				checks["extendHighSigned"] = []string{"<i32 4, i32 5, i32 6, i32 7>", "sext <4 x i16>"}
				checks["extendHighUnsigned"] = []string{"<i32 4, i32 5, i32 6, i32 7>", "zext <4 x i16>"}
			}
			for name, wants := range checks {
				fn := mod.NamedFunction("main." + name)
				if fn.IsNil() {
					t.Fatalf("missing function %s", name)
				}
				for _, want := range wants {
					if !strings.Contains(fn.String(), want) {
						t.Fatalf("%s missing %s:\n%s", name, want, fn.String())
					}
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
					assertSIMDBaselineAssembly(t, string(asm.Bytes()))
				}
				asm.Dispose()
			}
		})
	}
}
