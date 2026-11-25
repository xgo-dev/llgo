//go:build !llgo.idf

// Host-build tests for the touch surface. A host build has no touch
// controller and no glue, so nothing here claims a sensor exists: the tests
// pin what the pure-Go layer owns - the S31 channel->GPIO table (against the
// IDF header values), the IDF enum numberings the glue casts, the default
// configurations (against the IDF default macros), the out-of-range channel
// guard, the internal error segment, and the honest-stub contract - and
// that every driver binding reports errUnsupported instead of faking a
// sensor.
package touch

import (
	"errors"
	"testing"

	"github.com/xgo-dev/llgo/pkg/esp/esperr"
)

// TestGPIOMap pins the S31 channel->GPIO table to the
// TOUCH_PAD_NUMx_GPIO_NUM values of
// esp_hal_touch_sens/esp32s31/include/hal/touch_sensor_channel.h: a drifted
// entry would document (and let a demo scan) the wrong pad.
func TestGPIOMap(t *testing.T) {
	want := [14]int32{6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19}
	for ch, w := range want {
		if got := GPIO(ChannelID(ch)); got != w {
			t.Fatalf("GPIO(Chan%d) = %d, want %d (the IDF TOUCH_PAD_NUM%d_GPIO_NUM)", ch, got, w, ch)
		}
	}
	if got := GPIO(ChannelID(-1)); got != -1 {
		t.Fatalf("GPIO(-1) = %d, want -1", got)
	}
	if got := GPIO(ChannelID(14)); got != -1 {
		t.Fatalf("GPIO(14) = %d, want -1", got)
	}
}

// TestRange pins the S31 channel id range to the SOC caps (TOUCH_MIN_CHAN_ID
// 0, TOUCH_MAX_CHAN_ID 13): 14 channels, and the named constants sit in it.
func TestRange(t *testing.T) {
	if MinChannel != 0 || MaxChannel != 13 {
		t.Fatalf("channel range = [%d, %d], want [0, 13] (SOC_TOUCH_MIN/MAX_CHAN_ID)", MinChannel, MaxChannel)
	}
	if Chan2 != 2 || Chan13 != 13 {
		t.Fatalf("named channels drifted: Chan2=%d Chan13=%d", Chan2, Chan13)
	}
}

// TestEnumValues pins the IDF enum numberings the glue casts straight into
// driver arguments: a value that drifts here would silently configure a
// different filter or counting mode than the name says.
func TestEnumValues(t *testing.T) {
	smooth := map[SmoothFilter]uint32{
		SmoothNoFilter: 0, // TOUCH_SMOOTH_NO_FILTER
		SmoothIIR2:     1, // TOUCH_SMOOTH_IIR_FILTER_2
		SmoothIIR4:     2, // TOUCH_SMOOTH_IIR_FILTER_4
		SmoothIIR8:     3, // TOUCH_SMOOTH_IIR_FILTER_8
	}
	for v, want := range smooth {
		if uint32(v) != want {
			t.Fatalf("SmoothFilter %d != IDF enum value %d", v, want)
		}
	}
	bm := map[BenchmarkFilter]uint32{
		BenchmarkIIR4:   0, // TOUCH_BM_IIR_FILTER_4
		BenchmarkIIR8:   1,
		BenchmarkIIR16:  2,
		BenchmarkIIR32:  3,
		BenchmarkIIR64:  4,
		BenchmarkIIR128: 5,
		BenchmarkJitter: 6, // TOUCH_BM_JITTER_FILTER (v3: no IIR_256)
	}
	for v, want := range bm {
		if uint32(v) != want {
			t.Fatalf("BenchmarkFilter %d != IDF enum value %d", v, want)
		}
	}
	if uint32(OutAsData) != 0 || uint32(OutAsClock) != 1 {
		t.Fatalf("OutMode values drifted: data=%d clock=%d", OutAsData, OutAsClock)
	}
	if dataTypeSmooth != 0 || dataTypeBenchmark != 1 {
		t.Fatalf("channel data type values drifted: smooth=%d benchmark=%d", dataTypeSmooth, dataTypeBenchmark)
	}
}

// TestDefaults pins the default configurations to the IDF default macros the
// glue mirrors: DefaultOpts = TOUCH_SENSOR_DEFAULT_BASIC_CONFIG +
// TOUCH_SENSOR_V3_DEFAULT_SAMPLE_CONFIG, DefaultFilter =
// TOUCH_SENSOR_DEFAULT_FILTER_CONFIG.
func TestDefaults(t *testing.T) {
	o := DefaultOpts()
	if o.ChargeTimes != 500 || o.DivNum != 8 || o.PowerOnWaitUs != 256 ||
		o.MeasIntervalUs != 32 || o.MaxMeasTimeUs != 0 || o.OutMode != OutAsClock {
		t.Fatalf("DefaultOpts %+v, want the IDF default macro values", o)
	}
	f := DefaultFilter()
	if f.Benchmark != BenchmarkIIR16 || f.JitterStep != 4 || f.DenoiseLvl != 1 ||
		f.Smooth != SmoothIIR2 || f.ActiveHysteresis != 2 || f.DebounceCnt != 2 {
		t.Fatalf("DefaultFilter %+v, want the TOUCH_SENSOR_DEFAULT_FILTER_CONFIG values", f)
	}
}

