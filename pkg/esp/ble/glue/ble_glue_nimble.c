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
 * Mailbox design: four single-producer/single-consumer rings.
 *
 *   ctrl ring  - sync, reset, scan-done, connect, disconnect (both roles),
 *                GATT procedure completions, advertising and service-
 *                registration completions, MTU reports, command rejections;
 *                8 slots, no payload beyond codes and a connect event's peer
 *                address. These events are what a program's state machine
 *                depends on, so they never compete with a scan flood or a
 *                notification storm for space.
 *   periph ring - what a central does to this device's attributes while we
 *                serve it: characteristic writes and client-configuration
 *                (CCCD) writes; 8 slots of connection id + handle + status +
 *                up to BW_DATA_MAX bytes of payload. Its own ring keeps a
 *                write flood off the control tier, and the poll drains ctrl
 *                before periph, so an EV_PERIPH_CONNECT is never left queued
 *                behind the payload records it makes actionable.
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
#include <stdint.h>

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

#if !MYNEWT_VAL(BLE_GATTS)
/* This firmware's NimBLE has no GATT server (CONFIG_BT_NIMBLE_GATT_SERVER
 * off) and the stack library compiles ble_gatts_notify_custom out with it -
 * while ble_gatts_indicate_custom stays in unconditionally (its own body
 * refuses under !BLE_GATT_INDICATE). This definition carries the missing
 * name with the honest refusal: the ABI stays complete for every NimBLE
 * build shape, and a central-only firmware keeps working exactly as before.
 * (The mbuf follows the real function's contract: the caller releases
 * ownership.) */
int ble_gatts_notify_custom(uint16_t conn_handle, uint16_t chr_val_handle,
                            struct os_mbuf *txom)
{
    (void)conn_handle;
    (void)chr_val_handle;
    os_mbuf_free_chain(txom);
    return BLE_HS_ENOTSUP;
}
#endif /* !MYNEWT_VAL(BLE_GATTS) */

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
    EV_PERIPH_CONNECT = 13,    /* a central connected to our advertisement */
    EV_PERIPH_DISCONNECT = 14, /* a served connection ended */
    EV_CHAR_WRITE = 15,        /* a peer wrote a characteristic value */
    EV_SUBSCRIBED = 16,        /* a peer wrote a CCCD; status = its value */
    EV_ADV_DONE = 17,          /* a Start/StopAdvertising completed */
    EV_SERVICES_DONE = 18,     /* the local GATT server registration finished */
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
    uint8_t addr[6];     /* the peer address an EV_PERIPH_CONNECT carries;
                          * zero for every other kind */
    uint8_t addr_type;
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
static uint32_t s_notif_wr, s_notif_rd; /* wr: producer-owned (host task), rd: consumer-owned */
static uint32_t s_notif_dropped;        /* producer-owned */
static uint32_t s_notif_dropped_seen;   /* consumer-owned */

/* The peripheral tier: what a central does to this device's attributes -
 * characteristic writes and client-configuration (CCCD) writes. Its own ring
 * keeps a write flood off the control tier (and off the notification ring,
 * so a central and a peripheral peer cannot displace each other's payloads).
 * Only the host task writes it: the access callback and the GAP subscribe
 * event both run there. */
#define BW_PERIPH_SLOTS 8

typedef struct {
    int32_t kind;        /* EV_CHAR_WRITE or EV_SUBSCRIBED */
    uint16_t conn_handle;
    uint16_t attr_handle;
    int32_t status;      /* EV_SUBSCRIBED: the raw CCCD value; EV_CHAR_WRITE: 0 */
    uint8_t len;
    uint8_t data[BW_DATA_MAX];
} bw_periph_t;

static bw_periph_t s_periph[BW_PERIPH_SLOTS];
static uint32_t s_periph_wr, s_periph_rd; /* wr: producer-owned (host task), rd: consumer-owned */
static uint32_t s_periph_dropped;         /* producer-owned */
static uint32_t s_periph_dropped_seen;    /* consumer-owned */

/* Copies one peer action into the peripheral ring. The mbuf dies with the
 * access callback, so the copy has to happen here; a payload larger than one
 * slot is dropped whole (never truncated) and counted in the mailbox's
 * overflow report. */
static void bw_periph_push(int32_t kind, uint16_t conn_handle, uint16_t attr_handle,
                           int32_t status, const uint8_t *value, uint16_t value_len)
{
    uint32_t w = s_periph_wr;                    /* producer-owned */
    if ((uint32_t)(w - BW_ACQUIRE(s_periph_rd)) >= BW_PERIPH_SLOTS) {
        __atomic_store_n(&s_periph_dropped, s_periph_dropped + 1, __ATOMIC_RELAXED);
        return;
    }
    if (value_len > BW_DATA_MAX) {
        __atomic_store_n(&s_periph_dropped, s_periph_dropped + 1, __ATOMIC_RELAXED);
        return;
    }
    bw_periph_t *slot = &s_periph[w & (BW_PERIPH_SLOTS - 1)];
    slot->kind = kind;
    slot->conn_handle = conn_handle;
    slot->attr_handle = attr_handle;
    slot->status = status;
    slot->len = (uint8_t)value_len;
    if (value_len > 0 && value != NULL) {
        memcpy(slot->data, value, value_len);
    }
    BW_PUBLISH(s_periph_wr, w + 1);              /* release: the slot is now visible */
}

