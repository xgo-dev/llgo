/* Runtime symbols the ESP-IDF link must provide for the llgo fiber backend.
 *
 * tinygogc/gc_link.go reads _heapStart/_heapEnd, _globals_start/_globals_end
 * and _stack_top/_stack_end; proc_fiber_baremetal.go guards the bootstrap
 * context with the stack window and writes a canary word at &_stack_end.
 * In llgo.idf mode the scheduler runs on the FreeRTOS main-task stack, so
 * the stack window is a dedicated static region: it only has to be a
 * writable [lo, hi) window (canary + conservative scan), not the real task
 * stack. The GC heap window is carved out statically so it can never
 * collide with the IDF heap: everything here lives in .dram0.bss, below the
 * _heap_start symbol where the IDF heap allocator begins.
 *
 * Window sizing: on a PSRAM board the collector is segmented
 * (tinygogc/gc_segment_idf.go) - it takes further segments from the SPIRAM
 * heap (75% of the free size at the first growth) - so the static window
 * only has to cover the hot short-lived traffic; the bulk of the heap lives
 * in PSRAM and costs no internal SRAM. A board without PSRAM keeps the
 * static window only, so size it for the program's live set there.
 * LLGO_GC_WINDOW_KB (make-artifact.sh, default 64) sets the size.
 *
 * Layout note (gc_link.go): the runtime starts allocating at
 * &_heapStart + 2048, so the first 2 KiB after _heapStart are slack and the
 * usable heap is [_heapStart+2048, _heapEnd).
 */

#ifndef LLGO_GC_WINDOW_BYTES
#define LLGO_GC_WINDOW_BYTES (64 * 1024)
#endif

    .section .bss.llgo, "aw", @nobits

    .balign 64
    .globl _heapStart
_heapStart:
    .space 2048                      /* runtime reserve */
    .space LLGO_GC_WINDOW_BYTES      /* static GC window; PSRAM carries the rest */
    .globl _heapEnd
_heapEnd:

    .balign 64
    .globl _stack_end                /* floor: canary word written here */
_stack_end:
    .space 4096
    .globl _stack_top                /* ceiling of the bootstrap window */
_stack_top:

/* llgoFixStreams: kept for link compatibility with firmware built against
 * the old contract, but a no-op since the clite mechanism fix. clite's
 * stdout/stderr/stdin now reference the ESP-IDF libc's stream globals
 * directly (runtime/internal/clite/stdio_baremetal_idf.go), so no runtime
 * repointing is needed and app code no longer calls this at the top of
 * main. */
    .text
    .globl llgoFixStreams
llgoFixStreams:
    ret
