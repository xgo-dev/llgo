//go:build !llgo.idf

// Host-build tests. They exercise what a host build can exercise: the stub
// path (no BLE in this build) and the pure-Go pieces - address and advertising
// payload rendering, and the three error spaces. Device behavior is verified
// under esp-emu and on hardware, not here.
package ble

import (
	"errors"
	"testing"
	"unsafe"

	"github.com/xgo-dev/llgo/pkg/esp/esperr"
)

// TestWireLayout pins the layout of the records that cross the C boundary. The
// sizes and offsets are the glue's _Static_asserts (llgo_ble_event_t,
// llgo_ble_svc_t, llgo_ble_chr_t); a drift here would make the C side write
// past the Go storage, so it is pinned on this side too.
func TestWireLayout(t *testing.T) {
	var ev Event
	if got, want := unsafe.Sizeof(ev), uintptr(188); got != want {
		t.Errorf("sizeof Event = %d, want %d", got, want)
	}
	for _, c := range []struct {
		name string
		got  uintptr
		want uintptr
	}{
		{"Addr", unsafe.Offsetof(ev.Addr), 16},
		{"AddrType", unsafe.Offsetof(ev.AddrType), 22},
		{"Rssi", unsafe.Offsetof(ev.Rssi), 23},
		{"AdvLen", unsafe.Offsetof(ev.AdvLen), 24},
		{"Adv", unsafe.Offsetof(ev.Adv), 25},
		{"AttHandle", unsafe.Offsetof(ev.AttHandle), 56},
		{"DataLen", unsafe.Offsetof(ev.DataLen), 58},
		{"Indication", unsafe.Offsetof(ev.Indication), 59},
		{"Data", unsafe.Offsetof(ev.Data), 60},
	} {
		if c.got != c.want {
			t.Errorf("Event.%s offset = %d, want %d", c.name, c.got, c.want)
		}
	}
	var svc svcRec
	if got := unsafe.Sizeof(svc); got != 22 {
		t.Errorf("sizeof svcRec = %d, want 22", got)
	}
	if got := unsafe.Offsetof(svc.Start); got != 18 {
		t.Errorf("svcRec.Start offset = %d, want 18", got)
	}
	var chr chrRec
	if got := unsafe.Sizeof(chr); got != 22 {
		t.Errorf("sizeof chrRec = %d, want 22", got)
	}
	if got := unsafe.Offsetof(chr.Properties); got != 17 {
		t.Errorf("chrRec.Properties offset = %d, want 17", got)
	}
	if got := unsafe.Offsetof(chr.Value); got != 18 {
		t.Errorf("chrRec.Value offset = %d, want 18", got)
	}
	if MaxData != 128 {
		t.Errorf("MaxData = %d, but the layout above and BW_DATA_MAX are pinned to 128", MaxData)
	}
}

