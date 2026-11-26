// pull //go:linkname 声明集中在本文件：直连 C glue / IDF 符号（FreeRTOS、
// heap_caps、demo glue）以及 llgo runtime 内部符号（tinygogc —— runtime/
// internal 无法直接 import），llgo.idf 构建时由 ESP-IDF 链接环境解析。标准 Go
// 要求 //go:linkname 只能出现在 import 了 "unsafe" 的文件里，故本文件显式
// import；宿主构建时这些符号由 hosted.go 的同名 push 桩供给（gcExtBytes/
// gcExtHeapCap 两个 var 靠 linkname 改名自行定义，无需桩）。
package main

import "unsafe" // pull //go:linkname declarations live here; see file comment

// FreeRTOS kernel bindings. llgo emits an external reference to the C symbol
// and the ESP-IDF link environment resolves it against FreeRTOS.
//
//go:linkname xTaskGetTickCount C.xTaskGetTickCount
func xTaskGetTickCount() uint32

//go:linkname vTaskDelay C.vTaskDelay
func vTaskDelay(ticks uint32)

// heapStats reports the collector's own view of the Go heap: usable data
// bytes, reserved bytes including block-state bytes, the number of segments,
// and the bytes taken from PSRAM (runtime: tinygogc/gc_segment_idf.go).
//
//go:linkname heapStats github.com/xgo-dev/llgo/runtime/internal/runtime/tinygogc.HeapStats
func heapStats() (usable, reserved uintptr, segments int, external uintptr)

// gcCollect runs a full collection now. It is tinygogc.GC, the function behind
// gcCollect() on baremetal. Kept as a direct linkname: importing the whole
// runtime package for one function would pull its finalizer/reflect machinery
// into the image (the libffi build gap that once forced this workaround is
// fixed, but the image-size reason stands).
//
//go:linkname gcCollect github.com/xgo-dev/llgo/runtime/internal/runtime/tinygogc.GC
func gcCollect()

// gcExtBytes mirrors the C symbol the collector writes after each PSRAM
// segment, so that the Go side and the C side of the ledger can be compared.
//
//go:linkname gcExtBytes C.llgo_gc_ext_bytes
var gcExtBytes uintptr

// gcExtHeapCap mirrors the budget the C side compiled in (app_main.c).
//
//go:linkname gcExtHeapCap C.llgo_gc_ext_heap_cap
var gcExtHeapCap uintptr

//go:linkname heapCapsMalloc C.heap_caps_malloc
func heapCapsMalloc(size, caps uint32) unsafe.Pointer

//go:linkname heapCapsFree C.heap_caps_free
func heapCapsFree(ptr unsafe.Pointer)

//go:linkname rdcycle C.llgo_demo_rdcycle
func rdcycle() uint32

//go:linkname demoTLSCA C.llgo_demo_tls_ca
func demoTLSCA(buf *byte, cap int32, outLen *int32)

//go:linkname demoTLSCAWrong C.llgo_demo_tls_ca_wrong
func demoTLSCAWrong(buf *byte, cap int32, outLen *int32)
