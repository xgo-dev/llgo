/* Thin C glue between llgo-compiled Go code and the ESP-IDF esp_now stack.
 *
 * This file is a TEMPLATE: copy it into your project's main component so the
 * IDF build compiles it there (the llgo package never builds C code). Two
 * integration recipes - copy-into-main or CMake-reference - and the boundary
 * rules this glue implements are documented in pkg/esp/README.md.
 *
 * Rules that keep the llgo fiber runtime safe on the other side of the
 * boundary:
 *   (1) integer-and-pointer-only signatures - no float values cross the
 *       Go<->C link (see the wifi glue's header; the artifact pipeline
 *       patches the relocatable's float ABI because the boundary stays
 *       integer-only);
 *   (2) no large stack buffers. Go goroutines run on 2 KiB fiber stacks and
 *       glue runs on the 8 KiB main-task stack; bulk data (ESP-NOW payloads,
 *       up to ESP_NOW_MAX_DATA_LEN bytes) lives in the static ring below;
 *   (3) no allocation, ever. The esp_now callbacks fire on the WiFi task -
 *       outside the llgo fiber scheduler, whose --wrap redirects malloc/free
 *       to the lock-free tinygogc allocator that only the llgo task may
 *       enter. Both callbacks only memcpy into preallocated static storage.
 *   (4) callbacks never call back into Go. The WiFi task and the llgo fiber
 *       scheduler are single-task global state - an entry from a second
 *       FreeRTOS task would corrupt it. The callbacks push into the ring;
 *       Go polls it paced by vTaskDelay(10) = 100 ms, which also keeps the
 *       idle tasks fed for the watchdog.
 *
 * The WiFi driver must already be initialized and started when
 * llgo_espnow_init runs (esp_now_init requires it; pkg/esp/wifi owns the
 * radio's lifecycle - see the espnow package docs for the ordering
 * contract).
 *
 * Outside an ESP-IDF build the whole body compiles away, so stray globbing
 * (hosted builds, IDE tasks, over-broad CMake file lists) stays harmless.
 */

#if defined(ESP_PLATFORM)

#include <stddef.h>
#include <string.h>

#include "esp_wifi.h"
#include "esp_now.h"
#include "esp_mac.h"

/* ---- event ring (single producer: the WiFi task; single consumer: the llgo pump) ---- */

enum { EN_EV_RECV = 1, EN_EV_SEND_DONE = 2 };

/* One slot carries a whole record: the kind, the MAC (source for recv,
 * destination for send-done), the record's scalar (the frame RSSI for recv,
 * the esp_now_send_status_t for send-done) and the payload (recv only -
 * at most ESP_NOW_MAX_DATA_LEN bytes, the v1.0 frame ceiling this binding
 * sends and receives).
 *
 * Layout must stay identical to the Go Event + staging hand-off asserted by
 * the host tests: poll() marshals kind/mac/aux/len into the Go record and
 * memcpy's the payload into the Go staging buffer. */
#define EN_SLOTS 16
typedef struct {
    int32_t kind;
    int32_t aux;
    uint8_t mac[6];
    int32_t len;
    uint8_t data[ESP_NOW_MAX_DATA_LEN];
} en_slot_t;

static en_slot_t s_q[EN_SLOTS];

/* Single producer (WiFi task) / single consumer (the llgo pump goroutine's
 * glue call). The producer fills a slot and then publishes s_wr with a
 * release store; the consumer acquires s_wr before reading the slot and
 * publishes s_rd with a release store after copying it out. That ordering -
 * not atomicity of the slot fields - is what makes the record cross intact:
 * every field is written before the index that exposes it and read only
 * after the index that exposes it.
 *
 * Only plain stores and fences are used, no read-modify-write: RV32IMC has
 * no A extension, so an RMW would drag in libatomic. Both counters are
 * single-writer, so nothing needs it - s_dropped is written by the producer
 * alone, and the consumer keeps its own last-seen snapshot instead of
 * clearing the producer's counter. */
static uint32_t s_wr, s_rd;          /* s_wr: producer-owned, s_rd: consumer-owned */
static uint32_t s_dropped;           /* producer-owned */
static uint32_t s_dropped_seen;      /* consumer-owned */

#define EN_PUBLISH(v, val) __atomic_store_n(&(v), (val), __ATOMIC_RELEASE)
#define EN_ACQUIRE(v)      __atomic_load_n(&(v), __ATOMIC_ACQUIRE)

