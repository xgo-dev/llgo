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
 *       so bulk data (scan records, station lists) lives in static storage;
 *   (3) WIFI_INIT_CONFIG_DEFAULT() stays here: the macro embeds dozens of
 *       Kconfig constants and function pointers that Go cannot spell;
 *   (4) events never call back into Go. esp_event handlers run on the
 *       event-loop task and the llgo fiber scheduler is single-task global
 *       state - an entry from a second FreeRTOS task would corrupt it.
 *       Handlers only copy into the mailbox below; Go polls it paced by
 *       vTaskDelay(10) = 100ms, which also keeps the idle tasks fed for
 *       the watchdog.
 *
 * Station and soft-AP roles share one mailbox and one driver handle. The
 * soft-AP side adds its own event kinds, its own netif (created lazily by
 * llgo_wifi_set_ap, which is also where the DHCP server comes from) and the
 * station-list query; nothing in the station path changed shape.
 *
 * Outside an ESP-IDF build the whole body compiles away, so stray globbing
 * (hosted builds, IDE tasks, over-broad CMake file lists) stays harmless.
 */

#if defined(ESP_PLATFORM)

#include <stddef.h>
#include <string.h>
#include <stdio.h>

#include "esp_wifi.h"
/* The soft-AP station list with each client's leased address
 * (esp_wifi_ap_get_sta_list_with_ip, wifi_sta_mac_ip_list_t) lives in its own
 * header, not in esp_wifi.h. */
#include "esp_wifi_ap_get_sta_list.h"
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
       EV_SCAN_DONE = 4, EV_GOT_IP = 5, EV_LOST_IP = 6,
       EV_AP_START = 7, EV_AP_STOP = 8, EV_AP_STA_CONNECTED = 9,
       EV_AP_STA_DISCONNECTED = 10, EV_AP_STA_IPASSIGNED = 11 };

/* One slot carries every field any event kind needs, so the consumer reads a
 * whole record or nothing: kind, the driver's reason code (aux), the IP,
 * and the soft-AP client's MAC plus its association id. */
#define LWQ_SLOTS 16
typedef struct {
    int32_t kind;
    int32_t aux;
    uint32_t ip;
    uint8_t mac[6];
    uint16_t aid;
} lw_slot_t;

static lw_slot_t s_q[LWQ_SLOTS];

/* Single producer (the esp_event loop task) / single consumer (the Go task).
 * The producer fills a slot and then publishes s_wr with a release store; the
 * consumer acquires s_wr before reading the slot and publishes s_rd with a
 * release store after copying it out. That ordering - not any atomicity of the
 * slot fields - is what makes the record cross intact: every slot field is
 * written before the index that exposes it, and read only after the index that
 * exposes it.
 *
 * Only plain stores and fences are used, no read-modify-write: RV32IMC has no
 * A extension, so an RMW would drag in libatomic. Both counters are
 * single-writer, so nothing needs it - s_dropped is written by the producer
 * alone, and its consumer keeps its own last-seen snapshot instead of clearing
 * the producer's counter. */
static uint32_t s_wr, s_rd;          /* s_wr: producer-owned, s_rd: consumer-owned */
static uint32_t s_dropped;           /* producer-owned */
static uint32_t s_dropped_seen;      /* consumer-owned */

#define LW_PUBLISH(v, val) __atomic_store_n(&(v), (val), __ATOMIC_RELEASE)
#define LW_ACQUIRE(v)      __atomic_load_n(&(v), __ATOMIC_ACQUIRE)

static esp_netif_t *s_sta_netif;
static esp_netif_t *s_ap_netif;

/* Mode bookkeeping. s_sta_configured / s_ap_configured record which interfaces
 * the program configured; while the program has not picked a mode itself, the
 * driver mode is their union - STA, AP, or APSTA when both are configured.
 * That union is what keeps SetAP and SetSTA from undoing each other, in any
 * order. An explicit llgo_wifi_set_mode owns the mode from then on and the two
 * flags describe only the interfaces. s_mode mirrors the mode this glue last
 * wrote, so redundant calls (and the driver-side restart they can cause) are
 * skipped. Only the Go task calls into this glue, so plain variables are
 * enough here; the mailbox above is the only cross-task structure. */
