//go:build !llgo.idf

// Host-build tests for the soft-AP surface. A host build has no WiFi stack,
// so nothing here claims a radio exists: the tests pin what the pure-Go layer
// owns - APConfig validation and the error domain it reports, the ErrName
// table, the event-kind space, the mailbox fan-out (dispatch/Next/Close) and
// the flat records the C side marshals - and that every driver binding
// reports errUnsupported instead of faking success.
//
// The fan-out tests drive dispatch and Subscription directly instead of
// calling Subscribe: Subscribe starts the package's poller goroutine, and on a
// host vTaskDelay is a no-op, so that goroutine would spin at full CPU. That
// is a property of the stub build, not of the package on the device.
package wifi

import (
	"errors"
	"testing"
	"unsafe"

	"github.com/xgo-dev/llgo/pkg/esp/esperr"
)

// apCfg is a valid baseline configuration the validation tests mutate.
func apCfg() APConfig {
	return APConfig{
		SSID:     "llgo-ap",
		Password: "llgoappass",
		Channel:  6,
		MaxSTA:   2,
		AuthMode: AuthWPA2PSK,
	}
}

// TestSetAPValidation pins the client-side guards: every rejected shape comes
// back as the esp_err the driver would have used, before anything crosses into
// C, and no accepted shape is rejected.
func TestSetAPValidation(t *testing.T) {
	tests := []struct {
		name string
		cfg  APConfig
		want int32 // 0 = the call passes validation
	}{
		{"valid wpa2", apCfg(), 0},
		{"valid open", APConfig{SSID: "llgo-ap"}, 0},
		{"valid zero channel and maxsta", APConfig{SSID: "x", MaxSTA: 0, Channel: 0}, 0},
		{"valid transition mode", APConfig{SSID: "x", Password: "12345678", AuthMode: AuthWPA2WPA3PSK}, 0},
		{"valid wpa3", APConfig{SSID: "x", Password: "12345678", AuthMode: AuthWPA3PSK}, 0},
		{"valid max station limit", APConfig{SSID: "x", MaxSTA: MaxAPStations}, 0},

		{"empty ssid", APConfig{SSID: ""}, 0x300A},
		{"ssid too long", APConfig{SSID: "0123456789012345678901234567890123"}, 0x102},
		{"password too short", APConfig{SSID: "x", Password: "short", AuthMode: AuthWPA2PSK}, 0x300B},
		{"password too long", APConfig{SSID: "x", Password: string(make([]byte, 64)), AuthMode: AuthWPA2PSK}, 0x102},
		{"password with open auth", APConfig{SSID: "x", Password: "llgoappass"}, 0x300B},
		{"wpa2 without password", APConfig{SSID: "x", AuthMode: AuthWPA2PSK}, 0x300B},
		{"unsupported auth mode", APConfig{SSID: "x", Password: "llgoappass", AuthMode: AuthMode(1)}, 0x102},
		{"channel out of range", APConfig{SSID: "x", Channel: 15}, 0x102},
		{"negative max sta", APConfig{SSID: "x", MaxSTA: -1}, 0x102},
		{"max sta over the glue limit", APConfig{SSID: "x", MaxSTA: MaxAPStations + 1}, 0x102},
	}
	for _, tc := range tests {
		err := SetAP(tc.cfg)
		if tc.want == 0 {
			// Validation passed: the stub must then report exactly that
			// there is no driver here, not fake a configured AP.
			if !errors.Is(err, &esperr.Error{Code: errUnsupported}) {
				t.Fatalf("%s: SetAP = %v, want errUnsupported from the stub", tc.name, err)
			}
			continue
		}
		if got := esperr.Code(err); got != tc.want {
			t.Fatalf("%s: SetAP code = 0x%X (%v), want 0x%X", tc.name, got, err, tc.want)
		}
		if e, ok := err.(*esperr.Error); !ok || e.Name != ErrName(tc.want) {
			t.Fatalf("%s: SetAP error = %v, want the ErrName of 0x%X", tc.name, err, tc.want)
		}
	}
}

