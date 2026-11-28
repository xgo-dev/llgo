package sizereport

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestELFSizeSectionFlags(t *testing.T) {
	cases := []struct {
		name  string
		type_ elf.SectionType
		flags elf.SectionFlag
		want  sectionKind
	}{
		{".eh_frame", elf.SHT_PROGBITS, elf.SHF_ALLOC, sectionROData},
		{".ARM.exidx", elf.SectionType(0x70000001), elf.SHF_ALLOC, sectionROData},
		{".ctors", elf.SHT_INIT_ARRAY, elf.SHF_ALLOC | elf.SHF_WRITE, sectionData},
		{".vectors", elf.SHT_PROGBITS, elf.SHF_ALLOC | elf.SHF_EXECINSTR, sectionText},
		{".custom.ram", elf.SHT_NOBITS, elf.SHF_ALLOC | elf.SHF_WRITE, sectionBSS},
		{".tdata", elf.SHT_PROGBITS, elf.SHF_ALLOC | elf.SHF_WRITE | elf.SHF_TLS, sectionData},
		{".tbss", elf.SHT_NOBITS, elf.SHF_ALLOC | elf.SHF_WRITE | elf.SHF_TLS, sectionBSS},
		{".text.debug", elf.SHT_PROGBITS, 0, sectionUnknown},
		{".rodata.not_loaded", elf.SHT_PROGBITS, 0, sectionUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sec := &elf.Section{SectionHeader: elf.SectionHeader{Name: tc.name, Type: tc.type_, Flags: tc.flags}}
			if got := elfSectionKind(sec); got != tc.want {
				t.Fatalf("section kind = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestELFSizeSymbolRanges(t *testing.T) {
	f := &elf.File{
		FileHeader: elf.FileHeader{Type: elf.ET_EXEC},
		Sections: []*elf.Section{
			{},
			{SectionHeader: elf.SectionHeader{Name: ".text", Type: elf.SHT_PROGBITS, Flags: elf.SHF_ALLOC | elf.SHF_EXECINSTR, Addr: 0x1000, Size: 40}},
			// Overlapping VMAs are possible in embedded address spaces. Symbol
			// ownership must stay within the symbol's own section.
			{SectionHeader: elf.SectionHeader{Name: ".constant", Type: elf.SHT_PROGBITS, Flags: elf.SHF_ALLOC, Addr: 0x1000, Size: 12}},
			{SectionHeader: elf.SectionHeader{Name: ".debug_info", Type: elf.SHT_PROGBITS, Addr: 0x1000, Size: 8}},
		},
	}
	symbols := []elf.Symbol{
		{Name: "pkg.function", Section: 1, Info: byte(elf.STT_FUNC), Value: 0x1000, Size: 16},
		{Name: "pkg.alias", Section: 1, Info: byte(elf.STT_FUNC), Value: 0x1000, Size: 16},
		// A duplicate from .dynsym must not change attribution.
		{Name: "pkg.function", Section: 1, Info: byte(elf.STT_FUNC), Value: 0x1000, Size: 16},
		{Name: ".L0", Section: 1, Value: 0x1004},
		{Name: "$x.1", Section: 1, Value: 0x1000, Size: 16},
		{Name: "other.overlap", Section: 1, Info: byte(elf.STT_FUNC), Value: 0x1008, Size: 4},
		{Name: "next.function", Section: 1, Info: byte(elf.STT_FUNC), Value: 0x1018, Size: 8},
		{Name: "data.constant", Section: 2, Info: byte(elf.STT_OBJECT), Value: 0x1000, Size: 12},
		{Name: "outside.function", Section: 1, Info: byte(elf.STT_FUNC), Value: 0x1028, Size: 8},
		{Name: "absolute", Section: elf.SHN_ABS, Value: 0x1000, Size: 40},
		{Name: "source.c", Section: 1, Info: byte(elf.STT_FILE), Value: 0x1000, Size: 40},
		{Name: "section", Section: 1, Info: byte(elf.STT_SECTION), Value: 0x1000, Size: 40},
		{Name: "debug.object", Section: 3, Info: byte(elf.STT_OBJECT), Value: 0x1000, Size: 8},
	}
	report := buildELFSizeReport("test.elf", f, symbols, nil, "module")
	for name, want := range map[string]uint64{"pkg": 12, "(shared .text)": 4, "next": 8, "(unknown .text)": 16} {
		mod := report.Modules[name]
		if mod == nil || mod.Code != want {
			t.Fatalf("%s code = %+v, want %d", name, mod, want)
		}
	}
	if mod := report.Modules["data"]; mod == nil || mod.ROData != 12 || mod.Code != 0 {
		t.Fatalf("section-relative data attribution = %+v", mod)
	}
	if report.Total.Code != 40 || report.Total.ROData != 12 {
		t.Fatalf("section totals do not close: %+v", report.Total)
	}
	if len(report.Modules) != 5 {
		t.Fatalf("unexpected label/alias buckets: %+v", report.Modules)
	}
}

func TestELFSizeSymbolSectionOffsets(t *testing.T) {
	sec := &elf.Section{SectionHeader: elf.SectionHeader{Addr: 0x8000, Size: 32, Flags: elf.SHF_ALLOC | elf.SHF_TLS}}
	f := &elf.File{
		FileHeader: elf.FileHeader{Type: elf.ET_EXEC},
		Progs:      []*elf.Prog{{ProgHeader: elf.ProgHeader{Type: elf.PT_TLS, Vaddr: 0x7ff0, Memsz: 48}}},
	}
	sym := elf.Symbol{Info: byte(elf.STT_TLS), Value: 24}
	if got, ok := elfSymbolSectionOffset(f, sec, sym); !ok || got != 8 {
		t.Fatalf("TLS offset = %d, %t; want 8, true", got, ok)
	}
	f.Type = elf.ET_REL
	if got, ok := elfSymbolSectionOffset(f, sec, sym); !ok || got != 24 {
		t.Fatalf("relocatable offset = %d, %t; want 24, true", got, ok)
	}
	f.Machine = elf.EM_ARM
	sym.Info, sym.Value = byte(elf.STT_FUNC), 3
	if got, ok := elfSymbolSectionOffset(f, sec, sym); !ok || got != 2 {
		t.Fatalf("relocatable Thumb offset = %d, %t; want 2, true", got, ok)
	}
	f.Type, sym.Value = elf.ET_EXEC, 0x8003
	if got, ok := elfSymbolSectionOffset(f, sec, sym); !ok || got != 2 {
		t.Fatalf("linked Thumb offset = %d, %t; want 2, true", got, ok)
	}
	sym.Info = byte(elf.STT_GNU_IFUNC)
	if got, ok := elfSymbolSectionOffset(f, sec, sym); !ok || got != 2 {
		t.Fatalf("linked Thumb IFUNC offset = %d, %t; want 2, true", got, ok)
	}
	sym.Info = byte(elf.STT_OBJECT)
	if got, ok := elfSymbolSectionOffset(f, sec, sym); !ok || got != 3 {
		t.Fatalf("odd data address offset = %d, %t; want 3, true", got, ok)
	}
	sym.Value = 0x7fff
	if _, ok := elfSymbolSectionOffset(f, sec, sym); ok {
		t.Fatal("address before section base accepted")
	}
	sym.Info, sym.Value = byte(elf.STT_TLS), 24
	f.Progs = nil
	if _, ok := elfSymbolSectionOffset(f, sec, sym); ok {
		t.Fatal("TLS symbol without a TLS image accepted")
	}
}

func TestCollectELFSizeRejectsMalformedArtifacts(t *testing.T) {
	for _, tc := range []struct {
		name    string
		mutate  func([]byte)
		wantErr string
	}{
		{"bad magic", func(raw []byte) { raw[0] = 0 }, "reading ELF size"},
		{"bad static symbols", func(raw []byte) {
			mutateELFSizeSections(raw, func(section []byte) {
				if elf.SectionType(binary.LittleEndian.Uint32(section[4:8])) == elf.SHT_SYMTAB {
					binary.LittleEndian.PutUint64(section[32:40], 1) // incomplete symbol entry
				}
			})
		}, "reading ELF symbols"},
		{"bad dynamic symbols", func(raw []byte) {
			mutateELFSizeSections(raw, func(section []byte) {
				if elf.SectionType(binary.LittleEndian.Uint32(section[4:8])) == elf.SHT_SYMTAB {
					binary.LittleEndian.PutUint32(section[4:8], uint32(elf.SHT_DYNSYM))
					binary.LittleEndian.PutUint64(section[32:40], 1)
				}
			})
		}, "reading ELF dynamic symbols"},
		{"no allocated bytes", func(raw []byte) {
			mutateELFSizeSections(raw, func(section []byte) {
				flags := binary.LittleEndian.Uint64(section[8:16])
				binary.LittleEndian.PutUint64(section[8:16], flags&^uint64(elf.SHF_ALLOC))
			})
		}, "no allocatable sections"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := elfSizeFixture(t, true)
			tc.mutate(raw)
			path := filepath.Join(t.TempDir(), "broken.elf")
			if err := os.WriteFile(path, raw, 0o600); err != nil {
				t.Fatal(err)
			}
			if report, err := collectELFSize(path, nil, "full"); report != nil || err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("broken ELF report = %+v, error = %v; want %q", report, err, tc.wantErr)
			}
		})
	}
}

func mutateELFSizeSections(raw []byte, mutate func([]byte)) {
	offset := binary.LittleEndian.Uint64(raw[40:48])
	stride := binary.LittleEndian.Uint16(raw[58:60])
	count := binary.LittleEndian.Uint16(raw[60:62])
	for i := uint16(0); i < count; i++ {
		start := offset + uint64(i)*uint64(stride)
		mutate(raw[start : start+uint64(stride)])
	}
}

func TestCollectELFSize(t *testing.T) {
	for _, withSymbols := range []bool{true, false} {
		name := "stripped"
		if withSymbols {
			name = "symbols"
		}
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "fixture.elf")
			if err := os.WriteFile(path, elfSizeFixture(t, withSymbols), 0600); err != nil {
				t.Fatal(err)
			}
			report, err := collectELFSize(path, nil, "full")
			if err != nil {
				t.Fatal(err)
			}
			if got := report.Total; got.Code != 32 || got.ROData != 8 || got.Data != 8 || got.BSS != 16 || got.Flash() != 48 || got.RAM() != 24 {
				t.Fatalf("unexpected totals: %+v", got)
			}
			if withSymbols {
				mod := report.Modules["pkg.(*T).Method"]
				if mod == nil || mod.Code != 12 {
					t.Fatalf("sized function was truncated by label/name parsing: %+v", report.Modules)
				}
				if mod := report.Modules["(unknown .text)"]; mod == nil || mod.Code != 16 {
					t.Fatalf("symbol gaps attributed to preceding function: %+v", mod)
				}
			} else if mod := report.Modules["(unknown .text)"]; mod == nil || mod.Code != 32 {
				t.Fatalf("stripped text must remain accounted for: %+v", mod)
			}
		})
	}
}

