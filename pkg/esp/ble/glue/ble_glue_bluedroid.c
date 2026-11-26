/* Bluedroid backend of pkg/esp/ble: the C glue between llgo-compiled Go code
 * and the ESP-IDF Bluedroid host, implementing the exact same C ABI as
 * glue/ble_glue_nimble.c:
 * a BLE central (scan / connect / disconnect) plus a GATT client (discovery,
 * read, write, notification subscription), with the stack's asynchronous
 * results delivered through a bounded, three-tier mailbox.
 *
 * This file is a TEMPLATE: copy it into your project's main component so the
 * IDF build compiles it there (the llgo package never builds C code). The
 * boundary rules below are the same rules the NimBLE glue implements; the
 * differences are all on the stack side of that boundary.
 *
 * Rules that keep the llgo fiber runtime safe on the other side of the
 * boundary (identical to the NimBLE glue):
 *   (1) integer-and-pointer-only signatures - no float value crosses the
 *       Go<->C link;
 *   (2) no large stack buffers. Go goroutines run on 2 KiB fiber stacks
 *       (targets/esp32s31-idf.json default-stack-size; the main fiber gets
 *       8 KiB), so every record that travels - scan results, mailbox slots -
 *       lives in static storage; the flat llgo_ble_event_t is the only shape
 *       that crosses, by pointer, with scalar and fixed-size-array fields only;
 *   (3) Bluedroid never calls back into Go. Every application callback (GAP
 *       and GATTC) runs on the stack's single BTC task (btc_task.c runs one
 *       osi thread and dispatches every profile callback on it), and the llgo
 *       fiber scheduler is single-task global state - an entry from another
 *       FreeRTOS task would corrupt it. The callbacks copy into the mailbox
 *       instead; Go polls it from its own goroutine;
 *   (4) Go never blocks in a control path. The esp_ble_gap_* and
 *       esp_ble_gattc_* calls are thread-safe asynchronous submissions by
 *       construction: each one checks the stack state, deep-copies its
 *       arguments (btc_transfer_context, with per-action deep-copy helpers for
 *       pointed-to buffers) and posts to the BTC task's queue before
 *       returning. So - unlike the NimBLE glue, which had to route every GAP
 *       call through the host task's event queue - this glue has no command
 *       ring at all: a control call runs its esp_ble_* call right on the Go
 *       task and returns. The microsecond the deep copy costs is paid in the
 *       caller's context; no semaphore is ever taken.
 *
 * What differs from the NimBLE stack, and how this glue absorbs it (the Go
 * side of the ABI is byte-for-byte the same, including the 188-byte
 * llgo_ble_event_t and every event kind):
 *
 *   - Bring-up is a synchronous block (llgo_ble_init runs
 *     controller init/enable + esp_bluedroid_init/enable on the calling
 *     task; v6 has no stack-level callback - esp_bluedroid_register_callback
 *     was removed - and esp_bluedroid_enable returns only once the stack is
 *     up). The one asynchronous step left, esp_ble_gattc_app_register, is
 *     completed by ESP_GATTC_REG_EVT, and that event is what pushes EV_SYNC -
 *     the same "stack ready" signal the NimBLE sync callback produced.
 *   - There is no reset event in the Bluedroid API. NimBLE's host-reset
 *     signal has no equivalent (a controller-level failure shows up as
 *     per-connection disconnects), so EV_RESET is never produced by this
 *     glue; a program subscribed to it just never sees one.
 *   - Connecting must go through esp_ble_gattc_enh_open (the plain
 *     esp_ble_gattc_open only exists in BLE 4.2-feature builds), with the
 *     legacy-advertising is_aux=false form; the stack resolves the actual
 *     HCI procedure from its compiled feature set.
 *   - Discovery folds two NimBLE procedures into one stack procedure:
 *     esp_ble_gattc_search_service fills the service table, and the
 *     characteristics are already in the stack's cache when the search
 *     completes - the glue reads them out right there, in the search
 *     completion callback. The read is a BTC-task job on purpose: the cache
 *     is freed by the BTC task on a disconnect (and rebuilt by it on a
 *     re-search), so reading it from any other task would race a free and
 *     touch freed memory, while on the BTC task the read is serialized with
 *     every later cache mutation by construction. llgo_ble_disc_chrs - the
 *     Go task's call - then only slices the static characteristic table down
 *     to the service's handle range and reports the same EV_DISC_DONE the
 *     NimBLE discovery produced. Same call sequence in Go, same tables, one
 *     mailbox record per procedure.
 *   - Subscribing (from the Go side) is an ordinary write to the CCCD handle -
 *     the package has no subscribe symbol. NimBLE delivers whatever the peer
 *     pushes after that write; Bluedroid only delivers notifications for
 *     characteristic handles registered with
 *     esp_ble_gattc_register_for_notify, so this glue registers every
 *     notify/indicate-capable characteristic when the search completes (the
 *     cache has to exist first, as the API requires). The CCCD write
 *     still goes out as a plain write (esp_ble_gattc_write_char writes any
 *     writable handle), and the peer's CCCD gate stays the only thing that
 *     turns payloads on and off.
 *   - The MTU exchange is not automatic: the glue sets the local MTU at init
 *     (esp_ble_gatt_set_local_mtu, the value the package documents) and asks
 *     for the exchange as soon as a connection comes up
 *     (esp_ble_gattc_send_mtu_req), which is what CFG_MTU / EV_MTU report.
 *   - A write without response has no completion event (there is no
 *     no-response function either; it is the same esp_ble_gattc_write_char
 *     with ESP_GATT_WRITE_TYPE_NO_RSP), so the glue reports EV_WRITE_DONE
 *     from the submission itself - exactly the NimBLE behavior, where the
 *     write command's submission result is the whole outcome.
 *   - Status codes in mailbox records pass through in the stack's own
 *     namespace: esp_bt_status_t / esp_gatt_status_t / HCI reasons instead of
 *     BLE_HS_* (Backend() names the stack, and the Go error layer tags the
 *     namespace). The codes this glue produces itself - its own refusals and
 *     the two conditions the Bluedroid API has no code for (a discovery
 *     record that did not fit a table, and a connect attempt the glue's
 *     timeout timer cancelled) - reuse the NimBLE numeric space so the Go
 *     contract docs stay true for both backends; they are marked inline.
 *
 * Mailbox design: four single-producer/single-consumer rings (the ctrl ring
 * is multi-writer inside its critical section - see below).
 *
 *   ctrl ring  - sync, scan-done, connect, disconnect (both roles), GATT
 *                procedure completions, advertising and service-registration
 *                completions, MTU reports, command rejections; 8 slots, no
 *                payload beyond codes and a connect event's peer address.
 *                These events are what a program's state machine depends on,
 *                so they never compete with a scan flood or a notification
 *                storm for space.
 *   periph ring - what a central does to this device's attributes while we
 *                serve it: characteristic writes and client-configuration
 *                (CCCD) writes; 8 slots of connection id + handle + status +
 *                up to BW_DATA_MAX bytes of payload. Its own ring keeps a
 *                write flood off the control tier, and the poll drains ctrl
 *                before periph, so an EV_PERIPH_CONNECT is never left queued
 *                behind the payload records it makes actionable.
 *   notif ring - notifications and indications; 8 slots of connection id +
 *                value handle + flags + up to BW_DATA_MAX bytes of payload.
 *                Their own ring keeps them off the control tier.
 *   adv ring   - scan results only; 16 slots of address + RSSI + up to 31
 *                bytes of advertising data (the legacy maximum, which is what
 *                a legacy report carries).
 *
 * Overflow drops the newest record of that ring - or a notification whose
 * payload does not fit one slot - and is reported once as an EV_OVERFLOW slot
 * carrying the dropped count, so loss is never silent but also never stalls
 * the BTC task.
 *
 * Cross-task visibility is an explicit release/acquire contract on the ring
 * indexes (the same discipline wifi_glue.c implements): the producer fills its
 * slot fields and then publishes the write index with a release store
 * (BW_PUBLISH); the consumer acquires the write index before touching the slot
 * and publishes the read index with a release store after copying it out
 * (BW_ACQUIRE). No read-modify-write anywhere: RV32IMC has no A extension, so
 * an RMW would drag in libatomic. A dropped counter has a single writer (its
 * ring's producer), maintained with a plain read plus a relaxed atomic store;
 * the consumer keeps its own last-seen snapshot instead of clearing the
 * producer's counter.
 *
 * The one place this glue departs from the NimBLE ring discipline is the ctrl
 * ring's writer count. NimBLE had exactly one producer (the host task);
 * Bluedroid has several, because a few completions are produced synchronously
 * on the caller's task: the characteristic-discovery slice reports from the
 * Go task, a no-response write reports from the Go task, and the
 * connect-timeout timer reports from the esp_timer task - all while the BTC
 * task may be pushing a scan result or a notification. Those pushes therefore
 * take a short FreeRTOS critical section, which serializes every writer. The
 * same critical section owns the few words that both sides of the boundary
 * claim and clear: the scan and connect operation ids (a completion claims
 * its id - read plus clear - atomically against a concurrent submission, so
 * an id is never claimed twice or wiped by a stale completion from a
 * superseded attempt), and the GATT procedure context (so a completion and a
 * concurrently starting procedure can never attribute the same record
 * twice, and a teardown that ends a procedure publishes the completion its
 * Go-side waiter is still owed). The release/acquire contract still governs
 * what the consumer may touch and in which order. The adv and notif rings
 * keep the strict single-producer shape (only the BTC task writes them).
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
 * without Bluedroid the file still defines every symbol, returning
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

#if defined(CONFIG_BT_ENABLED) && !defined(CONFIG_BT_BLUEDROID_ENABLED)
#error "glue/ble_glue_bluedroid.c is the Bluedroid backend, but this build enables a different BT host stack (NimBLE?). Sync glue/ble_glue_nimble.c into main/ instead."
#elif defined(CONFIG_BT_ENABLED) && defined(CONFIG_BT_NIMBLE_ENABLED)
#error "glue/ble_glue_bluedroid.c needs the Bluedroid host stack, but this build also enables NimBLE; the two hosts are mutually exclusive. Configure the BT_HOST choice for one stack and sync the matching glue file."
#elif defined(CONFIG_BT_ENABLED) && defined(CONFIG_BT_BLUEDROID_ENABLED)

#include "esp_bt.h"
#include "esp_bt_main.h"
#include "esp_bt_device.h"
#include "esp_gatt_common_api.h"
#include "esp_gap_ble_api.h"
#include "esp_gattc_api.h"
#include "esp_gatts_api.h"

/* ---- mailbox ---- */

