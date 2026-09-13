// Package binaryfixture supplies native-format inputs for parser tests on hosts
// such as wasm that cannot execute a compiler. Fixtures contain real DWARF from
// testdata/fixture.c, not just enough header bytes to make Open succeed.
package binaryfixture

//go:generate go run generate.go

import (
	"compress/gzip"
	"encoding/base64"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func ELF(t testing.TB) string   { return write(t, "fixture.o", dataELF) }
func MachO(t testing.TB) string { return write(t, "fixture.o", dataMachO) }
func PE(t testing.TB) string    { return write(t, "fixture.exe", dataPE) }

func write(t testing.TB, name, encoded string) string {
	t.Helper()
	path, err := materialize(t.TempDir(), name, encoded)
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func materialize(dir, name, encoded string) (string, error) {
	r, err := gzip.NewReader(base64.NewDecoder(base64.StdEncoding, strings.NewReader(encoded)))
	if err != nil {
		return "", err
	}
	data, err := io.ReadAll(r)
	if closeErr := r.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}