// TestChanValidation pins the client-side channel guard: an id outside the
// S31 range is rejected with ESP_ERR_INVALID_ARG before anything crosses
// into C (the glue's channel-table index would otherwise be undefined
// behavior).
func TestChanValidation(t *testing.T) {
	bad := []ChannelID{-1, 14, 99}
	for _, ch := range bad {
		if err := Config(ch, ChanOpts{}); esperr.Code(err) != 0x102 {
			t.Fatalf("Config(%d) = %v, want ESP_ERR_INVALID_ARG (0x102)", ch, err)
		}
		if _, err := Read(ch); esperr.Code(err) != 0x102 {
			t.Fatalf("Read(%d) = %v, want ESP_ERR_INVALID_ARG (0x102)", ch, err)
		}
		if _, err := Baseline(ch); esperr.Code(err) != 0x102 {
			t.Fatalf("Baseline(%d) = %v, want ESP_ERR_INVALID_ARG (0x102)", ch, err)
		}
		if _, err := Delta(ch); esperr.Code(err) != 0x102 {
			t.Fatalf("Delta(%d) = %v, want ESP_ERR_INVALID_ARG (0x102)", ch, err)
		}
		if err := Recalibrate(ch); esperr.Code(err) != 0x102 {
			t.Fatalf("Recalibrate(%d) = %v, want ESP_ERR_INVALID_ARG (0x102)", ch, err)
		}
		if err := SetThreshold(ch, 1); esperr.Code(err) != 0x102 {
			t.Fatalf("SetThreshold(%d) = %v, want ESP_ERR_INVALID_ARG (0x102)", ch, err)
		}
	}
}

// TestErrorSegment pins the binding's internal segment value through the
// sentinel: 0x7301 sits outside every real esp_err range (the drivers'
// codes top out around 0x10x/0x2xx/0x6xx), so a refusal made at the glue
// can never be mistaken for one made by the IDF driver. The sentinel
// carries the segment code as its Error.Code annotation, so both match
// paths work: errors.Is against the sentinel (by identity) and against a
// raw-code Error (by code), and the rendering is the human text with the
// code annotation.
func TestErrorSegment(t *testing.T) {
	if esperr.Code(ErrNotInit) != 0x7301 {
		t.Fatalf("ErrNotInit carries code 0x%X, want the touch segment value 0x7301", esperr.Code(ErrNotInit))
	}
	if got, want := ErrNotInit.Error(), "esp/touch: not initialized (code 29441)"; got != want {
		t.Fatalf("ErrNotInit.Error() = %q, want %q", got, want)
	}
	if !errors.Is(errOf(errNotInit), ErrNotInit) {
		t.Fatal("the glue's 0x7301 refusal did not match the ErrNotInit sentinel")
	}
	if !errors.Is(errOf(errNotInit), &esperr.Error{Code: 0x7301}) {
		t.Fatal("the 0x7301 refusal did not match a raw-code Error (code path broken)")
	}
	var e *esperr.Error
	if !errors.As(errOf(errNotInit), &e) || e.Code != 0x7301 {
		t.Fatal("errors.As on the refusal did not surface the 0x7301 annotation")
	}
}

// TestStubReportsUnsupported pins the honest-stub contract: with no glue in
// the link, every binding reports errUnsupported (0x7F00) instead of faking
// a sensor, and PollEvent reports an empty ring rather than a fake event.
func TestStubReportsUnsupported(t *testing.T) {
	unsupported := &esperr.Error{Code: errUnsupported}
	if err := Init(DefaultOpts()); !errors.Is(err, unsupported) {
		t.Fatalf("Init on a host build = %v, want errUnsupported from the stub", err)
	}
	if err := ConfigFilter(DefaultFilter()); !errors.Is(err, unsupported) {
		t.Fatalf("ConfigFilter on a host build = %v, want errUnsupported", err)
	}
	if err := Config(Chan4, ChanOpts{ActiveThreshold: 100}); !errors.Is(err, unsupported) {
		t.Fatalf("Config(Chan4) on a host build = %v, want errUnsupported", err)
	}
	if err := Start(); !errors.Is(err, unsupported) {
		t.Fatalf("Start on a host build = %v, want errUnsupported", err)
	}
	if err := Stop(); !errors.Is(err, unsupported) {
		t.Fatalf("Stop on a host build = %v, want errUnsupported", err)
	}
	if _, err := Read(Chan4); !errors.Is(err, unsupported) {
		t.Fatalf("Read on a host build = %v, want errUnsupported", err)
	}
	if err := Recalibrate(Chan4); !errors.Is(err, unsupported) {
		t.Fatalf("Recalibrate on a host build = %v, want errUnsupported", err)
	}
	if err := SetThreshold(Chan4, 100); !errors.Is(err, unsupported) {
		t.Fatalf("SetThreshold on a host build = %v, want errUnsupported", err)
	}
	if err := Deinit(); !errors.Is(err, unsupported) {
		t.Fatalf("Deinit on a host build = %v, want errUnsupported", err)
	}
	if _, ok := PollEvent(); ok {
		t.Fatalf("PollEvent on a host build reported an event, want an empty ring")
	}
}

// TestDeltaErrorPropagation pins that Delta surfaces the read failures it
// is built from instead of a fabricated difference.
func TestDeltaErrorPropagation(t *testing.T) {
	_, err := Delta(Chan4)
	if !errors.Is(err, &esperr.Error{Code: errUnsupported}) {
		t.Fatalf("Delta on a host build = %v, want the stub's errUnsupported", err)
	}
}
