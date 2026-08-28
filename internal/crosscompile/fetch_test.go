//go:build !llgo
// +build !llgo

package crosscompile

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xgo-dev/llgo/internal/llvmpayload"
)

// Helper function to create a test tar.gz archive
func createTestTarGz(t *testing.T, files map[string]string) string {
	tempFile, err := os.CreateTemp("", "test*.tar.gz")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer tempFile.Close()

	gzw := gzip.NewWriter(tempFile)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("Failed to write tar header: %v", err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatalf("Failed to write tar content: %v", err)
		}
	}

	return tempFile.Name()
}

func createTestTarXz(t *testing.T, files map[string]string) string {
	t.Helper()
	_, xzErr := exec.LookPath("xz")
	if runtime.GOOS == "windows" && xzErr != nil {
		// Windows CI provides xz through MSYS2. Windows 11's bundled bsdtar is
		// a fallback for local development VMs that do not install xz separately.
		sourceDir := t.TempDir()
		for name, content := range files {
			file := filepath.Join(sourceDir, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		xzFile := filepath.Join(t.TempDir(), "test.tar.xz")
		tarCommand := filepath.Join(os.Getenv("SystemRoot"), "System32", "tar.exe")
		if output, err := exec.Command(tarCommand, "-cJf", xzFile, "-C", sourceDir, ".").CombinedOutput(); err != nil {
			t.Fatalf("compress test tar.xz: %v: %s", err, strings.TrimSpace(string(output)))
		}
		return xzFile
	}

	tarFile, err := os.CreateTemp("", "test*.tar")
	if err != nil {
		t.Fatal(err)
	}
	tw := tar.NewWriter(tarFile)
	for name, content := range files {
		hdr := &tar.Header{Name: name, Mode: 0o644, Size: int64(len(content))}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := tarFile.Close(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(tarFile.Name()) })

	compressed, err := exec.Command("xz", "-c", tarFile.Name()).Output()
	if err != nil {
		t.Fatalf("compress test tar.xz: %v", err)
	}
	xzFile := tarFile.Name() + ".xz"
	if err := os.WriteFile(xzFile, compressed, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(xzFile) })
	return xzFile
}

func TestWindowsArchiveTools(t *testing.T) {
	touch := func(t *testing.T, name string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, nil, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("7-Zip", func(t *testing.T) {
		root := t.TempDir()
		sevenZip := filepath.Join(root, "7-Zip", "7z.exe")
		touch(t, sevenZip)
		if got := windowsSevenZip(root); got != sevenZip {
			t.Fatalf("windowsSevenZip() = %q, want %q", got, sevenZip)
		}
	})

	t.Run("7-ZipFromPath", func(t *testing.T) {
		binDir := t.TempDir()
		sevenZip := filepath.Join(binDir, "7z.exe")
		touch(t, sevenZip)
		t.Setenv("PATH", binDir)
		if got := windowsSevenZip(""); got != sevenZip {
			t.Fatalf("windowsSevenZip() = %q, want %q", got, sevenZip)
		}
	})

	t.Run("NativeFallback", func(t *testing.T) {
		systemRoot := t.TempDir()
		nativeTar := filepath.Join(systemRoot, "System32", "tar.exe")
		touch(t, nativeTar)

		if got := windowsTarCommand(systemRoot); got != nativeTar {
			t.Fatalf("windowsTarCommand() = %q, want %q", got, nativeTar)
		}
	})

	t.Run("PathFallback", func(t *testing.T) {
		if got := windowsTarCommand(""); got != "tar" {
			t.Fatalf("windowsTarCommand() = %q, want tar", got)
		}
	})
}

func TestExtractTarXzWith7ZipCommand(t *testing.T) {
	if mode := os.Getenv("LLGO_7ZIP_HELPER"); mode != "" {
		switch mode {
		case "decompress":
			if os.Getenv("LLGO_7ZIP_FAIL") == mode {
				os.Exit(2)
			}
			_, _ = os.Stdout.WriteString("streamed tar payload")
		case "extract":
			payload, err := io.ReadAll(os.Stdin)
			if err != nil {
				os.Exit(3)
			}
			if err := os.WriteFile(os.Getenv("LLGO_7ZIP_OUTPUT"), payload, 0o644); err != nil {
				os.Exit(4)
			}
			if os.Getenv("LLGO_7ZIP_FAIL") == mode {
				os.Exit(5)
			}
		default:
			os.Exit(6)
		}
		os.Exit(0)
	}

	output := filepath.Join(t.TempDir(), "payload")
	t.Setenv("LLGO_7ZIP_OUTPUT", output)
	command := func(_ string, args ...string) *exec.Cmd {
		mode := "extract"
		if slices.Contains(args, "-so") {
			mode = "decompress"
		}
		cmd := exec.Command(os.Args[0], "-test.run=^TestExtractTarXzWith7ZipCommand$")
		cmd.Env = append(os.Environ(), "LLGO_7ZIP_HELPER="+mode)
		return cmd
	}

	if err := extractTarXzWith7ZipCommand("7z", "input.tar.xz", "output", command); err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(payload), "streamed tar payload"; got != want {
		t.Fatalf("streamed payload = %q, want %q", got, want)
	}

	for _, test := range []struct {
		mode string
		want string
	}{
		{mode: "decompress", want: "7-Zip xz decompression"},
		{mode: "extract", want: "7-Zip tar extraction"},
	} {
		t.Run(test.mode+" failure", func(t *testing.T) {
			t.Setenv("LLGO_7ZIP_FAIL", test.mode)
			if err := extractTarXzWith7ZipCommand("7z", "input.tar.xz", "output", command); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}

	t.Run("extract start failure", func(t *testing.T) {
		calls := 0
		factory := func(_ string, _ ...string) *exec.Cmd {
			calls++
			if calls == 2 {
				return exec.Command(filepath.Join(t.TempDir(), "missing"))
			}
			return command("7z", "-so")
		}
		if err := extractTarXzWith7ZipCommand("7z", "input.tar.xz", "output", factory); err == nil || !strings.Contains(err.Error(), "start 7-Zip tar extraction") {
			t.Fatalf("error = %v, want extract start failure", err)
		}
	})

	t.Run("decompress start failure", func(t *testing.T) {
		calls := 0
		factory := func(_ string, _ ...string) *exec.Cmd {
			calls++
			if calls == 1 {
				return exec.Command(filepath.Join(t.TempDir(), "missing"))
			}
			return command("7z", "-si")
		}
		if err := extractTarXzWith7ZipCommand("7z", "input.tar.xz", "output", factory); err == nil || !strings.Contains(err.Error(), "start 7-Zip xz decompression") {
			t.Fatalf("error = %v, want decompressor start failure", err)
		}
	})
}

func TestExtractTarXzForWindowsUses7Zip(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the test helper is a POSIX shell script")
	}
	programFiles := t.TempDir()
	sevenZip := filepath.Join(programFiles, "7-Zip", "7z.exe")
	if err := os.MkdirAll(filepath.Dir(sevenZip), 0o755); err != nil {
		t.Fatal(err)
	}
	script := `#!/bin/sh
if [ "$2" = "-so" ]; then
  printf 'streamed tar payload'
else
  cat > "$LLGO_7ZIP_OUTPUT"
fi
`
	if err := os.WriteFile(sevenZip, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "payload")
	t.Setenv("LLGO_7ZIP_OUTPUT", output)
	if err := extractTarXzForGOOS("windows", programFiles, "", "input.tar.xz", "output"); err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(payload), "streamed tar payload"; got != want {
		t.Fatalf("streamed payload = %q, want %q", got, want)
	}
}

// Helper function to create a test HTTP server
func createTestServer(t *testing.T, files map[string]string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if content, exists := files[path]; exists {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write([]byte(content))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestAcquireAndReleaseLock(t *testing.T) {
	if err := releaseLock(nil); err != nil {
		t.Fatalf("releaseLock(nil) = %v, want nil", err)
	}

	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "test.lock")

	// Test acquiring lock
	lockFile, err := acquireLock(lockPath)
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}

	// Check lock file exists
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Error("Lock file should exist")
	}

	// Test releasing lock
	if err := releaseLock(lockFile); err != nil {
		t.Errorf("Failed to release lock: %v", err)
	}

	// The lock file remains so every caller continues to lock the same file.
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("Lock file should remain after release: %v", err)
	}

	// A retained lock file can be acquired again.
	lockFile, err = acquireLock(lockPath)
	if err != nil {
		t.Fatalf("Failed to reacquire lock: %v", err)
	}
	if err := releaseLock(lockFile); err != nil {
		t.Errorf("Failed to release reacquired lock: %v", err)
	}

	// A closed handle exercises the platform unlock failure path and verifies
	// that releaseLock preserves enough context for callers to diagnose it.
	closedLock, err := acquireLock(filepath.Join(tempDir, "closed.lock"))
	if err != nil {
		t.Fatalf("Failed to acquire lock for error test: %v", err)
	}
	if err := closedLock.Close(); err != nil {
		t.Fatalf("Failed to close lock for error test: %v", err)
	}
	if err := releaseLock(closedLock); err == nil {
		t.Fatal("Expected release of a closed lock to fail")
	} else if !strings.Contains(err.Error(), "failed to release lock") {
		t.Fatalf("Unexpected closed lock release error: %v", err)
	}
}

func TestAcquireAndReleaseLockErrors(t *testing.T) {
	wantErr := errors.New("injected lock failure")
	var opened *os.File
	got, err := acquireLockWith(filepath.Join(t.TempDir(), "failed.lock"), func(file *os.File) error {
		opened = file
		return wantErr
	})
	if got != nil || !errors.Is(err, wantErr) {
		t.Fatalf("acquireLockWith = (%v, %v), want (nil, %v)", got, err, wantErr)
	}
	if opened == nil {
		t.Fatal("lock callback did not receive the opened file")
	}
	if err := opened.Close(); err == nil {
		t.Fatal("failed lock file remained open")
	}
}

func TestLockReleaseError(t *testing.T) {
	unlockErr := errors.New("unlock")
	closeErr := errors.New("close")
	for _, test := range []struct {
		name      string
		unlockErr error
		closeErr  error
		want      string
	}{
		{name: "success"},
		{name: "unlock", unlockErr: unlockErr, want: "failed to release lock: unlock"},
		{name: "close", closeErr: closeErr, want: "failed to close lock file: close"},
		{name: "unlock takes precedence", unlockErr: unlockErr, closeErr: closeErr, want: "failed to release lock: unlock"},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := lockReleaseError(test.unlockErr, test.closeErr)
			if test.want == "" {
				if err != nil {
					t.Fatalf("lockReleaseError() = %v, want nil", err)
				}
				return
			}
			if err == nil || err.Error() != test.want {
				t.Fatalf("lockReleaseError() = %v, want %q", err, test.want)
			}
		})
	}
}

