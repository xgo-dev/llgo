package build

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestSingleWASIMutatorProof(t *testing.T) {
	for _, tc := range []struct {
		name, module, importName, exportName string
		memoryFlags                          byte
		want                                 bool
	}{
		{"closed", "wasi_snapshot_preview1", "fd_write", "_start", 3, true},
		{"threads", "wasi", "thread-spawn", "_start", 3, false},
		{"thread extension", "wasi_snapshot_preview1", "thread-spawn", "_start", 3, false},
		{"opaque import", "env", "callback", "_start", 3, false},
		{"opaque export", "wasi_snapshot_preview1", "fd_write", "callback", 3, false},
		{"unshared", "wasi_snapshot_preview1", "fd_write", "_start", 1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			module := []byte{0, 'a', 's', 'm', 1, 0, 0, 0}
			section := func(id byte, payload []byte) {
				module = append(module, id)
				module = binary.AppendUvarint(module, uint64(len(payload)))
				module = append(module, payload...)
			}
			name := func(bytes []byte, id string) []byte {
				bytes = binary.AppendUvarint(bytes, uint64(len(id)))
				return append(bytes, id...)
			}
			section(1, []byte{1, 0x60, 0, 0})
			imports := name(name([]byte{2}, "env"), "memory")
			imports = append(imports, 2, tc.memoryFlags, 1, 8)
			imports = name(name(imports, tc.module), tc.importName)
			section(2, append(imports, 0, 0))
			section(3, []byte{1, 0})
			exports := append(name([]byte{2}, "memory"), 2, 0)
			exports = append(name(exports, tc.exportName), 0, 1)
			section(7, exports)
			section(10, []byte{1, 2, 0, 0x0b})
			path := filepath.Join(t.TempDir(), "app.wasm")
			if err := os.WriteFile(path, module, 0o644); err != nil {
				t.Fatal(err)
			}
			if got := singleWASIMutator(path); got != tc.want {
				t.Fatalf("single mutator proven = %v, want %v", got, tc.want)
			}
		})
	}
}
