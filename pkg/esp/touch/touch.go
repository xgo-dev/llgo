// Package touch drives the ESP32-S31 touch sensor (touch hardware v3) from
// llgo-compiled Go code.
//
// It follows the pkg/esp package template (see pkg/esp/README.md): three-file
// Go split plus a thin C glue template. The S31's touch controller is a
// single-instance peripheral whose IDF driver (driver/touch_sens.h, the
// esp_driver_touch_sens component) is handle-based; because there is exactly
// one controller and one channel table per chip, the C glue holds them in
// static storage and every binding here is a plain package-level function
// whose esp_err result is wrapped as the shared error value (nil = ESP_OK,
// the code stays inspectable on the error).
//
// # Hardware facts this API is shaped by
//
// The v3 hardware keeps ONE measurement datapath per channel. Reading a
// channel yields two values:
//
//   - the channel data (Read): the charge-discharge count of the last scan,
//     run through the smooth IIR filter when one is configured (the default
//     is IIR_2; with SmoothNoFilter the channel data IS the raw count - the
//     v3 hardware exposes no separate unfiltered register);
//   - the benchmark (Baseline): the hardware-maintained baseline, itself an
//     IIR-filtered (default IIR_16) track of the channel data - the
//     "filtered" value of the legacy two-register chips.
//
// Touch detection is a relative judgment: a channel is active while
// (channel data - benchmark) exceeds the configured ActiveThreshold.
// Delta reports the signed difference so a caller can apply the calibration
// flow documented below.
//
// # Usage
//
//	if err := touch.Init(touch.DefaultOpts()); err != nil { ... }
//	if err := touch.Config(touch.Chan4, touch.ChanOpts{ActiveThreshold: 100}); err != nil { ... }
//	if err := touch.Start(); err != nil { ... }
//	raw, err := touch.Read(touch.Chan4)
//	base, err := touch.Baseline(touch.Chan4)
//	delta, _ := touch.Delta(touch.Chan4) // raw minus baseline
//
// Channels are numbered 0..13. On the ESP32-S31 they map linearly onto
// GPIO6..GPIO19 (see GPIO and the S31 board notes: GPIO6/7 are claimed by
// Ethernet on Function-CoreBoard-1, so avoid Chan0/Chan1 there).
//
// # Calibration flow
//
// Power up idle (nothing touching the pads), Init + Config + Start, then
// take Baseline(ch) as the reference. During scanning the benchmark tracks
// the channel data with its IIR filter, so a touch shows up as Delta
// temporarily deviating from zero; judge it as a ratio, e.g.
// delta*1000/baseline (per-mille) against an empirical threshold (tune on
// hardware - start around 20-50 per-mille with the default sample config).
// When drift accumulates (temperature, aging), call Recalibrate(ch) with the
// pad untouched: it resets the benchmark to the current channel data.
//
// # Threshold events
//
// Init installs active/inactive event callbacks: the glue funnels the ISR
// events into a small static ring and PollEvent drains it from Go. The
// active/inactive judgment uses ActiveThreshold (set per channel via Config)
// with the filter's hysteresis and debounce. Note SetThreshold reconfigures
// the channel, which the IDF driver only allows while the controller is
// stopped - the binding runs the full stop/config/restart cycle and the
// benchmark starts over (recalibrate afterwards).
//
// # Glue integration
//
// Copy glue/touch_glue.c into your project's main component (WHOLE_ARCHIVE
// required; add esp_driver_touch_sens to the component REQUIRES); see
// pkg/esp/README.md for the two integration recipes.
package touch

import "github.com/xgo-dev/llgo/pkg/esp/esperr"

// The package's internal error segment: 0x73xx, reserved for the touch
// binding's refusals (pkg/esp/README.md's segment table). esp_err codes
// from the IDF driver pass through unchanged; these values are outside
// every real esp_err range so a refusal made here can never be mistaken
// for one made there.
//
// The not-init reporting is the glue's job: before Init has run its
// static controller handle is NULL and every glue entry returns the
// segment value below, so a caller can tell "the package was never brought
// up" apart from the driver's own ESP_ERR_INVALID_STATE (0x103, e.g. a
// second Start).
const errNotInit = 0x7301

