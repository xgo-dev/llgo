//go:build (baremetal && !nogc) || (wasm && llgo.wasm.gc.linear)

package tinygogc

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

var heapSegments [maxHeapSegments]heapSegment

func (segment *heapSegment) configure(end uintptr) bool {
	segment.end = end
	totalSize := end - segment.start
	metadataSize := (totalSize + blocksPerStateByte*bytesPerBlock) / (1 + blocksPerStateByte*bytesPerBlock)
	segment.metadata = end - metadataSize
	segment.last = segment.first + (segment.metadata-segment.start)/bytesPerBlock
	return segment.last > segment.first
}

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
	if !segment.configure(end) {
		return false
	}
	position := heapSegmentCount
	for maxHeapSegments > 1 && position > 0 && heapSegmentsByAddress[position-1].start > start {
		heapSegmentsByAddress[position] = heapSegmentsByAddress[position-1]
		position--
	}
	heapSegmentsByAddress[position] = segment
	c.Memset(unsafe.Pointer(segment.metadata), 0, end-segment.metadata)
	heapSegmentCount++
	endBlock = segment.last
	return true
}

func segmentForBlock(block uintptr) *heapSegment {
	if !segmentedHeap {
		segment := &heapSegments[0]
		if block >= segment.first && block <= segment.last {
			return segment
		}
		gcPanic(c.Str("gc: invalid heap block"))
		return nil
	}
	low, high := 0, heapSegmentCount
	for low < high {
		mid := low + (high-low)/2
		segment := &heapSegments[mid]
		if block < segment.first {
			high = mid
		} else if block > segment.last {
			low = mid + 1
		} else {
			return segment
		}
	}
	gcPanic(c.Str("gc: invalid heap block"))
	return nil
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
