/* Thin C glue between llgo-compiled Go code and the ESP-IDF new I2C master
 * driver (driver/i2c_master.h).
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

#include "driver/gpio.h"
#include "driver/i2c_master.h"

/* Install a master bus: port auto-selected, internal pull-ups off (real
 * buses want external ones), glitch filter left at the driver default.
 * Note: this IDF release's i2c_master_bus_config_t has no timeout field;
 * the only bound is the per-transfer xfer_timeout_ms wait that every
 * llgo_i2c_transmit/receive/transmit_receive call passes down (Go's
 * DefaultTimeoutMs).
 * The clock source MUST be spelled out: on ESP32-S31 the I2C sources are
 * XTAL and RC_FAST only, so a zeroed union is not "the default" but an
 * invalid source that fails the install with ESP_ERR_INVALID_ARG. */
int32_t llgo_i2c_new_bus(uintptr_t *out, int32_t sda, int32_t scl)
{
    i2c_master_bus_config_t cfg = {
        .i2c_port = -1, /* auto */
        .sda_io_num = (gpio_num_t)sda,
        .scl_io_num = (gpio_num_t)scl,
        .clk_source = I2C_CLK_SRC_DEFAULT,
        .glitch_ignore_cnt = 7,
        .flags.enable_internal_pullup = 0,
    };
    i2c_master_bus_handle_t handle;
    esp_err_t e = i2c_new_master_bus(&cfg, &handle);
    if (e != ESP_OK) return e;
    *out = (uintptr_t)handle;
    return ESP_OK;
}

int32_t llgo_i2c_add_device(uintptr_t bus, uintptr_t *out, uint32_t addr, uint32_t speed_hz)
{
    i2c_device_config_t cfg = {
        .dev_addr_length = I2C_ADDR_BIT_LEN_7,
        .device_address = addr,
        .scl_speed_hz = speed_hz,
    };
    i2c_master_dev_handle_t handle;
    esp_err_t e = i2c_master_bus_add_device((i2c_master_bus_handle_t)bus, &cfg, &handle);
    if (e != ESP_OK) return e;
    *out = (uintptr_t)handle;
    return ESP_OK;
}

int32_t llgo_i2c_transmit(uintptr_t dev, const uint8_t *data, int32_t len, int32_t timeout_ms)
{
    return i2c_master_transmit((i2c_master_dev_handle_t)dev, data, (size_t)len, timeout_ms);
}

int32_t llgo_i2c_receive(uintptr_t dev, uint8_t *data, int32_t len, int32_t timeout_ms)
{
    return i2c_master_receive((i2c_master_dev_handle_t)dev, data, (size_t)len, timeout_ms);
}

int32_t llgo_i2c_transmit_receive(uintptr_t dev, const uint8_t *write, int32_t write_len,
                                  uint8_t *read, int32_t read_len, int32_t timeout_ms)
{
    return i2c_master_transmit_receive((i2c_master_dev_handle_t)dev, write, (size_t)write_len,
                                       read, (size_t)read_len, timeout_ms);
}

int32_t llgo_i2c_rm_device(uintptr_t dev)
{
    return i2c_master_bus_rm_device((i2c_master_dev_handle_t)dev);
}

int32_t llgo_i2c_del_bus(uintptr_t bus)
{
    return i2c_del_master_bus((i2c_master_bus_handle_t)bus);
}

#else /* !ESP_PLATFORM */

/* Template guard: outside an ESP-IDF build this file compiles to an
 * otherwise empty translation unit. */
typedef int llgo_i2c_glue_not_in_idf_build;

#endif /* ESP_PLATFORM */
