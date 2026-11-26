//go:build !llgo.idf

// Host-build tests for the adc surface. A host build has no analog
// converter and no glue, so nothing here claims a unit exists: the tests
// pin what the pure-Go layer owns - the S31 analog pad table and its
// guard, the unit/atten/bitwidth guards, the internal error segment, and
// the honest-stub contract - and that every driver binding reports
// errUnsupported instead of faking a converter.
package adc

import (
	"errors"
	"testing"

	"github.com/xgo-dev/llgo/pkg/esp/esperr"
)

// TestAnalogPads pins the S31 analog pad table the binding guards with:
// the sixteen pads GPIO42..57 in one contiguous run - ADC1 GPIO42..49
// (channels 0..7), ADC2 GPIO50..57 (channels 0..7), read off
// soc/adc_channel.h - and the guard that accepts exactly them.
func TestAnalogPads(t *testing.T) {
	if minADCPin != 42 || maxADCPin != 57 {
		t.Fatalf("pad range = [%d, %d], want [42, 57] (soc/adc_channel.h)", minADCPin, maxADCPin)
	}
	for p := Pin(42); p <= 57; p++ {
		if !validPin(p) {
			t.Fatalf("validPin(%d) = false, want true (an analog pad)", p)
		}
	}
	for _, bad := range []Pin{-1, 0, 1, 3, 9, 41, 58, 63, 64} {
		if validPin(bad) {
			t.Fatalf("validPin(%d) = true, want false (not an analog pad)", bad)
		}
	}
}

// TestPinValidation pins the client-side pad guard: Config, Read and
// ReadMillivolts reject a pin outside GPIO42..57 with the ErrBadPin
// sentinel before anything crosses into C.
func TestPinValidation(t *testing.T) {
	for _, bad := range []Pin{-1, 0, 41, 58, 100} {
		if err := Config(bad, Atten0dB); !errors.Is(err, ErrBadPin) {
			t.Fatalf("Config(pin=%d) = %v, want ErrBadPin", bad, err)
		}
		if _, err := Read(bad); !errors.Is(err, ErrBadPin) {
			t.Fatalf("Read(pin=%d) = %v, want ErrBadPin", bad, err)
		}
		if _, err := ReadMillivolts(bad); !errors.Is(err, ErrBadPin) {
			t.Fatalf("ReadMillivolts(pin=%d) = %v, want ErrBadPin", bad, err)
		}
	}
}

// TestUnitAndOptsValidation pins the client-side guards for the remaining
// values that cross into C: units outside Unit1/Unit2, widths outside
// BitwidthDefault/Bitwidth17, and any atten beyond Atten0dB - the S31's
// only attenuation.
func TestUnitAndOptsValidation(t *testing.T) {
	if err := Init(Unit(2), DefaultOpts()); !errors.Is(err, ErrBadUnit) {
		t.Fatalf("Init(unit=2) = %v, want ErrBadUnit", err)
	}
	if err := Init(Unit(-1), DefaultOpts()); !errors.Is(err, ErrBadUnit) {
		t.Fatalf("Init(unit=-1) = %v, want ErrBadUnit", err)
	}
	if err := Deinit(Unit(5)); !errors.Is(err, ErrBadUnit) {
		t.Fatalf("Deinit(unit=5) = %v, want ErrBadUnit", err)
	}
	for _, bad := range []Bitwidth{-1, 1, 9, 12, 16, 18} {
		if err := Init(Unit1, Opts{Bitwidth: bad}); !errors.Is(err, ErrBadBitwidth) {
			t.Fatalf("Init(bitwidth=%d) = %v, want ErrBadBitwidth", bad, err)
		}
	}
	for _, bad := range []Atten{-1, 1, 2, 3} {
		if err := Config(42, bad); !errors.Is(err, ErrBadAtten) {
			t.Fatalf("Config(atten=%d) = %v, want ErrBadAtten", bad, err)
		}
	}
}

// TestWidths pins the two accepted widths and the default options: the
// driver accepts ADC_BITWIDTH_DEFAULT (0, expanded to the chip's 17) and
// the explicit 17 - nothing else on this chip (no 12-bit mode; the S31's
// oneshot conversion is 17-bit, ADC_LL_RTC_MIN/MAX_BITWIDTH).
func TestWidths(t *testing.T) {
	if BitwidthDefault != 0 || Bitwidth17 != 17 {
		t.Fatalf("width constants = %d/%d, want 0/17", BitwidthDefault, Bitwidth17)
	}
	if got := DefaultOpts(); got.Bitwidth != Bitwidth17 {
		t.Fatalf("DefaultOpts().Bitwidth = %d, want Bitwidth17 (the chip's real width)", got.Bitwidth)
	}
	if Atten0dB != 0 {
		t.Fatalf("Atten0dB = %d, want 0 (ADC_ATTEN_DB_0, the only attenuation)", Atten0dB)
	}
	if Unit1 != 0 || Unit2 != 1 {
		t.Fatalf("unit constants = %d/%d, want the IDF numbering 0/1", Unit1, Unit2)
	}
}

// TestErrorSegment pins the binding's internal segment values through the
// sentinels: 0x7001..0x7006 sit outside every real esp_err range (the
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
		{ErrBadUnit, errBadUnit},
		{ErrBadPin, errBadPin},
		{ErrBadAtten, errBadAtten},
		{ErrBadBitwidth, errBadBitwidth},
		{ErrCaliUnavailable, errCaliUnavailable},
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
	if got, want := ErrNotInit.Error(), "esp/adc: not initialized (code 28673)"; got != want {
		t.Fatalf("ErrNotInit.Error() = %q, want %q", got, want)
	}
	if got, want := ErrCaliUnavailable.Error(), "esp/adc: calibration unavailable (code 28678)"; got != want {
		t.Fatalf("ErrCaliUnavailable.Error() = %q, want %q", got, want)
	}
	if esperr.Code(errOf(0x107)) != 0x107 {
		t.Fatal("a non-segment driver code did not forward as a bare Coded value")
	}
}

// TestStubReportsUnsupported pins the honest-stub contract: with no glue in
// the link, every driver-touching binding reports errUnsupported (0x7F00)
// instead of faking a converter. The guard refusals of the calls above
// fire before this, so each case here uses arguments the guards accept.
func TestStubReportsUnsupported(t *testing.T) {
	unsupported := &esperr.Error{Code: errUnsupported}
	if err := Init(Unit1, DefaultOpts()); !errors.Is(err, unsupported) {
		t.Fatalf("Init on a host build = %v, want errUnsupported from the stub", err)
	}
	if err := Config(42, Atten0dB); !errors.Is(err, unsupported) {
		t.Fatalf("Config on a host build = %v, want errUnsupported", err)
	}
	if _, err := Read(42); !errors.Is(err, unsupported) {
		t.Fatalf("Read on a host build = %v, want errUnsupported", err)
	}
	if _, err := ReadMillivolts(42); !errors.Is(err, unsupported) {
		t.Fatalf("ReadMillivolts on a host build = %v, want errUnsupported", err)
	}
	if err := Deinit(Unit1); !errors.Is(err, unsupported) {
		t.Fatalf("Deinit on a host build = %v, want errUnsupported", err)
	}
	if _, err := CaliEfuseVersion(); !errors.Is(err, unsupported) {
		t.Fatalf("CaliEfuseVersion on a host build = %v, want errUnsupported", err)
	}
}
