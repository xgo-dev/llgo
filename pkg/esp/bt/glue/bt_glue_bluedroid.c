/* Thin C glue between llgo-compiled Go code and the ESP-IDF Bluedroid
 * classic Bluetooth stack (SPP server data channel + lifecycle events).
 *
 * This file is a TEMPLATE: copy it into your project's main component so the
 * IDF build compiles it there (the llgo package never builds C code). Two
 * integration recipes - copy-into-main or CMake-reference - and the boundary
 * rules this glue implements are documented in pkg/esp/README.md.
 *
 * Rules that keep the llgo fiber runtime safe on the other side of the
 * boundary:
 *   (1) integer-and-pointer-only signatures - no float values cross the
 *       Go<->C link (see the _demo/esp-idf-bt host/make-artifact.sh
 *       float-ABI step);
 *   (2) no large stack buffers: Go goroutines run on small fiber stacks, and
 *       the BTC task's callback stack is IDF's own - bulk data (received SPP
 *       payloads) lives in the static slot ring below, never on either stack;
 *   (3) BT_CONTROLLER_INIT_CONFIG_DEFAULT() and the esp_spp_cfg_t record stay
 *       here: the macro embeds dozens of Kconfig constants and the cfg embeds
 *       IDF enums that Go cannot spell;
 *   (4) events never call back into Go. Bluedroid profile callbacks run on
 *       the BTC task and the llgo fiber scheduler is single-task global
 *       state - an entry from a second FreeRTOS task would corrupt it. The
 *       callbacks only copy into the mailboxes below; Go polls the ctrl ring
 *       paced by vTaskDelay(1) and pulls data records with llgo_bt_spp_read.
 *
 * Mailbox layout (the two rings are independent, so a ctrl flood never
 * back-pressures data and vice versa):
 *
 *   ctrl ring   8 slots x {kind, status, handle, aux, addr[6]} - lifecycle
 *               events, SPSC: the BTC task pushes (s_ctrl_head), the Go
 *               poller pops (s_ctrl_tail), each side touches only its own
 *               index; drop counter is producer-owned.
 *   data ring   8 slots x {handle, len, used, data[1024]} - received SPP
 *               records; the BTC task fills free slots (s_data_alloc hint is
 *               producer-owned), the consumer clears the used flag after
 *               copying out. `used` is the only flag two sides touch: the
 *               producer stores 1 after the payload, the consumer stores 0
 *               after the copy - release/acquire orders make that a lock-free
 *               SPSC hand-off per slot. An overflowing ring increments the
 *               drop counter; the loss is reported once through the ctrl
 *               ring's EvOverflow edge (counters are monotonic, the Go poller
 *               keeps its seen-snapshot).
 *
 * No C->Go trampoline exists anywhere in this file, and no RV32/RISC-V
 * read-modify-write on shared flags: every shared word is written by exactly
 * one side (single-writer counters, per-slot flag hand-off), the only
 * cross-checkable state.
 *
 * Outside an ESP-IDF build the whole body compiles away, so stray globbing
 * (hosted builds, IDE tasks, over-broad CMake file lists) stays harmless.
 */

#if defined(ESP_PLATFORM)

#include <stddef.h>
#include <stdint.h>
#include <string.h>

#include "esp_bt.h"
#include "esp_bt_main.h"
#include "esp_gap_bt_api.h"
#include "esp_spp_api.h"
#include "esp_timer.h"
#include "nvs_flash.h"
#include "freertos/FreeRTOS.h"
#include "freertos/semphr.h"

#if !CONFIG_BT_BLUEDROID_ENABLED || !CONFIG_BT_CLASSIC_ENABLED || \
    !CONFIG_BT_SPP_ENABLED
#error "llgo bt glue needs classic Bluetooth: set CONFIG_BT_BLUEDROID_ENABLED, \
CONFIG_BT_CLASSIC_ENABLED and CONFIG_BT_SPP_ENABLED in sdkconfig.defaults \
(see _demo/esp-idf-bt/idfproj/sdkconfig.defaults)"
#endif

