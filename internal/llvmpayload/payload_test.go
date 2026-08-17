package llvmpayload

import (
	"encoding/hex"
	"strings"
	"testing"
)

func testManifest(t *testing.T, llvmVersion, payloadVersion string, wantMajor int) {
	t.Helper()
	manifest, err := ForLLVMVersion(llvmVersion)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.LLVMMajor() != wantMajor || manifest.Version() != payloadVersion {
		t.Fatalf("manifest identity = LLVM %d %s", manifest.LLVMMajor(), manifest.Version())
	}
	platforms := manifest.Platforms()
	if len(platforms) != 4 {
		t.Fatalf("platform count = %d, want 4: %v", len(platforms), platforms)
	}
	for _, platform := range platforms {
		artifact, err := manifest.Artifact(platform)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(artifact.URL, "clang-esp-"+manifest.Version()+"-"+platform+".tar.xz") {
			t.Errorf("artifact URL = %q", artifact.URL)
		}
		checksum, err := hex.DecodeString(artifact.SHA256)
		if err != nil || len(checksum) != 32 {
			t.Errorf("artifact checksum = %q, err %v", artifact.SHA256, err)
		}
	}
}

func TestLLVM19Manifest(t *testing.T) {
	testManifest(t, "LLVM 19.1.7", "19.1.2_20250905-3", 19)
}

func TestLLVM21Manifest(t *testing.T) {
	testManifest(t, "LLVM 21.1.8", "21.1.3_20260816", 21)
}

func TestPayloadErrors(t *testing.T) {
	if _, err := ForLLVMVersion("development"); err == nil {
		t.Fatal("invalid LLVM version accepted")
	}
	if _, err := ForMajor(20); err == nil {
		t.Fatal("unpublished LLVM major accepted")
	}
	manifest, err := ForMajor(19)
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
		{goos: "windows", goarch: "amd64", ok: false},
	}
	for _, test := range tests {
		got, ok := PlatformSuffix(test.goos, test.goarch)
		if got != test.want || ok != test.ok {
			t.Errorf("PlatformSuffix(%q, %q) = %q, %v; want %q, %v", test.goos, test.goarch, got, ok, test.want, test.ok)
		}
	}
}
