/* NimBLE backend of pkg/esp/ble: the C glue between llgo-compiled Go code
 * and the ESP-IDF NimBLE host:
 * a BLE central (scan / connect / disconnect) with the stack's asynchronous
 * results delivered through a bounded, two-tier mailbox.
 *
 * This file is a TEMPLATE: copy it into your project's main component so the
 * IDF build compiles it there (the llgo package never builds C code). Two
 * integration recipes - copy-into-main or CMake-reference - and the boundary
 * rules this glue implements are documented in pkg/esp/README.md.
 *
 * Rules that keep the llgo fiber runtime safe on the other side of the
 * boundary:
 *   (1) integer-and-pointer-only signatures - no float value crosses the
 *       Go<->C link (see the demo's host/make-artifact.sh step 4, which
 *       patches the relocatable's float ABI);
 *   (2) no large stack buffers. Go goroutines run on 8 KiB fiber stacks, so
 *       every record that travels - scan results, mailbox slots - lives in
 *       static storage; the flat llgo_ble_event_t is the only shape that
 *       crosses, by pointer, with scalar and fixed-size-array fields only;
 *   (3) NimBLE never calls back into Go. The host task runs the reset/sync/
 *       GAP callbacks, and the llgo fiber scheduler is single-task global
 *       state - an entry from another FreeRTOS task would corrupt it. The
 *       callbacks copy into the mailbox instead; Go polls it from its own
 *       goroutine;
 *   (4) Go never blocks in a control path. Scan/connect/disconnect are
 *       commands posted to the NimBLE host's own event queue, so the host
 *       task executes them on its own context (the pattern NimBLE itself
 *       uses for GAP procedures). The Go side only enqueues and returns.
 *
 * Mailbox design: two single-producer/single-consumer rings.
 *
 *   ctrl ring  - sync, reset, scan-done, connect, disconnect, command
 *                rejections; 8 slots, no payload beyond codes. These events
 *                are what a program's state machine depends on, so they never
 *                compete with a scan flood for space.
 *   adv ring   - scan results only; 16 slots of address + RSSI + up to 31
 *                bytes of advertising data (the legacy maximum, which is what
 *                ble_gap_disc reports).
 *
 * Overflow drops the newest record of that ring and is reported once as an
 * EV_OVERFLOW slot carrying the dropped count, so loss is never silent but
 * also never stalls the host task.
 *
 * Outside an ESP-IDF build the whole body compiles away; inside an IDF build
 * without NimBLE the file still defines every symbol, returning
 * ESP_ERR_NOT_SUPPORTED, so a program that imports the package against a
 * firmware build without BLE fails honestly instead of pretending to work.
 */

#if defined(ESP_PLATFORM)

#include <string.h>
#include <stdio.h>

#include "esp_err.h"
#include "esp_system.h"
#include "esp_timer.h"
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "nvs_flash.h"

#if defined(CONFIG_BT_ENABLED) && !defined(CONFIG_BT_NIMBLE_ENABLED)
#error "glue/ble_glue_nimble.c is the NimBLE backend, but this build enables a different BT host stack (Bluedroid?). Sync glue/ble_glue_bluedroid.c into main/ instead."
#elif defined(CONFIG_BT_ENABLED) && defined(CONFIG_BT_NIMBLE_ENABLED)

#include "nimble/nimble_port.h"
#include "nimble/nimble_port_freertos.h"
#include "nimble/nimble_npl.h"
#include "host/ble_hs.h"
#include "host/ble_hs_id.h"
#include "host/util/util.h"

/* ble_hs_evq_get() returns the host's default event queue. NimBLE declares it
 * in its private ble_hs_priv.h, which a component cannot include; the symbol
 * is public and stable, and posting to that queue is how an application task
 * hands work to the host task. */
struct ble_npl_eventq *ble_hs_evq_get(void);

/* ---- mailbox ---- */

