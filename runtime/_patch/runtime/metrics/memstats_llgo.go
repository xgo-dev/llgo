//go:build !nogc

package metrics

import "runtime"

func llgoReadMetricMemStats() runtime.MemStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	if m.HeapObjects == 0 {
		m.HeapObjects = llgoSaturatingSub(m.Mallocs, m.Frees)
	}
	return m
}
