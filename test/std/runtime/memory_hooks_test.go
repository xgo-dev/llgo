package runtime_test

import (
	"bytes"
	"runtime"
	"sync/atomic"
	"testing"
	"unsafe"
)

//go:linkname runtimeMemmove runtime.memmove
func runtimeMemmove(dst, src unsafe.Pointer, n uintptr)

//go:linkname runtimeMemclr runtime.memclrNoHeapPointers
func runtimeMemclr(ptr unsafe.Pointer, n uintptr)

// Resolve the addresses at initialization, as external runtime consumers do.
var runtimeMove = runtimeMemmove
var runtimeClear = runtimeMemclr

func TestMemoryMoveHook(t *testing.T) {
	direct := []byte{1, 2, 3, 4}
	runtimeMemmove(unsafe.Pointer(&direct[1]), unsafe.Pointer(&direct[0]), 3)
	if !bytes.Equal(direct, []byte{1, 1, 2, 3}) {
		t.Fatalf("direct memmove = %v", direct)
	}
	runtimeMemmove(nil, nil, 0)

	// Exercise direct and indirect calls, unaligned ranges, both overlap
	// directions, identical addresses, and a size larger than a machine word.
	for _, move := range []func(unsafe.Pointer, unsafe.Pointer, uintptr){runtimeMemmove, runtimeMove} {
		move(nil, nil, 0)
		for _, size := range []int{0, 1, 2, 7, 8, 15, 16, 31, 32, 65, 257} {
			for _, offset := range [][2]int{{0, 0}, {1, 0}, {0, 1}, {3, 17}, {17, 3}, {300, 0}} {
				got := make([]byte, 600)
				for i := range got {
					got[i] = byte(i*31 + 7)
				}
				want := append([]byte(nil), got...)
				dst, src := offset[0], offset[1]
				copy(want[dst:dst+size], want[src:src+size])
				move(unsafe.Pointer(&got[dst]), unsafe.Pointer(&got[src]), uintptr(size))
				if !bytes.Equal(got, want) {
					t.Fatalf("memmove dst=%d src=%d size=%d changed the wrong bytes", dst, src, size)
				}
			}
		}
	}
}

func TestMemoryClearHook(t *testing.T) {
	direct := []byte{1, 2, 3, 4}
	runtimeMemclr(unsafe.Pointer(&direct[1]), 2)
	if !bytes.Equal(direct, []byte{1, 0, 0, 4}) {
		t.Fatalf("direct memclr = %v", direct)
	}
	runtimeMemclr(nil, 0)

	for _, clearMemory := range []func(unsafe.Pointer, uintptr){runtimeMemclr, runtimeClear} {
		clearMemory(nil, 0)
		for _, size := range []int{0, 1, 2, 7, 8, 15, 16, 31, 32, 65, 257} {
			for _, offset := range []int{0, 1, 3, 17} {
				got := bytes.Repeat([]byte{0xa5}, 300)
				want := append([]byte(nil), got...)
				for i := offset; i < offset+size; i++ {
					want[i] = 0
				}
				clearMemory(unsafe.Pointer(&got[offset]), uintptr(size))
				if !bytes.Equal(got, want) {
					t.Fatalf("memclr offset=%d size=%d changed the wrong bytes", offset, size)
				}
			}
		}
	}
}

func TestMemoryHooksAlignedWords(t *testing.T) {
	word := unsafe.Sizeof(uintptr(0))
	for _, offsets := range [][2]int{{0, 0}, {0, 1}, {1, 0}, {4, 0}} {
		got := [8]uintptr{1, 2, 3, 4, 5, 6, 7, 8}
		want := got
		dst, src := offsets[0], offsets[1]
		copy(want[dst:dst+3], want[src:src+3])
		runtimeMove(unsafe.Pointer(&got[dst]), unsafe.Pointer(&got[src]), 3*word)
		if got != want {
			t.Fatalf("aligned memmove dst=%d src=%d: got %v, want %v", dst, src, got, want)
		}
	}
	got := [5]uintptr{1, 2, 3, 4, 5}
	runtimeClear(unsafe.Pointer(&got[1]), 3*word)
	if want := [5]uintptr{1, 0, 0, 0, 5}; got != want {
		t.Fatalf("aligned memclr: got %v, want %v", got, want)
	}
}

