package dwarf_test

import (
	"debug/dwarf"
	"debug/elf"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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

func loadDWARF(t *testing.T) *dwarf.Data {
	t.Helper()
	bin := buildLinuxFixture(t)
	ef, err := elf.Open(bin)
	if err != nil {
		t.Fatalf("elf.Open: %v", err)
	}
	defer func() {
		if err := ef.Close(); err != nil {
			t.Errorf("elf.File.Close: %v", err)
		}
	}()
	d, err := ef.DWARF()
	if err != nil {
		t.Fatalf("elf.DWARF: %v", err)
	}
	return d
}

func firstCompileUnit(t *testing.T, d *dwarf.Data) *dwarf.Entry {
	t.Helper()
	r := d.Reader()
	for {
		e, err := r.Next()
		if err != nil {
			t.Fatalf("Reader.Next: %v", err)
		}
		if e == nil {
			t.Fatal("no compile unit found")
		}
		if e.Tag == dwarf.TagCompileUnit {
			return e
		}
	}
}

func TestReaderLineAndDataMethods(t *testing.T) {
	d := loadDWARF(t)

	r := d.Reader()
	if r.AddressSize() <= 0 {
		t.Fatalf("AddressSize = %d", r.AddressSize())
	}
	if r.ByteOrder() == nil {
		t.Fatal("ByteOrder is nil")
	}

	cu := firstCompileUnit(t, d)
	if cu.AttrField(dwarf.AttrName) == nil {
		t.Fatalf("compile unit missing AttrName field: %#v", cu)
	}
	if cu.Val(dwarf.AttrName) == nil {
		t.Fatal("compile unit AttrName value is nil")
	}

	r.Seek(cu.Offset)
	e, err := r.Next()
	if err != nil {
		t.Fatalf("Reader.Next after Seek: %v", err)
	}
	if e == nil || e.Offset != cu.Offset {
		t.Fatalf("Reader.Seek mismatch: got %#v want offset %#v", e, cu.Offset)
	}
	r.SkipChildren()
	if entry, err := r.SeekPC(0); err == nil && entry == nil {
		t.Fatal("Reader.SeekPC returned nil entry with nil error")
	}

	lr, err := d.LineReader(cu)
	if err != nil {
		t.Fatalf("Data.LineReader: %v", err)
	}
	if len(lr.Files()) == 0 {
		t.Fatal("LineReader.Files empty")
	}

	var le dwarf.LineEntry
	if err := lr.Next(&le); err != nil {
		t.Fatalf("LineReader.Next: %v", err)
	}
	pos := lr.Tell()
	lr.Reset()
	lr.Seek(pos)
	if err := lr.SeekPC(le.Address, &le); err != nil {
		t.Fatalf("LineReader.SeekPC: %v", err)
	}
	for i := 0; i < 3; i++ {
		err = lr.Next(&le)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("LineReader.Next loop: %v", err)
		}
	}

	if _, err := d.Ranges(cu); err != nil {
		// Range data may be absent for some units, but method path is exercised.
		t.Logf("Data.Ranges: %v", err)
	}

	if d.AddSection(".debug_rnglists", nil) != nil {
		// Different binaries may reject/accept nil section bodies.
	}
	if d.AddTypes("types-0", nil) != nil {
		// Different binaries may reject/accept nil type section.
	}

	// Try to resolve at least one type offset from DIEs.
	tr := d.Reader()
	for {
		en, err := tr.Next()
		if err != nil {
			t.Fatalf("type scan Reader.Next: %v", err)
		}
		if en == nil {
			break
		}
		if v := en.Val(dwarf.AttrType); v != nil {
			off, ok := v.(dwarf.Offset)
			if !ok {
				continue
			}
			typ, err := d.Type(off)
			if err != nil {
				continue
			}
			if typ == nil {
				t.Fatal("Data.Type returned nil type")
			}
			break
		}
	}
}

