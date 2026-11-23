/* Thin C glue between llgo-compiled Go code and the ESP-IDF esp_wifi stack.
 *
 * This file is a TEMPLATE: copy it into your project's main component so the
 * IDF build compiles it there (the llgo package never builds C code). Two
 * integration recipes - copy-into-main or CMake-reference - and the boundary
 * rules this glue implements are documented in pkg/esp/README.md.
 *
 * Rules that keep the llgo fiber runtime safe on the other side of the
 * boundary:
 *   (1) integer-and-pointer-only signatures - no float values cross the
 *       Go<->C link (see the _demo/esp-idf-wifi host/make-artifact.sh
 *       step 4, which patches the relocatable's float ABI);
 *   (2) no large stack buffers. Go goroutines run on 8 KiB fiber stacks,
 *       so bulk data (scan records, ...) lives in static storage;
 *   (3) WIFI_INIT_CONFIG_DEFAULT() stays here: the macro embeds dozens of
 *       Kconfig constants and function pointers that Go cannot spell;
 *   (4) events never call back into Go. esp_event handlers run on the
 *       event-loop task and the llgo fiber scheduler is single-task global
 *       state - an entry from a second FreeRTOS task would corrupt it.
 *       Handlers only copy into the mailbox below; Go polls it paced by
 *       vTaskDelay(10) = 100ms, which also keeps the idle tasks fed for
 *       the watchdog.
 *
 * Outside an ESP-IDF build the whole body compiles away, so stray globbing
 * (hosted builds, IDE tasks, over-broad CMake file lists) stays harmless.
 */

#if defined(ESP_PLATFORM)

#include <string.h>
#include <stdio.h>

#include "esp_wifi.h"
#include "esp_event.h"
#include "esp_netif.h"
#include "esp_system.h"
#include "esp_heap_caps.h"
#include "esp_idf_version.h"
#include "nvs_flash.h"

/* Station credentials are compiled in from main/wifi_creds.h, a file kept
 * out of version control (see _demo/esp-idf-wifi/idfproj/main/
 * wifi_creds.h.example). Define LLGO_WIFI_SSID/LLGO_WIFI_PASS instead, or
 * leave both out for an empty default (scan-only runs). */
#if defined(__has_include)
#if __has_include("wifi_creds.h")
#include "wifi_creds.h"
#endif
#endif
#ifndef LLGO_WIFI_SSID
#define LLGO_WIFI_SSID ""
#endif
#ifndef LLGO_WIFI_PASS
#define LLGO_WIFI_PASS ""
#endif

/* ---- event mailbox (single producer: event-loop task; single consumer: llgo main) ---- */

enum { EV_NONE = 0, EV_STA_START = 1, EV_STA_CONNECTED = 2, EV_STA_DISCONNECTED = 3,
       EV_SCAN_DONE = 4, EV_GOT_IP = 5, EV_LOST_IP = 6 };

#define LWQ_SLOTS 16
typedef struct { int32_t kind; int32_t aux; uint32_t ip; } lw_slot_t;

static lw_slot_t s_q[LWQ_SLOTS];
static volatile uint32_t s_wr, s_rd, s_dropped;

static esp_netif_t *s_sta_netif;

/* Scan records live here, not on the stack: Go goroutines run on 8 KiB
 * fiber stacks and esp_wifi_scan_get_ap_records walks a deep call chain. */
#define LW_MAX_APS 20
static wifi_ap_record_t s_records[LW_MAX_APS];

/* Flat scan-record form handed to Go: scalar and fixed-size-array fields
 * only, no IDF bitfields or nested structs cross the boundary. */
typedef struct {
    uint8_t bssid[6];
    uint8_t ssid[33];
    uint8_t channel;
    int8_t rssi;
    int32_t authmode;
} llgo_ap_info_t;

static void lw_push(int32_t kind, int32_t aux, uint32_t ip)
{
    uint32_t w = s_wr;
    if ((uint32_t)(w - s_rd) >= LWQ_SLOTS) {
        s_dropped++;               /* full: drop the new event, count it */
        return;
    }
    s_q[w & (LWQ_SLOTS - 1)] = (lw_slot_t){ kind, aux, ip };
    s_wr = w + 1;
}

