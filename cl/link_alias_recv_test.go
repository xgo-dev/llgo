//go:build !llgo

package cl

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
)

func linkedAliasReceiverCases(t *testing.T, test func(*testing.T, string, string)) {
	t.Helper()
	for _, recv := range []struct {
		name string
		want string
	}{
		{"Data", "Data.Cursor"},
		{"*Data", "(*Data).Cursor"},
		{"Value", "Data.Cursor"},
		{"*Value", "(*Data).Cursor"},
		{"Ptr", "(*Data).Cursor"},
		{"Chain", "(*Data).Cursor"},
	} {
		method := recv.name + ".Cursor"
		methods := []string{method}
		if strings.HasPrefix(recv.name, "*") {
			methods[0] = "(" + recv.name + ").Cursor"
		} else {
			methods = append(methods, "("+recv.name+").Cursor")
		}
		for _, method := range methods {
			for _, directive := range []string{"go:linkname", "llgo:link", " llgo:link", "go:linkname-after"} {
				t.Run(method+"/"+directive, func(t *testing.T) {
					comment := fmt.Sprintf("//%s %s C.test_cursor", strings.TrimSuffix(directive, "-after"), method)
					after := ""
					if strings.HasSuffix(directive, "-after") {
						comment, after = "", comment
					}
					source := fmt.Sprintf(`package p
import _ "unsafe"
const LLGoPackage = "decl"
type Data struct{}
type Value = Data
type Ptr = *Value
type Chain = Ptr
%s
func (%s) Cursor() int32 { return -1 }
func Call(p *Data) int32 { return p.Cursor() }
%s
`, comment, recv.name, after)
					test(t, source, "p."+recv.want)
				})
			}
		}
	}
}

func TestLinkedAliasReceiverSyntax(t *testing.T) {
	linkedAliasReceiverCases(t, func(t *testing.T, source, want string) {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "p.go", source, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		prog := llssa.NewProgram(nil)
		defer prog.Dispose()
		// Match the build driver's preload callback: the type scope is empty.
		if err := ParsePkgSyntax(prog, fset, types.NewPackage("p", "p"), []*ast.File{file}); err != nil {
			t.Fatal(err)
		}
		if link, ok := prog.Linkname(want); !ok || link != "C.test_cursor" {
			t.Fatalf("Linkname(%q) = (%q, %v), want (C.test_cursor, true)", want, link, ok)
		}
	})
}

func TestLinkedAliasReceiverCompile(t *testing.T) {
	linkedAliasReceiverCases(t, func(t *testing.T, source, _ string) {
		_, m := mustCompileLLPkgFromSrc(t, source)
		ir := mustNamedFunction(t, m, "p.Call").String()
		if !strings.Contains(ir, "call i32 @test_cursor(") {
			t.Fatalf("call did not use the linked C symbol:\n%s", ir)
		}
	})
}

func TestLinkedAliasReceiverAcrossFiles(t *testing.T) {
	fset := token.NewFileSet()
	var files []*ast.File
	for i, source := range []string{
		`package p
// llgo:link Chain.Cursor C.test_cursor
func (Chain) Cursor() int32 { return -1 }
type Chain = Ptr`,
		`package p
type Ptr = *(Value)
type Value = Data
type Data struct{}`,
	} {
		file, err := parser.ParseFile(fset, fmt.Sprintf("p%d.go", i), source, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, file)
	}
	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	if err := ParsePkgSyntax(prog, fset, types.NewPackage("p", "p"), files); err != nil {
		t.Fatal(err)
	}
	if link, ok := prog.Linkname("p.(*Data).Cursor"); !ok || link != "C.test_cursor" {
		t.Fatalf("cross-file Linkname = (%q, %v), want (C.test_cursor, true)", link, ok)
	}
}

func TestReceiverAliasesLeaveInvalidTypesForTypeChecker(t *testing.T) {
	for _, decl := range []string{
		"type A = B; type B = A",
		"type A = *A",
		"type A = struct{}",
		"type A = other.Type",
	} {
		t.Run(decl, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "p.go", "package p\n"+decl+"\nfunc (A) M() {}", parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			prog := llssa.NewProgram(nil)
			defer prog.Dispose()
			// Syntax preloading must not panic or loop on invalid receivers.
			if err := ParsePkgSyntax(prog, fset, types.NewPackage("p", "p"), []*ast.File{file}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestLinkedAliasReceiverImport(t *testing.T) {
	linkedAliasReceiverCases(t, func(t *testing.T, source, want string) {
		path := filepath.Join(t.TempDir(), "p.go")
		if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, source, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		pkg, err := (&types.Config{Importer: importer.Default()}).Check("p", fset, []*ast.File{file}, nil)
		if err != nil {
			t.Fatal(err)
		}
		prog := llssa.NewProgram(nil)
		defer prog.Dispose()
		ctx := &context{prog: prog, fset: fset}
		ctx.importPkg(pkg, &pkgInfo{})
		if link, ok := prog.Linkname(want); !ok || link != "C.test_cursor" {
			t.Fatalf("imported Linkname(%q) = (%q, %v), want (C.test_cursor, true)", want, link, ok)
		}
	})
}

func TestLinkDirectiveErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  string
		want string
	}{
		{"wrong receiver", "//llgo:link Wrong.Cursor C.test_cursor\nfunc (Ptr) Cursor() int32 { return -1 }", `local name "Wrong.Cursor" does not match declaration "Ptr.Cursor"`},
		{"wrong method", "// llgo:link Ptr.Curosr C.test_cursor\nfunc (Ptr) Cursor() int32 { return -1 }", `local name "Ptr.Curosr" does not match declaration "Ptr.Cursor"`},
		{"wrong parenthesized method", "// llgo:link (Ptr).Curosr C.test_cursor\nfunc (Ptr) Cursor() int32 { return -1 }", `local name "(Ptr).Curosr" does not match declaration "Ptr.Cursor"`},
		{"wrong function", "//llgo:link Other C.test_cursor\nfunc Cursor() int32 { return -1 }", `local name "Other" does not match declaration "Cursor"`},
		{"missing target", "//llgo:link Ptr.Cursor\nfunc (Ptr) Cursor() int32 { return -1 }", "requires a local name and a target"},
		{"missing names", "// llgo:link\nfunc (Ptr) Cursor() int32 { return -1 }", "requires a local name and a target"},
		{"detached", "func (Ptr) Cursor() int32 { return -1 }\n//llgo:link Ptr.Cursor C.test_cursor", "is not attached to a declaration"},
		{"detached missing target", "func (Ptr) Cursor() int32 { return -1 }\n//llgo:link Ptr.Cursor", "requires a local name and a target"},
		{"invalid go method", "//go:linkname Ptr.Curosr C.test_cursor\nfunc (Ptr) Cursor() int32 { return -1 }", `local method "Ptr.Curosr" not found`},
		{"detached invalid go method", "func (Ptr) Cursor() int32 { return -1 }\n//go:linkname Wrong.Cursor C.test_cursor", `local method "Wrong.Cursor" not found`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			source := "package p\nimport _ \"unsafe\"\ntype Data struct{}\ntype Ptr = *Data\n" + tc.src
			file, err := parser.ParseFile(fset, "badlink.go", source, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			prog := llssa.NewProgram(nil)
			defer prog.Dispose()
			err = ParsePkgSyntax(prog, fset, types.NewPackage("p", "p"), []*ast.File{file})
			if err == nil || !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), "badlink.go:") {
				t.Fatalf("error = %v, want source position and %q", err, tc.want)
			}
		})
	}
}