// ErrNotInit reports a touch call made before Init. It is the package's
// semantic sentinel - match it with errors.Is - and it keeps the glue's
// 0x7301 segment code on Error.Code as the diagnostic annotation, so
// errors.As and esperr.Code still expose the number the C pipeline
// reported and code-style checks against the segment value keep working.
var ErrNotInit = newSentinel("esp/touch: not initialized", errNotInit)

// newSentinel declares a semantic sentinel that keeps its historical
// segment code on Error.Code as the annotation (esperr.New is text-only;
// the glue still reports the code, and the README segment table pins it).
func newSentinel(text string, code int32) *esperr.Error {
	e, _ := esperr.New(text).(*esperr.Error)
	e.Code = code
	return e
}

// errOf turns one glue result into the error this package returns (nil for
// ESP_OK): the not-init refusal becomes the ErrNotInit sentinel, every
// other code forwards as a bare esperr.Coded value - the driver's esp_err
// is the diagnostic, and the esp_err naming table lives in the wifi
// package.
func errOf(code int32) error {
	if code == 0 {
		return nil
	}
	if code == errNotInit {
		return ErrNotInit
	}
	return esperr.Coded(code, "")
}

// ChannelID identifies a touch channel by its hardware index. The S31
// exposes 0..MaxChannel (14 channels); each maps onto a fixed GPIO (see the
// GPIO function). Chan0/Chan1 = GPIO6/7 are Ethernet pins on
// Function-CoreBoard-1 - avoid them there.
type ChannelID int32

// The S31 channel range (SOC_TOUCH_MIN_CHAN_ID / SOC_TOUCH_MAX_CHAN_ID).
const (
	MinChannel ChannelID = 0
	MaxChannel ChannelID = 13
)

// Named channels for the channels the S31 breaks out (the board notes of
// Function-CoreBoard-1 claim Chan0/Chan1 = GPIO6/7 for Ethernet).
const (
	Chan2  ChannelID = 2  // GPIO8
	Chan3  ChannelID = 3  // GPIO9
	Chan4  ChannelID = 4  // GPIO10
	Chan5  ChannelID = 5  // GPIO11
	Chan6  ChannelID = 6  // GPIO12
	Chan7  ChannelID = 7  // GPIO13
	Chan8  ChannelID = 8  // GPIO14
	Chan9  ChannelID = 9  // GPIO15
	Chan10 ChannelID = 10 // GPIO16
	Chan11 ChannelID = 11 // GPIO17
	Chan12 ChannelID = 12 // GPIO18
	Chan13 ChannelID = 13 // GPIO19
)

// gpioMap pins the S31 channel -> GPIO table the IDF keeps in
// esp_hal_touch_sens/esp32s31/include/hal/touch_sensor_channel.h
// (TOUCH_PAD_NUMx_GPIO_NUM): channels 0..13 map onto GPIO 6..19 in order.
var gpioMap = [14]int32{6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19}

// GPIO returns the fixed GPIO a touch channel's pad lives on, or -1 for a
// channel id outside the S31's range.
func GPIO(ch ChannelID) int32 {
	if ch < MinChannel || ch > MaxChannel {
		return -1
	}
	return gpioMap[ch]
}

// Touch-pad data types the glue passes to touch_channel_read_data. The
// values are the touch_chan_data_type_t enum numbering - pinned by the host
// test, a drift would silently read a different register.
const (
	dataTypeSmooth    = 0 // TOUCH_CHAN_DATA_TYPE_SMOOTH
	dataTypeBenchmark = 1 // TOUCH_CHAN_DATA_TYPE_BENCHMARK
)

