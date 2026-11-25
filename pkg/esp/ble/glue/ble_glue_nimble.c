/* NimBLE backend of pkg/esp/ble: the C glue between llgo-compiled Go code
 * and the ESP-IDF NimBLE host:
 * a BLE central (scan / connect / disconnect) plus a GATT client (discovery,
 * read, write, notification subscription), with the stack's asynchronous
 * results delivered through a bounded, three-tier mailbox.
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
 *   (2) no large stack buffers. Go goroutines run on 2 KiB fiber stacks
 *       (targets/esp32s31-idf.json default-stack-size; the main fiber gets
 *       8 KiB), so every record that travels - scan results, mailbox slots -
 *       lives in static storage; the flat llgo_ble_event_t is the only shape
 *       that crosses, by pointer, with scalar and fixed-size-array fields only;
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
 * Mailbox design: three single-producer/single-consumer rings.
 *
 *   ctrl ring  - sync, reset, scan-done, connect, disconnect, GATT procedure
 *                completions, MTU reports, command rejections; 8 slots, no
 *                payload beyond codes. These events are what a program's state
 *                machine depends on, so they never compete with a scan flood
 *                or a notification storm for space.
 *   notif ring - notifications and indications; 8 slots of connection handle +
 *                value handle + flags + up to BW_DATA_MAX bytes of payload.
 *                Their own ring keeps them off the control tier.
 *   adv ring   - scan results only; 16 slots of address + RSSI + up to 31
 *                bytes of advertising data (the legacy maximum, which is what
 *                ble_gap_disc reports).
 *
 * Overflow drops the newest record of that ring - or a notification whose
 * payload does not fit one slot - and is reported once as an EV_OVERFLOW slot
 * carrying the dropped count, so loss is never silent but also never stalls the
 * host task.
 *
 * Cross-task visibility is an explicit release/acquire contract on the ring
 * indexes (the same discipline wifi_glue.c implements), not something the host
 * event queue's wake-up happens to provide: the producer fills its slot fields
 * and then publishes the write index with a release store (BW_PUBLISH); the
 * consumer acquires the write index before touching the slot and publishes the
 * read index with a release store after copying it out (BW_ACQUIRE). That the
 * two tasks happen to share one core in this build - where program order alone
 * would hold - is a coincidence of this firmware, not part of the contract.
 * No read-modify-write anywhere: RV32IMC has no A extension, so an RMW would
 * drag in libatomic. A dropped counter has a single writer (its ring's
 * producer), maintained with a plain read plus a relaxed atomic store; the
 * consumer keeps its own last-seen snapshot instead of clearing the producer's
 * counter. The GATT result slots (read value, discovery tables) hang off the
 * same contract: their producer fills them before the ctrl record that names
 * them is release-published, and the Go task touches them only after acquiring
 * that record's index.
 *
 * GATT results that a caller waits for (a discovered service list, a read
 * value) do not travel through a ring: discovery records land in fixed tables
 * and the read value in one slot, and the ctrl record that completes the
 * procedure says where they are. A table is deterministic - a record is either
 * there or the procedure ends with a status saying why not - while a ring can
 * drop, and dropping a record of a list the caller is about to consume is
 * exactly the failure mode a list API must not have.
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
#include "host/ble_hs_mbuf.h"
#include "host/ble_att.h"
#include "host/ble_gatt.h"
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
    EV_DISC_DONE = 8,  /* a service or characteristic discovery finished */
    EV_READ_DONE = 9,  /* a read finished; its value is in s_read */
    EV_WRITE_DONE = 10, /* a write finished */
    EV_NOTIFY = 11,    /* one notification or indication */
    EV_MTU = 12,       /* the negotiated ATT MTU changed */
    EV_OVERFLOW = -1,
};

/* The largest ATT payload one record carries. Mirrored by MaxData in ble.go,
 * where the reasoning lives: the bound is the goroutine stack the record
 * travels on (2 KiB fibers on this target), not the ATT MTU. A notification
 * that does not fit is dropped whole and counted; a read that does not fit is
 * reported as BLE_HS_EMSGSIZE. */
#define BW_DATA_MAX 128

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
    uint16_t att_handle; /* value handle an EV_NOTIFY came from */
    uint8_t data_len;    /* payload length this record carries */
    uint8_t indication;  /* 1 = indication, 0 = notification */
    uint8_t data[BW_DATA_MAX];
} llgo_ble_event_t;

_Static_assert(sizeof(llgo_ble_event_t) == 188, "llgo_ble_event_t ABI drifted");
_Static_assert(__builtin_offsetof(llgo_ble_event_t, addr) == 16, "addr offset");
_Static_assert(__builtin_offsetof(llgo_ble_event_t, adv) == 25, "adv offset");
_Static_assert(__builtin_offsetof(llgo_ble_event_t, att_handle) == 56, "att_handle offset");
_Static_assert(__builtin_offsetof(llgo_ble_event_t, data_len) == 58, "data_len offset");
_Static_assert(__builtin_offsetof(llgo_ble_event_t, indication) == 59, "indication offset");
_Static_assert(__builtin_offsetof(llgo_ble_event_t, data) == 60, "data offset");

