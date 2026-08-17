/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package pclnmap encodes LLGo's optional external runtime-symbolization map.
// The runtime has a deliberately dependency-free decoder for the same format;
// keep the layout constants and validation rules in sync with
// runtime/internal/lib/runtime/pclntab_external.go.
package pclnmap

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/xgo-dev/llgo/internal/build/funcinfo"
)

const (
	Magic      = "LLGOPCL1"
	Version    = uint32(4)
	ABIVersion = uint32(1)
	HeaderSize = uint32(256)

	EndianLittle = uint8(1)

	GOOSLinux  = uint8(1)
	GOOSDarwin = uint8(2)

	GOARCHAMD64 = uint8(1)
	GOARCHARM64 = uint8(2)
)

const (
	headerMagic       = 0
	headerVersion     = 8
	headerSize        = 12
	headerPointerSize = 16
	headerEndian      = 17
	headerGOOS        = 18
	headerGOARCH      = 19
	headerABIVersion  = 20
	headerFileSize    = 24
	headerPayloadHash = 32
	headerImageBase   = 40
	headerTextStart   = 48
	headerTextEnd     = 56
	headerIdentity    = 64
	headerDescriptors = 96
)

const (
	descRecords = iota
	descPCLines
	descStrings
	descStringOffsets
	descHash
	descSymbolIndex
	descEntrySites
	descPCSites
	descCount
)

const (
	recordSize       = uint64(funcinfo.EncodedRecordSize)
	pcLineSize       = uint64(funcinfo.EncodedPCLineRecordSize)
	stringOffsetSize = uint64(4)
	hashSize         = uint64(2)
	symbolIndexSize  = uint64(16)
	siteSize         = uint64(16)
)

// SymbolIndexEntry maps the stable symbol hash carried by a link-phase site
// record to the one-based funcinfo table index used by the runtime.
type SymbolIndexEntry struct {
	SymbolID  uint64
	FuncIndex uint32
}

// Site is a final linked PC expressed relative to the image base. ID is a
// symbol ID for EntrySites and a pcline record ID for PCSites.
type Site struct {
	PCOffset uint64
	ID       uint64
}

// Data is the complete immutable payload needed by the external runtime
// loader. EntrySites are physical function entries normalized and
// LTO-deduplicated by the post-link analyzer. Compiler-generated wrappers and
// adapters use ordinary function records and entry sites; calling conventions
// and closure environment transport are deliberately not represented here.
type Data struct {
	GOOS        string
	GOARCH      string
	PointerSize int
	ImageBase   uint64
	TextStart   uint64
	TextEnd     uint64
	Identity    [32]byte
	Table       funcinfo.Table
	SymbolIndex []SymbolIndexEntry
	EntrySites  []Site
	PCSites     []Site
}

// View describes validated sections in an encoded map. It is primarily used
// by tests and build tooling; the runtime publishes zero-copy typed views.
type View struct {
	GOOSCode, GOARCHCode uint8
	PointerSize          uint8
	ImageBase            uint64
	TextStartOffset      uint64
	TextEndOffset        uint64
	Identity             [32]byte
	Sections             [descCount]Section
}

type Section struct {
	Offset uint64
	Count  uint64
}

func targetCodes(goos, goarch string) (uint8, uint8, error) {
	var osCode uint8
	switch goos {
	case "linux":
		osCode = GOOSLinux
	case "darwin":
		osCode = GOOSDarwin
	default:
		return 0, 0, fmt.Errorf("unsupported GOOS %q", goos)
	}
	var archCode uint8
	switch goarch {
	case "amd64":
		archCode = GOARCHAMD64
	case "arm64":
		archCode = GOARCHARM64
	default:
		return 0, 0, fmt.Errorf("unsupported GOARCH %q", goarch)
	}
	return osCode, archCode, nil
}

func align(v, alignment uint64) uint64 {
	return (v + alignment - 1) &^ (alignment - 1)
}

func checkedBytes(count, size uint64) (uint64, error) {
	if size != 0 && count > math.MaxUint64/size {
		return 0, errors.New("pclntab section size overflows")
	}
	return count * size, nil
}

func putDescriptor(dst []byte, index int, off, count uint64) {
	base := headerDescriptors + index*16
	binary.LittleEndian.PutUint64(dst[base:], off)
	binary.LittleEndian.PutUint64(dst[base+8:], count)
}

