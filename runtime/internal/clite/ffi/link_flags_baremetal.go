//go:build baremetal && llgo.baremetal.stdio

package ffi

// Statically linked binaries may retain method entries for unsupported
// dynamic calls. Give them the same failure behavior as WASI.
const LLGoFiles = "_wrap/libffi_unavailable.c"
