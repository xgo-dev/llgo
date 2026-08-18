package macho_test

import (
	"bytes"
	"debug/macho"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func buildDarwinFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	src := filepath.Join(dir, "main.go")
	if err := os.WriteFile(src, []byte("package main\nfunc main(){}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main.go): %v", err)
	}
	out := filepath.Join(dir, "fixture")
	cmd := exec.Command("go", "build", "-o", out, src)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOOS=darwin", "GOARCH=amd64", "CGO_ENABLED=0")
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build darwin fixture: %v\n%s", err, b)
	}
	return out
}

func TestOpenMachoFixtureAndCoreMethods(t *testing.T) {
	bin := buildDarwinFixture(t)
	f, err := macho.Open(bin)
	if err != nil {
		t.Fatalf("macho.Open: %v", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Errorf("macho.File.Close: %v", err)
		}
	}()

	if f.Section("__text") == nil {
		t.Fatal("Section(__text)=nil")
	}
	sec := f.Section("__text")
	secData, err := sec.Data()
	if err != nil {
		t.Fatalf("Section.Data: %v", err)
	}
	if len(secData) == 0 {
		t.Fatal("Section.Data returned empty")
	}
	if _, err := sec.Open().Read(make([]byte, 1)); err != nil {
		t.Fatalf("Section.Open read: %v", err)
	}

	seg := f.Segment("__TEXT")
	if seg == nil {
		t.Fatal("Segment(__TEXT)=nil")
	}
	segData, err := seg.Data()
	if err != nil {
		t.Fatalf("Segment.Data: %v", err)
	}
	if len(segData) == 0 {
		t.Fatal("Segment.Data returned empty")
	}
	if _, err := seg.Open().Read(make([]byte, 1)); err != nil {
		t.Fatalf("Segment.Open read: %v", err)
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

	rf, err := os.Open(bin)
	if err != nil {
		t.Fatalf("os.Open fixture: %v", err)
	}
	defer func() {
		if err := rf.Close(); err != nil {
			t.Errorf("fixture file close: %v", err)
		}
	}()
	f2, err := macho.NewFile(rf)
	if err != nil {
		t.Fatalf("macho.NewFile: %v", err)
	}
	if err := f2.Close(); err != nil {
		t.Fatalf("macho.NewFile.Close: %v", err)
	}
}

func TestFatAndErrorPaths(t *testing.T) {
	bin := buildDarwinFixture(t)

	if ff, err := macho.OpenFat(bin); err == nil || err != macho.ErrNotFat || ff != nil {
		t.Fatalf("OpenFat(thin) = (%v, %v), want (nil, ErrNotFat)", ff, err)
	}

	b, err := os.ReadFile(bin)
	if err != nil {
		t.Fatalf("ReadFile fixture: %v", err)
	}
	if ff, err := macho.NewFatFile(bytes.NewReader(b)); err == nil || err != macho.ErrNotFat || ff != nil {
		t.Fatalf("NewFatFile(thin) = (%v, %v), want (nil, ErrNotFat)", ff, err)
	}

	ff := &macho.FatFile{
		Magic: macho.MagicFat,
		Arches: []macho.FatArch{{
			FatArchHeader: macho.FatArchHeader{Cpu: macho.CpuAmd64, Align: 12},
		}},
	}
	if err := ff.Close(); err != nil {
		t.Fatalf("FatFile.Close: %v", err)
	}

	if macho.ErrNotFat == nil || macho.ErrNotFat.Error() == "" {
		t.Fatal("ErrNotFat should be non-nil and have message")
	}
	var ferr *macho.FormatError = macho.ErrNotFat
	if ferr.Error() == "" {
		t.Fatal("FormatError.Error empty")
	}
}

func TestConstantsEnumsAndStructs(t *testing.T) {
	if macho.Magic32 == 0 || macho.Magic64 == 0 || macho.MagicFat == 0 {
		t.Fatal("magic constants should be non-zero")
	}

	flags := []uint32{
		macho.FlagNoUndefs,
		macho.FlagIncrLink,
		macho.FlagDyldLink,
		macho.FlagBindAtLoad,
		macho.FlagPrebound,
		macho.FlagSplitSegs,
		macho.FlagLazyInit,
		macho.FlagTwoLevel,
		macho.FlagForceFlat,
		macho.FlagNoMultiDefs,
		macho.FlagNoFixPrebinding,
		macho.FlagPrebindable,
		macho.FlagAllModsBound,
		macho.FlagSubsectionsViaSymbols,
		macho.FlagCanonical,
		macho.FlagWeakDefines,
		macho.FlagBindsToWeak,
		macho.FlagAllowStackExecution,
		macho.FlagRootSafe,
		macho.FlagSetuidSafe,
		macho.FlagNoReexportedDylibs,
		macho.FlagPIE,
		macho.FlagDeadStrippableDylib,
		macho.FlagHasTLVDescriptors,
		macho.FlagNoHeapExecution,
		macho.FlagAppExtensionSafe,
	}
	var mask uint32
	for _, f := range flags {
		mask |= f
	}
	if mask == 0 {
		t.Fatal("combined flag mask should be non-zero")
	}

	if macho.CpuAmd64.String() == "" || macho.CpuAmd64.GoString() == "" {
		t.Fatal("Cpu string methods empty")
	}
	if macho.LoadCmdSegment.String() == "" || macho.LoadCmdSegment.GoString() == "" {
		t.Fatal("LoadCmd string methods empty")
	}
	if macho.TypeObj.String() == "" || macho.TypeObj.GoString() == "" {
		t.Fatal("Type string methods empty")
	}
	if macho.RelocTypeARM(0).String() == "" || macho.RelocTypeARM(0).GoString() == "" {
		t.Fatal("RelocTypeARM string methods empty")
	}
	if macho.RelocTypeARM64(0).String() == "" || macho.RelocTypeARM64(0).GoString() == "" {
		t.Fatal("RelocTypeARM64 string methods empty")
	}
	if macho.RelocTypeGeneric(0).String() == "" || macho.RelocTypeGeneric(0).GoString() == "" {
		t.Fatal("RelocTypeGeneric string methods empty")
	}
	if macho.RelocTypeX86_64(0).String() == "" || macho.RelocTypeX86_64(0).GoString() == "" {
		t.Fatal("RelocTypeX86_64 string methods empty")
	}

	lb := macho.LoadBytes{1, 2, 3}
	if raw := lb.Raw(); len(raw) != 3 {
		t.Fatalf("LoadBytes.Raw len = %d, want 3", len(raw))
	}
	var ld macho.Load = lb
	if len(ld.Raw()) != 3 {
		t.Fatalf("Load.Raw len = %d, want 3", len(ld.Raw()))
	}

	_ = macho.Dylib{}
	_ = macho.DylibCmd{}
	_ = macho.Dysymtab{}
	_ = macho.DysymtabCmd{}
	_ = macho.FileHeader{}
	_ = macho.Nlist32{}
	_ = macho.Nlist64{}
	_ = macho.Regs386{}
	_ = macho.RegsAMD64{}
	_ = macho.Reloc{}
	_ = macho.Rpath{}
	_ = macho.RpathCmd{}
	_ = macho.Section32{}
	_ = macho.Section64{}
	_ = macho.SectionHeader{}
	_ = macho.Segment32{}
	_ = macho.Segment64{}
	_ = macho.SegmentHeader{}
	_ = macho.Symbol{}
	_ = macho.Symtab{}
	_ = macho.SymtabCmd{}
	_ = macho.Thread{}
}