/* ---- peripheral role: advertising, the local GATT server ----
 *
 * The same boundary rules as the central half, mirrored. The value model is
 * the read answer: every characteristic owns a fixed value cache in this
 * glue, the access callback (host task) answers a peer read straight from it
 * - no Go involvement, no deferred response, because the ATT server must
 * answer synchronously and the llgo scheduler cannot be entered from the
 * host task. WriteCharValue fills the same cache from the Go task; the copy
 * runs under the gatts critical section below, which is what keeps a cache
 * write and a read the host task is serving serialized.
 *
 * Advertising is the legacy procedure set (raw advertising and scan-response
 * payloads + start/stop), matching the scan path's legacy-only scope.
 */

/* Advertising state. The payloads live in static buffers: the Go task copies
 * them in at submission, and the host task (which owns the gap procedure)
 * hands the same buffers to ble_gap_adv_set_data when the command runs.
 * s_adv_op owns the one start/stop operation in flight - the Go side refuses
 * a second one with BLE_HS_EALREADY, so no claim protocol is needed here. */
static uint8_t s_adv_data[BW_ADV_MAX];
static uint16_t s_adv_data_len;
static uint8_t s_scan_rsp[BW_ADV_MAX];
static uint16_t s_scan_rsp_len;
static uint8_t s_adv_conn;    /* connectable advertisement */
static uint8_t s_adv_running;
static uint16_t s_adv_itvl_ms;
static uint32_t s_adv_op;

/* The local GATT server: the declaration the Go side serialized once (the
 * staging below), the attribute handles NimBLE assigned at registration (the
 * tree's val_handle pointers aim straight into these records), and one value
 * cache per characteristic. MaxLocalServices/MaxLocalChars in ble.go are
 * these two bounds. */
#define BW_GSVC_MAX 4
#define BW_GCHR_MAX 8

/* The declaration as the wire carries it (llgo_ble_gsvc_t / llgo_ble_gchr_t;
 * the Go side mirrors them byte for byte, pinned by ble_test.go). */
typedef struct {
    uint8_t uuid_len;
    uint8_t uuid[16];
    uint8_t _pad;
    uint16_t n_chars;
} llgo_ble_gsvc_t;

typedef struct {
    uint8_t uuid_len;
    uint8_t uuid[16];
    uint8_t props;
} llgo_ble_gchr_t;

_Static_assert(sizeof(llgo_ble_gsvc_t) == 20, "llgo_ble_gsvc_t ABI drifted");
_Static_assert(__builtin_offsetof(llgo_ble_gsvc_t, n_chars) == 18, "gsvc n_chars offset");
_Static_assert(sizeof(llgo_ble_gchr_t) == 18, "llgo_ble_gchr_t ABI drifted");
_Static_assert(__builtin_offsetof(llgo_ble_gchr_t, props) == 17, "gchr props offset");

/* The Go task fills the staging at llgo_ble_gatts_declare; the host task
 * builds the ble_gatt_svc_def tree from it when the command runs. One writer
 * per side, ordered by the command ring's release/acquire - the same shape
 * as the write staging above. */
static llgo_ble_gsvc_t s_decl_svcs[BW_GSVC_MAX];
static int32_t s_decl_svc_n;
static llgo_ble_gchr_t s_decl_chrs[BW_GCHR_MAX];
static int32_t s_decl_chr_n;
static uint32_t s_gatts_op;      /* the commit's operation id */
static uint8_t s_gatts_declared; /* the attribute table is live */

typedef struct {
    uint8_t uuid_len;
    uint8_t uuid[16];
    uint8_t props;
    uint8_t subs;         /* the CCCD's current value (bit0 notify, bit1
                           * indicate): what the SUBSCRIBE event reports */
    uint16_t handle;      /* the value attribute handle, filled by NimBLE at
                           * registration through the tree's val_handle */
    uint16_t cccd_handle; /* value handle + 1: NimBLE places the auto CCCD
                           * right after the value attribute */
    uint16_t value_len;
    uint8_t value[BW_DATA_MAX];
} bw_gchr_t;

static bw_gchr_t s_gchrs[BW_GCHR_MAX];
static int32_t s_gchr_n;
static uint8_t s_gatts_building; /* a registration is in flight */

/* The critical section guarding the value caches and the CCCD states: the Go
 * task writes them (llgo_ble_gatts_value), the host task reads and writes
 * them (the access callback). Short memcpys only. */
static portMUX_TYPE s_gatts_mux = portMUX_INITIALIZER_UNLOCKED;
#define BW_GATTS_LOCK()   taskENTER_CRITICAL(&s_gatts_mux)
#define BW_GATTS_UNLOCK() taskEXIT_CRITICAL(&s_gatts_mux)

/* The ble_gatt_svc_def tree NimBLE registers, built once per commit, wholly
 * static: the service and characteristic definition arrays, and the UUID
 * storage the definitions point into. */
static struct ble_gatt_svc_def s_gatt_defs[BW_GSVC_MAX + 1];
/* One zeroed terminator per service: the services' characteristic arrays are
 * slices of this one array, and each slice must end in its own zero entry or
 * the next service's first characteristic reads as a continuation. */
static struct ble_gatt_chr_def s_gatt_chrs[BW_GCHR_MAX + BW_GSVC_MAX];
static ble_uuid16_t s_gatt_uuid16[BW_GSVC_MAX + BW_GCHR_MAX];
static ble_uuid128_t s_gatt_uuid128[BW_GSVC_MAX + BW_GCHR_MAX];
static int32_t s_gatt_uuid16_n, s_gatt_uuid128_n;

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

