/* Demo-local C glue for the voice loop (mic → VAD → Opus → speaker).
 *
 * Three things live here because the C side is where they belong:
 *
 *   - llgo_vloop_now_us: the esp_timer domain clock, for the encode /
 *     decode / playback wall-time lines (the loop's realtime budget is
 *     the demo's whole point).
 *   - llgo_vloop_heap: the IDF pool ledger (same shape as the opus
 *     probe's), printed at boot and after the demo arms its buffers -
 *     the utterance buffer is the firmware's biggest allocation and the
 *     ledger lines make its PSRAM landing visible.
 *   - the energy VAD: a 30 ms-frame RMS follower with an adaptive noise
 *     floor, hysteresis and a hangover tail - all integer math, no
 *     esp-sr/AFE dependency (that stack is far too heavy for this
 *     firmware). The parameters travel from Go (the demo owns the
 *     tuning) and the state machine stays here so the capture loop only
 *     sees three verdicts: nothing / utterance opened / utterance
 *     closed.
 *
 * VAD shape: while idle, the floor follows the frame RMS with a slow IIR
 * (1/64 per frame) clamped to a sane window, and speech means
 * rms >= floor * openNum/openDen (default 2 - "6 dB above the floor").
 * openFrames consecutive speech frames open the utterance. While open,
 * the floor freezes (speech would inflate it) and the frame counts as
 * quiet when rms <= floor - classic hysteresis, so the utterance does
 * not flap on syllable gaps. hangoverFrames consecutive quiet frames
 * close it (the tail stays in the buffer, which is the "尾切" the
 * hangover is for).
 *
 * The Go<->C boundary carries integers and pointers only; the VAD keeps
 * no reference to Go memory beyond the call.
 */
#include <stdint.h>
#include <stdio.h>

#include "esp_heap_caps.h"
#include "esp_timer.h"

int64_t llgo_vloop_now_us(void)
{
    return esp_timer_get_time();
}

/* Print the two IDF pools at one demo stage (tag consumed immediately). */
void llgo_vloop_heap(const char *tag, int32_t len)
{
    printf("[c] vloop heap %.*s internal=%u psram=%u\n",
           (int)len, tag,
           (unsigned)heap_caps_get_free_size(MALLOC_CAP_INTERNAL),
           (unsigned)heap_caps_get_free_size(MALLOC_CAP_SPIRAM));
    fflush(stdout);
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

void llgo_vloop_vad_cfg(int32_t open_num, int32_t open_den, int32_t open_frames,
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

void llgo_vloop_vad_reset(void)
{
    vad.open_run = 0;
    vad.quiet_run = 0;
    vad.in_utt = 0;
    /* the floor survives: re-arming mid-noise must not drop it to zero */
}

/* Feed one 30 ms frame (n mono samples). Returns 0 = nothing, 1 = the
 * utterance just opened, 2 = it just closed (hangover elapsed). rms and
 * floor always carry this frame's measured values for the log lines. */
int32_t llgo_vloop_vad_feed(const int16_t *pcm, int32_t n, int32_t *rms, int32_t *floor)
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