enum {
    EV_NONE = 0,
    EV_SYNC = 1,
    EV_RESET = 2,
    EV_SCAN_DONE = 3,
    EV_CONNECT = 4,
    EV_DISCONNECT = 5,
    EV_ADV = 6,
    EV_ERROR = 7,
    EV_OVERFLOW = -1,
};

/* The one record shape that crosses into Go. Field order and widths are an
 * ABI: pkg/esp/ble's Event struct mirrors it byte for byte (asserted below). */
typedef struct {
    int32_t kind;
    uint32_t op_id;
    uint32_t conn_id;
    int32_t status;
    uint8_t addr[6];
    uint8_t addr_type;
    int8_t rssi;
    uint8_t adv_len;
    uint8_t adv[31];
} llgo_ble_event_t;

_Static_assert(sizeof(llgo_ble_event_t) == 56, "llgo_ble_event_t ABI drifted");
_Static_assert(__builtin_offsetof(llgo_ble_event_t, addr) == 16, "addr offset");
_Static_assert(__builtin_offsetof(llgo_ble_event_t, adv) == 25, "adv offset");

#define BW_CTRL_SLOTS 8
#define BW_ADV_SLOTS 16
#define BW_ADV_MAX 31

typedef struct {
    int32_t kind;
    uint32_t op_id;
    uint32_t conn_id;
    int32_t status;
} bw_ctrl_t;

typedef struct {
    uint8_t addr[6];
    uint8_t addr_type;
    int8_t rssi;
    uint8_t len;
    uint8_t data[BW_ADV_MAX];
} bw_adv_t;

static bw_ctrl_t s_ctrl[BW_CTRL_SLOTS];
static volatile uint32_t s_ctrl_wr, s_ctrl_rd, s_ctrl_dropped;

static bw_adv_t s_adv[BW_ADV_SLOTS];
static volatile uint32_t s_adv_wr, s_adv_rd, s_adv_dropped;

/* Written by the host task only (GAP callbacks and the command handler). */
static volatile int s_inited, s_synced;
static uint8_t s_own_type;
static uint32_t s_disc_op; /* op id of the running discovery, 0 = none */
static uint32_t s_conn_op; /* op id of the running connect attempt, 0 = none */

static void bw_ctrl_push(int32_t kind, uint32_t op_id, uint32_t conn_id, int32_t status)
{
    uint32_t w = s_ctrl_wr;
    if ((uint32_t)(w - s_ctrl_rd) >= BW_CTRL_SLOTS) {
        s_ctrl_dropped++;
        return;
    }
    s_ctrl[w & (BW_CTRL_SLOTS - 1)] = (bw_ctrl_t){ kind, op_id, conn_id, status };
    s_ctrl_wr = w + 1;
}

static void bw_adv_push(const uint8_t *addr, uint8_t addr_type, int8_t rssi,
                        const uint8_t *data, uint8_t len)
{
    uint32_t w = s_adv_wr;
    if ((uint32_t)(w - s_adv_rd) >= BW_ADV_SLOTS) {
        s_adv_dropped++;
        return;
    }
    if (len > BW_ADV_MAX) {
        len = BW_ADV_MAX;
    }
    bw_adv_t *slot = &s_adv[w & (BW_ADV_SLOTS - 1)];
    memcpy(slot->addr, addr, 6);
    slot->addr_type = addr_type;
    slot->rssi = rssi;
    slot->len = len;
    memcpy(slot->data, data, len);
    s_adv_wr = w + 1;
}

/* ---- connection ids: handle + generation ----
 *
 * The controller reuses HCI connection handles, so a handle alone cannot
 * identify a connection across a disconnect/reconnect cycle: a stale
 * ConnID would address the new connection. Each registration therefore takes
 * the next generation number, and the ConnID the application holds is
 * (generation << 16) | handle. A ConnID from an older generation is refused
 * instead of being applied to a different peer. */

#define BW_CONNS 4

typedef struct {
    uint8_t used;
    uint32_t gen;
    uint16_t handle;
} bw_conn_t;

