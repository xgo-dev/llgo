//go:build !llgo

package run

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xgo-dev/llgo/internal/mockable"
)

func TestRunAndCmpTestBuildFlagsAreIndependent(t *testing.T) {
	if runGoBuildFlags == cmpTestGoBuildFlags {
		t.Fatal("run and cmptest share a build-flag collector")
	}
	if Cmd.Flag.Lookup("ldflags") == nil || CmpTestCmd.Flag.Lookup("ldflags") == nil {
		t.Fatal("run or cmptest does not register -ldflags")
	}
	if Cmd.Flag.Lookup("timeout") == nil {
		t.Fatal("run does not register -timeout")
	}
	if CmpTestCmd.Flag.Lookup("timeout") != nil {
		t.Fatal("cmptest unexpectedly registers the run-only -timeout")
	}
	if runGoBuildFlags.Flag != &Cmd.Flag || cmpTestGoBuildFlags.Flag != &CmpTestCmd.Flag {
		t.Fatal("run or cmptest build flags are bound to the wrong command")
	}
}

func TestRunPreservesNativeExitCode(t *testing.T) {
	program := filepath.Join(t.TempDir(), "exit.go")
	if err := os.WriteFile(program, []byte("package main\nimport \"os\"\nfunc main() { os.Exit(23) }\n"), 0600); err != nil {
		t.Fatal(err)
	}
	mockable.EnableMock()
	defer mockable.DisableMock()
	defer func() {
		if recovered := recover(); recovered != "exit" || mockable.ExitCode() != 23 {
			t.Errorf("llgo run exit = (%v, %d), want (exit, 23)", recovered, mockable.ExitCode())
		}
	}()
	runCmd(Cmd, []string{program})
}