// TestParseUUID covers both canonical text forms, the case-insensitive and
// "0x"-prefixed spellings, the round trip through String, and the rejections.
func TestParseUUID(t *testing.T) {
	good := []struct {
		text    string
		want    UUID
		wantStr string // the canonical spelling String renders
	}{
		{"fff0", UUID16(0xfff0), "fff0"},
		{"FFF0", UUID16(0xfff0), "fff0"},
		{"0x180d", UUID16(0x180d), "180d"},
		{"0000180d-0000-1000-8000-00805f9b34fb", UUID16(0x180d), "180d"}, // base-UUID alias
		{"0000180d00001000800000805f9b34fb", UUID16(0x180d), "180d"},     // undashed form
		{
			"6c6c676f-1111-2222-3344-556677889900",
			UUID128([16]uint8{
				0x6c, 0x6c, 0x67, 0x6f, 0x11, 0x11, 0x22, 0x22,
				0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0x00,
			}),
			"6c6c676f-1111-2222-3344-556677889900",
		},
	}
	for _, c := range good {
		got, err := ParseUUID(c.text)
		if err != nil {
			t.Errorf("ParseUUID(%q) = %v, want a UUID", c.text, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParseUUID(%q) = %v, want %v", c.text, got, c.want)
		}
		if s := got.String(); s != c.wantStr {
			t.Errorf("String of %q = %q, want %q", c.text, s, c.wantStr)
		}
		if again, err := ParseUUID(got.String()); err != nil || again != got {
			t.Errorf("round trip of %q = %v, %v", c.text, again, err)
		}
	}
	bad := []string{"", "fff", "fffff", "0x", "zzzz", "6c6c676f-1111-2222-3344-55667788990",
		"6c6c676f_1111-2222-3344-556677889900", "6c6c676f-1111-2222-3344-55667788990g"}
	for _, text := range bad {
		if _, err := ParseUUID(text); !errors.Is(err, ErrBadUUID) {
			t.Errorf("ParseUUID(%q) = %v, want ErrBadUUID", text, err)
		}
	}
}

// TestUUIDForm pins the width, the zero value and the rendering of both
// widths, including the base-form collapse that makes a peer's spelling
// irrelevant to equality.
func TestUUIDForm(t *testing.T) {
	short := UUID16(0x2a00)
	if short.Len() != 2 || short.IsZero() {
		t.Fatalf("UUID16: len = %d, zero = %v", short.Len(), short.IsZero())
	}
	if got, want := short.String(), "2a00"; got != want {
		t.Fatalf("UUID16.String = %q, want %q", got, want)
	}
	var zero UUID
	if !zero.IsZero() || zero.Len() != 0 || zero.String() != "" {
		t.Fatalf("zero UUID: len = %d, string = %q", zero.Len(), zero.String())
	}
	// A 128-bit UUID outside the Bluetooth base pattern stays 128-bit.
	var vendor [16]uint8
	for i := range vendor {
		vendor[i] = uint8(i + 1)
	}
	long := UUID128(vendor)
	if long.Len() != 16 {
		t.Fatalf("UUID128 len = %d, want 16", long.Len())
	}
	if got, want := long.String(), "01020304-0506-0708-090a-0b0c0d0e0f10"; got != want {
		t.Fatalf("UUID128.String = %q, want %q", got, want)
	}
	if long == short {
		t.Fatal("a 128-bit UUID compared equal to a 16-bit one")
	}
	// The same UUID in its Bluetooth base form is one value: a peer that
	// reports 128 bits for it must compare equal to the 16-bit constant.
	base := [16]uint8{0x00, 0x00, 0x18, 0x0d, 0x00, 0x00, 0x10, 0x00,
		0x80, 0x00, 0x00, 0x80, 0x5f, 0x9b, 0x34, 0xfb}
	if got := UUID128(base); got != UUID16(0x180d) {
		t.Fatalf("base-form UUID128 = %v, want the 16-bit alias", got)
	}
	// uuidFromWire is the same normalization for a record the glue filled.
	wire := [16]uint8{0xfb, 0x34, 0x9b, 0x5f, 0x80, 0x00, 0x00, 0x80,
		0x00, 0x10, 0x00, 0x00, 0x0d, 0x18, 0x00, 0x00}
	if got := uuidFromWire(&wire, 16); got != UUID16(0x180d) {
		t.Fatalf("uuidFromWire(base form) = %v, want the 16-bit alias", got)
	}
	wire16 := [16]uint8{0xf0, 0xff}
	if got := uuidFromWire(&wire16, 2); got != UUID16(0xfff0) {
		t.Fatalf("uuidFromWire(16-bit) = %v, want fff0", got)
	}
}

// TestGattArguments pins the caller-side checks: they report a bad argument
// before the stack state, and the property rules that pick a write procedure.
func TestGattArguments(t *testing.T) {
	var zeroChr Characteristic // ValueHandle 0: never a real attribute
	if _, err := Read(ConnID(1), zeroChr); !errors.Is(err, ErrBadValue) {
		t.Errorf("Read(zero handle) = %v, want ErrBadValue", err)
	}
	if err := Write(ConnID(1), zeroChr, []byte("x")); !errors.Is(err, ErrBadValue) {
		t.Errorf("Write(zero handle) = %v, want ErrBadValue", err)
	}
	if _, err := SubscribeChar(ConnID(1), zeroChr); !errors.Is(err, ErrBadValue) {
		t.Errorf("SubscribeChar(zero handle) = %v, want ErrBadValue", err)
	}
	if err := Unsubscribe(nil); !errors.Is(err, ErrBadValue) {
		t.Errorf("Unsubscribe(nil) = %v, want ErrBadValue", err)
	}
	if _, err := Characteristics(ConnID(1), UUID{}); !errors.Is(err, ErrBadUUID) {
		t.Errorf("Characteristics(zero uuid) = %v, want ErrBadUUID", err)
	}

	readOnly := Characteristic{ValueHandle: 7, Properties: PropRead}
	if err := Write(ConnID(1), readOnly, []byte("x")); !errors.Is(err, ErrBadProperty) {
		t.Errorf("Write(read-only) = %v, want ErrBadProperty", err)
	}
	if _, err := SubscribeChar(ConnID(1), readOnly); !errors.Is(err, ErrBadProperty) {
		t.Errorf("SubscribeChar(no notify) = %v, want ErrBadProperty", err)
	}
	writable := Characteristic{ValueHandle: 7, Properties: PropRead | PropWriteNoRsp}
	if err := Write(ConnID(1), writable, []byte("x")); !errors.Is(err, ErrNotReady) {
		t.Errorf("Write(writable, no stack) = %v, want ErrNotReady", err)
	}
	if err := WriteWithResponse(ConnID(1), writable, make([]byte, MaxData+1)); !errors.Is(err, ErrTooLong) {
		t.Errorf("Write(too long) = %v, want ErrTooLong", err)
	}
	if !readOnly.CanRead() || readOnly.CanWrite() || readOnly.CanNotify() {
		t.Error("CanRead/CanWrite/CanNotify disagree with the properties")
	}
	notify := Characteristic{ValueHandle: 9, Properties: PropNotify}
	if !notify.CanNotify() || notify.CanWrite() {
		t.Error("CanNotify/CanWrite disagree with the properties")
	}
}

// TestGattStubUnsupported pins the hosted behaviour of the GATT entry points:
// without a stack they report ErrNotReady and never a forged result.
func TestGattStubUnsupported(t *testing.T) {
	chr := Characteristic{ValueHandle: 3, Properties: PropRead | PropWrite | PropNotify}
	if _, err := Services(ConnID(1)); !errors.Is(err, ErrNotReady) {
		t.Errorf("Services = %v, want ErrNotReady", err)
	}
	if _, err := Characteristics(ConnID(1), UUID16(0xfff0)); !errors.Is(err, ErrNotReady) {
		t.Errorf("Characteristics = %v, want ErrNotReady", err)
	}
	if _, err := Read(ConnID(1), chr); !errors.Is(err, ErrNotReady) {
		t.Errorf("Read = %v, want ErrNotReady", err)
	}
	if err := Write(ConnID(1), chr, []byte("x")); !errors.Is(err, ErrNotReady) {
		t.Errorf("Write = %v, want ErrNotReady", err)
	}
	if _, err := SubscribeChar(ConnID(1), chr); !errors.Is(err, ErrNotReady) {
		t.Errorf("SubscribeChar = %v, want ErrNotReady", err)
	}
	if _, err := MTU(ConnID(1)); !errors.Is(err, ErrNotReady) {
		t.Errorf("MTU = %v, want ErrNotReady", err)
	}
}

// TestEventPayload pins the payload accessors and the notification filter: a
// characteristic subscription takes only its own notifications.
func TestEventPayload(t *testing.T) {
	ev := Event{Kind: EvNotify, ConnID: ConnID(0x10001), AttHandle: 18, DataLen: 3}
	copy(ev.Data[:], "abc")
	if got := string(ev.DataBytes()); got != "abc" {
		t.Fatalf("DataBytes = %q, want %q", got, "abc")
	}
	if got := ev.MTU(); got != 0 {
		t.Fatalf("MTU of a non-MTU event = %d, want 0", got)
	}
	mtuEv := Event{Kind: EvMtu, ConnID: ConnID(0x10001), Status: 256}
	if got := mtuEv.MTU(); got != 256 {
		t.Fatalf("MTU = %d, want 256", got)
	}

	charSub := &Subscription{notif: &notifTarget{conn: ConnID(0x10001), handle: 18}}
	if !charSub.delivers(&ev) {
		t.Error("characteristic subscription rejected its own notification")
	}
	if charSub.delivers(&Event{Kind: EvNotify, ConnID: ConnID(0x10002), AttHandle: 18}) {
		t.Error("characteristic subscription took another connection's notification")
	}
	if charSub.delivers(&Event{Kind: EvNotify, ConnID: ConnID(0x10001), AttHandle: 19}) {
		t.Error("characteristic subscription took another handle's notification")
	}
	if charSub.delivers(&Event{Kind: EvAdv}) {
		t.Error("characteristic subscription took a scan result")
	}
	if !charSub.delivers(&Event{Kind: EvOverflow, Status: 1}) {
		t.Error("characteristic subscription missed the mailbox overflow report")
	}
	kindSub := &Subscription{kinds: []EventKind{EvAdv}}
	if !kindSub.delivers(&Event{Kind: EvAdv}) || kindSub.delivers(&ev) {
		t.Error("kind filter did not select by kind")
	}
}

// TestSentinels pins the Go-domain sentinels in their migrated shape: the
// human text renders first with the segment code as the annotation, and the
// code-era checks keep working - errors.Is against a raw literal of the
// segment value, the annotation reachable through esperr.Code and errors.As,
// and the sentinel matching itself by identity.
func TestSentinels(t *testing.T) {
	for _, c := range []struct {
		err  *esperr.Error
		code int32
		text string
	}{
		{ErrTimeout, codeTimeout, "esp/ble: timeout (code 31489)"},
		{ErrNotReady, codeNotReady, "esp/ble: not ready (code 31490)"},
		{ErrUnsupported, codeUnsupported, "esp/ble: unsupported (code 32512)"},
		{ErrBadUUID, codeBadUUID, "esp/ble: bad uuid (code 31491)"},
		{ErrBadValue, codeBadValue, "esp/ble: bad value (code 31492)"},
		{ErrBusy, codeBusy, "esp/ble: busy (code 31493)"},
		{ErrBadProperty, codeBadProperty, "esp/ble: bad property (code 31494)"},
		{ErrNotFound, codeNotFound, "esp/ble: not found (code 31495)"},
		{ErrTooLong, codeTooLong, "esp/ble: too long (code 31496)"},
		{ErrTooMany, codeTooMany, "esp/ble: too many (code 31497)"},
	} {
		if got := esperr.Code(c.err); got != c.code {
			t.Errorf("esperr.Code(%v) = %d, want %d", c.err, got, c.code)
		}
		if got := c.err.Error(); got != c.text {
			t.Errorf("Error() = %q, want %q", got, c.text)
		}
		if !errors.Is(error(c.err), c.err) {
			t.Errorf("errors.Is does not match the %q sentinel by identity", c.text)
		}
		if !errors.Is(error(c.err), &esperr.Error{Code: c.code}) {
			t.Errorf("the %q sentinel does not match a code-era literal of %#x", c.text, c.code)
		}
		var e *esperr.Error
		if !errors.As(error(c.err), &e) || e.Code != c.code {
			t.Errorf("errors.As did not surface the %#x annotation", c.code)
		}
	}
}

// TestStubReportsUnsupported pins the honest-stub contract: no call fakes
// success, and a control call before Init is ErrNotReady rather than a status.
// Backend names the compiled-in host stack: a host build carries none, and
// the report must say so instead of guessing.
func TestBackendHosted(t *testing.T) {
	if got := Backend(); got != "none" {
		t.Fatalf("Backend() on the host = %q, want %q", got, "none")
	}
}

func TestStubReportsUnsupported(t *testing.T) {
	if Ready() {
		t.Fatal("a host build reported a synced host")
	}
	if err := Init(); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Init = %v, want ErrUnsupported", err)
	}
	if err := Deinit(); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Deinit = %v, want ErrUnsupported", err)
	}
	if _, err := StartScan(ScanOptions{}); !errors.Is(err, ErrNotReady) {
		t.Fatalf("StartScan before Init = %v, want ErrNotReady", err)
	}
	if _, err := Connect(Peer{AddrType: 1}, ConnectOptions{}); !errors.Is(err, ErrNotReady) {
		t.Fatalf("Connect before Init = %v, want ErrNotReady", err)
	}
	if err := Disconnect(ConnID(0x10001)); !errors.Is(err, ErrNotReady) {
		t.Fatalf("Disconnect before Init = %v, want ErrNotReady", err)
	}
}

