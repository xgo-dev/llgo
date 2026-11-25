/* Demo-local C glue for the Opus acceptance probe.
 *
 * The probe's heap marks come from the C side because that is where both
 * allocators are visible: the IDF heap pools (internal SRAM and PSRAM).
 * Nothing here belongs to a package - a real project would print these
 * numbers itself, or use the package API of whatever peripheral it binds.
 *
 * The fflush(stdout) keeps the [c] ledger lines prompt on the console (the
 * Go print path flushes per line; printf buffers). The Go side's stdout
 * handle itself is fixed by llgoFixStreams (host/llgo_syms.S, called at the
 * top of the probe's main) - see that file for why it exists.
 */
#include <stdio.h>

#include "esp_heap_caps.h"

/* Print the two IDF pools at one probe stage. The tag travels as pointer +
 * length (the Go<->C boundary carries integers and pointers only), and it is
 * consumed immediately - C keeps no reference to Go memory. */
void llgo_opus_heap(const char *tag, int32_t len)
{
    printf("[c] opus heap %.*s internal=%u psram=%u\n",
           (int)len, tag,
           (unsigned)heap_caps_get_free_size(MALLOC_CAP_INTERNAL),
           (unsigned)heap_caps_get_free_size(MALLOC_CAP_SPIRAM));
    fflush(stdout);
}
