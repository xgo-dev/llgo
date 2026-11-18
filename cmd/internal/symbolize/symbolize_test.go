package symbolize

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/build/funcinfo"
	"github.com/xgo-dev/llgo/internal/pclnmap"
	"github.com/xgo-dev/llgo/internal/wasmdebug"
)

func TestRunResolvesMatchingWasmLocations(t *testing.T) {
	slot := wasmdebug.PCLNIdentityPlaceholder()
	module := append([]byte{
		0, 'a', 's', 'm', 1, 0, 0, 0,
		5, 3, 1, 0, 1, // one memory page
		11, 38, 1, 0, 0x41, 0, 0x0b, 32, // one initialized data segment
	}, slot[:]...)
	withNames := append(append([]byte{}, module...), 0, 5, 4, 'n', 'a', 'm', 'e')
	identity, err := wasmdebug.PCLNModuleIdentity(withNames)
	if err != nil {
		t.Fatal(err)
	}
	module, err = wasmdebug.ReplaceData(module, "identity", slot[:], identity[:])
	if err != nil {
		t.Fatal(err)
	}
	table, err := funcinfo.EncodeWithPCLines(
		[]funcinfo.Record{{Symbol: "main.crash", Name: "main.crash", File: "app/panic.go", Line: 8}},
		[]funcinfo.PCLineRecord{{ID: 42, Symbol: "main.crash", File: "app/panic.go", Line: 9}},
	)
	if err != nil {
		t.Fatal(err)
	}
	sidecar, err := pclnmap.Encode(pclnmap.Data{
		GOOS: "wasip1", GOARCH: "wasm", PointerSize: 4, TextStart: 1, TextEnd: 2,
		Identity: identity, Table: table,
	})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "app.wasm")
	write := func(path string, data []byte) {
		t.Helper()
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := Run(path, strings.NewReader(""), io.Discard); !os.IsNotExist(err) {
		t.Fatalf("missing module: %v", err)
	}
	write(path, module)
	if err := Run(path, strings.NewReader(""), io.Discard); !os.IsNotExist(err) {
		t.Fatalf("missing sidecar: %v", err)
	}
	write(path+".pclntab", []byte("invalid"))
	if err := Run(path, strings.NewReader(""), io.Discard); err == nil {
		t.Fatal("accepted invalid sidecar")
	}

	write(path+".pclntab", sidecar)
	const unknown = "??? [pc=16] [llgo-location=999](...)\n\t???:0\n"
	const unchanged = "??? [pc=16] [llgo-location=18446744073709551616](...)\n\t???:0\n"
	const trace = "panic: example\n??? [pc=16] [llgo-location=42](...)\n\t???:0\n" + unknown + unchanged
	const want = "panic: example\nmain.crash(...)\n\tapp/panic.go:9\n" + unknown + unchanged
	var out strings.Builder
	if err := Run(path, strings.NewReader(trace), &out); err != nil || out.String() != want {
		t.Fatalf("offline trace = %q, error = %v", out.String(), err)
	}

	refused := refusedTraceWriter{errors.New("write failed")}
	if err := Run(path, strings.NewReader(trace), refused); !errors.Is(err, refused.err) {
		t.Fatalf("write error = %v", err)
	}

	// The identity slot can still be found, but it no longer matches the code
	// or other bytes in this module.
	module[12] = 2 // change the memory reservation
	write(path, module)
	out.Reset()
	if err := Run(path, strings.NewReader(trace), &out); err == nil || out.Len() != 0 {
		t.Fatalf("mismatched module produced trace %q, error = %v", out.String(), err)
	}
}

type refusedTraceWriter struct{ err error }

func (w refusedTraceWriter) Write([]byte) (int, error) { return 0, w.err }
