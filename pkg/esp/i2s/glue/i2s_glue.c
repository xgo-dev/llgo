/* Thin C glue between llgo-compiled Go code and the ESP-IDF new I2S driver,
 * standard (Philips) mode only (driver/i2s_std.h).
 *
 * This file is a TEMPLATE: copy it into your project's main component so the
 * IDF build compiles it there. The integration recipes and the boundary
 * rules (integer-and-pointer-only signatures, esp_err passed through as
 * int32, IDF config macros/structs stay on the C side) are documented in
 * pkg/esp/README.md.
 *
 * Handles cross the boundary as uintptr integers (the Go side stores them
 * as uintptr so the GC never scans them); the glue casts them back to the
 * C handle types. All driver calls here are synchronous and bounded by the
 * driver's own transfer timeout, so no mailbox is needed.
 *
 * Outside an ESP-IDF build the whole body compiles away, so stray globbing
 * (hosted builds, IDE tasks, over-broad CMake file lists) stays harmless.
 */

#if defined(ESP_PLATFORM)

#include <stdint.h>
#include <stddef.h>

#include "driver/i2s_std.h"

/* Negative pin numbers mean "signal unused" on the Go side. */
static gpio_num_t lg_i2s_pin(int32_t pin)
{
    return (pin < 0) ? I2S_GPIO_UNUSED : (gpio_num_t)pin;
}

/* Alloc one TX or RX channel on any free port; the channel config macro
 * embeds the DMA descriptor/frame counts Go cannot spell. */
int32_t llgo_i2s_new_channel(uintptr_t *out, int32_t tx, int32_t slave)
{
    i2s_chan_config_t cfg = I2S_CHANNEL_DEFAULT_CONFIG(
        I2S_NUM_AUTO, slave ? I2S_ROLE_SLAVE : I2S_ROLE_MASTER);
    i2s_chan_handle_t handle = NULL;
    esp_err_t e;
    if (tx) {
        e = i2s_new_channel(&cfg, &handle, NULL);
    } else {
        e = i2s_new_channel(&cfg, NULL, &handle);
    }
    if (e != ESP_OK) return e;
    *out = (uintptr_t)handle;
    return ESP_OK;
}

/* Standard (Philips) mode init. The Philips slot defaults and the clock
 * defaults (256x MCLK multiple, default clock source) stay in the macros;
 * only the values Go knows - pins, bit width, slots, sample rate - are
 * passed in. A negative pin means the signal is unused (I2S_GPIO_UNUSED). */
int32_t llgo_i2s_init_std(uintptr_t chan, int32_t mclk, int32_t bclk, int32_t ws,
                          int32_t dout, int32_t din, int32_t bits, int32_t stereo,
                          int32_t sample_rate_hz)
{
    /* The Go API only spells Bits16/Bits32; anything else is a Go-side bug
     * and gets ESP_ERR_INVALID_ARG instead of a silent 16-bit fallback. */
    if (bits != 16 && bits != 32) {
        return ESP_ERR_INVALID_ARG;
    }
    i2s_data_bit_width_t width = (bits == 32) ? I2S_DATA_BIT_WIDTH_32BIT
                                              : I2S_DATA_BIT_WIDTH_16BIT;
    i2s_slot_mode_t mode = stereo ? I2S_SLOT_MODE_STEREO : I2S_SLOT_MODE_MONO;
    i2s_std_config_t std = {
        .clk_cfg = I2S_STD_CLK_DEFAULT_CONFIG((uint32_t)sample_rate_hz),
        .slot_cfg = I2S_STD_PHILIPS_SLOT_DEFAULT_CONFIG(width, mode),
        .gpio_cfg = {
            .mclk = lg_i2s_pin(mclk),
            .bclk = lg_i2s_pin(bclk),
            .ws = lg_i2s_pin(ws),
            .dout = lg_i2s_pin(dout),
            .din = lg_i2s_pin(din),
        },
    };
    return i2s_channel_init_std_mode((i2s_chan_handle_t)chan, &std);
}

int32_t llgo_i2s_enable(uintptr_t chan)
{
    return i2s_channel_enable((i2s_chan_handle_t)chan);
}

int32_t llgo_i2s_disable(uintptr_t chan)
{
    return i2s_channel_disable((i2s_chan_handle_t)chan);
}

int32_t llgo_i2s_write(uintptr_t chan, const uint8_t *data, int32_t len,
                       int32_t *written, int32_t timeout_ms)
{
    size_t n = 0;
    esp_err_t e = i2s_channel_write((i2s_chan_handle_t)chan, data, (size_t)len,
                                    &n, (uint32_t)timeout_ms);
    *written = (int32_t)n;
    return e;
}

int32_t llgo_i2s_read(uintptr_t chan, uint8_t *data, int32_t len,
                      int32_t *filled, int32_t timeout_ms)
{
    size_t n = 0;
    esp_err_t e = i2s_channel_read((i2s_chan_handle_t)chan, data, (size_t)len,
                                   &n, (uint32_t)timeout_ms);
    *filled = (int32_t)n;
    return e;
}

int32_t llgo_i2s_del(uintptr_t chan)
{
    return i2s_del_channel((i2s_chan_handle_t)chan);
}

#else /* !ESP_PLATFORM */

/* Template guard: outside an ESP-IDF build this file compiles to an
 * otherwise empty translation unit. */
typedef int llgo_i2s_glue_not_in_idf_build;

#endif /* ESP_PLATFORM */
