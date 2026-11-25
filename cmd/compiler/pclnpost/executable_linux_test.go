package pclnpost

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
)

// These lock checks must not run in parallel with tests that start processes:
// they inspect the process-global ForkLock, not a package-local mutex.
func TestExecutableWriteExcludesFork(t *testing.T) {
	unlock := lockExecutableWrite()
	if syscall.ForkLock.TryLock() {
		syscall.ForkLock.Unlock()
		unlock()
		t.Fatal("executable writes do not exclude fork")
	}
	// Writers share the read side; only starting a child must be excluded.
	if !syscall.ForkLock.TryRLock() {
		unlock()
		t.Fatal("executable writes cannot overlap")
	}
	syscall.ForkLock.RUnlock()
	unlock()
	if !syscall.ForkLock.TryLock() {
		t.Fatal("fork remains excluded after the executable is closed")
	}
	syscall.ForkLock.Unlock()
}

func TestStageBinaryReleasesForkLock(t *testing.T) {
	for _, name := range []string{"success", "create-error"} {
		t.Run(name, func(t *testing.T) {
			fail := name == "create-error"
			dir := t.TempDir()
			if fail {
				dir = filepath.Join(dir, "missing")
			}
			path, err := stageBinary(dir, "image-*", []byte("image"), 0751)
			if (err != nil) != fail {
				t.Fatalf("stageBinary error = %v, want failure %v", err, fail)
			}
			if !syscall.ForkLock.TryLock() {
				t.Fatal("stageBinary left fork excluded")
			}
			syscall.ForkLock.Unlock()
			if !fail {
				data, err := os.ReadFile(path)
				if err != nil || string(data) != "image" {
					t.Fatalf("staged image = %q, %v", data, err)
				}
				info, err := os.Stat(path)
				if err != nil {
					t.Fatal(err)
				}
				if info.Mode().Perm() != 0751 {
					t.Fatalf("staged mode = %v, want 0751", info.Mode())
				}
			}
		})
	}
}

func TestReplaceBinaryConcurrentExec(t *testing.T) {
	// Exercise the production publisher while other workers start binaries.
	// Use a script to check contents and execution, not just successful exec.
	const workers, iterations = 8, 16
	raw := []byte("#!/bin/sh\nprintf 'published executable\\n'\n")
	dir := t.TempDir()
	paths := make([]string, workers)
	for i := range paths {
		paths[i] = filepath.Join(dir, fmt.Sprintf("image-%d", i))
		if err := os.WriteFile(paths[i], raw, 0755); err != nil {
			t.Fatal(err)
		}
	}
	var wg sync.WaitGroup
	for _, path := range paths {
		wg.Go(func() {
			for range iterations {
				err := replaceBinary(path, raw, false, func(staged string) error {
					// Verification must run after the write guard is released:
					// a verifier (or codesign on macOS) may start a process.
					output, err := exec.Command("/bin/sh", "-n", staged).CombinedOutput()
					if err != nil {
						return fmt.Errorf("verify: %v: %s", err, output)
					}
					return nil
				})
				if err != nil {
					t.Error(err)
					return
				}
				output, err := exec.Command(path).CombinedOutput()
				if err != nil || !bytes.Equal(output, []byte("published executable\n")) {
					t.Errorf("execute published image: %v, output %q", err, output)
					return
				}
			}
		})
	}
	wg.Wait()
}
