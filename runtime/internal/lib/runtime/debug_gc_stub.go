//go:build nogc || baremetal || wasm

package runtime

func setCollectorGCPercent(percent int32) int32 {
	previous := llgoGCPercent
	llgoGCPercent = percent
	return previous
}
