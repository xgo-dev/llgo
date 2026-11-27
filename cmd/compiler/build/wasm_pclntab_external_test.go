package build

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/cl"
	"github.com/xgo-dev/llgo/cmd/compiler/build/funcinfo"
	"github.com/xgo-dev/llgo/cmd/compiler/pclnmap"
	"github.com/xgo-dev/llgo/cmd/compiler/wasmdebug"
)

func TestExternalWasmPCLNRetainsLogicalOwners(t *testing.T) {
	const owner = "main.inlined"
	const physical = "main.withoutLocation"
	table, err := funcinfo.EncodeWithPCLines(
		[]funcinfo.Record{
			{Symbol: owner, Name: owner, File: "app.go", Line: 3},
			{Symbol: physical, Name: physical, File: "app.go", Line: 5},
		},
		[]funcinfo.PCLineRecord{{ID: 42, Symbol: owner, File: "call.go", Line: 11}},
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := &context{
		wasmClosedProgram: true,
		frontendOptions:   cl.Options{CompactCallerLocations: true},
		buildConf:         &Config{Goos: "wasip1", Goarch: "wasm", BuildMode: BuildModeExe},
		pclnExternal: &pclnmap.Data{
			GOOS: "wasip1", GOARCH: "wasm", PointerSize: 4, Table: table,
			SymbolIndex: []pclnmap.SymbolIndexEntry{
				{SymbolID: funcInfoSymbolID(owner), FuncIndex: 1},
				{SymbolID: funcInfoSymbolID(physical), FuncIndex: 2},
			},
		},
	}
	// The complete name section lists only the physical function.
	module := []byte("\x00asm\x01\x00\x00\x00\x01\x04\x01\x60\x00\x00\x03\x02\x01\x00\x0a\x04\x01\x02\x00\x0b")
	names := append([]byte{1, 0, byte(len(physical))}, physical...)
	custom := append([]byte{4, 'n', 'a', 'm', 'e', 1, byte(len(names))}, names...)
	module = append(module, 0, byte(len(custom)))
	module = append(module, custom...)
	plan := &mainLinkPlan{pclnExternal: ctx.pclnExternal}
	pruned, err := linkExternalWasmPCLNData(newLinkExecutionContext(ctx, plan), module)
	if err != nil {
		t.Fatal(err)
	}
	if len(pruned.Table.Records) != 1 || pruned.Table.Symbol(pruned.Table.Records[0]) != owner {
		t.Fatalf("expected only the logical frame owner, got %+v", pruned.Table.Records)
	}
	if len(pruned.Table.PCLines) != 1 || pruned.Table.PCLines[0].ID != 42 || pruned.Table.PCLines[0].Func != 1 {
		t.Fatalf("logical frame location changed: %+v", pruned.Table.PCLines)
	}
	for _, closed := range []bool{true, false} {
		ctx.wasmClosedProgram = closed
		plan.runtimeSites = closed
		full, err := linkExternalWasmPCLNData(newLinkExecutionContext(ctx, plan), module)
		if err != nil || len(full.Table.Records) != 2 ||
			len(full.Table.PCLines) != 1 || full.Table.PCLines[0].ID != 42 {
			t.Fatalf("closed=%t: lost physical entry or inline location: %v, %+v", closed, err, full)
		}
	}
	ctx.frontendOptions.CompactCallerLocations = false
	debugger, err := linkExternalWasmPCLNData(newLinkExecutionContext(ctx, plan), module)
	if err != nil || len(debugger.Table.Records) != 1 || len(debugger.Table.PCLines) != 0 {
		t.Fatalf("debugger map must keep the linked-name policy: %v, %+v", err, debugger)
	}
	badNames := append(slices.Clone(module[:8]), 0, 6, 4, 'n', 'a', 'm', 'e', 1)
	if _, err := linkExternalWasmPCLNData(newLinkExecutionContext(ctx, plan), badNames); err == nil {
		t.Fatal("accepted an incomplete Wasm name subsection")
	}
}

func TestExternalWasmPCLNPairing(t *testing.T) {
	table, err := funcinfo.EncodeWithPCLines([]funcinfo.Record{{Symbol: "main.main", Name: "main.main", File: "main.go", Line: 1}}, []funcinfo.PCLineRecord{{ID: 7, Symbol: "main.main", File: "line.go", Line: 123}})
	if err != nil {
		t.Fatal(err)
	}
	slot := wasmdebug.PCLNIdentityPlaceholder()
	// One passive data segment; pairing must preserve every code/data offset.
	raw := append([]byte("\x00asm\x01\x00\x00\x00\x0b\x23\x01\x01\x20"), slot[:]...)
	for _, extension := range []string{".wasm", ".mjs"} {
		t.Run(extension, func(t *testing.T) {
			dir := t.TempDir()
			module := filepath.Join(dir, "app.wasm")
			out := &OutFmtDetails{Out: filepath.Join(dir, "app"+extension), PCLN: module + ".pclntab"}
			conf := &Config{Goos: "wasip1", Goarch: "wasm", PCLNMode: PCLNExternal}
			if extension == ".mjs" {
				conf.Goos = "js"
				if err := os.WriteFile(out.Out, []byte("glue"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			ctx := &context{buildConf: conf, pclnExternal: &pclnmap.Data{GOOS: conf.Goos, GOARCH: "wasm", PointerSize: 4, Table: table, SymbolIndex: []pclnmap.SymbolIndexEntry{{SymbolID: funcInfoSymbolID("main.main"), FuncIndex: 1}}}}
			if err := os.WriteFile(module, raw, 0600); err != nil {
				t.Fatal(err)
			}
			if err := finalizeRuntimePCLN(ctx, out, true); err != nil {
				t.Fatal(err)
			}
			sidecar, err := os.ReadFile(out.PCLN)
			if err != nil {
				t.Fatal(err)
			}
			view, err := pclnmap.Parse(sidecar)
			if err != nil {
				t.Fatal(err)
			}
			want := sha256.Sum256(raw)
			if view.Identity != want || view.PointerSize != 4 {
				t.Fatal("sidecar pairing identity or width changed")
			}
			patched, err := os.ReadFile(module)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(patched, bytes.Replace(raw, slot[:], want[:], 1)) {
				t.Fatal("rewrote bytes outside the identity slot")
			}
			if extension == ".mjs" {
				glue, _ := os.ReadFile(out.Out)
				if string(glue) != "glue" {
					t.Fatal("rewrote JS glue")
				}
			}
			// Rebuilds in both non-external policies remove the module's sidecar.
			for _, mode := range []PCLNMode{PCLNNone, PCLNEmbedded} {
				conf.PCLNMode = mode
				if err := finalizeRuntimePCLN(ctx, out, false); err != nil {
					t.Fatal(err)
				}
				if _, err := os.Stat(out.PCLN); !os.IsNotExist(err) {
					t.Fatalf("stale sidecar after %s", mode)
				}
				if err := os.WriteFile(out.PCLN, sidecar, 0600); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestExternalWasmPCLNFailureDoesNotPublish(t *testing.T) {
	for _, failure := range []string{"missing-module", "invalid-module", "missing-slot", "invalid-pointer-width", "invalid-symbol-index", "sidecar-write", "module-write"} {
		t.Run(failure, func(t *testing.T) {
			dir := t.TempDir()
			module := filepath.Join(dir, "app.wasm")
			out := &OutFmtDetails{Out: module, PCLN: module + ".pclntab"}
			ctx := &context{buildConf: &Config{Goos: "wasip1", Goarch: "wasm", PCLNMode: PCLNExternal},
				pclnExternal: &pclnmap.Data{GOOS: "wasip1", GOARCH: "wasm", PointerSize: 4}}
			slot := wasmdebug.PCLNIdentityPlaceholder()
			raw := append([]byte("\x00asm\x01\x00\x00\x00\x0b\x23\x01\x01\x20"), slot[:]...)
			switch failure {
			case "invalid-module":
				raw = []byte("invalid")
			case "missing-slot":
				raw = raw[:8]
			case "invalid-pointer-width":
				ctx.pclnExternal.PointerSize = 3
			case "invalid-symbol-index":
				ctx.pclnExternal.SymbolIndex = []pclnmap.SymbolIndexEntry{{SymbolID: 1, FuncIndex: 1}}
			case "sidecar-write":
				// A non-directory component prevents artifact staging.
				out.PCLN = filepath.Join(module, "sidecar")
			}
			if failure != "missing-module" {
				if err := os.WriteFile(module, raw, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if failure == "module-write" {
				if runtime.GOOS == "windows" || os.Geteuid() == 0 {
					t.Skip("requires Unix directory permission enforcement")
				}
				out.PCLN = filepath.Join(t.TempDir(), "app.wasm.pclntab")
				if err := os.Chmod(dir, 0500); err != nil {
					t.Fatal(err)
				}
				defer os.Chmod(dir, 0700)
			}
			if err := finalizeRuntimePCLN(ctx, out, false); err == nil {
				t.Fatal("accepted invalid external artifact")
			}
			if _, err := os.Stat(out.PCLN); err == nil {
				t.Fatal("published a sidecar on failure")
			}
			if failure != "missing-module" {
				got, err := os.ReadFile(module)
				if err != nil || !bytes.Equal(got, raw) {
					t.Fatal("changed module after failed pairing")
				}
			}
		})
	}
}