/* The A2DP sister glue (bt_glue_a2dp.c, same main component) shares this
 * file's ctrl ring publisher and ready flag; its audio path joins the same
 * lifecycle - Deinit tears it down with the stack, before the host goes
 * away (see bt_teardown below). */
extern volatile int32_t s_ready;
extern int32_t bt_a2dp_teardown(void);

/* Glue contract version: the Go package checks llgo_bt_glue_abi before any
 * control call and refuses a mismatch (BT_ERR_GLUE_ABI). Bump together with
 * bt.go's glueABIVersion when a signature or a mailbox kind changes. */
#define LLGO_BT_GLUE_ABI 2

/* ---- ctrl ring ---- */

#define CTRL_SLOTS 8

typedef struct {
    int32_t kind;
    int32_t status;
    uint32_t handle;
    int32_t aux;
    uint8_t addr[6];
} ctrl_slot_t;

static ctrl_slot_t s_ctrl[CTRL_SLOTS];
static volatile uint32_t s_ctrl_head; /* producer side: BTC task only */
static volatile uint32_t s_ctrl_tail; /* consumer side: Go poller only */
static volatile uint32_t s_ctrl_drop; /* producer-owned counter */

/* ---- SPP data ring ---- */

#define SPP_DATA_SLOTS 8
#define SPP_DATA_CAP   1024

typedef struct {
    volatile uint8_t used; /* hand-off flag, see file comment */
    uint32_t handle;
    uint32_t len;
    uint8_t data[SPP_DATA_CAP];
} data_slot_t;

static data_slot_t s_data[SPP_DATA_SLOTS];
static uint32_t s_data_alloc; /* producer side: BTC task only */
static volatile uint32_t s_data_drop; /* producer-owned counter */

/* The Go poller's seen-snapshot of the drop counters (consumer side only). */
static uint32_t s_drop_seen_ctrl;
static uint32_t s_drop_seen_data;

/* ctrl_push publishes one lifecycle event. BTC task context. Non-static on
 * purpose: the a2dp sister glue (bt_glue_a2dp.c, same main component)
 * publishes its profile events through this one ring - one mailbox, one
 * poller, one set of drop counters. */
void ctrl_push(int32_t kind, int32_t status, uint32_t handle,
               int32_t aux, const uint8_t *addr)
{
    uint32_t head = s_ctrl_head;
    uint32_t next = (head + 1) % CTRL_SLOTS;
    if (next == __atomic_load_n(&s_ctrl_tail, __ATOMIC_ACQUIRE)) {
        s_ctrl_drop++; /* producer-owned; overflow edges report the count */
        return;
    }
    ctrl_slot_t *s = &s_ctrl[head];
    s->kind = kind;
    s->status = status;
    s->handle = handle;
    s->aux = aux;
    if (addr != NULL) {
        memcpy(s->addr, addr, 6);
    } else {
        memset(s->addr, 0, 6);
    }
    __atomic_store_n(&s_ctrl_head, next, __ATOMIC_RELEASE);
}

/* data_push parks one received SPP record in a free slot. BTC task context.
 * len is clamped to the slot capacity; an RFCOMM frame is at most
 * ESP_SPP_MAX_MTU (990), so the clamp never fires in practice. */
static void data_push(uint32_t handle, int len, const uint8_t *data)
{
    if (len > SPP_DATA_CAP) {
        len = SPP_DATA_CAP;
    }
    for (uint32_t i = 0; i < SPP_DATA_SLOTS; i++) {
        data_slot_t *s = &s_data[(s_data_alloc + i) % SPP_DATA_SLOTS];
        if (__atomic_load_n(&s->used, __ATOMIC_ACQUIRE)) {
            continue;
        }
        s->handle = handle;
        s->len = (uint32_t)len;
        memcpy(s->data, data, (size_t)len);
        __atomic_store_n(&s->used, 1, __ATOMIC_RELEASE);
        s_data_alloc = (s_data_alloc + i + 1) % SPP_DATA_SLOTS;
        return;
    }
    s_data_drop++; /* producer-owned; overflow edges report the count */
}

