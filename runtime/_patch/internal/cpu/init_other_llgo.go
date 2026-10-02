//go:build (amd64 || arm64 || wasm) && !(linux && arm64) && !baremetal

package cpu

func llgoPrepareCPU() {}
