// Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
// Licensed under the Apache License, Version 2.0.

// Package dwarfcompress writes GNU-style zlib DWARF sections in native
// debugger artifacts. It operates on linked PE images and thin Mach-O dSYMs,
// after DWARF relocations have been resolved. ELF compression belongs to the
// native linker instead.
package dwarfcompress

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"math"
	"sort"
)

// Compress returns a new artifact with compressible DWARF sections encoded
// using the GNU ZLIB header. Already compressed sections and sections which
// would grow are left alone. The input is never modified, including on error.
func Compress(raw []byte) ([]byte, error) {
	if len(raw) >= 2 && string(raw[:2]) == "MZ" {
		return compressPE(raw)
	}
	if len(raw) >= 4 && binary.LittleEndian.Uint32(raw) == 0xfeedfacf {
		return compressMachO(raw)
	}
	return nil, fmt.Errorf("DWARF compression requires a PE image or a thin 64-bit Mach-O dSYM")
}

// compressed returns GNU-wrapped zlib data, or nil when it would not be smaller
// than the input.
func compressed(data []byte) []byte {
	var out bytes.Buffer
	out.WriteString("ZLIB")
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(data)))
	out.Write(size[:])
	w := zlib.NewWriter(&out)
	// bytes.Buffer cannot fail writes, including zlib's final flush.
	_, _ = w.Write(data)
	_ = w.Close()
	if out.Len() >= len(data) {
		return nil
	}
	return out.Bytes()
}

func inFile(raw []byte, off, size uint64) bool {
	return off <= uint64(len(raw)) && size <= uint64(len(raw))-off
}

func align(n, alignment uint64) uint64 {
	return (n + alignment - 1) &^ (alignment - 1)
}

type edit struct {
	start, end int
	data       []byte
}

type layout struct {
	edits        []edit
	size         int
	originalSize int
}

func newLayout(raw []byte, edits []edit) (*layout, error) {
	sort.Slice(edits, func(i, j int) bool { return edits[i].start < edits[j].start })
	size, end := int64(len(raw)), 0
	for _, e := range edits {
		if e.start < end || e.end < e.start || e.end > len(raw) {
			return nil, fmt.Errorf("overlapping or invalid native artifact ranges")
		}
		size += int64(len(e.data)) - int64(e.end-e.start)
		if size < 0 || size > math.MaxUint32 {
			return nil, fmt.Errorf("compressed native artifact exceeds 32-bit file offsets")
		}
		end = e.end
	}
	return &layout{edits: edits, size: int(size), originalSize: len(raw)}, nil
}

func (l *layout) offset(off uint64) (uint64, error) {
	if off > uint64(l.originalSize) {
		return 0, fmt.Errorf("native artifact file offset %#x is outside the input", off)
	}
	shift := int64(0)
	for _, e := range l.edits {
		if off <= uint64(e.start) {
			break
		}
		if off < uint64(e.end) {
			return 0, fmt.Errorf("file offset %#x points inside a rewritten DWARF range", off)
		}
		shift += int64(e.end-e.start) - int64(len(e.data))
	}
	value := int64(off) - shift
	if value < 0 || value > math.MaxUint32 {
		return 0, fmt.Errorf("native artifact file offset overflows")
	}
	return uint64(value), nil
}

func (l *layout) apply(raw []byte) []byte {
	out := make([]byte, 0, l.size)
	pos := 0
	for _, e := range l.edits {
		out = append(out, raw[pos:e.start]...)
		out = append(out, e.data...)
		pos = e.end
	}
	return append(out, raw[pos:]...)
}

func (l *layout) pointer32(raw []byte, pos int) error {
	off := binary.LittleEndian.Uint32(raw[pos:])
	if off == 0 {
		return nil
	}
	next, err := l.offset(uint64(off))
	if err != nil {
		return err
	}
	binary.LittleEndian.PutUint32(raw[pos:], uint32(next))
	return nil
}

func (l *layout) pointer64(raw []byte, pos int) error {
	off := binary.LittleEndian.Uint64(raw[pos:])
	if off == 0 {
		return nil
	}
	next, err := l.offset(off)
	if err != nil {
		return err
	}
	binary.LittleEndian.PutUint64(raw[pos:], next)
	return nil
}
