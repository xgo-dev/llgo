/* Thin C glue between llgo-compiled Go code and the ESP-IDF SPI master
 * driver (driver/spi_master.h, the esp_driver_spi component) for the
 * ESP32-S31's one general-purpose master, SPI2_HOST.
 *
 * This file is a TEMPLATE: copy it into your project's main component so the
 * IDF build compiles it there (add esp_driver_spi to the component's
 * REQUIRES). The integration recipes and the boundary rules
 * (integer-and-pointer-only signatures, esp_err passed through as int32, IDF
 * macros stay on the C side) are documented in pkg/esp/README.md.
 *
 * The S31 exposes one general-purpose SPI master, so this glue owns the bus
 * and the single device in static storage (the Go side stays handle-free;
 * static buffers only, the family rule) and every entry reports the esp_err
 * of the driver call it wraps.
 *
 * Every entry is synchronous. The transfers go through the driver's polling
 * path (spi_device_polling_start/portMAX_DELAY + spi_device_polling_end with
 * a bounded tick wait): a real transaction completes in microseconds to a
 * few milliseconds and the llgo task spins inside polling_end until it does
 * - exclusive by construction under llgo's single-task cooperative
 * scheduler. The bound is what keeps an ABSENT peripheral (an emulator
 * without an SPI model) from spinning forever: the wait ends with
 * ESP_ERR_TIMEOUT, leaving the driver's polling state stuck (documented on
 * the Go side; only a reboot clears it).
 *
 * Before Init has run the static device handle is NULL and every entry
 * returns 0x7101 - the spi binding's ErrNotInit segment value, outside the
 * real esp_err range, so "the package was never brought up" stays
 * distinguishable from the driver's own state errors. 0x7102
 * (ErrBadFrequency) and 0x7103 (ErrBusy) are the other two segment values,
 * repeated here defensively after the Go-side guards.
 *
 * The DMA story: the bus always initializes with SPI_DMA_CH_AUTO. Caller
 * buffers flow straight into the transaction - IDF v6.3 stages DMA-
 * unfriendly buffers (unaligned, internal/PSRAM placement) through its own
 * temporary copies because SPI_TRANS_DMA_BUFFER_ALIGN_MANUAL is never set,
 * and the Go side word-rounds the rx lengths (the RX DMA descriptor writes
 * whole 32-bit words). This glue therefore needs no staging buffers of its
 * own.
 *
 * Outside an ESP-IDF build the whole body compiles away, so stray globbing
 * (hosted builds, IDE tasks, over-broad CMake file lists) stays harmless.
 */

#if defined(ESP_PLATFORM)

#include <stdbool.h>
#include <stdint.h>

#include "driver/spi_master.h"
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"

/* The spi binding's segment values (spi.go). */
#define LG_SPI_ERR_NOT_INIT 0x7101
#define LG_SPI_ERR_BAD_FREQ 0x7102
#define LG_SPI_ERR_BUSY     0x7103

/* Bounded wait, in FreeRTOS ticks (10 ms each at the default 100 Hz): the
 * transfer wait has to cover the slowest real transaction (4092 bytes at
 * 10 MHz is ~3.3 ms) several times over while still ending a stuck one
 * (absent peripheral) in finite time. The bus-acquire lock has no such
 * knob - the driver refuses finite waits outright ("acquire finite time
 * not supported now") - and under llgo's single-task cooperative scheduler
 * the lock is uncontended, so its portMAX_DELAY wait is instantaneous in
 * practice. */
#define LG_SPI_TRANS_TICKS 30

/* The four frequencies the binding accepts (spi.go): the 80 MHz peripheral
 * ceiling over power-of-two divisions 1/2/4/8, the rates the driver's
 * integer divider reproduces exactly. */
static bool freq_ok(uint32_t hz)
{
    return hz == 10000000u || hz == 20000000u ||
           hz == 40000000u || hz == 80000000u;
}

/* Pin range guard, mirroring the Go side: [0, 64) like the gpio package,
 * with cs additionally accepting -1 (no CS line). */
static bool pin_ok(int32_t pin)
{
    return pin >= 0 && pin < 64;
}

static spi_device_handle_t s_dev;
static int32_t s_cs;    /* the cs pin the device was added with */
static uint32_t s_hz;   /* the device's configured clock rate */
static bool s_held;     /* an AcquireBus session is active */
static bool s_held_trans; /* a CS-pinned transfer ran while held (CS is
                           * asserted right now) */

