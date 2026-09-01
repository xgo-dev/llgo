package crosscompile

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llgo/internal/llvmpayload"
)

const espClangLicenseFile = "XGo-LLVM-Apache-2.0-WITH-LLVM-exception.txt"

// checkDownloadAndExtractWasiSDK downloads and extracts WASI SDK
func checkDownloadAndExtractWasiSDK(dir string) (wasiSdkRoot string, err error) {
	wasiSdkRoot = filepath.Join(dir, wasiMacosSubdir)

	// Check if already exists
	if _, err := os.Stat(wasiSdkRoot); err == nil {
		return wasiSdkRoot, nil
	}

	// Create lock file path for the parent directory (dir) since that's what we're operating on
	lockPath := dir + ".lock"
	lockFile, err := acquireLock(lockPath)
	if err != nil {
		return "", fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer releaseLock(lockFile)

	// Double-check after acquiring lock
	if _, err := os.Stat(wasiSdkRoot); err == nil {
		return wasiSdkRoot, nil
	}

	err = downloadAndExtractArchive(wasiSdkUrl, dir, "WASI SDK")
	return wasiSdkRoot, err
}

// checkDownloadAndExtractESPClang downloads and extracts ESP Clang binaries and libraries
func checkDownloadAndExtractESPClang(artifact llvmpayload.Artifact, dir string) error {
	// Check if already exists
	if _, err := os.Stat(dir); err == nil {
		return nil
	}

	// Create lock file path for the final destination
	lockPath := dir + ".lock"
	lockFile, err := acquireLock(lockPath)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer releaseLock(lockFile)

	// Double-check after acquiring lock
	if _, err := os.Stat(dir); err == nil {
		return nil
	}

	description := fmt.Sprintf("ESP Clang %s-%s", artifact.Version, artifact.Platform)

	// Use temporary extraction directory for ESP Clang special handling
	tempExtractDir := dir + ".extract"
	if err := downloadAndExtractArchiveWithChecksum(artifact.URL, tempExtractDir, description, artifact.SHA256); err != nil {
		return err
	}
	defer os.RemoveAll(tempExtractDir)

	// ESP Clang needs special handling: move esp-clang subdirectory to final destination
	espClangDir := filepath.Join(tempExtractDir, "esp-clang")
	if err := installESPClangLicense(espClangDir); err != nil {
		return err
	}
	if err := os.Rename(espClangDir, dir); err != nil {
		return fmt.Errorf("failed to rename esp-clang directory: %w", err)
	}

	return nil
}

// installESPClangLicense places the complete LLVM license next to an ESP Clang
// toolchain. The upstream archive currently contains only a short license
// pointer under include/llvm/Support; LLGo source trees and release archives
// carry the complete Apache-2.0 WITH LLVM-exception text under LICENSES.
func installESPClangLicense(clangDir string) error {
	license, err := os.ReadFile(filepath.Join(env.LLGoROOT(), "LICENSES", espClangLicenseFile))
	if err != nil {
		return fmt.Errorf("read ESP Clang license: %w", err)
	}
	if err := os.WriteFile(filepath.Join(clangDir, "LICENSE-LLVM.txt"), license, 0o644); err != nil {
		return fmt.Errorf("install ESP Clang license: %w", err)
	}
	return nil
}

func checkDownloadAndExtractLib(url, dstDir, internalArchiveSrcDir string) error {
	// Check if already exists
	if _, err := os.Stat(dstDir); err == nil {
		return nil
	}

	// Create lock file path for the final destination
	lockPath := dstDir + ".lock"
	lockFile, err := acquireLock(lockPath)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer releaseLock(lockFile)

	// Double-check after acquiring lock
	if _, err := os.Stat(dstDir); err == nil {
		return nil
	}
	fmt.Fprintf(os.Stderr, "%s not found in LLGO_ROOT or cache, will download and compile.\n", dstDir)

	description := fmt.Sprintf("lib %s", path.Base(url))

	// Use temporary extraction directory
	tempExtractDir := dstDir + ".extract"
	if err := downloadAndExtractArchive(url, tempExtractDir, description); err != nil {
		return err
	}
	defer os.RemoveAll(tempExtractDir)

	srcDir := tempExtractDir

	if internalArchiveSrcDir != "" {
		srcDir = filepath.Join(tempExtractDir, internalArchiveSrcDir)
	}

	if err := os.Rename(srcDir, dstDir); err != nil {
		return fmt.Errorf("failed to rename lib directory: %w", err)
	}

	return nil
}

// acquireLock creates and locks a file to prevent concurrent operations
func acquireLock(lockPath string) (*os.File, error) {
	return acquireLockWith(lockPath, lockFileHandle)
}

func acquireLockWith(lockPath string, lock func(*os.File) error) (*os.File, error) {
	// Ensure the parent directory exists
	if err := os.MkdirAll(filepath.Dir(lockPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create lock directory: %w", err)
	}

	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create lock file: %w", err)
	}
	if err := lock(lockFile); err != nil {
		_ = lockFile.Close()
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}
	return lockFile, nil
}

// releaseLock unlocks and closes the lock file. The file must remain in place:
// removing it could let a new caller lock a different file while another caller
// still holds the original one.
func releaseLock(lockFile *os.File) error {
	if lockFile == nil {
		return nil
	}
	unlockErr := unlockFileHandle(lockFile)
	closeErr := lockFile.Close()
	return lockReleaseError(unlockErr, closeErr)
}

func lockReleaseError(unlockErr, closeErr error) error {
	if unlockErr != nil {
		return fmt.Errorf("failed to release lock: %w", unlockErr)
	}
	if closeErr != nil {
		return fmt.Errorf("failed to close lock file: %w", closeErr)
	}
	return nil
}

// downloadAndExtractArchive downloads and extracts an archive to the destination directory (without locking)
func downloadAndExtractArchive(url, destDir, description string) error {
	return downloadAndExtractArchiveWithChecksum(url, destDir, description, "")
}

func downloadAndExtractArchiveWithChecksum(url, destDir, description, expectedSHA256 string) error {
	fmt.Fprintf(os.Stderr, "Downloading %s...\n", description)

	// Use temporary extraction directory
	tempDir := destDir + ".temp"
	os.RemoveAll(tempDir)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Download the archive
	urlPath := strings.Split(url, "/")
	filename := urlPath[len(urlPath)-1]
	localFile := filepath.Join(tempDir, filename)
	if err := downloadFile(url, localFile); err != nil {
		return fmt.Errorf("failed to download %s from %s: %w", description, url, err)
	}
	if expectedSHA256 != "" {
		actualSHA256, err := fileSHA256(localFile)
		if err != nil {
			return fmt.Errorf("calculate %s checksum: %w", description, err)
		}
		if !strings.EqualFold(actualSHA256, expectedSHA256) {
			return fmt.Errorf("%s checksum mismatch: got %s, want %s", description, actualSHA256, expectedSHA256)
		}
	}

	// Extract the archive
	fmt.Fprintf(os.Stderr, "Extracting %s...\n", description)
	if strings.HasSuffix(filename, ".tar.gz") || strings.HasSuffix(filename, ".tgz") {
		err := extractTarGz(localFile, tempDir)
		if err != nil {
			return fmt.Errorf("failed to extract %s archive: %w", description, err)
		}
	} else if strings.HasSuffix(filename, ".tar.xz") {
		err := extractTarXz(localFile, tempDir)
		if err != nil {
			return fmt.Errorf("failed to extract %s archive: %w", description, err)
		}
	} else if strings.HasSuffix(filename, ".zip") {
		err := extractZip(localFile, tempDir)
		if err != nil {
			return fmt.Errorf("failed to extract %s archive: %w", description, err)
		}
	} else {
		return fmt.Errorf("unsupported archive format: %s", filename)
	}
	// Rename temp directory to target directory
	if err := os.Rename(tempDir, destDir); err != nil {
		return fmt.Errorf("failed to rename directory: %w", err)
	}

	fmt.Fprintf(os.Stderr, "%s downloaded and extracted successfully.\n", description)
	return nil
}

func fileSHA256(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()
	return readerSHA256(file)
}

func readerSHA256(reader io.Reader) (string, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func downloadFile(url, filepath string) error {
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}
	_, err = io.Copy(out, resp.Body)
	return err
}

func extractTarGz(tarGzFile, dest string) error {
	file, err := os.Open(tarGzFile)
	if err != nil {
		return err
	}
	defer file.Close()
	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()
	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		target := filepath.Join(dest, header.Name)
		if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("%s: illegal file path", target)
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}
	return nil
}

