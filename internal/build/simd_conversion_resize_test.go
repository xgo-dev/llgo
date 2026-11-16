//go:build !llgo

package build

import (
	"github.com/xgo-dev/llvm"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSIMDConversionResizeLLVM(t *testing.T) {
	for _, target := range []struct{ os, arch string }{{"linux", "amd64"}, {"windows", "amd64"}, {"darwin", "arm64"}, {"wasip1", "wasm"}} {
		t.Run(target.os+"/"+target.arch, func(t *testing.T) {
			source := "package main\nimport \"simd/archsimd\"\nfunc main() {}\n"
			expected := map[string]string{}
			if target.arch == "amd64" {
				source += `
func widenFloat(x archsimd.Float32x4) archsimd.Float64x4 {return x.ConvertToFloat64()}
func narrowFloat(x archsimd.Float64x4) archsimd.Float32x4 {return x.ConvertToFloat32()}
func floatInt(x archsimd.Float64x4) archsimd.Int32x4 {return x.ConvertToInt32()}
func widenInt(x archsimd.Int8x16) archsimd.Int16x16 {return x.ExtendToInt16()}
func widenLo(x archsimd.Uint8x16) archsimd.Uint64x4 {return x.ExtendLo4ToUint64()}
func saturate(x archsimd.Int64x4) archsimd.Int8x16 {return x.SaturateToInt8()}
func truncate(x archsimd.Int64x4) archsimd.Int8x16 {return x.TruncToInt8()}
func grouped(x,y archsimd.Int32x8) archsimd.Uint16x16 {return x.SaturateToUint16ConcatGrouped(y)}
func ceil(x archsimd.Float32x8) archsimd.Float32x8 {return x.Ceil()}
func floor(x archsimd.Float64x4) archsimd.Float64x4 {return x.Floor()}
func trunc(x archsimd.Float32x8) archsimd.Float32x8 {return x.Trunc()}
func round(x archsimd.Float64x4) archsimd.Float64x4 {return x.Round()}
`
				expected = map[string]string{"widenFloat": "fpext", "narrowFloat": "fptrunc", "floatInt": "llvm.fptosi.sat", "widenInt": "sext", "widenLo": "zext", "saturate": "select", "truncate": "trunc <", "grouped": "shufflevector", "ceil": "llvm.ceil", "floor": "llvm.floor", "trunc": "llvm.trunc", "round": "select <4 x i1>"}
			} else {
				if target.arch == "arm64" {
					expected["widenFloat"] = "fpext"
					source += `func widenFloat(x archsimd.Float32x4) archsimd.Float64x2 {return x.ConvertLo2ToFloat64()}
func truncate(x archsimd.Int16x8) archsimd.Int8x16 {return x.TruncToInt8()}
`
					expected["truncate"] = "trunc <"
				} else {
					source += `func signed(x archsimd.Int32x4) archsimd.Float64x2 {return x.ConvertLo2ToFloat64()}
func unsigned(x archsimd.Uint32x4) archsimd.Float64x2 {return x.ConvertLo2ToFloat64()}
`
					expected["signed"], expected["unsigned"] = "sitofp", "uitofp"
				}
			}
			dir := t.TempDir()
			for name, body := range map[string]string{"go.mod": "module conversions\n\ngo 1.27\n", "main.go": source} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0600); err != nil {
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
			for _, pkg := range pkgs {
				defer pkg.LPkg.Prog.Dispose()
				mod := pkg.LPkg.Module()
				if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
				for name, op := range expected {
					fn := mod.NamedFunction("main." + name)
					if fn.IsNil() {
						t.Fatalf("missing %s", name)
					}
					body := fn.String()
					if !strings.Contains(body, op) || strings.Contains(body, "PanicSIMDUnimplemented") {
						t.Fatalf("missing %s in %s:\n%s", op, name, body)
					}
				}
			}
			simdCodegenVariants(t, []string{"O0", "O2"}, pkgs, func(t *testing.T, asm string) {
				if target.os == "linux" {
					assertSIMDBaselineAssembly(t, asm)
				}
			})
		})
	}
}
