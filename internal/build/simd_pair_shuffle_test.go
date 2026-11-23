//go:build !llgo

package build

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestSIMDPairShuffleLLVM(t *testing.T) {
	for _, goos := range []string{"linux", "windows"} {
		t.Run(goos, func(t *testing.T) {
			source := "package main\nimport \"simd/archsimd\"\nfunc main() {}\n"
			var intrinsics []string
			for _, width := range []int{128, 256} {
				for _, bits := range []int{32, 64} {
					for _, kind := range []string{"Int", "Uint", "Float"} {
						typ := fmt.Sprintf("%s%dx%d", kind, bits, width/bits)
						method, intrinsic := "ConcatPermuteScalars", "concatSelectedConstant"
						if width == 256 {
							method += "Grouped"
							intrinsic += "Grouped"
						}
						selectors := "a,b"
						if bits == 32 {
							selectors += ",c,d"
						}
						source += fmt.Sprintf("func select%s(x,y archsimd.%s,a,b,c,d uint8) archsimd.%s {return x.%s(%s,y)}\n", typ, typ, typ, method, selectors)
						intrinsics = append(intrinsics, typ+"."+intrinsic)
					}
				}
			}
			for _, typ := range []string{"Int8x32", "Uint8x32", "Int16x16", "Uint16x16", "Int32x8", "Uint32x8", "Int64x4", "Uint64x4", "Float32x8", "Float64x4"} {
				source += fmt.Sprintf("func halves%s(x,y archsimd.%s,lo,hi uint8) archsimd.%s {return x.ConcatPermute128Scalars(lo,hi,y)}\n", typ, typ, typ)
				intrinsics = append(intrinsics, typ+".ConcatPermute128Scalars")
			}
			source += `
func constant(x,y archsimd.Float32x8) archsimd.Float32x8 {return x.ConcatPermute128Scalars(3,0,y)}
func invalid(x,y archsimd.Uint64x4) archsimd.Uint64x4 {return x.ConcatPermute128Scalars(4,0,y)}
//go:noinline
func guarded(x,y archsimd.Float32x8,lo,hi uint8) archsimd.Float32x8 {
 if archsimd.X86.AVX2() { return x.ConcatPermute128Scalars(lo,hi,y) }
 return y.ConcatPermute128Scalars(hi,lo,x)
}
//go:noinline
func halvesUint8x32Guarded(x,y archsimd.Uint8x32,lo,hi uint8) archsimd.Uint8x32 {
 if archsimd.X86.AVX2() { return halvesUint8x32(x,y,lo,hi) }
 return halvesUint8x32(y,x,hi,lo)
}
`
			dir := t.TempDir()
			for name, body := range map[string]string{"go.mod": "module pairshuffle\n\ngo 1.27\n", "main.go": source} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0600); err != nil {
					t.Fatal(err)
				}
			}
			conf := NewDefaultConf(ModeGen)
			conf.Goos, conf.Goarch, conf.GOEXPERIMENT, conf.GOAMD64 = goos, "amd64", "simd", "v1"
			var callers []*aPackage
			for _, arg := range []string{".", "simd/archsimd"} {
				pkgs, err := Build(Invocation{Args: []string{arg}, Config: conf, Dir: dir})
				if err != nil {
					t.Fatal(err)
				}
				for _, pkg := range pkgs {
					defer pkg.LPkg.Prog.Dispose()
					mod := pkg.LPkg.Module()
					if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
						t.Fatal(err)
					}
					if pkg.PkgPath == "simd/archsimd" {
						// Callable intrinsic bodies must implement variable selectors,
						// even when the usual direct call uses an immediate constant.
						for _, name := range intrinsics {
							fn := mod.NamedFunction("simd/archsimd." + name)
							if fn.IsNil() {
								t.Fatalf("missing %s", name)
							}
							body := fn.String()
							if strings.Contains(body, "PanicSIMDUnimplemented") {
								t.Fatalf("missing shuffle in %s:\n%s", name, body)
							}
							if strings.HasSuffix(name, "ConcatPermute128Scalars") {
								if !strings.Contains(body, "PanicSIMDImmediate") {
									t.Fatalf("missing selector bounds in %s", name)
								}
								// Whole-half selection must not regress to dynamic scalar
								// extraction, which spills and reloads each byte on x86.
								if !strings.Contains(body, "shufflevector") || !strings.Contains(body, "select i1") || strings.Contains(body, "extractelement") || strings.Contains(body, "insertelement") {
									t.Fatalf("expected whole-half selection in %s:\n%s", name, body)
								}
							} else if !strings.Contains(body, "extractelement") {
								t.Fatalf("missing lane selection in %s:\n%s", name, body)
							}
						}
					} else {
						callers = append(callers, pkg)
						body := mod.NamedFunction("main.constant").String()
						if !strings.Contains(body, "<i32 12, i32 13, i32 14, i32 15, i32 0, i32 1, i32 2, i32 3>") {
							t.Fatalf("incorrect constant half shuffle:\n%s", body)
						}
						body = mod.NamedFunction("main.invalid").String()
						if !strings.Contains(body, "PanicSIMDImmediate") {
							t.Fatalf("missing invalid constant panic:\n%s", body)
						}
					}
				}
			}
			simdCodegenVariants(t, []string{"O0", "O2"}, callers, func(t *testing.T, mod llvm.Module, asm string) {
				if goos == "linux" {
					assertSIMDBaselineAssembly(t, mod, asm)
					bodies := regexp.MustCompile(`(?ms)^\s*\.type\s+([^\n]+),@function\n(.*?)^\s*\.size\s+[^\n]+`).FindAllStringSubmatch(asm, -1)
					foundBaseline, foundAVX2 := false, false
					byteLoad := regexp.MustCompile(`(?m)\bmovzbl\s+[^\n]*\([^\n]*%rsp`)
					for _, body := range bodies {
						if !strings.Contains(body[1], "main.halvesUint8x32") {
							continue
						}
						if strings.Contains(body[1], ".__llgo_fmv_avx2") {
							foundAVX2 = true
						} else {
							foundBaseline = true
						}
						if strings.Contains(body[2], "pinsrb") || byteLoad.MatchString(body[2]) {
							t.Fatalf("half shuffle reconstructed bytes from the stack in %s:\n%s", body[1], body[2])
						}
					}
					if !foundBaseline || !foundAVX2 {
						t.Fatal("missing baseline or AVX2 Uint8x32 half-shuffle assembly")
					}
				}
			})
		})
	}
}
