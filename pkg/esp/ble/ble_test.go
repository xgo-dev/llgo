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

// TestErrNameTable pins that every sentinel of this package carries the name
// ErrName reports for its code - the two tables must not drift.
func TestErrNameTable(t *testing.T) {
	for _, e := range []*esperr.Error{
		ErrTimeout, ErrNotReady, ErrUnsupported, ErrBadUUID, ErrBadValue,
		ErrBusy, ErrBadProperty, ErrNotFound, ErrTooLong,
	} {
		if e.Name == "" {
			t.Errorf("sentinel %d has no name", e.Code)
			continue
		}
		if got := ErrName(e.Code); got != e.Name {
			t.Errorf("ErrName(%#x) = %q, want %q", e.Code, got, e.Name)
		}
		if !errors.Is(error(e), e) {
			t.Errorf("errors.Is(error(%q), sentinel) = false", e.Name)
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
		0x00: "success",
		0x08: "connection timeout (supervision timeout)",
		0x13: "remote user terminated the connection",
		0x16: "local host terminated the connection",
		0x3b: "unacceptable connection interval",
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
