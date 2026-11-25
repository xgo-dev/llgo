// Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
// Licensed under the Apache License, Version 2.0.

package dwarfcompress

import (
	"encoding/binary"
)

func dwarfFixture(repeats int) (abbrev, info, str []byte) {
	// One DWARF 4 compilation unit with independently decoded inline strings.
	abbrev = []byte{1, 0x11, 0, 3, 8, 0x25, 8, 0, 0, 0}
	info = []byte{0, 0, 0, 0, 4, 0, 0, 0, 0, 0, 8, 1}
	info = append(info, []byte("fixture.go\x00")...)
	for range repeats {
		info = append(info, "producer "...)
	}
	info = append(info, 0)
	binary.LittleEndian.PutUint32(info, uint32(len(info)-4))
	for range repeats {
		str = append(str, "unused string\x00"...)
	}
	return
}

func machoFixture(t interface{ Helper() }) []byte {
	return machoFixtureSize(t, 8000)
}

func machoFixtureSize(t interface{ Helper() }, repeats int) []byte {
	t.Helper()
	abbrev, info, str := dwarfFixture(repeats)
	const start = 16384
	const nsec = 3
	const dwarfCommand = 72 + nsec*80
	const commands = dwarfCommand + 72 + 24 + 24
	data := [][]byte{abbrev, info, str}
	names := []string{"__debug_abbrev", "__debug_info", "__debug_str"}
	end := start
	for _, d := range data {
		end += len(d)
	}
	linkedit := ((end + 16383) / 16384) * 16384
	symbolStrings := []byte("\x00probe\x00")
	raw := make([]byte, linkedit+16+len(symbolStrings))
	u32 := func(p int, v uint32) { binary.LittleEndian.PutUint32(raw[p:], v) }
	u64 := func(p int, v uint64) { binary.LittleEndian.PutUint64(raw[p:], v) }
	u32(0, 0xfeedfacf)
	u32(4, 0x100000c)
	u32(12, 0xa)
	u32(16, 4)
	u32(20, commands)
	p := 32
	u32(p, 0x19)
	u32(p+4, dwarfCommand)
	copy(raw[p+8:], "__DWARF")
	u64(p+24, 0x10000)
	u64(p+32, uint64(linkedit-start))
	u64(p+40, start)
	u64(p+48, uint64(linkedit-start))
	u32(p+64, nsec)
	off := start
	for i, d := range data {
		s := p + 72 + i*80
		copy(raw[s:s+16], names[i])
		copy(raw[s+16:s+32], "__DWARF")
		u64(s+32, 0x10000+uint64(off-start))
		u64(s+40, uint64(len(d)))
		u32(s+48, uint32(off))
		u32(s+64, 0x02000000)
		copy(raw[off:], d)
		off += len(d)
	}
	p += dwarfCommand
	u32(p, 0x19)
	u32(p+4, 72)
	copy(raw[p+8:], "__LINKEDIT")
	u64(p+24, 0x10000+uint64(linkedit-start))
	u64(p+32, 16384)
	u64(p+40, uint64(linkedit))
	u64(p+48, uint64(16+len(symbolStrings)))
	p += 72
	u32(p, 2)
	u32(p+4, 24)
	u32(p+8, uint32(linkedit))
	u32(p+12, 1)
	u32(p+16, uint32(linkedit+16))
	u32(p+20, uint32(len(symbolStrings)))
	p += 24
	u32(p, 0x1b)
	u32(p+4, 24)
	for i := range 16 {
		raw[p+8+i] = byte(i)
	}
	u32(linkedit, 1)
	raw[linkedit+4] = 3 // N_ABS | N_EXT
	u64(linkedit+8, 0x1234)
	copy(raw[linkedit+16:], symbolStrings)
	return raw
}

func peFixture(t interface{ Helper() }, bits int) []byte {
	return peFixtureSize(t, bits, 8000)
}

func peFixtureSize(t interface{ Helper() }, bits, repeats int) []byte {
	t.Helper()
	abbrev, info, str := dwarfFixture(repeats)
	data := [][]byte{[]byte{0xc3}, abbrev, info, str, make([]byte, 64)}
	names := []string{".text", ".debug_abbrev", ".debug_info", ".debug_str", ".rdata"}
	const peoff = 0x80
	const coff = peoff + 4
	optionalSize, directory := 240, 112
	if bits == 32 {
		optionalSize, directory = 224, 96
	}
	optional := coff + 20
	sections := optional + optionalSize
	start := ((sections + len(data)*40 + 511) / 512) * 512
	offsets := make([]int, len(data))
	sizes := make([]int, len(data))
	end := start
	for i, d := range data {
		offsets[i] = end
		sizes[i] = ((len(d) + 511) / 512) * 512
		end += sizes[i]
	}
	stringsData := []byte{4, 0, 0, 0}
	strNames := make([]uint32, len(data))
	for i, name := range names {
		if len(name) > 8 {
			strNames[i] = uint32(len(stringsData))
			stringsData = append(stringsData, name...)
			stringsData = append(stringsData, 0)
		}
	}
	binary.LittleEndian.PutUint32(stringsData, uint32(len(stringsData)))
	raw := make([]byte, end+len(stringsData))
	u16 := func(p int, v uint16) { binary.LittleEndian.PutUint16(raw[p:], v) }
	u32 := func(p int, v uint32) { binary.LittleEndian.PutUint32(raw[p:], v) }
	copy(raw, "MZ")
	u32(60, peoff)
	copy(raw[peoff:], "PE\x00\x00")
	u16(coff, 0x8664)
	if bits == 32 {
		u16(coff, 0x14c)
	}
	u16(coff+2, uint16(len(data)))
	u32(coff+8, uint32(end))
	u16(coff+16, uint16(optionalSize))
	u16(coff+18, 2)
	u16(optional, 0x20b)
	if bits == 32 {
		u16(optional, 0x10b)
	}
	u32(optional+4, uint32(sizes[0]))
	u32(optional+8, uint32(end-start-sizes[0]))
	u32(optional+16, 0x1000)
	u32(optional+32, 4096)
	u32(optional+36, 512)
	u32(optional+56, 0x100000)
	u32(optional+60, uint32(start))
	u32(optional+directory-4, 16)
	rva := uint32(0x1000)
	for i, d := range data {
		p := sections + i*40
		if strNames[i] != 0 {
			copy(raw[p:p+8], "/"+decimal(strNames[i]))
		} else {
			copy(raw[p:p+8], names[i])
		}
		u32(p+8, uint32(len(d)))
		u32(p+12, rva)
		u32(p+16, uint32(sizes[i]))
		u32(p+20, uint32(offsets[i]))
		u32(p+36, 0x40000040)
		if i == 0 {
			u32(p+36, 0x60000020)
		}
		copy(raw[offsets[i]:], d)
		if i == len(data)-1 {
			u32(optional+directory+6*8, rva)
			u32(optional+directory+6*8+4, 28)
			u32(offsets[i]+12, 2) // IMAGE_DEBUG_TYPE_CODEVIEW
			u32(offsets[i]+16, 5)
			u32(offsets[i]+20, rva+28)
			u32(offsets[i]+24, uint32(offsets[i]+28))
			copy(raw[offsets[i]+28:], "RSDS\x00")
		}
		rva += uint32(((len(d) + 4095) / 4096) * 4096)
	}
	copy(raw[end:], stringsData)
	return raw
}

func decimal(v uint32) string {
	if v < 10 {
		return string(rune('0' + v))
	}
	return decimal(v/10) + string(rune('0'+v%10))
}
