package pe_test

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func buildWindowsFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	src := filepath.Join(dir, "main.go")
	if err := os.WriteFile(src, []byte("package main\nfunc main(){}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main.go): %v", err)
	}
	out := filepath.Join(dir, "fixture.exe")
	cmd := exec.Command("go", "build", "-o", out, src)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=0")
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build windows fixture: %v\n%s", err, b)
	}
	return out
}

func TestOpenNewFileAndSections(t *testing.T) {
	exe := buildWindowsFixture(t)

	f, err := pe.Open(exe)
	if err != nil {
		t.Fatalf("pe.Open: %v", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Errorf("pe.File.Close: %v", err)
		}
	}()

	if f.FileHeader.Machine != pe.IMAGE_FILE_MACHINE_AMD64 {
		t.Fatalf("Machine = %#x, want %#x", f.FileHeader.Machine, pe.IMAGE_FILE_MACHINE_AMD64)
	}
	if f.NumberOfSections == 0 || len(f.Sections) == 0 {
		t.Fatalf("unexpected empty sections: NumberOfSections=%d len=%d", f.NumberOfSections, len(f.Sections))
	}
	if f.Section(".text") == nil {
		t.Fatal("Section(.text) = nil")
	}

	sec := f.Sections[0]
	if _, err := sec.Data(); err != nil {
		t.Fatalf("Section.Data: %v", err)
	}
	r := sec.Open()
	if _, err := io.Copy(io.Discard, r); err != nil {
		t.Fatalf("Section.Open reader: %v", err)
	}

	if _, err := f.ImportedLibraries(); err != nil {
		t.Fatalf("ImportedLibraries: %v", err)
	}
	if _, err := f.ImportedSymbols(); err != nil {
		t.Fatalf("ImportedSymbols: %v", err)
	}
	if d, err := f.DWARF(); err != nil || d == nil {
		t.Fatalf("File.DWARF = (%v, %v), want non-nil data and nil err", d, err)
	}

	file, err := os.Open(exe)
	if err != nil {
		t.Fatalf("os.Open fixture: %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Errorf("fixture file close: %v", err)
		}
	}()
	f2, err := pe.NewFile(file)
	if err != nil {
		t.Fatalf("pe.NewFile: %v", err)
	}
	if err := f2.Close(); err != nil {
		t.Fatalf("NewFile.Close (no-op): %v", err)
	}

	switch f.OptionalHeader.(type) {
	case *pe.OptionalHeader64, *pe.OptionalHeader32:
	default:
		t.Fatalf("unexpected OptionalHeader type: %T", f.OptionalHeader)
	}
}

func TestStringTableAndCOFFSymbol(t *testing.T) {
	st := pe.StringTable([]byte("long_name\x00other\x00"))

	s, err := st.String(4)
	if err != nil {
		t.Fatalf("StringTable.String: %v", err)
	}
	if s != "long_name" {
		t.Fatalf("StringTable.String = %q", s)
	}
	if _, err := st.String(3); err == nil {
		t.Fatal("StringTable.String should fail for offset before table")
	}

	sym := &pe.COFFSymbol{}
	copy(sym.Name[:], []byte("short"))
	name, err := sym.FullName(st)
	if err != nil {
		t.Fatalf("COFFSymbol.FullName(short): %v", err)
	}
	if name != "short" {
		t.Fatalf("FullName(short) = %q", name)
	}

	var off [8]byte
	binary.LittleEndian.PutUint32(off[4:], 4)
	sym.Name = off
	name, err = sym.FullName(st)
	if err != nil {
		t.Fatalf("COFFSymbol.FullName(offset): %v", err)
	}
	if name != "long_name" {
		t.Fatalf("FullName(offset) = %q", name)
	}
}

func TestCOFFSectionAuxAndFormatError(t *testing.T) {
	f := &pe.File{COFFSymbols: []pe.COFFSymbol{{StorageClass: 3, NumberOfAuxSymbols: 1}, {}}}
	aux, err := f.COFFSymbolReadSectionDefAux(0)
	if err != nil {
		t.Fatalf("COFFSymbolReadSectionDefAux: %v", err)
	}
	if aux == nil {
		t.Fatal("COFFSymbolReadSectionDefAux returned nil aux")
	}

	if _, err := f.COFFSymbolReadSectionDefAux(-1); err == nil {
		t.Fatal("COFFSymbolReadSectionDefAux should fail for invalid index")
	}
	f2 := &pe.File{COFFSymbols: []pe.COFFSymbol{{StorageClass: 1, NumberOfAuxSymbols: 1}, {}}}
	if _, err := f2.COFFSymbolReadSectionDefAux(0); err == nil {
		t.Fatal("COFFSymbolReadSectionDefAux should fail for non-static symbol")
	}

	if got := (&pe.FormatError{}).Error(); got == "" {
		t.Fatal("FormatError.Error empty")
	}
}