// TestAddrString checks the little-endian-to-notation direction: the stack
// stores the least significant octet first, the printed form is the reverse.
func TestAddrString(t *testing.T) {
	got := AddrString([6]uint8{0x00, 0x00, 0xEE, 0xFF, 0xC0, 0x11})
	if want := "11:c0:ff:ee:00:00"; got != want {
		t.Fatalf("AddrString = %q, want %q", got, want)
	}
}

// TestAdvName walks the AD structures a scan result carries.
func TestAdvName(t *testing.T) {
	// flags (0x01, one byte) followed by the complete local name (0x09).
	adv := []uint8{0x02, 0x01, 0x06, 0x0A, 0x09, 'l', 'l', 'g', 'o', '-', 'p', 'e', 'e', 'r'}
	if got, want := AdvName(adv), "llgo-peer"; got != want {
		t.Fatalf("AdvName = %q, want %q", got, want)
	}
	// Shortened local name when no complete name is present.
	short := []uint8{0x04, 0x08, 'a', 'b', 'c'}
	if got, want := AdvName(short), "abc"; got != want {
		t.Fatalf("AdvName(shortened) = %q, want %q", got, want)
	}
	// No name, a truncated structure, and an empty payload all report "".
	if got := AdvName([]uint8{0x02, 0x01, 0x06}); got != "" {
		t.Fatalf("AdvName(no name) = %q, want empty", got)
	}
	if got := AdvName([]uint8{0x09, 0x09, 'x'}); got != "" {
		t.Fatalf("AdvName(truncated) = %q, want empty", got)
	}
	if got := AdvName(nil); got != "" {
		t.Fatalf("AdvName(nil) = %q, want empty", got)
	}
}

