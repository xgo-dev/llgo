//go:build !darwin && !linux

package atomic

func runtime_procPin() int {
	return 0
}

func runtime_procUnpin() {}
