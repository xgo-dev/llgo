//go:build llgo && !baremetal && !nogc

package pprof_test

import (
	"bytes"
	"compress/gzip"
	"io"
	"runtime"
	"runtime/pprof"
	"testing"
)

var heapProfileBuffers [80][]byte

//go:noinline
func allocateHeapProfileRecord(size int) []byte { return make([]byte, size) }

func TestWriteHeapProfileManyRecords(t *testing.T) {
	old := runtime.MemProfileRate
	defer func() { runtime.MemProfileRate = old; heapProfileBuffers = [80][]byte{} }()
	runtime.MemProfileRate = 1
	for i := range heapProfileBuffers {
		heapProfileBuffers[i] = allocateHeapProfileRecord(128 + i*32)
	}
	// Keep serialization allocations out of this deterministic sample set.
	runtime.MemProfileRate = 0
	var text bytes.Buffer
	if err := pprof.Lookup("allocs").WriteTo(&text, 1); err != nil {
		t.Fatal(err)
	}
	if n := bytes.Count(text.Bytes(), []byte(".allocateHeapProfileRecord+")); n < len(heapProfileBuffers) {
		t.Fatalf("serialized %d allocation records, want at least %d", n, len(heapProfileBuffers))
	}
	var binary bytes.Buffer
	if err := pprof.WriteHeapProfile(&binary); err != nil {
		t.Fatal(err)
	}
	zr, err := gzip.NewReader(&binary)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	if err := zr.Close(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte("allocateHeapProfileRecord")) {
		t.Fatal("binary heap profile lost the allocation function")
	}
	runtime.KeepAlive(heapProfileBuffers)
}
