/* The P0 pure-C WakeNet smoke test for ESP32-S31 - the llgo-free half of
 * the question "does esp-sr work on this chip with this IDF".
 *
 * What it proves, in order (every stage prints one `[c]` line):
 *
 *   1. the model partition maps and enumerates (esp_srmodel_init("model"))
 *   2. the engine resolves for the wn* model in the partition, and its
 *      create() succeeds - with the create latency printed (the number the
 *      esp-sr docs never state for wn9 on S31)
 *   3. the engine's geometry: chunk size (samples), sample rate, the model
 *      info string, and the internal-RAM/PSRAM bill create() paid
 *   4. detect on real content: the "Hi, Lexin" utterance (samples/hilexin.h,
 *      the same bytes the upstream test app feeds) goes through detect until
 *      the word triggers - avg/max per-call time over that run
 *   5. what detect reports AFTER a detection (does the engine stay in the
 *      verified state, or return to no-detect on its own) - this decides how
 *      a re-arming loop must work, because clean() is known to crash
 *      (wakenet9_quantized model_clean -> dl_convq_queue_bzero on a NULL
 *      queue, S31 + esp-sr 2.6.2 + IDF 6.3, reproduced under esp-emu; the
 *      upstream test app never calls clean)
 *   6. whether clean() survives on a FRESH engine (before any detection)
 *   7. destroy + deinit + the heap bill
 *
 * No audio hardware: stages 4-6 feed static buffers, exactly like the
 * upstream wakenet test app. A failure here points at esp-sr/IDF/chip; a
 * pass moves the question to the llgo binding (../idfproj + pkg/esp/wakenet).
 */
#include <stdio.h>
#include <string.h>

#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "esp_timer.h"
#include "esp_heap_caps.h"

#include "model_path.h"
#include "esp_wn_iface.h"
#include "esp_wn_models.h"

#include "hilexin.h"

/* Frames to keep feeding after a detection, to see the engine's state
 * machine without a clean() (one frame = one 32 ms chunk for wn9). */
#define POST_DETECT_FRAMES 100

typedef struct {
    const esp_wn_iface_t *wn;
    model_iface_data_t *md;
    char name[32];
    int chunk_samples;
    int rate;
    int16_t *buffer;
    size_t chunk_bytes;
} engine_t;

static int32_t ms_since(int64_t t0)
{
    return (int32_t)((esp_timer_get_time() - t0) / 1000);
}

static void heap_print(const char *tag)
{
    printf("[c] heap %-8s internal=%u psram=%u (free) | internal=%u (largest block)\n",
           tag,
           (unsigned)heap_caps_get_free_size(MALLOC_CAP_INTERNAL),
           (unsigned)heap_caps_get_free_size(MALLOC_CAP_SPIRAM),
           (unsigned)heap_caps_get_largest_free_block(MALLOC_CAP_INTERNAL));
}

/* Load the model into e (NULL result logged, engine left empty). */
static int engine_create(engine_t *e, srmodel_list_t *models, const char *model_name)
{
    memset(e, 0, sizeof(*e));
    e->wn = esp_wn_handle_from_name(model_name);
    if (e->wn == NULL) {
        printf("[c] FAIL esp_wn_handle_from_name(%s) returned NULL\n", model_name);
        return -1;
    }
    int64_t t0 = esp_timer_get_time();
    e->md = e->wn->create(model_name, DET_MODE_95);
    if (e->md == NULL) {
        printf("[c] FAIL create(%s, DET_MODE_95) returned NULL\n", model_name);
        return -1;
    }
    snprintf(e->name, sizeof(e->name), "%s", model_name);
    e->chunk_samples = e->wn->get_samp_chunksize(e->md);
    e->rate = e->wn->get_samp_rate(e->md);
    e->chunk_bytes = (size_t)e->chunk_samples * sizeof(int16_t);
    printf("[c] create(%s, DET_MODE_95) ok, latency = %d ms (chunk=%d rate=%d)\n",
           model_name, ms_since(t0), e->chunk_samples, e->rate);
    e->buffer = (int16_t *)malloc(e->chunk_bytes);
    if (e->buffer == NULL) {
        printf("[c] FAIL chunk buffer malloc\n");
        return -1;
    }
    return 0;
}

static void engine_destroy(engine_t *e)
{
    if (e->buffer) {
        free(e->buffer);
        e->buffer = NULL;
    }
    if (e->md) {
        e->wn->destroy(e->md);
        e->md = NULL;
    }
}

