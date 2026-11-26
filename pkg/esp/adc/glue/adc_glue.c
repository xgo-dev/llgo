/* Thin C glue between llgo-compiled Go code and the ESP-IDF ADC oneshot
 * driver (esp_adc/adc_oneshot.h + esp_adc/adc_cali_scheme.h, the esp_adc
 * component) for the ESP32-S31's analog converter.
 *
 * This file is a TEMPLATE: copy it into your project's main component so the
 * IDF build compiles it there (add esp_adc and esp_hal_ana_conv to the
 * component's REQUIRES). The integration recipes and the boundary rules
 * (integer-and-pointer-only signatures, esp_err passed through as int32, IDF
 * macros stay on the C side) are documented in pkg/esp/README.md.
 *
 * The driver is handle-based: one oneshot unit handle per ADC unit plus one
 * line-fitting calibration handle per unit. This glue owns all of them in a
 * static two-slot table (one slot per unit - the Go side stays handle-free;
 * static storage only, the family rule, no malloc/free here: the driver's
 * own handle allocations stay inside the IDF). Every entry reports the
 * esp_err of the driver call it wraps, or one of the segment values below
 * for the binding's own refusals.
 *
 * Before Init has brought a unit up, that slot's handle is NULL and the
 * entries that need it return 0x7001 - the adc binding's ErrNotInit segment
 * value, outside the real esp_err range, so "the unit was never brought up"
 * stays distinguishable from the driver's own ESP_ERR_INVALID_STATE
 * (0x103), which the read entries use for "the pin's channel was never
 * configured" (a state only this glue's bookkeeping can see). 0x7002
 * (ErrBadUnit), 0x7003 (ErrBadPin), 0x7004 (ErrBadAtten), 0x7005
 * (ErrBadBitwidth) and 0x7006 (ErrCaliUnavailable) are the other segment
 * values, repeated here defensively after the Go-side guards.
 *
 * S31 facts the glue encodes (see adc.go for the full story): the analog
 * pads are GPIO42..57 (ADC1 = 42..49, ADC2 = 50..57), the pad->channel map
 * is the driver's adc_oneshot_io_to_channel, the conversion width is fixed
 * at 17 bits, the only attenuation is ADC_ATTEN_DB_0, and the millivolt
 * conversion is the channel-aware line-fitting scheme (the classic
 * channel-blind adc_cali_raw_to_voltage is not supported on this chip).
 *
 * The calibration handle is created lazily, on the first ReadMillivolts of
 * a unit: the scheme allocates its characteristics from the IDF heap
 * (heap_caps_calloc inside the driver - not this glue's doing) and is
 * released by Deinit. When the chip's calibration eFuse block is not
 * burnt, the driver falls back to the nominal scale and the scheme still
 * succeeds - the Go side surfaces that regime through llgo_adc_efuse_version
 * instead of inventing a failure the driver does not report.
 *
 * Outside an ESP-IDF build the whole body compiles away, so stray globbing
 * (hosted builds, IDE tasks, over-broad CMake file lists) stays harmless.
 */

#if defined(ESP_PLATFORM)

#include <stdint.h>

#include "esp_adc/adc_cali.h"
#include "esp_adc/adc_cali_scheme.h"
#include "esp_adc/adc_oneshot.h"
#include "hal/adc_efuse_hal.h"
#include "hal/adc_ll.h" /* ADC_LL_RTC_MAX_BITWIDTH, the chip's fixed width */

/* The adc binding's segment values (adc.go). */
#define LG_ADC_ERR_NOT_INIT         0x7001
#define LG_ADC_ERR_BAD_UNIT         0x7002
#define LG_ADC_ERR_BAD_PIN          0x7003
#define LG_ADC_ERR_BAD_ATTEN        0x7004
#define LG_ADC_ERR_BAD_BITWIDTH     0x7005
#define LG_ADC_ERR_CALI_UNAVAILABLE 0x7006

