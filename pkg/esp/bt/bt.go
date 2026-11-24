// Package bt binds llgo-compiled Go code to the ESP-IDF Bluedroid classic
// Bluetooth stack: lifecycle control, an SPP (Serial Port Profile) server
// data channel, and an A2DP sink whose decoded-PCM audio path runs entirely
// in C (the stack's own SBC decoder feeds an I2S TX channel; see A2DPEnable).
// The lifecycle and event plumbing below are shared by every profile, so
// further profiles mean new API functions plus new mailbox kinds, not a
// redesign.
//
// # Layout
//
// Every package under pkg/esp follows the same template (glue lives in a
// subdirectory because the go tool rejects C sources inside a Go package):
//
//	bt.go               public API, no build tag
//	bt_idf.go           //go:build llgo.idf - bodyless //go:linkname externs
//	bt_stub.go          //go:build !llgo.idf - host-build stubs (go vet/gopls)
//	bt_test.go          host-build tests (stub honesty, error spaces)
//	glue/bt_glue_bluedroid.c - ESP-IDF C glue template for the stack
//	                    lifecycle, SPP and the shared ctrl ring (compiled
//	                    only under ESP_PLATFORM)
//	glue/bt_glue_a2dp.c - the A2DP sink sister glue: audio path
//	                    (PCM ring + pump task + I2S TX) and the profile's
//	                    control calls; shares the ctrl ring publisher and
//	                    the ready flag with the sibling (both templates get
//	                    copied into the project's main component)
//
// # Usage
//
// On the device (llgo build -target esp32s31-idf) the package talks to
// Bluedroid through the glue; on any other build the stubs return
// ErrUnsupported, so `go build` / `go vet` / gopls all stay clean:
//
//	err := bt.Init()                        // controller BTDM + bluedroid enable
//	bt.SetName("llgo-bt")
//	bt.SetDiscoverable(true)
//	err = bt.SPPListen(bt.SPPOptions{})     // SCN 1, service name "llgo-spp"
//	conn, err := bt.SPPAccept(30000)        // waits for a peer (bumble, phone)
//	n, err := conn.Read(buf[:], 5000)
//	conn.Write(buf[:n], 5000)
//	conn.Close()
//	bt.Deinit()
//
// Every driver result comes back as the family's shared error value: nil =
// ESP_OK, and the error carries the raw code (inspectable; bt.ErrName names
// the codes this pipeline can hit with their IDF names).
//
// # Events: mailbox, not callbacks
//
// Bluedroid profile callbacks run on the BTC task, which must never enter the
// single-task llgo fiber scheduler. The glue therefore only copies events into
// C-side rings; a single Go poller goroutine drains them and fans each event
// out to every matching Subscription (the same shape the ble package uses:
// lifecycle events and data reception need parallel waiters, so the
// single-consumer Poll model of the wifi package would make an Accept-waiting
// goroutine and an echo-loop goroutine steal events from each other).
//
// SPP received data does NOT travel through that mailbox: it lands in its own
// static slot ring, and SPPConn.Read pulls records straight out with one C
// call, so a 1 KiB RFCOMM payload never crosses a subscription channel.
//
// # Subscribers: one poller, many consumers
//
// Subscribe lazily starts the package's one poller goroutine and fans every
// polled event out to all matching subscriptions; Subscription.Next is a
// bounded wait that never holds a C-level call (each iteration yields with
// Gosched, and the poller rests the FreeRTOS task one tick per pass, which
// feeds the idle tasks and the watchdog).
//
// The package's own control calls wait through the same machinery: SPPListen
// subscribes for the SPP init/start confirmations, SPPAccept for the
// connection-open event, and a connection keeps a private subscription
// filtered to its own close events. Internal and user subscriptions receive
// independent copies - watching EvSPPConnOpen yourself does not steal an
// Accept's event.
//
// # Write flow control
//
// IDF v6.3's esp_spp_write copies the whole buffer into the profile's TX
// ringbuffer synchronously and reports queue-fullness through its return code
// (ESP_OK accepted, ESP_ERR_NO_MEM full). Write therefore needs no
// operation-id matching against ESP_SPP_WRITE_EVT: it chunks to MaxMTU, and on
// ESP_ERR_NO_MEM waits for the drain edge (ESP_SPP_CONG_EVT, reported here as
// EvSPPCong) before retrying, bounded by the call's timeout. Write returns
// once the bytes are queued in the TX buffer, which is earlier than on-air:
// the stack drains to the peer at RFCOMM speed afterwards.
//
// # Profiles and the shared plumbing
//
// The ctrl-ring kinds are the profile-shared vocabulary: SPP owns 1..8 and
// the A2DP sink (same package, glue/bt_glue_a2dp.c) owns 9..12; the
// internal error segment 0x77xx belongs to the A2DP domain (0x78 stays
// reserved for further classic-audio profiles, like an HFP stage). A new
// profile adds API functions, glue cases and kinds - the lifecycle, the
// mailbox and the fan-out below do not change shape.
//
// # Glue integration
//
// The C glue is a template: copy glue/bt_glue_bluedroid.c into your project's
// main component (WHOLE_ARCHIVE required - see pkg/esp/README.md for both
// integration recipes and the boundary rules the glue must obey).
package bt

import (
	"github.com/xgo-dev/llgo/pkg/esp/esperr"
)

