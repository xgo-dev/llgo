/* Demo-local C glue for the LP core acceptance probe.
 *
 * Three jobs, all demo-owned (a real project would bind only what its own
 * program needs):
 *
 *   1. heap marks (llgo_demo_heap) - both allocators are C-side facts.
 *   2. the embedded LP program's bytes (llgo_demo_lp_prog_*): the IDF ULP
 *      build links lp_core_main.bin into this firmware; the probe's Go side
 *      reads the pointer and the size through here and hands them to
 *      pkg/esp/lpcore.Load.
 *   3. the HP deep-sleep stretch (llgo_demo_lp_wakeup_cause,
 *      llgo_demo_sleep_timer, llgo_demo_deep_sleep): the esp_sleep surface
 *      is demo-level, deliberately outside pkg/esp/lpcore's API.
 *
 * The Go<->C boundary carries integers and pointers only; the 64-bit
 * microseconds travel as two 32-bit halves. C keeps no reference to Go
 * memory beyond the immediate call.
 */
#include <stdint.h>
#include <stdio.h>

#include "esp_bit_defs.h"
#include "esp_heap_caps.h"
#include "esp_sleep.h"

/* The LP program's exported markers, through the ULP build's generated
 * export header. */
#include "lp_core_main.h"

/* The LP program the IDF ULP build embedded: ulp_embed_binary objcopy's the
 * lp_core_main.bin into the link with _binary_ prefixed symbols (the same
 * externs the IDF lp_mailbox example declares). */
extern const uint8_t lp_core_main_bin_start[] asm("_binary_lp_core_main_bin_start");
extern const uint8_t lp_core_main_bin_end[]   asm("_binary_lp_core_main_bin_end");

const uint8_t *llgo_demo_lp_prog_ptr(void)
{
    return lp_core_main_bin_start;
}

int32_t llgo_demo_lp_prog_size(void)
{
    return (int32_t)(lp_core_main_bin_end - lp_core_main_bin_start);
}

/* The LP program's progress markers (idfproj/main/lp_core/main.c): boot
 * marker, mailbox-init result, the entry poll's result, and the receive
 * counter. The probe prints them right after the settle wait, so a broken LP
 * path reports exactly where it broke. */
int32_t llgo_demo_lp_markers(uint32_t *out)
{
    out[0] = ulp_lp_echo_boot;
    out[1] = ulp_lp_echo_mbinit;
    out[2] = ulp_lp_echo_poll;
    out[3] = ulp_lp_echo_count;
    return 0;
}

/* The LP RAM window as the HP side sees it: RTC_SLOW_MEM from
 * ulp_common_defs.h - the same pointer-sized macro ulp_lp_core_load_binary
 * memcpy's into (SOC_RTC_DATA_LOW, the LP RAM base). The probe reads the
 * first words back after Load and compares them with the program bytes,
 * which proves the load path landed in real, HP-visible memory. */
#include "ulp_common_defs.h"

int32_t llgo_demo_lp_ram_read(int32_t word_off, uint32_t *out, int32_t words)
{
    if (out == NULL || words < 0 || word_off < 0) {
        return ESP_ERR_INVALID_ARG;
    }
    for (int32_t i = 0; i < words; i++) {
        out[i] = RTC_SLOW_MEM[word_off + i];
    }
    return 0;
}

/* Print the two IDF pools at one probe stage. The tag travels as pointer +
 * length and is consumed immediately. */
void llgo_demo_heap(const char *tag, int32_t len)
{
    printf("[c] lp heap %.*s free=%u min=%u | internal=%u | psram=%u\n",
           (int)len, tag,
           (unsigned)heap_caps_get_free_size(MALLOC_CAP_DEFAULT),
           (unsigned)heap_caps_get_minimum_free_size(MALLOC_CAP_DEFAULT),
           (unsigned)heap_caps_get_free_size(MALLOC_CAP_INTERNAL),
           (unsigned)heap_caps_get_free_size(MALLOC_CAP_SPIRAM));
    fflush(stdout);
}

/* The wakeup cause as raw bitmap plus the IDF enum name of its first set
 * bit. IDF v6.3's esp_sleep_get_wakeup_causes returns BIT(cause) per wake
 * source (the one-cause accessor is deprecated); the names resolve here
 * because the enum's numbering is per-target (the esp_sleep.h members are
 * gated on soc caps), so the Go side compares names, never numbers. A cold
 * boot reports exactly ESP_SLEEP_WAKEUP_UNDEFINED. */
int32_t llgo_demo_lp_wakeup_cause(char *out, int32_t max)
{
    uint32_t causes = esp_sleep_get_wakeup_causes();
    const char *name = "ESP_SLEEP_WAKEUP_?";
    if (causes & BIT(ESP_SLEEP_WAKEUP_UNDEFINED)) {
        name = "ESP_SLEEP_WAKEUP_UNDEFINED";
    } else if (causes & BIT(ESP_SLEEP_WAKEUP_TIMER)) {
        name = "ESP_SLEEP_WAKEUP_TIMER";
    } else if (causes & BIT(ESP_SLEEP_WAKEUP_ULP)) {
        name = "ESP_SLEEP_WAKEUP_ULP";
    } else if (causes & BIT(ESP_SLEEP_WAKEUP_EXT0)) {
        name = "ESP_SLEEP_WAKEUP_EXT0";
    } else if (causes & BIT(ESP_SLEEP_WAKEUP_EXT1)) {
        name = "ESP_SLEEP_WAKEUP_EXT1";
    } else if (causes & BIT(ESP_SLEEP_WAKEUP_GPIO)) {
        name = "ESP_SLEEP_WAKEUP_GPIO";
    } else if (causes & BIT(ESP_SLEEP_WAKEUP_VAD)) {
        name = "ESP_SLEEP_WAKEUP_VAD";
    }
    snprintf(out, (size_t)max, "%s", name);
    return (int32_t)causes;
}

/* esp_sleep_enable_timer_wakeup with the 64-bit microseconds rebuilt from
 * two 32-bit halves (the llgo boundary passes 32-bit integers). */
int32_t llgo_demo_sleep_timer(int32_t us_lo, int32_t us_hi)
{
    uint64_t us = ((uint64_t)(uint32_t)us_hi << 32) | (uint32_t)us_lo;
    return (int32_t)esp_sleep_enable_timer_wakeup(us);
}

/* The noreturn deep-sleep entry. The LP core keeps running across the HP
 * core's sleep; the LP (RTC) timer armed above raises the HP core again. */
void llgo_demo_deep_sleep(void)
{
    esp_deep_sleep_start();
}
