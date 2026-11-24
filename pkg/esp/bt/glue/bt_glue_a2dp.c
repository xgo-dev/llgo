/* A2DP sink audio profile glue - the sister file of bt_glue_bluedroid.c.
 *
 * This file is a TEMPLATE, copied into the project's main component next to
 * bt_glue_bluedroid.c (integration way 1, pkg/esp/README.md). It shares the
 * sibling file's ctrl ring, lifecycle hooks and boundary rules; it adds the
 * A2DP sink profile and the analog audio output path.
 *
 * # Where the audio goes
 *
 * The Bluedroid stack decodes SBC itself (internal codec path) and hands the
 * glue DECODED PCM (S16 LE, mono or stereo per the negotiated config) on the
 * BTC task through esp_a2d_sink_register_data_callback. The realtime path
 * below never touches Go: BTC callback -> static PCM ring -> dedicated pump
 * task -> i2s_channel_write. Go only sees control-plane events (connected,
 * codec config, volume) and three counters.
 *
 * # Realtime discipline
 *
 * The data callback runs on the BTC task, which also services every other
 * classic-BT event (SPP included), so it must never block. i2s_channel_write
 * parks the caller whenever the TX DMA is full, so the callback only copies
 * into the static PCM ring (SPSC, BTC produces / pump consumes; on overflow
 * the whole chunk is dropped and counted) and hands the pump task a task
 * notification. The pump task owns every blocking call: it drains the ring
 * into I2S at the I2S's own consumption rate and performs the clock/slot
 * reconfiguration on codec changes (reconfig requires a disabled channel, so
 * it is sequenced there, off the BTC task). Losing audio beats blocking BTC:
 * loss is visible as the dropped counter, not as a stalled BT stack.
 *
 * Ring sizing: 32 KiB ≈ 186 ms of 44.1 kHz stereo S16 - deeper than every
 * legitimate scheduling hiccup on the pump path, small enough for internal
 * DRAM. The I2S DMA buffering (dma_desc_num x dma_frame_num frames) rides on
 * top; the pair absorbs the BT/I2S clock-drift mechanics.
 *
 * # Events (kinds 9..12 of the shared ctrl ring)
 *
 *   9 EvA2DPConnected    peer connected (handle, addr)
 *  10 EvA2DPDisconnected peer disconnected (Status = disc_rsn, addr)
 *  11 EvA2DPAudioCfg     codec config (Aux = sample rate Hz, Status = channel
 *                        count, handle); the glue re-clocks I2S accordingly
 *  12 EvA2DPVolume       volume 0..127 (Aux), from the phone's CT command or
 *                        the response to our own set command
 *
 * Outside an ESP-IDF build the whole body compiles away. Without
 * CONFIG_BT_A2DP_ENABLE the llgo_bt_a2dp_* symbols return
 * ESP_ERR_NOT_SUPPORTED so a firmware that skipped the profile stays honest
 * instead of linking dead code.
 */

#if defined(ESP_PLATFORM)

#include <stddef.h>
#include <stdint.h>
#include <string.h>

#include "esp_a2dp_api.h"
#include "esp_avrc_api.h"
#include "driver/i2s_std.h"
#include "hal/gpio_types.h"
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "freertos/semphr.h"

/* Shared with bt_glue_bluedroid.c (same main component): the ctrl ring's
 * publisher and the lifecycle ready flag. The sibling declares them
 * non-static for exactly this pair of files. */
extern void ctrl_push(int32_t kind, int32_t status, uint32_t handle,
                      int32_t aux, const uint8_t *addr);
extern volatile int32_t s_ready;

int32_t llgo_bt_a2dp_disable(void); /* used by bt_a2dp_teardown below */

/* The sibling file's #error guard covers the bluedroid/classic/SPP basics;
 * this file only adds the A2DP-optional part. */

/* ---- PCM ring (BTC producer, pump consumer) ---- */

#define A2D_PCM_RING_SIZE 32768

static uint8_t s_pcm_ring[A2D_PCM_RING_SIZE];
static volatile uint32_t s_pcm_head; /* producer side: BTC task only */
static volatile uint32_t s_pcm_tail; /* consumer side: pump task only */

