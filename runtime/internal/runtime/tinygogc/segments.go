//go:build (baremetal && !nogc) || (wasm && llgo.wasm.gc.linear)

package tinygogc

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

// Every arena owns its block-state bytes. A sentinel block separates adjacent
// arenas in the logical block index, so allocations never cross libc regions.
type heapSegment struct {
	start, end  uintptr
	metadata    uintptr
	first, last uintptr // [first, last) are usable blocks; last is the sentinel
}

const maxHeapSegments = 128

var heapSegments [maxHeapSegments]heapSegment
var heapSegmentCount int

func addHeapSegment(start, end uintptr) bool {
	if heapSegmentCount == maxHeapSegments || start >= end {
		return false
	}
	if heapSegmentCount != 0 && endBlock == ^uintptr(0) {
		return false
	}
	segment := &heapSegments[heapSegmentCount]
	segment.start, segment.end = start, end
	if heapSegmentCount != 0 {
		segment.first = endBlock + 1
	}
	totalSize := end - start
	metadataSize := (totalSize + blocksPerStateByte*bytesPerBlock) / (1 + blocksPerStateByte*bytesPerBlock)
	segment.metadata = end - metadataSize
	segment.last = segment.first + (segment.metadata-start)/bytesPerBlock
	if segment.last <= segment.first || segment.last < segment.first {
		return false
	}
	c.Memset(unsafe.Pointer(segment.metadata), 0, end-segment.metadata)
	heapSegmentCount++
	endBlock = segment.last
	return true
}

func segmentForBlock(block uintptr) *heapSegment {
	for index := 0; index < heapSegmentCount; index++ {
		segment := &heapSegments[index]
		if block >= segment.first && block <= segment.last {
			return segment
		}
	}
	gcPanic(c.Str("gc: invalid heap block"))
	return nil
}

func segmentForAddress(address uintptr) *heapSegment {
	for index := 0; index < heapSegmentCount; index++ {
		segment := &heapSegments[index]
		if address >= segment.start && address < segment.metadata {
			return segment
		}
	}
	return nil
}

func nextSegmentBlock(block uintptr) (uintptr, bool) {
	for index := 0; index < heapSegmentCount; index++ {
		if heapSegments[index].last == block {
			if index+1 < heapSegmentCount {
				return heapSegments[index+1].first, true
			}
			return 0, true
		}
	}
	return 0, false
}

func heapUsableSize() uintptr {
	var total uintptr
	for index := 0; index < heapSegmentCount; index++ {
		segment := &heapSegments[index]
		total += (segment.last - segment.first) * bytesPerBlock
	}
	return total
}

func heapReservedSize() uintptr {
	var total uintptr
	for index := 0; index < heapSegmentCount; index++ {
		segment := &heapSegments[index]
		total += segment.end - segment.start
	}
	return total
}

func heapMetadataSize() uintptr {
	var total uintptr
	for index := 0; index < heapSegmentCount; index++ {
		segment := &heapSegments[index]
		total += segment.end - segment.metadata
	}
	return total
}
