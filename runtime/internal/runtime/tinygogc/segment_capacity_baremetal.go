//go:build baremetal && !nogc

package tinygogc

// Bare-metal heaps grow in place. With segmentedHeap disabled, initGC
// adds the only segment and growHeap extends it without appending a segment.
const maxHeapSegments = 1
