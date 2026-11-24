//go:build !llgo.idf

// Host-build tests. They exercise what a host build can exercise: the stub
// path (no BLE in this build) and the pure-Go pieces - address and advertising
// payload rendering, and the three error spaces. Device behavior is verified
// under esp-emu and on hardware, not here.
package ble

import (
	"errors"
	"testing"
)

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
