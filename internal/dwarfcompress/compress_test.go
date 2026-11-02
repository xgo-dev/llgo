// Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
// Licensed under the Apache License, Version 2.0.

package dwarfcompress

import (
	"bytes"
	"compress/zlib"
	"debug/dwarf"
	"debug/macho"
	"debug/pe"
	"encoding/binary"
	"io"
	"reflect"
	"strings"
	"testing"
)

func dwarfEntries(t *testing.T, file interface{ DWARF() (*dwarf.Data, error) }) []*dwarf.Entry {
	t.Helper()
	data, err := file.DWARF()
	if err != nil {
		t.Fatal(err)
	}
	var entries []*dwarf.Entry
	r := data.Reader()
	for {
		e, err := r.Next()
		if err != nil {
			t.Fatal(err)
		}
		if e == nil {
			break
		}
		entries = append(entries, e)
	}
	if len(entries) == 0 {
		t.Fatalf("fixture lost its DWARF compilation unit: %v", entries)
	}
	return entries
}

func TestMachOCompression(t *testing.T) {
	raw := machoFixture(t)
	original := bytes.Clone(raw)
	before, err := macho.NewFile(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	oldEntries := dwarfEntries(t, before)
	out, err := Compress(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, original) || len(out) >= len(raw)/2 {
		t.Fatalf("compression mutated input or failed to reduce size: %d -> %d", len(raw), len(out))
	}
	after, err := macho.NewFile(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(oldEntries, dwarfEntries(t, after)) {
		t.Fatal("compressed Mach-O changed DWARF entries")
	}
	if after.Section("__zdebug_info") == nil || after.Section("__debug_info") != nil {
		t.Fatal("missing compressed Mach-O section")
	}
	if !reflect.DeepEqual(before.Symtab.Syms, after.Symtab.Syms) {
		t.Fatal("compression changed Mach-O symbols")
	}
	for i, command := range before.Loads {
		if binary.LittleEndian.Uint32(command.Raw()) == 0x1b && !bytes.Equal(command.Raw(), after.Loads[i].Raw()) {
			t.Fatal("compression changed dSYM UUID")
		}
	}
	again, err := Compress(out)
	if err != nil || !bytes.Equal(out, again) {
		t.Fatalf("second compression changed artifact: %v", err)
	}
}

func TestPECompression(t *testing.T) {
	for _, bits := range []int{32, 64} {
		t.Run(decimal(uint32(bits)), func(t *testing.T) {
			raw := peFixture(t, bits)
			original := bytes.Clone(raw)
			before, err := pe.NewFile(bytes.NewReader(raw))
			if err != nil {
				t.Fatal(err)
			}
			oldEntries := dwarfEntries(t, before)
			out, err := Compress(raw)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(raw, original) || len(out) >= len(raw)/2 {
				t.Fatalf("compression mutated input or failed to reduce size: %d -> %d", len(raw), len(out))
			}
			after, err := pe.NewFile(bytes.NewReader(out))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(oldEntries, dwarfEntries(t, after)) {
				t.Fatal("compressed PE changed DWARF entries")
			}
			if after.Section(".zdebug_info") == nil || after.Section(".debug_info") != nil {
				t.Fatal("missing compressed PE section")
			}
			for i, old := range before.Sections {
				next := after.Sections[i]
				if old.VirtualAddress != next.VirtualAddress || old.VirtualSize != next.VirtualSize {
					t.Fatalf("compression changed section %s memory layout", old.Name)
				}
				if !strings.HasPrefix(old.Name, ".debug_") && old.Name != ".rdata" {
					a, _ := old.Data()
					b, _ := next.Data()
					if !bytes.Equal(a, b) {
						t.Fatalf("compression changed section %s", old.Name)
					}
				}
			}
			debug := after.Section(".rdata")
			pointer := binary.LittleEndian.Uint32(out[int(debug.Offset)+24:])
			if pointer != debug.Offset+28 || string(out[pointer:pointer+5]) != "RSDS\x00" {
				t.Fatal("compression did not remap the CodeView raw-data pointer")
			}
			again, err := Compress(out)
			if err != nil || !bytes.Equal(out, again) {
				t.Fatalf("second compression changed artifact: %v", err)
			}
		})
	}
}

func TestMalformedArtifacts(t *testing.T) {
	mach := machoFixture(t)
	image := peFixture(t, 64)
	tests := []struct {
		name string
		raw  []byte
		edit func([]byte)
	}{
		{"empty", nil, nil},
		{"ELF belongs to linker", []byte("\x7fELF"), nil},
		{"truncated Mach-O", mach[:31], nil},
		{"Mach-O executable", mach, func(b []byte) { binary.LittleEndian.PutUint32(b[12:], 2) }},
		{"Mach-O load command", mach, func(b []byte) { binary.LittleEndian.PutUint32(b[32+(72+3*80)+72+24:], 0x1d) }},
		{"Mach-O relocation", mach, func(b []byte) { binary.LittleEndian.PutUint32(b[32+72+80+60:], 1) }},
		{"Mach-O overlapping sections", mach, func(b []byte) { binary.LittleEndian.PutUint32(b[32+72+2*80+48:], 16384) }},
		{"Mach-O huge symbol count", mach, func(b []byte) { binary.LittleEndian.PutUint32(b[32+(72+3*80)+72+12:], 0xffffffff) }},
		{"truncated PE", image[:63], nil},
		{"bad PE signature", image, func(b []byte) { b[0x80] = 0 }},
		{"signed PE", image, func(b []byte) { binary.LittleEndian.PutUint32(b[0x80+24+112+4*8+4:], 128) }},
		{"PE overlapping sections", image, func(b []byte) { binary.LittleEndian.PutUint32(b[0x80+24+240+2*40+20:], 1024) }},
		{"PE invalid directory size", image, func(b []byte) { binary.LittleEndian.PutUint32(b[0x80+24+112+6*8+4:], 27) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := bytes.Clone(tt.raw)
			if tt.edit != nil {
				tt.edit(raw)
			}
			original := bytes.Clone(raw)
			if _, err := Compress(raw); err == nil {
				t.Fatal("malformed or unsupported artifact was accepted")
			}
			if !bytes.Equal(raw, original) {
				t.Fatal("failed compression changed the input")
			}
		})
	}
}

func TestPESectionDefinitionSymbol(t *testing.T) {
	raw := peFixture(t, 64)
	const coff = 0x84
	symoff := int(binary.LittleEndian.Uint32(raw[coff+8:]))
	strings := bytes.Clone(raw[symoff:])
	symbols := make([]byte, 36)
	binary.LittleEndian.PutUint32(symbols[4:], 4+uint32(len(".debug_abbrev\x00"))) // .debug_info
	binary.LittleEndian.PutUint16(symbols[12:], 3)
	symbols[16], symbols[17] = 3, 1 // static section definition with one auxiliary entry
	raw = append(raw[:symoff], symbols...)
	raw = append(raw, strings...)
	binary.LittleEndian.PutUint32(raw[coff+12:], 2)
	out, err := Compress(raw)
	if err != nil {
		t.Fatal(err)
	}
	f, err := pe.NewFile(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Symbols) != 1 || f.Symbols[0].Name != ".zdebug_info" {
		t.Fatalf("section-definition symbol did not follow renamed section: %v", f.Symbols)
	}
	newOff := int(binary.LittleEndian.Uint32(out[coff+8:]))
	data, err := f.Section(".zdebug_info").Data()
	if err != nil {
		t.Fatal(err)
	}
	payload := bytes.NewReader(data[12:])
	z, err := zlib.NewReader(payload)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(io.Discard, z); err != nil {
		t.Fatal(err)
	}
	z.Close()
	if size := binary.LittleEndian.Uint32(out[newOff+18:]); size != uint32(len(data)-payload.Len()) {
		t.Fatal("section-definition auxiliary length was not updated")
	}
}

func FuzzCompress(f *testing.F) {
	f.Add(machoFixtureSize(f, 4))
	f.Add(peFixtureSize(f, 32, 4))
	f.Add(peFixtureSize(f, 64, 4))
	f.Add([]byte("MZ"))
	f.Add([]byte{0xcf, 0xfa, 0xed, 0xfe})
	f.Fuzz(func(t *testing.T, raw []byte) {
		original := bytes.Clone(raw)
		_, _ = Compress(raw)
		if !bytes.Equal(raw, original) {
			t.Fatal("compression mutated input")
		}
	})
}
