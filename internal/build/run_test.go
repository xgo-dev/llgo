//go:build !llgo && !windows

package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goplus/llgo/internal/processenv"
)

func TestRunInEmulatorValidationAndRequestContext(t *testing.T) {
	process := processenv.Context{
		Dir: t.TempDir(),
		Env: []string{"PATH=" + filepath.Dir("/bin/sh")},
	}
	ctx := &context{process: process}

	if err := runInEmulator(ctx, "", nil, "", "", &Config{CompileOnly: true}, ModeRun, false); err != nil {
		t.Fatalf("compile-only emulator run failed: %v", err)
	}
	if err := runInEmulator(ctx, "", nil, "", "", &Config{Target: "demo"}, ModeRun, false); err == nil {
		t.Fatal("missing emulator succeeded")
	}
	for _, mode := range []Mode{ModeRun, ModeTest} {
		err := runInEmulator(ctx, "/bin/sh -c 'exit 7'", nil, "", "", &Config{}, mode, false)
		if err == nil {
			t.Fatalf("emulator mode %v succeeded", mode)
		}
	}

	if err := runEmuCmd(process, nil, "'", nil, false, false); err == nil || !strings.Contains(err.Error(), "parse") {
		t.Fatalf("malformed emulator command error = %v", err)
	}
	if err := runEmuCmd(process, nil, "   ", nil, false, false); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("empty emulator command error = %v", err)
	}
}

func TestRunNativeWasmUsesConfigEnvironment(t *testing.T) {
	conf := &Config{
		Goos: "wasip1",
		environment: []string{
			llgoWasmRuntime + "=$REQUEST_WASM_RUNTIME",
			"REQUEST_WASM_RUNTIME=/definitely/missing/wasm-runtime",
		},
	}
	ctx := &context{process: processenv.Context{Dir: t.TempDir(), Env: os.Environ()}}
	if err := runNative(ctx, "app.wasm", "", "", conf, ModeRun); err == nil {
		t.Fatal("missing configured wasm runtime succeeded")
	}
}