// TestErrorSpaces pins the domain separation: a NimBLE status is tagged out of
// the esp_err range, esp_err and Go-level codes never masquerade as statuses.
func TestErrorSpaces(t *testing.T) {
	err := ErrStatus(0x213) // HCI_REMOTE_USER_TERM_CONN
	if err == nil {
		t.Fatal("ErrStatus(0x213) = nil")
	}
	if got := Status(err); got != 0x213 {
		t.Fatalf("Status = 0x%x, want 0x213", got)
	}
	if !errors.Is(err, ErrStatus(0x213)) {
		t.Fatal("errors.Is did not match the same status")
	}
	if errors.Is(err, ErrStatus(0x02)) {
		t.Fatal("errors.Is matched a different status")
	}
	// An esp_err value and this package's own codes stay outside the status
	// space, so Status reports -1 for them.
	if got := Status(ErrUnsupported); got != -1 {
		t.Fatalf("Status(ErrUnsupported) = %d, want -1", got)
	}
	if got := Status(espErr(0x106)); got != -1 {
		t.Fatalf("Status(espErr) = %d, want -1", got)
	}
	if got := Status(nil); got != 0 {
		t.Fatalf("Status(nil) = %d, want 0", got)
	}
}

// TestErrNames pins the name table entries the demo log relies on.
func TestErrNames(t *testing.T) {
	cases := []struct {
		code int32
		want string
	}{
		{0, "ESP_OK"},
		{0x101, "ESP_ERR_NO_MEM"},
		{0x106, "ESP_ERR_NOT_SUPPORTED"},
		{NimbleBase + 0x02, "BLE_HS_EALREADY"},
		{NimbleBase + 0x0D, "BLE_HS_ETIMEOUT"},
		{NimbleBase + 0x213, "HCI_REMOTE_USER_TERM_CONN"},
		{codeTimeout, "BLE_ERR_TIMEOUT"},
		{codeNotReady, "BLE_ERR_NOT_READY"},
		{codeTooMany, "BLE_ERR_TOO_MANY"},
		{codeUnsupported, "BLE_ERR_UNSUPPORTED"},
	}
	for _, c := range cases {
		if got := ErrName(c.code); got != c.want {
			t.Errorf("ErrName(0x%X) = %q, want %q", c.code, got, c.want)
		}
	}
}

