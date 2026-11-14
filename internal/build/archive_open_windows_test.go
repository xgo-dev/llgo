package build

import (
	"bufio"
	"bytes"
	stdcontext "context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"
)

func TestPCLNArchiveLockProcess(t *testing.T) {
	path := os.Getenv("LLGO_TEST_PCLN_LOCK_PATH")
	if path == "" {
		t.Skip("subprocess helper")
	}
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	h, err := syscall.CreateFile(name, syscall.GENERIC_READ, 0, nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer syscall.CloseHandle(h)
	// Check the native handle before involving the parent. This distinguishes
	// a failed lock fixture from premature helper exit or a different file.
	var info syscall.ByHandleFileInformation
	if err := syscall.GetFileInformationByHandle(h, &info); err != nil {
		t.Fatal(err)
	}
	probe, probeErr := syscall.CreateFile(name, syscall.GENERIC_READ, syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE, nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if probe != syscall.InvalidHandle {
		syscall.CloseHandle(probe)
	}
	f, openErr := os.Open(path)
	var readErr error
	if f != nil {
		_, readErr = f.ReadAt(make([]byte, 1), 0)
		f.Close()
	}
	if !errors.Is(probeErr, syscall.Errno(32)) || !errors.Is(openErr, syscall.Errno(32)) {
		fmt.Fprintf(os.Stderr, "lock self-check: path=%q handle=%#x volume=%x file=%x:%x nativeHandle=%#x nativeError=%v openError=%v readError=%v\n", path, h, info.VolumeSerialNumber, info.FileIndexHigh, info.FileIndexLow, probe, probeErr, openErr, readErr)
		t.Fatal("exclusive handle did not establish a sharing conflict in its own process")
	}
	fmt.Println("locked")
	// Require an explicit release command so an unexpected stdin EOF cannot
	// silently release the handle before the parent has exercised the reader.
	command := make([]byte, len("release\n"))
	if _, err := io.ReadFull(os.Stdin, command); err != nil || string(command) != "release\n" {
		fmt.Fprintf(os.Stderr, "archive lock release command: %q, %v\n", command, err)
		t.Fatal("missing archive lock release command")
	}
}

func lockPCLNTestArchive(t *testing.T) (string, func() error) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "pkg.a")
	if err := os.WriteFile(path, []byte("archive contents"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := stdcontext.WithTimeout(stdcontext.Background(), 30*time.Second)
	t.Cleanup(cancel)
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestPCLNArchiveLockProcess$")
	cmd.Env = append(os.Environ(), "LLGO_TEST_PCLN_LOCK_PATH="+path)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		t.Fatal(err)
	}
	var once sync.Once
	var waitErr error
	release := func() error {
		once.Do(func() {
			_, writeErr := io.WriteString(stdin, "release\n")
			stdin.Close()
			waitErr = errors.Join(writeErr, cmd.Wait())
		})
		return waitErr
	}
	t.Cleanup(func() {
		if err := release(); err != nil {
			t.Errorf("archive lock helper: %v\n%s", err, stderr.String())
		}
	})
	if line, err := bufio.NewReader(stdout).ReadString('\n'); err != nil || line != "locked\n" {
		t.Fatalf("archive lock helper readiness: %q, %v", line, err)
	}
	// Verify the fixture before testing the retry wrapper: a successful open
	// here must not turn the transient-conflict test into a false positive.
	f, err := os.Open(path)
	if f != nil {
		f.Close()
	}
	if !errors.Is(err, syscall.Errno(32)) {
		t.Fatalf("archive lock helper did not establish a sharing conflict: %v", err)
	}
	return path, release
}

func TestOpenPCLNArchiveSharingViolation(t *testing.T) {
	path, release := lockPCLNTestArchive(t)
	// Release a real conflicting Windows handle while the reader retries.
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
		t.Fatalf("read after lock release: %q, %v", buf, err)
	}
}

func TestOpenPCLNArchivePersistentSharingViolation(t *testing.T) {
	path, _ := lockPCLNTestArchive(t)
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
