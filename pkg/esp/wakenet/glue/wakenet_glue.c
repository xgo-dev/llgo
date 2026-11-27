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
 *   (512 samples @ 16 kHz), and the engine's compute path uses the S31's PIE
 *   coprocessor - which exists on core 1 only. The llgo fiber scheduler is
 *   single-task cooperative state whose SP lives on swapped-out fiber stacks
 *   in the GC heap, so neither the milliseconds of compute nor the PIE
 *   ownership rules may ever touch the scheduler task: a PIE instruction on
 *   core 0 makes the FreeRTOS port migrate the task to core 1, and the
 *   migration path validates the task TCB's stack pointer - which for the
 *   scheduler task is a fiber stack, outside the FreeRTOS stack bounds, and
 *   the kernel aborts. So the whole engine lives on THIS glue's own
 *   FreeRTOS task, pinned to core 1:
 *
 *     Init (Go, scheduler task): parse config, map the model partition,
 *       bring up the optional I2S RX channel - no PIE anywhere - then start
 *       the session task and wait (bounded) for its engine-ready/fail sign
 *     session task (core 1): create the engine (the PIE-heavy part), signal
 *       ready, then loop: I2S RX (16 kHz 16-bit mono, DMA descriptor ring)
 *       or the built-in test source -> one chunk -> detect (timed) ->
 *       detection/error into a bounded static event ring
 *
 *   Go only polls the ring (microseconds per record) and reads counters;
 *   detect never crosses from Go, and the scheduler task never executes a
 *   PIE instruction.
 *
 * The session task allocates nothing in its loop: the chunk buffer is one
 * allocation at Init, the event ring is static, and the test source reads a
 * caller-injected flash-resident PCM array through a pointer.
 *
 * Cross-task visibility: the ring indexes and the stats are single-writer
 * fields published with release stores and read with acquire loads (the same
 * contract ble_glue uses; no RMW atomics anywhere, so RV32IMC llgo code never
 * drags in libatomic, and the plain u32 stats cannot tear on RV32). The
 * engine state transitions are all written by the scheduler task (which owns
 * s_state, s_run, s_quit and never runs concurrently with itself), except
 * the counters the session task owns.
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

/* Event-group bits: the session task's signs to the scheduler task. */
#define WN_BIT_EXIT  0x1  /* session task left (Close path) */
#define WN_BIT_READY 0x2  /* engine created */
#define WN_BIT_FAIL  0x4  /* engine create failed (code in s_init_err) */

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

/* ---- configuration (written by the scheduler task during Init, read by
 * the session task) ---- */

#define WN_MODEL_NAME_MAX 32
#define WN_EV_SLOTS 8
#define WN_I2S_READ_TIMEOUT_MS 2000
#define WN_CREATE_WAIT_MS 10000 /* bounded Init wait for the engine create */
/* The session task's stack: 12 KiB is the measured wn9 floor (see Init). It
 * is a caller-provided block, so the word count and the byte count both get
 * names - xTaskCreateStatic wants words, heap_caps_malloc wants bytes. */
#define WN_TASK_STACK_WORDS 12288
#define WN_TASK_STACK_BYTES (WN_TASK_STACK_WORDS * (uint32_t)sizeof(StackType_t))

static int32_t s_state;                     /* scheduler-task written */
static int32_t s_source;                    /* scheduler-task written */
static int32_t s_threshold_milli;           /* scheduler-task written */
static char s_model_name[WN_MODEL_NAME_MAX];
/* The I2S RX pin set, kept so the capture channel can be released for the
 * length of a conversation round and rebuilt on Start. The board's I2S pins
 * are single-owner: a parked-but-live channel still routes BCLK/WS through
 * the GPIO matrix and the round's own channel then never clocks (measured:
 * playback write timeout 263). */
static int32_t s_i2s_bclk = -1, s_i2s_ws = -1, s_i2s_din = -1;
static volatile int s_bus_free = 1;         /* 1 = no capture channel exists */
static volatile int32_t s_last_rms;         /* RMS of the chunk the engine last saw */
static volatile int32_t s_peak_rms;         /* max RMS since Start */
static volatile int32_t s_hits;             /* chunks where detect() returned >0 since Start */

static i2s_chan_handle_t s_i2s;                    /* created at Init (scheduler
                                             * task), used by the session task */

/* ---- engine (session-task owned after its create; Close uses it only
 * after the task acknowledged exit) ---- */

