//go:build !llgo

package build

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/goplus/llgo/internal/mockable"
)

func TestRunCmdPassesGoBuildFlags(t *testing.T) {
	oldArgs := append([]string(nil), goBuildFlags.Args...)
	goBuildFlags.Args = nil
	defer func() { goBuildFlags.Args = oldArgs }()

	stderrFile, err := os.CreateTemp(t.TempDir(), "stderr")
	if err != nil {
		t.Fatal(err)
	}
	oldStderr := os.Stderr
	os.Stderr = stderrFile
	defer func() { os.Stderr = oldStderr }()

	mockable.EnableMock()
	defer mockable.DisableMock()
	exited := false
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				if recovered == "exit" {
					exited = true
					return
				}
				panic(recovered)
			}
		}()
		runCmd(Cmd, []string{"-gcflags=all=-N", "-ldflags=-s -w", filepath.Join(t.TempDir(), "missing")})
	}()
	if !exited || mockable.ExitCode() != 1 {
		t.Fatalf("runCmd exit = (%v, %d), want (true, 1)", exited, mockable.ExitCode())
	}
	if !reflect.DeepEqual(goBuildFlags.Args, []string{"-gcflags=all=-N", "-ldflags=-s -w"}) {
		t.Fatalf("go build flags = %v", goBuildFlags.Args)
	}
	if err := stderrFile.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(stderrFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "missing") {
		t.Fatalf("stderr = %q, want missing-package diagnostic", data)
	}
}

func TestBuildCommandHasSchedulerTraceFlag(t *testing.T) {
	flag := Cmd.Flag.Lookup("debug-trace")
	if flag == nil {
		t.Fatal("llgo build has no -debug-trace flag")
	}
	if !strings.Contains(flag.Usage, "Chrome/Perfetto") {
		t.Fatalf("-debug-trace usage = %q", flag.Usage)
	}
}
