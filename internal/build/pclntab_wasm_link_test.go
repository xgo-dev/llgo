package build

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xgo-dev/llgo/internal/build/funcinfo"
	"github.com/xgo-dev/llgo/internal/pclnmap"
	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestWasmPCLNLiveness(t *testing.T) {
	records := []funcInfoRecord{{symbol: "main.live", name: "main.live", file: "live.go", line: 2}, {symbol: "main.dead", name: "main.dead", file: "dead.go", line: 3}}
	table, err := funcinfo.EncodeWithPCLines(toFuncInfoRecords(records), nil)
	if err != nil {
		t.Fatal(err)
	}
	data := pclnmap.Data{GOARCH: "wasm", PointerSize: 4, Table: table, SymbolIndex: []pclnmap.SymbolIndexEntry{
		{SymbolID: funcInfoSymbolID("main.live"), FuncIndex: 1},
		{SymbolID: funcInfoSymbolID("main.dead"), FuncIndex: 2},
	}}
	for _, tc := range []struct {
		name         string
		functions    byte
		index        byte
		functionName string
		want         int
	}{
		{"promoted LTO name", 1, 0, "main.live.llvm.12", 1},
		{"absent map", 2, 0, "", 2},
		{"empty map", 2, 0, "empty-map", 2},
		{"fully inlined", 1, 0, "fully.inlined", 0},
		{"sparse map", 2, 0, "main.live", 2},
		{"out of range index", 1, 7, "main.live", 2},
		{"empty name", 1, 0, "empty-name", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			module := []byte("\x00asm\x01\x00\x00\x00\x01\x04\x01\x60\x00\x00")
			module = append(module, 3, tc.functions+1, tc.functions)
			for range tc.functions {
				module = append(module, 0)
			}
			module = append(module, 10, tc.functions*3+1, tc.functions)
			for range tc.functions {
				module = append(module, 2, 0, 0x0b)
			}
			if tc.functionName != "" {
				name := tc.functionName
				if name == "empty-name" {
					name = ""
				}
				payload := append([]byte{1, tc.index, byte(len(name))}, name...)
				if name == "empty-map" {
					payload = []byte{0}
				}
				custom := append([]byte{4, 'n', 'a', 'm', 'e', 1, byte(len(payload))}, payload...)
				module = append(module, 0, byte(len(custom)))
				module = append(module, custom...)
			}
			path := filepath.Join(t.TempDir(), "module.wasm")
			if err := os.WriteFile(path, module, 0600); err != nil {
				t.Fatal(err)
			}
			analysis, err := analyzeWasmPCLN(path, data)
			if err != nil {
				t.Fatal(err)
			}
			pruned, err := linkedPCLNData(data, analysis)
			if err != nil {
				t.Fatal(err)
			}
			if len(pruned.Table.Records) != tc.want {
				t.Fatalf("records=%d, want %d", len(pruned.Table.Records), tc.want)
			}
		})
	}
}

func TestWasmPCLNPruningScope(t *testing.T) {
	for _, tc := range []struct {
		provider string
		entries  bool
		want     bool
	}{
		{"wasi", false, true},
		{"wasi", true, false},
		{"emscripten", false, false},
		{"gojs", false, false},
	} {
		prog := llssa.NewProgram(&llssa.Target{GOOS: "wasip1", GOARCH: "wasm", WasmProvider: tc.provider, WasmFuncInfoEntries: tc.entries})
		ctx := &context{prog: prog, buildConf: &Config{Goarch: "wasm", BuildMode: BuildModeExe, PCLNMode: PCLNEmbedded}}
		if got := canPruneEmbeddedPCLN(ctx); got != tc.want {
			t.Fatalf("%s entries=%v: eligible=%v, want %v", tc.provider, tc.entries, got, tc.want)
		}
		ctx.buildConf.BuildMode = BuildModeCArchive
		if canPruneEmbeddedPCLN(ctx) {
			t.Fatal("pruning library metadata")
		}
		prog.Dispose()
	}
}

func TestWasmPCLNInvalidModule(t *testing.T) {
	path := filepath.Join(t.TempDir(), "module.wasm")
	if _, err := analyzeWasmPCLN(path, pclnmap.Data{}); err == nil {
		t.Fatal("accepted missing linked module")
	}
	if err := os.WriteFile(path, []byte("\x00asm"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := analyzeWasmPCLN(path, pclnmap.Data{}); err == nil {
		t.Fatal("accepted truncated linked module")
	}
}