// ---- error domain ----
//
// The package's own conditions live in the internal segment 0x76xx (esperr
// meta 0x7Cxx and ble 0x7Bxx are taken; 0x7D/0x7E for ws/tls, 0x7F00 is the
// stub tag). The classic-audio profile that shares this package has its own
// segment: 0x77xx is A2DP's. Raw esp_err codes, esp_bt_status_t and
// esp_spp_status_t values pass through unchanged and are named by ErrName.
const (
	codeGlueABI   = 0x7601
	codeNotReady  = 0x7602
	codeBusy      = 0x7603
	codeTimeout   = 0x107 // ESP_ERR_TIMEOUT, the esp_err domain
	codeUnsupport = 0x7F00

	// 0x77xx is the A2DP domain's internal segment (this package hosts the
	// classic-audio profile); no sentinel needs it yet - the profile's
	// control calls surface the stack's own ESP_ERR_* codes.
)

var (
	// ErrGlueABI reports a glue whose contract version differs from this
	// package's: the firmware's bt_glue.c copy and the Go package were not
	// built from the same source.
	ErrGlueABI = &esperr.Error{Code: codeGlueABI, Name: ErrName(codeGlueABI)}
	// ErrNotReady reports a control call made before Init completed (or
	// after Deinit): the controller is down, no SPP operation can start.
	ErrNotReady = &esperr.Error{Code: codeNotReady, Name: ErrName(codeNotReady)}
	// ErrBusy reports a concurrency guard: a second SPPAccept while one is
	// already waiting (the MVP serves one connection at a time).
	ErrBusy = &esperr.Error{Code: codeBusy, Name: ErrName(codeBusy)}
	// ErrTimeout reports a deadline that passed with the operation
	// unfinished (ESP_ERR_TIMEOUT).
	ErrTimeout = &esperr.Error{Code: codeTimeout, Name: ErrName(codeTimeout)}
	// ErrUnsupported is what a host build reports for every call: the
	// value the stub bindings return, named above the esperr/ws/tls meta
	// segments so it can never be mistaken for a real code.
	ErrUnsupported = &esperr.Error{Code: codeUnsupport, Name: ErrName(codeUnsupport)}
)

// errOf wraps a raw pipeline code (0 = nil) as the error value this package
// returns, named through ErrName so a printed error reads like the ErrName
// table's output.
func errOf(code int32) error {
	if code == 0 {
		return nil
	}
	return &esperr.Error{Code: code, Name: ErrName(code)}
}

// ---- constants ----
const (
	// DefaultSCN is the RFCOMM server channel SPPListen uses when
	// SPPOptions.SCN is zero.
	DefaultSCN = 1

	// DefaultSPPName is the SDP service name SPPListen publishes when
	// SPPOptions.Name is empty. A client discovering the service by name
	// (bumble does) looks for this string.
	DefaultSPPName = "llgo-spp"

	// MaxMTU is the default RFCOMM MTU the stack negotiates
	// (ESP_SPP_MAX_MTU, 990 bytes): Write's chunk size and the bound of
	// one on-air record. A peer may negotiate a smaller MTU; the stack
	// re-chunks queued data itself, so Write always uses this bound.
	MaxMTU = 990

	// maxRecord is the largest record the glue data ring holds. An RFCOMM
	// frame never exceeds MaxMTU, but the ring keeps headroom so an MTU
	// the stack grew beyond the default still fits.
	maxRecord = 1024

	// dataRingSlots is the glue data ring's slot count (SPP_DATA_SLOTS in
	// the C glue; the drain loop's bound).
	dataRingSlots = 8

	// SPPWaitMs bounds SPPListen's waits for the profile's INIT and START
	// confirmations (each is one BTC-task round trip; five seconds
	// covers a busy bring-up).
	SPPWaitMs = 5000

	// CloseWaitMs bounds SPPConn.Close's wait for the CLOSE confirmation
	// after a local disconnect.
	CloseWaitMs = 1000

	// SubQueueLen is the per-subscription event buffer. The poller drops
	// events for a subscriber that has not drained it instead of blocking
	// the fan-out for everyone; the loss is reported once through
	// EvOverflow.
	SubQueueLen = 8

	// pollQuantumMs is the wait quantum shared by the poller and by every
	// bounded call: one whole number of FreeRTOS ticks (see tickMs).
	pollQuantumMs = 20

	// tickMs is the FreeRTOS tick period the glue reports at init
	// (portTICK_PERIOD_MS, 10 at the IDF default 100 Hz).
	tickMs = 10

	// maxPollBatch bounds how many records the poller moves per pass, so
	// an event flood cannot hold the scheduler: the batch ends with a
	// Gosched and one tick of rest.
	maxPollBatch = 32
)

// glueABIVersion is the contract version this Go package expects from the
// glue (llgo_bt_glue_abi). Bump both sides together when a signature or
// mailbox kind changes.
const glueABIVersion = 1

// ---- events ----
//
// EventKind values are the ctrl-ring kinds. They are the profile-shared
// vocabulary: SPP owns 1..8, the A2DP sink 9..12; a further profile
// continues after them without renumbering anything.
type EventKind int32