func extractTarXz(tarXzFile, dest string) error {
	return extractTarXzForGOOS(runtime.GOOS, os.Getenv("ProgramFiles"), os.Getenv("SystemRoot"), tarXzFile, dest)
}

func extractTarXzForGOOS(goos, programFiles, systemRoot, tarXzFile, dest string) error {
	tarCommand := "tar"
	tarArgs := []string{"-xf", tarXzFile, "-C", dest}
	if goos == "windows" {
		if sevenZip := windowsSevenZip(programFiles); sevenZip != "" {
			return extractTarXzWith7Zip(sevenZip, tarXzFile, dest)
		}
		tarCommand = windowsTarCommand(systemRoot)
	}
	cmd := exec.Command(tarCommand, tarArgs...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("tar -xf: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func extractTarXzWith7Zip(sevenZip, tarXzFile, dest string) error {
	return extractTarXzWith7ZipCommand(sevenZip, tarXzFile, dest, exec.Command)
}

func extractTarXzWith7ZipCommand(sevenZip, tarXzFile, dest string, command func(string, ...string) *exec.Cmd) error {
	decompress := command(sevenZip, "x", "-so", tarXzFile)
	extract := command(sevenZip, "x", "-si", "-ttar", "-y", "-o"+dest)
	pipe, err := decompress.StdoutPipe()
	if err != nil {
		return fmt.Errorf("7-Zip xz pipe: %w", err)
	}
	extract.Stdin = pipe
	var decompressStderr, extractStderr bytes.Buffer
	decompress.Stderr = &decompressStderr
	extract.Stdout = io.Discard
	extract.Stderr = &extractStderr

	// Start the consumer first so the decompressor can stream a multi-gigabyte
	// tar without writing a second archive to the comparatively small Windows
	// hosted-runner disk.
	if err := extract.Start(); err != nil {
		return fmt.Errorf("start 7-Zip tar extraction: %w", err)
	}
	if err := decompress.Start(); err != nil {
		_ = extract.Process.Kill()
		_ = extract.Wait()
		return fmt.Errorf("start 7-Zip xz decompression: %w", err)
	}
	extractErr := extract.Wait()
	decompressErr := decompress.Wait()
	if decompressErr != nil {
		return fmt.Errorf("7-Zip xz decompression: %w: %s", decompressErr, strings.TrimSpace(decompressStderr.String()))
	}
	if extractErr != nil {
		return fmt.Errorf("7-Zip tar extraction: %w: %s", extractErr, strings.TrimSpace(extractStderr.String()))
	}
	return nil
}

func windowsSevenZip(programFiles string) string {
	if programFiles != "" {
		sevenZip := filepath.Join(programFiles, "7-Zip", "7z.exe")
		if fileExists(sevenZip) {
			return sevenZip
		}
	}
	if sevenZip, err := exec.LookPath("7z.exe"); err == nil {
		return sevenZip
	}
	return ""
}

func windowsTarCommand(systemRoot string) string {
	if systemRoot != "" {
		nativeTar := filepath.Join(systemRoot, "System32", "tar.exe")
		if fileExists(nativeTar) {
			return nativeTar
		}
	}
	return "tar"
}

func fileExists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

func extractZip(zipFile, dest string) error {
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer r.Close()
	decompress := func(file *zip.File) error {
		path := filepath.Join(dest, file.Name)

		if file.FileInfo().IsDir() {
			return os.MkdirAll(path, 0700)
		}

		fs, err := file.Open()
		if err != nil {
			return err
		}
		defer fs.Close()

		w, err := os.Create(path)
		if err != nil {
			return err
		}
		if _, err := io.Copy(w, fs); err != nil {
			w.Close()
			return err
		}
		if err := w.Close(); err != nil {
			return err
		}
		return nil
	}

	for _, file := range r.File {
		if err = decompress(file); err != nil {
			break
		}
	}
	return err
}
