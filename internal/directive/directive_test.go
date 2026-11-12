package directive

import (
	"go/ast"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		text string
		name string
		args string
		ok   bool
	}{
		{text: "// ordinary"},
		{text: "//go:"},
		{text: "//go:noinline", name: "go:noinline", ok: true},
		{text: "//llgointernal:tls", name: "llgointernal:tls", ok: true},
		{text: "// llgo:type C", name: "llgo:type", args: "C", ok: true},
		{text: "//llgo:cdecl", name: "llgo:cdecl", ok: true},
		{text: "// llgo:stdcall", name: "llgo:stdcall", ok: true},
		{text: "//llgo:link\tF C.f", name: "llgo:link", args: "F C.f", ok: true},
		{text: "//export F", name: "export", args: "F", ok: true},
	}
	if _, ok := Parse(nil); ok {
		t.Fatal("nil comment parsed as a directive")
	}
	if ParseGroup(nil) != nil {
		t.Fatal("nil comment group returned directives")
	}
	for _, test := range tests {
		got, ok := Parse(&ast.Comment{Text: test.text})
		if ok != test.ok || got.Name != test.name || got.Args != test.args || got.Raw != map[bool]string{true: test.text}[test.ok] {
			t.Fatalf("Parse(%q) = %+v, %v", test.text, got, ok)
		}
	}
}

func TestParseGroupPreservesSourceOrder(t *testing.T) {
	doc := &ast.CommentGroup{List: []*ast.Comment{
		{Text: "// ordinary"},
		{Text: "//go:noinline"},
		{Text: "//llgointernal:tls"},
	}}
	got := ParseGroup(doc)
	if len(got) != 2 || got[0].Name != "go:noinline" || got[1].Name != "llgointernal:tls" {
		t.Fatalf("ParseGroup = %+v", got)
	}
}

func TestParseLinknameArgs(t *testing.T) {
	for _, tc := range []struct{ args, local, target string }{
		{"", "", ""},
		{" Exported ", "Exported", ""},
		{" F C.f ", "F", "C.f"},
		{"F\tllvm.fshl.i8", "F", "llvm.fshl.i8"},
		{"(*T).Method\t C.method", "(*T).Method", "C.method"},
		{"F\t target  with\tspaces", "F", "target with spaces"},
	} {
		local, target := ParseLinknameArgs(tc.args)
		if local != tc.local || target != tc.target {
			t.Errorf("ParseLinknameArgs(%q) = (%q, %q), want (%q, %q)", tc.args, local, target, tc.local, tc.target)
		}
	}
}