func TestAcquireLockConcurrency(t *testing.T) {
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "concurrent.lock")

	var wg sync.WaitGroup
	var results []int
	var resultsMu sync.Mutex
	var active atomic.Int32

	// Start multiple goroutines trying to acquire the same lock
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			lockFile, err := acquireLock(lockPath)
			if err != nil {
				t.Errorf("Goroutine %d failed to acquire lock: %v", id, err)
				return
			}

			if n := active.Add(1); n != 1 {
				t.Errorf("Goroutine %d entered an occupied critical section (%d active)", id, n)
			}

			// Hold the lock for a short time.
			resultsMu.Lock()
			results = append(results, id)
			resultsMu.Unlock()

			time.Sleep(10 * time.Millisecond)
			active.Add(-1)

			if err := releaseLock(lockFile); err != nil {
				t.Errorf("Goroutine %d failed to release lock: %v", id, err)
			}
		}(i)
	}

	wg.Wait()

	// All goroutines should have successfully acquired and released the lock
	if len(results) != 5 {
		t.Errorf("Expected 5 successful lock acquisitions, got %d", len(results))
	}
}

func TestDownloadFile(t *testing.T) {
	// Create test server
	testContent := "test file content"
	server := createTestServer(t, map[string]string{
		"test.txt": testContent,
	})
	defer server.Close()

	tempDir := t.TempDir()
	targetFile := filepath.Join(tempDir, "downloaded.txt")

	// Test successful download
	err := downloadFile(server.URL+"/test.txt", targetFile)
	if err != nil {
		t.Fatalf("Failed to download file: %v", err)
	}

	// Check file content
	content, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("Expected content %q, got %q", testContent, string(content))
	}

	// Test download failure (404)
	err = downloadFile(server.URL+"/nonexistent.txt", targetFile)
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestExtractTarGz(t *testing.T) {
	// Create test archive
	files := map[string]string{
		"test-dir/file1.txt": "content of file1",
		"test-dir/file2.txt": "content of file2",
		"file3.txt":          "content of file3",
	}

	archivePath := createTestTarGz(t, files)

	// Extract to temp directory
	tempDir := t.TempDir()
	err := extractTarGz(archivePath, tempDir)
	if err != nil {
		t.Fatalf("Failed to extract tar.gz: %v", err)
	}

	// Check extracted files
	for name, expectedContent := range files {
		filePath := filepath.Join(tempDir, name)
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("Failed to read extracted file %s: %v", name, err)
			continue
		}
		if string(content) != expectedContent {
			t.Errorf("File %s: expected content %q, got %q", name, expectedContent, string(content))
		}
	}
}

