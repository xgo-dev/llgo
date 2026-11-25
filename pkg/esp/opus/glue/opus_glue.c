/* Opus codec glue - the C half of pkg/esp/opus.
 *
 * This file is a TEMPLATE, copied into the project's main component next to
 * the synced glues of the other pkg/esp packages (integration way 1,
 * pkg/esp/README.md). The firmware must register the codec dependency
 * (managed component 78/esp-opus, vanilla upstream Opus compiled
 * fixed-point: -DFIXED_POINT=1 -DDISABLE_FLOAT_API) so opus.h and the
 * library land in the link.
 *
 * # Shape
 *
 * Unlike the ble/bt glues there is no mailbox, ring or task here: Opus is a
 * synchronous library and every entry point below runs to completion on the
 * calling goroutine's own (FreeRTOS) task stack. That is also the memory
 * discipline: the library's malloc/free calls - the codec states at open,
 * per-frame scratch at encode/decode - happen on the Go scheduler's task,
 * the one context the firmware's --wrap=malloc (tinygogc) serves. None of
 * these entry points may be called from a callback or an interrupt.
 *
 * # Handles
 *
 * The codec states are C objects owned by this glue's static tables (Go
 * never sees a pointer): two encoder slots, two decoder slots. A handle
 * packs a per-slot generation and the slot number; a handle from a closed
 * or recycled codec answers ESP_ERR_INVALID_STATE instead of acting on
 * whatever now lives in the slot.
 *
 * # Error codes
 *
 * Return 0 on success. Negative returns are the Opus errors passed through
 * unchanged (the Go side keeps the sign, like the tls package's negative
 * mbedtls codes); positive returns are esp_err codes for conditions this
 * glue owns.
 *
 * Outside an ESP-IDF build the whole body compiles away and the entry
 * points report ESP_ERR_NOT_SUPPORTED, so a firmware that skipped the
 * component stays honest instead of linking dead code.
 */

#if defined(ESP_PLATFORM)

#include <stdint.h>

#include "esp_err.h"
#include "opus.h"

#define OPUS_ENC_SLOTS 2
#define OPUS_DEC_SLOTS 2

static OpusEncoder *s_enc[OPUS_ENC_SLOTS];
static OpusDecoder *s_dec[OPUS_DEC_SLOTS];
static uint32_t s_enc_gen[OPUS_ENC_SLOTS];
static uint32_t s_dec_gen[OPUS_DEC_SLOTS];

/* Pack (generation, slot) into the handle Go carries around. The slot byte
 * is 1-based so 0 stays an invalid handle. */
static int32_t opus_make_handle(uint32_t *gen, int slot)
{
    (*gen)++;
    return (int32_t)(((*gen) << 8) | (uint32_t)(slot + 1));
}

/* Decode a handle into its state, or NULL when the handle is stale: a slot
 * that has been closed (and not refilled by a codec of a new generation),
 * a slot byte out of range, or a generation mismatch. */
static OpusEncoder *opus_enc_of(int32_t h)
{
    uint32_t u = (uint32_t)h;
    int slot = (int)(u & 0xFF) - 1;
    if (slot < 0 || slot >= OPUS_ENC_SLOTS || (u >> 8) != s_enc_gen[slot]) {
        return NULL;
    }
    return s_enc[slot];
}

static OpusDecoder *opus_dec_of(int32_t h)
{
    uint32_t u = (uint32_t)h;
    int slot = (int)(u & 0xFF) - 1;
    if (slot < 0 || slot >= OPUS_DEC_SLOTS || (u >> 8) != s_dec_gen[slot]) {
        return NULL;
    }
    return s_dec[slot];
}

/* ---- encoder ---- */

