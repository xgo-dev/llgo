//go:build baremetal || wasm

package runtime

func appendOtherTracebacks(out []byte, system bool, limit int) []byte { return out }
func appendCurrentCreatedBy(out []byte) []byte                        { return out }