static void en_push(int32_t kind, int32_t aux, const uint8_t *mac,
                    const uint8_t *data, int32_t len)
{
    uint32_t w = s_wr;                       /* producer-owned */
    if (w - EN_ACQUIRE(s_rd) >= EN_SLOTS) {
        /* plain RMW on a variable the consumer reads atomically would be a
         * C11 data race; store the incremented value atomically instead
         * (single writer, no RMW primitive, nothing pulled from libatomic). */
        __atomic_store_n(&s_dropped, s_dropped + 1, __ATOMIC_RELAXED);
        return;                /* full: drop the new record, count it */
    }
    en_slot_t *slot = &s_q[w & (EN_SLOTS - 1)];
    slot->kind = kind;
    slot->aux = aux;
    if (mac) {
        memcpy(slot->mac, mac, 6);
    } else {
        memset(slot->mac, 0, 6);
    }
    slot->len = 0;
    if (data && len > 0) {
        if (len > ESP_NOW_MAX_DATA_LEN) len = ESP_NOW_MAX_DATA_LEN;
        memcpy(slot->data, data, (size_t)len);
        slot->len = len;
    }
    EN_PUBLISH(s_wr, w + 1);                 /* release: the slot is now visible */
}

/* Receive callback, WiFi task context. esp_now_recv_info_t is a local
 * variable - everything it points to is only valid during the call, so the
 * copy below is the whole transfer. The RSSI rides in the rx control
 * block; a NULL rx_ctrl reports 0 rather than dereferencing. */
static void en_on_recv(const esp_now_recv_info_t *info, const uint8_t *data, int data_len)
{
    int32_t rssi = 0;
    if (info && info->rx_ctrl) {
        rssi = (int32_t)((const wifi_pkt_rx_ctrl_t *)info->rx_ctrl)->rssi;
    }
    en_push(EN_EV_RECV, rssi, info ? info->src_addr : NULL,
            data, (int32_t)data_len);
}

/* Send callback, WiFi task context. esp_now_send_info_t (wifi_tx_info_t)
 * carries the destination MAC; status is ESP_NOW_SEND_SUCCESS (0) or
 * ESP_NOW_SEND_FAIL (1). The driver gives no sequence id - only the
 * destination MAC - which is why the Go side pairs results oldest-first
 * per MAC. */
static void en_on_send(const esp_now_send_info_t *tx_info, esp_now_send_status_t status)
{
    en_push(EN_EV_SEND_DONE, (int32_t)status,
            tx_info ? tx_info->des_addr : NULL, NULL, 0);
}

/* Contract version of this glue, checked by the package's Init before any
 * other call. The Go side declares the same number and refuses to run
 * against anything else - a stale template copy would otherwise keep every
 * symbol's name and silently answer with zeros (see the wifi glue's
 * llgo_wifi_glue_abi for the full reasoning). */
int32_t llgo_espnow_glue_abi(void)
{
    return 1;
}

/* esp_now_init + callback registration in one call: the glue owns both
 * static callbacks for the lifetime of the stack, so Go never passes
 * function pointers across the boundary (and there is nothing to
 * unregister - esp_now_deinit does it). */
int32_t llgo_espnow_init(void)
{
    esp_err_t e = esp_now_init();
    if (e != ESP_OK) return e;
    if ((e = esp_now_register_recv_cb(en_on_recv)) != ESP_OK) {
        esp_now_deinit();
        return e;
    }
    if ((e = esp_now_register_send_cb(en_on_send)) != ESP_OK) {
        esp_now_deinit();
        return e;
    }
    return ESP_OK;
}

int32_t llgo_espnow_deinit(void)
{
    return esp_now_deinit();
}

/* ESP-NOW protocol version (esp_now_get_version): 1 = v1.0 (250-byte
 * frames), 2 = v2.0 (1470-byte frames; receives both). Diagnostics only. */
int32_t llgo_espnow_version(uint32_t *ver)
{
    return esp_now_get_version(ver);
}

/* esp_now_send. The Go side validated the payload (1..ESP_NOW_MAX_DATA_LEN)
 * already; the check here is defensive, so a drifted binding reports
 * ESP_ERR_ESPNOW_ARG instead of trusting a bad length into the driver. */
int32_t llgo_espnow_send(const uint8_t *mac, const uint8_t *data, int32_t len)
{
    if (data == NULL || len < 1 || len > ESP_NOW_MAX_DATA_LEN) {
        return ESP_ERR_ESPNOW_ARG;
    }
    return esp_now_send(mac, data, (size_t)len);
}

/* Add a peer: MAC, optional pinning (channel, interface), optional
 * encryption with a per-peer LMK. encrypt/lmk travel separately because the
 * IDF peer struct embeds Kconfig-sized arrays; the glue builds it. */
