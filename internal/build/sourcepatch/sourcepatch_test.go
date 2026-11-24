package sourcepatch

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestParseDirective(t *testing.T) {
	tests := []struct {
		line    string
		skipAll bool
		names   []string
		ok      bool
	}{
		{line: "//llgo:skipall", skipAll: true, ok: true},
		{line: "// llgo:skipall", skipAll: true, ok: true},
		{line: "//llgo:skip Foo (*T).Bar", names: []string{"Foo", "(*T).Bar"}, ok: true},
		{line: "// llgo:skip Baz", names: []string{"Baz"}, ok: true},
		{line: "//go:skipall", ok: false},
		{line: "//llgo:link foo C.foo", ok: false},
		{line: "// plain comment", ok: false},
	}
	for _, tt := range tests {
		skipAll, names, ok := ParseDirective(tt.line)
		if skipAll != tt.skipAll || !slices.Equal(names, tt.names) || ok != tt.ok {
			t.Errorf("ParseDirective(%q) = (%v, %v, %v), want (%v, %v, %v)",
				tt.line, skipAll, names, ok, tt.skipAll, tt.names, tt.ok)
		}
	}
}

func TestFilterFileRemovesDeclarationsAndUnusedImports(t *testing.T) {
	src := []byte(`package demo

import (
	"fmt"
	"strings"
)

func Drop() {
	fmt.Println("drop")
}

func Keep(s string) string {
	return strings.TrimSpace(s)
}
`)
	filtered, changed, err := FilterFile(src, map[string]struct{}{"Drop": {}})
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected FilterFile to modify source")
	}
	text := string(filtered)
	if !strings.Contains(text, `// "fmt"`) {
		t.Fatalf("expected unused import fmt to be commented out:\n%s", text)
	}
	if strings.Contains(text, `// "strings"`) {
		t.Fatalf("used import strings was commented out:\n%s", text)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "demo.go", filtered, parser.ParseComments); err != nil {
		t.Fatalf("filtered file failed to parse: %v\n%s", err, text)
	}
}

func TestReleaseTagsForGoVersion(t *testing.T) {
	tags, err := ReleaseTagsForGoVersion("go1.25.3")
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 25 || tags[0] != "go1.1" || tags[24] != "go1.25" {
		t.Fatalf("ReleaseTagsForGoVersion(go1.25.3) = %v", tags)
	}
	if _, err := ReleaseTagsForGoVersion("invalid"); err == nil {
		t.Fatal("expected error for invalid Go version")
	}
}

func TestBuildInjectedFileSanitizesSkipDirectives(t *testing.T) {
	src := []byte("//llgo:skip Foo\npackage p\n")
	out := string(BuildInjectedFile("/tmp/patch.go", src))
	if !strings.HasPrefix(out, "//line /tmp/patch.go:1\n") {
		t.Fatalf("missing //line directive:\n%s", out)
	}
	if strings.Contains(out, "//llgo:skip") {
		t.Fatalf("skip directive was not sanitized:\n%s", out)
	}
}

func TestCloneOverlayAndBuildTagsAndMatchContext(t *testing.T) {
	if CloneOverlay(nil) != nil {
		t.Fatal("CloneOverlay(nil) != nil")
	}
	orig := map[string][]byte{"a.go": []byte("package a\n")}
	cloned := CloneOverlay(orig)
	cloned["a.go"][0] = 'X'
	if string(orig["a.go"]) != "package a\n" {
		t.Fatal("CloneOverlay mutated source slice")
	}

	tags := BuildTags([]string{"-tags", "foo,bar", "-tags=bar baz"}, []string{"extra"})
	if !slices.Equal(tags, []string{"foo", "bar", "baz", "extra"}) {
		t.Fatalf("BuildTags() = %v", tags)
	}

	cgo := false
	mc, err := NewMatchContext("/fake/goroot", BuildContext{
		GOOS:       "linux",
		GOARCH:     "amd64",
		GoVersion:  "go1.24.1",
		ToolTags:   []string{"tool1"},
		BuildFlags: []string{"-tags=custom"},
		TargetTags: []string{"tgt"},
		CgoEnabled: &cgo,
	})
	if err != nil {
		t.Fatalf("NewMatchContext() error: %v", err)
	}
	if mc.GOROOT != "/fake/goroot" || mc.GOOS != "linux" || mc.GOARCH != "amd64" || mc.CgoEnabled {
		t.Fatalf("unexpected MatchContext: %+v", mc)
	}
	if !slices.Equal(mc.BuildTags, []string{"custom", "tgt"}) || !slices.Equal(mc.ToolTags, []string{"tool1"}) {
		t.Fatalf("unexpected tags in MatchContext: build=%v tool=%v", mc.BuildTags, mc.ToolTags)
	}
	if _, err := NewMatchContext("", BuildContext{GoVersion: "bad"}); err == nil {
		t.Fatal("expected error for invalid GoVersion in NewMatchContext")
	}
}

