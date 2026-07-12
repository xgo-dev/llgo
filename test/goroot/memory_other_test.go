//go:build !darwin && !linux

package goroot

func systemMemoryMonitoringSupported() bool { return false }

func readSystemMemoryState() (systemMemoryState, error) {
	return systemMemoryState{}, nil
}
