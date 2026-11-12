package abi

// SmallStackBufferSize is the shared compiler/runtime contract for caller-owned
// temporary buffers: bytes for slices/strings, rune slots for UTF-8 decoding.
const SmallStackBufferSize = 32
