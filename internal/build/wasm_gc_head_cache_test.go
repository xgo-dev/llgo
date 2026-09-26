package build

import (
	"bytes"
	stdctx "context"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// Run the actual collector lookup functions against isolated metadata. This
// exercises eviction and compares cached/uncached costs without mutating the
// collector running the test. Timings describe host-compiled metadata lookup,
// not end-to-end Wasm collections, and are deliberately not pass/fail limits.
func TestWasmGCHeadCacheWorkloads(t *testing.T) {
	fset := token.NewFileSet()
	dir := filepath.Join("..", "..", "runtime", "internal", "runtime", "tinygogc")
	functions := map[string]bool{
		"gcFindHead": true, "gcFindHeadForMark": true,
		"gcStateByteOf": true, "gcStateFromByte": true, "gcStateOf": true,
	}
	var source bytes.Buffer
	source.WriteString(gcHeadCacheWorkloadSource)
	for _, name := range []string{"gc_tinygo.go", "head_cache.go"} {
		file, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			if name == "gc_tinygo.go" {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || !functions[fn.Name.Name] {
					continue
				}
				delete(functions, fn.Name.Name)
			}
			if err := format.Node(&source, fset, decl); err != nil {
				t.Fatal(err)
			}
			source.WriteByte('\n')
		}
	}
	if len(functions) != 0 {
		t.Fatalf("missing collector operations: %v", functions)
	}
	path := filepath.Join(t.TempDir(), "head_cache_test.go")
	if err := os.WriteFile(path, source.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := stdctx.WithTimeout(stdctx.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "test", "-count=1", "-timeout=25s", "-bench=BenchmarkHeadLookup", "-benchtime=50ms", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("collector workload checks: %v\n%s", err, output)
	}
	t.Logf("actual metadata lookup workloads:\n%s", output)
}

const gcHeadCacheWorkloadSource = `package headcache
import (
  "fmt"
  "runtime"
  "testing"
  "unsafe"
)
const (
  blockStateFree uint8 = iota
  blockStateHead
  blockStateTail
  blockStateMark
  blockStateMask = 3
  blockStateByteAllTails = 0xaa
  stateBits = 2
  blocksPerStateByte = 4
)
var metadataStart unsafe.Pointer
type heapSegment struct { first, metadata uintptr }
var testSegment heapSegment
func segmentForBlock(block uintptr) *heapSegment { return &testSegment }
var markHeads markHeadCache
var c = struct { Str func(string) string }{func(s string) string { return s }}
func gcPanic(message string) { panic(message) }

func makeMetadataAt(objects int, blocks, first uintptr) []byte {
  data := make([]byte, (uintptr(objects)*blocks+3)/4)
  for i := range data { data[i] = blockStateByteAllTails }
  for object := 0; object < objects; object++ {
    head := uintptr(object)*blocks
    // Cover both allocated and already-marked heads.
    state := blockStateHead
    if object%2 != 0 { state = blockStateMark }
    data[head/4] = data[head/4] &^ (3 << ((head%4)*2)) | state << ((head%4)*2)
  }
  metadataStart = unsafe.Pointer(&data[0])
  testSegment.first = first
  testSegment.metadata = uintptr(metadataStart)
  markHeads.reset()
  return data
}

func makeMetadata(objects int, blocks uintptr) []byte {
  return makeMetadataAt(objects, blocks, 0)
}

func TestInterleavedHeads(t *testing.T) {
  for _, blocks := range []uintptr{1, 4, 16, 256, 4096} {
    for _, objects := range []int{1, 4, 5, 10, 20} {
      data := makeMetadata(objects, blocks)
      for round := 0; round < 8; round++ {
        for object := 0; object < objects; object++ {
          head := uintptr(object)*blocks
          for _, offset := range []uintptr{0, blocks/2, blocks-1} {
            block := head+offset
            if got := gcFindHeadForMark(block); got != head || got != gcFindHead(block) {
              t.Fatalf("blocks=%d objects=%d round=%d block=%d head=%d want=%d", blocks, objects, round, block, got, head)
            }
          }
        }
      }
      markHeads.reset()
      runtime.KeepAlive(data)
    }
  }
}

func TestHeadsAfterUnalignedSegmentBoundary(t *testing.T) {
  for _, first := range []uintptr{1, 2, 3} {
    data := makeMetadataAt(3, 256, first)
    for object := uintptr(0); object < 3; object++ {
      head := first + object*256
      for _, offset := range []uintptr{0, 1, 127, 255} {
        if got := gcFindHeadForMark(head+offset); got != head {
          t.Fatalf("first=%d object=%d offset=%d: head=%d, want %d", first, object, offset, got, head)
        }
      }
    }
    runtime.KeepAlive(data)
  }
}

var lookupSink uintptr
func BenchmarkHeadLookup(b *testing.B) {
  for _, workload := range []struct { name string; objects int; blocks, offset uintptr }{
    {"small-heads", 256, 1, 0},
    {"small-tails", 256, 16, 15},
    // 327680 blocks represent a 5 MiB object in the wasm32 collector.
    {"large-1", 1, 327680, 327679},
    {"large-4", 4, 327680, 327679},
    {"large-5", 5, 327680, 327679},
    {"large-10", 10, 327680, 327679},
    {"large-20", 20, 327680, 327679},
  } {
    for _, cached := range []bool{false, true} {
      b.Run(fmt.Sprintf("%s/cached=%v", workload.name, cached), func(b *testing.B) {
        data := makeMetadata(workload.objects, workload.blocks)
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
          head := uintptr(i%workload.objects)*workload.blocks
          block := head+workload.offset
          if cached { lookupSink = gcFindHeadForMark(block) } else { lookupSink = gcFindHead(block) }
        }
        b.StopTimer()
        markHeads.reset()
        runtime.KeepAlive(data)
      })
    }
  }
}
`
