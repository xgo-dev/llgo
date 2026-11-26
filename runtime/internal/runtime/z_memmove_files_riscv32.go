//go:build tinygo.riscv32

package runtime

// newlib-esp32 ships only its PREFER_SIZE_OVER_SPEED build for RV32, so the
// libc memmove and memcpy are byte loops. The word-block implementations in
// _wrap/memmove_riscv32.c speed up slice copy and append and are compiled
// into the runtime through the LLGoFiles chain in z_rt.go.
const memmoveRiscv32LLGoFiles = "; _wrap/memmove_riscv32.c"
