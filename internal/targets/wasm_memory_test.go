package targets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWASIMemoryConfig(t *testing.T) {
	dir := t.TempDir()
	parent := `{"goos":"wasip1","goarch":"wasm","wasm-profile":"w32","wasm-provider":"wasi",
		"wasm-initial-memory":16777216,"wasm-max-memory":536870912,"wasm-stack-size":10485760}`
	if err := os.WriteFile(filepath.Join(dir, "parent.json"), []byte(parent), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, fields, wantError string
	}{
		{"inherit", ``, ""},
		{"override", `,"wasm-initial-memory":8388608,"wasm-stack-size":1048576`, ""},
		{"initial alignment", `,"wasm-initial-memory":65537`, "wasm-initial-memory"},
		{"maximum alignment", `,"wasm-max-memory":65537`, "wasm-max-memory"},
		{"initial overflow", `,"wasm-initial-memory":4295032832`, "wasm-initial-memory"},
		{"maximum overflow", `,"wasm-max-memory":4295032832`, "wasm-max-memory"},
		{"stack alignment", `,"wasm-stack-size":17`, "wasm-stack-size"},
		{"stack overflow", `,"wasm-stack-size":4294967296`, "wasm-stack-size"},
		{"inherited conflict deferred to linker", `,"wasm-max-memory":8388608`, ""},
		{"negative", `,"wasm-initial-memory":-65536`, "cannot unmarshal"},
		{"other profile", `,"wasm-profile":"j32","wasm-provider":"emscripten"`, "w32/wasi"},
		{"other OS", `,"goos":"js"`, "w32/wasi"},
		{"other architecture", `,"goarch":"amd64"`, "w32/wasi"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := `{"inherits":["parent"]` + tc.fields + `}`
			if err := os.WriteFile(filepath.Join(dir, "child.json"), []byte(data), 0o644); err != nil {
				t.Fatal(err)
			}
			cfg, err := NewResolver(dir).Resolve("child")
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("error = %v, want %q", err, tc.wantError)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			initial, stack, maximum := uint64(16<<20), uint64(10<<20), uint64(512<<20)
			if tc.name == "override" {
				initial, stack = 8<<20, 1<<20
			}
			if tc.name == "inherited conflict deferred to linker" {
				maximum = 8 << 20
			}
			if cfg.WasmInitialMemory != initial || cfg.WasmMaxMemory != maximum || cfg.WasmStackSize != stack {
				t.Fatalf("incorrect inherited memory settings: %+v", cfg)
			}
		})
	}
}
