//go:build baremetal && !tinygo.riscv32

package ffi

// Statically linked binaries may retain method entries for unsupported
// dynamic calls. Give them the same failure behavior as WASI.
const LLGoFiles = "_wrap/libffi_unavailable.c"
