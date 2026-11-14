package debug

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/browserdebug"
	"github.com/xgo-dev/llgo/internal/debugabi"
	"github.com/xgo-dev/llgo/internal/wasmdebug"
)

func TestWASIHostSelection(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing")
	t.Setenv("PATH", dir)
	t.Setenv("LLGO_WASI_HOST", missing)
	if _, err := findWASIHost(""); err == nil || !strings.Contains(err.Error(), "find WASI host") {
		t.Fatalf("missing environment executable: %v", err)
	}
	t.Setenv("LLGO_WASI_HOST", "")
	if _, err := findWASIHost(""); err == nil {
		t.Fatal("accepted missing default executable")
	}
	if runtime.GOOS == "windows" {
		return // The remaining fixtures are executable POSIX shell wrappers.
	}
	path := filepath.Join(dir, "llgo-wasi-debug")
	write := func(body string) {
		t.Helper()
		if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	write("printf 'llgo-wasi-debug 1.0.0\\n'")
	if got, err := findWASIHost(""); err != nil || got != path {
		t.Fatalf("PATH executable: %s, %v", got, err)
	}
	t.Setenv("LLGO_WASI_HOST", missing)
	if got, err := findWASIHost(path); err != nil || got != path {
		t.Fatalf("explicit executable must override environment: %s, %v", got, err)
	}
	t.Setenv("LLGO_WASI_HOST", path)
	if got, err := findWASIHost(""); err != nil || got != path {
		t.Fatalf("environment executable: %s, %v", got, err)
	}
	write("printf 'llgo-wasi-debug 2.0.0\\n'")
	if _, err := findWASIHost(path); err == nil || !strings.Contains(err.Error(), "ABI 1") {
		t.Fatalf("accepted incompatible runtime: %v", err)
	}
	write("exit 7")
	if _, err := findWASIHost(path); err == nil || !strings.Contains(err.Error(), "query WASI host version") {
		t.Fatalf("accepted failed version query: %v", err)
	}
}

func TestWASIHostVersions(t *testing.T) {
	for _, tc := range []struct {
		output string
		valid  bool
	}{
		{"llgo-wasi-debug 1.0.0 (Wasmtime 50.0.1)", true},
		{"llgo-wasi-debug 1.1.0", true},
		{"llgo-wasi-debug 2.0.0", false},
		{"llgo-wasi-debug 0.1.0", false},
		{"wasmtime 50.0.1", false},
		{"llgo-wasi-debug development", false},
		{"", false},
	} {
		t.Run(tc.output, func(t *testing.T) {
			err := wasiHostVersion(tc.output)
			if (err == nil) != tc.valid {
				t.Fatalf("version %q: %v", tc.output, err)
			}
		})
	}
}

func TestWASIDeployedCodeIdentity(t *testing.T) {
	clang, err := exec.LookPath("clang")
	if err != nil {
		t.Skip("clang is unavailable")
	}
	dir := t.TempDir()
	source, module := filepath.Join(dir, "probe.c"), filepath.Join(dir, "probe.wasm")
	compile := func(result int) []byte {
		t.Helper()
		if err := os.WriteFile(source, []byte(fmt.Sprintf("int answer(void) { return %d; }\n", result)), 0o600); err != nil {
			t.Fatal(err)
		}
		if out, err := exec.Command(clang, "--target=wasm32-unknown-unknown", "-O0", "-g", "-nostdlib",
			"-Wl,--no-entry", "-Wl,--export=answer", "-o", module, source).CombinedOutput(); err != nil {
			t.Fatalf("compile Wasm: %v\n%s", err, out)
		}
		raw, err := os.ReadFile(module)
		if err != nil {
			t.Fatal(err)
		}
		raw, err = wasmdebug.SetDebuggerRecord(raw, debugabi.NewRecord(4, debugabi.ByteOrderLittle))
		if err != nil {
			t.Fatal(err)
		}
		raw, _, err = wasmdebug.EnsureBuildID(raw)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	write := func(path string, data []byte) {
		t.Helper()
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	correct := compile(42)
	write(module, correct)
	if _, err := wasiDebugModule(module, nil); err != nil {
		t.Fatal(err)
	}
	for _, record := range []debugabi.Record{
		debugabi.NewRecord(8, debugabi.ByteOrderLittle),
		debugabi.NewRecord(4, debugabi.ByteOrderBig),
	} {
		unsupported, err := wasmdebug.SetDebuggerRecord(correct, record)
		if err != nil {
			t.Fatal(err)
		}
		write(module, unsupported)
		if _, err := wasiDebugModule(module, nil); err == nil || !strings.Contains(err.Error(), "little-endian W32") {
			t.Fatalf("accepted unsupported WASI record %+v: %v", record, err)
		}
	}
	deployed, err := wasmdebug.Externalize(correct, "probe%20debug.wasm")
	if err != nil {
		t.Fatal(err)
	}
	sidecar := filepath.Join(dir, "probe debug.wasm")
	write(module, deployed)
	write(sidecar, correct)
	if _, err := wasiDebugModule(module, nil); err != nil {
		t.Fatal(err)
	}
	// A sidecar is executable here. Matching claimed build IDs is insufficient:
	// it must reproduce every deployment byte when its DWARF is removed.
	changed := compile(43)
	id, _, err := wasmdebug.BuildID(correct)
	if err != nil {
		t.Fatal(err)
	}
	changed, err = wasmdebug.SetBuildID(changed, id)
	if err != nil {
		t.Fatal(err)
	}
	write(sidecar, changed)
	write(module, deployed)
	if _, err := browserdebug.Load(module, nil); err != nil {
		t.Fatalf("test must preserve the advertised identity: %v", err)
	}
	if _, err := wasiDebugModule(module, nil); err == nil || !strings.Contains(err.Error(), "code/data") {
		t.Fatalf("executed mismatched sidecar: %v", err)
	}
}

func TestWASIHostArguments(t *testing.T) {
	// The debugger owns these options; guest paths and debugger commands stay
	// separate argv entries. Keep actual WASI threads, shared memory and EH.
	args := wasiLLDBArguments("dir with space/llgo-wasi-debug", "dir with space/app.wasm", []string{"--batch", "-o", "breakpoint set --name main.main"})
	want := []string{"-O", "settings set plugin.jit-loader.gdb.enable on", "--batch", "-o", "breakpoint set --name main.main", "--", "dir with space/llgo-wasi-debug", "dir with space/app.wasm"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("WASI debugger arguments: %q", args)
	}
}
