//go:build !nogc && baremetal

package runtime

import (
	"runtime"

	"github.com/xgo-dev/llgo/runtime/internal/runtime/tinygogc"
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
