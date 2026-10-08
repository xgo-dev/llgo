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

// Exercise the collector's real metadata code over out-of-order arenas without
// letting the host test's own collector share that metadata.
func TestWasmGCSegmentBoundariesAndSweep(t *testing.T) {
	fset := token.NewFileSet()
	var source bytes.Buffer
	source.WriteString(gcSegmentWorkloadSource)
	functions := map[string]bool{"gcAddressOfIn": true, "gcStateByteOfIn": true,
		"gcStateFromByteIn": true, "gcStateOfIn": true, "gcSetStateIn": true,
		"gcMarkFreeIn": true, "gcUnmarkIn": true, "gcFindNextIn": true, "sweep": true}
	for _, name := range []string{"segments.go", "gc_tinygo.go"} {
		file, err := parser.ParseFile(fset, filepath.Join("..", "..", "runtime", "internal", "runtime", "tinygogc", name), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.IMPORT {
				continue
			}
			if name == "gc_tinygo.go" {
				switch node := decl.(type) {
				case *ast.FuncDecl:
					if !functions[node.Name.Name] {
						continue
					}
					delete(functions, node.Name.Name)
				case *ast.GenDecl:
					if node.Tok != token.CONST {
						continue
					}
					first := node.Specs[0].(*ast.ValueSpec).Names[0].Name
					if first != "blockStateByteAllTails" && first != "blocksPerStateWord" {
						continue
					}
				default:
					continue
				}
			}
			if err := format.Node(&source, fset, decl); err != nil {
				t.Fatal(err)
			}
			source.WriteByte('\n')
		}
	}
	if len(functions) != 0 {
		t.Fatalf("missing collector functions: %v", functions)
	}
	path := filepath.Join(t.TempDir(), "segments_test.go")
	if err := os.WriteFile(path, source.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("go", "test", "-count=1", "-timeout=20s", path).CombinedOutput(); err != nil {
		t.Fatalf("segment metadata regression: %v\n%s", err, out)
	}
}

const gcSegmentWorkloadSource = `package segments
import ("runtime"; "testing"; "unsafe")
const (
 blockStateFree uint8 = iota
 blockStateHead
 blockStateTail
 blockStateMark
 blockStateMask = 3
 stateBits = 2
 blocksPerStateByte = 4
 wordsPerBlock = 4
 bytesPerBlock = wordsPerBlock * unsafe.Sizeof(uintptr(0))
)
var segmentedHeap = true
var endBlock uintptr
var gcFrees, gcFreedBlocks uint64
var c = struct {
 Str func(string) string
 Memset func(unsafe.Pointer, int, uintptr) unsafe.Pointer
}{func(s string) string { return s }, func(p unsafe.Pointer, value int, n uintptr) unsafe.Pointer {
 for i := uintptr(0); i < n; i++ { *(*byte)(unsafe.Add(p, i)) = byte(value) }; return p
}}
func gcPanic(s string) { panic(s) }
func resetHeap() {
 heapSegmentCount = 0
 heapSegments = [maxHeapSegments]heapSegment{}
 heapSegmentsByAddress = [maxHeapSegments]*heapSegment{}
 endBlock, gcFrees, gcFreedBlocks = 0, 0, 0
}
func TestAllSegments(t *testing.T) {
 resetHeap()
 const stride = 4096
 data := make([]byte, maxHeapSegments*stride)
 base := uintptr(unsafe.Pointer(&data[0]))
 for i := 0; i < maxHeapSegments; i++ {
  // Force non-address-order insertion and holes between malloc-like arenas.
  start := base + uintptr((i*37)%maxHeapSegments)*stride
  if !addHeapSegment(start, start+stride-64) { t.Fatalf("segment %d rejected", i) }
 }
 for i := 0; i < heapSegmentCount; i++ {
  segment := &heapSegments[i]
  for block := segment.first; block <= segment.last; block++ {
   if segmentForBlock(block) != segment { t.Fatalf("block %d crosses segment", block) }
  }
  for _, address := range []uintptr{segment.start, segment.metadata-1} {
   if segmentForAddress(address) != segment { t.Fatalf("address %#x not in segment %d", address, i) }
  }
  for _, address := range []uintptr{segment.start-1, segment.metadata, segment.end-1, segment.end} {
   if segmentForAddress(address) != nil { t.Fatalf("gap/metadata %#x treated as a root", address) }
  }
  for block := segment.first; block < segment.last; block++ {
   gcSetStateIn(segment, block, blockStateHead)
   *(*uintptr)(unsafe.Pointer(gcAddressOfIn(segment, block))) = 123
  }
  gcSetStateIn(segment, segment.first, blockStateMark)
 }
 sweep()
 for i := 0; i < heapSegmentCount; i++ {
  segment := &heapSegments[i]
  for block := segment.first; block < segment.last; block++ {
   want := uint8(blockStateFree)
   if block == segment.first { want = blockStateHead }
   if got := gcStateOfIn(segment, block); got != want { t.Fatalf("segment %d block %d: %d != %d", i, block, got, want) }
   value := *(*uintptr)(unsafe.Pointer(gcAddressOfIn(segment, block)))
   if block == segment.first && value != 123 || block != segment.first && value != 0 { t.Fatalf("incorrect sweep payload at %d", block) }
  }
 }
 if addHeapSegment(base, base+stride) { t.Fatal("segment limit ignored") }
 runtime.KeepAlive(data)
}
func TestSweepMetadataBatches(t *testing.T) {
 for offset := uintptr(0); offset < 8; offset++ {
  for _, keep := range []bool{false, true} {
   resetHeap()
   data := make([]byte, 4096)
   base := uintptr(unsafe.Pointer(&data[0]))
   if !addHeapSegment(base, base+4096-offset) { t.Fatal("segment rejected") }
   segment := &heapSegments[0]
   gcSetStateIn(segment, 0, blockStateHead)
   if keep { gcSetStateIn(segment, 0, blockStateMark) }
   for block := uintptr(1); block < 40; block++ { gcSetStateIn(segment, block, blockStateTail) }
   for block := uintptr(0); block < 40; block++ {
    *(*uintptr)(unsafe.Pointer(gcAddressOfIn(segment, block))) = 123
   }
   if next := gcFindNextIn(segment, 0); next != 40 { t.Fatalf("tail boundary: %d != 40", next) }
   // The last partial metadata byte must remain bounded and retain a live head.
   last := segment.last - 1
   gcSetStateIn(segment, last, blockStateMark)
   *(*uintptr)(unsafe.Pointer(gcAddressOfIn(segment, last))) = 456
   if next := gcFindNextIn(segment, last); next != segment.last { t.Fatalf("final boundary: %d", next) }
   free := sweep()
   live := uintptr(1)
   if keep { live += 40 }
   if free != (segment.last-live)*bytesPerBlock { t.Fatalf("incorrect free count: %d", free) }
   for block := segment.first; block < segment.last; block++ {
    state, value := uint8(blockStateFree), uintptr(0)
    if keep && block < 40 {
     state, value = blockStateTail, 123
     if block == 0 { state = blockStateHead }
    }
    if block == last { state, value = blockStateHead, 456 }
    if gcStateOfIn(segment, block) != state || *(*uintptr)(unsafe.Pointer(gcAddressOfIn(segment, block))) != value {
     t.Fatalf("offset %d keep %v block %d corrupted", offset, keep, block)
    }
   }
   if keep && (gcFrees != 0 || gcFreedBlocks != 0) || !keep && (gcFrees != 1 || gcFreedBlocks != 40) {
    t.Fatalf("incorrect freed counts: %d objects, %d blocks", gcFrees, gcFreedBlocks)
   }
   runtime.KeepAlive(data)
  }
 }
}
func TestContiguousHeapLookup(t *testing.T) {
 segmentedHeap = false
 defer func() { segmentedHeap = true }()
 resetHeap()
 data := make([]byte, 4096)
 base := uintptr(unsafe.Pointer(&data[0]))
 if !addHeapSegment(base, base+4096) { t.Fatal("segment rejected") }
 segment := &heapSegments[0]
 for block := segment.first; block <= segment.last; block++ {
  if segmentForBlock(block) != segment { t.Fatalf("block %d not found", block) }
 }
 for _, address := range []uintptr{segment.start, segment.metadata-1} {
  if segmentForAddress(address) != segment { t.Fatalf("address %#x not found", address) }
 }
 for _, address := range []uintptr{segment.start-1, segment.metadata, segment.end} {
  if segmentForAddress(address) != nil { t.Fatalf("invalid address %#x accepted", address) }
 }
 runtime.KeepAlive(data)
}
`
