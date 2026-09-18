//go:build !llgo

package build

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"testing"
)

func TestDarwinDynamicImportLinkArgs(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "imports.go", `package p
//go:cgo_ldflag "-pthread"
//go:cgo_import_dynamic local CFRelease "/System/Library/Frameworks/CoreFoundation.framework/Versions/A/CoreFoundation"
//go:cgo_import_dynamic second CFRetain "/System/Library/Frameworks/CoreFoundation.framework/Versions/A/CoreFoundation"
//go:cgo_import_dynamic write write "/usr/lib/libSystem.B.dylib"
//go:cgo_import_dynamic _ _ "/custom/libfoo.dylib"
//go:cgo_import_dynamic typo mach_vm_region "/usr/lib/libSystem.B.dylib""
//go:cgo_import_dynamic absent absent
`, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	files := []*ast.File{f}
	want := []string{"-pthread", "-framework", "CoreFoundation", "-lSystem.B", "/custom/libfoo.dylib"}
	if got := goCgoLinkArgs(files, "darwin"); !reflect.DeepEqual(got, want) {
		t.Fatalf("link args = %q, want %q", got, want)
	}
	for _, goos := range []string{"linux", "windows"} {
		if got := goCgoLinkArgs(files, goos); !reflect.DeepEqual(got, []string{"-pthread"}) {
			t.Fatalf("%s args = %q", goos, got)
		}
	}
}

func TestDirectiveQuotedFields(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want []string
	}{
		{`a b "/path with spaces/lib.dylib"`, []string{"a", "b", "/path with spaces/lib.dylib"}},
		{`local"library"`, []string{"local", "library"}},
		{`a b "lib.dylib""`, []string{"a", "b", "lib.dylib"}},
		{`a "unterminated`, []string{"a"}},
	} {
		if got := splitDirectiveArgs(tc.in); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("fields(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestSingleTokenDynamicImport(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "imports.go", "package p\n//go:cgo_import_dynamic strlen\n", parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	_, got := collectGoCgoPragmas([]*ast.File{f})
	want := []cgoImportDynamicDecl{{local: "strlen", alias: "strlen"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("imports=%v, want %v", got, want)
	}
}
