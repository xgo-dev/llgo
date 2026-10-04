package sizereport

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"testing/iotest"
)

func sizeWasmUint(n uint64) []byte { return binary.AppendUvarint(nil, n) }
func sizeWasmName(s string) []byte { return append(sizeWasmUint(uint64(len(s))), s...) }
func sizeWasmSection(id byte, data []byte) []byte {
	return append(append([]byte{id}, sizeWasmUint(uint64(len(data)))...), data...)
}

func sizeWasmFixture(withNames bool) []byte {
	raw := []byte{0, 'a', 's', 'm', 1, 0, 0, 0}
	raw = append(raw, sizeWasmSection(1, []byte{1, 0x60, 0, 0})...)
	imports := append([]byte{2}, sizeWasmName("env")...)
	imports = append(imports, sizeWasmName("call")...)
	imports = append(imports, 0, 0) // function, type 0
	imports = append(imports, sizeWasmName("env")...)
	imports = append(imports, sizeWasmName("memory")...)
	imports = append(imports, 2, 3, 2, 4) // shared memory: min 2, max 4 pages
	raw = append(raw, sizeWasmSection(2, imports)...)
	raw = append(raw, sizeWasmSection(3, []byte{1, 0})...)
	raw = append(raw, sizeWasmSection(10, []byte{1, 2, 0, 0x0b})...)
	// Active and passive data, with exactly 3 + 2 stored bytes.
	raw = append(raw, sizeWasmSection(11, []byte{2, 0, 0x41, 0, 0x0b, 3, 'a', 'b', 'c', 1, 2, 'd', 'e'})...)
	if withNames {
		functions := append([]byte{1, 1}, sizeWasmName("main.(*T).Method")...)
		name := append(sizeWasmName("name"), sizeWasmSection(1, functions)...)
		raw = append(raw, sizeWasmSection(0, name)...)
	}
	return raw
}

func TestWasmSizeFinalEncoding(t *testing.T) {
	for _, names := range []bool{false, true} {
		raw := sizeWasmFixture(names)
		report, err := collectWasmSize("app.wasm", raw, nil, "full")
		if err != nil {
			t.Fatal(err)
		}
		if report.Total.Code != 2 || report.Total.Data != 5 || report.Total.BSS != 0 {
			t.Fatalf("payload: %+v", report.Total)
		}
		owner := "(unknown function 1)"
		if names {
			owner = "main.(*T).Method"
		}
		if report.Modules[owner] == nil || report.Modules[owner].Code != 2 {
			t.Fatalf("function attribution: %+v", report.Modules)
		}
		w := report.Wasm
		if got := report.Total.Code + report.Total.Data + w.CustomBytes + w.StructureBytes; got != uint64(len(raw)) {
			t.Fatalf("file accounting = %d, want %d", got, len(raw))
		}
		var sections uint64 = 8
		for _, s := range w.Sections {
			sections += s.Size
		}
		if sections != uint64(len(raw)) {
			t.Fatalf("section accounting = %d", sections)
		}
		if len(w.Memories) != 1 || !w.Memories[0].Imported || !w.Memories[0].Shared || w.Memories[0].InitialBytes != 2*65536 || *w.Memories[0].MaximumBytes != 4*65536 {
			t.Fatalf("memory: %+v", w.Memories)
		}
		var output bytes.Buffer
		if err := emitJSONReport(&output, report); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(output.String(), `"ram"`) || strings.Contains(output.String(), `"flash"`) {
			t.Fatalf("Wasm reservations reported as RAM/Flash: %s", output.String())
		}
	}
}

func TestWasmSizeMemory64AndImports(t *testing.T) {
	imports := []byte{3}
	for _, kind := range []byte{1, 3, 4} {
		imports = append(imports, sizeWasmName("env")...)
		imports = append(imports, sizeWasmName("item")...)
		imports = append(imports, kind)
		switch kind {
		case 1:
			imports = append(imports, 0x70, 1, 0, 1) // funcref table
		case 3:
			imports = append(imports, 0x63, 0x70, 0) // immutable nullable funcref global
		case 4:
			imports = append(imports, 0, 0) // exception tag
		}
	}
	raw := []byte{0, 'a', 's', 'm', 1, 0, 0, 0}
	raw = append(raw, sizeWasmSection(2, imports)...)
	raw = append(raw, sizeWasmSection(5, []byte{1, 5, 2, 4})...)
	// Explicit memory index, i64.const offset and zero stored data.
	raw = append(raw, sizeWasmSection(11, []byte{1, 2, 0, 0x42, 0, 0x0b, 0})...)
	report, err := collectWasmSize("64.wasm", raw, nil, "module")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Wasm.Memories) != 1 || !report.Wasm.Memories[0].Memory64 || report.Wasm.Memories[0].Imported {
		t.Fatalf("memory64: %+v", report.Wasm.Memories)
	}
}

