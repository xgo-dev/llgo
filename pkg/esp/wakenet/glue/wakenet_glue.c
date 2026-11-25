/* Thin C glue between llgo-compiled Go code and the ESP-SR WakeNet engine
 * (esp-sr component, pure WakeNet path: no AFE, no MultiNet).
 *
 * This file is a TEMPLATE: copy it into your project's main component so the
 * IDF build compiles it there (integration way 1 in pkg/esp/README.md); the
 * project links the esp-sr component (REQUIRES esp-sr) and declares a data
 * partition named "model" in its partition table, which is where the esp-sr
 * CMake packs the selected model (srmodels.bin).
 *
 * The boundary rules are the family's (pkg/esp/README.md): integer-and-pointer
 * -only signatures, esp_err passed through as int32, IDF config structs stay
 * on the C side, and the Go<->C record layouts are pinned by _Static_assert.
 *
 * Architecture (the reason this glue is more than a wrapper):
 *
 *   WakeNet detect is a synchronous 3-6.4 ms recomputation per 32 ms chunk
 *   (512 samples @ 16 kHz). Calling it from a llgo goroutine would freeze the
 *   fiber scheduler ~30 times per second - the llgo scheduler is single-task
 *   cooperative state, and no C call from Go may hold it for milliseconds.
 *   So the whole audio path lives on THIS glue's own FreeRTOS task:
 *
 *     I2S RX (16 kHz 16-bit mono, DMA ring)  -or-  built-in test source
 *       -> one chunk into a pre-allocated buffer
 *       -> wakenet detect (+ per-call timing)
 *       -> detection / error into a bounded event ring
 *
 *   Go only polls the ring from the scheduler task (microseconds per record)
 *   and reads the timing counters. The task allocates nothing in its loop:
 *   the chunk buffer is one heap_caps_malloc at Init, the event ring is
 *   static, and the test source reads a caller-injected flash-resident PCM
 *   array through a pointer.
 *
 * Cross-task visibility: the ring indexes and the stats are single-writer
 * fields published with release stores and read with acquire loads (the same
 * contract ble_glue uses; no RMW atomics anywhere, so RV32IMC llgo code never
 * drags in libatomic, and the plain u32 stats cannot tear on RV32).
 *
 * Outside an ESP-IDF build the whole body compiles away, so stray globbing
 * (hosted builds, IDE tasks, over-broad CMake file lists) stays harmless.
 */

#if defined(ESP_PLATFORM)

#include <stdint.h>
#include <stddef.h>
#include <string.h>

#include "esp_err.h"
#include "esp_timer.h"
#include "esp_heap_caps.h"
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "freertos/event_groups.h"
#include "driver/i2s_std.h"

#include "model_path.h"
#include "esp_wn_iface.h"
#include "esp_wn_models.h"

/* ---- states, sources, event kinds (mirrored in wakenet.go) ---- */

enum { WN_ST_UNINIT = 0, WN_ST_READY = 1, WN_ST_RUNNING = 2 };
enum { WN_SRC_TEST = 0, WN_SRC_I2S = 1 };
enum { WN_EV_DETECTED = 1, WN_EV_ERROR = 2 };

/* The one record shape that crosses into Go. Field order and widths are an
 * ABI: pkg/esp/wakenet's evRecord mirrors it field for field (all four-byte
 * scalars, so the layouts match on both compilers without packing). */
typedef struct {
    int32_t kind;
    int32_t code;    /* EvError: the esp_err / package code; 0 otherwise */
    uint32_t ts_ms;  /* esp_timer milliseconds since boot (u32, wraps after
                        ~49.7 days - documented on the Go side) */
} llgo_wakenet_ev_t;

_Static_assert(sizeof(llgo_wakenet_ev_t) == 12, "llgo_wakenet_ev_t ABI drifted");

/* ---- engine state (all mutated from the Go scheduler task only, except
 * what the detect task writes; that is marked task) ---- */

#define WN_MODEL_NAME_MAX 32
#define WN_EV_SLOTS 8
#define WN_I2S_READ_TIMEOUT_MS 2000

