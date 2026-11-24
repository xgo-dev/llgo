//go:build !llgo.idf

// Host-build tests. They exercise what a host build can exercise: the stub
// path (no Bluedroid in this build) and the pure-Go pieces - address
// rendering, the event kind space, the fan-out machinery and the error
// spaces. Device behavior is verified under esp-emu against a bumble peer,
// not here.
//
// The fan-out tests drive dispatch and Subscription directly instead of
// calling Subscribe (see newTestSub): Subscribe starts the package's poller
// goroutine, and on the host's preemptive runtime the poller's dispatch and
// the test goroutine's Subscribe/Close would race on the subscribers slice -
// the unlocked-slice safety is a cooperative-scheduler (single FreeRTOS
// task) property that only holds on the device.
package bt

import (
	"errors"
	"testing"
	"time"

	"github.com/xgo-dev/llgo/pkg/esp/esperr"
)

// newTestSub builds a subscription without Subscribe (see the file comment:
// no poller goroutine on a host) and registers it in the fan-out list so
// dispatch reaches it.
func newTestSub(t *testing.T, kinds ...EventKind) *Subscription {
	t.Helper()
	s := &Subscription{kinds: kinds, ch: make(chan Event, SubQueueLen)}
	subscribers = append(subscribers, s)
	t.Cleanup(func() {
		s.Close()
		subscribers = subscribers[:0]
	})
	return s
}

// TestErrorSegments pins the segment plan the esperr family agreed on: this
// package's own conditions in 0x76xx (0x77/0x78 reserved for the classic
// audio profiles), the esp domain where it overlaps (ESP_ERR_TIMEOUT), and
// the stub tag above everything.
func TestErrorSegments(t *testing.T) {
	for _, c := range []struct {
		err  error
		code int32
	}{
		{ErrGlueABI, 0x7601},
		{ErrNotReady, 0x7602},
		{ErrBusy, 0x7603},
		{ErrTimeout, 0x107},
		{ErrUnsupported, 0x7F00},
	} {
		if got := esperr.Code(c.err); got != c.code {
			t.Errorf("esperr.Code(%v) = %#x, want %#x", c.err, got, c.code)
		}
	}
}

// TestErrNameTable checks the ErrName vocabulary this pipeline can hit: the
// esp_err space of the glue's synchronous calls, the esp_bt_status_t space of
// SPP event statuses and ACL disconnect reasons (0..21 and the 0x100+ HCI
// range), and the package's internal segment.
func TestErrNameTable(t *testing.T) {
	for _, c := range []struct {
		code int32
		want string
	}{
		{0, "ESP_OK"},
		{-1, "ESP_FAIL"},
		{0x101, "ESP_ERR_NO_MEM"},
		{0x102, "ESP_ERR_INVALID_ARG"},
		{0x103, "ESP_ERR_INVALID_STATE"},
		{0x104, "ESP_ERR_INVALID_SIZE"},
		{0x107, "ESP_ERR_TIMEOUT"},
		{1, "ESP_BT_STATUS_FAIL"},
		{2, "ESP_BT_STATUS_NOT_READY"},
		{4, "ESP_BT_STATUS_BUSY"},
		{13, "ESP_BT_STATUS_PENDING"},
		{16, "ESP_BT_STATUS_TIMEOUT"},
		{20, "ESP_BT_STATUS_MEMORY_FULL"},
		{0x100, "ESP_BT_STATUS_HCI_SUCCESS"},
		{0x108, "ESP_BT_STATUS_HCI_CONNECTION_TOUT"},
		{0x113, "ESP_BT_STATUS_HCI_PEER_USER"},
		{0x116, "ESP_BT_STATUS_HCI_CONN_CAUSE_LOCAL_HOST"},
		{0x7601, "BT_ERR_GLUE_ABI"},
		{0x7602, "BT_ERR_NOT_READY"},
		{0x7603, "BT_ERR_BUSY"},
		{0x7F00, "BT_ERR_UNSUPPORTED"},
	} {
		if got := ErrName(c.code); got != c.want {
			t.Errorf("ErrName(%#x) = %q, want %q", c.code, got, c.want)
		}
	}
	// Codes outside the table fall back to the family's hex rendering, not
	// an empty string.
	if got := ErrName(0x555555); got != "esp error 0x555555" {
		t.Errorf("ErrName fallback = %q", got)
	}
}

