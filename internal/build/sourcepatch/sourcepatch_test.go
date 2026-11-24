package sourcepatch

import (
	"go/parser"
	"go/token"
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
