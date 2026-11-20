//go:build esp32c3 || (arm && baremetal && llgo.singlecontext)

package runtime

const softAtomicLLGoFiles = "; _wrap/softatomic_singlecore.c"
