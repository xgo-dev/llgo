// Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
// Licensed under the Apache License, Version 2.0.

package dwarfcompress

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Sizes and byte offsets within the PE/COFF on-disk headers.
const (
	peDOSHeaderSize      = 64
	peHeaderPointer      = 60 // IMAGE_DOS_HEADER.e_lfanew
	peSignatureSize      = 4
	peCOFFHeaderSize     = 20
	peOptionalMagic32    = 0x10b
	peOptionalMagic64    = 0x20b
	peSectionHeaderSize  = 40
	peSectionNameSize    = 8
	peDirectoryEntrySize = 8
	peDirectorySizeField = 4

	// IMAGE_FILE_HEADER
	coffSectionCount       = 2
	coffSymbolTable        = 8
	coffSymbolCount        = 12
	coffOptionalHeaderSize = 16

	// IMAGE_OPTIONAL_HEADER32 / IMAGE_OPTIONAL_HEADER64
	optionalInitializedData = 8
	optionalFileAlignment   = 36
	optionalChecksum        = 64
	optional32Directories   = 96
	optional64Directories   = 112

	// IMAGE_SECTION_HEADER
	sectionVirtualSize = 8
	sectionRawSize     = 16
	sectionRawData     = 20
	sectionRelocations = 24
	sectionLineNumbers = 28

	// IMAGE_SYMBOL
	symbolNameOffset   = 4
	symbolValue        = 8
	symbolSection      = 12
	symbolType         = 14
	symbolStorageClass = 16
	symbolAuxCount     = 17
	symbolClassStatic  = 3 // IMAGE_SYM_CLASS_STATIC

	// IMAGE_DEBUG_DIRECTORY
	peDebugDirectorySize  = 28
	debugDirectoryRawData = 24 // PointerToRawData
)

