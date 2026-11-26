// pull //go:linkname 声明集中在本文件：直连 C glue / IDF 符号，llgo.idf 构建
// 时由 ESP-IDF 链接环境解析。标准 Go 要求 //go:linkname 只能出现在 import 了
// "unsafe" 的文件里，故本文件显式 import；宿主构建时这些符号由 hosted.go 的
// 同名 push 桩供给。
package main

import _ "unsafe" // pull //go:linkname declarations live here; see file comment

//go:linkname vTaskDelay C.vTaskDelay
func vTaskDelay(ticks uint32)
