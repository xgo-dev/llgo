//go:build !llgo

package build

import (
	"fmt"
	"github.com/xgo-dev/llvm"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSIMDInterleaveLLVM(t *testing.T) {
	for _, target := range []struct{ os, arch string }{{"linux", "amd64"}, {"windows", "amd64"}, {"darwin", "arm64"}} {
		t.Run(target.os+"/"+target.arch, func(t *testing.T) {
			methods := []string{"InterleaveLo", "InterleaveHi"}
			widths := []int{16, 32, 64}
			if target.arch == "arm64" {
				widths = append(widths, 8)
				methods = append(methods, "ConcatEven", "ConcatOdd", "InterleaveEven", "InterleaveOdd")
			}
			source := "package main\nimport \"simd/archsimd\"\nfunc main() {}\n"
			var names []string
			add := func(typ, method string) {
				name := typ + method
				names = append(names, name)
				source += fmt.Sprintf("func %s(x,y archsimd.%s) archsimd.%s {return x.%s(y)}\n", name, typ, typ, method)
			}
			for _, width := range widths {
				for _, kind := range []string{"Int", "Uint"} {
					for _, method := range methods {
						add(fmt.Sprintf("%s%dx%d", kind, width, 128/width), method)
					}
					if target.arch == "amd64" {
						for _, method := range methods {
							add(fmt.Sprintf("%s%dx%d", kind, width, 256/width), method+"Grouped")
						}
					}
				}
			}
			dir := t.TempDir()
			for name, body := range map[string]string{"go.mod": "module interleave\n\ngo 1.27\n", "main.go": source} {
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
				for _, name := range names {
					fn := mod.NamedFunction("main." + name)
					if fn.IsNil() {
						t.Fatalf("missing %s", name)
					}
					body := fn.String()
					if !strings.Contains(body, "shufflevector") || strings.Contains(body, "PanicSIMDUnimplemented") {
						t.Fatalf("missing lane shuffle in %s:\n%s", name, body)
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
