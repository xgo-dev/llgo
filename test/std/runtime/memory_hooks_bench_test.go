package runtime_test

import (
	"bytes"
	"runtime"
	"strconv"
	"testing"
	"unsafe"
)

var memoryHookBenchmarkSink uint64

// These are hot-buffer, single-threaded hook benchmarks. Each operation also
// includes one byte store that changes its input, preventing a repeated copy
// or clear of constant data from becoming a misleading compiler-elision case.
// Setup, full-buffer correctness checks, and the result sink are not timed.
// SetBytes counts the requested copy/clear length, not total memory traffic.
var memoryHookWholeSizes = []int{8, 16, 64, 256, 4096, 65536, 1048576}
var memoryHookTailSizes = []int{1, 7, 9, 65, 257, 4097, 65537}

func memoryHookPattern(index int) byte { return byte(index*31 + 7) }

func memoryHookBuffer(size int) ([]byte, int) {
	word := int(unsafe.Sizeof(uintptr(0)))
	buffer := make([]byte, 2*size+8*word+16)
	for index := range buffer {
		buffer[index] = memoryHookPattern(index)
	}
	// Preserve a guard before the first range, and align relative to the actual
	// allocation rather than assuming a byte slice's alignment on either host.
	misalignment := int(uintptr(unsafe.Pointer(&buffer[0])) % uintptr(word))
	base := word + (word-misalignment)%word
	return buffer, base
}

func memoryHookConsume(buffer []byte) {
	var sum uint64
	for _, value := range buffer {
		sum += uint64(value)
	}
	memoryHookBenchmarkSink = sum
	runtime.KeepAlive(buffer)
}

func BenchmarkMemoryMoveHook(b *testing.B) {
	for _, layout := range []string{
		"Aligned", "AlignedTail", "Unaligned", "OverlapForward", "OverlapBackward",
		"UnalignedOverlapForward", "UnalignedOverlapBackward", "SameAddress",
	} {
		b.Run(layout, func(b *testing.B) {
			sizes := memoryHookWholeSizes
			if layout == "AlignedTail" {
				sizes = memoryHookTailSizes
			}
			for _, size := range sizes {
				word := int(unsafe.Sizeof(uintptr(0)))
				if (layout == "OverlapForward" || layout == "OverlapBackward") && size <= word {
					continue // A word of separation would not overlap this size.
				}
				b.Run("bytes_"+strconv.Itoa(size), func(b *testing.B) {
					benchmarkMemoryMove(b, layout, size)
				})
			}
		})
	}
}

func benchmarkMemoryMove(b *testing.B, layout string, size int) {
	buffer, base := memoryHookBuffer(size)
	word := int(unsafe.Sizeof(uintptr(0)))
	source, destination := base, base+(size+word-1)/word*word+word
	switch layout {
	case "Unaligned":
		source++
		destination += 3
	case "OverlapForward":
		source, destination = base+word, base
	case "OverlapBackward":
		source, destination = base, base+word
	case "UnalignedOverlapForward":
		source, destination = base+3, base+1
	case "UnalignedOverlapBackward":
		source, destination = base+1, base+3
	case "SameAddress":
		destination = source
	}
	src, dst := unsafe.Pointer(&buffer[source]), unsafe.Pointer(&buffer[destination])
	length := uintptr(size)
	move := runtimeMove // Address-taken invocation used by runtime consumers.

	// Use the untouched source to construct the expected result even for
	// overlapping ranges; the oracle does not call either memory hook.
	want := make([]byte, len(buffer))
	for index := range buffer {
		want[index] = buffer[index]
	}
	for index := 0; index < size; index++ {
		want[destination+index] = buffer[source+index]
	}
	move(dst, src, length)
	if !bytes.Equal(buffer, want) {
		b.Fatal("memmove failed the layout correctness check")
	}

	b.SetBytes(int64(size))
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		buffer[source] = byte(index)
		move(dst, src, length)
	}
	b.StopTimer()
	if b.N > 0 && buffer[destination] != byte(b.N-1) {
		b.Fatal("memmove did not copy the final changing input byte")
	}
	for index, value := range buffer {
		if (index < destination || index >= destination+size) && index != source && value != memoryHookPattern(index) {
			b.Fatalf("memmove changed guard byte %d", index)
		}
	}
	memoryHookConsume(buffer)
}

func BenchmarkMemoryClearHook(b *testing.B) {
	for _, layout := range []string{"Aligned", "AlignedTail", "Unaligned"} {
		b.Run(layout, func(b *testing.B) {
			sizes := memoryHookWholeSizes
			if layout == "AlignedTail" {
				sizes = memoryHookTailSizes
			}
			for _, size := range sizes {
				b.Run("bytes_"+strconv.Itoa(size), func(b *testing.B) {
					buffer, offset := memoryHookBuffer(size)
					if layout == "Unaligned" {
						offset++
					}
					ptr := unsafe.Pointer(&buffer[offset])
					length := uintptr(size)
					clearMemory := runtimeClear
					b.SetBytes(int64(size))
					b.ReportAllocs()
					b.ResetTimer()
					for index := 0; index < b.N; index++ {
						buffer[offset] = byte(index) | 1 // Always dirty before clearing.
						clearMemory(ptr, length)
					}
					b.StopTimer()
					for index, value := range buffer {
						want := memoryHookPattern(index)
						if index >= offset && index < offset+size {
							want = 0
						}
						if value != want {
							b.Fatalf("memclr changed the wrong value at byte %d", index)
						}
					}
					memoryHookConsume(buffer)
				})
			}
		})
	}
}

func BenchmarkMemoryHookZero(b *testing.B) {
	b.Run("move", func(b *testing.B) {
		move := runtimeMove
		b.ReportAllocs()
		b.ResetTimer()
		for index := 0; index < b.N; index++ {
			move(nil, nil, 0)
		}
	})
	b.Run("clear", func(b *testing.B) {
		clearMemory := runtimeClear
		b.ReportAllocs()
		b.ResetTimer()
		for index := 0; index < b.N; index++ {
			clearMemory(nil, 0)
		}
	})
}