int32_t llgo_opus_enc_open(int32_t sample_rate, int32_t channels,
                           int32_t application, int32_t *out)
{
    int slot = -1;
    for (int i = 0; i < OPUS_ENC_SLOTS; i++) {
        if (s_enc[i] == NULL) {
            slot = i;
            break;
        }
    }
    if (slot < 0) {
        return ESP_ERR_NO_MEM; /* both encoder slots are in use */
    }
    int opus_err = OPUS_OK;
    OpusEncoder *st = opus_encoder_create((opus_int32)sample_rate,
                                          (int)channels, (int)application,
                                          &opus_err);
    if (st == NULL || opus_err != OPUS_OK) {
        return (st != NULL) ? (int32_t)opus_err : OPUS_ALLOC_FAIL;
    }
    s_enc[slot] = st;
    *out = opus_make_handle(&s_enc_gen[slot], slot);
    return ESP_OK;
}

int32_t llgo_opus_enc_encode(int32_t h, const opus_int16 *pcm,
                             int32_t frame_size, uint8_t *out,
                             int32_t cap, int32_t *out_len)
{
    OpusEncoder *st = opus_enc_of(h);
    if (st == NULL) {
        return ESP_ERR_INVALID_STATE;
    }
    if (pcm == NULL || out == NULL || out_len == NULL ||
        frame_size <= 0 || cap <= 0) {
        return OPUS_BAD_ARG;
    }
    opus_int32 n = opus_encode(st, pcm, (int)frame_size, out,
                               (opus_int32)cap);
    if (n < 0) {
        return (int32_t)n;
    }
    *out_len = (int32_t)n;
    return ESP_OK;
}

int32_t llgo_opus_enc_set_bitrate(int32_t h, int32_t bps)
{
    OpusEncoder *st = opus_enc_of(h);
    if (st == NULL) {
        return ESP_ERR_INVALID_STATE;
    }
    return (int32_t)opus_encoder_ctl(st, OPUS_SET_BITRATE_REQUEST, (opus_int32)bps);
}

int32_t llgo_opus_enc_set_complexity(int32_t h, int32_t c)
{
    OpusEncoder *st = opus_enc_of(h);
    if (st == NULL) {
        return ESP_ERR_INVALID_STATE;
    }
    return (int32_t)opus_encoder_ctl(st, OPUS_SET_COMPLEXITY_REQUEST, (int)c);
}

/* The encoder's algorithmic delay, in samples per channel (OPUS_GET_LOOKAHEAD):
 * the decoded stream is shifted by this many samples relative to the encode
 * input, so a roundtrip comparison must align by it. */
int32_t llgo_opus_enc_lookahead(int32_t h)
{
    OpusEncoder *st = opus_enc_of(h);
    if (st == NULL) {
        return ESP_ERR_INVALID_STATE;
    }
    opus_int32 v = 0;
    int rc = opus_encoder_ctl(st, OPUS_GET_LOOKAHEAD_REQUEST, &v);
    if (rc != OPUS_OK) {
        return (int32_t)rc;
    }
    return (int32_t)v;
}

int32_t llgo_opus_enc_close(int32_t h)
{
    OpusEncoder *st = opus_enc_of(h);
    if (st == NULL) {
        return ESP_ERR_INVALID_STATE;
    }
    uint32_t u = (uint32_t)h;
    int slot = (int)(u & 0xFF) - 1;
    opus_encoder_destroy(st);
    s_enc[slot] = NULL;
    s_enc_gen[slot]++; /* the next open at this slot gets a new generation */
    return ESP_OK;
}

/* opus_encoder_get_size: the state size a firmware's memory ledger prints.
 * The Go layer never allocates states itself; this is reporting only. */
int32_t llgo_opus_enc_size(int32_t channels)
{
    return (int32_t)opus_encoder_get_size((int)channels);
}

/* ---- decoder ---- */

int32_t llgo_opus_dec_open(int32_t sample_rate, int32_t channels, int32_t *out)
{
    int slot = -1;
    for (int i = 0; i < OPUS_DEC_SLOTS; i++) {
        if (s_dec[i] == NULL) {
            slot = i;
            break;
        }
    }
    if (slot < 0) {
        return ESP_ERR_NO_MEM; /* both decoder slots are in use */
    }
    int opus_err = OPUS_OK;
    OpusDecoder *st = opus_decoder_create((opus_int32)sample_rate,
                                          (int)channels, &opus_err);
    if (st == NULL || opus_err != OPUS_OK) {
        return (st != NULL) ? (int32_t)opus_err : OPUS_ALLOC_FAIL;
    }
    s_dec[slot] = st;
    *out = opus_make_handle(&s_dec_gen[slot], slot);
    return ESP_OK;
}

