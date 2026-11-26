// Package adc drives the ESP32-S31 on-chip ADC (analog converter, the
// esp_hal_ana_conv peripheral) from llgo-compiled Go code, in the IDF's
// oneshot mode (esp_adc/adc_oneshot.h, the esp_adc component).
//
// It follows the pkg/esp package template (see pkg/esp/README.md): a
// three-file Go split plus a thin C glue template. The driver is
// handle-based (one handle per ADC unit, plus one calibration handle per
// unit), so the C glue owns every handle in a static two-slot table (the
// Go side stays handle-free; static storage only, the family rule) and
// every binding here is a plain package-level function whose esp_err
// result is wrapped as the family's shared error value (nil = ESP_OK, the
// code stays inspectable on the error).
//
// # The S31's ADC, as the silicon defines it
//
// The S31's converter is not the classic SAR ADC of the ESP32/S2/S3
// generation, and several properties a classic-ADC binding would assume
// simply do not hold here. Everything below is read off the IDF headers
// (components/soc/esp32s31/include/soc/adc_channel.h,
// components/esp_hal_ana_conv, components/esp_adc) of the toolchain this
// family builds against (espressif/idf:latest = v6.3-dev):
//
//   - The analog pads are GPIO42..57, sixteen pads in one contiguous run:
//     ADC1 owns GPIO42..49 (channels 0..7), ADC2 owns GPIO50..57
//     (channels 0..7). The binding pins that set as its Go-side pin guard;
//     a pin outside it is refused with ErrBadPin before anything crosses
//     into C. (The driver's own adc_io_to_channel would report
//     ESP_ERR_INVALID_ARG for a non-pad GPIO; the guard just makes the
//     refusal earlier and named.)
//   - The converter supports exactly ONE attenuation, ADC_ATTEN_DB_0
//     (SOC_ADC_ATTEN_NUM = 1). There is no 2.5/6/11 dB tap to select, and
//     the classic "11 dB full-scale ~3.1 V" recipe does not apply. Atten
//     exposes that single value; a different atten is refused with
//     ErrBadAtten.
//   - The oneshot conversion width is fixed: the hardware produces a
//     17-bit result (ADC_LL_RTC_MIN/MAX_BITWIDTH = 17) and the driver
//     accepts only that or the DEFAULT placeholder (0), which it expands
//     to 17. There is no 12-bit mode. Opts{Bitwidth} accepts exactly
//     Bitwidth17 and BitwidthDefault; DefaultOpts() picks Bitwidth17.
//   - The raw value is a 17-bit comparator vector, not a conventional
//     magnitude code: even-numbered channels are the N side and
//     odd-numbered channels the P side of the converter's differential
//     pairs (the calibration scheme converts each side with its own
//     characteristics). Read returns that vector as-is; the millivolt
//     reading is what carries physical meaning.
//   - Calibration is the LINE-FITTING scheme — the only scheme this chip
//     compiles with (ADC_CALI_SCHEME_LINE_FITTING_SUPPORTED). The
//     classic adc_cali_raw_to_voltage is channel-blind and the IDF marks
//     it not supported on the S31; the conversion goes through the
//     channel-aware adc_cali_chan_raw_to_voltage, which ReadMillivolts
//     uses. One calibration handle per unit, created lazily by the glue on
//     the first ReadMillivolts and released by Deinit.
//   - The scheme's characteristics come from the chip's calibration eFuse
//     block when it is burnt (block version >= 2 minor); when it is not,
//     the IDF falls back to the nominal scale factors and the scheme still
//     succeeds — the driver never fails "missing eFuse" on this chip. The
//     honest exposure of that fact is CaliEfuseVersion: 0 means the
//     readings come from the nominal model, >= 1 means they are
//     eFuse-corrected. ReadMillivolts reports whichever the driver
//     produces; it never invents a number the driver did not.
//
// # ADC2 and WiFi
//
// ADC2 shares the analog front end with the radio's power detector, and
// the S31 has no arbiter to negotiate: while WiFi is active (scanning,
// associated, any RF duty), ADC2 conversion results are not guaranteed.
// ADC1 (GPIO42..49) is unaffected. Read a sensor through ADC1 when the
// radio is up; treat any ADC2 reading taken during WiFi activity as
// undefined. Contentions the driver does detect surface as
// ESP_ERR_TIMEOUT (0x107) — the driver's adc_lock_try_acquire and its
// validity check both end there.
//
// # Lifecycle
//
//	Init(Unit1, DefaultOpts())  // claim the unit, fix the width
//	Config(42, Atten0dB)        // map the pad to its channel, set atten
//	v, err := Read(42)          // raw 17-bit vector
//	mv, err := ReadMillivolts(42) // calibrated millivolts
//	Deinit(Unit1)               // release unit + calibration
//
// Read and ReadMillivolts require the pin's channel to have been
// configured (Config) on its unit first; a read of a configured-lifecycle
// unit on a channel nobody configured reports the driver-domain
// ESP_ERR_INVALID_STATE (0x103) from the glue. A second Init of a unit the
// driver still reports the driver's ESP_ERR_NOT_FOUND (0x105, "the ADC
// peripheral to be claimed is already in use"); after a successful Deinit
// the package is back at the pre-Init state and every call on that unit
// reports ErrNotInit from the glue.
//
// # Concurrency
//
// llgo runs goroutines cooperatively on one FreeRTOS task, so every glue
// call here is exclusive by construction. A oneshot conversion takes
// microseconds; the driver spins inside adc_oneshot_hal_convert until the
// done event fires — synchronous, no callback, no mailbox, no ISR.
//
// # Continuous/DMA mode
//
// Not bound. The oneshot path covers the sensor-reading scenario end to
// end (single conversions on demand); the continuous/DMA controller
// (adc_continuous.h) is a separate driver with its own event machinery and
// is left as future work.
//
// # Glue integration
//
// Copy glue/adc_glue.c into your project's main component (WHOLE_ARCHIVE
// required; add esp_adc and esp_hal_ana_conv to the component REQUIRES);
// see pkg/esp/README.md for the two integration recipes.
package adc