static srmodel_list_t *s_models;
static const esp_wn_iface_t *s_wn;
static model_iface_data_t *s_md;
static int16_t *s_chunk;
/* I2S capture staging: the ES8311 sends stereo frames even with one
 * microphone (the idle slot repeats the active one - measured: the read
 * completed at 2x the expected rate with a mono slot config, i.e. the
 * driver kept both slots). The capture reads stereo into this buffer and
 * deinterleaves the left slot into s_chunk, so the engine always sees
 * contiguous 16 kHz mono. */
static int16_t *s_chunk_raw;
/* Tracks whether the capture channel is currently enabled: IDF logs an E
 * line for a disable on a disabled channel, and Close always has to make
 * sure the channel ends up disabled whichever side did it last. */
static volatile int s_i2s_enabled;
static int32_t s_chunk_samples;
static int32_t s_rate;

/* ---- session control ---- */

static TaskHandle_t s_task;                 /* session-task written */
static StackType_t *s_task_stack;           /* internal-RAM block, freed at
                                             * Close after the exit bit */
static StaticTask_t s_task_tcb;             /* caller-provided TCB storage */
static EventGroupHandle_t s_ev;             /* created at Init */
static volatile int32_t s_run;              /* scheduler-task written */
static volatile int32_t s_quit;             /* scheduler-task written */
static volatile int32_t s_init_err;         /* session-task written (create
                                             * result; read after BIT_FAIL) */
static volatile TaskHandle_t s_self;        /* session-task written */

/* detection state + stats: the session task is the only writer, Go only
 * reads. Plain u32 fields cannot tear on RV32; frames/avg/max may be one
 * frame apart from each other in a read, which a diagnostic does not mind. */
static volatile int32_t s_detected;
static volatile int32_t s_in_detect;        /* rising-edge trigger state */
static volatile uint32_t s_frames;
static volatile uint32_t s_avg_us;
static volatile uint32_t s_max_us;

/* test source: a caller-injected PCM array (flash-resident const data; the
 * glue never copies it) replayed in a loop, or silence when nothing was
 * injected. */
static const int16_t *s_test_pcm;
static int32_t s_test_pcm_len;
static int32_t s_test_pos;                  /* session-task owned */

/* ---- event ring (SPSC: the session task produces, the Go poller consumes;
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

/* Create the capture channel from the stored pin set. Runs on the session
 * task (Start edge) or the scheduler task (Init). */
static esp_err_t wn_i2s_open(void)
{
    i2s_chan_config_t ch_cfg = I2S_CHANNEL_DEFAULT_CONFIG(I2S_NUM_AUTO, I2S_ROLE_MASTER);
    ch_cfg.dma_desc_num = 8;
    ch_cfg.dma_frame_num = 256;
    esp_err_t e = i2s_new_channel(&ch_cfg, NULL, &s_i2s);
    if (e != ESP_OK) {
        s_i2s = NULL;
        return e;
    }
    i2s_std_config_t std_cfg = {
        .clk_cfg = I2S_STD_CLK_DEFAULT_CONFIG(16000),
        .slot_cfg = I2S_STD_PHILIPS_SLOT_DEFAULT_CONFIG(I2S_DATA_BIT_WIDTH_16BIT,
                                                        I2S_SLOT_MODE_STEREO),
        .gpio_cfg = {
            .mclk = I2S_GPIO_UNUSED,
            .bclk = (s_i2s_bclk >= 0) ? (gpio_num_t)s_i2s_bclk : I2S_GPIO_UNUSED,
            .ws = (s_i2s_ws >= 0) ? (gpio_num_t)s_i2s_ws : I2S_GPIO_UNUSED,
            .dout = I2S_GPIO_UNUSED,
            .din = (s_i2s_din >= 0) ? (gpio_num_t)s_i2s_din : I2S_GPIO_UNUSED,
        },
    };
    e = i2s_channel_init_std_mode(s_i2s, &std_cfg);
    if (e != ESP_OK) {
        i2s_del_channel(s_i2s);
        s_i2s = NULL;
        return e;
    }
    s_bus_free = 0;
    return ESP_OK;
}

/* Hand the pins back to whoever runs next (the conversation round). */
static void wn_i2s_release(void)
{
    if (s_i2s == NULL) {
        s_bus_free = 1;
        return;
    }
    if (s_i2s_enabled) {
        i2s_channel_disable(s_i2s);
        s_i2s_enabled = 0;
    }
    i2s_del_channel(s_i2s);
    s_i2s = NULL;
    s_bus_free = 1;
}

