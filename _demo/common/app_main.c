/* ESP-IDF shell around the llgo build artifact (llgo_out.o).
 *
 * The llgo program compiles with -target esp32s31-idf into a relocatable
 * object. Its generated C entry, main(int32, char**), hands package
 * initialization and main.main to runtime.RunMain: the fiber scheduler runs
 * on this FreeRTOS task's stack, and every goroutine gets a fiber stack
 * allocated from the llgo GC heap. When main returns, control comes back
 * here and the main task deletes itself.
 */
#include <stdint.h>
#include <stdio.h>
#include <unistd.h>
#include "esp_heap_caps.h"

/* Budget the Go heap may take from PSRAM (runtime: tinygogc/gc_segment_idf.go).
 * The collector takes segments from the SPIRAM heap once its static 256 KiB
 * window is used up: (uintptr_t)-1, the default, lets it have 75% of the
 * SPIRAM that is free when it first grows; 0 keeps the Go heap in the static
 * window; any other value is a byte count. Without this definition the
 * runtime's weak default (auto) applies. */
#ifndef LLGO_GC_PSRAM_CAP
#define LLGO_GC_PSRAM_CAP ((uintptr_t)-1)
#endif
uintptr_t llgo_gc_ext_heap_cap = LLGO_GC_PSRAM_CAP;

/* The largest free contiguous DMA-capable internal block (esp-idf-all local
 * helper for the memory ledger): the WiFi driver needs ~1.6 KiB contiguous
 * DMA blocks per packet, so this is the number that actually predicts
 * whether the radio keeps working, more than the free totals. */
#include "esp_heap_caps.h"
int32_t llgo_all_dma_max(void)
{
    return (int32_t)heap_caps_get_largest_free_block(MALLOC_CAP_DMA);
}
#include <freertos/FreeRTOS.h>
#include <freertos/task.h>

extern int main(int argc, char **argv);

/* The dedicated Go task (optional, llgo_syms.S LLGO_GO_TASK_STACK): app_main
 * spawns "llgo-go" on the static stack the syms file declares below the GC
 * window, so the fiber SP sits above the declared stack bottom and the
 * RISC-V port's coprocessor save-area check (port.c) holds when Go owns the
 * FPU across a WiFi preemption. The size must match the .space in
 * llgo_syms.S (8192). Without the symbol (every firmware that runs Go
 * directly on the IDF main task) the probe stays NULL and the run keeps the
 * historical shape. */
extern uint8_t _llgo_go_task_stack[] __attribute__((weak));
static StaticTask_t s_go_task_tcb;
static volatile int s_go_done;

static void go_task_fn(void *arg)
{
    /* IDF v6.1 preview (esp32s31): a statically-created task starts with
     * zeroed TLS, but picolibc resolves stdin/stdout/stderr through __thread
     * pointers when CONFIG_LIBC_PICOLIBC_NEWLIB_COMPATIBILITY is on - so the
     * first stdio call on this task faults reading FILE at offset 0x14
     * (tls_stdout == NULL, measured). Re-point the TLS slots at the static
     * FILE objects before Go runs. */
    extern __thread FILE *tls_stdin, *tls_stdout, *tls_stderr;
    tls_stdin = stdin;
    tls_stdout = stdout;
    tls_stderr = stderr;

    (void)arg;
    (void)main(0, NULL);
    s_go_done = 1;
    vTaskDelete(NULL);
}

void app_main(void)
{
    printf("[c] app_main: starting llgo runtime\n");
    if (_llgo_go_task_stack) {
        const uint32_t depth = 8192 / sizeof(StackType_t);
        xTaskCreateStaticPinnedToCore(go_task_fn, "llgo-go", depth, NULL,
                                      1 /* prio: same as the IDF main task */,
                                      (StackType_t *)_llgo_go_task_stack,
                                      &s_go_task_tcb, 0);
        while (!s_go_done)
            vTaskDelay(pdMS_TO_TICKS(100));
    } else {
        (void)main(0, NULL);
    }

    /* Both allocators' ledgers, from the C side: the budget above, the bytes
     * the collector reports having taken from SPIRAM (weak symbol, runtime:
     * tinygogc/gc_segment_idf.go), and the two IDF pools. */
    extern uintptr_t llgo_gc_ext_bytes __attribute__((weak));
    uintptr_t ext = &llgo_gc_ext_bytes ? llgo_gc_ext_bytes : 0;
    printf("[c] gc ext heap: cap = %u, bytes = %u | idf internal free = %u, psram free = %u\n",
           (unsigned)llgo_gc_ext_heap_cap, (unsigned)ext,
           (unsigned)heap_caps_get_free_size(MALLOC_CAP_INTERNAL),
           (unsigned)heap_caps_get_free_size(MALLOC_CAP_SPIRAM));

    printf("[c] app_main: llgo main returned, deleting main task\n");
    vTaskDelete(NULL);
}

/* The llgo runtime references the non-reentrant POSIX _write from a cold
 * diagnostic path. IDF's picolibc port only provides the reentrant
 * _write_r, so route the plain name through the VFS layer (fd 1/2 are the
 * console). */
ssize_t _write(int fd, const void *data, size_t size)
{
    return write(fd, data, size);
}

/* Go's fatal path (unrecovered panic, scheduler deadlock) calls exit(2).
 * IDF's _exit restarts the chip, which under esp-emu turns one panic into a
 * boot loop and destroys the diagnostics. The component links with
 * --wrap=_exit, so every _exit reference - newlib's exit() included -
 * lands here instead, and the fatal output stays the last thing in the
 * console log. */
void __wrap__exit(int status)
{
    printf("[c] _exit(%d): parking main task\n", status);
    for (;;) {
        vTaskDelay(portMAX_DELAY);
    }
}
