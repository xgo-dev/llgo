package sizereport

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"
)

func TestReadReadelfAndNativeOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "native")
	if err := os.WriteFile(path, []byte("native artifact"), 0o600); err != nil {
		t.Fatal(err)
	}
	report, err := ReadReadelf(path, strings.NewReader("Format: Mach-O arm64\n\n"+sampleReadelf), nil, "module")
	if err != nil {
		t.Fatal(err)
	}
	if report.FileSize != 15 || report.Format != "native" || report.Total.Flash() != 48 || report.Total.RAM() != 24 {
		t.Fatalf("native metadata/totals = %+v", report)
	}
	report.Artifacts = []Artifact{{Path: path, Format: "macho", Role: "deployment", Size: 15}}
	var text bytes.Buffer
	if err := Write(&text, report, "text"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text.String(), "flash     ram") || !strings.Contains(text.String(), "Final file: 15") {
		t.Fatalf("native text = %s", text.String())
	}
	var encoded bytes.Buffer
	if err := Write(&encoded, report, "json"); err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Total     struct{ Flash, RAM uint64 }
		Artifacts []Artifact
	}
	if err := json.Unmarshal(encoded.Bytes(), &payload); err != nil || payload.Total.Flash != 48 || payload.Total.RAM != 24 || len(payload.Artifacts) != 1 {
		t.Fatalf("native JSON = %s, error = %v", encoded.String(), err)
	}
	if _, err := ReadReadelf(path, strings.NewReader("no section information"), nil, "full"); err == nil {
		t.Fatal("missing native sections accepted")
	}
	if _, err := ReadReadelf(path, iotest.ErrReader(io.ErrUnexpectedEOF), nil, "full"); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("native reader failure = %v", err)
	}
	if _, err := ReadReadelf(path+".missing", strings.NewReader(sampleReadelf), nil, "full"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("native stat failure = %v", err)
	}
	if _, err := Collect(path, nil, "full"); !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("native tool fallback signal = %v", err)
	}
}

func TestReadelfSectionCategories(t *testing.T) {
	// Segment names are necessary for Mach-O sections whose own names do not
	// identify whether they contain code, constants or writable data.
	for _, tc := range []struct {
		name, segment string
		kind          sectionKind
	}{
		{".text", "", sectionText}, {".plt", "", sectionText},
		{".rodata", "", sectionROData}, {"__cstring", "", sectionROData},
		{".bss", "", sectionBSS}, {"__common", "", sectionBSS},
		{".got", "", sectionData}, {".init_array", "", sectionData},
		{"unusual", "__TEXT", sectionText}, {"unusual", "__DATA_CONST", sectionROData},
		{"unusual", "__DATA", sectionData}, {".debug_info", "", sectionUnknown},
	} {
		if got := classifySection(tc.name, tc.segment); got != tc.kind {
			t.Fatalf("section %q segment %q = %v, want %v", tc.name, tc.segment, got, tc.kind)
		}
	}
}

func TestReadelfRangesAndMissingSymbols(t *testing.T) {
	data := &readelfData{sections: map[int]*sectionInfo{
		1: {Name: "__text", Address: 0x1000, Size: 32, Kind: sectionText},
		2: {Name: "__const", Size: 8, Kind: sectionROData},
		3: {Name: ".debug_info", Size: 64, Kind: sectionUnknown},
		4: {Name: "empty", Kind: sectionData},
		5: nil,
	}, symbols: map[int][]symbolInfo{1: {
		{Name: "main.first", Address: 0xfff}, // clip to the section start
		{Name: "main.second", Address: 0x1010},
		{Name: "past.the.end", Address: 0x1030},
	}}}
	report := buildSizeReport("native", data, nil, "full")
	if report.Total.Code != 32 || report.Total.ROData != 8 || report.Modules["main.first"].Code != 16 || report.Modules["main.second"].Code != 16 {
		t.Fatalf("native section bounds = %+v", report)
	}
	if len(report.sortedModules()) != 3 {
		t.Fatal("empty/debug sections must not appear as owners")
	}
	data.symbols[1] = []symbolInfo{{Name: "main.tail", Address: 0x1008}}
	report = buildSizeReport("native", data, nil, "full")
	if report.Modules["(padding __text)"].Code != 8 || report.Modules["main.tail"].Code != 24 {
		t.Fatalf("leading padding = %+v", report)
	}
	data.symbols[1] = []symbolInfo{{Name: "invalid", Address: 0x1030}}
	report = buildSizeReport("native", data, nil, "full")
	if report.Modules["(padding __text)"].Code != 32 {
		t.Fatalf("unowned tail = %+v", report)
	}
	if buildSizeReport("native", nil, nil, "full") == nil {
		t.Fatal("missing reader data must return an empty report")
	}
}

func TestReadelfRejectsOversizedLines(t *testing.T) {
	if _, err := parseReadelfOutput(strings.NewReader(strings.Repeat("x", readelfMaxBuffer+1))); err == nil {
		t.Fatal("oversized readelf line accepted")
	}
}
