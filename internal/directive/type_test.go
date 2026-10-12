package directive

import (
	"go/ast"
	"strings"
	"testing"
)

func TestCollectPreparedCallingConventions(t *testing.T) {
	for _, tc := range []struct {
		name, source, background, replacement, diagnostic string
	}{
		{"cdecl", "//llgo:cdecl\ntype T func()", "C", "", ""},
		{"stdcall", "// llgo:stdcall\ntype T func()", "stdcall", "", ""},
		{"alias", "//llgo:cdecl\ntype T = func()", "C", "", ""},
		{"struct", "//llgo:cdecl\ntype T struct{}", "C", "", ""},
		{"trailing prose", "//llgo:cdecl\n// Callback documentation.\ntype T func()", "C", "", ""},
		{"legacy C", "//llgo:type C\ntype T func()", "C", "//llgo:cdecl", ""},
		{"legacy stdcall", "// llgo:type stdcall\ntype T func()", "stdcall", "//llgo:stdcall", ""},
		{"legacy tab and trailing prose", "//llgo:type\tC\n// Callback documentation.\ntype T func()", "C", "//llgo:cdecl", ""},
		{"other legacy form", "//llgo:type Python\ntype T func()", "", "", ""},
		{"single grouped type", "//llgo:cdecl\ntype (T func())", "C", "", ""},
		{"unannotated group", "type (T func(); U func())", "", "", ""},
		{"spec comment ignored", "type (\n//llgo:cdecl\nT func()\n)", "", "", ""},
		{"field comment ignored", "type T struct {\n//llgo:cdecl\nCallback func()\n}", "", "", ""},
		{"new conflict", "//llgo:cdecl\n//llgo:stdcall\ntype T func()", "", "", "conflicting calling-convention directives"},
		{"legacy conflict", "//llgo:type C\n//llgo:stdcall\ntype T func()", "", "", "conflicting calling-convention directives"},
		{"duplicate", "//llgo:cdecl\n//llgo:cdecl\ntype T func()", "", "", "conflicting calling-convention directives"},
		{"cdecl argument", "//llgo:cdecl extra\ntype T func()", "", "", "does not accept arguments"},
		{"stdcall argument", "//llgo:stdcall extra\ntype T func()", "", "", "does not accept arguments"},
		{"annotated group", "//llgo:cdecl\ntype (T func(); U func())", "", "", "single type declaration"},
		{"legacy annotated group", "//llgo:type C\ntype (T func(); U func())", "", "", "single type declaration"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fset, file := parseSource(t, "package p\n"+tc.source)
			index := new(Index)
			files := index.Files([]*ast.File{file})
			index.Freeze()
			// Collection must consume the snapshot, even if source comments
			// are rewritten after preparation.
			for _, group := range file.Comments {
				for _, comment := range group.List {
					comment.Text = "// removed"
				}
			}
			records, err := Collect(fset, files, false, false)
			if tc.diagnostic != "" {
				if err == nil || !strings.Contains(err.Error(), tc.diagnostic) || !strings.Contains(err.Error(), "directives.go:") {
					t.Fatalf("diagnostic = %v, want positioned %q", err, tc.diagnostic)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			record := records.Names["T"].(*TypeDecl)
			if record.Background != tc.background {
				t.Fatalf("background = %q, want %q", record.Background, tc.background)
			}
			if tc.replacement == "" {
				if record.Deprecation != "" {
					t.Fatalf("unexpected warning: %s", record.Deprecation)
				}
			} else if !strings.Contains(record.Deprecation, "DEPRECATED:") || !strings.Contains(record.Deprecation, "directives.go:2:") || !strings.Contains(record.Deprecation, "use "+tc.replacement+" instead") {
				t.Fatalf("warning = %q, want source position and replacement %q", record.Deprecation, tc.replacement)
			}
		})
	}
}