func TestExtractTarGzPathTraversal(t *testing.T) {
	// Create a malicious archive with path traversal
	tempFile, err := os.CreateTemp("", "malicious*.tar.gz")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer tempFile.Close()
	defer os.Remove(tempFile.Name())

	gzw := gzip.NewWriter(tempFile)
	tw := tar.NewWriter(gzw)

	// Add a file with path traversal attack
	hdr := &tar.Header{
		Name:     "../../../etc/passwd",
		Mode:     0644,
		Size:     5,
		Typeflag: tar.TypeReg,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("Failed to write tar header: %v", err)
	}
	if _, err := tw.Write([]byte("pwned")); err != nil {
		t.Fatalf("Failed to write tar content: %v", err)
	}

	// Close writers to flush all data
	if err := tw.Close(); err != nil {
		t.Fatalf("Failed to close tar writer: %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("Failed to close gzip writer: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	tempDir := t.TempDir()
	err = extractTarGz(tempFile.Name(), tempDir)
	if err == nil {
		t.Error("Expected error for path traversal attack, got nil")
	}
	if !strings.Contains(err.Error(), "illegal file path") {
		t.Errorf("Expected 'illegal file path' error, got: %v", err)
	}
}

func TestDownloadAndExtractArchive(t *testing.T) {
	// Create test archive
	files := map[string]string{
		"test-app/bin/app":    "#!/bin/bash\necho hello",
		"test-app/lib/lib.so": "fake library content",
		"test-app/README":     "This is a test application",
	}

	archivePath := createTestTarGz(t, files)
	defer os.Remove(archivePath)

	// Create test server to serve the archive
	archiveContent, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("Failed to read test archive: %v", err)
	}

	server := createTestServer(t, map[string]string{
		"test-app.tar.gz": string(archiveContent),
	})
	defer server.Close()

	// Test download and extract
	tempDir := t.TempDir()
	destDir := filepath.Join(tempDir, "extracted")

	err = downloadAndExtractArchive(server.URL+"/test-app.tar.gz", destDir, "Test App")
	if err != nil {
		t.Fatalf("Failed to download and extract: %v", err)
	}

	// Check extracted files
	for name, expectedContent := range files {
		filePath := filepath.Join(destDir, name)
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("Failed to read extracted file %s: %v", name, err)
			continue
		}
		if string(content) != expectedContent {
			t.Errorf("File %s: expected content %q, got %q", name, expectedContent, string(content))
		}
	}
}

