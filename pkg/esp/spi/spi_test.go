//go:build !llgo.idf

// Host-build tests for the spi surface. A host build has no SPI controller
// and no glue, so nothing here claims a bus exists: the tests pin what the
// pure-Go layer owns - the frequency table and its guard, the pin-range
// guard, the internal error segment, the rx DMA word-rounding staging, the
// no-op buffer shapes, and the honest-stub contract - and that every
// driver binding reports errUnsupported instead of faking a bus.
package spi

import (
	"errors"
	"testing"

	"github.com/xgo-dev/llgo/pkg/esp/esperr"
)

// TestFrequencies pins the four working frequencies to their Hz values (the
// 80 MHz ceiling over divisions 1/2/4/8) and the guard that accepts exactly
// them.
func TestFrequencies(t *testing.T) {
	want := map[uint32]uint32{
		Freq10MHz: 10_000_000,
		Freq20MHz: 20_000_000,
		Freq40MHz: 40_000_000,
		Freq80MHz: 80_000_000,
	}
	for got, w := range want {
		if got != w {
			t.Fatalf("frequency constant %d != %d", got, w)
		}
		if !validFreq(got) {
			t.Fatalf("validFreq(%d) = false, want true (an accepted rate)", got)
		}
	}
	for _, bad := range []uint32{0, 1, 8_000_000, 16_000_000, 26_000_000, 40_000_001, 160_000_000} {
		if validFreq(bad) {
			t.Fatalf("validFreq(%d) = true, want false (not an exact power-of-two division of 80 MHz)", bad)
		}
	}
}

// TestFrequencyValidation pins the client-side frequency guard: Init and
// SetFrequency reject a rate outside the four exact values with the
// ErrBadFrequency sentinel before anything crosses into C.
func TestFrequencyValidation(t *testing.T) {
	for _, bad := range []uint32{0, 1_000_000, 8_000_000, 160_000_000} {
		if err := Init(11, 12, 13, 10, bad); !errors.Is(err, ErrBadFrequency) {
			t.Fatalf("Init(hz=%d) = %v, want ErrBadFrequency", bad, err)
		}
		if err := SetFrequency(bad); !errors.Is(err, ErrBadFrequency) {
			t.Fatalf("SetFrequency(hz=%d) = %v, want ErrBadFrequency", bad, err)
		}
	}
}

// TestPinValidation pins the client-side pin guard: pins outside [0, 64)
// are rejected with ESP_ERR_INVALID_ARG (0x102) before anything crosses
// into C, cs additionally accepts NoCS (-1).
func TestPinValidation(t *testing.T) {
	good := spiPins{11, 12, 13, 10}
	bad := []spiPins{
		{-1, 12, 13, 10}, // mosi out of range
		{11, 64, 13, 10}, // miso out of range
		{11, 12, 13, 64}, // cs out of range (NoCS is fine, 64 is not)
		{11, 12, 13, -2}, // cs below NoCS
	}
	for _, p := range bad {
		if err := Init(p.mosi, p.miso, p.sclk, p.cs, Freq10MHz); esperr.Code(err) != 0x102 {
			t.Fatalf("Init(%+v) = %v, want ESP_ERR_INVALID_ARG (0x102)", p, err)
		}
	}
	if err := Init(good.mosi, good.miso, good.sclk, NoCS, Freq10MHz); esperr.Code(err) != errUnsupported {
		t.Fatalf("Init with NoCS = %v, want the stub's errUnsupported (pins valid)", err)
	}
}

type spiPins struct{ mosi, miso, sclk, cs Pin }

// TestErrorSegment pins the binding's internal segment values through the
// sentinels: 0x7101/0x7102/0x7103 sit outside every real esp_err range (the
// drivers' codes top out well below 0x7000), so a refusal made at the glue
// can never be mistaken for one made by the IDF driver. Each sentinel
// carries its segment code as its Error.Code annotation, so both match
// paths work: errors.Is against the sentinel (by identity) and against a
// raw-code Error (by code), and the rendering is the human text with the
// code annotation.
func TestErrorSegment(t *testing.T) {
	for _, tc := range []struct {
		sent *esperr.Error
		code int32
	}{
		{ErrNotInit, errNotInit},
		{ErrBadFrequency, errBadFrequency},
		{ErrBusy, errBusy},
	} {
		if esperr.Code(tc.sent) != tc.code {
			t.Fatalf("sentinel carries code 0x%X, want 0x%X", esperr.Code(tc.sent), tc.code)
		}
		if !errors.Is(errOf(tc.code), tc.sent) {
			t.Fatalf("the glue's 0x%X refusal did not match its sentinel", tc.code)
		}
		if !errors.Is(errOf(tc.code), &esperr.Error{Code: tc.code}) {
			t.Fatalf("the 0x%X refusal did not match a raw-code Error (code path broken)", tc.code)
		}
		var e *esperr.Error
		if !errors.As(errOf(tc.code), &e) || e.Code != tc.code {
			t.Fatalf("errors.As on the 0x%X refusal did not surface the annotation", tc.code)
		}
	}
	if got, want := ErrNotInit.Error(), "esp/spi: not initialized (code 28929)"; got != want {
		t.Fatalf("ErrNotInit.Error() = %q, want %q", got, want)
	}
	if got, want := ErrBadFrequency.Error(), "esp/spi: bad frequency (code 28930)"; got != want {
		t.Fatalf("ErrBadFrequency.Error() = %q, want %q", got, want)
	}
	if esperr.Code(errOf(0x103)) != 0x103 {
		t.Fatal("a non-segment driver code did not forward as a bare Coded value")
	}
}