/* ---- Bluedroid callbacks (all BTC task context) ---- */

/* Profile-teardown handshake: esp_spp_deinit only POSTS the deinit to the BTC
 * task and returns, and racing it against esp_bluedroid_disable (which stops
 * the BTC/BTU threads) aborts the stack in osi_thread_post when an in-flight
 * profile message lands on an already-freed thread. The handshake serializes
 * the two: the UNINIT event (the profile's last callback) gives the
 * semaphore, llgo_bt_deinit takes it before touching the host. The wait is a
 * one-shot freeze of the llgo task inside a C call - the same class as the
 * OS-semaphore park inside esp_bluedroid_enable/disable themselves. */
static SemaphoreHandle_t s_uninit_sem;

static void spp_cb(esp_spp_cb_event_t event, esp_spp_cb_param_t *param)
{
    switch (event) {
    case ESP_SPP_INIT_EVT:
        ctrl_push(1, (int32_t)param->init.status, 0, 0, NULL);
        break;
    case ESP_SPP_START_EVT: /* server listening: scn in aux */
        ctrl_push(2, (int32_t)param->start.status, param->start.handle,
                  (int32_t)param->start.scn, NULL);
        break;
    case ESP_SPP_SRV_OPEN_EVT: /* peer connected: peer address attached */
        ctrl_push(3, (int32_t)param->srv_open.status, param->srv_open.handle,
                  0, param->srv_open.rem_bda);
        break;
    case ESP_SPP_CLOSE_EVT: /* aux 1 = peer initiated (async FALSE is local) */
        ctrl_push(4, (int32_t)param->close.status, param->close.handle,
                  param->close.async ? 1 : 0, NULL);
        break;
    case ESP_SPP_CONG_EVT: /* TX buffer drained; v6.3 reports the edge only */
        ctrl_push(5, (int32_t)param->cong.status, param->cong.handle,
                  param->cong.cong ? 1 : 0, NULL);
        break;
    case ESP_SPP_DATA_IND_EVT: /* received record -> data ring */
        data_push(param->data_ind.handle, param->data_ind.len,
                  param->data_ind.data);
        break;
    case ESP_SPP_SRV_STOP_EVT:
        ctrl_push(6, (int32_t)param->srv_stop.status, 0,
                  (int32_t)param->srv_stop.scn, NULL);
        break;
    case ESP_SPP_UNINIT_EVT:
        ctrl_push(7, (int32_t)param->uninit.status, 0, 0, NULL);
        if (s_uninit_sem != NULL) {
            xSemaphoreGive(s_uninit_sem);
        }
        break;
    case ESP_SPP_WRITE_EVT:
        /* TX chunk-drain report. Flow control goes through the esp_spp_write
         * return code plus CONG_EVT (v6.3 semantics), so this record carries
         * no information the Go side needs. */
        break;
    default:
        break;
    }
}

/* GAP callback: the ACL link accounting lands here - the link count lets
 * Deinit wait for the baseband teardown, and the disconnect reason explains
 * link loss behind an SPP close (status in the 0x100+ HCI range). Adding
 * discovery/pairing later means new cases here and new kinds in bt.go,
 * nothing else. */
static volatile int32_t s_acl_links;

static void gap_cb(esp_bt_gap_cb_event_t event, esp_bt_gap_cb_param_t *param)
{
    switch (event) {
    case ESP_BT_GAP_ACL_CONN_CMPL_STAT_EVT:
        s_acl_links++;
        break;
    case ESP_BT_GAP_ACL_DISCONN_CMPL_STAT_EVT:
        if (s_acl_links > 0) {
            s_acl_links--;
        }
        ctrl_push(8, (int32_t)param->acl_disconn_cmpl_stat.reason,
                  (uint32_t)param->acl_disconn_cmpl_stat.handle, 0,
                  param->acl_disconn_cmpl_stat.bda);
        break;
    default:
        break;
    }
}

