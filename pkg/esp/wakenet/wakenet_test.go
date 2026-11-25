//go:build !llgo.idf

// Host-build tests. They exercise what a host build can exercise: the stub
// path (no engine in this build), the argument validation the Go layer does
// before any binding runs, the state-machine guards, and the error spaces.
// Device behavior - the engine's real detections, the glue's task and event
// ring - is verified under esp-emu by the _demo/esp-idf-wakenet acceptance
// probe, not here.
package wakenet

import (
	"errors"
	"testing"

	"github.com/xgo-dev/llgo/pkg/esp/esperr"
)

// TestErrorSegments pins the segment plan the esperr family agreed on: this
// package's own conditions in 0x7Axx (0x79 opus, 0x7B ble, 0x7C esperr meta,
// 0x7D/7E ws/tls, 0x7F stubs), raw esp_err passing through below 0x4000, and
// ESP_ERR_NOT_SUPPORTED - "no engine in this firmware" - translated to the
// stub-tagged ErrUnsupported.
func TestErrorSegments(t *testing.T) {
	for _, c := range []struct {
		err  error
		code int32
	}{
		{ErrNotReady, 0x7A01},
		{ErrInvalidState, 0x7A02},
		{ErrTimeout, 0x7A03},
		{ErrBadValue, 0x7A04},
		{ErrNotFound, 0x7A05},
		{ErrUnsupported, 0x7F00},
	} {
		if got := esperr.Code(c.err); got != c.code {
			t.Errorf("esperr.Code(%v) = %#x, want %#x", c.err, got, c.code)
		}
	}
	// The segments stay disjoint: no Go-domain code can collide with an
	// esp_err of the same magnitude (0x103 is ESP_ERR_INVALID_STATE, not
	// this package's sentinel).
	if got := esperr.Code(espErr(0x103)); got != 0x103 {
		t.Errorf("esperr.Code(espErr(ESP_ERR_INVALID_STATE)) = %#x, want 0x103", got)
	}
	// The translation: a firmware without the engine reports
	// ESP_ERR_NOT_SUPPORTED, and the package renders it as ErrUnsupported so
	// callers errors.Is against one value for "nothing wakenet here",
	// whatever side reported it.
	if !errors.Is(espErr(0x106), ErrUnsupported) {
		t.Error("espErr(ESP_ERR_NOT_SUPPORTED) does not match ErrUnsupported")
	}
	// ESP_OK wraps to nil.
	if espErr(0) != nil {
		t.Error("espErr(0) is not nil")
	}
}

// TestErrNameTable checks the ErrName vocabulary: the esp_err codes the
// lifecycle calls can return, this package's own segment, and the family's
// hex fallback for anything else.
func TestErrNameTable(t *testing.T) {
	for _, c := range []struct {
		code int32
		want string
	}{
		{0, "ESP_OK"},
		{0x101, "ESP_ERR_NO_MEM"},
		{0x102, "ESP_ERR_INVALID_ARG"},
		{0x103, "ESP_ERR_INVALID_STATE"},
		{0x107, "ESP_ERR_TIMEOUT"},
		{0x7A01, "WAKENET_ERR_NOT_READY"},
		{0x7A02, "WAKENET_ERR_INVALID_STATE"},
		{0x7A03, "WAKENET_ERR_TIMEOUT"},
		{0x7A04, "WAKENET_ERR_BAD_VALUE"},
		{0x7A05, "WAKENET_ERR_NOT_FOUND"},
		{0x7F00, "WAKENET_ERR_UNSUPPORTED"},
		{0x555555, "esp_err 0x555555"},
	} {
		if got := ErrName(c.code); got != c.want {
			t.Errorf("ErrName(%#x) = %q, want %q", c.code, got, c.want)
		}
	}
}

// TestConfigValidation checks the refusals Init makes before anything
// crosses into C: they must fire on a host build too (ErrBadValue before the
// stub's ErrUnsupported), so a bad config fails the same way hosted and on
// target.
func TestConfigValidation(t *testing.T) {
	for _, cfg := range []ModelConfig{
		{Threshold: 0.1}, // below the engine's 0.4 floor
		{Threshold: 1.0}, // above its 0.9999 ceiling
		{Model: "wn9_hilexin_and_then_some_more_xx"}, // longer than MaxModelName
		{Source: Source(9)},                          // not a source this package knows
		{Source: SourceI2S, BCLK: -1, WS: 5, DIN: 6}, // no BCLK pin
		{Source: SourceI2S, BCLK: 4, WS: -1, DIN: 6}, // no WS pin
		{Source: SourceI2S, BCLK: 4, WS: 5, DIN: -1}, // no DIN pin
	} {
		err := Init(cfg)
		if !errors.Is(err, ErrBadValue) {
			t.Errorf("Init(%+v) = %v, want ErrBadValue", cfg, err)
		}
	}
	// The zero value stays a working configuration (the host stub refuses it
	// with ErrUnsupported, never with a validation error).
	if err := Init(ModelConfig{}); !errors.Is(err, ErrUnsupported) {
		t.Errorf("Init(ModelConfig{}) = %v, want ErrUnsupported", err)
	}
}

// TestGuardsBeforeInit checks every call made outside a session: queries
// report their zero values, lifecycle calls report the error that says why
// (ErrNotReady before any Init, ErrInvalidState for a Close of an engine
// that was never opened - the i2s package's "nothing to close" convention).
func TestGuardsBeforeInit(t *testing.T) {
	if Events() != nil {
		t.Error("Events() before Init is not nil")
	}
	if ModelName() != "" || ChunkSamples() != 0 || SampleRate() != 0 {
		t.Error("model info before Init is not zero-valued")
	}
	if Detected() {
		t.Error("Detected() before Init is true")
	}
	if f, a, m := Stats(); f != 0 || a != 0 || m != 0 {
		t.Errorf("Stats() before Init = %d, %d, %d, want zeros", f, a, m)
	}
	if err := Start(); !errors.Is(err, ErrNotReady) {
		t.Errorf("Start() = %v, want ErrNotReady", err)
	}
	if err := Stop(); !errors.Is(err, ErrNotReady) {
		t.Errorf("Stop() = %v, want ErrNotReady", err)
	}
	if err := Close(); !errors.Is(err, ErrInvalidState) {
		t.Errorf("Close() = %v, want ErrInvalidState", err)
	}
}

// TestEventKinds pins the event values: they are the C glue's WN_EV_* enum,
// so a drifted constant silently mislabels every event.
func TestEventKinds(t *testing.T) {
	for _, c := range []struct {
		k    EventKind
		want int32
	}{
		{EvNone, 0},
		{EvDetected, 1},
		{EvError, 2},
		{EvOverflow, -1},
	} {
		if int32(c.k) != c.want {
			t.Errorf("event kind %d = %d, want %d", c.k, c.k, c.want)
		}
	}
}

// TestThresholdConversion pins the float-to-thousandths conversion at the
// boundary: floats never cross into C, and the rounding must not turn 0.5
// into 499. A converted 1000 (from the 0.9999 ceiling) means "1.0" to the
// engine - the C side re-divides by 1000, so the value survives the round
// trip as the fraction the caller named.
func TestThresholdConversion(t *testing.T) {
	for _, c := range []struct {
		f    float32
		want int32
	}{
		{0.4, 400},
		{0.5, 500},
		{0.75, 750},
		{0.9999, 1000},
	} {
		if got := int32(c.f*1000 + 0.5); got != c.want {
			t.Errorf("threshold conversion of %v = %d, want %d", c.f, got, c.want)
		}
	}
}
