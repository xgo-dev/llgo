// Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
// Licensed under the Apache License, Version 2.0.

package dwarfcompress

import (
	"bytes"
	"debug/macho"
	"encoding/binary"
	"fmt"
	"strings"
)

// Load-command values from <mach-o/loader.h> not exported by debug/macho.
const (
	lcUUID                   macho.LoadCmd = 0x1b       // LC_UUID
	lcSegmentSplitInfo       macho.LoadCmd = 0x1e       // LC_SEGMENT_SPLIT_INFO
	lcVersionMinMacOSX       macho.LoadCmd = 0x24       // LC_VERSION_MIN_MACOSX
	lcVersionMinIPhoneOS     macho.LoadCmd = 0x25       // LC_VERSION_MIN_IPHONEOS
	lcFunctionStarts         macho.LoadCmd = 0x26       // LC_FUNCTION_STARTS
	lcDataInCode             macho.LoadCmd = 0x29       // LC_DATA_IN_CODE
	lcSourceVersion          macho.LoadCmd = 0x2a       // LC_SOURCE_VERSION
	lcDylibCodeSignDRs       macho.LoadCmd = 0x2b       // LC_DYLIB_CODE_SIGN_DRS
	lcLinkerOptimizationHint macho.LoadCmd = 0x2e       // LC_LINKER_OPTIMIZATION_HINT
	lcVersionMinTVOS         macho.LoadCmd = 0x2f       // LC_VERSION_MIN_TVOS
	lcVersionMinWatchOS      macho.LoadCmd = 0x30       // LC_VERSION_MIN_WATCHOS
	lcNote                   macho.LoadCmd = 0x31       // LC_NOTE
	lcBuildVersion           macho.LoadCmd = 0x32       // LC_BUILD_VERSION
	lcDyldExportsTrie        macho.LoadCmd = 0x80000033 // LC_DYLD_EXPORTS_TRIE | LC_REQ_DYLD
	lcDyldChainedFixups      macho.LoadCmd = 0x80000034 // LC_DYLD_CHAINED_FIXUPS | LC_REQ_DYLD
)

// Sizes and byte offsets within the on-disk structures in <mach-o/loader.h>.
const (
	machoHeader64Size      = 32
	machoTypeDSYM          = 0xa // MH_DSYM
	machoFileType          = 12
	machoCommandCount      = 16
	machoCommandsSize      = 20
	machoNameSize          = 16
	machoCommandHeaderSize = 8
	machoCommandSize       = 4
	machoSymbol64Size      = 16 // struct nlist_64 in <mach-o/nlist.h>
	machoRelocationSize    = 8  // struct relocation_info

	// struct segment_command_64
	segment64Size         = 72
	segment64Name         = 8
	segment64VMAddress    = 24
	segment64VMSize       = 32
	segment64FileOffset   = 40
	segment64FileSize     = 48
	segment64SectionCount = 64

	// struct section_64
	section64Size            = 80
	section64Address         = 32
	section64DataSize        = 40
	section64FileOffset      = 48
	section64Alignment       = 52
	section64Relocations     = 56
	section64RelocationCount = 60

	// struct symtab_command
	symtabCommandSize = 24
	symtabSymbols     = 8
	symtabSymbolCount = 12
	symtabStrings     = 16
	symtabStringsSize = 20

	// struct dysymtab_command
	dysymtabCommandSize      = 80
	dysymTOC                 = 32
	dysymModules             = 40
	dysymExternalRefs        = 48
	dysymIndirectSymbols     = 56
	dysymIndirectSymbolCount = 60
	dysymExternalRelocs      = 64
	dysymLocalRelocs         = 72

	// struct linkedit_data_command and struct note_command
	linkeditCommandSize = 16
	linkeditDataOffset  = 8
	noteCommandSize     = 40
	noteDataOffset      = 24
)