func TestDisconnectReasonName(t *testing.T) {
	cases := map[int32]string{
		0x00:  "success",
		0x08:  "connection timeout (supervision timeout)",
		0x13:  "remote user terminated the connection",
		0x16:  "local host terminated the connection",
		0x3b:  "unacceptable connection interval",
		0x100: "connection establish failed",
	}
	for code, want := range cases {
		if got := DisconnectReasonName(code); got != want {
			t.Errorf("DisconnectReasonName(%#x) = %q, want %q", code, got, want)
		}
	}
	if got := DisconnectReasonName(0x7e); got != "reason 0x7e" {
		t.Errorf("unknown reason = %q, want the hex form", got)
	}
}

// resetRegistry clears the peripheral registry between tests. The registry is
// package state; every peripheral test starts from the same blank slate, and
// the go test runner is serial within one package, so no locking is needed.
func resetRegistry() {
	gsvcs = nil
	gchrs = nil
	committed = false
}

// TestWirePeripheralLayout pins the layout of the peripheral declaration
// records, the same contract the glue's _Static_asserts pin on the C side.
func TestWirePeripheralLayout(t *testing.T) {
	var s gsvcWire
	if got, want := unsafe.Sizeof(s), uintptr(20); got != want {
		t.Errorf("sizeof gsvcWire = %d, want %d", got, want)
	}
	if got, want := unsafe.Offsetof(s.NChars), uintptr(18); got != want {
		t.Errorf("gsvcWire.NChars offset = %d, want %d", got, want)
	}
	var c gchrWire
	if got, want := unsafe.Sizeof(c), uintptr(18); got != want {
		t.Errorf("sizeof gchrWire = %d, want %d", got, want)
	}
	if got, want := unsafe.Offsetof(c.Props), uintptr(17); got != want {
		t.Errorf("gchrWire.Props offset = %d, want %d", got, want)
	}
}

// TestAdvPayloads walks the advertising payload composition: the default
// flags packet, complete and shortened name placement, the scan-response
// overflow, raw passthrough, and the refusals.
func TestAdvPayloads(t *testing.T) {
	// The zero value: general-discovery flags, no name, connectable by
	// omission of NonConnectable.
	adv, rsp, err := buildAdvPayloads(AdvParams{})
	if err != nil {
		t.Fatalf("buildAdvPayloads(zero) = %v", err)
	}
	if want := []byte{0x02, 0x01, advFlagsGeneral}; string(adv) != string(want) || rsp != nil {
		t.Fatalf("zero params: adv = %v rsp = %v, want flags-only and nil rsp", adv, rsp)
	}

	// A short name rides the advertising payload as a Complete Local Name.
	adv, rsp, err = buildAdvPayloads(AdvParams{Name: "llgo-periph"})
	if err != nil {
		t.Fatalf("buildAdvPayloads(name) = %v", err)
	}
	if want := []byte{0x02, 0x01, advFlagsGeneral, 0x0c, 0x09, 'l', 'l', 'g', 'o', '-', 'p', 'e', 'r', 'i', 'p', 'h'}; string(adv) != string(want) || rsp != nil {
		t.Fatalf("short name: adv = %v rsp = %v, want name appended complete", adv, rsp)
	}

	// A raw payload the caller brought is used verbatim; the name is appended
	// to it when it fits (adv-first is the rule - a passive scanner never
	// fetches the scan response).
	raw := []byte{0x03, 0x02, 0x12, 0x18}
	adv, rsp, err = buildAdvPayloads(AdvParams{Name: "n", AdvData: raw})
	if err != nil {
		t.Fatalf("buildAdvPayloads(raw) = %v", err)
	}
	want := append(append([]byte{}, raw...), 0x02, 0x09, 'n')
	if string(adv) != string(want) || rsp != nil {
		t.Fatalf("raw adv: %v rsp = %v, want name appended to the raw payload", adv, rsp)
	}

	// A raw payload that leaves no room for even a short name sends it to the
	// scan response, shortened to the packet size, complete there because the
	// whole 30-character name fits one response packet.
	nearly := make([]byte, MaxAdvData-4)
	adv, rsp, err = buildAdvPayloads(AdvParams{Name: "012345678901234567890123456789", AdvData: nearly})
	if err != nil {
		t.Fatalf("buildAdvPayloads(long) = %v", err)
	}
	// The whole 30-character name cannot fit any packet (structure 32 > 31),
	// so it travels shortened - and even the response packet only takes the
	// 29-character prefix (1 length byte + 1 type byte + 29).
	if string(adv) != string(nearly) || len(rsp) != 31 || rsp[1] != 0x08 || string(rsp[2:]) != "01234567890123456789012345678" {
		t.Fatalf("long name: adv = %v rsp = %v, want the shortened name in the scan response", adv, rsp)
	}

	// A caller-owned scan response is respected: nothing is appended to it.
	ownRsp := make([]byte, 30)
	adv, rsp, err = buildAdvPayloads(AdvParams{Name: "0123456789", AdvData: raw, ScanRsp: ownRsp})
	if err != nil {
		t.Fatalf("buildAdvPayloads(own rsp) = %v", err)
	}
	if string(rsp) != string(ownRsp) || len(adv) != len(raw)+12 {
		// the name structure rides the advertising packet: 2 + 10 bytes
		t.Fatalf("own rsp rewritten or name misplaced: adv len %d rsp len %d", len(adv), len(rsp))
	}

	// Nothing fits: a full advertising packet plus a full scan response.
	full := make([]byte, MaxAdvData)
	if _, _, err := buildAdvPayloads(AdvParams{Name: "x", AdvData: full, ScanRsp: full}); err == nil {
		t.Fatal("a name with nowhere to go was accepted")
	}
	// Nothing fits anywhere: a nearly-full advertising packet, a nearly-full
	// caller-owned response, and a name of four characters.
	tight := make([]byte, MaxAdvData-4)
	if _, _, err := buildAdvPayloads(AdvParams{Name: "0123", AdvData: tight, ScanRsp: make([]byte, MaxAdvData-1)}); err == nil {
		t.Fatal("a name with nowhere to go was accepted")
	}
	// More than one packet of raw data, on either side.
	if _, _, err := buildAdvPayloads(AdvParams{AdvData: make([]byte, MaxAdvData+1)}); err != ErrTooMany {
		t.Fatalf("oversized AdvData = %v, want ErrTooMany", err)
	}
	if _, _, err := buildAdvPayloads(AdvParams{ScanRsp: make([]byte, MaxAdvData+1)}); err != ErrTooMany {
		t.Fatalf("oversized ScanRsp = %v, want ErrTooMany", err)
	}
}

