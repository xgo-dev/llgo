package elf_test

import (
	"debug/elf"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func buildLinuxFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	src := filepath.Join(dir, "main.go")
	if err := os.WriteFile(src, []byte("package main\nfunc main(){}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main.go): %v", err)
	}
	out := filepath.Join(dir, "fixture")
	cmd := exec.Command("go", "build", "-o", out, src)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0")
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build linux fixture: %v\n%s", err, b)
	}
	return out
}

func TestOpenELFAndCoreMethods(t *testing.T) {
	bin := buildLinuxFixture(t)
	f, err := elf.Open(bin)
	if err != nil {
		t.Fatalf("elf.Open: %v", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Errorf("elf.File.Close: %v", err)
		}
	}()

	if f.Class == elf.ELFCLASSNONE {
		t.Fatalf("unexpected ELF class: %v", f.Class)
	}
	if f.Section(".text") == nil {
		t.Fatal("Section(.text)=nil")
	}
	if f.SectionByType(elf.SHT_PROGBITS) == nil {
		t.Fatal("SectionByType(SHT_PROGBITS)=nil")
	}

	sec := f.Section(".text")
	b, err := sec.Data()
	if err != nil {
		t.Fatalf("Section.Data: %v", err)
	}
	if len(b) == 0 {
		t.Fatal("Section.Data empty")
	}
	if _, err := sec.Open().Read(make([]byte, 1)); err != nil {
		t.Fatalf("Section.Open read: %v", err)
	}

	if len(f.Progs) > 0 {
		if _, err := io.Copy(io.Discard, f.Progs[0].Open()); err != nil {
			t.Fatalf("Prog.Open read: %v", err)
		}
	}

	if _, err := f.Symbols(); err != nil && !errors.Is(err, elf.ErrNoSymbols) {
		t.Fatalf("File.Symbols: %v", err)
	}
	if _, err := f.DynamicSymbols(); err != nil && !errors.Is(err, elf.ErrNoSymbols) {
		t.Fatalf("File.DynamicSymbols: %v", err)
	}
	if _, err := f.ImportedLibraries(); err != nil {
		t.Fatalf("File.ImportedLibraries: %v", err)
	}
	if _, err := f.ImportedSymbols(); err != nil && !errors.Is(err, elf.ErrNoSymbols) {
		t.Fatalf("File.ImportedSymbols: %v", err)
	}
	if _, err := f.DynString(elf.DT_NEEDED); err != nil {
		t.Fatalf("File.DynString: %v", err)
	}
	if _, err := f.DynValue(elf.DT_NEEDED); err != nil {
		t.Fatalf("File.DynValue: %v", err)
	}
	if _, err := f.DynamicVersions(); err != nil && !errors.Is(err, elf.ErrNoSymbols) {
		t.Fatalf("File.DynamicVersions: %v", err)
	}
	if _, err := f.DynamicVersionNeeds(); err != nil && !errors.Is(err, elf.ErrNoSymbols) {
		t.Fatalf("File.DynamicVersionNeeds: %v", err)
	}
	if d, err := f.DWARF(); err != nil || d == nil {
		t.Fatalf("File.DWARF = (%v, %v), want non-nil data and nil err", d, err)
	}

	rf, err := os.Open(bin)
	if err != nil {
		t.Fatalf("os.Open fixture: %v", err)
	}
	defer func() {
		if err := rf.Close(); err != nil {
			t.Errorf("fixture file close: %v", err)
		}
	}()
	f2, err := elf.NewFile(rf)
	if err != nil {
		t.Fatalf("elf.NewFile: %v", err)
	}
	if err := f2.Close(); err != nil {
		t.Fatalf("elf.NewFile.Close: %v", err)
	}
}