func TestDeclKeysRecvKeyAndFilterSpecs(t *testing.T) {
	src := []byte(`package demo

// TypeDoc comment
type (
	DropType int // DropType line comment
	KeepType string
)

// VarDoc comment
var (
	DropVar = 1
	KeepVar = 2
)

const DropConst = 3

func (t *KeepType) Method() {}
func (t KeepType) ValMethod() {}
`)
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "demo.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	for _, decl := range file.Decls {
		keys = append(keys, DeclKeys(decl)...)
	}
	wantKeys := []string{"DropType", "KeepType", "DropVar", "KeepVar", "DropConst", "(*KeepType).Method", "KeepType.ValMethod"}
	if !slices.Equal(keys, wantKeys) {
		t.Fatalf("DeclKeys() = %v, want %v", keys, wantKeys)
	}
	if DeclKeys(&ast.BadDecl{}) != nil {
		t.Fatal("DeclKeys(BadDecl) != nil")
	}

	// Cover all RecvKey AST expression forms.
	ident := ast.NewIdent("T")
	if got := RecvKey(&ast.ParenExpr{X: ident}); got != "T" {
		t.Fatalf("RecvKey(Paren) = %q", got)
	}
	if got := RecvKey(&ast.IndexExpr{X: ident}); got != "T" {
		t.Fatalf("RecvKey(Index) = %q", got)
	}
	if got := RecvKey(&ast.IndexListExpr{X: ident}); got != "T" {
		t.Fatalf("RecvKey(IndexList) = %q", got)
	}
	if got := RecvKey(&ast.SelectorExpr{X: ast.NewIdent("pkg"), Sel: ident}); got != "T" {
		t.Fatalf("RecvKey(Selector) = %q", got)
	}

	skips := map[string]struct{}{
		"DropType":  {},
		"DropVar":   {},
		"DropConst": {},
	}
	if !MayContainSkip(src, skips) {
		t.Fatal("MayContainSkip() = false, want true")
	}
	if MayContainSkip(src, map[string]struct{}{"(*T).NotPresent": {}}) {
		t.Fatal("MayContainSkip() = true for absent symbol")
	}

	filtered, changed, err := FilterFile(src, skips)
	if err != nil || !changed {
		t.Fatalf("FilterFile() = (changed=%v, err=%v)", changed, err)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "demo.go", filtered, parser.ParseComments); err != nil {
		t.Fatalf("filtered file failed to parse: %v\n%s", err, filtered)
	}
	text := string(filtered)
	if !strings.Contains(text, "KeepType") || !strings.Contains(text, "KeepVar") {
		t.Fatalf("kept declarations disappeared:\n%s", text)
	}
}

func TestApplyForPkgSkipAndSkipAll(t *testing.T) {
	tmp := t.TempDir()
	goroot := filepath.Join(tmp, "goroot")
	rtDir := filepath.Join(tmp, "runtime")

	origPkgDir := filepath.Join(goroot, "src", "mypkg")
	patchDir := filepath.Join(rtDir, "_patch", "mypkg")
	if err := os.MkdirAll(origPkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(patchDir, 0o755); err != nil {
		t.Fatal(err)
	}
	origFile := filepath.Join(origPkgDir, "a.go")
	if err := os.WriteFile(origFile, []byte("package mypkg\n\nfunc Foo() int { return 1 }\nfunc Bar() int { return 2 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	patchFile := filepath.Join(patchDir, "patch.go")
	if err := os.WriteFile(patchFile, []byte("//llgo:skip Foo\npackage mypkg\n\nfunc Foo() int { return 42 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, out, files, err := ApplyForPkg(nil, nil, rtDir, goroot, "mypkg", BuildContext{GOOS: "linux", GOARCH: "amd64", GoVersion: "go1.24"})
	if err != nil {
		t.Fatalf("ApplyForPkg() error: %v", err)
	}
	if !changed || len(files) != 1 || out[origFile] == nil {
		t.Fatalf("unexpected ApplyForPkg result: changed=%v files=%v hasOrig=%v", changed, files, out[origFile] != nil)
	}
	if strings.Contains(string(out[patchFile]), "//llgo:skip") {
		t.Fatalf("injected patch still contains directive:\n%s", out[patchFile])
	}

	// Nonexistent patch package returns changed=false cleanly.
	changed, _, _, err = ApplyForPkg(nil, nil, rtDir, goroot, "no/such/pkg", BuildContext{GOOS: "linux", GOARCH: "amd64"})
	if err != nil || changed {
		t.Fatalf("ApplyForPkg(missing) = (%v, %v), want (false, nil)", changed, err)
	}

	// Exercise //llgo:skipall path.
	if err := os.WriteFile(patchFile, []byte("//llgo:skipall\npackage mypkg\n\nfunc Foo() int { return 99 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	changed, out, _, err = ApplyForPkg(nil, nil, rtDir, goroot, "mypkg", BuildContext{GOOS: "linux", GOARCH: "amd64", GoVersion: "go1.24"})
	if err != nil || !changed {
		t.Fatalf("ApplyForPkg(skipall) = (%v, %v), want (true, nil)", changed, err)
	}
	if strings.TrimSpace(string(out[origFile])) != "package mypkg" {
		t.Fatalf("skipall stub = %q, want %q", string(out[origFile]), "package mypkg")
	}
}
