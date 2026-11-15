//go:build !llgo

package debug

import (
	"debug/macho"
	"encoding/binary"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDarwinLTOSymbolsRejectsMissingDebugObjects(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Darwin dSYM sessions")
	}
	directory := t.TempDir()
	artifact := filepath.Join(directory, "program with spaces")
	if args, cleanup, err := darwinLTOSymbols(artifact, false); err != nil || len(args) != 0 {
		t.Fatalf("ordinary artifact prepared LTO symbols: %v, %v", args, err)
	} else {
		cleanup()
	}
	if err := os.Mkdir(artifact+".llgo-debug-lto", 0o700); err != nil {
		t.Fatal(err)
	}
	// A historical sidecar does not make an ordinary build an LTO session.
	if args, cleanup, err := darwinLTOSymbols(artifact, false); err != nil || len(args) != 0 {
		t.Fatalf("historical objects selected an LTO session: %v, %v", args, err)
	} else {
		cleanup()
	}
	writeDarwinDebugMap(t, artifact, false)
	if args, cleanup, err := darwinLTOSymbols(artifact, true); err != nil || len(args) != 0 {
		t.Fatalf("stripped image invoked dsymutil: %v, %v", args, err)
	} else {
		cleanup()
	}
	// A custom object_path_lto has a debug map without LLGo's sidecar.
	if err := os.Remove(artifact + ".llgo-debug-lto"); err != nil {
		t.Fatal(err)
	}
	writeDarwinDebugMap(t, artifact, true)
	// dsymutil can return success after warning about unavailable objects.
	// Never attach an empty/missing symbol file in that case.
	if err := os.WriteFile(filepath.Join(directory, "dsymutil"), []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("TMPDIR", directory)
	args, cleanup, err := darwinLTOSymbols(artifact, true)
	defer cleanup()
	if err == nil || !strings.Contains(err.Error(), "read Darwin LTO symbols") || len(args) != 0 {
		t.Fatalf("unavailable symbols were accepted: %v, %v", args, err)
	}
	if dirs, err := filepath.Glob(filepath.Join(directory, "llgo-debug-dsym-*")); err != nil || len(dirs) != 0 {
		t.Fatalf("failed preparation leaked private symbols: %v, %v", dirs, err)
	}
}

func writeDarwinDebugMap(t *testing.T, path string, hasMap bool) {
	t.Helper()
	names := []byte("\x00custom-lto-object.o\x00")
	raw := make([]byte, 72+len(names))
	for offset, value := range map[int]uint32{
		0: uint32(macho.Magic64), 4: uint32(macho.CpuArm64), 12: uint32(macho.TypeExec),
		16: 1, 20: 24, 32: uint32(macho.LoadCmdSymtab), 36: 24,
		40: 56, 48: 72, 52: uint32(len(names)), 56: 1,
	} {
		binary.LittleEndian.PutUint32(raw[offset:], value)
	}
	if hasMap {
		binary.LittleEndian.PutUint32(raw[44:], 1)
		raw[60] = 0x66 // N_OSO
	}
	copy(raw[72:], names)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
}