// TestRegistry walks the local GATT server's registry: the argument checks
// both Add* functions run, the handle-budget check, and the refusal of a
// registry beyond the static bounds.
func TestRegistry(t *testing.T) {
	resetRegistry()
	svc := UUID16(0xfff0)
	chr := UUID16(0xfff1)
	not := UUID16(0xfff2)

	if err := AddService(UUID{}, 4); !errors.Is(err, ErrBadUUID) {
		t.Errorf("AddService(zero uuid) = %v, want ErrBadUUID", err)
	}
	if err := AddService(svc, 0); !errors.Is(err, ErrBadValue) {
		t.Errorf("AddService(no handles) = %v, want ErrBadValue", err)
	}
	if err := AddService(svc, 8); err != nil {
		t.Fatalf("AddService = %v", err)
	}
	if err := AddService(svc, 8); !errors.Is(err, ErrBadUUID) {
		t.Errorf("duplicate AddService = %v, want ErrBadUUID", err)
	}
	if err := AddCharacteristic(UUID{}, chr, PropRead); !errors.Is(err, ErrBadUUID) {
		t.Errorf("AddCharacteristic(zero svc) = %v, want ErrBadUUID", err)
	}
	if err := AddCharacteristic(svc, UUID{}, PropRead); !errors.Is(err, ErrBadUUID) {
		t.Errorf("AddCharacteristic(zero chr) = %v, want ErrBadUUID", err)
	}
	if err := AddCharacteristic(svc, chr, PropBroadcast); !errors.Is(err, ErrBadProperty) {
		t.Errorf("AddCharacteristic(broadcast) = %v, want ErrBadProperty", err)
	}
	if err := AddCharacteristic(UUID16(0x1111), chr, PropRead); !errors.Is(err, ErrNotFound) {
		t.Errorf("AddCharacteristic(unknown svc) = %v, want ErrNotFound", err)
	}
	if err := AddCharacteristic(svc, chr, PropRead|PropWrite); err != nil {
		t.Fatalf("AddCharacteristic = %v", err)
	}
	if err := AddCharacteristic(svc, chr, PropRead); !errors.Is(err, ErrBadUUID) {
		t.Errorf("duplicate AddCharacteristic = %v, want ErrBadUUID", err)
	}
	// The budget the service declared is checked at commit time: one
	// characteristic with a CCCD needs 4 handles; 8 covers it.
	if err := checkRegistry(); err != nil {
		t.Fatalf("checkRegistry = %v, want nil", err)
	}
	// Shrink the declared budget: the same registry must now be refused.
	resetRegistry()
	if err := AddService(svc, 2); err != nil {
		t.Fatalf("AddService = %v", err)
	}
	if err := AddCharacteristic(svc, chr, PropNotify); err != nil {
		t.Fatalf("AddCharacteristic = %v", err)
	}
	if err := checkRegistry(); !errors.Is(err, ErrBadValue) {
		t.Errorf("checkRegistry(tight budget) = %v, want ErrBadValue", err)
	}

	// Static bounds.
	resetRegistry()
	for i := 0; i < MaxLocalServices; i++ {
		if err := AddService(UUID16(uint16(0xffa0+i)), 2); err != nil {
			t.Fatalf("AddService %d = %v", i, err)
		}
	}
	if err := AddService(UUID16(0xbfff), 2); !errors.Is(err, ErrTooMany) {
		t.Errorf("MaxLocalServices+1 = %v, want ErrTooMany", err)
	}
	resetRegistry()
	if err := AddService(svc, 32); err != nil {
		t.Fatalf("AddService = %v", err)
	}
	for i := 0; i < MaxLocalChars; i++ {
		if err := AddCharacteristic(svc, UUID16(uint16(0xffc0+i)), PropRead); err != nil {
			t.Fatalf("AddCharacteristic %d = %v", i, err)
		}
	}
	if err := AddCharacteristic(svc, not, PropRead); !errors.Is(err, ErrTooMany) {
		t.Errorf("MaxLocalChars+1 = %v, want ErrTooMany", err)
	}
}