import "github.com/xgo-dev/llgo/pkg/esp/esperr"

// The package's internal error segment: 0x70xx, reserved for the adc
// binding's refusals (pkg/esp/README.md's segment table; 0x71 is spi, 0x72
// temp, 0x73 touch, 0x74 espnow, 0x75 nvs, 0x76-0x78 bt/uart, 0x79 opus,
// 0x7A wakenet, 0x7B ble, 0x7C+ the shared/meta segments). esp_err codes
// from the IDF driver pass through unchanged; these values are outside
// every real esp_err range so a refusal made here can never be mistaken
// for one made there.
//
// The not-init reporting is the glue's job: the unit handles live in the
// glue's static table and every glue entry checks the slot it needs, so a
// caller can tell "the unit was never brought up" (0x7001) apart from the
// driver's own ESP_ERR_INVALID_STATE (0x103). ErrBadUnit, ErrBadPin,
// ErrBadAtten and ErrBadBitwidth are refused on both sides: the Go guards
// below catch the misuse before anything crosses into C, and the glue
// repeats the same segment values defensively. ErrCaliUnavailable is
// glue-only: it reports the one failure a lazy calibration-handle creation
// can hit (the driver's ESP_ERR_NO_MEM), mapped to a named condition.
const (
	errNotInit         = 0x7001
	errBadUnit         = 0x7002
	errBadPin          = 0x7003
	errBadAtten        = 0x7004
	errBadBitwidth     = 0x7005
	errCaliUnavailable = 0x7006
)

// ErrNotInit reports an adc call on a unit that Init has not brought up
// (or that Deinit already tore down). It is the package's semantic
// sentinel - match it with errors.Is - and it keeps the glue's 0x7001
// segment code on Error.Code as the diagnostic annotation, so errors.As
// and esperr.Code still expose the number the C pipeline reported.
var ErrNotInit = newSentinel("esp/adc: not initialized", errNotInit)

// ErrBadUnit reports a Unit outside the chip's two units (Unit1/Unit2).
// The Go guard rejects the call before anything crosses into C; the glue
// repeats the same segment value defensively.
var ErrBadUnit = newSentinel("esp/adc: bad unit", errBadUnit)

// ErrBadPin reports a pin that is not one of the S31's analog input pads
// (GPIO42..57, see the package doc). The Go guard rejects the call before
// anything crosses into C; the glue repeats the same segment value
// defensively.
var ErrBadPin = newSentinel("esp/adc: not an adc pin", errBadPin)

// ErrBadAtten reports an attenuation the S31 does not have. The converter
// supports exactly one attenuation (Atten0dB); anything else is refused by
// the Go guard, and the glue repeats the same segment value defensively.
var ErrBadAtten = newSentinel("esp/adc: bad attenuation", errBadAtten)

// ErrBadBitwidth reports a conversion width the S31 does not support. The
// converter produces 17-bit results; Opts{Bitwidth} accepts Bitwidth17 and
// BitwidthDefault (0, which the driver expands to 17), and refuses the
// rest Go-side with the glue repeating the same segment value.
var ErrBadBitwidth = newSentinel("esp/adc: bad bitwidth", errBadBitwidth)

