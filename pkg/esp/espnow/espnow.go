// Package espnow binds llgo-compiled Go code to the ESP-IDF esp_now stack -
// Espressif's connectionless, low-latency vendor protocol carried in WiFi
// vendor-specific action frames. ESP-NOW is not IP: a frame goes straight to
// a peer MAC (or the broadcast MAC) with a payload of at most MaxDataLen
// bytes, no association, no socket, no stack - typically tens of microseconds
// shorter a round trip than a UDP exchange over the same radio.
//
// # Layout
//
// The pkg/esp three-file template (see pkg/esp/README.md; the glue lives in a
// subdirectory because the go tool rejects C sources inside a Go package):
//
//	espnow.go          public API, no build tag
//	espnow_idf.go      //go:build llgo.idf - bodyless //go:linkname externs
//	espnow_stub.go     //go:build !llgo.idf - host-build stubs (go vet/gopls)
//	glue/espnow_glue.c ESP-IDF C glue template, compiled only under
//	                   ESP_PLATFORM; copy into the project's main component
//	                   (WHOLE_ARCHIVE required - see pkg/esp/README.md)
//
// On the device (llgo build -target esp32s31-idf) the package talks to IDF
// through the glue; on any other build the stubs return errUnsupported, so
// `go build` / `go vet` / gopls stay clean.
//
// # Initialization order: ESP-NOW rides a started WiFi driver
//
// esp_now lives inside the esp_wifi library and requires the driver to be
// initialized AND started (station or soft-AP mode, a connection is NOT
// required - ESP-NOW works from the unassociated state and reaches peers on
// the same channel). This package deliberately does not touch esp_wifi's
// lifecycle - pkg/esp/wifi owns the radio. The program brings the radio up
// first and calls Init afterwards:
//
//	err := wifi.Setup()      // NVS + netif + event loop + esp_wifi_init
//	wifi.SetModeSTA()        // or wifi.SetModeAP() / SetModeAPSTA()
//	wifi.Start()
//	if err := espnow.Init(); err != nil { ... }
//
// esp_now_init before the driver runs reports ESP_ERR_ESPNOW_INTERNAL
// (0x306a); before esp_wifi_init it fails outright. Init after Deinit is
// legal (esp_now_init re-initializes), and the demo
// (_demo/esp-idf-espnow) shows the full sequence as a runnable probe.
//
// # Events: static ring, Go-side pump
//
// Both driver callbacks (a frame received, a send confirmed) fire on the
// WiFi task - outside the single-task llgo fiber scheduler, where a call
// into Go would corrupt scheduler state, and where an allocation is equally
// forbidden: llgo's --wrap redirects malloc/free to the lock-free tinygogc
// allocator, which only the llgo task may enter. The glue therefore only
// memcpy's the callback's bytes into a static SPSC ring (no malloc, no
// callbacks into Go, C11 release/acquire index publication - the exact
// mailbox discipline of wifi_glue.c), and Go polls the ring with the same
// bounded-quantum pump the wifi package uses: drain what the ring holds,
// fan records out, hand the scheduler away, then rest one FreeRTOS quantum
// in vTaskDelay (which also feeds the idle tasks and the watchdog).
//
// The pump goroutine starts lazily when Init succeeds and never stops,
// like the wifi package's subscriber poller. Nothing runs in the background
// on a host build: the stubs never start the pump (vTaskDelay is a no-op
// there, so its loop would spin at full CPU).
//
// Ring capacity: recvRingSlots events, ~recvRingSlots*276 bytes of static
// RAM. A burst that outruns the pump drops the newest record and counts it;
// Dropped() reports the count. Because ESP-NOW is datagram traffic, a drop
// is the protocol's own behavior under load, not an error - design the
// application's retransmit around it (or raise recvRingSlots in the glue
// copy your firmware links).
//
// Event kinds (this package's own numbering, starting at 1 like every
// pkg/esp event table; never aliased with wifi's or ble's kinds - the tables
// belong to their packages):
//
//	1  EvRecv      a frame arrived: source MAC + payload, delivered to the
//	               handler SetRecvHandler registered
//	2  EvSendDone  a send confirmed: destination MAC + delivery status,
//	               resolving the SendResult Wait returned
//	-1 EvOverflow  the ring dropped records (the pump turns this into the
//	               Dropped() counter; there is no callback for it)
//
// # Sending: synchronous enqueue, asynchronous confirmation
//
// esp_now_send is synchronous and only queues the frame; the verdict
// (ESP_NOW_SEND_SUCCESS / ESP_NOW_SEND_FAIL) arrives later on the send
// callback. Send mirrors that split: it validates and hands the frame to the
// driver (the returned error is the driver's synchronous verdict - NOT_INIT,
// ARG, NOT_FOUND for an unregistered peer MAC, NO_MEM when the driver's own
// send queue is full, IF or CHAN for a peer whose interface/channel does not
// match the radio), and the confirmation arrives through the *SendResult it
// returns:
//
//	res, err := espnow.Send(peer, []byte("ping"))
//	if err != nil { ... }              // the driver refused the send
//	status, err := res.Wait(500)       // bounded wait for the callback
//	// status == espnow.SendSuccess once the MAC layer confirms delivery;
//	// broadcast (no ACK expected) reports success once the frame is out.
//
// The send callback carries no sequence id - only the destination MAC - so
// results resolve oldest-first per MAC (ESP-NOW serializes a peer's frames
// through the driver, so statuses arrive in send order). Resolve every
// SendResult with a bounded Wait: a result whose callback never fires (the
// driver stopped mid-flight) keeps its place at the head of that MAC's
// queue, and the next confirmation resolves it first; a Wait that timed out
// leaves the buffer channel in place, so the queue keeps flowing and never
// mis-delivers into a Wait that already returned.
//
// Broadcast(data) is Send(BroadcastMAC, data): the all-ones MAC every
// ESP-NOW device receives without being added as a peer. The broadcast peer
// itself must still be added to the peer list before the driver accepts a
// broadcast send (AddPeer with the zero PeerConfig below).
//
// # Receiving
//
//	func handler(from [6]uint8, data []byte) { ... }
//	espnow.SetRecvHandler(handler)
//
// The pump invokes the handler once per received frame, with data freshly
// allocated per delivery (the callback may retain it). A frame that arrives
// while the handler is nil is drained and lost - register the handler before
// traffic flows, the same rule as subscribing before a wifi scan. Frames
// are bounded by MaxDataLen (250, protocol v1.0); a v2.0 sender's longer
// frames are truncated by the driver at 250 (v1 receive path).
//
// # Peers and encryption
//
// The peer list is the driver's (20 entries at most): AddPeer registers a
// MAC + interface (+ channel, + optional encryption), DeletePeer removes,
// PeerIsKnown queries, PeerCount counts. The broadcast peer needs nothing
// but its MAC. An encrypted peer carries a 16-byte local master key (LMK)
// set at AddPeer time or later through SetLMK; the whole encryption domain
// unlocks with SetPMK, the 16-byte primary master key that protects the
// LMKs on the air. At most ESP_NOW_MAX_ENCRYPT_PEER_NUM peers may be
// encrypted (the Kconfig CONFIG_ESP_WIFI_ESPNOW_MAX_ENCRYPT_NUM ceiling -
// the driver, not this package, is the authority; a surplus reports
// ESP_ERR_ESPNOW_FULL). Key agreement is out of scope: both sides must be
// provisioned with the same PMK/LMKs by the deployer.
//
// # Not exposed (driver has it; add here when needed)
//
// esp_now_set_peer_rate_config (per-peer PHY rate), the esp-now v2.0 frame
// format (esp_now_get_version reports which the firmware speaks; Send stays
// at the v1.0 250-byte ceiling so every generation receives), the channel
// switching / remain-on-channel pair (esp_now_switch_channel_tx /
// esp_now_remain_on_channel), wake window tuning, and the user OUI override.
//
// # Emulator fidelity
//
// esp-emu (1.2.0) implements the esp_now API surface: init/deinit, callback
// registration, peer management and broadcast sends all return their
// documented codes, and the send callback fires. What an emulator cannot
// do is OTA frames between instances - two esp-emu processes do not hear
// each other, and there is no virtual ESP-NOW peer - so the receive path
// and true delivery semantics are hardware items (see
// _demo/esp-idf-espnow/README.md for the measured probe and the checklist).
package espnow