func TestDownloadAndExtractArchiveUnsupportedFormat(t *testing.T) {
	server := createTestServer(t, map[string]string{
		"test.7z": "fake zip content",
	})
	defer server.Close()

	tempDir := t.TempDir()
	destDir := filepath.Join(tempDir, "extracted")

	err := downloadAndExtractArchive(server.URL+"/test.7z", destDir, "Test Archive")
	if err == nil {
		t.Error("Expected error for unsupported format, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported archive format") {
		t.Errorf("Expected 'unsupported archive format' error, got: %v", err)
	}
}

func TestCheckDownloadAndExtractLib(t *testing.T) {
	files := map[string]string{
		"lib-src/file1.c":       "int func1() { return 1; }",
		"lib-src/file2.c":       "int func2() { return 2; }",
		"lib-src/include/lib.h": "#define LIB_VERSION 1",
	}

	archivePath := createTestTarGz(t, files)
	defer os.Remove(archivePath)

	archiveContent, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("Failed to read test archive: %v", err)
	}

	server := createTestServer(t, map[string]string{
		"test-lib.tar.gz": string(archiveContent),
	})
	defer server.Close()

	tempDir := t.TempDir()
	destDir := filepath.Join(tempDir, "test-lib")

	t.Run("LibAlreadyExists", func(t *testing.T) {
		if err := os.MkdirAll(destDir, 0755); err != nil {
			t.Fatalf("Failed to create existing lib dir: %v", err)
		}

		err := checkDownloadAndExtractLib(server.URL+"/test-lib.tar.gz", destDir, "")
		if err != nil {
			t.Errorf("Expected no error when lib exists, got: %v", err)
		}
	})

	t.Run("DownloadAndExtractWithoutInternalDir", func(t *testing.T) {
		os.RemoveAll(destDir)

		err := checkDownloadAndExtractLib(server.URL+"/test-lib.tar.gz", destDir, "lib-src")
		if err != nil {
			t.Fatalf("Failed to download and extract lib: %v", err)
		}
		cmd := exec.Command("ls", destDir)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		cmd.Run()

		for name, expectedContent := range files {
			relPath := strings.TrimPrefix(name, "lib-src/")
			filePath := filepath.Join(destDir, relPath)

			fmt.Println(filePath, destDir)
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Errorf("Failed to read extracted file %s: %v", relPath, err)
				continue
			}
			if string(content) != expectedContent {
				t.Errorf("File %s: expected content %q, got %q", relPath, expectedContent, string(content))
			}
		}
	})

	t.Run("DownloadAndExtractWithInternalDir", func(t *testing.T) {
		os.RemoveAll(destDir)

		err := checkDownloadAndExtractLib(server.URL+"/test-lib.tar.gz", destDir, "lib-src")
		if err != nil {
			t.Fatalf("Failed to download and extract lib: %v", err)
		}

		for name, expectedContent := range files {
			relPath := strings.TrimPrefix(name, "lib-src/")
			filePath := filepath.Join(destDir, relPath)
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Errorf("Failed to read extracted file %s: %v", relPath, err)
				continue
			}
			if string(content) != expectedContent {
				t.Errorf("File %s: expected content %q, got %q", relPath, expectedContent, string(content))
			}
		}
	})

	t.Run("DownloadFailure", func(t *testing.T) {
		os.RemoveAll(destDir)

		err := checkDownloadAndExtractLib(server.URL+"/nonexistent.tar.gz", destDir, "")
		if err == nil {
			t.Error("Expected error for non-existent archive, got nil")
		}
	})

	t.Run("RenameFailure", func(t *testing.T) {
		os.RemoveAll(destDir)

		err := checkDownloadAndExtractLib(server.URL+"/test-lib.tar.gz", destDir, "lib-src222")
		if err == nil {
			t.Error("Expected error for rename failure, got nil")
		}
	})
}