int32_t llgo_espnow_add_peer(const uint8_t *mac, const uint8_t *lmk,
                             int32_t channel, int32_t ifidx, int32_t encrypt)
{
    esp_now_peer_info_t peer;
    memset(&peer, 0, sizeof peer);
    memcpy(peer.peer_addr, mac, 6);
    if (lmk && encrypt) {
        memcpy(peer.lmk, lmk, ESP_NOW_KEY_LEN);
    }
    peer.channel = (uint8_t)channel;
    peer.ifidx = (wifi_interface_t)ifidx;
    peer.encrypt = (encrypt != 0);
    return esp_now_add_peer(&peer);
}

int32_t llgo_espnow_del_peer(const uint8_t *mac)
{
    return esp_now_del_peer(mac);
}

int32_t llgo_espnow_peer_exists(const uint8_t *mac)
{
    return esp_now_is_peer_exist(mac) ? 1 : 0;
}

int32_t llgo_espnow_peer_num(int32_t *total, int32_t *encrypted)
{
    esp_now_peer_num_t num;
    memset(&num, 0, sizeof num);
    esp_err_t e = esp_now_get_peer_num(&num);
    if (e != ESP_OK) return e;
    *total = (int32_t)num.total_num;
    *encrypted = (int32_t)num.encrypt_num;
    return ESP_OK;
}

int32_t llgo_espnow_set_pmk(const uint8_t *key)
{
    return esp_now_set_pmk(key);
}

/* Replace an existing peer's LMK. IDF 6.x has no esp_now_set_peer_key
 * anymore - the key travels only inside the peer struct - so the update is
 * read-modify-write: esp_now_get_peer (NOT_FOUND for an unregistered MAC,
 * the error the Go docs promise), swap the key, esp_now_mod_peer. */
int32_t llgo_espnow_set_peer_key(const uint8_t *mac, const uint8_t *key)
{
    esp_now_peer_info_t peer;
    memset(&peer, 0, sizeof peer);
    esp_err_t e = esp_now_get_peer(mac, &peer);
    if (e != ESP_OK) return e;
    memcpy(peer.lmk, key, ESP_NOW_KEY_LEN);
    return esp_now_mod_peer(&peer);
}

/* Move the radio to a primary channel (esp_wifi_set_channel, secondary
 * pinned to NONE). A connected station refuses (association owns the
 * channel); two ESP-NOW nodes that share no AP both call this with the
 * same channel while unconnected. */
int32_t llgo_espnow_set_channel(int32_t channel)
{
    return esp_wifi_set_channel((uint8_t)channel, WIFI_SECOND_CHAN_NONE);
}

/* This device's MAC for one interface (esp_wifi_get_mac): the address a
 * peer must add to reach it. Zero-filled when the driver has no such
 * interface. */
void llgo_espnow_own_mac(uint8_t *mac, int32_t ifidx)
{
    if (esp_wifi_get_mac((wifi_interface_t)ifidx, mac) != ESP_OK) {
        memset(mac, 0, 6);
    }
}

/* Pop one ring record into the Go hand-off: 1 = record delivered, 0 =
 * empty, one EN_EV_OVERFLOW slot first if records were ever dropped
 * (kind = -1; the Go pump turns it into its Dropped counter). kind/aux/mac/
 * len marshal into the Go Event record, the payload into the Go staging
 * buffer - the same split the host tests pin. */
#define EN_EV_OVERFLOW (-1)
int32_t llgo_espnow_poll(int32_t *kind, uint8_t *mac, int32_t *aux,
                         uint8_t *data, int32_t *len)
{
    uint32_t r = s_rd;                       /* consumer-owned */
    if (EN_ACQUIRE(s_wr) != r) {
        en_slot_t s = s_q[r & (EN_SLOTS - 1)];
        EN_PUBLISH(s_rd, r + 1);             /* release: the slot is free again */
        *kind = s.kind; *aux = s.aux; *len = s.len;
        memcpy(mac, s.mac, 6);
        if (s.len > 0) memcpy(data, s.data, (size_t)s.len);
        return 1;
    }
    /* The overflow report is edge-triggered on the consumer's own snapshot,
     * so the producer never has to clear (and no RMW is needed). */
    uint32_t dropped = EN_ACQUIRE(s_dropped);
    if (dropped != s_dropped_seen) {
        s_dropped_seen = dropped;
        *kind = EN_EV_OVERFLOW; *aux = 0; *len = 0;
        memset(mac, 0, 6);
        return 1;
    }
    return 0;
}

#else /* !ESP_PLATFORM */

/* Template guard: outside an ESP-IDF build this file compiles to an
 * otherwise empty translation unit. */
typedef int llgo_espnow_glue_not_in_idf_build;

#endif /* ESP_PLATFORM */