static bw_conn_t s_conns[BW_CONNS];
static uint32_t s_conn_gen;

static uint32_t bw_conn_add(uint16_t handle)
{
    for (int i = 0; i < BW_CONNS; i++) {
        if (!s_conns[i].used) {
            s_conns[i].used = 1;
            s_conns[i].handle = handle;
            s_conns[i].gen = ++s_conn_gen;
            return (s_conns[i].gen << 16) | handle;
        }
    }
    return 0; /* table full: the connection stays untracked */
}

static uint32_t bw_conn_remove(uint16_t handle)
{
    for (int i = 0; i < BW_CONNS; i++) {
        if (s_conns[i].used && s_conns[i].handle == handle) {
            uint32_t id = (s_conns[i].gen << 16) | handle;
            s_conns[i].used = 0;
            return id;
        }
    }
    return handle; /* unknown handle: report it with generation 0 */
}

static int bw_conn_find(uint32_t conn_id)
{
    uint16_t handle = (uint16_t)(conn_id & 0xFFFF);
    uint32_t gen = conn_id >> 16;
    for (int i = 0; i < BW_CONNS; i++) {
        if (s_conns[i].used && s_conns[i].handle == handle && s_conns[i].gen == gen) {
            return (int)handle;
        }
    }
    return -1;
}

/* ---- NimBLE callbacks (host task context) ---- */

static void bw_on_sync(void)
{
    int rc = ble_hs_id_infer_auto(0, &s_own_type);
    s_synced = 1;
    bw_ctrl_push(EV_SYNC, 0, 0, rc);
}

static void bw_on_reset(int reason)
{
    s_synced = 0;
    bw_ctrl_push(EV_RESET, 0, 0, reason);
}

static int bw_on_gap(struct ble_gap_event *ev, void *arg)
{
    (void)arg;
    switch (ev->type) {
    case BLE_GAP_EVENT_DISC:
        bw_adv_push(ev->disc.addr.val, ev->disc.addr.type, ev->disc.rssi,
                    ev->disc.data, ev->disc.length_data);
        return 0;
    case BLE_GAP_EVENT_DISC_COMPLETE:
        if (s_disc_op != 0) {
            bw_ctrl_push(EV_SCAN_DONE, s_disc_op, 0, ev->disc_complete.reason);
            s_disc_op = 0;
        }
        return 0;
    case BLE_GAP_EVENT_CONNECT: {
        uint32_t op = s_conn_op;
        s_conn_op = 0;
        if (ev->connect.status == 0) {
            bw_ctrl_push(EV_CONNECT, op, bw_conn_add(ev->connect.conn_handle), 0);
        } else if (op != 0) {
            bw_ctrl_push(EV_CONNECT, op, 0, ev->connect.status);
        }
        /* A failure with no operation in flight is the completion of an
         * attempt the application cancelled (or one this glue never started):
         * nothing to attribute it to, and the application already knows. */
        return 0;
    }
    case BLE_GAP_EVENT_DISCONNECT:
        bw_ctrl_push(EV_DISCONNECT, 0, bw_conn_remove(ev->disconnect.conn.conn_handle),
                     ev->disconnect.reason);
        return 0;
    default:
        /* Every other GAP event (connection updates, MTU exchanges, ...)
         * belongs to features this package does not expose yet; ignoring it
         * keeps the mailbox bounded. */
        return 0;
    }
}

/* ---- command queue (Go task -> host task) ----
 *
 * Submission is non-blocking and executes on the host task, so no GAP call
 * ever runs on the goroutine that issued it: a control path cannot hold the
 * llgo scheduler across a driver wait. */

enum {
    CMD_SCAN_START = 1,
    CMD_SCAN_STOP = 2,
    CMD_CONNECT = 3,
    CMD_CANCEL = 4,
    CMD_DISCONNECT = 5,
};

#define BW_CMD_SLOTS 4

