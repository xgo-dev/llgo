/* Demo-local diagnostic glue for the I2S RX bring-up (HW mode only).
 *
 * The board's record path never produced a byte: every driver call reports
 * success but i2s_channel_read never fills. These helpers turn the chain
 * into observable facts the Go diagnostics print:
 *
 *   - lg_edges: software edge counter on one GPIO. Input enable only - the
 *     pad's function, matrix routing and output drive stay untouched, so
 *     counting works while an I2S channel drives the pin. MCLK (4.096 MHz)
 *     and BCLK (512 kHz) are heavily undersampled by a 240 MHz core; the
 *     count separates "line alive" (hundreds of flips) from "line dead"
 *     (0) - the absolute value carries no meaning.
 *   - llgo_i2s_diag_rxint_seen: high-frequency sampling of the I2S0 RX
 *     interrupt raw bits over one window, returning the OR of every sample
 *     (bit0 rx_done = one RX DMA buffer completed, bit2 rx_hung = the DMA
 *     did not keep up). The driver's ISR clears rx_done as each buffer
 *     lands, so only an OR over samples catches it; a zero means no data
 *     ever reached the controller in that window.
 *   - llgo_i2s_diag_tx_edges: one TX reference probe (the playback path
 *     this board verified): build a TX channel on the codec's clock pins,
 *     count the edges, tear it down. Sanity-checks the counting method and
 *     the pins themselves before any RX verdict is read.
 *   - llgo_i2s_diag_rx_probe: one RX probe - build, init, enable, count
 *     edges on all four codec pins, sample the rx_done delta, try two
 *     short reads with the driver's timeout, snapshot the driver's RX
 *     register programming, tear down - all from one call so the Go side
 *     only prints. The explicit port parameter (the pkg glue only spells
 *     I2S_NUM_AUTO) lets the same probe run on I2S1 as a controller swap
 *     experiment.
 *
 * Integer-and-pointer-only boundary; the out array is Go heap memory
 * (Go never relocates heap objects, so the pointer stays valid), the
 * channel lives only inside one call. The RX probes are sequential and
 * bounded (~3 s each: edge windows plus two 1 s-timeout reads), so no
 * mailbox is needed.
 */
#include <stdint.h>
#include <stddef.h>

#include "driver/gpio.h"
#include "driver/i2s_std.h"
#include "esp_timer.h"
#include "hal/gpio_ll.h"
#include "soc/i2s_reg.h"
#include "soc/i2s_struct.h"

static int32_t lg_edges(int32_t pin, int32_t ms)
{
    gpio_dev_t *g = GPIO_LL_GET_HW(0);
    gpio_ll_input_enable(g, (uint32_t)pin);
    int32_t last = (int32_t)gpio_ll_get_level(g, (uint32_t)pin);
    int32_t flips = 0;
    int64_t end = esp_timer_get_time() + (int64_t)ms * 1000;
    while (esp_timer_get_time() < end) {
        int32_t v = (int32_t)gpio_ll_get_level(g, (uint32_t)pin);
        if (v != last) {
            flips++;
            last = v;
        }
    }
    return flips;
}

/* OR of the RX raw bits over one sampling window (50 ms). */
static int32_t lg_rxint_seen(void)
{
    const uint32_t mask = I2S_RX_DONE_INT_RAW | I2S_RX_HUNG_INT_RAW;
    uint32_t seen = 0;
    int64_t end = esp_timer_get_time() + 50000;
    while (esp_timer_get_time() < end) {
        seen |= I2S0.int_raw.val & mask;
    }
    return (int32_t)seen;
}

/* TX reference probe: [0]=err, [1]=mclk edges, [2]=bclk edges, [3]=ws
 * edges, [4]=dout edges. Returns the entry count (5). */