func TestCheckDownloadAndExtractLibInternalDir(t *testing.T) {
	files := map[string]string{
		"project-1.0.0/src/file1.c":   "int func1() { return 1; }",
		"project-1.0.0/include/lib.h": "#define LIB_VERSION 1",
		"project-1.0.0/README.md":     "Project documentation",
	}

	archivePath := createTestTarGz(t, files)
	defer os.Remove(archivePath)

	archiveContent, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("Failed to read test archive: %v", err)
	}

	server := createTestServer(t, map[string]string{
		"project.tar.gz": string(archiveContent),
	})
	defer server.Close()

	tempDir := t.TempDir()
	destDir := filepath.Join(tempDir, "project-lib")

	t.Run("CorrectInternalDir", func(t *testing.T) {
		err := checkDownloadAndExtractLib(server.URL+"/project.tar.gz", destDir, "project-1.0.0")
		if err != nil {
			t.Fatalf("Failed to download and extract lib: %v", err)
		}

		for name, expectedContent := range files {
			relPath := strings.TrimPrefix(name, "project-1.0.0/")
			filePath := filepath.Join(destDir, relPath)
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Errorf("Failed to read extracted file %s: %v", relPath, err)
				continue
			}
			if string(content) != expectedContent {
				t.Errorf("File %s: expected content %q, got %q", relPath, expectedContent, string(content))
			}
		}
	})

	t.Run("IncorrectInternalDir", func(t *testing.T) {
		os.RemoveAll(destDir)

		err := checkDownloadAndExtractLib(server.URL+"/project.tar.gz", destDir, "wrong-dir")
		if err == nil {
			t.Error("Expected error for missing internal dir, got nil")
		}
	})
}

// Mock test for WASI SDK (without actual download)
func TestWasiSDKExtractionLogic(t *testing.T) {
	tempDir := t.TempDir()

	// Create fake WASI SDK directory structure
	wasiSdkDir := filepath.Join(tempDir, wasiMacosSubdir)
	binDir := filepath.Join(wasiSdkDir, "bin")
	err := os.MkdirAll(binDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create fake WASI SDK structure: %v", err)
	}

	// Create fake clang binary
	clangPath := filepath.Join(binDir, "clang")
	err = os.WriteFile(clangPath, []byte("fake clang"), 0755)
	if err != nil {
		t.Fatalf("Failed to create fake clang: %v", err)
	}

	// Test that function returns correct path for existing SDK
	sdkRoot, err := checkDownloadAndExtractWasiSDK(tempDir)
	if err != nil {
		t.Fatalf("checkDownloadAndExtractWasiSDK failed: %v", err)
	}

	expectedRoot := filepath.Join(tempDir, wasiMacosSubdir)
	if sdkRoot != expectedRoot {
		t.Errorf("Expected SDK root %q, got %q", expectedRoot, sdkRoot)
	}
}

