//go:build (!baremetal || nogc) && (!wasm || !llgo.wasm.gc.linear)

package tinygogc

// Host tests use the full segment index without building the collector.
const maxHeapSegments = 128
