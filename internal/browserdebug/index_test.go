//go:build !llgo

package browserdebug

import (
	"bytes"
	"debug/dwarf"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/debugabi"
	"github.com/xgo-dev/llgo/internal/wasmdebug"
)

func TestLoadEmbeddedAndExternal(t *testing.T) {
	dir := t.TempDir()
	sourceDir := filepath.Join(dir, "local-source")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(sourceDir, "fixture.c")
	if err := os.WriteFile(source, []byte("int add(int a, int b) { int result = a + b; return result; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	embedded := filepath.Join(dir, "fixture.wasm")
	compileWasmFixture(t, source, embedded)
	raw, err := os.ReadFile(embedded)
	if err != nil {
		t.Fatal(err)
	}
	raw, err = wasmdebug.SetDebuggerRecord(raw, debugabi.NewRecord(4, debugabi.ByteOrderLittle))
	if err != nil {
		t.Fatal(err)
	}
	raw, _, err = wasmdebug.EnsureBuildID(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(embedded, raw, 0o755); err != nil {
		t.Fatal(err)
	}

	bundle, err := Load(embedded, nil)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Index.Artifact != "embedded" || bundle.Index.Record.SchemaVersion != 1 || bundle.Index.BuildID == "" {
		t.Fatalf("embedded index header = %+v", bundle.Index)
	}
	if !hasSourceSuffix(bundle.Index.Sources, "fixture.c") || !hasFunction(bundle.Index.Functions, "add") || len(bundle.Index.Lines) == 0 {
		t.Fatalf("embedded index lacks source/function/lines: sources=%+v functions=%+v lines=%d diagnostics=%v",
			bundle.Index.Sources, bundle.Index.Functions, len(bundle.Index.Lines), bundle.Index.Diagnostics)
	}

	sidecar := filepath.Join(dir, "fixture debug.wasm")
	if err := os.WriteFile(sidecar, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	main, err := wasmdebug.Externalize(raw, "fixture%20debug.wasm")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(embedded, main, 0o755); err != nil {
		t.Fatal(err)
	}
	bundle, err = Load(embedded, nil)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Index.Artifact != "external" || bundle.SymbolsPath != sidecar {
		t.Fatalf("external bundle = mode %q symbols %q", bundle.Index.Artifact, bundle.SymbolsPath)
	}

	if err := os.Remove(sidecar); err != nil {
		t.Fatal(err)
	}
	_, err = Load(embedded, nil)
	var missing *MissingSymbolsError
	if !errors.As(err, &missing) || missing.URL != "fixture%20debug.wasm" {
		t.Fatalf("missing sidecar error = %T %v", err, err)
	}
	if !errors.Is(err, os.ErrNotExist) || missing.Path != sidecar {
		t.Fatalf("missing sidecar lost its path or filesystem cause: %v", err)
	}
	if message := err.Error(); !strings.Contains(message, strconv.Quote(sidecar)) || !strings.Contains(message, strconv.Quote(missing.URL)) {
		t.Fatalf("missing sidecar diagnostic omitted its quoted URL or path: %s", message)
	}

	stale, err := wasmdebug.SetBuildID(raw, bytes.Repeat([]byte{0xaa}, 32))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sidecar, stale, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(embedded, nil); err == nil || !strings.Contains(err.Error(), "stale external") {
		t.Fatalf("stale sidecar error = %v", err)
	}
	wrongRecord, err := wasmdebug.SetDebuggerRecord(raw, debugabi.NewRecord(8, debugabi.ByteOrderLittle))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sidecar, wrongRecord, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(embedded, nil); err == nil || !strings.Contains(err.Error(), "record does not match") {
		t.Fatalf("mismatched sidecar ABI error = %v", err)
	}
}

// These helpers construct the WebAssembly custom-section envelope; the DWARF
// used below comes from Clang so diagnostics are checked against real units.
func appendBrowserCustomSection(module []byte, name string, contents []byte) []byte {
	payload := binary.AppendUvarint(nil, uint64(len(name)))
	payload = append(payload, name...)
	payload = append(payload, contents...)
	result := append(bytes.Clone(module), 0)
	result = binary.AppendUvarint(result, uint64(len(payload)))
	return append(result, payload...)
}

func browserTestIdentity(t *testing.T, module []byte) []byte {
	t.Helper()
	module, err := wasmdebug.SetDebuggerRecord(module, debugabi.NewRecord(4, debugabi.ByteOrderLittle))
	if err != nil {
		t.Fatal(err)
	}
	module, err = wasmdebug.SetBuildID(module, []byte("browser-test-build"))
	if err != nil {
		t.Fatal(err)
	}
	return module
}

func TestLoadRejectsInvalidArtifactAndSidecarContracts(t *testing.T) {
	dir := t.TempDir()
	header := []byte{0, 'a', 's', 'm', 1, 0, 0, 0}
	// Container checks precede DWARF decoding; arbitrary DWARF bytes are enough
	// to establish that these invalid artifacts never reach the decoder.
	withDWARF := appendBrowserCustomSection(header, ".debug_info", []byte{1})
	validIdentity := browserTestIdentity(t, withDWARF)
	noBuildID, err := wasmdebug.SetDebuggerRecord(withDWARF, debugabi.NewRecord(4, debugabi.ByteOrderLittle))
	if err != nil {
		t.Fatal(err)
	}
	emptyBuildID := appendBrowserCustomSection(noBuildID, "build_id", []byte{0})
	brokenBuildID := appendBrowserCustomSection(noBuildID, "build_id", []byte{0x80})
	url := "symbols.wasm"
	urlContents := append(binary.AppendUvarint(nil, uint64(len(url))), url...)
	external, err := wasmdebug.Externalize(validIdentity, url)
	if err != nil {
		t.Fatal(err)
	}
	sidecarNoRecord, err := wasmdebug.SetBuildID(withDWARF, []byte("browser-test-build"))
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, want    string
		main, sidecar []byte
	}{
		{name: "unreadable artifact", want: "read browser WebAssembly artifact"},
		{name: "invalid container", main: []byte("not wasm"), want: "invalid WebAssembly header"},
		{name: "missing ABI record", main: withDWARF, want: "no LLGo debugger ABI record"},
		{name: "missing build ID", main: noBuildID, want: "no build_id"},
		{name: "empty build ID", main: emptyBuildID, want: "no build_id"},
		{name: "corrupt build ID", main: brokenBuildID, want: "invalid WebAssembly build_id"},
		{name: "missing DWARF", main: browserTestIdentity(t, header), want: "contains no DWARF"},
		{name: "ambiguous DWARF", main: appendBrowserCustomSection(validIdentity, "external_debug_info", urlContents), want: "both embedded and external DWARF"},
		{name: "corrupt external URL", main: appendBrowserCustomSection(browserTestIdentity(t, header), "external_debug_info", []byte{5, 'a'}), want: "invalid external_debug_info section"},
		{name: "duplicate DWARF", main: appendBrowserCustomSection(validIdentity, ".debug_info", []byte{2}), want: "multiple .debug_info"},
		{name: "corrupt sidecar", main: external, sidecar: []byte("not wasm"), want: "read external WebAssembly DWARF build ID"},
		{name: "sidecar without build ID", main: external, sidecar: noBuildID, want: "external WebAssembly DWARF has no build_id"},
		{name: "sidecar without ABI", main: external, sidecar: sidecarNoRecord, want: "record does not match"},
		{name: "sidecar with corrupt ABI", main: external, sidecar: appendBrowserCustomSection(sidecarNoRecord, debugabi.WasmSectionName, []byte{1}), want: "read external WebAssembly debugger record"},
		{name: "sidecar without DWARF", main: external, sidecar: browserTestIdentity(t, header), want: "sidecar contains no DWARF"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caseDir := filepath.Join(dir, strings.ReplaceAll(tt.name, " ", "-"))
			if err := os.Mkdir(caseDir, 0o755); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(caseDir, "main.wasm")
			if tt.main != nil {
				if err := os.WriteFile(path, tt.main, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if tt.sidecar != nil {
				if err := os.WriteFile(filepath.Join(caseDir, url), tt.sidecar, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			bundle, err := Load(path, nil)
			if err == nil || !strings.Contains(err.Error(), tt.want) || bundle != nil {
				t.Fatalf("Load returned bundle=%v, error=%v; want %q", bundle != nil, err, tt.want)
			}
		})
	}
}

func TestLoadRejectsNonLocalExternalURLs(t *testing.T) {
	header := []byte{0, 'a', 's', 'm', 1, 0, 0, 0}
	module := browserTestIdentity(t, appendBrowserCustomSection(header, ".debug_info", []byte{1}))
	for _, reference := range []string{"https://example.invalid/symbols.wasm", "//example.invalid/symbols.wasm", "symbols.wasm?version=1", "symbols.wasm#section", "symbols%ZZ.wasm", "", "../secret.wasm", "%2e%2e/secret.wasm", "/secret.wasm", "%2fsecret.wasm", "..%5csecret.wasm", "C:%5csecret.wasm"} {
		t.Run(reference, func(t *testing.T) {
			// Externalize rejects empty references; write that malformed metadata
			// directly to exercise the reader of artifacts from other producers.
			var main []byte
			var err error
			if reference != "" {
				main, err = wasmdebug.Externalize(module, reference)
			} else {
				main = appendBrowserCustomSection(browserTestIdentity(t, header), "external_debug_info", []byte{0})
			}
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "main.wasm")
			if err := os.WriteFile(path, main, 0o644); err != nil {
				t.Fatal(err)
			}
			_, err = Load(path, nil)
			var missing *MissingSymbolsError
			if err == nil || errors.As(err, &missing) || !strings.Contains(err.Error(), "external WebAssembly DWARF URL") {
				t.Fatalf("nonlocal/invalid URL %q reached symbol-file access: %v", reference, err)
			}
		})
	}
}

func TestLoadReportsCorruptDWARF(t *testing.T) {
	dir := t.TempDir()
	source, output := filepath.Join(dir, "fixture.c"), filepath.Join(dir, "fixture.wasm")
	if err := os.WriteFile(source, []byte("int add(int a, int b) { return a + b; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	compileWasmFixture(t, source, output)
	raw, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	sections, err := wasmdebug.DWARFSections(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		section string
		data    []byte
		want    string
		fatal   bool
	}{
		{section: ".debug_info", data: []byte{1}, want: "read WebAssembly DWARF:", fatal: true},
		{section: ".debug_abbrev", data: []byte{0}, want: "read WebAssembly DWARF entry:", fatal: true},
		{section: ".debug_line", data: []byte{1}, want: "read line table at"},
	} {
		t.Run(tt.section, func(t *testing.T) {
			module := browserTestIdentity(t, []byte{0, 'a', 's', 'm', 1, 0, 0, 0})
			for name, content := range sections {
				if name == tt.section {
					content = tt.data
				}
				module = appendBrowserCustomSection(module, name, content)
			}
			path := filepath.Join(t.TempDir(), "corrupt.wasm")
			if err := os.WriteFile(path, module, 0o644); err != nil {
				t.Fatal(err)
			}
			bundle, err := Load(path, nil)
			if tt.fatal {
				if err == nil || !strings.Contains(err.Error(), tt.want) || bundle != nil {
					t.Fatalf("corrupt DWARF returned bundle=%v, error=%v; want %q", bundle != nil, err, tt.want)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(strings.Join(bundle.Index.Diagnostics, "\n"), tt.want) || !hasFunction(bundle.Index.Functions, "add") {
				t.Fatalf("corrupt line table lost its diagnostic or valid function: %+v", bundle.Index)
			}
		})
	}
}

func TestInvalidLocationListsProduceDiagnostics(t *testing.T) {
	// One wasm32 range [1,2), followed by a two-byte expression length.
	rangeOnly := []byte{1, 0, 0, 0, 2, 0, 0, 0}
	for _, tt := range []struct {
		name, want string
		data       []byte
		offset     int64
		width      int
	}{
		{name: "unsupported address width", width: 2, want: "unsupported DWARF address size"},
		{name: "negative offset", width: 4, offset: -1, want: "outside .debug_loc"},
		{name: "out of bounds offset", width: 4, offset: 1, want: "outside .debug_loc"},
		{name: "truncated address", width: 4, data: rangeOnly[:3], want: "unexpected EOF"},
		{name: "truncated expression length", width: 4, data: append(bytes.Clone(rangeOnly), 2), want: "unexpected EOF"},
		{name: "truncated expression", width: 4, data: append(bytes.Clone(rangeOnly), 2, 0, 0x91), want: "unexpected EOF"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			builder := indexBuilder{addressSize: tt.width, sections: map[string][]byte{".debug_loc": tt.data}}
			if got := builder.locations(tt.offset, "localValue"); len(got) != 0 {
				t.Fatalf("invalid location produced storage: %+v", got)
			}
			if len(builder.index.Diagnostics) != 1 || !strings.Contains(builder.index.Diagnostics[0], "localValue") || !strings.Contains(builder.index.Diagnostics[0], tt.want) {
				t.Fatalf("diagnostics = %v, want variable name and %q", builder.index.Diagnostics, tt.want)
			}
		})
	}
}

func TestInvalidRangeListProducesDiagnostic(t *testing.T) {
	// A DWARF4 unit whose range attribute points before .debug_ranges must
	// not invent breakpoint addresses or make the entire index unusable.
	info := []byte{8, 0, 0, 0, 4, 0, 0, 0, 0, 0, 4, 0}
	data, err := dwarf.New([]byte{0}, nil, nil, info, nil, nil, []byte{0}, nil)
	if err != nil {
		t.Fatal(err)
	}
	builder := indexBuilder{data: data}
	entry := &dwarf.Entry{
		Offset: 11, Tag: dwarf.TagCompileUnit,
		Field: []dwarf.Field{{Attr: dwarf.AttrRanges, Val: int64(-1)}},
	}
	if got := builder.ranges(entry); len(got) != 0 {
		t.Fatalf("corrupt range list produced breakpoint addresses: %v", got)
	}
	if len(builder.index.Diagnostics) != 1 || !strings.Contains(builder.index.Diagnostics[0], "invalid range offset -1") {
		t.Fatalf("corrupt range list lost its diagnostic: %v", builder.index.Diagnostics)
	}
}

func TestConstantValuesPreserveKindAndBytes(t *testing.T) {
	for _, tt := range []struct {
		input       any
		kind, value string
	}{
		{input: int64(-1), kind: "signed", value: "-1"},
		{input: ^uint64(0), kind: "unsigned", value: "18446744073709551615"},
		{input: "a\x00b", kind: "string", value: "a\x00b"},
		{input: []byte{0, 0x80, 0xff}, kind: "bytes", value: "0080ff"},
	} {
		t.Run(fmt.Sprintf("%T", tt.input), func(t *testing.T) {
			got := constantValue(tt.input)
			if got == nil || got.Kind != tt.kind || got.Value != tt.value {
				t.Fatalf("constant = %+v, want %s %q", got, tt.kind, tt.value)
			}
		})
	}
}

func TestUnreadableTypePreservesVariableStorage(t *testing.T) {
	// A valid empty DWARF4 unit with 32-bit addresses and a null DIE. The
	// variable below refers past that unit rather than a valid type DIE.
	info := []byte{8, 0, 0, 0, 4, 0, 0, 0, 0, 0, 4, 0}
	data, err := dwarf.New([]byte{0}, nil, nil, info, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	builder := indexBuilder{data: data}
	builder.addVariable(&dwarf.Entry{
		Tag: dwarf.TagVariable,
		Field: []dwarf.Field{
			{Attr: dwarf.AttrName, Val: "brokenType"},
			{Attr: dwarf.AttrType, Val: dwarf.Offset(0x100)},
			{Attr: dwarf.AttrLocation, Val: []byte{0x91, 0}},
		},
	}, scopeState{function: true, ranges: []AddressRange{{Start: 10, End: 20}}}, 1)
	if len(builder.index.Variables) != 1 || builder.index.Variables[0].Type != "" || len(builder.index.Variables[0].Locations) != 1 {
		t.Fatalf("unreadable type discarded readable storage or invented a type: %+v", builder.index.Variables)
	}
	if len(builder.index.Diagnostics) != 1 || !strings.Contains(builder.index.Diagnostics[0], "DWARF type at 0x100") {
		t.Fatalf("missing type failure diagnostic: %v", builder.index.Diagnostics)
	}
}

func TestUnlocatedFunctionsDoNotLeakLocals(t *testing.T) {
	entry := &dwarf.Entry{Tag: dwarf.TagVariable, Field: []dwarf.Field{
		{Attr: dwarf.AttrName, Val: "value"},
		{Attr: dwarf.AttrLocation, Val: []byte{0xed, 0, 0, 0x9f}},
	}}
	builder := indexBuilder{}
	builder.addVariable(entry, scopeState{function: true}, 1)
	entry.Tag = dwarf.TagFormalParameter
	builder.addVariable(entry, scopeState{}, 1)
	if len(builder.index.Variables) != 0 {
		t.Fatalf("unlocated declarations became visible in unrelated frames: %+v", builder.index.Variables)
	}
	builder.addVariable(entry, scopeState{function: true, ranges: []AddressRange{{Start: 10, End: 20}}}, 1)
	entry.Tag = dwarf.TagVariable
	builder.addVariable(entry, scopeState{}, 0)
	if len(builder.index.Variables) != 2 || builder.index.Variables[0].Scope != "PARAMETER" || builder.index.Variables[1].Scope != "GLOBAL" {
		t.Fatalf("lost live parameter or global: %+v", builder.index.Variables)
	}
}

func TestDebugLocAddressWidth(t *testing.T) {
	for _, width := range []int{4, 8} {
		t.Run(string(rune('0'+width)), func(t *testing.T) {
			var raw []byte
			appendAddress := func(value uint64) {
				var buf [8]byte
				binary.LittleEndian.PutUint64(buf[:], value)
				raw = append(raw, buf[:width]...)
			}
			base := uint64(0x1000)
			maximum := uint64(0xffffffff)
			if width == 8 {
				base += 1 << 32
				maximum = ^uint64(0)
			}
			appendAddress(maximum)
			appendAddress(base)
			appendAddress(2)
			appendAddress(9)
			raw = append(raw, 2, 0, 0x91, 0x78) // DW_OP_fbreg -8
			appendAddress(0)
			appendAddress(0)
			locations, err := parseDebugLoc(raw, 0, width)
			if err != nil || len(locations) != 1 {
				t.Fatalf("location list = %+v, %v", locations, err)
			}
			if got := locations[0]; got.Start != base+2 || got.End != base+9 || got.Expression != "9178" {
				t.Fatalf("location = %+v", got)
			}
			if _, err := parseDebugLoc(raw[:len(raw)-1], 0, width); err == nil {
				t.Fatal("truncated terminator accepted")
			}
		})
	}
}

func TestPathMapping(t *testing.T) {
	mapping, err := ParsePathMapping("/build/source=/local/source")
	if err != nil {
		t.Fatal(err)
	}
	if suffix, ok := pathPrefix(filepath.Join("/build/source", "pkg/main.go"), mapping.From); !ok || suffix != filepath.Join("pkg", "main.go") {
		t.Fatalf("pathPrefix = %q, %v", suffix, ok)
	}
	if suffix, ok := pathPrefix(mapping.From, mapping.From); !ok || suffix != "" {
		t.Fatalf("exact source root = %q, %v", suffix, ok)
	}
	if _, ok := pathPrefix(filepath.Join("/build/source-other", "main.go"), mapping.From); ok {
		t.Fatal("source mapping matched a sibling directory")
	}
	if _, err := ParsePathMapping("missing-separator"); err == nil {
		t.Fatal("invalid source mapping was accepted")
	}
}

func TestLoadUsesLongestSourcePathMapping(t *testing.T) {
	dir := t.TempDir()
	recordedRoot := filepath.Join(dir, "recorded")
	recordedPackage := filepath.Join(recordedRoot, "pkg")
	if err := os.MkdirAll(recordedPackage, 0o755); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(recordedPackage, "fixture.c")
	artifact := filepath.Join(dir, "fixture.wasm")
	if err := os.WriteFile(source, []byte("int add(int a, int b) { return a + b; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	compileWasmFixture(t, source, artifact)
	raw, err := os.ReadFile(artifact)
	if err != nil {
		t.Fatal(err)
	}
	raw, err = wasmdebug.SetDebuggerRecord(raw, debugabi.NewRecord(4, debugabi.ByteOrderLittle))
	if err != nil {
		t.Fatal(err)
	}
	raw, _, err = wasmdebug.EnsureBuildID(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifact, raw, 0o755); err != nil {
		t.Fatal(err)
	}

	relocatedRoot := filepath.Join(dir, "relocated")
	if err := os.Rename(recordedRoot, relocatedRoot); err != nil {
		t.Fatal(err)
	}
	bundle, err := Load(artifact, []PathMapping{
		{From: recordedRoot, To: filepath.Join(dir, "wrong")},
		{From: recordedPackage, To: filepath.Join(relocatedRoot, "pkg")},
	})
	if err != nil {
		t.Fatal(err)
	}
	wantSource, err := filepath.EvalSymlinks(filepath.Join(relocatedRoot, "pkg", "fixture.c"))
	if err != nil {
		t.Fatal(err)
	}
	for _, indexed := range bundle.Index.Sources {
		if indexed.Path != source {
			continue
		}
		if !indexed.Local || bundle.SourceFiles[indexed.ID] != wantSource {
			t.Fatalf("mapped source = %+v, file %q, want %q", indexed, bundle.SourceFiles[indexed.ID], wantSource)
		}
		return
	}
	t.Fatalf("recorded source %q is absent from %+v", source, bundle.Index.Sources)
}

func TestLoadLLGoArtifact(t *testing.T) {
	path := os.Getenv("LLGO_BROWSER_DEBUG_ARTIFACT")
	if path == "" {
		t.Skip("LLGO_BROWSER_DEBUG_ARTIFACT is unset")
	}
	bundle, err := Load(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("sources=%d lines=%d functions=%d variables=%d types=%d diagnostics=%d",
		len(bundle.Index.Sources), len(bundle.Index.Lines), len(bundle.Index.Functions),
		len(bundle.Index.Variables), len(bundle.Index.Types), len(bundle.Index.Diagnostics))
	if !hasFunction(bundle.Index.Functions, "main.main") {
		t.Fatalf("LLGo index does not contain main.main")
	}
}

func TestLoadLLGoRuntimeFixture(t *testing.T) {
	path := os.Getenv("LLGO_BROWSER_DEBUG_RUNTIME_ARTIFACT")
	if path == "" {
		t.Skip("LLGO_BROWSER_DEBUG_RUNTIME_ARTIFACT is unset")
	}
	bundle, err := Load(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"text", "values", "mapping", "queue", "greeter", "closure"} {
		if !hasVariable(bundle.Index.Variables, name) {
			t.Errorf("LLGo browser runtime fixture does not contain variable %q", name)
		}
	}
	for _, pattern := range []string{"string", "[]", "map[", "chan ", "interface{"} {
		if !hasTypePattern(bundle.Index.Types, pattern) {
			t.Errorf("LLGo browser runtime fixture does not contain a type matching %q", pattern)
		}
	}
}

func compileWasmFixture(t *testing.T, source, output string) {
	t.Helper()
	clang, err := exec.LookPath("clang")
	if err != nil {
		t.Skip("clang is unavailable")
	}
	command := exec.Command(clang,
		"--target=wasm32-unknown-unknown", "-O0", "-g", "-nostdlib",
		"-Wl,--no-entry", "-Wl,--export=add", "-o", output, source,
	)
	if data, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile WebAssembly fixture on %s/%s: %v\n%s", runtime.GOOS, runtime.GOARCH, err, data)
	}
}

func hasSourceSuffix(sources []Source, suffix string) bool {
	for _, source := range sources {
		if strings.HasSuffix(source.Path, suffix) {
			return true
		}
	}
	return false
}

func hasFunction(functions []Function, name string) bool {
	for _, function := range functions {
		if function.Name == name {
			return true
		}
	}
	return false
}

func hasVariable(variables []Variable, name string) bool {
	for _, variable := range variables {
		if variable.Name == name {
			return true
		}
	}
	return false
}

func hasTypePattern(types []Type, pattern string) bool {
	for _, item := range types {
		if strings.Contains(item.Name, pattern) {
			return true
		}
	}
	return false
}

func TestSidecarAndSourceSymlinkContainment(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	secret := filepath.Join(outside, "secret.wasm")
	if err := os.WriteFile(secret, []byte("private file"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "linked.wasm")
	if err := os.Symlink(secret, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := localExternalPath(filepath.Join(root, "main.wasm"), "linked.wasm"); err == nil {
		t.Fatal("sidecar symlink escaped artifact directory")
	}
	if err := os.Symlink("cycle.wasm", filepath.Join(root, "cycle.wasm")); err != nil {
		t.Fatal(err)
	}
	if _, err := localExternalPath(filepath.Join(root, "main.wasm"), "cycle.wasm"); err == nil || os.IsNotExist(err) {
		t.Fatalf("cyclic sidecar was accepted or reported as a missing file: %v", err)
	}
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	builder := indexBuilder{sourceRoots: []string{canonicalRoot}, sourceByPath: make(map[string]string), sourceFiles: make(map[string]string)}
	id := builder.addSource(link)
	if builder.index.Sources[0].Local || builder.sourceFiles[id] != "" {
		t.Fatal("source symlink escaped trusted root")
	}
	source := filepath.Join(root, "local.c")
	if err := os.WriteFile(source, []byte("local source"), 0o600); err != nil {
		t.Fatal(err)
	}
	id = builder.addSource(source)
	bundle := Bundle{SourceFiles: builder.sourceFiles, sourceRoots: builder.sourceRoots}
	if data, err := bundle.ReadSource(id); err != nil || string(data) != "local source" {
		t.Fatalf("local source = %q, %v", data, err)
	}
	if err := os.Remove(source); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, source); err != nil {
		t.Fatal(err)
	}
	if data, err := bundle.ReadSource(id); err == nil {
		t.Fatalf("request-time symlink replacement exposed %q", data)
	}
	// SourceFiles is public, so reading must enforce the root boundary even
	// if a caller accidentally inserts an untrusted path after indexing.
	bundle.SourceFiles["outside"] = secret
	if _, err := bundle.ReadSource("outside"); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("untrusted source read = %v, want permission error", err)
	}
	if _, err := bundle.ReadSource("unknown"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unknown source read = %v, want missing-file error", err)
	}
	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}
	if _, err := bundle.ReadSource(id); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("removed source root read = %v, want missing-file error", err)
	}
}
