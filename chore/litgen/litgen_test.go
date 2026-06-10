package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/goplus/mod"
)

func TestProcessPath_SingleFileUsesContainingDir(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, _, err := mod.FindGoMod(wd)
	if err != nil {
		t.Fatal(err)
	}
	pkgDir, err := os.MkdirTemp(wd, "processpath-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(pkgDir)
	})
	sourceFile := filepath.Join(pkgDir, "in.go")
	if err := os.WriteFile(sourceFile, []byte("// LITTEST\npackage main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := processPath(sourceFile); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(sourceFile)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Contains(text, "command-line-arguments") {
		t.Fatalf("single-file mode should compile the containing package, got:\n%s", text)
	}
	relPath, err := filepath.Rel(root, pkgDir)
	if err != nil {
		t.Fatal(err)
	}
	want := regexp.MustCompile(`(?m)^// CHECK-LABEL: define void @"\{\{.*\}\}/` + regexp.QuoteMeta(filepath.ToSlash(relPath)) + `\.main"\(\)(?:\{\{.*\}\}|(?: #[0-9]+)?(?: !dbg ![0-9]+)?) \{$`)
	if !want.MatchString(text) {
		t.Fatalf("missing package-qualified main check:\n%s", text)
	}
}