func TestWasmSizeTable64LimitsAreElements(t *testing.T) {
	imports := append([]byte{1}, sizeWasmName("env")...)
	imports = append(imports, sizeWasmName("table")...)
	imports = append(imports, 1, 0x70, 5, 0)
	imports = append(imports, sizeWasmUint(1<<63)...)
	raw := append([]byte{0, 'a', 's', 'm', 1, 0, 0, 0}, sizeWasmSection(2, imports)...)
	if report, err := collectWasmSize("table.wasm", raw, nil, "module"); err != nil || len(report.Wasm.Memories) != 0 {
		t.Fatalf("table limits must not be converted to memory bytes: %v", err)
	}
}

func FuzzWasmSizeEncoding(f *testing.F) {
	f.Add(sizeWasmFixture(true))
	f.Add(sizeWasmFixture(false))
	f.Fuzz(func(t *testing.T, raw []byte) {
		report, err := collectWasmSize("fuzz.wasm", raw, nil, "full")
		if err != nil {
			return
		}
		if got := report.Total.Code + report.Total.Data + report.Wasm.CustomBytes + report.Wasm.StructureBytes; got != uint64(len(raw)) {
			t.Fatalf("size accounting: %d != %d", got, len(raw))
		}
	})
}

func TestWasmSizeReaderSignedBoundaries(t *testing.T) {
	for _, tc := range []struct {
		bits uint
		data []byte
		bad  bool
	}{
		{32, []byte{0x80, 0x80, 0x80, 0x80, 0x78}, false},
		{32, []byte{0xff, 0xff, 0xff, 0xff, 0x07}, false},
		{32, []byte{0x80, 0x80, 0x80, 0x80, 0x08}, true},
		{64, []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x7f}, false},
		{64, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0}, false},
		{64, []byte{0x80}, true},
	} {
		r := wasmSizeReader{data: tc.data}
		r.signed(tc.bits)
		if (r.done() != nil) != tc.bad {
			t.Fatalf("signed%d %x: %v", tc.bits, tc.data, r.done())
		}
	}
}

func TestWasmSizeMalformedAndOptionalNames(t *testing.T) {
	header := []byte{0, 'a', 's', 'm', 1, 0, 0, 0}
	for _, tail := range [][]byte{
		{10, 5, 1}, {10, 0x80}, {10, 0xff, 0xff, 0xff, 0xff, 0x10},
		sizeWasmSection(10, []byte{1, 3, 0}),
		sizeWasmSection(11, []byte{1, 3}),
		sizeWasmSection(11, []byte{1, 0, 0xff}),
		sizeWasmSection(5, []byte{1, 8}),
		sizeWasmSection(0, []byte{2, 'a'}),
		append(sizeWasmSection(5, []byte{0}), sizeWasmSection(5, []byte{0})...),
	} {
		if _, err := collectWasmSize("bad.wasm", append(bytes.Clone(header), tail...), nil, "module"); err == nil {
			t.Fatalf("accepted malformed tail %x", tail)
		}
	}
	// Optional debug names cannot make an otherwise measurable module fail.
	badNames := append(sizeWasmName("name"), []byte{1, 3, 1}...)
	raw := append(sizeWasmFixture(false), sizeWasmSection(0, badNames)...)
	report, err := collectWasmSize("names.wasm", raw, nil, "module")
	if err != nil || len(report.Warnings) != 1 || report.Total.Code != 2 {
		t.Fatalf("optional names = %+v, %v", report, err)
	}
}

