//go:build !llgo.idf && (!llgo || !baremetal)

// Host-build tests for the i2c surface. A host build has no glue, so
// nothing here claims a bus exists: the tests pin what the pure-Go layer
// owns - the honest-stub contract (the driver path reports errUnsupported
// instead of faking a transfer), the Go-side guards (the nil/zero-value
// handle refusals and the idempotent-Close ErrClosed, with their op
// prefixes), and the error shape throughout (errors.Is by sentinel identity
// and by code, the code annotation reachable through esperr.Code and
// errors.As).
package i2c

import (
	"errors"
	"testing"

	"github.com/xgo-dev/llgo/pkg/esp/esperr"
)

// TestStubReportsUnsupported pins the honest-stub contract: with no glue in
// the link, every driver path reports errUnsupported (0x7F00) instead of
// faking a transfer. The errors arrive in the new shape (bare Coded values)
// but match the code-era literals the same way.
func TestStubReportsUnsupported(t *testing.T) {
	unsupported := &esperr.Error{Code: errUnsupported}
	bus, err := NewBus(8, 9)
	if !errors.Is(err, unsupported) {
		t.Fatalf("NewBus on a host build = %v, want errUnsupported from the stub", err)
	}
	if bus != nil {
		t.Fatal("NewBus on a host build returned a bus, want nil")
	}
	var zeroBus *Bus
	if _, err := zeroBus.Device(0x50, 100000); !errors.Is(err, ErrInvalidArg) {
		t.Fatalf("Device on a nil bus = %v, want the ErrInvalidArg sentinel", err)
	}
	zeroDev := &Device{}
	if err := zeroDev.Write([]byte{0}); !errors.Is(err, ErrInvalidArg) {
		t.Fatalf("Write on a zero device = %v, want the ErrInvalidArg sentinel", err)
	}
}

// TestInvalidArgGuard pins the Go-side misuse guard end to end: the
// sentinel matches by identity, by its 0x102 code annotation (against a
// code-era literal), the annotation stays reachable through esperr.Code and
// errors.As through the Wrap, and the rendering names the operation first
// (the "i2c: <op>" op prefix) with the human text and code annotation after.
func TestInvalidArgGuard(t *testing.T) {
	var zeroDev *Device
	err := zeroDev.WriteRead([]byte{0xFD}, make([]byte, 1))
	if !errors.Is(err, ErrInvalidArg) {
		t.Fatalf("WriteRead on a nil device = %v, want the ErrInvalidArg sentinel", err)
	}
	if !errors.Is(err, &esperr.Error{Code: errInvalidArg}) {
		t.Fatal("the guard error did not match a raw 0x102 Error (code path broken)")
	}
	if got := esperr.Code(err); got != errInvalidArg {
		t.Fatalf("esperr.Code = %#x, want 0x102", got)
	}
	var e *esperr.Error
	if !errors.As(err, &e) || e.Code != errInvalidArg {
		t.Fatal("errors.As did not surface the *esperr.Error with the 0x102 annotation")
	}
	if got, want := err.Error(), "i2c: write transaction: esp/i2c: invalid argument (code 258)"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
	// Every guard names its own operation.
	cases := []struct {
		name string
		err  error
		op   string
	}{
		{"nil bus Device", func() error { _, e := (*Bus)(nil).Device(0x50, 100000); return e }(), "i2c: bind device"},
		{"nil device Write", (*Device)(nil).Write([]byte{0}), "i2c: write"},
		{"nil device Read", (*Device)(nil).Read(make([]byte, 1)), "i2c: read"},
		{"nil device WriteRead", (*Device)(nil).WriteRead(nil, make([]byte, 1)), "i2c: write transaction"},
	}
	for _, c := range cases {
		if !errors.Is(c.err, ErrInvalidArg) {
			t.Errorf("%s = %v, want the ErrInvalidArg sentinel", c.name, c.err)
		}
		if got := c.err.Error(); len(got) < len(c.op) || got[:len(c.op)] != c.op {
			t.Errorf("%s Error() = %q, want the %q op prefix", c.name, got, c.op)
		}
	}
}

// TestClosedSentinel pins the idempotent-Close contract in the new shape:
// a Close on a nil or consumed handle reports esperr.ErrClosed (the family's
// transport sentinel, matched by identity and by its EBADF code) under the
// "i2c: close ..." op, so the rendering is "op: closed (code 9)".
func TestClosedSentinel(t *testing.T) {
	err := (*Device)(nil).Close()
	if !errors.Is(err, esperr.ErrClosed) {
		t.Fatalf("nil Device Close = %v, want esperr.ErrClosed", err)
	}
	if got := esperr.Code(err); got != 9 {
		t.Fatalf("esperr.Code = %d, want the EBADF 9 annotation", got)
	}
	if got, want := err.Error(), "i2c: close device: closed (code 9)"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
	err = (*Bus)(nil).Close()
	if !errors.Is(err, esperr.ErrClosed) {
		t.Fatalf("nil Bus Close = %v, want esperr.ErrClosed", err)
	}
	if got, want := err.Error(), "i2c: close bus: closed (code 9)"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
	if errors.Is(err, esperr.ErrTimeout) {
		t.Fatal("the closed error matched ErrTimeout - sentinel isolation broken")
	}
}

// TestStubProbeReportsUnsupported pins the honest-stub contract for the probe
// primitive: with no driver in the link a scan reports errUnsupported on the
// first address instead of reporting an empty (or full) bus.
func TestStubProbeReportsUnsupported(t *testing.T) {
	bus := &Bus{handle: 1} // white-box: a handle the stub backend cannot serve
	found, err := bus.Scan(0x08, 0x77)
	if !errors.Is(err, &esperr.Error{Code: errUnsupported}) {
		t.Fatalf("Scan on a host build = %v, want errUnsupported from the stub", err)
	}
	if len(found) != 0 {
		t.Fatalf("Scan on a host build found %v, want nothing", found)
	}
}

// TestScanGuards pins the two refusals a scan performs before it reaches a
// backend: a nil bus and an inverted range, both reported as ErrInvalidArg
// under the "i2c: scan" operation.
func TestScanGuards(t *testing.T) {
	_, err := (*Bus)(nil).Scan(0x08, 0x77)
	if !errors.Is(err, ErrInvalidArg) {
		t.Fatalf("Scan on a nil bus = %v, want ErrInvalidArg", err)
	}
	if got, want := err.Error(), "i2c: scan: esp/i2c: invalid argument (code 258)"; got != want {
		t.Fatalf("nil-bus Scan error = %q, want %q", got, want)
	}
	_, err = (&Bus{handle: 1}).Scan(0x40, 0x20)
	if !errors.Is(err, ErrInvalidArg) {
		t.Fatalf("Scan with first > last = %v, want ErrInvalidArg", err)
	}
}
