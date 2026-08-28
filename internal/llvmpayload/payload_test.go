package llvmpayload

import (
	"encoding/hex"
	"strings"
	"testing"
)

func testManifest(t *testing.T, llvmVersion, payloadVersion, compilerRTVersion string, wantMajor int) {
	t.Helper()
	manifest, err := ForLLVMVersion(llvmVersion)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.LLVMMajor() != wantMajor || manifest.Version() != payloadVersion {
		t.Fatalf("manifest identity = LLVM %d %s", manifest.LLVMMajor(), manifest.Version())
	}
	if manifest.CompilerRTVersion() != compilerRTVersion {
		t.Fatalf("compiler-rt version = %q, want %q", manifest.CompilerRTVersion(), compilerRTVersion)
	}
	platforms := manifest.Platforms()
	if len(platforms) != 5 {
		t.Fatalf("platform count = %d, want 5: %v", len(platforms), platforms)
	}
	for _, platform := range platforms {
		artifact, err := manifest.Artifact(platform)
		if err != nil {
			t.Fatal(err)
		}
		if artifact.Platform != platform {
			t.Errorf("artifact platform = %q, want %q", artifact.Platform, platform)
		}
		if !strings.HasSuffix(artifact.URL, "clang-esp-"+artifact.Version+"-"+platform+".tar.xz") {
			t.Errorf("artifact URL = %q", artifact.URL)
		}
		checksum, err := hex.DecodeString(artifact.SHA256)
		if err != nil || len(checksum) != 32 {
			t.Errorf("artifact checksum = %q, err %v", artifact.SHA256, err)
		}
	}
}

func TestLLVM21Manifest(t *testing.T) {
	testManifest(t, "LLVM 21.1.8", "21.1.3_20260816", "xtensa_release_21.1.3_20260408", 21)

	manifest, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	if manifest.LLVMMajor() != 21 || manifest.Version() != "21.1.3_20260816" {
		t.Fatalf("default manifest identity = LLVM %d %s", manifest.LLVMMajor(), manifest.Version())
	}
	windows, err := manifest.Artifact("x86_64-w64-mingw32")
	if err != nil {
		t.Fatal(err)
	}
	if windows.Version != "21.1.3_20260408" {
		t.Fatalf("Windows artifact version = %q", windows.Version)
	}
	if windows.URL != "https://github.com/espressif/llvm-project/releases/download/esp-21.1.3_20260408/clang-esp-21.1.3_20260408-x86_64-w64-mingw32.tar.xz" {
		t.Fatalf("Windows artifact URL = %q", windows.URL)
	}
	if windows.SHA256 != "415566ace6f47a9abc302b4ba79776d27668fd3f4e9c0d26861ec4f970323618" {
		t.Fatalf("Windows artifact checksum = %q", windows.SHA256)
	}
}

func TestPayloadErrors(t *testing.T) {
	if _, err := ForLLVMVersion("development"); err == nil {
		t.Fatal("invalid LLVM version accepted")
	}
	if _, err := ForLLVMVersion(strings.Repeat("9", 100) + ".1.0"); err == nil {
		t.Fatal("overflowing LLVM major accepted")
	}
	if _, err := ForMajor(19); err == nil {
		t.Fatal("unpublished LLVM major accepted")
	}
	manifest, err := ForMajor(21)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manifest.Artifact("arm-linux-gnueabihf"); err == nil {
		t.Fatal("unpublished platform accepted")
	}
}

func TestPlatformSuffix(t *testing.T) {
	tests := []struct {
		goos, goarch string
		want         string
		ok           bool
	}{
		{goos: "darwin", goarch: "amd64", want: "x86_64-apple-darwin", ok: true},
		{goos: "darwin", goarch: "arm64", want: "aarch64-apple-darwin", ok: true},
		{goos: "linux", goarch: "amd64", want: "x86_64-linux-gnu", ok: true},
		{goos: "linux", goarch: "arm64", want: "aarch64-linux-gnu", ok: true},
		{goos: "linux", goarch: "arm", ok: false},
		{goos: "windows", goarch: "amd64", want: "x86_64-w64-mingw32", ok: true},
		{goos: "windows", goarch: "arm64", want: "x86_64-w64-mingw32", ok: true},
	}
	for _, test := range tests {
		got, ok := PlatformSuffix(test.goos, test.goarch)
		if got != test.want || ok != test.ok {
			t.Errorf("PlatformSuffix(%q, %q) = %q, %v; want %q, %v", test.goos, test.goarch, got, ok, test.want, test.ok)
		}
	}
}