typedef struct {
    int32_t op;
    uint32_t op_id;
    uint32_t conn_id;
    uint16_t itvl;   /* 0.625 ms units */
    uint16_t window; /* 0.625 ms units */
    uint8_t addr[6];
    uint8_t addr_type;
    uint8_t passive;
    uint8_t filter_dup;
    int32_t duration_ms;
    int32_t timeout_ms;
} bw_cmd_t;

static bw_cmd_t s_cmd[BW_CMD_SLOTS];
static volatile uint32_t s_cmd_wr, s_cmd_rd;
static struct ble_npl_event s_cmd_ev;

static void bw_exec(const bw_cmd_t *c)
{
    int rc;

    switch (c->op) {
    case CMD_SCAN_START: {
        struct ble_gap_disc_params dp;
        memset(&dp, 0, sizeof dp);
        dp.itvl = c->itvl;
        dp.window = c->window;
        dp.passive = c->passive;
        dp.filter_duplicates = c->filter_dup;
        s_disc_op = c->op_id;
        rc = ble_gap_disc(s_own_type, c->duration_ms, &dp, bw_on_gap, NULL);
        if (rc != 0) {
            s_disc_op = 0;
            bw_ctrl_push(EV_ERROR, c->op_id, 0, rc);
        }
        break;
    }
    case CMD_SCAN_STOP:
        /* NimBLE reports nothing when a discovery is stopped by request, so a
         * successful stop is silent. A refusal (no scan running) reaches the
         * application as EV_ERROR. */
        rc = ble_gap_disc_cancel();
        if (rc != 0) {
            bw_ctrl_push(EV_ERROR, c->op_id, 0, rc);
        } else {
            s_disc_op = 0;
        }
        break;
    case CMD_CANCEL:
        if (c->op_id != 0 && c->op_id == s_disc_op) {
            rc = ble_gap_disc_cancel();
            if (rc == 0) {
                s_disc_op = 0;
            }
        } else if (c->op_id != 0 && c->op_id == s_conn_op) {
            rc = ble_gap_conn_cancel();
            if (rc == 0) {
                s_conn_op = 0;
            }
        } else {
            rc = BLE_HS_ENOENT; /* not the operation in flight */
        }
        if (rc != 0) {
            bw_ctrl_push(EV_ERROR, c->op_id, 0, rc);
        }
        break;
    case CMD_CONNECT: {
        ble_addr_t peer;
        struct ble_gap_conn_params cp;
        memset(&peer, 0, sizeof peer);
        peer.type = c->addr_type;
        memcpy(peer.val, c->addr, 6);
        memset(&cp, 0, sizeof cp);
        cp.scan_itvl = 16;         /* 10 ms */
        cp.scan_window = 16;
        cp.itvl_min = 24;          /* 30 ms */
        cp.itvl_max = 40;          /* 50 ms */
        cp.latency = 0;
        cp.supervision_timeout = 400; /* 4 s */
        s_conn_op = c->op_id;
        rc = ble_gap_connect(s_own_type, &peer, c->timeout_ms, &cp, bw_on_gap, NULL);
        if (rc != 0) {
            s_conn_op = 0;
            bw_ctrl_push(EV_ERROR, c->op_id, 0, rc);
        }
        break;
    }
    case CMD_DISCONNECT: {
        int handle = bw_conn_find(c->conn_id);
        if (handle < 0) {
            bw_ctrl_push(EV_ERROR, 0, c->conn_id, BLE_HS_ENOENT);
        } else {
            rc = ble_gap_terminate((uint16_t)handle, BLE_ERR_REM_USER_CONN_TERM);
            if (rc != 0) {
                bw_ctrl_push(EV_ERROR, 0, c->conn_id, rc);
            }
        }
        break;
    }
    default:
        break;
    }
}

static void bw_on_cmd(struct ble_npl_event *ev)
{
    (void)ev;
    while (s_cmd_rd != s_cmd_wr) {
        bw_cmd_t c = s_cmd[s_cmd_rd & (BW_CMD_SLOTS - 1)];
        s_cmd_rd++;
        bw_exec(&c);
    }
}

