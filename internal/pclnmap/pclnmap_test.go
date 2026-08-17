package pclnmap

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"

	"github.com/xgo-dev/llgo/internal/build/funcinfo"
)

func sampleData(t *testing.T) Data {
	t.Helper()
	table, err := funcinfo.EncodeWithPCLines(
		[]funcinfo.Record{{Symbol: "main.main", Name: "main.main", File: "/src/main.go", Line: 3}},
		[]funcinfo.PCLineRecord{{ID: 7, Symbol: "main.main", File: "/src/main.go", Line: 4}},
	)
	if err != nil {
		t.Fatal(err)
	}
	var identity [32]byte
	copy(identity[:], "one exact linked executable")
	return Data{
		GOOS:        "linux",
		GOARCH:      "amd64",
		PointerSize: 8,
		ImageBase:   0x400000,
		TextStart:   0x401000,
		TextEnd:     0x403000,
		Identity:    identity,
		Table:       table,
		SymbolIndex: []SymbolIndexEntry{{SymbolID: 11, FuncIndex: 1}},
		EntrySites:  []Site{{PCOffset: 0x1010, ID: 11}},
		PCSites:     []Site{{PCOffset: 0x1020, ID: 7}},
	}
}

func TestEncodeParse(t *testing.T) {
	raw, err := Encode(sampleData(t))
	if err != nil {
		t.Fatal(err)
	}
	view, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if view.GOOSCode != GOOSLinux || view.GOARCHCode != GOARCHAMD64 || view.PointerSize != 8 {
		t.Fatalf("target = (%d, %d, %d)", view.GOOSCode, view.GOARCHCode, view.PointerSize)
	}
	if view.TextStartOffset != 0x1000 || view.TextEndOffset != 0x3000 {
		t.Fatalf("text offsets = [%#x,%#x)", view.TextStartOffset, view.TextEndOffset)
	}
	if got := view.Sections[descRecords].Count; got != 1 {
		t.Fatalf("record count = %d", got)
	}
	if got := view.Sections[descPCSites].Count; got != 1 {
		t.Fatalf("pcsite count = %d", got)
	}
	raw2, err := Encode(sampleData(t))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, raw2) {
		t.Fatal("encoding is not deterministic")
	}
}

func TestVersion4FunctionCentricDescriptorLayout(t *testing.T) {
	if Version != 4 {
		t.Fatalf("format version = %d, want 4", Version)
	}
	if descRecords != 0 || descPCLines != 1 || descStrings != 2 ||
		descStringOffsets != 3 || descHash != 4 || descSymbolIndex != 5 ||
		descEntrySites != 6 || descPCSites != 7 || descCount != 8 {
		t.Fatalf("unexpected v4 descriptor layout: records=%d pclines=%d strings=%d offsets=%d hash=%d symbols=%d entries=%d pcsites=%d count=%d",
			descRecords, descPCLines, descStrings, descStringOffsets, descHash,
			descSymbolIndex, descEntrySites, descPCSites, descCount)
	}
}

func TestEncodePreservesUint32StringIDs(t *testing.T) {
	data := sampleData(t)
	const (
		symbolNameID = uint32(1 << 16)
		fileNameID   = symbolNameID + 1
	)
	data.Table.Records[0].SymbolName = symbolNameID
	data.Table.PCLines[0].FileName = fileNameID
	data.Table.StringOffsets = make([]uint32, fileNameID+1)

	raw, err := Encode(data)
	if err != nil {
		t.Fatal(err)
	}
	view, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	record := raw[view.Sections[descRecords].Offset:]
	if got := binary.LittleEndian.Uint32(record[4:]); got != symbolNameID {
		t.Fatalf("symbol name id = %d, want %d", got, symbolNameID)
	}
	pcLine := raw[view.Sections[descPCLines].Offset:]
	if got := binary.LittleEndian.Uint32(pcLine[16:]); got != fileNameID {
		t.Fatalf("pcline file name id = %d, want %d", got, fileNameID)
	}
}

func TestParseRejectsCorruptionAndTruncation(t *testing.T) {
	raw, err := Encode(sampleData(t))
	if err != nil {
		t.Fatal(err)
	}
	corrupt := append([]byte(nil), raw...)
	corrupt[len(corrupt)-1] ^= 0xff
	if _, err := Parse(corrupt); err == nil {
		t.Fatal("Parse accepted corrupt payload")
	}
	if _, err := Parse(raw[:len(raw)-1]); err == nil {
		t.Fatal("Parse accepted truncated payload")
	}
	wrongABI := append([]byte(nil), raw...)
	wrongABI[headerABIVersion]++
	if _, err := Parse(wrongABI); err == nil {
		t.Fatal("Parse accepted an incompatible runtime ABI")
	}
}

