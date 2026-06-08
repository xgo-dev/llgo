//go:build !llgo
// +build !llgo

package crosscompile

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
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

	// Check lock file is removed
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Error("Lock file should be removed after release")
	}
}

func TestAcquireLockConcurrency(t *testing.T) {
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "concurrent.lock")

	var wg sync.WaitGroup
	var results []int
	var resultsMu sync.Mutex

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

			// Hold the lock for a short time
			resultsMu.Lock()
			results = append(results, id)
			resultsMu.Unlock()

			time.Sleep(10 * time.Millisecond)

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
	oldDelay := downloadRetryDelay
	downloadRetryDelay = 0
	defer func() { downloadRetryDelay = oldDelay }()

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

func TestDownloadFileRetriesServerError(t *testing.T) {
	oldDelay := downloadRetryDelay
	downloadRetryDelay = 0
	defer func() { downloadRetryDelay = oldDelay }()

	var attempts int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			http.Error(w, "temporary error", http.StatusBadGateway)
			return
		}
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	targetFile := filepath.Join(t.TempDir(), "downloaded.txt")
	if err := downloadFile(server.URL+"/archive.tar.gz", targetFile); err != nil {
		t.Fatalf("downloadFile failed after retry: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("download attempts = %d, want 2", attempts)
	}
	content, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(content) != "ok" {
		t.Fatalf("downloaded content = %q, want ok", content)
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
	defer os.Remove(archivePath)

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
	err = checkDownloadAndExtractESPClang("linux", espClangDir)
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

	archivePath := createTestTarGz(t, files)
	defer os.Remove(archivePath)

	// Read the archive content
	archiveContent, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("Failed to read test archive: %v", err)
	}

	server := createTestServer(t, map[string]string{
		fmt.Sprintf("clang-esp-%s-linux.tar.xz", espClangVersion): string(archiveContent),
	})
	defer server.Close()

	// Override cacheRoot to use a temporary directory
	tempCacheRoot := t.TempDir()
	originalCacheRoot := cacheRoot
	cacheRoot = func() string { return tempCacheRoot }
	defer func() { cacheRoot = originalCacheRoot }()

	// Override espClangBaseUrl to use our test server
	originalEspClangBaseUrl := espClangBaseUrl
	espClangBaseUrl = server.URL
	defer func() { espClangBaseUrl = originalEspClangBaseUrl }()

	// Use a fresh temp directory that doesn't have ESP Clang
	espClangDir := filepath.Join(tempCacheRoot, "esp-clang-test")

	// Test download and extract when directory doesn't exist
	err = checkDownloadAndExtractESPClang("linux", espClangDir)
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

	// 3. Test non-writable destination
	t.Run("UnwritableDestination", func(t *testing.T) {
		// Create test ZIP file
		if err := createTestZip(zipPath); err != nil {
			t.Fatal(err)
		}

		// Create read-only destination directory
		readOnlyDir := filepath.Join(tempDir, "readonly")
		if err := os.MkdirAll(readOnlyDir, 0400); err != nil {
			t.Fatal(err)
		}

		// Execute extraction and expect error
		if err := extractZip(zipPath, readOnlyDir); err == nil {
			t.Error("Expected error for unwritable destination, got nil")
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