/* ---- lifecycle ---- */

/* Non-static: the a2dp sister glue gates its control calls on it too. */
volatile int32_t s_ready;
static StaticSemaphore_t s_uninit_sem_buf;

/* Bring-up stage bookkeeping. Init can fail between stages, and without a
 * record of how far it got the intermediate state is unrecoverable: a retry
 * would hit INVALID_STATE on the already-initialized stage, and a deinit
 * gated on s_ready would refuse to clean up. Every stage marks itself here
 * on success; a failure path (or a Deinit) then tears down exactly what
 * came up, in reverse order. */
#define BT_STAGE_OFF       0 /* nothing initialized */
#define BT_STAGE_CTRL_INIT 1 /* esp_bt_controller_init done */
#define BT_STAGE_CTRL_EN   2 /* controller enabled (BTDM) */
#define BT_STAGE_HOST_INIT 3 /* esp_bluedroid_init done */
#define BT_STAGE_READY     4 /* bluedroid enabled, GAP registered */
static volatile int32_t s_stage;
static volatile int32_t s_spp_up; /* the SPP profile came up (SPPListen) */

/* Tear down whatever came up, in reverse order. Best-effort: every step
 * runs regardless of the previous one's result, and the first error is
 * reported. Safe on any stage - including mid-bring-up failure states and
 * the full-stack state. */
static int32_t bt_teardown(void)
{
    esp_err_t err = ESP_OK, first = ESP_OK;
    if (s_stage >= BT_STAGE_HOST_INIT) {
        if (s_spp_up) {
            esp_spp_deinit(); /* posts to BTC; the handshake serializes it */
            if (s_uninit_sem != NULL &&
                xSemaphoreTake(s_uninit_sem, pdMS_TO_TICKS(8000)) != pdTRUE) {
                /* keep going: the host teardown decides if it can come
                 * down cleanly without the confirmation */
            }
            s_spp_up = 0;
        }
        if ((err = bt_a2dp_teardown()) != ESP_OK && first == ESP_OK) {
            first = err; /* wedged pump: report, and let the caller retry
                          * Deinit - the channel and pump stay alive */
        }
        /* Bluedroid auto-closes an idle ACL about five seconds after its
         * last profile drops, and a disconnect completion racing
         * esp_bluedroid_disable aborts the stack (the event posts to an
         * already-freed host thread). Wait the link count to zero out
         * before stopping the host, so the teardown never overlaps an
         * in-flight baseband event. */
        for (int32_t waited = 0; s_acl_links > 0 && waited < 8000;
             waited += 50) {
            vTaskDelay(pdMS_TO_TICKS(50));
        }
        if ((err = esp_bluedroid_disable()) != ESP_OK) {
            first = err;
        }
        if ((err = esp_bluedroid_deinit()) != ESP_OK && first == ESP_OK) {
            first = err;
        }
    }
    if (s_stage >= BT_STAGE_CTRL_EN) {
        if ((err = esp_bt_controller_disable()) != ESP_OK && first == ESP_OK) {
            first = err;
        }
    }
    if (s_stage >= BT_STAGE_CTRL_INIT) {
        if ((err = esp_bt_controller_deinit()) != ESP_OK && first == ESP_OK) {
            first = err;
        }
    }
    s_stage = BT_STAGE_OFF;
    s_ready = 0;
    s_acl_links = 0;
    /* The BTC task is gone by the time bluedroid_deinit returns (when the
     * host ever came up), so nothing can race this reset: a following Init
     * starts with empty mailboxes. On a pre-host failure state there was
     * never a producer either. */
    s_ctrl_head = s_ctrl_tail = 0;
    s_ctrl_drop = 0;
    s_drop_seen_ctrl = s_drop_seen_data = 0;
    s_data_drop = 0;
    for (int i = 0; i < SPP_DATA_SLOTS; i++) {
        s_data[i].used = 0;
    }
    return first;
}

int32_t llgo_bt_glue_abi(void)
{
    return LLGO_BT_GLUE_ABI;
}

