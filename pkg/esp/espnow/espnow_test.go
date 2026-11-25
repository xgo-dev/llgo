//go:build !llgo.idf

// Host-build tests for the espnow package. A host build has no ESP-NOW
// stack, so nothing here claims a radio exists: the tests pin what the
// pure-Go layer owns - the payload validation and the error domain it
// reports, the ErrName table, the event-kind space, the recv dispatch into
// the handler, the send-result queues (per-MAC FIFO, failed-send cleanup,
// timeout) and the dropped counter - and that every driver binding reports
// errUnsupported instead of faking success.
//
// The dispatch tests drive handle/resolveSend directly instead of calling
// Init: Init starts the package's pump goroutine, and on a host vTaskDelay
// is a no-op, so that goroutine would spin at full CPU. That is a property
// of the stub build, not of the package on the device.
package espnow

import (
	"errors"
	"testing"

	"github.com/xgo-dev/llgo/pkg/esp/esperr"
	"github.com/xgo-dev/llgo/pkg/esp/wifi"
)

// peer is an arbitrary unicast MAC the queue tests key on.
var peer = [6]uint8{0x24, 0x6f, 0x28, 0xaa, 0xbb, 0xcc}

// other is a second MAC, to prove queues are per-destination.
var other = [6]uint8{0x11, 0x22, 0x33, 0x44, 0x55, 0x66}

// clearHandler resets the package state a test touched (deferred by the
// dispatch tests, so one test's handler never leaks into the next).
func clearHandler() { handler = nil }

// TestSendValidation pins the client-side guard: a payload must be
// 1..MaxDataLen bytes, reported as ESP_ERR_INVALID_ARG before anything
// crosses into C, and no rejected send may leave a pending result behind
// (the driver never saw the frame, so its confirmation would hang a future
// send's queue).
func TestSendValidation(t *testing.T) {
	tests := []struct {
		name string
		len  int
	}{
		{"zero length", 0},
		{"over the ceiling", MaxDataLen + 1},
		{"way over", 4096},
	}
	for _, tc := range tests {
		data := make([]byte, tc.len)
		res, err := Send(peer, data)
		if esperr.Code(err) != 0x102 {
			t.Fatalf("%s: Send = (%v, %v), want ESP_ERR_INVALID_ARG (0x102)", tc.name, res, err)
		}
		if res != nil {
			t.Fatalf("%s: Send returned a result for a rejected send", tc.name)
		}
		if q := pending[peer]; len(q) != 0 {
			t.Fatalf("%s: pending queue grew for a rejected send (%d entries)", tc.name, len(q))
		}
	}

	// The accepted shapes reach the stub, which reports errUnsupported - and
	// Send must hand back no result at all (the driver never saw the frame;
	// there is nothing to wait on) and leave no pending entry behind.
	for _, tc := range []struct {
		name string
		peer [6]uint8
		len  int
	}{
		{"unicast at the ceiling", peer, MaxDataLen},
		{"unicast one byte", peer, 1},
		{"broadcast", BroadcastMAC, 5},
	} {
		res, err := Send(tc.peer, make([]byte, tc.len))
		if !errors.Is(err, unsupported) {
			t.Fatalf("%s: Send = %v, want errUnsupported from the stub", tc.name, err)
		}
		if res != nil {
			t.Fatalf("%s: Send returned a result for a refused send", tc.name)
		}
		if q := pending[tc.peer]; len(q) != 0 {
			t.Fatalf("%s: refused send left %d pending entries", tc.name, len(q))
		}
	}
}