// TestStubReportsUnsupported pins the honest-stub contract for the AP
// bindings: with a valid configuration nothing pretends to have configured a
// radio, and the queries keep their documented shapes (nil slice, zero IP).
func TestStubReportsUnsupported(t *testing.T) {
	unsupported := &esperr.Error{Code: errUnsupported}

	for name, err := range map[string]error{
		"Setup":          Setup(),
		"EventsRegister": EventsRegister(),
		"SetModeAP":      SetModeAP(),
		"SetModeAPSTA":   SetModeAPSTA(),
		"SetAP":          SetAP(apCfg()),
		"Start":          Start(),
		"Stop":           Stop(),
		"DeauthStation":  DeauthStation([6]uint8{1, 2, 3, 4, 5, 6}),
		"SetModeSTA":     SetModeSTA(),
		"SetSTA":         SetSTA("ssid", "password"),
		"Connect":        Connect(),
		"Disconnect":     Disconnect(),
		"ScanStart":      ScanStart(false),
		"PrintVersion":   PrintVersion(),
		"SetPowerSave":   SetPowerSave(PowerSaveMinModem),
	} {
		if !errors.Is(err, unsupported) {
			t.Fatalf("%s on a host build = %v, want errUnsupported (0x%X)", name, err, errUnsupported)
		}
	}

	if _, err := GetAPConfig(); !errors.Is(err, unsupported) {
		t.Fatalf("GetAPConfig = %v, want errUnsupported", err)
	}
	if _, err := GetPowerSave(); !errors.Is(err, unsupported) {
		t.Fatalf("GetPowerSave = %v, want errUnsupported", err)
	}
	if ip, err := GetAPIP(); ip != 0 || !errors.Is(err, unsupported) {
		t.Fatalf("GetAPIP = (%#x, %v), want (0, errUnsupported)", ip, err)
	}
	if sts, err := APStations(); sts != nil || !errors.Is(err, unsupported) {
		t.Fatalf("APStations = (%v, %v), want (nil, errUnsupported)", sts, err)
	}
	if _, err := Scan(ScanOptions{}); err == nil {
		t.Fatal("Scan on a host build reported success")
	}

	// The awkward stub values keep their historical meaning: nothing is
	// listening, so the mailbox is empty and the heaps read zero.
	var ev Event
	if n := Poll(&ev); n != 0 {
		t.Fatalf("Poll on a host build = %d, want 0", n)
	}
	if ip := GetIP(); ip != 0 {
		t.Fatalf("GetIP = %#x, want 0", ip)
	}
	if free, min := HeapInternalFree(), HeapMin(); free != 0 || min != 0 {
		t.Fatalf("heap marks = (%d, %d), want zeros", free, min)
	}
}

// TestErrUnsupportedOutsideEspRange pins the stub code's home: below the
// esp_err family (0x100..0x3xxx) and outside the transport/package segments,
// so no caller can mistake errUnsupported for a driver code.
func TestErrUnsupportedOutsideEspRange(t *testing.T) {
	if errUnsupported == 0 || errUnsupported < 0x3FFF {
		t.Fatalf("errUnsupported = 0x%X, want a value outside the esp_err range", errUnsupported)
	}
	if name := ErrName(errUnsupported); name != "esp_err 0x007F00" {
		t.Fatalf("ErrName(errUnsupported) = %q, want the unnamed hex form", name)
	}
}

// TestErrName covers the table the whole pipeline prints through: the esp_err
// values this package can return (including the AP-specific WIFI_SSID /
// WIFI_PASSWORD / NOT_ASSOC paths), the family-shared codes, and the Hex
// fallback for codes the table does not name.
func TestErrName(t *testing.T) {
	tests := map[int32]string{
		0:      "ESP_OK",
		-1:     "ESP_FAIL",
		0x101:  "ESP_ERR_NO_MEM",
		0x102:  "ESP_ERR_INVALID_ARG",
		0x103:  "ESP_ERR_INVALID_STATE",
		0x105:  "ESP_ERR_NOT_FOUND", // esp_wifi_ap_get_sta_aid: no such station
		0x107:  "ESP_ERR_TIMEOUT",
		0x110D: "ESP_ERR_NVS_NO_FREE_PAGES",
		0x1110: "ESP_ERR_NVS_NEW_VERSION_FOUND",
		0x3001: "ESP_ERR_WIFI_NOT_INIT",
		0x3002: "ESP_ERR_WIFI_NOT_STARTED",
		0x3003: "ESP_ERR_WIFI_NOT_STOPPED",
		0x3004: "ESP_ERR_WIFI_IF",
		0x3005: "ESP_ERR_WIFI_MODE",
		0x3006: "ESP_ERR_WIFI_STATE",
		0x3007: "ESP_ERR_WIFI_CONN",
		0x300A: "ESP_ERR_WIFI_SSID",
		0x300B: "ESP_ERR_WIFI_PASSWORD",
		0x300C: "ESP_ERR_WIFI_TIMEOUT",
		0x3015: "ESP_ERR_WIFI_NOT_ASSOC",
		0x7F01: "LLGO_ERR_GLUE_ABI",
		0x1234: "esp_err 0x001234",
	}
	for code, want := range tests {
		if got := ErrName(code); got != want {
			t.Fatalf("ErrName(0x%X) = %q, want %q", code, got, want)
		}
	}
}

