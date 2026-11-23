/* SPIRAM budget of the Go heap on ESP-IDF targets (tinygogc/gc_segment_idf.go).
 *
 * llgo_gc_ext_heap_cap bounds how much external memory the collector may take
 * as heap segments: 0 disables the external heap, the default (uintptr_t)-1
 * means three quarters of the SPIRAM free when the heap first grows, and any
 * other value is a byte count. An application overrides the weak definition
 * to choose a budget. llgo_gc_ext_bytes is written by the collector after each
 * successful segment; applications read it to report the Go heap's share.
 *
 * The runtime's C sources are compiled without target-specific defines, so the
 * guard keys on the architecture ESP-IDF targets share; whether the file is
 * built at all is decided by the LLGoFiles chain in z_rt.go.
 */
#include <stdint.h>

#if defined(__riscv) && __riscv_xlen == 32

__attribute__((weak, section(".data"))) uintptr_t llgo_gc_ext_heap_cap = (uintptr_t)-1;
__attribute__((weak)) uintptr_t llgo_gc_ext_bytes = 0;

#endif