// TestStubReportsUnsupported pins the honest-stub contract: with a device
// build nowhere in sight, nothing pretends a radio exists. PeerIsKnown and
// OwnMAC keep their no-error shapes (false / all-zero MAC); the pump must
// not be running.
func TestStubReportsUnsupported(t *testing.T) {
	for name, err := range map[string]error{
		"Init":       Init(),
		"Deinit":     Deinit(),
		"SetPMK":     SetPMK([16]uint8{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}),
		"SetLMK":     SetLMK(peer, [16]uint8{}),
		"DeletePeer": DeletePeer(peer),
		"SetChannel": SetChannel(6),
	} {
		if !errors.Is(err, unsupported) {
			t.Fatalf("%s on a host build = %v, want errUnsupported (0x%X)", name, err, errUnsupported)
		}
	}

	if _, err := Send(peer, []byte("ping")); err == nil || !errors.Is(err, unsupported) {
		t.Fatalf("Send on a host build = %v, want errUnsupported", err)
	}
	if _, err := Broadcast([]byte("ping")); err == nil || !errors.Is(err, unsupported) {
		t.Fatalf("Broadcast on a host build = %v, want errUnsupported", err)
	}
	if n, err := PeerCount(); n != (PeerNum{}) || !errors.Is(err, unsupported) {
		t.Fatalf("PeerCount = (%+v, %v), want (zero, errUnsupported)", n, err)
	}
	if _, err := Version(); err == nil || !errors.Is(err, unsupported) {
		t.Fatalf("Version on a host build = %v, want errUnsupported", err)
	}
	if PeerIsKnown(peer) {
		t.Fatal("PeerIsKnown on a host build reported a known peer")
	}
	if mac := OwnMAC(IfSTA); mac != ([6]uint8{}) {
		t.Fatalf("OwnMAC on a host build = %v, want zeros", mac)
	}

	var ev event
	if n := poll(&ev.Kind, &ev.MAC[0], &ev.Aux, &staging[0], &ev.Len); n != 0 {
		t.Fatalf("poll on a host build = %d, want 0", n)
	}
	if pumpRunning {
		t.Fatal("a host build started the pump goroutine")
	}
}

// TestAddPeerValidation pins the one client-side guard on the peer path:
// an encrypted peer without a key is refused before C (the driver would
// accept it and silently send plaintext).
func TestAddPeerValidation(t *testing.T) {
	err := AddPeer(PeerConfig{MAC: peer, Encrypt: true})
	if esperr.Code(err) != 0x3066 {
		t.Fatalf("AddPeer(encrypt, no LMK) = %v, want ESP_ERR_ESPNOW_ARG (0x3066)", err)
	}
	err = AddPeer(PeerConfig{MAC: peer, Encrypt: true, LMK: [16]uint8{1}})
	if !errors.Is(err, unsupported) {
		t.Fatalf("AddPeer(encrypt, LMK) = %v, want errUnsupported from the stub", err)
	}
	err = AddPeer(PeerConfig{MAC: BroadcastMAC})
	if !errors.Is(err, unsupported) {
		t.Fatalf("AddPeer(broadcast) = %v, want errUnsupported from the stub", err)
	}
}

// unsupported is the stub sentinel as a value, so the tests can errors.Is
// against it (matching by code, the family's rule).
var unsupported = &esperr.Error{Code: errUnsupported}

// TestSetChannelValidation pins the channel guard: 1..14, everything else
// ESP_ERR_INVALID_ARG before the driver is asked.
func TestSetChannelValidation(t *testing.T) {
	for _, ch := range []int{0, -1, 15, 100} {
		if err := SetChannel(ch); esperr.Code(err) != 0x102 {
			t.Fatalf("SetChannel(%d) = %v, want ESP_ERR_INVALID_ARG", ch, err)
		}
	}
	for _, ch := range []int{1, 6, 13, 14} {
		if err := SetChannel(ch); !errors.Is(err, unsupported) {
			t.Fatalf("SetChannel(%d) = %v, want errUnsupported from the stub", ch, err)
		}
	}
}

