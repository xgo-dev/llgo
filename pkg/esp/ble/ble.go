// Package ble binds llgo-compiled Go code to the ESP-IDF NimBLE host as a BLE
// central: bring-up, scanning, connecting, disconnecting, the GATT client
// (discovery, read, write, notifications), and the stack's asynchronous
// results as a fan-out event mailbox.
//
// # Layout
//
// Every package under pkg/esp follows the same template (glue lives in a
// subdirectory because the go tool rejects C sources inside a Go package):
//
//	ble.go           public API, no build tag
//	ble_idf.go       //go:build llgo.idf - bodyless //go:linkname externs
//	ble_stub.go      //go:build !llgo.idf - host-build stubs (go vet/gopls)
//	glue/ble_glue_nimble.c  NimBLE backend glue; a Bluedroid sibling
//	                    (ble_glue_bluedroid.c) implements the same C ABI
//	                 ESP_PLATFORM
//
// # Usage
//
//	err := ble.Init()                             // controller + host + sync
//	sub := ble.Subscribe(ble.EvAdv, ble.EvConnect, ble.EvDisconnect)
//	op, err := ble.StartScan(ble.ScanOptions{})
//	ev, err := sub.Next(8000)                     // EvAdv: the scan result
//	stopErr := ble.StopScan()
//	op, err = ble.Connect(peer, ble.ConnectOptions{TimeoutMs: 5000})
//	ev, err = sub.Next(5000)                      // EvConnect: ConnID or status
//	conn := ev.ConnID
//	svcs, err := ble.Services(conn)               // GATT discovery
//	chars, err := ble.Characteristics(conn, svcs[0].UUID)
//	v, err := ble.Read(conn, chars[0])
//	err = ble.Write(conn, chars[0], []byte("hi"))
//	nsub, err := ble.SubscribeChar(conn, chars[1]) // CCCD on
//	ev, err = nsub.Next(5000)                      // EvNotify: the payload
//	unsubErr := ble.Unsubscribe(nsub)             // CCCD off
//	disErr := ble.Disconnect(conn)
//	ev, err = sub.Next(5000)                      // EvDisconnect: reason
//
// Control calls never block: StartScan/Connect/Disconnect hand their command
// to the NimBLE host's own event queue and return an operation id (OpID), so
// the caller's goroutine keeps the llgo scheduler while the host task runs the
// GAP procedure. The outcome always arrives through the mailbox - a returned
// error means "the command could not even be submitted" (not ready, queue
// full, BLE not in this firmware), never a forged result.
//
// The GATT calls (Services, Characteristics, Read, Write, SubscribeChar,
// Unsubscribe, MTU) are the synchronous half of the same mechanism: each one
// submits its procedure to the host task and then waits, bounded by
// GattTimeoutMs, for the mailbox record that completes it. They are the shape
// a GATT client wants (a discovered list, a value, a status) and they keep the
// task discipline: no NimBLE procedure ever runs on the goroutine that asked
// for it.
//
// # Why a mailbox, not callbacks
//
// NimBLE runs its reset/sync/GAP callbacks on the host task, and the llgo
// fiber scheduler is single-task global state: an entry into Go from a second
// FreeRTOS task would corrupt it. The glue copies every result into three
// bounded C-side rings (control results in one, notifications in a second,
// scan results in a third, so neither a scan flood nor a notification storm
// can displace a connect/disconnect or a GATT result), and one poller
// goroutine drains them and fans them out to every subscription - a slow
// consumer loses its own events, never anyone else's. Overflow is reported
// once as EvOverflow carrying the dropped count.
//
// # GATT client
//
// One GATT procedure runs at a time (NimBLE's ATT bearer carries one
// outstanding request anyway, and this package's guard reports ErrBusy instead
// of letting two procedures overwrite each other's result):
//
//	Services           discover all primary services (16 records max - a peer
//	                   with more ends the discovery with BLE_HS_EMSGSIZE)
//	Characteristics    discover a service's characteristics (32 records max)
//	Read               ATT Read Request
//	Write              ATT Write Request or Write Command, chosen from the
//	                   characteristic's properties (WriteWithResponse and
//	                   WriteWithoutResponse pick one explicitly)
//	SubscribeChar      write the characteristic's client configuration (CCCD)
//	                   and start delivering its notifications
//	MTU                the negotiated ATT MTU
//
// Discovery results live in the glue's fixed tables, not in the mailbox: a
// table is deterministic (a record is either there or the discovery ends with
// a status saying why it is not), while a ring can drop. The mailbox carries
// what is unsolicited: notifications and indications arrive as EvNotify with
// the payload copied out of the ATT mbuf, the same three-ring discipline the
// scan path uses (a notification flood cannot displace a control result).
//
// A call that returns ErrTimeout leaves the stack's procedure running: NimBLE
// ends an unanswered one after its own 30 s unresponsive limit, which also
// terminates the connection, and until then the next GATT call is refused with
// BLE_HS_EALREADY (ErrStatus(StatusEALREADY)) rather than queued behind it.
//
// A characteristic's CCCD is written at value handle + 1, the layout the GATT
// specification mandates for a characteristic that notifies or indicates.
// Subscribing delivers through the ordinary fan-out: the returned
// *Subscription filters the machine to that connection and value handle, and a
// plain Subscribe(EvNotify) sees every notification of every characteristic.
//
// # Scheduling
//
// The poller drains a bounded batch (maxPollBatch) per pass, hands the
// scheduler away with runtime.Gosched, then rests the FreeRTOS task for
// exactly one tick (10 ms at the IDF's default 100 Hz) - one scheduler tick,
// not a fixed millisecond quantum that ages badly against the tick rate. A
// bounded wait (Subscription.Next) measures its deadline against the glue's
// millisecond clock instead of counting quanta, so a 300 ms bound means
// 300 ms, not "three quanta". No goroutine
// holds the scheduler across an unbounded C wait: the two C calls that do
// block - Init (controller bring-up) and Deinit (host stop) - are one-shot
// lifecycle calls, documented as such.
//
// Keep one BLE session on one goroutine: a C-level wait gives the llgo
// scheduler no yield point (the same architecture property the wifi, net and
// tls packages document).
//
// # Error model
//
// Failures are the family's shared *esperr.Error, in three explicitly
// separated code spaces (Code stays inspectable; errors.Is matches by code,
// which is exactly why they must not share one):
//
//	esp domain     raw esp_err, below 0x4000 - Init/Deinit and the
//	               firmware-without-BLE case (ESP_ERR_NOT_SUPPORTED, 0x106)
//	nimble domain  NimbleBase(0x8000) | status - every NimBLE host-core
//	               (BLE_HS_*), ATT (0x1xx), HCI (0x2xx) and L2CAP (0x3xx)
//	               status, as the stack reports them. Status() extracts the
//	               raw value, ErrStatus() rebuilds the error value.
//	Go domain      0x7Bxx - this package's own conditions: ErrTimeout (a
//	               bounded wait found nothing), ErrNotReady (Init not done or
//	               no host sync), ErrBusy (a GATT procedure is in flight),
//	               ErrBadUUID/ErrBadValue/ErrBadProperty (an argument this
//	               package refuses), ErrNotFound, ErrTooLong, ErrTooMany,
//	               ErrUnsupported.
//
// The offset matters: NimBLE's status 0x02 (BLE_HS_EALREADY) and an HCI error
// 0x02 are different conditions in different namespaces, and an esp_err 0x02
// is none of them. Tagging the nimble domain keeps errors.Is from mixing
// them, and keeps an esp_err comparison from ever matching a host status.
//
// # Scope
//
// Both BLE roles share this package. The central side is GAP scanning (legacy
// discovery), connect, cancel, disconnect and the GATT client; the peripheral
// side is advertising, the local GATT server (a registry of services and
// characteristics the glue builds once), the peer's writes and subscriptions
// as events, and server-side notify/indicate. Pairing parameters are
// configurable (SetSecurityParams) but no pairing state machine is exposed;
// no descriptor discovery, no long read/write - those are separate stages; the
// glue's mailbox has the shape they need (a control tier for results a state
// machine depends on, payload tiers with a bounded copy).
package ble

import "github.com/xgo-dev/llgo/pkg/esp/esperr"

// NimbleBase offsets a NimBLE host status (BLE_HS_*, ATT, HCI or L2CAP, all
// below 0x8000) into this package's code space. See the error model above.
const NimbleBase = 0x8000

// The Go-domain codes: conditions this package detects itself, below the
// esperr meta segment (0x7Cxx) and below the tag stubs use (0x7F00).
const (
	codeTimeout     = 0x7B01
	codeNotReady    = 0x7B02
	codeBadUUID     = 0x7B03
	codeBadValue    = 0x7B04
	codeBusy        = 0x7B05
	codeBadProperty = 0x7B06
	codeNotFound    = 0x7B07
	codeTooLong     = 0x7B08
	codeTooMany     = 0x7B09
	codeUnsupported = 0x7F00
)

// ErrTimeout reports a bounded wait that found nothing: ESP_ERR_TIMEOUT-style
// semantics in this package's own code space - Subscription.Next saw no event
// inside its bound, or Init saw no host sync.
var ErrTimeout = &esperr.Error{Code: codeTimeout, Name: "BLE_ERR_TIMEOUT"}

// ErrNotReady reports a control call made before Init completed (or after
// Deinit): the host stack has no sync yet, so no GAP procedure can start.
var ErrNotReady = &esperr.Error{Code: codeNotReady, Name: "BLE_ERR_NOT_READY"}

// ErrUnsupported reports that this firmware (or build) has no BLE at all: a
// host build, or an ESP-IDF firmware compiled without the NimBLE host. The
// glue reports ESP_ERR_NOT_SUPPORTED and the package renders it in this
// package's code space, where it can never be mistaken for a real status.
var ErrUnsupported = &esperr.Error{Code: codeUnsupported, Name: "BLE_ERR_UNSUPPORTED"}

// ErrBadUUID reports a UUID value this package cannot use: a text form that is
// neither four hex digits nor the 36-character 128-bit form, or a zero UUID
// where a service or characteristic UUID is required.
var ErrBadUUID = &esperr.Error{Code: codeBadUUID, Name: "BLE_ERR_BAD_UUID"}

// ErrBadValue reports a caller-side argument this package refuses before
// submitting anything: a zero value handle in a Characteristic, or a nil
// Subscription.
var ErrBadValue = &esperr.Error{Code: codeBadValue, Name: "BLE_ERR_BAD_VALUE"}

// ErrBusy reports that another GATT procedure is still in flight. The ATT
// bearer carries one outstanding request per connection, and this package's
// result slots are single, so a second procedure is refused instead of being
// queued behind the first.
var ErrBusy = &esperr.Error{Code: codeBusy, Name: "BLE_ERR_BUSY"}

