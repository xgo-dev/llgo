package directive

import (
	"go/ast"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLegacyLinkDialect(t *testing.T) {
	for _, tt := range []struct {
		line          string
		export        bool
		status        int
		valid         bool
		local, target string
		isExport      bool
	}{
		{"//go:linkname F runtime.f", false, HasLinkname, true, "F", "runtime.f", false},
		{"// llgo:link F C.f", false, HasLinkname, true, "F", "C.f", false},
		{"//llgo:link F   C.f", false, HasLinkname, true, "F", "C.f", false},
		{"//export Public", true, HasLinkname, true, "Public", "Public", true},
		{"//export Public", false, NoDirective, false, "", "", false},
		{"//go:linkname F\tC.f", false, HasLinkname, false, "", "", false},
		{"//go:linkname F", false, HasLinkname, false, "", "", false},
		{"//go:noinline", false, UnknownDirective, false, "", "", false},
		{"// ordinary", false, NoDirective, false, "", "", false},
	} {
		t.Run(tt.line, func(t *testing.T) {
			got := ParseLegacyLink(tt.line, tt.export)
			if got.Status != tt.status || got.Valid != tt.valid || got.Local != tt.local || got.Target != tt.target || got.Export != tt.isExport || got.Raw != tt.line {
				t.Fatalf("link = %+v", got)
			}
		})
	}

}

func TestImportedLinkSnapshotsCacheSuccessAndFailure(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "imported.go")
	source := "package p\n//go:linkname F runtime.f\n// llgo:link G C.g\n//export H\n //go:linkname indented ignored\n// ordinary\nfunc F() {}\n"
	if err := os.WriteFile(name, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	index := new(Index)
	got := index.ReadLegacyLinks(name)
	if len(got) != 3 || got[0].Target != "runtime.f" || got[1].Target != "C.g" || !got[2].Export {
		t.Fatalf("imported links = %+v", got)
	}
	missing := filepath.Join(dir, "missing.go")
	if links := index.ReadLegacyLinks(missing); len(links) != 0 {
		t.Fatalf("missing file links = %+v", links)
	}
	if err := os.Remove(name); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(missing, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	index.Freeze()
	if !reflect.DeepEqual(index.ReadLegacyLinks(name), got) {
		t.Fatal("prepared source was reread")
	}
	if len(index.ReadLegacyLinks(missing)) != 0 {
		t.Fatal("failed discovery was retried after freeze")
	}
}

func TestFreezeClosesEveryDiscoveryEntry(t *testing.T) {
	for _, tt := range []struct {
		name     string
		discover func(*Index)
	}{
		{"file", func(s *Index) { s.File(&ast.File{}) }},
		{"group", func(s *Index) { s.Group(&ast.CommentGroup{}) }},
		{"function", func(s *Index) { s.Function(&ast.FuncDecl{}) }},
		{"import", func(s *Index) { s.ReadLegacyLinks("unprepared.go") }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := new(Index)
			s.Freeze()
			defer func() {
				r := recover()
				if r == nil || !strings.Contains(r.(string), "not prepared before lowering") {
					t.Fatalf("discovery panic = %v", r)
				}
			}()
			tt.discover(s)
		})
	}
}