// TestErrUnsupportedOutsideEspRange pins the stub code's home: below the
// esp_err family and outside every internal segment, so no caller can
// mistake errUnsupported for a driver code.
func TestErrUnsupportedOutsideEspRange(t *testing.T) {
	if errUnsupported == 0 || errUnsupported < 0x3FFF {
		t.Fatalf("errUnsupported = 0x%X, want a value outside the esp_err range", errUnsupported)
	}
	if name := ErrName(errUnsupported); name != "esp_err 0x007F00" {
		t.Fatalf("ErrName(errUnsupported) = %q, want the unnamed hex form", name)
	}
}

// TestErrName covers the table the package prints through: the esp_err
// family codes the call paths hit (including the whole ESP_ERR_ESPNOW
// segment), the stub's unnamed value, and the hex fallback.
func TestErrName(t *testing.T) {
	tests := map[int32]string{
		0:      "ESP_OK",
		-1:     "ESP_FAIL",
		0x101:  "ESP_ERR_NO_MEM",
		0x102:  "ESP_ERR_INVALID_ARG",
		0x103:  "ESP_ERR_INVALID_STATE",
		0x105:  "ESP_ERR_NOT_FOUND",
		0x107:  "ESP_ERR_TIMEOUT",
		0x3001: "ESP_ERR_WIFI_NOT_INIT",
		0x3002: "ESP_ERR_WIFI_NOT_STARTED",
		0x3005: "ESP_ERR_WIFI_MODE",
		0x3006: "ESP_ERR_WIFI_STATE",
		0x3007: "ESP_ERR_WIFI_CONN",
		0x3065: "ESP_ERR_ESPNOW_NOT_INIT",
		0x3066: "ESP_ERR_ESPNOW_ARG",
		0x3067: "ESP_ERR_ESPNOW_NO_MEM",
		0x3068: "ESP_ERR_ESPNOW_FULL",
		0x3069: "ESP_ERR_ESPNOW_NOT_FOUND",
		0x306A: "ESP_ERR_ESPNOW_INTERNAL",
		0x306B: "ESP_ERR_ESPNOW_EXIST",
		0x306C: "ESP_ERR_ESPNOW_IF",
		0x306D: "ESP_ERR_ESPNOW_CHAN",
		0x7401: "LLGO_ERR_GLUE_ABI",
		0x1234: "esp_err 0x001234",
	}
	for code, want := range tests {
		if got := ErrName(code); got != want {
			t.Fatalf("ErrName(0x%X) = %q, want %q", code, got, want)
		}
	}
}

// TestErrOfPinsErrorContract pins the shared error-value behavior: ESP_OK is
// nil, anything else carries code + name, errors.Is matches by code, and the
// package sentinels are what their segments promise.
func TestErrOfPinsErrorContract(t *testing.T) {
	if err := errOf(0); err != nil {
		t.Fatalf("errOf(0) = %v, want nil", err)
	}
	err := errOf(0x3069)
	if esperr.Code(err) != 0x3069 {
		t.Fatalf("errOf(0x3069) code = 0x%X", esperr.Code(err))
	}
	if err.Error() != "ESP_ERR_ESPNOW_NOT_FOUND (12393)" {
		t.Fatalf("errOf(0x3069) = %q", err.Error())
	}
	if !errors.Is(err, &esperr.Error{Code: 0x3069}) {
		t.Fatal("errors.Is did not match by code")
	}
	if ErrGlueABI.Code != 0x7401 || ErrGlueABI.Name != "LLGO_ERR_GLUE_ABI" {
		t.Fatalf("ErrGlueABI = %+v, want the 0x7401 sentinel", ErrGlueABI)
	}
	if !errors.Is(ErrGlueABI, &esperr.Error{Code: 0x7401}) {
		t.Fatal("errors.Is does not match ErrGlueABI by code")
	}
	if !errors.Is(ErrTimeout, &esperr.Error{Code: 0x107}) {
		t.Fatal("ErrTimeout does not satisfy errors.Is against ESP_ERR_TIMEOUT")
	}
}

