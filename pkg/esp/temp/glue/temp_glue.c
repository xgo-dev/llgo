/* Thin C glue between llgo-compiled Go code and the ESP-IDF temperature
 * sensor driver (driver/temperature_sensor.h, the esp_driver_tsens
 * component).
 *
 * This file is a TEMPLATE: copy it into your project's main component so the
 * IDF build compiles it there (add esp_driver_tsens to the component's
 * REQUIRES). The integration recipes and the boundary rules
 * (integer-and-pointer-only signatures, esp_err passed through as int32, IDF
 * macros stay on the C side) are documented in pkg/esp/README.md.
 *
 * The sensor is a single-instance peripheral and the driver is handle-based,
 * so this glue owns the handle in static storage (the Go side stays
 * handle-free; static buffers only, the family rule) and every entry reports
 * the esp_err of the driver call it wraps.
 *
 * Before Install has run the static handle is NULL and the lifecycle/read
 * entries return 0x7201 - the temp binding's ErrNotInstalled segment value,
 * outside the real esp_err range, so "the sensor was never installed" stays
 * distinguishable from the driver's own state errors.
 *
 * The driver's reading is a float (degrees Celsius); the boundary carries it
 * as integer milli-degrees, converted right here - no float crosses the
 * link.
 *
 * Outside an ESP-IDF build the whole body compiles away, so stray globbing
 * (hosted builds, IDE tasks, over-broad CMake file lists) stays harmless.
 */

#if defined(ESP_PLATFORM)

#include <stdint.h>

#include "driver/temperature_sensor.h"

/* The temp binding's ErrNotInstalled segment value (temp.go). */
#define LG_TEMP_ERR_NOT_INSTALLED 0x7201

static temperature_sensor_handle_t s_tsens;

int32_t llgo_temp_install(int32_t range_min_c, int32_t range_max_c, int32_t allow_pd)
{
    temperature_sensor_config_t cfg = TEMPERATURE_SENSOR_CONFIG_DEFAULT(range_min_c, range_max_c);
    if (allow_pd) {
        cfg.flags.allow_pd = 1;
    }
    return temperature_sensor_install(&cfg, &s_tsens);
}

int32_t llgo_temp_start(void)
{
    if (s_tsens == NULL) return LG_TEMP_ERR_NOT_INSTALLED;
    return temperature_sensor_enable(s_tsens);
}

int32_t llgo_temp_read_milli(int32_t *out_milli_c)
{
    if (s_tsens == NULL) return LG_TEMP_ERR_NOT_INSTALLED;
    float celsius;
    esp_err_t e = temperature_sensor_get_celsius(s_tsens, &celsius);
    if (e != ESP_OK) return e;
    /* One rounding, in the C domain, to the integer the boundary carries. */
    *out_milli_c = (int32_t)(celsius * 1000.0f + (celsius >= 0 ? 0.5f : -0.5f));
    return ESP_OK;
}

int32_t llgo_temp_stop(void)
{
    if (s_tsens == NULL) return LG_TEMP_ERR_NOT_INSTALLED;
    return temperature_sensor_disable(s_tsens);
}

int32_t llgo_temp_uninstall(void)
{
    if (s_tsens == NULL) return LG_TEMP_ERR_NOT_INSTALLED;
    esp_err_t e = temperature_sensor_uninstall(s_tsens);
    s_tsens = NULL;
    return e;
}

#else /* !ESP_PLATFORM */

/* Template guard: outside an ESP-IDF build this file compiles to an
 * otherwise empty translation unit. */
typedef int llgo_temp_glue_not_in_idf_build;

#endif /* ESP_PLATFORM */
