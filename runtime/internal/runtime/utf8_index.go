//go:build !go1.27

package runtime

import _ "unsafe"

// Keep the compiler-facing string iterator ABI independent of GOROOT changes.
func decodeNonASCII(s string, k int) (r rune, pos int) {
	return decoderune(s, k)
}

// Metadata-only declaration; removed by the LLGo source overlay.
//
//go:linkname decoderune
func decoderune(s string, k int) (r rune, pos int)