// TestSentinels pins the package's two sentinels in the new shape: text
// first in the rendering with the historical code annotation after, the
// annotation reachable through errors.As and esperr.Code, matching both by
// identity and by code, and isolation from unrelated sentinels.
func TestSentinels(t *testing.T) {
	if got, want := ErrGlueABI.Error(), "esp/espnow: glue ABI mismatch (code 29697)"; got != want {
		t.Fatalf("ErrGlueABI.Error() = %q, want %q", got, want)
	}
	if got, want := ErrTimeout.Error(), "esp/espnow: timeout (code 263)"; got != want {
		t.Fatalf("ErrTimeout.Error() = %q, want %q", got, want)
	}
	if esperr.Code(ErrGlueABI) != 0x7401 || esperr.Code(ErrTimeout) != 0x107 {
		t.Fatalf("sentinel codes drifted: %v %v", esperr.Code(ErrGlueABI), esperr.Code(ErrTimeout))
	}
	for _, s := range []error{ErrGlueABI, ErrTimeout} {
		var e *esperr.Error
		if !errors.As(s, &e) || e.Name == "" {
			t.Fatalf("%v lost its ErrName annotation", s)
		}
	}
	if !errors.Is(ErrTimeout, wifi.ErrTimeout) || !errors.Is(wifi.ErrTimeout, ErrTimeout) {
		t.Fatal("the 0x107 sentinels no longer match each other across packages")
	}
	if errors.Is(ErrTimeout, esperr.ErrTimeout) {
		t.Fatal("espnow.ErrTimeout matched the lwip ETIMEDOUT sentinel")
	}
}

// TestWaitTimeoutShape pins the SendResult.Wait give-up end to end: the
// error wraps the ErrTimeout sentinel under the "espnow: wait confirm" op,
// so the rendering names the operation first while errors.Is (identity and
// by the 0x107 code), errors.As and esperr.Code all see through the wrap.
func TestWaitTimeoutShape(t *testing.T) {
	r := &SendResult{Peer: peer, ch: make(chan SendStatus, 1)}
	_, err := r.Wait(0)
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("Wait(0) = %v, want the ErrTimeout sentinel", err)
	}
	if !errors.Is(err, &esperr.Error{Code: 0x107}) {
		t.Fatal("the wait error did not match a raw 0x107 Error (code path broken)")
	}
	if got := esperr.Code(err); got != 0x107 {
		t.Fatalf("esperr.Code = %#x, want 0x107", got)
	}
	var e *esperr.Error
	if !errors.As(err, &e) || e.Code != 0x107 || e.Name != "ESP_ERR_TIMEOUT" {
		t.Fatal("errors.As did not surface the *esperr.Error with the 0x107 annotation")
	}
	if got, want := err.Error(), "espnow: wait confirm: esp/espnow: timeout (code 263)"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
	// The nil-result guard (a result that never entered flight) reports the
	// same wrapped sentinel.
	_, err = (*SendResult)(nil).Wait(0)
	if !errors.Is(err, ErrTimeout) || !gotPrefix(err.Error(), "espnow: wait confirm") {
		t.Fatalf("nil-result Wait = %v, want the wrapped ErrTimeout", err)
	}
}