// SmoothFilter selects the IIR filter applied to the channel data as it is
// measured. The values are the touch_smooth_filter_mode_t enum numbering.
type SmoothFilter uint32

const (
	SmoothNoFilter SmoothFilter = 0 // channel data = the raw count
	SmoothIIR2     SmoothFilter = 1 // 1/2 raw + 1/2 last (typical)
	SmoothIIR4     SmoothFilter = 2 // 1/4 raw + 3/4 last
	SmoothIIR8     SmoothFilter = 3 // 1/8 raw + 7/8 last
)

// BenchmarkFilter selects the filter the benchmark baseline runs through.
// The values are the touch_benchmark_filter_mode_t enum numbering (v3
// hardware; the IIR_256 entry only exists on v2).
type BenchmarkFilter uint32

const (
	BenchmarkIIR4   BenchmarkFilter = 0 // 1/4 raw + 3/4 benchmark
	BenchmarkIIR8   BenchmarkFilter = 1
	BenchmarkIIR16  BenchmarkFilter = 2 // typical
	BenchmarkIIR32  BenchmarkFilter = 3
	BenchmarkIIR64  BenchmarkFilter = 4
	BenchmarkIIR128 BenchmarkFilter = 5
	BenchmarkJitter BenchmarkFilter = 6 // raw +/- jitter step
)

// OutMode selects how the hardware counts the channel's binarized output.
// The values are the touch_out_mode_t enum numbering.
type OutMode uint32

const (
	OutAsData  OutMode = 0 // count as data: smaller value, more sensitive
	OutAsClock OutMode = 1 // count as clock: accurate (default)
)

// Opts configures the sensor controller and its single sample configuration
// at Init time. Use DefaultOpts for the IDF-recommended starting point.
type Opts struct {
	// ChargeTimes is the charge/discharge count per measurement; the
	// channel data scales with it. Default 500.
	ChargeTimes uint32
	// DivNum divides the output pulse count - the data gain. Lower it if
	// readings saturate. Default 8.
	DivNum uint32
	// PowerOnWaitUs waits between powering a channel on and measuring it,
	// for data stability. Default 256.
	PowerOnWaitUs uint32
	// MeasIntervalUs spaces consecutive channel measurements. Default 32.
	MeasIntervalUs uint32
	// MaxMeasTimeUs caps one channel measurement; 0 disables the cap
	// (default). Set it only after tuning on hardware.
	MaxMeasTimeUs uint32
	// OutMode picks the counting mode. Zero value = OutAsClock (default).
	OutMode OutMode
}

// DefaultOpts returns the IDF-recommended controller + sample configuration
// (TOUCH_SENSOR_DEFAULT_BASIC_CONFIG with
// TOUCH_SENSOR_V3_DEFAULT_SAMPLE_CONFIG's charge times and a mid-range
// div_num): charge_times 500, div_num 8, 256 us power-on wait, 32 us
// measurement interval, no measurement cap, counting the output as clock.
func DefaultOpts() Opts {
	return Opts{
		ChargeTimes:    500,
		DivNum:         8,
		PowerOnWaitUs:  256,
		MeasIntervalUs: 32,
		MaxMeasTimeUs:  0,
		OutMode:        OutAsClock,
	}
}

// FilterOpts configures the benchmark and data filters (installed by Init;
// re-installable anytime with ConfigFilter). Use DefaultFilter for the
// IDF-recommended starting point.
type FilterOpts struct {
	// Benchmark selects the baseline filter. Default BenchmarkIIR16.
	Benchmark BenchmarkFilter
	// JitterStep is the jitter filter step; only used when Benchmark is
	// BenchmarkJitter. Range [0,15]. Default 4.
	JitterStep uint32
	// DenoiseLvl is the noise-bouncing range that will not trigger a
	// benchmark update: 0 disables, 4 is the strongest. Default 1.
	DenoiseLvl uint32
	// Smooth selects the channel-data IIR. Default SmoothIIR2.
	Smooth SmoothFilter
	// ActiveHysteresis widens the active/inactive judgment by this margin
	// around ActiveThreshold to avoid chattering. Default 2.
	ActiveHysteresis uint32
	// DebounceCnt is how many consecutive off-threshold scans a state
	// change must hold before it takes. Default 2.
	DebounceCnt uint32
}