static void lw_on_event(void *arg, esp_event_base_t base, int32_t id, void *data)
{
    (void)arg;
    if (base == WIFI_EVENT) {
        switch (id) {
        case WIFI_EVENT_STA_START:     lw_push(EV_STA_START, 0, 0); break;
        case WIFI_EVENT_STA_CONNECTED: lw_push(EV_STA_CONNECTED, 0, 0); break;
        case WIFI_EVENT_STA_DISCONNECTED:
            lw_push(EV_STA_DISCONNECTED,
                    ((const wifi_event_sta_disconnected_t *)data)->reason, 0);
            break;
        case WIFI_EVENT_SCAN_DONE:     lw_push(EV_SCAN_DONE, 0, 0); break;
        default: break;
        }
    } else if (base == IP_EVENT && id == IP_EVENT_STA_GOT_IP) {
        lw_push(EV_GOT_IP, 0, ((const ip_event_got_ip_t *)data)->ip_info.ip.addr);
    } else if (base == IP_EVENT && id == IP_EVENT_STA_LOST_IP) {
        lw_push(EV_LOST_IP, 0, 0);
    }
}

/* ---- one-shot setup: NVS + netif + event loop + esp_wifi_init ---- */

int32_t llgo_wifi_setup(void)
{
    esp_err_t e = nvs_flash_init();
    if (e == ESP_ERR_NVS_NO_FREE_PAGES || e == ESP_ERR_NVS_NEW_VERSION_FOUND) {
        if ((e = nvs_flash_erase()) != ESP_OK) return e;
        e = nvs_flash_init();
    }
    if (e != ESP_OK) return e;
    if ((e = esp_netif_init()) != ESP_OK) return e;
    if ((e = esp_event_loop_create_default()) != ESP_OK) return e;
    s_sta_netif = esp_netif_create_default_wifi_sta();
    if (s_sta_netif == NULL) return ESP_FAIL;
    wifi_init_config_t cfg = WIFI_INIT_CONFIG_DEFAULT();
    return esp_wifi_init(&cfg);
}

int32_t llgo_wifi_events_register(void)
{
    esp_event_handler_instance_t inst;
    esp_err_t e = esp_event_handler_instance_register(WIFI_EVENT, ESP_EVENT_ANY_ID,
                                                      lw_on_event, NULL, &inst);
    if (e != ESP_OK) return e;
    /* ESP_EVENT_ANY_ID here too: the GOT_IP/LOST_IP dispatch already happens
     * by id inside lw_on_event, and registering the base alone keeps the two
     * subscriptions symmetrical (and LOST_IP actually reachable). */
    return esp_event_handler_instance_register(IP_EVENT, ESP_EVENT_ANY_ID,
                                               lw_on_event, NULL, &inst);
}

/* Diagnostics printed from the C side keep the Go<->C boundary integer-only. */
int32_t llgo_wifi_print_version(void)
{
    printf("[c] wifi glue: IDF %s, free heap %u bytes\n",
           esp_get_idf_version(), (unsigned)esp_get_free_heap_size());
    return ESP_OK;
}

int32_t llgo_wifi_start(void)
{
    return esp_wifi_start();
}

int32_t llgo_wifi_connect(void)
{
    return esp_wifi_connect();
}

int32_t llgo_wifi_disconnect(void)
{
    return esp_wifi_disconnect();
}

/* Station IP in network byte order, 0 = nothing acquired yet. */
uint32_t llgo_wifi_get_ip(void)
{
    esp_netif_ip_info_t info;
    memset(&info, 0, sizeof info);
    if (s_sta_netif == NULL || esp_netif_get_ip_info(s_sta_netif, &info) != ESP_OK) {
        return 0;
    }
    return info.ip.addr;
}

/* Switch the driver to station mode without configuring a target SSID.
 * esp_wifi_set_config rejects an empty SSID (ESP_ERR_WIFI_SSID), so a
 * scan-only run needs this instead of set_sta: the default mode is soft AP and
 * STA_START never fires there. */
int32_t llgo_wifi_set_mode_sta(void)
{
    return (int32_t)esp_wifi_set_mode(WIFI_MODE_STA);
}

/* Hand the compiled-in credentials to Go through the integer/pointer
 * boundary; lengths are clamped to the Go-side buffers. */
void llgo_wifi_default_sta(uint8_t *ssid, int32_t ssid_max, int32_t *ssid_len,
                           uint8_t *pass, int32_t pass_max, int32_t *pass_len)
{
    const char *s = LLGO_WIFI_SSID;
    const char *p = LLGO_WIFI_PASS;
    size_t s_cap = (ssid_max > 0) ? (size_t)ssid_max : 0;
    size_t p_cap = (pass_max > 0) ? (size_t)pass_max : 0;
    size_t sl = strnlen(s, s_cap);
    size_t pl = strnlen(p, p_cap);
    memcpy(ssid, s, sl);
    memcpy(pass, p, pl);
    *ssid_len = (int32_t)sl;
    *pass_len = (int32_t)pl;
}

