package directive

import (
	"bytes"
	"go/ast"
	"reflect"
	"strings"
	"testing"
)

func TestPatchSanitizationPreservesSourceCoordinates(t *testing.T) {
	for _, newline := range []string{"\n", "\r\n"} {
		source := strings.Join([]string{"package p", "  //llgo:skip A\tB", "// llgo:skipall", "//llgo:env", "//go:noinline", "func f() {}"}, newline)
		original := []byte(source)
		got := SanitizeSourcePatchLines(original)
		want := strings.ReplaceAll(strings.ReplaceAll(source, "//llgo:skip", "//llgo_skip"), "// llgo:skip", "// llgo_skip")
		if string(got) != want || len(got) != len(original) || string(original) != source {
			t.Fatalf("sanitized source = %q", got)
		}
		if !bytes.Equal(SanitizeSourcePatchLines(got), got) {
			t.Fatal("sanitization is not idempotent")
		}
		_, file := parseSource(t, source)
		index := new(Index)
		snapshot := index.File(file)
		if !snapshot.PatchSkip.All || !reflect.DeepEqual(snapshot.PatchSkip.Names, []string{"A", "B"}) {
			t.Fatalf("patch commands = %+v", snapshot.PatchSkip)
		}
	}
}

func TestIndexAssociatesConstructedSyntax(t *testing.T) {
	doc := &ast.CommentGroup{List: []*ast.Comment{nil, {Text: "//llgointernal:tls"}, {Text: "//llgo:skipall"}}}
	file := &ast.File{Comments: []*ast.CommentGroup{nil, doc}}
	index := new(Index)
	record := index.File(file)
	if len(record.Internal) != 1 || record.Internal[0].Name != "llgointernal:tls" || !record.Group(doc).Has("llgo:skipall") || !record.Group(doc).Skip.All {
		t.Fatalf("file directives = %+v", record)
	}
	fn := &ast.FuncDecl{Doc: &ast.CommentGroup{List: []*ast.Comment{{Text: "//go:noinline"}}}}
	if !index.Function(fn).NoInline {
		t.Fatal("standalone function not discovered")
	}
	fn.Doc = nil
	index.Freeze()
	if props, ok := index.LookupFunction(fn); !ok || !props.NoInline {
		t.Fatalf("prepared function = %+v, %v", props, ok)
	}
	if _, ok := index.LookupFunction(&ast.FuncDecl{}); ok {
		t.Fatal("unprepared function appeared in snapshot")
	}
	if index.Function(nil) != (Function{}) || index.Group(nil).Has("go:noinline") {
		t.Fatal("nil syntax acquired directives")
	}
}
