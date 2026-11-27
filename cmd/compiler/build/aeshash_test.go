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
	// Use maphash.Comparable on a 384-byte string so the full length reaches
	// memhashAES (>128 B loop with n == 2) without maphash.Bytes's 128 B chunking.
	buf := make([]byte, 384)
	for i := range buf {
		buf[i] = byte(i*17 + 3)
	}
	base := maphash.Comparable(seed, string(buf))
	for _, pos := range []int{17, 199, 383} {
		buf[pos] ^= 1
		flipped := maphash.Comparable(seed, string(buf))
		buf[pos] ^= 1
		if flipped == base {
			panic("hash did not change after bit flip")
		}
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