func TestEnumsConstantsAndHelpers(t *testing.T) {
	if elf.ErrNoSymbols == nil {
		t.Fatal("ErrNoSymbols should not be nil")
	}
	if elf.ARM_MAGIC_TRAMP_NUMBER == 0 {
		t.Fatal("ARM_MAGIC_TRAMP_NUMBER should not be zero")
	}
	if !strings.Contains(elf.ELFMAG, "ELF") {
		t.Fatalf("ELFMAG = %q", elf.ELFMAG)
	}
	if elf.Sym32Size != 16 || elf.Sym64Size != 24 {
		t.Fatalf("unexpected symbol sizes: %d/%d", elf.Sym32Size, elf.Sym64Size)
	}

	if elf.EI_CLASS != 4 || elf.EI_DATA != 5 || elf.EI_VERSION != 6 || elf.EI_OSABI != 7 || elf.EI_ABIVERSION != 8 || elf.EI_PAD != 9 || elf.EI_NIDENT != 16 {
		t.Fatal("EI_* constants changed unexpectedly")
	}

	info32 := elf.R_INFO32(0x12, 0x34)
	if elf.R_SYM32(info32) != 0x12 || elf.R_TYPE32(info32) != 0x34 {
		t.Fatalf("R_INFO32 helpers mismatch: %#x", info32)
	}
	info64 := elf.R_INFO(0x56, 0x78)
	if elf.R_SYM64(info64) != 0x56 || elf.R_TYPE64(info64) != 0x78 {
		t.Fatalf("R_INFO helpers mismatch: %#x", info64)
	}
	if elf.ST_INFO(elf.STB_GLOBAL, elf.STT_FUNC) == 0 {
		t.Fatal("ST_INFO should produce non-zero for GLOBAL/FUNC")
	}

	if elf.ELFCLASS64.String() == "" || elf.ELFCLASS64.GoString() == "" {
		t.Fatal("Class stringers empty")
	}
	if elf.COMPRESS_ZLIB.String() == "" || elf.COMPRESS_ZLIB.GoString() == "" {
		t.Fatal("CompressionType stringers empty")
	}
	if elf.ELFDATA2LSB.String() == "" || elf.ELFDATA2LSB.GoString() == "" {
		t.Fatal("Data stringers empty")
	}
	if elf.DF_BIND_NOW.String() == "" || elf.DF_BIND_NOW.GoString() == "" {
		t.Fatal("DynFlag stringers empty")
	}
	if elf.DF_1_NOW.String() == "" || elf.DF_1_NOW.GoString() == "" {
		t.Fatal("DynFlag1 stringers empty")
	}
	if elf.DT_NEEDED.String() == "" || elf.DT_NEEDED.GoString() == "" {
		t.Fatal("DynTag stringers empty")
	}
	if elf.EM_X86_64.String() == "" || elf.EM_X86_64.GoString() == "" {
		t.Fatal("Machine stringers empty")
	}
	if elf.NType(0).String() == "" || elf.NType(0).GoString() == "" {
		t.Fatal("NType stringers empty")
	}
	if elf.ELFOSABI_LINUX.String() == "" || elf.ELFOSABI_LINUX.GoString() == "" {
		t.Fatal("OSABI stringers empty")
	}
	if elf.PF_R.String() == "" || elf.PF_R.GoString() == "" {
		t.Fatal("ProgFlag stringers empty")
	}
	if elf.PT_LOAD.String() == "" || elf.PT_LOAD.GoString() == "" {
		t.Fatal("ProgType stringers empty")
	}
	if elf.SHF_ALLOC.String() == "" || elf.SHF_ALLOC.GoString() == "" {
		t.Fatal("SectionFlag stringers empty")
	}
	if elf.SHN_UNDEF.String() == "" || elf.SHN_UNDEF.GoString() == "" {
		t.Fatal("SectionIndex stringers empty")
	}
	if elf.SHT_PROGBITS.String() == "" || elf.SHT_PROGBITS.GoString() == "" {
		t.Fatal("SectionType stringers empty")
	}
	if elf.STB_GLOBAL.String() == "" || elf.STB_GLOBAL.GoString() == "" {
		t.Fatal("SymBind stringers empty")
	}
	if elf.STT_FUNC.String() == "" || elf.STT_FUNC.GoString() == "" {
		t.Fatal("SymType stringers empty")
	}
	if elf.STV_DEFAULT.String() == "" || elf.STV_DEFAULT.GoString() == "" {
		t.Fatal("SymVis stringers empty")
	}
	if elf.ET_EXEC.String() == "" || elf.ET_EXEC.GoString() == "" {
		t.Fatal("Type stringers empty")
	}
	if elf.EV_CURRENT.String() == "" || elf.EV_CURRENT.GoString() == "" {
		t.Fatal("Version stringers empty")
	}

	if elf.R_386(0).String() == "" || elf.R_386(0).GoString() == "" {
		t.Fatal("R_386 stringers empty")
	}
	if elf.R_390(0).String() == "" || elf.R_390(0).GoString() == "" {
		t.Fatal("R_390 stringers empty")
	}
	if elf.R_AARCH64(0).String() == "" || elf.R_AARCH64(0).GoString() == "" {
		t.Fatal("R_AARCH64 stringers empty")
	}
	if elf.R_ALPHA(0).String() == "" || elf.R_ALPHA(0).GoString() == "" {
		t.Fatal("R_ALPHA stringers empty")
	}
	if elf.R_ARM(0).String() == "" || elf.R_ARM(0).GoString() == "" {
		t.Fatal("R_ARM stringers empty")
	}
	if elf.R_LARCH(0).String() == "" || elf.R_LARCH(0).GoString() == "" {
		t.Fatal("R_LARCH stringers empty")
	}
	if elf.R_MIPS(0).String() == "" || elf.R_MIPS(0).GoString() == "" {
		t.Fatal("R_MIPS stringers empty")
	}
	if elf.R_PPC(0).String() == "" || elf.R_PPC(0).GoString() == "" {
		t.Fatal("R_PPC stringers empty")
	}
	if elf.R_PPC64(0).String() == "" || elf.R_PPC64(0).GoString() == "" {
		t.Fatal("R_PPC64 stringers empty")
	}
	if elf.R_RISCV(0).String() == "" || elf.R_RISCV(0).GoString() == "" {
		t.Fatal("R_RISCV stringers empty")
	}
	if elf.R_SPARC(0).String() == "" || elf.R_SPARC(0).GoString() == "" {
		t.Fatal("R_SPARC stringers empty")
	}
	if elf.R_X86_64(0).String() == "" || elf.R_X86_64(0).GoString() == "" {
		t.Fatal("R_X86_64 stringers empty")
	}

	vi := elf.VersionIndex(0x8002)
	if !vi.IsHidden() || vi.Index() != 2 {
		t.Fatalf("VersionIndex helpers mismatch: hidden=%v index=%d", vi.IsHidden(), vi.Index())
	}
}

