//go:build !llgo.idf

// Host build (go build / go vet): linkname.go declares pull //go:linkname
// externs (llgo_vloop_*, llgoFixStreams) whose definitions live in the
// ESP-IDF link environment (main/demo_glue.c, host/llgo_syms.S). Push
// linkname supplies same-named inert stubs on this side of the build split,
// so the demo builds hosted; the real implementations come back on the
// llgo.idf build, where this file is excluded.
package main

import _ "unsafe" // for go:linkname

//go:linkname stubHeapLedger C.llgo_vloop_heap
func stubHeapLedger(tag *byte, n int32) { _, _ = tag, n }

//go:linkname stubFixStreams C.llgoFixStreams
func stubFixStreams() {}

//go:linkname stubNowUs C.llgo_vloop_now_us
func stubNowUs() int64 { return 0 }

//go:linkname stubVadCfg C.llgo_vloop_vad_cfg
func stubVadCfg(openNum, openDen, openFrames, hangoverFrames, floorMin, floorMax int32) {
	_, _, _, _, _, _ = openNum, openDen, openFrames, hangoverFrames, floorMin, floorMax
}

//go:linkname stubVadReset C.llgo_vloop_vad_reset
func stubVadReset() {}

//go:linkname stubVadFeed C.llgo_vloop_vad_feed
func stubVadFeed(pcm *int16, n int32, rms *int32, floor *int32) int32 {
	_, _, _ = pcm, n, rms
	*floor = 0
	return 0
}