/* Counters, all single-writer (BTC or pump): plain increments here, and
 * llgo_bt_a2dp_stats reads them with acquire loads. No RMW atomics anywhere
 * - the RV32 build has no A extension. */
static volatile uint32_t s_pcm_total;   /* PCM bytes handed over by the stack */
static volatile uint32_t s_pcm_dropped; /* PCM bytes lost: ring was full */
static volatile uint32_t s_i2s_written; /* PCM bytes the pump fed to I2S */

/* ---- pump task ---- */

/* s_pump_task has exactly two writers: enable (create) and the teardown
 * path AFTER a successful handshake take (clear). pcm_push reads it only to
 * notify a task that is known-running (s_a2dp_on true implies the handshake
 * completed), so a give never lands on a freed TCB. */
static TaskHandle_t s_pump_task;
static volatile int32_t s_pump_quit;
static StaticSemaphore_t s_pump_dead_buf;
static SemaphoreHandle_t s_pump_dead;

/* Wait for the pump to acknowledge its exit. The pump's own exit is the
 * only giver, and enable drains any stale give at create time, so one
 * bounded take is the whole handshake: pdTRUE = the pump exited; a timeout
 * means it is stuck in a write that outlived every bound - the caller MUST
 * keep the channel and the pump alive (deleting either would UAF the write
 * in flight, and a later enable would revive the pump as a second ring
 * consumer). */
static bool pump_wait_dead(uint32_t timeout_ms)
{
    return xSemaphoreTake(s_pump_dead, pdMS_TO_TICKS(timeout_ms)) == pdTRUE;
}

/* Pending codec config, packed as ONE word the BTC task writes in a single
 * store (rate in the high 24 bits, channels in the low 8): the pump reads it
 * once per loop, so it can never tear a new rate together with an old
 * channel count. Zero = nothing pending. */
#define A2D_CFG_WORD(rate, ch) (((uint32_t)(rate) << 8) | (uint32_t)(ch))
static volatile uint32_t s_pending_cfg;
static int32_t s_i2s_rate; /* clock the channel runs at now (pump-owned) */
static int32_t s_i2s_ch;   /* slot mode the channel runs at now (pump-owned) */

static i2s_chan_handle_t s_tx;
static volatile int32_t s_a2dp_on;
static volatile esp_a2d_conn_hdl_t s_conn; /* latest A2DP connection */
static volatile int32_t s_conn_valid;
/* AVRCP transaction label: 4 bits per the spec, and v6.3's implementation
 * passes the raw byte through to BTA unchecked (verified against
 * btc_avrc_ct_send_set_absolute_volume_cmd), so the wrap is ours to do. */
static uint8_t s_vol_tl; /* BTC task only */

/* PCM chunk: one contiguous span of the ring (wraps are two spans). */
static int32_t pcm_pop(uint8_t **dst, int32_t *len)
{
    uint32_t head = __atomic_load_n(&s_pcm_head, __ATOMIC_ACQUIRE);
    uint32_t tail = s_pcm_tail;
    if (tail == head) {
        return 0;
    }
    *dst = &s_pcm_ring[tail];
    *len = (head > tail) ? (int32_t)(head - tail)
                         : (int32_t)(A2D_PCM_RING_SIZE - tail);
    return 1;
}

static void pcm_advance(int32_t len)
{
    /* RELEASE: pairs with the producer's acquire read of the tail, so the
     * space this declares free cannot become visible before the copy it
     * covers is done - the tail direction of the SPSC pair. */
    __atomic_store_n(&s_pcm_tail,
                     (s_pcm_tail + (uint32_t)len) % A2D_PCM_RING_SIZE,
                     __ATOMIC_RELEASE);
}

/* Push one callback chunk. BTC task context: never blocks, drops on full.
 * The s_a2dp_on guard closes the disable race: esp_a2d_sink_deinit only
 * POSTS to BTC, so a callback can still be in flight (or already running)
 * after llgo_bt_a2dp_disable started tearing the path down - the guard
 * turns such a straggler into a plain drop instead of a store into a ring
 * the teardown is resetting. */
