package targets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoaderRejectsInvalidTargetNames(t *testing.T) {
	loader := NewLoader(t.TempDir())
	for _, bad := range []string{"", ".", "..", "../escape", "sub/target", `sub\target`, "/etc/passwd"} {
		if _, err := loader.LoadRaw(bad); err == nil {
			t.Errorf("LoadRaw(%q) succeeded, want error", bad)
		}
		if _, err := loader.Load(bad); err == nil {
			t.Errorf("Load(%q) succeeded, want error", bad)
		}
	}
}

func TestLoaderDetectsInheritanceCycle(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"a.json":    `{"inherits": ["b"]}`,
		"b.json":    `{"inherits": ["a"]}`,
		"self.json": `{"inherits": ["self"]}`,
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	loader := NewLoader(dir)
	for _, target := range []string{"a", "self"} {
		_, err := loader.Load(target)
		if err == nil || !strings.Contains(err.Error(), "cycle") {
			t.Fatalf("Load(%q) err = %v, want inheritance cycle error", target, err)
		}
	}
}
