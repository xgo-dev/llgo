// pull //go:linkname 声明集中在本文件：直连 C glue / IDF 符号，llgo.idf 构建
// 时由 ESP-IDF 链接环境解析。标准 Go 要求 //go:linkname 只能出现在 import 了
// "unsafe" 的文件里，故本文件显式 import；宿主构建时这些符号由 hosted.go 的
// 同名 push 桩供给。
package main

import _ "unsafe" // pull //go:linkname declarations live here; see file comment

// heapLedger is the C side's view of both IDF pools (demo_glue.c) - the
// memory ledger's stage lines, interleaved with the probe's own [go] lines.
//
//go:linkname heapLedger C.llgo_opus_heap
func heapLedger(tag *byte, n int32)

// fixStreams repoints clite's stdout/stderr/stdin handles at the ESP-IDF
// picolibc streams (host/llgo_syms.S): on baremetal clite opens its own
// streams with Fopen("/dev/stdout"), which fails under ESP-IDF (no such VFS
// device) and would leave every Go println writing to a NULL FILE*. Must run
// after the Go package inits and before the first print.
//
// TODO(runtime): a mechanism-level fix is landing in clite's stdio selection
// (under llgo.idf, reference the C streams directly instead of Fopen); once
// it ships, this call and the llgo_syms.S shim are dead weight - remove them
// with that change.
//
//go:linkname fixStreams C.llgoFixStreams
func fixStreams()