static void pcm_push(const uint8_t *buf, uint32_t len)
{
    if (!s_a2dp_on) {
        return;
    }
    s_pcm_total += len;
    uint32_t tail = __atomic_load_n(&s_pcm_tail, __ATOMIC_ACQUIRE);
    uint32_t head = s_pcm_head;
    uint32_t free_bytes = (tail > head)
                              ? (tail - head - 1)
                              : (A2D_PCM_RING_SIZE - head + tail - 1);
    if (len > free_bytes) {
        s_pcm_dropped += len;
        return; /* drop the whole chunk: a partial block is silence anyway */
    }
    uint32_t first = A2D_PCM_RING_SIZE - head;
    if (first > len) {
        first = len;
    }
    memcpy(&s_pcm_ring[head], buf, first);
    if (len > first) {
        memcpy(&s_pcm_ring[0], buf + first, len - first);
    }
    __atomic_store_n(&s_pcm_head, (head + len) % A2D_PCM_RING_SIZE,
                     __ATOMIC_RELEASE);
    if (s_pump_task != NULL) {
        xTaskNotifyGive(s_pump_task); /* wake the pump */
    }
}

/* The analog-side consumer: blocking i2s_channel_write lives here, nowhere
 * near the BTC task. Higher-than-idle but below BT tasks' priority. The
 * write bound (ms, see pdMS_TO_TICKS at the call) is strictly shorter than
 * the teardown handshake (A2D_PUMP_HANDSHAKE_MS below the control plane):
 * a stalled channel releases the pump before the handshake could give up. */
#define A2D_PUMP_STACK        4096
#define A2D_PUMP_PRIO         5
#define A2D_WRITE_TIMEOUT_MS  1500
#define A2D_PUMP_HANDSHAKE_MS 5000

static void pump_task(void *arg)
{
    (void)arg;
    for (;;) {
        if (s_pump_quit) {
            break;
        }
        /* Apply a pending codec config: I2S reconfig needs the channel
         * disabled, and the new clock must be up before the stack feeds
         * more PCM (the stack re-encodes nothing: PCM follows the config).
         * One atomic read of the packed word; the gate fires on a rate OR a
         * channel-count change (a 44.1 kHz mono source after a stereo one
         * must re-slot, or mono PCM plays at double speed on the left
         * channel only). */
        uint32_t pending = s_pending_cfg;
        int32_t pending_rate = (int32_t)(pending >> 8);
        int32_t pending_ch = (int32_t)(pending & 0xFF);
        if (pending != 0 && (pending_rate != s_i2s_rate ||
                             pending_ch != s_i2s_ch)) {
            i2s_channel_disable(s_tx);
            i2s_std_clk_config_t clk = I2S_STD_CLK_DEFAULT_CONFIG(
                (uint32_t)pending_rate);
            i2s_std_slot_config_t slot =
                I2S_STD_PHILIPS_SLOT_DEFAULT_CONFIG(I2S_DATA_BIT_WIDTH_16BIT,
                                                    (pending_ch == 1)
                                                        ? I2S_SLOT_MODE_MONO
                                                        : I2S_SLOT_MODE_STEREO);
            i2s_channel_reconfig_std_clock(s_tx, &clk);
            i2s_channel_reconfig_std_slot(s_tx, &slot);
            i2s_channel_enable(s_tx);
            s_i2s_rate = pending_rate;
            s_i2s_ch = pending_ch;
            s_pending_cfg = 0; /* consumed; BTC writes the next one whole */
        }
        uint8_t *chunk;
        int32_t len;
        if (!pcm_pop(&chunk, &len)) {
            /* Ring empty: audio is suspended or the peer paused. Park until
             * the next callback notification or 20 ms, whichever first. */
            ulTaskNotifyTake(pdTRUE, pdMS_TO_TICKS(20));
            continue;
        }
        size_t written = 0;
        /* Bounded write in MILLISECONDS (pdMS_TO_TICKS): strictly shorter
         * than the 2 s disable handshake, so a stalled channel releases the
         * pump before the handshake could ever give up on it. A stall drops
         * the chunk - losing audio beats wedging the teardown. */
        if (i2s_channel_write(s_tx, chunk, (size_t)len, &written,
                              pdMS_TO_TICKS(A2D_WRITE_TIMEOUT_MS)) == ESP_OK) {
            /* 32-bit writes are atomic on the target; the counter is
             * single-writer (this task), the reader only ever loads. */
            s_i2s_written += (uint32_t)written;
            pcm_advance(len);
        } else if (!s_pump_quit) {
            /* A failed write means channel trouble, not a full ring: drop
             * the chunk so the loop makes progress. The loss shows up as a
             * written-vs-total gap; `dropped` stays the BTC side's
             * ring-full counter (its single-writer contract). */
            pcm_advance(len);
        }
    }
    /* Do NOT clear s_pump_task here: the teardown path clears it after its
     * handshake take succeeds (single-writer rule), and a straggler notify
     * between give and take would then miss a freed-TCB write. */
    xSemaphoreGive(s_pump_dead);
    vTaskDelete(NULL);
}

