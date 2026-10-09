package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall/js"
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/wasmworkers"
)

const LLGoFiles = "_wrap/files.c"

//go:linkname cReadWrite C.llgo_browser_fs_read_write
func cReadWrite(path *byte) int32

//go:linkname gmpForTesting github.com/xgo-dev/llgo/runtime/internal/runtime.GMPForTesting
func gmpForTesting() (goid, parentGoid uint64, mid int64, pid int32, gstatus, pstatus uint32, linked bool)

func check(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func main() {
	testFilesystem()
	// Single-worker browsers stay alive for callbacks. Only a successful
	// return after cleanup emits the marker used by their acceptance runner.
	println("wasm filesystem ok")
}

func testFilesystem() {
	dir, err := os.MkdirTemp("", "llgo-browser-fs-")
	check(err)
	defer os.RemoveAll(dir)
	name := filepath.Join(dir, "shared")
	f, err := os.OpenFile(name, os.O_CREATE|os.O_RDWR, 0600)
	check(err)
	defer f.Close()
	_, err = f.WriteString("Go file")
	check(err)
	done := make(chan int64)
	workers := make(map[int64]bool)
	multiWorker := len(os.Args) > 1 && strings.Contains(os.Args[1], "workers")
	gcStress := multiWorker && len(os.Args) > 2 && os.Args[2] == "--gc-stress"
	operations := 16
	if gcStress {
		operations = 128
	}
	for range operations {
		wasmworkers.GoIndependent(func() {
			_, _, mid, _, _, _, _ := gmpForTesting()
			if gcStress {
				runtime.GC()
			}
			contents, err := os.ReadFile(name)
			check(err)
			if string(contents) != "Go file" {
				panic("workers have different filesystems")
			}
			buf := make([]byte, 7)
			_, err = f.ReadAt(buf, 0)
			check(err)
			if string(buf) != "Go file" {
				panic("file descriptor is not shared")
			}
			st, err := f.Stat()
			check(err)
			if st.Size() != 7 {
				panic("wrong fstat size")
			}
			done <- mid
		})
	}
	for range operations {
		workers[<-done] = true
	}
	if multiWorker && len(workers) < 2 {
		panic("filesystem test did not exercise distinct workers")
	}
	if _, err = os.Stat(filepath.Join(dir, "missing")); !os.IsNotExist(err) {
		panic("ENOENT was lost")
	}
	oldwd, err := os.Getwd()
	check(err)
	wasmworkers.GoIndependent(func() { check(os.Chdir(dir)); done <- 0 })
	<-done
	wd, err := os.Getwd()
	check(err)
	if wd != dir {
		// Node resolves /var to /private/var on macOS.
		actual, err := os.Stat(wd)
		check(err)
		want, err := os.Stat(dir)
		check(err)
		if !os.SameFile(actual, want) {
			panic("cwd is not shared: " + wd + " != " + dir)
		}
	}
	check(os.Chdir(oldwd))

	// Browser Go and C use the same Emscripten filesystem. In Node, Go uses
	// node:fs and C keeps Emscripten's own filesystem, as before.
	if js.Global().Get("process").Get("platform").String() == "browser" {
		path := append([]byte(name), 0)
		if cReadWrite((*byte)(unsafe.Pointer(&path[0]))) != 0 {
			panic("C cannot read Go file")
		}
		contents, err := os.ReadFile(name)
		check(err)
		if string(contents) != "C file" {
			panic("Go cannot read C file")
		}
	}
	check(os.Rename(name, name+".renamed"))
	check(os.Remove(name + ".renamed"))
	// fstat is valid after unlink, so it must inspect the stream's node.
	_, err = f.Stat()
	check(err)
	runtime.GC()
}
