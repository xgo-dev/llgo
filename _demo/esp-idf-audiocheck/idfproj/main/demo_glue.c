/* Demo-local C glue for the audiocheck firmware (opus self-check + voice
 * loop + wakenet, one firmware).
 *
 * Everything here is demo-owned (nothing belongs to a package template):
 *
 *   - llgo_ac_now_us / llgo_ac_now_ms: the esp_timer domain clocks, for the
 *     phase timings (encode/decode wall time, heartbeats, the acceptance
 *     window). Under esp-emu these do not run at board speed - the demo
 *     says so on its timing lines; the HW numbers are the real ones.
 *   - llgo_ac_heap: the IDF pool ledger (the family diagnostic), printed at
 *     boot and at every phase boundary - the capture buffers and the wakenet
 *     weights are the firmware's big allocations and the psram number is
 *     where they land.
 *   - the energy VAD: a 30 ms-frame RMS follower with an adaptive noise
 *     floor, hysteresis and a hangover tail (esp-idf-voice-loop's engine,
 *     unchanged). All integer math, no esp-sr/AFE dependency; the
 *     parameters travel from Go (the demo owns the tuning) and the state
 *     machine stays here so the capture loop only sees three verdicts.
 *   - llgo_ac_test_pcm: the test-source injection for phase C's EMU run -
 *     the checked-in "Hi, Lexin" utterance (wakenet_pcm.c, the esp-sr test
 *     app's bytes) is handed to the wakenet glue's built-in test source.
 *     The array is const flash data; llgo_wakenet_set_test_pcm only keeps
 *     the pointer.
 *   - llgo_audiocheck_mclk / llgo_ac_mclk: the MCLK pin the synced wakenet
 *     glue wires into its I2S RX channel (post-sync.sh patches the
 *     template's gpio_cfg to read this; -1 keeps the upstream unused-pin
 *     default). The board microphone sits behind the ES8311, which clocks
 *     itself from MCLK - without this the HW capture would read nothing.
 *
 * The Go<->C boundary carries integers and pointers only; the VAD keeps no
 * reference to Go memory beyond the call.
 */
#include <stdint.h>
#include <stdio.h>

#include "esp_heap_caps.h"
#include "esp_timer.h"

#include "wakenet_pcm.h"

extern void llgo_wakenet_set_test_pcm(const int16_t *pcm, int32_t samples);

int64_t llgo_ac_now_us(void)
{
    return esp_timer_get_time();
}

uint32_t llgo_ac_now_ms(void)
{
    return (uint32_t)(esp_timer_get_time() / 1000);
}

/* Print the two IDF pools at one demo stage (tag consumed immediately). */
void llgo_ac_heap(const char *tag, int32_t len)
{
    printf("[c] ac heap %.*s internal=%u psram=%u\n",
           (int)len, tag,
           (unsigned)heap_caps_get_free_size(MALLOC_CAP_INTERNAL),
           (unsigned)heap_caps_get_free_size(MALLOC_CAP_SPIRAM));
    fflush(stdout);
}

/* Hand the utterance to the wakenet glue's test source; returns the sample
 * count the demo prints. */
int32_t llgo_ac_test_pcm(void)
{
    const int32_t samples = (int32_t)(sizeof(hilexin_pcm) / sizeof(hilexin_pcm[0]));
    llgo_wakenet_set_test_pcm(hilexin_pcm, samples);
    return samples;
}

/* ---- phase C capture-energy hook (post-sync.sh patches the wakenet
 * template's capture loop to call this with every chunk that feeds
 * detect) ----
 *
 * The 0-hit question needs the engine's input energy beside the Go
 * capture's (phase B): same mic, different read path. The session task
 * stores the per-chunk RMS here plus a chunk counter; the Go heartbeat
 * reads and clears them (llgo_ac_chunk_rms_get), so the printed window is
 * one heartbeat period. Single-writer counters: the counters are plain
 * 32-bit stores, and a heartbeat read one chunk behind is fine for a
 * diagnostic. EMU never calls the hook (the test source does not run
 * through the capture loop's I2S branch). */
static volatile int32_t s_ac_chunk_rms;
static volatile int32_t s_ac_chunks;

static int64_t isqrt64(int64_t x); /* defined with the VAD below */

void llgo_ac_chunk_rms(const int16_t *pcm, int32_t n)
{
    int64_t sumsq = 0;
    for (int32_t i = 0; i < n; i++) {
        int32_t v = pcm[i];
        sumsq += (int64_t)v * v;
    }
    s_ac_chunk_rms = (int32_t)isqrt64(sumsq / n);
    s_ac_chunks++;
}

