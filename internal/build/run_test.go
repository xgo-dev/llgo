//go:build !windows

package build

import (
	"os"
	"strings"
	"testing"
)

func TestRunInEmulatorValidation(t *testing.T) {
	commands := commandEnv{dir: t.TempDir(), environ: os.Environ()}
	if err := runInEmulator(commands, "", nil, "", "", &Config{CompileOnly: true}, ModeRun, false); err != nil {
		t.Fatalf("compile-only emulator run failed: %v", err)
	}
	if err := runInEmulator(commands, "", nil, "", "", &Config{Target: "demo"}, ModeRun, false); err == nil {
		t.Fatal("missing emulator succeeded")
	}
	if err := runEmuCmd(commands, nil, "'", nil, false, false); err == nil || !strings.Contains(err.Error(), "parse") {
		t.Fatalf("malformed emulator command error = %v", err)
	}
	if err := runEmuCmd(commands, nil, "   ", nil, false, false); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("empty emulator command error = %v", err)
	}
}
