//go:build (baremetal && !nogc) || (wasm && llgo_wasm_gc)

package runtime

import (
	"runtime"

	"github.com/goplus/llgo/runtime/internal/runtime/tinygogc"
)

func ReadMemStats(m *runtime.MemStats) {
	stats := tinygogc.ReadGCStats()
	m.Alloc = stats.Alloc
	m.TotalAlloc = stats.TotalAlloc
	m.Sys = stats.Sys
	m.Mallocs = stats.Mallocs
	m.Frees = stats.Frees
	m.HeapAlloc = stats.HeapAlloc
	m.HeapSys = stats.HeapSys
	m.HeapIdle = stats.HeapIdle
	m.HeapInuse = stats.HeapInuse
	m.StackInuse = stats.StackInuse
	m.StackSys = stats.StackSys
	m.GCSys = stats.GCSys
}

func GC() {
	tinygogc.GC()
}