// gotPrefix reports whether s starts with prefix.
func gotPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// TestGuardErrorsStayBare pins the contract the espnow probe depends on: the
// client-side refusals (payload bounds, encryption without a key) come back
// as bare *esperr.Error values named through ErrName - the probe's want and
// espCode helpers type-assert *esperr.Error to print "ok: ESP_ERR_ESPNOW_ARG"
// lines and read codes, so a wrapped guard error would flip a correct probe
// run into FAIL lines. Unlike nvs, this package's demo reads the error value
// itself, not just its code.
func TestGuardErrorsStayBare(t *testing.T) {
	err := AddPeer(PeerConfig{MAC: peer, Encrypt: true})
	e, ok := err.(*esperr.Error)
	if !ok || e.Code != 0x3066 || e.Name != "ESP_ERR_ESPNOW_ARG" {
		t.Fatalf("AddPeer guard = %v (%T), want a bare *esperr.Error named ESP_ERR_ESPNOW_ARG", err, err)
	}
	_, err = Send(peer, nil)
	e, ok = err.(*esperr.Error)
	if !ok || e.Code != 0x102 || e.Name != "ESP_ERR_INVALID_ARG" {
		t.Fatalf("Send guard = %v (%T), want a bare *esperr.Error named ESP_ERR_INVALID_ARG", err, err)
	}
	if got := err.Error(); got != "ESP_ERR_INVALID_ARG (258)" {
		t.Fatalf("Send guard renders %q, want the code-era form", got)
	}
}

// TestIDFConstants pins the IDF numbers the glue casts straight through: a
// value that drifts here would silently configure a different interface or
// report a different verdict than the name says.
func TestIDFConstants(t *testing.T) {
	if int32(IfSTA) != 0 || int32(IfAP) != 1 {
		t.Fatalf("IfIdx values drifted: STA=%d AP=%d, want IDF's 0/1", IfSTA, IfAP)
	}
	if !SendSuccess.OK() || SendFail.OK() {
		t.Fatal("SendStatus.OK drifted")
	}
	if SendSuccess.String() != "success" || SendFail.String() != "fail" {
		t.Fatal("SendStatus.String drifted")
	}
	if BroadcastMAC != [6]uint8{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF} {
		t.Fatal("BroadcastMAC is not the all-ones address")
	}
	if MaxDataLen != 250 || MaxPeers != 20 || KeyLen != 16 {
		t.Fatalf("bounds drifted: %d/%d/%d, want ESP_NOW's 250/20/16", MaxDataLen, MaxPeers, KeyLen)
	}
	if pollQuantumMs%tickMs != 0 || pollQuantumMs/tickMs < 1 {
		t.Fatalf("pollQuantumMs %d is not a positive multiple of tickMs %d",
			pollQuantumMs, tickMs)
	}
	if recvRingSlots&(recvRingSlots-1) != 0 || recvRingSlots == 0 {
		t.Fatalf("recvRingSlots = %d, want a power of two (the ring indexes mask)", recvRingSlots)
	}
}

// TestEventKindSpace pins the kind numbering: this package's own table
// starts at 1 (never aliased with wifi's or ble's kinds - the tables belong
// to their packages), and EvOverflow stays the negative global report.
func TestEventKindSpace(t *testing.T) {
	tests := map[int32]int32{
		EvNone:     0,
		EvRecv:     1,
		EvSendDone: 2,
		EvOverflow: -1,
	}
	seen := map[int32]bool{}
	for kind, want := range tests {
		if kind != want {
			t.Fatalf("event kind %d, want %d", kind, want)
		}
		if seen[kind] {
			t.Fatalf("event kind %d is used twice", kind)
		}
		seen[kind] = true
	}
}