const (
	// EvOverflow reports dropped mailbox records: at least one ctrl or
	// SPP data record was lost because its ring was full when the event
	// arrived. It goes to every subscription; Status counts ctrl-ring
	// drops and Aux counts SPP data drops since boot (the counters are
	// monotonic, so each report is one edge).
	EvOverflow EventKind = -1
	// EvSPPInit reports the SPP profile finished (or failed: Status) its
	// enhanced-init round trip. SPPListen waits for it internally.
	EvSPPInit EventKind = 1
	// EvSPPListen reports the SPP server started (or failed: Status):
	// the SDP record is up and the RFCOMM channel is listening. Aux is
	// the server channel (SCN). SPPListen waits for it internally.
	EvSPPListen EventKind = 2
	// EvSPPConnOpen reports a peer connected to the SPP server: Handle is
	// the connection handle, Addr the peer's BD address. SPPAccept waits
	// for it internally.
	EvSPPConnOpen EventKind = 3
	// EvSPPConnClose reports an SPP connection closed: Handle is the
	// connection, Aux is 1 when the peer initiated the close (SPPRead
	// then reports esperr.ErrEOF) and 0 when the local side did
	// (SPPConn.Close waits for it internally).
	EvSPPConnClose EventKind = 4
	// EvSPPCong reports the SPP TX buffer drained (congestion lifted) for
	// connection Handle: Write's retry signal. The v6.3 stack reports the
	// edge only - once per drain - never a congested state.
	EvSPPCong EventKind = 5
	// EvSPPStop reports the SPP server stopped listening (Aux is the SCN
	// it served). Part of Deinit's teardown.
	EvSPPStop EventKind = 6
	// EvSPPUninit reports the SPP profile deinitialized. Part of Deinit's
	// teardown.
	EvSPPUninit EventKind = 7
	// EvACLDisconnect reports the ACL (baseband) link under an SPP
	// connection tore down: Handle is the ACL handle, Addr the peer, and
	// Status the disconnection reason (an esp_bt_status_t value in the
	// HCI range 0x100+; ErrName names the common ones). An SPP
	// EvSPPConnClose always accompanies or follows it.
	EvACLDisconnect EventKind = 8

	// The A2DP kinds. The audio stream itself never crosses into Go - the
	// glue pipes the decoded PCM straight into I2S (see A2DPEnable) - so
	// these are control-plane records only, and they reuse the Event's
	// existing fields instead of growing the struct:
	EvA2DPConnected EventKind = 9
	// EvA2DPDisconnected reports the A2DP transport tore down: Addr is the
	// peer, Status the reason (0 = normal local/remote disconnect, 1 =
	// abnormal link loss), Handle the connection handle.
	EvA2DPDisconnected EventKind = 10
	// EvA2DPAudioCfg reports the negotiated SBC codec config: Aux is the
	// sample rate in Hz, Status the channel count (1 or 2), Addr the peer.
	// The glue re-clocks the I2S channel to match before the next PCM
	// chunk plays; the event exists so a control plane can log/report what
	// the analog path now runs (A2DP has no get-config API - this event is
	// the only source).
	EvA2DPAudioCfg EventKind = 11
	// EvA2DPVolume reports a volume in Aux (0..127): either the phone's
	// absolute-volume command arrived (AVRCP TG side) or the peer confirmed
	// our own A2DPSetVolume (CT response).
	EvA2DPVolume EventKind = 12
)

// Event is one ctrl-ring record. The fields are meaningful per kind (see the
// Ev* constants); everything a kind does not use is zero.
type Event struct {
	// Kind is one of the Ev* values above.
	Kind int32
	// Status carries the event's raw status code: an esp_bt_status_t or
	// esp_spp_status_t for SPP lifecycle kinds (0 = success), and the
	// disconnection reason for EvACLDisconnect. EvOverflow uses it for
	// the ctrl-ring drop count.
	Status int32
	// Handle is the SPP connection handle for the connection kinds and
	// the ACL handle for EvACLDisconnect.
	Handle uint32
	// Aux carries the kind's secondary value: the SCN for
	// EvSPPListen/EvSPPStop, 1 for a peer-initiated EvSPPConnClose, the
	// TX-drain edge for EvSPPCong, the SPP data-ring drop count for
	// EvOverflow, the sample rate in Hz for EvA2DPAudioCfg, and the
	// 0..127 volume for EvA2DPVolume.
	Aux int32
	// Addr is the peer's BD address for the kinds that carry one
	// (EvSPPConnOpen, EvACLDisconnect), in Bluedroid's storage order:
	// most-significant octet first (Addr[0] is the MSB - the layout
	// AddrString renders, and the order a bumble peer's address compares
	// equal in).
	Addr [6]uint8
}

// addrString renders a 6-byte BD address in Bluedroid's storage order:
// the array already holds the most-significant octet in Addr[0], and the
// rendering walks it front to back ("aa:bb:cc:dd:ee:01" for a peer whose
// conventional address reads AA:BB:CC:DD:EE:01). Note this is the opposite
// of the ble package's rendering: NimBLE stores addresses least-significant
// first and flips on display, Bluedroid does not. Event.AddrString and
// SPPConn.AddrString share it.
func addrString(a [6]uint8) string {
	const hex = "0123456789abcdef"
	var out [17]byte
	for i := 0; i < 6; i++ {
		v := a[i]
		out[i*3] = hex[v>>4]
		out[i*3+1] = hex[v&0xF]
		if i < 5 {
			out[i*3+2] = ':'
		}
	}
	return string(out[:17])
}