/* Flush the C-side stdio before the program parks: pending buffered UART
 * output must not race the halt path (a parked task never flushes). */
void llgo_wifi_finish(void)
{
    fflush(stdout);
    fflush(stderr);
}

/* IDF heap watermarks. These describe the IDF allocator pool only: the
 * llgo GC heap is a separate static window that never takes from this pool.
 * With PSRAM enabled the collector additionally grows with segments from the
 * SPIRAM pool; that share shows in the psram figure, never in the internal
 * ones. */
uint32_t llgo_wifi_heap_free(void)
{
    return (uint32_t)esp_get_free_heap_size();
}

uint32_t llgo_wifi_heap_min(void)
{
    return (uint32_t)esp_get_minimum_free_heap_size();
}

/* Caps-scoped pools. With PSRAM enabled the IDF heap is split in two: internal
 * SRAM (what esp_get_free_heap_size reports) and external PSRAM, which only
 * appears in MALLOC_CAP_SPIRAM. Reporting both keeps the two layers apart. */
uint32_t llgo_wifi_heap_internal_free(void)
{
    return (uint32_t)heap_caps_get_free_size(MALLOC_CAP_INTERNAL);
}

uint32_t llgo_wifi_heap_spiram_free(void)
{
    return (uint32_t)heap_caps_get_free_size(MALLOC_CAP_SPIRAM);
}

/* Station credentials. Zero the config, copy ssid/pass, and pick the
 * minimum matching authmode from the password length (empty password ->
 * open). Storage stays in RAM so the acceptance runs are self-contained. */
int32_t llgo_wifi_set_sta(const uint8_t *ssid, int32_t ssid_len,
                          const uint8_t *pass, int32_t pass_len)
{
    if (ssid_len <= 0 || ssid_len > 32 || pass_len < 0 || pass_len > 64) {
        return ESP_ERR_WIFI_SSID;
    }
    wifi_config_t c;
    memset(&c, 0, sizeof c);
    memcpy(c.sta.ssid, ssid, ssid_len);
    memcpy(c.sta.password, pass, pass_len);
    c.sta.threshold.authmode = (pass_len >= 8) ? WIFI_AUTH_WPA2_PSK : WIFI_AUTH_OPEN;
    esp_err_t e = esp_wifi_set_mode(WIFI_MODE_STA);
    if (e != ESP_OK) return e;
    e = esp_wifi_set_config(WIFI_IF_STA, &c);
    if (e != ESP_OK) return e;
    return esp_wifi_set_storage(WIFI_STORAGE_RAM);
}

/* Blocking scan with the default config (NULL): active scan on all
 * channels. The call blocks on a driver semaphore, which yields the CPU,
 * so the frozen llgo scheduler is safe here. */
int32_t llgo_wifi_scan(int32_t blocking)
{
    return esp_wifi_scan_start(NULL, blocking != 0);
}

/* Take the scan result list (which also frees the driver's memory) and
 * marshal it into the flat Go-side form. */
int32_t llgo_wifi_scan_fetch(llgo_ap_info_t *out, int32_t max, int32_t *count)
{
    if (max > LW_MAX_APS) max = LW_MAX_APS;
    uint16_t n = (uint16_t)max;
    esp_err_t e = esp_wifi_scan_get_ap_records(&n, s_records);
    if (e != ESP_OK) return e;
    for (uint16_t i = 0; i < n; i++) {
        memcpy(out[i].bssid, s_records[i].bssid, 6);
        memcpy(out[i].ssid, s_records[i].ssid, 33);
        out[i].channel = s_records[i].primary;
        out[i].rssi = s_records[i].rssi;
        out[i].authmode = (int32_t)s_records[i].authmode;
    }
    *count = n;
    return ESP_OK;
}

/* Pop one mailbox slot: 1 = event, 0 = empty, one EV_OVERFLOW slot first if
 * events were ever dropped (reported via kind = -1). */
int32_t llgo_wifi_poll(int32_t *kind, int32_t *aux, uint32_t *ip)
{
    if (s_wr != s_rd) {
        lw_slot_t s = s_q[s_rd & (LWQ_SLOTS - 1)];
        s_rd++;
        *kind = s.kind; *aux = s.aux; *ip = s.ip;
        return 1;
    }
    if (s_dropped) {
        s_dropped = 0;
        *kind = -1; *aux = 0; *ip = 0;
        return 1;
    }
    return 0;
}

#else /* !ESP_PLATFORM */

/* Template guard: outside an ESP-IDF build this file compiles to an
 * otherwise empty translation unit. */
typedef int llgo_wifi_glue_not_in_idf_build;

#endif /* ESP_PLATFORM */
