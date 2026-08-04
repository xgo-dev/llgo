//go:build !llgo

package gotest

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

const largeArrayReturnProbe = `package main

const size = 128*1024 + 1

type params struct { x, y, z int }

func main() {
	a := f(1, 99)
	b := g(size-1, 98)
	c := h(size-1, 98)
	d := withAggregateParam(params{1, 2, 3})
	println(a[1], b[1], c[1], a[size-1], b[size-1], c[size-1], d[0])
	println(f(1, 97)[1])
}

//go:noinline
func f(i, y int) (a [size]byte) {
	a[i] = byte(y)
	return
}

//go:noinline
func g(i, y int) [size]byte {
	var a [size]byte
	a[i] = byte(y)
	return a
}

//go:noinline
func h(i, y int) (a [size]byte) {
	a[i] = byte(y)
	return a
}

//go:noinline
func withAggregateParam(p params) (a [size]byte) {
	a[0] = byte(p.x + p.y + p.z)
	return
}
`

var (
	largeArrayLLGoOnce sync.Once
	largeArrayLLGoBin  string
	largeArrayLLGoErr  string
)

func largeArrayLLGo(t *testing.T) string {
	t.Helper()
	root := findRepoRoot(t)
	t.Setenv("LLGO_ROOT", root)
	largeArrayLLGoOnce.Do(func() {
		dir, err := os.MkdirTemp("", "llgo-large-array-bin")
		if err != nil {
			largeArrayLLGoErr = err.Error()
			return
		}
		largeArrayLLGoBin = filepath.Join(dir, "llgo")
		cmd := exec.Command("go", "build", "-tags", "dev", "-o", largeArrayLLGoBin, "./cmd/llgo")
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			largeArrayLLGoErr = fmt.Sprintf("%v\n%s", err, out)
		}
	})
	if largeArrayLLGoErr != "" {
		t.Fatalf("building llgo failed: %s", largeArrayLLGoErr)
	}
	return largeArrayLLGoBin
}

func TestLargeArrayReturnAllABIModes(t *testing.T) {
	testLargeArrayReturn(t, nil, 0, 1, 2)
}

func TestLargeArrayReturnDWARF(t *testing.T) {
	testLargeArrayReturn(t, []string{"-ldflags=-w=false"}, 2)
}

func testLargeArrayReturn(t *testing.T, buildFlags []string, modes ...int) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module largearray\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(largeArrayReturnProbe), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := largeArrayLLGo(t)
	for _, mode := range modes {
		t.Run(fmt.Sprintf("abi%d", mode), func(t *testing.T) {
			args := []string{"run", fmt.Sprintf("-abi=%d", mode)}
			args = append(args, buildFlags...)
			args = append(args, ".")
			cmd := exec.Command(bin, args...)
			cmd.Dir = dir
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("llgo run failed: %v\n%s", err, out)
			}
			lines := strings.Split(strings.TrimSpace(string(out)), "\n")
			if len(lines) < 2 {
				t.Fatalf("output has fewer than two lines: %q", out)
			}
			if got, want := strings.Join(lines[len(lines)-2:], "\n"), "99 0 0 0 98 98 6\n97"; got != want {
				t.Fatalf("output = %q, want %q", got, want)
			}
		})
	}
}