/* Discovery records: the same flat shape the Go side mirrors (svcRec/chrRec in
 * ble.go). The UUID travels as a width plus 16 little-endian bytes - the byte
 * order the ATT PDU carries - and the Go type normalizes it. */
typedef struct {
    uint8_t uuid_len;
    uint8_t uuid[16];
    uint16_t start_handle;
    uint16_t end_handle;
} llgo_ble_svc_t;

typedef struct {
    uint8_t uuid_len;
    uint8_t uuid[16];
    uint8_t properties;
    uint16_t value_handle;
    uint16_t def_handle;
} llgo_ble_chr_t;

_Static_assert(sizeof(llgo_ble_svc_t) == 22, "llgo_ble_svc_t ABI drifted");
_Static_assert(__builtin_offsetof(llgo_ble_svc_t, start_handle) == 18, "svc start offset");
_Static_assert(sizeof(llgo_ble_chr_t) == 22, "llgo_ble_chr_t ABI drifted");
_Static_assert(__builtin_offsetof(llgo_ble_chr_t, properties) == 17, "chr properties offset");
_Static_assert(__builtin_offsetof(llgo_ble_chr_t, value_handle) == 18, "chr value offset");

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

/* The rings' indexes and the overflow bookkeeping. Each index is single-writer
 * (the producer owns _wr, the consumer owns _rd), and visibility across the
 * two tasks is the release/acquire contract documented in the file header.
 * No RMW anywhere: RV32IMC has no A extension, so fetch_add would drag in
 * libatomic. A dropped counter belongs to its ring's producer alone; the
 * consumer keeps a last-seen snapshot instead of clearing it. */
#define BW_PUBLISH(v, val) __atomic_store_n(&(v), (val), __ATOMIC_RELEASE)
#define BW_ACQUIRE(v)      __atomic_load_n(&(v), __ATOMIC_ACQUIRE)

static bw_ctrl_t s_ctrl[BW_CTRL_SLOTS];
static uint32_t s_ctrl_wr, s_ctrl_rd; /* wr: producer-owned, rd: consumer-owned */
static uint32_t s_ctrl_dropped;       /* producer-owned */
static uint32_t s_ctrl_dropped_seen;  /* consumer-owned */

static bw_adv_t s_adv[BW_ADV_SLOTS];
static uint32_t s_adv_wr, s_adv_rd;   /* wr: producer-owned, rd: consumer-owned */
static uint32_t s_adv_dropped;        /* producer-owned */
static uint32_t s_adv_dropped_seen;   /* consumer-owned */

/* The notification tier: what the peer pushes unsolicited. Its own ring keeps
 * a talkative characteristic from displacing a GATT completion or a scan
 * result, and a notification longer than one slot is dropped whole (counted,
 * never truncated). */
#define BW_NOTIF_SLOTS 8

typedef struct {
    uint16_t conn_handle;
    uint16_t attr_handle;
    uint8_t indication;
    uint8_t len;
    uint8_t data[BW_DATA_MAX];
} bw_notif_t;

static bw_notif_t s_notif[BW_NOTIF_SLOTS];
static uint32_t s_notif_wr, s_notif_rd; /* wr: producer-owned, rd: consumer-owned */
static uint32_t s_notif_dropped;        /* producer-owned */
static uint32_t s_notif_dropped_seen;   /* consumer-owned */

/* GATT client state. One procedure runs at a time (the Go side serializes
 * them; NIMBLE's ATT bearer carries one outstanding request anyway), so the
 * result slots below are single. The host task fills a table while its
 * discovery runs; the Go task reads it only after acquiring the EV_DISC_DONE
 * ctrl record, whose release publish (bw_gop_done -> bw_ctrl_push) is what
 * made the table visible - the same contract the rings use. */
#define BW_SVC_MAX 16
#define BW_CHR_MAX 32

static llgo_ble_svc_t s_svcs[BW_SVC_MAX];
static int32_t s_svc_n;
static uint32_t s_svc_conn;

static llgo_ble_chr_t s_chrs[BW_CHR_MAX];
static int32_t s_chr_n;
static uint32_t s_chr_conn;

/* The one GATT procedure in flight, with what its callbacks need to report it
 * (they run on the host task, and any task may hold a proc's callback, so the
 * context cannot live on a stack). */
typedef struct {
    uint8_t active;
    uint8_t full; /* a discovery table could not take every record */
    uint8_t kind; /* bw_cmd_t op of the procedure */
    uint32_t op_id;
    uint32_t conn_id;
} bw_gop_t;

static bw_gop_t s_gop;

/* The completion of a read carries the value: copied out of the ATT mbuf in
 * the callback and picked up by the poller, so a Go caller never sees stack
 * memory and the copy happens once. The callback fills both fields before
 * bw_gop_done release-publishes the EV_READ_DONE ctrl record; the poller's
 * acquire of that record's index is what makes them visible. */
