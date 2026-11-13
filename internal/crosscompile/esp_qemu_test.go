package crosscompile

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestESPQEMUSelection(t *testing.T) {
	for _, tc := range []struct {
		command string
		args    []string
		arch    string
	}{
		{"qemu-system-riscv32", []string{"-machine", "esp32c3"}, "riscv32"},
		{"qemu-system-xtensa", []string{"-M", "esp32"}, "xtensa"},
		{"qemu-system-xtensa", []string{"-machine=esp32,usb=off"}, "xtensa"},
		{"qemu-system-riscv32", []string{"-machine", "virt,aclint=on"}, ""},
		{"qemu-system-riscv32", []string{"-machine", "esp32c3", "-M", "virt"}, ""},
		{"qemu-system-riscv32", []string{"-machine"}, ""},
		{"qemu-system-xtensa", nil, ""},
		{"qemu-system-arm", []string{"-machine", "esp32"}, ""},
		{"/custom path/qemu-system-xtensa", []string{"-machine", "esp32"}, ""},
	} {
		if got := espQEMUArchitecture(tc.command, tc.args); got != tc.arch {
			t.Errorf("%s %v: got %q, want %q", tc.command, tc.args, got, tc.arch)
		}
		if tc.arch == "" {
			if got, err := ResolveESPQEMU(tc.command, tc.args); err != nil || got != tc.command {
				t.Errorf("unrelated runner changed: %q, %v", got, err)
			}
		}
	}
	for _, goos := range []string{"darwin", "linux"} {
		for _, goarch := range []string{"amd64", "arm64"} {
			platform := getESPClangPlatform(goos, goarch)
			for _, arch := range []string{"riscv32", "xtensa"} {
				checksum, err := hex.DecodeString(espQEMUChecksums[platform][arch])
				if err != nil || len(checksum) != sha256.Size {
					t.Errorf("missing pinned checksum for %s/%s %s", goos, goarch, arch)
				}
			}
		}
	}
	for _, host := range [][2]string{{"linux", "386"}, {"linux", "riscv64"}, {"freebsd", "amd64"}, {"windows", "amd64"}} {
		command := "qemu-system-xtensa"
		if got, err := resolveESPQEMU(command, []string{"-machine", "esp32"}, host[0], host[1]); err != nil || got != command {
			t.Errorf("unsupported host %v changed PATH resolution: %q, %v", host, got, err)
		}
	}
}

func espQEMUArchive(t *testing.T, executable string) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range map[string]string{
		"qemu/bin/" + executable: "#!/bin/sh\nexit 0\n",
		"qemu/share/qemu/rom":    "rom data",
	} {
		if err := tw.WriteHeader(&tar.Header{Name: name, Size: int64(len(body)), Mode: 0755}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	archive := buf.Bytes()
	return archive, fmt.Sprintf("%x", sha256.Sum256(archive))
}

func TestResolveESPQEMU(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("automatic ESP QEMU installation supports macOS and Linux")
	}
	if _, err := exec.LookPath("xz"); err != nil {
		t.Skip("xz is required to create the pinned archive format")
	}
	archive, _ := espQEMUArchive(t, "qemu-system-riscv32")
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	compress := exec.Command("xz", "-c")
	compress.Stdin = gz
	xz, err := compress.Output()
	if err != nil {
		t.Fatal(err)
	}
	platform := getESPClangPlatform(runtime.GOOS, runtime.GOARCH)
	asset := "qemu-riscv32-softmmu-esp_develop_9.2.2_20250817-" + platform + ".tar.xz"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/"+asset {
			t.Errorf("requested wrong host payload: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(xz)
	}))
	defer server.Close()
	root := t.TempDir()
	bin := t.TempDir()
	installed := filepath.Join(bin, "qemu-system-riscv32")
	if err := os.WriteFile(installed, []byte("#!/bin/sh\necho 'virt Generic RISC-V machine'\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	oldCache, oldURL, oldChecksums := cacheRoot, espQEMUBaseURL, espQEMUChecksums
	cacheRoot, espQEMUBaseURL = func() string { return root }, server.URL
	espQEMUChecksums = map[string]map[string]string{platform: {"riscv32": fmt.Sprintf("%x", sha256.Sum256(xz))}}
	t.Cleanup(func() {
		cacheRoot, espQEMUBaseURL, espQEMUChecksums = oldCache, oldURL, oldChecksums
	})
	want := filepath.Join(root, "crosscompile", "esp-qemu-"+espQEMUVersion, platform, "riscv32", "bin", "qemu-system-riscv32")
	if got, err := ResolveESPQEMU("qemu-system-riscv32", []string{"-machine", "esp32c3"}); err != nil || got != want {
		t.Fatalf("resolved emulator = %q, %v; want %s", got, err, want)
	}
	// An installed ESP build takes precedence even when a cache already exists.
	server.Close()
	if err := os.WriteFile(installed, []byte("#!/bin/sh\necho 'esp32c3 Espressif ESP32-C3'\n"), 0755); err != nil {
		t.Fatal(err)
	}
	if got, err := ResolveESPQEMU("qemu-system-riscv32", []string{"-machine", "esp32c3"}); err != nil || got != installed {
		t.Fatalf("installed ESP QEMU was ignored: %q, %v", got, err)
	}
	if err := os.WriteFile(installed, []byte("#!/bin/sh\nexit 1\n"), 0755); err != nil {
		t.Fatal(err)
	}
	if got, err := ResolveESPQEMU("qemu-system-riscv32", []string{"-machine", "esp32c3"}); err != nil || got != want {
		t.Fatalf("failed PATH probe did not fall back to cache: %q, %v", got, err)
	}
}

func TestESPQEMUConcurrentInstallAndCache(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("automatic ESP QEMU installation is Unix-only")
	}
	const command = "qemu-system-riscv32"
	archive, checksum := espQEMUArchive(t, command)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = w.Write(archive)
	}))
	defer server.Close()
	dir := filepath.Join(t.TempDir(), "cache with spaces", "riscv32")
	// Recover a stale, incomplete install as well as its abandoned extraction.
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir+".extract", 0755); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			executable, err := installESPQEMU(dir, command, server.URL+"/qemu.tar.gz", checksum)
			if err != nil || executable != filepath.Join(dir, "bin", command) {
				t.Errorf("install = %q, %v", executable, err)
			}
		}()
	}
	wg.Wait()
	if n := requests.Load(); n != 1 {
		t.Fatalf("downloaded %d times, want 1", n)
	}
	if data, err := os.ReadFile(filepath.Join(dir, "share/qemu/rom")); err != nil || string(data) != "rom data" {
		t.Fatalf("ROM payload not retained: %q, %v", data, err)
	}
	server.Close()
	if _, err := installESPQEMU(dir, command, server.URL+"/qemu.tar.gz", checksum); err != nil {
		t.Fatalf("cached emulator required network access: %v", err)
	}
}

