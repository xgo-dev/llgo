//go:build !llgo

package cl

import (
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestFloatToIntegerConversionMode(t *testing.T) {
	const baseSrc = `package floatconvert
func F32(x float32) uint32 { return uint32(x) }
func F64(x float64) uint32 { return uint32(x) }
`
	const saturatingSrc = `
func I32(x float64) int32 { return int32(x) }
func I64(x float64) int64 { return int64(x) }
func U64(x float64) uint64 { return uint64(x) }
`
	for _, arch := range []string{"amd64", "arm64", "386"} {
		for _, saturating := range []bool{false, true} {
			mode := "legacy"
			if saturating {
				mode = "saturating"
			}
			t.Run(arch+"/"+mode, func(t *testing.T) {
				src := baseSrc
				if saturating {
					src += saturatingSrc
				}
				ssaPkg, _, files := buildGoSSAPkg(t, src)
				prog := newLLSSAProgForTarget(t, &llssa.Target{
					GOOS: "linux", GOARCH: arch,
					SaturatingFloatToInt: saturating,
				})
				defer prog.Dispose()
				pkg, err := compileTestPackage(prog, ssaPkg, files)
				if err != nil {
					t.Fatal(err)
				}
				for _, name := range []string{"F32", "F64"} {
					ir := mustNamedFunction(t, pkg.Module(), "floatconvert."+name).String()
					want := "fptosi "
					if saturating {
						want = "@llvm.fptoui.sat.i32." + strings.ToLower(name) + "("
					} else if arch == "arm64" {
						want = "@llvm.fptosi.sat.i64." + strings.ToLower(name) + "("
					}
					if !strings.Contains(ir, want) {
						t.Fatalf("%s conversion IR missing %q in %s mode:\n%s", name, want, mode, ir)
					}
					if !saturating && !strings.Contains(ir, "trunc i64") {
						t.Fatalf("%s legacy uint32 conversion must truncate i64:\n%s", name, ir)
					}
				}
				if saturating {
					for _, conversion := range []struct {
						name string
						want string
					}{
						{"I32", "@llvm.fptosi.sat.i32.f64("},
						{"I64", "@llvm.fptosi.sat.i64.f64("},
						{"U64", "@llvm.fptoui.sat.i64.f64("},
					} {
						ir := mustNamedFunction(t, pkg.Module(), "floatconvert."+conversion.name).String()
						if !strings.Contains(ir, conversion.want) {
							t.Fatalf("%s conversion IR missing %q in %s mode:\n%s", conversion.name, conversion.want, mode, ir)
						}
					}
				}
			})
		}
	}
}