// DefaultFilter returns the IDF-recommended filter configuration
// (TOUCH_SENSOR_DEFAULT_FILTER_CONFIG): benchmark IIR_16, jitter step 4,
// denoise level 1, data IIR_2, hysteresis 2, debounce 2.
func DefaultFilter() FilterOpts {
	return FilterOpts{
		Benchmark:        BenchmarkIIR16,
		JitterStep:       4,
		DenoiseLvl:       1,
		Smooth:           SmoothIIR2,
		ActiveHysteresis: 2,
		DebounceCnt:      2,
	}
}

// ChanOpts configures one touch channel (Config). The zero value is a valid
// configuration with a zero threshold - configure ActiveThreshold to take
// part in the active/inactive event judgment.
type ChanOpts struct {
	// ActiveThreshold is the relative threshold of the active/inactive
	// judgment: the channel reports active while (channel data -
	// benchmark) exceeds it, in channel-data units. The right value
	// depends on the sample config's gain and the pad; start around 100
	// with the default sample config and tune on hardware.
	ActiveThreshold uint32
}

// Init brings the touch sensor up: allocates the controller with opts'
// sample configuration, installs the IDF-recommended filter (see
// DefaultFilter; ConfigFilter replaces it anytime) and registers the
// active/inactive event callbacks that feed PollEvent's ring. The sensor
// stays idle until Start - Config the channels first (the IDF driver only
// allocates channels while the controller is disabled), then Start begins
// the scans.
//
// The esp_err codes it can report: ESP_ERR_NO_MEM (0x101), ESP_ERR_INVALID_ARG
// (0x102) for an impossible configuration, ESP_ERR_INVALID_STATE (0x103)
// when the controller is already allocated - Deinit first to re-init.
func Init(opts Opts) error {
	return errOf(init_(int32(opts.ChargeTimes), int32(opts.DivNum),
		int32(opts.PowerOnWaitUs), int32(opts.MeasIntervalUs),
		int32(opts.MaxMeasTimeUs), int32(opts.OutMode)))
}

// ConfigFilter replaces the benchmark/data filter configuration (the one
// Init installed from DefaultFilter). The IDF driver allows this while the
// sensor is enabled - the filters re-arm in place.
func ConfigFilter(opts FilterOpts) error {
	return errOf(configFilter(int32(opts.Benchmark), int32(opts.JitterStep),
		int32(opts.DenoiseLvl), int32(opts.Smooth),
		int32(opts.ActiveHysteresis), int32(opts.DebounceCnt)))
}

// Config allocates (or reconfigures, see SetThreshold's note) one touch
// channel: pads it onto its fixed GPIO and sets its active threshold. Must
// run while the sensor is idle (after Init, before Start - the IDF driver
// only allocates channels while the controller is disabled); on a running
// sensor this reports the driver's ESP_ERR_INVALID_STATE (0x103). Config
// every channel before Start; Start scans exactly the configured set.
func Config(ch ChannelID, opts ChanOpts) error {
	if ch < MinChannel || ch > MaxChannel {
		return errOf(0x102) // ESP_ERR_INVALID_ARG, before anything crosses into C
	}
	return errOf(configChannel(int32(ch), int32(opts.ActiveThreshold)))
}

// Start enables the controller and begins continuous scanning of the
// configured channels (the hardware FSM re-measures each in turn). Config
// the channels first; a second Start reports the driver's ESP_ERR_INVALID
// STATE (0x103).
func Start() error {
	return errOf(start())
}

// Stop halts continuous scanning and disables the controller, back to the
// post-Init state: channels can be Config'd again, Start resumes. Stopping
// while not running reports ESP_ERR_INVALID_STATE.
func Stop() error {
	return errOf(stop())
}

