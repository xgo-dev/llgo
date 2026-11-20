//go:build tinygo.riscv32

package runtime

// The atomic libcall shim's interrupt-masking helpers use CSR instructions
// that Go cannot express, so they live in C next to the shim and are compiled
// into the runtime through the LLGoFiles chain in z_rt.go.
const atomicRiscv32LLGoFiles = "; _wrap/atomic_riscv32.c"
