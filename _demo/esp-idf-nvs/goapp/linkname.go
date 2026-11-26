// pull //go:linkname 声明集中在本文件：直连 C glue / IDF 符号，llgo.idf 构建
// 时由 ESP-IDF 链接环境解析。标准 Go 要求 //go:linkname 只能出现在 import 了
// "unsafe" 的文件里，故本文件显式 import；宿主构建时这些符号由 hosted.go 的
// 同名 push 桩供给。
package main

import _ "unsafe" // pull //go:linkname declarations live here; see file comment

// restart is the IDF esp_restart: a soft reset. On the target the flash
// keeps its contents; the probe's whole persistence claim rests on that.
//
//go:linkname restart C.esp_restart
func restart()

// heapPrint is the demo glue's ledger line: both IDF pools at one probe
// stage ([c] nvs heap ...), for the report's memory note.
//
//go:linkname heapPrint C.llgo_demo_heap
func heapPrint(tag *byte, n int32)
