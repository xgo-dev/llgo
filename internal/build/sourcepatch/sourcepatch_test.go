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

func TestSourcePatchFullCoverageEdges(t *testing.T) {
	// ReleaseTagsForGoVersion error branches.
	for _, bad := range []string{"bad", "x.2", "1.y", "2.1", "1.0"} {
		if _, err := ReleaseTagsForGoVersion(bad); err == nil {
			t.Fatalf("ReleaseTagsForGoVersion(%q) succeeded, want error", bad)
		}
	}

	// BuildInjectedFile with spaced directive, no directives, and CRLF.
	if got := string(BuildInjectedFile("/tmp/a.go", []byte("// llgo:skip Foo\r\npackage p\r\n"))); !strings.HasPrefix(got, "//line /tmp/a.go:1\r\n") || strings.Contains(got, "// llgo:skip") {
		t.Fatalf("BuildInjectedFile(spaced+CRLF) = %q", got)
	}
	if got := string(BuildInjectedFile("/tmp/plain.go", []byte("package p\n"))); !strings.HasSuffix(got, "package p\n") {
		t.Fatalf("BuildInjectedFile(plain) = %q", got)
	}

	// FilterFile edge cases: unchanged, parse error, blank import, single removed import,
	// all-kept imports, mixed multi-name ValueSpec, untouched GenDecl, blank line inside removed func.
	if _, _, err := FilterFile([]byte("package {"), map[string]struct{}{"A": {}}); err == nil {
		t.Fatal("FilterFile accepted invalid syntax")
	}
	unchangedSrc := []byte("package p\n\nvar Keep = 1\n")
	if _, changed, err := FilterFile(unchangedSrc, map[string]struct{}{"Missing": {}}); err != nil || changed {
		t.Fatalf("FilterFile(unchanged) = (changed=%v, err=%v)", changed, err)
	}
	complexSrc := []byte(`package p

import "fmt"
import (
	_ "unsafe"
	"strings"
)

var Untouched = 1
var DropA, KeepB = 1, 2

func DropWithBlank() {
	fmt.Println("hi")

	_ = strings.TrimSpace
}

func Keep() string {
	return strings.TrimSpace("x")
}
`)
	filtered, changed, err := FilterFile(complexSrc, map[string]struct{}{"DropA": {}, "DropWithBlank": {}})
	if err != nil || !changed {
		t.Fatalf("FilterFile(complex) = (changed=%v, err=%v)", changed, err)
	}
	text := string(filtered)
	if !strings.Contains(text, `// import "fmt"`) || strings.Contains(text, `// _ "unsafe"`) || !strings.Contains(text, "// var DropA, KeepB") {
		t.Fatalf("unexpected FilterFile(complex) output:\n%s", text)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "complex.go", filtered, parser.ParseComments); err != nil {
		t.Fatalf("FilterFile(complex) unparseable: %v\n%s", err, text)
	}

	// Internal AST/span helper edge cases.
	fset := token.NewFileSet()
	tokFile := fset.AddFile("x.go", -1, 10)
	if got, removed, _, err := filterDecl(tokFile, &ast.BadDecl{}, nil); err != nil || removed || got == nil {
		t.Fatalf("filterDecl(BadDecl) = (%v, %v, %v)", got, removed, err)
	}
	if got, removed, _ := filterSpec(tokFile, &ast.ImportSpec{}, nil); removed || got == nil {
		t.Fatalf("filterSpec(ImportSpec) = (%v, %v)", got, removed)
	}
	if spans := commentGroupsToSpans(tokFile, []*ast.CommentGroup{nil}); len(spans) != 0 {
		t.Fatalf("commentGroupsToSpans([nil]) = %v", spans)
	}
	if got := collectSpecComments(&ast.ImportSpec{}); got != nil {
		t.Fatalf("collectSpecComments(ImportSpec) = %v", got)
	}
	if got := string(commentSpans([]byte("abc"), nil)); got != "abc" {
		t.Fatalf("commentSpans(nil) = %q", got)
	}
	// Exercise span sorting (same start, different end, equal), clamping, empty, and overlapping merge.
	gotCommented := string(commentSpans([]byte("line1\nline2\n"), []byteSpan{
		{start: 0, end: 5},
		{start: -2, end: 3},
		{start: 0, end: 5},
		{start: 0, end: 8},
		{start: 4, end: 4},
		{start: 2, end: 99},
	}))
	if !strings.Contains(gotCommented, "// line1") || !strings.Contains(gotCommented, "// line2") {
		t.Fatalf("commentSpans(merged) = %q", gotCommented)
	}

	// RecvKey panic on unsupported expression.
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("RecvKey(BasicLit) did not panic")
			}
		}()
		_ = RecvKey(&ast.BasicLit{})
	}()

	// packageStubSource error when package clause is absent.
	if _, err := packageStubSource([]byte("// no package clause\n")); err == nil {
		t.Fatal("packageStubSource succeeded without package clause")
	}

	// ApplyForPkg & BuildOverlayForGOROOT filesystem, asm replacement, Go 1.23 path mapping, and error paths.
	tmp := t.TempDir()
	goroot := filepath.Join(tmp, "goroot")
	rtDir := filepath.Join(tmp, "runtime")

	// Empty runtime/goroot: BuildOverlayForGOROOT returns base unchanged.
	base := map[string][]byte{"existing.go": []byte("package e\n")}
	gotOverlay, gotFiles, err := BuildOverlayForGOROOT(base, rtDir, goroot, BuildContext{GOOS: "linux", GOARCH: "amd64", GoVersion: "go1.24"})
	if err != nil || gotFiles != nil || string(gotOverlay["existing.go"]) != "package e\n" {
		t.Fatalf("BuildOverlayForGOROOT(empty) = (%v, %v, %v)", gotOverlay, gotFiles, err)
	}

	// Set up internal/bytealg (with asm replacement + baremetal tag), runtime/internal/atomic,
	// runtime/internal/syscall, plus subdirs, non-go/test files, non-matching tags, and base/current overlays.
	bytealgSrc := filepath.Join(goroot, "src", "internal", "bytealg")
	bytealgPatch := filepath.Join(rtDir, "_patch", "internal", "bytealg")
	for _, dir := range []string{
		bytealgSrc,
		filepath.Join(bytealgSrc, "subdir"),
		bytealgPatch,
		filepath.Join(bytealgPatch, "subdir"),
		filepath.Join(goroot, "src", "runtime", "internal", "atomic"),
		filepath.Join(rtDir, "_patch", "internal", "runtime", "atomic"),
		filepath.Join(goroot, "src", "runtime", "internal", "syscall"),
		filepath.Join(rtDir, "_patch", "internal", "runtime", "syscall"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite := func(path, content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite(filepath.Join(bytealgSrc, "index.go"), "package bytealg\n\nfunc Index() int { return 0 }\n")
	mustWrite(filepath.Join(bytealgSrc, "other.go"), "//go:build customtag\n\npackage bytealg\n\n// comment mentions Index\nfunc KeepOther() {}\n")
	mustWrite(filepath.Join(bytealgSrc, "unrelated.go"), "package bytealg\n\nfunc Unrelated() {}\n")
	mustWrite(filepath.Join(bytealgSrc, "index_test.go"), "package bytealg\n")
	mustWrite(filepath.Join(bytealgSrc, "index_amd64.s"), "TEXT ·Index(SB),0,$0\n")
	mustWrite(filepath.Join(bytealgSrc, "index_arm64.s"), "TEXT ·Index(SB),0,$0\n")
	mustWrite(filepath.Join(bytealgSrc, "index_test.s"), "TEXT ·Test(SB),0,$0\n")
	mustWrite(filepath.Join(bytealgPatch, "README.txt"), "ignore\n")
	mustWrite(filepath.Join(bytealgPatch, "only_windows.go"), "//go:build windows\n\npackage bytealg\n")
	mustWrite(filepath.Join(bytealgPatch, "patch.go"), "// plain comment\n//llgo:skip Index\npackage bytealg\n\nfunc Index() int { return 1 }\n")

	mustWrite(filepath.Join(goroot, "src", "runtime", "internal", "atomic", "a.go"), "package atomic\n\nfunc Load() {}\n")
	mustWrite(filepath.Join(rtDir, "_patch", "internal", "runtime", "atomic", "patch.go"), "//llgo:skipall\npackage atomic\n")
	mustWrite(filepath.Join(goroot, "src", "runtime", "internal", "syscall", "s.go"), "package syscall\n")
	mustWrite(filepath.Join(rtDir, "_patch", "internal", "runtime", "syscall", "only_windows.go"), "//go:build windows\n\npackage syscall\n")

	ctx := BuildContext{GOOS: "linux", GOARCH: "amd64", GoVersion: "go1.24", TargetTags: []string{"baremetal"}}
	gotOverlay, gotFiles, err = BuildOverlayForGOROOT(base, rtDir, goroot, ctx)
	if err != nil {
		t.Fatalf("BuildOverlayForGOROOT() error: %v", err)
	}
	if len(gotFiles["internal/bytealg"]) == 0 || gotOverlay[filepath.Join(bytealgSrc, "index_amd64.s")] == nil {
		t.Fatalf("expected asm and go patch for internal/bytealg: files=%v", gotFiles)
	}
	if _, replacedArm := gotOverlay[filepath.Join(bytealgSrc, "index_arm64.s")]; replacedArm {
		t.Fatal("non-matching arm64 asm should not be replaced on amd64")
	}

	// Exercise readOverlay hitting base and current maps, and runtime/internal/atomic + syscall mappings.
	cur := map[string][]byte{filepath.Join(goroot, "src", "runtime", "internal", "atomic", "a.go"): []byte("package atomic\n")}
	if changed, _, _, err := ApplyForPkg(base, cur, rtDir, goroot, "runtime/internal/atomic", ctx); err != nil || !changed {
		t.Fatalf("ApplyForPkg(runtime/internal/atomic) = (%v, %v)", changed, err)
	}
	if changed, _, _, err := ApplyForPkg(base, nil, rtDir, goroot, "runtime/internal/syscall", ctx); err != nil || changed {
		t.Fatalf("ApplyForPkg(runtime/internal/syscall unmatched) = (%v, %v)", changed, err)
	}
	// Missing stdlib srcDir when patchDir exists returns false, nil.
	if changed, _, _, err := ApplyForPkg(nil, nil, rtDir, filepath.Join(tmp, "empty-goroot"), "internal/bytealg", ctx); err != nil || changed {
		t.Fatalf("ApplyForPkg(missing srcDir) = (%v, %v)", changed, err)
	}
	// Base overlay read path for patch file.
	patchPath := filepath.Join(bytealgPatch, "patch.go")
	baseWithPatch := map[string][]byte{patchPath: []byte("//llgo:skip Index\npackage bytealg\nfunc Index() int { return 2 }\n")}
	if changed, _, _, err := ApplyForPkg(baseWithPatch, nil, rtDir, goroot, "internal/bytealg", ctx); err != nil || !changed {
		t.Fatalf("ApplyForPkg(baseWithPatch) = (%v, %v)", changed, err)
	}

	// Error branches in ApplyForPkg & BuildOverlayForGOROOT.
	if _, _, err := BuildOverlayForGOROOT(nil, rtDir, goroot, BuildContext{GoVersion: "bad"}); err == nil {
		t.Fatal("BuildOverlayForGOROOT accepted invalid GoVersion")
	}
	fileAsDir := filepath.Join(rtDir, "_patch", "notadir")
	mustWrite(fileAsDir, "x")
	if _, _, _, err := ApplyForPkg(nil, nil, rtDir, goroot, "notadir", ctx); err == nil {
		t.Fatal("ApplyForPkg accepted non-directory patchDir")
	}
	badSrcPatchDir := filepath.Join(rtDir, "_patch", "badsrc")
	if err := os.MkdirAll(badSrcPatchDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(filepath.Join(goroot, "src", "badsrc"), "not a dir")
	if _, _, _, err := ApplyForPkg(nil, nil, rtDir, goroot, "badsrc", ctx); err == nil {
		t.Fatal("ApplyForPkg accepted non-directory srcDir")
	}
	badSyntaxPkg := "badsyntax"
	for _, d := range []string{filepath.Join(goroot, "src", badSyntaxPkg), filepath.Join(rtDir, "_patch", badSyntaxPkg)} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite(filepath.Join(goroot, "src", badSyntaxPkg, "a.go"), "package badsyntax\n")
	mustWrite(filepath.Join(rtDir, "_patch", badSyntaxPkg, "p.go"), "package {")
	if _, _, _, err := ApplyForPkg(nil, nil, rtDir, goroot, badSyntaxPkg, ctx); err == nil {
		t.Fatal("ApplyForPkg accepted invalid patch syntax")
	}
	// Invalid stdlib file under skipAll (missing package clause) and under skips (syntax error).
	mustWrite(filepath.Join(rtDir, "_patch", badSyntaxPkg, "p.go"), "//llgo:skipall\npackage badsyntax\n")
	mustWrite(filepath.Join(goroot, "src", badSyntaxPkg, "a.go"), "// no package clause\n")
	mustWrite(filepath.Join(goroot, "src", badSyntaxPkg, "ignore_test.go"), "package badsyntax\n")
	if err := os.MkdirAll(filepath.Join(goroot, "src", badSyntaxPkg, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := ApplyForPkg(nil, nil, rtDir, goroot, badSyntaxPkg, ctx); err == nil {
		t.Fatal("ApplyForPkg(skipall) accepted file without package clause")
	}
	mustWrite(filepath.Join(rtDir, "_patch", badSyntaxPkg, "p.go"), "//llgo:skip Bad\npackage badsyntax\n")
	mustWrite(filepath.Join(goroot, "src", badSyntaxPkg, "a.go"), "package { // Bad\n")
	if _, _, _, err := ApplyForPkg(nil, nil, rtDir, goroot, badSyntaxPkg, ctx); err == nil {
		t.Fatal("ApplyForPkg(skip) accepted unparseable stdlib file")
	}
}
