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
		store := new(Store)
		snapshot := store.File(file)
		if !snapshot.PatchSkip.All || !reflect.DeepEqual(snapshot.PatchSkip.Names, []string{"A", "B"}) {
			t.Fatalf("patch commands = %+v", snapshot.PatchSkip)
		}
	}
}

func TestStoreAssociatesConstructedSyntax(t *testing.T) {
	doc := &ast.CommentGroup{List: []*ast.Comment{nil, {Text: "//llgointernal:tls"}, {Text: "//llgo:skipall"}}}
	file := &ast.File{Comments: []*ast.CommentGroup{nil, doc}}
	store := new(Store)
	record := store.File(file)
	if len(record.Internal) != 1 || record.Internal[0].Name != "llgointernal:tls" || !record.Group(doc).Has("llgo:skipall") || !record.Group(doc).Skip.All {
		t.Fatalf("file directives = %+v", record)
	}
	fn := &ast.FuncDecl{Doc: &ast.CommentGroup{List: []*ast.Comment{{Text: "//go:noinline"}}}}
	if !store.Function(fn).NoInline {
		t.Fatal("standalone function not discovered")
	}
	fn.Doc = nil
	store.Freeze()
	if props, ok := store.LookupFunction(fn); !ok || !props.NoInline {
		t.Fatalf("prepared function = %+v, %v", props, ok)
	}
	if _, ok := store.LookupFunction(&ast.FuncDecl{}); ok {
		t.Fatal("unprepared function appeared in snapshot")
	}
	if store.Function(nil) != (Function{}) || store.Group(nil).Has("go:noinline") {
		t.Fatal("nil syntax acquired directives")
	}
}