// TestRecvDispatch covers the receive path: a polled record reaches the
// registered handler with its own copy of the payload (mutating the staging
// window afterwards changes nothing the handler kept), MAC and RSSI ride
// along, a nil handler drains silently, and an unknown kind is a no-op.
func TestRecvDispatch(t *testing.T) {
	defer clearHandler()

	type recv struct {
		from [6]uint8
		data []byte
	}
	var got []recv
	SetRecvHandler(func(from [6]uint8, data []byte) {
		got = append(got, recv{from, data})
	})

	copy(staging[:], []byte("hello frame"))
	handle(&event{Kind: EvRecv, MAC: peer, Aux: -42, Len: 11})
	if len(got) != 1 {
		t.Fatalf("handler saw %d frames, want 1", len(got))
	}
	if got[0].from != peer {
		t.Fatalf("handler from = %v, want %v", got[0].from, peer)
	}
	if string(got[0].data) != "hello frame" {
		t.Fatalf("handler data = %q, want %q", got[0].data, "hello frame")
	}

	// The copy is the handler's: overwrite the staging window and confirm
	// the delivered slice did not change with it.
	copy(staging[:], []byte("OVERWRITTEN!!"))
	if string(got[0].data) != "hello frame" {
		t.Fatalf("handler data aliased the staging window: %q", got[0].data)
	}

	// A nil handler drains without dispatching (and without panicking).
	clearHandler()
	handle(&event{Kind: EvRecv, MAC: other, Aux: 0, Len: 3})
	if len(got) != 1 {
		t.Fatalf("nil handler dispatched a frame (%d total)", len(got))
	}

	// Overflow bumps the dropped counter, and only that.
	before := dropped
	handle(&event{Kind: EvOverflow, Aux: 3})
	if dropped != before+1 {
		t.Fatalf("overflow: dropped %d -> %d, want +1", before, dropped)
	}
	if Dropped() != dropped {
		t.Fatal("Dropped() does not mirror the counter")
	}

	// An unknown kind is ignored, never dispatched as anything else.
	clearHandler()
	SetRecvHandler(func(from [6]uint8, data []byte) { got = append(got, recv{from, data}) })
	handle(&event{Kind: EvNone})
	handle(&event{Kind: 99})
	if len(got) != 1 {
		t.Fatalf("unknown kinds dispatched as recv (%d total)", len(got))
	}
}

// resolve poll helper for the queue tests: dispatch a send-done record the
// way the pump would.
func resolve(mac [6]uint8, status SendStatus) {
	handle(&event{Kind: EvSendDone, MAC: mac, Aux: int32(status)})
}

// TestSendResultQueue covers the per-MAC FIFO: confirmations resolve the
// oldest result first, another MAC's queue is untouched, an unwaited result
// still consumes its slot so later results keep pairing in order, and a
// queue that empties is deleted (no map leak).
func TestSendResultQueue(t *testing.T) {
	// The stub build's Send never queues (the driver refuses before the
	// result enters flight); drive the queue the way a device build's
	// successful Send does.
	r1 := &SendResult{Peer: peer, ch: make(chan SendStatus, 1)}
	r2 := &SendResult{Peer: peer, ch: make(chan SendStatus, 1)}
	r3 := &SendResult{Peer: other, ch: make(chan SendStatus, 1)}
	pending[peer] = append(pending[peer], r1, r2)
	pending[other] = append(pending[other], r3)

	resolve(peer, SendSuccess)
	if s, err := r1.Wait(0); err != nil || s != SendSuccess {
		t.Fatalf("first result = (%v, %v), want (success, nil)", s, err)
	}
	if _, err := r2.Wait(0); !errors.Is(err, ErrTimeout) {
		t.Fatalf("second result resolved with the first's status: %v", err)
	}
	resolve(peer, SendFail)
	if s, err := r2.Wait(0); err != nil || s != SendFail {
		t.Fatalf("second result = (%v, %v), want (fail, nil)", s, err)
	}
	if q := pending[peer]; len(q) != 0 {
		t.Fatalf("peer queue still holds %d results, want empty (map deleted)", len(q))
	}
	if q := pending[other]; len(q) != 1 {
		t.Fatalf("other MAC's queue disturbed: %d entries", len(q))
	}

	// An unsolicited confirmation (queue empty) is dropped, not delivered
	// anywhere and not counted as anything.
	resolve(other, SendSuccess)
	if s, err := r3.Wait(0); err != nil || s != SendSuccess {
		t.Fatalf("third result = (%v, %v), want (success, nil)", s, err)
	}
	if _, err := r3.Wait(0); !errors.Is(err, ErrTimeout) {
		t.Fatalf("a resolved result re-delivered: %v", err)
	}

	resolve(peer, SendSuccess) // no pending left for peer
	if len(pending) != 0 {
		t.Fatalf("pending map leaked entries: %v", pending)
	}
}

