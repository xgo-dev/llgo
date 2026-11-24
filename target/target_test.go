package target

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveSpecHostedTargets(t *testing.T) {
	amd64 := ResolveSpec(Spec{GOOS: "linux", GOARCH: "amd64", GOAMD64: "v3"})
	if amd64.Triple != "x86_64-unknown-linux-gnu" || amd64.CPU != "x86-64-v3" || amd64.WordSize != 8 || amd64.PtrSize != 8 {
		t.Fatalf("linux/amd64 spec = %+v", amd64)
	}
	darwinArm := ResolveSpec(Spec{GOOS: "darwin", GOARCH: "arm64"})
	if !IsDarwinTriple(darwinArm.Triple) || darwinArm.Triple != "arm64-apple-macosx" || strings.Contains(darwinArm.Features, "-fmv") {
		t.Fatalf("darwin/arm64 spec = %+v", darwinArm)
	}
	win386 := ResolveSpec(Spec{GOOS: "windows", GOARCH: "386"})
	if !IsWindowsTriple(win386.Triple) || win386.Triple != "i686-pc-windows-msvc" || win386.WordSize != 4 || win386.HostAlign64 != 8 {
		t.Fatalf("windows/386 spec = %+v", win386)
	}
	w32 := ResolveSpec(Spec{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32", WasmProvider: "wasi"})
	if !w32.UsesWideGoStorage() || w32.WordSize != 8 || w32.PtrSize != 4 {
		t.Fatalf("wasip1/w32 spec = %+v", w32)
	}
	arm7 := ResolveSpec(Spec{GOOS: "linux", GOARCH: "arm", GOARM: "7"})
	if arm7.WordSize != 4 || arm7.PtrSize != 4 || !arm7.HostUsesNativeGo || arm7.HostAlign64 != 8 {
		t.Fatalf("linux/arm spec = %+v", arm7)
	}
}

func TestLoaderSecurityAndInheritance(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name+".json"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("base", `{"goos":"linux","goarch":"arm","cpu":"cortex-m0","build-tags":["a"]}`)
	write("child", `{"inherits":["base"],"build-tags":["b"]}`)
	write("cyc-a", `{"inherits":["cyc-b"]}`)
	write("cyc-b", `{"inherits":["cyc-a"]}`)

	loader := NewLoader(dir)
	cfg, err := loader.Load("child")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.GOOS != "linux" || cfg.GOARCH != "arm" || cfg.CPU != "cortex-m0" || len(cfg.BuildTags) != 2 {
		t.Fatalf("resolved child = %+v", cfg)
	}
	if spec := cfg.ToSpec(); spec.CPU != "cortex-m0" || spec.WordSize != 4 {
		t.Fatalf("child.ToSpec() = %+v", spec)
	}
	for _, bad := range []string{"", ".", "..", "../escape", `a\b`, "/etc/passwd"} {
		if _, err := loader.Load(bad); err == nil {
			t.Fatalf("Load(%q) unexpectedly succeeded", bad)
		}
	}
	if _, err := loader.Load("cyc-a"); err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("Load(cyc-a) = %v, want cycle error", err)
	}
}