func TestMemoryHooksAlignedBoundaries(t *testing.T) {
	// Cover every remainder around several unrolled blocks, with overlap
	// distances both smaller and larger than a block, in both directions.
	// Compare the entire allocation so underruns and overruns fail too.
	word := unsafe.Sizeof(uintptr(0))
	for size := 0; size <= 65; size++ {
		offsets := [][2]int{{1, 1}, {1, size + 2}, {size + 2, 1}}
		for distance := 1; distance <= 17; distance++ {
			offsets = append(offsets, [2]int{1, 1 + distance}, [2]int{1 + distance, 1})
		}
		for _, offset := range offsets {
			var got, want [166]uintptr
			for i := range got {
				got[i] = ^uintptr(i*31 + 7)
				want[i] = got[i]
			}
			dst, src := offset[0], offset[1]
			// Read from the untouched input, independently of either hook.
			for i := 0; i < size; i++ {
				want[dst+i] = got[src+i]
			}
			runtimeMove(unsafe.Pointer(&got[dst]), unsafe.Pointer(&got[src]), uintptr(size)*word)
			if got != want {
				t.Fatalf("memmove words=%d dst=%d src=%d changed the wrong words", size, dst, src)
			}
			for i := 0; i < size; i++ {
				want[dst+i] = 0
			}
			runtimeClear(unsafe.Pointer(&got[dst]), uintptr(size)*word)
			if got != want {
				t.Fatalf("memclr words=%d offset=%d changed the wrong words", size, dst)
			}
		}
	}
}

func TestMemoryHooksAtomicWrites(t *testing.T) {
	// An atomic reader may observe either complete pattern or zero, but never
	// a partially copied/cleared pointer-sized word. The patterns work on both
	// 32-bit and 64-bit targets.
	a := ^uintptr(0) / 3
	b := ^a
	dst := a
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 10000; i++ {
			runtimeMove(unsafe.Pointer(&dst), unsafe.Pointer(&a), unsafe.Sizeof(a))
			runtimeMove(unsafe.Pointer(&dst), unsafe.Pointer(&b), unsafe.Sizeof(b))
			runtimeClear(unsafe.Pointer(&dst), unsafe.Sizeof(dst))
			if i%100 == 0 {
				runtime.Gosched()
			}
		}
	}()
	var invalid uintptr
	for i := 0; i < 10000; i++ {
		if value := atomic.LoadUintptr(&dst); value != 0 && value != a && value != b {
			invalid = value
		}
		if i%100 == 0 {
			runtime.Gosched()
		}
	}
	<-done
	if invalid != 0 {
		t.Fatalf("observed a torn memory-hook write: %#x", invalid)
	}
}

func TestMemoryHooksLargeAlignedRanges(t *testing.T) {
	word := unsafe.Sizeof(uintptr(0))
	for _, size := range []int{32, 512, 8192} {
		for _, offsets := range [][2]int{{1, 1}, {1, 2}, {2, 1}, {size + 2, 1}} {
			got := make([]uintptr, 2*size+4)
			for i := range got {
				got[i] = ^uintptr(i*31 + 7)
			}
			want := append([]uintptr(nil), got...)
			dst, src := offsets[0], offsets[1]
			copy(want[dst:dst+size], want[src:src+size])
			runtimeMove(unsafe.Pointer(&got[dst]), unsafe.Pointer(&got[src]), uintptr(size)*word)
			for i := range got {
				if got[i] != want[i] {
					t.Fatalf("memmove words=%d dst=%d src=%d: word %d = %#x, want %#x", size, dst, src, i, got[i], want[i])
				}
			}
			runtimeClear(unsafe.Pointer(&got[dst]), uintptr(size)*word)
			for i := range got {
				if i >= dst && i < dst+size {
					want[i] = 0
				}
				if got[i] != want[i] {
					t.Fatalf("memclr words=%d offset=%d: word %d = %#x, want %#x", size, dst, i, got[i], want[i])
				}
			}
		}
	}
}

func TestMemoryHooksLargeAtomicWrites(t *testing.T) {
	const words = 1024
	var a, b, dst [words]uintptr
	patternA := ^uintptr(0) / 3
	patternB := ^patternA
	for i := range a {
		a[i], b[i] = patternA, patternB
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 1000; i++ {
			runtimeMove(unsafe.Pointer(&dst[0]), unsafe.Pointer(&a[0]), unsafe.Sizeof(dst))
			runtimeMove(unsafe.Pointer(&dst[0]), unsafe.Pointer(&b[0]), unsafe.Sizeof(dst))
			runtimeClear(unsafe.Pointer(&dst[0]), unsafe.Sizeof(dst))
			if i%10 == 0 {
				runtime.Gosched()
			}
		}
	}()
	var invalid uintptr
	for i := 0; i < 1000; i++ {
		for j := range dst {
			if value := atomic.LoadUintptr(&dst[j]); value != 0 && value != patternA && value != patternB {
				invalid = value
			}
		}
		if i%10 == 0 {
			runtime.Gosched()
		}
	}
	<-done
	if invalid != 0 {
		t.Fatalf("observed a torn large memory-hook write: %#x", invalid)
	}
}
