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

func TestSIMDHorizontalLLVM(t *testing.T) {
	for _, target := range []struct{ os, arch string }{{"linux", "amd64"}, {"windows", "amd64"}, {"darwin", "arm64"}} {
		t.Run(target.os+"/"+target.arch, func(t *testing.T) {
			types := []string{"Float32x4", "Float64x2", "Int16x8", "Uint16x8", "Int32x4", "Uint32x4"}
			if target.arch == "arm64" {
				types = append(types, "Int64x2", "Uint64x2")
			} else {
				types = append(types, "Float32x8", "Float64x4", "Int16x16", "Uint16x16", "Int32x8", "Uint32x8")
			}
			wide := map[string]bool{"Float32x8": true, "Float64x4": true, "Int16x16": true, "Uint16x16": true, "Int32x8": true, "Uint32x8": true}
			source := "package main\nimport \"simd/archsimd\"\nfunc main() {}\n"
			for _, typ := range types {
				op := "ConcatAddPairs"
				if wide[typ] {
					op += "Grouped"
				}
				source += fmt.Sprintf("func add%s(x,y archsimd.%s) archsimd.%s { return x.%s(y) }\n", typ, typ, typ, op)
			}
			if target.arch == "amd64" {
				source += `//go:noinline
func guarded(x,y archsimd.Float32x8) archsimd.Float32x8 {
 if archsimd.X86.AVX2() { return x.ConcatAddPairsGrouped(y) }
 return y.ConcatAddPairsGrouped(x)
}
`
			}
			dir := t.TempDir()
			for name, body := range map[string]string{"go.mod": "module horizontal\n\ngo 1.27\n", "main.go": source} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0600); err != nil {
					t.Fatal(err)
				}
			}
			conf := NewDefaultConf(ModeGen)
			conf.Goos, conf.Goarch, conf.GOEXPERIMENT = target.os, target.arch, "simd"
			if target.arch == "amd64" {
				conf.GOAMD64 = "v1"
			}
			// ModeGen returns a single initial package. Build each root explicitly
			// so both direct lowering and callable intrinsic bodies are checked.
			var pkgs []*aPackage
			for _, arg := range []string{".", "simd/archsimd"} {
				built, err := Build(Invocation{Args: []string{arg}, Config: conf, Dir: dir})
				if err != nil {
					t.Fatal(err)
				}
				pkgs = append(pkgs, built...)
			}
			var callers []*aPackage
			for _, pkg := range pkgs {
				if pkg.PkgPath != "simd/archsimd" {
					callers = append(callers, pkg)
				}
				defer pkg.LPkg.Prog.Dispose()
				mod := pkg.LPkg.Module()
				if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
				for _, typ := range types {
					name := "main.add" + typ
					if pkg.PkgPath == "simd/archsimd" {
						name = "simd/archsimd." + typ + ".ConcatAddPairs"
						if wide[typ] {
							name += "Grouped"
						}
					}
					fn := mod.NamedFunction(name)
					if fn.IsNil() {
						t.Fatalf("missing %s", name)
					}
					body := fn.String()
					op := "add <"
					if strings.HasPrefix(typ, "Float") {
						op = "fadd <"
					}
					if !strings.Contains(body, "shufflevector") || !strings.Contains(body, op) || strings.Contains(body, "PanicSIMDUnimplemented") {
						t.Fatalf("invalid horizontal operation %s:\n%s", name, body)
					}
					if typ == "Float32x8" && (!strings.Contains(body, "<i32 0, i32 2, i32 8, i32 10, i32 4, i32 6, i32 12, i32 14>") || !strings.Contains(body, "<i32 1, i32 3, i32 9, i32 11, i32 5, i32 7, i32 13, i32 15>")) {
						t.Fatalf("incorrect 128-bit groups:\n%s", body)
					}
				}
			}
			// Codegen the callers with independent IR copies for each pipeline.
			// Callable definitions are checked above and exercised by runtime tests;
			// avoid optimizing every unrelated archsimd intrinsic in this test.
			if len(callers) != 1 {
				t.Fatalf("expected one caller module, got %d", len(callers))
			}
			simdCodegenVariants(t, []string{"O0", "O2"}, callers, func(t *testing.T, asm string) {
				if target.os == "linux" {
					assertSIMDBaselineAssembly(t, asm)
				}
			})
		})
	}
}
