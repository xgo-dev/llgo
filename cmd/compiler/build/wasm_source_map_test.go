package build

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRelocateWasmSourceMap(t *testing.T) {
	for _, tc := range []struct{ name, root, wantRoot string }{
		{name: "relative sources"},
		{name: "relative root", root: "../src", wantRoot: "src"},
		{name: "URL root", root: "https://debug.example/src", wantRoot: "https://debug.example/src"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			stage := filepath.Join(dir, "stage")
			if err := os.Mkdir(stage, 0700); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(stage, "app.wasm.map")
			absolute := filepath.Join(dir, "absolute.go")
			original := map[string]any{"version": 3, "sourceRoot": tc.root, "sources": []string{"../main.go", "https://debug.example/main.go", absolute}, "sourcesContent": []string{"package main"}, "names": []string{}, "mappings": "AAAA"}
			raw, err := json.Marshal(original)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			if err := relocateWasmSourceMap(path, dir); err != nil {
				t.Fatal(err)
			}
			raw, err = os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var got struct {
				SourceRoot              string
				Sources, SourcesContent []string
				Mappings                string
			}
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatal(err)
			}
			wantSource := "../main.go"
			if tc.root == "" {
				wantSource = "main.go"
			}
			if got.SourceRoot != tc.wantRoot || got.Sources[0] != wantSource || got.Sources[1] != "https://debug.example/main.go" || got.Sources[2] != absolute || got.SourcesContent[0] != "package main" || got.Mappings != "AAAA" {
				t.Fatalf("relocated map: %s", raw)
			}
		})
	}
}

func TestRelocateWasmSourceMapSymlinks(t *testing.T) {
	dir := t.TempDir()
	realDir := filepath.Join(dir, "real")
	stage := filepath.Join(realDir, "stage")
	if err := os.MkdirAll(stage, 0700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(dir, "alias")
	if err := os.Symlink(realDir, alias); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	path := filepath.Join(alias, "stage", "app.wasm.map")
	if err := os.WriteFile(path, []byte(`{"version":3,"sources":["../main.go"],"mappings":"AAAA"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := relocateWasmSourceMap(path, alias); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got struct{ Sources []string }
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.Sources[0] != "main.go" {
		t.Fatalf("symlink relocation=%s", raw)
	}
}

func TestRelocateWasmSourceMapInvalid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.wasm.map")
	if err := relocateWasmSourceMap(path, filepath.Dir(path)); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{"not JSON", `{"sources":42}`, `{"sourceRoot":42}`, `{"sources":[42]}`} {
		if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		if err := relocateWasmSourceMap(path, filepath.Dir(path)); err == nil {
			t.Fatalf("accepted invalid map %s", raw)
		}
		got, err := os.ReadFile(path)
		if err != nil || string(got) != raw {
			t.Fatalf("invalid map was modified: %q, %v", got, err)
		}
	}
}
