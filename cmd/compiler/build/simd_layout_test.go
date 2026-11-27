//go:build !llgo

package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSIMDMapStorageDiagnostic(t *testing.T) {
	for _, source := range []string{
		"type Unused map[int]archsimd.Float32x4",
		"type V = archsimd.Float32x4; type Unused map[int]V",
		"type V archsimd.Float32x4; type Unused map[int]V",
		"type Unused map[int]struct { X [0]archsimd.Float32x4 }",
		"type M[T any] map[int]T; type Unused M[archsimd.Float32x4]",
	} {
		t.Run(source, func(t *testing.T) {
			dir := t.TempDir()
			for name, text := range map[string]string{"go.mod": "module layouttest\n\ngo 1.27\n", "main.go": "package main\nimport \"simd/archsimd\"\n" + source + "\nfunc main(){}\n"} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0600); err != nil {
					t.Fatal(err)
				}
			}
			conf := NewDefaultConf(ModeGen)
			conf.Goos, conf.Goarch, conf.GOAMD64, conf.GOEXPERIMENT = "linux", "amd64", "v1", "simd"
			_, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: dir})
			if err == nil || !strings.Contains(err.Error(), "vector storage is not supported in map value") || !strings.Contains(err.Error(), "main.go:3:") {
				t.Fatalf("positioned diagnostic: %v", err)
			}
		})
	}
}

func TestSIMDGenericMapStorageDiagnostic(t *testing.T) {
	for _, tc := range []struct{ name, declarations, call, diagnostic string }{
		{"body", "func f[T any]() { m := make(map[int]T); m[0] = *new(T); println(len(m)) }", "f[archsimd.Float32x4]()", "value"},
		{"nested", "func f[T any]() { m := make(map[int]struct{ V [0]T }); println(len(m)) }", "f[archsimd.Float32x4]()", "value"},
		{"method", "type Box[T any] struct{}; func (Box[T]) f() { m := make(map[int]T); println(len(m)) }", "Box[archsimd.Float32x4]{}.f()", "value"},
		{"closure", "func f[T any]() { g := func() { m := make(map[int]T); println(len(m)) }; g() }", "f[archsimd.Float32x4]()", "value"},
		{"scalar", "func f[T any]() { m := make(map[int]T); println(len(m)) }", "f[int]()", ""},
		{"scalar-array", "func f[T any]() { m := make(map[int]T); println(len(m)) }", "f[[4]float32]()", ""},
		{"vector-pointer", "func f[T any]() { m := make(map[int]T); println(len(m)) }", "f[*archsimd.Float32x4]()", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			source := "package main\nimport \"simd/archsimd\"\nvar _ archsimd.Float32x4\n" + tc.declarations + "\nfunc main(){" + tc.call + "}\n"
			for name, data := range map[string]string{"go.mod": "module genericlayouttest\n\ngo 1.27\n", "main.go": source} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0600); err != nil {
					t.Fatal(err)
				}
			}
			conf := NewDefaultConf(ModeGen)
			conf.Goos, conf.Goarch, conf.GOAMD64, conf.GOEXPERIMENT = "linux", "amd64", "v1", "simd"
			_, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: dir})
			if tc.diagnostic == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), "vector storage is not supported in map "+tc.diagnostic) || !strings.Contains(err.Error(), "main.go:4:") {
				t.Fatalf("positioned diagnostic: %v", err)
			}
		})
	}
}