int32_t llgo_opus_dec_decode(int32_t h, const uint8_t *data, int32_t len,
                             opus_int16 *pcm, int32_t frame_cap,
                             int32_t *out_samples)
{
    OpusDecoder *st = opus_dec_of(h);
    if (st == NULL) {
        return ESP_ERR_INVALID_STATE;
    }
    if (pcm == NULL || out_samples == NULL || frame_cap <= 0) {
        return OPUS_BAD_ARG;
    }
    /* data == NULL, len == 0 is Opus' packet-loss concealment request: the
     * decoder invents the continuation of the stream. */
    int n = opus_decode(st, data, (opus_int32)len, pcm, (int)frame_cap, 0);
    if (n < 0) {
        return (int32_t)n;
    }
    *out_samples = (int32_t)n;
    return ESP_OK;
}

int32_t llgo_opus_dec_close(int32_t h)
{
    OpusDecoder *st = opus_dec_of(h);
    if (st == NULL) {
        return ESP_ERR_INVALID_STATE;
    }
    uint32_t u = (uint32_t)h;
    int slot = (int)(u & 0xFF) - 1;
    opus_decoder_destroy(st);
    s_dec[slot] = NULL;
    s_dec_gen[slot]++;
    return ESP_OK;
}

int32_t llgo_opus_dec_size(int32_t channels)
{
    return (int32_t)opus_decoder_get_size((int)channels);
}

#else /* !ESP_PLATFORM: honest stubs, the firmware has no codec here */

#include <stdint.h>

#include "esp_err.h"

int32_t llgo_opus_enc_open(int32_t sample_rate, int32_t channels,
                           int32_t application, int32_t *out)
{
    (void)sample_rate; (void)channels; (void)application; (void)out;
    return ESP_ERR_NOT_SUPPORTED;
}

int32_t llgo_opus_enc_encode(int32_t h, const int16_t *pcm,
                             int32_t frame_size, uint8_t *out,
                             int32_t cap, int32_t *out_len)
{
    (void)h; (void)pcm; (void)frame_size; (void)out; (void)cap; (void)out_len;
    return ESP_ERR_NOT_SUPPORTED;
}

int32_t llgo_opus_enc_set_bitrate(int32_t h, int32_t bps)
{
    (void)h; (void)bps;
    return ESP_ERR_NOT_SUPPORTED;
}

int32_t llgo_opus_enc_set_complexity(int32_t h, int32_t c)
{
    (void)h; (void)c;
    return ESP_ERR_NOT_SUPPORTED;
}

int32_t llgo_opus_enc_close(int32_t h)
{
    (void)h;
    return ESP_ERR_NOT_SUPPORTED;
}

int32_t llgo_opus_enc_size(int32_t channels)
{
    (void)channels;
    return 0;
}

int32_t llgo_opus_dec_open(int32_t sample_rate, int32_t channels, int32_t *out)
{
    (void)sample_rate; (void)channels; (void)out;
    return ESP_ERR_NOT_SUPPORTED;
}

int32_t llgo_opus_dec_decode(int32_t h, const uint8_t *data, int32_t len,
                             int16_t *pcm, int32_t frame_cap,
                             int32_t *out_samples)
{
    (void)h; (void)data; (void)len; (void)pcm; (void)frame_cap; (void)out_samples;
    return ESP_ERR_NOT_SUPPORTED;
}

int32_t llgo_opus_dec_close(int32_t h)
{
    (void)h;
    return ESP_ERR_NOT_SUPPORTED;
}

int32_t llgo_opus_dec_size(int32_t channels)
{
    (void)channels;
    return 0;
}

#endif
