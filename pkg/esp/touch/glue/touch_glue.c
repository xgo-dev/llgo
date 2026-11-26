/* Thin C glue between llgo-compiled Go code and the ESP-IDF touch sensor
 * driver (driver/touch_sens.h, the esp_driver_touch_sens component - the
 * touch hardware v3 API the ESP32-S31 ships).
 *
 * This file is a TEMPLATE: copy it into your project's main component so the
 * IDF build compiles it there (add esp_driver_touch_sens to the component's
 * REQUIRES). The integration recipes and the boundary rules
 * (integer-and-pointer-only signatures, esp_err passed through as int32, IDF
 * macros stay on the C side) are documented in pkg/esp/README.md.
 *
 * The touch controller is a single-instance peripheral and the driver is
 * handle-based, so this glue owns the controller and channel state in static
 * storage (the Go side stays handle-free; static buffers only, the family
 * rule) and every entry reports the esp_err of the driver call it wraps.
 *
 * Before Init has run the static controller handle is NULL and every entry
 * returns 0x7301 - the touch binding's ErrNotInit segment value, outside the
 * real esp_err range, so "the package was never brought up" stays
 * distinguishable from the driver's own state errors.
 *
 * Outside an ESP-IDF build the whole body compiles away, so stray globbing
 * (hosted builds, IDE tasks, over-broad CMake file lists) stays harmless.
 */

#if defined(ESP_PLATFORM)

#include <stdbool.h>
#include <stdint.h>

#include "driver/touch_sens.h"
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"

/* The number of channels the chip exposes (14 on the ESP32-S31); the Go side
 * rejects out-of-range ids before they reach the table index here. */
#define LG_TOUCH_CHANS TOUCH_TOTAL_CHAN_NUM

/* The touch binding's ErrNotInit segment value (touch.go). */
#define LG_TOUCH_ERR_NOT_INIT 0x7301

static touch_sensor_handle_t s_sens;
static touch_channel_handle_t s_chans[LG_TOUCH_CHANS];

/* The controller and sample configuration, kept static because the driver
 * holds the sample_cfg pointer past the new_controller call. One sample
 * configuration (no frequency hopping) covers the binding's surface. */
static touch_sensor_sample_config_t s_sample;
static touch_sensor_config_t s_sens_cfg;

/* Event ring: the active/inactive callbacks run in ISR context and only
 * enqueue; Go drains with llgo_touch_poll_event. Sixteen slots absorb a full
 * 14-channel scan's transitions between two polls. Overflow drops the NEWEST
 * events (the ring keeps the oldest first): transitions, not data - a caller
 * polling slower than the scan loses detail, never consistency. */
#define LG_TOUCH_RING_CAP 16
static struct {
    int32_t chan;
    int32_t active; /* 0/1 */
} s_ring[LG_TOUCH_RING_CAP];
static volatile uint32_t s_ring_head; /* dequeue index (Go side) */
static volatile uint32_t s_ring_tail; /* enqueue index (ISR side) */
static portMUX_TYPE s_ring_mux = portMUX_INITIALIZER_UNLOCKED;

static void ring_push(int chan_id, bool active)
{
    portENTER_CRITICAL(&s_ring_mux);
    uint32_t next = (s_ring_tail + 1) % LG_TOUCH_RING_CAP;
    if (next != s_ring_head) { /* full: drop the event, head keeps draining */
        s_ring[s_ring_tail].chan = chan_id;
        s_ring[s_ring_tail].active = active ? 1 : 0;
        s_ring_tail = next;
    }
    portEXIT_CRITICAL(&s_ring_mux);
}

static bool IRAM_ATTR on_active(touch_sensor_handle_t sens,
                                const touch_active_event_data_t *event, void *ctx)
{
    (void)sens; (void)ctx;
    ring_push(event->chan_id, true);
    return false;
}

static bool IRAM_ATTR on_inactive(touch_sensor_handle_t sens,
                                  const touch_inactive_event_data_t *event, void *ctx)
{
    (void)sens; (void)ctx;
    ring_push(event->chan_id, false);
    return false;
}

/* The single sample configuration's fixed tail (the fields the Go Opts does
 * not expose): rc filter 1.5K/20fF, driver outputs low, mid bias voltage -
 * the TOUCH_SENSOR_V3_DEFAULT_SAMPLE_CONFIG defaults. */
static void sample_defaults(void)
{
    s_sample.rc_filter_res = 1;
    s_sample.rc_filter_cap = 1;
    s_sample.low_drv = 0;
    s_sample.high_drv = 0;
    s_sample.bias_volt = 5;
}

/* The filter configuration Init installs (DefaultFilter in touch.go):
 * benchmark IIR_16, jitter step 4, denoise level 1, data IIR_2, hysteresis 2,
 * debounce 2 - the TOUCH_SENSOR_DEFAULT_FILTER_CONFIG values. ConfigFilter
 * replaces any field of it at runtime. */