void app_main(void)
{
    printf("[c] wakenet csmoke: boot\n");
    heap_print("boot");

    /* 1. the model partition */
    srmodel_list_t *models = esp_srmodel_init("model");
    if (models == NULL) {
        printf("[c] FAIL esp_srmodel_init returned NULL - no usable \"model\" partition\n");
        return;
    }
    printf("[c] model partition ok, %d model(s):\n", models->num);
    for (int i = 0; i < models->num; i++) {
        printf("[c]   model[%d] = %s (%s)\n", i, models->model_name[i],
               models->model_info[i] ? models->model_info[i] : "?");
    }

    /* 2. resolve */
    char *model_name = esp_srmodel_filter(models, ESP_WN_PREFIX, NULL);
    if (model_name == NULL) {
        printf("[c] FAIL no wn* model in the partition (CONFIG_SR_WN_WN9_HILEXIN?)\n");
        esp_srmodel_deinit(models);
        return;
    }

    const uint8_t *data = (const uint8_t *)hilexin;
    const size_t data_size = sizeof(hilexin);

    /* 3. first engine: geometry + the memory bill create() paid */
    engine_t e;
    if (engine_create(&e, models, model_name) != 0) {
        esp_srmodel_deinit(models);
        return;
    }
    float threshold = e.wn->get_det_threshold(e.md, 1);
    printf("[c] geometry: threshold=%.3f words=%d | internal free after create = %u, psram free = %u\n",
           threshold,
           e.wn->get_word_num(e.md),
           (unsigned)heap_caps_get_free_size(MALLOC_CAP_INTERNAL),
           (unsigned)heap_caps_get_free_size(MALLOC_CAP_SPIRAM));

    /* 4. detect on the real utterance, until the word triggers */
    int chunks = 0;
    int detected = 0;
    int64_t total_us = 0;
    uint32_t max_us = 0;
    printf("[c] feeding the hilexin utterance (%u bytes = %.2f s)...\n",
           (unsigned)data_size, (float)data_size / 2.0f / (float)e.rate);
    int64_t t_run = esp_timer_get_time();
    while (!detected) {
        if ((chunks + 1) * e.chunk_bytes <= data_size) {
            memcpy(e.buffer, data + chunks * e.chunk_bytes, e.chunk_bytes);
        } else {
            memset(e.buffer, 0, e.chunk_bytes); /* pad the tail with silence */
        }
        int64_t t1 = esp_timer_get_time();
        wakenet_state_t st = e.wn->detect(e.md, e.buffer);
        uint32_t dt = (uint32_t)(esp_timer_get_time() - t1);
        total_us += dt;
        if (dt > max_us) {
            max_us = dt;
        }
        chunks++;
        if (st > 0) {
            detected = 1;
            printf("[c] DETECTED after %d chunks (state=%d, ts=%lld ms)\n",
                   chunks, (int)st, (long long)(esp_timer_get_time() / 1000));
            printf("[c] detect benchmark (utterance): frames=%d wall=%d ms avg=%u us max=%u us\n",
                   chunks, ms_since(t_run), (uint32_t)(total_us / chunks), max_us);
        }
        if (chunks > 10000) { /* ~5 minutes of audio: something is wrong */
            break;
        }
    }
    if (!detected) {
        printf("[c] FAIL no detection over %d chunks of the utterance\n", chunks);
        engine_destroy(&e);
        esp_srmodel_deinit(models);
        return;
    }

    /* 5. the post-detection state machine, WITHOUT clean() - this decides the
     * re-arm policy of the llgo glue (clean crashes on this build; see the
     * file header). */
    int zero_frames = 0;
    int nonzero_frames = 0;
    int last_state = 99;
    for (int i = 0; i < POST_DETECT_FRAMES; i++) {
        memset(e.buffer, 0, e.chunk_bytes); /* silence */
        wakenet_state_t st = e.wn->detect(e.md, e.buffer);
        if ((int)st == last_state) {
            /* compress runs of equal states */
        } else {
            printf("[c] post-detect frame %3d: state=%d\n", i, (int)st);
            last_state = (int)st;
        }
        if (st == 0) {
            zero_frames++;
        } else {
            nonzero_frames++;
        }
    }
    printf("[c] post-detect summary: %d frames of silence fed, state=0 in %d, nonzero in %d\n",
           POST_DETECT_FRAMES, zero_frames, nonzero_frames);
    engine_destroy(&e);
    printf("[c] engine 1 destroyed\n");

    /* 6. clean() on a FRESH engine (no detection yet): does the crash need a
     * detected state first? The print before the call marks the crash point -
     * the OK line missing means clean died. */
    engine_t e2;
    if (engine_create(&e2, models, model_name) != 0) {
        esp_srmodel_deinit(models);
        return;
    }
    for (int i = 0; i < 10; i++) {
        memset(e2.buffer, 0, e2.chunk_bytes);
        e2.wn->detect(e2.md, e2.buffer);
    }
    printf("[c] testing clean() on the fresh engine...\n");
    e2.wn->clean(e2.md);
    printf("[c] clean ok on fresh engine\n");

    /* 7. teardown + the heap bill */
    engine_destroy(&e2);
    esp_srmodel_deinit(models);
    heap_print("closed");
    printf("[c] wakenet csmoke: done\n");
}
