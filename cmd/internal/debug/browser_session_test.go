//go:build !llgo

package debug

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/debugabi"
	"github.com/xgo-dev/llgo/cmd/compiler/wasmdebug"
	"github.com/xgo-dev/llgo/cmd/internal/browser"
	"github.com/xgo-dev/llgo/target"
)

func TestBrowserSessionUsesChromeAndOwnsProfile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper uses a POSIX shell")
	}
	clang, err := exec.LookPath("clang")
	if err != nil {
		t.Skip("clang is unavailable")
	}
	dir := t.TempDir()
	source, artifact := filepath.Join(dir, "fixture.c"), filepath.Join(dir, "fixture.wasm")
	if err := os.WriteFile(source, []byte("int answer(void) { int value = 42; return value; }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command(clang, "--target=wasm32-unknown-unknown", "-O0", "-g", "-nostdlib", "-Wl,--no-entry", "-Wl,--export=answer", "-o", artifact, source).CombinedOutput(); err != nil {
		t.Fatalf("compile browser fixture: %v\n%s", err, out)
	}
	raw, err := os.ReadFile(artifact)
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
	if err := os.WriteFile(artifact, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	capture, chrome := filepath.Join(dir, "arguments"), filepath.Join(dir, "chrome")
	t.Setenv("LLGO_DEBUG_CHROME_CAPTURE", capture)
	if err := os.WriteFile(chrome, []byte(`#!/bin/sh
if [ "$1" = "--version" ]; then
  echo 'Google Chrome for Testing 152.0.0.0'
  exit 0
fi
printf '%s\n' "$@" > "$LLGO_DEBUG_CHROME_CAPTURE"
`), 0o700); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := runSession(session{
		backend: backendBrowser, artifact: artifact,
		target:       &target.Config{Name: "emscripten"}, // No native debug server is needed.
		debuggerArgs: []string{"--window-size=900,700"},
		options:      options{browser: browser.Options{Chrome: chrome, DisableTools: true}},
	}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	rawArgs, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	args := strings.Split(strings.TrimSpace(string(rawArgs)), "\n")
	var profile string
	for _, arg := range args {
		if strings.HasPrefix(arg, "--user-data-dir=") {
			profile = strings.TrimPrefix(arg, "--user-data-dir=")
		}
	}
	if profile == "" || !strings.Contains(string(rawArgs), "--window-size=900,700") || !strings.HasPrefix(args[len(args)-1], "http://127.0.0.1:") || !strings.Contains(args[len(args)-1], "llgo-devtools=disabled") {
		t.Fatalf("Chrome did not receive the browser session options: %q", rawArgs)
	}
	if strings.Contains(string(rawArgs), "--auto-open-devtools-for-tabs") {
		t.Fatal("disabled DevTools were opened")
	}
	if _, err := os.Stat(profile); !os.IsNotExist(err) {
		t.Fatalf("owned browser profile still exists: %v", err)
	}
}