// TestErrOf pins the error-value contract the family shares: ESP_OK is nil,
// anything else carries both the code and its name, and errors.Is matches by
// code through esperr.
func TestErrOf(t *testing.T) {
	if err := errOf(0); err != nil {
		t.Fatalf("errOf(0) = %v, want nil", err)
	}
	err := errOf(0x3005)
	if esperr.Code(err) != 0x3005 {
		t.Fatalf("errOf(0x3005) code = 0x%X", esperr.Code(err))
	}
	if err.Error() != "ESP_ERR_WIFI_MODE (12293)" {
		t.Fatalf("errOf(0x3005) = %q", err.Error())
	}
	if !errors.Is(err, &esperr.Error{Code: 0x3005}) {
		t.Fatal("errors.Is did not match by code")
	}
	if errors.Is(err, ErrTimeout) {
		t.Fatal("errors.Is matched an unrelated sentinel")
	}
	if !errors.Is(ErrTimeout, &esperr.Error{Code: 0x107}) {
		t.Fatal("ErrTimeout does not satisfy errors.Is against ESP_ERR_TIMEOUT")
	}
}

// TestAuthModeValues pins the constants to the IDF wifi_auth_mode_t numbers
// the glue casts straight through: a value that drifts here would silently
// configure a different security mode than the name says.
func TestAuthModeValues(t *testing.T) {
	tests := map[AuthMode]int32{
		AuthOpen:        0, // WIFI_AUTH_OPEN
		AuthWPA2PSK:     3, // WIFI_AUTH_WPA2_PSK
		AuthWPA3PSK:     6, // WIFI_AUTH_WPA3_PSK
		AuthWPA2WPA3PSK: 7, // WIFI_AUTH_WPA2_WPA3_PSK
	}
	for mode, want := range tests {
		if int32(mode) != want {
			t.Fatalf("%d = %d, want the IDF WIFI_AUTH_* value %d", mode, int32(mode), want)
		}
	}
	if DefaultMaxSTA != 4 {
		t.Fatalf("DefaultMaxSTA = %d, want the IDF default 4", DefaultMaxSTA)
	}
}

// TestPowerSaveModeValues pins the constants to the IDF wifi_ps_type_t
// numbers (esp_wifi_types_generic.h), which the glue casts straight through:
// a value that drifts here would configure a different sleep strategy than
// the name says, and the String rendering the probes print must stay the
// driver's own WIFI_PS_* spelling.
func TestPowerSaveModeValues(t *testing.T) {
	tests := map[PowerSaveMode]struct {
		value int32
		name  string
	}{
		PowerSaveNone:     {0, "NONE"},      // WIFI_PS_NONE
		PowerSaveMinModem: {1, "MIN_MODEM"}, // WIFI_PS_MIN_MODEM
		PowerSaveMaxModem: {2, "MAX_MODEM"}, // WIFI_PS_MAX_MODEM
	}
	for mode, want := range tests {
		if int32(mode) != want.value {
			t.Fatalf("%d = %d, want the IDF WIFI_PS_* value %d", mode, int32(mode), want.value)
		}
		if got := mode.String(); got != want.name {
			t.Fatalf("String(%d) = %q, want %q", int32(mode), got, want.name)
		}
	}
	if got := PowerSaveMode(7).String(); got != "INVALID" {
		t.Fatalf("String(7) = %q, want INVALID", got)
	}
}