static int32_t bw_submit(const bw_cmd_t *c)
{
    if (!s_inited) {
        return BLE_HS_EINVAL; /* no host stack: nothing to submit to */
    }
    uint32_t w = s_cmd_wr;
    if ((uint32_t)(w - s_cmd_rd) >= BW_CMD_SLOTS) {
        return BLE_HS_ENOMEM; /* command queue full: acknowledged, not applied */
    }
    s_cmd[w & (BW_CMD_SLOTS - 1)] = *c;
    s_cmd_wr = w + 1;
    ble_npl_eventq_put(ble_hs_evq_get(), &s_cmd_ev);
    return 0;
}

/* ---- lifecycle ---- */

static void bw_host_task(void *param)
{
    (void)param;
    nimble_port_run(); /* returns after nimble_port_stop */
    nimble_port_freertos_deinit();
}

int32_t llgo_ble_init(void)
{
    if (s_inited) {
        return ESP_ERR_INVALID_STATE;
    }

    /* Bonding storage lives in NVS; a BLE-only firmware has no other NVS
     * user. The glue never formats the partition - a fresh-flash error is
     * reported instead of erased, because a shared NVS may hold data this
     * package knows nothing about. */
    esp_err_t e = nvs_flash_init();
    if (e != ESP_OK && e != ESP_ERR_INVALID_STATE) {
        return e;
    }

    /* One call brings up controller init + BLE enable + NimBLE host init
     * (components/bt/host/nimble .../nimble_port.c). Never call
     * esp_bt_controller_init/enable separately as well. */
    e = nimble_port_init();
    if (e != ESP_OK) {
        return e;
    }

    ble_hs_cfg.sync_cb = bw_on_sync;
    ble_hs_cfg.reset_cb = bw_on_reset;

    ble_npl_event_init(&s_cmd_ev, bw_on_cmd, NULL);

    e = nimble_port_freertos_init(bw_host_task);
    if (e != ESP_OK) {
        ble_npl_event_deinit(&s_cmd_ev);
        return e;
    }
    s_inited = 1;
    return ESP_OK;
}

int32_t llgo_ble_deinit(void)
{
    if (!s_inited) {
        return ESP_ERR_INVALID_STATE;
    }
    s_inited = 0;
    /* nimble_port_stop posts the stop event and waits for the host task to
     * service it; it must run on a task other than the host task (the Go
     * task), which is why the shutdown is a documented blocking call. */
    int rc = nimble_port_stop();
    /* Return the command event's BTDM OSAL pool element. NimBLE's npl is
     * implemented over the BTDM OSAL on this chip (nimble_npl_os.h:
     * struct ble_npl_event wraps struct btdm_osal_event), so
     * ble_npl_event_init() takes one element of s_btdm_osal_ev_pool and keeps
     * it until ble_npl_event_deinit(). Left out, the controller's own deinit
     * asserts that the pool is empty (btdm_osal_elem_mempool_deinit), which is
     * an abort in the middle of esp_bt_controller_disable(). The host task has
     * stopped by now, so the event cannot be queued any more. */
    ble_npl_event_deinit(&s_cmd_ev);
    if (rc != 0) {
        return rc;
    }
    return nimble_port_deinit();
}

int32_t llgo_ble_ready(void)
{
    return s_synced ? 1 : 0;
}

int32_t llgo_ble_print_version(void)
{
    uint8_t addr[6];
    int have = 0;
    if (s_synced) {
        have = ble_hs_id_copy_addr(s_own_type, addr, NULL) == 0;
    }
    if (have) {
        printf("[c] ble glue: free heap %u, own address %02x:%02x:%02x:%02x:%02x:%02x\n",
               (unsigned)esp_get_free_heap_size(),
               addr[5], addr[4], addr[3], addr[2], addr[1], addr[0]);
    } else {
        printf("[c] ble glue: free heap %u, no own address (not synced)\n",
               (unsigned)esp_get_free_heap_size());
    }
    return ESP_OK;
}

