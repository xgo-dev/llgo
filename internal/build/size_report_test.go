package build

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

const sampleReadelf = `Sections [
  Section {
    Index: 0
    Name: __text (5F)
    Segment: __TEXT (5F)
    Address: 0x1000
    Size: 0x20
  }
  Section {
    Index: 1
    Name: __data
    Segment: __DATA
    Address: 0x2000
    Size: 0x10
  }
  Section {
    Index: 2
    Name: __common
    Segment: __DATA
    Address: 0x3000
    Size: 0x8
  }
]
Symbols [
  Symbol {
    Name: _main.main
    Section: __text (0x1)
    Value: 0x1000
  }
  Symbol {
    Name: _runtime.init
    Section: __text (0x1)
    Value: 0x1010
  }
  Symbol {
    Name: _main.dataVar
    Section: __data (0x2)
    Value: 0x2000
  }
  Symbol {
    Name: _runtime.dataVar
    Section: __data (0x2)
    Value: 0x2008
  }
  Symbol {
    Name: _runtime.bssVar
    Section: __common (0x3)
    Value: 0x3000
  }
]
`

func TestParseReadelfOutput(t *testing.T) {
	parsed, err := parseReadelfOutput(strings.NewReader(sampleReadelf))
	if err != nil {
		t.Fatalf("parseReadelfOutput: %v", err)
	}
	report := buildSizeReport("fake.bin", parsed, nil, "")
	if report == nil {
		t.Fatal("expected report")
	}
	modules := report.Modules
	if len(modules) == 0 {
		t.Fatal("expected modules in report")
	}
	mainMod, ok := modules["main"]
	if !ok {
		t.Fatalf("expected main module, got %v", modules)
	}
	if mainMod.Code != 0x10 {
		t.Fatalf("unexpected main code size: %d", mainMod.Code)
	}
	if mainMod.Data != 0x8 {
		t.Fatalf("unexpected main data size: %d", mainMod.Data)
	}
	runtimeMod := modules["runtime"]
	if runtimeMod.Code != 0x10 {
		t.Fatalf("unexpected runtime code size: %d", runtimeMod.Code)
	}
	if runtimeMod.Data != 0x8 {
		t.Fatalf("unexpected runtime data size: %d", runtimeMod.Data)
	}
	if runtimeMod.BSS != 0x8 {
		t.Fatalf("unexpected runtime bss size: %d", runtimeMod.BSS)
	}
	if report.Total.Flash() != 0x10+0x10+0x8+0x8 {
		t.Fatalf("unexpected flash total: %d", report.Total.Flash())
	}
	if report.Total.RAM() != 0x8+0x8+0x8 {
		t.Fatalf("unexpected ram total: %d", report.Total.RAM())
	}
}

func TestBuildSizeReportSkipsSameAddressDuplicates(t *testing.T) {
	data := &readelfData{
		sections: map[int]*sectionInfo{
			1: {
				Index:   1,
				Name:    "__text",
				Address: 0x1000,
				Size:    0x20,
				Kind:    sectionText,
			},
		},
		symbols: map[int][]symbolInfo{
			1: {
				{Name: "_module.first", SectionIndex: 1, Address: 0x1000},
				{Name: "_module.alias", SectionIndex: 1, Address: 0x1000},
				{Name: "_module.second", SectionIndex: 1, Address: 0x1010},
			},
		},
	}
	report := buildSizeReport("fake.bin", data, nil, "")
	if report == nil {
		t.Fatal("expected report")
	}
	mod := report.Modules["module"]
	if mod == nil {
		t.Fatalf("module bucket missing: %v", report.Modules)
	}
	if want := uint64(0x20); mod.Code != want {
		t.Fatalf("module code size = %d, want %d", mod.Code, want)
	}
	if want := uint64(0x20); report.Total.Code != want {
		t.Fatalf("total code size = %d, want %d", report.Total.Code, want)
	}
}