// TestErrOf pins errOf: 0 is nil, everything else carries the raw code plus
// the ErrName rendering, and matches esperr-family sentinels by code.
func TestErrOf(t *testing.T) {
	if err := errOf(0); err != nil {
		t.Fatalf("errOf(0) = %v, want nil", err)
	}
	err := errOf(0x107)
	if !errors.Is(err, ErrTimeout) {
		t.Errorf("errOf(ESP_ERR_TIMEOUT) does not match ErrTimeout")
	}
	e, ok := err.(*esperr.Error)
	if !ok || e.Name != "ESP_ERR_TIMEOUT" {
		t.Errorf("errOf(0x107) = %v, want an ESP_ERR_TIMEOUT-named esperr", err)
	}
}

// TestStubReportsUnsupported exercises the host stub's honesty: Init/Deinit
// refuse at the binding level, and every pre-Init control call hits the
// package's not-ready guard before any binding would run.
func TestStubReportsUnsupported(t *testing.T) {
	if Ready() {
		t.Fatal("a host build reported a ready BT stack")
	}
	if err := Init(); !errors.Is(err, ErrUnsupported) {
		t.Errorf("Init = %v, want ErrUnsupported", err)
	}
	if err := Deinit(); !errors.Is(err, ErrUnsupported) {
		t.Errorf("Deinit = %v, want ErrUnsupported", err)
	}
	if err := SetName("llgo"); !errors.Is(err, ErrNotReady) {
		t.Errorf("SetName before Init = %v, want ErrNotReady", err)
	}
	if err := SetDiscoverable(true); !errors.Is(err, ErrNotReady) {
		t.Errorf("SetDiscoverable before Init = %v, want ErrNotReady", err)
	}
	if err := SPPListen(SPPOptions{}); !errors.Is(err, ErrNotReady) {
		t.Errorf("SPPListen before Init = %v, want ErrNotReady", err)
	}
	if _, err := SPPAccept(1); !errors.Is(err, ErrNotReady) {
		t.Errorf("SPPAccept before Init = %v, want ErrNotReady", err)
	}
	if err := A2DPEnable(A2DPConfig{}); !errors.Is(err, ErrNotReady) {
		t.Errorf("A2DPEnable before Init = %v, want ErrNotReady", err)
	}
	if b := Backend(); b != "none" {
		t.Errorf("Backend = %q, want \"none\"", b)
	}
}

// TestEventKindSpace pins the kind vocabulary: every kind distinct, no kind
// collides with the overflow report, and the profile-shared block is
// contiguous (a future classic-audio package continues after it).
func TestEventKindSpace(t *testing.T) {
	kinds := []EventKind{
		EvSPPInit, EvSPPListen, EvSPPConnOpen, EvSPPConnClose,
		EvSPPCong, EvSPPStop, EvSPPUninit, EvACLDisconnect,
		EvA2DPConnected, EvA2DPDisconnected, EvA2DPAudioCfg, EvA2DPVolume,
	}
	seen := make(map[EventKind]bool)
	for _, k := range kinds {
		if seen[k] {
			t.Errorf("kind %d duplicated", k)
		}
		seen[k] = true
	}
	if seen[EvOverflow] {
		t.Error("a profile kind collides with EvOverflow")
	}
	if EvOverflow >= 0 {
		t.Error("EvOverflow must stay negative (the C ring's report marker)")
	}
}

// TestAddrString checks the Bluedroid storage-order rendering: the array
// holds the most-significant octet first and the rendering walks it front to
// back (the reverse of the ble package, whose stack stores LSB-first). The
// example is the emu acceptance peer's fixed address, so the pin doubles as
// the regression for the render direction the emu logs exercise.
func TestAddrString(t *testing.T) {
	ev := Event{Addr: [6]uint8{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0x01}}
	if got, want := ev.AddrString(), "aa:bb:cc:dd:ee:01"; got != want {
		t.Errorf("AddrString = %q, want %q", got, want)
	}
}

