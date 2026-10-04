package sizereport

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPESizeFinalArtifact(t *testing.T) {
	for _, machine := range []uint16{pe.IMAGE_FILE_MACHINE_I386, pe.IMAGE_FILE_MACHINE_AMD64} {
		for _, symbols := range []bool{false, true} {
			raw := peSizeFixture(t, machine, symbols)
			path := filepath.Join(t.TempDir(), "app.exe")
			if err := os.WriteFile(path, raw, 0o600); err != nil {
				t.Fatal(err)
			}
			report, err := Collect(path, nil, "full")
			if err != nil {
				t.Fatal(err)
			}
			// The code's 512-byte raw section holds 32 bytes of payload; data
			// has 8 stored bytes and 16 virtual bytes. Debug bytes are not loaded.
			if report.Format != "pe" || report.FileSize != uint64(len(raw)) || report.Total.Code != 32 || report.Total.ROData != 8 || report.Total.Data != 8 || report.Total.BSS != 24 {
				t.Fatalf("machine=%#x symbols=%t: %+v", machine, symbols, report)
			}
			owner := "(unknown .text)"
			if symbols {
				owner = "main.(*T).Method"
			}
			if got := report.Modules[owner]; got == nil || got.Code != 32 {
				t.Fatalf("code attribution = %+v", report.Modules)
			}
			if symbols && (report.Modules["main.zero"] == nil || report.Modules["main.zero"].BSS != 8) {
				t.Fatalf("zero-fill attribution = %+v", report.Modules)
			}
		}
	}
}

func TestPESizeRejectsMalformedArtifact(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.exe")
	for _, raw := range [][]byte{
		{'M', 'Z', 0, 0},
		peSizeFixture(t, pe.IMAGE_FILE_MACHINE_AMD64, false)[:128],
	} {
		if err := os.WriteFile(path, raw, 0o600); err != nil {
			t.Fatal(err)
		}
		if report, err := Collect(path, nil, "full"); report != nil || err == nil {
			t.Fatalf("malformed PE report = %+v, error = %v", report, err)
		}
	}
	if _, err := collectPESize(filepath.Join(t.TempDir(), "missing"), nil, "full"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing PE error = %v", err)
	}
	if _, err := collectPESize(writeEmptyPE(t), nil, "full"); err == nil {
		t.Fatal("PE without loaded sections accepted")
	}
}

func TestPESizeSectionRelativeSymbols(t *testing.T) {
	f := &pe.File{Sections: []*pe.Section{
		{SectionHeader: pe.SectionHeader{Name: ".text", VirtualAddress: 0x1000, Size: 32, Characteristics: pe.IMAGE_SCN_MEM_EXECUTE}},
		{SectionHeader: pe.SectionHeader{Name: ".debug", VirtualAddress: 0x2000, Size: 64}},
	}, Symbols: []*pe.Symbol{
		{Name: "main.first", SectionNumber: 1},
		{Name: "main.alias", SectionNumber: 1},
		{Name: "main.second", SectionNumber: 1, Value: 16},
		{Name: "undefined", SectionNumber: 0},
		{Name: "absolute", SectionNumber: -1},
		{Name: "bad.section", SectionNumber: 3},
		{Name: "out.of.range", SectionNumber: 1, Value: 32},
		{Name: "debug.only", SectionNumber: 2},
		{SectionNumber: 1},
	}}
	report := buildPESizeReport("app.exe", f, nil, "module")
	if report.Total.Code != 32 || len(report.Modules) != 1 || report.Modules["main"].Code != 32 {
		t.Fatalf("PE aliases/invalid symbols = %+v", report)
	}
}

func writeEmptyPE(t *testing.T) string {
	t.Helper()
	raw := peSizeFixture(t, pe.IMAGE_FILE_MACHINE_AMD64, false)
	// NumberOfSections follows the machine field in the COFF header.
	binary.LittleEndian.PutUint16(raw[134:], 0)
	path := filepath.Join(t.TempDir(), "empty.exe")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func peSizeFixture(t *testing.T, machine uint16, symbols bool) []byte {
	t.Helper()
	var out bytes.Buffer
	write := func(v any) {
		if err := binary.Write(&out, binary.LittleEndian, v); err != nil {
			t.Fatal(err)
		}
	}
	dos := make([]byte, 128)
	copy(dos, "MZ")
	binary.LittleEndian.PutUint32(dos[0x3c:], 128)
	out.Write(dos)
	out.WriteString("PE\x00\x00")
	header := pe.FileHeader{Machine: machine, NumberOfSections: 5}
	var optional any = pe.OptionalHeader64{Magic: 0x20b, NumberOfRvaAndSizes: 16}
	if machine == pe.IMAGE_FILE_MACHINE_I386 {
		optional = pe.OptionalHeader32{Magic: 0x10b, NumberOfRvaAndSizes: 16}
	}
	header.SizeOfOptionalHeader = uint16(binary.Size(optional))
	write(header)
	write(optional)
	dataStart := out.Len() + int(header.NumberOfSections)*binary.Size(pe.SectionHeader32{})
	for _, sec := range []struct {
		name        string
		memory, raw uint32
		flags       uint32
	}{
		{".text", 32, 512, pe.IMAGE_SCN_MEM_READ | pe.IMAGE_SCN_MEM_EXECUTE},
		{".rdata", 8, 8, pe.IMAGE_SCN_MEM_READ},
		{".data", 16, 8, pe.IMAGE_SCN_MEM_READ | pe.IMAGE_SCN_MEM_WRITE},
		{".bss", 16, 0, pe.IMAGE_SCN_MEM_READ | pe.IMAGE_SCN_MEM_WRITE | pe.IMAGE_SCN_CNT_UNINITIALIZED_DATA},
		{".debug", 64, 64, 0},
	} {
		sh := pe.SectionHeader32{VirtualSize: sec.memory, SizeOfRawData: sec.raw, PointerToRawData: uint32(dataStart), Characteristics: sec.flags}
		copy(sh.Name[:], sec.name)
		write(sh)
		dataStart += int(sec.raw)
	}
	out.Write(make([]byte, 512+8+8+64))
	if symbols {
		header.PointerToSymbolTable = uint32(out.Len())
		names := []string{"main.(*T).Method", "main.data", "main.zero"}
		strings := []byte{0, 0, 0, 0}
		for i, name := range names {
			sym := pe.COFFSymbol{SectionNumber: 1, StorageClass: 2}
			if i > 0 {
				sym.SectionNumber, sym.Value = 3, uint32((i-1)*8)
			}
			binary.LittleEndian.PutUint32(sym.Name[4:], uint32(len(strings)))
			strings = append(strings, append([]byte(name), 0)...)
			write(sym)
		}
		header.NumberOfSymbols = uint32(len(names))
		binary.LittleEndian.PutUint32(strings, uint32(len(strings)))
		out.Write(strings)
	}
	raw := bytes.Clone(out.Bytes())
	out.Reset()
	write(header)
	copy(raw[132:], out.Bytes())
	return raw
}
