package directive

import (
	"go/ast"
	"reflect"
	"strings"
	"testing"
)

func TestEmbedRecordPatternsAndDeferredErrors(t *testing.T) {
	for _, tt := range []struct {
		line     string
		present  bool
		patterns []string
		err      string
	}{
		{`//go:embed plain "two words" ` + "`raw path`", true, []string{"plain", "two words", "raw path"}, ""},
		{`// go:embed "a\"b"`, true, []string{`a"b`}, ""},
		{"//go:embed", true, nil, "missing pattern"},
		{`//go:embed "unterminated`, true, nil, "quoted pattern"},
		{`//go:embed "bad\q"`, true, nil, "quoted pattern"},
		{"//go:embedx other", false, nil, ""},
		{"// ordinary", false, nil, ""},
	} {
		t.Run(tt.line, func(t *testing.T) {
			store := new(Store)
			doc := &ast.CommentGroup{List: []*ast.Comment{nil, {Text: tt.line}}}
			record := store.Group(doc)
			doc.List = nil
			store.Freeze()
			patterns, present, err := EmbedPatterns(store.Group(nil), record)
			if present != tt.present || !reflect.DeepEqual(patterns, tt.patterns) {
				t.Fatalf("patterns = %q, %v", patterns, present)
			}
			if tt.err == "" && err != nil || tt.err != "" && (err == nil || !strings.Contains(err.Error(), tt.err)) {
				t.Fatalf("error = %v, want %q", err, tt.err)
			}
		})
	}
	store := new(Store)
	a := store.Group(&ast.CommentGroup{List: []*ast.Comment{{Text: "//go:embed first"}}})
	b := store.Group(&ast.CommentGroup{List: []*ast.Comment{{Text: "//go:embed second"}}})
	if p, has, err := EmbedPatterns(a, b); err != nil || !has || !reflect.DeepEqual(p, []string{"first", "second"}) {
		t.Fatalf("combined patterns = %v, %v, %v", p, has, err)
	}
	if p, has, err := parseEmbedPatterns(nil); err != nil || has || len(p) != 0 {
		t.Fatal("nil document produced an embed directive")
	}
	if args, err := SplitEmbedArgs(" \tfoo  \t"); err != nil || !reflect.DeepEqual(args, []string{"foo"}) {
		t.Fatalf("trailing whitespace = %q, %v", args, err)
	}
	for _, comment := range []*ast.Comment{nil, {Text: "/* go:embed data */"}} {
		if IsEmbedComment(comment) {
			t.Fatal("loader recognized a non-line comment")
		}
	}
}
