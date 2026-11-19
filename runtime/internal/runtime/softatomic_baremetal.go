//go:build esp32c3 || (cortexm && qemu && lm3s6965)

package runtime

const softAtomicLLGoFiles = "; _wrap/softatomic_singlecore.c"