static bool s_sta_configured, s_ap_configured, s_mode_explicit;
static wifi_mode_t s_mode = WIFI_MODE_NULL;   /* what esp_wifi_init leaves */

static esp_err_t lw_apply_mode(bool want_sta, bool want_ap)
{
    wifi_mode_t m = WIFI_MODE_NULL;
    if (want_sta) m |= WIFI_MODE_STA;
    if (want_ap) m |= WIFI_MODE_AP;
    if (m == s_mode) return ESP_OK;
    esp_err_t e = esp_wifi_set_mode(m);
    if (e == ESP_OK) s_mode = m;
    return e;
}

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

/* Soft-AP station list: the flat form of wifi_sta_info_t plus the address the
 * AP's DHCP server leased, keyed by MAC. Layout must stay identical to the Go
 * Station struct (asserted below). */
typedef struct {
    uint8_t mac[6];
    int8_t rssi;
    uint32_t ip;
} llgo_wifi_sta_t;

_Static_assert(sizeof(llgo_ap_info_t) == 48, "Go ApInfo struct layout drift");
_Static_assert(offsetof(llgo_ap_info_t, ssid) == 6, "Go ApInfo struct layout drift");
_Static_assert(offsetof(llgo_ap_info_t, channel) == 39, "Go ApInfo struct layout drift");
_Static_assert(offsetof(llgo_ap_info_t, rssi) == 40, "Go ApInfo struct layout drift");
_Static_assert(offsetof(llgo_ap_info_t, authmode) == 44, "Go ApInfo struct layout drift");

_Static_assert(sizeof(llgo_wifi_sta_t) == 12, "Go Station struct layout drift");
_Static_assert(offsetof(llgo_wifi_sta_t, rssi) == 6, "Go Station struct layout drift");
_Static_assert(offsetof(llgo_wifi_sta_t, ip) == 8, "Go Station struct layout drift");

/* Both lists are ESP_WIFI_MAX_CONN_NUM entries (15 on this chip, 4 on a C2);
 * they are static because the driver fills them in place. */
static wifi_sta_list_t s_sta_list;
static wifi_sta_mac_ip_list_t s_sta_ip_list;

static void lw_push(int32_t kind, int32_t aux, uint32_t ip,
                    const uint8_t *mac, uint16_t aid)
{
    uint32_t w = s_wr;                       /* producer-owned */
    if (w - LW_ACQUIRE(s_rd) >= LWQ_SLOTS) {
        /* plain RMW on a variable the consumer reads atomically would be a
         * C11 data race; store the incremented value atomically instead
         * (single writer, no RMW primitive, nothing pulled from libatomic). */
        __atomic_store_n(&s_dropped, s_dropped + 1, __ATOMIC_RELAXED);
        return;                /* full: drop the new event, count it */
    }
    lw_slot_t *slot = &s_q[w & (LWQ_SLOTS - 1)];
    slot->kind = kind;
    slot->aux = aux;
    slot->ip = ip;
    slot->aid = aid;
    if (mac) {
        memcpy(slot->mac, mac, 6);
    } else {
        memset(slot->mac, 0, 6);
    }
    LW_PUBLISH(s_wr, w + 1);                 /* release: the slot is now visible */
}