int32_t llgo_bt_ready(void)
{
    return s_ready;
}

int32_t llgo_bt_init(void)
{
    if (s_ready) {
        return ESP_OK; /* idempotent: Go's Init is safe to call twice */
    }
    if (s_stage != BT_STAGE_OFF) {
        /* A previous bring-up failed midway: clear it before retrying,
         * otherwise the next controller call hits INVALID_STATE forever. */
        bt_teardown();
    }
    esp_err_t err;
    /* The stack persists its config (link keys) to NVS; without the
     * partition initialized every bring-up logs a config_save failure. The
     * partition is never formatted here - erasing is an application
     * decision, not a stack bring-up's. */
    err = nvs_flash_init();
    if (err != ESP_OK && err != ESP_ERR_NVS_NO_FREE_PAGES &&
        err != ESP_ERR_NVS_NEW_VERSION_FOUND) {
        return err;
    }
    if (s_uninit_sem == NULL) {
        s_uninit_sem = xSemaphoreCreateBinaryStatic(&s_uninit_sem_buf);
    }
    /* Clear a stale give from an earlier lifecycle (the deinit path takes
     * with a bound, so a timeout leave can in principle leave one). */
    xSemaphoreTake(s_uninit_sem, 0);
    esp_bt_controller_config_t cfg = BT_CONTROLLER_INIT_CONFIG_DEFAULT();
    err = esp_bt_controller_init(&cfg);
    if (err != ESP_OK && err != ESP_ERR_INVALID_STATE) {
        return err;
    }
    s_stage = BT_STAGE_CTRL_INIT;
    /* BTDM: BR/EDR + LE. The classic profiles (SPP now, A2DP later) live on
     * the BR/EDR half; the dual mode is the S31 controller's only setting
     * that turns BR/EDR on. */
    err = esp_bt_controller_enable(ESP_BT_MODE_BTDM);
    if (err != ESP_OK && err != ESP_ERR_INVALID_STATE) {
        bt_teardown(); /* leave a clean OFF state behind the failure */
        return err;
    }
    s_stage = BT_STAGE_CTRL_EN;
    err = esp_bluedroid_init();
    if (err != ESP_OK) {
        bt_teardown();
        return err;
    }
    s_stage = BT_STAGE_HOST_INIT;
    err = esp_bluedroid_enable();
    if (err != ESP_OK) {
        bt_teardown();
        return err;
    }
    /* esp_bt_gap_register_callback requires an enabled stack, and the ACL
     * disconnect source should exist before any profile comes up. */
    err = esp_bt_gap_register_callback(gap_cb);
    if (err != ESP_OK) {
        bt_teardown();
        return err;
    }
    s_ready = 1;
    s_stage = BT_STAGE_READY;
    s_acl_links = 0;
    return ESP_OK;
}

int32_t llgo_bt_deinit(void)
{
    /* Works on any bring-up stage: a full stack tears down in IDF's own
     * order (see bt_teardown), and a failed bring-up's leftovers go too -
     * which is what makes a retry after an Init failure possible. */
    if (s_stage == BT_STAGE_OFF) {
        return ESP_ERR_INVALID_STATE;
    }
    return bt_teardown();
}

/* ---- GAP control ---- */

static char s_dev_name[ESP_BT_GAP_MAX_BDNAME_LEN + 1]; /* async-read storage */

int32_t llgo_bt_set_name(const uint8_t *name, int32_t len)
{
    if (!s_ready) {
        return ESP_ERR_INVALID_STATE;
    }
    if (len < 0) {
        len = 0;
    }
    if (len > ESP_BT_GAP_MAX_BDNAME_LEN) {
        len = ESP_BT_GAP_MAX_BDNAME_LEN;
    }
    memcpy(s_dev_name, name, (size_t)len);
    s_dev_name[len] = '\0';
    return esp_bt_gap_set_device_name(s_dev_name);
}