// TestSendResultWaitTimeout covers the bounded wait on a live-looking but
// unresolved result: Wait with a positive bound times out (the host's
// quanta are no-ops, so this is a bounded spin, not a real second), a
// zero/negative bound is a single non-blocking check, and a zero-value
// result (no channel - never queued) reports timeout instead of hanging.
func TestSendResultWaitTimeout(t *testing.T) {
	r := &SendResult{Peer: peer, ch: make(chan SendStatus, 1)}
	if _, err := r.Wait(1); !errors.Is(err, ErrTimeout) {
		t.Fatalf("Wait(1) = %v, want ErrTimeout", err)
	}
	if _, err := r.Wait(0); !errors.Is(err, ErrTimeout) {
		t.Fatalf("Wait(0) = %v, want ErrTimeout", err)
	}
	if _, err := r.Wait(-5); !errors.Is(err, ErrTimeout) {
		t.Fatalf("Wait(-5) = %v, want ErrTimeout", err)
	}
	var zero *SendResult
	if _, err := zero.Wait(0); !errors.Is(err, ErrTimeout) {
		t.Fatalf("nil-result Wait = %v, want ErrTimeout", err)
	}
	value := SendResult{}
	if _, err := value.Wait(0); !errors.Is(err, ErrTimeout) {
		t.Fatalf("zero-value result Wait = %v, want ErrTimeout", err)
	}
}

// TestWaitDeliversLateConfirmation covers the read side of the
// abandoned-head rule: a Wait that timed out leaves the channel in place,
// so the pump's later delivery lands in the buffer (never mis-delivered to
// a Wait that already returned), and a second Wait still sees the value.
func TestWaitDeliversLateConfirmation(t *testing.T) {
	r := &SendResult{Peer: peer, ch: make(chan SendStatus, 1)}
	pending[peer] = append(pending[peer], r)
	defer func() { delete(pending, peer) }()
	if _, err := r.Wait(0); !errors.Is(err, ErrTimeout) {
		t.Fatalf("pre-delivery Wait = %v, want ErrTimeout", err)
	}
	resolve(peer, SendSuccess)
	if s, err := r.Wait(0); err != nil || s != SendSuccess {
		t.Fatalf("post-delivery Wait = (%v, %v), want (success, nil)", s, err)
	}
}

// TestMACString pins the MAC rendering the handler and probe output use.
func TestMACString(t *testing.T) {
	tests := map[[6]uint8]string{
		{0x24, 0x6f, 0x28, 0x0a, 0xbc, 0xde}: "24:6f:28:0a:bc:de",
		BroadcastMAC:                         "ff:ff:ff:ff:ff:ff",
		{}:                                   "00:00:00:00:00:00",
	}
	for mac, want := range tests {
		if got := MACString(mac); got != want {
			t.Fatalf("MACString = %q, want %q", got, want)
		}
	}
}

// TestGlueABIContract covers the version gate: the package and the glue it
// is linked with must agree, a mismatch is the named ErrGlueABI, and on a
// host the stub always agrees so Init keeps reporting errUnsupported instead
// of a fake mismatch.
func TestGlueABIContract(t *testing.T) {
	if glueABIVersion < 1 {
		t.Fatalf("glueABIVersion = %d, want the espnow contract version", glueABIVersion)
	}
	if got := glueABI(); got != glueABIVersion {
		t.Fatalf("stub glueABI = %d, want %d (a host build must not fake a mismatch)",
			got, glueABIVersion)
	}
	if err := Init(); !errors.Is(err, &esperr.Error{Code: errUnsupported}) {
		t.Fatalf("Init on a host build = %v, want errUnsupported", err)
	}
	if pumpRunning {
		t.Fatal("Init on a host build started the pump")
	}
}
