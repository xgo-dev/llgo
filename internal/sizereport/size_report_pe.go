package sizereport

import (
	"debug/pe"
	"fmt"
)

func collectPESize(path string, pkgs []Package, level string) (*Report, error) {
	f, err := pe.Open(path)
	if err != nil {
		return nil, fmt.Errorf("reading PE size: %w", err)
	}
	defer f.Close()
	report := buildPESizeReport(path, f, pkgs, level)
	if report == nil || len(report.Modules) == 0 {
		return nil, fmt.Errorf("size report: no allocatable sections found in %s", path)
	}
	return report, nil
}

func peSectionKind(sec *pe.Section) sectionKind {
	flags := sec.Characteristics
	if flags&(pe.IMAGE_SCN_MEM_READ|pe.IMAGE_SCN_MEM_WRITE|pe.IMAGE_SCN_MEM_EXECUTE) == 0 {
		return sectionUnknown
	}
	if flags&pe.IMAGE_SCN_MEM_EXECUTE != 0 {
		return sectionText
	}
	if flags&pe.IMAGE_SCN_MEM_WRITE != 0 {
		return sectionData
	}
	return sectionROData
}

func peSectionSizes(sec *pe.Section) (stored, memory uint64) {
	memory = uint64(sec.VirtualSize)
	if memory == 0 {
		memory = uint64(sec.Size)
	}
	stored = min(uint64(sec.Size), memory)
	if sec.Characteristics&pe.IMAGE_SCN_CNT_UNINITIALIZED_DATA != 0 {
		stored = 0
	}
	return stored, memory
}

// COFF symbols use section-relative offsets but have no ELF-style st_size.
// Reuse the native address-range estimates, with section characteristics for
// classification. Raw alignment padding is excluded; virtual zero-fill is BSS.
func buildPESizeReport(path string, f *pe.File, pkgs []Package, level string) *Report {
	data := &readelfData{sections: make(map[int]*sectionInfo), symbols: make(map[int][]symbolInfo)}
	for i, sec := range f.Sections {
		kind := peSectionKind(sec)
		if kind == sectionUnknown {
			continue
		}
		stored, memory := peSectionSizes(sec)
		base := uint64(sec.VirtualAddress)
		data.sections[i] = &sectionInfo{Name: sec.Name, Address: base, Size: stored, Kind: kind}
		if memory > stored {
			data.sections[i+len(f.Sections)] = &sectionInfo{
				Name: sec.Name, Address: base + stored, Size: memory - stored, Kind: sectionBSS,
			}
		}
	}
	for _, sym := range f.Symbols {
		i := int(sym.SectionNumber) - 1
		if i < 0 || i >= len(f.Sections) || sym.Name == "" {
			continue
		}
		stored, memory := peSectionSizes(f.Sections[i])
		if uint64(sym.Value) >= memory {
			continue
		}
		if uint64(sym.Value) >= stored {
			i += len(f.Sections)
		}
		if data.sections[i] == nil {
			continue
		}
		base := uint64(f.Sections[int(sym.SectionNumber)-1].VirtualAddress)
		data.symbols[i] = append(data.symbols[i], symbolInfo{Name: sym.Name, Address: base + uint64(sym.Value)})
	}
	return buildSizeReport(path, data, pkgs, level)
}