// ErrCaliUnavailable reports that ReadMillivolts could not obtain the
// calibration handle the millivolt conversion needs (the line-fitting
// scheme creation failed - in practice the driver's ESP_ERR_NO_MEM). The
// binding never substitutes a nominal number for a failed calibration
// read: the call fails instead.
var ErrCaliUnavailable = newSentinel("esp/adc: calibration unavailable", errCaliUnavailable)

// newSentinel declares a semantic sentinel that keeps its historical
// segment code on Error.Code as the annotation (esperr.New is text-only;
// the glue still reports the code, and the README segment table pins it).
func newSentinel(text string, code int32) *esperr.Error {
	e, _ := esperr.New(text).(*esperr.Error)
	e.Code = code
	return e
}

// errOf turns one glue result into the error this package returns (nil for
// ESP_OK): the glue's segment-value refusals become the package's
// sentinels, every other code forwards as a bare esperr.Coded value - the
// driver's esp_err is the diagnostic, and the esp_err naming table lives
// in the wifi package.
func errOf(code int32) error {
	switch code {
	case 0:
		return nil
	case errNotInit:
		return ErrNotInit
	case errBadUnit:
		return ErrBadUnit
	case errBadPin:
		return ErrBadPin
	case errBadAtten:
		return ErrBadAtten
	case errBadBitwidth:
		return ErrBadBitwidth
	case errCaliUnavailable:
		return ErrCaliUnavailable
	}
	return esperr.Coded(code, "")
}

// Unit identifies an ADC unit, numbered as the IDF numbers them
// (ADC_UNIT_1 = 0, ADC_UNIT_2 = 1): Unit1 owns GPIO42..49, Unit2 owns
// GPIO50..57. Each pkg/esp package is kept dependency-free, so this is a
// separate type from the other packages' pin/unit types.
type Unit int32

const (
	// Unit1 is ADC_UNIT_1, the eight analog pads GPIO42..49. Reads on
	// Unit1 are unaffected by WiFi activity.
	Unit1 Unit = 0
	// Unit2 is ADC_UNIT_2, the eight analog pads GPIO50..57. ADC2 shares
	// the analog front end with the radio and there is no arbiter: while
	// WiFi is active its conversion results are not guaranteed (see the
	// package doc).
	Unit2 Unit = 1
)

// Pin identifies a GPIO by number (same numbering as the gpio package;
// each pkg/esp package is kept dependency-free, so this is a separate
// type). The S31's analog pads are the contiguous run GPIO42..57:
// Unit1 GPIO42..49 (channels 0..7), Unit2 GPIO50..57 (channels 0..7).
type Pin int32

// The S31 analog pad range, read off soc/adc_channel.h (the chip's own
// pad table: ADC1_GPIO42_CHANNEL .. ADC2_CHANNEL_7_GPIO_NUM). The binding
// pins it as the Go-side pin guard; the glue repeats the generic GPIO
// range [0, 64) defensively and the driver's io->channel map answers for
// anything in between.
const (
	minADCPin Pin = 42
	maxADCPin Pin = 57
)

// validPin reports whether p is one of the S31's analog input pads.
func validPin(p Pin) bool { return p >= minADCPin && p <= maxADCPin }

// Atten selects an ADC attenuation. The S31's converter has exactly one
// attenuation stage; the type and constant exist so the API matches the
// IDF's per-channel config shape and future chips with a real selector
// need no signature change.
type Atten int32

// Atten0dB is ADC_ATTEN_DB_0 - the only attenuation the S31's converter
// implements (SOC_ADC_ATTEN_NUM = 1). The classic 2.5/6/11 dB recipe of
// the ESP32-generation SAR ADC does not exist on this chip.
const Atten0dB Atten = 0

// Bitwidth selects the oneshot conversion width. The S31 produces 17-bit
// conversion results; the driver accepts exactly that width or the DEFAULT
// placeholder, which it expands to 17 itself.
type Bitwidth int32

const (
	// BitwidthDefault is ADC_BITWIDTH_DEFAULT: the driver picks its
	// maximum width (17 on this chip).
	BitwidthDefault Bitwidth = 0
	// Bitwidth17 is the S31's oneshot conversion width
	// (ADC_LL_RTC_MIN_BITWIDTH = ADC_LL_RTC_MAX_BITWIDTH = 17).
	Bitwidth17 Bitwidth = 17
)

// validBitwidth reports whether w is one of the two accepted widths.
func validBitwidth(w Bitwidth) bool { return w == BitwidthDefault || w == Bitwidth17 }

// Opts carries the per-unit options Init accepts. The S31 has no
// attenuation selector and no width selector beyond the fixed 17-bit
// conversion, so Bitwidth is the whole surface; the struct exists so a
// future chip with real options needs no signature change.
type Opts struct {
	// Bitwidth is the unit's oneshot conversion width (BitwidthDefault or
	// Bitwidth17; anything else is refused with ErrBadBitwidth).
	Bitwidth Bitwidth
}