static int32_t s_state;                     /* Go-written only */
static int32_t s_source;                    /* Go-written only */
static srmodel_list_t *s_models;            /* Init, closed at Close */
static const esp_wn_iface_t *s_wn;          /* Init, closed at Close */
static model_iface_data_t *s_md;            /* Init, closed at Close */
static int16_t *s_chunk;                    /* pre-allocated detect input */
static int32_t s_chunk_samples;             /* int16 samples per detect call */
static int32_t s_rate;                      /* model sample rate (Hz) */
static char s_model_name[WN_MODEL_NAME_MAX];
static i2s_chan_handle_t s_i2s;             /* SourceI2S only */
static EventGroupHandle_t s_exit_ev;        /* task-exit acknowledgment */
static volatile int32_t s_run;              /* task keeps looping while set */
static volatile int32_t s_gen;              /* bumped per Start (Go-written) */
static volatile TaskHandle_t s_task;        /* task-written */

/* detection state + stats: the detect task is the only writer, Go only
 * reads. Plain u32 fields cannot tear on RV32; frames/avg/max may be one
 * frame apart from each other in a read, which a diagnostic does not mind. */
static volatile int32_t s_detected;
static volatile uint32_t s_frames;
static volatile uint32_t s_avg_us;
static volatile uint32_t s_max_us;

/* test source: a caller-injected PCM array (flash-resident const data; the
 * glue never copies it) replayed in a loop, or silence when nothing was
 * injected. */
static const int16_t *s_test_pcm;
static int32_t s_test_pcm_len;
static int32_t s_test_pos;

/* ---- event ring (SPSC: the detect task produces, the Go poller consumes;
 * index ownership and the release/acquire contract follow ble_glue) ---- */

#define WN_PUBLISH(v, val) __atomic_store_n(&(v), (val), __ATOMIC_RELEASE)
#define WN_ACQUIRE(v)      __atomic_load_n(&(v), __ATOMIC_ACQUIRE)

static llgo_wakenet_ev_t s_ring[WN_EV_SLOTS];
static uint32_t s_ring_wr, s_ring_rd; /* wr: producer-owned, rd: consumer-owned */
static uint32_t s_dropped;            /* producer-owned overflow counter */
static uint32_t s_dropped_seen;       /* consumer-owned snapshot */

/* post_event copies one record into the ring. Full ring: drop the record and
 * count it - the Go side reports the drop as an EvOverflow delta (a
 * program that never consumes events loses them, and is told so). */
static void wn_post_event(int32_t kind, int32_t code)
{
    uint32_t wr = s_ring_wr;
    if (wr - s_ring_rd >= WN_EV_SLOTS) {
        s_dropped++;
        return;
    }
    llgo_wakenet_ev_t *e = &s_ring[wr % WN_EV_SLOTS];
    e->kind = kind;
    e->code = code;
    e->ts_ms = (uint32_t)(esp_timer_get_time() / 1000);
    WN_PUBLISH(s_ring_wr, wr + 1);
}

/* ---- test source: fill one chunk, looping the injected PCM ---- */

static void wn_test_fill(int16_t *buf, int32_t samples)
{
    if (s_test_pcm == NULL || s_test_pcm_len <= 0) {
        memset(buf, 0, (size_t)samples * sizeof(int16_t));
        return;
    }
    int32_t n = samples;
    while (n > 0) {
        if (s_test_pos >= s_test_pcm_len) {
            s_test_pos = 0;
        }
        int32_t take = s_test_pcm_len - s_test_pos;
        if (take > n) {
            take = n;
        }
        memcpy(buf, &s_test_pcm[s_test_pos], (size_t)take * sizeof(int16_t));
        buf += take;
        s_test_pos += take;
        n -= take;
    }
}

/* ---- the audio + detect task ---- */

