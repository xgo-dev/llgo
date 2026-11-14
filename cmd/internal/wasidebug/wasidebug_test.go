//go:build !llgo

package wasidebug

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/browserdebug"
	"github.com/xgo-dev/llgo/internal/debugabi"
	"github.com/xgo-dev/llgo/internal/wasmdebug"
)

func sessionFixture(t *testing.T) *browserdebug.Bundle {
	t.Helper()
	raw, err := wasmdebug.SetDebuggerRecord([]byte{'\x00', 'a', 's', 'm', 1, 0, 0, 0}, debugabi.NewRecord(4, debugabi.ByteOrderLittle))
	if err != nil {
		t.Fatal(err)
	}
	raw, id, err := wasmdebug.EnsureBuildID(raw)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "original.wasm")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return &browserdebug.Bundle{SymbolsPath: path, Index: browserdebug.Index{
		Contract: "llgo.browser.debug", Version: browserdebug.IndexVersion, BuildID: hex.EncodeToString(id),
		Sources: []browserdebug.Source{}, Types: []browserdebug.Type{}, Variables: []browserdebug.Variable{},
		Record: browserdebug.Record{RecordVersion: 1, SchemaVersion: 1, RuntimeLayoutVersion: 2,
			LLGoABIVersion: 1, PointerSize: 4, ByteOrder: debugabi.ByteOrderLittle},
	}}
}

func TestPrivateSessionAndDecoderCompatibility(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node.js is unavailable")
	}
	bundle := sessionFixture(t)
	directory := filepath.Join(t.TempDir(), "private session with spaces")
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	module, plugin, err := Prepare(directory, bundle)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(module)
	if err != nil {
		t.Fatal(err)
	}
	if plugin != filepath.Join(directory, "llgo_wasi.py") {
		t.Fatalf("unexpected LLDB plugin path: %s", plugin)
	}
	// The running engine and decoder own a snapshot, not the mutable source.
	if err := os.WriteFile(bundle.SymbolsPath, []byte("replaced"), 0o600); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(module)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("source mutation changed the private module: %v", err)
	}
	run := func() ([]byte, error) {
		cmd := exec.Command(node, filepath.Join(directory, "decoder.js"))
		cmd.Stdin = strings.NewReader("{\"expression\":\"$goroutines\",\"depth\":0,\"stopId\":\"1\"}\n")
		return cmd.CombinedOutput()
	}
	output, err := run()
	if err != nil || !bytes.Contains(output, []byte("build with llgo.wasm.debugger")) || bytes.Contains(output, []byte(`"read"`)) {
		t.Fatalf("ordinary module decoder status: %v\n%s", err, output)
	}
	// Unsupported schema disables presentation before any memory request.
	var schema map[string]any
	if err := json.Unmarshal(debugabi.SchemaV1(), &schema); err != nil {
		t.Fatal(err)
	}
	schema["schema_version"] = 99
	changed, err := json.Marshal(schema)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "schema.json"), changed, 0o600); err != nil {
		t.Fatal(err)
	}
	output, err = run()
	if err == nil || !bytes.Contains(output, []byte("unsupported LLGo debugger schema")) || bytes.Contains(output, []byte(`"read"`)) {
		t.Fatalf("unsupported module used runtime memory: %v\n%s", err, output)
	}
}

func TestPrepareUnavailableArtifacts(t *testing.T) {
	bundle := sessionFixture(t)
	if err := os.Remove(bundle.SymbolsPath); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Prepare(t.TempDir(), bundle); err == nil {
		t.Fatal("prepared a session without its executable module")
	}
	bundle = sessionFixture(t)
	directory := t.TempDir()
	if err := os.Mkdir(filepath.Join(directory, "module.wasm"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Prepare(directory, bundle); err == nil {
		t.Fatal("prepared a session without a writable module snapshot")
	}
	directory = t.TempDir()
	for _, asset := range []string{"plugin.js", "goroutines.js"} {
		if err := os.Mkdir(filepath.Join(directory, asset), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err := Prepare(directory, bundle); err == nil {
		t.Fatal("prepared a session without its runtime decoder")
	}
}