static uint8_t s_read_data[BW_DATA_MAX];
static uint16_t s_read_len;

/* The payload of the write being submitted. Copied out of Go memory at
 * submission, before the command is release-published on the cmd ring; the
 * host task reads it only after acquiring the write index, and it only ever
 * re-fills it once the previous write's completion record came back. A Go-side
 * waitOp that times out (host task stalled for the whole GattTimeoutMs) leaves
 * the old command queued, so the next Write overwrites s_wr_data under it:
 * reachable only with a host task stopped for >= 30 s, acknowledged here
 * rather than defended. */
static uint8_t s_wr_data[BW_DATA_MAX];
static uint16_t s_wr_len;

/* Written by the host task only (GAP callbacks and the command handler). */
static volatile int s_inited, s_synced;
static uint8_t s_own_type;
static uint32_t s_disc_op; /* op id of the running discovery, 0 = none */
static uint32_t s_conn_op; /* op id of the running connect attempt, 0 = none */

static void bw_ctrl_push(int32_t kind, uint32_t op_id, uint32_t conn_id, int32_t status)
{
    uint32_t w = s_ctrl_wr;                      /* producer-owned */
    if ((uint32_t)(w - BW_ACQUIRE(s_ctrl_rd)) >= BW_CTRL_SLOTS) {
        __atomic_store_n(&s_ctrl_dropped, s_ctrl_dropped + 1, __ATOMIC_RELAXED);
        return;
    }
    s_ctrl[w & (BW_CTRL_SLOTS - 1)] = (bw_ctrl_t){ kind, op_id, conn_id, status };
    BW_PUBLISH(s_ctrl_wr, w + 1);                /* release: the slot is now visible */
}

