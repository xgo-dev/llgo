// pull //go:linkname 声明集中在本文件：直连 C glue / IDF 符号（gosched 直连
// llgo runtime 的 Gosched），llgo.idf 构建时由 ESP-IDF 链接环境解析。标准 Go
// 要求 //go:linkname 只能出现在 import 了 "unsafe" 的文件里，故本文件显式
// import；宿主构建时这些符号由 hosted.go 的同名 push 桩供给。
package main

import _ "unsafe" // pull //go:linkname declarations live here; see file comment

// llgoFixStreams is the assembly helper merged into llgo_out.o
// (host/llgo_syms.S): it repoints clite's stdout/stderr/stdin handles at the
// ESP-IDF picolibc streams, because clite's baremetal init fopen fails under
// IDF's VFS. Must run after package init and before the first print.
//
//go:linkname fixStreams C.llgoFixStreams
func fixStreams()

// heapPrint prints the IDF heap pools from the C side (main/demo_glue.c).
//
//go:linkname heapPrint C.llgo_demo_heap
func heapPrint(tag *byte, n int32)

// injectTestPcm hands the demo's hilexin utterance to the glue's test source
// (main/demo_glue.c) and returns the sample count.
//
//go:linkname injectTestPcm C.llgo_demo_test_pcm
func injectTestPcm() int32

// nowMs is the glue's millisecond clock (wakenet_glue.c): the probe's run
// window and stats cadence measure against it.
//
//go:linkname nowMs C.llgo_wakenet_now_ms
func nowMs() uint32

// gosched hands the fiber scheduler around while the probe waits - without
// it the poller goroutine would never run.
//
//go:linkname gosched github.com/xgo-dev/llgo/runtime/internal/runtime.Gosched
func gosched()
