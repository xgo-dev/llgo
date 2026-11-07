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
	"github.com/xgo-dev/llvm"
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

	// GOROOT/test/fixedbugs/issue38125.go uses an alias to an anonymous
	// receiver. Compile each retention separately: emitting T's metadata
	// first can hide a mismatched method symbol in the *T metadata path.
	const source = `package p
type I int
func (I) M() {}
type T = struct{ I }
func F() {
	_ = T.M
	_ = struct{ I }.M
}
`
	for _, target := range []llssa.Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "linux", GOARCH: "arm64"},
		{GOOS: "linux", GOARCH: "arm"},
		{GOOS: "darwin", GOARCH: "amd64"},
		{GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "windows", GOARCH: "amd64"},
		{GOOS: "windows", GOARCH: "arm64"},
		{GOOS: "windows", GOARCH: "386"},
		{GOOS: "wasip1", GOARCH: "wasm"},
	} {
		t.Run("promoted/"+target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			for _, test := range []struct {
				name       string
				decl       string
				staticItab bool
			}{
				{"alias", "var metadata any = (*T)(nil)", false},
				{"alias-chain", "type alias = T; var metadata any = (*alias)(nil)", false},
				{"pointer-alias", "type pointer = *T; var metadata any = pointer(nil)", false},
				{"interface", "var value interface{ M() } = T{}", false},
				{"expression", "var metadata any = (*T)(nil); var expression = T.M", false},
				{"bound", "var metadata any = (*T)(nil); var bound = T{}.M", false},
				{"static-interface", "var value interface{ M() } = T{}", true},
				{"static-pointer", "var value interface{ M() } = (*T)(nil)", true},
			} {
				t.Run(test.name, func(t *testing.T) {
					ssaPkg, _, files := buildGoSSAPkg(t, source+test.decl)
					prog := newLLSSAProgForTarget(t, &target)
					defer prog.Dispose()
					if test.staticItab {
						prog.EnableGoGlobalDCE(true)
						prog.EnableLTOPluginMarkers(true)
					}
					pkg, err := NewPackage(prog, ssaPkg, files)
					if err != nil {
						t.Fatal(err)
					}
					mod := pkg.Module()
					if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
						t.Fatal(err)
					}
					if test.staticItab {
						var found bool
						for global := mod.FirstGlobal(); !global.IsNil(); global = llvm.NextGlobal(global) {
							if strings.HasPrefix(global.Name(), "_llgo_itab$") {
								found = true
							}
						}
						if !found {
							t.Fatal("missing static itab template")
						}
					}
					for _, name := range []string{"p.struct{p.I}.M", "p.*struct{p.I}.M"} {
						if fn := mustNamedFunction(t, mod, name); fn.IsDeclaration() {
							t.Errorf("method table target %q has no body", name)
						}
					}
					for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
						if strings.HasPrefix(fn.Name(), "p.") && strings.HasSuffix(fn.Name(), ".M") && fn.IsDeclaration() {
							t.Errorf("unresolved promoted method %q", fn.Name())
						}
					}
				})
			}
		})
	}
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
