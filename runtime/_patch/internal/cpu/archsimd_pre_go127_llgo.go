//go:build (amd64 || 386 || arm64) && !baremetal && !go1.27

package cpu

func llgoArchSIMDFeatures() uint64 {
	return 0
}
