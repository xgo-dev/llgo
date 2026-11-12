package build

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/packages"
)

func TestDarwinStdlibImportTrampolines(t *testing.T) {
	const decl = `package unix
//go:cgo_import_dynamic libc_arc4random_buf arc4random_buf "/usr/lib/libSystem.B.dylib"
func libc_arc4random_buf_trampoline()
`
	const source = "TEXT ·libc_arc4random_buf_trampoline(SB),4,$0-0\nJMP libc_arc4random_buf(SB)\n"
	for _, arch := range []string{"amd64", "arm64"} {
		for _, tc := range []struct {
			name, decl, source, path string
			want                     bool
		}{
			{"SDK import", decl, source, "asm_darwin.s", true},
			{"symbolic flags", decl, strings.Replace(source, ",4,", ",NOSPLIT,", 1), "asm_darwin.s", true},
			{"different target", decl, strings.Replace(source, "JMP libc_arc4random_buf", "JMP other", 1), "asm_darwin.s", false},
			{"different file", decl, source, "custom.s", false},
			{"extra instruction", decl, source + "RET\n", "asm_darwin.s", false},
			{"DATA", decl, source + "DATA ·value(SB)/8,$1\nGLOBL ·value(SB),$8\n", "asm_darwin.s", false},
			{"frame", decl, strings.Replace(source, "$0-0", "$8-0", 1), "asm_darwin.s", false},
			{"arguments", strings.Replace(decl, "trampoline()", "trampoline(x int)", 1), source, "asm_darwin.s", false},
			{"Go body", strings.Replace(decl, "trampoline()", "trampoline() {}", 1), source, "asm_darwin.s", false},
			{"no import", "package unix\nfunc libc_arc4random_buf_trampoline()\n", source, "asm_darwin.s", false},
		} {
			t.Run(arch+"/"+tc.name, func(t *testing.T) {
				fset := token.NewFileSet()
				file, err := parser.ParseFile(fset, "unix.go", tc.decl, parser.ParseComments)
				if err != nil {
					t.Fatal(err)
				}
				typ, err := new(types.Config).Check("internal/syscall/unix", fset, []*ast.File{file}, nil)
				if err != nil {
					t.Fatal(err)
				}
				ctx := &context{buildConf: &Config{Goos: "darwin", Goarch: arch}}
				pkg := &packages.Package{PkgPath: typ.Path(), Types: typ, Syntax: []*ast.File{file}}
				if got := darwinStdlibImportTrampolines(ctx, pkg, tc.path, []byte(tc.source)); got != tc.want {
					t.Fatalf("import trampoline = %v, want %v", got, tc.want)
				}
			})
		}
	}
}
