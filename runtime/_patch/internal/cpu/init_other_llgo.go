//go:build (amd64 || 386 || arm64 || wasm) && !(linux && arm64) && !(amd64 && go1.26) && !baremetal

package cpu

func llgoInitializeCPU(env string) uint64 {
	Initialize(env)
	return 0
}