int32_t llgo_i2s_diag_tx_edges(int32_t *out, int32_t mclk, int32_t bclk,
                               int32_t ws, int32_t dout, int32_t rate)
{
    i2s_chan_config_t cfg = I2S_CHANNEL_DEFAULT_CONFIG(I2S_NUM_AUTO, I2S_ROLE_MASTER);
    i2s_chan_handle_t ch = NULL;
    out[0] = (int32_t)i2s_new_channel(&cfg, &ch, NULL);
    if (out[0] != ESP_OK) return 5;
    i2s_std_config_t std = {
        .clk_cfg = I2S_STD_CLK_DEFAULT_CONFIG((uint32_t)rate),
        .slot_cfg = I2S_STD_PHILIPS_SLOT_DEFAULT_CONFIG(I2S_DATA_BIT_WIDTH_16BIT,
                                                        I2S_SLOT_MODE_STEREO),
        .gpio_cfg = {
            .mclk = (mclk < 0) ? I2S_GPIO_UNUSED : (gpio_num_t)mclk,
            .bclk = (bclk < 0) ? I2S_GPIO_UNUSED : (gpio_num_t)bclk,
            .ws = (ws < 0) ? I2S_GPIO_UNUSED : (gpio_num_t)ws,
            .dout = (dout < 0) ? I2S_GPIO_UNUSED : (gpio_num_t)dout,
            .din = I2S_GPIO_UNUSED,
        },
    };
    out[0] = (int32_t)i2s_channel_init_std_mode(ch, &std);
    if (out[0] == ESP_OK) {
        out[0] = (int32_t)i2s_channel_enable(ch);
    }
    if (out[0] == ESP_OK) {
        out[1] = lg_edges(mclk, 200);
        out[2] = lg_edges(bclk, 200);
        out[3] = lg_edges(ws, 200);
        out[4] = lg_edges(dout, 200);
    }
    i2s_channel_disable(ch);
    i2s_del_channel(ch);
    return 5;
}

/* RX probe: [0]=new err, [1]=init err, [2]=enable err, [3]=mclk edges,
 * [4]=bclk edges, [5]=ws edges, [6]=din edges, [7]=rxint OR over 50 ms,
 * [8]=read1 n, [9]=read1 err, [10]=read2 n, [11]=read2 err, [12]=rx_conf
 * raw, [13]=rx_conf1 raw. Returns the entry count (14). */
int32_t llgo_i2s_diag_rx_probe(int32_t *out, int32_t port, int32_t mclk,
                               int32_t bclk, int32_t ws, int32_t din,
                               int32_t bits, int32_t stereo, int32_t rate)
{
    for (int i = 0; i < 14; i++) {
        out[i] = 0;
    }
    i2s_chan_config_t cfg = I2S_CHANNEL_DEFAULT_CONFIG(port, I2S_ROLE_MASTER);
    i2s_chan_handle_t ch = NULL;
    out[0] = (int32_t)i2s_new_channel(&cfg, NULL, &ch);
    if (out[0] != ESP_OK) return 14;
    i2s_data_bit_width_t width = (bits == 32) ? I2S_DATA_BIT_WIDTH_32BIT
                                              : I2S_DATA_BIT_WIDTH_16BIT;
    i2s_slot_mode_t mode = stereo ? I2S_SLOT_MODE_STEREO : I2S_SLOT_MODE_MONO;
    i2s_std_config_t std = {
        .clk_cfg = I2S_STD_CLK_DEFAULT_CONFIG((uint32_t)rate),
        .slot_cfg = I2S_STD_PHILIPS_SLOT_DEFAULT_CONFIG(width, mode),
        .gpio_cfg = {
            .mclk = (mclk < 0) ? I2S_GPIO_UNUSED : (gpio_num_t)mclk,
            .bclk = (bclk < 0) ? I2S_GPIO_UNUSED : (gpio_num_t)bclk,
            .ws = (ws < 0) ? I2S_GPIO_UNUSED : (gpio_num_t)ws,
            .dout = I2S_GPIO_UNUSED,
            .din = (din < 0) ? I2S_GPIO_UNUSED : (gpio_num_t)din,
        },
    };
    out[1] = (int32_t)i2s_channel_init_std_mode(ch, &std);
    if (out[1] == ESP_OK) {
        out[2] = (int32_t)i2s_channel_enable(ch);
    }
    if (out[2] == ESP_OK) {
        out[3] = lg_edges(mclk, 200);
        out[4] = lg_edges(bclk, 200);
        out[5] = lg_edges(ws, 200);
        out[6] = lg_edges(din, 200);
        out[7] = lg_rxint_seen();
        uint8_t buf[240];
        size_t n;
        for (int r = 0; r < 2; r++) {
            n = 0;
            esp_err_t e = i2s_channel_read(ch, buf, sizeof(buf), &n, 1000);
            out[8 + r * 2] = (int32_t)n;
            out[9 + r * 2] = (int32_t)e;
            if (e != ESP_OK) {
                out[10] = -1; /* the second read is not reached */
                out[11] = -1;
                break;
            }
        }
        out[12] = (int32_t)I2S0.rx_conf.val;
        out[13] = (int32_t)I2S0.rx_conf1.val;
    }
    i2s_channel_disable(ch);
    i2s_del_channel(ch);
    return 14;
}