// Test ESP Clang extraction logic with existing directory
func TestESPClangExtractionLogic(t *testing.T) {
	tempDir := t.TempDir()
	espClangDir := filepath.Join(tempDir, "esp-clang")

	// Create fake ESP Clang directory structure
	binDir := filepath.Join(espClangDir, "bin")
	err := os.MkdirAll(binDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create fake ESP Clang structure: %v", err)
	}

	// Create fake clang binary
	clangPath := filepath.Join(binDir, "clang")
	err = os.WriteFile(clangPath, []byte("fake esp clang"), 0755)
	if err != nil {
		t.Fatalf("Failed to create fake esp clang: %v", err)
	}

	// Test that function skips download for existing directory
	err = checkDownloadAndExtractESPClang(llvmpayload.Artifact{}, espClangDir)
	if err != nil {
		t.Fatalf("checkDownloadAndExtractESPClang failed: %v", err)
	}

	// Check that the directory still exists and has the right content
	if _, err := os.Stat(clangPath); os.IsNotExist(err) {
		t.Error("ESP Clang binary should exist")
	}
}

// Test WASI SDK download and extraction when directory doesn't exist
func TestWasiSDKDownloadWhenNotExists(t *testing.T) {
	// Create fake WASI SDK archive with proper structure
	files := map[string]string{
		"wasi-sdk-25.0-x86_64-macos/bin/clang":       "fake wasi clang binary",
		"wasi-sdk-25.0-x86_64-macos/lib/libm.a":      "fake math library",
		"wasi-sdk-25.0-x86_64-macos/include/stdio.h": "#include <stdio.h>",
	}

	archivePath := createTestTarGz(t, files)
	defer os.Remove(archivePath)

	// Create test server to serve the archive
	archiveContent, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("Failed to read test archive: %v", err)
	}

	server := createTestServer(t, map[string]string{
		"wasi-sdk-25.0-x86_64-macos.tar.gz": string(archiveContent),
	})
	defer server.Close()

	// Override cacheRoot to use a temporary directory
	tempCacheRoot := t.TempDir()
	originalCacheRoot := cacheRoot
	cacheRoot = func() string { return tempCacheRoot }
	defer func() { cacheRoot = originalCacheRoot }()

	// Override wasiSdkUrl to use our test server
	originalWasiSdkUrl := wasiSdkUrl
	wasiSdkUrl = server.URL + "/wasi-sdk-25.0-x86_64-macos.tar.gz"
	defer func() { wasiSdkUrl = originalWasiSdkUrl }()

	// Use the cache directory structure
	extractDir := filepath.Join(tempCacheRoot, "crosscompile", "wasi")

	// Test download and extract when directory doesn't exist
	sdkRoot, err := checkDownloadAndExtractWasiSDK(extractDir)
	if err != nil {
		t.Fatalf("checkDownloadAndExtractWasiSDK failed: %v", err)
	}

	expectedRoot := filepath.Join(extractDir, wasiMacosSubdir)
	if sdkRoot != expectedRoot {
		t.Errorf("Expected SDK root %q, got %q", expectedRoot, sdkRoot)
	}

	// Check that files were extracted correctly
	for name, expectedContent := range files {
		filePath := filepath.Join(extractDir, name)
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("Failed to read extracted file %s: %v", name, err)
			continue
		}
		if string(content) != expectedContent {
			t.Errorf("File %s: expected content %q, got %q", name, expectedContent, string(content))
		}
	}
}

// Test ESP Clang download and extraction when directory doesn't exist
func TestESPClangDownloadWhenNotExists(t *testing.T) {
	// Create fake ESP Clang archive with proper structure
	files := map[string]string{
		"esp-clang/bin/clang":       "fake esp clang binary",
		"esp-clang/lib/libc.a":      "fake c library",
		"esp-clang/include/esp32.h": "#define ESP32 1",
	}

	archivePath := createTestTarXz(t, files)

	// Read the archive content
	archiveContent, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("Failed to read test archive: %v", err)
	}

	const filename = "clang-esp-test-linux.tar.xz"
	server := createTestServer(t, map[string]string{filename: string(archiveContent)})
	defer server.Close()

	// Override cacheRoot to use a temporary directory
	tempCacheRoot := t.TempDir()
	originalCacheRoot := cacheRoot
	cacheRoot = func() string { return tempCacheRoot }
	defer func() { cacheRoot = originalCacheRoot }()

	// Use a fresh temp directory that doesn't have ESP Clang
	espClangDir := filepath.Join(tempCacheRoot, "esp-clang-test")
	checksum, err := fileSHA256(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	artifact := llvmpayload.Artifact{
		Platform: "linux",
		Version:  "test",
		URL:      server.URL + "/" + filename,
		SHA256:   checksum,
	}

	// Test download and extract when directory doesn't exist
	err = checkDownloadAndExtractESPClang(artifact, espClangDir)
	if err != nil {
		t.Fatalf("checkDownloadAndExtractESPClang failed: %v", err)
	}

	// Check that the target directory exists
	if _, err := os.Stat(espClangDir); os.IsNotExist(err) {
		t.Error("ESP Clang directory should exist after extraction")
	}

	// Check that files were extracted correctly to the final destination
	for name, expectedContent := range files {
		// Remove "esp-clang/" prefix since it gets moved to the final destination
		relativePath := strings.TrimPrefix(name, "esp-clang/")
		filePath := filepath.Join(espClangDir, relativePath)
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("Failed to read extracted file %s: %v", relativePath, err)
			continue
		}
		if string(content) != expectedContent {
			t.Errorf("File %s: expected content %q, got %q", relativePath, expectedContent, string(content))
		}
	}
}

