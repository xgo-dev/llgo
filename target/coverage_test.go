package target

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveSpecAllArchitecturesAndLoaderDirs(t *testing.T) {
	cases := []Spec{
		{GOOS: "linux", GOARCH: "386", GO386: "softfloat"},
		{GOOS: "linux", GOARCH: "386", GO386: "sse2"},
		{GOOS: "linux", GOARCH: "amd64", GOAMD64: "v3"},
		{GOOS: "linux", GOARCH: "arm", GOARM: "5,softfloat"},
		{GOOS: "linux", GOARCH: "arm", GOARM: "5,hardfloat"},
		{GOOS: "linux", GOARCH: "arm", GOARM: "6"},
		{GOOS: "linux", GOARCH: "arm", GOARM: "7,softfloat"},
		{GOOS: "linux", GOARCH: "arm64", GOARM64: "v9.0,lse,crypto"},
		{GOOS: "darwin", GOARCH: "arm64", GOARM64: "v8.2"},
		{GOOS: "linux", GOARCH: "riscv64"},
		{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32", LLVMTarget: "wasm32-unknown-wasip1"},
		{GOOS: "windows", GOARCH: "amd64"},
		{GOOS: "linux", GOARCH: "arm", Name: "cortex-m3", LLVMTarget: "thumbv7m-none-eabi"},
		{},
	}
	for _, tc := range cases {
		resolved := ResolveSpec(tc)
		if resolved.Triple == "" || resolved.PhysicalTriple == "" || resolved.WordSize == 0 {
			t.Fatalf("ResolveSpec(%+v) produced incomplete spec: %+v", tc, resolved)
		}
	}

	if got := Triple("windows", "amd64"); !IsWindowsTriple(got) {
		t.Fatalf("Triple(windows, amd64) = %q, want windows triple", got)
	}
	if IsWindowsTriple("x86_64-unknown-linux-gnu") {
		t.Fatal("linux triple reported as windows")
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "alpha.json"), []byte(`{"llvm-target":"thumbv7m-none-eabi","cpu":"cortex-m3","goos":"linux","goarch":"arm"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "beta.json"), []byte(`{"inherits":["alpha"],"features":"+dsp"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("ignore"), 0o644); err != nil {
		t.Fatal(err)
	}
	loader := NewLoader(dir)
	if loader.GetTargetsDir() != dir {
		t.Fatalf("GetTargetsDir() = %q, want %q", loader.GetTargetsDir(), dir)
	}
	names, err := loader.ListTargets()
	if err != nil {
		t.Fatalf("ListTargets() error: %v", err)
	}
	if strings.Join(names, ",") != "alpha,beta" {
		t.Fatalf("ListTargets() = %v, want [alpha beta]", names)
	}
	cfg, err := loader.Load("beta")
	if err != nil {
		t.Fatalf("Load(beta) error: %v", err)
	}
	spec := cfg.ToSpec()
	if spec.PhysicalTriple != "thumbv7m-none-eabi" {
		t.Fatalf("beta PhysicalTriple = %q", spec.PhysicalTriple)
	}
}
