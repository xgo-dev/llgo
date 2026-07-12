//go:build darwin

package goroot

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func systemMemoryMonitoringSupported() bool { return true }

func readSystemMemoryState() (systemMemoryState, error) {
	pressureOutput, err := exec.Command("memory_pressure", "-Q").Output()
	if err != nil {
		return systemMemoryState{}, err
	}
	const freePrefix = "System-wide memory free percentage: "
	freePercent := -1
	for _, line := range strings.Split(string(pressureOutput), "\n") {
		if !strings.HasPrefix(line, freePrefix) {
			continue
		}
		value := strings.TrimSuffix(strings.TrimSpace(strings.TrimPrefix(line, freePrefix)), "%")
		freePercent, err = strconv.Atoi(value)
		if err != nil {
			return systemMemoryState{}, fmt.Errorf("parse memory_pressure line %q: %w", line, err)
		}
		break
	}
	if freePercent < 0 {
		return systemMemoryState{}, fmt.Errorf("memory_pressure did not report a free percentage")
	}

	swapOutput, err := exec.Command("sysctl", "-n", "vm.swapusage").Output()
	if err != nil {
		return systemMemoryState{}, err
	}
	fields := strings.Fields(string(swapOutput))
	var swapFree uint64
	foundSwapFree := false
	for i := 0; i+2 < len(fields); i++ {
		if fields[i] != "free" || fields[i+1] != "=" {
			continue
		}
		swapFree, err = parseDarwinMemorySize(fields[i+2])
		if err != nil {
			return systemMemoryState{}, err
		}
		foundSwapFree = true
		break
	}
	if !foundSwapFree {
		return systemMemoryState{}, fmt.Errorf("sysctl vm.swapusage did not report free swap: %q", swapOutput)
	}
	return systemMemoryState{freePercent: freePercent, swapFree: swapFree, swapPresent: true}, nil
}

func parseDarwinMemorySize(value string) (uint64, error) {
	if len(value) < 2 {
		return 0, fmt.Errorf("invalid Darwin memory size %q", value)
	}
	multiplier := float64(1)
	switch value[len(value)-1] {
	case 'K':
		multiplier = 1 << 10
	case 'M':
		multiplier = 1 << 20
	case 'G':
		multiplier = 1 << 30
	case 'T':
		multiplier = 1 << 40
	default:
		return 0, fmt.Errorf("invalid Darwin memory size %q", value)
	}
	number, err := strconv.ParseFloat(value[:len(value)-1], 64)
	if err != nil {
		return 0, fmt.Errorf("parse Darwin memory size %q: %w", value, err)
	}
	return uint64(number * multiplier), nil
}
