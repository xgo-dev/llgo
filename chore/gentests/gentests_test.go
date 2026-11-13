package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xgo-dev/llgo/internal/build"
)

func TestRunExpectDirPreservesMarkers(t *testing.T) {
	for _, marker := range []string{";", ";\r\n", "; ir-only", "; ir-only\n"} {
		t.Run(marker, func(t *testing.T) {
			root := t.TempDir()
			dir := filepath.Join(root, "fixtures", "skip")
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}
			file := filepath.Join(dir, "expect.txt")
			if err := os.WriteFile(file, []byte(marker), 0644); err != nil {
				t.Fatal(err)
			}
			runExpectDir(root, "fixtures", func(*build.Config) {
				t.Fatal("marker fixture must not be executed when refreshing goldens")
			})
			if got, err := os.ReadFile(file); err != nil || string(got) != marker {
				t.Fatalf("marker changed: %q, %v", got, err)
			}
		})
	}
}

func TestRunExpectDirPreservesGoldenOnFailure(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "fixtures", "failure")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string]string{
		"go.mod":     "module example.com/failure\ngo 1.20\n",
		"main.go":    "package main\nfunc main() { panic(\"intentional failure\") }\n",
		"expect.txt": "keep this golden\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0644); err != nil {
			t.Fatal(err)
		}
	}
	runExpectDir(root, "fixtures")
	if got, err := os.ReadFile(filepath.Join(dir, "expect.txt")); err != nil || string(got) != "keep this golden\n" {
		t.Fatalf("failed execution changed golden: %q, %v", got, err)
	}
}