// DefaultOpts returns the options a sensor-reading program wants: the
// chip's real conversion width (17 bits), explicitly.
func DefaultOpts() Opts { return Opts{Bitwidth: Bitwidth17} }

// Init brings one ADC unit up: creates the driver's oneshot unit handle
// (default clock source, ULP disabled) and records the unit's conversion
// width for the channel configs Config makes. Units are independent:
// Unit1 and Unit2 can be brought up separately and torn down separately.
//
// A second Init of a unit still up reports the driver's ESP_ERR_NOT_FOUND
// (0x105, "the ADC peripheral to be claimed is already in use"). The
// driver can also report ESP_ERR_NO_MEM (0x101) for its handle
// allocation.
func Init(u Unit, opts Opts) error {
	if u != Unit1 && u != Unit2 {
		return errOf(errBadUnit)
	}
	if !validBitwidth(opts.Bitwidth) {
		return errOf(errBadBitwidth)
	}
	return errOf(init_(int32(u), int32(opts.Bitwidth)))
}

// Config maps an analog pad to its ADC channel and applies the per-channel
// configuration: atten as given, the unit's conversion width from Init.
// The pin must be one of the S31's analog pads (ErrBadPin otherwise) and
// its unit must be up (ErrNotInit otherwise). Re-Config of a configured
// pin just re-applies the values.
func Config(pin Pin, atten Atten) error {
	if !validPin(pin) {
		return errOf(errBadPin)
	}
	if atten != Atten0dB {
		return errOf(errBadAtten)
	}
	return errOf(config_(int32(pin), int32(atten)))
}

// Read runs one oneshot conversion on the pin's channel and returns the
// raw result: the S31's 17-bit comparator vector (even channels N side,
// odd channels P side - see the package doc), not a conventional
// magnitude code. The pin's channel must have been Config'd on its unit
// (ESP_ERR_INVALID_STATE, 0x103, otherwise).
//
// The driver reports ESP_ERR_TIMEOUT (0x107) when the conversion could not
// be completed or validated - reachable on ADC2 while the radio holds the
// analog front end (no arbiter on this chip), and the one honest failure
// the read path has.
func Read(pin Pin) (uint32, error) {
	if !validPin(pin) {
		return 0, errOf(errBadPin)
	}
	var raw int32
	if e := read_(int32(pin), &raw); e != 0 {
		return 0, errOf(e)
	}
	return uint32(raw), nil
}

// ReadMillivolts runs one oneshot conversion on the pin's channel and
// converts it through the chip's calibration: the LINE-FITTING scheme, the
// only scheme the S31 compiles with, applied channel-aware (the scheme
// keeps separate characteristics for the N and P sides; the classic
// channel-blind adc_cali_raw_to_voltage is not supported on this chip).
// The conversion values come from the chip's calibration eFuse when that
// block is burnt and from the nominal scale otherwise - CaliEfuseVersion
// tells the two regimes apart, and the binding never invents a number the
// driver did not produce (a failed calibration-handle creation fails the
// call with ErrCaliUnavailable).
//
// The same Config-first and timeout semantics as Read apply.
func ReadMillivolts(pin Pin) (int32, error) {
	if !validPin(pin) {
		return 0, errOf(errBadPin)
	}
	var mv int32
	if e := readMillivolts_(int32(pin), &mv); e != 0 {
		return 0, errOf(e)
	}
	return mv, nil
}

// Deinit tears a unit down: deletes the unit's calibration handle (if
// ReadMillivolts created one) and the oneshot unit handle, releasing the
// unit back to the driver. After a successful Deinit the unit is back at
// the pre-Init state and Init can run again; calls on it report ErrNotInit
// (0x7001) from the glue.
func Deinit(u Unit) error {
	if u != Unit1 && u != Unit2 {
		return errOf(errBadUnit)
	}
	return errOf(deinit_(int32(u)))
}

// CaliEfuseVersion reports the version of the chip's ADC calibration
// eFuse block, the exact value the line-fitting scheme keys its
// characteristics on: 0 means the block is not burnt (the scheme runs the
// nominal scale - readings are model-corrected, not eFuse-corrected) and
// >= 1 means the burnt characteristics are in use. It is a chip-level
// query, independent of Init; on a host build it reports the stub's
// errUnsupported like every other entry.
func CaliEfuseVersion() (int32, error) {
	var ver int32
	if e := efuseVersion(&ver); e != 0 {
		return 0, errOf(e)
	}
	return ver, nil
}
