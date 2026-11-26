//go:build !llgo.idf

// Host build (go build / go vet): main.go declares pull //go:linkname externs
// whose definitions live in the ESP-IDF link environment (C glue, the llgo
// runtime's stream handles). Without definitions the hosted link fails on the
// unresolved relocations. Push linkname supplies same-named symbols that exist
// only on this side of the build split, so the demo builds hosted with inert
// stubs; the real implementations come back on the llgo.idf build, where this
// file is excluded.
package main

import _ "unsafe" // for go:linkname

//go:linkname stubFixStreams C.llgoFixStreams
func stubFixStreams() {}

//go:linkname stubHeapLedger C.llgo_ac_heap
func stubHeapLedger(tag *byte, n int32) { _, _ = tag, n }

//go:linkname stubNowUs C.llgo_ac_now_us
func stubNowUs() int64 { return 0 }

//go:linkname stubNowMs C.llgo_ac_now_ms
func stubNowMs() uint32 { return 0 }

//go:linkname stubVadCfg C.llgo_ac_vad_cfg
func stubVadCfg(openNum, openDen, openFrames, hangoverFrames, floorMin, floorMax int32) {
	_, _, _, _, _, _ = openNum, openDen, openFrames, hangoverFrames, floorMin, floorMax
}

//go:linkname stubVadReset C.llgo_ac_vad_reset
func stubVadReset() {}

//go:linkname stubVadFeed C.llgo_ac_vad_feed
func stubVadFeed(pcm *int16, n int32, rms *int32, floor *int32) int32 {
	_, _, _ = pcm, n, rms
	*floor = 0
	return 0
}

//go:linkname stubTestPcm C.llgo_ac_test_pcm
func stubTestPcm() int32 { return 0 }

//go:linkname stubMclk C.llgo_ac_mclk
func stubMclk(pin int32) { _ = pin }

//go:linkname stubGosched github.com/xgo-dev/llgo/runtime/internal/runtime.Gosched
func stubGosched() {}

// fixStreams, heapLedger, the clock/VAD hooks, injectTestPcm, setWnMclk and
// gosched keep their names on both sides; the stubs above claim the C
// symbols on the host, so main.go's declarations resolve either way. The
// llgo.idf build resolves the same names against llgo_syms.S and the demo
// glue instead.

// Diagnostics (HW only; never reached on the host build's golden path).
//go:linkname stubDiagTxEdges C.llgo_i2s_diag_tx_edges
func stubDiagTxEdges(out *int32, mclk, bclk, ws, dout, rate int32) int32 { return 0 }

//go:linkname stubDiagRxProbe C.llgo_i2s_diag_rx_probe
func stubDiagRxProbe(out *int32, port, mclk, bclk, ws, din, bits, stereo, rate int32) int32 {
	return 0
}

// chunkRmsGet: the phase C capture-energy window (HW only).
//go:linkname stubChunkRmsGet C.llgo_ac_chunk_rms_get
func stubChunkRmsGet(rms *int32, chunks *int32) { *rms, *chunks = 0, 0 }
