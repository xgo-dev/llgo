package build

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"
)

// Use a direct Win32 reader to exercise real sharing violations. On the
// native Windows CI hosts os.Open can read through this read-only exclusive
// handle even while this CreateFile call returns ERROR_SHARING_VIOLATION.
// The retry policy must handle the error when it occurs, not require every
// reader and host to reject an otherwise successful read.
func openPCLNTestArchive(path string) (*os.File, error) {
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	h, err := syscall.CreateFile(name, syscall.GENERIC_READ, syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE, nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: path, Err: err}
	}
	return os.NewFile(uintptr(h), path), nil
}

func lockPCLNTestArchive(t *testing.T) (string, func() error) {
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
	var once sync.Once
	var closeErr error
	release := func() error {
		once.Do(func() { closeErr = syscall.CloseHandle(h) })
		return closeErr
	}
	t.Cleanup(func() {
		if err := release(); err != nil {
			t.Error(err)
		}
	})
	f, err := openPCLNTestArchive(path)
	if f != nil {
		f.Close()
	}
	if !errors.Is(err, syscall.Errno(32)) {
		t.Fatalf("exclusive handle did not establish a sharing conflict: %v", err)
	}
	return path, release
}

// Exercise the production os.Open path too. It may open immediately or after
// release, but must return a readable archive with the original contents.
func TestOpenPCLNArchiveNativeReader(t *testing.T) {
	path, release := lockPCLNTestArchive(t)
	released := make(chan error, 1)
	go func() {
		time.Sleep(100 * time.Millisecond)
		released <- release()
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
		t.Fatalf("native read: %q, %v", buf, err)
	}
}

func TestOpenPCLNArchivePermanentError(t *testing.T) {
	want := &os.PathError{Op: "open", Path: "archive.a", Err: syscall.ERROR_ACCESS_DENIED}
	calls := 0
	f, err := openPCLNArchiveWithOpener(want.Path, func(string) (*os.File, error) {
		calls++
		return nil, want
	})
	if f != nil || err != want || calls != 1 {
		t.Fatalf("permanent error: file=%v err=%v calls=%d", f, err, calls)
	}
}

func TestOpenPCLNArchiveSharingViolation(t *testing.T) {
	path, release := lockPCLNTestArchive(t)
	// Release a real conflicting Windows handle while the reader retries.
	released := make(chan error, 1)
	go func() {
		time.Sleep(100 * time.Millisecond)
		released <- release()
	}()
	f, err := openPCLNArchiveWithOpener(path, openPCLNTestArchive)
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
	path, _ := lockPCLNTestArchive(t)
	start := time.Now()
	f, err := openPCLNArchiveWithOpener(path, openPCLNTestArchive)
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
