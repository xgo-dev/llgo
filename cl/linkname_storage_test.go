package cl

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/ssa/ssatest"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func TestLinknameVariableStorageIR(t *testing.T) {
	const source = `package shared
import _ "unsafe"
//go:linkname Zero shared_zero
var Zero int32
//go:linkname Static shared_static
var Static int32 = 41
//go:linkname Callback shared_callback
var Callback func(int32) int32
//go:linkname Native C.shared_native
var Native func(int32) int32
//go:linkname Alias shared_static
var Alias int32
//go:linkname Reserved go:shared_value
var Reserved int32
//go:cgo_import_dynamic shared_import imported "libexample.so"
//go:linkname Imported shared_import
var Imported uintptr
//go:cgo_import_dynamic shared.Default imported "libexample.so"
var Default uintptr
//go:cgo_import_static static_import
//go:linkname StaticImport static_import
var StaticImport byte
//go:cgo_import_dynamic shared.Initialized imported "libexample.so"
var Initialized uintptr = 7
`
	for _, target := range []*llssa.Target{
		{GOOS: "darwin", GOARCH: "arm64"}, {GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "windows", GOARCH: "386"}, {GOOS: "windows", GOARCH: "amd64"},
		{GOOS: "windows", GOARCH: "arm64"}, {GOOS: "wasip1", GOARCH: "wasm"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			ir := compileWithRewritesTarget(t, source, nil, target)
			for _, want := range []string{
				"@shared_zero = common global i32 0", "@shared_static = global i32 41",
				"@shared_callback = common global { ptr, ptr } zeroinitializer",
				"@shared_native = external global ptr", `@"go:shared_value" = common global i32 0`,
				"@static_import = external global i8", "@shared_import = external global i", "@shared.Default = external global i",
				fmt.Sprintf("@shared.Initialized = global i%d 7", map[bool]int{true: 32, false: 64}[target.GOARCH == "386" || target.GOARCH == "wasm"]),
			} {
				if target.GOARCH == "wasm" || target.GOOS == "darwin" {
					want = strings.ReplaceAll(want, "common global", "weak global")
				}
				if !strings.Contains(ir, want) {
					t.Errorf("missing %q:\n%s", want, ir)
				}
			}
		})
	}
}

func TestLinknameDuplicateStaticDiagnostic(t *testing.T) {
	const source = `package duplicate
import _ "unsafe"
//go:linkname A shared_static
var A int32 = 41
//go:linkname B shared_static
var B int32 = 42
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "duplicate.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	imp := importer.Default()
	files := []*ast.File{file}
	pkg, _, err := ssautil.BuildPackage(&types.Config{Importer: imp}, fset, types.NewPackage("duplicate", "duplicate"), files, ssa.SanityCheckFunctions)
	if err != nil {
		t.Fatal(err)
	}
	prog := ssatest.NewProgramEx(t, nil, imp)
	_, _, err = NewPackageEx(prog, nil, nil, pkg, files)
	if err == nil {
		t.Fatal("duplicate definitions compiled")
	}
	for _, want := range []string{"duplicate.go:6:", "B:", "duplicate definition", "shared_static"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("diagnostic %q is missing %q", err, want)
		}
	}
}

func TestLinknameExternalImportAcrossPackages(t *testing.T) {
	fset := token.NewFileSet()
	imp := importer.Default()
	prog := ssatest.NewProgramEx(t, nil, imp)
	imports, err := parser.ParseFile(fset, "imports.go", `package imports
//go:cgo_import_dynamic shared_import remote "libexample.so"
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	if err := ParsePkgSyntax(prog, fset, types.NewPackage("imports", "imports"), []*ast.File{imports}); err != nil {
		t.Fatal(err)
	}
	consumer, err := parser.ParseFile(fset, "consumer.go", `package consumer
import _ "unsafe"
//go:linkname Value shared_import
var Value uintptr
var Unrelated uintptr
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	files := []*ast.File{consumer}
	pkg, _, err := ssautil.BuildPackage(&types.Config{Importer: imp}, fset, types.NewPackage("consumer", "consumer"), files, ssa.SanityCheckFunctions)
	if err != nil {
		t.Fatal(err)
	}
	compiled, _, err := NewPackageEx(prog, nil, nil, pkg, files)
	if err != nil {
		t.Fatal(err)
	}
	mod := compiled.Module()
	if !mod.NamedGlobal("shared_import").Initializer().IsNil() {
		t.Fatal("import in another package did not replace eligible zero storage")
	}
	if mod.NamedGlobal("consumer.Unrelated").Initializer().IsNil() {
		t.Fatal("unrelated Go variable lost storage")
	}
}

func TestLinknameImportCanonicalSpelling(t *testing.T) {
	for _, tc := range []struct {
		directive string
		imported  bool
	}{
		{"//go:cgo_import_dynamic shared remote \"libexample.so\"", true},
		{"//go:cgo_import_static shared", true},
		{"// go:cgo_import_static shared", false},
		{"/*go:cgo_import_static shared*/", false},
		{"//go:cgo_import_staticX shared", false},
		{"//go:cgo_import_static\tshared", false},
		{"//go:cgo_import_static _", false},
	} {
		t.Run(tc.directive, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "imports.go", "package imports\n"+tc.directive+"\n", parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			prog := ssatest.NewProgramEx(t, nil, importer.Default())
			if err := ParsePkgSyntax(prog, fset, types.NewPackage("imports", "imports"), []*ast.File{file}); err != nil {
				t.Fatal(err)
			}
			if got := prog.IsExternalImport("shared"); got != tc.imported {
				t.Fatalf("IsExternalImport = %v, want %v", got, tc.imported)
			}
			if prog.IsExternalImport("_") {
				t.Fatal("library-only pragma created a variable import")
			}
		})
	}
}