/* ---- Bluedroid callbacks (all BTC task context) ---- */

static void a2d_data_cb(const uint8_t *buf, uint32_t len)
{
    pcm_push(buf, len);
}

static void a2d_cb(esp_a2d_cb_event_t event, esp_a2d_cb_param_t *param)
{
    switch (event) {
    case ESP_A2D_CONNECTION_STATE_EVT:
        if (param->conn_stat.state == ESP_A2D_CONNECTION_STATE_CONNECTED) {
            s_conn = param->conn_stat.conn_hdl;
            s_conn_valid = 1;
            ctrl_push(9, 0, (uint32_t)param->conn_stat.conn_hdl, 0,
                      param->conn_stat.remote_bda);
        } else if (param->conn_stat.state ==
                   ESP_A2D_CONNECTION_STATE_DISCONNECTED) {
            s_conn_valid = 0;
            ctrl_push(10, (int32_t)param->conn_stat.disc_rsn,
                      (uint32_t)param->conn_stat.conn_hdl, 0,
                      param->conn_stat.remote_bda);
        }
        break;
    case ESP_A2D_AUDIO_CFG_EVT: {
        /* Only an SBC config is an SBC CIE: for any other media codec the
         * union bytes would be misread as samp_freq/ch_mode, so report a
         * zero config (a subscriber sees rate=0) instead of re-clocking
         * I2S on garbage. */
        if (param->audio_cfg.mcc.type != ESP_A2D_MCT_SBC) {
            ctrl_push(11, 0, (uint32_t)param->audio_cfg.conn_hdl, 0,
                      param->audio_cfg.remote_bda);
            break;
        }
        const esp_a2d_cie_sbc_t *cie = &param->audio_cfg.mcc.cie.sbc_info;
        int32_t rate = 0;
        switch (cie->samp_freq) {
        case ESP_A2D_SBC_CIE_SF_16K:
            rate = 16000;
            break;
        case ESP_A2D_SBC_CIE_SF_32K:
            rate = 32000;
            break;
        case ESP_A2D_SBC_CIE_SF_44K:
            rate = 44100;
            break;
        case ESP_A2D_SBC_CIE_SF_48K:
            rate = 48000;
            break;
        default:
            rate = 0;
            break;
        }
        int32_t chans = 0;
        if (cie->ch_mode == ESP_A2D_SBC_CIE_CH_MODE_MONO) {
            chans = 1;
        } else if (cie->ch_mode & (ESP_A2D_SBC_CIE_CH_MODE_STEREO |
                                   ESP_A2D_SBC_CIE_CH_MODE_JOINT_STEREO |
                                   ESP_A2D_SBC_CIE_CH_MODE_DUAL_CHANNEL)) {
            chans = 2;
        }
        if (rate != 0 && chans != 0) {
            s_pending_cfg = A2D_CFG_WORD(rate, chans);
        }
        /* Aux = rate, Status = channels: the probe and a control-plane
         * consumer get the negotiated format without a second query (A2DP
         * has no get-config API - the event is the only source). */
        ctrl_push(11, chans, (uint32_t)param->audio_cfg.conn_hdl, rate,
                  param->audio_cfg.remote_bda);
        break;
    }
    default:
        break;
    }
}

