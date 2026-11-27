package pclnpost

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"
)

type stageTestFile struct {
	*os.File
	kind   string
	before func(string) error
	closes int
}

func (f *stageTestFile) Chmod(mode os.FileMode) error {
	if err := f.before("chmod"); err != nil {
		return err
	}
	return f.File.Chmod(mode)
}

func (f *stageTestFile) Write(data []byte) (int, error) {
	if err := f.before("write"); err != nil {
		n, _ := f.File.Write(data[:len(data)/2])
		return n, err // Leave a partial image to exercise cleanup.
	}
	return f.File.Write(data)
}

func (f *stageTestFile) Sync() error {
	if err := f.before("sync"); err != nil {
		return err
	}
	return f.File.Sync()
}

func (f *stageTestFile) Close() error {
	f.closes++
	err := f.before("close-" + f.kind)
	closeErr := f.File.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func TestStageBinaryIOFailuresAndForkWindow(t *testing.T) {
	// Do not parallelize: the assertions observe the process-global ForkLock.
	for _, fail := range []string{"", "create", "chmod", "write", "close-writer", "open-reader", "sync", "close-reader"} {
		name := fail
		if name == "" {
			name = "success"
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			wantErr := errors.New("injected " + fail)
			cleanupErr := errors.New("cleanup close failed")
			var calls []string
			before := func(op string) error {
				calls = append(calls, op)
				wantLocked := slices.Contains([]string{"create", "open-reader", "chmod", "write", "close-writer"}, op)
				locked := !syscall.ForkLock.TryLock()
				if !locked {
					syscall.ForkLock.Unlock()
				}
				if locked != wantLocked {
					t.Errorf("%s: fork excluded = %v, want %v", op, locked, wantLocked)
				}
				if op == fail {
					return wantErr
				}
				if fail == "write" && op == "close-writer" {
					return cleanupErr // Must not replace the original write error.
				}
				return nil
			}
			var opened []*stageTestFile
			wrap := func(f *os.File, err error, kind string) (binaryStageFile, error) {
				if err != nil {
					return nil, err
				}
				file := &stageTestFile{File: f, kind: kind, before: before}
				opened = append(opened, file)
				return file, nil
			}
			files := binaryStageFiles{
				createTemp: func(dir, pattern string) (binaryStageFile, error) {
					if err := before("create"); err != nil {
						return nil, err
					}
					f, err := os.CreateTemp(dir, pattern)
					return wrap(f, err, "writer")
				},
				openReadOnly: func(path string) (binaryStageFile, error) {
					if err := before("open-reader"); err != nil {
						return nil, err
					}
					f, err := os.Open(path)
					if err == nil {
						flags, _, errno := syscall.Syscall(syscall.SYS_FCNTL, f.Fd(), syscall.F_GETFL, 0)
						if errno != 0 || flags&syscall.O_ACCMODE != syscall.O_RDONLY {
							t.Errorf("sync descriptor flags = %#x, errno = %v; want read-only", flags, errno)
						}
					}
					return wrap(f, err, "reader")
				},
			}
			raw := []byte("staged executable")
			path, err := stageBinaryWithFiles(dir, "image-*", raw, 0751, files)
			if fail == "" {
				if err != nil {
					t.Fatal(err)
				}
				if got := strings.Join(calls, ","); got != "create,open-reader,chmod,write,close-writer,sync,close-reader" {
					t.Errorf("I/O order = %s", got)
				}
				if got, err := os.ReadFile(path); err != nil || !bytes.Equal(got, raw) {
					t.Errorf("staged contents = %q, %v", got, err)
				}
			} else {
				if !errors.Is(err, wantErr) {
					t.Errorf("stage error = %v, want %v", err, wantErr)
				}
				if entries, err := os.ReadDir(dir); err != nil || len(entries) != 0 {
					t.Errorf("failed staging left files: %v, %v", entries, err)
				}
			}
			for _, f := range opened {
				if f.closes != 1 {
					t.Errorf("%s closed %d times, want once", f.kind, f.closes)
				}
			}
			if !syscall.ForkLock.TryLock() {
				t.Fatal("stage left fork excluded")
			}
			syscall.ForkLock.Unlock()
		})
	}
}

func TestStageBinaryExecuteOnlyMode(t *testing.T) {
	path, err := stageBinary(t.TempDir(), "image-*", []byte("image"), 0111)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0111 {
		t.Fatalf("execute-only staged image: %v, %v", info, err)
	}
}

func TestStageBinarySlowSyncAllowsExec(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	staged := make(chan error, 1)
	dir := t.TempDir()
	go func() {
		_, err := stageBinaryWithFiles(dir, "image-*", []byte("image"), 0755, binaryStageFiles{
			createTemp: func(dir, pattern string) (binaryStageFile, error) {
				return os.CreateTemp(dir, pattern)
			},
			openReadOnly: func(path string) (binaryStageFile, error) {
				f, err := os.Open(path)
				if err != nil {
					return nil, err
				}
				return &stageTestFile{File: f, kind: "reader", before: func(op string) error {
					if op == "sync" {
						close(entered)
						<-release
					}
					return nil
				}}, nil
			},
		})
		staged <- err
	}()
	select {
	case <-entered:
	case err := <-staged:
		t.Fatalf("staging ended before sync: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	started := make(chan error, 1)
	go func() { started <- exec.CommandContext(ctx, "/bin/true").Run() }()
	select {
	case err := <-started:
		if err != nil {
			t.Error(err)
		}
	case <-ctx.Done():
		t.Error("subprocess creation blocked behind the file sync")
	}
	close(release)
	if err := <-staged; err != nil {
		t.Fatal(err)
	}
}