static void lw_on_event(void *arg, esp_event_base_t base, int32_t id, void *data)
{
    (void)arg;
    if (base == WIFI_EVENT) {
        switch (id) {
        case WIFI_EVENT_STA_START:     lw_push(EV_STA_START, 0, 0, NULL, 0); break;
        case WIFI_EVENT_STA_CONNECTED: lw_push(EV_STA_CONNECTED, 0, 0, NULL, 0); break;
        case WIFI_EVENT_STA_DISCONNECTED:
            lw_push(EV_STA_DISCONNECTED,
                    ((const wifi_event_sta_disconnected_t *)data)->reason, 0, NULL, 0);
            break;
        case WIFI_EVENT_SCAN_DONE:     lw_push(EV_SCAN_DONE, 0, 0, NULL, 0); break;
        case WIFI_EVENT_AP_START:      lw_push(EV_AP_START, 0, 0, NULL, 0); break;
        case WIFI_EVENT_AP_STOP:       lw_push(EV_AP_STOP, 0, 0, NULL, 0); break;
        case WIFI_EVENT_AP_STACONNECTED: {
            const wifi_event_ap_staconnected_t *e = data;
            lw_push(EV_AP_STA_CONNECTED, 0, 0, e->mac, e->aid);
            break;
        }
        case WIFI_EVENT_AP_STADISCONNECTED: {
            const wifi_event_ap_stadisconnected_t *e = data;
            lw_push(EV_AP_STA_DISCONNECTED, e->reason, 0, e->mac, e->aid);
            break;
        }
        default: break;
        }
    } else if (base == IP_EVENT && id == IP_EVENT_STA_GOT_IP) {
        lw_push(EV_GOT_IP, 0, ((const ip_event_got_ip_t *)data)->ip_info.ip.addr, NULL, 0);
    } else if (base == IP_EVENT && id == IP_EVENT_STA_LOST_IP) {
        lw_push(EV_LOST_IP, 0, 0, NULL, 0);
    } else if (base == IP_EVENT && id == IP_EVENT_ASSIGNED_IP_TO_CLIENT) {
        /* The soft AP's DHCP server leased an address. IDF reports this on
         * IP_EVENT (the id the pre-5.2 stack called
         * WIFI_EVENT_AP_STAIPASSIGNED still aliases to it), so the AP side of
         * the mailbox learns about DHCP leases without a WIFI_EVENT case. */
        const ip_event_assigned_ip_to_client_t *e = data;
        lw_push(EV_AP_STA_IPASSIGNED, 0, e->ip.addr, e->mac, 0);
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
    /* The AP netif is NOT created here: it carries the soft-AP DHCP server
     * and most of AP mode's fixed RAM, and a station-only program should not
     * pay for it. llgo_wifi_set_ap creates it on first use. */
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
     * subscriptions symmetrical (and LOST_IP actually reachable). The AP side
     * needs the same base: IP_EVENT_ASSIGNED_IP_TO_CLIENT is dispatched by id
     * there. */
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

int32_t llgo_wifi_stop(void)
{
    return esp_wifi_stop();
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

/* Soft-AP IP in network byte order. Unlike the station read this one reports
 * why there is no address: the AP netif only exists after llgo_wifi_set_ap,
 * so a program that asks before configuring the AP gets ESP_ERR_WIFI_MODE
 * (0x3005) instead of a silent 0. */
int32_t llgo_wifi_get_ap_ip(uint32_t *ip)
{
    esp_netif_ip_info_t info;
    memset(&info, 0, sizeof info);
    if (s_ap_netif == NULL) return ESP_ERR_WIFI_MODE;
    esp_err_t e = esp_netif_get_ip_info(s_ap_netif, &info);
    if (e != ESP_OK) return e;
    *ip = info.ip.addr;
    return ESP_OK;
}

/* Pick the driver mode explicitly (1 STA, 2 AP, 3 APSTA: IDF wifi_mode_t).
 * After this call SetSTA/SetAP stop touching the mode, which is what lets an
 * APSTA program configure both interfaces in either order. */
int32_t llgo_wifi_set_mode(int32_t mode)
{
    esp_err_t e = esp_wifi_set_mode((wifi_mode_t)mode);
    if (e == ESP_OK) {
        s_mode = (wifi_mode_t)mode;
        s_mode_explicit = true;
    }
    return e;
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

    bool was_configured = s_sta_configured;
    esp_err_t e = ESP_OK;
    if (!s_mode_explicit) {
        /* Add this interface to the mode union: STA alone, or APSTA when an
         * AP was configured first. */
        e = lw_apply_mode(true, s_ap_configured);
        if (e != ESP_OK) return e;
    }
    e = esp_wifi_set_config(WIFI_IF_STA, &c);
    if (e == ESP_OK) e = esp_wifi_set_storage(WIFI_STORAGE_RAM);
    if (e != ESP_OK) {
        /* Leave the caller's state as it was: the interface flag and, if this
         * call widened the mode for an interface that was not configured
         * before, the mode. */
        if (!s_mode_explicit && !was_configured) {
            lw_apply_mode(false, s_ap_configured);
        }
        return e;
    }
    s_sta_configured = true;
    return ESP_OK;
}

/* Create the default AP netif once. esp_netif_create_default_wifi_ap() gives
 * the interface its static 192.168.4.1 address and attaches the soft-AP DHCP
 * server (CONFIG_LWIP_DHCPS); the server starts leasing when the interface
 * comes up on WIFI_EVENT_AP_START. Called from llgo_wifi_set_ap, so the AP
 * netif's RAM shows up between a program's two heap marks around that call. */
static esp_err_t lw_ensure_ap_netif(void)
{
    if (s_ap_netif != NULL) return ESP_OK;
    s_ap_netif = esp_netif_create_default_wifi_ap();
    return (s_ap_netif != NULL) ? ESP_OK : ESP_FAIL;
}

/* Soft-AP configuration. channel 0 was already resolved to 1 by the Go side;
 * we validate the ranges the driver would reject anyway so the error codes
 * are the package's (ESP_ERR_WIFI_SSID / ESP_ERR_WIFI_PASSWORD / ...) rather
 * than whichever check fires first in the driver. */
int32_t llgo_wifi_set_ap(const uint8_t *ssid, int32_t ssid_len,
                         const uint8_t *pass, int32_t pass_len,
                         int32_t channel, int32_t hidden, int32_t max_sta,
                         int32_t auth_mode)
{
    if (ssid_len > 32) return ESP_ERR_INVALID_ARG;   /* does not fit ssid[32] */
    if (ssid_len <= 0) return ESP_ERR_WIFI_SSID;     /* the driver refuses an empty SSID */
    if (pass_len < 0 || pass_len > 63) return ESP_ERR_INVALID_ARG;  /* does not fit ssid/pass */
    if (pass_len != 0 && pass_len < 8) return ESP_ERR_WIFI_PASSWORD;
    if (channel < 1 || channel > 14) return ESP_ERR_INVALID_ARG;
    if (max_sta < 1 || max_sta > ESP_WIFI_MAX_CONN_NUM) return ESP_ERR_INVALID_ARG;
    switch (auth_mode) {
    case WIFI_AUTH_OPEN:
        if (pass_len != 0) return ESP_ERR_WIFI_PASSWORD;
        break;
    case WIFI_AUTH_WPA2_PSK:
    case WIFI_AUTH_WPA3_PSK:
    case WIFI_AUTH_WPA2_WPA3_PSK:
        if (pass_len < 8) return ESP_ERR_WIFI_PASSWORD;
        break;
    default:
        return ESP_ERR_INVALID_ARG;
    }

    wifi_config_t c;
    memset(&c, 0, sizeof c);
    memcpy(c.ap.ssid, ssid, ssid_len);
    c.ap.ssid_len = (uint8_t)ssid_len;
    memcpy(c.ap.password, pass, pass_len);
    c.ap.channel = (uint8_t)channel;
    c.ap.authmode = (wifi_auth_mode_t)auth_mode;
    c.ap.ssid_hidden = (uint8_t)(hidden != 0);
    c.ap.max_connection = (uint8_t)max_sta;

    bool was_configured = s_ap_configured;
    esp_err_t e = ESP_OK;
    if (!s_mode_explicit) {
        /* Add this interface to the mode union: AP alone, or APSTA when a
         * station was configured first. */
        e = lw_apply_mode(s_sta_configured, true);
        if (e != ESP_OK) return e;
    }
    /* Order matters for the failure contract: the driver takes the config
     * first (it checks that the chosen mode has an AP interface and reports
     * ESP_ERR_WIFI_MODE when an explicit SetModeSTA forbids it), and only a
     * config the driver accepted gets an AP netif. A failed call therefore
     * leaves no AP interface behind, which is what lets GetAPIP, APStations
     * and GetAPConfig keep reporting ESP_ERR_WIFI_MODE for "no AP interface"
     * instead of handing out an address nobody configured. */
    e = esp_wifi_set_storage(WIFI_STORAGE_RAM);
    if (e == ESP_OK) e = esp_wifi_set_config(WIFI_IF_AP, &c);
    if (e == ESP_OK) e = lw_ensure_ap_netif();
    if (e != ESP_OK) {
        /* Roll the implicit mode change back for an interface that was not
         * configured before this call. A netif is only created once the
         * config was accepted, so there is nothing to destroy; if the config
         * itself was accepted but the netif creation failed, the config
         * stays in the driver where nothing reads it - GetAPIP, APStations
         * and GetAPConfig all answer ESP_ERR_WIFI_MODE while s_ap_netif is
         * NULL, and a retry of SetAP re-applies the rolled-back mode. */
        if (!s_mode_explicit && !was_configured) {
            lw_apply_mode(s_sta_configured, false);
        }
        return e;
    }
    s_ap_configured = true;
    return ESP_OK;
}

/* Read the soft-AP config back out of the driver. The password is never
 * marshalled: a readback exists to prove channel/authmode/limits took effect,
 * and the passphrase would then travel back over the boundary for nothing. */
int32_t llgo_wifi_get_ap_config(uint8_t *ssid, int32_t ssid_max, int32_t *ssid_len,
                               int32_t *channel, int32_t *hidden,
                               int32_t *max_sta, int32_t *auth_mode)
{
    if (s_ap_netif == NULL) return ESP_ERR_WIFI_MODE;
    wifi_config_t c;
    memset(&c, 0, sizeof c);
    esp_err_t e = esp_wifi_get_config(WIFI_IF_AP, &c);
    if (e != ESP_OK) return e;
    int32_t n = c.ap.ssid_len;
    if (n <= 0) n = (int32_t)strnlen((const char *)c.ap.ssid, sizeof c.ap.ssid);
    if (n > ssid_max) n = ssid_max;
    memcpy(ssid, c.ap.ssid, (size_t)n);
    *ssid_len = n;
    *channel = c.ap.channel;
    *hidden = c.ap.ssid_hidden;
    *max_sta = c.ap.max_connection;
    *auth_mode = (int32_t)c.ap.authmode;
    return ESP_OK;
}

/* Soft-AP station list: MAC and RSSI from the driver, plus each station's
 * leased address from the AP's own DHCP server (esp_wifi_ap_get_sta_list_with_ip
 * reads the DHCP lease table, falling back to the ARP cache). Go never speaks
 * DHCP; this is a read of the records IDF keeps. */
int32_t llgo_wifi_ap_sta_list(llgo_wifi_sta_t *out, int32_t max, int32_t *count)
{
    if (s_ap_netif == NULL) return ESP_ERR_WIFI_MODE;
    if (max <= 0) { *count = 0; return ESP_OK; }
    if (max > ESP_WIFI_MAX_CONN_NUM) max = ESP_WIFI_MAX_CONN_NUM;

    esp_err_t e = esp_wifi_ap_get_sta_list(&s_sta_list);
    if (e != ESP_OK) return e;
    int n = s_sta_list.num;
    if (n < 0) n = 0;
    if (n > max) n = max;

    /* A failing lease lookup is not an error for the caller - the stations
     * are associated, their addresses just are not known yet (or DHCPS is
     * compiled out), so the IP stays 0 and APStations still reports them. */
    memset(&s_sta_ip_list, 0, sizeof s_sta_ip_list);
    bool have_ip = esp_wifi_ap_get_sta_list_with_ip(&s_sta_list, &s_sta_ip_list) == ESP_OK;

    for (int i = 0; i < n; i++) {
        memcpy(out[i].mac, s_sta_list.sta[i].mac, 6);
        out[i].rssi = s_sta_list.sta[i].rssi;
        out[i].ip = 0;
        if (!have_ip) continue;
        for (int j = 0; j < s_sta_ip_list.num; j++) {
            if (memcmp(s_sta_ip_list.sta[j].mac, out[i].mac, 6) == 0) {
                out[i].ip = s_sta_ip_list.sta[j].ip.addr;
                break;
            }
        }
    }
    *count = n;
    return ESP_OK;
}

/* Deauthenticate one station by MAC. esp_wifi_deauth_sta() takes an AID and
 * the station list carries none, so the AID comes from the driver itself
 * (esp_wifi_ap_get_sta_aid): no bookkeeping here, and a MAC that is not
 * associated makes the driver report ESP_ERR_NOT_FOUND instead of the glue
 * guessing an AID. */
int32_t llgo_wifi_deauth_sta(const uint8_t *mac)
{
    if (s_ap_netif == NULL) return ESP_ERR_WIFI_MODE;
    uint16_t aid = 0;
    esp_err_t e = esp_wifi_ap_get_sta_aid(mac, &aid);
    if (e != ESP_OK) return e;
    return esp_wifi_deauth_sta(aid);
}

/* ---- station modem power save ----
 *
 * The 802.11 power-save knob lives on the station interface: the driver
 * itself accepts esp_wifi_set_ps in any mode - its net80211 entry point
 * validates only the 0..2 enum range and the processing callback always
 * reports success - and the setting simply never engages without a station
 * association. That silent swallow is exactly what a caller in AP-only mode
 * should not get, so both entry points refuse it here first: s_mode mirrors
 * what this glue last wrote to esp_wifi_set_mode (it is the only writer),
 * and a mode without the station bit reports ESP_ERR_WIFI_MODE, the same
 * "this interface does not exist" code the other interface-bound queries
 * (llgo_wifi_get_ap_config, llgo_wifi_get_ap_ip, llgo_wifi_deauth_sta)
 * report for a missing AP. */
int32_t llgo_wifi_set_ps(int32_t ps)
{
    if (ps < WIFI_PS_NONE || ps > WIFI_PS_MAX_MODEM) return ESP_ERR_INVALID_ARG;
    if ((s_mode & WIFI_MODE_STA) == 0) return ESP_ERR_WIFI_MODE;
    return esp_wifi_set_ps((wifi_ps_type_t)ps);
}

int32_t llgo_wifi_get_ps(int32_t *ps)
{
    if ((s_mode & WIFI_MODE_STA) == 0) return ESP_ERR_WIFI_MODE;
    wifi_ps_type_t type;
    esp_err_t e = esp_wifi_get_ps(&type);
    if (e != ESP_OK) return e;
    *ps = (int32_t)type;
    return ESP_OK;
}

/* Soft-AP station capacity of this chip (ESP_WIFI_MAX_CONN_NUM: 15 on the
 * S31, 4 on a C2). The Go side sizes its station slice from this instead of
 * assuming the S31 number, and a MaxSTA above it is rejected below. */
int32_t llgo_wifi_max_sta(void)
{
    return ESP_WIFI_MAX_CONN_NUM;
}

/* Contract version of this glue, checked by the package's Setup before any
 * other call.
 *
 * Version 2 is the current contract: llgo_wifi_poll() reports the soft-AP
 * client's MAC and AID in its last two out-parameters, the mailbox carries
 * them, and the soft-AP entry points (mode, set/get config, AP address,
 * station list, deauth, stop, capacity) plus the station power-save pair
 * (set/get ps) exist. Projects copy this file as a
 * template, so a stale copy is a real failure mode - and because the poll
 * symbol keeps its name across versions, such a copy would silently report
 * zeroed MAC/AID rather than fail the link. The Go side declares the same
 * number and refuses to run against anything else. */
int32_t llgo_wifi_glue_abi(void)
{
    return 2;
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
 * events were ever dropped (reported via kind = -1). mac/aid are filled for
 * the soft-AP station kinds and zeroed otherwise. */
int32_t llgo_wifi_poll(int32_t *kind, int32_t *aux, uint32_t *ip,
                       uint8_t *mac, uint16_t *aid)
{
    uint32_t r = s_rd;                       /* consumer-owned */
    if (LW_ACQUIRE(s_wr) != r) {
        lw_slot_t s = s_q[r & (LWQ_SLOTS - 1)];
        LW_PUBLISH(s_rd, r + 1);             /* release: the slot is free again */
        *kind = s.kind; *aux = s.aux; *ip = s.ip; *aid = s.aid;
        memcpy(mac, s.mac, 6);
        return 1;
    }
    /* The overflow report is edge-triggered on the consumer's own snapshot, so
     * the producer never has to clear (and no RMW is needed). */
    uint32_t dropped = LW_ACQUIRE(s_dropped);
    if (dropped != s_dropped_seen) {
        s_dropped_seen = dropped;
        *kind = -1; *aux = 0; *ip = 0; *aid = 0;
        memset(mac, 0, 6);
        return 1;
    }
    return 0;
}

#else /* !ESP_PLATFORM */

/* Template guard: outside an ESP-IDF build this file compiles to an
 * otherwise empty translation unit. */
typedef int llgo_wifi_glue_not_in_idf_build;

#endif /* ESP_PLATFORM */
