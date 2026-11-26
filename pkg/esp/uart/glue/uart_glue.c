/* Thin C glue between llgo-compiled Go code and the ESP-IDF UART driver
 * (driver/uart.h, the esp_driver_uart component).
 *
 * This file is a TEMPLATE: copy it into your project's main component so the
 * IDF build compiles it there (add esp_driver_uart to the component's
 * REQUIRES). The integration recipes and the boundary rules
 * (integer-and-pointer-only signatures, esp_err passed through as int32, IDF
 * macros stay on the C side) are documented in pkg/esp/README.md.
 *
 * The driver is port-numbered global state, so this glue tracks which ports
 * it has brought up in one static table (the Go side stays handle-free;
 * static buffers only, the family rule) and every entry reports an esp_err:
 * 0 = ESP_OK, the LG_UART_ERR_NOT_INIT segment value for a port never Init'd
 * (outside the real esp_err range, so "the port was never brought up" stays
 * distinguishable from the driver's own state errors - and the guard keeps
 * the driver's NULL port object out of its own hands), and the driver's own
 * esp_err otherwise, passed through unchanged.
 *
 * The install is deliberately the synchronous-polling shape: a 256-byte RX
 * ring buffer (the driver refuses anything up to the 128-byte hardware
 * FIFO), a 256-byte TX ring buffer (Write enqueues and returns - the
 * driver's ISR feeds the hardware FIFO from there; without a TX ring the
 * write loop parks on a FIFO-space semaphore that only the TX interrupt
 * gives, and a stalled line would hold the single llgo task forever), and
 * NO event queue (size 0 / NULL handle - nothing here allocates a FreeRTOS
 * queue, and the driver's only allocations are its own internal ones, made
 * from the llgo task like every driver install in this family; this glue
 * itself calls no allocator).
 *
 * Outside an ESP-IDF build the whole body compiles away, so stray globbing
 * (hosted builds, IDE tasks, over-broad CMake file lists) stays harmless.
 */

#if defined(ESP_PLATFORM)

#include <stdbool.h>
#include <stdint.h>

#include "driver/uart.h"
#include "freertos/FreeRTOS.h" /* pdMS_TO_TICKS */

/* The uart binding's ErrNotInit segment value (uart.go). */
#define LG_UART_ERR_NOT_INIT 0x7802

/* RX and TX ring buffer sizes for the driver install: comfortably above the
 * 128-byte hardware FIFO the driver requires, small enough to stay honest
 * about RAM. The TX ring is what keeps Write enqueue-and-return: the driver
 * feeds the FIFO from it on the ISR. */
#define LG_UART_RX_BUF 256
#define LG_UART_TX_BUF 256

static bool s_installed[UART_NUM_MAX];

/* The port guard every entry but init runs through first: range against the
 * chip's own UART_NUM_MAX (an out-of-range index would be undefined behavior
 * on the static table below), then the installed table. */
static int32_t lg_uart_check(int32_t port)
{
    if (port < 0 || port >= UART_NUM_MAX) {
        return ESP_ERR_INVALID_ARG;
    }
    if (!s_installed[port]) {
        return LG_UART_ERR_NOT_INIT;
    }
    return ESP_OK;
}

int32_t llgo_uart_init(int32_t port, int32_t tx_pin, int32_t rx_pin, int32_t baud)
{
    if (port < 0 || port >= UART_NUM_MAX) {
        return ESP_ERR_INVALID_ARG;
    }
    if (baud <= 0) {
        return ESP_ERR_INVALID_ARG;
    }
    /* The default frame format (8N1, no flow control) and the chip's default
     * source clock - spelled out, like every config struct in this family:
     * the macro stays on the C side. */
    uart_config_t cfg = {
        .source_clk = UART_SCLK_DEFAULT,
        .baud_rate = baud,
        .data_bits = UART_DATA_8_BITS,
        .parity = UART_PARITY_DISABLE,
        .stop_bits = UART_STOP_BITS_1,
        .flow_ctrl = UART_HW_FLOWCTRL_DISABLE,
    };
    /* Ring buffers for RX and TX, no event queue (0 / NULL) - the
     * synchronous shape. intr flags 0: the driver's default interrupt
     * allocation. */
    esp_err_t e = uart_driver_install((uart_port_t)port, LG_UART_RX_BUF, LG_UART_TX_BUF, 0, NULL, 0);
    if (e != ESP_OK) {
        return e;
    }
    e = uart_param_config((uart_port_t)port, &cfg);
    if (e == ESP_OK) {
        e = uart_set_pin(port, tx_pin, rx_pin, UART_PIN_NO_CHANGE, UART_PIN_NO_CHANGE);
    }
    if (e != ESP_OK) {
        /* Roll the install back so the port stays in the pre-init state the
         * static table promises. */
        uart_driver_delete((uart_port_t)port);
        return e;
    }
    s_installed[port] = true;
    return ESP_OK;
}

int32_t llgo_uart_set_baud(int32_t port, int32_t baud)
{
    esp_err_t e = lg_uart_check(port);
    if (e != ESP_OK) {
        return e;
    }
    if (baud <= 0) {
        return ESP_ERR_INVALID_ARG;
    }
    return uart_set_baudrate((uart_port_t)port, (uint32_t)baud);
}