// ErrBadProperty reports a characteristic whose properties do not allow the
// requested operation: write on a characteristic that declares neither write
// nor write-without-response, or a subscription on one that declares neither
// notify nor indicate.
var ErrBadProperty = &esperr.Error{Code: codeBadProperty, Name: "BLE_ERR_BAD_PROPERTY"}

// ErrNotFound reports a service that the last discovery did not report: a
// Characteristics call takes a UUID that Services returned for that same
// connection.
var ErrNotFound = &esperr.Error{Code: codeNotFound, Name: "BLE_ERR_NOT_FOUND"}

// ErrTooLong reports a payload larger than one event or one ATT write carries
// (MaxData bytes). The value is refused rather than truncated.
var ErrTooLong = &esperr.Error{Code: codeTooLong, Name: "BLE_ERR_TOO_LONG"}

// ErrTooMany reports a registry or argument count beyond what this package
// holds statically: more than MaxLocalServices services, MaxLocalChars
// characteristics, or an advertising payload that does not fit one legacy
// advertising packet even with the name moved to the scan response.
var ErrTooMany = &esperr.Error{Code: codeTooMany, Name: "BLE_ERR_TOO_MANY"}

// espErr wraps a raw esp_err code (0 = nil) - the esp domain of this package.
func espErr(code int32) error {
	if code == 0 {
		return nil
	}
	if code == 0x106 { // ESP_ERR_NOT_SUPPORTED: no NimBLE host in the firmware
		return ErrUnsupported
	}
	return &esperr.Error{Code: code, Name: ErrName(code)}
}

// nimbleErr wraps a raw NimBLE host status (0 = nil) into the tagged nimble
// domain.
func nimbleErr(status int32) error {
	if status == 0 {
		return nil
	}
	code := NimbleBase + status
	return &esperr.Error{Code: code, Name: ErrName(code)}
}

// ErrStatus builds the error value one NimBLE host status maps to (nil for
// 0), so callers can compare with errors.Is against a specific status:
//
//	errors.Is(err, ble.ErrStatus(ble.StatusENOENT))
func ErrStatus(status int32) error { return nimbleErr(status) }

// Status extracts the raw NimBLE host status from an error of this package:
// 0 for nil, the untagged status for a nimble-domain error, and -1 for
// anything outside that domain (esp_err failures, this package's own codes, a
// foreign error).
func Status(err error) int32 {
	if err == nil {
		return 0
	}
	e, ok := err.(*esperr.Error)
	if !ok || e.Code < NimbleBase {
		return -1
	}
	return e.Code - NimbleBase
}

// NimBLE host-core statuses this package's control paths can report, named
// for readability. A status outside this list keeps its numeric value and gets
// a generic name from ErrName.
const (
	StatusEALREADY    = 0x02 // a procedure of that kind is already running
	StatusEINVAL      = 0x03 // bad parameter
	StatusEMSGSIZE    = 0x04 // a result is larger than one event carries
	StatusENOMEM      = 0x06 // out of memory
	StatusENOENT      = 0x05 // no such entry (also: stale ConnID)
	StatusENOTCONN    = 0x07 // not connected
	StatusETIMEOUT    = 0x0D // the controller gave up on the procedure
	StatusECONTROLLER = 0x0C // controller error
)

// Event kinds delivered by the glue's mailbox (see ble_glue_nimble.c).
const (
	// EvNone is the zero-value sentinel, never delivered: a zero Event and a
	// timed-out Next both carry it.
	EvNone = 0

	// EvSync reports that the host stack synced and GAP procedures may run.
	// Status carries the address-inference result (0 = fine).
	EvSync = 1

	// EvReset reports a host reset; the stack needs a new sync before any
	// further command. Status carries the reset reason.
	EvReset = 2

	// EvScanDone reports that the running discovery ended on its own (its
	// duration expired, or the controller stopped it). OpID is the scan
	// operation, Status the driver's reason. A discovery stopped through
	// StopScan or Cancel reports nothing - NimBLE stays silent when a scan is
	// stopped by request - so there is no EvScanDone for it.
	EvScanDone = 3

	// EvConnect reports the outcome of a Connect attempt: OpID is the
	// operation, Status the stack's result (0 = connected), and ConnID the
	// connection to use when Status is 0 (0 otherwise).
	EvConnect = 4

	// EvDisconnect reports that a connection ended. ConnID is the connection
	// that was reported connected, Status the HCI disconnection reason
	// (NimBLE reports it tagged with its HCI base, e.g. 0x213 for a
	// peer-initiated disconnect: 0x13 in the low byte).
	EvDisconnect = 5

	// EvAdv is one scan result: Addr/AddrType/Rssi and up to MaxAdvData bytes
	// of advertising payload in Adv[:AdvLen]. OpID is 0 - a scan result is not
	// tied to one operation, and several scans may be running in sequence.
	EvAdv = 6

	// EvError reports a control command the glue could not carry out at all:
	// Status is the NimBLE status that refused it, OpID the operation (0 for
	// Disconnect) and ConnID the connection (0 for scan operations). A
	// submitted command either produces its natural event or this one.
	EvError = 7

	// EvDiscDone reports that a service or characteristic discovery finished:
	// OpID is the operation, Status the stack's result (0 = the records are in
	// the glue's table, anything else the reason it stopped).
	EvDiscDone = 8

	// EvReadDone reports a read's result: OpID the operation, Status the ATT or
	// host status and, on success, the value in Data[:DataLen].
	EvReadDone = 9

	// EvWriteDone reports a write's result - a value write or the client
	// configuration write of SubscribeChar/Unsubscribe: OpID the operation,
	// Status the result. A write-without-response is never acknowledged on the
	// wire, so its Status reports the submission only.
	EvWriteDone = 10

	// EvNotify is one server-pushed notification or indication: ConnID the
	// connection, AttHandle the characteristic's value handle, DataLen/Data the
	// payload and Indication 1 for an indication (the host confirms it) or 0
	// for a notification.
	EvNotify = 11

	// EvMtu reports a negotiated ATT MTU: ConnID the connection, Status the MTU
	// value (see Event.MTU). The glue asks for the exchange right after a
	// connection comes up.
	EvMtu = 12

	// The peripheral-role kinds (13..18), delivered through the same mailbox.
	// EvPeriphConnect and EvPeriphDisconnect travel on the control tier - the
	// ring that a scan flood or a notification storm can never displace - and
	// EvCharWrite and EvSubscribed travel on the peripheral's own payload tier,
	// so an advertising flood on either side of the link cannot crowd them out
	// either (see the glue's fourth ring).

	// EvPeriphConnect reports that a central connected to this device's
	// advertising: ConnID the connection to serve (packed like a central-side
	// ConnID, from the same generation counter), Addr/AddrType the peer.
	EvPeriphConnect = 13

	// EvPeriphDisconnect reports that a served connection ended: ConnID the
	// connection, Status the HCI disconnection reason (render it with
	// DisconnectReasonName).
	EvPeriphDisconnect = 14

	// EvCharWrite reports that a peer wrote a characteristic's value: ConnID
	// the connection, AttHandle the value handle, DataLen/Data the bytes
	// written. The glue has already copied the bytes into the characteristic's
	// value cache (what a later read returns), so this event is the program's
	// notice - an echo, a command or a config write - not a data lifeline.
	EvCharWrite = 15

	// EvSubscribed reports a write to a characteristic's client configuration
	// (CCCD): ConnID the connection, AttHandle the value handle of the
	// characteristic the CCCD belongs to, Status the raw CCCD value - bit 0
	// set when notifications are on, bit 1 when indications are.
	EvSubscribed = 16

	// EvAdvDone reports the outcome of a StartAdvertising or StopAdvertising
	// operation: OpID the operation, Status the stack's result (0 = done).
	// The advertising payload itself keeps running silently; this is the
	// lifecycle event only.
	EvAdvDone = 17

	// EvServicesDone reports the outcome of CommitServices: OpID the
	// operation, Status the stack's result (0 = the attribute table is live
	// and the assigned handles are queryable with LocalHandle).
	EvServicesDone = 18

	// EvOverflow reports mailbox loss: Status is the number of records dropped
	// since the last report (a record that did not fit its ring, or a
	// notification payload larger than MaxData). Control records are dropped
	// only when the control ring is full, which takes a program that stops
	// consuming events.
	EvOverflow = -1
)

// EventKind identifies one mailbox event kind; it aliases int32 so the Ev*
// constants pass directly as Subscribe arguments.
type EventKind = int32

// OpID identifies one submitted operation (a scan, a connect attempt). Ids
// come from a monotonic counter and are never reused, so a late completion
// from an earlier operation can never be mistaken for the current one.
type OpID uint32

// ConnID identifies one connection. It packs the controller's HCI connection
// handle with a generation counter, because the controller reuses handles
// across connections: a ConnID from an earlier connection is refused
// (EvError with StatusENOENT) instead of being applied to whatever the handle
// became.
type ConnID uint32

// Limits and defaults of this package.
const (
	// MaxAdvData is the advertising payload a scan result can carry: the
	// legacy maximum, which is what the legacy discovery procedure reports.
	MaxAdvData = 31

	// DefaultScanIntervalMs and DefaultScanWindowMs are the scan timing
	// ScanOptions leaves at zero: a 50 % duty cycle at the controller's
	// conventional 100 ms interval.
	DefaultScanIntervalMs = 100
	DefaultScanWindowMs   = 50

	// DefaultConnectTimeoutMs bounds a connect attempt when
	// ConnectOptions.TimeoutMs is zero: the controller stops looking after
	// this long and reports StatusETIMEOUT.
	DefaultConnectTimeoutMs = 30000

	// InitWaitMs bounds Init's wait for the host sync.
	InitWaitMs = 5000

	// SubQueueLen is the per-subscription event buffer. The poller drops events
	// for a subscriber that has not drained it instead of blocking the fan-out
	// for everyone.
	SubQueueLen = 8

	// maxPollBatch bounds how many records the poller moves per pass, so a
	// scan flood cannot hold the scheduler: the batch ends with a Gosched and
	// one tick of rest.
	maxPollBatch = 32

	// MaxData is the largest ATT payload one mailbox record carries - a
	// notification or indication, or a read value - and the largest value
	// Write accepts.
	//
	// The bound is the goroutine stack, not the ATT MTU. Every record travels
	// through a channel and through the non-blocking receive select of
	// Subscription.Next, and those frames hold several copies of it; on this
	// target goroutines run on 2 KiB fiber stacks (targets/esp32s31-idf.json
	// default-stack-size), so a record much larger than this one overflows the
	// stack instead of being delivered - measured under esp-emu: a 316-byte
	// record (MaxData 253) aborted the firmware during Init, the 188-byte one
	// this constant gives runs the whole GATT acceptance. A program that builds
	// with a larger goroutine stack (llgo build -goroutine-stack-size) can
	// raise MaxData and the glue's BW_DATA_MAX together.
	//
	// What happens to a payload that does not fit is uniform and never silent:
	// a notification or indication longer than MaxData is dropped whole by the
	// glue (never truncated) and counted in the mailbox's overflow report, and
	// a read that returns more reports BLE_HS_EMSGSIZE (ErrStatus(StatusEMSGSIZE)). At
	// the ATT MTU of 256 the glue negotiates, a peer may push 253 bytes, so a
	// talkative characteristic with large values needs a larger stack and a
	// larger MaxData.
	MaxData = 128

	// MaxServices and MaxCharacteristics bound the glue's discovery tables: a
	// service or characteristic discovery stops with BLE_HS_EMSGSIZE when the
	// peer has more, instead of silently dropping records.
	MaxServices        = 16
	MaxCharacteristics = 32

	// MaxLocalServices and MaxLocalChars bound the local GATT server's
	// registry - the tables the glue holds statically (and the Bluedroid
	// creation state machine walks). A registry beyond them is refused with
	// ErrTooMany before anything is submitted.
	MaxLocalServices = 4
	MaxLocalChars    = 8

	// DefaultAdvIntervalMs is the advertising interval AdvParams leaves at
	// zero: 100 ms, the slow end of the conventional 20 ms..10 s range, fine
	// for a device a user or peer looks for once.
	DefaultAdvIntervalMs = 100

	// GattTimeoutMs bounds a GATT call's wait for its completion record:
	// NimBLE's own unresponsive-procedure timeout, so a peer that stops
	// answering normally ends in the stack's ETIMEOUT status (and the stack
	// tearing the connection down) rather than in a Go-side timeout.
	GattTimeoutMs = 30000
)

