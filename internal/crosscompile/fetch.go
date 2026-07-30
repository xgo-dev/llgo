package crosscompile

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/goplus/llgo/internal/processenv"
)

// checkDownloadAndExtractWasiSDK downloads and extracts WASI SDK
func checkDownloadAndExtractWasiSDK(dir string) (wasiSdkRoot string, err error) {
	return checkDownloadAndExtractWasiSDKWithContext(dir, processenv.Context{})
}

func checkDownloadAndExtractWasiSDKWithContext(dir string, process processenv.Context) (wasiSdkRoot string, err error) {
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

	err = downloadAndExtractArchiveWithContext(wasiSdkUrl, dir, "WASI SDK", process)
	return wasiSdkRoot, err
}

// checkDownloadAndExtractESPClang downloads and extracts ESP Clang binaries and libraries
func checkDownloadAndExtractESPClang(platformSuffix, dir string) error {
	return checkDownloadAndExtractESPClangWithContext(platformSuffix, dir, processenv.Context{})
}

func checkDownloadAndExtractESPClangWithContext(platformSuffix, dir string, process processenv.Context) error {
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

	clangUrl := fmt.Sprintf("%s/clang-esp-%s-%s.tar.xz", espClangBaseUrl, espClangVersion, platformSuffix)
	description := fmt.Sprintf("ESP Clang %s-%s", espClangVersion, platformSuffix)

	// Use temporary extraction directory for ESP Clang special handling
	tempExtractDir := dir + ".extract"
	if err := downloadAndExtractArchiveWithContext(clangUrl, tempExtractDir, description, process); err != nil {
		return err
	}
	defer os.RemoveAll(tempExtractDir)

	// ESP Clang needs special handling: move esp-clang subdirectory to final destination
	espClangDir := filepath.Join(tempExtractDir, "esp-clang")
	if err := os.Rename(espClangDir, dir); err != nil {
		return fmt.Errorf("failed to rename esp-clang directory: %w", err)
	}

	return nil
}

func checkDownloadAndExtractLib(url, dstDir, internalArchiveSrcDir string) error {
	return checkDownloadAndExtractLibWithContext(url, dstDir, internalArchiveSrcDir, processenv.Context{})
}

func checkDownloadAndExtractLibWithContext(url, dstDir, internalArchiveSrcDir string, process processenv.Context) error {
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
	if err := downloadAndExtractArchiveWithContext(url, tempExtractDir, description, process); err != nil {
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
	// Ensure the parent directory exists
	if err := os.MkdirAll(filepath.Dir(lockPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create lock directory: %w", err)
	}

	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create lock file: %w", err)
	}
	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
		lockFile.Close()
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}
	return lockFile, nil
}

// releaseLock unlocks and removes the lock file
func releaseLock(lockFile *os.File) error {
	if lockFile == nil {
		return nil
	}
	lockPath := lockFile.Name()
	syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)
	lockFile.Close()
	os.Remove(lockPath)
	return nil
}

// downloadAndExtractArchive downloads and extracts an archive to the destination directory (without locking)
func downloadAndExtractArchive(url, destDir, description string) error {
	return downloadAndExtractArchiveWithContext(url, destDir, description, processenv.Context{})
}

func downloadAndExtractArchiveWithContext(url, destDir, description string, process processenv.Context) error {
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

	// Extract the archive
	fmt.Fprintf(os.Stderr, "Extracting %s...\n", description)
	if strings.HasSuffix(filename, ".tar.gz") || strings.HasSuffix(filename, ".tgz") {
		err := extractTarGz(localFile, tempDir)
		if err != nil {
			return fmt.Errorf("failed to extract %s archive: %w", description, err)
		}
	} else if strings.HasSuffix(filename, ".tar.xz") {
		err := extractTarXzWithContext(localFile, tempDir, process)
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
	return extractTarXzWithContext(tarXzFile, dest, processenv.Context{})
}

func extractTarXzWithContext(tarXzFile, dest string, process processenv.Context) error {
	// Use external tar command to extract .tar.xz files
	cmd := process.Command("tar", "-xf", tarXzFile, "-C", dest)
	return cmd.Run()
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
