package sizereport

import (
	"debug/elf"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// collectELFSize reads the final ELF directly: section names do not determine
// whether bytes are loaded, and local labels do not determine function sizes.
func collectELFSize(path string, pkgs []Package, level string) (*Report, error) {
	f, err := elf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("reading ELF size: %w", err)
	}
	defer f.Close()

	symbols, err := f.Symbols()
	if err != nil && !errors.Is(err, elf.ErrNoSymbols) {
		return nil, fmt.Errorf("reading ELF symbols: %w", err)
	}
	dynamic, err := f.DynamicSymbols()
	if err != nil && !errors.Is(err, elf.ErrNoSymbols) {
		return nil, fmt.Errorf("reading ELF dynamic symbols: %w", err)
	}
	report := buildELFSizeReport(path, f, append(symbols, dynamic...), pkgs, level)
	if len(report.Modules) == 0 {
		return nil, fmt.Errorf("size report: no allocatable sections found in %s", path)
	}
	return report, nil
}

func elfSectionKind(sec *elf.Section) sectionKind {
	if sec.Flags&elf.SHF_ALLOC == 0 {
		return sectionUnknown
	}
	if sec.Type == elf.SHT_NOBITS {
		return sectionBSS
	}
	if sec.Flags&elf.SHF_EXECINSTR != 0 {
		return sectionText
	}
	if sec.Flags&elf.SHF_WRITE != 0 {
		return sectionData
	}
	return sectionROData
}

type elfSizeEvent struct {
	offset uint64
	owner  string
	delta  int
}

func buildELFSizeReport(path string, f *elf.File, symbols []elf.Symbol, pkgs []Package, level string) *Report {
	report := &Report{Binary: path, Modules: make(map[string]*Module)}
	resolver := newNameResolver(level, pkgs)
	events := make(map[elf.SectionIndex][]elfSizeEvent)
	for _, sym := range symbols {
		if sym.Size == 0 || sym.Name == "" || sym.Section == elf.SHN_UNDEF ||
			int(sym.Section) >= len(f.Sections) || sym.Section >= elf.SHN_LORESERVE {
			continue
		}
		switch elf.ST_TYPE(sym.Info) {
		case elf.STT_FILE, elf.STT_SECTION:
			continue
		}
		// Assembler labels and instruction/data mapping symbols are not owners.
		if strings.HasPrefix(sym.Name, ".L") || isELFMappingSymbol(sym.Name) {
			continue
		}
		sec := f.Sections[sym.Section]
		if elfSectionKind(sec) == sectionUnknown {
			continue
		}
		offset, ok := elfSymbolSectionOffset(f, sec, sym)
		if !ok || offset >= sec.Size {
			continue
		}
		// Clip malformed/outlying symbol ranges without overflowing an address.
		size := min(sym.Size, sec.Size-offset)
		owner := resolver.resolve(sym.Name)
		events[sym.Section] = append(events[sym.Section],
			elfSizeEvent{offset, owner, 1}, elfSizeEvent{offset + size, owner, -1})
	}
	for i, sec := range f.Sections {
		kind := elfSectionKind(sec)
		if kind == sectionUnknown || sec.Size == 0 {
			continue
		}
		addELFSectionSizes(report, sec.Name, kind, sec.Size, events[elf.SectionIndex(i)])
	}
	return report
}

func isELFMappingSymbol(name string) bool {
	for _, prefix := range []string{"$a", "$t", "$x", "$d"} {
		if name == prefix || strings.HasPrefix(name, prefix+".") {
			return true
		}
	}
	return false
}

func elfSymbolSectionOffset(f *elf.File, sec *elf.Section, sym elf.Symbol) (uint64, bool) {
	value := sym.Value
	typ := elf.ST_TYPE(sym.Info)
	if f.Machine == elf.EM_ARM && (typ == elf.STT_FUNC || typ == elf.STT_GNU_IFUNC) {
		// The low bit of an ARM function symbol indicates Thumb instructions;
		// it is not part of the function's byte address (including ET_REL).
		value &^= 1
	}
	if f.Type == elf.ET_REL {
		return value, true
	}
	base := sec.Addr
	if elf.ST_TYPE(sym.Info) == elf.STT_TLS {
		// Linked TLS symbol values are offsets in the TLS image, not VMAs.
		found := false
		for _, prog := range f.Progs {
			if prog.Type == elf.PT_TLS && sec.Addr >= prog.Vaddr {
				base = sec.Addr - prog.Vaddr
				found = true
				break
			}
		}
		if !found {
			return 0, false
		}
	}
	if value < base {
		return 0, false
	}
	return value - base, true
}

// Sweep section-relative ranges so aliases, overlapping symbols and duplicate
// static/dynamic entries never count physical bytes more than once. Unknown
// bytes may contain padding or unnamed data; the symbol table cannot tell which.
func addELFSectionSizes(report *Report, section string, kind sectionKind, size uint64, events []elfSizeEvent) {
	sort.Slice(events, func(i, j int) bool { return events[i].offset < events[j].offset })
	owners := make(map[string]int)
	addRange := func(n uint64) {
		owner := "(unknown " + section + ")"
		if len(owners) == 1 {
			for name := range owners {
				owner = name
			}
		} else if len(owners) > 1 {
			owner = "(shared " + section + ")"
		}
		report.add(owner, kind, n)
	}
	var cursor uint64
	for i := 0; i < len(events); {
		offset := events[i].offset
		addRange(offset - cursor)
		for i < len(events) && events[i].offset == offset {
			event := events[i]
			owners[event.owner] += event.delta
			if owners[event.owner] == 0 {
				delete(owners, event.owner)
			}
			i++
		}
		cursor = offset
	}
	addRange(size - cursor)
}
