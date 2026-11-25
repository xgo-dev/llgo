package build

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Run the real arena sizing and segment layout against a fake libc allocator.
// This covers every start alignment and overflow without reserving huge heaps.
func TestWasmGCArenaCapacity(t *testing.T) {
	fset := token.NewFileSet()
	var source bytes.Buffer
	source.WriteString(gcArenaTestSource)
	for _, name := range []string{"gc_segment_wasi_threads.go", "gc_wasm.go", "segments.go", "segment_lookup.go", "gc_tinygo.go"} {
		file, err := parser.ParseFile(fset, filepath.Join("..", "..", "runtime", "internal", "runtime", "tinygogc", name), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.IMPORT {
				continue
			}
			if name != "segments.go" && name != "segment_lookup.go" {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || (fn.Name.Name != "newHeapSegment" && fn.Name.Name != "alignUp" && fn.Name.Name != "alignDown" && fn.Name.Name != "growHeapWithWorldStopped") {
					continue
				}
			}
			if err := format.Node(&source, fset, decl); err != nil {
				t.Fatal(err)
			}
			source.WriteByte('\n')
		}
	}
	path := filepath.Join(t.TempDir(), "arena_test.go")
	if err := os.WriteFile(path, source.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("go", "test", "-count=1", "-timeout=20s", path).CombinedOutput(); err != nil {
		t.Fatalf("arena capacity regression: %v\n%s", err, out)
	}
}

const gcArenaTestSource = `package arena
import ("testing"; "unsafe")
const blocksPerStateByte = 4
const wasmPageSize = uintptr(65536)
var bytesPerBlock = uintptr(32)
var endBlock, arenaStart, arenaSize uintptr
var segmentedHeap = true
var stopped, resumed, grown int
var stopAllowed bool
func gcStopWorld() bool { stopped++; return stopAllowed }
func gcResumeWorld() { resumed++ }
func growHeap(uintptr) bool { grown++; return true }
var c = struct {
 Str func(string) string
 Memset func(unsafe.Pointer, int, uintptr) unsafe.Pointer
}{func(s string) string { return s }, func(p unsafe.Pointer, _ int, _ uintptr) unsafe.Pointer { return p }}
func gcPanic(s string) { panic(s) }
func gcWasmNewArena(size uintptr) uintptr { arenaSize = size; return arenaStart }
func TestCapacity(t *testing.T) {
 for _, block := range []uintptr{16, 32} {
  bytesPerBlock = block
  for _, nominal := range []uintptr{1<<20, 2<<20, 4<<20, 8<<20, 16<<20, 32<<20, 64<<20} {
   capacity := (nominal-(nominal+block*4)/(1+block*4))/block*block
   for _, minimum := range []uintptr{capacity-block, capacity, capacity+block, nominal-block, nominal, nominal+block} {
    for offset := uintptr(0); offset < block; offset++ {
     for _, count := range []int{0, 1, 5, 127} {
      heapSegmentCount = count
      arenaStart, arenaSize = 65536+offset, 0
      start, end := newHeapSegment(minimum)
      if start == 0 || start < arenaStart || end > arenaStart+arenaSize { t.Fatal("invalid arena bounds") }
      heapSegmentCount, endBlock = 0, 0
      if !addHeapSegment(start, end) { t.Fatal("arena rejected") }
      if usable := heapUsableSize(); usable < minimum {
       t.Fatalf("block=%d nominal=%d minimum=%d offset=%d count=%d: usable=%d arena=%d", block, nominal, minimum, offset, count, usable, arenaSize)
      }
     }
    }
   }
  }
 }
}
func TestAllocationFailure(t *testing.T) {
 arenaStart = 0
 if start, end := newHeapSegment(32<<20); start != 0 || end != 0 { t.Fatal("malloc failure ignored") }
}
func TestOverflow(t *testing.T) {
 for _, minimum := range []uintptr{^uintptr(0), ^uintptr(0)-wasmPageSize, ^uintptr(0)-wasmPageSize*2} {
  arenaStart, arenaSize = 65536, 0
  if start, end := newHeapSegment(minimum); start != 0 || end != 0 || arenaSize != 0 { t.Fatal("overflow reached allocator") }
 }
}
func TestGrowthRendezvous(t *testing.T) {
 for _, segmented := range []bool{false, true} {
  for _, allowed := range []bool{false, true} {
   segmentedHeap, stopAllowed = segmented, allowed
   stopped, resumed, grown = 0, 0, 0
   if !growHeapWithWorldStopped(32<<20) || grown != 1 { t.Fatal("growth lost") }
   wantStops, wantResumes := 0, 0
   if !segmented { wantStops = 1; if allowed { wantResumes = 1 } }
   if stopped != wantStops || resumed != wantResumes { t.Fatalf("segmented=%v allowed=%v: stopped=%d resumed=%d", segmented, allowed, stopped, resumed) }
  }
 }
}
`