func TestWasmSizeRejectsInvalidModuleFields(t *testing.T) {
	header := []byte{0, 'a', 's', 'm', 1, 0, 0, 0}
	importField := func(descriptor ...byte) []byte {
		payload := append([]byte{1}, sizeWasmName("env")...)
		payload = append(payload, sizeWasmName("item")...)
		return sizeWasmSection(2, append(payload, descriptor...))
	}
	for _, tc := range []struct {
		name    string
		raw     []byte
		wantErr string
	}{
		{"short header", header[:7], "invalid WebAssembly header"},
		{"unknown version", []byte{0, 'a', 's', 'm', 2, 0, 0, 0}, "invalid WebAssembly header"},
		{"non UTF-8 custom name", append(bytes.Clone(header), sizeWasmSection(0, []byte{1, 0xff})...), "invalid UTF-8"},
		{"unknown import kind", append(bytes.Clone(header), importField(5)...), "unsupported WebAssembly import kind"},
		{"invalid table element type", append(bytes.Clone(header), importField(1, 0, 0, 0)...), "unsupported WebAssembly value type"},
		{"truncated global import", append(bytes.Clone(header), importField(3, 0x7f)...), "truncated WebAssembly encoding"},
		{"trailing memory bytes", append(bytes.Clone(header), sizeWasmSection(5, []byte{0, 0})...), "unexpected trailing WebAssembly bytes"},
		{"memory byte overflow", append(bytes.Clone(header), sizeWasmSection(5, append([]byte{1, 4}, sizeWasmUint(1<<48)...))...), "memory byte size overflow"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			report, err := collectWasmSize("invalid.wasm", tc.raw, nil, "full")
			if report != nil || err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("invalid module report = %+v, error = %v; want %q", report, err, tc.wantErr)
			}
		})
	}
}

func TestWasmSizeExtendedOffsetsAndOptionalNameSubsections(t *testing.T) {
	raw := []byte{0, 'a', 's', 'm', 1, 0, 0, 0}
	raw = append(raw, sizeWasmSection(1, []byte{1, 0x60, 0, 0})...) // () -> ()
	imports := append([]byte{1}, sizeWasmName("env")...)
	imports = append(imports, sizeWasmName("base")...)
	imports = append(imports, 3, 0x7f, 0) // immutable i32 global
	raw = append(raw, sizeWasmSection(2, imports)...)
	raw = append(raw, sizeWasmSection(3, []byte{1, 0})...)
	raw = append(raw, sizeWasmSection(5, []byte{1, 0, 1})...)
	raw = append(raw, sizeWasmSection(10, []byte{1, 2, 0, 0x0b})...)
	// Active data at global.get(0) + 4. Its offset expression is encoding
	// overhead; only the three stored bytes belong to the data payload.
	raw = append(raw, sizeWasmSection(11, []byte{1, 0, 0x23, 0, 0x41, 4, 0x6a, 0x0b, 3, 'a', 'b', 'c'})...)
	names := append(sizeWasmName("name"), sizeWasmSection(0, sizeWasmName("example"))...)
	functions := append([]byte{1, 0}, sizeWasmName("main.work")...)
	names = append(names, sizeWasmSection(1, functions)...)
	raw = append(raw, sizeWasmSection(0, names)...)
	report, err := collectWasmSize("offset.wasm", raw, nil, "full")
	if err != nil {
		t.Fatal(err)
	}
	if owner := report.Modules["main.work"]; owner == nil || owner.Code != 2 || report.Total.Data != 3 || len(report.Warnings) != 0 {
		t.Fatalf("offset/name attribution = %+v, warnings = %v", report.Modules, report.Warnings)
	}
	if sum := report.Total.Code + report.Total.Data + report.Wasm.CustomBytes + report.Wasm.StructureBytes; sum != uint64(len(raw)) {
		t.Fatalf("encoded byte total = %d, want %d", sum, len(raw))
	}
}

func TestWasmSizeDuplicateFunctionNamesRemainOptional(t *testing.T) {
	functions := []byte{2}
	for _, name := range []string{"first", "second"} {
		functions = append(functions, 1) // both entries claim the same function
		functions = append(functions, sizeWasmName(name)...)
	}
	names := append(sizeWasmName("name"), sizeWasmSection(1, functions)...)
	raw := append(sizeWasmFixture(false), sizeWasmSection(0, names)...)
	report, err := collectWasmSize("duplicate-names.wasm", raw, nil, "full")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Warnings) != 1 || !strings.Contains(report.Warnings[0], "duplicate WebAssembly function name index") {
		t.Fatalf("duplicate names warning = %v", report.Warnings)
	}
	if owner := report.Modules["(unknown function 1)"]; owner == nil || owner.Code != 2 {
		t.Fatalf("unreliable names replaced physical accounting: %+v", report.Modules)
	}
}

type sizeFailWriter struct{}