static void bw_adv_push(const uint8_t *addr, uint8_t addr_type, int8_t rssi,
                        const uint8_t *data, uint8_t len)
{
    uint32_t w = s_adv_wr;                       /* producer-owned */
    if ((uint32_t)(w - BW_ACQUIRE(s_adv_rd)) >= BW_ADV_SLOTS) {
        __atomic_store_n(&s_adv_dropped, s_adv_dropped + 1, __ATOMIC_RELAXED);
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
    BW_PUBLISH(s_adv_wr, w + 1);                 /* release: the slot is now visible */
}

/* Copies one notification or indication into the notification ring. The mbuf
 * belongs to the host task and is freed as soon as the GAP callback returns, so
 * the copy has to happen here; a payload larger than one slot is dropped whole
 * (truncating a notification would hand the reader a value the peer never
 * sent) and counted in the mailbox's overflow report. */
static void bw_notif_push(uint16_t conn_handle, uint16_t attr_handle, uint8_t indication,
                          const struct os_mbuf *om)
{
    uint32_t w = s_notif_wr;                     /* producer-owned */
    if ((uint32_t)(w - BW_ACQUIRE(s_notif_rd)) >= BW_NOTIF_SLOTS) {
        __atomic_store_n(&s_notif_dropped, s_notif_dropped + 1, __ATOMIC_RELAXED);
        return;
    }
    bw_notif_t *slot = &s_notif[w & (BW_NOTIF_SLOTS - 1)];
    uint16_t len = 0;
    int rc = ble_hs_mbuf_to_flat(om, slot->data, BW_DATA_MAX, &len);
    if (rc != 0) {
        __atomic_store_n(&s_notif_dropped, s_notif_dropped + 1, __ATOMIC_RELAXED);
        return;
    }
    slot->conn_handle = conn_handle;
    slot->attr_handle = attr_handle;
    slot->indication = indication ? 1 : 0;
    slot->len = (uint8_t)len;
    BW_PUBLISH(s_notif_wr, w + 1);               /* release: the slot is now visible */
}

/* Stores a stack UUID in the record form: width plus little-endian bytes. A
 * 32-bit UUID is widened to 128 bits because the Go type has no 32-bit form;
 * its base-form value then matches the 16-bit alias when the upper half is
 * zero, which is what the Bluetooth base-UUID rule says it is. */
static const uint8_t bw_base_tail_le[12] = {
    0xFB, 0x34, 0x9B, 0x5F, 0x80, 0x00, 0x00, 0x80, 0x00, 0x10, 0x00, 0x00,
};

static void bw_uuid_store(uint8_t *dst, uint8_t *len_out, const ble_uuid_t *u)
{
    memset(dst, 0, 16);
    switch (u->type) {
    case BLE_UUID_TYPE_16: {
        const ble_uuid16_t *u16 = (const ble_uuid16_t *)u;
        dst[0] = (uint8_t)(u16->value & 0xFF);
        dst[1] = (uint8_t)(u16->value >> 8);
        *len_out = 2;
        break;
    }
    case BLE_UUID_TYPE_32: {
        const ble_uuid32_t *u32 = (const ble_uuid32_t *)u;
        memcpy(dst, bw_base_tail_le, sizeof bw_base_tail_le);
        dst[12] = (uint8_t)(u32->value & 0xFF);
        dst[13] = (uint8_t)((u32->value >> 8) & 0xFF);
        dst[14] = (uint8_t)((u32->value >> 16) & 0xFF);
        dst[15] = (uint8_t)((u32->value >> 24) & 0xFF);
        *len_out = 16;
        break;
    }
    case BLE_UUID_TYPE_128:
        memcpy(dst, ((const ble_uuid128_t *)u)->value, 16);
        *len_out = 16;
        break;
    default:
        *len_out = 0;
        break;
    }
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
    return 0; /* table full: the caller reports the failed connect */
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

/* The ConnID a live handle is registered under, for the records that carry a
 * bare handle: notifications and MTU reports. The table belongs to the host
 * task while the poller runs on the Go task; an entry is complete before the
 * ConnID that names it is published (the connect record that carries it goes
 * through the same ring), so a reader holding a live ConnID sees a consistent
 * entry, and a stale one can only produce a status error. */
static uint32_t bw_conn_id_of(uint16_t handle)
{
    for (int i = 0; i < BW_CONNS; i++) {
        if (s_conns[i].used && s_conns[i].handle == handle) {
            return (s_conns[i].gen << 16) | handle;
        }
    }
    return handle;
}

/* ---- GATT procedure context (host task) ---- */

static void bw_gop_begin(uint8_t kind, uint32_t op_id, uint32_t conn_id)
{
    s_gop.active = 1;
    s_gop.full = 0;
    s_gop.kind = kind;
    s_gop.op_id = op_id;
    s_gop.conn_id = conn_id;
}

/* Ends the procedure in flight and pushes the one ctrl record that reports it.
 * NimBLE calls a procedure's callback exactly once per result and once at
 * completion, so the callbacks below are careful to keep the "completion" call
 * and the "one record per result" rule apart: a repeating callback (discovery)
 * pushes nothing until it sees a non-zero status. The release publish inside
 * bw_ctrl_push is also what makes a discovery's table (s_svcs/s_chrs and their
 * counts) visible to the Go task. */
static void bw_gop_done(int32_t kind, int32_t status)
{
    if (!s_gop.active) {
        return;
    }
    s_gop.active = 0;
    bw_ctrl_push(kind, s_gop.op_id, s_gop.conn_id, status);
}

/* A submission that failed before the procedure started: the completion record
 * is still owed to the caller, unless NimBLE already reported it synchronously
 * through the callback (which ends the procedure itself). */
static void bw_gop_submit_failed(int rc)
{
    if (s_gop.active) {
        bw_gop_done(EV_ERROR, rc);
    }
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
            uint32_t id = bw_conn_add(ev->connect.conn_handle);
            if (id == 0) {
                /* Connection table full: report the failure instead of a
                 * conn_id 0, which the Go side would read as a successful
                 * connect. */
                bw_ctrl_push(EV_CONNECT, op, 0, BLE_HS_ENOMEM);
            } else {
                bw_ctrl_push(EV_CONNECT, op, id, 0);
                /* Ask for the ATT MTU exchange right away: on the central side the
                 * host does not do it by itself, and both the notification payload
                 * size and the largest single read depend on the result. This runs
                 * on the host task, which owns the procedure table, and the value
                 * arrives as BLE_GAP_EVENT_MTU. */
                ble_gattc_exchange_mtu(ev->connect.conn_handle, NULL, NULL);
            }
        } else if (op != 0) {
            bw_ctrl_push(EV_CONNECT, op, 0, ev->connect.status);
        }
        /* A failure with no operation in flight is the completion of an
         * attempt the application cancelled (or one this glue never started):
         * nothing to attribute it to, and the application already knows. */
        return 0;
    }
    case BLE_GAP_EVENT_DISCONNECT: {
        uint32_t id = bw_conn_remove(ev->disconnect.conn.conn_handle);
        bw_ctrl_push(EV_DISCONNECT, 0, id, ev->disconnect.reason);
        /* The discovery tables describe that connection's attribute table;
         * keeping them would let a stale handle be read or written. */
        if (s_svc_conn == id) {
            s_svc_n = 0;
            s_svc_conn = 0;
        }
        if (s_chr_conn == id) {
            s_chr_n = 0;
            s_chr_conn = 0;
        }
        return 0;
    }
    case BLE_GAP_EVENT_NOTIFY_RX:
        /* A notification or an indication; the host has already built the
         * confirmation for an indication, so the glue only copies the payload
         * (the mbuf dies with this callback). */
        bw_notif_push(ev->notify_rx.conn_handle, ev->notify_rx.attr_handle,
                      ev->notify_rx.indication, ev->notify_rx.om);
        return 0;
    case BLE_GAP_EVENT_MTU:
        bw_ctrl_push(EV_MTU, 0, bw_conn_id_of(ev->mtu.conn_handle), ev->mtu.value);
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
    CMD_DISC_SVCS = 6,
    CMD_DISC_CHRS = 7,
    CMD_READ = 8,
    CMD_WRITE = 9,
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
    uint16_t att_handle;   /* value handle of a read or write */
    uint16_t start_handle; /* service range of a characteristic discovery */
    uint16_t end_handle;
    uint8_t write_mode; /* BW_WRITE_* */
} bw_cmd_t;

