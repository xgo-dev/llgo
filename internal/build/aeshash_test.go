//go:build (darwin || linux || windows) && (amd64 || arm64 || 386)

package build

import (
	stdctx "context"
	gobuild "go/build"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	llssa "github.com/xgo-dev/llgo/ssa"
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

func TestRuntimeAESKeySchedAlignmentAndTargetTags(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	conf := NewDefaultConf(ModeGen)
	pkgs, err := Build(Invocation{Args: []string{llssa.PkgRuntime}, Config: conf})
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 || pkgs[0].LPkg == nil {
		t.Fatal("missing compiled runtime package")
	}
	defer pkgs[0].LPkg.Prog.Dispose()
	sched := pkgs[0].LPkg.Module().NamedGlobal(llssa.PkgRuntime + ".aeskeysched")
	if sched.IsNil() {
		t.Fatal("missing runtime.aeskeysched global")
	}
	if got := sched.Alignment(); got < 16 {
		t.Fatalf("runtime.aeskeysched alignment = %d, want >= 16", got)
	}

	rtDir := filepath.Join(root, "runtime", "internal", "runtime")
	for _, tc := range []struct {
		name     string
		goos     string
		goarch   string
		tags     []string
		wantFile string
		denyFile string
	}{
		{
			name:     "android-arm64-aeshash",
			goos:     "android",
			goarch:   "arm64",
			tags:     []string{"llgo"},
			wantFile: "aeshash_cpu_linux_arm64.go",
			denyFile: "aeshash_cpu_other.go",
		},
		{
			name:     "windows-baremetal-maxprocs",
			goos:     "windows",
			goarch:   "amd64",
			tags:     []string{"llgo", "baremetal"},
			wantFile: "maxprocs_stub.go",
			denyFile: "maxprocs_windows.go",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := gobuild.Default
			ctx.GOOS = tc.goos
			ctx.GOARCH = tc.goarch
			ctx.BuildTags = tc.tags
			want, err := ctx.MatchFile(rtDir, tc.wantFile)
			if err != nil {
				t.Fatal(err)
			}
			if !want {
				t.Fatalf("%s did not match for %s/%s tags=%v", tc.wantFile, tc.goos, tc.goarch, tc.tags)
			}
			deny, err := ctx.MatchFile(rtDir, tc.denyFile)
			if err != nil {
				t.Fatal(err)
			}
			if deny {
				t.Fatalf("%s unexpectedly matched for %s/%s tags=%v", tc.denyFile, tc.goos, tc.goarch, tc.tags)
			}
		})
	}
}
