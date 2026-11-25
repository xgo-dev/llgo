package target

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveSpecHostedTargets(t *testing.T) {
	amd64 := ResolveSpec(Spec{GOOS: "linux", GOARCH: "amd64", GOAMD64: "v3"})
	if amd64.Triple != "x86_64-unknown-linux-gnu" || amd64.PhysicalTriple != amd64.Triple || amd64.ArchLevel != "v3" || amd64.CPU != "x86-64-v3" || amd64.WordSize != 8 || amd64.PtrSize != 8 {
		t.Fatalf("linux/amd64 spec = %+v", amd64)
	}
	namedAMD64 := ResolveSpec(Spec{Name: "custom", GOOS: "linux", GOARCH: "amd64", GOAMD64: "v4"})
	if namedAMD64.ArchLevel != "v1" || namedAMD64.CPU != "x86-64" {
		t.Fatalf("named target should ignore GOAMD64, got %+v", namedAMD64)
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
	write("base", `{"goos":"linux","goarch":"arm","llvm-target":"thumbv7m-none-eabi","cpu":"cortex-m0","build-tags":["a"]}`)
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
	if spec := cfg.ToSpec(); spec.CPU != "cortex-m0" || spec.WordSize != 4 ||
		spec.Triple != "armv7-unknown-linux-gnueabihf" || spec.PhysicalTriple != "thumbv7m-none-eabi" {
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

func TestResolve386(t *testing.T) {
	for _, test := range []struct {
		value   string
		want    string
		wantErr bool
	}{
		{value: "", want: "sse2"},
		{value: "sse2", want: "sse2"},
		{value: "softfloat", want: "softfloat"},
		{value: "387", want: "sse2", wantErr: true},
		{value: "invalid", want: "sse2", wantErr: true},
	} {
		got, err := Resolve386(test.value)
		if got != test.want || (err != nil) != test.wantErr {
			t.Errorf("Resolve386(%q) = %q, %v; want %q, error=%v", test.value, got, err, test.want, test.wantErr)
		}
	}
	if _, err := Resolve386("387"); err == nil || err.Error() != "unsupported setting GO386=387. Consider using GO386=softfloat instead." {
		t.Fatalf("Resolve386(387) error = %v", err)
	}
}

func TestResolveAMD64(t *testing.T) {
	for _, test := range []struct {
		value   string
		want    string
		wantErr bool
	}{
		{value: "", want: "v1"},
		{value: "v1", want: "v1"},
		{value: "v2", want: "v2"},
		{value: "v3", want: "v3"},
		{value: "v4", want: "v4"},
		{value: "v5", want: "v1", wantErr: true},
	} {
		got, err := ResolveAMD64(test.value)
		if got != test.want || (err != nil) != test.wantErr {
			t.Errorf("ResolveAMD64(%q) = %q, %v; want %q, error=%v", test.value, got, err, test.want, test.wantErr)
		}
	}
}

func TestParseARM(t *testing.T) {
	for _, test := range []struct {
		value    string
		want     string
		wantSoft bool
		wantErr  bool
	}{
		{value: "", want: "7,hardfloat"},
		{value: "5", want: "5,softfloat", wantSoft: true},
		{value: "5,softfloat", want: "5,softfloat", wantSoft: true},
		{value: "5,hardfloat", want: "5,hardfloat"},
		{value: "6", want: "6,hardfloat"},
		{value: "6,softfloat", want: "6,softfloat", wantSoft: true},
		{value: "7,hardfloat", want: "7,hardfloat"},
		{value: "7,hardfloat,softfloat", want: "7,softfloat", wantSoft: true, wantErr: true},
		{value: "4", want: "7,hardfloat", wantErr: true},
		{value: "7,softfloat,hardfloat", want: "7,hardfloat", wantErr: true},
	} {
		got, err := ParseARM(test.value)
		if got.String() != test.want || got.SoftFloat != test.wantSoft || (err != nil) != test.wantErr {
			t.Errorf("ParseARM(%q) = %q (soft=%v), %v; want %q (soft=%v), error=%v", test.value, got.String(), got.SoftFloat, err, test.want, test.wantSoft, test.wantErr)
		}
	}
}

func TestParseARM64(t *testing.T) {
	for _, test := range []struct {
		value   string
		want    string
		wantErr bool
	}{
		{value: "", want: "v8.0"},
		{value: "v8.0", want: "v8.0"},
		{value: "v8.0,lse", want: "v8.0,lse"},
		{value: "v8.0,crypto", want: "v8.0,crypto"},
		{value: "v8.0,crypto,lse", want: "v8.0,lse,crypto"},
		{value: "v8.0,lse,crypto", want: "v8.0,lse,crypto"},
		{value: "v8.0,lse,lse,crypto", want: "v8.0,lse,crypto"},
		{value: "v8.1", want: "v8.1,lse"},
		{value: "v8.9", want: "v8.9,lse"},
		{value: "v9.0", want: "v9.0,lse"},
		{value: "v9.5,crypto", want: "v9.5,lse,crypto"},
		{value: "v9.6", want: "v8.0", wantErr: true},
		{value: "8.0", want: "v8.0", wantErr: true},
	} {
		got, err := ParseARM64(test.value)
		if got.String() != test.want || (err != nil) != test.wantErr {
			t.Errorf("ParseARM64(%q) = %q, %v; want %q, error=%v", test.value, got.String(), err, test.want, test.wantErr)
		}
	}
}

func TestIsDarwinTriple(t *testing.T) {
	for _, triple := range []string{"arm64-apple-macosx", "aarch64-apple-ios", "aarch64-unknown-darwin", "AARCH64-APPLE-DARWIN", "aarch64-unknown-macosx", "aarch64-unknown-macosx14.0.0", "AARCH64-UNKNOWN-MACOSX"} {
		if !IsDarwinTriple(triple) {
			t.Errorf("missed Apple target %q", triple)
		}
	}
	for _, triple := range []string{"", "aarch64-linux-gnu", "aarch64-pc-windows-gnu", "aarch64-linux-android"} {
		if IsDarwinTriple(triple) {
			t.Errorf("misclassified target %q", triple)
		}
	}
}

func TestTripleAndTripleWithGOARM(t *testing.T) {
	for _, tc := range []struct {
		goos, goarch, want string
	}{
		{"wasip1", "wasm", "wasm32-unknown-wasip1"},
		{"linux", "amd64", "x86_64-unknown-linux-gnu"},
		{"linux", "386", "i386-unknown-linux-gnu"},
		{"linux", "arm64", "aarch64-unknown-linux-gnu"},
		{"linux", "arm", "armv7-unknown-linux-gnueabihf"},
		{"darwin", "amd64", "x86_64-apple-macosx"},
		{"darwin", "arm64", "arm64-apple-macosx"},
		{"windows", "amd64", "x86_64-pc-windows-msvc"},
		{"windows", "386", "i686-pc-windows-msvc"},
		{"windows", "arm64", "aarch64-pc-windows-msvc"},
		{"js", "wasm", "wasm32-unknown-js"},
	} {
		if got := Triple(tc.goos, tc.goarch); got != tc.want {
			t.Errorf("Triple(%q, %q) = %q, want %q", tc.goos, tc.goarch, got, tc.want)
		}
	}
	for _, test := range []struct {
		goarm string
		want  string
	}{
		{"5", "armv5-unknown-linux-gnueabi"},
		{"5,hardfloat", "armv5-unknown-linux-gnueabihf"},
		{"6", "armv6-unknown-linux-gnueabihf"},
		{"6,softfloat", "armv6-unknown-linux-gnueabi"},
		{"7", "armv7-unknown-linux-gnueabihf"},
		{"", "armv7-unknown-linux-gnueabihf"},
	} {
		if got := TripleWithGOARM("linux", "arm", test.goarm); got != test.want {
			t.Errorf("TripleWithGOARM(linux, arm, %q) = %q, want %q", test.goarm, got, test.want)
		}
	}
}
