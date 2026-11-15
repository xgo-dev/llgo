package build

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"

	"github.com/xgo-dev/llgo/internal/build/funcinfo"
	"github.com/xgo-dev/llgo/internal/pclnmap"
	"github.com/xgo-dev/llgo/internal/wasmdebug"
)

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
	for _, failure := range []string{"missing-module", "invalid-module", "missing-slot", "invalid-pointer-width", "sidecar-write"} {
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
			case "sidecar-write":
				// A non-directory component prevents artifact staging.
				out.PCLN = filepath.Join(module, "sidecar")
			}
			if failure != "missing-module" {
				if err := os.WriteFile(module, raw, 0600); err != nil {
					t.Fatal(err)
				}
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
