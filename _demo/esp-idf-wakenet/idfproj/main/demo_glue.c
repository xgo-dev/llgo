/* Demo-local C glue for the WakeNet acceptance probe.
 *
 * Two jobs, both demo-owned (nothing here belongs to the package template):
 *
 *   - the heap marks the probe prints between stages (the same diagnostic the
 *     other demos carry; internal RAM is the number this engine's memory
 *     story lives or dies by),
 *   - the test-source injection: the probe has no microphone, so the glue's
 *     built-in test source is pointed at the checked-in "Hi, Lexin" utterance
 *     (wakenet_pcm.c) - the same bytes the esp-sr test app feeds. The array
 *     is const flash data; llgo_wakenet_set_test_pcm only keeps the pointer.
 */
#include <stdio.h>

#include "esp_heap_caps.h"

#include "wakenet_pcm.h"

extern void llgo_wakenet_set_test_pcm(const int16_t *pcm, int32_t samples);

/* Hand the utterance to the glue's test source; returns the sample count the
 * probe prints. */
int32_t llgo_demo_test_pcm(void)
{
    const int32_t samples = (int32_t)(sizeof(hilexin_pcm) / sizeof(hilexin_pcm[0]));
    llgo_wakenet_set_test_pcm(hilexin_pcm, samples);
    return samples;
}

/* Print the IDF heap pools at one probe stage. The tag travels as pointer +
 * length (the Go<->C boundary carries integers and pointers only), and it is
 * consumed immediately - C keeps no reference to Go memory. */
void llgo_demo_heap(const char *tag, int32_t len)
{
    printf("[c] wakenet heap %.*s free=%u min=%u | internal=%u | psram=%u\n",
           (int)len, tag,
           (unsigned)heap_caps_get_free_size(MALLOC_CAP_DEFAULT),
           (unsigned)heap_caps_get_minimum_free_size(MALLOC_CAP_DEFAULT),
           (unsigned)heap_caps_get_free_size(MALLOC_CAP_INTERNAL),
           (unsigned)heap_caps_get_free_size(MALLOC_CAP_SPIRAM));
    fflush(stdout);
}