func (sizeFailWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestSizeReportErrorsAndConfiguration(t *testing.T) {
	for _, format := range []string{"text", "json"} {
		if err := Write(sizeFailWriter{}, &Report{}, format); err == nil {
			t.Fatalf("%s write error ignored", format)
		}
	}
}

func TestFinalSizeTextAndReadErrors(t *testing.T) {
	report, err := collectWasmSize("app.wasm", sizeWasmFixture(true), nil, "full")
	if err != nil {
		t.Fatal(err)
	}
	report.Format = "wasm"
	report.Warnings = []string{"optional names unavailable"}
	report.Artifacts = []Artifact{{Path: "app.wasm", Format: "wasm", Role: "deployment", Size: 99}}
	var output bytes.Buffer
	if err := Write(&output, report, "text"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"encoded function bodies", "main.(*T).Method", "Memory 0: initial=131072 maximum=262144", "shared=true", "Artifact: 99", "optional names unavailable"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("text missing %q: %s", want, output.String())
		}
	}
	if err := Write(io.Discard, report, "bad"); err == nil {
		t.Fatal("unknown report format accepted")
	}
	short := filepath.Join(t.TempDir(), "short")
	if err := os.WriteFile(short, []byte{0}, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Collect(short, nil, "full"); err == nil {
		t.Fatal("truncated header accepted")
	}
	if _, err := Collect(t.TempDir(), nil, "full"); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("directory size report = %v", err)
	}
}

func TestWasmSizeReadBounds(t *testing.T) {
	raw := sizeWasmFixture(false)
	for _, tc := range []struct {
		name string
		size int64
		bad  bool
	}{
		{"exact", int64(len(raw)), false},
		{"shrunken", int64(len(raw)) + 1, true},
		{"grown", int64(len(raw)) - 1, true},
		{"negative", -1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := readWasmSizeBytes(bytes.NewReader(raw), tc.size)
			if (err != nil) != tc.bad {
				t.Fatalf("read %d bytes: %v", tc.size, err)
			}
			if !tc.bad && !bytes.Equal(got, raw) {
				t.Fatal("artifact bytes changed")
			}
		})
	}
	if strconv.IntSize == 32 {
		if _, err := readWasmSizeBytes(bytes.NewReader(raw), 1<<32); err == nil {
			t.Fatal("file size outside host int range accepted")
		}
	}
	reader := io.MultiReader(bytes.NewReader(raw), iotest.ErrReader(io.ErrClosedPipe))
	if _, err := readWasmSizeBytes(reader, int64(len(raw))); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("read failure after expected file bytes = %v", err)
	}
}

func TestFinalSizeFormatDispatch(t *testing.T) {
	dir := t.TempDir()
	if report, err := Collect(filepath.Join(dir, "missing"), nil, "full"); report != nil || !os.IsNotExist(err) {
		t.Fatalf("missing input report = %+v, error = %v", report, err)
	}
	elfPath := filepath.Join(dir, "app.elf")
	raw := elfSizeFixture(t, true)
	if err := os.WriteFile(elfPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	report, err := Collect(elfPath, nil, "full")
	if err != nil {
		t.Fatal(err)
	}
	if report.Format != "elf" || report.FileSize != uint64(len(raw)) || report.Total.Code != 32 {
		t.Fatalf("ELF dispatch report = %+v", report)
	}
	wasmPath := filepath.Join(dir, "app.wasm")
	raw = sizeWasmFixture(true)
	if err := os.WriteFile(wasmPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	report, err = Collect(wasmPath, nil, "full")
	if err != nil || report.Format != "wasm" || report.FileSize != uint64(len(raw)) || report.Modules["main.(*T).Method"].Code != 2 {
		t.Fatalf("Wasm dispatch report = %+v, error = %v", report, err)
	}
	raw[4] = 2
	if err := os.WriteFile(wasmPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Collect(wasmPath, nil, "full"); err == nil {
		t.Fatal("unsupported Wasm version accepted")
	}
}

func TestSizeReportPreservesMethodNames(t *testing.T) {
	for raw, want := range map[string]string{"pkg.(*T).Method (123)": "pkg.(*T).Method", "pkg.(T).Method": "pkg.(T).Method", "__text (5F)": "__text", "name (not an index)": "name (not an index)"} {
		if got := parseNameField(raw); got != want {
			t.Fatalf("%q -> %q, want %q", raw, got, want)
		}
	}
}
