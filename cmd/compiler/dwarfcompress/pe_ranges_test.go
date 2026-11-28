// Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
// Licensed under the Apache License, Version 2.0.

package dwarfcompress

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"fmt"
	"math"
	"testing"
)

const fixtureCOFF = 0x84
const fixturePEOptional = fixtureCOFF + 20
const fixturePESections = fixturePEOptional + 240

func renamePESection(raw []byte, index int, name string) []byte {
	p := fixturePESections + index*40
	clear(raw[p : p+8])
	if len(name) > 8 {
		str := binary.LittleEndian.Uint32(raw[fixtureCOFF+8:]) + binary.LittleEndian.Uint32(raw[fixtureCOFF+12:])*18
		size := binary.LittleEndian.Uint32(raw[str:])
		raw = append(raw, name...)
		raw = append(raw, 0)
		binary.LittleEndian.PutUint32(raw[str:], size+uint32(len(name)+1))
		name = "/" + decimal(size)
	}
	copy(raw[p:p+8], name)
	return raw
}

func TestPEWithoutSymbolTable(t *testing.T) {
	for _, bits := range []int{32, 64} {
		for _, tt := range []struct {
			name    string
			repeats int
			dwarf   bool
			nsym    uint32
			wantErr string
		}{
			{"no DWARF", 8000, false, 0, ""},
			{"no compression gain", 0, true, 0, ""},
			{"compressed names need a table", 8000, true, 0, "PE DWARF section names require a COFF string table"},
			{"symbol count without table", 8000, false, 1, "PE symbol count without a symbol table"},
		} {
			t.Run(fmt.Sprintf("PE%d/%s", bits, tt.name), func(t *testing.T) {
				raw := peFixtureSize(t, bits, tt.repeats)
				// Use inline names, then remove the table entirely. Even the
				// original .debug_* names must not depend on a string table.
				sections := fixturePEOptional + int(binary.LittleEndian.Uint16(raw[fixtureCOFF+16:]))
				for i, name := range []string{".text", ".debug_a", ".debug_i", ".debug_s", ".rdata"} {
					if !tt.dwarf && i > 0 && i < 4 {
						name = fmt.Sprintf(".data%d", i)
					}
					p := sections + i*40
					clear(raw[p : p+8])
					copy(raw[p:p+8], name)
				}
				raw = raw[:binary.LittleEndian.Uint32(raw[fixtureCOFF+8:])]
				binary.LittleEndian.PutUint32(raw[fixtureCOFF+8:], 0)
				binary.LittleEndian.PutUint32(raw[fixtureCOFF+12:], tt.nsym)
				original := bytes.Clone(raw)
				out, err := Compress(raw)
				if tt.wantErr != "" {
					if err == nil || err.Error() != tt.wantErr || out != nil {
						t.Fatalf("got output length %d, error %v; want %q without output", len(out), err, tt.wantErr)
					}
				} else if err != nil || !bytes.Equal(out, raw) {
					t.Fatalf("image without a COFF table should remain unchanged: %v", err)
				}
				if !bytes.Equal(raw, original) {
					t.Fatal("compression modified the input without a COFF table")
				}
			})
		}
	}
}

