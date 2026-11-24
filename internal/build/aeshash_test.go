//go:build (darwin || linux || windows) && (amd64 || arm64 || 386)

package build

import (
	stdctx "context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRuntimeAESHashDispatch(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "main.go")
	if err := os.WriteFile(source, []byte(`package main

import (
	"hash/maphash"
	_ "unsafe"
)

//go:linkname useAeshash github.com/xgo-dev/llgo/runtime/internal/runtime.useAeshash
var useAeshash bool

func main() {
	seed := maphash.MakeSeed()
	buf := make([]byte, 256)
	for i := range buf {
		buf[i] = byte(i*17 + 3)
	}
	h1 := maphash.Bytes(seed, buf)
	buf[199] ^= 1
	h2 := maphash.Bytes(seed, buf)
	if h1 == h2 {
		panic("hash did not change after bit flip")
	}
	if useAeshash {
		println("aes=on")
	} else {
		println("aes=off")
	}
}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	outFile := filepath.Join(dir, "aeshash-probe")
	if runtime.GOOS == "windows" {
		outFile += ".exe"
	}
	conf := NewDefaultConf(ModeBuild)
	conf.OutFile = outFile
	if _, err := Do([]string{source}, conf); err != nil {
		t.Fatalf("build AES hash probe: %v", err)
	}

	run := func(godebug string) string {
		t.Helper()
		ctx, cancel := stdctx.WithTimeout(stdctx.Background(), time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, outFile)
		var env []string
		for _, kv := range os.Environ() {
			key, _, _ := strings.Cut(kv, "=")
			if strings.EqualFold(key, "GODEBUG") {
				continue
			}
			env = append(env, kv)
		}
		if godebug != "" {
			env = append(env, "GODEBUG="+godebug)
		}
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("run probe GODEBUG=%q: %v\n%s", godebug, err, out)
		}
		return strings.TrimSpace(string(out))
	}

	def := run("")
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" && def != "aes=on" {
		t.Fatalf("darwin/arm64 default = %q, want aes=on", def)
	}
	if got := run("cpu.aes=off"); got != "aes=off" {
		t.Fatalf("GODEBUG=cpu.aes=off = %q, want aes=off", got)
	}
	if got := run("cpu.all=off"); got != "aes=off" {
		t.Fatalf("GODEBUG=cpu.all=off = %q, want aes=off", got)
	}
	if got := run("cpu.aes=off,cpu.aes=on"); got != def {
		t.Fatalf("GODEBUG=cpu.aes=off,cpu.aes=on = %q, want %q", got, def)
	}
}