import "github.com/xgo-dev/llgo/pkg/esp/esperr"

// errOf wraps a raw esp_err code as the error value this package returns
// (nil for ESP_OK), named through ErrName so a printed error reads like the
// driver's own name for the code.
func errOf(code int32) error {
	if code == 0 {
		return nil
	}
	return &esperr.Error{Code: code, Name: ErrName(code)}
}

// errUnsupported is what every host stub below returns: it lies between the
// real esp_err ranges so it can never be mistaken for one (the shared
// pkg/esp stub code; esperr 0x7F00).
const errUnsupported = 0x7F00

// ErrGlueABI reports a glue/binding contract mismatch: the espnow_glue.c a
// firmware links was written for a different version of this package. The C
// symbols keep their names across versions, so a stale copy would not fail
// the link - it would silently answer with zeros or drop records. Init
// compares the glue's version with glueABIVersion and refuses to run on a
// mismatch, so the failure arrives as one named error instead of as wrong
// behavior.
var ErrGlueABI = &esperr.Error{Code: 0x7401, Name: ErrName(0x7401)}

// ErrTimeout reports a bounded wait that found nothing: ESP_ERR_TIMEOUT
// (0x107), the esp_err family's own code for the condition (the same value
// wifi.ErrTimeout carries; errors.Is matches them both).
var ErrTimeout = &esperr.Error{Code: 0x107, Name: ErrName(0x107)}

