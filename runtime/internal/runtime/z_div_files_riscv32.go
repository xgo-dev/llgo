//go:build tinygo.riscv32

package runtime

// The 64-bit division override is plain portable C (no CSR tricks), but it
// follows the atomic shim's hooking shape: a build-tagged Go file supplies
// the file list fragment that z_rt.go appends to the runtime's LLGoFiles
// chain, so the C objects link into the runtime archive ahead of the
// compiler-rt builtins archive.
const divRiscv32LLGoFiles = "; _wrap/divdi3_riscv32.c"