// Peer is the advertiser to connect to: the address a scan result reported
// (Addr in the stack's little-endian order - Addr[0] is the least significant
// octet, which is what AddrString renders last) and its address type.
type Peer struct {
	Addr     [6]uint8
	AddrType uint8
}

// MaxBonds is the number of stored bonds BondedPeers can report at once.
const MaxBonds = 8

// BondedPeers lists the peers whose pairing keys this node has stored. A
// bonded peer does not need to be rediscovered by scanning: it either waits
// for the central to connect (directed advertising, or no advertising at
// all) or is already accepting - Connect straight to the address returned
// here. Reconnecting a bond also skips the pairing procedure, so the link is
// encrypted with the stored key.
//
// A node with no stored bonds returns an empty slice.
func BondedPeers() ([]Peer, error) {
	if !Ready() {
		return nil, ErrNotReady
	}
	buf := make([]uint8, MaxBonds*7)
	n := bondedPeers(&buf[0], MaxBonds)
	if n <= 0 {
		return nil, nil
	}
	if n > MaxBonds {
		n = MaxBonds
	}
	peers := make([]Peer, 0, n)
	for i := int32(0); i < n; i++ {
		var pr Peer
		copy(pr.Addr[:], buf[i*7:i*7+6])
		pr.AddrType = buf[i*7+6]
		peers = append(peers, pr)
	}
	return peers, nil
}

// ScanOptions configures StartScan. The zero value is a usable configuration:
// active scanning, 100 ms interval / 50 ms window, duplicate filtering on, no
// duration bound (the scan runs until StopScan).
type ScanOptions struct {
	// IntervalMs is the scan interval (time between scan windows).
	IntervalMs int32

	// WindowMs is the scan window inside each interval; a value above
	// IntervalMs is clamped to it (a window larger than the interval has no
	// meaning).
	WindowMs int32

	// Passive selects a passive scan: no scan requests are sent, so a peer's
	// scan-response payload (and its complete local name, when the advertiser
	// put the name there) never arrives.
	Passive bool

	// FilterDuplicates asks the controller to report each advertiser once.
	FilterDuplicates bool

	// DurationMs bounds the scan on the controller side; 0 means "until
	// StopScan". Expiry is reported as EvScanDone.
	DurationMs int32
}

// ConnectOptions bounds a connect attempt.
type ConnectOptions struct {
	// TimeoutMs is how long the controller keeps trying; 0 means
	// DefaultConnectTimeoutMs. Connect has no unbounded form on purpose: an
	// attempt that can never finish leaves the controller busy forever. The
	// attempt's end - success, a peer refusal, or the controller giving up -
	// arrives as EvConnect with that OpID; a non-zero Status means it did not
	// connect.
	TimeoutMs int32
}

// UUID is a GATT UUID: a 16-bit Bluetooth SIG value or a 128-bit vendor value.
// UUID is comparable, so == is the equality test:
//
//	if chr.UUID == ble.UUID16(0x2a00) { ... }
//
// A 128-bit UUID whose first 32 bits are zero and whose remaining 96 bits are
// the Bluetooth base UUID is the same UUID as its 16-bit alias (Core Vol 3,
// Part B 2.5.1), and this type holds it as that alias: ParseUUID of
// "0000180d-0000-1000-8000-00805f9b34fb" yields UUID16(0x180d) and String
// prints "180d". Keeping one value for both spellings is what makes a
// discovery result compare equal however the peer wrote the UUID on the wire.
type UUID struct {
	// v holds the UUID little-endian, the byte order the ATT PDU and the stack
	// use; for a 16-bit UUID only v[0:2] are meaningful.
	v [16]uint8
	// n is the width in bytes: 2 or 16, and 0 for the zero UUID.
	n uint8
}

// baseTailLE is the Bluetooth base UUID's lower 96 bits in the little-endian
// order UUID.v uses: text form "0000xxxx-0000-1000-8000-00805f9b34fb" read
// right to left.
var baseTailLE = [12]uint8{0xFB, 0x34, 0x9B, 0x5F, 0x80, 0x00, 0x00, 0x80, 0x00, 0x10, 0x00, 0x00}

// UUID16 returns the 16-bit UUID v, a Bluetooth SIG assigned value.
func UUID16(v uint16) UUID { return UUID{v: [16]uint8{uint8(v), uint8(v >> 8)}, n: 2} }

// UUID128 returns a 128-bit UUID from b in canonical text order: b[0] is the
// most significant octet, the first byte of the printed form. A Bluetooth
// base-form value collapses to its 16-bit alias - the two are one UUID.
func UUID128(b [16]uint8) UUID {
	var u UUID
	for i := 0; i < 16; i++ {
		u.v[15-i] = b[i]
	}
	u.n = 16
	return u.alias()
}

// alias collapses a Bluetooth base-form 128-bit UUID to its 16-bit alias. A
// 32-bit alias (upper 96 bits zero, the value in the top 32 bits) whose upper
// half is not zero stays 128-bit: this type has no 32-bit form.
func (u UUID) alias() UUID {
	if u.n != 16 {
		return u
	}
	for i := 0; i < 12; i++ {
		if u.v[i] != baseTailLE[i] {
			return u
		}
	}
	if u.v[14] != 0 || u.v[15] != 0 {
		return u
	}
	return UUID16(uint16(u.v[12]) | uint16(u.v[13])<<8)
}

// Len reports the UUID's width in bytes: 2 for a 16-bit UUID, 16 for a 128-bit
// one and 0 for the zero UUID.
func (u UUID) Len() int { return int(u.n) }

// IsZero reports the zero UUID, which no attribute carries: an unset or
// rejected UUID value reads as zero.
func (u UUID) IsZero() bool { return u.n == 0 }

// String renders the UUID in canonical text form: four lowercase hex digits for
// a 16-bit UUID ("180d"), the dashed 36-character form for a 128-bit one. The
// zero UUID renders as the empty string, and ParseUUID(String()) returns the
// same value.
func (u UUID) String() string {
	switch u.n {
	case 2:
		return hex4(uint16(u.v[0]) | uint16(u.v[1])<<8)
	case 16:
	default:
		return ""
	}
	var out [36]byte
	j := 0
	for i := 0; i < 16; i++ {
		if i == 4 || i == 6 || i == 8 || i == 10 {
			out[j] = '-'
			j++
		}
		b := u.v[15-i]
		out[j] = hexDigits[b>>4]
		out[j+1] = hexDigits[b&0xF]
		j += 2
	}
	return string(out[:])
}

// ParseUUID parses a UUID in either canonical text form: four hex digits
// ("180d", also accepted with a "0x" prefix) for a 16-bit UUID, or the 36-
// character dashed form ("6c6c676f-1111-2222-3344-556677889900", also accepted
// undashed) for a 128-bit one. Case does not matter, and a Bluetooth base-form
// 128-bit UUID parses to its 16-bit alias. Anything else is ErrBadUUID.
func ParseUUID(s string) (UUID, error) {
	if len(s) >= 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		s = s[2:]
	}
	if len(s) == 4 {
		v, ok := hex16(s)
		if !ok {
			return UUID{}, ErrBadUUID
		}
		return UUID16(v), nil
	}
	dashed := len(s) == 36
	if len(s) != 32 && !dashed {
		return UUID{}, ErrBadUUID
	}
	var b [16]uint8
	i := 0
	for j := 0; j < 16; j++ {
		if dashed && (j == 4 || j == 6 || j == 8 || j == 10) {
			if s[i] != '-' {
				return UUID{}, ErrBadUUID
			}
			i++
		}
		hi, ok1 := hexDigit(s[i])
		lo, ok2 := hexDigit(s[i+1])
		if !ok1 || !ok2 {
			return UUID{}, ErrBadUUID
		}
		b[j] = hi<<4 | lo
		i += 2
	}
	return UUID128(b), nil
}

// uuidFromWire builds a UUID from the glue's record form: n bytes (2 or 16)
// little-endian, as the ATT PDU carried them.
func uuidFromWire(b *[16]uint8, n uint8) UUID {
	switch n {
	case 2:
		return UUID16(uint16(b[0]) | uint16(b[1])<<8)
	case 16:
		return UUID{v: *b, n: 16}.alias()
	}
	return UUID{}
}

// Service is one GATT service a discovery reported. Services are primary
// services: the discovery procedure this package runs does not report
// secondary ones.
type Service struct {
	// UUID is the service's UUID.
	UUID UUID
	// Handle is the service declaration's handle, End the last handle the
	// service owns. Characteristics searches that range.
	Handle uint16
	End    uint16
}

// Characteristic is one GATT characteristic a discovery reported.
type Characteristic struct {
	UUID UUID
	// DeclHandle is the characteristic declaration's handle, ValueHandle the
	// handle of its value - what Read and Write address, and what the
	// notifications of SubscribeChar carry.
	DeclHandle  uint16
	ValueHandle uint16
	// Properties is the ATT property bitmask (the Prop* constants).
	Properties uint8
}

// The ATT characteristic property bits, as the peer declares them.
const (
	PropBroadcast  = 0x01 // the value can be broadcast in an advertisement
	PropRead       = 0x02 // readable with a read request
	PropWriteNoRsp = 0x04 // writable with a write command (unacknowledged)
	PropWrite      = 0x08 // writable with a write request (acknowledged)
	PropNotify     = 0x10 // pushes notifications
	PropIndicate   = 0x20 // pushes indications (each one acknowledged)
	PropAuthSigned = 0x40 // accepts authenticated signed writes
	PropExtProps   = 0x80 // has extended properties
)