#define LG_ADC_UNIT_NUM 2 /* ADC_UNIT_1, ADC_UNIT_2 */

/* One slot per unit: the oneshot handle, the lazy calibration handle, the
 * conversion width Init recorded, and the bitmask of channels Config has
 * configured (the read entries refuse a never-configured channel with the
 * driver-domain ESP_ERR_INVALID_STATE - a state only this bookkeeping
 * sees). */
static struct {
    adc_oneshot_unit_handle_t unit;
    adc_cali_handle_t cali;
    int32_t bitwidth;
    uint32_t configured;
} s_slots[LG_ADC_UNIT_NUM];

static bool unit_ok(int32_t unit)
{
    return unit >= 0 && unit < LG_ADC_UNIT_NUM;
}

/* Generic GPIO range guard (the Go side enforces the chip's analog pad set
 * 42..57; this repeats the wider bound defensively, and anything narrower
 * than an analog pad is answered by the driver's io->channel map). */
static bool pin_ok(int32_t pin)
{
    return pin >= 0 && pin < 64;
}

static bool bitwidth_ok(int32_t w)
{
    return w == ADC_BITWIDTH_DEFAULT || w == ADC_LL_RTC_MAX_BITWIDTH;
}

int32_t llgo_adc_init(int32_t unit_id, int32_t bitwidth)
{
    if (!unit_ok(unit_id)) {
        return LG_ADC_ERR_BAD_UNIT;
    }
    if (s_slots[unit_id].unit != NULL) {
        return ESP_ERR_INVALID_STATE;
    }
    if (!bitwidth_ok(bitwidth)) {
        return LG_ADC_ERR_BAD_BITWIDTH;
    }
    adc_oneshot_unit_init_cfg_t init = {};
    init.unit_id = (adc_unit_t)unit_id; /* clk_src 0 = driver default, ULP off */
    adc_oneshot_unit_handle_t handle = NULL;
    esp_err_t e = adc_oneshot_new_unit(&init, &handle);
    if (e != ESP_OK) {
        return e;
    }
    s_slots[unit_id].unit = handle;
    s_slots[unit_id].bitwidth = bitwidth;
    s_slots[unit_id].configured = 0;
    return ESP_OK;
}

int32_t llgo_adc_config(int32_t pin, int32_t atten)
{
    if (!pin_ok(pin)) {
        return LG_ADC_ERR_BAD_PIN;
    }
    if (atten != ADC_ATTEN_DB_0) { /* the S31's only attenuation */
        return LG_ADC_ERR_BAD_ATTEN;
    }
    adc_unit_t unit_id;
    adc_channel_t chan;
    esp_err_t e = adc_oneshot_io_to_channel(pin, &unit_id, &chan);
    if (e != ESP_OK) {
        return e; /* ESP_ERR_INVALID_ARG for a GPIO that is not an analog pad */
    }
    if (s_slots[unit_id].unit == NULL) {
        return LG_ADC_ERR_NOT_INIT;
    }
    adc_oneshot_chan_cfg_t cfg = {};
    cfg.atten = (adc_atten_t)atten;
    cfg.bitwidth = (adc_bitwidth_t)s_slots[unit_id].bitwidth;
    e = adc_oneshot_config_channel(s_slots[unit_id].unit, chan, &cfg);
    if (e == ESP_OK) {
        s_slots[unit_id].configured |= 1u << chan;
    }
    return e;
}

/* The shared read path: map the pad, check the unit and its configured
 * bitmask, run one oneshot conversion. */
static esp_err_t read_raw(int32_t pin, adc_unit_t *unit_id, adc_channel_t *chan,
                          int *out_raw)
{
    esp_err_t e = adc_oneshot_io_to_channel(pin, unit_id, chan);
    if (e != ESP_OK) {
        return e;
    }
    if (s_slots[*unit_id].unit == NULL) {
        return LG_ADC_ERR_NOT_INIT;
    }
    if (!(s_slots[*unit_id].configured & (1u << *chan))) {
        return ESP_ERR_INVALID_STATE;
    }
    return adc_oneshot_read(s_slots[*unit_id].unit, *chan, out_raw);
}