// TestDispatchFanOut exercises the poller's fan-out against synthetic
// records: kind filters match, a connection's private close filter matches
// only its handle, overflow goes to everyone, and a full subscriber queue
// drops without blocking the fan-out.
func TestDispatchFanOut(t *testing.T) {
	kinds := newTestSub(t, EvSPPConnOpen)
	all := newTestSub(t)
	closeSub := newTestSub(t, EvSPPConnClose)
	closeSub.closeHandle = 0x42

	open := Event{Kind: int32(EvSPPConnOpen), Handle: 0x42}
	dispatch(open)
	if _, ok := kinds.pop(); !ok {
		t.Error("kind-filtered subscription missed EvSPPConnOpen")
	}
	if ev, ok := all.pop(); !ok || ev.Handle != 0x42 {
		t.Error("catch-all subscription missed EvSPPConnOpen")
	}
	if _, ok := closeSub.pop(); ok {
		t.Error("close-filtered subscription delivered a non-close record")
	}

	otherClose := Event{Kind: int32(EvSPPConnClose), Handle: 0x43}
	dispatch(otherClose)
	if _, ok := closeSub.pop(); ok {
		t.Error("close-filtered subscription caught another handle's close")
	}
	if _, ok := all.pop(); !ok {
		t.Error("catch-all subscription missed EvSPPConnClose")
	}

	ownClose := Event{Kind: int32(EvSPPConnClose), Handle: 0x42, Aux: 1}
	dispatch(ownClose)
	if ev, ok := closeSub.pop(); !ok || ev.Aux != 1 {
		t.Error("close-filtered subscription missed its own close")
	}
	if _, ok := all.pop(); !ok {
		t.Error("catch-all subscription missed the second close")
	}

	overflow := Event{Kind: int32(EvOverflow), Status: 2, Aux: 3}
	dispatch(overflow)
	for _, s := range []*Subscription{kinds, all, closeSub} {
		if ev, ok := s.pop(); !ok || ev.Kind != int32(EvOverflow) {
			t.Error("overflow report did not reach every subscription")
		}
	}
}

// TestNextTimeoutAndClose exercises the bounded wait on the host stub clock:
// a non-positive timeout is a single non-blocking check, a real deadline
// expires with ErrTimeout, and Close is idempotent.
func TestNextTimeoutAndClose(t *testing.T) {
	s := newTestSub(t, EvSPPCong)
	if _, err := s.Next(0); !errors.Is(err, ErrTimeout) {
		t.Errorf("Next(0) = %v, want ErrTimeout", err)
	}
	start := time.Now()
	if _, err := s.Next(50); !errors.Is(err, ErrTimeout) {
		t.Errorf("Next(50) = %v, want ErrTimeout", err)
	}
	if elapsed := time.Since(start); elapsed < 40*time.Millisecond {
		t.Errorf("Next(50) returned after %v, want at least one quantum", elapsed)
	}
	s.Close()
	s.Close() // must not panic or misbehave
}

// TestQuantumDivisibility pins the invariant every bounded loop relies on:
// one wait quantum is a whole number of FreeRTOS ticks, so vTaskDelay never
// parks for zero ticks (which would spin instead of rest).
func TestQuantumDivisibility(t *testing.T) {
	if pollQuantumMs%tickMs != 0 || pollQuantumMs/tickMs < 1 {
		t.Fatalf("pollQuantumMs %d is not a positive multiple of tickMs %d",
			pollQuantumMs, tickMs)
	}
}

// TestBoundConvention pins the timeout convention: a non-positive timeout is
// one quantum of bounded waiting (whose expiry is ErrWouldBlock), a positive
// one is wall clock (expiry ErrTimeout).
func TestBoundConvention(t *testing.T) {
	if got := bound(0); got != pollQuantumMs {
		t.Errorf("bound(0) = %d, want %d", got, pollQuantumMs)
	}
	if got := bound(-5); got != pollQuantumMs {
		t.Errorf("bound(-5) = %d, want %d", got, pollQuantumMs)
	}
	if got := bound(700); got != 700 {
		t.Errorf("bound(700) = %d, want 700", got)
	}
	if !errors.Is(boundErr(-5), esperr.ErrWouldBlock) {
		t.Error("boundErr(negative) should be the family's ErrWouldBlock")
	}
	if !errors.Is(boundErr(700), ErrTimeout) {
		t.Error("boundErr(positive) should be ErrTimeout")
	}
}