// CanRead reports whether the peer declares the read property.
func (c Characteristic) CanRead() bool { return c.Properties&PropRead != 0 }

// CanWrite reports whether the peer declares write or write-without-response.
func (c Characteristic) CanWrite() bool { return c.Properties&(PropWrite|PropWriteNoRsp) != 0 }

// CanNotify reports whether the peer declares notify or indicate.
func (c Characteristic) CanNotify() bool { return c.Properties&(PropNotify|PropIndicate) != 0 }

// Event is one mailbox record. The fields are meaningful per kind (see the
// Ev* constants); everything a kind does not use is zero.
type Event struct {
	Kind     int32
	OpID     OpID
	ConnID   ConnID
	Status   int32
	Addr     [6]uint8
	AddrType uint8
	Rssi     int8
	AdvLen   uint8
	Adv      [MaxAdvData]uint8

	// AttHandle is the characteristic value handle an EvNotify came from (0 for
	// every other kind).
	AttHandle uint16

	// DataLen is the payload length this event carries and Data the payload: a
	// notification or indication (EvNotify) or a read value (EvReadDone). The
	// glue copies it out of the ATT mbuf, so it is detached from the stack; it
	// is never longer than MaxData.
	DataLen uint8

	// Indication reports an indication rather than a notification (EvNotify).
	Indication uint8

	Data [MaxData]uint8
}

// AdvData returns the advertising payload this event carries, as a slice over
// the event's own array: the bytes were copied out of the HCI buffer in the
// glue, so the payload is already detached from the stack and stays valid for
// as long as the Event does.
func (e *Event) AdvData() []uint8 { return e.Adv[:e.AdvLen] }

// Peer returns the advertiser address as a connectable Peer.
func (e *Event) Peer() Peer { return Peer{Addr: e.Addr, AddrType: e.AddrType} }

// DataBytes returns the payload this event carries, as a slice over the event's
// own array (see DataLen): a notification or indication for EvNotify, the value
// for EvReadDone.
func (e *Event) DataBytes() []uint8 { return e.Data[:e.DataLen] }

// MTU returns the ATT MTU an EvMtu event reports.
func (e *Event) MTU() int { return int(e.Status) }

// Subscription receives a copy of every mailbox event whose kind it asked for
// (EvOverflow goes to everyone). Create one with Subscribe, consume it with
// Next, retire it with Close. A Subscription is single-consumer: exactly one
// goroutine calls Next on it.
//
// A characteristic subscription (SubscribeChar) is the same type with a
// narrower filter: it delivers that characteristic's notifications only, so a
// program that watches one notify characteristic does not have to sort the
// mailbox itself.
type Subscription struct {
	kinds []EventKind
	ch    chan Event
	// notif is non-nil for a characteristic subscription: the fan-out then
	// delivers only EvNotify records of that connection and value handle.
	notif *notifTarget
}

// notifTarget is the filter of a characteristic subscription.
type notifTarget struct {
	conn   ConnID
	handle uint16
}

// subscribers is the poller's fan-out list. The llgo scheduler is cooperative
// (all goroutines multiplex one FreeRTOS task and only switch at blocking
// calls), so plain slices are race-free here as long as no function yields
// between reading a slice and acting on it - which Subscribe, Close and
// dispatch all honor.
var subscribers []*Subscription

// pollerStarted gates the lazy start of the single poller goroutine.
var pollerStarted bool

// opSeq is the monotonic operation-id source. Never 0: an Event with OpID 0
// means "no operation" (a scan result, a disconnection).
var opSeq uint32

// Subscribe starts the package's single poller goroutine (once, lazily) and
// returns a subscription that will receive a copy of every subsequent mailbox
// event matching one of kinds. An empty kinds list matches every kind. Events
// already sitting in the mailbox are not retroactively delivered - but they do
// wait there (the C rings are not drained until the poller starts), so
// subscribing before an operation is enough; subscribe before Init if you need
// the sync event.
func Subscribe(kinds ...EventKind) *Subscription {
	s := &Subscription{kinds: kinds, ch: make(chan Event, SubQueueLen)}
	join(s)
	return s
}

// join adds s to the fan-out and starts the poller on first use.
func join(s *Subscription) {
	subscribers = append(subscribers, s)
	if !pollerStarted {
		pollerStarted = true
		go pollEvents()
	}
}

// Close unsubscribes: the poller stops delivering to s, and events already
// buffered stay readable through Next. Closing twice is harmless.
func (s *Subscription) Close() {
	for i, e := range subscribers {
		if e == s {
			last := len(subscribers) - 1
			subscribers[i] = subscribers[last]
			subscribers[last] = nil
			subscribers = subscribers[:last]
			return
		}
	}
}

// Next waits up to timeoutMs for the subscription's next event: (event, nil)
// when one arrived, (zero Event, ErrTimeout) when the bound passed with
// nothing. A non-positive timeout makes a single non-blocking check. The bound
// is measured against the glue's millisecond clock, so accuracy is one
// FreeRTOS tick (10 ms at the IDF default) rather than one whole quantum, and
// the wait never holds a C-level call: each iteration yields with
// runtime.Gosched, which also lets the poller run.
func (s *Subscription) Next(timeoutMs int32) (Event, error) {
	if ev, ok := s.pop(); ok {
		return ev, nil
	}
	if timeoutMs <= 0 {
		return Event{}, ErrTimeout
	}
	deadline := int32(nowMs()) + timeoutMs
	for {
		gosched()
		if ev, ok := s.pop(); ok {
			return ev, nil
		}
		if int32(nowMs())-deadline >= 0 {
			return Event{}, ErrTimeout
		}
	}
}

// pop takes one buffered event without blocking.
func (s *Subscription) pop() (Event, bool) {
	select {
	case ev := <-s.ch:
		return ev, true
	default:
		return Event{}, false
	}
}

// pollEvents is the package's single mailbox consumer: drain up to
// maxPollBatch records, fan them out, hand the rest of the scheduling slice to
// other goroutines (Gosched), then rest the FreeRTOS task for one real tick -
// which also feeds the idle tasks and the watchdog. The tick is a measured
// period (the glue reports portTICK_PERIOD_MS), not an assumed quantum.
func pollEvents() {
	for {
		for n := 0; n < maxPollBatch; n++ {
			var ev Event
			if poll(&ev) != 1 {
				break
			}
			dispatch(ev)
		}
		gosched()
		vTaskDelay(1) // one FreeRTOS tick: 10 ms at the IDF default 100 Hz
	}
}

// dispatch hands ev to every subscriber whose filter matches, without ever
// blocking: a subscriber that has not drained its buffer in time loses the
// event (documented at SubQueueLen). Iterating the subscribers slice directly
// is safe under the cooperative scheduler: Subscribe/Close only run when the
// poller is parked in a blocking call, never mid-loop.
func dispatch(ev Event) {
	for _, s := range subscribers {
		if s.delivers(&ev) {
			select {
			case s.ch <- ev:
			default:
			}
		}
	}
}

// delivers reports whether ev goes to s. A characteristic subscription takes
// that characteristic's notifications; a kind filter takes its kinds (an empty
// one takes every kind). The mailbox's global overflow report goes to everyone.
func (s *Subscription) delivers(ev *Event) bool {
	if ev.Kind == EvOverflow {
		return true
	}
	if s.notif != nil {
		return ev.Kind == EvNotify && ev.ConnID == s.notif.conn && ev.AttHandle == s.notif.handle
	}
	if len(s.kinds) == 0 {
		return true
	}
	for _, k := range s.kinds {
		if k == EventKind(ev.Kind) {
			return true
		}
	}
	return false
}

// Init brings the BLE stack up: NVS (for the bond store - never formatted),
// controller init + BLE enable + NimBLE host init (one nimble_port_init call),
// the host task, and the wait for the stack's sync event. It blocks for the
// duration of the controller bring-up (the same one-shot freeze wifi.Setup
// has) and returns ErrTimeout when no sync arrived within InitWaitMs, or the
// controller status when the sync's own address inference failed. A host reset
// while Init waits is reported as BLE_HS_ECONTROLLER: nothing usable came up.
func Init() error {
	sub := Subscribe(EvSync, EvReset)
	defer sub.Close()
	if err := espErr(initHost()); err != nil {
		return err
	}
	for {
		ev, err := sub.Next(InitWaitMs)
		if err != nil {
			return ErrTimeout // no sync: the controller or the host task is stuck
		}
		switch ev.Kind {
		case EvSync:
			if ev.Status != 0 {
				return nimbleErr(ev.Status)
			}
			return nil
		case EvReset:
			// The controller reset while the host was waiting for sync;
			// nothing usable came up.
			return nimbleErr(StatusECONTROLLER)
		}
	}
}

// Backend names the BT host stack this firmware runs, for logs and
// diagnostics: "nimble" for the NimBLE build, "none" when the build carries no
// BLE host. The glue selects the backend; a program's own code never has to.
func Backend() string {
	var buf [16]byte
	n := backendName(&buf[0], int32(len(buf)))
	if n <= 0 {
		return "unknown"
	}
	return string(buf[:n])
}

// Ready reports whether the host stack has synced and GAP procedures can run.
func Ready() bool { return ready() == 1 }

// Deinit stops the host task and deinitializes controller and host. It blocks
// until the host task has serviced the stop (that is how nimble_port_stop
// works: it must run on a task other than the host task). A second call - or
// any call outside a live stack - reports ESP_ERR_INVALID_STATE rather than
// pretending there was something to stop.
func Deinit() error { return espErr(deinitHost()) }

// PrintVersion prints the C side's view (free heap and the adapter's own
// address) to the console; the string never crosses into Go (integer-only
// boundary).
func PrintVersion() error { return espErr(printVersion()) }

// StartScan submits a discovery procedure and returns its OpID. Scan results
// arrive as EvAdv; the procedure's own end arrives as EvScanDone unless it is
// stopped by request. A command the glue could not carry out (a discovery
// already running, no host sync) is reported as EvError with that OpID rather
// than as a returned error - submission itself is asynchronous.
func StartScan(opts ScanOptions) (OpID, error) {
	if !Ready() {
		return 0, ErrNotReady
	}
	// The API's defaults belong here; the glue keeps a defensive clamp so a
	// hand-built command cannot send a window larger than its interval.
	itvl, window := opts.IntervalMs, opts.WindowMs
	if itvl <= 0 {
		itvl = DefaultScanIntervalMs
	}
	if window <= 0 {
		window = DefaultScanWindowMs
	}
	if window > itvl {
		window = itvl
	}
	op := nextOp()
	passive, filter := int32(0), int32(0)
	if opts.Passive {
		passive = 1
	}
	if opts.FilterDuplicates {
		filter = 1
	}
	if e := nimbleErr(scanStart(itvl, window, passive, filter, opts.DurationMs, uint32(op))); e != nil {
		return 0, e
	}
	return op, nil
}