func TestESPQEMUFailedInstallCanRetry(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("automatic ESP QEMU installation is Unix-only")
	}
	const command = "qemu-system-xtensa"
	archive, checksum := espQEMUArchive(t, command)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(archive)
	}))
	defer server.Close()
	for _, tc := range []struct{ name, command, checksum, message string }{
		{"checksum", command, strings.Repeat("0", 64), "checksum mismatch"},
		{"missing executable", "qemu-system-riscv32", checksum, "does not contain executable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "qemu")
			if _, err := installESPQEMU(dir, tc.command, server.URL+"/qemu.tar.gz", tc.checksum); err == nil || !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("failed payload error = %v", err)
			}
			for _, path := range []string{dir, dir + ".extract", dir + ".extract.temp"} {
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Fatalf("failed install left %s: %v", path, err)
				}
			}
			if _, err := installESPQEMU(dir, command, server.URL+"/qemu.tar.gz", checksum); err != nil {
				t.Fatalf("retry failed: %v", err)
			}
		})
	}
}

func TestESPQEMUFilesystemFailures(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("requires Unix permissions enforced for a non-root user")
	}
	const command = "qemu-system-riscv32"
	archive, checksum := espQEMUArchive(t, command)
	for _, name := range []string{"unreadable cache", "invalid lock", "stale extraction", "incomplete cache"} {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "qemu")
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				_, _ = w.Write(archive)
			}))
			defer server.Close()
			blocked := dir
			wantRequests := int32(0)
			switch name {
			case "invalid lock":
				blocked = dir + ".lock"
			case "stale extraction":
				blocked = filepath.Join(dir+".extract", "readonly")
			case "incomplete cache":
				blocked = filepath.Join(dir, "readonly")
				wantRequests = 1
			}
			if err := os.MkdirAll(blocked, 0755); err != nil {
				t.Fatal(err)
			}
			if name != "invalid lock" {
				if err := os.WriteFile(filepath.Join(blocked, "retained"), []byte("data"), 0600); err != nil {
					t.Fatal(err)
				}
				mode := os.FileMode(0555)
				if name == "unreadable cache" {
					mode = 0
				}
				if err := os.Chmod(blocked, mode); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = os.Chmod(blocked, 0755) })
			}
			if _, err := installESPQEMU(dir, command, server.URL+"/qemu.tar.gz", checksum); err == nil {
				t.Fatal("installation ignored a filesystem failure")
			}
			if got := requests.Load(); got != wantRequests {
				t.Fatalf("requests = %d, want %d", got, wantRequests)
			}
		})
	}
}