// TestPeriphStubUnsupported pins the hosted behaviour of the peripheral entry
// points: before a commit every call is refused, and none forges success.
func TestPeriphStubUnsupported(t *testing.T) {
	resetRegistry()
	// A valid registry gets past the caller-side checks and hits the
	// stack-state refusal; an empty one is the caller's bug (ErrBadValue,
	// pinned in TestCommitRefusals).
	if err := AddService(UUID16(0xfff0), 4); err != nil {
		t.Fatalf("AddService = %v", err)
	}
	if err := CommitServices(); !errors.Is(err, ErrNotReady) {
		t.Errorf("CommitServices = %v, want ErrNotReady", err)
	}
	if _, err := LocalHandle(UUID16(0xfff1)); !errors.Is(err, ErrNotReady) {
		t.Errorf("LocalHandle before commit = %v, want ErrNotReady", err)
	}
	if err := WriteCharValue(4, []byte("x")); !errors.Is(err, ErrNotReady) {
		t.Errorf("WriteCharValue = %v, want ErrNotReady", err)
	}
	if err := Notify(ConnID(1), 4, []byte("x")); !errors.Is(err, ErrNotReady) {
		t.Errorf("Notify = %v, want ErrNotReady", err)
	}
	if err := Indicate(ConnID(1), 4, []byte("x")); !errors.Is(err, ErrNotReady) {
		t.Errorf("Indicate = %v, want ErrNotReady", err)
	}
	if err := DisconnectPeer(ConnID(1)); !errors.Is(err, ErrNotReady) {
		t.Errorf("DisconnectPeer = %v, want ErrNotReady", err)
	}
	if err := SetSecurityParams(SecurityParams{}); !errors.Is(err, ErrNotReady) {
		t.Errorf("SetSecurityParams = %v, want ErrNotReady", err)
	}
	if _, err := StartAdvertising(AdvParams{}); !errors.Is(err, ErrNotReady) {
		t.Errorf("StartAdvertising = %v, want ErrNotReady", err)
	}
	if err := StopAdvertising(); !errors.Is(err, ErrNotReady) {
		t.Errorf("StopAdvertising = %v, want ErrNotReady", err)
	}

	// Argument checks that run before the stack-state check: the oversized
	// payloads and the bad handles are refused on the host exactly as on the
	// device, and the advertising payload composition is exercised for real.
	if _, err := StartAdvertising(AdvParams{AdvData: make([]byte, MaxAdvData+1)}); !errors.Is(err, ErrTooMany) {
		t.Errorf("StartAdvertising(oversize) = %v, want ErrTooMany", err)
	}
	if err := WriteCharValue(0, []byte("x")); !errors.Is(err, ErrBadValue) {
		t.Errorf("WriteCharValue(0) = %v, want ErrBadValue", err)
	}
	if err := WriteCharValue(4, make([]byte, MaxData+1)); !errors.Is(err, ErrTooLong) {
		t.Errorf("WriteCharValue(too long) = %v, want ErrTooLong", err)
	}
	if err := Notify(ConnID(1), 0, []byte("x")); !errors.Is(err, ErrBadValue) {
		t.Errorf("Notify(0) = %v, want ErrBadValue", err)
	}
	if err := Indicate(ConnID(1), 4, make([]byte, MaxData+1)); !errors.Is(err, ErrTooLong) {
		t.Errorf("Indicate(too long) = %v, want ErrTooLong", err)
	}
	resetRegistry()
}

// TestCommitRegistryPins the commit path's registry-side refusals: an empty
// registry, a double commit, and additions after the freeze. The commit
// itself reports ErrNotReady on the host (no stack), so the freeze never
// happens here - additions-after-commit are exercised by resetting the flag.
func TestCommitRefusals(t *testing.T) {
	resetRegistry()
	if err := CommitServices(); !errors.Is(err, ErrBadValue) {
		t.Errorf("CommitServices(empty) = %v, want ErrBadValue", err)
	}
	resetRegistry()
	if err := AddService(UUID16(0xfff0), 8); err != nil {
		t.Fatalf("AddService = %v", err)
	}
	committed = true // pretend the commit went through
	if err := CommitServices(); !errors.Is(err, ErrBadValue) {
		t.Errorf("CommitServices(twice) = %v, want ErrBadValue", err)
	}
	if err := AddService(UUID16(0xffe0), 4); !errors.Is(err, ErrBadValue) {
		t.Errorf("AddService(after commit) = %v, want ErrBadValue", err)
	}
	if err := AddCharacteristic(UUID16(0xfff0), UUID16(0xfff1), PropRead); !errors.Is(err, ErrBadValue) {
		t.Errorf("AddCharacteristic(after commit) = %v, want ErrBadValue", err)
	}
	resetRegistry()
}