// StopScan stops the running discovery. NimBLE reports nothing when a scan is
// stopped by request - there is no EvScanDone for it - so a successful
// StopScan is silent; a refusal (no scan running) arrives as EvError. The stop
// is queued like every other command, so a StartScan or Connect submitted right
// afterwards runs after it, never concurrently.
func StopScan() error {
	if !Ready() {
		return ErrNotReady
	}
	return nimbleErr(scanStop(uint32(nextOp())))
}

// Connect submits a connect attempt to peer and returns its OpID. The outcome
// arrives as EvConnect carrying that OpID: Status 0 means connected, and the
// event's ConnID is the connection to use from then on. A discovery running at
// the time is not stopped implicitly - call StopScan first (the command queue
// keeps the order).
func Connect(peer Peer, opts ConnectOptions) (OpID, error) {
	if !Ready() {
		return 0, ErrNotReady
	}
	timeout := opts.TimeoutMs
	if timeout <= 0 {
		timeout = DefaultConnectTimeoutMs
	}
	op := nextOp()
	addr := peer.Addr
	if e := nimbleErr(connect(&addr[0], int32(peer.AddrType), timeout, uint32(op))); e != nil {
		return 0, e
	}
	return op, nil
}

// Cancel abandons the operation op: a running scan or a connect attempt. The
// cancelled operation reports nothing further - a stopped discovery is silent
// in NimBLE, and a cancelled connect attempt's late completion has no
// operation to attribute it to - so a caller that cancels knows the outcome
// without waiting. A stale or unknown OpID arrives as EvError with
// StatusENOENT.
func Cancel(op OpID) error {
	if !Ready() {
		return ErrNotReady
	}
	return nimbleErr(cancel(uint32(op)))
}

// Disconnect terminates the connection id. The disconnection itself arrives as
// EvDisconnect (Status carries the HCI reason); a ConnID that no longer
// matches a live connection - a stale generation, an already-closed
// connection - arrives as EvError with StatusENOENT instead of being applied
// to whatever connection reuses that handle.
func Disconnect(id ConnID) error {
	if !Ready() {
		return ErrNotReady
	}
	return nimbleErr(disconnect(uint32(id)))
}

// ---- GATT client ----

// gattBusy serializes this package's GATT procedures. The llgo scheduler is
// cooperative, so a plain flag is race-free: a caller only observes it between
// its own scheduling points, and the one goroutine that set it is parked in the
// mailbox wait. NimBLE's ATT bearer carries one outstanding request per
// connection anyway, and the glue's result slots are single - a second
// procedure would overwrite the first one's discovery table or read buffer.
var gattBusy bool

// gattBegin takes the GATT guard: ErrNotReady outside a live stack, ErrBusy
// while another procedure is in flight.
func gattBegin() error {
	if !Ready() {
		return ErrNotReady
	}
	if gattBusy {
		return ErrBusy
	}
	gattBusy = true
	return nil
}

// gattEnd releases the guard.
func gattEnd() { gattBusy = false }

// waitOp waits up to GattTimeoutMs for the mailbox record that completes op,
// ignoring records that belong to something else (another kind, another
// operation). What it ignores is this waiter's own fan-out copy: dispatch
// hands every subscriber an independent copy, so a record dropped here is
// lost to no one else - a global report like EvOverflow still reaches every
// other subscriber independently. It reports ErrTimeout when the bound passes.
func waitOp(sub *Subscription, op OpID) (Event, error) {
	deadline := int32(nowMs()) + GattTimeoutMs
	for {
		left := deadline - int32(nowMs())
		if left <= 0 {
			return Event{}, ErrTimeout
		}
		ev, err := sub.Next(left)
		if err != nil {
			return Event{}, err
		}
		if ev.OpID == op {
			return ev, nil
		}
	}
}

// Services discovers the peer's primary services and returns them. Each call
// re-runs the discovery, so the result is a snapshot of the peer's attribute
// table at that moment; MaxServices records are the most the glue keeps (a peer
// with more ends the discovery with BLE_HS_EMSGSIZE rather than losing records
// silently).
func Services(conn ConnID) ([]Service, error) {
	if err := gattBegin(); err != nil {
		return nil, err
	}
	defer gattEnd()
	op := nextOp()
	sub := Subscribe(EvDiscDone, EvError)
	defer sub.Close()
	if e := nimbleErr(discSvcs(uint32(conn), uint32(op))); e != nil {
		return nil, e
	}
	ev, err := waitOp(sub, op)
	if err != nil {
		return nil, err
	}
	if ev.Status != 0 {
		return nil, nimbleErr(ev.Status)
	}
	n := int(svcCount(uint32(conn)))
	out := make([]Service, 0, n)
	for i := 0; i < n; i++ {
		var r svcRec
		if svcAt(uint32(conn), int32(i), &r) != 1 {
			break
		}
		out = append(out, Service{UUID: uuidFromWire(&r.UUID, r.UUIDLen), Handle: r.Start, End: r.End})
	}
	return out, nil
}

// Characteristics discovers the characteristics of the service svc, which must
// be one that Services reported for this connection (ErrNotFound otherwise -
// the discovery ranges over the service's handle range, so that range has to
// come from a real discovery). MaxCharacteristics records are the most the glue
// keeps; a peer with more ends the discovery with BLE_HS_EMSGSIZE.
func Characteristics(conn ConnID, svc UUID) ([]Characteristic, error) {
	if svc.IsZero() {
		return nil, ErrBadUUID
	}
	if err := gattBegin(); err != nil {
		return nil, err
	}
	defer gattEnd()
	var rng Service
	found := false
	n := int(svcCount(uint32(conn)))
	for i := 0; i < n; i++ {
		var r svcRec
		if svcAt(uint32(conn), int32(i), &r) != 1 {
			break
		}
		if uuidFromWire(&r.UUID, r.UUIDLen) == svc {
			rng, found = Service{UUID: svc, Handle: r.Start, End: r.End}, true
			break
		}
	}
	if !found {
		return nil, ErrNotFound
	}
	op := nextOp()
	sub := Subscribe(EvDiscDone, EvError)
	defer sub.Close()
	if e := nimbleErr(discChrs(uint32(conn), rng.Handle, rng.End, uint32(op))); e != nil {
		return nil, e
	}
	ev, err := waitOp(sub, op)
	if err != nil {
		return nil, err
	}
	if ev.Status != 0 {
		return nil, nimbleErr(ev.Status)
	}
	m := int(chrCount(uint32(conn)))
	out := make([]Characteristic, 0, m)
	for i := 0; i < m; i++ {
		var r chrRec
		if chrAt(uint32(conn), int32(i), &r) != 1 {
			break
		}
		out = append(out, Characteristic{
			UUID:        uuidFromWire(&r.UUID, r.UUIDLen),
			DeclHandle:  r.Def,
			ValueHandle: r.Value,
			Properties:  r.Properties,
		})
	}
	return out, nil
}

// Read reads chr's value with an ATT Read Request. The result is at most
// MaxData bytes: a read response carries ATT MTU - 1 bytes, so a longer value
// is truncated by the peer (a long read is a later stage).
func Read(conn ConnID, chr Characteristic) ([]byte, error) {
	if chr.ValueHandle == 0 {
		return nil, ErrBadValue
	}
	if err := gattBegin(); err != nil {
		return nil, err
	}
	defer gattEnd()
	op := nextOp()
	sub := Subscribe(EvReadDone, EvError)
	defer sub.Close()
	if e := nimbleErr(readAttr(uint32(conn), chr.ValueHandle, uint32(op))); e != nil {
		return nil, e
	}
	ev, err := waitOp(sub, op)
	if err != nil {
		return nil, err
	}
	if ev.Status != 0 {
		return nil, nimbleErr(ev.Status)
	}
	out := make([]byte, ev.DataLen)
	copy(out, ev.Data[:ev.DataLen])
	return out, nil
}

// Write writes data to chr with the ATT procedure the characteristic's
// properties allow: write-with-response when the peer declares write, else
// write-without-response. The acknowledged form wins when a peer declares both,
// because it is the only one that reports a peer-side failure. A characteristic
// that declares neither is ErrBadProperty; more than MaxData bytes is
// ErrTooLong.
func Write(conn ConnID, chr Characteristic, data []byte) error {
	if chr.ValueHandle == 0 {
		return ErrBadValue
	}
	switch {
	case chr.Properties&PropWrite != 0:
		return writeValue(conn, chr.ValueHandle, data, writeWithResponse)
	case chr.Properties&PropWriteNoRsp != 0:
		return writeValue(conn, chr.ValueHandle, data, writeWithoutResponse)
	}
	return ErrBadProperty
}

// WriteWithResponse writes data with an ATT Write Request: the peer
// acknowledges it, so a failure (an ATT error response, a lost link) is
// reported here. It does not consult the characteristic's properties.
func WriteWithResponse(conn ConnID, chr Characteristic, data []byte) error {
	return writeValue(conn, chr.ValueHandle, data, writeWithResponse)
}

// WriteWithoutResponse writes data with an ATT Write Command: the peer does not
// acknowledge it, so the returned error can only report a submission failure
// (queue full, no such connection). It does not consult the characteristic's
// properties; a peer that cannot take the write answers with nothing, and the
// next operation that needs a response is where the mismatch surfaces.
func WriteWithoutResponse(conn ConnID, chr Characteristic, data []byte) error {
	return writeValue(conn, chr.ValueHandle, data, writeWithoutResponse)
}

// writeMode selects the ATT write procedure the glue submits.
type writeMode int32

const (
	writeWithResponse    writeMode = 1 // ATT Write Request
	writeWithoutResponse writeMode = 2 // ATT Write Command
)

// writeValue submits one write and waits for its completion record.
func writeValue(conn ConnID, handle uint16, data []byte, mode writeMode) error {
	if handle == 0 {
		return ErrBadValue
	}
	if len(data) > MaxData {
		return ErrTooLong
	}
	if err := gattBegin(); err != nil {
		return err
	}
	defer gattEnd()
	var p *uint8
	if len(data) > 0 {
		p = &data[0]
	}
	op := nextOp()
	sub := Subscribe(EvWriteDone, EvError)
	defer sub.Close()
	if e := nimbleErr(writeAttr(uint32(conn), handle, p, int32(len(data)), int32(mode), uint32(op))); e != nil {
		return e
	}
	ev, err := waitOp(sub, op)
	if err != nil {
		return err
	}
	return nimbleErr(ev.Status)
}