func TestParseReadelfRealBinary(t *testing.T) {
	path := os.Getenv("LLGO_SIZE_REPORT_BIN")
	if path == "" {
		return
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}
	cmd := exec.Command("llvm-readelf", "--all", absPath)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("llvm-readelf failed: %v", err)
	}
	parsed, err := parseReadelfOutput(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("parseReadelfOutput(real): %v", err)
	}
	if len(parsed.sections) == 0 {
		t.Fatal("expected sections in real binary")
	}
	report := buildSizeReport(absPath, parsed, nil, "")
	if len(report.Modules) == 0 {
		t.Fatalf("expected modules for %s", path)
	}
}

func TestNameResolver(t *testing.T) {
	pkgs := []Package{
		&aPackage{Package: &packages.Package{PkgPath: "github.com/foo/bar", Module: &packages.Module{Path: "github.com/foo"}}},
	}
	symbol := "_github.com/foo/bar.Type.method"
	if got := newNameResolver("package", pkgs).resolve(symbol); got != "github.com/foo/bar" {
		t.Fatalf("package level want github.com/foo/bar, got %q", got)
	}
	if got := newNameResolver("module", pkgs).resolve(symbol); got != "github.com/foo" {
		t.Fatalf("module level want github.com/foo, got %q", got)
	}
	full := newNameResolver("full", pkgs).resolve(symbol)
	if full != symbol {
		t.Fatalf("full level unexpected: %q", full)
	}
	if got := newNameResolver("package", nil).resolve("_llgo_helper.foo"); got != "llgo_helper" {
		t.Fatalf("unmatched symbol grouping failed: %q", got)
	}
	generic := "_slices.SortFunc[[]io/fs.DirEntry,io/fs.DirEntry]"
	if got := newNameResolver("module", nil).resolve(generic); got != "slices" {
		t.Fatalf("module level generic want slices, got %q", got)
	}
}

func TestModuleNameFromSymbolSpecialBrackets(t *testing.T) {
	cases := []struct {
		sym  string
		want string
	}{
		{"_slices.SortFunc[[]io/fs.DirEntry,io/fs.DirEntry]", "slices"},
		{"_slices.pdqsortCmpFunc[io/fs.DirEntry]", "slices"},
		{"_slices.choosePivotCmpFunc[io/fs.DirEntry]", "slices"},
		{"[]_llgo_float64", "llgo_float64"},
		{"[200]_llgo_int8", "llgo_int8"},
		{"*[]_llgo_Pointer", "llgo_Pointer"},
	}
	for _, tc := range cases {
		if got := moduleNameFromSymbol(tc.sym); got != tc.want {
			t.Fatalf("%q => %q, want %q", tc.sym, got, tc.want)
		}
	}
}

func TestBuildSizeReportIgnoresDollarDedup(t *testing.T) {
	data := &readelfData{
		sections: map[int]*sectionInfo{
			1: {Index: 1, Name: "__text", Address: 0x1000, Size: 0x20, Kind: sectionText},
		},
		symbols: map[int][]symbolInfo{
			1: {
				{Name: "$x", Address: 0x1000},
				{Name: "runtime.main", Address: 0x1000},
				{Name: "$d", Address: 0x1010},
				{Name: "main.main", Address: 0x1010},
				{Name: "other.sym", Address: 0x1018},
			},
		},
	}
	report := buildSizeReport("bin", data, nil, "module")
	if report == nil {
		t.Fatal("expected report")
	}
	if got := report.Modules["runtime"]; got == nil || got.Code != 0x10 {
		t.Fatalf("runtime should get 0x10 bytes, got %v", got)
	}
	if got := report.Modules["main"]; got == nil || got.Code != 0x8 {
		t.Fatalf("main should get 0x8 bytes, got %v", got)
	}
	if _, ok := report.Modules["$x"]; ok {
		t.Fatalf("$x should be ignored when other aliases exist")
	}
}
