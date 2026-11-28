package runtime

// ParseGOMAXPROCS matches the positive int32 startup environment parser used
// by the Go runtime. An invalid, zero, negative, or overflowing value returns 0.
func ParseGOMAXPROCS(value string) int {
	const maxEnvironmentValue = 1<<31 - 1
	n := 0
	for i := 0; i < len(value); i++ {
		digit := int(value[i]) - '0'
		if digit < 0 || digit > 9 || n > (maxEnvironmentValue-digit)/10 {
			return 0
		}
		n = n*10 + digit
	}
	return n
}

// CPUCount returns the number of CPUs available to the current process,
// always at least 1.
func CPUCount() int {
	if n := int(c_maxprocs()); n > 0 {
		return n
	}
	return 1
}

// targetBDWGCMarkers decides the parallel marker count to request before
// GC_init(), returning 0 to leave BDWGC's default unchanged.
// gcMarkersSet distinguishes an unset GC_MARKERS from an explicit empty value,
// since BDWGC checks GETENV("GC_MARKERS") != NULL.
func targetBDWGCMarkers(gomaxprocs string, gcMarkersSet bool, gcNprocs string, cpuCount int) int {
	if gcMarkersSet || gcNprocs != "" {
		return 0
	}
	n := ParseGOMAXPROCS(gomaxprocs)
	if n == 0 {
		return 0
	}
	if cpuCount > 0 && n > cpuCount {
		n = cpuCount
	}
	return n
}