// SubscribeChar switches a characteristic's notifications (or indications, when
// that is all it declares) on - it writes the client characteristic
// configuration descriptor at value handle + 1 - and returns a subscription
// that receives them. Consume it with Next, retire it with Unsubscribe.
//
// Delivery is the ordinary mailbox fan-out: the subscription is filtered to
// this connection and value handle, and a plain Subscribe(EvNotify) sees every
// notification of every characteristic. Notifications are not queued by the
// peer: one that arrives while this subscription's buffer is full is dropped
// for this subscriber (and reported in the mailbox's overflow count).
func SubscribeChar(conn ConnID, chr Characteristic) (*Subscription, error) {
	if chr.ValueHandle == 0 {
		return nil, ErrBadValue
	}
	var cccd uint16
	switch {
	case chr.Properties&PropNotify != 0:
		cccd = 1
	case chr.Properties&PropIndicate != 0:
		cccd = 2
	default:
		return nil, ErrBadProperty
	}
	// Register before the CCCD write: the peer may push a notification as soon
	// as it sees the write, and a subscription created afterwards would lose
	// that first packet. A refused write retires it again.
	s := &Subscription{
		ch:    make(chan Event, SubQueueLen),
		notif: &notifTarget{conn: conn, handle: chr.ValueHandle},
	}
	join(s)
	value := [2]uint8{uint8(cccd), uint8(cccd >> 8)}
	if err := writeValue(conn, chr.ValueHandle+1, value[:], writeWithResponse); err != nil {
		s.Close()
		return nil, err
	}
	return s, nil
}

// Unsubscribe retires a subscription. A characteristic subscription has its
// client configuration switched off at the peer first (one write, so a peer
// that refuses or is gone reports it), then leaves the fan-out; a plain mailbox
// subscription is only removed from the fan-out, exactly like Close. Events
// already buffered stay readable through Next either way.
func Unsubscribe(s *Subscription) error {
	if s == nil {
		return ErrBadValue
	}
	var err error
	if s.notif != nil {
		var off [2]uint8 // 0x0000: notifications and indications off
		err = writeValue(s.notif.conn, s.notif.handle+1, off[:], writeWithResponse)
	}
	s.Close()
	return err
}

// MTU reports the ATT MTU in effect on the connection: what the exchange the
// glue requests right after a connection comes up negotiated (EvMtu reports the
// same value as it happens). Before that exchange completes it is the ATT
// default, 23, which is what a notification payload of 20 bytes implies.
func MTU(conn ConnID) (int, error) {
	if !Ready() {
		return 0, ErrNotReady
	}
	var mtu int32
	if e := nimbleErr(mtuGet(uint32(conn), &mtu)); e != nil {
		return 0, e
	}
	return int(mtu), nil
}

// ---- peripheral: advertising, the local GATT server, security ----
//
// The same mailbox and the same conventions as the central half, mirrored:
// StartAdvertising is a control call whose outcome arrives as EvAdvDone,
// CommitServices is a synchronous procedure that waits for its completion
// record, and what the peer does to this device's attributes arrives as
// events. A characteristic's value is a fixed slot in the glue (one value
// cache per characteristic): WriteCharValue fills it from Go, a peer's write
// overwrites it (and reports EvCharWrite), and a peer's read is answered
// straight from it by the glue - the GATT server never calls back into Go
// for a value, because a read must be answered synchronously on the stack's
// own task, and the llgo scheduler cannot be entered from there.

// AdvParams configures StartAdvertising. The zero value is a usable,
// connectable advertisement at DefaultAdvIntervalMs with no name.
type AdvParams struct {
	// Name is the local name the advertisement carries, as a Complete Local
	// Name AD structure when it fits the advertising payload, otherwise (or
	// when AdvData already fills it) as a Shortened one in the scan response.
	// An empty name adds nothing.
	Name string

	// IntervalMs is the advertising interval; 0 means DefaultAdvIntervalMs.
	// Values below the BLE minimum (20 ms) are clamped to it.
	IntervalMs int32

	// AdvData is a raw advertising payload (the AD-structure sequence a scan
	// result reports), replacing the default one-structure payload (the
	// general-discovery flags). The payload is copied into the glue's static
	// staging buffer; more than MaxAdvData bytes is ErrTooMany. When AdvData
	// already fills the packet, Name goes to the scan response.
	AdvData []byte

	// ScanRsp is a raw scan-response payload. When it is empty the package
	// builds one if the name needs the space; when the caller supplies it,
	// the name must fit the advertising payload (or it is refused: a passive
	// scanner would never see it in a response the caller owns).
	ScanRsp []byte

	// NonConnectable suppresses the connectable flag: the advertisement
	// announces presence only, and no central can connect to it.
	NonConnectable bool
}

// advFlags is the flags AD structure StartAdvertising composes when the
// caller did not bring one: general discovery, BR/EDR not supported (or, for
// a non-connectable advertisement, discovery without the LE-limited bit).
const advFlagsGeneral = 0x06

// buildAdvPayloads composes the two wire payloads out of params: the
// advertising packet (flags unless AdvData says otherwise, plus the name when
// it fits) and the scan response (the caller's, or the name's overflow). The
// composition is pure Go - the host tests walk it without a stack.
func buildAdvPayloads(p AdvParams) (adv, rsp []byte, err error) {
	if len(p.AdvData) > MaxAdvData || len(p.ScanRsp) > MaxAdvData {
		return nil, nil, ErrTooMany
	}
	adv = p.AdvData
	if adv == nil {
		// The conventional minimal payload: general discovery + BR/EDR off.
		adv = []byte{0x02, 0x01, advFlagsGeneral}
	}
	rsp = p.ScanRsp
	if p.Name == "" {
		return adv, rsp, nil
	}
	// The name becomes one AD structure: length byte, type byte, then the
	// name. It goes into the advertising payload when the whole (Complete
	// Local Name, 0x09) structure fits, else into the scan response, else as
	// a Shortened Local Name (0x08) - the longest prefix either packet
	// takes - and ErrTooMany when even that fits nowhere.
	for _, complete := range [2]bool{true, false} {
		n := len(p.Name)
		if !complete && n > MaxAdvData-2 {
			n = MaxAdvData - 2 // the longest prefix one packet can take
		}
		// The AD structure's Length field counts the type byte plus the name
		// (everything after the length byte itself), so a structure is
		// n+2 bytes on the wire with Length = n+1.
		st := []byte{byte(1 + n), 0x09}
		if !complete {
			st[1] = 0x08
		}
		st = append(st, p.Name[:n]...)
		if len(adv)+len(st) <= MaxAdvData {
			return append(adv, st...), rsp, nil
		}
		if rsp == nil {
			rsp = []byte{}
		}
		if len(rsp)+len(st) <= MaxAdvData {
			return adv, append(rsp, st...), nil
		}
	}
	return nil, nil, ErrTooMany
}

// StartAdvertising submits the advertisement and returns its OpID. The
// outcome arrives as EvAdvDone carrying that OpID: Status 0 means the
// controller is advertising with these payloads, anything else is the stack's
// refusal (an advertising procedure already running reports it there, not
// here - submission is asynchronous, like every control call).
func StartAdvertising(p AdvParams) (OpID, error) {
	// The argument checks run before the stack-state check, like Write's: a
	// bad payload is the caller's bug, and a host build must report it the
	// same way a device would.
	adv, rsp, err := buildAdvPayloads(p)
	if err != nil {
		return 0, err
	}
	if !Ready() {
		return 0, ErrNotReady
	}
	interval := p.IntervalMs
	if interval <= 0 {
		interval = DefaultAdvIntervalMs
	}
	if interval < 20 {
		interval = 20 // the BLE minimum for a legacy connectable advertisement
	}
	if interval > 10240 {
		interval = 10240 // the BLE maximum (10.24 s); the field is 16 bits of 0.625 ms
	}
	connectable := int32(1)
	if p.NonConnectable {
		connectable = 0
	}
	op := nextOp()
	var pa, pr *uint8
	if len(adv) > 0 {
		pa = &adv[0]
	}
	if len(rsp) > 0 {
		pr = &rsp[0]
	}
	if e := nimbleErr(advStart(pa, int32(len(adv)), pr, int32(len(rsp)), interval,
		connectable, uint32(op))); e != nil {
		return 0, e
	}
	return op, nil
}

// StopAdvertising stops the running advertisement. The outcome arrives as
// EvAdvDone (OpID-tagged, like StartAdvertising); a refusal (nothing is
// advertising) arrives there too. An advertisement that ended because a
// central connected reports nothing - the connect is the end, and
// EvPeriphConnect says so.
func StopAdvertising() error {
	if !Ready() {
		return ErrNotReady
	}
	return nimbleErr(advStop(uint32(nextOp())))
}

// The local GATT server's registry, Go-side: services in the order added,
// characteristics in the order added per service. The registry is what
// CommitServices serializes to the glue (once - the attribute table is built
// from these two flat arrays and never changes shape at runtime).
type gsvcReg struct {
	uuid       UUID
	numHandles int
	first      int // index of the service's first characteristic in gchrs
	nchars     int
}

type gchrReg struct {
	svc    int // index into gsvcs
	uuid   UUID
	props  uint8
	handle uint16 // the value attribute handle the stack assigned, 0 until CommitServices
}

var (
	gsvcs     []gsvcReg
	gchrs     []gchrReg
	committed bool // the registry was serialized to the glue (and frozen)
)

// AddService declares one primary service of the local GATT server: its UUID
// and the attribute-table size the stack should reserve (Bluedroid takes the
// count up front; NimBLE derives it, and the two agree when the count covers
// every attribute). The service holds 1 + (2, or 3 with a CCCD) handles per
// characteristic: declare the characteristics first as AddCharacteristic
// calls that follow, or leave headroom - CommitServices refuses a registry
// that does not fit its declared counts.
func AddService(uuid UUID, numHandles int) error {
	if uuid.IsZero() {
		return ErrBadUUID
	}
	if numHandles < 1 {
		return ErrBadValue
	}
	if committed {
		return ErrBadValue // the attribute table is already built
	}
	if len(gsvcs) >= MaxLocalServices {
		return ErrTooMany
	}
	for i := range gsvcs {
		if gsvcs[i].uuid == uuid {
			return ErrBadUUID // one UUID, one service
		}
	}
	gsvcs = append(gsvcs, gsvcReg{uuid: uuid, numHandles: numHandles, first: len(gchrs)})
	return nil
}

// AddCharacteristic declares one characteristic of the service svc (a UUID an
// AddService call added), with the ATT property bitmask the Prop* constants
// compose - the same bits a discovery reports. Notify and Indicate each add a
// client configuration descriptor (the stack's subscription gate), which is
// part of the handle budget AddService declared. Call order is attribute
// order: the characteristic appears in discovery results the way it was
// added here.
func AddCharacteristic(svc, chr UUID, props uint8) error {
	if svc.IsZero() || chr.IsZero() {
		return ErrBadUUID
	}
	if props&^(PropRead|PropWriteNoRsp|PropWrite|PropNotify|PropIndicate) != 0 {
		return ErrBadProperty // the package builds no broadcast/signed/extended properties
	}
	if committed {
		return ErrBadValue
	}
	if len(gchrs) >= MaxLocalChars {
		return ErrTooMany
	}
	for i := range gsvcs {
		if gsvcs[i].uuid != svc {
			continue
		}
		for j := gsvcs[i].first; j < len(gchrs); j++ {
			if gchrs[j].uuid == chr {
				return ErrBadUUID // one UUID, one characteristic per service
			}
		}
		gsvcs[i].nchars++
		gchrs = append(gchrs, gchrReg{svc: i, uuid: chr, props: props})
		return nil
	}
	return ErrNotFound // no such local service
}

