package build

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func lockPCLNTestArchive(t *testing.T) (string, syscall.Handle) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "pkg.a")
	if err := os.WriteFile(path, []byte("archive contents"), 0600); err != nil {
		t.Fatal(err)
	}
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	h, err := syscall.CreateFile(name, syscall.GENERIC_READ, 0, nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	return path, h
}

func TestOpenPCLNArchiveSharingViolation(t *testing.T) {
	path, handle := lockPCLNTestArchive(t)
	// Release a real conflicting Windows handle while the reader retries.
	released := make(chan error, 1)
	go func() {
		time.Sleep(100 * time.Millisecond)
		released <- syscall.CloseHandle(handle)
	}()
	f, err := openPCLNArchive(path)
	if f != nil {
		defer f.Close()
	}
	if closeErr := <-released; closeErr != nil {
		t.Fatal(closeErr)
	}
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, len("archive contents"))
	if _, err := f.ReadAt(buf, 0); err != nil || string(buf) != "archive contents" {
		t.Fatalf("read after lock release: %q, %v", buf, err)
	}
}

func TestOpenPCLNArchivePersistentSharingViolation(t *testing.T) {
	path, handle := lockPCLNTestArchive(t)
	defer syscall.CloseHandle(handle)
	start := time.Now()
	f, err := openPCLNArchive(path)
	if f != nil {
		f.Close()
		t.Fatal("opened exclusively locked archive")
	}
	if !errors.Is(err, syscall.Errno(32)) || time.Since(start) < 2*time.Second {
		t.Fatalf("persistent sharing violation: %v after %v", err, time.Since(start))
	}
}

func TestOpenPCLNArchiveMissing(t *testing.T) {
	f, err := openPCLNArchive(filepath.Join(t.TempDir(), "missing.a"))
	if f != nil {
		f.Close()
		t.Fatal("opened missing archive")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing archive: %v", err)
	}
}