func TestStructsAndErrorType(t *testing.T) {
	fe := &elf.FormatError{}
	if fe.Error() == "" {
		t.Fatal("FormatError.Error empty")
	}

	_ = elf.Chdr32{}
	_ = elf.Chdr64{}
	_ = elf.Dyn32{}
	_ = elf.Dyn64{}
	_ = elf.DynamicVersion{}
	_ = elf.DynamicVersionDep{}
	if flag := elf.DynamicVersionFlag(elf.VER_FLG_BASE); flag != elf.VER_FLG_BASE {
		t.Fatalf("DynamicVersionFlag conversion mismatch: got %v, want %v", flag, elf.VER_FLG_BASE)
	}
	_ = elf.DynamicVersionNeed{}
	_ = elf.File{}
	_ = elf.FileHeader{}
	_ = elf.Header32{}
	_ = elf.Header64{}
	_ = elf.ImportedSymbol{}
	_ = elf.Prog{}
	_ = elf.Prog32{}
	_ = elf.Prog64{}
	_ = elf.ProgHeader{}
	_ = elf.Rel32{}
	_ = elf.Rel64{}
	_ = elf.Rela32{}
	_ = elf.Rela64{}
	_ = elf.Section{}
	_ = elf.Section32{}
	_ = elf.Section64{}
	_ = elf.SectionHeader{}
	_ = elf.Sym32{}
	_ = elf.Sym64{}
	_ = elf.Symbol{}
}
