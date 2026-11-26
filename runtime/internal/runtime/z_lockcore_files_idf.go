//go:build llgo && baremetal && tinygo.riscv32 && llgo.idf

package runtime

// The core-pin hook behind runtime.LockCore (lockcore_idf.go) is a weak C
// symbol, so a firmware without a core-affinity glue still links: the weak
// default reports "not provided" and LockCore surfaces it as an error. A
// project that implements the real pin defines a strong llgoLockCore in its
// IDF component (see _demo/esp-idf-lockcore/main/lockcore_glue.c); the
// whole-archive main-component pattern keeps that object in the link
// unconditionally, and a strong definition overrides the weak default. The
// tag deliberately omits !nogc, mirroring z_psram_files_idf.go: the hook is
// inert there (an unreferenced weak function), while a nogc project defining
// it stays linkable.
const lockCoreIdfLLGoFiles = "; _wrap/lockcore_idf.c"
