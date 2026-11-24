// Package ble binds llgo-compiled Go code to the ESP-IDF NimBLE host as a BLE
// central: bring-up, scanning, connecting, disconnecting, and the stack's
// asynchronous results as a fan-out event mailbox.
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
//	disErr := ble.Disconnect(ev.ConnID)
//	ev, err = sub.Next(5000)                      // EvDisconnect: reason
//
// Control calls never block: StartScan/Connect/Disconnect hand their command
// to the NimBLE host's own event queue and return an operation id (OpID), so
// the caller's goroutine keeps the llgo scheduler while the host task runs the
// GAP procedure. The outcome always arrives through the mailbox - a returned
// error means "the command could not even be submitted" (not ready, queue
// full, BLE not in this firmware), never a forged result.
//
// # Why a mailbox, not callbacks
//
// NimBLE runs its reset/sync/GAP callbacks on the host task, and the llgo
// fiber scheduler is single-task global state: an entry into Go from a second
// FreeRTOS task would corrupt it. The glue copies every result into two
// bounded C-side rings (control events in one, scan results in the other, so
// a scan flood can never displace a connect/disconnect), and one poller
// goroutine drains them and fans them out to every subscription - a slow
// consumer loses its own events, never anyone else's. Overflow is reported
// once as EvOverflow carrying the dropped count.
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
//	               no host sync), ErrUnsupported.
//
// The offset matters: NimBLE's status 0x02 (BLE_HS_EALREADY) and an HCI error
// 0x02 are different conditions in different namespaces, and an esp_err 0x02
// is none of them. Tagging the nimble domain keeps errors.Is from mixing
// them, and keeps an esp_err comparison from ever matching a host status.
//
// # Scope
//
// This is the central half of GAP only: scanning (legacy discovery), connect,
// cancel, disconnect. No GATT client (discovery, read, write, notifications)
// and no peripheral role - those are separate stages; the glue's
// mailbox has the shape they need (a control tier for results a state machine
// depends on, a payload tier with a bounded copy).
package ble

import "github.com/xgo-dev/llgo/pkg/esp/esperr"

// NimbleBase offsets a NimBLE host status (BLE_HS_*, ATT, HCI or L2CAP, all
// below 0x8000) into this package's code space. See the error model above.
const NimbleBase = 0x8000

// The Go-domain codes: conditions this package detects itself, above the
// esperr meta segment (0x7Cxx) and below the tag stubs use (0x7F00).
const (
	codeTimeout     = 0x7B01
	codeNotReady    = 0x7B02
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

	// EvOverflow reports mailbox loss: Status is the number of records dropped
	// since the last report. Control records are dropped only when the control
	// ring is full, which takes a program that stops consuming events.
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
)

// Peer is the advertiser to connect to: the address a scan result reported
// (Addr in the stack's little-endian order - Addr[0] is the least significant
// octet, which is what AddrString renders last) and its address type.
type Peer struct {
	Addr     [6]uint8
	AddrType uint8
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
}

// AdvData returns the advertising payload this event carries, as a slice over
// the event's own array: the bytes were copied out of the HCI buffer in the
// glue, so the payload is already detached from the stack and stays valid for
// as long as the Event does.
func (e *Event) AdvData() []uint8 { return e.Adv[:e.AdvLen] }

// Peer returns the advertiser address as a connectable Peer.
func (e *Event) Peer() Peer { return Peer{Addr: e.Addr, AddrType: e.AddrType} }

// Subscription receives a copy of every mailbox event whose kind it asked for
// (EvOverflow goes to everyone). Create one with Subscribe, consume it with
// Next, retire it with Close. A Subscription is single-consumer: exactly one
// goroutine calls Next on it.
type Subscription struct {
	kinds []EventKind
	ch    chan Event
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
	subscribers = append(subscribers, s)
	if !pollerStarted {
		pollerStarted = true
		go pollEvents()
	}
	return s
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
		if s.delivers(ev.Kind) {
			select {
			case s.ch <- ev:
			default:
			}
		}
	}
}

// delivers reports whether events of kind go to s. An empty filter matches
// every kind; the mailbox's global overflow report matches all of them.
func (s *Subscription) delivers(kind EventKind) bool {
	if kind == EvOverflow {
		return true
	}
	if len(s.kinds) == 0 {
		return true
	}
	for _, k := range s.kinds {
		if k == kind {
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
