//go:build !llgo.idf

// Host-build tests for the UART surface. A host build has no ports and no
// glue, so nothing here claims hardware exists: the tests pin what the
// pure-Go layer owns - the default frame format, the internal error segment
// with its sentinel renderings, the Wrap op context, the Read semantics'
// refusal paths, and the honest-stub contract that every driver binding
// reports errUnsupported instead of faking a port.
package uart

import (
	"errors"
	"testing"

	"github.com/xgo-dev/llgo/pkg/esp/esperr"
)

// TestDefaults pins the default frame format to the 8N1 the glue's Init
// installs (the uart_config_t fields).
func TestDefaults(t *testing.T) {
	c := DefaultConfig()
	if c.WordLen != Word8 || c.StopBits != Stop1 || c.Parity != ParityNone {
		t.Fatalf("DefaultConfig = %+v, want the 8N1 frame the Init installs", c)
	}
	if UART1 != Port(1) || UART2 != Port(2) || UART0 != Port(0) {
		t.Fatal("the Port constants drifted from the driver's port numbering")
	}
}

// TestErrorSegment pins the binding's internal segment values through the
// sentinels: 0x7801..0x7803 sit outside every real esp_err range, so a
// refusal made at the glue can never be mistaken for one made by the IDF
// driver. Each sentinel carries its segment code as the Error.Code
// annotation, so both match paths work: errors.Is against the sentinel (by
// identity) and against a raw-code Error (by code), and the rendering is the
// human text with the code annotation.
func TestErrorSegment(t *testing.T) {
	if got, want := ErrConsoleReserved.Error(), "esp/uart: console reserved (code 30721)"; got != want {
		t.Fatalf("ErrConsoleReserved.Error() = %q, want %q", got, want)
	}
	if got, want := ErrNotInit.Error(), "esp/uart: not initialized (code 30722)"; got != want {
		t.Fatalf("ErrNotInit.Error() = %q, want %q", got, want)
	}
	if got, want := ErrBadValue.Error(), "esp/uart: bad value (code 30723)"; got != want {
		t.Fatalf("ErrBadValue.Error() = %q, want %q", got, want)
	}
	// ErrTimeout carries the esp_err family's own timeout code (0x107) - the
	// value uart_wait_tx_done reports for an expired flush budget - so a raw
	// code from the driver matches the sentinel through the errors.Is code
	// path without any wrapping step.
	if esperr.Code(ErrTimeout) != 0x107 {
		t.Fatalf("ErrTimeout carries code 0x%X, want the esp_err ESP_ERR_TIMEOUT 0x107", esperr.Code(ErrTimeout))
	}
	if !errors.Is(errOf(0x107), ErrTimeout) {
		t.Fatal("the driver's ESP_ERR_TIMEOUT did not match the ErrTimeout sentinel")
	}
	if !errors.Is(errOf(0x107), &esperr.Error{Code: 0x107}) {
		t.Fatal("ESP_ERR_TIMEOUT did not match a raw-code Error (code path broken)")
	}

	for _, tc := range []struct {
		name string
		code int32
		want error
	}{{"console", errConsoleReserved, ErrConsoleReserved},
		{"not-init", errNotInit, ErrNotInit},
		{"bad-value", errBadValue, ErrBadValue}} {
		if !errors.Is(errOf(tc.code), tc.want) {
			t.Errorf("the glue's 0x%04X refusal did not match its sentinel (%s)", tc.code, tc.name)
		}
		var e *esperr.Error
		if !errors.As(errOf(tc.code), &e) || e.Code != tc.code {
			t.Errorf("errors.As on the 0x%04X refusal did not surface the code annotation", tc.code)
		}
	}
	// A code outside the segment and outside the esp_err domain forwards as a
	// bare Coded value: the code-era rendering (no name table), the code as
	// the diagnostic.
	e := errOf(0x102)
	ce, ok := e.(*esperr.Error)
	if !ok || ce.Code != 0x102 || ce.Name != "" || ce.Error() != "esp error 258" {
		t.Fatalf("errOf(0x102) = %v, want a bare Coded forwarding of the code", e)
	}
	if err := errOf(0); err != nil {
		t.Fatalf("errOf(0) = %v, want nil", err)
	}
}