int32_t llgo_adc_read(int32_t pin, int32_t *out_raw)
{
    if (!pin_ok(pin)) {
        return LG_ADC_ERR_BAD_PIN;
    }
    if (out_raw == NULL) {
        return ESP_ERR_INVALID_ARG;
    }
    adc_unit_t unit_id;
    adc_channel_t chan;
    int raw = 0;
    esp_err_t e = read_raw(pin, &unit_id, &chan, &raw);
    if (e != ESP_OK) {
        return e;
    }
    *out_raw = (int32_t)raw; /* the 17-bit comparator vector, as-is */
    return ESP_OK;
}

int32_t llgo_adc_read_millivolts(int32_t pin, int32_t *out_mv)
{
    if (!pin_ok(pin)) {
        return LG_ADC_ERR_BAD_PIN;
    }
    if (out_mv == NULL) {
        return ESP_ERR_INVALID_ARG;
    }
    adc_unit_t unit_id;
    adc_channel_t chan;
    int raw = 0;
    esp_err_t e = read_raw(pin, &unit_id, &chan, &raw);
    if (e != ESP_OK) {
        return e;
    }
    if (s_slots[unit_id].cali == NULL) {
        /* Lazy line-fitting scheme creation: the chip's only calibration
         * scheme, at the unit's recorded width and the only attenuation. */
        adc_cali_line_fitting_config_t ccfg = {};
        ccfg.unit_id = unit_id;
        ccfg.atten = ADC_ATTEN_DB_0;
        ccfg.bitwidth = (adc_bitwidth_t)s_slots[unit_id].bitwidth;
        adc_cali_handle_t cali = NULL;
        if (adc_cali_create_scheme_line_fitting(&ccfg, &cali) != ESP_OK) {
            return LG_ADC_ERR_CALI_UNAVAILABLE;
        }
        s_slots[unit_id].cali = cali;
    }
    int mv = 0;
    e = adc_cali_chan_raw_to_voltage(s_slots[unit_id].cali, chan, raw, &mv);
    if (e != ESP_OK) {
        return e;
    }
    *out_mv = (int32_t)mv;
    return ESP_OK;
}

int32_t llgo_adc_deinit(int32_t unit_id)
{
    if (!unit_ok(unit_id)) {
        return LG_ADC_ERR_BAD_UNIT;
    }
    if (s_slots[unit_id].unit == NULL) {
        return LG_ADC_ERR_NOT_INIT;
    }
    if (s_slots[unit_id].cali != NULL) {
        esp_err_t e = adc_cali_delete_scheme_line_fitting(s_slots[unit_id].cali);
        s_slots[unit_id].cali = NULL;
        if (e != ESP_OK) {
            return e;
        }
    }
    esp_err_t e = adc_oneshot_del_unit(s_slots[unit_id].unit);
    if (e != ESP_OK) {
        return e; /* the slot stays as the driver left it */
    }
    s_slots[unit_id].unit = NULL;
    s_slots[unit_id].configured = 0;
    return ESP_OK;
}

int32_t llgo_adc_efuse_version(int32_t *out_ver)
{
    if (out_ver == NULL) {
        return ESP_ERR_INVALID_ARG;
    }
    /* The exact value the line-fitting scheme keys its characteristics on:
     * 0 = no burnt block, the scheme runs the nominal scale. */
    *out_ver = (int32_t)adc_hal_efuse_get_version();
    return ESP_OK;
}

#else /* !ESP_PLATFORM */

/* Template guard: outside an ESP-IDF build this file compiles to an
 * otherwise empty translation unit. */
typedef int llgo_adc_glue_not_in_idf_build;

#endif /* ESP_PLATFORM */
