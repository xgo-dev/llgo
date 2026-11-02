//go:build linux && (amd64 || arm64)

package cgo

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Exercise the real entry and library constructors. The fixture enables only
// the private transport, with one controlled host thread and no foreign thread
// creators. Public APIs remain behind their libc/cgo ENOTSUP guard.
func TestAllThreadsMainThreadOwnership(t *testing.T) {
	dir, err := filepath.Abs("testdata/allthreadsownership")
	if err != nil {
		t.Fatal(err)
	}
	flags, err := exec.Command("pkg-config", "--libs", "bdw-gc", "libffi").CombinedOutput()
	if err != nil {
		t.Fatalf("native library flags: %v\n%s", err, flags)
	}
	if unwind, err := exec.Command("pkg-config", "--libs", "libunwind").Output(); err == nil {
		flags = append(flags, unwind...)
	}
	for _, mode := range []string{"exe", "c-shared", "c-archive"} {
		t.Run(mode, func(t *testing.T) {
			output := filepath.Join(t.TempDir(), "owned")
			if mode == "c-shared" {
				output += ".so"
			} else if mode == "c-archive" {
				output += ".a"
			}
			buildCExportFixture(t, dir, output, mode)
			binary, want := output, "allthreads-owned-main-ok"
			if mode != "exe" {
				binary, want = output+".host", "allthreads-adopted-host-ok"
				object := output + ".host.o"
				if out, err := exec.Command("cc", "-Wall", "-Wextra", "-Werror", "-c", filepath.Join(dir, "host/main.c"), "-o", object).CombinedOutput(); err != nil {
					t.Fatalf("compile host: %v\n%s", err, out)
				}
				args := []string{object, output, "-o", binary, "-pthread", "-ldl", "-lm", "-lresolv", "-Wl,-rpath," + filepath.Dir(output)}
				args = append(args, strings.Fields(string(flags))...)
				if out, err := exec.Command("clang++", args...).CombinedOutput(); err != nil {
					t.Fatalf("link host: %v\n%s", err, out)
				}
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			cmd := exec.CommandContext(ctx, binary)
			cmd.Env = append(os.Environ(), "GC_MARKERS=1")
			if out, err := cmd.CombinedOutput(); err != nil || !strings.Contains(string(out), want) {
				t.Fatalf("%s main-thread ownership: %v\n%s", mode, err, out)
			}
		})
	}
}