// TestInitRefusesConsole pins the binding-side refusal: port 0 is the
// console, Init rejects it before the glue runs - on a host build too, where
// every other call still reports errUnsupported. The Wrap prefix names the
// operation, and errors.Is walks through it.
func TestInitRefusesConsole(t *testing.T) {
	err := Init(UART0, 17, 18, 115200)
	if !errors.Is(err, ErrConsoleReserved) {
		t.Fatalf("Init(UART0, ...) = %v, want ErrConsoleReserved", err)
	}
	if got, want := err.Error(), "uart: init: esp/uart: console reserved (code 30721)"; got != want {
		t.Fatalf("Init(UART0, ...).Error() = %q, want %q", got, want)
	}
	// The refusal is Init-only policy: it must not leak into any other call.
	if err := SetBaud(UART0, 115200); !errors.Is(err, stubUnsupported()) {
		t.Fatalf("SetBaud(UART0, ...) = %v, want errUnsupported (the console refusal is Init-only)", err)
	}
}

// TestBadValueRefusal pins the negative-budget refusal on the two calls that
// take a timeout: Read and FlushOutput.
func TestBadValueRefusal(t *testing.T) {
	if _, err := Read(UART1, make([]byte, 8), -1); !errors.Is(err, ErrBadValue) {
		t.Fatalf("Read(UART1, buf, -1) = %v, want ErrBadValue", err)
	}
	if err := FlushOutput(UART1, -1); !errors.Is(err, ErrBadValue) {
		t.Fatalf("FlushOutput(UART1, -1) = %v, want ErrBadValue", err)
	}
	// An empty buffer asks for nothing: (0, nil) without touching the port.
	if n, err := Read(UART1, nil, 100); n != 0 || err != nil {
		t.Fatalf("Read(UART1, nil, 100) = (%d, %v), want (0, nil)", n, err)
	}
	if n, err := Write(UART1, nil); n != 0 || err != nil {
		t.Fatalf("Write(UART1, nil) = (%d, %v), want (0, nil)", n, err)
	}
}

// TestStubReportsUnsupported pins the honest-stub contract: with no glue in
// the link, every binding reports errUnsupported (0x7F00) instead of faking
// a port, and both I/O shapes carry the same error.
func TestStubReportsUnsupported(t *testing.T) {
	unsupported := stubUnsupported()

	if err := Init(UART1, 17, 18, 115200); !errors.Is(err, unsupported) {
		t.Fatalf("Init on a host build = %v, want errUnsupported from the stub", err)
	}
	if err := SetBaud(UART1, 9600); !errors.Is(err, unsupported) {
		t.Fatalf("SetBaud on a host build = %v, want errUnsupported", err)
	}
	if err := SetConfig(UART1, DefaultConfig()); !errors.Is(err, unsupported) {
		t.Fatalf("SetConfig on a host build = %v, want errUnsupported", err)
	}
	if n, err := Write(UART1, []byte("llgo")); n != 0 || !errors.Is(err, unsupported) {
		t.Fatalf("Write on a host build = (%d, %v), want (0, errUnsupported)", n, err)
	}
	if n, err := Read(UART1, make([]byte, 8), 200); n != 0 || !errors.Is(err, unsupported) {
		t.Fatalf("Read on a host build = (%d, %v), want (0, errUnsupported)", n, err)
	}
	if n, err := Available(UART1); n != 0 || !errors.Is(err, unsupported) {
		t.Fatalf("Available on a host build = (%d, %v), want (0, errUnsupported)", n, err)
	}
	if err := FlushInput(UART1); !errors.Is(err, unsupported) {
		t.Fatalf("FlushInput on a host build = %v, want errUnsupported", err)
	}
	if err := FlushOutput(UART1, 200); !errors.Is(err, unsupported) {
		t.Fatalf("FlushOutput on a host build = %v, want errUnsupported", err)
	}
	if err := Deinit(UART1); !errors.Is(err, unsupported) {
		t.Fatalf("Deinit on a host build = %v, want errUnsupported", err)
	}
}

// stubUnsupported builds the error value every host stub reports (the
// family's 0x7F00, between the real esp_err ranges), so the tests match the
// contract without hardcoding the magic number twice.
func stubUnsupported() error {
	return esperr.Coded(errUnsupported, "")
}