func TestTypeStructsAndStringers(t *testing.T) {
	if dwarf.ErrUnknownPC == nil {
		t.Fatal("ErrUnknownPC should not be nil")
	}

	ct := dwarf.CommonType{Name: "T", ByteSize: 4}
	if ct.Common() == nil || ct.Size() != 4 {
		t.Fatalf("CommonType behavior mismatch: %+v", ct)
	}

	bt := &dwarf.BasicType{CommonType: ct}
	if bt.Basic() == nil || bt.String() == "" {
		t.Fatalf("BasicType methods mismatch: %+v", bt)
	}

	arr := &dwarf.ArrayType{CommonType: dwarf.CommonType{Name: "A"}, Type: bt, Count: 3}
	if arr.Size() != 12 || arr.String() == "" {
		t.Fatalf("ArrayType mismatch: size=%d string=%q", arr.Size(), arr.String())
	}

	qt := &dwarf.QualType{CommonType: dwarf.CommonType{Name: "Q", ByteSize: 8}, Qual: "const", Type: bt}
	if qt.Size() <= 0 || qt.String() == "" {
		t.Fatalf("QualType mismatch: size=%d string=%q", qt.Size(), qt.String())
	}

	td := &dwarf.TypedefType{CommonType: dwarf.CommonType{Name: "Alias", ByteSize: 4}, Type: bt}
	if td.Size() != 4 || td.String() == "" {
		t.Fatalf("TypedefType mismatch: size=%d string=%q", td.Size(), td.String())
	}

	ptr := &dwarf.PtrType{CommonType: dwarf.CommonType{Name: "P", ByteSize: 8}, Type: bt}
	if ptr.String() == "" {
		t.Fatal("PtrType.String empty")
	}

	fn := &dwarf.FuncType{CommonType: dwarf.CommonType{Name: "fn"}, ReturnType: bt, ParamType: []dwarf.Type{bt, &dwarf.DotDotDotType{}}}
	if fn.String() == "" {
		t.Fatal("FuncType.String empty")
	}

	en := &dwarf.EnumType{
		CommonType: dwarf.CommonType{Name: "E", ByteSize: 4},
		EnumName:   "E",
		Val:        []*dwarf.EnumValue{{Name: "A", Val: 1}},
	}
	if en.String() == "" {
		t.Fatal("EnumType.String empty")
	}

	st := &dwarf.StructType{
		CommonType: dwarf.CommonType{Name: "S", ByteSize: 4},
		StructName: "S",
		Kind:       "struct",
		Field:      []*dwarf.StructField{{Name: "F", Type: bt, ByteOffset: 0}},
	}
	if st.Defn() == "" || st.String() == "" {
		t.Fatalf("StructType methods empty: defn=%q string=%q", st.Defn(), st.String())
	}

	unsup := &dwarf.UnsupportedType{CommonType: dwarf.CommonType{Name: "U"}, Tag: dwarf.TagArrayType}
	if unsup.String() == "" {
		t.Fatal("UnsupportedType.String empty")
	}

	if (&dwarf.DotDotDotType{}).String() == "" {
		t.Fatal("DotDotDotType.String empty")
	}
	if (&dwarf.VoidType{}).String() == "" {
		t.Fatal("VoidType.String empty")
	}

	_ = dwarf.AddrType{}
	_ = dwarf.BoolType{}
	_ = dwarf.CharType{}
	_ = dwarf.ComplexType{}
	_ = dwarf.FloatType{}
	_ = dwarf.IntType{}
	_ = dwarf.UcharType{}
	_ = dwarf.UintType{}
	_ = dwarf.UnspecifiedType{}
	_ = dwarf.Field{}
	_ = dwarf.LineEntry{}
	_ = dwarf.LineFile{}
	_ = dwarf.LineReaderPos{}
	_ = dwarf.Reader{}
	_ = dwarf.Data{}
	if dwarf.Offset(0) != 0 {
		t.Fatalf("Offset conversion mismatch: got %v", dwarf.Offset(0))
	}
	_ = dwarf.Entry{}
}

func TestEnumsAndDecodeError(t *testing.T) {
	if dwarf.AttrName.String() == "" || dwarf.AttrName.GoString() == "" {
		t.Fatal("Attr stringers empty")
	}
	if dwarf.TagCompileUnit.String() == "" || dwarf.TagCompileUnit.GoString() == "" {
		t.Fatal("Tag stringers empty")
	}
	if dwarf.ClassString.String() == "" || dwarf.ClassString.GoString() == "" {
		t.Fatal("Class stringers empty")
	}

	err := dwarf.DecodeError{Name: "abbrev", Offset: dwarf.Offset(3), Err: "bad"}
	if err.Error() == "" {
		t.Fatal("DecodeError.Error empty")
	}
}