func compressMachO(raw []byte) ([]byte, error) {
	if len(raw) < machoHeader64Size || binary.LittleEndian.Uint32(raw[machoFileType:]) != machoTypeDSYM {
		return nil, fmt.Errorf("Mach-O DWARF compression requires a dSYM, not an executable or relocatable object")
	}
	ncmds, cmdsize := binary.LittleEndian.Uint32(raw[machoCommandCount:]), binary.LittleEndian.Uint32(raw[machoCommandsSize:])
	if !inFile(raw, machoHeader64Size, uint64(cmdsize)) || uint64(ncmds)*machoCommandHeaderSize > uint64(cmdsize) {
		return nil, fmt.Errorf("invalid Mach-O load commands")
	}
	if err := validateMachORanges(raw, ncmds, cmdsize); err != nil {
		return nil, err
	}
	f, err := macho.NewFile(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	// Removing whole 16 KiB pages preserves both x86-64 and arm64 segment
	// alignment. Smaller gains still compress the section, but retain padding.
	const page = uint64(16384)
	var edits []edit
	sizes := make(map[uint32]uint64)
	names := make(map[string]bool)
	for _, s := range f.Sections {
		names[s.Seg+"/"+s.Name] = true
	}
	for _, s := range f.Sections {
		if s.Seg != "__DWARF" || !strings.HasPrefix(s.Name, "__debug_") || s.Size == 0 {
			names[s.Seg+"/"+s.Name] = true
			continue
		}
		if s.Nreloc != 0 || !inFile(raw, uint64(s.Offset), s.Size) || uint64(s.Offset) < machoHeader64Size+uint64(cmdsize) {
			return nil, fmt.Errorf("invalid or unrelocated Mach-O DWARF section %s", s.Name)
		}
		data := compressed(raw[uint64(s.Offset) : uint64(s.Offset)+s.Size])
		if data == nil {
			names[s.Seg+"/"+s.Name] = true
			continue
		}
		name := "__zdebug_" + strings.TrimPrefix(s.Name, "__debug_")
		if len(name) > machoNameSize {
			name = name[:machoNameSize]
		}
		if names[s.Seg+"/"+name] {
			return nil, fmt.Errorf("duplicate compressed Mach-O section %s", name)
		}
		names[s.Seg+"/"+name] = true
		sizes[s.Offset] = uint64(len(data))
		stored := s.Size - (s.Size-uint64(len(data)))/page*page
		data = append(data, make([]byte, int(stored)-len(data))...)
		edits = append(edits, edit{int(s.Offset), int(uint64(s.Offset) + s.Size), data})
	}
	if len(edits) == 0 {
		return raw, nil
	}
	l, err := newLayout(raw, edits)
	if err != nil {
		return nil, err
	}
	// dSYM symbols describe the original executable's addresses. Reject the
	// unusual case of symbols referring into the debug-only segment itself.
	if f.Symtab != nil {
		for _, s := range f.Symtab.Syms {
			if s.Sect != 0 && int(s.Sect) <= len(f.Sections) && f.Sections[s.Sect-1].Seg == "__DWARF" {
				return nil, fmt.Errorf("Mach-O symbol %s refers to a rewritten debug section", s.Name)
			}
		}
	}
	out := l.apply(raw)
	if len(out) >= len(raw) {
		return raw, nil
	}
	for pos, i := machoHeader64Size, uint32(0); i < ncmds; i++ {
		// validateMachORanges already checked the unchanged command sizes,
		// segment extents, section counts, and symbol-table command layouts.
		kind, size := macho.LoadCmd(binary.LittleEndian.Uint32(out[pos:])), int(binary.LittleEndian.Uint32(out[pos+machoCommandSize:]))
		switch kind {
		case macho.LoadCmdSegment64:
			oldOff, oldSize := binary.LittleEndian.Uint64(out[pos+segment64FileOffset:]), binary.LittleEndian.Uint64(out[pos+segment64FileSize:])
			newOff, err := l.offset(oldOff)
			if err != nil {
				return nil, err
			}
			end, err := l.offset(oldOff + oldSize)
			if err != nil || end < newOff || end-newOff > oldSize {
				return nil, fmt.Errorf("invalid Mach-O segment file range")
			}
			binary.LittleEndian.PutUint64(out[pos+segment64FileOffset:], newOff)
			binary.LittleEndian.PutUint64(out[pos+segment64FileSize:], end-newOff)
			// File compaction does not change virtual addresses or vmsize.
			// Virtual gaps and section order can differ from the file layout.
			nsect := binary.LittleEndian.Uint32(out[pos+segment64SectionCount:])
			for j := uint32(0); j < nsect; j++ {
				p := pos + segment64Size + int(j)*section64Size
				off := binary.LittleEndian.Uint32(out[p+section64FileOffset:])
				if off != 0 {
					next, err := l.offset(uint64(off))
					if err != nil {
						return nil, err
					}
					binary.LittleEndian.PutUint32(out[p+section64FileOffset:], uint32(next))
				}
				if compressedSize, ok := sizes[off]; ok {
					name := "__zdebug_" + strings.TrimPrefix(string(bytes.TrimRight(out[p:p+machoNameSize], "\x00")), "__debug_")
					clear(out[p : p+machoNameSize])
					copy(out[p:p+machoNameSize], name)
					binary.LittleEndian.PutUint64(out[p+section64DataSize:], compressedSize)
				}
				if err := l.pointer32(out, p+section64Relocations); err != nil {
					return nil, err
				}
			}
		case macho.LoadCmdSymtab:
			for _, field := range []int{symtabSymbols, symtabStrings} {
				if err := l.pointer32(out, pos+field); err != nil {
					return nil, err
				}
			}
		case macho.LoadCmdDysymtab:
			for _, field := range []int{dysymTOC, dysymModules, dysymExternalRefs, dysymIndirectSymbols, dysymExternalRelocs, dysymLocalRelocs} {
				if err := l.pointer32(out, pos+field); err != nil {
					return nil, err
				}
			}
		case lcSegmentSplitInfo, lcFunctionStarts, lcDataInCode, lcDylibCodeSignDRs, lcLinkerOptimizationHint, lcDyldExportsTrie, lcDyldChainedFixups:
			if size != linkeditCommandSize {
				return nil, fmt.Errorf("invalid Mach-O linkedit data command")
			}
			if err := l.pointer32(out, pos+linkeditDataOffset); err != nil {
				return nil, err
			}
		case lcNote:
			if size != noteCommandSize {
				return nil, fmt.Errorf("invalid Mach-O note command")
			}
			if err := l.pointer64(out, pos+noteDataOffset); err != nil {
				return nil, err
			}
		case lcUUID, lcVersionMinMacOSX, lcVersionMinIPhoneOS, lcSourceVersion, lcVersionMinTVOS, lcVersionMinWatchOS, lcBuildVersion:
			// These commands have no file offsets; preserve them verbatim.
		default:
			return nil, fmt.Errorf("unsupported Mach-O dSYM load command %#x", uint32(kind))
		}
		pos += size
	}
	if _, err := macho.NewFile(bytes.NewReader(out)); err != nil {
		return nil, fmt.Errorf("invalid compressed Mach-O: %w", err)
	}
	return out, nil
}

// Check sizes before debug/macho allocates tables from load-command counts.
// File-backed sections must be disjoint, including sections we do not rewrite.
func validateMachORanges(raw []byte, ncmds, cmdsize uint32) error {
	var spans []edit
	pos, end := machoHeader64Size, machoHeader64Size+int(cmdsize)
	for i := uint32(0); i < ncmds; i++ {
		if pos > end-machoCommandHeaderSize {
			return fmt.Errorf("truncated Mach-O load command")
		}
		kind, size := macho.LoadCmd(binary.LittleEndian.Uint32(raw[pos:])), uint64(binary.LittleEndian.Uint32(raw[pos+machoCommandSize:]))
		if size < machoCommandHeaderSize || size > uint64(end-pos) {
			return fmt.Errorf("invalid Mach-O load command size")
		}
		switch kind {
		case macho.LoadCmdSegment64:
			if size < segment64Size || uint64(binary.LittleEndian.Uint32(raw[pos+segment64SectionCount:]))*section64Size > size-segment64Size {
				return fmt.Errorf("truncated Mach-O segment sections")
			}
			segOff, segSize := binary.LittleEndian.Uint64(raw[pos+segment64FileOffset:]), binary.LittleEndian.Uint64(raw[pos+segment64FileSize:])
			if !inFile(raw, segOff, segSize) {
				return fmt.Errorf("invalid Mach-O segment range")
			}
			isDWARF := string(bytes.TrimRight(raw[pos+segment64Name:pos+segment64Name+machoNameSize], "\x00")) == "__DWARF"
			vmAddr, vmSize := binary.LittleEndian.Uint64(raw[pos+segment64VMAddress:]), binary.LittleEndian.Uint64(raw[pos+segment64VMSize:])
			if isDWARF && vmAddr+vmSize < vmAddr {
				return fmt.Errorf("invalid Mach-O debug segment virtual range")
			}
			for j := uint32(0); j < binary.LittleEndian.Uint32(raw[pos+segment64SectionCount:]); j++ {
				p := pos + segment64Size + int(j)*section64Size
				off, length := uint64(binary.LittleEndian.Uint32(raw[p+section64FileOffset:])), binary.LittleEndian.Uint64(raw[p+section64DataSize:])
				addr := binary.LittleEndian.Uint64(raw[p+section64Address:])
				if isDWARF && length != 0 && (addr < vmAddr || length > vmSize || addr-vmAddr > vmSize-length) {
					return fmt.Errorf("invalid Mach-O debug section virtual range")
				}
				reloff, nreloc := uint64(binary.LittleEndian.Uint32(raw[p+section64Relocations:])), uint64(binary.LittleEndian.Uint32(raw[p+section64RelocationCount:]))
				if !inFile(raw, reloff, nreloc*machoRelocationSize) {
					return fmt.Errorf("invalid Mach-O relocation range")
				}
				if off == 0 || length == 0 {
					continue // dSYM code sections describe virtual addresses only.
				}
				if binary.LittleEndian.Uint32(raw[p+section64Alignment:]) > 14 {
					return fmt.Errorf("unsupported Mach-O section alignment above 16 KiB")
				}
				if off < uint64(end) || !inFile(raw, off, length) || off < segOff || length > segSize || off-segOff > segSize-length {
					return fmt.Errorf("invalid Mach-O section range")
				}
				spans = append(spans, edit{start: int(off), end: int(off + length)})
			}
		case macho.LoadCmdSymtab:
			if size != symtabCommandSize || !inFile(raw, uint64(binary.LittleEndian.Uint32(raw[pos+symtabSymbols:])), uint64(binary.LittleEndian.Uint32(raw[pos+symtabSymbolCount:]))*machoSymbol64Size) ||
				!inFile(raw, uint64(binary.LittleEndian.Uint32(raw[pos+symtabStrings:])), uint64(binary.LittleEndian.Uint32(raw[pos+symtabStringsSize:]))) {
				return fmt.Errorf("invalid Mach-O symbol table range")
			}
			for _, field := range []struct{ off, count, unit int }{{symtabSymbols, symtabSymbolCount, machoSymbol64Size}, {symtabStrings, symtabStringsSize, 1}} {
				off := uint64(binary.LittleEndian.Uint32(raw[pos+field.off:]))
				length := uint64(binary.LittleEndian.Uint32(raw[pos+field.count:])) * uint64(field.unit)
				if length != 0 {
					if off < uint64(end) {
						return fmt.Errorf("Mach-O symbol table overlaps load commands")
					}
					spans = append(spans, edit{start: int(off), end: int(off + length)})
				}
			}
		case macho.LoadCmdDysymtab:
			if size != dysymtabCommandSize || !inFile(raw, uint64(binary.LittleEndian.Uint32(raw[pos+dysymIndirectSymbols:])), uint64(binary.LittleEndian.Uint32(raw[pos+dysymIndirectSymbolCount:]))*4) {
				return fmt.Errorf("invalid Mach-O indirect symbol table range")
			}
		case macho.LoadCmdSegment: // 32-bit segment commands do not belong in this layout.
			return fmt.Errorf("unsupported 32-bit Mach-O segment")
		}
		pos += int(size)
	}
	if pos != end {
		return fmt.Errorf("inconsistent Mach-O load command count")
	}
	_, err := newLayout(raw, spans)
	return err
}