/* ---- the session task (core 1): engine create + audio + detect ---- */

static void wakenet_task(void *arg)
{
    (void)arg;
    s_self = xTaskGetCurrentTaskHandle();

    /* 1. create the engine - the PIE-heavy part, on the only core that has
     * PIE. DET_MODE_95 loads the weights to PSRAM (S31 has 16 MB; the
     * flash-load modes are wakenet10-only per esp_wn_iface.h). */
    s_md = s_wn->create(s_model_name, DET_MODE_95);
    if (s_md == NULL) {
        s_init_err = ESP_ERR_NO_MEM;
        xEventGroupSetBits(s_ev, WN_BIT_FAIL);
        goto done;
    }
    s_chunk_samples = s_wn->get_samp_chunksize(s_md);
    s_rate = s_wn->get_samp_rate(s_md);

    s_chunk = (int16_t *)heap_caps_malloc(
        (size_t)s_chunk_samples * sizeof(int16_t),
        MALLOC_CAP_INTERNAL | MALLOC_CAP_8BIT);
    if (s_chunk == NULL) {
        s_wn->destroy(s_md);
        s_md = NULL;
        s_init_err = ESP_ERR_NO_MEM;
        xEventGroupSetBits(s_ev, WN_BIT_FAIL);
        goto done;
    }
    if (s_source == WN_SRC_I2S) {
        s_chunk_raw = (int16_t *)heap_caps_malloc(
            (size_t)s_chunk_samples * 2 * sizeof(int16_t),
            MALLOC_CAP_INTERNAL | MALLOC_CAP_8BIT);
        if (s_chunk_raw == NULL) {
            heap_caps_free(s_chunk);
            s_chunk = NULL;
            s_wn->destroy(s_md);
            s_md = NULL;
            s_init_err = ESP_ERR_NO_MEM;
            xEventGroupSetBits(s_ev, WN_BIT_FAIL);
            goto done;
        }
    }

    /* optional detection threshold, in thousandths (floats never cross the
     * llgo soft-float / IDF hard-float boundary; the Go side validated the
     * 0.4..0.9999 range already) */
    if (s_threshold_milli > 0) {
        s_wn->set_det_threshold(s_md, (float)s_threshold_milli / 1000.0f, 1);
    }

    s_init_err = ESP_OK;
    xEventGroupSetBits(s_ev, WN_BIT_READY);

    /* 2. the audio + detect loop, until Close. Stop parks the loop; Start
     * resumes it. Every run transition resets the per-run counters here -
     * single writer, no races with the scheduler task. */
    int32_t was_run = 0;
    while (!s_quit) {
        if (!s_run) {
            was_run = 0;
            vTaskDelay(pdMS_TO_TICKS(10)); /* parked: 10 ms poll */
            continue;
        }
        if (!was_run) {
            /* rising edge of s_run: reset the per-run state */
            was_run = 1;
            s_detected = 0;
            s_in_detect = 0;
            s_frames = 0;
            s_avg_us = 0;
            s_max_us = 0;
            s_test_pos = 0;
            s_peak_rms = 0;
            s_hits = 0;
            if (s_source == WN_SRC_I2S) {
                if (s_i2s == NULL) {
                    esp_err_t oe = wn_i2s_open();
                    if (oe != ESP_OK) {
                        wn_post_event(WN_EV_ERROR, (int32_t)oe);
                        s_run = 0;
                        continue;
                    }
                }
                i2s_channel_enable(s_i2s);
                s_i2s_enabled = 1;
                /* TX stays disabled until the first playback write: with
                 * the pair enabled together the RX frames carried audio
                 * (levels normal) but the engine never detected again
                 * (measured: hits stuck at 0), so the playback side is
                 * armed lazily. */
            }
        }
        const size_t chunk_bytes = (size_t)s_chunk_samples * sizeof(int16_t);
        const int64_t chunk_us = (int64_t)s_chunk_samples * 1000000 / s_rate;
        int64_t next_us = esp_timer_get_time();

        while (s_run && !s_quit) {
            /* one chunk of audio into the pre-allocated buffer. The I2S DMA
             * descriptor ring is the capture buffer: i2s_channel_read blocks
             * until the driver has one chunk, so no second task and no RAM
             * ring of our own (the internal-RAM budget this engine runs in
             * is ~16 KB). */
            if (s_source == WN_SRC_I2S) {
                /* The bus carries stereo frames (16-bit, 2 slots), so one
                 * engine chunk is chunk_bytes * 2 on the wire. Reading only
                 * chunk_bytes made the loop run at twice the chunk rate and
                 * handed the engine an interleaved L/R stream - both break
                 * detection (measured on hardware: 2x frame rate, zero
                 * hits). Read stereo, then take the left slot. */
                size_t br = 0;
                esp_err_t e = i2s_channel_read(s_i2s, s_chunk_raw, chunk_bytes * 2, &br,
                                               pdMS_TO_TICKS(WN_I2S_READ_TIMEOUT_MS));
                if (e != ESP_OK || br != chunk_bytes * 2) {
                    wn_post_event(WN_EV_ERROR, (e != ESP_OK) ? (int32_t)e : ESP_ERR_INVALID_SIZE);
                    vTaskDelay(pdMS_TO_TICKS(100)); /* back off a dying bus */
                    continue;
                }
                for (int32_t i = 0; i < s_chunk_samples; i++) {
                    s_chunk[i] = s_chunk_raw[i * 2];
                }
            } else {
                wn_test_fill(s_chunk, s_chunk_samples);
            }

            /* RMS of what the engine is about to see: the difference
             * between "no wake word" and "no microphone" - the frame
             * counter alone cannot tell them apart. */
            {
                int64_t acc = 0;
                for (int32_t i = 0; i < s_chunk_samples; i++) {
                    int32_t v = s_chunk[i];
                    acc += (int64_t)v * v;
                }
                uint32_t mean = (uint32_t)(acc / s_chunk_samples);
                uint32_t r = 0;
                for (uint32_t bit = 1u << 15; bit != 0; bit >>= 1) {
                    uint32_t trial = r | bit;
                    if (trial * trial <= mean) {
                        r = trial;
                    }
                }
                s_last_rms = (int32_t)r;
                if ((int32_t)r > s_peak_rms) {
                    s_peak_rms = (int32_t)r;
                }
            }

            /* detect, with the per-call timing the Go side prints. */
            int64_t t0 = esp_timer_get_time();
            wakenet_state_t st = s_wn->detect(s_md, s_chunk);
            uint32_t dt = (uint32_t)(esp_timer_get_time() - t0);

            if (st > 0) {
                s_hits = s_hits + 1;
            }

            uint32_t f = s_frames + 1;
            s_avg_us = (uint32_t)((int32_t)s_avg_us
                                  + ((int32_t)dt - (int32_t)s_avg_us) / (int32_t)f);
            s_frames = f;
            if (dt > s_max_us) {
                s_max_us = dt;
            }

            /* a detection: one event on the rising edge. The engine re-arms
             * itself - measured on this combination (S31, esp-sr 2.6.2, wn9):
             * after WAKENET_DETECTED the state returns to 0 within a few
             * frames of silence on its own - so the glue must NOT call
             * clean(): model_clean crashes on this build (dl_convq_queue_bzero
             * on a NULL queue, before and after a detection alike; the
             * upstream test app never calls it, which is why upstream CI
             * never sees it). The edge trigger keeps a detection's trailing
             * frames from flooding events; the test source rewinds so the
             * injected wake word replays. */
            if (st > 0 && !s_in_detect) {
                s_in_detect = 1;
                s_detected = 1;
                wn_post_event(WN_EV_DETECTED, (int32_t)st);
                s_test_pos = 0;
            } else if (st == 0) {
                s_in_detect = 0;
            }

            /* keep test-source time: one chunk per real chunk period, so the
             * printed CPU numbers mean what they say (detect ~x ms of every
             * 32). Overruns (detect longer than the chunk) skip ahead
             * instead of accumulating a backlog of fictional audio. */
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

        /* s_run dropped: hand the I2S pins back so the conversation round
         * can open its own channels; Start rebuilds the capture channel. */
        if (s_source == WN_SRC_I2S) {
            wn_i2s_release();
        }
    }

done:
    /* Acknowledge exit before the last statement: a Close that saw s_task
     * non-NULL waits on this bit, and the only thing left afterwards is the
     * self-delete. */
    xEventGroupSetBits(s_ev, WN_BIT_EXIT);
    s_task = NULL;
    vTaskDelete(NULL);
}

/* ---- Go-facing ABI ---- */

/* Parse the config, map the model partition, resolve the model name ("" =
 * the first wn* model), bring up the optional I2S RX channel, then start the
 * session task and wait (bounded) for its engine create. NOTHING here runs
 * the engine's compute: wn9's create/detect path executes PIE instructions,
 * and PIE exists on core 1 only - a PIE instruction on the scheduler task
 * (core 0) makes the FreeRTOS port migrate the task to core 1 and abort on
 * the llgo scheduler's swapped-out SP. The blocking here is a bounded wait
 * on the session task's ready sign (26 ms measured for wn9 on S31), the
 * same one-shot lifecycle freeze as ble.Init. */
int32_t llgo_wakenet_init(const uint8_t *model, int32_t model_len,
                          int32_t source, int32_t bclk, int32_t ws, int32_t din,
                          int32_t threshold_milli)
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

    if (source == WN_SRC_I2S) {
        /* The channel itself opens lazily (wn_i2s_open): the pins are
         * single-owner, so the engine only holds them while it is running.
         * The DMA ring is 8 x 256 frames = 128 ms at 16 kHz; the bus is
         * configured STEREO because that is what the codec actually drives
         * (measured: a mono slot config still delivered 2 bytes per frame
         * per slot, doubling the read rate). */
        s_i2s_bclk = bclk;
        s_i2s_ws = ws;
        s_i2s_din = din;
    }

    s_ev = xEventGroupCreate();
    if (s_ev == NULL) {
        if (s_i2s != NULL) {
            i2s_del_channel(s_i2s);
            s_i2s = NULL;
        }
        esp_srmodel_deinit(s_models);
        s_models = NULL;
        return ESP_ERR_NO_MEM;
    }

    /* the session task: 12288 stack WORDS = 48 KiB (wn9's detect frames are
     * the dominant cost - 4 Ki words dies on the first call, "Stack overflow
     * while saving coprocessor context", measured under esp-emu; the esp-sr
     * CI runs its synchronous detect on a 10 Ki task stack), pinned to core 1
     * (PIE). The stack is carved from internal RAM explicitly: the IDF heap
     * gives a plain xTaskCreate stack a PSRAM block when internal RAM runs
     * tight (WiFi + wakenet in one firmware, measured), and a coprocessor
     * context cannot be saved on a PSRAM stack - the first cache freeze dies
     * with s_task_stack_is_sane_when_cache_frozen(). xTaskCreateStatic with a
     * MALLOC_CAP_INTERNAL block pins the placement. */
    s_init_err = ESP_FAIL;
    s_source = source;
    s_threshold_milli = threshold_milli;
    s_run = 0;
    s_quit = 0;
    s_task_stack = heap_caps_malloc(WN_TASK_STACK_BYTES,
                                    MALLOC_CAP_INTERNAL | MALLOC_CAP_8BIT);
    if (s_task_stack == NULL) {
        vEventGroupDelete(s_ev);
        s_ev = NULL;
        if (s_i2s != NULL) {
            i2s_del_channel(s_i2s);
            s_i2s = NULL;
        }
        esp_srmodel_deinit(s_models);
        s_models = NULL;
        return ESP_ERR_NO_MEM;
    }
    /* IDF's variant returns the handle (NULL on failure) and takes no
     * pxCreatedTask out-param, unlike vanilla FreeRTOS xTaskCreateStatic. */
    s_task = xTaskCreateStaticPinnedToCore(wakenet_task, "llgo_wakenet",
                                           WN_TASK_STACK_WORDS, NULL, 5,
                                           s_task_stack, &s_task_tcb, 1);
    if (s_task == NULL) {
        s_task = NULL;
        heap_caps_free(s_task_stack);
        s_task_stack = NULL;
        vEventGroupDelete(s_ev);
        s_ev = NULL;
        if (s_i2s != NULL) {
            i2s_del_channel(s_i2s);
            s_i2s = NULL;
        }
        esp_srmodel_deinit(s_models);
        s_models = NULL;
        return ESP_ERR_NO_MEM;
    }

    /* bounded wait for the engine create result */
    EventBits_t bits = xEventGroupWaitBits(s_ev, WN_BIT_READY | WN_BIT_FAIL,
                                           pdFALSE, pdFALSE,
                                           pdMS_TO_TICKS(WN_CREATE_WAIT_MS));
    if (!(bits & (WN_BIT_READY | WN_BIT_FAIL))) {
        return ESP_ERR_TIMEOUT; /* the create never finished; Close tears down */
    }
    if (bits & WN_BIT_FAIL) {
        /* the task freed what it allocated; only the model list, channel,
         * event group and task remain for Close */
        return s_init_err;
    }

    s_state = WN_ST_READY;
    return ESP_OK;
}