func compressPE(raw []byte) ([]byte, error) {
	if len(raw) < peDOSHeaderSize {
		return nil, fmt.Errorf("truncated PE DOS header")
	}
	header := uint64(binary.LittleEndian.Uint32(raw[peHeaderPointer:]))
	if !inFile(raw, header, peSignatureSize+peCOFFHeaderSize) || string(raw[header:header+peSignatureSize]) != "PE\x00\x00" {
		return nil, fmt.Errorf("invalid PE signature")
	}
	coff := int(header) + peSignatureSize
	nsect := int(binary.LittleEndian.Uint16(raw[coff+coffSectionCount:]))
	optionalSize := int(binary.LittleEndian.Uint16(raw[coff+coffOptionalHeaderSize:]))
	optional := coff + peCOFFHeaderSize
	sections := optional + optionalSize
	if optionalSize < optional32Directories || !inFile(raw, uint64(optional), uint64(optionalSize+nsect*peSectionHeaderSize)) {
		return nil, fmt.Errorf("truncated PE optional/section headers")
	}
	var directoryOffset int
	switch binary.LittleEndian.Uint16(raw[optional:]) {
	case peOptionalMagic32:
		directoryOffset = optional32Directories
	case peOptionalMagic64:
		directoryOffset = optional64Directories
	default:
		return nil, fmt.Errorf("unsupported PE optional header")
	}
	if optionalSize < directoryOffset {
		return nil, fmt.Errorf("truncated PE data directories")
	}
	ndirs := int(binary.LittleEndian.Uint32(raw[optional+directoryOffset-peDirectorySizeField:]))
	if ndirs > (optionalSize-directoryOffset)/peDirectoryEntrySize {
		return nil, fmt.Errorf("truncated PE data directories")
	}
	if ndirs > pe.IMAGE_DIRECTORY_ENTRY_SECURITY {
		certificate := optional + directoryOffset + pe.IMAGE_DIRECTORY_ENTRY_SECURITY*peDirectoryEntrySize
		if binary.LittleEndian.Uint32(raw[certificate+peDirectorySizeField:]) != 0 {
			return nil, fmt.Errorf("cannot rewrite an Authenticode-signed PE image")
		}
	}
	fileAlign := uint64(binary.LittleEndian.Uint32(raw[optional+optionalFileAlignment:]))
	if fileAlign == 0 || fileAlign > 65536 || fileAlign&(fileAlign-1) != 0 {
		return nil, fmt.Errorf("invalid PE file alignment")
	}
	symoff := uint64(binary.LittleEndian.Uint32(raw[coff+coffSymbolTable:]))
	nsym := uint64(binary.LittleEndian.Uint32(raw[coff+coffSymbolCount:]))
	if symoff == 0 && nsym != 0 {
		return nil, fmt.Errorf("PE symbol count without a symbol table")
	}
	if !inFile(raw, symoff, nsym*pe.COFFSymbolSize) {
		return nil, fmt.Errorf("invalid PE symbol table")
	}
	var stringOff uint64
	var originalStringsSize int
	var stringsData []byte
	if symoff != 0 {
		stringOff = symoff + nsym*pe.COFFSymbolSize
		if !inFile(raw, stringOff, 4) {
			return nil, fmt.Errorf("truncated PE string table")
		}
		size := uint64(binary.LittleEndian.Uint32(raw[stringOff:]))
		if size < 4 || !inFile(raw, stringOff, size) {
			return nil, fmt.Errorf("invalid PE string table")
		}
		originalStringsSize = int(size)
		stringsData = bytes.Clone(raw[stringOff : stringOff+size])
	}
	// Bound the symbol/string allocations before invoking debug/pe.
	f, err := pe.NewFile(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	type compressedSection struct {
		index      int
		size       uint32
		name       string
		nameOffset uint32
	}
	var compressedSections []compressedSection
	var edits []edit
	var spans []edit
	names := make(map[string]bool)
	for _, s := range f.Sections {
		names[s.Name] = true
	}
	for i, s := range f.Sections {
		if s.Size == 0 {
			continue
		}
		if !inFile(raw, uint64(s.Offset), uint64(s.Size)) || int(s.Offset) < sections+nsect*peSectionHeaderSize {
			return nil, fmt.Errorf("invalid PE section %s", s.Name)
		}
		spans = append(spans, edit{start: int(s.Offset), end: int(uint64(s.Offset) + uint64(s.Size))})
		if !strings.HasPrefix(s.Name, ".debug_") || s.Name == ".debug_gdb_scripts" {
			continue
		}
		if s.NumberOfRelocations != 0 || s.NumberOfLineNumbers != 0 {
			return nil, fmt.Errorf("unrelocated PE DWARF section %s", s.Name)
		}
		size := s.VirtualSize
		if size == 0 {
			size = s.Size
		}
		if size > s.Size {
			return nil, fmt.Errorf("PE DWARF section %s has a virtual tail", s.Name)
		}
		data := compressed(raw[uint64(s.Offset) : uint64(s.Offset)+uint64(size)])
		if data == nil {
			continue
		}
		if align(uint64(len(data)), fileAlign) >= uint64(s.Size) {
			continue
		}
		if symoff == 0 {
			return nil, fmt.Errorf("PE DWARF section names require a COFF string table")
		}
		name := ".zdebug_" + strings.TrimPrefix(s.Name, ".debug_")
		if names[name] {
			return nil, fmt.Errorf("duplicate compressed PE section %s", name)
		}
		names[name] = true
		nameOffset := len(stringsData)
		if nameOffset > 9999999 {
			return nil, fmt.Errorf("PE section name exceeds decimal COFF string offsets")
		}
		stringsData = append(stringsData, name...)
		stringsData = append(stringsData, 0)
		compressedSections = append(compressedSections, compressedSection{i, uint32(len(data)), "/" + strconv.Itoa(nameOffset), uint32(nameOffset)})
		data = append(data, make([]byte, int(align(uint64(len(data)), fileAlign))-len(data))...)
		edits = append(edits, edit{int(s.Offset), int(uint64(s.Offset) + uint64(s.Size)), data})
	}
	// Validate even sections which were not selected for compression: an edit
	// must never overlap code, runtime metadata, or another section.
	if _, err := newLayout(raw, spans); err != nil {
		return nil, err
	}
	if len(edits) == 0 {
		return raw, nil
	}
	for _, s := range spans {
		if symoff < uint64(s.end) {
			return nil, fmt.Errorf("PE symbol/string table precedes section data")
		}
	}
	// Keep later overlay data's alignment when extending the string table.
	growth := align(uint64(len(stringsData)-originalStringsSize), fileAlign)
	stringsData = append(stringsData, make([]byte, originalStringsSize+int(growth)-len(stringsData))...)
	binary.LittleEndian.PutUint32(stringsData, uint32(len(stringsData)))
	edits = append(edits, edit{int(stringOff), int(stringOff) + originalStringsSize, stringsData})
	l, err := newLayout(raw, edits)
	if err != nil {
		return nil, err
	}
	out := l.apply(raw)
	if len(out) >= len(raw) {
		return raw, nil
	}
	if err := l.pointer32(out, coff+8); err != nil {
		return nil, err
	}
	var saved uint64
	for i, s := range f.Sections {
		pos := sections + i*peSectionHeaderSize
		for _, field := range []int{sectionRawData, sectionRelocations, sectionLineNumbers} {
			if err := l.pointer32(out, pos+field); err != nil {
				return nil, err
			}
		}
		for _, c := range compressedSections {
			if c.index != i {
				continue
			}
			clear(out[pos : pos+peSectionNameSize])
			copy(out[pos:pos+peSectionNameSize], c.name)
			// Keep the mapped extent and all RVAs unchanged. Shrinking
			// VirtualSize would leave address-space gaps before later sections,
			// which the Windows image loader rejects. The smaller raw payload
			// is zero-extended to the original virtual size by the loader.
			if s.VirtualSize == 0 {
				binary.LittleEndian.PutUint32(out[pos+sectionVirtualSize:], s.Size)
			}
			stored := align(uint64(c.size), fileAlign)
			binary.LittleEndian.PutUint32(out[pos+sectionRawSize:], uint32(stored))
			if s.Characteristics&pe.IMAGE_SCN_CNT_INITIALIZED_DATA != 0 {
				saved += uint64(s.Size) - stored
			}
		}
	}
	initialized := uint64(binary.LittleEndian.Uint32(out[optional+optionalInitializedData:]))
	if saved > initialized {
		return nil, fmt.Errorf("PE initialized-data size underflows")
	}
	binary.LittleEndian.PutUint32(out[optional+optionalInitializedData:], uint32(initialized-saved))
	if err := remapPEDebugDirectory(raw, out, f, l, optional+directoryOffset, ndirs); err != nil {
		return nil, err
	}
	// Resolved PE image symbols do not participate in linking again. Preserve
	// their values and names; update section-definition names and lengths.
	newSymoff := uint64(binary.LittleEndian.Uint32(out[coff+coffSymbolTable:]))
	for i := uint64(0); i < nsym; {
		pos := int(newSymoff + i*pe.COFFSymbolSize)
		aux := uint64(out[pos+symbolAuxCount])
		if i+aux >= nsym {
			return nil, fmt.Errorf("invalid PE auxiliary symbol count")
		}
		sect := int(int16(binary.LittleEndian.Uint16(out[pos+symbolSection:])))
		for _, c := range compressedSections {
			if sect == c.index+1 {
				if aux != 0 && out[pos+symbolStorageClass] == symbolClassStatic && binary.LittleEndian.Uint16(out[pos+symbolType:]) == 0 && binary.LittleEndian.Uint32(out[pos+symbolValue:]) == 0 {
					clear(out[pos : pos+symbolNameOffset])
					binary.LittleEndian.PutUint32(out[pos+symbolNameOffset:], c.nameOffset)
					binary.LittleEndian.PutUint32(out[pos+pe.COFFSymbolSize:], c.size)
				} else {
					return nil, fmt.Errorf("PE symbol refers into a rewritten debug section")
				}
			}
		}
		i += 1 + aux
	}
	binary.LittleEndian.PutUint32(out[optional+optionalChecksum:], peChecksum(out, optional+optionalChecksum))
	if _, err := pe.NewFile(bytes.NewReader(out)); err != nil {
		return nil, fmt.Errorf("invalid compressed PE: %w", err)
	}
	return out, nil
}

func remapPEDebugDirectory(raw, out []byte, f *pe.File, l *layout, directory, count int) error {
	if count <= pe.IMAGE_DIRECTORY_ENTRY_DEBUG {
		return nil
	}
	entry := directory + pe.IMAGE_DIRECTORY_ENTRY_DEBUG*peDirectoryEntrySize
	rva := uint64(binary.LittleEndian.Uint32(raw[entry:]))
	size := uint64(binary.LittleEndian.Uint32(raw[entry+peDirectorySizeField:]))
	if size == 0 {
		return nil
	}
	if size%peDebugDirectorySize != 0 {
		return fmt.Errorf("invalid PE debug directory size")
	}
	for _, s := range f.Sections {
		if rva < uint64(s.VirtualAddress) || rva-uint64(s.VirtualAddress) > uint64(s.Size) || size > uint64(s.Size)-(rva-uint64(s.VirtualAddress)) {
			continue
		}
		off := uint64(s.Offset) + rva - uint64(s.VirtualAddress)
		next, err := l.offset(off)
		if err != nil || !inFile(out, next, size) {
			return fmt.Errorf("invalid PE debug directory file range")
		}
		for pos := next; pos < next+size; pos += peDebugDirectorySize {
			if err := l.pointer32(out, int(pos)+debugDirectoryRawData); err != nil {
				return err
			}
		}
		return nil
	}
	return fmt.Errorf("PE debug directory is outside section data")
}

func peChecksum(raw []byte, checksum int) uint32 {
	var sum uint64
	for i := 0; i < len(raw); i += 2 {
		if i >= checksum && i < checksum+4 {
			continue
		}
		word := uint64(raw[i])
		if i+1 < len(raw) {
			word |= uint64(raw[i+1]) << 8
		}
		sum += word
		sum = (sum & math.MaxUint16) + (sum >> 16)
	}
	sum = (sum & math.MaxUint16) + (sum >> 16)
	return uint32(sum) + uint32(len(raw))
}
