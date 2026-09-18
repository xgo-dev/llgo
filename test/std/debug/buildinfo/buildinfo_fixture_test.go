//go:build llgo || wasm

package buildinfo_test

import (
	"bytes"
	"debug/buildinfo"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

// A minimal ELF executable containing a valid inline .go.buildinfo section.
// Keeping the fixture in the target binary exercises the parser without
// requiring a nested Go toolchain inside a wasm sandbox.
const buildInfoELF = `f0VMRgIBAQAAAAAAAAAAAAIAPgABAAAAXBEgAAAAAABAAAAAAAAAAPABAAAAAAAAAAAAAEAAOAAEAEAABwAFAAYAAAAEAAAAQAAAAAAAAABAACAAAAAAAEAAIAAAAAAA4AAAAAAAAADgAAAAAAAAAAgAAAAAAAAAAQAAAAQAAAAAAAAAAAAAAAAAIAAAAAAAAAAgAAAAAABaAQAAAAAAAFoBAAAAAAAAABAAAAAAAAABAAAABQAAAFwBAAAAAAAAXBEgAAAAAABcESAAAAAAAAEAAAAAAAAAAQAAAAAAAAAAEAAAAAAAAFHldGQGAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA/yBHbyBidWlsZGluZjoIAgAAAAAAAAAAAAAAAAAAAAAIZ28xLjI3LjAAAAAAAAAAAAAAAAAAAAAAAAAAw0xpbmtlcjogSG9tZWJyZXcgTExEIDIzLjEuMAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAABAAAgBcESAAAAAAAAAAAAAAAAAAAC5nby5idWlsZGluZm8ALnRleHQALmNvbW1lbnQALnN5bXRhYgAuc2hzdHJ0YWIALnN0cnRhYgAAX3N0YXJ0AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAQAAAAIAAAAAAAAAIAEgAAAAAAAgAQAAAAAAADoAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAADwAAAAEAAAAGAAAAAAAAAFwRIAAAAAAAXAEAAAAAAAABAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAABUAAAABAAAAMAAAAAAAAAAAAAAAAAAAAF0BAAAAAAAAHAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAQAAAAAAAAAeAAAAAgAAAAAAAAAAAAAAAAAAAAAAAACAAQAAAAAAADAAAAAAAAAABgAAAAEAAAAIAAAAAAAAABgAAAAAAAAAJgAAAAMAAAAAAAAAAAAAAAAAAAAAAAAAsAEAAAAAAAA4AAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAADAAAAADAAAAAAAAAAAAAAAAAAAAAAAAAOgBAAAAAAAACAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAA=`

func buildInfoFixture(t *testing.T) []byte {
	t.Helper()
	data, err := base64.StdEncoding.DecodeString(buildInfoELF)
	if err != nil {
		t.Fatalf("decode build-info fixture: %v", err)
	}
	return data
}

func TestReadFileAndType(t *testing.T) {
	path := filepath.Join(t.TempDir(), "buildinfo.elf")
	if err := os.WriteFile(path, buildInfoFixture(t), 0o644); err != nil {
		t.Fatalf("write build-info fixture: %v", err)
	}
	info, err := buildinfo.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if info == nil || info.GoVersion != "go1.27.0" {
		t.Fatalf("ReadFile info = %#v", info)
	}
	var _ *buildinfo.BuildInfo = info
}

func TestRead(t *testing.T) {
	info, err := buildinfo.Read(bytes.NewReader(buildInfoFixture(t)))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if info == nil || info.GoVersion != "go1.27.0" {
		t.Fatalf("Read info = %#v", info)
	}
}
