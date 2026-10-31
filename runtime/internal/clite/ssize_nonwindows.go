//go:build !windows

package c

// On supported Unix and wasm C targets, signed size_t has C long's width.
type SsizeT = Long