int32_t llgo_bt_set_scan_mode(int32_t discoverable)
{
    if (!s_ready) {
        return ESP_ERR_INVALID_STATE;
    }
    if (discoverable) {
        return esp_bt_gap_set_scan_mode(ESP_BT_CONNECTABLE,
                                        ESP_BT_GENERAL_DISCOVERABLE);
    }
    return esp_bt_gap_set_scan_mode(ESP_BT_NON_CONNECTABLE,
                                    ESP_BT_NON_DISCOVERABLE);
}

/* ---- SPP ---- */

int32_t llgo_bt_spp_init(void)
{
    if (!s_ready) {
        return ESP_ERR_INVALID_STATE;
    }
    esp_err_t err;
    if ((err = esp_spp_register_callback(spp_cb)) != ESP_OK) {
        return err;
    }
    esp_spp_cfg_t cfg = {
        .mode = ESP_SPP_MODE_CB,          /* records land in the data ring */
        .enable_l2cap_ertm = true,        /* v6.3 default transport */
        .tx_buffer_size = ESP_SPP_DEFAULT_TX_BUFFER_SIZE, /* 9900 B */
    };
    /* Returns immediately; the profile's INIT event confirms through the
     * ctrl ring and SPPListen waits for it in Go. From here on a teardown
     * has to bring the profile down as well. */
    err = esp_spp_enhanced_init(&cfg);
    if (err == ESP_OK) {
        s_spp_up = 1;
    }
    return err;
}

static char s_srv_name[65]; /* async-read storage (see llgo_bt_set_name) */

int32_t llgo_bt_spp_start_srv(int32_t scn, const uint8_t *name, int32_t len)
{
    if (!s_ready) {
        return ESP_ERR_INVALID_STATE;
    }
    if (len < 0) {
        len = 0;
    }
    if (len > 64) {
        len = 64;
    }
    memcpy(s_srv_name, name, (size_t)len);
    s_srv_name[len] = '\0';
    return esp_spp_start_srv(ESP_SPP_SEC_NONE, ESP_SPP_ROLE_SLAVE, scn,
                             s_srv_name);
}

int32_t llgo_bt_spp_disconnect(uint32_t handle)
{
    return esp_spp_disconnect(handle);
}

/* v6.3 esp_spp_write copies the whole buffer into the profile's TX
 * ringbuffer before returning, so the Go-side memory is free on return and
 * the raw return code is the flow-control answer (ESP_OK queued,
 * ESP_ERR_NO_MEM full - Go waits the CONG edge and retries). */
int32_t llgo_bt_spp_write(uint32_t handle, const uint8_t *data, int32_t len)
{
    return esp_spp_write(handle, (int)len, (uint8_t *)data);
}

/* Copies one queued record of `handle` out of the data ring: returns its
 * length (capped at cap - a smaller cap drops the remainder), 0 when no
 * record of this handle is queued. Records are scanned in slot order, which
 * is FIFO per connection: the BTC task fills slots in arrival order and
 * never reuses a slot the consumer has not cleared. */
int32_t llgo_bt_spp_read(uint32_t handle, uint8_t *dst, int32_t cap)
{
    if (cap <= 0) {
        return 0;
    }
    for (int i = 0; i < SPP_DATA_SLOTS; i++) {
        data_slot_t *s = &s_data[i];
        if (!__atomic_load_n(&s->used, __ATOMIC_ACQUIRE)) {
            continue;
        }
        if (s->handle != handle) {
            continue;
        }
        uint32_t n = s->len;
        if (n > (uint32_t)cap) {
            n = (uint32_t)cap;
        }
        memcpy(dst, s->data, n);
        __atomic_store_n(&s->used, 0, __ATOMIC_RELEASE);
        return (int32_t)n;
    }
    return 0;
}

/* ---- mailbox drain ---- */