// TestEventKindSpace pins the kind numbering: station kinds keep the values
// the STA era shipped, the AP kinds occupy fresh positives (the mailbox
// indexes its ring by kind, and Subscribe filters by value), and EvOverflow
// stays the negative global report.
func TestEventKindSpace(t *testing.T) {
	tests := map[int32]int32{
		EvNone:              0,
		EvStaStart:          1,
		EvStaConnected:      2,
		EvStaDisconnected:   3,
		EvScanDone:          4,
		EvGotIP:             5,
		EvLostIP:            6,
		EvAPStart:           7,
		EvAPStop:            8,
		EvAPStaConnected:    9,
		EvAPStaDisconnected: 10,
		EvAPStaIPAssigned:   11,
		EvOverflow:          -1,
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

// newTestSub builds a subscription without Subscribe (see the file comment:
// no poller goroutine on a host) and registers it in the fan-out list so
// dispatch reaches it.
func newTestSub(t *testing.T, kinds ...EventKind) *Subscription {
	t.Helper()
	s := &Subscription{kinds: kinds, ch: make(chan Event, subQueueLen)}
	subscribers = append(subscribers, s)
	t.Cleanup(func() {
		s.Close()
		subscribers = subscribers[:0]
	})
	return s
}

// TestDispatchAPStationEvent covers the AP fan-out path: an association event
// with MAC and AID reaches a subscriber that asked for that kind, keeps its
// fields intact, and is invisible to subscriptions filtering other kinds.
func TestDispatchAPStationEvent(t *testing.T) {
	apOnly := newTestSub(t, EvAPStaConnected, EvAPStaIPAssigned)
	staOnly := newTestSub(t, EvStaConnected)
	everything := newTestSub(t)

	mac := [6]uint8{0x24, 0x6f, 0x28, 0xaa, 0xbb, 0xcc}
	dispatch(Event{Kind: EvAPStaConnected, MAC: mac, AID: 1})

	ev, err := apOnly.Next(0)
	if err != nil {
		t.Fatalf("AP subscription saw no association event: %v", err)
	}
	if ev.Kind != EvAPStaConnected || ev.MAC != mac || ev.AID != 1 {
		t.Fatalf("AP event = %+v, want the association event with MAC/AID", ev)
	}
	if _, err := staOnly.Next(0); err == nil {
		t.Fatal("a station-kind subscription received a soft-AP event")
	}
	if ev, err := everything.Next(0); err != nil || ev.Kind != EvAPStaConnected {
		t.Fatalf("catch-all subscription = (%+v, %v), want the association event", ev, err)
	}
}

// TestDispatchOverflowBroadcast pins the one event every subscription sees:
// the mailbox's global overflow report, whatever its filter.
func TestDispatchOverflowBroadcast(t *testing.T) {
	sta := newTestSub(t, EvStaStart)
	dispatch(Event{Kind: EvOverflow, Aux: 3})
	ev, err := sta.Next(0)
	if err != nil || ev.Kind != EvOverflow || ev.Aux != 3 {
		t.Fatalf("overflow = (%+v, %v), want a broadcast EvOverflow", ev, err)
	}
}

// TestNextTimeoutAndClose covers the bounded wait and the retire path: an
// empty subscription reports ErrTimeout (never blocks), and a closed
// subscription stops receiving new events while keeping what it buffered.
func TestNextTimeoutAndClose(t *testing.T) {
	s := newTestSub(t, EvAPStart)
	if _, err := s.Next(100); !errors.Is(err, ErrTimeout) {
		t.Fatalf("Next on an empty subscription = %v, want ErrTimeout", err)
	}
	dispatch(Event{Kind: EvAPStart})
	s.Close()
	dispatch(Event{Kind: EvAPStart})
	if ev, err := s.Next(0); err != nil || ev.Kind != EvAPStart {
		t.Fatalf("buffered event after Close = (%+v, %v), want the pre-close event", ev, err)
	}
	if _, err := s.Next(0); !errors.Is(err, ErrTimeout) {
		t.Fatalf("second Next after Close = %v, want ErrTimeout", err)
	}
	s.Close() // idempotent
}

// TestQuantumDivisibility pins the invariant Subscription.Next relies on: one
// wait quantum is a whole number of FreeRTOS ticks. If tickMs ever grew past
// pollQuantumMs, vTaskDelay(0) in Next would spin instead of park, and a
// timeout bound would burn CPU instead of measuring time.
func TestQuantumDivisibility(t *testing.T) {
	if pollQuantumMs%tickMs != 0 || pollQuantumMs/tickMs < 1 {
		t.Fatalf("pollQuantumMs %d is not a positive multiple of tickMs %d",
			pollQuantumMs, tickMs)
	}
}

// TestIPString covers the dotted-quad formatter for the addresses the AP path
// reports: the IDF soft-AP default and the first DHCP lease.
func TestIPString(t *testing.T) {
	tests := map[uint32]string{
		0:          "0.0.0.0",
		0x0104A8C0: "192.168.4.1", // network order C0 A8 04 01 in memory
		0x0204A8C0: "192.168.4.2",
		0x0100007F: "127.0.0.1",
		0xFFFFFFFF: "255.255.255.255",
		0x0101A8C0: "192.168.1.1",
	}
	for ip, want := range tests {
		if got := IPString(ip); got != want {
			t.Fatalf("IPString(%#x) = %q, want %q", ip, got, want)
		}
	}
}

// TestStationMACString pins the MAC rendering the AP probe prints.
func TestStationMACString(t *testing.T) {
	s := Station{MAC: [6]uint8{0x24, 0x6f, 0x28, 0x0a, 0xbc, 0xde}, RSSI: -42, IP: 0x0204A8C0}
	if got := s.MACString(); got != "24:6f:28:0a:bc:de" {
		t.Fatalf("MACString = %q", got)
	}
}

// TestRecordLayout pins the Go side of the two records the C glue marshals.
// The glue asserts the same numbers on its side (_Static_assert), so a field
// added or reordered here can no longer drift silently into garbage MACs and
// IPs; wifi_test.go is the only place that sees both halves of the boundary
// outside a firmware build.
func TestRecordLayout(t *testing.T) {
	var st Station
	// C: typedef struct { uint8_t mac[6]; int8_t rssi; uint32_t ip; }
	if got := unsafe.Sizeof(st); got != 12 {
		t.Fatalf("sizeof(Station) = %d, want 12", got)
	}
	if got := unsafe.Offsetof(st.MAC); got != 0 {
		t.Fatalf("offset(Station.MAC) = %d, want 0", got)
	}
	if got := unsafe.Offsetof(st.RSSI); got != 6 {
		t.Fatalf("offset(Station.RSSI) = %d, want 6", got)
	}
	if got := unsafe.Offsetof(st.IP); got != 8 {
		t.Fatalf("offset(Station.IP) = %d, want 8", got)
	}

	var ap ApInfo
	// C: typedef struct { uint8_t bssid[6]; uint8_t ssid[33]; uint8_t channel;
	//                     int8_t rssi; int32_t authmode; }
	if got := unsafe.Sizeof(ap); got != 48 {
		t.Fatalf("sizeof(ApInfo) = %d, want 48", got)
	}
	for _, f := range []struct {
		name string
		got  uintptr
		want uintptr
	}{
		{"Bssid", unsafe.Offsetof(ap.Bssid), 0},
		{"Ssid", unsafe.Offsetof(ap.Ssid), 6},
		{"Channel", unsafe.Offsetof(ap.Channel), 39},
		{"Rssi", unsafe.Offsetof(ap.Rssi), 40},
		{"AuthMode", unsafe.Offsetof(ap.AuthMode), 44},
	} {
		if f.got != f.want {
			t.Fatalf("offset(ApInfo.%s) = %d, want %d", f.name, f.got, f.want)
		}
	}
}

// TestGlueABIContract covers the version gate: the package and the glue it is
// linked with must agree, a mismatch is the named ErrGlueABI, and on a host the
// stub always agrees so Setup keeps reporting errUnsupported instead of a fake
// mismatch.
func TestGlueABIContract(t *testing.T) {
	if glueABIVersion < 2 {
		t.Fatalf("glueABIVersion = %d, want the soft-AP contract version", glueABIVersion)
	}
	if got := glueABI(); got != glueABIVersion {
		t.Fatalf("stub glueABI = %d, want %d (a host build must not fake a mismatch)",
			got, glueABIVersion)
	}
	if err := Setup(); !errors.Is(err, &esperr.Error{Code: errUnsupported}) {
		t.Fatalf("Setup on a host build = %v, want errUnsupported", err)
	}
	if ErrGlueABI.Code != 0x7F01 || ErrGlueABI.Name != "LLGO_ERR_GLUE_ABI" {
		t.Fatalf("ErrGlueABI = %+v, want the 0x7F01 sentinel", ErrGlueABI)
	}
	if !errors.Is(ErrGlueABI, &esperr.Error{Code: 0x7F01}) {
		t.Fatal("errors.Is does not match ErrGlueABI by code")
	}
	if got, want := ErrGlueABI.Error(), "esp/wifi: glue ABI mismatch (code 32513)"; got != want {
		t.Fatalf("ErrGlueABI.Error() = %q, want %q", got, want)
	}
}

// TestTimeoutSentinel pins the wait-timeout sentinel end to end: the value
// Subscription.Next reports on a missed bound matches by identity and by its
// ESP_ERR_TIMEOUT (0x107) code annotation, the annotation stays reachable
// through esperr.Code and errors.As through the Wrap, and the rendering names
// the operation first (the "wifi: wait event" op prefix) with the human text
// and code annotation after.
func TestTimeoutSentinel(t *testing.T) {
	s := &Subscription{kinds: []EventKind{EvStaStart}, ch: make(chan Event, subQueueLen)}
	ev, err := s.Next(0) // non-positive bound: one non-blocking check, empty mailbox
	if err == nil {
		t.Fatal("Next(0) on an empty subscription reported success")
	}
	if ev != (Event{}) {
		t.Fatalf("Next(0) event = %+v, want the zero Event", ev)
	}
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("Next(0) = %v, want the ErrTimeout sentinel", err)
	}
	if !errors.Is(err, &esperr.Error{Code: 0x107}) {
		t.Fatal("the timeout error did not match a raw 0x107 Error (code path broken)")
	}
	if got := esperr.Code(err); got != 0x107 {
		t.Fatalf("esperr.Code = %#x, want 0x107", got)
	}
	var e *esperr.Error
	if !errors.As(err, &e) || e.Code != 0x107 || e.Name != "ESP_ERR_TIMEOUT" {
		t.Fatal("errors.As did not surface the *esperr.Error with the 0x107 annotation")
	}
	if got, want := err.Error(), "wifi: wait event: esp/wifi: timeout (code 263)"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
	if got, want := ErrTimeout.Error(), "esp/wifi: timeout (code 263)"; got != want {
		t.Fatalf("ErrTimeout.Error() = %q, want %q", got, want)
	}
	if esperr.Code(ErrTimeout) != 0x107 {
		t.Fatalf("esperr.Code(ErrTimeout) = %#x, want 0x107", esperr.Code(ErrTimeout))
	}
	// The scan wait reports the same sentinel under its own op; the site is
	// unreachable on a host (the stub scan refuses to start first), so the
	// assertion here is the sentinel identity the site wraps.
	if !errors.Is(esperr.Wrap("wifi: scan", ErrTimeout), ErrTimeout) {
		t.Fatal("Wrap does not carry the ErrTimeout identity")
	}
}

// TestReconnectIdleWithoutSubscribing covers the AP-mode guarantee the
// connectionless probe relies on: the auto-reconnect machine is off unless a
// program turns it on, disabling is idempotent, and its phases name
// themselves. SetAutoReconnect(true) is deliberately not exercised here - it
// subscribes, and a host build must not start the poller.
func TestReconnectIdleWithoutSubscribing(t *testing.T) {
	SetAutoReconnect(false)
	if got := ReconnectState(); got != ReconnIdle {
		t.Fatalf("ReconnectState = %v, want idle", got)
	}
	if ReconnIdle.String() != "idle" || ReconnBackoff.String() != "backoff" ||
		ReconnConnecting.String() != "connecting" {
		t.Fatal("ReconnectPhase names drifted")
	}
}