// TestLocalHandle pins the handle lookup: unknown UUIDs after a (simulated)
// commit, and the found case.
func TestLocalHandle(t *testing.T) {
	resetRegistry()
	if err := AddService(UUID16(0xfff0), 8); err != nil {
		t.Fatalf("AddService = %v", err)
	}
	if err := AddCharacteristic(UUID16(0xfff0), UUID16(0xfff1), PropRead|PropNotify); err != nil {
		t.Fatalf("AddCharacteristic = %v", err)
	}
	committed = true
	gchrs[0].handle = 16 // the assignment CommitServices pulls from the glue
	if h, err := LocalHandle(UUID16(0xfff1)); err != nil || h != 16 {
		t.Errorf("LocalHandle = %d, %v, want 16, nil", h, err)
	}
	if _, err := LocalHandle(UUID16(0x1234)); !errors.Is(err, ErrNotFound) {
		t.Errorf("LocalHandle(unknown) = %v, want ErrNotFound", err)
	}
	resetRegistry()
}

// TestGuardWrapShapes pins the argument-refusal guards' migrated shape: the
// sentinel rides under the operation's esperr.Wrap prefix, the rendering is
// "ble: <op>: esp/ble: <text> (code N)", and errors.Is / errors.As /
// esperr.Code all walk through the wrap. The state and wait sentinels stay
// bare - their text already names the package and condition - which the two
// identity checks at the end pin.
func TestGuardWrapShapes(t *testing.T) {
	zeroChr := Characteristic{} // ValueHandle 0: never a real attribute
	readOnly := Characteristic{ValueHandle: 7, Properties: PropRead}
	writable := Characteristic{ValueHandle: 7, Properties: PropRead | PropWriteNoRsp}

	cCases := []struct {
		name string
		err  error
		text string
	}{
		{"read zero handle", func() error { _, e := Read(ConnID(1), zeroChr); return e }(),
			"ble: read: esp/ble: bad value (code 31492)"},
		{"write zero handle", Write(ConnID(1), zeroChr, []byte("x")),
			"ble: write: esp/ble: bad value (code 31492)"},
		{"write read-only", Write(ConnID(1), readOnly, []byte("x")),
			"ble: write: esp/ble: bad property (code 31494)"},
		{"write too long", WriteWithResponse(ConnID(1), writable, make([]byte, MaxData+1)),
			"ble: write: esp/ble: too long (code 31496)"},
		{"subscribe char zero handle", func() error { _, e := SubscribeChar(ConnID(1), zeroChr); return e }(),
			"ble: subscribe char: esp/ble: bad value (code 31492)"},
		{"unsubscribe nil", Unsubscribe(nil),
			"ble: unsubscribe: esp/ble: bad value (code 31492)"},
		{"characteristics zero uuid", func() error { _, e := Characteristics(ConnID(1), UUID{}); return e }(),
			"ble: characteristics: esp/ble: bad uuid (code 31491)"},
		{"add service zero uuid", AddService(UUID{}, 4),
			"ble: add service: esp/ble: bad uuid (code 31491)"},
		{"commit empty registry", func() error { resetRegistry(); defer resetRegistry(); return CommitServices() }(),
			"ble: commit services: esp/ble: bad value (code 31492)"},
		{"write char value zero handle", WriteCharValue(0, []byte("x")),
			"ble: write char value: esp/ble: bad value (code 31492)"},
		{"notify zero handle", Notify(ConnID(1), 0, []byte("x")),
			"ble: notify: esp/ble: bad value (code 31492)"},
		{"indicate too long", Indicate(ConnID(1), 4, make([]byte, MaxData+1)),
			"ble: indicate: esp/ble: too long (code 31496)"},
		{"start advertising oversize", func() error { _, e := StartAdvertising(AdvParams{AdvData: make([]byte, MaxAdvData+1)}); return e }(),
			"ble: start advertising: esp/ble: too many (code 31497)"},
	}
	for _, c := range cCases {
		if got := c.err.Error(); got != c.text {
			t.Errorf("%s: Error() = %q, want %q", c.name, got, c.text)
		}
		var e *esperr.Error
		if !errors.As(c.err, &e) {
			t.Errorf("%s: errors.As did not walk through the wrap", c.name)
		} else if got := esperr.Code(c.err); got != e.Code {
			t.Errorf("%s: esperr.Code = %d, want the annotation %d", c.name, got, e.Code)
		}
	}

	// The sentinel behind a wrapped guard still matches by identity (the
	// wrap unwraps to it) and by its code-era literal.
	err := Unsubscribe(nil)
	if !errors.Is(err, ErrBadValue) || !errors.Is(err, &esperr.Error{Code: codeBadValue}) {
		t.Fatal("the wrapped guard lost its sentinel or code identity")
	}

	// State and wait sentinels return bare: the pointer identity a demo's
	// sentinel comparison relies on stays intact.
	if _, err := StartScan(ScanOptions{}); err != ErrNotReady {
		t.Fatalf("StartScan before Init = %v, want the bare ErrNotReady sentinel", err)
	}
	sub := &Subscription{kinds: []EventKind{EvAdv}}
	if _, err := sub.Next(0); err != ErrTimeout {
		t.Fatalf("Next(0) = %v, want the bare ErrTimeout sentinel", err)
	}
}