static void avrc_tg_cb(esp_avrc_tg_cb_event_t event,
                       esp_avrc_tg_cb_param_t *param)
{
    if (event == ESP_AVRC_TG_SET_ABSOLUTE_VOLUME_CMD_EVT) {
        /* The stack answers the CT command itself; this event only tells us
         * (and Go) the phone moved the volume. Aux = 0..127. */
        ctrl_push(12, 0, (uint32_t)param->set_abs_vol.conn_hdl,
                  (int32_t)param->set_abs_vol.volume, NULL);
    }
}

static void avrc_ct_cb(esp_avrc_ct_cb_event_t event,
                       esp_avrc_ct_cb_param_t *param)
{
    if (event == ESP_AVRC_CT_SET_ABSOLUTE_VOLUME_RSP_EVT) {
        /* Our own A2DPSetVolume round trip came back: report the volume the
         * peer confirmed through the same kind as the TG path. */
        ctrl_push(12, 0, (uint32_t)param->set_volume_rsp.conn_hdl,
                  (int32_t)param->set_volume_rsp.volume, NULL);
    }
}

/* Tear the audio path down, whatever state it is in. Called by the sibling
 * glue's bt_teardown (Deinit must not leak a pump task, an I2S channel or a
 * live s_a2dp_on past a stack power-down - the next Init/A2DPEnable would
 * otherwise refuse forever) and by llgo_bt_a2dp_disable's normal path is a
 * superset of this. Returns the first error, best-effort throughout. On a
 * wedged pump it keeps the pump alive and reports failure, exactly like
 * llgo_bt_a2dp_disable. Non-static: shared pair with bt_glue_bluedroid.c. */
int32_t bt_a2dp_teardown(void)
{
#if !CONFIG_BT_A2DP_ENABLE
    return ESP_OK; /* nothing was ever built */
#else
    if (!s_a2dp_on) {
        if (s_tx == NULL && s_pump_task == NULL) {
            return ESP_OK;
        }
        /* Half-set-up state (a failed enable's leftovers): free what is
         * there; no pump handshake needed because enable only creates the
         * pump after the channel and the profile calls succeeded. */
        if (s_pump_task != NULL) {
            s_pump_quit = 1;
            xTaskNotifyGive(s_pump_task);
            if (!pump_wait_dead(A2D_PUMP_HANDSHAKE_MS)) {
                return ESP_ERR_INVALID_STATE;
            }
            s_pump_task = NULL;
        }
        if (s_tx != NULL) {
            i2s_channel_disable(s_tx);
            i2s_del_channel(s_tx);
            s_tx = NULL;
        }
        s_a2dp_on = 0;
        return ESP_OK;
    }
    return llgo_bt_a2dp_disable();
#endif
}

/* ---- control plane (called from Go, one-shot esp_* calls) ---- */

