// pull //go:linkname 声明集中在本文件：直连 C glue / IDF 符号，llgo.idf 构建
// 时由 ESP-IDF 链接环境解析。标准 Go 要求 //go:linkname 只能出现在 import 了
// "unsafe" 的文件里，故本文件显式 import；宿主构建时这些符号由 hosted.go 的
// 同名 push 桩供给。
package main

import _ "unsafe" // pull //go:linkname declarations live here; see file comment

// fixStreams repoints clite's stdout/stderr/stdin handles at the ESP-IDF
// picolibc streams (host/llgo_syms.S): on baremetal clite opens its own
// streams with Fopen("/dev/stdout"), which fails under ESP-IDF (no such VFS
// device) and would leave every Go println writing to a NULL FILE*.
//
// TODO(runtime): the mechanism-level fix landed in clite (156cc64ca -
// under llgo.idf the C streams are referenced directly); the shim is now
// link-compatibility only. Remove this call with the shim itself when the
// family sweeps the demos.
//
//go:linkname fixStreams C.llgoFixStreams
func fixStreams()

// heapLedger prints the two IDF pools at one demo stage (demo_glue.c).
//
//go:linkname heapLedger C.llgo_vloop_heap
func heapLedger(tag *byte, n int32)

//go:linkname nowUs C.llgo_vloop_now_us
func nowUs() int64

// The VAD's tunables travel from Go (the consts below) into the C engine;
// feed returns 0 = nothing, 1 = utterance opened, 2 = closed.
//
//go:linkname vadCfg C.llgo_vloop_vad_cfg
func vadCfg(openNum, openDen, openFrames, hangoverFrames, floorMin, floorMax int32)

//go:linkname vadReset C.llgo_vloop_vad_reset
func vadReset()

//go:linkname vadFeed C.llgo_vloop_vad_feed
func vadFeed(pcm *int16, n int32, rms *int32, floor *int32) int32