/* Milliseconds since boot, from the IDF timer: the clock a bounded Go wait
 * measures its deadline against (the poller rests one real FreeRTOS tick, so
 * the two advance together). */
uint32_t llgo_ble_now_ms(void)
{
    return (uint32_t)(esp_timer_get_time() / 1000);
}

/* ---- mailbox read (Go task) ---- */

int32_t llgo_ble_poll(llgo_ble_event_t *out)
{
    if (out == NULL) {
        return 0;
    }
    memset(out, 0, sizeof *out);

    if (s_ctrl_dropped != 0 || s_adv_dropped != 0) {
        out->kind = EV_OVERFLOW;
        out->status = (int32_t)(s_ctrl_dropped + s_adv_dropped);
        s_ctrl_dropped = 0;
        s_adv_dropped = 0;
        return 1;
    }

    if (s_ctrl_rd != s_ctrl_wr) {
        bw_ctrl_t c = s_ctrl[s_ctrl_rd & (BW_CTRL_SLOTS - 1)];
        s_ctrl_rd++;
        out->kind = c.kind;
        out->op_id = c.op_id;
        out->conn_id = c.conn_id;
        out->status = c.status;
        return 1;
    }

    if (s_adv_rd != s_adv_wr) {
        bw_adv_t a = s_adv[s_adv_rd & (BW_ADV_SLOTS - 1)];
        s_adv_rd++;
        out->kind = EV_ADV;
        memcpy(out->addr, a.addr, 6);
        out->addr_type = a.addr_type;
        out->rssi = a.rssi;
        out->adv_len = a.len;
        memcpy(out->adv, a.data, a.len);
        return 1;
    }

    return 0;
}

/* ---- central commands (Go task, non-blocking submission) ----
 *
 * Every function returns 0 when the command was accepted into the host queue
 * and a NimBLE host status (BLE_HS_*, positive) when it was not; the outcome
 * of an accepted command arrives through the mailbox, never as a return
 * value. */

int32_t llgo_ble_scan_start(int32_t itvl_ms, int32_t window_ms, int32_t passive,
                           int32_t filter_dup, int32_t duration_ms, uint32_t op_id)
{
    bw_cmd_t c;
    memset(&c, 0, sizeof c);
    c.op = CMD_SCAN_START;
    c.op_id = op_id;
    if (itvl_ms <= 0) {
        itvl_ms = 100;
    }
    if (window_ms <= 0) {
        window_ms = 50;
    }
    if (window_ms > itvl_ms) {
        window_ms = itvl_ms;
    }
    /* The controller takes 0.625 ms units in a uint16. */
    c.itvl = (uint16_t)((itvl_ms * 8) / 5);
    c.window = (uint16_t)((window_ms * 8) / 5);
    c.passive = passive ? 1 : 0;
    c.filter_dup = filter_dup ? 1 : 0;
    c.duration_ms = duration_ms;
    return bw_submit(&c);
}

int32_t llgo_ble_scan_stop(uint32_t op_id)
{
    bw_cmd_t c;
    memset(&c, 0, sizeof c);
    c.op = CMD_SCAN_STOP;
    c.op_id = op_id;
    return bw_submit(&c);
}

int32_t llgo_ble_connect(const uint8_t *addr, int32_t addr_type, int32_t timeout_ms,
                        uint32_t op_id)
{
    bw_cmd_t c;
    if (addr == NULL) {
        return BLE_HS_EINVAL;
    }
    memset(&c, 0, sizeof c);
    c.op = CMD_CONNECT;
    c.op_id = op_id;
    memcpy(c.addr, addr, 6);
    c.addr_type = (uint8_t)addr_type;
    c.timeout_ms = timeout_ms;
    return bw_submit(&c);
}