// Encode returns one deterministic, versioned sidecar image.
func Encode(data Data) ([]byte, error) {
	osCode, archCode, err := targetCodes(data.GOOS, data.GOARCH)
	if err != nil {
		return nil, err
	}
	if data.PointerSize != 8 {
		return nil, fmt.Errorf("unsupported pointer size %d", data.PointerSize)
	}
	if data.TextStart < data.ImageBase || data.TextEnd <= data.TextStart {
		return nil, fmt.Errorf("invalid text range [%#x,%#x) for image base %#x", data.TextStart, data.TextEnd, data.ImageBase)
	}

	type sectionInput struct {
		count uint64
		size  uint64
		align uint64
	}
	inputs := [descCount]sectionInput{
		{uint64(len(data.Table.Records)), recordSize, 4},
		{uint64(len(data.Table.PCLines)), pcLineSize, 8},
		{uint64(len(data.Table.Strings)), 1, 1},
		{uint64(len(data.Table.StringOffsets)), stringOffsetSize, 4},
		{uint64(len(data.Table.Hash)), hashSize, 2},
		{uint64(len(data.SymbolIndex)), symbolIndexSize, 8},
		{uint64(len(data.EntrySites)), siteSize, 8},
		{uint64(len(data.PCSites)), siteSize, 8},
	}

	var sections [descCount]Section
	off := uint64(HeaderSize)
	for i, in := range inputs {
		off = align(off, in.align)
		n, err := checkedBytes(in.count, in.size)
		if err != nil || off > math.MaxUint64-n {
			return nil, errors.New("pclntab map is too large")
		}
		sections[i] = Section{Offset: off, Count: in.count}
		off += n
	}
	if off > uint64(math.MaxInt) {
		return nil, errors.New("pclntab map exceeds addressable memory")
	}
	out := make([]byte, int(off))
	copy(out[headerMagic:], Magic)
	binary.LittleEndian.PutUint32(out[headerVersion:], Version)
	binary.LittleEndian.PutUint32(out[headerSize:], HeaderSize)
	out[headerPointerSize] = byte(data.PointerSize)
	out[headerEndian] = EndianLittle
	out[headerGOOS] = osCode
	out[headerGOARCH] = archCode
	binary.LittleEndian.PutUint32(out[headerABIVersion:], ABIVersion)
	binary.LittleEndian.PutUint64(out[headerFileSize:], uint64(len(out)))
	binary.LittleEndian.PutUint64(out[headerImageBase:], data.ImageBase)
	binary.LittleEndian.PutUint64(out[headerTextStart:], data.TextStart-data.ImageBase)
	binary.LittleEndian.PutUint64(out[headerTextEnd:], data.TextEnd-data.ImageBase)
	copy(out[headerIdentity:headerIdentity+len(data.Identity)], data.Identity[:])
	for i, sec := range sections {
		putDescriptor(out, i, sec.Offset, sec.Count)
	}

	pos := sections[descRecords].Offset
	for _, rec := range data.Table.Records {
		p := out[pos : pos+recordSize]
		binary.LittleEndian.PutUint32(p[0:], rec.SymbolPkg)
		binary.LittleEndian.PutUint32(p[4:], rec.SymbolName)
		binary.LittleEndian.PutUint32(p[8:], rec.NamePkg)
		binary.LittleEndian.PutUint32(p[12:], rec.NameName)
		binary.LittleEndian.PutUint32(p[16:], rec.FileRoot)
		binary.LittleEndian.PutUint32(p[20:], rec.FileName)
		binary.LittleEndian.PutUint32(p[24:], rec.Line)
		pos += recordSize
	}
	pos = sections[descPCLines].Offset
	for _, rec := range data.Table.PCLines {
		p := out[pos : pos+pcLineSize]
		binary.LittleEndian.PutUint64(p[0:], rec.ID)
		binary.LittleEndian.PutUint32(p[8:], rec.Func)
		binary.LittleEndian.PutUint32(p[12:], rec.FileRoot)
		binary.LittleEndian.PutUint32(p[16:], rec.FileName)
		binary.LittleEndian.PutUint32(p[20:], rec.Line)
		pos += pcLineSize
	}
	copy(out[sections[descStrings].Offset:], data.Table.Strings)
	pos = sections[descStringOffsets].Offset
	for _, value := range data.Table.StringOffsets {
		binary.LittleEndian.PutUint32(out[pos:], value)
		pos += stringOffsetSize
	}
	pos = sections[descHash].Offset
	for _, value := range data.Table.Hash {
		binary.LittleEndian.PutUint16(out[pos:], value)
		pos += hashSize
	}
	pos = sections[descSymbolIndex].Offset
	for _, value := range data.SymbolIndex {
		binary.LittleEndian.PutUint64(out[pos:], value.SymbolID)
		binary.LittleEndian.PutUint32(out[pos+8:], value.FuncIndex)
		pos += symbolIndexSize
	}
	writeSites := func(sec Section, sites []Site) {
		pos := sec.Offset
		for _, site := range sites {
			binary.LittleEndian.PutUint64(out[pos:], site.PCOffset)
			binary.LittleEndian.PutUint64(out[pos+8:], site.ID)
			pos += siteSize
		}
	}
	writeSites(sections[descEntrySites], data.EntrySites)
	writeSites(sections[descPCSites], data.PCSites)
	binary.LittleEndian.PutUint64(out[headerPayloadHash:], fnv64(out[HeaderSize:]))
	return out, nil
}