func TestPEMetadataRanges(t *testing.T) {
	u32 := binary.LittleEndian.PutUint32
	u16 := binary.LittleEndian.PutUint16
	tests := []struct {
		name string
		edit func([]byte) []byte
	}{
		{"duplicate compressed section", func(b []byte) []byte { return renamePESection(b, 3, ".zdebug_info") }},
		{"unrepresentable name offset", func(b []byte) []byte {
			str := binary.LittleEndian.Uint32(b[fixtureCOFF+8:])
			b = append(b, make([]byte, 10_000_000)...)
			u32(b[str:], uint32(len(b))-str)
			return b
		}},
		{"name table overlaps mapped data", func(b []byte) []byte {
			str := binary.LittleEndian.Uint32(b[fixtureCOFF+8:])
			copy(b[700:], b[str:])
			u32(b[fixtureCOFF+8:], 700)
			return b
		}},
		{"debug directory inside rewritten bytes", func(b []byte) []byte {
			u32(b[fixturePEOptional+112+6*8:], binary.LittleEndian.Uint32(b[fixturePESections+2*40+12:])+1)
			return b
		}},
		{"CodeView pointer inside rewritten bytes", func(b []byte) []byte {
			rdata := binary.LittleEndian.Uint32(b[fixturePESections+4*40+20:])
			u32(b[rdata+24:], binary.LittleEndian.Uint32(b[fixturePESections+2*40+20:])+1)
			return b
		}},
		{"debug symbol", func(b []byte) []byte {
			symoff := binary.LittleEndian.Uint32(b[fixtureCOFF+8:])
			symbol := make([]byte, 18)
			copy(symbol, "debug")
			u32(symbol[8:], 1)
			u16(symbol[12:], 3)
			symbol[16] = 2
			str := bytes.Clone(b[symoff:])
			b = append(b[:symoff], symbol...)
			b = append(b, str...)
			u32(b[fixtureCOFF+12:], 1)
			return b
		}},
		{"auxiliary symbol count", func(b []byte) []byte {
			symoff := binary.LittleEndian.Uint32(b[fixtureCOFF+8:])
			symbol := make([]byte, 18)
			copy(symbol, "debug")
			symbol[17] = 3
			str := bytes.Clone(b[symoff:])
			b = append(b[:symoff], symbol...)
			b = append(b, str...)
			u32(b[fixtureCOFF+12:], 1)
			return b
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := tt.edit(peFixture(t, 64))
			before := bytes.Clone(raw)
			if _, err := Compress(raw); err == nil {
				t.Fatal("invalid PE metadata accepted")
			}
			if !bytes.Equal(raw, before) {
				t.Fatal("failed compression changed the image")
			}
		})
	}
	for _, name := range []string{"zero virtual size", "no debug directory", "empty debug directory"} {
		t.Run(name, func(t *testing.T) {
			raw := peFixture(t, 64)
			size := binary.LittleEndian.Uint32(raw[fixturePESections+2*40+16:])
			switch name {
			case "zero virtual size":
				u32(raw[fixturePESections+2*40+8:], 0)
			case "no debug directory":
				const optionalSize = 112 + 6*8
				copy(raw[fixturePEOptional+optionalSize:], raw[fixturePESections:fixturePESections+5*40])
				binary.LittleEndian.PutUint16(raw[fixtureCOFF+16:], optionalSize)
				u32(raw[fixturePEOptional+108:], 6)
			case "empty debug directory":
				u32(raw[fixturePEOptional+112+6*8+4:], 0)
			}
			out, err := Compress(raw)
			if err != nil {
				t.Fatal(err)
			}
			if name == "zero virtual size" {
				f, err := pe.NewFile(bytes.NewReader(out))
				if err != nil || f.Section(".zdebug_info").VirtualSize != size {
					t.Fatalf("zero VirtualSize lost the original mapped raw extent: %v", err)
				}
			}
		})
	}
	raw := renamePESection(peFixtureSize(t, 64, 64), 3, ".padding")
	if out, err := Compress(raw); err != nil || !bytes.Equal(out, raw) {
		t.Fatalf("string-table growth must not inflate a small PE: %v", err)
	}
}

func TestLayoutOffsetBoundaries(t *testing.T) {
	l := &layout{originalSize: 10, edits: []edit{{start: 2, end: 8, data: []byte{0}}}}
	for _, off := range []uint64{3, 11, math.MaxUint64} {
		if _, err := l.offset(off); err == nil {
			t.Fatalf("accepted rewritten/out-of-file offset %d", off)
		}
	}
	if uint64(^uint(0)) > math.MaxUint32 {
		wide := uint64(math.MaxUint32) + 1
		l := &layout{originalSize: int(wide)}
		if _, err := l.offset(wide); err == nil {
			t.Fatal("accepted file offset above the 32-bit limit")
		}
	}
}
