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
 * LLGO_GC_WINDOW_KB (make-artifact.sh, default 40 here) sets the size.
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

/* Redirect llgo's stdout/stderr/stdin handles to the ESP-IDF streams.
 *
 * On baremetal, clite opens its own streams at package init
 * (Fopen("/dev/stdout", "w")); under ESP-IDF that fopen fails, so the Go
 * print path would fwrite to a NULL FILE*. goapp/main.go calls this
 * (fixStreams) at the top of main, after the Go package inits ran and
 * before main.main prints anything; IDF's
 * stdout/stderr/stdin FILE* globals are valid from startup on.
 */
    .text
    .globl llgoFixStreams
llgoFixStreams:
    la t0, stdout
    lw t1, 0(t0)
    la t2, "github.com/xgo-dev/llgo/runtime/internal/clite.Stdout"
    sw t1, 0(t2)
    la t3, stderr
    lw t3, 0(t3)
    la t2, "github.com/xgo-dev/llgo/runtime/internal/clite.Stderr"
    sw t3, 0(t2)
    la t3, stdin
    lw t3, 0(t3)
    la t2, "github.com/xgo-dev/llgo/runtime/internal/clite.Stdin"
    sw t3, 0(t2)
    ret
