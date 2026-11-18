package main

import (
	"go/version"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/xgo-dev/llgo/internal/build"
)

func TestRunExpectDirPreservesMarkers(t *testing.T) {
	for _, marker := range []string{";", ";\r\n", "; ir-only", "; ir-only\n"} {
		for _, filename := range []string{"expect.txt", "expect_" + version.Lang(runtime.Version()) + ".txt"} {
			t.Run(filename+"/"+marker, func(t *testing.T) {
				root := t.TempDir()
				dir := filepath.Join(root, "fixtures", "skip")
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				base := filepath.Join(dir, "expect.txt")
				if err := os.WriteFile(base, []byte("fallback output\n"), 0644); err != nil {
					t.Fatal(err)
				}
				file := filepath.Join(dir, filename)
				if err := os.WriteFile(file, []byte(marker), 0644); err != nil {
					t.Fatal(err)
				}
				runExpectDir(root, "fixtures", func(*build.Config) {
					t.Fatal("marker fixture must not be executed when refreshing goldens")
				})
				if got, err := os.ReadFile(file); err != nil || string(got) != marker {
					t.Fatalf("marker changed: %q, %v", got, err)
				}
				if file != base {
					if got, err := os.ReadFile(base); err != nil || string(got) != "fallback output\n" {
						t.Fatalf("fallback changed: %q, %v", got, err)
					}
				}
			})
		}
	}
}

func TestRunExpectDirLiteralMarkers(t *testing.T) {
	for _, literal := range []string{";\n\n", " ;", "; ir-only\n\n"} {
		t.Run(literal, func(t *testing.T) {
			root := t.TempDir()
			dir := filepath.Join(root, "fixtures", "literal")
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "expect.txt"), []byte(literal), 0644); err != nil {
				t.Fatal(err)
			}
			// Stop at configuration: a literal golden must reach execution,
			// while marker preservation must return before this point.
			const configured = "literal golden selected for execution"
			defer func() {
				if got := recover(); got != configured {
					t.Fatalf("execution selection = %v", got)
				}
			}()
			runExpectDir(root, "fixtures", func(*build.Config) { panic(configured) })
		})
	}
}

func TestRunExpectDirRefreshesVersionedGolden(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs a fixture program")
	}
	root := t.TempDir()
	dir := filepath.Join(root, "fixtures", "versioned")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	versioned := "expect_" + version.Lang(runtime.Version()) + ".txt"
	for name, data := range map[string]string{
		"go.mod":     "module example.com/versioned\ngo 1.20\n",
		"main.go":    "package main\nfunc main() { println(\"updated\") }\n",
		"expect.txt": "keep fallback\n",
		versioned:    "old versioned output\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0644); err != nil {
			t.Fatal(err)
		}
	}
	runExpectDir(root, "fixtures")
	for name, want := range map[string]string{"expect.txt": "keep fallback\n", versioned: "updated\n"} {
		if got, err := os.ReadFile(filepath.Join(dir, name)); err != nil || string(got) != want {
			t.Fatalf("%s: %q, %v; want %q", name, got, err, want)
		}
	}
}

func TestRunExpectDirPreservesGoldenOnFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs a fixture program")
	}
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
