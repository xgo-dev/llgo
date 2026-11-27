//go:build wasm && !nogc

package wasmtest

import (
	"runtime"
	"testing"
)

type gcBitmapNode struct {
	value int
	data  []byte
	next  *gcBitmapNode
}

//go:noinline
func dirtyGCBitmapAllocation(size int) {
	data := make([]byte, size)
	for i := range data {
		data[i] = 0x7d
	}
	runtime.KeepAlive(data)
}

func TestGCBitmapSpans(t *testing.T) {
	// A wide graph forces the bounded mark stack to overflow. Varied object
	// lengths mix heads, tails and free space across metadata-byte boundaries.
	sizes := [...]int{1, 31, 32, 33, 95, 127, 128, 129, 255, 256, 257, 513}
	nodes := make([]*gcBitmapNode, 192)
	for i := range nodes {
		size := sizes[i%len(sizes)]
		dirtyGCBitmapAllocation(size)
		leaf := &gcBitmapNode{value: i + 1, data: make([]byte, size)}
		for j := range leaf.data {
			leaf.data[j] = byte(i + j)
		}
		nodes[i] = &gcBitmapNode{next: leaf}
	}
	for round := 0; round < 3; round++ {
		runtime.GC()
		for i, node := range nodes {
			if node.next == nil || node.next.value != i+1 {
				t.Fatalf("round %d: graph edge %d was lost", round, i)
			}
			for j, value := range node.next.data {
				if value != byte(i+j) {
					t.Fatalf("round %d: live object %d byte %d changed", round, i, j)
				}
			}
			dirtyGCBitmapAllocation(sizes[i%len(sizes)])
			for j, value := range make([]byte, sizes[i%len(sizes)]) {
				if value != 0 {
					t.Fatalf("round %d: new object %d byte %d was not zeroed", round, i, j)
				}
			}
		}
	}
}