// attrCount is the attribute count one characteristic costs: declaration,
// value, and a CCCD when it notifies or indicates.
func attrCount(props uint8) int {
	n := 2
	if props&(PropNotify|PropIndicate) != 0 {
		n++
	}
	return n
}

// checkRegistry validates the whole registry against the declared handle
// budgets, so a table that cannot fit its characteristics is refused in Go,
// before the glue's state machine could strand mid-build.
func checkRegistry() error {
	for i := range gsvcs {
		need := 1
		for j := gsvcs[i].first; j < gsvcs[i].first+gsvcs[i].nchars; j++ {
			need += attrCount(gchrs[j].props)
		}
		if need > gsvcs[i].numHandles {
			return ErrBadValue
		}
	}
	return nil
}

// gsvcWire and gchrWire mirror the glue's declaration records byte for byte
// (llgo_ble_gsvc_t / llgo_ble_gchr_t in the glue files, whose
// _Static_asserts pin the offsets): a UUID as width plus 16 little-endian
// bytes, the per-service characteristic count, the property bitmask.
type gsvcWire struct {
	Len    uint8
	UUID   [16]uint8
	_      uint8
	NChars uint16
}

type gchrWire struct {
	Len   uint8
	UUID  [16]uint8
	Props uint8
}

// CommitServices builds the local GATT server from the registry: the two flat
// arrays go to the glue once, the glue walks its stack's service-creation
// procedure (asynchronous, on the stack's own task), and the call returns
// when the attribute table is live - handles assigned, characteristics
// discoverable, values servable. The registry freezes: further AddService or
// AddCharacteristic calls are refused, and a second commit is ErrBadValue
// (rebuilding a live table is a Deinit-and-restart operation, not a refresh).
// A call whose wait expires (ErrTimeout) may still complete later; the
// completion event carries the operation id either way.
func CommitServices() error {
	// The registry-side refusals run before the stack-state check: an empty
	// or double commit is the caller's bug, reportable the same way on a
	// host build as on a device.
	if committed {
		return ErrBadValue
	}
	if len(gsvcs) == 0 {
		return ErrBadValue
	}
	if err := checkRegistry(); err != nil {
		return err
	}
	if !Ready() {
		return ErrNotReady
	}
	svcs := make([]gsvcWire, len(gsvcs))
	chrs := make([]gchrWire, len(gchrs))
	ci := 0
	for i := range gsvcs {
		svcs[i].Len = uint8(gsvcs[i].uuid.Len())
		svcs[i].UUID = gsvcs[i].uuid.v
		svcs[i].NChars = uint16(gsvcs[i].nchars)
		for j := gsvcs[i].first; j < gsvcs[i].first+gsvcs[i].nchars; j++ {
			chrs[ci].Len = uint8(gchrs[j].uuid.Len())
			chrs[ci].UUID = gchrs[j].uuid.v
			chrs[ci].Props = gchrs[j].props
			ci++
		}
	}
	op := nextOp()
	sub := Subscribe(EvServicesDone, EvError)
	defer sub.Close()
	if e := nimbleErr(gattsDeclare(&svcs[0], int32(len(svcs)), &chrs[0], int32(len(chrs)), uint32(op))); e != nil {
		return e
	}
	ev, err := waitOp(sub, op)
	if err != nil {
		return err
	}
	if ev.Status != 0 {
		return nimbleErr(ev.Status)
	}
	committed = true
	// The stack assigned a value handle per characteristic during creation;
	// pull the whole assignment in registration order (the glue's table is
	// indexed the same way).
	for i := range gchrs {
		var h uint16
		if gattsAttr(int32(i), &h) == 1 {
			gchrs[i].handle = h
		}
	}
	return nil
}

// findLocalChr resolves a value handle to its registration (the reverse of
// LocalHandle: what the write and subscribe events carry).
func findLocalChr(handle uint16) (gchrReg, bool) {
	for i := range gchrs {
		if gchrs[i].handle == handle {
			return gchrs[i], true
		}
	}
	return gchrReg{}, false
}

// findLocalByUUID resolves a characteristic UUID to its registration.
func findLocalByUUID(uuid UUID) (gchrReg, bool) {
	for i := range gchrs {
		if gchrs[i].uuid == uuid {
			return gchrs[i], true
		}
	}
	return gchrReg{}, false
}

// LocalHandle reports the value attribute handle the stack assigned to the
// characteristic chr: the handle WriteCharValue takes, the one EvCharWrite
// and EvSubscribed carry, and the one Notify addresses. It is meaningful only
// after CommitServices (0 and ErrNotReady before - handles do not exist
// before the attribute table is built). An unknown UUID is ErrNotFound.
func LocalHandle(chr UUID) (uint16, error) {
	if !committed {
		return 0, ErrNotReady
	}
	reg, ok := findLocalByUUID(chr)
	if !ok {
		return 0, ErrNotFound
	}
	return reg.handle, nil
}

// WriteCharValue sets a characteristic's local value: what the glue answers
// a peer's read with, and what a peer sees before its first write. The call
// copies the bytes into the glue's static cache and returns; a value larger
// than MaxData is ErrTooLong (truncation would invent data), a handle that is
// not a committed characteristic is ErrNotFound.
func WriteCharValue(handle uint16, data []byte) error {
	if handle == 0 {
		return ErrBadValue
	}
	if len(data) > MaxData {
		return ErrTooLong
	}
	if !committed {
		return ErrNotReady
	}
	if _, ok := findLocalChr(handle); !ok {
		return ErrNotFound
	}
	var p *uint8
	if len(data) > 0 {
		p = &data[0]
	}
	return nimbleErr(gattsValue(handle, p, int32(len(data))))
}

// Notify pushes data to every subscribed peer of the characteristic handle,
// unacknowledged (the ATT notification procedure). The value travels to the
// glue as an immediate copy, so the caller's buffer is free on return; the
// returned error reports the submission only. On the NimBLE backend the send
// itself runs later on the host task, and a send it refuses there (a peer
// that is gone, out of mbufs) is reported as EvError - a plain Subscribe
// watcher sees it. An unsubscribed peer stays silent either way: the ATT
// notification has no failure report for it, which is what
// "unacknowledged" means on the wire.
func Notify(conn ConnID, handle uint16, data []byte) error {
	return gattsPush(conn, handle, data, false)
}

// Indicate pushes data to every subscribed peer of the characteristic handle,
// acknowledged (the ATT indication procedure): the stack waits for the
// peer's confirmation and enforces the one-outstanding-indication rule
// itself. Like Notify, the returned error reports the submission only.
func Indicate(conn ConnID, handle uint16, data []byte) error {
	return gattsPush(conn, handle, data, true)
}

// gattsPush is the shared body of Notify and Indicate.
func gattsPush(conn ConnID, handle uint16, data []byte, indicate bool) error {
	if handle == 0 {
		return ErrBadValue
	}
	if len(data) > MaxData {
		return ErrTooLong
	}
	if !Ready() {
		return ErrNotReady
	}
	reg, ok := findLocalChr(handle)
	if !ok {
		return ErrNotFound
	}
	if indicate && reg.props&PropIndicate == 0 {
		return ErrBadProperty
	}
	if !indicate && reg.props&PropNotify == 0 {
		return ErrBadProperty
	}
	var p *uint8
	if len(data) > 0 {
		p = &data[0]
	}
	ind := int32(0)
	if indicate {
		ind = 1
	}
	return nimbleErr(gattsNotify(uint32(conn), handle, p, int32(len(data)), ind))
}

// DisconnectPeer terminates a peripheral-side connection (a central that
// connected to this device's advertising). The teardown arrives as
// EvPeriphDisconnect (Status carries the HCI reason); a ConnID that no longer
// matches a live connection arrives as EvError with StatusENOENT.
func DisconnectPeer(id ConnID) error {
	if !Ready() {
		return ErrNotReady
	}
	return nimbleErr(gattsClose(uint32(id)))
}

// IoCap names the device's input/output capability for the pairing
// procedure. The numeric values are the Bluetooth-assigned ones, identical in
// both stacks' APIs.
type IoCap uint8

const (
	IoCapDisplayOnly     IoCap = 0 // has a display, no input: show the passkey
	IoCapYesNo           IoCap = 1 // has a display and yes/no input
	IoCapKeyboardOnly    IoCap = 2 // has a keyboard, no display: enter the passkey
	IoCapNoInputNoOutput IoCap = 3 // neither: just-works pairing (no MITM)
	IoCapKeyboardDisplay IoCap = 4 // both
)

// SecurityParams configures the pairing posture (SetSecurityParams). The
// zero value is the posture Init installs anyway: DisplayOnly, bonding with
// MITM where the peer supports it. Passkey 0 means "no fixed passkey": MITM
// then comes from the IO-capability exchange alone.
type SecurityParams struct {
	// IoCap is what this device can display and input (the constants above).
	IoCap IoCap

	// Passkey is a 6-digit fixed passkey (0..999999) for peers whose IO
	// combination resolves to passkey entry. A fixed passkey plus MITM makes
	// a phone's first-time pairing show the code this device displays.
	Passkey uint32

	// Bond stores the pairing keys (NVS) so a reconnecting peer skips pairing.
	// Without it, every reconnection pairs again.
	Bond bool

	// MITM requires man-in-the-middle protection on the pairing - the
	// property that makes an over-the-air passkey comparison meaningful. A
	// NoInputNoOutput device cannot provide it, and pairing with a peer that
	// demands it fails (the honest outcome: refusing silently would fake
	// security).
	MITM bool

	// KeySize is the encryption key size in bytes (7..16). The zero value and
	// every other value are clamped to 16, the BLE maximum: a smaller key
	// ships no security this stack could not negotiate up to, and - the
	// Bluedroid trap - a MAX_KEY_SIZE that was never set distributes no key
	// at all while the pairing still reports success.
	KeySize uint8
}

// SetSecurityParams installs the pairing posture. Call it once, after Init
// and before the first connection: the parameters are read when a pairing
// starts, and changing them mid-pairing is undefined in both stacks. This is
// the knob the provisioning stage needs (a phone pairs and encrypts before it
// writes WiFi credentials); without a call, Init's default posture stays
// active.
func SetSecurityParams(p SecurityParams) error {
	if !Ready() {
		return ErrNotReady
	}
	keySize := p.KeySize
	if keySize < 7 || keySize > 16 {
		keySize = 16
	}
	passkey := p.Passkey
	if passkey > 999999 {
		passkey = 0 // not a 6-digit value: refuse the fixed passkey silently
	}
	mitm, bond := int32(0), int32(0)
	if p.MITM {
		mitm = 1
	}
	if p.Bond {
		bond = 1
	}
	return espErr(securityParams(int32(p.IoCap), mitm, bond, passkey, int32(keySize)))
}