static esp_err_t add_device(uint32_t hz)
{
    spi_device_interface_config_t dev = {};
    dev.mode = 0;               /* CPOL 0 / CPHA 0 */
    dev.clock_speed_hz = (int)hz;
    dev.spics_io_num = (int)s_cs;
    dev.queue_size = 1;         /* polling path only */
    return spi_bus_add_device(SPI2_HOST, &dev, &s_dev);
}

/* Run one polling transaction with the CS-pinned flag while an
 * AcquireBus session is active. spi_device_polling_start only accepts
 * portMAX_DELAY for its (uncontended) bus-lock wait; the completion wait
 * is the bounded polling_end. Returns the first driver error, or
 * ESP_ERR_TIMEOUT when the transaction did not complete within
 * LG_SPI_TRANS_TICKS - which also leaves the driver's polling state stuck
 * (see the file header). */
static esp_err_t poll_trans(uint32_t flags, size_t bits, size_t rxbits,
                            const void *tx, void *rx)
{
    spi_transaction_t t = {};
    t.flags = flags;
    t.length = bits;
    t.rxlength = rxbits;
    t.tx_buffer = tx;
    t.rx_buffer = rx;
    esp_err_t e = spi_device_polling_start(s_dev, &t, portMAX_DELAY);
    if (e != ESP_OK) {
        return e;
    }
    return spi_device_polling_end(s_dev, LG_SPI_TRANS_TICKS);
}

int32_t llgo_spi_init(int32_t mosi, int32_t miso, int32_t sclk, int32_t cs,
                      uint32_t hz)
{
    if (s_dev != NULL) {
        return ESP_ERR_INVALID_STATE;
    }
    if (!freq_ok(hz)) {
        return LG_SPI_ERR_BAD_FREQ;
    }
    if (!pin_ok(mosi) || !pin_ok(miso) || !pin_ok(sclk) ||
        (cs != -1 && !pin_ok(cs))) {
        return ESP_ERR_INVALID_ARG;
    }
    spi_bus_config_t bus = {};
    bus.mosi_io_num = (int)mosi;
    bus.miso_io_num = (int)miso;
    bus.sclk_io_num = (int)sclk;
    bus.max_transfer_sz = 0;    /* driver default: 4092 bytes with DMA */
    esp_err_t e = spi_bus_initialize(SPI2_HOST, &bus, SPI_DMA_CH_AUTO);
    if (e != ESP_OK) {
        return e;
    }
    s_cs = cs;
    s_hz = hz;
    e = add_device(hz);
    if (e != ESP_OK) {
        spi_bus_free(SPI2_HOST); /* keep the failed Init from half-existing */
        return e;
    }
    s_held = false;
    s_held_trans = false;
    return ESP_OK;
}

int32_t llgo_spi_set_frequency(uint32_t hz)
{
    if (s_dev == NULL) {
        return LG_SPI_ERR_NOT_INIT;
    }
    if (!freq_ok(hz)) {
        return LG_SPI_ERR_BAD_FREQ;
    }
    if (s_held) {
        return LG_SPI_ERR_BUSY; /* no rebind under a pinned device */
    }
    esp_err_t e = spi_bus_remove_device(s_dev);
    if (e != ESP_OK) {
        return e;
    }
    s_dev = NULL;
    e = add_device(hz);
    if (e != ESP_OK) {
        return e; /* the package is honestly back at the not-init state */
    }
    s_hz = hz;
    return ESP_OK;
}

int32_t llgo_spi_transfer(const uint8_t *tx, int32_t tx_len,
                          uint8_t *rx, int32_t rx_len)
{
    if (s_dev == NULL) {
        return LG_SPI_ERR_NOT_INIT;
    }
    int32_t n = (tx_len > rx_len) ? tx_len : rx_len;
    if (n <= 0) {
        return ESP_OK;
    }
    uint32_t flags = s_held ? SPI_TRANS_CS_KEEP_ACTIVE : 0u;
    esp_err_t e = poll_trans(flags, (size_t)n * 8, (size_t)rx_len * 8,
                             (tx_len > 0) ? (const void *)tx : NULL,
                             (rx_len > 0) ? (void *)rx : NULL);
    if (e != ESP_OK) {
        return e;
    }
    s_held_trans = s_held_trans || s_held;
    return ESP_OK;
}