func TestParseRejectsOverlappingSections(t *testing.T) {
	raw, err := Encode(sampleData(t))
	if err != nil {
		t.Fatal(err)
	}
	recordOffset := binary.LittleEndian.Uint64(raw[headerDescriptors:])
	stringDescriptor := headerDescriptors + descStrings*16
	binary.LittleEndian.PutUint64(raw[stringDescriptor:], recordOffset)
	if _, err := Parse(raw); err == nil {
		t.Fatal("Parse accepted overlapping record and string sections")
	}
}

func TestParseRejectsMisalignedSections(t *testing.T) {
	raw, err := Encode(sampleData(t))
	if err != nil {
		t.Fatal(err)
	}
	pclineDescriptor := headerDescriptors + descPCLines*16
	offset := binary.LittleEndian.Uint64(raw[pclineDescriptor:])
	binary.LittleEndian.PutUint64(raw[pclineDescriptor:], offset+1)
	if _, err := Parse(raw); err == nil {
		t.Fatal("Parse accepted a misaligned pcline section")
	}
}

func TestEncodeRejectsUnsupportedTarget(t *testing.T) {
	data := sampleData(t)
	data.GOOS = "windows"
	if _, err := Encode(data); err == nil {
		t.Fatal("Encode accepted unsupported GOOS")
	}
	data = sampleData(t)
	data.GOARCH = "386"
	if _, err := Encode(data); err == nil {
		t.Fatal("Encode accepted unsupported GOARCH")
	}
	data = sampleData(t)
	data.PointerSize = 4
	if _, err := Encode(data); err == nil {
		t.Fatal("Encode accepted 32-bit pointer size")
	}
}

func TestEncodeDarwinArm64AndInvalidTextRanges(t *testing.T) {
	data := sampleData(t)
	data.GOOS, data.GOARCH = "darwin", "arm64"
	raw, err := Encode(data)
	if err != nil {
		t.Fatal(err)
	}
	view, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if view.GOOSCode != GOOSDarwin || view.GOARCHCode != GOARCHARM64 {
		t.Fatalf("target codes = (%d, %d)", view.GOOSCode, view.GOARCHCode)
	}

	for name, mutate := range map[string]func(*Data){
		"start below image": func(data *Data) { data.TextStart = data.ImageBase - 1 },
		"empty text":        func(data *Data) { data.TextEnd = data.TextStart },
	} {
		t.Run(name, func(t *testing.T) {
			data := sampleData(t)
			mutate(&data)
			if _, err := Encode(data); err == nil {
				t.Fatal("Encode accepted an invalid text range")
			}
		})
	}
}

func TestParseRejectsInvalidHeadersAndDescriptors(t *testing.T) {
	original, err := Encode(sampleData(t))
	if err != nil {
		t.Fatal(err)
	}
	clone := func() []byte { return append([]byte(nil), original...) }
	tests := map[string]func([]byte) []byte{
		"short header": func(raw []byte) []byte { return raw[:HeaderSize-1] },
		"magic": func(raw []byte) []byte {
			raw[headerMagic] ^= 0xff
			return raw
		},
		"version": func(raw []byte) []byte {
			binary.LittleEndian.PutUint32(raw[headerVersion:], Version+1)
			return raw
		},
		"header size": func(raw []byte) []byte {
			binary.LittleEndian.PutUint32(raw[headerSize:], HeaderSize+1)
			return raw
		},
		"file size": func(raw []byte) []byte {
			binary.LittleEndian.PutUint64(raw[headerFileSize:], uint64(len(raw)+1))
			return raw
		},
		"endianness": func(raw []byte) []byte {
			raw[headerEndian] = 0xff
			return raw
		},
		"pointer size": func(raw []byte) []byte {
			raw[headerPointerSize] = 4
			return raw
		},
		"text range": func(raw []byte) []byte {
			binary.LittleEndian.PutUint64(raw[headerTextEnd:], binary.LittleEndian.Uint64(raw[headerTextStart:]))
			return raw
		},
		"section count overflow": func(raw []byte) []byte {
			binary.LittleEndian.PutUint64(raw[headerDescriptors+8:], math.MaxUint64)
			return raw
		},
		"section before header": func(raw []byte) []byte {
			binary.LittleEndian.PutUint64(raw[headerDescriptors:], uint64(HeaderSize-1))
			return raw
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse(mutate(clone())); err == nil {
				t.Fatal("Parse accepted invalid input")
			}
		})
	}
}

func TestParseAllowsEmptySections(t *testing.T) {
	raw, err := Encode(sampleData(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, index := range []int{descRecords, descStrings} {
		base := headerDescriptors + index*16
		binary.LittleEndian.PutUint64(raw[base+8:], 0)
	}
	if _, err := Parse(raw); err != nil {
		t.Fatalf("Parse rejected empty sections: %v", err)
	}
}

func TestDescriptorAndOverflowHelpers(t *testing.T) {
	if got := descriptorSize(descCount); got != 0 {
		t.Fatalf("descriptorSize(out of range) = %d", got)
	}
	if _, err := checkedBytes(math.MaxUint64, 2); err == nil {
		t.Fatal("checkedBytes accepted an overflow")
	}
}