func TestExtractTarXzError(t *testing.T) {
	err := extractTarXz(filepath.Join(t.TempDir(), "missing.tar.xz"), t.TempDir())
	if err == nil {
		t.Fatal("extractTarXz succeeded for a missing archive")
	}
	want := "tar -xf:"
	if runtime.GOOS == "windows" && windowsSevenZip(os.Getenv("ProgramFiles")) != "" {
		want = "7-Zip xz decompression:"
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("extractTarXz error = %q, want %q command context", err, want)
	}
}

func TestFileSHA256MissingFile(t *testing.T) {
	if _, err := fileSHA256(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("fileSHA256 accepted a missing file")
	}
}

func TestESPClangDownloadLicenseFailure(t *testing.T) {
	archivePath := createTestTarXz(t, map[string]string{
		"esp-clang/bin/clang": "fake esp clang binary",
	})
	defer os.Remove(archivePath)
	archiveContent, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	const filename = "clang-esp-test-linux.tar.xz"
	server := createTestServer(t, map[string]string{filename: string(archiveContent)})
	defer server.Close()
	checksum, err := fileSHA256(archivePath)
	if err != nil {
		t.Fatal(err)
	}

	llgoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(llgoRoot, "runtime"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(llgoRoot, "runtime", "go.mod"),
		[]byte("module github.com/xgo-dev/llgo/runtime\n"), 0o644,
	); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", llgoRoot)

	destDir := filepath.Join(t.TempDir(), "esp-clang")
	err = checkDownloadAndExtractESPClang(llvmpayload.Artifact{
		Platform: "linux",
		Version:  "test",
		URL:      server.URL + "/" + filename,
		SHA256:   checksum,
	}, destDir)
	if err == nil || !strings.Contains(err.Error(), "read ESP Clang license") {
		t.Fatalf("checkDownloadAndExtractESPClang() error = %v, want license read error", err)
	}
	if _, err := os.Stat(destDir); !os.IsNotExist(err) {
		t.Fatalf("destination status after failed install = %v, want not exist", err)
	}
	if _, err := os.Stat(destDir + ".extract"); !os.IsNotExist(err) {
		t.Fatalf("temporary extraction directory status = %v, want not exist", err)
	}
}

func TestInstallESPClangLicense(t *testing.T) {
	llgoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(llgoRoot, "runtime"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(llgoRoot, "runtime", "go.mod"),
		[]byte("module github.com/xgo-dev/llgo/runtime\n"), 0o644,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(llgoRoot, "LICENSES"), 0o755); err != nil {
		t.Fatal(err)
	}
	const want = "complete LLVM license\n"
	if err := os.WriteFile(
		filepath.Join(llgoRoot, "LICENSES", espClangLicenseFile),
		[]byte(want), 0o644,
	); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", llgoRoot)

	clangDir := t.TempDir()
	if err := installESPClangLicense(clangDir); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(clangDir, "LICENSE-LLVM.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("installed license = %q, want %q", got, want)
	}
}

func TestInstallESPClangLicenseMissing(t *testing.T) {
	llgoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(llgoRoot, "runtime"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(llgoRoot, "runtime", "go.mod"),
		[]byte("module github.com/xgo-dev/llgo/runtime\n"), 0o644,
	); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", llgoRoot)

	err := installESPClangLicense(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), espClangLicenseFile) {
		t.Fatalf("installESPClangLicense() error = %v, want missing license error", err)
	}
}