/* Start the audio loop inside the session task. Returns immediately; the
 * task resets the per-run counters itself on the run transition (single
 * writer). Failures inside the loop arrive as events, not as a return code. */
int32_t llgo_wakenet_start(void)
{
    if (s_state == WN_ST_UNINIT) {
        return ESP_ERR_INVALID_STATE;
    }
    if (s_state == WN_ST_RUNNING) {
        return ESP_ERR_INVALID_STATE;
    }
    s_state = WN_ST_RUNNING;
    s_run = 1;
    return ESP_OK;
}

/* Park the audio loop. Cooperative: the task notices at its loop top and
 * stops capturing within one chunk period, never mid-C-call. */
int32_t llgo_wakenet_stop(void)
{
    if (s_state != WN_ST_RUNNING) {
        return ESP_ERR_INVALID_STATE;
    }
    s_run = 0;
    s_state = WN_ST_READY;
    /* I2S source: block (bounded) until the session task has deleted the
     * capture channel. Coming back before that lets the caller open its own
     * channel on pins the parked one still routes. */
    if (s_source == WN_SRC_I2S) {
        for (int i = 0; i < 200; i++) { /* 2 s cap */
            if (s_bus_free) {
                return ESP_OK;
            }
            vTaskDelay(pdMS_TO_TICKS(10));
        }
        return ESP_ERR_TIMEOUT;
    }
    return ESP_OK;
}