/* The payload of a server-side notification being submitted (CMD_GATTS_NOTIFY).
 * Copied out of Go memory at submission; the host task reads it after
 * acquiring the command's index - the same ownership pattern as s_wr_data. */
static uint8_t s_push_data[BW_DATA_MAX];
static uint16_t s_push_len;

/* Written by the host task only (GAP callbacks and the command handler). */
static volatile int s_inited, s_synced;
static uint8_t s_own_type;
static uint32_t s_disc_op; /* op id of the running discovery, 0 = none */
static uint32_t s_conn_op; /* op id of the running connect attempt, 0 = none */

/* The address-carrying form, for the peripheral connect event. addr may be
 * NULL; the plain form delegates here with a NULL addr. */
static void bw_ctrl_push_addr(int32_t kind, uint32_t op_id, uint32_t conn_id, int32_t status,
                              const uint8_t *addr, uint8_t addr_type)
{
    uint32_t w = s_ctrl_wr;                      /* producer-owned */
    if ((uint32_t)(w - BW_ACQUIRE(s_ctrl_rd)) >= BW_CTRL_SLOTS) {
        __atomic_store_n(&s_ctrl_dropped, s_ctrl_dropped + 1, __ATOMIC_RELAXED);
        return;
    }
    s_ctrl[w & (BW_CTRL_SLOTS - 1)] = (bw_ctrl_t){ kind, op_id, conn_id, status,
                                                   {0, 0, 0, 0, 0, 0}, 0 };
    if (addr != NULL) {
        memcpy(s_ctrl[w & (BW_CTRL_SLOTS - 1)].addr, addr, 6);
        s_ctrl[w & (BW_CTRL_SLOTS - 1)].addr_type = addr_type;
    }
    BW_PUBLISH(s_ctrl_wr, w + 1);                /* release: the slot is now visible */
}

