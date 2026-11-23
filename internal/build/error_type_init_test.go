package build

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testErrorTypeSource = `package errors
import "internal/reflectlite"
var errorType = reflectlite.TypeOf((*error)(nil)).Elem()
func As(t reflectlite.Type) bool {
	return t.Kind() == reflectlite.Interface || t.Implements(errorType)
}
`

func TestDeferredErrorTypeSource(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "src", "errors")
	if _, got := deferredErrorTypeSource(root, nil); got != nil {
		t.Fatal("source patch accepted an absent package")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, got := deferredErrorTypeSource(root, nil); got != nil {
		t.Fatal("source patch accepted a missing wrap.go")
	}
	path := filepath.Join(dir, "wrap.go")
	source := []byte(testErrorTypeSource)
	if err := os.WriteFile(path, source, 0o644); err != nil {
		t.Fatal(err)
	}
	gotPath, got := deferredErrorTypeSource(root, nil)
	if gotPath != path || !bytes.Contains(got, []byte("t.Implements(llgoErrorType())")) || bytes.Contains(got, []byte("var errorType")) {
		t.Fatalf("unexpected source patch %q:\n%s", gotPath, got)
	}
	if bytes.Count(got, []byte{'\n'}) != bytes.Count(source, []byte{'\n'}) {
		t.Fatal("source patch shifted source lines")
	}
	if _, got := deferredErrorTypeSource(root, map[string][]byte{path: source}); got != nil {
		t.Fatal("source patch replaced an explicit overlay")
	}
	overlay, files, err := buildSourcePatchOverlayForGOROOT(nil, t.TempDir(), root, sourcePatchBuildContext{deferErrorType: true})
	if err != nil {
		t.Fatal(err)
	}
	cacheFile := filepath.Join(dir, errorTypeCacheFile)
	if injected := files["errors"]; len(injected) != 1 || injected[0] != cacheFile ||
		!bytes.Equal(overlay[path], got) || string(overlay[cacheFile]) != errorTypeCacheSource {
		t.Fatalf("unexpected errors overlay files %v", injected)
	}
	if err := os.WriteFile(cacheFile, []byte("package errors"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, got := deferredErrorTypeSource(root, nil); got != nil {
		t.Fatal("source patch shadowed an existing source file")
	}
	if err := os.Remove(cacheFile); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "other.go"), []byte("package errors; var copyType = errorType"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, got := deferredErrorTypeSource(root, nil); got != nil {
		t.Fatal("source patch removed a descriptor used by another file")
	}
}

func TestDeferredErrorTypeRequiresOneValidationUse(t *testing.T) {
	for _, source := range []string{
		"package errors\nfunc (",
		"package errors\nimport \"reflect\"\n",
		"package errors\nimport \"internal/reflectlite\"\n",
		strings.ReplaceAll(testErrorTypeSource, "reflectlite.TypeOf((*error)(nil)).Elem()", "descriptor()"),
		testErrorTypeSource + "func usedElsewhere() { _ = errorType }\n",
		strings.ReplaceAll(testErrorTypeSource, "func As(", "func lookup("),
		testErrorTypeSource + "//go:linkname alias errors.errorType\n",
		testErrorTypeSource + "func llgoErrorType() {}\n",
	} {
		if got := deferErrorTypeInit([]byte(source)); got != nil {
			t.Fatalf("rewrote unmatched package shape:\n%s", source)
		}
	}
}
