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

func TestSIMDScalarPermuteLLVM(t *testing.T) {
	for _, target := range []struct{ os, baseline string }{{"linux", "v1"}, {"linux", "v3"}, {"windows", "v1"}} {
		goos := target.os
		t.Run(goos+"/"+target.baseline, func(t *testing.T) {
			source := "package main\nimport \"simd/archsimd\"\nfunc main() {}\n"
			var intrinsics []string
			for _, family := range []struct {
				bits, lanes int
				method      string
			}{
				{32, 4, "PermuteScalars"}, {32, 8, "PermuteScalarsGrouped"},
				{16, 8, "PermuteScalarsHi"}, {16, 8, "PermuteScalarsLo"},
				{16, 16, "PermuteScalarsHiGrouped"}, {16, 16, "PermuteScalarsLoGrouped"},
			} {
				for _, kind := range []string{"Int", "Uint"} {
					typ := fmt.Sprintf("%s%dx%d", kind, family.bits, family.lanes)
					name := typ + family.method
					intrinsic := typ + ".p" + family.method[1:]
					intrinsics = append(intrinsics, intrinsic)
					source += fmt.Sprintf(`
//go:noinline
func scalar%s(x archsimd.%s,a,b,c,d uint8) archsimd.%s { return x.%s(a,b,c,d) }
func constant%s(x archsimd.%s) archsimd.%s { return x.%s(2,3,0,1) }
//go:noinline
func dispatch%s(x archsimd.%s,a,b,c,d uint8) archsimd.%s {
 if archsimd.X86.AVX2() { return scalar%s(x,a,b,c,d) }
 return scalar%s(x,d,c,b,a)
}
`, name, typ, typ, family.method, name, typ, typ, family.method, name, typ, typ, name, name)
				}
			}
			dir := t.TempDir()
			for name, body := range map[string]string{"go.mod": "module scalarpermute\n\ngo 1.27\n", "main.go": source} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0600); err != nil {
					t.Fatal(err)
				}
			}
			conf := NewDefaultConf(ModeGen)
			conf.Goos, conf.Goarch, conf.GOEXPERIMENT, conf.GOAMD64 = goos, "amd64", "simd", target.baseline
			var callers []*aPackage
			var intrinsicBitcode string
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
						for _, name := range intrinsics {
							fn := mod.NamedFunction("simd/archsimd." + name)
							if fn.IsNil() {
								t.Fatalf("missing %s", name)
							}
							body := fn.String()
							if strings.Contains(body, "PanicSIMD") || strings.Contains(body, "extractelement") || !strings.Contains(body, "select <") {
								t.Fatalf("missing vector selector lowering in %s:\n%s", name, body)
							}
						}
						intrinsicBitcode = filepath.Join(dir, "archsimd.bc")
						bitcode := llvm.WriteBitcodeToMemoryBuffer(mod)
						err := os.WriteFile(intrinsicBitcode, bitcode.Bytes(), 0600)
						bitcode.Dispose()
						if err != nil {
							t.Fatal(err)
						}
					} else {
						callers = append(callers, pkg)
					}
				}
			}
			// Public PermuteScalars helpers are Go bodies, not declarations. Link
			// their definitions before checking codegen so a bare forwarding call
			// cannot satisfy the no-scalar-extraction assertions below.
			for _, pkg := range callers {
				mod := pkg.LPkg.Module()
				defs, err := mod.Context().ParseBitcodeFile(intrinsicBitcode)
				if err != nil {
					t.Fatal(err)
				}
				if err := llvm.LinkModules(mod, defs); err != nil {
					t.Fatal(err)
				}
				// Retain only the fixture's call graph. Code-generating every
				// unrelated archsimd helper (including large selector switches)
				// would make this focused regression unnecessarily expensive.
				for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
					if fn.BasicBlocksCount() != 0 && !strings.HasPrefix(fn.Name(), "main.scalar") && !strings.HasPrefix(fn.Name(), "main.constant") && !strings.HasPrefix(fn.Name(), "main.dispatch") {
						fn.SetLinkage(llvm.InternalLinkage)
					}
				}
				for g := mod.FirstGlobal(); !g.IsNil(); g = llvm.NextGlobal(g) {
					if !g.Initializer().IsNil() && !strings.HasPrefix(g.Name(), "llvm.") {
						g.SetLinkage(llvm.InternalLinkage)
					}
				}
				opts := llvm.NewPassBuilderOptions()
				defer opts.Dispose()
				opts.SetVerifyEach(true)
				if err := mod.RunPasses("globaldce", pkg.LPkg.Prog.TargetMachine(), opts); err != nil {
					t.Fatal(err)
				}
			}
			simdCodegenVariants(t, []string{"O0", "O2"}, callers, func(t *testing.T, mod llvm.Module, asm string) {
				if goos != "linux" {
					return
				}
				if target.baseline == "v1" {
					assertSIMDBaselineAssembly(t, mod, asm)
				}
				bodies := regexp.MustCompile(`(?ms)^\s*\.type\s+([^\n]+),@function\n(.*?)^\s*\.size\s+[^\n]+`).FindAllStringSubmatch(asm, -1)
				indexedStack := regexp.MustCompile(`\([^\n)]*%(?:rsp|rbp),`)
				scalars, constants := 0, 0
				shuffle := regexp.MustCompile(`\b(?:v?pshuf[a-z]+|v?shufp[sd]|vpermilps)\b`)
				for _, body := range bodies {
					if strings.Contains(body[1], "main.constant") && strings.HasSuffix(t.Name(), "/O2") {
						constants++
						if strings.Contains(body[2], "call") || strings.Contains(body[2], "jmp") || !shuffle.MatchString(body[2]) {
							t.Fatalf("constant selectors did not fold to a shuffle in %s:\n%s", body[1], body[2])
						}
					}
					if !strings.Contains(body[1], "main.scalar") {
						continue
					}
					scalars++
					if strings.HasSuffix(t.Name(), "/O2") && (strings.Contains(body[2], "call") || strings.Contains(body[2], "jmp")) {
						t.Fatalf("dynamic selector retained a helper call in %s:\n%s", body[1], body[2])
					}
					if strings.Contains(body[2], "pextr") || indexedStack.MatchString(body[2]) {
						t.Fatalf("scalar permutation spilled/indexed lanes in %s:\n%s", body[1], body[2])
					}
				}
				if strings.HasSuffix(t.Name(), "/O2") && constants != len(intrinsics) {
					t.Fatalf("checked %d constant functions, want %d", constants, len(intrinsics))
				}
				if scalars != len(intrinsics) {
					t.Fatalf("checked %d scalar functions, want %d", scalars, len(intrinsics))
				}
			})
		})
	}
}