func fnv64(data []byte) uint64 {
	const (
		offset = uint64(14695981039346656037)
		prime  = uint64(1099511628211)
	)
	h := offset
	for _, b := range data {
		h ^= uint64(b)
		h *= prime
	}
	return h
}

func descriptorSize(index int) uint64 {
	switch index {
	case descRecords:
		return recordSize
	case descPCLines:
		return pcLineSize
	case descStrings:
		return 1
	case descStringOffsets:
		return stringOffsetSize
	case descHash:
		return hashSize
	case descSymbolIndex:
		return symbolIndexSize
	case descEntrySites, descPCSites:
		return siteSize
	default:
		return 0
	}
}

func descriptorAlignment(index int) uint64 {
	switch index {
	case descRecords, descStringOffsets:
		return 4
	case descPCLines, descSymbolIndex, descEntrySites, descPCSites:
		return 8
	case descHash:
		return 2
	default:
		return 1
	}
}

// Parse validates the common header, checksum and all section bounds.
func Parse(raw []byte) (View, error) {
	var view View
	if len(raw) < int(HeaderSize) || string(raw[headerMagic:headerMagic+len(Magic)]) != Magic {
		return view, errors.New("invalid pclntab map magic")
	}
	if binary.LittleEndian.Uint32(raw[headerVersion:]) != Version || binary.LittleEndian.Uint32(raw[headerSize:]) != HeaderSize {
		return view, errors.New("unsupported pclntab map version")
	}
	if binary.LittleEndian.Uint32(raw[headerABIVersion:]) != ABIVersion {
		return view, errors.New("unsupported pclntab ABI version")
	}
	if binary.LittleEndian.Uint64(raw[headerFileSize:]) != uint64(len(raw)) {
		return view, errors.New("pclntab map size mismatch")
	}
	if raw[headerEndian] != EndianLittle || raw[headerPointerSize] != 8 {
		return view, errors.New("unsupported pclntab map target encoding")
	}
	if binary.LittleEndian.Uint64(raw[headerPayloadHash:]) != fnv64(raw[HeaderSize:]) {
		return view, errors.New("pclntab map checksum mismatch")
	}
	view.PointerSize = raw[headerPointerSize]
	view.GOOSCode = raw[headerGOOS]
	view.GOARCHCode = raw[headerGOARCH]
	view.ImageBase = binary.LittleEndian.Uint64(raw[headerImageBase:])
	view.TextStartOffset = binary.LittleEndian.Uint64(raw[headerTextStart:])
	view.TextEndOffset = binary.LittleEndian.Uint64(raw[headerTextEnd:])
	copy(view.Identity[:], raw[headerIdentity:headerIdentity+len(view.Identity)])
	if view.TextEndOffset <= view.TextStartOffset {
		return View{}, errors.New("invalid pclntab text range")
	}
	for i := range view.Sections {
		base := headerDescriptors + i*16
		sec := Section{Offset: binary.LittleEndian.Uint64(raw[base:]), Count: binary.LittleEndian.Uint64(raw[base+8:])}
		n, err := checkedBytes(sec.Count, descriptorSize(i))
		if err != nil || sec.Offset < uint64(HeaderSize) || sec.Offset > uint64(len(raw)) || n > uint64(len(raw))-sec.Offset {
			return View{}, fmt.Errorf("invalid pclntab section %d", i)
		}
		if alignment := descriptorAlignment(i); sec.Offset&(alignment-1) != 0 {
			return View{}, fmt.Errorf("misaligned pclntab section %d", i)
		}
		view.Sections[i] = sec
	}
	for i := range view.Sections {
		left := view.Sections[i]
		if left.Count == 0 {
			continue
		}
		leftEnd := left.Offset + left.Count*descriptorSize(i)
		for j := i + 1; j < len(view.Sections); j++ {
			right := view.Sections[j]
			if right.Count == 0 {
				continue
			}
			rightEnd := right.Offset + right.Count*descriptorSize(j)
			if left.Offset < rightEnd && right.Offset < leftEnd {
				return View{}, fmt.Errorf("overlapping pclntab sections %d and %d", i, j)
			}
		}
	}
	return view, nil
}