/* Tear everything down: the session task (waited for, bounded), the I2S
 * channel, the engine, the model list, the buffers. */
int32_t llgo_wakenet_close(void)
{
    if (s_state == WN_ST_UNINIT) {
        return ESP_ERR_INVALID_STATE;
    }
    s_run = 0;
    s_quit = 1;
    if (s_task != NULL) {
        EventBits_t bits = xEventGroupWaitBits(s_ev, WN_BIT_EXIT, pdFALSE, pdFALSE,
                                               pdMS_TO_TICKS(3000));
        if (!(bits & WN_BIT_EXIT)) {
            return ESP_ERR_TIMEOUT; /* task wedged; nothing was freed */
        }
    }
    if (s_task_stack != NULL) {
        heap_caps_free(s_task_stack);
        s_task_stack = NULL;
    }
    if (s_source == WN_SRC_I2S && s_i2s != NULL) {
        if (s_i2s_enabled) {
            i2s_channel_disable(s_i2s);
            s_i2s_enabled = 0;
        }
        i2s_del_channel(s_i2s);
        s_i2s = NULL;
    }
    if (s_md != NULL) {
        s_wn->destroy(s_md);
        s_md = NULL;
    }
    if (s_chunk_raw != NULL) {
        heap_caps_free(s_chunk_raw);
        s_chunk_raw = NULL;
    }
    if (s_chunk != NULL) {
        heap_caps_free(s_chunk);
        s_chunk = NULL;
    }
    if (s_models != NULL) {
        esp_srmodel_deinit(s_models);
        s_models = NULL;
    }
    if (s_ev != NULL) {
        vEventGroupDelete(s_ev);
        s_ev = NULL;
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

int32_t llgo_wakenet_peak_rms(void)
{
    return s_peak_rms;
}

int32_t llgo_wakenet_hits(void)
{
    return s_hits;
}

int32_t llgo_wakenet_last_rms(void)
{
    return s_last_rms;
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

int32_t llgo_wakenet_chunk(int32_t *chunk_samples, int32_t *sample_rate)
{
    if (chunk_samples != NULL) {
        *chunk_samples = s_chunk_samples;
    }
    if (sample_rate != NULL) {
        *sample_rate = s_rate;
    }
    return ESP_OK;
}

uint32_t llgo_wakenet_model_name(uint8_t *out, int32_t cap)
{
    int32_t n = (int32_t)strlen(s_model_name);
    if (out != NULL && cap > 0) {
        if (n > cap) {
            n = cap;
        }
        memcpy(out, s_model_name, (size_t)n);
    }
    return n;
}

int32_t llgo_wakenet_now_ms(void)
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