static esp_err_t install_default_filter(void)
{
    touch_sensor_filter_config_t f = {};
    f.benchmark.filter_mode = TOUCH_BM_IIR_FILTER_16;
    f.benchmark.jitter_step = 4;
    f.benchmark.denoise_lvl = 1;
    f.data.smooth_filter = TOUCH_SMOOTH_IIR_FILTER_2;
    f.data.active_hysteresis = 2;
    f.data.debounce_cnt = 2;
    return touch_sensor_config_filter(s_sens, &f);
}

int32_t llgo_touch_init(int32_t charge_times, int32_t div_num,
                        int32_t power_on_wait_us, int32_t meas_interval_us,
                        int32_t max_meas_time_us, int32_t out_mode)
{
    if (s_sens != NULL) {
        return ESP_ERR_INVALID_STATE;
    }
    sample_defaults();
    s_sample.div_num = (uint32_t)div_num;
    s_sample.charge_times = (uint32_t)charge_times;

    s_sens_cfg.power_on_wait_us = (uint32_t)power_on_wait_us;
    s_sens_cfg.meas_interval_us = (float)meas_interval_us; /* the only float on this boundary, built here from an int */
    s_sens_cfg.max_meas_time_us = (uint32_t)max_meas_time_us;
    s_sens_cfg.output_mode = (out_mode == 0) ? TOUCH_PAD_OUT_AS_DATA : TOUCH_PAD_OUT_AS_CLOCK;
    s_sens_cfg.sample_cfg_num = 1;
    s_sens_cfg.trigger_rise_cnt = TOUCH_SENSOR_AUTO_TRIGGER_RISE_CNT;
    s_sens_cfg.sample_cfg = &s_sample;

    esp_err_t e = touch_sensor_new_controller(&s_sens_cfg, &s_sens);
    if (e != ESP_OK) return e;

    /* The callbacks must be registered while the controller is disabled
     * (INIT state), i.e. right here - that is why Init registers them and
     * there is no separate registration call. */
    touch_event_callbacks_t cbs = {};
    cbs.on_active = on_active;
    cbs.on_inactive = on_inactive;
    e = touch_sensor_register_callbacks(s_sens, &cbs, NULL);
    if (e != ESP_OK) goto fail;

    e = install_default_filter();
    if (e != ESP_OK) goto fail;

    return ESP_OK;

fail:
    touch_sensor_del_controller(s_sens);
    s_sens = NULL;
    return e;
}

int32_t llgo_touch_config_filter(int32_t bm_mode, int32_t jitter_step,
                                 int32_t denoise_lvl, int32_t smooth,
                                 int32_t hysteresis, int32_t debounce)
{
    if (s_sens == NULL) return LG_TOUCH_ERR_NOT_INIT;
    touch_sensor_filter_config_t f = {};
    f.benchmark.filter_mode = (touch_benchmark_filter_mode_t)bm_mode;
    f.benchmark.jitter_step = (uint32_t)jitter_step;
    f.benchmark.denoise_lvl = (int)denoise_lvl;
    f.data.smooth_filter = (touch_smooth_filter_mode_t)smooth;
    f.data.active_hysteresis = (uint32_t)hysteresis;
    f.data.debounce_cnt = (uint32_t)debounce;
    return touch_sensor_config_filter(s_sens, &f);
}

int32_t llgo_touch_config_channel(int32_t chan_id, int32_t active_thresh)
{
    if (s_sens == NULL) return LG_TOUCH_ERR_NOT_INIT;
    if (chan_id < 0 || chan_id >= LG_TOUCH_CHANS) return ESP_ERR_INVALID_ARG;
    touch_channel_config_t cfg = {};
    cfg.active_thresh[0] = (uint32_t)active_thresh;
    esp_err_t e;
    if (s_chans[chan_id] != NULL) {
        /* An allocated channel can only be reconfigured while the controller
         * is disabled; surface that instead of silently mutating a running
         * sensor (SetThreshold drives the full cycle explicitly). */
        e = touch_sensor_reconfig_channel(s_chans[chan_id], &cfg);
    } else {
        e = touch_sensor_new_channel(s_sens, chan_id, &cfg, &s_chans[chan_id]);
        if (e != ESP_OK) s_chans[chan_id] = NULL;
    }
    return e;
}

int32_t llgo_touch_start(void)
{
    if (s_sens == NULL) return LG_TOUCH_ERR_NOT_INIT;
    esp_err_t e = touch_sensor_enable(s_sens);
    if (e != ESP_OK) return e;
    e = touch_sensor_start_continuous_scanning(s_sens);
    if (e != ESP_OK) touch_sensor_disable(s_sens); /* keep the states paired */
    return e;
}

