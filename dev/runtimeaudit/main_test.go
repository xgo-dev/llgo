package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/build"
)

func TestAuditCommand(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	t.Setenv("GOFLAGS", "")
	var stdout bytes.Buffer
	if err := run([]string{"-goos=linux", "-goarch=amd64", "-tags=audit_fixture, audit_extra"}, &stdout); err != nil {
		t.Fatal(err)
	}
	var report build.RuntimeOwnership
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.GOOS != "linux" || report.GOARCH != "amd64" || len(report.Symbols) < 100 || len(report.Missing) != 0 {
		t.Fatalf("unexpected report: %+v", report)
	}
	if !slices.Contains(report.Tags, "audit_fixture") || !slices.Contains(report.Tags, "audit_extra") {
		t.Fatalf("lost requested build tags: %v", report.Tags)
	}
	if err := run([]string{"-goos=linux", "-goarch=amd64", "-runtime=" + filepath.Join(t.TempDir(), "missing")}, io.Discard); err == nil {
		t.Fatal("accepted a missing runtime tree")
	}
	output := filepath.Join(t.TempDir(), "report.json")
	if err := run([]string{"-goos=windows", "-goarch=386", "-runtime=" + filepath.Join(root, "runtime"), "-o=" + output}, io.Discard); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(output); err != nil || !json.Valid(data) {
		t.Fatalf("report file: %v", err)
	}
	// A raw Wasm selection without the hosted GC profile exposes missing
	// runtime APIs. Preserve the diagnostic report even though the command fails.
	stdout.Reset()
	err = run([]string{"-goos=js", "-goarch=wasm"}, &stdout)
	if err == nil || !strings.Contains(err.Error(), "unresolved runtime references") || !json.Valid(stdout.Bytes()) {
		t.Fatalf("missing-hook result: %v, report bytes %d", err, stdout.Len())
	}
	for _, args := range [][]string{{"unexpected"}, {"-unknown"}, {"-goos=invalid"}, {"-goos=linux", "-goarch=amd64", "-o=" + t.TempDir()}} {
		if err := run(args, io.Discard); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}