int32_t llgo_uart_set_config(int32_t port, int32_t word_len, int32_t stop_bits, int32_t parity)
{
    esp_err_t e = lg_uart_check(port);
    if (e != ESP_OK) {
        return e;
    }
    /* Go-side frame values (uart.go) onto the driver's encodings, validated
     * before anything is applied. */
    uart_word_length_t w;
    uart_stop_bits_t s;
    uart_parity_t p;
    switch (word_len) {
    case 5: w = UART_DATA_5_BITS; break;
    case 6: w = UART_DATA_6_BITS; break;
    case 7: w = UART_DATA_7_BITS; break;
    case 8: w = UART_DATA_8_BITS; break;
    default: return ESP_ERR_INVALID_ARG;
    }
    switch (stop_bits) {
    case 1: s = UART_STOP_BITS_1; break;
    case 15: s = UART_STOP_BITS_1_5; break;
    case 2: s = UART_STOP_BITS_2; break;
    default: return ESP_ERR_INVALID_ARG;
    }
    switch (parity) {
    case 0: p = UART_PARITY_DISABLE; break;
    case 1: p = UART_PARITY_EVEN; break;
    case 2: p = UART_PARITY_ODD; break;
    default: return ESP_ERR_INVALID_ARG;
    }
    e = uart_set_word_length((uart_port_t)port, w);
    if (e != ESP_OK) return e;
    e = uart_set_stop_bits((uart_port_t)port, s);
    if (e != ESP_OK) return e;
    return uart_set_parity((uart_port_t)port, p);
}

int32_t llgo_uart_write(int32_t port, const uint8_t *data, int32_t len, int32_t *out_n)
{
    esp_err_t e = lg_uart_check(port);
    if (e != ESP_OK) {
        return e;
    }
    if (len <= 0 || data == NULL) {
        return ESP_ERR_INVALID_ARG;
    }
    /* Enqueue into the driver's TX ring buffer; the driver's ISR feeds the
     * hardware FIFO from there. Blocks only while the ring is full (a live
     * line at any baud drains faster than message-size writes fill it).
     * -1 maps to ESP_FAIL, the driver's own failure value. */
    int n = uart_write_bytes((uart_port_t)port, data, (size_t)len);
    if (n < 0) {
        return ESP_FAIL;
    }
    *out_n = (int32_t)n;
    return ESP_OK;
}

int32_t llgo_uart_read(int32_t port, uint8_t *buf, int32_t len, int32_t timeout_ms, int32_t *out_n)
{
    esp_err_t e = lg_uart_check(port);
    if (e != ESP_OK) {
        return e;
    }
    if (len <= 0 || buf == NULL || timeout_ms < 0) {
        return ESP_ERR_INVALID_ARG;
    }
    /* Milliseconds to the FreeRTOS tick budget here on the C side, where the
     * macro lives (the boundary stays integer-only). uart_read_bytes returns
     * what arrived within the budget (partial fills are normal), -1 on a
     * driver-level failure; 0 bytes within the budget is ESP_OK + n = 0 and
     * the Go side surfaces it as ErrTimeout. */
    int n = uart_read_bytes((uart_port_t)port, buf, (uint32_t)len,
                            (uint32_t)pdMS_TO_TICKS(timeout_ms));
    if (n < 0) {
        return ESP_FAIL;
    }
    *out_n = (int32_t)n;
    return ESP_OK;
}

int32_t llgo_uart_available(int32_t port, int32_t *out_n)
{
    esp_err_t e = lg_uart_check(port);
    if (e != ESP_OK) {
        return e;
    }
    size_t n = 0;
    e = uart_get_buffered_data_len((uart_port_t)port, &n);
    if (e != ESP_OK) {
        return e;
    }
    *out_n = (int32_t)n;
    return ESP_OK;
}

int32_t llgo_uart_flush_input(int32_t port)
{
    esp_err_t e = lg_uart_check(port);
    if (e != ESP_OK) {
        return e;
    }
    return uart_flush_input((uart_port_t)port);
}

int32_t llgo_uart_flush_output(int32_t port, int32_t timeout_ms)
{
    esp_err_t e = lg_uart_check(port);
    if (e != ESP_OK) {
        return e;
    }
    if (timeout_ms < 0) {
        return ESP_ERR_INVALID_ARG;
    }
    /* A bounded wait for the TX FIFO to drain; expiry reports the driver's
     * ESP_ERR_TIMEOUT (0x107), surfaced as ErrTimeout on the Go side. */
    return uart_wait_tx_done((uart_port_t)port, (uint32_t)pdMS_TO_TICKS(timeout_ms));
}

int32_t llgo_uart_deinit(int32_t port)
{
    esp_err_t e = lg_uart_check(port);
    if (e != ESP_OK) {
        return e;
    }
    e = uart_driver_delete((uart_port_t)port);
    if (e != ESP_OK) {
        return e;
    }
    s_installed[port] = false;
    return ESP_OK;
}

#else /* !ESP_PLATFORM */

/* Template guard: outside an ESP-IDF build this file compiles to an
 * otherwise empty translation unit. */
typedef int llgo_uart_glue_not_in_idf_build;

#endif /* ESP_PLATFORM */
