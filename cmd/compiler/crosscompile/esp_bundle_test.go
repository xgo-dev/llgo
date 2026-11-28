package crosscompile

import (
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
)

func bundledESPFixture(t *testing.T) (bundle, cached string, requests *atomic.Int32) {
	t.Helper()
	writeWasmTargetFixture(t, "", "")
	platform := getESPClangPlatform(runtime.GOOS, runtime.GOARCH)
	if platform == "" {
		t.Skip("no ESP toolchain for this host")
	}
	root := os.Getenv("LLGO_ROOT")
	if err := os.MkdirAll(filepath.Join(root, "LICENSES"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "LICENSES", espClangLicenseFile), []byte("LLVM license"), 0644); err != nil {
		t.Fatal(err)
	}
	archive := createTestTarXz(t, map[string]string{"esp-clang/bin/clang": "bundled compiler"})
	content, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	bundle = filepath.Join(root, "crosscompile", espPins.Payloads[platform].Asset)
	if err := os.MkdirAll(filepath.Dir(bundle), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bundle, content, 0644); err != nil {
		t.Fatal(err)
	}
	checksum, err := fileSHA256(bundle)
	if err != nil {
		t.Fatal(err)
	}
	oldCache, oldURL, oldChecksums := cacheRoot, espClangBaseUrl, espClangSHA256
	cache := t.TempDir()
	cacheRoot = func() string { return cache }
	espClangSHA256 = maps.Clone(espClangSHA256)
	espClangSHA256[platform] = checksum
	requests = new(atomic.Int32)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Error(w, "ESP network downloads forbidden", http.StatusForbidden)
	}))
	espClangBaseUrl = server.URL
	t.Cleanup(func() {
		cacheRoot, espClangBaseUrl, espClangSHA256 = oldCache, oldURL, oldChecksums
		server.Close()
	})
	return bundle, espClangCacheDir(platform), requests
}

func TestBundledESPClangInstallsWithoutNetwork(t *testing.T) {
	bundle, cached, requests := bundledESPFixture(t)
	// Native system-package builds keep using the system compiler.
	if root, err := getESPClangRoot(false); err != nil || root != "" {
		t.Fatalf("native compiler selection = %q, %v", root, err)
	}
	root, err := getESPClangRoot(true)
	if err != nil || root != cached {
		t.Fatalf("bundled toolchain = %q, %v; want %q", root, err, cached)
	}
	for name, expected := range map[string]string{"bin/clang": "bundled compiler", "LICENSE-LLVM.txt": "LLVM license"} {
		content, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || string(content) != expected {
			t.Fatalf("installed %s = %q, %v", name, content, err)
		}
	}
	// Cache reuse does not need the installation archive to remain accessible.
	if err := os.Remove(bundle); err != nil {
		t.Fatal(err)
	}
	if again, err := getESPClangRoot(true); err != nil || again != cached {
		t.Fatalf("cached toolchain = %q, %v", again, err)
	}
	if requests.Load() != 0 {
		t.Fatal("bundled installation attempted an ESP network download")
	}
}

func TestCorruptBundledESPClangDoesNotFallBackToDownload(t *testing.T) {
	_, cached, requests := bundledESPFixture(t)
	platform := getESPClangPlatform(runtime.GOOS, runtime.GOARCH)
	espClangSHA256[platform] = strings.Repeat("0", 64)
	root, err := getESPClangRoot(true)
	if root != "" || err == nil || !strings.Contains(err.Error(), "bundled ESP Clang checksum mismatch") {
		t.Fatalf("corrupt bundle = %q, %v", root, err)
	}
	for _, path := range []string{cached, cached + ".extract"} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("partial installation remains at %s: %v", path, err)
		}
	}
	if requests.Load() != 0 {
		t.Fatal("a corrupt local archive must fail without a network fallback")
	}
}

func TestBundledESPClangInstallFailures(t *testing.T) {
	for _, scenario := range []string{"missing archive", "invalid archive", "missing license"} {
		t.Run(scenario, func(t *testing.T) {
			bundle, cached, requests := bundledESPFixture(t)
			want := "read bundled ESP Clang archive"
			switch scenario {
			case "missing archive":
				if err := os.Remove(bundle); err != nil {
					t.Fatal(err)
				}
			case "invalid archive":
				if err := os.WriteFile(bundle, []byte("invalid archive"), 0644); err != nil {
					t.Fatal(err)
				}
				want = "extract bundled ESP Clang archive"
			case "missing license":
				if err := os.Remove(filepath.Join(os.Getenv("LLGO_ROOT"), "LICENSES", espClangLicenseFile)); err != nil {
					t.Fatal(err)
				}
				want = "read ESP Clang license"
			}
			checksum, _ := fileSHA256(bundle)
			err := checkInstallESPClangArchive(bundle, checksum, cached)
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("installation error = %v; want %q", err, want)
			}
			for _, path := range []string{cached, cached + ".extract"} {
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Fatalf("partial installation remains at %s: %v", path, err)
				}
			}
			if requests.Load() != 0 {
				t.Fatal("local installation used the network")
			}
		})
	}
}

func TestESPClangRejectsMissingAsset(t *testing.T) {
	writeWasmTargetFixture(t, "", "")
	platform := getESPClangPlatform(runtime.GOOS, runtime.GOARCH)
	if platform == "" {
		t.Skip("no ESP toolchain for this host")
	}
	original := espPins.Payloads
	espPins.Payloads = maps.Clone(original)
	delete(espPins.Payloads, platform)
	t.Cleanup(func() { espPins.Payloads = original })
	root, err := getESPClangRoot(true)
	if root != "" || err == nil || err.Error() != "missing ESP Clang asset for "+platform {
		t.Fatalf("missing asset = %q, %v", root, err)
	}
}