// Read returns the channel's current data (the smooth value; with
// SmoothNoFilter configured this is the raw charge-discharge count - the
// v3 hardware keeps no separate unfiltered register). The value updates
// every scan cycle; read it while Start is running or after a scan.
func Read(ch ChannelID) (uint32, error) {
	return readChan(ch, dataTypeSmooth)
}

// Baseline returns the channel's benchmark - the hardware-maintained
// baseline the active/inactive judgment subtracts from the channel data.
// Take the first reading after a fresh calibration as the reference for
// touch detection (see the package's calibration flow).
func Baseline(ch ChannelID) (uint32, error) {
	return readChan(ch, dataTypeBenchmark)
}

// readChan reads one data type for a channel, rejecting an out-of-range
// channel id before anything crosses into C (the glue's array index would
// otherwise be undefined behavior) and unwrapping the pointer out-arg.
func readChan(ch ChannelID, dataType int32) (uint32, error) {
	if ch < MinChannel || ch > MaxChannel {
		return 0, errOf(0x102) // ESP_ERR_INVALID_ARG
	}
	var v uint32
	if e := readChan_(int32(ch), dataType, &v); e != 0 {
		return 0, errOf(e)
	}
	return v, nil
}

// Delta returns the signed difference between the channel's current data
// and its benchmark - the quantity the touch judgment thresholds on. A
// touch shows up as a positive excursion (data rises when a finger adds
// capacitance with the default counting mode); drift around zero is the
// normal state. The difference is computed in the 32-bit domain: with the
// default sample config the counts stay far below 2^31, and a value that
// did not would have no touch-detection meaning anyway.
func Delta(ch ChannelID) (int32, error) {
	raw, err := Read(ch)
	if err != nil {
		return 0, err
	}
	base, err := Baseline(ch)
	if err != nil {
		return 0, err
	}
	return int32(int64(raw) - int64(base)), nil
}

// Recalibrate resets the channel's benchmark to its current channel data -
// the drift-correction step of the calibration flow. Call it with the pad
// untouched; the IDF driver allows it while scanning.
func Recalibrate(ch ChannelID) error {
	if ch < MinChannel || ch > MaxChannel {
		return errOf(0x102)
	}
	return errOf(calibrate(int32(ch)))
}

// SetThreshold changes a channel's active threshold. The IDF driver only
// reconfigures a channel while the controller is disabled, so this runs the
// full stop -> reconfigure -> restart cycle: scanning pauses briefly and the
// benchmark starts over (recalibrate afterwards). The cycle always ends
// running - calling it on a stopped sensor starts it again.
func SetThreshold(ch ChannelID, thresh uint32) error {
	if ch < MinChannel || ch > MaxChannel {
		return errOf(0x102)
	}
	return errOf(setThreshold(int32(ch), int32(thresh)))
}

// Event is one active/inactive transition reported by PollEvent. Active is
// true when the channel went active (a touch began) and false when it went
// inactive again.
type Event struct {
	Channel ChannelID
	Active  bool
}

// PollEvent drains the touch event ring the glue's ISR callbacks fill.
// It returns the oldest event and true when one was waiting, and the zero
// Event and false when the ring is empty - a poll, never a block: the
// events are transitions, not data, and a caller that needs them all runs
// this in a loop between scans.
func PollEvent() (Event, bool) {
	var ev Event
	var chanID, active int32
	if !pollEvent(&chanID, &active) {
		return Event{}, false
	}
	ev.Channel = ChannelID(chanID)
	ev.Active = active != 0
	return ev, true
}

// Deinit tears the sensor down: stops scanning, disables the controller,
// deletes every allocated channel and frees the controller itself. After a
// successful Deinit the package is back at the pre-Init state and Init can
// run again. Deinit before Init reports ErrNotInit (0x7301) from the glue.
func Deinit() error {
	return errOf(deinit())
}
