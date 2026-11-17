//go:build !llgo

package build

import (
	"fmt"
	"os"
	"path/filepath"
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
							if !strings.Contains(body, "extractelement") || strings.Contains(body, "PanicSIMDUnimplemented") {
								t.Fatalf("missing shuffle in %s:\n%s", name, body)
							}
							if strings.HasSuffix(name, "ConcatPermute128Scalars") && !strings.Contains(body, "PanicSIMDImmediate") {
								t.Fatalf("missing selector bounds in %s", name)
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
			simdCodegenVariants(t, []string{"O0", "O2"}, callers, func(t *testing.T, asm string) {
				if goos == "linux" {
					assertSIMDBaselineAssembly(t, asm)
				}
			})
		})
	}
}