int32_t llgo_bt_a2dp_enable(int32_t bclk, int32_t ws, int32_t dout,
                            int32_t dma_desc, int32_t dma_frames)
{
#if !CONFIG_BT_A2DP_ENABLE
    (void)bclk; (void)ws; (void)dout; (void)dma_desc; (void)dma_frames;
    return ESP_ERR_NOT_SUPPORTED; /* firmware built without the profile */
#else
    if (!s_ready) {
        return ESP_ERR_INVALID_STATE;
    }
    if (s_a2dp_on) {
        return ESP_ERR_INVALID_STATE;
    }
    if (s_pump_task != NULL) {
        /* A pump from a previous enable/disable cycle is still alive: its
         * teardown handshake failed and the disable refused to free anything
         * (see llgo_bt_a2dp_disable). Creating a second one here would make
         * two consumers for the ring - refuse until the stuck one unwedges. */
        return ESP_ERR_INVALID_STATE;
    }
    esp_err_t err;
    /* I2S TX channel. The DMA depth is the caller's choice (0 = driver
     * defaults): it is the jitter absorber between the BT clock that feeds
     * PCM and the local audio clock that consumes it. */
    i2s_chan_config_t chan = I2S_CHANNEL_DEFAULT_CONFIG(I2S_NUM_0,
                                                        I2S_ROLE_MASTER);
    if (dma_desc > 0) {
        chan.dma_desc_num = (uint32_t)dma_desc;
    }
    if (dma_frames > 0) {
        chan.dma_frame_num = (uint32_t)dma_frames;
    }
    err = i2s_new_channel(&chan, &s_tx, NULL);
    if (err != ESP_OK) {
        return err;
    }
    i2s_std_config_t std = {
        .clk_cfg = I2S_STD_CLK_DEFAULT_CONFIG(44100),
        .slot_cfg = I2S_STD_PHILIPS_SLOT_DEFAULT_CONFIG(
            I2S_DATA_BIT_WIDTH_16BIT, I2S_SLOT_MODE_STEREO),
        .gpio_cfg = {
            .mclk = I2S_GPIO_UNUSED,
            .bclk = (gpio_num_t)bclk,
            .ws = (gpio_num_t)ws,
            .dout = (gpio_num_t)dout,
            .din = I2S_GPIO_UNUSED,
        },
    };
    err = i2s_channel_init_std_mode(s_tx, &std);
    if (err != ESP_OK) {
        i2s_del_channel(s_tx);
        s_tx = NULL;
        return err;
    }
    s_i2s_rate = 44100;
    s_i2s_ch = 2;
    s_pending_cfg = 0;
    /* The channel runs from now (silence until a stream starts): the pump
     * would otherwise see the codec-config event as its only enable
     * trigger, and a source that reuses the negotiated defaults without a
     * fresh AUDIO_CFG (or a config equal to the power-on default) would
     * write into a disabled channel forever. */
    err = i2s_channel_enable(s_tx);
    if (err != ESP_OK) {
        i2s_del_channel(s_tx);
        s_tx = NULL;
        return err;
    }
    if (s_pump_dead == NULL) {
        s_pump_dead = xSemaphoreCreateBinaryStatic(&s_pump_dead_buf);
    } else {
        xSemaphoreTake(s_pump_dead, 0);
    }
    s_pump_quit = 0;
    if (xTaskCreate(pump_task, "llgo-a2dp", A2D_PUMP_STACK, NULL,
                    A2D_PUMP_PRIO, &s_pump_task) != pdPASS) {
        i2s_del_channel(s_tx);
        s_tx = NULL;
        return ESP_ERR_NO_MEM;
    }
    err = esp_a2d_register_callback(a2d_cb);
    if (err == ESP_OK) {
        err = esp_a2d_sink_register_data_callback(a2d_data_cb);
    }
    if (err == ESP_OK) {
        err = esp_avrc_tg_register_callback(avrc_tg_cb);
    }
    if (err == ESP_OK) {
        err = esp_avrc_ct_register_callback(avrc_ct_cb);
    }
    if (err == ESP_OK) {
        err = esp_a2d_sink_init();
    }
    if (err == ESP_OK) {
        err = esp_avrc_tg_init();
    }
    if (err == ESP_OK) {
        err = esp_avrc_ct_init();
    }
    if (err != ESP_OK) {
        /* Roll the profile back out. The pump has written nothing yet (the
         * ring is empty, the channel is enabled but idle), so a stuck
         * handshake here means a wedged driver - keep the pump and channel
         * alive (deleting under a live pump would UAF its write) and fail
         * the call; the stuck pair blocks any later enable via the
         * s_pump_task guard at the top. */
        s_pump_quit = 1;
        xTaskNotifyGive(s_pump_task);
        if (!pump_wait_dead(A2D_PUMP_HANDSHAKE_MS)) {
            return err;
        }
        s_pump_task = NULL;
        i2s_channel_disable(s_tx);
        i2s_del_channel(s_tx);
        s_tx = NULL;
        return err;
    }
    s_a2dp_on = 1;
    s_conn_valid = 0;
    s_pcm_head = s_pcm_tail = 0;
    return ESP_OK;
#endif
}

