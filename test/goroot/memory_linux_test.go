//go:build linux

package goroot

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func systemMemoryMonitoringSupported() bool { return true }

func readSystemMemoryState() (systemMemoryState, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return systemMemoryState{}, err
	}
	values := make(map[string]uint64)
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSuffix(fields[0], ":")
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return systemMemoryState{}, fmt.Errorf("parse /proc/meminfo line %q: %w", line, err)
		}
		values[key] = value << 10
	}
	total := values["MemTotal"]
	available := values["MemAvailable"]
	if total == 0 {
		return systemMemoryState{}, fmt.Errorf("/proc/meminfo did not report MemTotal")
	}
	return systemMemoryState{
		freePercent: int(available * 100 / total),
		swapFree:    values["SwapFree"],
		swapPresent: values["SwapTotal"] > 0,
	}, nil
}