// TestRxStaging pins the rx DMA word-rounding staging: the SPI DMA engine
// writes MISO data in whole 32-bit words, so a rx whose length is not a
// multiple of four must be received through a fresh word-rounded buffer
// (its tail bytes belong to the staging allocation, never to the caller's
// slice), and a word-multiple rx must flow through without staging.
func TestRxStaging(t *testing.T) {
	// Word-multiple lengths: no staging, the caller's buffer goes as-is.
	rx := make([]byte, 8)
	if p := rxStaged(rx); p != &rx[0] {
		t.Fatal("rxStaged staged a word-multiple rx, want the caller's buffer")
	}

	// A non-word-multiple length: staged, and copyBack moves the bytes.
	rx = make([]byte, 5)
	p := rxStaged(rx)
	if s_stagedRx == nil || len(s_stagedRx) != 8 {
		t.Fatalf("staged buffer = %v, want an 8-byte (word-rounded) allocation", s_stagedRx)
	}
	if p != &s_stagedRx[0] {
		t.Fatal("rxStaged did not return the staging buffer's first element")
	}
	for i := range s_stagedRx {
		s_stagedRx[i] = byte(i)
	}
	copyBackStaged(rx)
	for i := range rx {
		if rx[i] != byte(i) {
			t.Fatalf("copyBackStaged left rx[%d] = %d, want %d", i, rx[i], i)
		}
	}
	if s_stagedRx != nil {
		t.Fatal("copyBackStaged did not clear the staging state")
	}

	// Empty rx: neither staging nor a pointer.
	if p := rxStaged(nil); p != nil {
		t.Fatal("rxStaged(nil) returned a pointer, want nil")
	}
}

// TestNoOpShapes pins the empty-buffer no-op contract: calls whose buffers
// are all empty succeed without touching the driver (on a host build a
// stub trip would report errUnsupported instead).
func TestNoOpShapes(t *testing.T) {
	if n, err := Transfer(nil, nil); n != 0 || err != nil {
		t.Fatalf("Transfer(nil, nil) = (%d, %v), want (0, nil)", n, err)
	}
	if n, err := Write(nil); n != 0 || err != nil {
		t.Fatalf("Write(nil) = (%d, %v), want (0, nil)", n, err)
	}
	if err := WriteRead(nil, nil); err != nil {
		t.Fatalf("WriteRead(nil, nil) = %v, want nil", err)
	}
}

// TestStubReportsUnsupported pins the honest-stub contract: with no glue in
// the link, every driver-touching binding reports errUnsupported (0x7F00)
// instead of faking a bus.
func TestStubReportsUnsupported(t *testing.T) {
	unsupported := &esperr.Error{Code: errUnsupported}
	if err := Init(11, 12, 13, 10, Freq10MHz); !errors.Is(err, unsupported) {
		t.Fatalf("Init on a host build = %v, want errUnsupported from the stub", err)
	}
	if err := SetFrequency(Freq20MHz); !errors.Is(err, unsupported) {
		t.Fatalf("SetFrequency on a host build = %v, want errUnsupported", err)
	}
	if _, err := Transfer([]byte{1, 2}, make([]byte, 4)); !errors.Is(err, unsupported) {
		t.Fatalf("Transfer on a host build = %v, want errUnsupported", err)
	}
	if _, err := Write([]byte{0x9F}); !errors.Is(err, unsupported) {
		t.Fatalf("Write on a host build = %v, want errUnsupported", err)
	}
	if err := WriteRead([]byte{0x9F}, make([]byte, 3)); !errors.Is(err, unsupported) {
		t.Fatalf("WriteRead on a host build = %v, want errUnsupported", err)
	}
	if _, err := ActualFreqKHz(); !errors.Is(err, unsupported) {
		t.Fatalf("ActualFreqKHz on a host build = %v, want errUnsupported", err)
	}
	if err := AcquireBus(); !errors.Is(err, unsupported) {
		t.Fatalf("AcquireBus on a host build = %v, want errUnsupported", err)
	}
	if err := ReleaseBus(); !errors.Is(err, unsupported) {
		t.Fatalf("ReleaseBus on a host build = %v, want errUnsupported", err)
	}
	if err := Deinit(); !errors.Is(err, unsupported) {
		t.Fatalf("Deinit on a host build = %v, want errUnsupported", err)
	}
}