func TestInstallESPClangLicenseWriteFailure(t *testing.T) {
	llgoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(llgoRoot, "runtime"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(llgoRoot, "runtime", "go.mod"),
		[]byte("module github.com/xgo-dev/llgo/runtime\n"), 0o644,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(llgoRoot, "LICENSES"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(llgoRoot, "LICENSES", espClangLicenseFile),
		[]byte("complete LLVM license\n"), 0o644,
	); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", llgoRoot)

	notDir := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(notDir, []byte("file"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := installESPClangLicense(notDir)
	if err == nil || !strings.Contains(err.Error(), "install ESP Clang license") {
		t.Fatalf("installESPClangLicense() error = %v, want write error", err)
	}
}

func TestESPClangRejectsChecksumMismatch(t *testing.T) {
	archivePath := createTestTarGz(t, map[string]string{"esp-clang/bin/clang": "fake"})
	defer os.Remove(archivePath)

	archiveContent, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	server := createTestServer(t, map[string]string{"clang-esp-test-linux.tar.xz": string(archiveContent)})
	defer server.Close()

	destination := filepath.Join(t.TempDir(), "esp-clang")
	artifact := llvmpayload.Artifact{
		Platform: "linux",
		Version:  "test",
		URL:      server.URL + "/clang-esp-test-linux.tar.xz",
		SHA256:   strings.Repeat("0", 64),
	}
	err = checkDownloadAndExtractESPClang(artifact, destination)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("checksum mismatch error = %v", err)
	}
	if _, statErr := os.Stat(destination); !os.IsNotExist(statErr) {
		t.Fatalf("destination exists after rejected download: %v", statErr)
	}
}

func TestExtractZip(t *testing.T) {
	// Create temporary test directory
	tempDir := t.TempDir()
	zipPath := filepath.Join(tempDir, "test.zip")
	destDir := filepath.Join(tempDir, "extracted")

	// 1. Test successful extraction
	t.Run("SuccessfulExtraction", func(t *testing.T) {
		// Create test ZIP file
		if err := createTestZip(zipPath); err != nil {
			t.Fatalf("Failed to create test zip: %v", err)
		}

		// Execute extraction
		if err := extractZip(zipPath, destDir); err != nil {
			t.Fatalf("extractZip failed: %v", err)
		}

		// Verify extraction results
		verifyExtraction(t, destDir)
	})

	// 2. Test invalid ZIP file
	t.Run("InvalidZipFile", func(t *testing.T) {
		// Create invalid ZIP file (actually a text file)
		if err := os.WriteFile(zipPath, []byte("not a zip file"), 0644); err != nil {
			t.Fatal(err)
		}

		// Execute extraction and expect error
		if err := extractZip(zipPath, destDir); err == nil {
			t.Error("Expected error for invalid zip file, got nil")
		}
	})

	// 3. Test a destination that cannot contain extracted files. Unlike Unix
	// permission bits, this remains deterministic on Windows and as root.
	t.Run("NonDirectoryDestination", func(t *testing.T) {
		// Create test ZIP file
		if err := createTestZip(zipPath); err != nil {
			t.Fatal(err)
		}

		notDirectory := filepath.Join(tempDir, "not-a-directory")
		if err := os.WriteFile(notDirectory, nil, 0o644); err != nil {
			t.Fatal(err)
		}

		// Execute extraction and expect error
		if err := extractZip(zipPath, notDirectory); err == nil {
			t.Error("Expected error for non-directory destination, got nil")
		}
	})
}

// Create test ZIP file
func createTestZip(zipPath string) error {
	// Create ZIP file
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Add directory
	dirHeader := &zip.FileHeader{
		Name:     "testdir/",
		Method:   zip.Deflate,
		Modified: time.Now(),
	}
	dirHeader.SetMode(os.ModeDir | 0755)
	if _, err := zipWriter.CreateHeader(dirHeader); err != nil {
		return err
	}

	// Add file1
	file1, err := zipWriter.Create("file1.txt")
	if err != nil {
		return err
	}
	if _, err := file1.Write([]byte("Hello from file1")); err != nil {
		return err
	}

	// Add nested file
	nestedFile, err := zipWriter.Create("testdir/nested.txt")
	if err != nil {
		return err
	}
	if _, err := nestedFile.Write([]byte("Nested content")); err != nil {
		return err
	}

	return nil
}

// Verify extraction results
func verifyExtraction(t *testing.T, destDir string) {
	// Verify directory exists
	if _, err := os.Stat(filepath.Join(destDir, "testdir")); err != nil {
		t.Errorf("Directory not extracted: %v", err)
	}

	// Verify file1 content
	file1Path := filepath.Join(destDir, "file1.txt")
	content, err := os.ReadFile(file1Path)
	if err != nil {
		t.Errorf("Failed to read file1: %v", err)
	}
	if string(content) != "Hello from file1" {
		t.Errorf("File1 content mismatch. Got: %s", content)
	}

	// Verify nested file content
	nestedPath := filepath.Join(destDir, "testdir", "nested.txt")
	content, err = os.ReadFile(nestedPath)
	if err != nil {
		t.Errorf("Failed to read nested file: %v", err)
	}
	if string(content) != "Nested content" {
		t.Errorf("Nested file content mismatch. Got: %s", content)
	}
}