int32_t llgo_bt_poll(int32_t *kind, int32_t *status, uint32_t *handle,
                     int32_t *aux, uint8_t *addr)
{
    /* Edge-triggered overflow report: the producer-owned counters are
     * monotonic, and the Go poller (this function's only caller) keeps its
     * seen-snapshot here, so each batch of losses collapses into exactly one
     * EvOverflow record. */
    uint32_t dc = __atomic_load_n(&s_ctrl_drop, __ATOMIC_ACQUIRE);
    uint32_t dd = __atomic_load_n(&s_data_drop, __ATOMIC_ACQUIRE);
    if (dc != s_drop_seen_ctrl || dd != s_drop_seen_data) {
        s_drop_seen_ctrl = dc;
        s_drop_seen_data = dd;
        *kind = -1;
        *status = (int32_t)dc;
        *handle = 0;
        *aux = (int32_t)dd;
        memset(addr, 0, 6);
        return 1;
    }
    uint32_t head = __atomic_load_n(&s_ctrl_head, __ATOMIC_ACQUIRE);
    if (head == s_ctrl_tail) {
        return 0;
    }
    ctrl_slot_t *s = &s_ctrl[s_ctrl_tail];
    *kind = s->kind;
    *status = s->status;
    *handle = s->handle;
    *aux = s->aux;
    memcpy(addr, s->addr, 6);
    __atomic_store_n(&s_ctrl_tail, (s_ctrl_tail + 1) % CTRL_SLOTS,
                     __ATOMIC_RELEASE);
    return 1;
}

uint32_t llgo_bt_now_ms(void)
{
    return (uint32_t)(esp_timer_get_time() / 1000LL);
}

int32_t llgo_bt_backend_name(uint8_t *buf, int32_t n)
{
    const char *name = "bluedroid";
    if (n <= 0) {
        return 0;
    }
    int32_t len = (int32_t)strlen(name);
    if (len > n - 1) {
        len = n - 1;
    }
    memcpy(buf, name, (size_t)len);
    return len;
}

#else /* !ESP_PLATFORM */

/* Hosted builds: the symbols exist (a stray glob or IDE task linking this
 * file stays happy) but every control call reports the package's unsupported
 * tag, the same value the Go host stubs carry. */
#define LLGO_BT_UNSUPPORTED 0x7F00

int32_t llgo_bt_glue_abi(void) { return 0; }
int32_t llgo_bt_ready(void) { return 0; }
int32_t llgo_bt_init(void) { return LLGO_BT_UNSUPPORTED; }
int32_t llgo_bt_deinit(void) { return LLGO_BT_UNSUPPORTED; }
int32_t llgo_bt_set_name(const uint8_t *name, int32_t len)
{
    (void)name; (void)len;
    return LLGO_BT_UNSUPPORTED;
}
int32_t llgo_bt_set_scan_mode(int32_t discoverable)
{
    (void)discoverable;
    return LLGO_BT_UNSUPPORTED;
}
int32_t llgo_bt_spp_init(void) { return LLGO_BT_UNSUPPORTED; }
int32_t llgo_bt_spp_start_srv(int32_t scn, const uint8_t *name, int32_t len)
{
    (void)scn; (void)name; (void)len;
    return LLGO_BT_UNSUPPORTED;
}
int32_t llgo_bt_spp_disconnect(uint32_t handle)
{
    (void)handle;
    return LLGO_BT_UNSUPPORTED;
}
int32_t llgo_bt_spp_write(uint32_t handle, const uint8_t *data, int32_t len)
{
    (void)handle; (void)data; (void)len;
    return LLGO_BT_UNSUPPORTED;
}
int32_t llgo_bt_spp_read(uint32_t handle, uint8_t *dst, int32_t cap)
{
    (void)handle; (void)dst; (void)cap;
    return 0;
}
int32_t llgo_bt_poll(int32_t *kind, int32_t *status, uint32_t *handle,
                     int32_t *aux, uint8_t *addr)
{
    (void)kind; (void)status; (void)handle; (void)aux; (void)addr;
    return 0;
}
uint32_t llgo_bt_now_ms(void) { return 0; }
int32_t llgo_bt_backend_name(uint8_t *buf, int32_t n)
{
    (void)buf; (void)n;
    return 0;
}

#endif /* ESP_PLATFORM */