int32_t llgo_touch_stop(void)
{
    if (s_sens == NULL) return LG_TOUCH_ERR_NOT_INIT;
    esp_err_t e = touch_sensor_stop_continuous_scanning(s_sens);
    if (e != ESP_OK) return e;
    return touch_sensor_disable(s_sens);
}

int32_t llgo_touch_read(int32_t chan_id, int32_t data_type, uint32_t *out)
{
    if (s_sens == NULL) return LG_TOUCH_ERR_NOT_INIT;
    if (chan_id < 0 || chan_id >= LG_TOUCH_CHANS || out == NULL) return ESP_ERR_INVALID_ARG;
    if (s_chans[chan_id] == NULL) return ESP_ERR_INVALID_STATE;
    touch_chan_data_type_t t = (data_type == 1) ? TOUCH_CHAN_DATA_TYPE_BENCHMARK
                                                : TOUCH_CHAN_DATA_TYPE_SMOOTH;
    /* One sample configuration, so the data array length is one. */
    return touch_channel_read_data(s_chans[chan_id], t, out);
}

int32_t llgo_touch_recalibrate(int32_t chan_id)
{
    if (s_sens == NULL) return LG_TOUCH_ERR_NOT_INIT;
    if (chan_id < 0 || chan_id >= LG_TOUCH_CHANS) return ESP_ERR_INVALID_ARG;
    if (s_chans[chan_id] == NULL) return ESP_ERR_INVALID_STATE;
    touch_chan_benchmark_config_t bm = {};
    bm.do_reset = true; /* benchmark <- the channel's current data */
    return touch_channel_config_benchmark(s_chans[chan_id], &bm);
}

int32_t llgo_touch_set_threshold(int32_t chan_id, int32_t thresh)
{
    if (s_sens == NULL) return LG_TOUCH_ERR_NOT_INIT;
    if (chan_id < 0 || chan_id >= LG_TOUCH_CHANS) return ESP_ERR_INVALID_ARG;
    if (s_chans[chan_id] == NULL) return ESP_ERR_INVALID_STATE;
    /* touch_sensor_reconfig_channel needs the controller disabled, so run
     * the full stop -> disable -> reconfigure -> enable -> restart cycle;
     * scanning pauses briefly and the benchmark starts over. The cycle
     * always ends running: calling this on a stopped sensor starts it
     * again. */
    esp_err_t e = touch_sensor_stop_continuous_scanning(s_sens);
    if (e != ESP_OK && e != ESP_ERR_INVALID_STATE) return e;
    e = touch_sensor_disable(s_sens);
    if (e != ESP_OK && e != ESP_ERR_INVALID_STATE) return e;
    touch_channel_config_t cfg = {};
    cfg.active_thresh[0] = (uint32_t)thresh;
    e = touch_sensor_reconfig_channel(s_chans[chan_id], &cfg);
    esp_err_t e2 = touch_sensor_enable(s_sens);
    if (e == ESP_OK) e = e2;
    e2 = touch_sensor_start_continuous_scanning(s_sens);
    if (e == ESP_OK) e = e2;
    return e;
}

bool llgo_touch_poll_event(int32_t *out_chan, int32_t *out_active)
{
    bool have = false;
    portENTER_CRITICAL(&s_ring_mux);
    if (s_ring_head != s_ring_tail) {
        *out_chan = s_ring[s_ring_head].chan;
        *out_active = s_ring[s_ring_head].active;
        s_ring_head = (s_ring_head + 1) % LG_TOUCH_RING_CAP;
        have = true;
    }
    portEXIT_CRITICAL(&s_ring_mux);
    return have; /* 1 = event, 0 = ring empty (not an esp_err) */
}

int32_t llgo_touch_deinit(void)
{
    if (s_sens == NULL) return LG_TOUCH_ERR_NOT_INIT;
    esp_err_t e = touch_sensor_stop_continuous_scanning(s_sens);
    if (e != ESP_OK && e != ESP_ERR_INVALID_STATE) return e;
    e = touch_sensor_disable(s_sens);
    if (e != ESP_OK && e != ESP_ERR_INVALID_STATE) return e;
    for (int i = 0; i < LG_TOUCH_CHANS; i++) {
        if (s_chans[i] != NULL) {
            touch_sensor_del_channel(s_chans[i]);
            s_chans[i] = NULL;
        }
    }
    e = touch_sensor_del_controller(s_sens);
    s_sens = NULL;
    s_ring_head = s_ring_tail = 0;
    return e;
}

#else /* !ESP_PLATFORM */

/* Template guard: outside an ESP-IDF build this file compiles to an
 * otherwise empty translation unit. */
typedef int llgo_touch_glue_not_in_idf_build;

#endif /* ESP_PLATFORM */