// AddrString renders the event's peer address (see addrString).
func (e *Event) AddrString() string { return addrString(e.Addr) }

// ---- subscription machinery ----
//
// The llgo scheduler is cooperative (all goroutines multiplex one FreeRTOS
// task and only switch at blocking calls), so plain slices are race-free here
// as long as no function yields between reading a slice and acting on it -
// which Subscribe, Close and dispatch all honor.

type Subscription struct {
	kinds []EventKind
	ch    chan Event
	// closeHandle filters a connection's private subscription: when
	// non-zero, only EvSPPConnClose records of that handle are delivered.
	// It is set synchronously with Subscribe (no yield between), so no
	// dispatch can observe the unfiltered window.
	closeHandle uint32
}

// subscribers is the poller's fan-out list.
var subscribers []*Subscription

// pollerStarted gates the lazy start of the single poller goroutine.
var pollerStarted bool

// Subscribe starts the package's single poller goroutine (once, lazily) and
// returns a subscription that will receive a copy of every subsequent mailbox
// event matching one of kinds. An empty kinds list matches every kind. Events
// already sitting in the mailbox are not retroactively delivered - but they
// do wait there (the C rings are not drained until the poller starts), so
// subscribing before an operation is enough; subscribe before SPPListen if
// you want the profile's lifecycle events yourself.
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
// nothing. A non-positive timeout makes a single non-blocking check. The
// bound is measured against the glue's millisecond clock, so accuracy is one
// FreeRTOS tick (10 ms at the IDF default) rather than one whole quantum, and
// the wait never holds a C-level call: each iteration yields with Gosched,
// which also lets the poller run.
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
// which also feeds the idle tasks and the watchdog.
func pollEvents() {
	for {
		for n := 0; n < maxPollBatch; n++ {
			var ev Event
			if poll(&ev.Kind, &ev.Status, &ev.Handle, &ev.Aux, &ev.Addr[0]) != 1 {
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

// delivers reports whether ev goes to s. A connection's private subscription
// takes that connection's close records; a kind filter takes its kinds (an
// empty one takes every kind). The mailbox's global overflow report goes to
// everyone.
func (s *Subscription) delivers(ev *Event) bool {
	if ev.Kind == int32(EvOverflow) {
		return true
	}
	if s.closeHandle != 0 {
		return ev.Kind == int32(EvSPPConnClose) && ev.Handle == s.closeHandle
	}
	if len(s.kinds) == 0 {
		return true
	}
	for _, k := range s.kinds {
		if int32(k) == ev.Kind {
			return true
		}
	}
	return false
}

// ---- lifecycle ----

// Init brings the classic Bluetooth stack up: the controller in BTDM mode
// (BR/EDR + LE; the S31 dual-mode controller's only mode the SPP profile
// works under) and the Bluedroid host, and registers the glue's GAP callback
// (the ACL disconnect source). It blocks for the duration of the bring-up -
// esp_bluedroid_enable parks on an OS semaphore inside IDF, which yields the
// CPU - and returns as soon as GAP control calls are legal. A second call is
// a no-op returning nil.
//
// The SPP profile is NOT part of Init: SPPListen brings it up, so a program
// that only uses GAP pays neither the profile's memory nor its mailbox
// traffic.
func Init() error {
	if glueABI() != glueABIVersion {
		return ErrGlueABI
	}
	return errOf(initBT())
}

// Deinit tears the stack down in IDF's own order (SPP profile deinit - which
// closes any SPP connection still up - then Bluedroid disable/deinit, then
// controller disable/deinit). Every open SPPConn is invalidated: Read/Write/
// Close on them report esperr.ErrClosed afterwards. The rings are reset, so a
// following Init starts with a clean mailbox.
func Deinit() error {
	err := errOf(deinitBT())
	for _, c := range conns {
		c.closed = true
	}
	conns = conns[:0]
	return err
}

// Ready reports whether Init has completed and GAP control calls are legal.
func Ready() bool { return readyBT() == 1 }

// Backend names the BT host stack this firmware runs, for logs and
// diagnostics: "bluedroid" for the classic build, "none" when the build
// carries no host. The glue selects the backend; a program's own code never
// has to.
func Backend() string {
	var buf [16]byte
	n := backendName(&buf[0], int32(len(buf)))
	if n <= 0 {
		return "unknown"
	}
	return string(buf[:n])
}

// SetName sets the local device name (the string inquiry results and SDP
// browsers show, and what bumble lists as the device's name). The name is
// copied into glue-static storage before the async set lands, so the Go
// string can go out of scope immediately. Longer than 248 bytes truncates.
func SetName(name string) error {
	if !Ready() {
		return ErrNotReady
	}
	var buf [248]byte
	n := copy(buf[:], name)
	return errOf(setName(&buf[0], int32(n)))
}

// SetDiscoverable turns page and inquiry scan on (true: the device answers
// connection attempts and inquiry - what an SPP server needs) or off (false:
// existing connections stay up, no new one connects). IDF maps the two
// boolean states onto its connection/discovery mode pair inside the glue.
func SetDiscoverable(on bool) error {
	if !Ready() {
		return ErrNotReady
	}
	mode := int32(0)
	if on {
		mode = 1
	}
	return errOf(setScanMode(mode))
}

// ---- SPP server ----

// SPPOptions describes the SPP server SPPListen brings up. The zero value is
// a working server: SCN DefaultSCN, service name DefaultSPPName.
type SPPOptions struct {
	// SCN is the RFCOMM server channel to listen on (1..30). Zero picks
	// DefaultSCN.
	SCN int
	// Name is the SDP service name published for the server (what a
	// client's service browser shows, and what bumble's SPP discovery
	// matches). Longer than 64 bytes truncates.
	Name string
}

// SPPListen brings the SPP profile up and starts the server: enhanced-init
// (CB mode - received data arrives as BTC-task callbacks the glue copies into
// the data ring), then the SDP record + RFCOMM listener on the options'
// channel. Each stage waits for its BTC-task confirmation (EvSPPInit,
// EvSPPListen) with a SPPWaitMs bound, so the call returns after the server
// is actually listening, not merely accepted. A listen failure surfaces as
// the event's raw status; calling SPPListen twice returns the stack's
// ESP_ERR_INVALID_STATE.
func SPPListen(opts SPPOptions) error {
	if !Ready() {
		return ErrNotReady
	}
	scn := opts.SCN
	if scn <= 0 {
		scn = DefaultSCN
	}
	if scn > 30 {
		return errOf(0x102) // ESP_ERR_INVALID_ARG: SCN out of RFCOMM range
	}
	name := opts.Name
	if name == "" {
		name = DefaultSPPName
	}
	var nameBuf [64]byte
	n := copy(nameBuf[:], name)

	sub := Subscribe(EvSPPInit, EvSPPListen)
	defer sub.Close()
	if err := errOf(sppInit()); err != nil {
		return err
	}
	ev, err := nextBounded(sub, SPPWaitMs)
	if err != nil {
		return err
	}
	if ev.Kind != int32(EvSPPInit) {
		return errOf(0x103) // ESP_ERR_INVALID_STATE: unexpected event shape
	}
	if ev.Status != 0 {
		return errOf(ev.Status)
	}
	if err := errOf(sppStartSrv(int32(scn), &nameBuf[0], int32(n))); err != nil {
		return err
	}
	ev, err = nextBounded(sub, SPPWaitMs)
	if err != nil {
		return err
	}
	if ev.Kind != int32(EvSPPListen) {
		return errOf(0x103)
	}
	if ev.Status != 0 {
		return errOf(ev.Status)
	}
	return nil
}

// SPPAccept waits for a peer to connect to the SPP server and returns the
// connection. timeoutMs bounds the wait (wall clock; non-positive means one
// pollQuantumMs quantum). Only one Accept may wait at a time - a concurrent
// one reports ErrBusy - but the MVP serves one connection anyway, and an
// accepted server keeps listening: after a connection closes, the next
// SPPAccept catches the following peer.
func SPPAccept(timeoutMs int32) (*SPPConn, error) {
	if acceptBusy {
		return nil, ErrBusy
	}
	if !Ready() {
		return nil, ErrNotReady
	}
	acceptBusy = true
	sub := Subscribe(EvSPPConnOpen)
	ev, err := nextBounded(sub, timeoutMs)
	sub.Close() // the open event is consumed; keep the fan-out list lean
	acceptBusy = false
	if err != nil {
		return nil, err
	}
	c := &SPPConn{
		handle:   ev.Handle,
		addr:     ev.Addr,
		closeSub: Subscribe(EvSPPConnClose),
	}
	c.closeSub.closeHandle = ev.Handle
	conns = append(conns, c)
	return c, nil
}

// acceptBusy guards a single waiting SPPAccept (see above).
var acceptBusy bool

// conns tracks every open SPPAccept result so Deinit can invalidate them
// all; removeConn keeps it to the live set (see SPPConn.Close).
var conns []*SPPConn

// removeConn drops c from the tracked set (called once per close).
func removeConn(c *SPPConn) {
	for i, e := range conns {
		if e == c {
			last := len(conns) - 1
			conns[i] = conns[last]
			conns[last] = nil
			conns = conns[:last]
			return
		}
	}
}

// SPPConn is one SPP connection to a peer. It is a single-goroutine handle:
// exactly one goroutine calls its Read (the same contract as a Subscription),
// because Read consumes both the data ring and the connection's private
// close-event subscription.
type SPPConn struct {
	handle   uint32
	addr     [6]uint8
	closeSub *Subscription
	remote   bool // peer initiated the close (Read reports esperr.ErrEOF)
	closed   bool // invalid (locally closed, or stack gone)
}

// Handle returns the SPP connection handle (the value EvSPPConnOpen and
// EvSPPConnClose carry).
func (c *SPPConn) Handle() uint32 { return c.handle }

// Addr returns the peer's BD address in Bluedroid's storage order:
// most-significant octet first (Addr[0] is the MSB; see AddrString).
func (c *SPPConn) Addr() [6]uint8 { return c.addr }

// AddrString renders the peer's address (see addrString).
func (c *SPPConn) AddrString() string { return addrString(c.addr) }

// bound turns a call's timeout argument into the wall-clock bound the loops
// use: non-positive means one pollQuantumMs quantum.
func bound(timeoutMs int32) int32 {
	if timeoutMs > 0 {
		return timeoutMs
	}
	return pollQuantumMs
}

// boundErr is the error a bound expiry reports: ErrTimeout for a real
// deadline, the family's poll-style ErrWouldBlock for the quantum of a
// non-positive one.
func boundErr(timeoutMs int32) error {
	if timeoutMs > 0 {
		return ErrTimeout
	}
	return esperr.ErrWouldBlock
}

// nextBounded is Subscription.Next with the package's timeout convention:
// non-positive means one quantum.
func nextBounded(s *Subscription, timeoutMs int32) (Event, error) {
	if timeoutMs <= 0 {
		timeoutMs = pollQuantumMs
	}
	return s.Next(timeoutMs)
}

// Read pulls the next SPP data record into buf and returns its length. The
// glue data ring holds whole RFCOMM frames (at most MaxMTU on-air bytes, up
// to maxRecord per record); a record longer than buf is truncated and the
// remainder is dropped - pass a MaxMTU-sized buffer to never lose bytes. When
// no record is queued, Read waits: timeoutMs bounds the wait in wall-clock
// time, a non-positive timeout means one pollQuantumMs quantum. A stall
// reports ErrTimeout (or esperr.ErrWouldBlock for the quantum form); the
// peer's close surfaces as esperr.ErrEOF once the ring is drained; a locally
// closed (or stack-invalidated) connection reports esperr.ErrClosed.
func (c *SPPConn) Read(buf []byte, timeoutMs int32) (int, error) {
	if c.closed {
		return 0, esperr.ErrClosed
	}
	if len(buf) == 0 {
		return 0, nil
	}
	n := len(buf)
	if n > maxRecord {
		n = maxRecord
	}
	deadline := int32(nowMs()) + bound(timeoutMs)
	for {
		got := sppRead(c.handle, &buf[0], int32(n))
		if got > 0 {
			return int(got), nil
		}
		if got < 0 { // handle no longer known: stack went down under us
			c.closed = true
			return 0, esperr.ErrClosed
		}
		if !c.remote {
			if ev, ok := c.closeSub.pop(); ok {
				c.remote = ev.Aux != 0
			}
		}
		if c.remote { // peer closed: drain-first already done above
			return 0, esperr.ErrEOF
		}
		if int32(nowMs())-deadline >= 0 {
			return 0, boundErr(timeoutMs)
		}
		gosched()
		vTaskDelay(1)
	}
}

// Write queues data to the peer, in MaxMTU-sized chunks, and returns the
// number of bytes accepted. v6.3's esp_spp_write copies each chunk into the
// profile's TX ringbuffer synchronously, so the caller's memory is free as
// soon as the chunk is counted (accepted does not mean on-air: the stack
// drains to the peer at RFCOMM speed afterwards). When the TX buffer is full
// the write reports ESP_ERR_NO_MEM and Write waits for the drain edge
// (EvSPPCong) before retrying, bounded by timeoutMs like Read - a stalled
// peer therefore surfaces as a partial return plus ErrTimeout instead of
// blocking forever. A connection that closed under the write reports
// esperr.ErrClosed after the bytes that did land.
func (c *SPPConn) Write(data []byte, timeoutMs int32) (int, error) {
	if c.closed {
		return 0, esperr.ErrClosed
	}
	if len(data) == 0 {
		return 0, nil
	}
	cong := Subscribe(EvSPPCong)
	defer cong.Close()
	deadline := int32(nowMs()) + bound(timeoutMs)
	total := 0
	for total < len(data) {
		n := len(data) - total
		if n > MaxMTU {
			n = MaxMTU
		}
		rc := sppWrite(c.handle, &data[total], int32(n))
		if rc == 0 { // accepted into the TX ringbuffer
			total += n
			continue
		}
		if rc == 0x101 { // ESP_ERR_NO_MEM: TX buffer full, wait the drain edge
			if !c.waitCong(cong, deadline) {
				return total, boundErr(timeoutMs)
			}
			continue
		}
		if rc == 0x103 { // ESP_ERR_INVALID_STATE: connection gone
			c.closed = true
			return total, esperr.ErrClosed
		}
		return total, errOf(rc)
	}
	return total, nil
}

// waitCong waits for the subscription's next drain edge until the deadline
// passes (stale edges delivered between calls only cause an extra harmless
// retry).
func (c *SPPConn) waitCong(cong *Subscription, deadline int32) bool {
	for {
		if ev, ok := cong.pop(); ok && ev.Kind == int32(EvSPPCong) {
			return true
		}
		if int32(nowMs())-deadline >= 0 {
			return false
		}
		gosched()
		vTaskDelay(1)
	}
}

// Close disconnects the peer and waits (CloseWaitMs) for the CLOSE
// confirmation, then drains whatever unread records of this connection are
// still parked in the glue data ring (records of a kicked client would
// otherwise squat on their slots forever and starve the next peer once the
// ring fills). Afterwards every method reports esperr.ErrClosed. Closing an
// already-closed - or remotely closed, whose handle is already gone -
// connection reports esperr.ErrClosed too.
func (c *SPPConn) Close() error {
	if c.closed {
		// Also removes a conn that another path (a failed Read, Deinit's
		// sweep) already closed; removeConn is a no-op when absent.
		removeConn(c)
		return esperr.ErrClosed
	}
	if c.remote {
		// The peer's close already tore the RFCOMM link down and Read (or
		// the close-event stream) consumed its CLOSE record; a disconnect
		// call would just park this goroutine waiting for a confirmation
		// that never comes.
		c.closed = true
		removeConn(c)
		c.drain()
		return esperr.ErrClosed
	}
	if err := errOf(sppDisconnect(c.handle)); err != nil {
		c.closed = true
		removeConn(c)
		return err
	}
	c.closed = true
	deadline := int32(nowMs()) + CloseWaitMs
	for {
		if ev, ok := c.closeSub.pop(); ok {
			if ev.Aux == 0 { // local-initiated close confirmed
				removeConn(c)
				c.drain()
				return nil
			}
			// The peer closed at the same moment; the disconnect call
			// already reported success, so treat this as the answer.
			removeConn(c)
			c.drain()
			return nil
		}
		if int32(nowMs())-deadline >= 0 {
			removeConn(c)
			c.drain()
			return ErrTimeout
		}
		gosched()
		vTaskDelay(1)
	}
}

// drain pops every queued record of this connection out of the glue data
// ring. llgo_bt_spp_read only ever returns records of this handle, so the
// loop cannot starve another connection's data; it is bounded by the ring's
// slot count, and one 1-byte pull consumes a whole record (the C side
// releases the slot regardless of the copy length).
func (c *SPPConn) drain() {
	var sink [1]byte
	for i := 0; i < dataRingSlots; i++ {
		if sppRead(c.handle, &sink[0], 1) <= 0 {
			return
		}
	}
}

// ---- A2DP sink (audio profile) ----
//
// The audio path never enters Go. The stack decodes SBC to PCM (S16 LE) and
// the glue pipes it into an I2S standard-mode TX channel on dedicated pins:
// BTC data callback -> static PCM ring -> glue pump task -> DMA. Go's surface
// is the control plane only - bring-up with pin assignment, and the counters
// that make audio loss visible (the realtime path drops whole chunks rather
// than ever blocking the BT tasks; see A2DPStats).

// A2DPConfig describes the analog audio endpoint A2DPEnable creates.
type A2DPConfig struct {
	// BclkPin, WsPin, DoutPin are the GPIO numbers of the I2S standard-mode
	// TX bit clock, word select and data-out pins (the DAC/codec side).
	// MCLK is not driven; DIN is unused.
	BclkPin int
	WsPin   int
	DoutPin int
	// DmaDescNum and DmaFrameNum size the TX DMA ring (descriptors x frames
	// of one stereo S16 sample) - the jitter absorber between the BT clock
	// feeding PCM and the local audio clock consuming it. Zero picks the
	// driver defaults (6 x 240 frames ≈ 32 ms at 44.1 kHz); a serious
	// loudspeaker path wants tens of milliseconds more, e.g. 12 x 511
	// (≈ 139 ms). The static PCM ring on top adds ~186 ms at 44.1 kHz.
	DmaDescNum  int
	DmaFrameNum int
}

// A2DPEnable brings the A2DP sink profile up and wires its decoded-PCM
// output to the I2S pins in cfg: profile init, the data callback into the
// glue's audio path, AVRCP TG/CT init (volume commands), and the audio pump
// task. It must be called after Init (the stack bring-up) and may run next
// to an SPP server - the two profiles share one stack, one ctrl ring and
// one poller. Playback begins when a phone connects and pushes a stream;
// the negotiated codec config arrives as EvA2DPAudioCfg and the glue
// re-clocks I2S on its own. A2DPEnable on a firmware built without
// CONFIG_BT_A2DP_ENABLE reports ESP_ERR_NOT_SUPPORTED.
func A2DPEnable(cfg A2DPConfig) error {
	if !Ready() {
		return ErrNotReady
	}
	return errOf(a2dpEnable(int32(cfg.BclkPin), int32(cfg.WsPin),
		int32(cfg.DoutPin), int32(cfg.DmaDescNum), int32(cfg.DmaFrameNum)))
}

// A2DPDisable tears the profile and its audio path down (closes any A2DP
// connection, stops the pump, deletes the I2S channel, zeroes the counters).
func A2DPDisable() error {
	if !Ready() {
		return ErrNotReady
	}
	return errOf(a2dpDisable())
}

// A2DPSetVolume pushes an absolute volume (0..127) to the connected phone's
// AVRCP controller. Fire-and-forget: the value the peer confirms arrives
// later as an EvA2DPVolume record. Reports ErrNotReady before Init and
// ESP_ERR_INVALID_STATE when the profile is not enabled (A2DPEnable has not
// run) or there is no A2DP connection to address.
func A2DPSetVolume(volume uint8) error {
	if !Ready() {
		return ErrNotReady
	}
	return errOf(a2dpVolumeSet(int32(volume)))
}

// A2DPStats reports the audio path's byte counters since A2DPEnable (or the
// last A2DPDisable): total PCM the stack handed over, written the bytes the
// pump fed into I2S, and dropped the bytes lost to a full ring. The three
// values are snapshots taken at slightly different moments (two BTC-side
// counters and one pump-side counter), so total does not have to equal
// written + dropped for any single read - small skews are the read's own
// race, not lost audio; the same read races the counter resets at
// A2DPDisable, which can shift a value by one in-flight chunk. On a healthy
// link written tracks total within the ring's water depth; a steadily
// growing dropped means the source outruns the speaker path (or the pump
// starved). Treat the numbers as flow telemetry, not as an audio meter.
func A2DPStats() (total, written, dropped uint32) {
	a2dpStats(&total, &written, &dropped)
	return total, written, dropped
}

// ---- error rendering ----

// ErrName names the codes this pipeline can hit, in their IDF vocabulary:
// the esp_err space the glue's synchronous calls return (ESP_ERR_*), the
// esp_bt_status_t space SPP lifecycle events and ACL disconnect reasons
// carry (0..21 and the 0x100+ HCI range), and the package's own 0x76xx
// segment.
//
// The domains overlap by value, and the table reads every code as esp_err
// first, then esp_bt_status_t. Both overlaps are disclosed here:
//
//   - 0x101..0x107 are esp_err codes (ESP_ERR_NO_MEM .. ESP_ERR_TIMEOUT);
//     read as esp_bt_status_t they would name unrelated statuses, and read as
//     the HCI range they would name page-timeout (0x104) and friends. An ACL
//     disconnect reason in that window therefore renders with the esp_err
//     name - the raw code stays inspectable on the Event for a real
//     disambiguation.
//   - 1..8 are also the esp_spp_status_t values the SPP events report. The
//     v6.3 enum is SUCCESS=0, FAILURE=1, BUSY=2, NO_DATA=3, NO_RESOURCE=4,
//     NEED_INIT=5, NEED_DEINIT=6, NO_CONNECTION=7, NO_SERVER=8 (verified
//     against esp_spp_api.h); it agrees with esp_bt_status_t on 1 (failure)
//     and 2 (busy) and diverges from 3 up, where the table names the broader
//     esp_bt_status_t reading (ACL reasons only exist there).
func ErrName(code int32) string {
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
	case 0x104:
		return "ESP_ERR_INVALID_SIZE"
	case 0x105:
		return "ESP_ERR_NOT_FOUND"
	case 0x106:
		return "ESP_ERR_NOT_SUPPORTED"
	case 0x107:
		return "ESP_ERR_TIMEOUT"

	// esp_bt_status_t (SPP event statuses, ACL disconnect reasons)
	case 1:
		return "ESP_BT_STATUS_FAIL"
	case 2:
		return "ESP_BT_STATUS_NOT_READY"
	case 3:
		return "ESP_BT_STATUS_NOMEM"
	case 4:
		return "ESP_BT_STATUS_BUSY"
	case 5:
		return "ESP_BT_STATUS_DONE"
	case 6:
		return "ESP_BT_STATUS_UNSUPPORTED"
	case 7:
		return "ESP_BT_STATUS_PARM_INVALID"
	case 8:
		return "ESP_BT_STATUS_UNHANDLED"
	case 9:
		return "ESP_BT_STATUS_AUTH_FAILURE"
	case 10:
		return "ESP_BT_STATUS_RMT_DEV_DOWN"
	case 11:
		return "ESP_BT_STATUS_AUTH_REJECTED"
	case 12:
		return "ESP_BT_STATUS_INVALID_STATIC_RAND_ADDR"
	case 13:
		return "ESP_BT_STATUS_PENDING"
	case 14:
		return "ESP_BT_STATUS_UNACCEPT_CONN_INTERVAL"
	case 15:
		return "ESP_BT_STATUS_PARAM_OUT_OF_RANGE"
	case 16:
		return "ESP_BT_STATUS_TIMEOUT"
	case 17:
		return "ESP_BT_STATUS_PEER_LE_DATA_LEN_UNSUPPORTED"
	case 18:
		return "ESP_BT_STATUS_CONTROL_LE_DATA_LEN_UNSUPPORTED"
	case 19:
		return "ESP_BT_STATUS_ERR_ILLEGAL_PARAMETER_FMT"
	case 20:
		return "ESP_BT_STATUS_MEMORY_FULL"
	case 21:
		return "ESP_BT_STATUS_EIR_TOO_LARGE"

	// The HCI range (esp_bt_status_t = 0x100 + HCI error code): the
	// disconnect reasons a real link loss reports.
	case 0x100:
		return "ESP_BT_STATUS_HCI_SUCCESS"
	case 0x108:
		return "ESP_BT_STATUS_HCI_CONNECTION_TOUT"
	case 0x109:
		return "ESP_BT_STATUS_HCI_MAX_NUM_OF_CONNECTIONS"
	case 0x113:
		return "ESP_BT_STATUS_HCI_PEER_USER"
	case 0x116:
		return "ESP_BT_STATUS_HCI_CONN_CAUSE_LOCAL_HOST"

	// This package's internal segment.
	case codeGlueABI:
		return "BT_ERR_GLUE_ABI"
	case codeNotReady:
		return "BT_ERR_NOT_READY"
	case codeBusy:
		return "BT_ERR_BUSY"
	case codeUnsupport:
		return "BT_ERR_UNSUPPORTED"
	}
	return "esp error 0x" + hex6(uint32(code))
}

// hex6 renders a code as six hex digits, the family's fallback format.
func hex6(v uint32) string {
	const hex = "0123456789abcdef"
	var out [6]byte
	for i := 5; i >= 0; i-- {
		out[i] = hex[v&0xF]
		v >>= 4
	}
	return string(out[:])
}