int32_t llgo_ble_cancel(uint32_t op_id)
{
    bw_cmd_t c;
    memset(&c, 0, sizeof c);
    c.op = CMD_CANCEL;
    c.op_id = op_id;
    return bw_submit(&c);
}

/* Copies the host-stack name into buf; returns the length copied. The Go side
 * reports it through Backend() so a program can log which stack it runs. */
int32_t llgo_ble_backend_name(uint8_t *buf, int32_t n)
{
    static const char name[] = "nimble";
    int32_t i = 0;
    if (n <= 0) {
        return 0;
    }
    for (; name[i] != 0 && i < n - 1; i++) {
        buf[i] = (uint8_t)name[i];
    }
    return i;
}

int32_t llgo_ble_disconnect(uint32_t conn_id)
{
    bw_cmd_t c;
    memset(&c, 0, sizeof c);
    c.op = CMD_DISCONNECT;
    c.conn_id = conn_id;
    return bw_submit(&c);
}

#else /* !(CONFIG_BT_ENABLED && CONFIG_BT_NIMBLE_ENABLED) */

/* IDF build without the NimBLE host: every entry point exists (the Go side
 * links against these names) and reports that BLE is not in this firmware.
 * Nothing pretends to succeed. */

typedef struct {
    int32_t kind;
    uint32_t op_id;
    uint32_t conn_id;
    int32_t status;
    uint8_t addr[6];
    uint8_t addr_type;
    int8_t rssi;
    uint8_t adv_len;
    uint8_t adv[31];
} llgo_ble_event_t;

int32_t llgo_ble_init(void) { return ESP_ERR_NOT_SUPPORTED; }
int32_t llgo_ble_deinit(void) { return ESP_ERR_NOT_SUPPORTED; }
int32_t llgo_ble_ready(void) { return 0; }
int32_t llgo_ble_print_version(void) { return ESP_ERR_NOT_SUPPORTED; }
uint32_t llgo_ble_now_ms(void) { return (uint32_t)(esp_timer_get_time() / 1000); }
int32_t llgo_ble_poll(llgo_ble_event_t *out) { (void)out; return 0; }
int32_t llgo_ble_scan_start(int32_t itvl_ms, int32_t window_ms, int32_t passive,
                           int32_t filter_dup, int32_t duration_ms, uint32_t op_id)
{
    (void)itvl_ms; (void)window_ms; (void)passive; (void)filter_dup;
    (void)duration_ms; (void)op_id;
    return ESP_ERR_NOT_SUPPORTED;
}
int32_t llgo_ble_scan_stop(uint32_t op_id) { (void)op_id; return ESP_ERR_NOT_SUPPORTED; }
int32_t llgo_ble_connect(const uint8_t *addr, int32_t addr_type, int32_t timeout_ms,
                        uint32_t op_id)
{
    (void)addr; (void)addr_type; (void)timeout_ms; (void)op_id;
    return ESP_ERR_NOT_SUPPORTED;
}
int32_t llgo_ble_cancel(uint32_t op_id) { (void)op_id; return ESP_ERR_NOT_SUPPORTED; }
int32_t llgo_ble_disconnect(uint32_t conn_id) { (void)conn_id; return ESP_ERR_NOT_SUPPORTED; }

int32_t llgo_ble_backend_name(uint8_t *buf, int32_t n)
{
    static const char name[] = "none";
    int32_t i = 0;
    if (n <= 0) {
        return 0;
    }
    for (; name[i] != 0 && i < n - 1; i++) {
        buf[i] = (uint8_t)name[i];
    }
    return i;
}

#endif /* CONFIG_BT_ENABLED && CONFIG_BT_NIMBLE_ENABLED */

#else /* !ESP_PLATFORM */

/* Template guard: outside an ESP-IDF build this file compiles to an
 * otherwise empty translation unit. */
typedef int llgo_ble_glue_not_in_idf_build;

#endif /* ESP_PLATFORM */
