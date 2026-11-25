package main

import (
	_ "unsafe" // for the //go:linkname declaration below

	"github.com/xgo-dev/llgo/pkg/esp/wifi"
)

// dmaMax is the largest free contiguous DMA-capable internal block
// (app_main.c helper): the WiFi driver needs ~1.6 KB of contiguous DMA
// memory per packet, so this is the ledger line that actually predicts
// whether the radio keeps working.
//
//go:linkname dmaMax C.llgo_all_dma_max
func dmaMax() int32

// memWatchMs is the ledger cadence during steady state. The bring-up
// watermarks answer "did this stage fit"; a periodic line answers "is it
// still fitting" - the internal minimum is the radio's headroom and dma max
// is whether the next packet still has a contiguous block to live in.
const memWatchMs = 30000

// memWatch prints the ledger when the cadence elapsed and returns the updated
// timestamp, so the caller only has to carry one value through its loop.
func memWatch(lastMs uint32) uint32 {
	now := nowMs()
	if now-lastMs < memWatchMs {
		return lastMs
	}
	heapLedger("periodic")
	return now
}

func heapLedger(stage string) {
	println("[go] heap", stage, "internal free =", int(wifi.HeapInternalFree()),
		"min =", int(wifi.HeapMin()),
		"| total free =", int(wifi.HeapFree()),
		"| psram =", int(wifi.HeapSpiramFree()),
		"| dma max =", int(dmaMax()))
}