// Package-level bounds, straight from esp_now.h. They are pinned by the
// host tests: a driver constant that drifts must not silently change what
// the Go side validates.
const (
	// MaxDataLen is ESP_NOW_MAX_DATA_LEN (250): the v1.0 frame payload
	// ceiling, the size the glue's ring slots hold.
	MaxDataLen = 250
	// MaxPeers is ESP_NOW_MAX_TOTAL_PEER_NUM (20).
	MaxPeers = 20
	// KeyLen is ESP_NOW_KEY_LEN (16): the PMK and every LMK.
	KeyLen = 16
)

// BroadcastMAC is the all-ones destination: every ESP-NOW device on the
// channel receives the frame without being added as a peer. Sending to it
// still requires the broadcast peer to be added (AddPeer with the zero
// PeerConfig beyond the MAC).
var BroadcastMAC = [6]uint8{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

// IfIdx names the WiFi interface a peer lives on. The values are IDF's
// wifi_interface_t numbers, passed through unchanged.
type IfIdx int32

const (
	IfSTA IfIdx = 0 // WIFI_IF_STA: the station interface (the usual choice)
	IfAP  IfIdx = 1 // WIFI_IF_AP: the soft-AP interface
)

// Event kinds delivered by the C ring (see on_recv/on_send in
// espnow_glue.c). This package's own table, starting at 1; EvOverflow keeps
// the family-wide negative report value.
const (
	// EvNone is the zero-value sentinel, never delivered.
	EvNone = 0
	// EvRecv marks a received frame: MAC carries the source, Aux the
	// frame's RSSI, Data the payload.
	EvRecv = 1
	// EvSendDone marks a send confirmation: MAC carries the destination,
	// Aux the esp_now_send_status_t (0 success, 1 fail).
	EvSendDone = 2
	// EvOverflow is the family-wide negative overflow report; the pump
	// turns it into the Dropped() counter rather than a callback.
	EvOverflow = -1
)

// recvRingSlots is the C ring's capacity: how many received frames / send
// confirmations fit while the pump is between quanta, and the static RAM the
// glue holds (recvRingSlots * sizeof(en_slot_t), ~276 bytes per slot).
const recvRingSlots = 16

// SendStatus is the send callback's verdict: whether the frame was
// delivered (an ACK for unicast; for broadcast, the driver having written
// the frame - nothing ACKs a broadcast).
type SendStatus int32

const (
	SendSuccess SendStatus = 0 // ESP_NOW_SEND_SUCCESS
	SendFail    SendStatus = 1 // ESP_NOW_SEND_FAIL
)

// OK reports whether the status says delivered.
func (s SendStatus) OK() bool { return s == SendSuccess }

// String names the status ("success" / "fail"), for logs and probe output.
func (s SendStatus) String() string {
	if s == SendSuccess {
		return "success"
	}
	return "fail"
}

// SendResult is the asynchronous half of a Send: the driver accepted the
// frame, and this value will carry its delivery status once the send
// callback fires. Wait resolves it. The pump (not the caller) delivers
// into the result - results are oldest-first per destination MAC, the
// pairing the callback's MAC-only record allows (see the Send docs).
type SendResult struct {
	Peer [6]uint8
	ch   chan SendStatus // capacity 1, written once by the pump
}

// Wait blocks up to timeoutMs for the send's confirmation. It returns
// (status, nil) once the callback delivered, and (SendFail, ErrTimeout) when
// the bound passed with nothing - a non-positive timeout makes a single
// non-blocking check. When error is non-nil the status is undefined: always
// branch on the error first.
//
// The bound is wall clock, one 100 ms quantum at a time (the pump's own
// cadence), so it can overshoot by up to a quantum; on a host build every
// quantum is a no-op and the loop is just a bounded spin.
func (r *SendResult) Wait(timeoutMs int32) (SendStatus, error) {
	if r == nil || r.ch == nil {
		return SendFail, ErrTimeout
	}
	for ms := int32(0); ms < timeoutMs; ms += pollQuantumMs {
		select {
		case s := <-r.ch:
			return s, nil
		default:
		}
		gosched()
		vTaskDelay(pollQuantumMs / tickMs)
	}
	select {
	case s := <-r.ch:
		return s, nil
	default:
		return SendFail, ErrTimeout
	}
}

// pollQuantumMs and tickMs are the pump's pacing constants, shared with the
// wifi package: one vTaskDelay(10) = 100 ms quantum at the IDF's default
// 100 Hz tick. Every wait in this package rests in whole quanta, so the
// FreeRTOS task yields and the idle tasks and watchdog stay fed.
const (
	pollQuantumMs = 100
	tickMs        = 10
)

// glueABIVersion is the contract version this package speaks; the C glue
// reports the version it implements (llgo_espnow_glue_abi). It changes
// whenever a Go-facing signature or field set does.
const glueABIVersion = 1

// handler owns the receive callback the pump dispatches to. Set by
// SetRecvHandler, read by the pump; under the cooperative scheduler the
// write is visible to the pump by its next dispatch pass.
var handler func(from [6]uint8, data []byte)

// pending indexes the in-flight SendResults by destination MAC,
// oldest-first. Send appends (the calling goroutine), the pump pops and
// delivers. The llgo scheduler is cooperative - all goroutines multiplex
// one FreeRTOS task and only switch at blocking calls - so a plain map and
// slices are race-free here as long as no function yields between reading a
// key and acting on it, which Send and resolveSend both honor.
var pending = map[[6]uint8][]*SendResult{}

// pumpRunning gates the lazy start of the single pump goroutine. It starts
// only on the device path where Init succeeded; a host build never starts
// it (the stub vTaskDelay is a no-op, so the pump would busy-spin).
var pumpRunning bool

// dropped counts the ring records the C side dropped because the ring was
// full. Frames drained while no receive handler was registered are not
// counted - only genuine ring overflow is. The pump turns each EvOverflow
// report into an increment.
var dropped uint32

// Dropped reports how many ring records were dropped because the ring was
// full - the honest loss counter behind EvOverflow. A nonzero value means
// the pump could not keep up with the frame rate (or sat starved behind a
// long C-level wait elsewhere in the program); raise recvRingSlots in the
// firmware's glue copy, or slow the sender.
func Dropped() uint32 { return dropped }

// SetRecvHandler registers the receive callback. It may be set before or
// after Init; frames that arrive while it is nil are drained and lost, so
// set it before traffic flows. The pump invokes it sequentially - one frame
// at a time, never concurrently - and a slow handler delays the following
// frames' dispatch (the ring keeps absorbing; a full ring drops and counts).
//
// data is a fresh copy per delivery, valid after the handler returns. from
// is the sender's MAC.
func SetRecvHandler(fn func(from [6]uint8, data []byte)) {
	handler = fn
}

// Init brings ESP-NOW up: glue ABI check, esp_now_init, and the glue's
// callback registration (recv + send - both static C functions copying into
// the ring), then starts the pump goroutine. The WiFi driver must already
// be initialized and started (see the package docs' ordering contract).
// Errors: ErrGlueABI (a stale glue template), then the driver's
// (ESP_ERR_ESPNOW_INTERNAL when the WiFi task state is wrong, e.g. the
// driver never started).
func Init() error {
	if v := glueABI(); v != glueABIVersion {
		return ErrGlueABI
	}
	err := errOf(initNow())
	if err != nil {
		return err
	}
	if !pumpRunning {
		pumpRunning = true
		go pumpEvents()
	}
	return nil
}

// Deinit tears ESP-NOW down (esp_now_deinit): the driver unregisters both
// callbacks and frees its per-peer state. Results already in flight do not
// resolve; the pump keeps running (it drains whatever the ring still holds
// and then idles, like the wifi package's poller). A later Init restarts
// cleanly.
func Deinit() error { return errOf(deinitNow()) }

// SetChannel moves the radio to a primary 2.4 GHz channel (1..14), the way
// two ESP-NOW nodes find each other without a common AP: both sides call it
// with the same channel while unconnected (a connected station refuses -
// ESP_ERR_WIFI_CONN - because association owns the channel; a soft-AP
// interface takes calls on its configured channel instead). It is
// esp_wifi_set_channel with the secondary channel pinned to NONE.
func SetChannel(ch int) error {
	if ch < 1 || ch > 14 {
		return errOf(0x102) // ESP_ERR_INVALID_ARG: 2.4 GHz only
	}
	return errOf(setChannel(int32(ch)))
}

// PeerConfig describes one peer for AddPeer. The zero value is a peer that
// inherits the radio's current state - fill MAC, and the rest only when it
// differs.
type PeerConfig struct {
	// MAC is the peer's WiFi MAC (its station or soft-AP address, 6 bytes).
	MAC [6]uint8
	// Channel is the channel the peer listens on: 0 inherits the channel
	// this radio is currently on (the usual choice after SetChannel);
	// 1..14 pins the peer to a channel. A mismatch is reported at Send
	// time as ESP_ERR_ESPNOW_CHAN, not here.
	Channel uint8
	// If is the interface the peer lives on: IfSTA (the zero value) or
	// IfAP. A mismatch is reported at Send time as ESP_ERR_ESPNOW_IF.
	If IfIdx
	// Encrypt turns on link encryption for this peer; it requires LMK
	// (and the other side to provision the same keys) and consumes one
	// of the ESP_NOW_MAX_ENCRYPT_PEER_NUM encrypted-peer slots.
	Encrypt bool
	// LMK is this peer's 16-byte local master key, required when Encrypt
	// is set. SetLMK replaces it later.
	LMK [16]uint8
}

// AddPeer registers a peer (esp_now_add_peer). The broadcast MAC is a peer
// like any other: AddPeer(PeerConfig{MAC: espnow.BroadcastMAC}) before the
// first Broadcast. Errors: the driver's - ESP_ERR_ESPNOW_EXIST when the MAC
// is already registered, ESP_ERR_ESPNOW_FULL when the peer list or the
// encrypted-peer set is full, ESP_ERR_ESPNOW_ARG for a malformed record
// (an all-zero MAC among them), ESP_ERR_ESPNOW_NOT_INIT.
func AddPeer(cfg PeerConfig) error {
	if cfg.Encrypt && cfg.LMK == ([16]uint8{}) {
		return errOf(0x3066) // ESP_ERR_ESPNOW_ARG: encryption without a key
	}
	return errOf(addPeer(&cfg.MAC[0], &cfg.LMK[0], int32(cfg.Channel),
		int32(cfg.If), boolToInt(cfg.Encrypt)))
}

// DeletePeer removes a peer from the list (esp_now_del_peer). Errors: the
// driver's - ESP_ERR_ESPNOW_NOT_FOUND when the MAC was never added,
// ESP_ERR_ESPNOW_NOT_INIT.
func DeletePeer(mac [6]uint8) error {
	return errOf(delPeer(&mac[0]))
}

// PeerIsKnown reports whether this MAC is in the peer list
// (esp_now_is_peer_exist). It never fails: an unknown MAC is simply false.
func PeerIsKnown(mac [6]uint8) bool {
	return peerExists(&mac[0]) != 0
}

// PeerNum counts the peer list: total entries and how many of them are
// encrypted. Errors: the driver's (ESP_ERR_ESPNOW_NOT_INIT before Init).
type PeerNum struct {
	Total     int
	Encrypted int
}

// PeerCount reads the peer-list counters (esp_now_get_peer_num).
func PeerCount() (PeerNum, error) {
	var total, enc int32
	if e := peerNum(&total, &enc); e != 0 {
		return PeerNum{}, errOf(e)
	}
	return PeerNum{Total: int(total), Encrypted: int(enc)}, nil
}

// Send hands data to the driver for delivery to peer (esp_now_send). The
// returned error is the driver's synchronous verdict - nil means the frame
// was queued: ESP_ERR_ESPNOW_NOT_INIT (Init never ran), ESP_ERR_ESPNOW_ARG
// (empty or over-long payload - the ceiling is MaxDataLen bytes), ESP_ERR_ESPNOW_NOT_FOUND
// (peer not in the list - add it first), ESP_ERR_ESPNOW_NO_MEM (the driver's
// send queue is momentarily full; back off and retry), ESP_ERR_ESPNOW_IF or
// ESP_ERR_ESPNOW_CHAN (peer's interface or channel does not match the radio).
// The delivery verdict itself arrives later: Wait on the returned SendResult
// (see the package docs for the pairing rules).
func Send(peer [6]uint8, data []byte) (*SendResult, error) {
	if len(data) < 1 || len(data) > MaxDataLen {
		return nil, errOf(0x102) // ESP_ERR_INVALID_ARG: payload 1..MaxDataLen
	}
	res := &SendResult{Peer: peer, ch: make(chan SendStatus, 1)}
	pending[peer] = append(pending[peer], res)
	if err := errOf(send(&peer[0], &data[0], int32(len(data)))); err != nil {
		popPending(peer, res)
		return nil, err
	}
	return res, nil
}

// Broadcast is Send(BroadcastMAC, data): one frame, every ESP-NOW device on
// the channel hears it, no peer needs to know the sender. The broadcast
// peer itself must be in the peer list (AddPeer(PeerConfig{MAC:
// BroadcastMAC})); the confirmation reports success once the frame is out -
// nothing ACKs a broadcast.
func Broadcast(data []byte) (*SendResult, error) {
	return Send(BroadcastMAC, data)
}

// SetPMK sets the primary master key - the 16-byte root of ESP-NOW
// encryption, which protects the peers' LMKs on the air
// (esp_now_set_pmk). Both sides must share it; set it before traffic with
// encrypted peers. Errors: ESP_ERR_ESPNOW_NOT_INIT.
func SetPMK(key [16]uint8) error {
	return errOf(setPMK(&key[0]))
}

// SetLMK replaces an existing peer's local master key. IDF 6.x exposes no
// dedicated key setter - the key travels only inside the peer struct - so
// this is a read-modify-write on the peer record (glue: esp_now_get_peer,
// swap the key, esp_now_mod_peer). For a peer encrypted at AddPeer time
// this rotates its key; for an unencrypted peer it arms the key without
// turning encryption on (flip Encrypt with AddPeer - the driver has no
// post-hoc encrypt toggle this package exposes). Errors: the driver's -
// ESP_ERR_ESPNOW_NOT_FOUND for an unregistered MAC.
func SetLMK(peer [6]uint8, key [16]uint8) error {
	return errOf(setPeerKey(&peer[0], &key[0]))
}

// Version reports the ESP-NOW protocol version the firmware speaks
// (esp_now_get_version): 1 = v1.0 (250-byte frames), 2 = v2.0 (1470-byte
// frames, receives both). This package sends v1.0-sized frames, so peers of
// either generation receive them. Errors: ESP_ERR_ESPNOW_NOT_INIT.
func Version() (uint32, error) {
	var v uint32
	if e := version(&v); e != 0 {
		return 0, errOf(e)
	}
	return v, nil
}

// OwnMAC returns this device's MAC for the given interface
// (esp_wifi_get_mac): the address a peer must add to reach it on that
// interface. All zeros when the driver has no such interface. The WiFi
// driver must be initialized (after wifi.Setup).
func OwnMAC(ifIdx IfIdx) [6]uint8 {
	var mac [6]uint8
	ownMAC(&mac[0], int32(ifIdx))
	return mac
}

// MACString formats a MAC as aa:bb:cc:dd:ee:ff - the rendering the recv
// handler and SendResult.Peer print under. No strconv: the package stays
// stdlib-free like the rest of pkg/esp.
func MACString(mac [6]uint8) string {
	const digits = "0123456789abcdef"
	buf := make([]byte, 0, 17)
	for i, b := range mac {
		if i > 0 {
			buf = append(buf, ':')
		}
		buf = append(buf, digits[b>>4], digits[b&0xF])
	}
	return string(buf)
}

// pumpEvents is the package's single ring consumer: drain whatever the C
// ring holds, dispatch each record (recv -> handler, send-done -> the MAC's
// oldest SendResult, overflow -> the Dropped counter), hand the scheduler
// away between records, then rest one quantum in vTaskDelay - a real
// FreeRTOS yield that also feeds the idle tasks and the watchdog. The loop
// never holds a C-level wait across a scheduling point, so other goroutines
// regain control at least once per quantum.
func pumpEvents() {
	for {
		for {
			var ev event
			if poll(&ev.Kind, &ev.MAC[0], &ev.Aux, &staging[0], &ev.Len) != 1 {
				break
			}
			handle(&ev)
		}
		gosched()
		vTaskDelay(pollQuantumMs / tickMs)
	}
}

// staging is the payload hand-off buffer between the C ring and a recv
// dispatch. Only the pump touches it (it is the ring's single consumer), so
// a package-level buffer keeps the 250-byte payload off the 2 KiB goroutine
// fiber stacks; handlers receive a fresh copy, never the staging window.
var staging [MaxDataLen]byte

// event is one ring record in the pump's hand: the flat C-side form (kind,
// MAC, aux, length) with the payload staged in `staging` while the record is
// dispatched. It is deliberately unexported - users see the SetRecvHandler
// callback and SendResult instead.
type event struct {
	Kind int32
	// MAC is the record's address: the source for EvRecv, the destination
	// for EvSendDone.
	MAC [6]uint8
	// Aux is the record's scalar: the frame RSSI for EvRecv, the
	// esp_now_send_status_t for EvSendDone.
	Aux int32
	// Len is the payload length in staging (0 for EvSendDone).
	Len int32
}

// handle dispatches one polled record. Split from pumpEvents so the host
// tests can drive the dispatch path directly (the pump itself never runs on
// a host - see the stub file).
func handle(ev *event) {
	switch ev.Kind {
	case EvRecv:
		if handler == nil {
			return // drained and lost; documented at SetRecvHandler
		}
		data := make([]byte, ev.Len)
		copy(data, staging[:ev.Len])
		handler(ev.MAC, data)
	case EvSendDone:
		resolveSend(ev.MAC, SendStatus(ev.Aux))
	case EvOverflow:
		dropped++
	}
}

// resolveSend delivers a send confirmation to the MAC's oldest pending
// SendResult. Delivery is non-blocking: every result's channel is capacity
// 1 and written at most once, so the send always succeeds and a result the
// user abandoned (a timed-out Wait) still consumes the head slot - the
// queue keeps pairing later confirmations with later sends, in order.
func resolveSend(mac [6]uint8, status SendStatus) {
	q := pending[mac]
	if len(q) == 0 {
		return // an unsolicited confirmation (no live waiter): dropped
	}
	res := q[0]
	q[0] = nil
	q = q[1:]
	if len(q) == 0 {
		delete(pending, mac)
	} else {
		pending[mac] = q
	}
	res.ch <- status // capacity 1, single writer: never blocks
}

// popPending removes a SendResult that never entered flight (Send's driver
// call failed): the queue never resolves it.
func popPending(mac [6]uint8, res *SendResult) {
	q := pending[mac]
	for i, r := range q {
		if r == res {
			q = append(q[:i], q[i+1:]...)
			if len(q) == 0 {
				delete(pending, mac)
			} else {
				pending[mac] = q
			}
			return
		}
	}
}

func boolToInt(b bool) int32 {
	if b {
		return 1
	}
	return 0
}

// ErrName maps the esp_err codes this package can hit to their IDF names:
// the family-shared codes, the WiFi driver's (SetChannel and the peer
// interface/channel mismatches surface them) and the ESP_ERR_ESPNOW family
// (base 0x3064 = ESP_ERR_WIFI_BASE + 100, the codes esp_now.h defines).
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
	case 0x105:
		return "ESP_ERR_NOT_FOUND"
	case 0x107:
		return "ESP_ERR_TIMEOUT"
	case 0x3001:
		return "ESP_ERR_WIFI_NOT_INIT"
	case 0x3002:
		return "ESP_ERR_WIFI_NOT_STARTED"
	case 0x3003:
		return "ESP_ERR_WIFI_NOT_STOPPED"
	case 0x3004:
		return "ESP_ERR_WIFI_IF"
	case 0x3005:
		return "ESP_ERR_WIFI_MODE"
	case 0x3006:
		return "ESP_ERR_WIFI_STATE"
	case 0x3007:
		return "ESP_ERR_WIFI_CONN"
	case 0x3065:
		return "ESP_ERR_ESPNOW_NOT_INIT"
	case 0x3066:
		return "ESP_ERR_ESPNOW_ARG"
	case 0x3067:
		return "ESP_ERR_ESPNOW_NO_MEM"
	case 0x3068:
		return "ESP_ERR_ESPNOW_FULL"
	case 0x3069:
		return "ESP_ERR_ESPNOW_NOT_FOUND"
	case 0x306A:
		return "ESP_ERR_ESPNOW_INTERNAL"
	case 0x306B:
		return "ESP_ERR_ESPNOW_EXIST"
	case 0x306C:
		return "ESP_ERR_ESPNOW_IF"
	case 0x306D:
		return "ESP_ERR_ESPNOW_CHAN"
	case 0x7401:
		return "LLGO_ERR_GLUE_ABI"
	}
	return "esp_err 0x" + hex6(uint32(code))
}

// hex6 formats the low 24 bits as fixed-width hex (esp_err codes fit).
func hex6(v uint32) string {
	const digits = "0123456789ABCDEF"
	var buf [6]byte
	for i := 0; i < 6; i++ {
		buf[5-i] = digits[(v>>(uint(i)*4))&0xF]
	}
	return string(buf[:])
}