/* The ATT write procedures: the Go side picks one from the characteristic's
 * properties, or names it explicitly. */
enum {
    BW_WRITE_RSP = 1,  /* ATT Write Request: the peer acknowledges */
    BW_WRITE_NRSP = 2, /* ATT Write Command: the peer does not */
};

static bw_cmd_t s_cmd[BW_CMD_SLOTS];
static uint32_t s_cmd_wr, s_cmd_rd; /* s_cmd_wr: Go task owns, s_cmd_rd: host task owns */
static struct ble_npl_event s_cmd_ev;

/* ---- GATT procedure callbacks (host task) ---- */

static void bw_svc_reset(uint32_t conn_id)
{
    s_svc_n = 0;
    s_svc_conn = conn_id;
}

static void bw_chr_reset(uint32_t conn_id)
{
    s_chr_n = 0;
    s_chr_conn = conn_id;
}

static int bw_on_disc_svc(uint16_t conn_handle, const struct ble_gatt_error *err,
                          const struct ble_gatt_svc *svc, void *arg)
{
    (void)conn_handle;
    (void)arg;
    if (err->status == 0 && svc != NULL) {
        if (s_svc_n >= BW_SVC_MAX) {
            s_gop.full = 1;
            return 0;
        }
        llgo_ble_svc_t *rec = &s_svcs[s_svc_n++];
        bw_uuid_store(rec->uuid, &rec->uuid_len, &svc->uuid.u);
        rec->start_handle = svc->start_handle;
        rec->end_handle = svc->end_handle;
        return 0;
    }
    int32_t status = (err->status == BLE_HS_EDONE) ? 0 : (int32_t)err->status;
    if (status == 0 && s_gop.full) {
        status = BLE_HS_EMSGSIZE; /* more services than the table keeps */
    }
    bw_gop_done(EV_DISC_DONE, status);
    return 0;
}

static int bw_on_disc_chr(uint16_t conn_handle, const struct ble_gatt_error *err,
                          const struct ble_gatt_chr *chr, void *arg)
{
    (void)conn_handle;
    (void)arg;
    if (err->status == 0 && chr != NULL) {
        if (s_chr_n >= BW_CHR_MAX) {
            s_gop.full = 1;
            return 0;
        }
        llgo_ble_chr_t *rec = &s_chrs[s_chr_n++];
        bw_uuid_store(rec->uuid, &rec->uuid_len, &chr->uuid.u);
        rec->properties = chr->properties;
        rec->value_handle = chr->val_handle;
        rec->def_handle = chr->def_handle;
        return 0;
    }
    int32_t status = (err->status == BLE_HS_EDONE) ? 0 : (int32_t)err->status;
    if (status == 0 && s_gop.full) {
        status = BLE_HS_EMSGSIZE;
    }
    bw_gop_done(EV_DISC_DONE, status);
    return 0;
}

static int bw_on_read(uint16_t conn_handle, const struct ble_gatt_error *err,
                      struct ble_gatt_attr *attr, void *arg)
{
    (void)conn_handle;
    (void)arg;
    int32_t status = (int32_t)err->status;
    if (status == BLE_HS_EDONE) {
        status = 0; /* some procedures report completion after the value */
    }
    s_read_len = 0;
    if (status == 0 && attr != NULL && attr->om != NULL) {
        uint16_t len = 0;
        int rc = ble_hs_mbuf_to_flat(attr->om, s_read_data, BW_DATA_MAX, &len);
        if (rc != 0) {
            status = rc; /* the value does not fit one event */
        } else {
            s_read_len = len;
        }
    }
    bw_gop_done(EV_READ_DONE, status);
    return 0;
}