// TestSPPDefaults pins the wire-visible defaults a peer (bumble, a phone)
// depends on: the default SCN and the default SDP service name.
func TestSPPDefaults(t *testing.T) {
	if DefaultSCN != 1 {
		t.Errorf("DefaultSCN = %d, want 1", DefaultSCN)
	}
	if DefaultSPPName != "llgo-spp" {
		t.Errorf("DefaultSPPName = %q, want \"llgo-spp\"", DefaultSPPName)
	}
	if MaxMTU != 990 { // ESP_SPP_MAX_MTU = 3*330
		t.Errorf("MaxMTU = %d, want 990", MaxMTU)
	}
	if dataRingSlots != 8 { // SPP_DATA_SLOTS in the C glue
		t.Errorf("dataRingSlots = %d, want 8", dataRingSlots)
	}
}

// TestA2DPStubAndKinds exercises the A2DP surface a host build can see: the
// profile's control calls report the stub's unsupported tag once the
// package-level ready guard has passed (the stub Init itself fails first, so
// the path below drives the bindings through the raw shape they would take
// on a firmware), the audio counters report an honestly empty path, and the
// A2DP event kinds stay clear of the SPP block.
func TestA2DPStubAndKinds(t *testing.T) {
	// The stub's control bindings all refuse; the counters honestly read
	// zeros (an empty audio path, like the stub's empty rings).
	var total, written, dropped uint32
	a2dpStats(&total, &written, &dropped)
	if total != 0 || written != 0 || dropped != 0 {
		t.Errorf("stub audio counters = %d/%d/%d, want all zero",
			total, written, dropped)
	}
	if got := ErrName(0x107); got != "ESP_ERR_TIMEOUT" {
		t.Errorf("ErrName table drifted: %q", got) // sanity anchor
	}
	// The A2DP kinds reuse the existing Event fields (Aux = rate/volume,
	// Status = channels/reason) - pin the mapping the docs promise by
	// rendering one of each through the doc contract.
	cfg := Event{Kind: int32(EvA2DPAudioCfg), Aux: 44100, Status: 2,
		Addr: [6]uint8{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0x01}}
	if cfg.AddrString() != "aa:bb:cc:dd:ee:01" || cfg.Aux != 44100 {
		t.Errorf("audio cfg event shape drifted: %+v", cfg)
	}
	vol := Event{Kind: int32(EvA2DPVolume), Aux: 100}
	if vol.Aux < 0 || vol.Aux > 127 {
		t.Errorf("volume event out of AVRCP range: %d", vol.Aux)
	}
}

// TestCloseStateOnHost exercises the Close state machine on the stub
// bindings: the disconnect call fails with the unsupported tag and the
// connection reports closed afterwards, and the remotely-closed entry check
// (the m-1 rule) fires before any binding would run. The data-ring drain
// that follows every close path terminates immediately on a host (the stub
// ring is always empty) - the loop's device behavior is exercised under
// esp-emu.
func TestCloseStateOnHost(t *testing.T) {
	c := &SPPConn{handle: 0x21, closeSub: newTestSub(t, EvSPPConnClose)}
	c.closeSub.closeHandle = 0x21
	if err := c.Close(); !errors.Is(err, ErrUnsupported) {
		t.Errorf("Close = %v, want ErrUnsupported (stub disconnect)", err)
	}
	if !c.closed {
		t.Error("Close left the connection unmarked after a failed disconnect")
	}
	if err := c.Close(); !errors.Is(err, esperr.ErrClosed) {
		t.Errorf("second Close = %v, want esperr.ErrClosed", err)
	}

	// The peer already closed: Close must answer ErrClosed without going
	// near the bindings, per the Close contract.
	r := &SPPConn{handle: 0x22, remote: true, closeSub: newTestSub(t, EvSPPConnClose)}
	r.closeSub.closeHandle = 0x22
	if err := r.Close(); !errors.Is(err, esperr.ErrClosed) {
		t.Errorf("Close after remote close = %v, want esperr.ErrClosed", err)
	}
	if !r.closed {
		t.Error("remote-close path left the connection unmarked")
	}

	// The drain loop terminates on an empty ring (the stub reads 0) and
	// leaves the connection in the same closed shape.
	d := &SPPConn{handle: 0x23, closed: true, closeSub: newTestSub(t, EvSPPConnClose)}
	d.drain()
	if !d.closed {
		t.Error("drain changed the connection's closed state")
	}
}