void llgo_ac_chunk_rms_get(int32_t *rms, int32_t *chunks)
{
    *rms = s_ac_chunk_rms;
    *chunks = s_ac_chunks;
    s_ac_chunks = 0;
}

/* ---- phase C input-level fix (post-sync.sh inserts the call at the end
 * of the capture loop's I2S branch, before detect) ----
 *
 * The board's record path measures speech at rms ~270 (-42 dBFS) - 15 dB
 * below the esp-sr test utterance that detects cleanly (rms 1529), and the
 * codec's 0x16 digital gain does not deliver its +24 dB here (readback
 * reads zero whatever is written). The engine gets its level from this
 * saturated multiply instead: x8 (+18 dB) puts speech near the reference
 * while the clamp keeps transients inside int16. Runs on the live capture
 * buffer, I2S source only - the EMU test source skips this branch, so the
 * emulator's detect input is untouched. */
void llgo_ac_chunk_gain(int16_t *pcm, int32_t n)
{
    for (int32_t i = 0; i < n; i++) {
        int32_t v = (int32_t)pcm[i] * 8;
        if (v > 32767) {
            v = 32767;
        } else if (v < -32768) {
            v = -32768;
        }
        pcm[i] = (int16_t)v;
    }
}

/* The wakenet glue's MCLK pin hook (see post-sync.sh; Go sets it through
 * llgo_ac_mclk before wakenet.Init in HW mode). */
int32_t llgo_audiocheck_mclk = -1;

void llgo_ac_mclk(int32_t pin)
{
    llgo_audiocheck_mclk = pin;
}

static int64_t isqrt64(int64_t x)
{
    int64_t r;
    if (x <= 0)
        return 0;
    r = x;
    while (r * r > x)
        r = (r + x / r) / 2;
    return r;
}

static struct {
    int32_t open_num;        /* speech when rms * open_den >= floor * open_num */
    int32_t open_den;
    int32_t open_frames;     /* consecutive speech frames to open  */
    int32_t hangover_frames; /* consecutive quiet frames to close  */
    int32_t floor_min;       /* floor clamp window (amplitude domain) */
    int32_t floor_max;
    int32_t floor;
    int32_t open_run;
    int32_t quiet_run;
    int32_t in_utt;
} vad;

void llgo_ac_vad_cfg(int32_t open_num, int32_t open_den, int32_t open_frames,
                     int32_t hangover_frames, int32_t floor_min, int32_t floor_max)
{
    vad.open_num = open_num;
    vad.open_den = open_den;
    vad.open_frames = open_frames;
    vad.hangover_frames = hangover_frames;
    vad.floor_min = floor_min;
    vad.floor_max = floor_max;
    vad.floor = floor_min;
    vad.open_run = 0;
    vad.quiet_run = 0;
    vad.in_utt = 0;
}

void llgo_ac_vad_reset(void)
{
    vad.open_run = 0;
    vad.quiet_run = 0;
    vad.in_utt = 0;
    /* the floor survives: re-arming mid-noise must not drop it to zero */
}

/* Feed one 30 ms frame (n mono samples). Returns 0 = nothing, 1 = the
 * utterance just opened, 2 = it just closed (hangover elapsed). rms and
 * floor always carry this frame's measured values for the log lines.
 * Phase B opens on 1 and records a fixed window, so 2 never arrives there. */
int32_t llgo_ac_vad_feed(const int16_t *pcm, int32_t n, int32_t *rms, int32_t *floor)
{
    int64_t sumsq = 0;
    int32_t i, r;

    for (i = 0; i < n; i++) {
        int32_t v = pcm[i];
        sumsq += (int64_t)v * v;
    }
    r = (int32_t)isqrt64(sumsq / n);
    *rms = r;
    *floor = vad.floor;

    if (!vad.in_utt) {
        vad.floor = (vad.floor * 63 + r) / 64;
        if (vad.floor < vad.floor_min)
            vad.floor = vad.floor_min;
        if (vad.floor > vad.floor_max)
            vad.floor = vad.floor_max;
        if ((int64_t)r * vad.open_den >= (int64_t)vad.floor * vad.open_num) {
            if (++vad.open_run >= vad.open_frames) {
                vad.in_utt = 1;
                vad.quiet_run = 0;
                return 1;
            }
        } else {
            vad.open_run = 0;
        }
    } else {
        if ((int64_t)r <= (int64_t)vad.floor) {
            if (++vad.quiet_run >= vad.hangover_frames) {
                vad.in_utt = 0;
                vad.open_run = 0;
                return 2;
            }
        } else {
            vad.quiet_run = 0;
        }
    }
    return 0;
}
