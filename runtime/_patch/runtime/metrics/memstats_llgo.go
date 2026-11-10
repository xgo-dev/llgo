//go:build !nogc

package metrics

import "runtime"

func llgoReadMetricMemStats() runtime.MemStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// A collector may expose allocation events without live/free object
	// counts. Do not turn an unavailable live count into cumulative Mallocs.
	return m
}
