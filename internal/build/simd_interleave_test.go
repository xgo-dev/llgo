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
			var operations []struct{ typ, method string }
			add := func(typ, method string) {
				name := typ + method
				operations = append(operations, struct{ typ, method string }{typ, method})
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
			// Build each root explicitly: direct calls and method values use
			// different entry points, and both must lower to the same shuffle.
			var pkgs, callers []*aPackage
			for _, arg := range []string{".", "simd/archsimd"} {
				built, err := Build(Invocation{Args: []string{arg}, Config: conf, Dir: dir})
				if err != nil {
					t.Fatal(err)
				}
				pkgs = append(pkgs, built...)
			}
			// Literal masks independently pin every operation's lane order,
			// including the boundary between the two 128-bit groups.
			masks := map[string]string{
				"Int32x4.InterleaveLo":        "<i32 0, i32 4, i32 1, i32 5>",
				"Int32x4.InterleaveHi":        "<i32 2, i32 6, i32 3, i32 7>",
				"Int32x4.ConcatEven":          "<i32 0, i32 2, i32 4, i32 6>",
				"Int32x4.ConcatOdd":           "<i32 1, i32 3, i32 5, i32 7>",
				"Int32x4.InterleaveEven":      "<i32 0, i32 4, i32 2, i32 6>",
				"Int32x4.InterleaveOdd":       "<i32 1, i32 5, i32 3, i32 7>",
				"Int32x8.InterleaveLoGrouped": "<i32 0, i32 8, i32 1, i32 9, i32 4, i32 12, i32 5, i32 13>",
				"Int32x8.InterleaveHiGrouped": "<i32 2, i32 10, i32 3, i32 11, i32 6, i32 14, i32 7, i32 15>",
			}
			for _, pkg := range pkgs {
				if pkg.PkgPath != "simd/archsimd" {
					callers = append(callers, pkg)
				}
				defer pkg.LPkg.Prog.Dispose()
				mod := pkg.LPkg.Module()
				if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
				for _, op := range operations {
					name := "main." + op.typ + op.method
					if pkg.PkgPath == "simd/archsimd" {
						name = "simd/archsimd." + op.typ + "." + op.method
					}
					fn := mod.NamedFunction(name)
					if fn.IsNil() {
						t.Fatalf("missing %s", name)
					}
					body := fn.String()
					if !strings.Contains(body, "shufflevector") || strings.Contains(body, "PanicSIMDUnimplemented") {
						t.Fatalf("missing lane shuffle in %s:\n%s", name, body)
					}
					if mask := masks[op.typ+"."+op.method]; mask != "" && !strings.Contains(body, mask) {
						t.Fatalf("incorrect lane mask in %s, want %s:\n%s", name, mask, body)
					}
				}
			}
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