int32_t llgo_spi_write(const uint8_t *tx, int32_t tx_len)
{
    if (s_dev == NULL) {
        return LG_SPI_ERR_NOT_INIT;
    }
    if (tx_len <= 0) {
        return ESP_OK;
    }
    uint32_t flags = s_held ? SPI_TRANS_CS_KEEP_ACTIVE : 0u;
    esp_err_t e = poll_trans(flags, (size_t)tx_len * 8, 0, tx, NULL);
    if (e != ESP_OK) {
        return e;
    }
    s_held_trans = s_held_trans || s_held;
    return ESP_OK;
}

/* The register-read shape: the tx bytes in one transaction, then the rx
 * bytes in a second with the MOSI line idle. Without a held bus CS
 * deasserts between the halves (documented); with one, both halves carry
 * SPI_TRANS_CS_KEEP_ACTIVE and CS stays asserted across them. */
int32_t llgo_spi_write_read(const uint8_t *tx, int32_t tx_len,
                            uint8_t *rx, int32_t rx_len)
{
    if (s_dev == NULL) {
        return LG_SPI_ERR_NOT_INIT;
    }
    uint32_t flags = s_held ? SPI_TRANS_CS_KEEP_ACTIVE : 0u;
    if (tx_len > 0) {
        esp_err_t e = poll_trans(flags, (size_t)tx_len * 8, 0, tx, NULL);
        if (e != ESP_OK) {
            return e;
        }
        s_held_trans = s_held_trans || s_held;
    }
    if (rx_len > 0) {
        esp_err_t e = poll_trans(flags, (size_t)rx_len * 8, (size_t)rx_len * 8,
                                 NULL, rx);
        if (e != ESP_OK) {
            return e;
        }
        s_held_trans = s_held_trans || s_held;
    }
    return ESP_OK;
}

int32_t llgo_spi_actual_freq_khz(int32_t *out_khz)
{
    if (s_dev == NULL) {
        return LG_SPI_ERR_NOT_INIT;
    }
    if (out_khz == NULL) {
        return ESP_ERR_INVALID_ARG;
    }
    int khz = 0;
    esp_err_t e = spi_device_get_actual_freq(s_dev, &khz);
    if (e != ESP_OK) {
        return e;
    }
    *out_khz = (int32_t)khz;
    return ESP_OK;
}

int32_t llgo_spi_acquire_bus(void)
{
    if (s_dev == NULL) {
        return LG_SPI_ERR_NOT_INIT;
    }
    if (s_held) {
        return LG_SPI_ERR_BUSY;
    }
    /* spi_device_acquire_bus refuses finite waits outright, so the wait is
     * the driver's portMAX_DELAY; the lock is uncontended in an llgo
     * program (one task, one device) and returns immediately. */
    esp_err_t e = spi_device_acquire_bus(s_dev, portMAX_DELAY);
    if (e != ESP_OK) {
        return e;
    }
    s_held = true;
    s_held_trans = false;
    return ESP_OK;
}

int32_t llgo_spi_release_bus(void)
{
    if (s_dev == NULL) {
        return LG_SPI_ERR_NOT_INIT;
    }
    if (!s_held) {
        return LG_SPI_ERR_BUSY;
    }
    if (s_held_trans) {
        /* Drop CS the pinned sequence left asserted: one zero-phase
         * transaction (no data phases) without SPI_TRANS_CS_KEEP_ACTIVE -
         * the hardware asserts CS (already low, no glitch), clocks nothing
         * and deasserts at the end. A stuck zero-phase transaction (absent
         * peripheral) reports ESP_ERR_TIMEOUT and leaves the bus held. */
        esp_err_t e = poll_trans(0, 0, 0, NULL, NULL);
        if (e != ESP_OK) {
            return e;
        }
    }
    spi_device_release_bus(s_dev);
    s_held = false;
    s_held_trans = false;
    return ESP_OK;
}

int32_t llgo_spi_deinit(void)
{
    if (s_dev == NULL) {
        return LG_SPI_ERR_NOT_INIT;
    }
    if (s_held) {
        return LG_SPI_ERR_BUSY; /* release first */
    }
    esp_err_t e = spi_bus_remove_device(s_dev);
    s_dev = NULL;
    s_held_trans = false;
    if (e != ESP_OK) {
        return e;
    }
    return spi_bus_free(SPI2_HOST);
}

#else /* !ESP_PLATFORM */

/* Template guard: outside an ESP-IDF build this file compiles to an
 * otherwise empty translation unit. */
typedef int llgo_spi_glue_not_in_idf_build;

#endif /* ESP_PLATFORM */