// A small linked ELF fixture keeps parser coverage independent of a host C
// compiler, linker and operating system. Its nonallocated .text.debug section
// deliberately has a misleading name, and .bss deliberately has no file bytes.
func elfSizeFixture(t *testing.T, withSymbols bool) []byte {
	t.Helper()
	var out bytes.Buffer
	write := func(value any) {
		if err := binary.Write(&out, binary.LittleEndian, value); err != nil {
			t.Fatal(err)
		}
	}
	var header elf.Header64
	copy(header.Ident[:], []byte(elf.ELFMAG))
	header.Ident[elf.EI_CLASS] = byte(elf.ELFCLASS64)
	header.Ident[elf.EI_DATA] = byte(elf.ELFDATA2LSB)
	header.Ident[elf.EI_VERSION] = byte(elf.EV_CURRENT)
	header.Type, header.Machine, header.Version = uint16(elf.ET_EXEC), uint16(elf.EM_X86_64), uint32(elf.EV_CURRENT)
	header.Ehsize, header.Shentsize = uint16(binary.Size(header)), uint16(binary.Size(elf.Section64{}))
	write(header)
	sections := []elf.Section64{{}}
	sectionNames := []byte{0}
	addSection := func(name string, typ elf.SectionType, flags elf.SectionFlag, addr uint64, data []byte) {
		sections = append(sections, elf.Section64{
			Name: uint32(len(sectionNames)), Type: uint32(typ), Flags: uint64(flags), Addr: addr,
			Off: uint64(out.Len()), Size: uint64(len(data)), Addralign: 1,
		})
		sectionNames = append(sectionNames, append([]byte(name), 0)...)
		if typ != elf.SHT_NOBITS {
			out.Write(data)
		}
	}
	addSection(".text", elf.SHT_PROGBITS, elf.SHF_ALLOC|elf.SHF_EXECINSTR, 0x1000, make([]byte, 32))
	addSection(".eh_frame", elf.SHT_PROGBITS, elf.SHF_ALLOC, 0x2000, make([]byte, 8))
	addSection(".init_array", elf.SHT_INIT_ARRAY, elf.SHF_ALLOC|elf.SHF_WRITE, 0x3000, make([]byte, 8))
	addSection(".bss", elf.SHT_NOBITS, elf.SHF_ALLOC|elf.SHF_WRITE, 0x4000, make([]byte, 16))
	addSection(".text.debug", elf.SHT_PROGBITS, 0, 0, make([]byte, 64))
	if withSymbols {
		strings := []byte{0}
		syms := []elf.Sym64{{}}
		for _, sym := range []elf.Symbol{
			{Name: ".L0", Info: byte(elf.STT_NOTYPE), Value: 0x1004},
			{Name: "pkg.(*T).Method", Info: byte(elf.STT_FUNC), Value: 0x1000, Size: 12},
			{Name: "next.helper", Info: byte(elf.STT_FUNC), Value: 0x1018, Size: 4},
		} {
			syms = append(syms, elf.Sym64{Name: uint32(len(strings)), Info: sym.Info, Shndx: 1, Value: sym.Value, Size: sym.Size})
			strings = append(strings, append([]byte(sym.Name), 0)...)
		}
		strIndex := len(sections)
		addSection(".strtab", elf.SHT_STRTAB, 0, 0, strings)
		start := out.Len()
		write(syms)
		symBytes := append([]byte(nil), out.Bytes()[start:]...)
		out.Truncate(start)
		addSection(".symtab", elf.SHT_SYMTAB, 0, 0, symBytes)
		sections[len(sections)-1].Link = uint32(strIndex)
		sections[len(sections)-1].Entsize = uint64(binary.Size(elf.Sym64{}))
	}
	header.Shstrndx = uint16(len(sections))
	// addSection appends its own name after receiving the data, so include the
	// final section name before constructing its string-table payload.
	nameOffset := len(sectionNames)
	sectionNames = append(sectionNames, []byte(".shstrtab\x00")...)
	sections = append(sections, elf.Section64{Name: uint32(nameOffset), Type: uint32(elf.SHT_STRTAB), Off: uint64(out.Len()), Size: uint64(len(sectionNames)), Addralign: 1})
	out.Write(sectionNames)
	header.Shoff, header.Shnum = uint64(out.Len()), uint16(len(sections))
	write(sections)
	result := append([]byte(nil), out.Bytes()...)
	out.Reset()
	write(header)
	copy(result, out.Bytes())
	return result
}