static void wakenet_task(void *arg)
{
    (void)arg;
    const int32_t gen = s_gen;
    const size_t chunk_bytes = (size_t)s_chunk_samples * sizeof(int16_t);
    const int64_t chunk_us = (int64_t)s_chunk_samples * 1000000 / s_rate;
    int64_t next_us = esp_timer_get_time();

    s_task = xTaskGetCurrentTaskHandle();

    while (s_run && s_gen == gen) {
        /* 1. one chunk of audio into the pre-allocated buffer. The I2S DMA
         * descriptor ring is the capture buffer: i2s_channel_read blocks
         * until the driver has one chunk, so no second task and no RAM ring
         * of our own (the internal-RAM budget this engine runs in is ~16 KB). */
        if (s_source == WN_SRC_I2S) {
            size_t br = 0;
            esp_err_t e = i2s_channel_read(s_i2s, s_chunk, chunk_bytes, &br,
                                           pdMS_TO_TICKS(WN_I2S_READ_TIMEOUT_MS));
            if (e != ESP_OK || br != chunk_bytes) {
                wn_post_event(WN_EV_ERROR, (e != ESP_OK) ? (int32_t)e : ESP_ERR_INVALID_SIZE);
                vTaskDelay(pdMS_TO_TICKS(100)); /* back off a dying bus */
                continue;
            }
        } else {
            wn_test_fill(s_chunk, s_chunk_samples);
        }

        /* 2. detect, with the per-call timing the Go side prints. */
        int64_t t0 = esp_timer_get_time();
        wakenet_state_t st = s_wn->detect(s_md, s_chunk);
        uint32_t dt = (uint32_t)(esp_timer_get_time() - t0);

        uint32_t f = s_frames + 1;
        s_avg_us = (uint32_t)((int32_t)s_avg_us
                              + ((int32_t)dt - (int32_t)s_avg_us) / (int32_t)f);
        s_frames = f;
        if (dt > s_max_us) {
            s_max_us = dt;
        }

        /* 3. a detection: one event, then re-arm. clean() resets the engine's
         * frame state so the next utterance can trigger again; the test
         * source rewinds so the injected wake word replays. */
        if (st > 0) {
            s_detected = 1;
            wn_post_event(WN_EV_DETECTED, (int32_t)st);
            s_wn->clean(s_md);
            s_test_pos = 0;
        }

        /* 4. keep test-source time: one chunk per real chunk period, so the
         * printed CPU numbers mean what they say (detect ~x ms of every 32).
         * Overruns (detect longer than the chunk) skip ahead instead of
         * accumulating a backlog of fictional audio. */
        if (s_source == WN_SRC_TEST) {
            next_us += chunk_us;
            int64_t now = esp_timer_get_time();
            if (next_us > now) {
                int64_t d = (next_us - now + 999) / 1000;
                vTaskDelay(pdMS_TO_TICKS((uint32_t)d));
            } else {
                next_us = now;
            }
        }
    }

    /* Acknowledge exit before the last statement: a Close that saw s_task
     * non-NULL waits on this bit, and the only thing left afterwards is the
     * self-delete. */
    if (s_exit_ev != NULL) {
        xEventGroupSetBits(s_exit_ev, 1);
    }
    s_task = NULL;
    vTaskDelete(NULL);
}

/* ---- Go-facing ABI ---- */

/* Resolve the model ("" = the first wn* model in the partition), create the
 * engine, allocate the detect buffer and - for SourceI2S - bring up the RX
 * channel (not enabled until Start). One-shot and blocking: the model create
 * decompresses weights once (a lifecycle freeze of the same class as
 * ble.Init; the recurring heavy call, detect, never runs here). */