enum {
    EV_NONE = 0,
    EV_SYNC = 1,
    EV_RESET = 2,   /* never produced: Bluedroid has no reset event */
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

/* The NimBLE numeric space of the codes this glue produces itself. The stack
 * reports its own esp_bt_status_t / esp_gatt_status_t / HCI values in the
 * mailbox (Backend() names the namespace); these few are the glue's own
 * refusals and the conditions the Bluedroid API has no code for, kept at the
 * values ble.go's error docs quote so both backends speak the same contract
 * for them. */
#define BW_EALREADY 0x02 /* a procedure of that kind is already running */
#define BW_EINVAL   0x03 /* bad parameter */
#define BW_EMSGSIZE 0x04 /* a result is larger than one event carries */
#define BW_ENOENT   0x05 /* no such entry (stale ConnID or OpID) */
#define BW_ENOMEM   0x06 /* no room for the connection (the connect table is full) */
#define BW_ENOTCONN 0x07 /* not connected */
#define BW_ETIMEOUT 0x0D /* the glue's connect timeout fired */

/* The largest ATT payload one record carries. Mirrored by MaxData in ble.go,
 * where the reasoning lives: the bound is the goroutine stack the record
 * travels on (2 KiB fibers on this target), not the ATT MTU. A notification
 * that does not fit is dropped whole and counted; a read that does not fit is
 * reported as BW_EMSGSIZE. */
#define BW_DATA_MAX 128

/* The one record shape that crosses into Go. Field order and widths are an
 * ABI: pkg/esp/ble's Event struct mirrors it byte for byte (asserted below),
 * and the NimBLE glue pins the same layout. */
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
 * ble.go) and the NimBLE glue fills. The UUID travels as a width plus 16
 * little-endian bytes - the byte order the ATT PDU carries. Bluedroid's
 * esp_bt_uuid_t.uuid128 keeps those same PDU bytes (the stack debug-prints
 * uuid128[15] first, i.e. the array is little-endian on the wire order, like
 * NimBLE's ble_uuid128_t), so the two backends fill identical records. */
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

/* The local GATT server's declaration records, the wire form CommitServices
 * passes to llgo_ble_gatts_declare. The Go side mirrors them byte for byte
 * (gsvcWire/gchrWire in ble.go, pinned by ble_test.go): a UUID as width plus
 * 16 little-endian bytes, the per-service characteristic count, the property
 * bitmask. */
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
 * per ring - except the ctrl ring, whose writers the critical section in
 * bw_ctrl_push serializes (see the file header): the BTC task callbacks, the
 * Go task's synchronous completions, and the connect-timeout timer. Visibility
 * across tasks is the release/acquire contract documented in the file header. */
#define BW_PUBLISH(v, val) __atomic_store_n(&(v), (val), __ATOMIC_RELEASE)
#define BW_ACQUIRE(v)      __atomic_load_n(&(v), __ATOMIC_ACQUIRE)

static bw_ctrl_t s_ctrl[BW_CTRL_SLOTS];
static uint32_t s_ctrl_wr, s_ctrl_rd; /* wr: producers (critical section), rd: consumer */
static uint32_t s_ctrl_dropped;       /* producers (critical section) */
static uint32_t s_ctrl_dropped_seen;  /* consumer-owned */

static bw_adv_t s_adv[BW_ADV_SLOTS];
static uint32_t s_adv_wr, s_adv_rd;   /* wr: producer-owned (BTC task), rd: consumer-owned */
static uint32_t s_adv_dropped;        /* producer-owned */
static uint32_t s_adv_dropped_seen;   /* consumer-owned */

/* The notification tier: what the peer pushes unsolicited. Its own ring keeps
 * a talkative characteristic from displacing a GATT completion or a scan
 * result, and a notification longer than one slot is dropped whole (counted,
 * never truncated). Only the BTC task writes it: the stack delivers a
 * notification only for handles this glue registered, and registration
 * happens only on values discovered from that task's search. */
#define BW_NOTIF_SLOTS 8

typedef struct {
    uint16_t conn_handle; /* the stack's conn_id (fits 16 bits: gattc_if<<8|id) */
    uint16_t attr_handle;
    uint8_t indication;
    uint8_t len;
    uint8_t data[BW_DATA_MAX];
} bw_notif_t;

static bw_notif_t s_notif[BW_NOTIF_SLOTS];
static uint32_t s_notif_wr, s_notif_rd; /* wr: producer-owned (BTC task), rd: consumer-owned */
static uint32_t s_notif_dropped;        /* producer-owned */
static uint32_t s_notif_dropped_seen;   /* consumer-owned */

/* The peripheral tier: what a central does to this device's attributes -
 * characteristic writes and client-configuration (CCCD) writes. Its own ring
 * keeps a write flood off the control tier (and off the notification ring,
 * so a central and a peripheral peer cannot displace each other's payloads).
 * Only the BTC task writes it: every GATTS event producer runs there. */
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
static uint32_t s_periph_wr, s_periph_rd; /* wr: producer-owned (BTC task), rd: consumer-owned */
static uint32_t s_periph_dropped;         /* producer-owned */
static uint32_t s_periph_dropped_seen;    /* consumer-owned */

/* GATT client state. One procedure runs at a time (the Go side serializes
 * them; an ATT bearer carries one outstanding request anyway), so the result
 * slots below are single. The BTC task fills the service table while the
 * search runs; the Go task reads it only after acquiring the EV_DISC_DONE
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

/* The full characteristic set of the last completed search, as the BTC task
 * staged it from the stack's cache (search completion): every service's
 * records, not one service's. The view table above holds the slice the
 * current Characteristics call asked for. The split keeps one writer per
 * table - the BTC task stages, the Go task slices - so the second (third,
 * ...) Characteristics call of a connection slices the same full set the
 * first one did, which is exactly how NimBLE's re-run discovery behaved.
 * The staging table publishes with the search's EV_DISC_DONE, the same
 * release contract the view uses. */
static llgo_ble_chr_t s_chrs_all[BW_CHR_MAX];
static int32_t s_chr_all_n;
static uint32_t s_chr_all_conn;

/* The characteristic-cache read buffer: esp_ble_gattc_get_all_char copies
 * records into a caller buffer synchronously; the only caller here is the
 * search-completion fill on the BTC task, and the buffer is static because
 * no record that travels may live on a task stack. BW_CHR_MAX entries cover
 * one fill; each per-service page reuses it. */
static esp_gattc_char_elem_t s_charbuf[BW_CHR_MAX];

/* The one GATT procedure in flight, with what its callbacks need to report it
 * (they run on the BTC task, and the context cannot live on a stack). */
typedef struct {
    uint8_t active;
    uint8_t full; /* a discovery table could not take every record */
    uint8_t kind; /* which procedure: the bw_gop_* values below */
    uint32_t op_id;
    uint32_t conn_id;
} bw_gop_t;

enum {
    GOP_DISC_SVCS = 6, /* the NimBLE glue's command numbers, kept for symmetry */
    GOP_DISC_CHRS = 7,
    GOP_READ = 8,
    GOP_WRITE = 9,
};

static bw_gop_t s_gop;

/* The completion of a read carries the value: copied out of the callback's
 * buffer in the BTC task and picked up by the poller, so a Go caller never
 * sees stack memory and the copy happens once. The callback fills both fields
 * before bw_gop_done release-publishes the EV_READ_DONE ctrl record; the
 * poller's acquire of that record's index is what makes them visible. */
static uint8_t s_read_data[BW_DATA_MAX];
static uint16_t s_read_len;

/* Written by the lifecycle calls (Go task) and the callbacks (BTC task);
 * every field is set before the mailbox record that exposes it is
 * release-published. */
static volatile int s_inited, s_synced;
static uint8_t s_own_type;   /* BLE_ADDR_TYPE_*: the address type all our own-side calls use */
static esp_gatt_if_t s_gattc_if; /* set by ESP_GATTC_REG_EVT, the EV_SYNC source */
/* The operation ids of the scan and the connect attempt in flight. Both
 * cross tasks - the Go task sets them at submission, the BTC task (and the
 * timeout timer) claim them at completion - so every claim (a read that also
 * clears) and every set runs inside the ctrl ring's critical section. A
 * stale completion that arrives after a new submission then claims nothing
 * the new submission owns, and the same id can never be claimed twice. */
static uint32_t s_disc_op;   /* op id of the running scan, 0 = none */
static uint32_t s_stop_op;   /* op id a StopScan owns until its completion event arrives, 0 = none */
static uint32_t s_conn_op;   /* op id of the running connect attempt, 0 = none */
static esp_bd_addr_t s_conn_bda; /* the peer s_conn_op is trying to reach,
                                  * in the stack's own address order */

/* The connect attempt's own parameters. esp_ble_gattc_enh_open deep-copies
 * the pointed-to connection parameters into the BTC message on submission,
 * but the scan-parameter structs are re-sent on every scan and the cancel
 * struct is filled right before its call - keeping all of them static makes
 * the ownership trivially safe without depending on where each API copies. */
static esp_ble_scan_params_t s_scan_params;
static esp_ble_conn_params_t s_conn_link;
static esp_ble_gattc_cancel_open_params_t s_cancel_open;

/* The connect attempt's timeout. NimBLE bounds ble_gap_connect natively;
 * Bluedroid's direct connect runs on stack-level timeouts, so the glue keeps
 * its own one-shot timer: when it fires with an attempt still in flight, the
 * attempt is cancelled and reported as failed. Created once at init, armed
 * per attempt, disarmed by the first connect completion. */
static esp_timer_handle_t s_conn_timer;

/* ---- mailbox writes ---- */

/* The ctrl ring's multi-writer gate: the FreeRTOS critical section disables
 * task scheduling (and, on this port, takes the spinlock and masks interrupts)
 * for the few instructions that read the write index, fill one slot and
 * publish it. Without it, a synchronous completion on the Go task and a
 * callback on the BTC task could read the same index and overwrite each
 * other's slot; with it, the ring's release/acquire discipline stays intact
 * for the consumer. */
static portMUX_TYPE s_ctrl_mux = portMUX_INITIALIZER_UNLOCKED;
#define BW_CTRL_LOCK()   taskENTER_CRITICAL(&s_ctrl_mux)
#define BW_CTRL_UNLOCK() taskEXIT_CRITICAL(&s_ctrl_mux)

/* The address-carrying form, for the peripheral connect event. addr may be
 * NULL; the plain form delegates here with a NULL addr. */
static void bw_ctrl_push_addr(int32_t kind, uint32_t op_id, uint32_t conn_id, int32_t status,
                              const uint8_t *addr, uint8_t addr_type);

static void bw_ctrl_push(int32_t kind, uint32_t op_id, uint32_t conn_id, int32_t status)
{
    bw_ctrl_push_addr(kind, op_id, conn_id, status, NULL, 0);
}

static void bw_ctrl_push_addr(int32_t kind, uint32_t op_id, uint32_t conn_id, int32_t status,
                              const uint8_t *addr, uint8_t addr_type)
{
    BW_CTRL_LOCK();
    uint32_t w = s_ctrl_wr;
    if ((uint32_t)(w - BW_ACQUIRE(s_ctrl_rd)) >= BW_CTRL_SLOTS) {
        __atomic_store_n(&s_ctrl_dropped, s_ctrl_dropped + 1, __ATOMIC_RELAXED);
        BW_CTRL_UNLOCK();
        return;
    }
    s_ctrl[w & (BW_CTRL_SLOTS - 1)] = (bw_ctrl_t){ kind, op_id, conn_id, status,
                                                   {0, 0, 0, 0, 0, 0}, 0 };
    if (addr != NULL) {
        memcpy(s_ctrl[w & (BW_CTRL_SLOTS - 1)].addr, addr, 6);
        s_ctrl[w & (BW_CTRL_SLOTS - 1)].addr_type = addr_type;
    }
    BW_PUBLISH(s_ctrl_wr, w + 1);                /* release: the slot is now visible */
    BW_CTRL_UNLOCK();
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

/* Copies one peer action into the peripheral ring. The value buffer belongs
 * to the BTC task and dies with this callback, so the copy has to happen
 * here; a payload larger than one slot is dropped whole (never truncated)
 * and counted in the mailbox's overflow report. */
static void bw_periph_push(int32_t kind, uint16_t conn_id, uint16_t attr_handle,
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
    slot->conn_handle = conn_id;
    slot->attr_handle = attr_handle;
    slot->status = status;
    slot->len = (uint8_t)value_len;
    if (value_len > 0 && value != NULL) {
        memcpy(slot->data, value, value_len);
    }
    BW_PUBLISH(s_periph_wr, w + 1);              /* release: the slot is now visible */
}

/* Copies one notification or indication into the notification ring. The value
 * buffer belongs to the BTC task and dies with this callback, so the copy has
 * to happen here; a payload larger than one slot is dropped whole (truncating
 * a notification would hand the reader a value the peer never sent) and
 * counted in the mailbox's overflow report - the same treatment NimBLE gave a
 * notification that did not fit its mbuf flattening. */
static void bw_notif_push(uint16_t conn_id, uint16_t attr_handle, uint8_t indication,
                          const uint8_t *value, uint16_t value_len)
{
    uint32_t w = s_notif_wr;                     /* producer-owned */
    if ((uint32_t)(w - BW_ACQUIRE(s_notif_rd)) >= BW_NOTIF_SLOTS) {
        __atomic_store_n(&s_notif_dropped, s_notif_dropped + 1, __ATOMIC_RELAXED);
        return;
    }
    if (value_len > BW_DATA_MAX) {
        __atomic_store_n(&s_notif_dropped, s_notif_dropped + 1, __ATOMIC_RELAXED);
        return;
    }
    bw_notif_t *slot = &s_notif[w & (BW_NOTIF_SLOTS - 1)];
    slot->conn_handle = conn_id;
    slot->attr_handle = attr_handle;
    slot->indication = indication ? 1 : 0;
    slot->len = (uint8_t)value_len;
    memcpy(slot->data, value, value_len);
    BW_PUBLISH(s_notif_wr, w + 1);               /* release: the slot is now visible */
}

/* The connect attempt's timeout callback (esp_timer task). It pushes through
 * the ctrl ring's critical-sectioned path like every other producer, and
 * claims the attempt's operation id under the same lock - so it can never
 * report the same attempt the CONNECT handler just reported, and a report
 * here is the only reason it cancels an open (canceling after a completed
 * connection would tear down a link the application is using). */
static void bw_conn_timer_cb(void *arg)
{
    (void)arg;
    uint32_t op = 0;
    BW_CTRL_LOCK();
    if (s_inited) {
        op = s_conn_op;
        s_conn_op = 0;
    }
    BW_CTRL_UNLOCK();
    if (op == 0) {
        return; /* connected (or given up) before the deadline: nothing to do */
    }
    /* Cancel the pending direct connect; ESP_GATTC_CANCEL_OPEN_EVT arrives
     * later and is ignored - the attempt's outcome is this record. The code
     * is the glue's own (Bluedroid reports a timeout nowhere app-visible). */
    s_cancel_open.gattc_if = s_gattc_if;
    memcpy(s_cancel_open.remote_bda, s_conn_bda, ESP_BD_ADDR_LEN);
    esp_ble_gattc_cancel_open(&s_cancel_open);
    bw_ctrl_push(EV_CONNECT, op, 0, BW_ETIMEOUT);
}

/* Stores a stack UUID in the record form: width plus little-endian bytes. A
 * 32-bit UUID is widened to 128 bits because the Go type has no 32-bit form;
 * its base-form value then matches the 16-bit alias when the upper half is
 * zero, which is what the Bluetooth base-UUID rule says it is. */
static const uint8_t bw_base_tail_le[12] = {
    0xFB, 0x34, 0x9B, 0x5F, 0x80, 0x00, 0x00, 0x80, 0x00, 0x10, 0x00, 0x00,
};

static void bw_uuid_store(uint8_t *dst, uint8_t *len_out, const esp_bt_uuid_t *u)
{
    memset(dst, 0, 16);
    switch (u->len) {
    case ESP_UUID_LEN_16:
        dst[0] = (uint8_t)(u->uuid.uuid16 & 0xFF);
        dst[1] = (uint8_t)(u->uuid.uuid16 >> 8);
        *len_out = 2;
        break;
    case ESP_UUID_LEN_32:
        memcpy(dst, bw_base_tail_le, sizeof bw_base_tail_le);
        dst[12] = (uint8_t)(u->uuid.uuid32 & 0xFF);
        dst[13] = (uint8_t)((u->uuid.uuid32 >> 8) & 0xFF);
        dst[14] = (uint8_t)((u->uuid.uuid32 >> 16) & 0xFF);
        dst[15] = (uint8_t)((u->uuid.uuid32 >> 24) & 0xFF);
        *len_out = 16;
        break;
    case ESP_UUID_LEN_128:
        memcpy(dst, u->uuid.uuid128, 16);
        *len_out = 16;
        break;
    default:
        *len_out = 0;
        break;
    }
}

/* ---- connection ids: raw conn_id + generation ----
 *
 * Bluedroid's conn_id (gattc_if << 8 | client id) is reused across
 * connect/disconnect cycles just like an HCI handle, so a raw conn_id alone
 * cannot identify a connection: a stale ConnID would address the new
 * connection. Each registration therefore takes the next generation number,
 * and the ConnID the application holds is (generation << 16) | conn_id - the
 * same packed form the NimBLE glue produced from its handles. A ConnID from
 * an older generation is refused instead of being applied to a different
 * peer. The peer's address rides along because notification registration
 * (esp_ble_gattc_register_for_notify) is keyed by address, not conn_id. */

#define BW_CONNS 4

typedef struct {
    uint8_t used;
    uint32_t gen;
    uint16_t conn_id;
    esp_bd_addr_t bda; /* the peer address in the stack's own order: what
                        * esp_ble_gattc_register_for_notify keys on */
    uint16_t mtu; /* the negotiated ATT MTU, BW mtu default until CFG_MTU */
} bw_conn_t;

#define BW_ATT_DEFAULT_MTU 23

static bw_conn_t s_conns[BW_CONNS];
static uint32_t s_conn_gen;

/* Adds a connection and returns its packed id (0 = table full: the caller
 * reports the failed connect instead of a ConnID 0, which the Go side would
 * read as success). Runs on the BTC task. */
static uint32_t bw_conn_add(uint16_t conn_id, const esp_bd_addr_t bda)
{
    for (int i = 0; i < BW_CONNS; i++) {
        if (!s_conns[i].used) {
            s_conns[i].used = 1;
            s_conns[i].conn_id = conn_id;
            memcpy(s_conns[i].bda, bda, ESP_BD_ADDR_LEN);
            s_conns[i].mtu = BW_ATT_DEFAULT_MTU;
            s_conns[i].gen = ++s_conn_gen;
            return (s_conns[i].gen << 16) | conn_id;
        }
    }
    return 0;
}

/* Removes a connection and returns the packed id it was registered under
 * (its raw conn_id when unknown: report it with generation 0). Runs on the
 * BTC task. */
static uint32_t bw_conn_remove(uint16_t conn_id)
{
    for (int i = 0; i < BW_CONNS; i++) {
        if (s_conns[i].used && s_conns[i].conn_id == conn_id) {
            uint32_t id = (s_conns[i].gen << 16) | conn_id;
            s_conns[i].used = 0;
            return id;
        }
    }
    return conn_id;
}

/* Resolves a packed ConnID to the raw conn_id of a live connection, or -1.
 * Runs on whichever task holds a live ConnID (the Go task's control calls). */
static int bw_conn_find(uint32_t conn_id)
{
    uint16_t raw = (uint16_t)(conn_id & 0xFFFF);
    uint32_t gen = conn_id >> 16;
    for (int i = 0; i < BW_CONNS; i++) {
        if (s_conns[i].used && s_conns[i].conn_id == raw && s_conns[i].gen == gen) {
            return (int)raw;
        }
    }
    return -1;
}

/* The ConnID a live raw conn_id is registered under, for the records that
 * carry a bare conn_id: notifications and MTU reports. The table belongs to
 * the BTC task while the poller runs on the Go task; an entry is complete
 * before the ConnID that names it is published (the connect record that
 * carries it goes through the same ring), so a reader holding a live ConnID
 * sees a consistent entry, and a stale one can only produce a status error. */
static uint32_t bw_conn_id_of(uint16_t conn_id)
{
    for (int i = 0; i < BW_CONNS; i++) {
        if (s_conns[i].used && s_conns[i].conn_id == conn_id) {
            return (s_conns[i].gen << 16) | conn_id;
        }
    }
    return conn_id;
}

/* Copies a live connection's peer address into out (0 = not live). The
 * synchronous characteristic fill needs it for notification registration. */
static int bw_conn_bda_of(uint32_t conn_id, esp_bd_addr_t out)
{
    uint16_t raw = (uint16_t)(conn_id & 0xFFFF);
    uint32_t gen = conn_id >> 16;
    for (int i = 0; i < BW_CONNS; i++) {
        if (s_conns[i].used && s_conns[i].conn_id == raw && s_conns[i].gen == gen) {
            memcpy(out, s_conns[i].bda, ESP_BD_ADDR_LEN);
            return 0;
        }
    }
    return -1;
}

/* Reverses a 6-byte address between the two orders that meet at this
 * boundary: Bluedroid's esp_bd_addr_t keeps the most significant octet first
 * (esp_bt_dev_get_address and every callback parameter), while the package's
 * Peer.Addr - like NimBLE's ble_addr_t.val - keeps the least significant one
 * ("the stack's little-endian order", which AddrString renders last). The
 * glue owns the translation so the Go contract holds on both backends. */
static void bw_addr_swap(uint8_t *dst, const uint8_t *src)
{
    for (int i = 0; i < 6; i++) {
        dst[i] = src[5 - i];
    }
}

/* ---- peripheral role: advertising, the local GATT server ----
 *
 * The same boundary rules as the central half: the BTC task builds the
 * attribute table (the esp_ble_gatts_* creation calls run through it) and
 * answers the peer's reads straight from a static value cache, so a read is
 * never deferred into Go - the GATT server cannot call into the llgo
 * scheduler, and the read must be answered synchronously on the stack's own
 * task. WriteCharValue fills the same cache from the Go task; the copy runs
 * inside the ctrl ring's critical section, which is what keeps a cache write
 * and a read the BTC task is serving serialized.
 *
 * Advertising is the legacy procedure set (raw advertising and scan-response
 * payloads + start/stop), the same CONFIG_BT_BLE_42_FEATURES_SUPPORTED split
 * the scan path already makes; a BLE 5.0-only build reports NOT_SUPPORTED
 * for advertising rather than half-working.
 */

/* Advertising state. The payloads live in static buffers (the config calls
 * deep-copy, but the chain of set-rsp-then-start events needs the bytes
 * around after the submission returned); s_adv_op owns the one start/stop
 * operation in flight, the same claim pattern the scan and connect ids use.
 * s_adv_running tracks whether the controller is actually advertising (a
 * connectable advertisement auto-stops on connection, with no stop event). */
static uint8_t s_adv_data[BW_ADV_MAX];
static uint8_t s_adv_data_len;
static uint8_t s_scan_rsp[BW_ADV_MAX];
static uint8_t s_scan_rsp_len;
static uint8_t s_adv_conn;    /* connectable advertisement */
static uint8_t s_adv_running;
static uint16_t s_adv_itvl;   /* the interval in 0.625 ms units */
static esp_ble_adv_params_t s_adv_params;
/* The one start/stop operation in flight, claimed like the scan and connect
 * ids: the completions arrive on the BTC task while the submission ran on
 * the Go task, and a set must be atomic against their claims. */
static uint32_t s_adv_op;

/* The local GATT server: the declaration the Go side serialized once
 * (llgo_ble_gatts_declare copies it here), and what the stack assigned while
 * building it. One value cache per characteristic is the read answer: a peer
 * read is served from it synchronously in the GATTS callback, a peer write
 * overwrites it (and reports EV_CHAR_WRITE), and WriteCharValue fills it from
 * Go. MaxLocalServices/MaxLocalChars in ble.go are these two bounds. */
#define BW_GSVC_MAX 4
#define BW_GCHR_MAX 8
#define BW_CCCD_UUID16 0x2902

typedef struct {
    uint8_t uuid_len;
    uint8_t uuid[16];
    uint16_t n_chars;     /* characteristics this service declared */
    uint16_t first_chr;   /* global index of its first characteristic */
    uint16_t num_handles; /* the attribute budget the Go side declared */
    uint16_t svc_handle;  /* the stack's service handle (0 until created) */
} bw_gsvc_t;

typedef struct {
    uint8_t uuid_len;
    uint8_t uuid[16];
    uint8_t props;
    uint8_t subs;         /* the CCCD's current value (bit0 notify, bit1
                           * indicate): what a CCCD read answers with */
    uint16_t handle;      /* the value attribute handle, 0 until assigned */
    uint16_t cccd_handle; /* 0 = this characteristic has no CCCD */
    uint16_t value_len;
    uint8_t value[BW_DATA_MAX];
} bw_gchr_t;  /* 154 bytes: 8 characteristics cost ~1.2 KiB of .bss */

static bw_gsvc_t s_gsvcs[BW_GSVC_MAX];
static int32_t s_gsvc_n;
static bw_gchr_t s_gchrs[BW_GCHR_MAX];
static int32_t s_gchr_n;
static uint8_t s_gatts_building; /* a service registration is in flight */
static uint32_t s_gatts_op;      /* the commit's operation id */

/* The creation state machine's cursor. It runs on the BTC task: CREATE ->
 * (ADD_CHAR <-> ADD_DESCR)* -> START per service, then the next service, then
 * the one EV_SERVICES_DONE record. The static uuid/control staging keeps the
 * esp_ble_gatts_add_* arguments out of the BTC task stack. */
static int32_t s_gs_cur_svc; /* the service being built */
static int32_t s_gs_cur_chr; /* the characteristic being added (-1 = none) */
static esp_gatt_srvc_id_t s_gs_srvc_id;
static esp_bt_uuid_t s_gs_uuid;
static esp_bt_uuid_t s_gs_cccd_uuid;
static esp_attr_control_t s_gs_ctrl; /* ESP_GATT_RSP_BY_APP: the glue answers */

/* The stack's interface for the GATT server role, set by ESP_GATTS_REG_EVT.
 * EV_SYNC now waits for both registrations (the central's and the server's):
 * CommitServices immediately after Init would otherwise race the server
 * interface's own asynchronous registration. A build that cannot register
 * the server (GATTS compiled out) still syncs - the central half is then
 * fully usable and CommitServices reports the refusal honestly. */
static esp_gatt_if_t s_gatts_if;
static uint8_t s_gattc_reg, s_gatts_reg, s_gatts_fail;

static void bw_sync_if_ready(void)
{
    if (!s_synced && s_gattc_reg && (s_gatts_reg || s_gatts_fail)) {
        s_synced = 1;
        bw_ctrl_push(EV_SYNC, 0, 0, 0);
    }
}

/* ---- peripheral connections: raw conn_id + generation ----
 *
 * The same packing the central side uses ((generation << 16) | raw conn_id,
 * the generation counter shared with it, so an id is unique across both
 * tables), with the GATTS interface's own connection ids. */

#define BW_PCONN_MAX 2

typedef struct {
    uint8_t used;
    uint32_t gen;
    uint16_t conn_id;
} bw_pconn_t;

static bw_pconn_t s_pconns[BW_PCONN_MAX];

static uint32_t bw_pconn_add(uint16_t conn_id)
{
    for (int i = 0; i < BW_PCONN_MAX; i++) {
        if (!s_pconns[i].used) {
            s_pconns[i].used = 1;
            s_pconns[i].conn_id = conn_id;
            s_pconns[i].gen = ++s_conn_gen;
            return (s_pconns[i].gen << 16) | conn_id;
        }
    }
    return 0;
}

static uint32_t bw_pconn_remove(uint16_t conn_id)
{
    for (int i = 0; i < BW_PCONN_MAX; i++) {
        if (s_pconns[i].used && s_pconns[i].conn_id == conn_id) {
            uint32_t id = (s_pconns[i].gen << 16) | conn_id;
            s_pconns[i].used = 0;
            return id;
        }
    }
    return conn_id;
}

static int bw_pconn_find(uint32_t conn_id)
{
    uint16_t raw = (uint16_t)(conn_id & 0xFFFF);
    uint32_t gen = conn_id >> 16;
    for (int i = 0; i < BW_PCONN_MAX; i++) {
        if (s_pconns[i].used && s_pconns[i].conn_id == raw && s_pconns[i].gen == gen) {
            return (int)raw;
        }
    }
    return -1;
}

/* The packed id a live raw conn_id is registered under, for the peripheral
 * payload records (writes, subscriptions) that carry a bare conn_id - same
 * shape and same visibility contract as bw_conn_id_of. */
static uint32_t bw_pconn_id_of(uint16_t conn_id)
{
    for (int i = 0; i < BW_PCONN_MAX; i++) {
        if (s_pconns[i].used && s_pconns[i].conn_id == conn_id) {
            return (s_pconns[i].gen << 16) | conn_id;
        }
    }
    return conn_id;
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

/* The characteristic that owns an attribute handle: the value handle first,
 * then the CCCD. Returns the table index, or -1. Runs on the BTC task (the
 * GATTS callbacks) and on the Go task (WriteCharValue/Notify); the table is
 * frozen between commits, so the answer is stable while a connection lives. */
static int bw_gchr_by_handle(uint16_t handle)
{
    for (int i = 0; i < s_gchr_n; i++) {
        if (s_gchrs[i].handle == handle) {
            return i;
        }
    }
    return -1;
}

static int bw_gchr_by_cccd(uint16_t handle)
{
    for (int i = 0; i < s_gchr_n; i++) {
        if (s_gchrs[i].cccd_handle == handle) {
            return i;
        }
    }
    return -1;
}

/* ---- the service-creation state machine (BTC task) ---- */

/* Ends a failed commit: the Go-side waiter is owed its completion record. */
static void bw_gatts_fail(int32_t status)
{
    uint32_t op = s_gatts_op;
    s_gatts_op = 0;
    s_gatts_building = 0;
    if (op != 0) {
        bw_ctrl_push(EV_SERVICES_DONE, op, 0, status);
    }
}

/* Submits the next esp_ble_gatts_add_char of the current service, or - when
 * its characteristics are exhausted - starts the service. */
static void bw_gatts_add_next_char(void)
{
    bw_gsvc_t *svc = &s_gsvcs[s_gs_cur_svc];
    if (s_gs_cur_chr < 0 || s_gs_cur_chr >= svc->first_chr + svc->n_chars) {
        /* The service's table is complete: make it discoverable. */
        esp_err_t e = esp_ble_gatts_start_service(svc->svc_handle);
        if (e != ESP_OK) {
            bw_gatts_fail(e);
        }
        return;
    }
    bw_gchr_t *chr = &s_gchrs[s_gs_cur_chr];
    if (chr->uuid_len == 2) {
        s_gs_uuid.len = ESP_UUID_LEN_16;
        s_gs_uuid.uuid.uuid16 = (uint16_t)(chr->uuid[0] | (chr->uuid[1] << 8));
    } else {
        s_gs_uuid.len = ESP_UUID_LEN_128;
        memcpy(s_gs_uuid.uuid.uuid128, chr->uuid, 16);
    }
    esp_gatt_perm_t perm = 0;
    if (chr->props & ESP_GATT_CHAR_PROP_BIT_READ) {
        perm |= ESP_GATT_PERM_READ;
    }
    if (chr->props & (ESP_GATT_CHAR_PROP_BIT_WRITE | ESP_GATT_CHAR_PROP_BIT_WRITE_NR)) {
        perm |= ESP_GATT_PERM_WRITE;
    }
    esp_err_t e = esp_ble_gatts_add_char(svc->svc_handle, &s_gs_uuid, perm,
                                         (esp_gatt_char_prop_t)chr->props,
                                         NULL /* value: the glue's cache answers reads */,
                                         &s_gs_ctrl /* ESP_GATT_RSP_BY_APP: the glue answers */);
    if (e != ESP_OK) {
        bw_gatts_fail(e);
    }
}

/* Submits esp_ble_gatts_create_service for the current service. */
static void bw_gatts_create_next_svc(void)
{
    if (s_gs_cur_svc >= s_gsvc_n) {
        /* Every service started: the commit is done. Handles stay assigned in
         * s_gchrs, where llgo_ble_gatts_attr reads them. */
        uint32_t op = s_gatts_op;
        s_gatts_op = 0;
        s_gatts_building = 0;
        if (op != 0) {
            bw_ctrl_push(EV_SERVICES_DONE, op, 0, 0);
        }
        return;
    }
    bw_gsvc_t *svc = &s_gsvcs[s_gs_cur_svc];
    memset(&s_gs_srvc_id, 0, sizeof s_gs_srvc_id);
    s_gs_srvc_id.is_primary = true;
    s_gs_srvc_id.id.inst_id = (uint8_t)s_gs_cur_svc;
    if (svc->uuid_len == 2) {
        s_gs_srvc_id.id.uuid.len = ESP_UUID_LEN_16;
        s_gs_srvc_id.id.uuid.uuid.uuid16 = (uint16_t)(svc->uuid[0] | (svc->uuid[1] << 8));
    } else {
        s_gs_srvc_id.id.uuid.len = ESP_UUID_LEN_128;
        memcpy(s_gs_srvc_id.id.uuid.uuid.uuid128, svc->uuid, 16);
    }
    esp_err_t e = esp_ble_gatts_create_service(s_gatts_if, &s_gs_srvc_id, svc->num_handles);
    if (e != ESP_OK) {
        bw_gatts_fail(e);
    }
}

/* ---- GATTS callback (BTC task) ---- */

/* Answers one read request from the value cache, synchronously - this is the
 * whole read model: no Go involvement, no deferred response, so a peer read
 * costs one memcpy inside the callback. Long reads (an offset past the MTU)
 * slice the cache; an offset past its end is the ATT invalid-offset error. */
static void bw_gatts_respond_read(esp_gatt_if_t gatts_if, uint16_t conn_id,
                                  uint32_t trans_id, uint16_t handle, uint16_t offset)
{
    int idx = bw_gchr_by_handle(handle);
    static esp_gatt_rsp_t s_read_rsp; /* 517-byte value field: never on a stack */
    if (idx < 0) {
        idx = bw_gchr_by_cccd(handle);
        if (idx < 0) {
            esp_ble_gatts_send_response(gatts_if, conn_id, trans_id,
                                        ESP_GATT_READ_NOT_PERMIT, NULL);
            return;
        }
        /* A readable CCCD: answer the current subscription state. */
        memset(&s_read_rsp, 0, sizeof s_read_rsp);
        s_read_rsp.attr_value.handle = handle;
        s_read_rsp.attr_value.len = 2;
        s_read_rsp.attr_value.value[0] = s_gchrs[idx].subs;
        esp_ble_gatts_send_response(gatts_if, conn_id, trans_id, ESP_GATT_OK,
                                    &s_read_rsp);
        return;
    }
    BW_CTRL_LOCK();
    uint16_t len = s_gchrs[idx].value_len;
    BW_CTRL_UNLOCK();
    if (offset > len) {
        esp_ble_gatts_send_response(gatts_if, conn_id, trans_id,
                                    ESP_GATT_INVALID_OFFSET, NULL);
        return;
    }
    memset(&s_read_rsp, 0, sizeof s_read_rsp);
    s_read_rsp.attr_value.handle = handle;
    BW_CTRL_LOCK();
    s_read_rsp.attr_value.len = len - offset;
    memcpy(s_read_rsp.attr_value.value, s_gchrs[idx].value + offset, len - offset);
    BW_CTRL_UNLOCK();
    esp_ble_gatts_send_response(gatts_if, conn_id, trans_id, ESP_GATT_OK, &s_read_rsp);
}

static void bw_gatts_cb(esp_gatts_cb_event_t event, esp_gatt_if_t gatts_if,
                        esp_ble_gatts_cb_param_t *param)
{
    switch (event) {
    case ESP_GATTS_REG_EVT:
        /* The server interface is ready (or the build has no GATT server).
         * Both registrations must settle before EV_SYNC - see the state
         * block's comment. */
        if (param->reg.status == ESP_GATT_OK) {
            s_gatts_if = gatts_if;
            s_gatts_reg = 1;
        } else {
            s_gatts_fail = 1;
        }
        bw_sync_if_ready();
        break;
    case ESP_GATTS_CREATE_EVT:
        if (!s_gatts_building || s_gs_cur_svc >= s_gsvc_n) {
            break; /* no commit in flight: nothing to attribute it to */
        }
        if (param->create.status != ESP_GATT_OK) {
            bw_gatts_fail(param->create.status);
            break;
        }
        s_gsvcs[s_gs_cur_svc].svc_handle = param->create.service_handle;
        s_gs_cur_chr = s_gsvcs[s_gs_cur_svc].n_chars > 0 ? s_gsvcs[s_gs_cur_svc].first_chr : -1;
        bw_gatts_add_next_char();
        break;
    case ESP_GATTS_ADD_CHAR_EVT:
        if (!s_gatts_building || s_gs_cur_chr < 0) {
            break;
        }
        if (param->add_char.status != ESP_GATT_OK) {
            bw_gatts_fail(param->add_char.status);
            break;
        }
        s_gchrs[s_gs_cur_chr].handle = param->add_char.attr_handle;
        if (s_gchrs[s_gs_cur_chr].props & (ESP_GATT_CHAR_PROP_BIT_NOTIFY | ESP_GATT_CHAR_PROP_BIT_INDICATE)) {
            /* The subscription gate: the client configuration descriptor at
             * its own handle. A CCCD write arrives as an ordinary WRITE_EVT. */
            s_gs_cccd_uuid.len = ESP_UUID_LEN_16;
            s_gs_cccd_uuid.uuid.uuid16 = BW_CCCD_UUID16;
            esp_err_t e = esp_ble_gatts_add_char_descr(
                s_gsvcs[s_gs_cur_svc].svc_handle, &s_gs_cccd_uuid,
                (esp_gatt_perm_t)(ESP_GATT_PERM_READ | ESP_GATT_PERM_WRITE),
                NULL, &s_gs_ctrl);
            if (e != ESP_OK) {
                bw_gatts_fail(e);
            }
            break;
        }
        s_gs_cur_chr++;
        bw_gatts_add_next_char();
        break;
    case ESP_GATTS_ADD_CHAR_DESCR_EVT:
        if (!s_gatts_building || s_gs_cur_chr < 0) {
            break;
        }
        if (param->add_char_descr.status != ESP_GATT_OK) {
            bw_gatts_fail(param->add_char_descr.status);
            break;
        }
        s_gchrs[s_gs_cur_chr].cccd_handle = param->add_char_descr.attr_handle;
        s_gs_cur_chr++;
        bw_gatts_add_next_char();
        break;
    case ESP_GATTS_START_EVT:
        if (!s_gatts_building || s_gs_cur_svc >= s_gsvc_n) {
            break;
        }
        if (param->start.status != ESP_GATT_OK) {
            bw_gatts_fail(param->start.status);
            break;
        }
        s_gs_cur_svc++;
        bw_gatts_create_next_svc();
        break;
    case ESP_GATTS_CONNECT_EVT: {
        /* A central connected to this device's advertisement. Advertising is
         * over by definition (the connect is the end - StopAdvertising after
         * this reports its refusal honestly through EV_ADV_DONE). */
        s_adv_running = 0;
        uint32_t id = bw_pconn_add(param->connect.conn_id);
        if (id == 0) {
            /* Connection table full: drop the link rather than serve a
             * connection the application cannot address. */
            esp_ble_gatts_close(gatts_if, param->connect.conn_id);
            break;
        }
        uint8_t addr[6];
        bw_addr_swap(addr, param->connect.remote_bda);
        bw_ctrl_push_addr(EV_PERIPH_CONNECT, 0, id, 0, addr,
                          (uint8_t)param->connect.ble_addr_type);
        break;
    }
    case ESP_GATTS_DISCONNECT_EVT: {
        uint32_t id = bw_pconn_remove(param->disconnect.conn_id);
        for (int i = 0; i < s_gchr_n; i++) {
            s_gchrs[i].subs = 0;
        }
        bw_ctrl_push(EV_PERIPH_DISCONNECT, 0, id, (int32_t)param->disconnect.reason);
        break;
    }
    case ESP_GATTS_READ_EVT:
        if (param->read.need_rsp) {
            bw_gatts_respond_read(gatts_if, param->read.conn_id, param->read.trans_id,
                                  param->read.handle, param->read.offset);
        }
        break;
    case ESP_GATTS_WRITE_EVT:
        if (param->write.is_prep) {
            /* Prepared writes are not part of this package's surface; each
             * prepared write is refused with the ATT error the procedure
             * defines, so a reliable-write client sees the refusal instead of
             * an exec that silently drops data. */
            if (param->write.need_rsp) {
                esp_ble_gatts_send_response(gatts_if, param->write.conn_id,
                                            param->write.trans_id,
                                            ESP_GATT_REQ_NOT_SUPPORTED, NULL);
            }
            break;
        }
        {
            int idx = bw_gchr_by_handle(param->write.handle);
            if (idx >= 0) {
                /* A value write: cache it (what the next read answers), then
                 * report. A write longer than one cache is refused with the
                 * attribute-length error - truncation would invent data. */
                if (param->write.len > BW_DATA_MAX) {
                    if (param->write.need_rsp) {
                        esp_ble_gatts_send_response(gatts_if, param->write.conn_id,
                                                    param->write.trans_id,
                                                    ESP_GATT_INVALID_ATTR_LEN, NULL);
                    }
                    break;
                }
                BW_CTRL_LOCK();
                memcpy(s_gchrs[idx].value, param->write.value, param->write.len);
                s_gchrs[idx].value_len = param->write.len;
                BW_CTRL_UNLOCK();
                bw_periph_push(EV_CHAR_WRITE, param->write.conn_id, param->write.handle,
                               0, param->write.value, param->write.len);
            } else if ((idx = bw_gchr_by_cccd(param->write.handle)) >= 0) {
                /* The subscription gate: report it with the characteristic's
                 * value handle and the raw CCCD value as the status. */
                uint16_t cccd = 0;
                if (param->write.len >= 2) {
                    cccd = (uint16_t)(param->write.value[0] | (param->write.value[1] << 8));
                }
                s_gchrs[idx].subs = (uint8_t)cccd;
                bw_periph_push(EV_SUBSCRIBED, param->write.conn_id,
                               s_gchrs[idx].handle, (int32_t)cccd, NULL, 0);
            } else if (param->write.need_rsp) {
                esp_ble_gatts_send_response(gatts_if, param->write.conn_id,
                                            param->write.trans_id,
                                            ESP_GATT_WRITE_NOT_PERMIT, NULL);
                break;
            }
            if (param->write.need_rsp) {
                esp_ble_gatts_send_response(gatts_if, param->write.conn_id,
                                            param->write.trans_id, ESP_GATT_OK, NULL);
            }
        }
        break;
    case ESP_GATTS_EXEC_WRITE_EVT:
        /* Nothing was ever stored from a prepared write (each one is refused
         * above), so an execute is answered as a no-op - the honest reply to
         * a procedure this server does not run. */
        esp_ble_gatts_send_response(gatts_if, param->exec_write.conn_id,
                                    param->exec_write.trans_id, ESP_GATT_OK, NULL);
        break;
    default:
        /* Congestion, confirmation receipts, MTU (the exchange the central
         * starts is answered inside the stack) - nothing here carries a
         * record the package's surface needs; ignoring keeps the mailbox
         * bounded. */
        break;
    }
}

/* Starts the advertisement with the payloads already staged. Runs on the BTC
 * task, from the raw-data-set completions. */
static void bw_adv_start_now(void)
{
    memset(&s_adv_params, 0, sizeof s_adv_params);
    s_adv_params.adv_int_min = s_adv_itvl;
    s_adv_params.adv_int_max = s_adv_itvl;
    s_adv_params.adv_type = s_adv_conn ? ADV_TYPE_IND : ADV_TYPE_NONCONN_IND;
    s_adv_params.own_addr_type = s_own_type;
    s_adv_params.channel_map = ADV_CHNL_ALL;
    s_adv_params.adv_filter_policy = ADV_FILTER_ALLOW_SCAN_ANY_CON_ANY;
    esp_err_t e = esp_ble_gap_start_advertising(&s_adv_params);
    if (e != ESP_OK) {
        uint32_t op = s_adv_op;
        s_adv_op = 0;
        if (op != 0) {
            bw_ctrl_push(EV_ADV_DONE, op, 0, e);
        }
    }
}

/* ---- GATT procedure context ----
 *
 * The context is claimed from both sides: the Go task starts a procedure and
 * reports the synchronous completions, the BTC task reports the asynchronous
 * ones, and the teardown path (a disconnect while a procedure runs) ends it.
 * Every claim (a read that also decides) and every clear therefore runs under
 * the ctrl ring's critical section: without it, a completion and a
 * concurrently starting procedure could attribute the same record twice, and
 * a teardown could clear the context a completion is about to claim. */

/* Claims the procedure slot for a new procedure: atomic busy-check + begin
 * under the lock (a busy check that reads and a begin that writes as two
 * separate sections would leave a window for a teardown or a completion to
 * squeeze into). Returns 0 when another procedure is still in flight. */
static int bw_gop_claim(uint8_t kind, uint32_t op_id, uint32_t conn_id)
{
    BW_CTRL_LOCK();
    int claimed = !s_gop.active;
    if (claimed) {
        s_gop.active = 1;
        s_gop.full = 0;
        s_gop.kind = kind;
        s_gop.op_id = op_id;
        s_gop.conn_id = conn_id;
    }
    BW_CTRL_UNLOCK();
    return claimed;
}

/* Whether a completion belongs to the procedure in flight: its kind and its
 * raw connection must both match (Bluedroid completions carry a raw conn_id;
 * the procedure was registered with a packed one). Checked under the lock, so
 * a procedure starting on the Go task is never attributed a stale record. */
static int bw_gop_match(uint8_t kind, uint16_t raw_conn)
{
    BW_CTRL_LOCK();
    int match = 0;
    if (s_gop.active && s_gop.kind == kind) {
        int live = bw_conn_find(s_gop.conn_id);
        match = live >= 0 && (uint16_t)live == raw_conn;
    }
    BW_CTRL_UNLOCK();
    return match;
}

/* Ends the procedure in flight and pushes the one ctrl record that reports it.
 * The claim (read + clear) runs under the lock; the record is pushed after it,
 * because the push itself takes the same lock (spinlocks do not nest). The
 * release publish inside bw_ctrl_push is also what makes a discovery's table
 * (s_svcs/s_chrs and their counts) visible to the Go task. */
static void bw_gop_done(int32_t kind, int32_t status)
{
    uint32_t op_id = 0;
    uint32_t conn_id = 0;
    BW_CTRL_LOCK();
    if (s_gop.active) {
        s_gop.active = 0;
        op_id = s_gop.op_id;
        conn_id = s_gop.conn_id;
    }
    BW_CTRL_UNLOCK();
    if (op_id != 0) {
        bw_ctrl_push(kind, op_id, conn_id, status);
    }
}

/* A submission that failed before the procedure started: the completion
 * record is still owed to the caller. */
static void bw_gop_submit_failed(int32_t status)
{
    uint32_t op_id = 0;
    uint32_t conn_id = 0;
    BW_CTRL_LOCK();
    if (s_gop.active) {
        s_gop.active = 0;
        op_id = s_gop.op_id;
        conn_id = s_gop.conn_id;
    }
    BW_CTRL_UNLOCK();
    if (op_id != 0) {
        bw_ctrl_push(EV_ERROR, op_id, conn_id, status);
    }
}

/* Retires a connection: drops the discovery tables tagged with it, ends a
 * GATT procedure in flight on it - with the error record its Go-side waiter
 * is still owed, because the stack aborts the procedure silently with the
 * link and no completion event would ever arrive (without this, every later
 * GATT call would be refused EALREADY until Deinit) - and pushes the one
 * EV_DISCONNECT record. Runs on the BTC task; the id is the packed ConnID
 * bw_conn_remove returned. */
static void bw_conn_retire(uint32_t id, int32_t reason)
{
    if (s_svc_conn == id) {
        s_svc_n = 0;
        s_svc_conn = 0;
    }
    if (s_chr_conn == id) {
        s_chr_n = 0;
        s_chr_conn = 0;
    }
    if (s_chr_all_conn == id) {
        s_chr_all_n = 0;
        s_chr_all_conn = 0;
    }
    uint32_t gop_op = 0;
    uint32_t gop_conn = 0;
    BW_CTRL_LOCK();
    if (s_gop.active && s_gop.conn_id == id) {
        s_gop.active = 0;
        gop_op = s_gop.op_id;
        gop_conn = s_gop.conn_id;
    }
    BW_CTRL_UNLOCK();
    if (gop_op != 0) {
        /* BW_ENOTCONN: the NimBLE-space code for a procedure whose
         * connection went away - the same condition NimBLE reported. */
        bw_ctrl_push(EV_ERROR, gop_op, gop_conn, BW_ENOTCONN);
    }
    bw_ctrl_push(EV_DISCONNECT, 0, id, reason);
}

/* ---- discovery: the folded model ----
 *
 * NimBLE discovered services and characteristics as two procedures; Bluedroid
 * discovers both in one search: the characteristics land in the stack's cache
 * while the search runs, and the cache is final when the search completes.
 * This glue keeps the Go-facing shape of the two procedures, with one BTC-task
 * fill in between: the search completion callback (which runs on the BTC
 * task) reads the cache into the static characteristic table, and
 * llgo_ble_disc_chrs - the Go task's call - only slices that static table
 * down to the requested service's handle range. The split is not a style
 * choice: the cache is freed by the BTC task on a disconnect (and rebuilt by
 * it on a re-search), so reading it from the Go task would race the free and
 * touch freed memory; on the BTC task the read is serialized with every
 * later cache mutation by construction. */

/* Stages the characteristics of every discovered service into s_chrs_all,
 * straight from the stack's cache. Runs on the BTC task from the search
 * completion,
 * where the cache is guaranteed final (the completion is what the stack sends
 * after it commits the cache) and where every later cache mutation runs, so
 * the read is same-task-serial with the frees. Returns the procedure status:
 * 0, BW_EMSGSIZE when the table filled (the overflow contract the NimBLE
 * discovery had - the discovery ends with EMSGSIZE instead of losing records
 * silently; a peer whose characteristics exceed BW_CHR_MAX across all
 * services reports it here, where NimBLE reported it per range discovery),
 * or the cache accessor's own status for a real failure.
 * ESP_GATT_NOT_FOUND and ESP_GATT_INVALID_OFFSET are the accessor's "range
 * exhausted" answers, not failures - a service with no characteristics ends
 * the same way NimBLE's discovery of an empty range did: success, zero
 * records. */
static int32_t bw_fill_chrs(uint16_t raw_conn)
{
    int32_t status = 0;
    for (int32_t i = 0; i < s_svc_n && status == 0 && !s_gop.full; i++) {
        uint16_t offset = 0;
        for (;;) {
            if (s_chr_all_n >= BW_CHR_MAX) {
                s_gop.full = 1;
                break;
            }
            uint16_t want = (uint16_t)(BW_CHR_MAX - s_chr_all_n);
            uint16_t count = want;
            esp_gatt_status_t st = esp_ble_gattc_get_all_char(s_gattc_if, raw_conn,
                                                              s_svcs[i].start_handle,
                                                              s_svcs[i].end_handle,
                                                              s_charbuf, &count, offset);
            if (st == ESP_GATT_NOT_FOUND || st == ESP_GATT_INVALID_OFFSET) {
                break; /* this service holds nothing more */
            }
            if (st != ESP_GATT_OK) {
                status = st; /* a real cache failure */
                break;
            }
            for (uint16_t k = 0; k < count; k++) {
                llgo_ble_chr_t *rec = &s_chrs_all[s_chr_all_n++];
                bw_uuid_store(rec->uuid, &rec->uuid_len, &s_charbuf[k].uuid);
                rec->properties = s_charbuf[k].properties;
                rec->value_handle = s_charbuf[k].char_handle;
                /* The declaration handle is not in the cache accessor's
                 * element; the package's CCCD math goes through the value
                 * handle anyway (the layout the GATT specification mandates),
                 * so the record carries 0 where NimBLE could report a real
                 * declaration. */
                rec->def_handle = 0;
            }
            if (s_chr_all_n >= BW_CHR_MAX) {
                s_gop.full = 1;
                break;
            }
            if (count < want) {
                break; /* the service's cache records are exhausted */
            }
            offset += count; /* the buffer filled exactly: ask whether more exist */
        }
    }
    if (status == 0 && s_gop.full) {
        status = BW_EMSGSIZE; /* more characteristics than the table keeps */
    }
    return status;
}

/* Registers every notify/indicate-capable characteristic of s_chrs for
 * notification delivery. Bluedroid only routes ESP_GATTC_NOTIFY_EVT for
 * handles the application registered, and the registration is keyed by peer
 * address; the Go package turns payloads on and off by writing the CCCD, so
 * the glue pre-registers all candidates right after the cache fill - the
 * registration itself never enables anything. The API requires the cache to
 * exist (it validates each handle against it and refuses non-notify values),
 * which is why this runs at the search completion, after bw_fill_chrs. */
static void bw_register_notifs(uint32_t conn_id)
{
    esp_bd_addr_t bda;
    if (bw_conn_bda_of(conn_id, bda) != 0) {
        return; /* the connection is gone: nothing to register for */
    }
    for (int32_t i = 0; i < s_chr_all_n; i++) {
        if (s_chrs_all[i].properties & (ESP_GATT_CHAR_PROP_BIT_NOTIFY | ESP_GATT_CHAR_PROP_BIT_INDICATE)) {
            /* The completion event (ESP_GATTC_REG_FOR_NOTIFY_EVT) is ignored:
             * a refusal here only means the peer will never push on this
             * handle, which a CCCD write would surface anyway. */
            esp_ble_gattc_register_for_notify(s_gattc_if, bda, s_chrs_all[i].value_handle);
        }
    }
}

/* ---- GAP callback (BTC task) ----
 *
 * The scan API is one of the two places the BLE 4.2 / BLE 5.0 feature split
 * reaches this glue: the legacy set-scan-params/start-scanning pair and its
 * callback events exist only in a BLE 4.2-feature build, and the extended
 * pair (set-ext-scan-params / start-ext-scan) only in a BLE 5.0 one. The two
 * branches produce the same mailbox records: a scan stopped by request stays
 * silent (what StopScan's contract promises), a duration expiry reports
 * EV_SCAN_DONE, a refused start reports EV_ERROR. */

static void bw_on_scan_done(void)
{
    /* The claim (read + clear) is atomic against the Go task's submissions: a
     * duration expiry of an old scan must not swallow the operation id of a
     * scan the application started microseconds ago. Bluedroid's scan events
     * carry no operation identity of their own, so the claim ties the event
     * to whatever scan the glue currently owns; the narrow window where a
     * superseding submission slips between leaves an expiry unreported (the
     * scan itself is unaffected, and StopScan still works). NimBLE never had
     * this window because its host task serialized submissions and events. */
    uint32_t op = 0;
    BW_CTRL_LOCK();
    op = s_disc_op;
    s_disc_op = 0;
    BW_CTRL_UNLOCK();
    if (op != 0) {
        bw_ctrl_push(EV_SCAN_DONE, op, 0, 0);
    }
}

static void bw_gap_cb(esp_gap_ble_cb_event_t event, esp_ble_gap_cb_param_t *param)
{
#if defined(CONFIG_BT_BLE_42_FEATURES_SUPPORTED)
    switch (event) {
    case ESP_GAP_BLE_SCAN_PARAM_SET_COMPLETE_EVT:
        /* The legacy scan is a two-call procedure (set params, start); this
         * event is the first call's acknowledgement, nothing a program waits
         * on - the start's own completion follows. */
        break;
    case ESP_GAP_BLE_SCAN_START_COMPLETE_EVT:
        /* A refused start reports against the operation it tried to start,
         * claimed atomically so it cannot be confused with a scan submitted
         * since; a successful start is silent (the operation stays owned for
         * its duration-expiry report). */
        if (param->scan_start_cmpl.status != ESP_BT_STATUS_SUCCESS) {
            uint32_t op = 0;
            BW_CTRL_LOCK();
            op = s_disc_op;
            s_disc_op = 0;
            BW_CTRL_UNLOCK();
            if (op != 0) {
                bw_ctrl_push(EV_ERROR, op, 0, param->scan_start_cmpl.status);
            }
        }
        break;
    case ESP_GAP_BLE_SCAN_RESULT_EVT:
        if (param->scan_rst.search_evt == ESP_GAP_SEARCH_INQ_RES_EVT) {
            /* One advertising report. The buffer carries the advertising
             * payload first and the scan response after it; the record keeps
             * the advertising segment (falling back to the response when the
             * report has none - the observable content of NimBLE's second
             * DISC report). */
            uint8_t len = param->scan_rst.adv_data_len;
            const uint8_t *data = param->scan_rst.ble_adv;
            if (len == 0 && param->scan_rst.scan_rsp_len > 0) {
                len = param->scan_rst.scan_rsp_len;
                data = param->scan_rst.ble_adv + param->scan_rst.adv_data_len;
            }
            uint8_t addr[6];
            bw_addr_swap(addr, param->scan_rst.bda);
            bw_adv_push(addr, param->scan_rst.ble_addr_type,
                        (int8_t)param->scan_rst.rssi, data, len);
        } else if (param->scan_rst.search_evt == ESP_GAP_SEARCH_INQ_CMPL_EVT) {
            bw_on_scan_done(); /* the scan's duration ran out */
        }
        break;
    case ESP_GAP_BLE_SCAN_STOP_COMPLETE_EVT:
        /* A stop by request is silent on success (the contract StopScan
         * documents); a refused stop reports against the operation the stop
         * owned (s_stop_op, set at the StopScan submission and claimed here) -
         * never against whatever scan runs now. A completion with no owner is
         * the stop of an already-ended (or cancelled) scan: silent, exactly
         * like NimBLE's cancel. */
        {
            uint32_t op = 0;
            BW_CTRL_LOCK();
            op = s_stop_op;
            s_stop_op = 0;
            BW_CTRL_UNLOCK();
            if (op != 0 && param->scan_stop_cmpl.status != ESP_BT_STATUS_SUCCESS) {
                bw_ctrl_push(EV_ERROR, op, 0, param->scan_stop_cmpl.status);
            }
        }
        break;
    /* The peripheral half of the legacy GAP event set: the raw-payload
     * configuration completions chain into the start (the stack allows one
     * pending configuration at a time, so the chain is also the ordering),
     * and the start/stop completions claim the advertising operation and
     * report it as EV_ADV_DONE. */
    case ESP_GAP_BLE_ADV_DATA_RAW_SET_COMPLETE_EVT:
        if (param->adv_data_raw_cmpl.status != ESP_BT_STATUS_SUCCESS) {
            uint32_t op = s_adv_op;
            s_adv_op = 0;
            if (op != 0) {
                bw_ctrl_push(EV_ADV_DONE, op, 0, param->adv_data_raw_cmpl.status);
            }
            break;
        }
        if (s_scan_rsp_len > 0) {
            esp_err_t e = esp_ble_gap_config_scan_rsp_data_raw(s_scan_rsp, s_scan_rsp_len);
            if (e != ESP_OK) {
                uint32_t op = s_adv_op;
                s_adv_op = 0;
                if (op != 0) {
                    bw_ctrl_push(EV_ADV_DONE, op, 0, e);
                }
            }
            break;
        }
        bw_adv_start_now();
        break;
    case ESP_GAP_BLE_SCAN_RSP_DATA_RAW_SET_COMPLETE_EVT:
        if (param->scan_rsp_data_cmpl.status != ESP_BT_STATUS_SUCCESS) {
            uint32_t op = s_adv_op;
            s_adv_op = 0;
            if (op != 0) {
                bw_ctrl_push(EV_ADV_DONE, op, 0, param->scan_rsp_data_cmpl.status);
            }
            break;
        }
        bw_adv_start_now();
        break;
    case ESP_GAP_BLE_ADV_START_COMPLETE_EVT:
        {
            uint32_t op = s_adv_op;
            s_adv_op = 0;
            if (op != 0) {
                s_adv_running = param->adv_start_cmpl.status == ESP_BT_STATUS_SUCCESS;
                bw_ctrl_push(EV_ADV_DONE, op, 0, param->adv_start_cmpl.status);
            }
        }
        break;
    case ESP_GAP_BLE_ADV_STOP_COMPLETE_EVT:
        {
            uint32_t op = s_adv_op;
            s_adv_op = 0;
            if (op != 0) {
                s_adv_running = 0;
                bw_ctrl_push(EV_ADV_DONE, op, 0, param->adv_stop_cmpl.status);
            }
        }
        break;
    default:
        break;
    }
#else /* BLE 5.0-only build: the legacy scan events do not exist */
    switch (event) {
    case ESP_GAP_BLE_SET_EXT_SCAN_PARAMS_COMPLETE_EVT:
        break; /* first half of the extended start, nothing to report */
    case ESP_GAP_BLE_EXT_SCAN_START_COMPLETE_EVT:
        if (param->ext_scan_start.status != ESP_BT_STATUS_SUCCESS) {
            uint32_t op = 0;
            BW_CTRL_LOCK();
            op = s_disc_op;
            s_disc_op = 0;
            BW_CTRL_UNLOCK();
            if (op != 0) {
                bw_ctrl_push(EV_ERROR, op, 0, param->ext_scan_start.status);
            }
        }
        break;
    case ESP_GAP_BLE_EXT_ADV_REPORT_EVT:
        {
            uint8_t addr[6];
            bw_addr_swap(addr, param->ext_adv_report.params.addr);
            bw_adv_push(addr,
                        param->ext_adv_report.params.addr_type,
                        param->ext_adv_report.params.rssi,
                        param->ext_adv_report.params.adv_data,
                        param->ext_adv_report.params.adv_data_len);
        }
        break;
    case ESP_GAP_BLE_EXT_SCAN_STOP_COMPLETE_EVT:
        {
            uint32_t op = 0;
            BW_CTRL_LOCK();
            op = s_stop_op;
            s_stop_op = 0;
            BW_CTRL_UNLOCK();
            if (op != 0 && param->ext_scan_stop.status != ESP_BT_STATUS_SUCCESS) {
                bw_ctrl_push(EV_ERROR, op, 0, param->ext_scan_stop.status);
            }
        }
        break;
    case ESP_GAP_BLE_SCAN_TIMEOUT_EVT:
        bw_on_scan_done(); /* the scan's duration ran out */
        break;
    default:
        break;
    }
#endif
}

/* ---- GATTC callback (BTC task) ---- */

static void bw_gattc_cb(esp_gattc_cb_event_t event, esp_gatt_if_t gattc_if,
                        esp_ble_gattc_cb_param_t *param)
{
    switch (event) {
    case ESP_GATTC_REG_EVT:
        /* The one asynchronous bring-up step of the central half: the client
         * interface is ready. This is the Bluedroid counterpart of NimBLE's
         * sync callback - but EV_SYNC now waits for the server interface's
         * registration too (bw_sync_if_ready), so a CommitServices right
         * after Init never races its own interface. A refused registration
         * leaves the glue unusable; that failure surfaces as EV_ERROR (no
         * EV_SYNC ever comes, so Init's bounded wait times out with the
         * reason on record). */
        if (param->reg.status == ESP_GATT_OK) {
            s_gattc_if = gattc_if;
            s_gattc_reg = 1;
            bw_sync_if_ready();
        } else {
            bw_ctrl_push(EV_ERROR, 0, 0, param->reg.status);
        }
        break;
    case ESP_GATTC_CONNECT_EVT: {
        /* The ACL link is up. The attempt's outcome is reported here - the
         * OPEN event that follows only confirms the GATT layer on top, and a
         * direct connect that never comes up reports through OPEN alone. The
         * claim is atomic against the timeout timer's: whichever side takes
         * the operation id first is the only one that reports or cancels. */
        uint32_t op = 0;
        BW_CTRL_LOCK();
        op = s_conn_op;
        s_conn_op = 0;
        BW_CTRL_UNLOCK();
        esp_timer_stop(s_conn_timer);
        uint32_t id = bw_conn_add(param->connect.conn_id, param->connect.remote_bda);
        if (id == 0) {
            /* Connection table full: report the failure instead of a ConnID
             * 0, which the Go side would read as a successful connect. */
            bw_ctrl_push(EV_CONNECT, op, 0, BW_ENOMEM);
            break;
        }
        bw_ctrl_push(EV_CONNECT, op, id, 0);
        /* Ask for the ATT MTU exchange right away: the stack never does it by
         * itself (the ATT bearer starts at the 23-byte default), and both the
         * notification payload size and the largest single read depend on the
         * result. The negotiated value arrives as ESP_GATTC_CFG_MTU_EVT. */
        esp_ble_gattc_send_mtu_req(gattc_if, param->connect.conn_id);
        break;
    }
    case ESP_GATTC_OPEN_EVT:
        /* Connected connections arrive here with status OK after CONNECT
         * (nothing to report); a direct connect that failed reports here
         * alone, attributed to the attempt still in flight - claimed under
         * the same lock the timer and the CONNECT handler use. */
        if (param->open.status != ESP_GATT_OK) {
            uint32_t op = 0;
            BW_CTRL_LOCK();
            op = s_conn_op;
            s_conn_op = 0;
            BW_CTRL_UNLOCK();
            esp_timer_stop(s_conn_timer);
            if (op != 0) {
                bw_ctrl_push(EV_CONNECT, op, 0, param->open.status);
            }
            /* No attempt in flight: the completion of an attempt the
             * application cancelled (or the timeout timer already reported) -
             * nothing to attribute it to, and the application already knows. */
        }
        break;
    case ESP_GATTC_CANCEL_OPEN_EVT:
        break; /* the canceller (Cancel or the timeout timer) reported already */
    case ESP_GATTC_CFG_MTU_EVT:
        /* The negotiated MTU: cached on the connection (the MTU query reads
         * the same value) and reported once. */
        for (int i = 0; i < BW_CONNS; i++) {
            if (s_conns[i].used && s_conns[i].conn_id == param->cfg_mtu.conn_id) {
                s_conns[i].mtu = param->cfg_mtu.mtu;
            }
        }
        bw_ctrl_push(EV_MTU, 0, bw_conn_id_of(param->cfg_mtu.conn_id), param->cfg_mtu.mtu);
        break;
    case ESP_GATTC_SEARCH_RES_EVT: {
        /* One primary service of the search: fill the table record directly
         * (this is the NimBLE discovery callback's role). The match check
         * keeps a stray record of an older search from touching the table of
         * the procedure now in flight. */
        if (!bw_gop_match(GOP_DISC_SVCS, param->search_res.conn_id)) {
            break;
        }
        if (!param->search_res.is_primary) {
            break; /* the package discovers primary services only */
        }
        if (s_svc_n >= BW_SVC_MAX) {
            s_gop.full = 1;
            break;
        }
        llgo_ble_svc_t *rec = &s_svcs[s_svc_n++];
        bw_uuid_store(rec->uuid, &rec->uuid_len, &param->search_res.srvc_id.uuid);
        rec->start_handle = param->search_res.start_handle;
        rec->end_handle = param->search_res.end_handle;
        break;
    }
    case ESP_GATTC_SEARCH_CMPL_EVT: {
        /* The search is over and the stack's cache is final: read the
         * characteristics out right here, on the BTC task - the task that
         * also frees this cache on a disconnect, so this read can never race
         * a free (the old shape read it from the Go task; that was a
         * use-after-free waiting for a disconnect to land mid-read). The
         * completion record that follows publishes both tables - the service
         * records streamed in above, the characteristic records from
         * bw_fill_chrs - and llgo_ble_disc_chrs later slices the
         * characteristic table per range without touching the stack. The
         * match check mirrors SEARCH_RES: a second completion (the stack can
         * report one per result source) must not end a different procedure. */
        if (!bw_gop_match(GOP_DISC_SVCS, param->search_cmpl.conn_id)) {
            break;
        }
        int32_t status = param->search_cmpl.status;
        if (status == ESP_GATT_OK) {
            s_chr_all_n = 0;
            s_chr_all_conn = s_gop.conn_id;
            status = bw_fill_chrs(param->search_cmpl.conn_id);
            if (status == ESP_GATT_OK) {
                bw_register_notifs(s_gop.conn_id);
            }
        }
        bw_gop_done(EV_DISC_DONE, status);
        break;
    }
    case ESP_GATTC_READ_CHAR_EVT: {
        if (!bw_gop_match(GOP_READ, param->read.conn_id)) {
            break;
        }
        int32_t status = param->read.status;
        s_read_len = 0;
        if (status == ESP_GATT_OK && param->read.value_len > BW_DATA_MAX) {
            /* The value does not fit one event: the NimBLE backend's
             * flattening reported the same condition this way. */
            status = BW_EMSGSIZE;
        } else if (status == ESP_GATT_OK) {
            s_read_len = param->read.value_len;
            if (s_read_len > 0 && param->read.value != NULL) {
                memcpy(s_read_data, param->read.value, s_read_len);
            }
        }
        bw_gop_done(EV_READ_DONE, status);
        break;
    }
    case ESP_GATTC_WRITE_CHAR_EVT: {
        if (!bw_gop_match(GOP_WRITE, param->write.conn_id)) {
            break;
        }
        bw_gop_done(EV_WRITE_DONE, param->write.status);
        break;
    }
    case ESP_GATTC_NOTIFY_EVT:
        /* A notification or an indication; the stack has already built the
         * confirmation for an indication, so the glue only copies the payload
         * (the buffer dies with this callback). */
        bw_notif_push(param->notify.conn_id, param->notify.handle,
                      param->notify.is_notify ? 0 : 1,
                      param->notify.value, param->notify.value_len);
        break;
    case ESP_GATTC_REG_FOR_NOTIFY_EVT:
    case ESP_GATTC_UNREG_FOR_NOTIFY_EVT:
        break; /* the registration bookkeeping belongs to the discovery fill */
    case ESP_GATTC_CLOSE_EVT:
        /* The GATT virtual connection closed: a local close reports here
         * first, a link loss reports here too - either way this is the first
         * notice, so it reports the disconnection and retires the connection.
         * DISCONNECT (below) carries the link-layer reason; when both events
         * arrive, the second one finds no connection and stays silent, which
         * keeps one mailbox record per teardown. bw_conn_id_of returns the
         * raw conn_id when no live entry matches, so "equal" means already
         * retired. */
        if (bw_conn_id_of(param->close.conn_id) != param->close.conn_id) {
            bw_conn_retire(bw_conn_remove(param->close.conn_id), param->close.reason);
        }
        break;
    case ESP_GATTC_DISCONNECT_EVT: {
        /* The physical link terminated (the reason is the HCI code the peer
         * or controller reported). Only report when the CLOSE event above has
         * not already retired this connection. */
        if (bw_conn_id_of(param->disconnect.conn_id) != param->disconnect.conn_id) {
            bw_conn_retire(bw_conn_remove(param->disconnect.conn_id),
                           param->disconnect.reason);
        }
        break;
    }
    default:
        /* Every other GATTC event (congestion, queue fullness, prepared
         * writes, ...) belongs to features this package does not expose yet;
         * ignoring it keeps the mailbox bounded. */
        break;
    }
}

/* ---- lifecycle (Go task, synchronous like NimBLE's Init/Deinit) ---- */

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

    /* Controller first. BT_CONTROLLER_INIT_CONFIG_DEFAULT carries the
     * sdkconfig's mode (CONFIG_BTDM_CTRL_MODE_BLE_ONLY for the BLE-only
     * firmware this package targets); the enable call's mode argument is
     * checked against that compiled-in mode, not used to pick it. */
    esp_bt_controller_config_t cfg = BT_CONTROLLER_INIT_CONFIG_DEFAULT();
    e = esp_bt_controller_init(&cfg);
    if (e != ESP_OK) {
        return e;
    }
    e = esp_bt_controller_enable(ESP_BT_MODE_BLE);
    if (e != ESP_OK) {
        esp_bt_controller_deinit();
        return e;
    }

    /* Host stack. Plain esp_bluedroid_init is enough for the central role
     * (no classic-BT pairing profile: SSP/SC configuration belongs to an
     * esp_bluedroid_init_with_cfg firmware). esp_bluedroid_enable is the
     * synchronous step NimBLE spread across its sync callback: it returns
     * only when the whole stack is up. */
    e = esp_bluedroid_init();
    if (e != ESP_OK) {
        esp_bt_controller_disable();
        esp_bt_controller_deinit();
        return e;
    }
    e = esp_bluedroid_enable();
    if (e != ESP_OK) {
        esp_bluedroid_deinit();
        esp_bt_controller_disable();
        esp_bt_controller_deinit();
        return e;
    }

    /* Pairing posture: bonding with MITM (secure connections where the peer
     * supports them), display-only IO so a KeyboardOnly peer resolves to
     * passkey entry with the passkey printed on this console, and both
     * encryption and identity keys distributed in both directions. Every
     * parameter is set explicitly: Bluedroid keeps no usable defaults, and a
     * pairing left without them reports success while distributing no key
     * (AUTH_CMPL key_present = 0), so nothing is stored and a bonded peer
     * has to re-pair on every reconnect. */
    {
        uint8_t auth_req = ESP_LE_AUTH_REQ_SC_MITM_BOND;
        uint8_t io_cap = ESP_IO_CAP_OUT;
        uint8_t key_size = 16; /* BLE maximum; unset => no key distributed */
        uint8_t keys = ESP_BLE_ENC_KEY_MASK | ESP_BLE_ID_KEY_MASK;
        esp_ble_gap_set_security_param(ESP_BLE_SM_AUTHEN_REQ_MODE, &auth_req,
                                       sizeof auth_req);
        esp_ble_gap_set_security_param(ESP_BLE_SM_IOCAP_MODE, &io_cap,
                                       sizeof io_cap);
        esp_ble_gap_set_security_param(ESP_BLE_SM_MAX_KEY_SIZE, &key_size,
                                       sizeof key_size);
        esp_ble_gap_set_security_param(ESP_BLE_SM_SET_INIT_KEY, &keys,
                                       sizeof keys);
        esp_ble_gap_set_security_param(ESP_BLE_SM_SET_RSP_KEY, &keys,
                                       sizeof keys);
    }

    /* The address type every own-side call carries. NimBLE inferred it from
     * the controller's identity; here the device address is available as soon
     * as the stack is enabled, and public-vs-random is the same distinction. */
    s_own_type = esp_bt_dev_get_address() != NULL ? BLE_ADDR_TYPE_PUBLIC
                                                  : BLE_ADDR_TYPE_RANDOM;

    /* The negotiated-MTU target. 256 is the value the package documents (the
     * NimBLE firmware declared it as its preferred MTU); the peer may cap the
     * exchange below it. */
    esp_ble_gatt_set_local_mtu(256);

    /* Callbacks and the client interface. app_register completes
     * asynchronously (ESP_GATTC_REG_EVT above), which is what turns
     * s_synced on - together with the server interface's own registration,
     * also asynchronous, so a CommitServices right after Init cannot race
     * its interface. A build without the GATT server component refuses the
     * server registration and still syncs: the central half is fully usable,
     * CommitServices reports the refusal honestly. */
    e = esp_ble_gap_register_callback(bw_gap_cb);
    if (e != ESP_OK) {
        goto rollback;
    }
    e = esp_ble_gattc_register_callback(bw_gattc_cb);
    if (e != ESP_OK) {
        goto rollback;
    }
    e = esp_ble_gattc_app_register(0);
    if (e != ESP_OK) {
        goto rollback;
    }
    e = esp_ble_gatts_register_callback(bw_gatts_cb);
    if (e != ESP_OK) {
        s_gatts_fail = 1;
        bw_sync_if_ready();
    } else {
        e = esp_ble_gatts_app_register(0);
        if (e != ESP_OK) {
            /* Not a rollback: the central half is alive, only the server
             * role is unavailable in this firmware. */
            s_gatts_fail = 1;
            bw_sync_if_ready();
        }
    }

    /* The connect-timeout timer. One-shot, re-armed per attempt. */
    esp_timer_create_args_t timer_args = {
        .callback = bw_conn_timer_cb,
        .name = "bw-conn",
    };
    e = esp_timer_create(&timer_args, &s_conn_timer);
    if (e != ESP_OK) {
        goto rollback;
    }

    s_inited = 1;
    return ESP_OK;

rollback:
    esp_bluedroid_disable();
    esp_bluedroid_deinit();
    esp_bt_controller_disable();
    esp_bt_controller_deinit();
    /* The stack is down, so no further callbacks can fire: drain anything it
     * pushed while it was dying. Without this, a sync that arrived between
     * app_register and the timer failure would sit in the ring and
     * masquerade as the next Init's sync - a bring-up that failed would look
     * like one that succeeded. s_synced/s_gattc_if and the server role's
     * flags go down with it: the interfaces the dying stack handed out are
     * void. */
    s_synced = 0;
    s_gattc_if = 0;
    s_gattc_reg = 0;
    s_gatts_reg = 0;
    s_gatts_fail = 0;
    s_gatts_if = 0;
    BW_CTRL_LOCK();
    s_ctrl_rd = BW_ACQUIRE(s_ctrl_wr); /* consumer-side write: this is the consumer task */
    BW_CTRL_UNLOCK();
    return e;
}

int32_t llgo_ble_deinit(void)
{
    if (!s_inited) {
        return ESP_ERR_INVALID_STATE;
    }
    s_inited = 0;
    s_synced = 0;
    /* The teardown order is the bring-up's mirror, each step synchronous: the
     * host stack stops answering callbacks before the controller under it
     * goes away. (Bluedroid's documented constraint: a disable/deinit stack
     * cannot re-enable in place - a fresh init/enable cycle is required,
     * which is what a second Init runs.) */
    esp_timer_stop(s_conn_timer);
    esp_timer_delete(s_conn_timer);
    s_conn_op = 0;
    s_disc_op = 0;
    /* The peripheral half's in-flight state: an advertising operation or a
     * service registration that never completed has no stack left to complete
     * on, the connections are gone, and a fresh Init must not inherit them. */
    s_adv_op = 0;
    s_adv_running = 0;
    s_gatts_op = 0;
    s_gatts_building = 0;
    s_gsvc_n = 0;
    s_gchr_n = 0;
    for (int i = 0; i < BW_PCONN_MAX; i++) {
        s_pconns[i].used = 0;
    }
    int32_t err = ESP_OK;
    esp_err_t e = esp_bluedroid_disable();
    if (e != ESP_OK) {
        err = e;
    }
    e = esp_bluedroid_deinit();
    if (e != ESP_OK && err == ESP_OK) {
        err = e;
    }
    e = esp_bt_controller_disable();
    if (e != ESP_OK && err == ESP_OK) {
        err = e;
    }
    e = esp_bt_controller_deinit();
    if (e != ESP_OK && err == ESP_OK) {
        err = e;
    }
    return err;
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
        const uint8_t *dev = esp_bt_dev_get_address();
        if (dev != NULL) {
            bw_addr_swap(addr, dev); /* print the conventional MSB-first form */
            have = 1;
        }
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
        BW_PUBLISH(s_periph_rd, s_periph_rd + 1);
        return 1;
    }

    if (BW_ACQUIRE(s_notif_wr) != s_notif_rd) {
        /* Read in place - a local copy would put a whole slot on the Go task's
         * (fiber) stack - and publish the read index only after the copy, so
         * the BTC task cannot reuse the slot under us. */
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

/* ---- central commands (Go task) ----
 *
 * Every function returns 0 when the stack accepted the call and a status when
 * it refused it; the outcome of an accepted call arrives through the mailbox,
 * never as a return value. Unlike the NimBLE glue there is no intermediate
 * queue: the esp_ble_* call deep-copies its arguments into the BTC task's
 * message on submission, which is exactly the "hand the work to the stack
 * task" step the command ring used to implement. */

int32_t llgo_ble_scan_start(int32_t itvl_ms, int32_t window_ms, int32_t passive,
                           int32_t filter_dup, int32_t duration_ms, uint32_t op_id)
{
    if (!s_inited) {
        return BW_EINVAL; /* no stack: nothing to submit to */
    }
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
    uint16_t itvl = (uint16_t)((itvl_ms * 8) / 5);
    uint16_t window = (uint16_t)((window_ms * 8) / 5);
    /* The operation id goes up under the ctrl ring's lock, before the calls:
     * the stack's acknowledgements arrive on the BTC task, which can run as
     * soon as the first submission is posted, so the set must be atomic
     * against their claims. A refused start reports EV_ERROR against it, the
     * same record the NimBLE host produced for a discovery it could not start
     * (and an already-running scan keeps running with its own operation,
     * exactly as NimBLE's failed second discovery did). */
    BW_CTRL_LOCK();
    s_disc_op = op_id;
    BW_CTRL_UNLOCK();
    esp_err_t e;
#if defined(CONFIG_BT_BLE_42_FEATURES_SUPPORTED)
    memset(&s_scan_params, 0, sizeof s_scan_params);
    s_scan_params.scan_type = passive ? BLE_SCAN_TYPE_PASSIVE : BLE_SCAN_TYPE_ACTIVE;
    s_scan_params.own_addr_type = s_own_type;
    s_scan_params.scan_filter_policy = BLE_SCAN_FILTER_ALLOW_ALL;
    s_scan_params.scan_duplicate = filter_dup ? BLE_SCAN_DUPLICATE_ENABLE
                                              : BLE_SCAN_DUPLICATE_DISABLE;
    s_scan_params.scan_interval = itvl;
    s_scan_params.scan_window = window;
    e = esp_ble_gap_set_scan_params(&s_scan_params);
    if (e == ESP_OK) {
        /* The legacy duration counts seconds. Only a positive DurationMs
         * bounds the scan (rounded up, so any bound is honored): 0 and a
         * negative value both scan until StopScan - what DurationMs 0
         * promises, and what NimBLE's ble_gap_disc did with a non-positive
         * duration (no expiry). ble.go passes the field through unclamped, so
         * the parity here is deliberate. */
        uint32_t seconds = 0;
        if (duration_ms > 0) {
            seconds = ((uint32_t)duration_ms + 999) / 1000;
        }
        e = esp_ble_gap_start_scanning(seconds);
    }
#else
    memset(&s_scan_params, 0, sizeof s_scan_params);
    s_scan_params.own_addr_type = s_own_type;
    s_scan_params.filter_policy = BLE_SCAN_FILTER_ALLOW_ALL;
    s_scan_params.scan_duplicate = filter_dup ? BLE_SCAN_DUPLICATE_ENABLE
                                              : BLE_SCAN_DUPLICATE_DISABLE;
    s_scan_params.cfg_mask = ESP_BLE_GAP_EXT_SCAN_CFG_UNCODE_MASK;
    s_scan_params.uncoded_cfg.scan_type = passive ? BLE_SCAN_TYPE_PASSIVE
                                                  : BLE_SCAN_TYPE_ACTIVE;
    s_scan_params.uncoded_cfg.scan_interval = itvl;
    s_scan_params.uncoded_cfg.scan_window = window;
    e = esp_ble_gap_set_ext_scan_params(&s_scan_params);
    if (e == ESP_OK) {
        /* The extended duration counts 10 ms windows. Same non-positive rule
         * as the legacy branch: 0 and negative scan until StopScan, matching
         * NimBLE. */
        uint32_t windows = 0;
        if (duration_ms > 0) {
            windows = ((uint32_t)duration_ms + 9) / 10;
        }
        e = esp_ble_gap_start_ext_scan(windows, 0);
    }
#endif
    if (e != ESP_OK) {
        BW_CTRL_LOCK();
        s_disc_op = 0;
        BW_CTRL_UNLOCK();
        return e;
    }
    return 0;
}

int32_t llgo_ble_scan_stop(uint32_t op_id)
{
    if (!s_inited) {
        return BW_EINVAL;
    }
    /* The running scan's id moves under the lock from s_disc_op to s_stop_op:
     * the stop now owns it, and its completion event claims exactly that
     * ownership - never whatever scan runs by then (the old unconditional
     * clear could kill the operation of a scan submitted between the stop and
     * its completion, leaving it unreportable). A successful stop stays
     * silent; a refused one reports EV_ERROR against the owned operation. A
     * stop with no scan running owns nothing and is silent, which is the
     * shape NimBLE's successful stop had. */
    (void)op_id;
    uint32_t op = 0;
    BW_CTRL_LOCK();
    op = s_disc_op;
    s_disc_op = 0;
    s_stop_op = op; /* 0 when no scan was running */
    BW_CTRL_UNLOCK();
    if (op == 0) {
        return 0;
    }
#if defined(CONFIG_BT_BLE_42_FEATURES_SUPPORTED)
    esp_err_t e = esp_ble_gap_stop_scanning();
#else
    esp_err_t e = esp_ble_gap_stop_ext_scan();
#endif
    if (e != ESP_OK) {
        /* The stack refused the stop synchronously: no completion event will
         * come for it, so give the scan its operation back instead of leaving
         * it owned by a stop that never happened. */
        BW_CTRL_LOCK();
        s_disc_op = op;
        s_stop_op = 0;
        BW_CTRL_UNLOCK();
        return e;
    }
    return 0;
}

/* Stored bonds: one seven-byte record per peer (six address bytes in stack
 * order, then the address type). A caller connects straight to a returned
 * address - a bonded keyboard waits for the central instead of advertising,
 * and the stored key skips the pairing procedure. Static storage on purpose:
 * this runs outside the Go scheduler, where the wrapped allocator has no
 * locking story. */
int32_t llgo_ble_bond_list(uint8_t *out, int32_t max_peers)
{
    static esp_ble_bond_dev_t s_bonds[8];
    if (!s_inited || out == NULL || max_peers <= 0) {
        return 0;
    }
    int num = esp_ble_get_bond_device_num();
    if (num <= 0) {
        return 0;
    }
    int want = num > 8 ? 8 : num;
    if (want > max_peers) {
        want = max_peers;
    }
    int got = want;
    if (esp_ble_get_bond_device_list(&got, s_bonds) != ESP_OK) {
        return 0;
    }
    for (int i = 0; i < got; i++) {
        /* The Go record carries the stack's little-endian order, exactly
         * like every address in an event (see bw_addr_swap); connecting
         * swaps it back. A raw memcpy here hands Connect a reversed
         * address, which fails to establish (0x100). */
        bw_addr_swap(out + i * 7, s_bonds[i].bd_addr);
        out[i * 7 + 6] = 0; /* bonded peers are public-address devices here */
    }
    return got;
}

int32_t llgo_ble_connect(const uint8_t *addr, int32_t addr_type, int32_t timeout_ms,
                        uint32_t op_id)
{
    if (!s_inited) {
        return BW_EINVAL;
    }
    if (addr == NULL) {
        return BW_EINVAL;
    }
    /* The same link parameters the NimBLE glue submitted (30-50 ms interval,
     * 4 s supervision): a central's conventional first attempt. enh_open
     * copies the pointed-to params into the BTC message before returning. */
    memset(&s_conn_link, 0, sizeof s_conn_link);
    s_conn_link.interval_min = 24;     /* 30 ms */
    s_conn_link.interval_max = 40;     /* 50 ms */
    s_conn_link.latency = 0;
    s_conn_link.supervision_timeout = 400; /* 4 s */
    esp_ble_gatt_creat_conn_params_t cp;
    memset(&cp, 0, sizeof cp);
    /* The peer address travels stack-order (see bw_addr_swap): the connect
     * request and the cancel-open that a Cancel or the timeout may later
     * send both key off s_conn_bda. */
    bw_addr_swap(s_conn_bda, addr);
    memcpy(cp.remote_bda, s_conn_bda, ESP_BD_ADDR_LEN);
    cp.remote_addr_type = (esp_ble_addr_type_t)addr_type;
    cp.is_direct = true;   /* a one-shot attempt, not a background re-connect */
    cp.is_aux = false;     /* connect through legacy advertising (the stack
                            * resolves the HCI procedure from its feature set) */
    cp.own_addr_type = s_own_type;
    cp.phy_mask = ESP_BLE_PHY_1M_PREF_MASK;
    cp.phy_1m_conn_params = &s_conn_link;
    /* The attempt goes in flight under the ctrl ring's lock, before the
     * call, for the same reason the scan's operation does: the BTC task (and
     * the timeout timer) can answer it right away, and the claim pattern must
     * match theirs. A refused open reports EV_ERROR against it from the OPEN
     * event, the record the NimBLE host produced for an attempt it could not
     * start. */
    BW_CTRL_LOCK();
    s_conn_op = op_id;
    BW_CTRL_UNLOCK();
    esp_err_t e = esp_ble_gattc_enh_open(s_gattc_if, &cp);
    if (e != ESP_OK) {
        BW_CTRL_LOCK();
        s_conn_op = 0;
        BW_CTRL_UNLOCK();
        return e;
    }
    /* NimBLE's connect carries the deadline natively; Bluedroid's does not,
     * so the glue's timer enforces it (the timeout's failure record uses the
     * code the package documents for this condition). */
    if (timeout_ms > 0) {
        esp_timer_start_once(s_conn_timer, (uint64_t)timeout_ms * 1000);
    }
    return 0;
}

int32_t llgo_ble_cancel(uint32_t op_id)
{
    if (!s_inited) {
        return BW_EINVAL;
    }
    /* Both branches claim the operation id under the lock (read + clear): a
     * completion racing the cancel - the CONNECT handler, the timeout timer -
     * then either wins the claim and reports, or loses it and stays silent,
     * never both. */
    if (op_id != 0) {
        uint32_t held = 0;
        BW_CTRL_LOCK();
        if (s_disc_op == op_id) {
            s_disc_op = 0; /* the cancelled scan reports nothing further */
            held = 1;
        }
        BW_CTRL_UNLOCK();
        if (held) {
#if defined(CONFIG_BT_BLE_42_FEATURES_SUPPORTED)
            esp_err_t e = esp_ble_gap_stop_scanning();
#else
            esp_err_t e = esp_ble_gap_stop_ext_scan();
#endif
            if (e != ESP_OK) {
                bw_ctrl_push(EV_ERROR, op_id, 0, e);
            }
            return 0;
        }
        BW_CTRL_LOCK();
        if (s_conn_op == op_id) {
            s_conn_op = 0;
            held = 1;
        }
        BW_CTRL_UNLOCK();
        if (held) {
            /* Cancel the pending direct connect. The attempt reports nothing
             * further - the same silent-cancel contract Cancel documents; a
             * connection that completes anyway arrives as an unattributed
             * EV_CONNECT (OpID 0), exactly like NimBLE's late completion. */
            esp_timer_stop(s_conn_timer);
            s_cancel_open.gattc_if = s_gattc_if;
            memcpy(s_cancel_open.remote_bda, s_conn_bda, ESP_BD_ADDR_LEN);
            esp_err_t e = esp_ble_gattc_cancel_open(&s_cancel_open);
            if (e != ESP_OK) {
                bw_ctrl_push(EV_ERROR, op_id, 0, e);
            }
            return 0;
        }
    }
    bw_ctrl_push(EV_ERROR, op_id, 0, BW_ENOENT); /* not the operation in flight */
    return 0;
}

/* Copies the host-stack name into buf; returns the length copied. The Go side
 * reports it through Backend() so a program can log which stack it runs. */
int32_t llgo_ble_backend_name(uint8_t *buf, int32_t n)
{
    static const char name[] = "bluedroid";
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
    if (!s_inited) {
        return BW_EINVAL;
    }
    int raw = bw_conn_find(conn_id);
    if (raw < 0) {
        bw_ctrl_push(EV_ERROR, 0, conn_id, BW_ENOENT);
        return 0;
    }
    /* Closes the GATT virtual connection and the ACL under it; the teardown
     * arrives as CLOSE/DISCONNECT on the mailbox. */
    esp_err_t e = esp_ble_gattc_close(s_gattc_if, (uint16_t)raw);
    if (e != ESP_OK) {
        bw_ctrl_push(EV_ERROR, 0, conn_id, e);
    }
    return 0;
}

/* ---- GATT client (Go task) ----
 *
 * Same contract as the GAP commands: 0 means "the stack accepted the
 * procedure", a status means "not submitted", and the outcome always arrives
 * through the mailbox. The accessors below are plain reads of what the BTC
 * task (or, for the characteristic fill, this very task) published, and -
 * like every other result the mailbox announces - the record a caller reads
 * is complete before the completion record that points at it. */

int32_t llgo_ble_disc_svcs(uint32_t conn_id, uint32_t op_id)
{
    if (!s_inited) {
        return BW_EINVAL;
    }
    int raw = bw_conn_find(conn_id);
    if (raw < 0) {
        bw_ctrl_push(EV_ERROR, op_id, conn_id, BW_ENOENT);
        return 0;
    }
    if (!bw_gop_claim(GOP_DISC_SVCS, op_id, conn_id)) {
        bw_ctrl_push(EV_ERROR, op_id, conn_id, BW_EALREADY);
        return 0;
    }
    s_svc_n = 0;
    s_svc_conn = conn_id;
    /* Both characteristic tables reset here: the staging table refills at the
     * search completion, the view table at each disc_chrs slice. */
    s_chr_all_n = 0;
    s_chr_all_conn = 0;
    s_chr_n = 0;
    s_chr_conn = 0;
    /* A NULL filter discovers every service; the primary ones land in the
     * table as the SEARCH_RES records stream past, and the search completion
     * reads the characteristics out of the stack's cache (on the BTC task). */
    esp_err_t e = esp_ble_gattc_search_service(s_gattc_if, (uint16_t)raw, NULL);
    if (e != ESP_OK) {
        s_svc_n = 0;
        bw_gop_submit_failed(e);
    }
    return 0;
}

int32_t llgo_ble_disc_chrs(uint32_t conn_id, uint16_t start_handle, uint16_t end_handle,
                           uint32_t op_id)
{
    if (!s_inited) {
        return BW_EINVAL;
    }
    int raw = bw_conn_find(conn_id);
    if (raw < 0) {
        bw_ctrl_push(EV_ERROR, op_id, conn_id, BW_ENOENT);
        return 0;
    }
    if (!bw_gop_claim(GOP_DISC_CHRS, op_id, conn_id)) {
        bw_ctrl_push(EV_ERROR, op_id, conn_id, BW_EALREADY);
        return 0;
    }
    /* The slice, not a discovery: the search that staged s_chrs_all completed
     * before this call (ble.go only names ranges its Services() reported, on
     * this same connection), and this call copies the staged records of the
     * requested handle range into the view table - no stack cache access,
     * which is what a disconnect's cache free would race. Copying (not
     * compacting) keeps the staged set intact, so every Characteristics call
     * of the connection slices the same full set - the observable behavior of
     * NimBLE's discovery, which re-ran per call. The staging table changes
     * only at the next search's completion, which cannot overlap this slice:
     * the Go-side guard holds this procedure open while it runs. The view
     * table holds the slice (bounded by the staging table's cap, so it cannot
     * overflow), and - like NimBLE's per-range discovery - a range with no
     * characteristics ends here with success and zero records. The completion
     * record shape and the return codes are the NimBLE procedure's: 0
     * accepted, the outcome through the mailbox. */
    int32_t n = 0;
    for (int32_t i = 0; i < s_chr_all_n; i++) {
        if (s_chrs_all[i].value_handle >= start_handle && s_chrs_all[i].value_handle <= end_handle) {
            if (n < BW_CHR_MAX) {
                s_chrs[n++] = s_chrs_all[i];
            }
        }
    }
    s_chr_n = n;
    s_chr_conn = conn_id;
    bw_gop_done(EV_DISC_DONE, 0);
    return 0;
}

int32_t llgo_ble_read(uint32_t conn_id, uint16_t handle, uint32_t op_id)
{
    if (!s_inited) {
        return BW_EINVAL;
    }
    if (handle == 0) {
        return BW_EINVAL;
    }
    int raw = bw_conn_find(conn_id);
    if (raw < 0) {
        bw_ctrl_push(EV_ERROR, op_id, conn_id, BW_ENOENT);
        return 0;
    }
    if (!bw_gop_claim(GOP_READ, op_id, conn_id)) {
        bw_ctrl_push(EV_ERROR, op_id, conn_id, BW_EALREADY);
        return 0;
    }
    s_read_len = 0;
    esp_err_t e = esp_ble_gattc_read_char(s_gattc_if, (uint16_t)raw, handle,
                                          ESP_GATT_AUTH_REQ_NONE);
    if (e != ESP_OK) {
        bw_gop_submit_failed(e);
    }
    return 0;
}

/* The ATT write procedures: the Go side picks one from the characteristic's
 * properties, or names it explicitly (the same numbers the NimBLE glue's
 * command carried). */
enum {
    BW_WRITE_RSP = 1,  /* ATT Write Request: the peer acknowledges */
    BW_WRITE_NRSP = 2, /* ATT Write Command: the peer does not */
};

int32_t llgo_ble_write(uint32_t conn_id, uint16_t handle, const uint8_t *data, int32_t len,
                       int32_t mode, uint32_t op_id)
{
    if (!s_inited) {
        return BW_EINVAL;
    }
    if (handle == 0 || len < 0 || (len > 0 && data == NULL)) {
        return BW_EINVAL;
    }
    if (len > BW_DATA_MAX) {
        return BW_EMSGSIZE;
    }
    if (mode != BW_WRITE_RSP && mode != BW_WRITE_NRSP) {
        return BW_EINVAL;
    }
    int raw = bw_conn_find(conn_id);
    if (raw < 0) {
        bw_ctrl_push(EV_ERROR, op_id, conn_id, BW_ENOENT);
        return 0;
    }
    if (!bw_gop_claim(GOP_WRITE, op_id, conn_id)) {
        bw_ctrl_push(EV_ERROR, op_id, conn_id, BW_EALREADY);
        return 0;
    }
    /* The value pointer goes straight through: the API deep-copies it into
     * the BTC message before returning (no NimBLE-style staging slot, and no
     * window where a timed-out Go caller could race the stack's copy). The
     * mode numbers are the ABI's; the stack's enum counts the other way,
     * hence the flip. */
    esp_gatt_write_type_t write_type = (mode == BW_WRITE_NRSP) ? ESP_GATT_WRITE_TYPE_NO_RSP
                                                               : ESP_GATT_WRITE_TYPE_RSP;
    esp_err_t e = esp_ble_gattc_write_char(s_gattc_if, (uint16_t)raw, handle,
                                           (uint16_t)len, (uint8_t *)data, write_type,
                                           ESP_GATT_AUTH_REQ_NONE);
    if (mode == BW_WRITE_NRSP) {
        /* A write command is never acknowledged, so there is no callback to
         * wait for: the submission result is the whole outcome. (This push
         * comes from the Go task - the ctrl ring's critical section makes it
         * safe next to a concurrent BTC-task completion.) */
        bw_gop_done(e == ESP_OK ? EV_WRITE_DONE : EV_ERROR,
                    e == ESP_OK ? 0 : (int32_t)e);
    } else if (e != ESP_OK) {
        bw_gop_submit_failed(e);
    }
    return 0;
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

/* The negotiated ATT MTU of a live connection: the value the CFG_MTU event
 * cached when the glue-triggered exchange completed (the ATT default before
 * that, exactly what NimBLE's ble_att_mtu reported pre-exchange). */
int32_t llgo_ble_mtu(uint32_t conn_id, int32_t *out)
{
    if (out == NULL) {
        return BW_EINVAL;
    }
    if (!s_inited) {
        return BW_ENOTCONN;
    }
    uint16_t raw = (uint16_t)(conn_id & 0xFFFF);
    uint32_t gen = conn_id >> 16;
    for (int i = 0; i < BW_CONNS; i++) {
        if (s_conns[i].used && s_conns[i].conn_id == raw && s_conns[i].gen == gen) {
            *out = (int32_t)s_conns[i].mtu;
            return 0;
        }
    }
    return BW_ENOENT;
}

/* ---- peripheral (Go task) ----
 *
 * Same contract as the central commands: 0 means the stack accepted the call
 * (the outcome arrives through the mailbox), a status means it was refused
 * synchronously. The config/start chain of an advertisement, the service
 * creation, and the connection teardown all complete on the BTC task through
 * EV_ADV_DONE / EV_SERVICES_DONE / EV_PERIPH_DISCONNECT. */

int32_t llgo_ble_adv_start(const uint8_t *adv, int32_t adv_len, const uint8_t *rsp,
                           int32_t rsp_len, int32_t itvl_ms, int32_t connectable,
                           uint32_t op_id)
{
#if defined(CONFIG_BT_BLE_42_FEATURES_SUPPORTED)
    if (!s_inited || !s_synced) {
        return BW_EINVAL;
    }
    if (adv_len < 0 || adv_len > BW_ADV_MAX || rsp_len < 0 || rsp_len > BW_ADV_MAX) {
        return BW_EMSGSIZE;
    }
    if ((adv_len > 0 && adv == NULL) || (rsp_len > 0 && rsp == NULL)) {
        return BW_EINVAL;
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
    /* One advertisement lifecycle at a time: a second start while the first
     * is still completing is refused, the same guard a scan start has. */
    BW_CTRL_LOCK();
    if (s_adv_op != 0) {
        BW_CTRL_UNLOCK();
        return BW_EALREADY;
    }
    s_adv_op = op_id;
    BW_CTRL_UNLOCK();
    memcpy(s_adv_data, adv, (size_t)adv_len);
    s_adv_data_len = (uint8_t)adv_len;
    memcpy(s_scan_rsp, rsp, (size_t)rsp_len);
    s_scan_rsp_len = (uint8_t)rsp_len;
    s_adv_conn = connectable ? 1 : 0;
    s_adv_itvl = (uint16_t)((itvl_ms * 8) / 5); /* 0.625 ms units */
    esp_err_t e = esp_ble_gap_config_adv_data_raw(s_adv_data, (uint32_t)adv_len);
    if (e != ESP_OK) {
        BW_CTRL_LOCK();
        s_adv_op = 0;
        BW_CTRL_UNLOCK();
        return e;
    }
    return 0;
#else
    /* The legacy advertising procedures do not exist in a BLE 5.0-only
     * feature build; this package's peripheral role is legacy-only, matching
     * the scan path's scope. */
    (void)adv; (void)adv_len; (void)rsp; (void)rsp_len;
    (void)itvl_ms; (void)connectable; (void)op_id;
    return ESP_ERR_NOT_SUPPORTED;
#endif
}

int32_t llgo_ble_adv_stop(uint32_t op_id)
{
#if defined(CONFIG_BT_BLE_42_FEATURES_SUPPORTED)
    if (!s_inited || !s_synced) {
        return BW_EINVAL;
    }
    BW_CTRL_LOCK();
    if (s_adv_op != 0) {
        BW_CTRL_UNLOCK();
        return BW_EALREADY;
    }
    if (!s_adv_running) {
        BW_CTRL_UNLOCK();
        /* Nothing is advertising (never started, or a connect ended it):
         * the operation completes here rather than dangling. */
        bw_ctrl_push(EV_ADV_DONE, op_id, 0, BW_EINVAL);
        return 0;
    }
    s_adv_op = op_id;
    BW_CTRL_UNLOCK();
    esp_err_t e = esp_ble_gap_stop_advertising();
    if (e != ESP_OK) {
        BW_CTRL_LOCK();
        s_adv_op = 0;
        BW_CTRL_UNLOCK();
        return e;
    }
    return 0;
#else
    (void)op_id;
    return ESP_ERR_NOT_SUPPORTED;
#endif
}

int32_t llgo_ble_gatts_declare(const llgo_ble_gsvc_t *svcs, int32_t nsvc,
                               const llgo_ble_gchr_t *chrs, int32_t nchr, uint32_t op_id)
{
    if (!s_inited || !s_synced) {
        return BW_EINVAL;
    }
    if (svcs == NULL || chrs == NULL || nsvc <= 0 || nchr <= 0 ||
        nsvc > BW_GSVC_MAX || nchr > BW_GCHR_MAX) {
        return BW_EINVAL;
    }
    if (s_gatts_fail || s_gatts_if == 0) {
        /* This firmware has no GATT server (the registration failed or the
         * component is compiled out); the refusal is the honest answer. */
        return ESP_ERR_NOT_SUPPORTED;
    }
    BW_CTRL_LOCK();
    if (s_gatts_building || s_pconn_any()) {
        /* One registration at a time, and re-declaring under a connected
         * peer would pull attributes out from under its reads. */
        BW_CTRL_UNLOCK();
        return BW_EALREADY;
    }
    s_gatts_building = 1;
    s_gatts_op = op_id;
    BW_CTRL_UNLOCK();

    /* Serialize the declaration into the static tables. The characteristic
     * count per service re-derives the attribute budget the Go side must
     * have declared (1 + 2..3 per characteristic); a mismatch is refused
     * here rather than left to strand the creation mid-build. */
    s_gsvc_n = 0;
    s_gchr_n = 0;
    int32_t ci = 0;
    for (int32_t i = 0; i < nsvc; i++) {
        bw_gsvc_t *svc = &s_gsvcs[s_gsvc_n];
        memset(svc, 0, sizeof *svc);
        svc->uuid_len = svcs[i].uuid_len;
        memcpy(svc->uuid, svcs[i].uuid, 16);
        svc->n_chars = svcs[i].n_chars;
        svc->first_chr = (uint16_t)s_gchr_n;
        if (s_gchr_n + (int32_t)svc->n_chars > nchr || s_gchr_n + (int32_t)svc->n_chars > BW_GCHR_MAX) {
            /* The flat characteristic array does not cover this service's
             * count: the wire is inconsistent, refuse before anything is
             * built (the Go side's own checkRegistry should have caught it). */
            s_gsvc_n = 0;
            s_gchr_n = 0;
            s_gatts_building = 0;
            s_gatts_op = 0;
            return BW_EINVAL;
        }
        uint16_t need = 1;
        for (int32_t k = 0; k < (int32_t)svc->n_chars; k++, ci++) {
            bw_gchr_t *chr = &s_gchrs[s_gchr_n];
            memset(chr, 0, sizeof *chr);
            chr->uuid_len = chrs[ci].uuid_len;
            memcpy(chr->uuid, chrs[ci].uuid, 16);
            chr->props = chrs[ci].props;
            need += 2;
            if (chr->props & (ESP_GATT_CHAR_PROP_BIT_NOTIFY | ESP_GATT_CHAR_PROP_BIT_INDICATE)) {
                need += 1;
            }
            s_gchr_n++;
        }
        svc->num_handles = need;
        s_gsvc_n++;
    }
    if (ci != nchr) {
        s_gsvc_n = 0;
        s_gchr_n = 0;
        s_gatts_building = 0;
        s_gatts_op = 0;
        return BW_EINVAL;
    }
    /* The first create submission runs right here on the Go task; the
     * ESP_GATTS_CREATE_EVT on the BTC task drives the rest of the chain. */
    s_gs_cur_svc = 0;
    s_gs_cur_chr = -1;
    s_gs_ctrl.auto_rsp = ESP_GATT_RSP_BY_APP;
    bw_gatts_create_next_svc();
    return 0;
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
 * under the ctrl ring's critical section against the read the BTC task may
 * be answering from the same cache right now. */
int32_t llgo_ble_gatts_value(uint16_t handle, const uint8_t *data, int32_t len)
{
    if (!s_inited) {
        return BW_EINVAL;
    }
    if (handle == 0 || len < 0 || (len > 0 && data == NULL)) {
        return BW_EINVAL;
    }
    if (len > BW_DATA_MAX) {
        return BW_EMSGSIZE;
    }
    int idx = bw_gchr_by_handle(handle);
    if (idx < 0) {
        return BW_ENOENT;
    }
    BW_CTRL_LOCK();
    if (len > 0) {
        memcpy(s_gchrs[idx].value, data, (size_t)len);
    }
    s_gchrs[idx].value_len = (uint16_t)len;
    BW_CTRL_UNLOCK();
    return 0;
}

int32_t llgo_ble_gatts_notify(uint32_t conn_id, uint16_t handle, const uint8_t *data,
                              int32_t len, int32_t indicate)
{
    if (!s_inited || !s_synced) {
        return BW_EINVAL;
    }
    if (handle == 0 || len < 0 || (len > 0 && data == NULL)) {
        return BW_EINVAL;
    }
    if (len > BW_DATA_MAX) {
        return BW_EMSGSIZE;
    }
    if (bw_gchr_by_handle(handle) < 0) {
        return BW_ENOENT;
    }
    int raw = bw_pconn_find(conn_id);
    if (raw < 0) {
        return BW_ENOENT;
    }
    /* The value pointer goes straight through: the API deep-copies it into
     * the BTC message before returning. A subscribed CCCD is the peer's own
     * gate; this call is the wire-level send either way. */
    esp_err_t e = esp_ble_gatts_send_indicate(s_gatts_if, (uint16_t)raw, handle,
                                              (uint16_t)len, (uint8_t *)data,
                                              indicate ? true : false);
    return (int32_t)e;
}

int32_t llgo_ble_gatts_close(uint32_t conn_id)
{
    if (!s_inited || !s_synced) {
        return BW_EINVAL;
    }
    int raw = bw_pconn_find(conn_id);
    if (raw < 0) {
        bw_ctrl_push(EV_ERROR, 0, conn_id, BW_ENOENT);
        return 0;
    }
    esp_err_t e = esp_ble_gatts_close(s_gatts_if, (uint16_t)raw);
    if (e != ESP_OK) {
        bw_ctrl_push(EV_ERROR, 0, conn_id, e);
    }
    return 0;
}

/* The pairing posture. Every parameter is set explicitly - Bluedroid keeps
 * no usable defaults, and a pairing left without them reports success while
 * distributing no key (AUTH_CMPL key_present = 0), so nothing is stored and
 * a bonded peer has to re-pair on every reconnect. MAX_KEY_SIZE in particular:
 * unset, no key is distributed at all. */
int32_t llgo_ble_security(int32_t io_cap, int32_t mitm, int32_t bond, uint32_t passkey,
                          int32_t key_size)
{
    if (!s_inited) {
        return BW_EINVAL;
    }
    if (io_cap < 0 || io_cap > ESP_IO_CAP_KBDISP) {
        return BW_EINVAL;
    }
    uint8_t auth_req;
    if (mitm) {
        auth_req = bond ? ESP_LE_AUTH_REQ_SC_MITM_BOND : ESP_LE_AUTH_REQ_SC_MITM;
    } else {
        auth_req = bond ? ESP_LE_AUTH_REQ_SC_BOND : ESP_LE_AUTH_REQ_SC_ONLY;
    }
    uint8_t io = (uint8_t)io_cap;
    uint8_t ks = (uint8_t)key_size;
    uint8_t keys = ESP_BLE_ENC_KEY_MASK | ESP_BLE_ID_KEY_MASK;
    esp_err_t e = esp_ble_gap_set_security_param(ESP_BLE_SM_AUTHEN_REQ_MODE, &auth_req,
                                                 sizeof auth_req);
    if (e == ESP_OK) {
        e = esp_ble_gap_set_security_param(ESP_BLE_SM_IOCAP_MODE, &io, sizeof io);
    }
    if (e == ESP_OK) {
        e = esp_ble_gap_set_security_param(ESP_BLE_SM_MAX_KEY_SIZE, &ks, sizeof ks);
    }
    if (e == ESP_OK) {
        e = esp_ble_gap_set_security_param(ESP_BLE_SM_SET_INIT_KEY, &keys, sizeof keys);
    }
    if (e == ESP_OK) {
        e = esp_ble_gap_set_security_param(ESP_BLE_SM_SET_RSP_KEY, &keys, sizeof keys);
    }
    if (e == ESP_OK && passkey != 0) {
        uint32_t pk = passkey;
        e = esp_ble_gap_set_security_param(ESP_BLE_SM_SET_STATIC_PASSKEY, &pk, sizeof pk);
    }
    return (int32_t)e;
}

#else /* BT host configurations this glue does not implement */

/* IDF build without the Bluedroid host: every entry point exists (the Go side
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

#endif /* BT host configuration */

#else /* !ESP_PLATFORM */

/* Template guard: outside an ESP-IDF build this file compiles to an
 * otherwise empty translation unit. */
typedef int llgo_ble_glue_not_in_idf_build;

#endif /* ESP_PLATFORM */
