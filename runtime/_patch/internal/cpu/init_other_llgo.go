//go:build (amd64 || 386 || arm64 || wasm) && !(linux && arm64) && !baremetal

package cpu

func llgoPrepareCPU() {}