int32_t llgo_wakenet_init(const uint8_t *model, int32_t model_len,
                          int32_t source, int32_t bclk, int32_t ws, int32_t din,
                          int32_t threshold_milli,
                          int32_t *chunk_samples, int32_t *sample_rate,
                          uint8_t *name_out, int32_t name_cap, int32_t *name_len)
{
    if (s_state != WN_ST_UNINIT) {
        return ESP_ERR_INVALID_STATE;
    }
    s_models = esp_srmodel_init("model");
    if (s_models == NULL) {
        return ESP_ERR_NOT_FOUND; /* no usable "model" partition */
    }

    /* resolve the model name */
    if (model_len > 0) {
        if (model_len >= WN_MODEL_NAME_MAX) {
            esp_srmodel_deinit(s_models);
            s_models = NULL;
            return ESP_ERR_INVALID_ARG;
        }
        memcpy(s_model_name, model, (size_t)model_len);
        s_model_name[model_len] = '\0';
        if (esp_srmodel_exists(s_models, s_model_name) < 0) {
            esp_srmodel_deinit(s_models);
            s_models = NULL;
            return ESP_ERR_NOT_FOUND;
        }
    } else {
        char *first = esp_srmodel_filter(s_models, ESP_WN_PREFIX, NULL);
        if (first == NULL) {
            esp_srmodel_deinit(s_models);
            s_models = NULL;
            return ESP_ERR_NOT_FOUND;
        }
        if (strlen(first) >= WN_MODEL_NAME_MAX) {
            esp_srmodel_deinit(s_models);
            s_models = NULL;
            return ESP_ERR_INVALID_ARG;
        }
        strcpy(s_model_name, first);
    }

    s_wn = esp_wn_handle_from_name(s_model_name);
    if (s_wn == NULL) {
        esp_srmodel_deinit(s_models);
        s_models = NULL;
        return ESP_ERR_NOT_FOUND;
    }
    /* DET_MODE_95: weights load to PSRAM (S31 has 16 MB; the flash-load
     * modes are wakenet10-only per esp_wn_iface.h). */
    s_md = s_wn->create(s_model_name, DET_MODE_95);
    if (s_md == NULL) {
        esp_srmodel_deinit(s_models);
        s_models = NULL;
        return ESP_ERR_NO_MEM;
    }
    s_chunk_samples = s_wn->get_samp_chunksize(s_md);
    s_rate = s_wn->get_samp_rate(s_md);

    s_chunk = (int16_t *)heap_caps_malloc(
        (size_t)s_chunk_samples * sizeof(int16_t),
        MALLOC_CAP_INTERNAL | MALLOC_CAP_8BIT);
    if (s_chunk == NULL) {
        s_wn->destroy(s_md);
        s_md = NULL;
        esp_srmodel_deinit(s_models);
        s_models = NULL;
        return ESP_ERR_NO_MEM;
    }

    /* optional detection threshold, in thousandths (floats never cross the
     * llgo soft-float / IDF hard-float boundary; the Go side validated the
     * 0.4..0.9999 range already) */
    if (threshold_milli > 0) {
        s_wn->set_det_threshold(s_md, (float)threshold_milli / 1000.0f, 1);
    }

    if (source == WN_SRC_I2S) {
        /* The DMA descriptor ring is the capture buffer the task reads from:
         * 8 descriptors x 256 frames = 2048 samples = 128 ms of slack at
         * 16 kHz, far above the worst single-detect time. Mono takes the
         * left slot (WS low) - the conventional digital-mic wiring. */
        i2s_chan_config_t ch_cfg = I2S_CHANNEL_DEFAULT_CONFIG(I2S_NUM_AUTO, I2S_ROLE_MASTER);
        ch_cfg.dma_desc_num = 8;
        ch_cfg.dma_frame_num = 256;
        esp_err_t e = i2s_new_channel(&ch_cfg, NULL, &s_i2s);
        if (e != ESP_OK) {
            heap_caps_free(s_chunk);
            s_chunk = NULL;
            s_wn->destroy(s_md);
            s_md = NULL;
            esp_srmodel_deinit(s_models);
            s_models = NULL;
            return e;
        }
        i2s_std_config_t std_cfg = {
            .clk_cfg = I2S_STD_CLK_DEFAULT_CONFIG((uint32_t)s_rate),
            .slot_cfg = I2S_STD_PHILIPS_SLOT_DEFAULT_CONFIG(I2S_DATA_BIT_WIDTH_16BIT,
                                                            I2S_SLOT_MODE_MONO),
            .gpio_cfg = {
                .mclk = I2S_GPIO_UNUSED,
                .bclk = (bclk >= 0) ? (gpio_num_t)bclk : I2S_GPIO_UNUSED,
                .ws = (ws >= 0) ? (gpio_num_t)ws : I2S_GPIO_UNUSED,
                .dout = I2S_GPIO_UNUSED,
                .din = (din >= 0) ? (gpio_num_t)din : I2S_GPIO_UNUSED,
            },
        };
        e = i2s_channel_init_std_mode(s_i2s, &std_cfg);
        if (e != ESP_OK) {
            i2s_del_channel(s_i2s);
            s_i2s = NULL;
            heap_caps_free(s_chunk);
            s_chunk = NULL;
            s_wn->destroy(s_md);
            s_md = NULL;
            esp_srmodel_deinit(s_models);
            s_models = NULL;
            return e;
        }
    }

    s_exit_ev = xEventGroupCreate();
    if (s_exit_ev == NULL) {
        /* unimplemented cleanup path mirrors above, with the i2s channel */
        if (s_i2s != NULL) {
            i2s_del_channel(s_i2s);
            s_i2s = NULL;
        }
        heap_caps_free(s_chunk);
        s_chunk = NULL;
        s_wn->destroy(s_md);
        s_md = NULL;
        esp_srmodel_deinit(s_models);
        s_models = NULL;
        return ESP_ERR_NO_MEM;
    }

    s_source = source;
    s_run = 0;
    s_gen = 0;
    s_detected = 0;
    s_frames = 0;
    s_avg_us = 0;
    s_max_us = 0;
    s_state = WN_ST_READY;

    if (chunk_samples != NULL) {
        *chunk_samples = s_chunk_samples;
    }
    if (sample_rate != NULL) {
        *sample_rate = s_rate;
    }
    int32_t n = (int32_t)strlen(s_model_name);
    if (name_out != NULL && name_cap > 0) {
        if (n > name_cap) {
            n = name_cap;
        }
        memcpy(name_out, s_model_name, (size_t)n);
    }
    if (name_len != NULL) {
        *name_len = n;
    }
    return ESP_OK;
}

