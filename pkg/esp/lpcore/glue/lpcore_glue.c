/* Thin C glue between llgo-compiled Go code and the ESP-IDF LP core (ULP)
 * driver.
 *
 * This file is a TEMPLATE: copy it into your project's main component so the
 * IDF build compiles it there (REQUIRES ulp; WHOLE_ARCHIVE required - see
 * pkg/esp/README.md for the recipes and the boundary rules: integer-and-
 * pointer-only signatures, esp_err passed through as int32, IDF structs stay
 * on the C side).
 *
 * Calls here stay short and bounded: the mailbox receive polls with a zero
 * tick timeout (the Go side paces its own retry loop with vTaskDelay), and
 * the mailbox send waits at most 500 ms for the LP side's acknowledgement
 * (a zero-tick send can never see the interrupt-delivered ACK, and caller-
 * level retries would advance the TX slot cursor and wipe unconsumed
 * slots). The 500 ms ceiling keeps the llgo fiber scheduler freeze (see the
 * concurrency note in lpcore.go) short and deterministic. The LP core
 * itself - the program the HP side loaded - runs on its own RISC-V core
 * and never enters this glue.
 *
 * The LP core firmware binary is NOT part of this glue: the consuming
 * project builds it with the IDF ULP build system (ulp_embed_binary in the
 * component CMakeLists, per the IDF lp_core examples) and hands the bytes
 * to Go, which passes them to llgo_lpcore_load.
 *
 * Outside an ESP-IDF build the whole body compiles away, so stray globbing
 * (hosted builds, IDE tasks, over-broad CMake file lists) stays harmless.
 */

#if defined(ESP_PLATFORM)

#include <stdbool.h>
#include <stdint.h>

#include "freertos/FreeRTOS.h"
#include "lp_core_mailbox.h"
#include "ulp_lp_core.h"

/* The HP-side mailbox handle lives here, C-side static, like the wifi
 * package's netif: Go never needs the opaque handle, and a struct with
 * FreeRTOS primitives must not travel the Go boundary. It initializes once
 * on the first mailbox call; the LP core initializes its own side inside its
 * program (the S31 mailbox is the hardware controller, so the two sides'
 * init order does not matter - the software fallback implementation is the
 * one that requires the LP core to init first). A static does not survive a
 * deep-sleep reboot, which is fine: the whole HP side restarts with it. */
static lp_mailbox_t s_mailbox;
static bool s_mailbox_ready;

static esp_err_t lpcore_mailbox_init_once(void)
{
    if (s_mailbox_ready) {
        return ESP_OK;
    }
    esp_err_t e = lp_core_mailbox_init(&s_mailbox, NULL);
    if (e == ESP_OK) {
        s_mailbox_ready = true;
    }
    return e;
}

int32_t llgo_lpcore_load(const uint8_t *prog, int32_t len)
{
    if (prog == NULL || len < 0) {
        return ESP_ERR_INVALID_ARG;
    }
    /* The driver rejects sizes over CONFIG_ULP_COPROC_RESERVE_MEM with
     * ESP_ERR_INVALID_SIZE; the Go side has already rejected anything over
     * the LP RAM's physical 32 KiB ceiling with the same code. */
    return ulp_lp_core_load_binary(prog, (size_t)len);
}

int32_t llgo_lpcore_run(int32_t wakeup_src, int32_t timer_sleep_us)
{
    /* Build the driver's config struct here: its shape carries Kconfig /
     * soc_caps conditionals (a skip_lp_rom_boot field exists only on chips
     * with an LP ROM, which the S31 lacks), so it never crosses to Go.
     * Named initializers keep this correct on every target. */
    ulp_lp_core_cfg_t cfg = {
        .wakeup_source = (uint32_t)wakeup_src,
        .lp_timer_sleep_duration_us = (uint32_t)timer_sleep_us,
    };
    return ulp_lp_core_run(&cfg);
}

int32_t llgo_lpcore_stop(void)
{
    /* ulp_lp_core_stop is void: stopping an already-stopped core is
     * harmless, and there is no error to report - the Go side returns nil
     * for the ESP_OK this hands back. */
    ulp_lp_core_stop();
    return ESP_OK;
}

int32_t llgo_lpcore_mailbox_send(int32_t msg)
{
    esp_err_t e = lpcore_mailbox_init_once();
    if (e != ESP_OK) {
        return e;
    }
    /* Bounded blocking send: the HP driver's ACK arrives through the
     * mailbox interrupt + semaphore, and the timeout is FreeRTOS ticks. A
     * zero-tick send would return before the interrupt machinery can ever
     * deliver the ACK - and each send advances the TX slot cursor, so a
     * caller-level retry storm wraps around and wipes slots the LP side
     * has not consumed yet. One bounded wait per message is the driver's
     * intended shape; the bound (500 ms) keeps the scheduler freeze (see
     * the concurrency note in lpcore.go) short and deterministic.
     */
    return lp_core_mailbox_send(s_mailbox, (lp_message_t)msg, pdMS_TO_TICKS(500));
}

int32_t llgo_lpcore_mailbox_recv(int32_t *out)
{
    if (out == NULL) {
        return ESP_ERR_INVALID_ARG;
    }
    esp_err_t e = lpcore_mailbox_init_once();
    if (e != ESP_OK) {
        return e;
    }
    lp_message_t msg = 0;
    e = lp_core_mailbox_receive(s_mailbox, &msg, 0);
    if (e == ESP_OK) {
        *out = (int32_t)msg;
    }
    return e;
}

#else /* !ESP_PLATFORM */

/* Template guard: outside an ESP-IDF build this file compiles to an
 * otherwise empty translation unit. */
typedef int llgo_lpcore_glue_not_in_idf_build;

#endif /* ESP_PLATFORM */