static void bw_ctrl_push(int32_t kind, uint32_t op_id, uint32_t conn_id, int32_t status)
{
    bw_ctrl_push_addr(kind, op_id, conn_id, status, NULL, 0);
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

/* ---- peripheral connections: handle + generation ----
 *
 * The same packing and the same shared generation counter as the central
 * table, over the connections NimBLE's advertising accepted. Both roles can
 * run in one firmware; each connection lives in exactly one table. */

#define BW_PCONN_MAX 2

typedef struct {
    uint8_t used;
    uint32_t gen;
    uint16_t handle;
} bw_pconn_t;

static bw_pconn_t s_pconns[BW_PCONN_MAX];

static uint32_t bw_pconn_add(uint16_t handle)
{
    for (int i = 0; i < BW_PCONN_MAX; i++) {
        if (!s_pconns[i].used) {
            s_pconns[i].used = 1;
            s_pconns[i].handle = handle;
            s_pconns[i].gen = ++s_conn_gen;
            return (s_pconns[i].gen << 16) | handle;
        }
    }
    return 0;
}

static uint32_t bw_pconn_remove(uint16_t handle)
{
    for (int i = 0; i < BW_PCONN_MAX; i++) {
        if (s_pconns[i].used && s_pconns[i].handle == handle) {
            uint32_t id = (s_pconns[i].gen << 16) | handle;
            s_pconns[i].used = 0;
            return id;
        }
    }
    return handle;
}

static int bw_pconn_find(uint32_t conn_id)
{
    uint16_t handle = (uint16_t)(conn_id & 0xFFFF);
    uint32_t gen = conn_id >> 16;
    for (int i = 0; i < BW_PCONN_MAX; i++) {
        if (s_pconns[i].used && s_pconns[i].handle == handle && s_pconns[i].gen == gen) {
            return (int)handle;
        }
    }
    return -1;
}

/* The packed id a live handle is registered under, for the peripheral
 * payload records (writes, subscriptions) that carry a bare handle - same
 * shape and same visibility contract as bw_conn_id_of. */
static uint32_t bw_pconn_id_of(uint16_t handle)
{
    for (int i = 0; i < BW_PCONN_MAX; i++) {
        if (s_pconns[i].used && s_pconns[i].handle == handle) {
            return (s_pconns[i].gen << 16) | handle;
        }
    }
    return handle;
}

static int s_pconn_any(void)
{
    for (int i = 0; i < BW_PCONN_MAX; i++) {
        if (s_pconns[i].used) {
            return 1;
        }
    }
    return 0;
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

/* ---- peripheral callbacks (host task context) ---- */

/* The gap event handler of the advertisement. Registered by CMD_ADV_START,
 * it sees only the connections the advertisement accepted - the central
 * side's procedures run through bw_on_gap below, so the two roles never
 * share a callback's conn table. */
static int bw_on_gap_periph(struct ble_gap_event *ev, void *arg)
{
    (void)arg;
    switch (ev->type) {
    case BLE_GAP_EVENT_CONNECT: {
        /* A connectable advertisement's life ends here; a StopAdvertising
         * after this reports its refusal honestly through EV_ADV_DONE. */
        s_adv_running = 0;
        if (ev->connect.status != 0) {
            return 0;
        }
        uint32_t id = bw_pconn_add(ev->connect.conn_handle);
        if (id == 0) {
            /* Connection table full: drop the link rather than serve a
             * connection the application cannot address. */
            ble_gap_terminate(ev->connect.conn_handle, BLE_ERR_REM_USER_CONN_TERM);
            return 0;
        }
        static struct ble_gap_conn_desc s_conn_desc; /* host task only */
        if (ble_gap_conn_find(ev->connect.conn_handle, &s_conn_desc) == 0) {
            bw_ctrl_push_addr(EV_PERIPH_CONNECT, 0, id, 0,
                              s_conn_desc.peer_id_addr.val,
                              s_conn_desc.peer_id_addr.type);
        } else {
            bw_ctrl_push_addr(EV_PERIPH_CONNECT, 0, id, 0, NULL, 0);
        }
        return 0;
    }
    case BLE_GAP_EVENT_DISCONNECT: {
        uint32_t id = bw_pconn_remove(ev->disconnect.conn.conn_handle);
        BW_GATTS_LOCK();
        for (int i = 0; i < s_gchr_n; i++) {
            s_gchrs[i].subs = 0;
        }
        BW_GATTS_UNLOCK();
        bw_ctrl_push(EV_PERIPH_DISCONNECT, 0, id, ev->disconnect.reason);
        return 0;
    }
    case BLE_GAP_EVENT_SUBSCRIBE: {
        /* The CCCD gate of a notify/indicate characteristic. NimBLE manages
         * the descriptor itself (the access callback never sees its writes);
         * attr_handle is the characteristic's value handle - exactly what
         * EvSubscribed carries. */
        int idx = -1;
        for (int i = 0; i < s_gchr_n; i++) {
            if (s_gchrs[i].handle == ev->subscribe.attr_handle) {
                idx = i;
                break;
            }
        }
        if (idx < 0) {
            return 0;
        }
        uint8_t cccd = (uint8_t)((ev->subscribe.cur_notify ? 1 : 0) |
                                 (ev->subscribe.cur_indicate ? 2 : 0));
        BW_GATTS_LOCK();
        s_gchrs[idx].subs = cccd;
        BW_GATTS_UNLOCK();
        bw_periph_push(EV_SUBSCRIBED, ev->subscribe.conn_handle,
                       ev->subscribe.attr_handle, (int32_t)cccd, NULL, 0);
        return 0;
    }
    case BLE_GAP_EVENT_ADV_COMPLETE:
        s_adv_running = 0;
        return 0;
    default:
        /* MTU (the exchange the central starts is answered inside the stack),
         * PHY updates, ... - nothing here carries a record the package's
         * surface needs. */
        return 0;
    }
}

/* The access callback of every characteristic the tree declares. arg is the
 * characteristic's table index; reads are answered from the value cache
 * synchronously, writes land in the cache (and the peripheral ring) under
 * the gatts critical section. NimBLE chunks long reads itself, so the
 * callback only ever hands out the whole cached value. */
static int bw_gatts_access(uint16_t conn_handle, uint16_t attr_handle,
                           struct ble_gatt_access_ctxt *ctx, void *arg)
{
    int idx = (int)(intptr_t)arg;
    if (idx < 0 || idx >= s_gchr_n || s_gchrs[idx].handle != attr_handle) {
        return BLE_ATT_ERR_UNLIKELY;
    }
    if (ctx->op == BLE_GATT_ACCESS_OP_READ_CHR) {
        BW_GATTS_LOCK();
        uint16_t len = s_gchrs[idx].value_len;
        int rc = os_mbuf_append(ctx->om, s_gchrs[idx].value, len);
        BW_GATTS_UNLOCK();
        return rc == 0 ? 0 : BLE_ATT_ERR_INSUFFICIENT_RES;
    }
    if (ctx->op == BLE_GATT_ACCESS_OP_WRITE_CHR) {
        if (OS_MBUF_PKTLEN(ctx->om) > BW_DATA_MAX) {
            /* Truncation would invent data; the ATT error is the honest
             * refusal (the Go-side bound is the same BW_DATA_MAX). */
            return BLE_ATT_ERR_INSUFFICIENT_RES;
        }
        static uint8_t s_write_buf[BW_DATA_MAX]; /* host task only */
        uint16_t len = 0;
        int rc = ble_hs_mbuf_to_flat(ctx->om, s_write_buf, BW_DATA_MAX, &len);
        if (rc != 0) {
            return BLE_ATT_ERR_UNLIKELY;
        }
        BW_GATTS_LOCK();
        memcpy(s_gchrs[idx].value, s_write_buf, len);
        s_gchrs[idx].value_len = len;
        BW_GATTS_UNLOCK();
        bw_periph_push(EV_CHAR_WRITE, conn_handle, attr_handle, 0, s_write_buf, len);
        return 0;
    }
    return BLE_ATT_ERR_UNLIKELY;
}

/* The attribute table NimBLE registers: the ble_gatt_svc_def tree built from
 * the declaration staging, wholly static - the service and characteristic
 * definition arrays, the UUID storage they point into, and the zero
 * terminators the memset below provides. */
static const ble_uuid_t *bw_gatt_uuid_of(uint8_t len, const uint8_t *bytes)
{
    if (len == 2) {
        ble_uuid16_t *u = &s_gatt_uuid16[s_gatt_uuid16_n++];
        u->u.type = BLE_UUID_TYPE_16;
        u->value = (uint16_t)(bytes[0] | (bytes[1] << 8));
        return (const ble_uuid_t *)u;
    }
    ble_uuid128_t *u = &s_gatt_uuid128[s_gatt_uuid128_n++];
    u->u.type = BLE_UUID_TYPE_128;
    memcpy(u->value, bytes, 16); /* NimBLE keeps the wire byte order */
    return (const ble_uuid_t *)u;
}

static void bw_gatts_build_tree(void)
{
    memset(s_gatt_defs, 0, sizeof s_gatt_defs);
    memset(s_gatt_chrs, 0, sizeof s_gatt_chrs);
    s_gatt_uuid16_n = 0;
    s_gatt_uuid128_n = 0;
    int32_t ci = 0;
    int32_t chr_out = 0;
    for (int32_t i = 0; i < s_decl_svc_n; i++) {
        struct ble_gatt_svc_def *svc = &s_gatt_defs[i];
        svc->type = BLE_GATT_SVC_TYPE_PRIMARY;
        svc->uuid = bw_gatt_uuid_of(s_decl_svcs[i].uuid_len, s_decl_svcs[i].uuid);
        svc->characteristics = &s_gatt_chrs[chr_out];
        for (int32_t k = 0; k < (int32_t)s_decl_svcs[i].n_chars; k++, chr_out++, ci++) {
            struct ble_gatt_chr_def *chr = &s_gatt_chrs[chr_out];
            chr->uuid = bw_gatt_uuid_of(s_decl_chrs[ci].uuid_len, s_decl_chrs[ci].uuid);
            chr->access_cb = bw_gatts_access;
            /* arg and val_handle key on the REGISTRY index (ci): the
             * definition array carries per-service terminator gaps the
             * registry does not have. NimBLE fills the assigned value handle
             * right here, through the definition - the same record
             * llgo_ble_gatts_attr reads. */
            chr->arg = (void *)(intptr_t)ci;
            chr->flags = s_decl_chrs[ci].props; /* the ATT bits are NimBLE's flags */
            chr->val_handle = &s_gchrs[ci].handle;
        }
        chr_out++; /* this service's slice ends in the memset-zeroed entry */
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
    CMD_ADV_START = 10,
    CMD_ADV_STOP = 11,
    CMD_GATTS_DECLARE = 12,
    CMD_GATTS_NOTIFY = 13,
    CMD_GATTS_CLOSE = 14,
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
    uint8_t indicate;   /* CMD_GATTS_NOTIFY: indication, not notification */
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
    case CMD_ADV_START: {
        struct ble_gap_adv_params ap;
        memset(&ap, 0, sizeof ap);
        ap.conn_mode = s_adv_conn ? BLE_GAP_CONN_MODE_UND : BLE_GAP_CONN_MODE_NON;
        ap.disc_mode = BLE_GAP_DISC_MODE_GEN;
        uint32_t itvl = ((uint32_t)s_adv_itvl_ms * 8) / 5; /* 0.625 ms units */
        if (itvl < 32) {
            itvl = 32; /* 20 ms, the legacy minimum */
        }
        if (itvl > 16384) {
            itvl = 16384; /* 10.24 s, the legacy maximum */
        }
        ap.itvl_min = (uint16_t)itvl;
        ap.itvl_max = (uint16_t)itvl;
        int rc = ble_gap_adv_set_data(s_adv_data, (int)s_adv_data_len);
        if (rc == 0 && s_scan_rsp_len > 0) {
            rc = ble_gap_adv_rsp_set_data(s_scan_rsp, (int)s_scan_rsp_len);
        }
        if (rc == 0) {
            /* The advertisement runs until StopAdvertising or a connection;
             * the Go side carries no duration - an expiry would be a silent
             * end, and nothing in the API documents one. */
            rc = ble_gap_adv_start(s_own_type, NULL, BLE_HS_FOREVER, &ap,
                                   bw_on_gap_periph, NULL);
        }
        s_adv_running = rc == 0;
        bw_ctrl_push(EV_ADV_DONE, c->op_id, 0, rc);
        break;
    }
    case CMD_ADV_STOP:
        /* Nothing advertising (never started, or a connect ended it): the
         * operation completes here rather than dangling. */
        bw_ctrl_push(EV_ADV_DONE, c->op_id, 0,
                     s_adv_running ? ble_gap_adv_stop() : BLE_HS_EINVAL);
        s_adv_running = 0;
        break;
    case CMD_GATTS_DECLARE: {
        /* A connected peer or a running advertisement would have attributes
         * pulled out from under it. One registration at a time is the Go
         * side's s_gatts_building flag (set at submission); checking it here
         * too would trip on this very command. */
        int rc = 0;
        if (s_pconn_any() || s_adv_running) {
            rc = BLE_HS_EBUSY;
        } else {
            s_gatts_building = 1;
            s_gchr_n = 0;
            bw_gatts_build_tree();
            /* Fresh value caches: the declaration is the only writer until a
             * peer (or WriteCharValue) touches them. */
            for (int32_t i = 0; i < s_decl_chr_n; i++) {
                s_gchrs[i].value_len = 0;
                s_gchrs[i].handle = 0;
                s_gchrs[i].subs = 0;
                s_gchrs[i].cccd_handle = 0;
            }
            s_gchr_n = s_decl_chr_n;
#if MYNEWT_VAL(BLE_DYNAMIC_SERVICE)
            /* Synchronous: NimBLE registers the tree (and fills the val
             * handles through the definitions) before returning. This is the
             * one after-sync registration entry point the host offers; a
             * firmware without the option refuses the commit here instead of
             * pretending. */
            rc = ble_gatts_add_dynamic_svcs(s_gatt_defs);
#else
            rc = BLE_HS_ENOTSUP; /* firmware lacks CONFIG_BT_NIMBLE_DYNAMIC_SERVICE */
#endif
            if (rc == 0) {
                s_gatts_declared = 1;
                /* NimBLE places the auto-added CCCD right after the value
                 * attribute - the layout the GATT specification mandates. */
                for (int32_t i = 0; i < s_gchr_n; i++) {
                    if (s_gchrs[i].props & (BLE_GATT_CHR_F_NOTIFY | BLE_GATT_CHR_F_INDICATE)) {
                        s_gchrs[i].cccd_handle = (uint16_t)(s_gchrs[i].handle + 1);
                    }
                }
            }
            s_gatts_building = 0;
        }
        bw_ctrl_push(EV_SERVICES_DONE, c->op_id, 0, rc);
        break;
    }
    case CMD_GATTS_NOTIFY: {
        int handle = bw_pconn_find(c->conn_id);
        if (handle < 0) {
            bw_ctrl_push(EV_ERROR, c->op_id, c->conn_id, BLE_HS_ENOENT);
            break;
        }
        int rc;
        if (c->indicate) {
#if MYNEWT_VAL(BLE_GATTS)
            struct os_mbuf *om = ble_hs_mbuf_from_flat(s_push_data, s_push_len);
            if (om == NULL) {
                bw_ctrl_push(EV_ERROR, c->op_id, c->conn_id, BLE_HS_ENOMEM);
                break;
            }
            rc = ble_gatts_indicate_custom((uint16_t)handle, c->att_handle, om);
#else
            /* No GATT server in this firmware: the real entry point would
             * pull the ATT server's internals into the link. The refusal is
             * the observable the fallback path would have produced. */
            rc = BLE_HS_ENOTSUP;
#endif
        } else {
            struct os_mbuf *om = ble_hs_mbuf_from_flat(s_push_data, s_push_len);
            if (om == NULL) {
                bw_ctrl_push(EV_ERROR, c->op_id, c->conn_id, BLE_HS_ENOMEM);
                break;
            }
            rc = ble_gatts_notify_custom((uint16_t)handle, c->att_handle, om);
        }
        if (rc != 0) {
            bw_ctrl_push(EV_ERROR, c->op_id, c->conn_id, rc);
        }
        break;
    }
    case CMD_GATTS_CLOSE: {
        int handle = bw_pconn_find(c->conn_id);
        if (handle < 0) {
            bw_ctrl_push(EV_ERROR, 0, c->conn_id, BLE_HS_ENOENT);
        } else {
            int rc = ble_gap_terminate((uint16_t)handle, BLE_ERR_REM_USER_CONN_TERM);
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
    /* The peripheral half's in-flight state: an advertising operation or a
     * service registration that never completed has no stack left to complete
     * on, the connections are gone, and a fresh Init must not inherit them. */
    s_adv_op = 0;
    s_adv_running = 0;
    s_gatts_op = 0;
    s_gatts_building = 0;
    s_gatts_declared = 0;
    s_decl_svc_n = 0;
    s_decl_chr_n = 0;
    s_gchr_n = 0;
    for (int i = 0; i < BW_PCONN_MAX; i++) {
        s_pconns[i].used = 0;
    }
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
    uint32_t periph_d = BW_ACQUIRE(s_periph_dropped);
    if (ctrl_d != s_ctrl_dropped_seen || adv_d != s_adv_dropped_seen ||
        notif_d != s_notif_dropped_seen || periph_d != s_periph_dropped_seen) {
        out->kind = EV_OVERFLOW;
        out->status = (int32_t)((ctrl_d - s_ctrl_dropped_seen) +
                                (adv_d - s_adv_dropped_seen) +
                                (notif_d - s_notif_dropped_seen) +
                                (periph_d - s_periph_dropped_seen));
        s_ctrl_dropped_seen = ctrl_d;
        s_adv_dropped_seen = adv_d;
        s_notif_dropped_seen = notif_d;
        s_periph_dropped_seen = periph_d;
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
        if (c.kind == EV_PERIPH_CONNECT) {
            /* The peer address of a peripheral connect: filled by the
             * producer before the release publish, visible after the
             * acquire above - the rings' standing contract. */
            memcpy(out->addr, c.addr, 6);
            out->addr_type = c.addr_type;
        }
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

    if (BW_ACQUIRE(s_periph_wr) != s_periph_rd) {
        /* The peripheral tier, drained right after the control tier: a peer
         * action is actionable exactly when its connection record has been
         * seen, so it outranks the central-side payload tiers. Read in place
         * like the notification ring - a local copy would put a whole slot on
         * the Go task's (fiber) stack. */
        const bw_periph_t *p = &s_periph[s_periph_rd & (BW_PERIPH_SLOTS - 1)];
        out->kind = p->kind;
        out->conn_id = bw_pconn_id_of(p->conn_handle);
        out->att_handle = p->attr_handle;
        out->status = p->status;
        out->data_len = p->len;
        memcpy(out->data, p->data, p->len);
        BW_PUBLISH(s_periph_rd, s_periph_rd + 1); /* release: the slot is free again */
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

/* ---- peripheral (Go task) ----
 *
 * Same contract as the central commands: 0 means the command was accepted
 * into the host queue and the outcome arrives through the mailbox
 * (EV_ADV_DONE, EV_SERVICES_DONE; a send the stack refused after queueing
 * arrives as EV_ERROR). The value-cache write and the accessors run on the
 * Go task against host-task-published state, under the gatts critical
 * section. */

int32_t llgo_ble_adv_start(const uint8_t *adv, int32_t adv_len, const uint8_t *rsp,
                           int32_t rsp_len, int32_t itvl_ms, int32_t connectable,
                           uint32_t op_id)
{
    bw_cmd_t c;
    if (adv_len < 0 || adv_len > BW_ADV_MAX || rsp_len < 0 || rsp_len > BW_ADV_MAX) {
        return BLE_HS_EMSGSIZE;
    }
    if ((adv_len > 0 && adv == NULL) || (rsp_len > 0 && rsp == NULL)) {
        return BLE_HS_EINVAL;
    }
    if (itvl_ms <= 0) {
        itvl_ms = 100;
    }
    if (itvl_ms < 20) {
        itvl_ms = 20;
    }
    if (itvl_ms > 10240) {
        itvl_ms = 10240;
    }
    if (s_adv_op != 0) {
        return BLE_HS_EALREADY; /* one advertisement lifecycle at a time */
    }
    s_adv_op = op_id;
    memcpy(s_adv_data, adv, (size_t)adv_len);
    s_adv_data_len = (uint16_t)adv_len;
    memcpy(s_scan_rsp, rsp, (size_t)rsp_len);
    s_scan_rsp_len = (uint16_t)rsp_len;
    s_adv_conn = connectable ? 1 : 0;
    s_adv_itvl_ms = (uint16_t)itvl_ms;
    memset(&c, 0, sizeof c);
    c.op = CMD_ADV_START;
    c.op_id = op_id;
    int32_t rc = bw_submit(&c);
    if (rc != 0) {
        s_adv_op = 0;
    }
    return rc;
}

int32_t llgo_ble_adv_stop(uint32_t op_id)
{
    bw_cmd_t c;
    if (s_adv_op != 0) {
        return BLE_HS_EALREADY;
    }
    s_adv_op = op_id;
    memset(&c, 0, sizeof c);
    c.op = CMD_ADV_STOP;
    c.op_id = op_id;
    int32_t rc = bw_submit(&c);
    if (rc != 0) {
        s_adv_op = 0;
    }
    return rc;
}

int32_t llgo_ble_gatts_declare(const llgo_ble_gsvc_t *svcs, int32_t nsvc,
                               const llgo_ble_gchr_t *chrs, int32_t nchr, uint32_t op_id)
{
    bw_cmd_t c;
    if (svcs == NULL || chrs == NULL || nsvc <= 0 || nchr <= 0 ||
        nsvc > BW_GSVC_MAX || nchr > BW_GCHR_MAX) {
        return BLE_HS_EINVAL;
    }
    if (s_gatts_declared || s_gatts_building) {
        /* The attribute table is built once per Init; rebuilding is a
         * Deinit-and-restart operation (the Go side refuses a second commit
         * the same way). */
        return BLE_HS_EALREADY;
    }
    /* Serialize the declaration into the staging tables - the host task
     * builds the registration tree from them when the command runs. The
     * flat-array consistency check mirrors the Go side's own. */
    s_decl_svc_n = 0;
    s_decl_chr_n = 0;
    int32_t ci = 0;
    for (int32_t i = 0; i < nsvc; i++) {
        if (s_decl_chr_n + (int32_t)svcs[i].n_chars > nchr) {
            return BLE_HS_EINVAL;
        }
        s_decl_svcs[s_decl_svc_n++] = svcs[i];
        for (int32_t k = 0; k < (int32_t)svcs[i].n_chars; k++, ci++) {
            s_decl_chrs[s_decl_chr_n++] = chrs[ci];
        }
    }
    if (ci != nchr) {
        s_decl_svc_n = 0;
        s_decl_chr_n = 0;
        return BLE_HS_EINVAL;
    }
    s_gatts_building = 1;
    memset(&c, 0, sizeof c);
    c.op = CMD_GATTS_DECLARE;
    c.op_id = op_id;
    int32_t rc = bw_submit(&c);
    if (rc != 0) {
        s_gatts_building = 0;
        s_decl_svc_n = 0;
        s_decl_chr_n = 0;
    }
    return rc;
}

int32_t llgo_ble_gatts_attr(int32_t idx, uint16_t *out)
{
    if (out == NULL || idx < 0 || idx >= s_gchr_n) {
        return 0;
    }
    *out = s_gchrs[idx].handle;
    return 1;
}

/* Replaces a characteristic's cached value (WriteCharValue): the copy runs
 * under the gatts critical section against the read the host task may be
 * answering from the same cache right now. */
int32_t llgo_ble_gatts_value(uint16_t handle, const uint8_t *data, int32_t len)
{
    if (handle == 0 || len < 0 || (len > 0 && data == NULL)) {
        return BLE_HS_EINVAL;
    }
    if (len > BW_DATA_MAX) {
        return BLE_HS_EMSGSIZE;
    }
    int idx = -1;
    for (int i = 0; i < s_gchr_n; i++) {
        if (s_gchrs[i].handle == handle) {
            idx = i;
            break;
        }
    }
    if (idx < 0) {
        return BLE_HS_ENOENT;
    }
    BW_GATTS_LOCK();
    if (len > 0) {
        memcpy(s_gchrs[idx].value, data, (size_t)len);
    }
    s_gchrs[idx].value_len = (uint16_t)len;
    BW_GATTS_UNLOCK();
    return 0;
}

int32_t llgo_ble_gatts_notify(uint32_t conn_id, uint16_t handle, const uint8_t *data,
                              int32_t len, int32_t indicate)
{
    bw_cmd_t c;
    if (handle == 0 || len < 0 || (len > 0 && data == NULL)) {
        return BLE_HS_EINVAL;
    }
    if (len > BW_DATA_MAX) {
        return BLE_HS_EMSGSIZE;
    }
    int known = 0;
    for (int i = 0; i < s_gchr_n; i++) {
        if (s_gchrs[i].handle == handle) {
            known = 1;
            break;
        }
    }
    if (!known) {
        return BLE_HS_ENOENT;
    }
    /* Copy out of Go memory now: the host task runs the send later. */
    if (len > 0) {
        memcpy(s_push_data, data, (size_t)len);
    }
    s_push_len = (uint16_t)len;
    memset(&c, 0, sizeof c);
    c.op = CMD_GATTS_NOTIFY;
    c.op_id = 0; /* fire-and-forget: the record carries no operation */
    c.conn_id = conn_id;
    c.att_handle = handle;
    c.indicate = indicate ? 1 : 0;
    return bw_submit(&c);
}

int32_t llgo_ble_gatts_close(uint32_t conn_id)
{
    bw_cmd_t c;
    memset(&c, 0, sizeof c);
    c.op = CMD_GATTS_CLOSE;
    c.conn_id = conn_id;
    return bw_submit(&c);
}

/* The pairing posture. NimBLE reads these fields when a pairing starts, so
 * they are settable after sync as long as no pairing is in flight - the same
 * call-once-after-Init contract the Go side documents. A fixed passkey has
 * no static knob on this stack: NimBLE delivers passkey entry through the
 * GAP passkey-action events, which this package does not expose yet - the
 * refusal is the honest answer rather than a silently ignored passkey. */
int32_t llgo_ble_security(int32_t io_cap, int32_t mitm, int32_t bond, uint32_t passkey,
                          int32_t key_size)
{
    if (io_cap < 0 || io_cap > BLE_HS_IO_KEYBOARD_DISPLAY) {
        return BLE_HS_EINVAL;
    }
    if (passkey != 0) {
        return BLE_HS_ENOTSUP;
    }
    if (key_size != 0 && key_size != 16) {
        /* NimBLE has no runtime key-size knob (the firmware's Kconfig owns
         * it, default 16); refusing a smaller request is the honest answer
         * rather than a silently ignored parameter. */
        return BLE_HS_ENOTSUP;
    }
    ble_hs_cfg.sm_io_cap = (uint8_t)io_cap;
    ble_hs_cfg.sm_bonding = bond ? 1 : 0;
    ble_hs_cfg.sm_mitm = mitm ? 1 : 0;
    ble_hs_cfg.sm_sc = 1; /* secure connections wherever the peer supports them */
    ble_hs_cfg.sm_our_key_dist = BLE_SM_PAIR_KEY_DIST_ENC | BLE_SM_PAIR_KEY_DIST_ID;
    ble_hs_cfg.sm_their_key_dist = BLE_SM_PAIR_KEY_DIST_ENC | BLE_SM_PAIR_KEY_DIST_ID;
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

typedef struct {
    uint8_t uuid_len;
    uint8_t uuid[16];
    uint8_t _pad;
    uint16_t n_chars;
} llgo_ble_gsvc_t;

typedef struct {
    uint8_t uuid_len;
    uint8_t uuid[16];
    uint8_t props;
} llgo_ble_gchr_t;

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

/* The peripheral half: the same names the Go side links, all refusing
 * honestly. */
int32_t llgo_ble_adv_start(const uint8_t *adv, int32_t adv_len, const uint8_t *rsp,
                           int32_t rsp_len, int32_t itvl_ms, int32_t connectable,
                           uint32_t op_id)
{
    (void)adv; (void)adv_len; (void)rsp; (void)rsp_len;
    (void)itvl_ms; (void)connectable; (void)op_id;
    return ESP_ERR_NOT_SUPPORTED;
}
int32_t llgo_ble_adv_stop(uint32_t op_id) { (void)op_id; return ESP_ERR_NOT_SUPPORTED; }
int32_t llgo_ble_gatts_declare(const llgo_ble_gsvc_t *svcs, int32_t nsvc,
                               const llgo_ble_gchr_t *chrs, int32_t nchr, uint32_t op_id)
{
    (void)svcs; (void)nsvc; (void)chrs; (void)nchr; (void)op_id;
    return ESP_ERR_NOT_SUPPORTED;
}
int32_t llgo_ble_gatts_attr(int32_t idx, uint16_t *out)
{
    (void)idx; (void)out;
    return 0;
}
int32_t llgo_ble_gatts_value(uint16_t handle, const uint8_t *data, int32_t len)
{
    (void)handle; (void)data; (void)len;
    return ESP_ERR_NOT_SUPPORTED;
}
int32_t llgo_ble_gatts_notify(uint32_t conn_id, uint16_t handle, const uint8_t *data,
                              int32_t len, int32_t indicate)
{
    (void)conn_id; (void)handle; (void)data; (void)len; (void)indicate;
    return ESP_ERR_NOT_SUPPORTED;
}
int32_t llgo_ble_gatts_close(uint32_t conn_id) { (void)conn_id; return ESP_ERR_NOT_SUPPORTED; }
int32_t llgo_ble_security(int32_t io_cap, int32_t mitm, int32_t bond, uint32_t passkey,
                          int32_t key_size)
{
    (void)io_cap; (void)mitm; (void)bond; (void)passkey; (void)key_size;
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