// svcRec and chrRec mirror the glue's discovery records byte for byte
// (llgo_ble_svc_t / llgo_ble_chr_t in glue/ble_glue_nimble.c, whose
// _Static_asserts pin the offsets): the UUID travels as a width plus 16
// little-endian bytes.
type svcRec struct {
	UUIDLen uint8
	UUID    [16]uint8
	Start   uint16
	End     uint16
}

type chrRec struct {
	UUIDLen    uint8
	UUID       [16]uint8
	Properties uint8
	Value      uint16
	Def        uint16
}

// nextOp hands out the next operation id (never 0).
func nextOp() OpID {
	opSeq++
	if opSeq == 0 {
		opSeq = 1
	}
	return OpID(opSeq)
}

// AddrString renders a 6-byte BLE address most-significant octet first
// ("c0:ff:ee:00:00:01"), the conventional notation for what the stack stores
// least-significant first.
func AddrString(a [6]uint8) string {
	const hex = "0123456789abcdef"
	var out [17]byte
	for i := 0; i < 6; i++ {
		v := a[5-i]
		out[i*3] = hex[v>>4]
		out[i*3+1] = hex[v&0xF]
		if i < 5 {
			out[i*3+2] = ':'
		}
	}
	return string(out[:17])
}

// AdvName returns the local name an advertising payload carries (the first
// Complete Local Name, or else the Shortened Local Name), or "" when it has
// neither. The payload is the sequence of AD structures the advertiser sent:
// length, type, value.
func AdvName(adv []uint8) string {
	for i := 0; i < len(adv); {
		l := int(adv[i])
		if l == 0 || i+1+l > len(adv) {
			return ""
		}
		t := adv[i+1]
		if t == 0x09 || t == 0x08 { // Complete / Shortened Local Name
			name := adv[i+2 : i+1+l]
			return string(name)
		}
		i += 1 + l
	}
	return ""
}

// ErrName maps a code this package can report to its name: the Go-domain
// codes, the NimBLE host statuses a central meets (named BLE_HS_*), the common
// HCI disconnection reasons, and the esp_err codes the lifecycle calls can
// return.
// DisconnectReasonName renders the HCI reason an EvDisconnect carries (the
// Status field, also the value the reason accessor reports) as text: a bare
// 0x08 or 0x13 in a log says nothing, "connection timeout" and "remote user
// terminated" tell the reader what happened. Stack-side codes above 0xFF use
// the Bluedroid GATT extension names.
func DisconnectReasonName(code int32) string {
	switch code {
	case 0x00:
		return "success"
	case 0x01:
		return "unknown HCI error"
	case 0x02:
		return "unknown connection identifier"
	case 0x03:
		return "hardware failure"
	case 0x04:
		return "page timeout"
	case 0x05:
		return "authentication failure"
	case 0x06:
		return "PIN or key missing"
	case 0x07:
		return "memory capacity exceeded"
	case 0x08:
		return "connection timeout (supervision timeout)"
	case 0x09:
		return "connection limit exceeded"
	case 0x0a:
		return "synchronous connection limit exceeded"
	case 0x0b:
		return "connection already exists"
	case 0x0c:
		return "command disallowed"
	case 0x0d:
		return "connection rejected: limited resources"
	case 0x0e:
		return "connection rejected: security reasons"
	case 0x0f:
		return "connection rejected: unacceptable BD_ADDR"
	case 0x10:
		return "connection accept timeout"
	case 0x11:
		return "unsupported feature or parameter value"
	case 0x12:
		return "invalid HCI command parameters"
	case 0x13:
		return "remote user terminated the connection"
	case 0x14:
		return "remote device terminated: low resources"
	case 0x15:
		return "remote device terminated: power off"
	case 0x16:
		return "local host terminated the connection"
	case 0x17:
		return "repeated attempts"
	case 0x18:
		return "pairing not allowed"
	case 0x19:
		return "unknown LMP PDU"
	case 0x1a:
		return "unsupported remote feature"
	case 0x1b:
		return "unsupported LMP parameter value"
	case 0x1c:
		return "role change not allowed"
	case 0x1d:
		return "LMP response timeout"
	case 0x1e:
		return "LMP error transaction collision"
	case 0x1f:
		return "LMP PDU not allowed"
	case 0x20:
		return "encryption mode not acceptable"
	case 0x21:
		return "link key cannot be changed"
	case 0x22:
		return "requested QoS not supported"
	case 0x23:
		return "instant passed"
	case 0x24:
		return "pairing with unit key not supported"
	case 0x25:
		return "different transaction collision"
	case 0x28:
		return "connection parameters unacceptable"
	case 0x29:
		return "peer rejected: QoS parameters"
	case 0x2a:
		return "peer rejected: insufficient security"
	case 0x3b:
		return "unacceptable connection interval"
	case 0x3d:
		return "MIC failure (link not secure)"
	case 0x3e:
		return "connection failed to be established"
	case 0x100:
		return "connection establish failed"
	case 0x101:
		return "LMP response timeout"
	case 0x102:
		return "connection cancelled"
	case 0x103:
		return "no connection to cancel"
	}
	h := hex4(uint16(code))
	for len(h) > 2 && h[0] == '0' { // "007e" reads worse than "0x7e"
		h = h[1:]
	}
	return "reason 0x" + h
}

func ErrName(code int32) string {
	if code >= NimbleBase {
		return nimbleName(code - NimbleBase)
	}
	switch code {
	case 0:
		return "ESP_OK"
	case -1:
		return "ESP_FAIL"
	case 0x101:
		return "ESP_ERR_NO_MEM"
	case 0x102:
		return "ESP_ERR_INVALID_ARG"
	case 0x103:
		return "ESP_ERR_INVALID_STATE"
	case 0x106:
		return "ESP_ERR_NOT_SUPPORTED"
	case 0x107:
		return "ESP_ERR_TIMEOUT"
	case 0x110D:
		return "ESP_ERR_NVS_NO_FREE_PAGES"
	case 0x1110:
		return "ESP_ERR_NVS_NEW_VERSION_FOUND"
	case codeTimeout:
		return "BLE_ERR_TIMEOUT"
	case codeNotReady:
		return "BLE_ERR_NOT_READY"
	case codeBadUUID:
		return "BLE_ERR_BAD_UUID"
	case codeBadValue:
		return "BLE_ERR_BAD_VALUE"
	case codeBusy:
		return "BLE_ERR_BUSY"
	case codeBadProperty:
		return "BLE_ERR_BAD_PROPERTY"
	case codeNotFound:
		return "BLE_ERR_NOT_FOUND"
	case codeTooLong:
		return "BLE_ERR_TOO_LONG"
	case codeTooMany:
		return "BLE_ERR_TOO_MANY"
	case codeUnsupported:
		return "BLE_ERR_UNSUPPORTED"
	}
	return "esp_err 0x" + hex6(uint32(code))
}

// nimbleName names a NimBLE host status: the host-core codes, the HCI
// disconnection reasons a central's lifetime produces, and a generic form for
// everything else.
func nimbleName(status int32) string {
	switch status {
	case 0x01:
		return "BLE_HS_EAGAIN"
	case 0x02:
		return "BLE_HS_EALREADY"
	case 0x03:
		return "BLE_HS_EINVAL"
	case 0x04:
		return "BLE_HS_EMSGSIZE"
	case 0x05:
		return "BLE_HS_ENOENT"
	case 0x06:
		return "BLE_HS_ENOMEM"
	case 0x07:
		return "BLE_HS_ENOTCONN"
	case 0x08:
		return "BLE_HS_ENOTSUP"
	case 0x09:
		return "BLE_HS_EAPP"
	case 0x0A:
		return "BLE_HS_EBADDATA"
	case 0x0B:
		return "BLE_HS_EOS"
	case 0x0C:
		return "BLE_HS_ECONTROLLER"
	case 0x0D:
		return "BLE_HS_ETIMEOUT"
	case 0x0E:
		return "BLE_HS_EDONE"
	case 0x0F:
		return "BLE_HS_EBUSY"
	}
	if status >= 0x0200 { // HCI namespace: BLE_HS_ERR_HCI_BASE | HCI error code
		switch status & 0xFF {
		case 0x01:
			return "HCI_UNKNOWN_CONN_ID"
		case 0x02:
			return "HCI_UNKNOWN_CONN_ID"
		case 0x03:
			return "HCI_HARDWARE_FAILURE"
		case 0x05:
			return "HCI_AUTH_FAILURE"
		case 0x06:
			return "HCI_PIN_OR_KEY_MISSING"
		case 0x08:
			return "HCI_CONN_TIMEOUT"
		case 0x0C:
			return "HCI_CONN_REJECTED"
		case 0x13:
			return "HCI_REMOTE_USER_TERM_CONN"
		case 0x16:
			return "HCI_LOCAL_HOST_TERM_CONN"
		case 0x22:
			return "HCI_LMP_RESPONSE_TIMEOUT"
		case 0x28:
			return "HCI_INSTANT_PASSED"
		case 0x3B:
			return "HCI_UNACCEPTABLE_CONN_PARAMS"
		case 0x3E:
			return "HCI_CONN_FAILED_TO_BE_ESTABLISHED"
		}
	}
	if status >= 0x0100 {
		return "ATT 0x" + hex2(uint8(status))
	}
	return "nimble 0x" + hex6(uint32(status))
}

// hexDigits is the lowercase hex alphabet: the canonical UUID and address
// spelling (hex2 below uppercases only the ATT status rendering).
const hexDigits = "0123456789abcdef"

// hexDigit decodes one lowercase or uppercase hex digit.
func hexDigit(c uint8) (uint8, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}

// hex16 decodes exactly four hex digits.
func hex16(s string) (uint16, bool) {
	var v uint16
	for i := 0; i < 4; i++ {
		d, ok := hexDigit(s[i])
		if !ok {
			return 0, false
		}
		v = v<<4 | uint16(d)
	}
	return v, true
}

// hex4 formats a 16-bit value as four lowercase hex digits.
func hex4(v uint16) string {
	return string([]byte{
		hexDigits[v>>12&0xF], hexDigits[v>>8&0xF], hexDigits[v>>4&0xF], hexDigits[v&0xF],
	})
}

// hex2 formats the low byte as two hex digits.
func hex2(v uint8) string {
	const digits = "0123456789ABCDEF"
	return string([]byte{digits[v>>4], digits[v&0xF]})
}

// hex6 formats the low 24 bits as fixed-width hex.
func hex6(v uint32) string {
	const digits = "0123456789ABCDEF"
	var buf [6]byte
	for i := 0; i < 6; i++ {
		buf[5-i] = digits[(v>>(uint(i)*4))&0xF]
	}
	return string(buf[:])
}
