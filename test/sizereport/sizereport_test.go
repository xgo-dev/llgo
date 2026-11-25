//go:build !wasm

package sizereport_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/sizereport"
)

func TestCollectFinalSizeRealBinary(t *testing.T) {
	path := os.Getenv("LLGO_SIZE_REPORT_BIN")
	if path == "" {
		// The size reporter runs on the compiler host. Build a real linked
		// Wasm artifact so this smoke test also runs without a supplied binary.
		dir := t.TempDir()
		source := filepath.Join(dir, "main.go")
		if err := os.WriteFile(source, []byte("package main\nfunc main() { println(\"size-report fixture\") }\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		path = filepath.Join(dir, "main.wasm")
		cmd := exec.Command("go", "build", "-o", path, source)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm", "CGO_ENABLED=0", "GO111MODULE=off", "GOWORK=off", "GOENV=off", "GOFLAGS=")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("build size-report fixture: %v\n%s", err, output)
		}
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
