//go:build go1.27

package runtime

import _ "unsafe"

// Go 1.27 uses unsigned offsets internally; LLGo's string iterator uses int.
func decodeNonASCII(s string, k int) (r rune, pos int) {
	r, next := decoderune(s, uint(k))
	return r, int(next)
}

// Metadata-only declaration; removed by the LLGo source overlay.
//
//go:linkname decoderune
func decoderune(s string, k uint) (r rune, pos uint)