static int bw_on_write(uint16_t conn_handle, const struct ble_gatt_error *err,
                       struct ble_gatt_attr *attr, void *arg)
{
    (void)conn_handle;
    (void)attr;
    (void)arg;
    int32_t status = (int32_t)err->status;
    if (status == BLE_HS_EDONE) {
        status = 0;
    }
    bw_gop_done(EV_WRITE_DONE, status);
    return 0;
}

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
    case CMD_DISC_SVCS: {
        int handle = bw_conn_find(c->conn_id);
        if (handle < 0) {
            bw_ctrl_push(EV_ERROR, c->op_id, c->conn_id, BLE_HS_ENOENT);
            break;
        }
        if (s_gop.active) {
            bw_ctrl_push(EV_ERROR, c->op_id, c->conn_id, BLE_HS_EALREADY);
            break;
        }
        bw_svc_reset(c->conn_id);
        bw_gop_begin(CMD_DISC_SVCS, c->op_id, c->conn_id);
        rc = ble_gattc_disc_all_svcs((uint16_t)handle, bw_on_disc_svc, NULL);
        if (rc != 0) {
            bw_gop_submit_failed(rc);
        }
        break;
    }
    case CMD_DISC_CHRS: {
        int handle = bw_conn_find(c->conn_id);
        if (handle < 0) {
            bw_ctrl_push(EV_ERROR, c->op_id, c->conn_id, BLE_HS_ENOENT);
            break;
        }
        if (s_gop.active) {
            bw_ctrl_push(EV_ERROR, c->op_id, c->conn_id, BLE_HS_EALREADY);
            break;
        }
        bw_chr_reset(c->conn_id);
        bw_gop_begin(CMD_DISC_CHRS, c->op_id, c->conn_id);
        rc = ble_gattc_disc_all_chrs((uint16_t)handle, c->start_handle, c->end_handle,
                                     bw_on_disc_chr, NULL);
        if (rc != 0) {
            bw_gop_submit_failed(rc);
        }
        break;
    }
    case CMD_READ: {
        int handle = bw_conn_find(c->conn_id);
        if (handle < 0) {
            bw_ctrl_push(EV_ERROR, c->op_id, c->conn_id, BLE_HS_ENOENT);
            break;
        }
        if (s_gop.active) {
            bw_ctrl_push(EV_ERROR, c->op_id, c->conn_id, BLE_HS_EALREADY);
            break;
        }
        s_read_len = 0;
        bw_gop_begin(CMD_READ, c->op_id, c->conn_id);
        rc = ble_gattc_read((uint16_t)handle, c->att_handle, bw_on_read, NULL);
        if (rc != 0) {
            bw_gop_submit_failed(rc);
        }
        break;
    }
    case CMD_WRITE: {
        int handle = bw_conn_find(c->conn_id);
        if (handle < 0) {
            bw_ctrl_push(EV_ERROR, c->op_id, c->conn_id, BLE_HS_ENOENT);
            break;
        }
        if (s_gop.active) {
            bw_ctrl_push(EV_ERROR, c->op_id, c->conn_id, BLE_HS_EALREADY);
            break;
        }
        bw_gop_begin(CMD_WRITE, c->op_id, c->conn_id);
        if (c->write_mode == BW_WRITE_NRSP) {
            /* A write command is never acknowledged, so there is no callback to
             * wait for: the submission result is the whole outcome. */
            rc = ble_gattc_write_no_rsp_flat((uint16_t)handle, c->att_handle, s_wr_data,
                                             s_wr_len);
            bw_gop_done(rc != 0 ? EV_ERROR : EV_WRITE_DONE, rc);
        } else {
            rc = ble_gattc_write_flat((uint16_t)handle, c->att_handle, s_wr_data, s_wr_len,
                                      bw_on_write, NULL);
            if (rc != 0) {
                bw_gop_submit_failed(rc);
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
    uint32_t r = s_cmd_rd;                       /* consumer-owned (host task) */
    while (BW_ACQUIRE(s_cmd_wr) != r) {
        bw_cmd_t c = s_cmd[r & (BW_CMD_SLOTS - 1)];
        BW_PUBLISH(s_cmd_rd, r + 1);             /* release: the slot is free again */
        bw_exec(&c);
        r++;
    }
}

static int32_t bw_submit(const bw_cmd_t *c)
{
    if (!s_inited) {
        return BLE_HS_EINVAL; /* no host stack: nothing to submit to */
    }
    uint32_t w = s_cmd_wr;                       /* producer-owned (the Go task) */
    if ((uint32_t)(w - BW_ACQUIRE(s_cmd_rd)) >= BW_CMD_SLOTS) {
        return BLE_HS_ENOMEM; /* command queue full: acknowledged, not applied */
    }
    s_cmd[w & (BW_CMD_SLOTS - 1)] = *c;
    /* The release here is the visibility mechanism, not the eventq_put below:
     * it is what makes the command - and the s_wr_data filled before it -
     * visible to the host task, which acquires the index in bw_on_cmd. */
    BW_PUBLISH(s_cmd_wr, w + 1);
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

    /* The overflow report is edge-triggered on the consumer's own snapshots, so
     * the producers never clear their counters (and no RMW is needed anywhere). */
    uint32_t ctrl_d = BW_ACQUIRE(s_ctrl_dropped);
    uint32_t adv_d = BW_ACQUIRE(s_adv_dropped);
    uint32_t notif_d = BW_ACQUIRE(s_notif_dropped);
    if (ctrl_d != s_ctrl_dropped_seen || adv_d != s_adv_dropped_seen ||
        notif_d != s_notif_dropped_seen) {
        out->kind = EV_OVERFLOW;
        out->status = (int32_t)((ctrl_d - s_ctrl_dropped_seen) +
                                (adv_d - s_adv_dropped_seen) +
                                (notif_d - s_notif_dropped_seen));
        s_ctrl_dropped_seen = ctrl_d;
        s_adv_dropped_seen = adv_d;
        s_notif_dropped_seen = notif_d;
        return 1;
    }

    uint32_t r = s_ctrl_rd;                      /* consumer-owned */
    if (BW_ACQUIRE(s_ctrl_wr) != r) {
        bw_ctrl_t c = s_ctrl[r & (BW_CTRL_SLOTS - 1)];
        BW_PUBLISH(s_ctrl_rd, r + 1);            /* release: the slot is free again */
        out->kind = c.kind;
        out->op_id = c.op_id;
        out->conn_id = c.conn_id;
        out->status = c.status;
        if (c.kind == EV_READ_DONE && c.status == 0) {
            /* The value of the read: the callback filled s_read_data/s_read_len
             * before release-publishing this record, so the acquire above is
             * what makes them visible. Only one read is ever in flight, so the
             * slot cannot belong to another. */
            uint16_t len = s_read_len;
            if (len > BW_DATA_MAX) {
                len = BW_DATA_MAX;
            }
            memcpy(out->data, s_read_data, len);
            out->data_len = (uint8_t)len;
        }
        return 1;
    }

    if (BW_ACQUIRE(s_notif_wr) != s_notif_rd) {
        /* Read in place - a local copy would put a whole slot on the Go task's
         * (fiber) stack - and publish the read index only after the copy, so
         * the host task cannot reuse the slot under us. */
        const bw_notif_t *n = &s_notif[s_notif_rd & (BW_NOTIF_SLOTS - 1)];
        out->kind = EV_NOTIFY;
        out->conn_id = bw_conn_id_of(n->conn_handle);
        out->att_handle = n->attr_handle;
        out->indication = n->indication;
        out->data_len = n->len;
        memcpy(out->data, n->data, n->len);
        BW_PUBLISH(s_notif_rd, s_notif_rd + 1);  /* release: the slot is free again */
        return 1;
    }

    r = s_adv_rd;                                /* consumer-owned */
    if (BW_ACQUIRE(s_adv_wr) != r) {
        bw_adv_t a = s_adv[r & (BW_ADV_SLOTS - 1)];
        BW_PUBLISH(s_adv_rd, r + 1);             /* release: the slot is free again */
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

/* Stored bonds, same record layout as the Bluedroid sibling (six address
 * bytes in stack order, then the address type). See that file for why the
 * caller wants this: a bonded peer stops advertising and waits to be
 * connected, and the stored key skips pairing on the way back up. */
int32_t llgo_ble_bond_list(uint8_t *out, int32_t max_peers)
{
    if (!s_synced || out == NULL || max_peers <= 0) {
        return 0;
    }
    ble_addr_t peers[8];
    int num = 0;
    int want = max_peers > 8 ? 8 : (int)max_peers;
    int rc = ble_store_util_bonded_peers(peers, &num, want);
    if (rc != 0 || num <= 0) {
        return 0;
    }
    if (num > want) {
        num = want;
    }
    for (int i = 0; i < num; i++) {
        memcpy(out + i * 7, peers[i].val, 6);
        out[i * 7 + 6] = peers[i].type;
    }
    return num;
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

/* ---- GATT client (Go task) ----
 *
 * Same contract as the GAP commands: 0 means "queued for the host task", a
 * NimBLE status means "not submitted", and the outcome always arrives through
 * the mailbox. The accessors below are plain reads of what the host task
 * published, and - like every other result the mailbox announces - the record
 * a caller reads is complete before the completion record that points at it. */

int32_t llgo_ble_disc_svcs(uint32_t conn_id, uint32_t op_id)
{
    bw_cmd_t c;
    memset(&c, 0, sizeof c);
    c.op = CMD_DISC_SVCS;
    c.conn_id = conn_id;
    c.op_id = op_id;
    return bw_submit(&c);
}

int32_t llgo_ble_disc_chrs(uint32_t conn_id, uint16_t start_handle, uint16_t end_handle,
                           uint32_t op_id)
{
    bw_cmd_t c;
    memset(&c, 0, sizeof c);
    c.op = CMD_DISC_CHRS;
    c.conn_id = conn_id;
    c.op_id = op_id;
    c.start_handle = start_handle;
    c.end_handle = end_handle;
    return bw_submit(&c);
}

int32_t llgo_ble_read(uint32_t conn_id, uint16_t handle, uint32_t op_id)
{
    bw_cmd_t c;
    if (handle == 0) {
        return BLE_HS_EINVAL;
    }
    memset(&c, 0, sizeof c);
    c.op = CMD_READ;
    c.conn_id = conn_id;
    c.op_id = op_id;
    c.att_handle = handle;
    return bw_submit(&c);
}

int32_t llgo_ble_write(uint32_t conn_id, uint16_t handle, const uint8_t *data, int32_t len,
                       int32_t mode, uint32_t op_id)
{
    bw_cmd_t c;
    if (handle == 0 || len < 0 || (len > 0 && data == NULL)) {
        return BLE_HS_EINVAL;
    }
    if (len > BW_DATA_MAX) {
        return BLE_HS_EMSGSIZE;
    }
    if (mode != BW_WRITE_RSP && mode != BW_WRITE_NRSP) {
        return BLE_HS_EINVAL;
    }
    /* Copy out of Go memory now: the host task runs the write later, and a Go
     * stack must not be dereferenced from another task. */
    if (len > 0) {
        memcpy(s_wr_data, data, (size_t)len);
    }
    s_wr_len = (uint16_t)len;
    memset(&c, 0, sizeof c);
    c.op = CMD_WRITE;
    c.conn_id = conn_id;
    c.op_id = op_id;
    c.att_handle = handle;
    c.write_mode = (uint8_t)mode;
    return bw_submit(&c);
}

int32_t llgo_ble_svc_count(uint32_t conn_id)
{
    return (conn_id != 0 && conn_id == s_svc_conn) ? s_svc_n : 0;
}

int32_t llgo_ble_svc_at(uint32_t conn_id, int32_t idx, llgo_ble_svc_t *out)
{
    if (out == NULL || conn_id == 0 || conn_id != s_svc_conn || idx < 0 || idx >= s_svc_n) {
        return 0;
    }
    *out = s_svcs[idx];
    return 1;
}

int32_t llgo_ble_chr_count(uint32_t conn_id)
{
    return (conn_id != 0 && conn_id == s_chr_conn) ? s_chr_n : 0;
}

int32_t llgo_ble_chr_at(uint32_t conn_id, int32_t idx, llgo_ble_chr_t *out)
{
    if (out == NULL || conn_id == 0 || conn_id != s_chr_conn || idx < 0 || idx >= s_chr_n) {
        return 0;
    }
    *out = s_chrs[idx];
    return 1;
}

/* The negotiated ATT MTU of a live connection. ble_att_mtu() takes the host lock
 * itself, so asking from the Go task is safe; the peer never answers an MTU
 * exchange, so the value stays at the ATT default (23) until it does. */
int32_t llgo_ble_mtu(uint32_t conn_id, int32_t *out)
{
    if (out == NULL) {
        return BLE_HS_EINVAL;
    }
    if (!s_inited) {
        return BLE_HS_ENOTCONN;
    }
    int handle = bw_conn_find(conn_id);
    if (handle < 0) {
        return BLE_HS_ENOENT;
    }
    *out = (int32_t)ble_att_mtu((uint16_t)handle);
    return 0;
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
    uint16_t att_handle;
    uint8_t data_len;
    uint8_t indication;
    uint8_t data[253]; /* MaxData in ble.go: same shape as the real record */
} llgo_ble_event_t;

typedef struct {
    uint8_t uuid_len;
    uint8_t uuid[16];
    uint16_t start_handle;
    uint16_t end_handle;
} llgo_ble_svc_t;

typedef struct {
    uint8_t uuid_len;
    uint8_t uuid[16];
    uint8_t properties;
    uint16_t value_handle;
    uint16_t def_handle;
} llgo_ble_chr_t;

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
int32_t llgo_ble_disc_svcs(uint32_t conn_id, uint32_t op_id)
{
    (void)conn_id; (void)op_id;
    return ESP_ERR_NOT_SUPPORTED;
}
int32_t llgo_ble_disc_chrs(uint32_t conn_id, uint16_t start_handle, uint16_t end_handle,
                           uint32_t op_id)
{
    (void)conn_id; (void)start_handle; (void)end_handle; (void)op_id;
    return ESP_ERR_NOT_SUPPORTED;
}
int32_t llgo_ble_read(uint32_t conn_id, uint16_t handle, uint32_t op_id)
{
    (void)conn_id; (void)handle; (void)op_id;
    return ESP_ERR_NOT_SUPPORTED;
}
int32_t llgo_ble_write(uint32_t conn_id, uint16_t handle, const uint8_t *data, int32_t len,
                       int32_t mode, uint32_t op_id)
{
    (void)conn_id; (void)handle; (void)data; (void)len; (void)mode; (void)op_id;
    return ESP_ERR_NOT_SUPPORTED;
}
int32_t llgo_ble_svc_count(uint32_t conn_id) { (void)conn_id; return 0; }
int32_t llgo_ble_svc_at(uint32_t conn_id, int32_t idx, llgo_ble_svc_t *out)
{
    (void)conn_id; (void)idx; (void)out;
    return 0;
}
int32_t llgo_ble_chr_count(uint32_t conn_id) { (void)conn_id; return 0; }
int32_t llgo_ble_chr_at(uint32_t conn_id, int32_t idx, llgo_ble_chr_t *out)
{
    (void)conn_id; (void)idx; (void)out;
    return 0;
}
int32_t llgo_ble_mtu(uint32_t conn_id, int32_t *out)
{
    (void)conn_id; (void)out;
    return ESP_ERR_NOT_SUPPORTED;
}

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