/* Start the audio + detect task. Returns immediately: the task's own
 * failures (I2S read errors) arrive as events, not as a return code. Stats
 * and the sticky detected flag reset here, before the task exists, so no
 * writer races the clear. */
int32_t llgo_wakenet_start(void)
{
    if (s_state == WN_ST_UNINIT) {
        return ESP_ERR_INVALID_STATE;
    }
    if (s_state == WN_ST_RUNNING) {
        return ESP_ERR_INVALID_STATE;
    }
    if (s_source == WN_SRC_I2S) {
        esp_err_t e = i2s_channel_enable(s_i2s);
        if (e != ESP_OK) {
            return e;
        }
    }
    s_detected = 0;
    s_frames = 0;
    s_avg_us = 0;
    s_max_us = 0;
    s_run = 1;
    s_gen++;
    s_state = WN_ST_RUNNING;
    xEventGroupClearBits(s_exit_ev, 1);
    BaseType_t ok = xTaskCreatePinnedToCore(wakenet_task, "llgo_wakenet", 4096,
                                            NULL, 5, NULL, 1);
    if (ok != pdPASS) {
        s_run = 0;
        s_state = WN_ST_READY;
        if (s_source == WN_SRC_I2S) {
            i2s_channel_disable(s_i2s);
        }
        return ESP_ERR_NO_MEM;
    }
    return ESP_OK;
}

/* Stop the task. Cooperative: the task notices s_run == 0 at its loop top,
 * so it exits within one chunk period (never mid-C-call), and an immediate
 * re-Start is safe because the old task also bails when the generation it
 * captured no longer matches. The I2S channel stays enabled until Close so a
 * blocked i2s_channel_read still completes and the exit is fast. */
int32_t llgo_wakenet_stop(void)
{
    if (s_state != WN_ST_RUNNING) {
        return ESP_ERR_INVALID_STATE;
    }
    s_run = 0;
    s_state = WN_ST_READY;
    return ESP_OK;
}

