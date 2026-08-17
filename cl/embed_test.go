//go:build !llgo
// +build !llgo

package cl

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/goplus/gogen/packages"
	"github.com/xgo-dev/llgo/internal/goembed"
	llssa "github.com/xgo-dev/llgo/ssa"
	gossa "golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func mustPanicContains(t *testing.T, want string, fn func()) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic containing %q", want)
		}
		msg := fmt.Sprint(r)
		if !strings.Contains(msg, want) {
			t.Fatalf("panic = %q, want %q", msg, want)
		}
	}()
	fn()
}

func TestParseEmbedPatterns(t *testing.T) {
	doc := &ast.CommentGroup{
		List: []*ast.Comment{
			{Text: "// comment"},
			{Text: "//go:embed testdata/hello.txt assets/*.json"},
			{Text: "//go:embed \"space name.txt\""},
		},
	}
	got, has, err := goembed.ParsePatterns(doc)
	if err != nil {
		t.Fatalf("parseEmbedPatterns error: %v", err)
	}
	if !has {
		t.Fatalf("parseEmbedPatterns did not detect directive")
	}
	want := []string{"testdata/hello.txt", "assets/*.json", "space name.txt"}
	if len(got) != len(want) {
		t.Fatalf("pattern count = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("pattern[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestParseEmbedPatterns_InvalidQuoted(t *testing.T) {
	doc := &ast.CommentGroup{
		List: []*ast.Comment{
			{Text: "//go:embed \"unclosed.txt"},
		},
	}
	_, has, err := goembed.ParsePatterns(doc)
	if !has {
		t.Fatalf("parseEmbedPatterns should detect directive")
	}
	if err == nil {
		t.Fatalf("parseEmbedPatterns should fail for invalid quoted pattern")
	}
}

func TestFileImportsEmbed(t *testing.T) {
	file := &ast.File{
		Imports: []*ast.ImportSpec{
			{Path: &ast.BasicLit{Value: `"fmt"`}},
			{Path: &ast.BasicLit{Value: `"embed"`}},
		},
	}
	if !goembed.FileImportsEmbed(file) {
		t.Fatalf("fileImportsEmbed = false, want true")
	}
}

func TestResolveEmbedPatterns_HiddenAndAll(t *testing.T) {
	dir := t.TempDir()
	mustWrite := func(rel, data string) {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	mustWrite("testdata/hello.txt", "Hello, World!")
	mustWrite("testdata/.hidden.txt", "hidden")
	mustWrite("testdata/sub/world.txt", "world")
	mustWrite("testdata/sub/.deep.txt", "deep")
	mustWrite("testdata/.hidden-dir/tip.txt", "tip")

	got, err := goembed.ResolvePatterns(dir, []string{"testdata"})
	if err != nil {
		t.Fatalf("resolveEmbedPatterns: %v", err)
	}
	seen := map[string]bool{}
	for _, f := range got {
		seen[f.Name] = true
	}
	if !seen["testdata/hello.txt"] {
		t.Fatalf("missing embedded file testdata/hello.txt: %+v", got)
	}
	if !seen["testdata/sub/world.txt"] {
		t.Fatalf("missing embedded file testdata/sub/world.txt: %+v", got)
	}
	if seen["testdata/.hidden.txt"] {
		t.Fatalf("unexpected hidden file in default mode: %+v", got)
	}
	if seen["testdata/sub/.deep.txt"] {
		t.Fatalf("unexpected hidden nested file in default mode: %+v", got)
	}
	if seen["testdata/.hidden-dir/tip.txt"] {
		t.Fatalf("unexpected hidden directory file in default mode: %+v", got)
	}

	gotAll, err := goembed.ResolvePatterns(dir, []string{"all:testdata"})
	if err != nil {
		t.Fatalf("resolveEmbedPatterns(all:): %v", err)
	}
	seenAll := map[string]bool{}
	for _, f := range gotAll {
		seenAll[f.Name] = true
	}
	if !seenAll["testdata/.hidden.txt"] {
		t.Fatalf("missing hidden file in all: mode: %+v", gotAll)
	}
	if !seenAll["testdata/sub/.deep.txt"] {
		t.Fatalf("missing hidden nested file in all: mode: %+v", gotAll)
	}
}

func TestResolveEmbedPatterns_Errors(t *testing.T) {
	dir := t.TempDir()
	mustWrite := func(rel, data string) {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	mustWrite("data/file.txt", "ok")
	if err := os.MkdirAll(filepath.Join(dir, "empty"), 0o755); err != nil {
		t.Fatalf("mkdir empty: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "submod"), 0o755); err != nil {
		t.Fatalf("mkdir submod: %v", err)
	}
	mustWrite("submod/go.mod", "module x\n")
	mustWrite("submod/a.txt", "submodule")

	cases := []struct {
		pattern string
		wantErr string
	}{
		{pattern: ".", wantErr: "invalid pattern syntax"},
		{pattern: "no_such_*.txt", wantErr: "no matching files found"},
		{pattern: "empty", wantErr: "contains no embeddable files"},
		{pattern: "submod", wantErr: "in different module"},
	}

	for _, tc := range cases {
		_, err := goembed.ResolvePatterns(dir, []string{tc.pattern})
		if err == nil {
			t.Fatalf("resolveEmbedPatterns(%q) should fail", tc.pattern)
		}
		if !strings.Contains(err.Error(), tc.wantErr) {
			t.Fatalf("resolveEmbedPatterns(%q) error = %v, want substring %q", tc.pattern, err, tc.wantErr)
		}
	}
}

func TestResolveEmbedPatterns_RejectIrregularAndInvalidName(t *testing.T) {
	dir := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Skip("file name with ':' is not portable on windows")
	}

	mustWrite := func(rel, data string) {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	mustWrite("bad:name.txt", "bad")
	_, err := goembed.ResolvePatterns(dir, []string{"bad:name.txt"})
	if err == nil || !strings.Contains(err.Error(), "invalid name") {
		t.Fatalf("resolveEmbedPatterns invalid-name error = %v, want invalid name", err)
	}

	mustWrite("target.txt", "ok")
	if err := os.Symlink(filepath.Join(dir, "target.txt"), filepath.Join(dir, "link.txt")); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}
	_, err = goembed.ResolvePatterns(dir, []string{"link.txt"})
	if err == nil || !strings.Contains(err.Error(), "irregular file") {
		t.Fatalf("resolveEmbedPatterns irregular-file error = %v, want irregular file", err)
	}
}

func TestBuildEmbedFSEntries(t *testing.T) {
	files := []goembed.FileData{
		{Name: "testdata/hello.txt", Data: []byte("hello")},
		{Name: "assets/static/app.js", Data: []byte("app")},
	}
	got := goembed.BuildFSEntries(files)
	seen := map[string]bool{}
	for _, e := range got {
		seen[e.Name] = true
	}
	for _, name := range []string{
		"assets/",
		"assets/static/",
		"assets/static/app.js",
		"testdata/",
		"testdata/hello.txt",
	} {
		if !seen[name] {
			t.Fatalf("missing entry %q in %+v", name, got)
		}
	}

	// root directories should be listed before child files in split(dir, elem) order.
	pos := map[string]int{}
	for i, e := range got {
		pos[e.Name] = i
	}
	if pos["testdata/"] >= pos["testdata/hello.txt"] {
		t.Fatalf("directory entry should precede file entry: %+v", got)
	}
}

func TestLoadEmbedDirectives(t *testing.T) {
	dir := t.TempDir()
	mainFile := filepath.Join(dir, "main.go")
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write hello.txt: %v", err)
	}

	src := `package foo

import "embed"

//go:embed hello.txt
var content string
`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, mainFile, src, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	embedMap, err := goembed.LoadDirectives(fset, []*ast.File{f})
	if err != nil {
		t.Fatalf("LoadDirectives: %v", err)
	}
	info, ok := embedMap["content"]
	if !ok {
		t.Fatalf("missing embed var content: %+v", embedMap)
	}
	if len(info.Files) != 1 || info.Files[0].Name != "hello.txt" {
		t.Fatalf("unexpected files: %+v", info.Files)
	}
}

func TestLoadEmbedDirectives_EarlyReturnsAndSkips(t *testing.T) {
	fset := token.NewFileSet()
	embedMap, err := goembed.LoadDirectives(fset, nil)
	if err != nil {
		t.Fatalf("LoadDirectives(nil): %v", err)
	}
	if len(embedMap) != 0 {
		t.Fatalf("embedMap should be empty for nil input")
	}

	f, err := parser.ParseFile(fset, "", `package foo`, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	embedMap, err = goembed.LoadDirectives(fset, []*ast.File{f})
	if err != nil {
		t.Fatalf("LoadDirectives: %v", err)
	}
	if len(embedMap) != 0 {
		t.Fatalf("embedMap should stay empty for files without filename")
	}
}

func TestLoadEmbedDirectives_Panics(t *testing.T) {
	makeFile := func(t *testing.T, src string, extras map[string]string) (*token.FileSet, []*ast.File) {
		t.Helper()
		dir := t.TempDir()
		mainFile := filepath.Join(dir, "main.go")
		for rel, data := range extras {
			full := filepath.Join(dir, rel)
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				t.Fatalf("mkdir %s: %v", rel, err)
			}
			if err := os.WriteFile(full, []byte(data), 0o644); err != nil {
				t.Fatalf("write %s: %v", rel, err)
			}
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, mainFile, src, parser.ParseComments)
		if err != nil {
			t.Fatalf("ParseFile: %v", err)
		}
		return fset, []*ast.File{f}
	}

	fsetMissingImport, filesMissingImport := makeFile(t, `package foo

//go:embed hello.txt
var s string
	`, map[string]string{"hello.txt": "hi"})
	_, err := goembed.LoadDirectives(fsetMissingImport, filesMissingImport)
	if err == nil || !strings.Contains(err.Error(), `import "embed"`) {
		t.Fatalf("LoadDirectives missing-import error = %v", err)
	}

	fsetNoMatch, filesNoMatch := makeFile(t, `package foo
import "embed"

//go:embed no_such_file.txt
var s string
	`, nil)
	_, err = goembed.LoadDirectives(fsetNoMatch, filesNoMatch)
	if err == nil || !strings.Contains(err.Error(), "no matching files found") {
		t.Fatalf("LoadDirectives no-match error = %v", err)
	}

	fsetInvalid, filesInvalid := makeFile(t, "package foo\nimport \"embed\"\n//go:embed \"bad\nvar s string\n", nil)
	_, err = goembed.LoadDirectives(fsetInvalid, filesInvalid)
	if err == nil || !strings.Contains(err.Error(), "invalid //go:embed quoted pattern") {
		t.Fatalf("LoadDirectives invalid-pattern error = %v", err)
	}
}

func TestLoadEmbedDirectives_SkipBranches(t *testing.T) {
	dir := t.TempDir()
	mainFile := filepath.Join(dir, "main.go")
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write hello.txt: %v", err)
	}

	src := `package foo

import "embed"

var plain string

	var a, b string

	var (
		//go:embed hello.txt
		c string
		d string
	)
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, mainFile, src, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	embedMap, err := goembed.LoadDirectives(fset, []*ast.File{f})
	if err != nil {
		t.Fatalf("LoadDirectives: %v", err)
	}
	if len(embedMap) != 1 || len(embedMap["c"].Files) != 1 {
		t.Fatalf("embedMap should load grouped single-name declaration: %+v", embedMap)
	}
	if _, ok := embedMap["a"]; ok {
		t.Fatalf("multi-name declaration without directive should be skipped: %+v", embedMap)
	}
}

func TestEmbedTypeChecks(t *testing.T) {
	if !isStringType(types.Typ[types.String]) {
		t.Fatalf("isStringType(string) = false, want true")
	}
	typeName := types.NewTypeName(token.NoPos, nil, "MyString", nil)
	if !isStringType(types.NewNamed(typeName, types.Typ[types.String], nil)) {
		t.Fatalf("isStringType(named string) = false, want true")
	}
	if isStringType(types.Typ[types.Int]) {
		t.Fatalf("isStringType(int) = true, want false")
	}

	if !isByteSliceType(types.NewSlice(types.Typ[types.Byte])) {
		t.Fatalf("isByteSliceType([]byte) = false, want true")
	}
	byteAlias := types.NewTypeName(token.NoPos, nil, "MyByte", nil)
	if !isByteSliceType(types.NewSlice(types.NewNamed(byteAlias, types.Typ[types.Byte], nil))) {
		t.Fatalf("isByteSliceType([]MyByte) = false, want true")
	}
	if isByteSliceType(types.NewSlice(types.Typ[types.Int])) {
		t.Fatalf("isByteSliceType([]int) = true, want false")
	}

	embedPkg := types.NewPackage("embed", "embed")
	fsObj := types.NewTypeName(token.NoPos, embedPkg, "FS", nil)
	fsType := types.NewNamed(fsObj, types.NewStruct(nil, nil), nil)
	if !isEmbedFSType(fsType) {
		t.Fatalf("isEmbedFSType(embed.FS) = false, want true")
	}
	if isEmbedFSType(types.Typ[types.String]) {
		t.Fatalf("isEmbedFSType(string) = true, want false")
	}
}

func TestApplyEmbedInits_NoPending(t *testing.T) {
	p := &context{}
	p.applyEmbedInits(nil)
}

func TestParseEmbedDirective(t *testing.T) {
	tests := []struct {
		line     string
		wantArgs string
		wantOK   bool
	}{
		{line: "go:embed", wantArgs: "", wantOK: true},
		{line: "go:embed\tfoo.txt", wantArgs: "foo.txt", wantOK: true},
		{line: "go:embedfoo.txt", wantArgs: "", wantOK: false},
		{line: "xxgo:embed foo.txt", wantArgs: "", wantOK: false},
	}
	for _, tc := range tests {
		args, ok := goembed.ParseDirective(tc.line)
		if ok != tc.wantOK || args != tc.wantArgs {
			t.Fatalf("parseEmbedDirective(%q) = (%q,%v), want (%q,%v)", tc.line, args, ok, tc.wantArgs, tc.wantOK)
		}
	}
}

func TestSplitEmbedArgs_TrailingWhitespace(t *testing.T) {
	got, err := goembed.SplitArgs("a\t ")
	if err != nil {
		t.Fatalf("splitEmbedArgs error: %v", err)
	}
	if len(got) != 1 || got[0] != "a" {
		t.Fatalf("splitEmbedArgs result = %v, want [a]", got)
	}
}

func TestParseEmbedPatterns_ExtraBranches(t *testing.T) {
	docWithNil := &ast.CommentGroup{
		List: []*ast.Comment{
			nil,
			{Text: "//go:embed plain.txt"},
		},
	}
	patterns, has, err := goembed.ParsePatterns(docWithNil)
	if err != nil || !has || len(patterns) != 1 || patterns[0] != "plain.txt" {
		t.Fatalf("parseEmbedPatterns nil-comment branch failed: patterns=%v has=%v err=%v", patterns, has, err)
	}

	docMissing := &ast.CommentGroup{
		List: []*ast.Comment{{Text: "//go:embed"}},
	}
	_, has, err = goembed.ParsePatterns(docMissing)
	if !has || err == nil || !strings.Contains(err.Error(), "missing pattern") {
		t.Fatalf("missing pattern not rejected, has=%v err=%v", has, err)
	}

	docBadQuoted := &ast.CommentGroup{
		List: []*ast.Comment{{Text: "//go:embed \"\\xZZ\""}},
	}
	_, has, err = goembed.ParsePatterns(docBadQuoted)
	if !has || err == nil || !strings.Contains(err.Error(), "invalid //go:embed quoted pattern") {
		t.Fatalf("bad quoted pattern not rejected, has=%v err=%v", has, err)
	}
}

func TestFileImportsEmbed_NegativeBranches(t *testing.T) {
	file := &ast.File{
		Imports: []*ast.ImportSpec{
			nil,
			{Path: nil},
			{Path: &ast.BasicLit{Value: `bad`}},
			{Path: &ast.BasicLit{Value: `"fmt"`}},
		},
	}
	if goembed.FileImportsEmbed(file) {
		t.Fatalf("fileImportsEmbed should be false for invalid/non-embed imports")
	}
}

func TestValidEmbedPattern(t *testing.T) {
	tests := []struct {
		pattern string
		want    bool
	}{
		{pattern: ".", want: false},
		{pattern: "ok/file.txt", want: true},
		{pattern: "vendor/file.txt", want: true},
		{pattern: "a/vendor/file.txt", want: true},
	}
	for _, tc := range tests {
		if got := goembed.ValidPattern(tc.pattern); got != tc.want {
			t.Fatalf("validEmbedPattern(%q) = %v, want %v", tc.pattern, got, tc.want)
		}
	}
}

func TestLoadEmbedDirectives_MultiNameVarDirectivePanics(t *testing.T) {
	dir := t.TempDir()
	mainFile := filepath.Join(dir, "main.go")
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write hello.txt: %v", err)
	}
	src := `package foo

import "embed"

//go:embed hello.txt
var a, b string
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, mainFile, src, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	_, err = goembed.LoadDirectives(fset, []*ast.File{f})
	if err == nil || !strings.Contains(err.Error(), "go:embed cannot apply to multiple vars") {
		t.Fatalf("LoadDirectives multi-name error = %v", err)
	}
}

func TestLoadEmbedDirectives_GroupLevelDirectiveMisplaced(t *testing.T) {
	dir := t.TempDir()
	mainFile := filepath.Join(dir, "main.go")
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write hello.txt: %v", err)
	}
	src := `package foo

import "embed"

//go:embed hello.txt
var (
	a string
	b string
)
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, mainFile, src, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	_, err = goembed.LoadDirectives(fset, []*ast.File{f})
	if err == nil || !strings.Contains(err.Error(), "misplaced go:embed directive") {
		t.Fatalf("LoadDirectives misplaced-directive error = %v", err)
	}
}

func TestEmbedRelPath_Outside(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Dir(dir)
	_, err := goembed.RelPath(dir, outside)
	if err == nil || !strings.Contains(err.Error(), "outside package directory") {
		t.Fatalf("embedRelPath outside error = %v", err)
	}
}

func TestCheckEmbedPath_ErrorBranches(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := goembed.CheckPath(dir, filepath.Join(dir, "missing.txt"), map[string]bool{}); err == nil {
		t.Fatalf("checkEmbedPath should fail for missing file")
	}

	outDir := t.TempDir()
	outFile := filepath.Join(outDir, "out.txt")
	if err := os.WriteFile(outFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("write outFile: %v", err)
	}
	if _, _, err := goembed.CheckPath(dir, outFile, map[string]bool{}); err == nil || !strings.Contains(err.Error(), "outside package directory") {
		t.Fatalf("checkEmbedPath outside error = %v", err)
	}

	inside := filepath.Join(dir, ".git", "sub", "x.txt")
	if err := os.MkdirAll(filepath.Dir(inside), 0o755); err != nil {
		t.Fatalf("mkdir .git/sub: %v", err)
	}
	if err := os.WriteFile(inside, []byte("x"), 0o644); err != nil {
		t.Fatalf("write .git/sub/x.txt: %v", err)
	}
	if _, _, err := goembed.CheckPath(dir, inside, map[string]bool{}); err == nil || !strings.Contains(err.Error(), "in invalid directory .git") {
		t.Fatalf("checkEmbedPath invalid dir error = %v", err)
	}
}

func TestResolveEmbedPatterns_DuplicateAndReadFailure(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "dup.txt")
	if err := os.WriteFile(file, []byte("dup"), 0o644); err != nil {
		t.Fatalf("write dup.txt: %v", err)
	}
	got, err := goembed.ResolvePatterns(dir, []string{"dup.txt", "dup.txt"})
	if err != nil {
		t.Fatalf("resolveEmbedPatterns duplicate pattern failed: %v", err)
	}
	if len(got) != 1 || got[0].Name != "dup.txt" {
		t.Fatalf("resolveEmbedPatterns duplicate pattern got: %+v", got)
	}

	if runtime.GOOS == "windows" {
		return
	}
	noRead := filepath.Join(dir, "noread.txt")
	if err := os.WriteFile(noRead, []byte("secret"), 0o600); err != nil {
		t.Fatalf("write noread.txt: %v", err)
	}
	if err := os.Chmod(noRead, 0); err != nil {
		t.Fatalf("chmod noread.txt: %v", err)
	}
	defer os.Chmod(noRead, 0o600)
	if _, err := goembed.ResolvePatterns(dir, []string{"noread.txt"}); err == nil {
		t.Fatalf("resolveEmbedPatterns should fail for unreadable file")
	}
}

func TestResolveEmbedPatterns_WalkDirError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permissions behavior differs on windows")
	}
	dir := t.TempDir()
	root := filepath.Join(dir, "root")
	locked := filepath.Join(root, "locked")
	if err := os.MkdirAll(filepath.Join(locked, "sub"), 0o755); err != nil {
		t.Fatalf("mkdir root/locked/sub: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "ok.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatalf("write root/ok.txt: %v", err)
	}
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatalf("chmod locked: %v", err)
	}
	defer os.Chmod(locked, 0o755)

	if _, err := goembed.ResolvePatterns(dir, []string{"root"}); err == nil {
		t.Fatalf("resolveEmbedPatterns should fail when walk cannot read directory")
	}
}

func TestResolveEmbedPatterns_SkipNestedModuleAndIrregularInWalk(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink behavior differs on windows")
	}
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "tree", "submod"), 0o755); err != nil {
		t.Fatalf("mkdir tree/submod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tree", "go.txt"), []byte("go"), 0o644); err != nil {
		t.Fatalf("write tree/go.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tree", "submod", "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatalf("write tree/submod/go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tree", "submod", "hidden.txt"), []byte("skip"), 0o644); err != nil {
		t.Fatalf("write tree/submod/hidden.txt: %v", err)
	}
	if err := os.Symlink(filepath.Join(dir, "tree", "go.txt"), filepath.Join(dir, "tree", "link")); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	got, err := goembed.ResolvePatterns(dir, []string{"tree"})
	if err != nil {
		t.Fatalf("resolveEmbedPatterns(tree): %v", err)
	}
	seen := map[string]bool{}
	for _, item := range got {
		seen[item.Name] = true
	}
	if !seen["tree/go.txt"] {
		t.Fatalf("expected tree/go.txt in result: %+v", got)
	}
	if seen["tree/submod/hidden.txt"] {
		t.Fatalf("submodule file should be skipped: %+v", got)
	}
	if seen["tree/link"] {
		t.Fatalf("irregular symlink should be skipped in walk: %+v", got)
	}
}

func TestIsBadEmbedName_Reserved(t *testing.T) {
	for _, name := range []string{".bzr", ".git", ".hg", ".svn"} {
		if !goembed.IsBadName(name) {
			t.Fatalf("isBadEmbedName(%q) = false, want true", name)
		}
	}
}

func TestIsEmbedFSType_NoPackage(t *testing.T) {
	obj := types.NewTypeName(token.NoPos, nil, "FS", nil)
	typ := types.NewNamed(obj, types.NewStruct(nil, nil), nil)
	if isEmbedFSType(typ) {
		t.Fatalf("isEmbedFSType should be false for named type without package")
	}
}

func TestTryEmbedGlobalInit_EarlyAndDefaultBranches(t *testing.T) {
	global := buildTestSSAGlobal(t, "Num", `package foo; var Num int`)
	p := &context{}
	if got := p.tryEmbedGlobalInit(nil, global, nil, "foo.Num"); got {
		t.Fatalf("tryEmbedGlobalInit should return false when embedMap is empty")
	}

	p.embedMap = goembed.VarMap{
		"Other": {Files: []goembed.FileData{{Name: "a.txt", Data: []byte("a")}}},
	}
	if got := p.tryEmbedGlobalInit(nil, global, nil, "foo.Num"); got {
		t.Fatalf("tryEmbedGlobalInit should return false for missing variable entry")
	}

	p.embedMap = goembed.VarMap{
		"Num": {Files: []goembed.FileData{{Name: "a.txt", Data: []byte("a")}}},
	}
	mustPanicContains(t, "go:embed cannot apply to var of type int", func() {
		_ = p.tryEmbedGlobalInit(nil, global, nil, "foo.Num")
	})
}

func TestApplyEmbedInits_SortAndMissingGlobal(t *testing.T) {
	prog := llssa.NewProgram(nil)
	pkg := prog.NewPackage("foo", "foo")
	p := &context{
		pkg: pkg,
		embedInits: []embedInit{
			{globalName: "z", kind: embedInitFS},
			{globalName: "a", kind: 999},
		},
	}
	p.applyEmbedInits(nil)
	if p.embedInits != nil {
		t.Fatalf("applyEmbedInits should clear pending inits")
	}
}

func buildTestSSAGlobal(t *testing.T, name, src string) *gossa.Global {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "main.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	imp := packages.NewImporter(fset)
	pkg := types.NewPackage("foo", "foo")
	mode := gossa.SanityCheckFunctions | gossa.InstantiateGenerics
	ssaPkg, _, err := ssautil.BuildPackage(&types.Config{Importer: imp}, fset, pkg, []*ast.File{f}, mode)
	if err != nil {
		t.Fatalf("BuildPackage: %v", err)
	}
	member, ok := ssaPkg.Members[name]
	if !ok {
		t.Fatalf("missing global %q in ssa package", name)
	}
	global, ok := member.(*gossa.Global)
	if !ok {
		t.Fatalf("member %q is %T, want *ssa.Global", name, member)
	}
	return global
}
