//go:build (amd64 || 386 || arm64) && !(linux && arm64) && !baremetal

package cpu

func llgoPrepareCPU() {}