int32_t llgo_bt_a2dp_disable(void)
{
#if !CONFIG_BT_A2DP_ENABLE
    return ESP_ERR_NOT_SUPPORTED;
#else
    if (!s_a2dp_on) {
        return ESP_ERR_INVALID_STATE;
    }
    esp_err_t err;
    /* Profile down first (BTC stops *scheduling* new PCM chunks; a callback
     * already in flight or queued behind the deinit is caught by the
     * s_a2dp_on guard in pcm_push), then the pump, then the channel. */
    err = esp_a2d_sink_deinit();
    esp_avrc_tg_deinit();
    esp_avrc_ct_deinit();
    s_a2dp_on = 0;
    s_conn_valid = 0;
    s_pump_quit = 1;
    if (s_pump_task != NULL) {
        xTaskNotifyGive(s_pump_task);
        if (!pump_wait_dead(A2D_PUMP_HANDSHAKE_MS)) {
            /* The pump did not exit within its own bound times the
             * handshake margin: it is wedged in a stalled write. Keep the
             * channel and the pump alive (deleting either would UAF the
             * write) and report the failure - A2DPDisable stays
             * retryable, and llgo_bt_a2dp_enable refuses to run a second
             * pump while this one lives. */
            s_pump_quit = 0;
            return err == ESP_OK ? ESP_ERR_INVALID_STATE : err;
        }
        s_pump_task = NULL; /* handshake done: this side owns the handle */
    }
    if (s_tx != NULL) {
        i2s_channel_disable(s_tx);
        i2s_del_channel(s_tx);
        s_tx = NULL;
    }
    /* Counter reset races a straggler BTC `+=` (the deinit POST may not
     * have run yet): a lost increment shifts the telemetry by one chunk at
     * worst - acceptable and documented at A2DPStats, not worth a BTC-side
     * generation counter. */
    s_pcm_head = s_pcm_tail = 0;
    s_pcm_total = 0;
    s_pcm_dropped = 0;
    s_i2s_written = 0;
    s_pending_cfg = 0;
    s_i2s_rate = 44100;
    s_i2s_ch = 2;
    return err;
#endif
}

int32_t llgo_bt_a2dp_volume_set(int32_t volume)
{
#if !CONFIG_BT_A2DP_ENABLE
    (void)volume;
    return ESP_ERR_NOT_SUPPORTED;
#else
    if (!s_a2dp_on || !s_conn_valid) {
        return ESP_ERR_INVALID_STATE;
    }
    if (volume < 0) {
        volume = 0;
    }
    if (volume > 127) {
        volume = 127;
    }
    /* Fire-and-forget control command: the confirmed value comes back as an
     * EvA2DPVolume record (kinds are documented in bt.go). */
    uint8_t tl = s_vol_tl;
    s_vol_tl = (s_vol_tl + 1) & 0x0F; /* AVRCP TL is a 4-bit field */
    return esp_avrc_ct_send_set_absolute_volume_cmd_by_handle(
        s_conn, tl, (uint8_t)volume);
#endif
}

int32_t llgo_bt_a2dp_stats(uint32_t *total, uint32_t *written,
                           uint32_t *dropped)
{
    /* Acquire loads of single-writer counters (see the declaration). */
    *total = __atomic_load_n((const uint32_t *)&s_pcm_total, __ATOMIC_ACQUIRE);
    *written = __atomic_load_n((const uint32_t *)&s_i2s_written,
                               __ATOMIC_ACQUIRE);
    *dropped = __atomic_load_n((const uint32_t *)&s_pcm_dropped,
                               __ATOMIC_ACQUIRE);
    return ESP_OK;
}

#else /* !ESP_PLATFORM */

#define LLGO_BT_UNSUPPORTED 0x7F00

int32_t llgo_bt_a2dp_enable(int32_t bclk, int32_t ws, int32_t dout,
                            int32_t dma_desc, int32_t dma_frames)
{
    (void)bclk; (void)ws; (void)dout; (void)dma_desc; (void)dma_frames;
    return LLGO_BT_UNSUPPORTED;
}
int32_t llgo_bt_a2dp_disable(void) { return LLGO_BT_UNSUPPORTED; }
int32_t llgo_bt_a2dp_volume_set(int32_t volume)
{
    (void)volume;
    return LLGO_BT_UNSUPPORTED;
}
int32_t llgo_bt_a2dp_stats(uint32_t *total, uint32_t *written,
                           uint32_t *dropped)
{
    (void)total; (void)written; (void)dropped;
    return 0;
}

#endif /* ESP_PLATFORM */
