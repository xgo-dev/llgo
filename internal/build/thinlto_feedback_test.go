package build

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/xgo-dev/llgo/internal/meta"
)

func TestThinLTOFeedbackModulePaths(t *testing.T) {
	dir := t.TempDir()
	direct := filepath.Join(dir, "entry.o")
	archive := filepath.Join(dir, "pkg.a")
	paths := []string{
		direct + ".4.opt.bc",
		archive + "(pkg.o at 128).4.opt.bc",
	}
	for _, path := range paths {
		if err := os.WriteFile(path, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := thinLTOFeedbackModulePaths([]string{direct, archive})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, paths) {
		t.Fatalf("thinLTOFeedbackModulePaths() = %#v, want %#v", got, paths)
	}
	if _, err := thinLTOFeedbackModulePaths([]string{filepath.Join(dir, "missing.o")}); err == nil {
		t.Fatal("thinLTOFeedbackModulePaths accepted missing backend modules")
	}
}

func TestThinLTOFeedbackCandidates(t *testing.T) {
	buildMeta := func(names ...string) *meta.PackageMeta {
		b := meta.NewBuilder()
		for _, name := range names {
			b.MarkReflect(b.Sym(name))
		}
		pm, err := b.Build()
		if err != nil {
			t.Fatal(err)
		}
		return pm
	}
	pkgs := []Package{
		&aPackage{Meta: buildMeta("pkg.z", "pkg.a")},
		nil,
		&aPackage{Meta: buildMeta("pkg.a", "pkg.m")},
	}
	want := []string{"pkg.a", "pkg.m", "pkg.z"}
	if got := thinLTOFeedbackCandidates(pkgs); !reflect.DeepEqual(got, want) {
		t.Fatalf("thinLTOFeedbackCandidates() = %#v, want %#v", got, want)
	}
}

func TestInsertThinLTOOverlays(t *testing.T) {
	inputs := []string{"entry.o", "extra.o", "one.a", "two.a"}
	overlays := []string{"a.overlay.o", "b.overlay.o"}
	want := []string{"entry.o", "extra.o", "a.overlay.o", "b.overlay.o", "one.a", "two.a"}
	if got := insertThinLTOOverlays(inputs, overlays); !reflect.DeepEqual(got, want) {
		t.Fatalf("insertThinLTOOverlays() = %#v, want %#v", got, want)
	}
	withoutArchive := []string{"entry.o", "a.overlay.o", "b.overlay.o"}
	if got := insertThinLTOOverlays([]string{"entry.o"}, overlays); !reflect.DeepEqual(got, withoutArchive) {
		t.Fatalf("insertThinLTOOverlays(no archive) = %#v, want %#v", got, withoutArchive)
	}
}
