//go:build nogc

package metrics

import "runtime"

func llgoReadMetricMemStats() runtime.MemStats {
	return runtime.MemStats{}
}
