package sizereport_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/xgo-dev/llgo/internal/sizereport"
)

func TestCollectFinalSizeRealBinary(t *testing.T) {
	path := os.Getenv("LLGO_SIZE_REPORT_BIN")
	if path == "" {
		t.Skip("set LLGO_SIZE_REPORT_BIN for a final-artifact smoke test")
	}
	report, err := sizereport.Collect(path, nil, "full")
	if err != nil {
		t.Fatal(err)
	}
	if report.Wasm != nil && report.Total.Code+report.Total.Data+report.Wasm.StructureBytes+report.Wasm.CustomBytes != report.FileSize {
		t.Fatal("file size does not close")
	}
	var output bytes.Buffer
	if err := sizereport.Write(&output, report, "json"); err != nil {
		t.Fatal(err)
	}
	if path := os.Getenv("LLGO_SIZE_REPORT_JSON"); path != "" {
		if err := os.WriteFile(path, output.Bytes(), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
