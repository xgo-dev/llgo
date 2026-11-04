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

// This tests only the source address role. Frame/body legality is independently
// enforced by the existing native backend in signature discovery and codegen.
func TestPlan9AsmAddressOnlyNativeEntrySourceRole(t *testing.T) {
	files := nativeAddressFixtureFiles(t)
	fset := token.NewFileSet()
	syntax, err := parser.ParseFile(fset, "main.go", files["main.go"], parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, path := range []string{"example.com/first", "example.com/second"} {
		config := types.Config{Sizes: types.SizesFor("gc", "amd64")}
		typed, err := config.Check(path, fset, []*ast.File{syntax}, nil)
		if err != nil {
			t.Fatal(err)
		}
		pkg := &packages.Package{Types: typed, Syntax: []*ast.File{syntax}}
		entries := nativeAddressEntries(pkg, []byte(files["entry.s"]), "amd64")
		if len(entries) != 1 || entries["rawEntry"] == "" {
			t.Fatalf("exact uintptr DATA role missing: %v", entries)
		}
		again := nativeAddressEntries(pkg, []byte(files["entry.s"]), "amd64")
		if again["rawEntry"] != entries["rawEntry"] {
			t.Fatal("source binding identity is unstable")
		}
		names = append(names, entries["rawEntry"])
		for _, source := range []string{
			strings.ReplaceAll(files["entry.s"], "·entry(SB)", "entry(SB)"),
			strings.ReplaceAll(files["entry.s"], "·entry(SB)", "other·entry(SB)"),
			files["entry.s"] + "\nDATA ·entry(SB)/8, $rawEntry(SB)\n",
			files["entry.s"] + "\nGLOBL ·entry(SB), RODATA|NOPTR, $8\n",
		} {
			if entries := nativeAddressEntries(pkg, []byte(source), "amd64"); len(entries) != 0 {
				t.Fatalf("ambiguous or nonlocal DATA acquired an address role: %v", entries)
			}
		}
	}
	if names[0] == names[1] {
		t.Fatal("unrelated packages share a native entry identity")
	}
}

// Source directives can rename a global or intentionally collide with the
// generated carrier name. Both cases must remain declaration-driven failures.
func TestPlan9AsmAddressOnlyNativeEntryAliasRoles(t *testing.T) {
	base := nativeAddressFixtureFiles(t)
	_, apkg := nativeAddressFixturePackage(t, base, "linux", "amd64", "x86_64-unknown-linux-gnu")
	object := nativeAddressEntries(apkg.Package, []byte(base["entry.s"]), "amd64")["rawEntry"]
	if object == "" {
		t.Fatal("positive source role missing")
	}
	for _, row := range []struct{ name, directive string }{
		{"LLGo alias", "//llgo:link entry example.com/other.entry"},
		{"spaced LLGo alias", "// llgo:link entry example.com/other.entry"},
		{"normalized Go alias", "//go: linkname entry example.com/other.entry"},
		{"Go symbol collision", "//go:linkname shadow " + object + "\nfunc shadow()"},
		{"native import collision", "//go:cgo_import_dynamic imported " + object},
	} {
		t.Run(row.name, func(t *testing.T) {
			source := strings.Replace(base["main.go"], "var entry uintptr", row.directive+"\nvar entry uintptr", 1)
			if strings.Contains(row.directive, "go:linkname") {
				source = strings.Replace(source, "package main\n", "package main\nimport _ \"unsafe\"\n", 1)
			}
			files := map[string]string{"go.mod": base["go.mod"], "main.go": source, "entry.s": base["entry.s"]}
			_, apkg := nativeAddressFixturePackage(t, files, "linux", "amd64", "x86_64-unknown-linux-gnu")
			if entries := nativeAddressEntries(apkg.Package, []byte(files["entry.s"]), "amd64"); len(entries) != 0 {
				t.Fatalf("source alias/collision acquired an address role: %v", entries)
			}
		})
	}
}