func TestConstantsAndTypesSanity(t *testing.T) {
	machineVals := []uint16{
		pe.IMAGE_FILE_MACHINE_UNKNOWN,
		pe.IMAGE_FILE_MACHINE_AM33,
		pe.IMAGE_FILE_MACHINE_AMD64,
		pe.IMAGE_FILE_MACHINE_ARM,
		pe.IMAGE_FILE_MACHINE_ARMNT,
		pe.IMAGE_FILE_MACHINE_ARM64,
		pe.IMAGE_FILE_MACHINE_EBC,
		pe.IMAGE_FILE_MACHINE_I386,
		pe.IMAGE_FILE_MACHINE_IA64,
		pe.IMAGE_FILE_MACHINE_LOONGARCH32,
		pe.IMAGE_FILE_MACHINE_LOONGARCH64,
		pe.IMAGE_FILE_MACHINE_M32R,
		pe.IMAGE_FILE_MACHINE_MIPS16,
		pe.IMAGE_FILE_MACHINE_MIPSFPU,
		pe.IMAGE_FILE_MACHINE_MIPSFPU16,
		pe.IMAGE_FILE_MACHINE_POWERPC,
		pe.IMAGE_FILE_MACHINE_POWERPCFP,
		pe.IMAGE_FILE_MACHINE_R4000,
		pe.IMAGE_FILE_MACHINE_SH3,
		pe.IMAGE_FILE_MACHINE_SH3DSP,
		pe.IMAGE_FILE_MACHINE_SH4,
		pe.IMAGE_FILE_MACHINE_SH5,
		pe.IMAGE_FILE_MACHINE_THUMB,
		pe.IMAGE_FILE_MACHINE_WCEMIPSV2,
		pe.IMAGE_FILE_MACHINE_RISCV32,
		pe.IMAGE_FILE_MACHINE_RISCV64,
		pe.IMAGE_FILE_MACHINE_RISCV128,
	}
	if len(machineVals) < 10 {
		t.Fatalf("machine constants unexpectedly short: %d", len(machineVals))
	}

	dirs := []int{
		pe.IMAGE_DIRECTORY_ENTRY_EXPORT,
		pe.IMAGE_DIRECTORY_ENTRY_IMPORT,
		pe.IMAGE_DIRECTORY_ENTRY_RESOURCE,
		pe.IMAGE_DIRECTORY_ENTRY_EXCEPTION,
		pe.IMAGE_DIRECTORY_ENTRY_SECURITY,
		pe.IMAGE_DIRECTORY_ENTRY_BASERELOC,
		pe.IMAGE_DIRECTORY_ENTRY_DEBUG,
		pe.IMAGE_DIRECTORY_ENTRY_ARCHITECTURE,
		pe.IMAGE_DIRECTORY_ENTRY_GLOBALPTR,
		pe.IMAGE_DIRECTORY_ENTRY_TLS,
		pe.IMAGE_DIRECTORY_ENTRY_LOAD_CONFIG,
		pe.IMAGE_DIRECTORY_ENTRY_BOUND_IMPORT,
		pe.IMAGE_DIRECTORY_ENTRY_IAT,
		pe.IMAGE_DIRECTORY_ENTRY_DELAY_IMPORT,
		pe.IMAGE_DIRECTORY_ENTRY_COM_DESCRIPTOR,
	}
	for i, v := range dirs {
		if i != v {
			t.Fatalf("directory constants not contiguous: dirs[%d]=%d", i, v)
		}
	}

	fileFlags := []uint16{
		pe.IMAGE_FILE_RELOCS_STRIPPED,
		pe.IMAGE_FILE_EXECUTABLE_IMAGE,
		pe.IMAGE_FILE_LINE_NUMS_STRIPPED,
		pe.IMAGE_FILE_LOCAL_SYMS_STRIPPED,
		pe.IMAGE_FILE_AGGRESIVE_WS_TRIM,
		pe.IMAGE_FILE_LARGE_ADDRESS_AWARE,
		pe.IMAGE_FILE_BYTES_REVERSED_LO,
		pe.IMAGE_FILE_32BIT_MACHINE,
		pe.IMAGE_FILE_DEBUG_STRIPPED,
		pe.IMAGE_FILE_REMOVABLE_RUN_FROM_SWAP,
		pe.IMAGE_FILE_NET_RUN_FROM_SWAP,
		pe.IMAGE_FILE_SYSTEM,
		pe.IMAGE_FILE_DLL,
		pe.IMAGE_FILE_UP_SYSTEM_ONLY,
		pe.IMAGE_FILE_BYTES_REVERSED_HI,
	}
	var orFileFlags uint16
	for _, v := range fileFlags {
		orFileFlags |= v
	}
	if orFileFlags == 0 {
		t.Fatal("file flags should not all be zero")
	}

	subsystems := []uint16{
		pe.IMAGE_SUBSYSTEM_UNKNOWN,
		pe.IMAGE_SUBSYSTEM_NATIVE,
		pe.IMAGE_SUBSYSTEM_WINDOWS_GUI,
		pe.IMAGE_SUBSYSTEM_WINDOWS_CUI,
		pe.IMAGE_SUBSYSTEM_OS2_CUI,
		pe.IMAGE_SUBSYSTEM_POSIX_CUI,
		pe.IMAGE_SUBSYSTEM_NATIVE_WINDOWS,
		pe.IMAGE_SUBSYSTEM_WINDOWS_CE_GUI,
		pe.IMAGE_SUBSYSTEM_EFI_APPLICATION,
		pe.IMAGE_SUBSYSTEM_EFI_BOOT_SERVICE_DRIVER,
		pe.IMAGE_SUBSYSTEM_EFI_RUNTIME_DRIVER,
		pe.IMAGE_SUBSYSTEM_EFI_ROM,
		pe.IMAGE_SUBSYSTEM_XBOX,
		pe.IMAGE_SUBSYSTEM_WINDOWS_BOOT_APPLICATION,
	}
	if len(subsystems) < 10 {
		t.Fatalf("subsystem constants unexpectedly short: %d", len(subsystems))
	}

	dllChars := []uint16{
		pe.IMAGE_DLLCHARACTERISTICS_HIGH_ENTROPY_VA,
		pe.IMAGE_DLLCHARACTERISTICS_DYNAMIC_BASE,
		pe.IMAGE_DLLCHARACTERISTICS_FORCE_INTEGRITY,
		pe.IMAGE_DLLCHARACTERISTICS_NX_COMPAT,
		pe.IMAGE_DLLCHARACTERISTICS_NO_ISOLATION,
		pe.IMAGE_DLLCHARACTERISTICS_NO_SEH,
		pe.IMAGE_DLLCHARACTERISTICS_NO_BIND,
		pe.IMAGE_DLLCHARACTERISTICS_APPCONTAINER,
		pe.IMAGE_DLLCHARACTERISTICS_WDM_DRIVER,
		pe.IMAGE_DLLCHARACTERISTICS_GUARD_CF,
		pe.IMAGE_DLLCHARACTERISTICS_TERMINAL_SERVER_AWARE,
	}
	if len(dllChars) != 11 {
		t.Fatalf("unexpected dllChars len: %d", len(dllChars))
	}

	sectionFlags := []uint32{
		pe.IMAGE_SCN_CNT_CODE,
		pe.IMAGE_SCN_CNT_INITIALIZED_DATA,
		pe.IMAGE_SCN_CNT_UNINITIALIZED_DATA,
		pe.IMAGE_SCN_LNK_COMDAT,
		pe.IMAGE_SCN_MEM_DISCARDABLE,
		pe.IMAGE_SCN_MEM_EXECUTE,
		pe.IMAGE_SCN_MEM_READ,
		pe.IMAGE_SCN_MEM_WRITE,
	}
	if len(sectionFlags) != 8 {
		t.Fatalf("unexpected sectionFlags len: %d", len(sectionFlags))
	}

	comdat := []uint8{
		pe.IMAGE_COMDAT_SELECT_NODUPLICATES,
		pe.IMAGE_COMDAT_SELECT_ANY,
		pe.IMAGE_COMDAT_SELECT_SAME_SIZE,
		pe.IMAGE_COMDAT_SELECT_EXACT_MATCH,
		pe.IMAGE_COMDAT_SELECT_ASSOCIATIVE,
		pe.IMAGE_COMDAT_SELECT_LARGEST,
	}
	for i, v := range comdat {
		if int(v) != i+1 {
			t.Fatalf("comdat constant mismatch at %d: %d", i, v)
		}
	}

	if pe.COFFSymbolSize != 18 {
		t.Fatalf("COFFSymbolSize = %d, want 18", pe.COFFSymbolSize)
	}

	_ = pe.FileHeader{}
	_ = pe.OptionalHeader32{}
	_ = pe.OptionalHeader64{}
	_ = pe.ImportDirectory{}
	_ = pe.DataDirectory{}
	_ = pe.SectionHeader{}
	_ = pe.SectionHeader32{}
	_ = pe.Reloc{}
	_ = pe.Symbol{}
	_ = pe.COFFSymbolAuxFormat5{}
	st := pe.StringTable(bytes.Repeat([]byte{0}, 4))
	if len(st) != 4 {
		t.Fatalf("StringTable conversion length = %d, want 4", len(st))
	}
}