/* Tear everything down: the task (waited for, bounded), the I2S channel, the
 * engine, the model list, the buffers. */
int32_t llgo_wakenet_close(void)
{
    if (s_state == WN_ST_UNINIT) {
        return ESP_ERR_INVALID_STATE;
    }
    s_run = 0;
    if (s_task != NULL) {
        EventBits_t bits = xEventGroupWaitBits(s_exit_ev, 1, pdFALSE, pdFALSE,
                                               pdMS_TO_TICKS(3000));
        if (!(bits & 1)) {
            return ESP_ERR_TIMEOUT; /* task wedged; nothing was freed */
        }
    }
    if (s_source == WN_SRC_I2S && s_i2s != NULL) {
        if (s_state == WN_ST_RUNNING || s_state == WN_ST_READY) {
            /* enabled since Start; disable is legal either way for this
             * driver once the reader is gone */
            i2s_channel_disable(s_i2s);
        }
        i2s_del_channel(s_i2s);
        s_i2s = NULL;
    }
    if (s_md != NULL) {
        s_wn->destroy(s_md);
        s_md = NULL;
    }
    if (s_chunk != NULL) {
        heap_caps_free(s_chunk);
        s_chunk = NULL;
    }
    if (s_models != NULL) {
        esp_srmodel_deinit(s_models);
        s_models = NULL;
    }
    if (s_exit_ev != NULL) {
        vEventGroupDelete(s_exit_ev);
        s_exit_ev = NULL;
    }
    s_test_pcm = NULL;
    s_test_pcm_len = 0;
    s_test_pos = 0;
    s_state = WN_ST_UNINIT;
    return ESP_OK;
}

int32_t llgo_wakenet_detected(void)
{
    return s_detected;
}

/* Pop one event record (1) or report an empty ring (0). Consumer-owned read
 * index; the acquire on wr makes the record visible before the index says so. */
int32_t llgo_wakenet_poll(llgo_wakenet_ev_t *out)
{
    uint32_t rd = s_ring_rd;
    if (rd == WN_ACQUIRE(s_ring_wr)) {
        return 0;
    }
    *out = s_ring[rd % WN_EV_SLOTS];
    WN_PUBLISH(s_ring_rd, rd + 1);
    return 1;
}

/* How many records the producer dropped since the previous call. The
 * consumer keeps its own snapshot - the producer's counter is never cleared,
 * so a delta read racing a producer increment can only miss the very last
 * drop, the same tolerance ble_glue documents. */
int32_t llgo_wakenet_overflow_delta(void)
{
    uint32_t d = WN_ACQUIRE(s_dropped);
    int32_t delta = (int32_t)(d - s_dropped_seen);
    s_dropped_seen = d;
    return delta;
}

int32_t llgo_wakenet_stats(int32_t *frames, int32_t *avg_us, int32_t *max_us)
{
    if (frames != NULL) {
        *frames = (int32_t)s_frames;
    }
    if (avg_us != NULL) {
        *avg_us = (int32_t)s_avg_us;
    }
    if (max_us != NULL) {
        *max_us = (int32_t)s_max_us;
    }
    return ESP_OK;
}

uint32_t llgo_wakenet_now_ms(void)
{
    return (uint32_t)(esp_timer_get_time() / 1000);
}

/* Demo hook: point the test source at a PCM array (16 kHz 16-bit mono, e.g.
 * a wake-word utterance embedded as const data). The glue only keeps the
 * pointer - the array must stay live for the engine's lifetime, which a
 * flash-resident const array naturally does. Calling this is optional: the
 * test source feeds silence without it. */
void llgo_wakenet_set_test_pcm(const int16_t *pcm, int32_t samples)
{
    s_test_pcm = pcm;
    s_test_pcm_len = samples;
    s_test_pos = 0;
}

#else /* !ESP_PLATFORM */

/* Template guard: outside an ESP-IDF build this file compiles to an
 * otherwise empty translation unit. */
typedef int llgo_wakenet_glue_not_in_idf_build;

#endif /* ESP_PLATFORM */
