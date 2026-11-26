// Package temp reads the ESP32-S31's built-in temperature sensor from
// llgo-compiled Go code.
//
// It follows the pkg/esp package template (see pkg/esp/README.md): three-file
// Go split plus a thin C glue template. The sensor is a single-instance
// peripheral whose IDF driver (driver/temperature_sensor.h, the
// esp_driver_tsens component) is handle-based; because there is exactly one
// sensor per chip, the C glue holds the handle in static storage and every
// binding here is a plain package-level function whose esp_err result is
// wrapped as the shared error value (nil = ESP_OK, the code stays
// inspectable on the error).
//
// The sensor resolution is ±1 °C on the S31 - it reads the die's ambient
// temperature, useful for drift/thermal monitoring, not a precision
// thermometer. The IDF v6.3 driver names its lifecycle install / enable /
// disable / uninstall; the binding maps them onto Install / Start / Stop /
// Uninstall and passes every driver error through unadorned (no version
// wrapping - an unsupported range or an already-installed sensor surfaces as
// the driver's own esp_err).
//
// # Usage
//
//	if err := temp.Install(temp.DefaultOpts()); err != nil { ... }
//	if err := temp.Start(); err != nil { ... }
//	c, err := temp.Read()           // degrees Celsius
//	milli, err := temp.ReadMilliC() // the same reading in m°C, integer
//	if err := temp.Stop(); err != nil { ... }
//	if err := temp.Uninstall(); err != nil { ... }
//
// ReadMilliC returns the raw conversion (degrees × 1000, integer) for
// integer-only logging - the bare-metal println path has no float
// formatting, and keeping the reading in ints avoids dragging soft-float
// helpers into a size-sensitive image.
//
// The C boundary stays integer-and-pointer-only (the family rule): the
// driver's float reading is converted to milli-degrees inside the glue, and
// Read() divides on the Go side.
//
// # Glue integration
//
// Copy glue/temp_glue.c into your project's main component (WHOLE_ARCHIVE
// required; add esp_driver_tsens to the component REQUIRES); see
// pkg/esp/README.md for the two integration recipes.
package temp

import "github.com/xgo-dev/llgo/pkg/esp/esperr"

// The package's internal error segment: 0x72xx, reserved for the temp
// binding's refusals (pkg/esp/README.md's segment table). esp_err codes from
// the IDF driver pass through unchanged; these values are outside every real
// esp_err range so a refusal made at the glue can never be mistaken for one
// made by the driver.
//
// The not-installed reporting is the glue's job: before Install has run its
// static handle is NULL and Start/Read/Stop/Uninstall return the segment
// value below, so a caller can tell "the sensor was never installed" apart
// from the driver's own ESP_ERR_INVALID_STATE (0x103, e.g. a second Start).
const errNotInstalled = 0x7201

// ErrNotInstalled reports a sensor call made before Install. It is the
// package's semantic sentinel - match it with errors.Is - and it keeps the
// glue's 0x7201 segment code on Error.Code as the diagnostic annotation, so
// errors.As and esperr.Code still expose the number the C pipeline reported
// and code-style checks against the segment value keep working.
var ErrNotInstalled = newSentinel("esp/temp: not installed", errNotInstalled)

// newSentinel declares a semantic sentinel that keeps its historical
// segment code on Error.Code as the annotation (esperr.New is text-only;
// the glue still reports the code, and the README segment table pins it).
func newSentinel(text string, code int32) *esperr.Error {
	e, _ := esperr.New(text).(*esperr.Error)
	e.Code = code
	return e
}

// errOf turns one glue result into the error this package returns (nil for
// ESP_OK): the not-installed refusal becomes the ErrNotInstalled sentinel,
// every other code forwards as a bare esperr.Coded value - the driver's
// esp_err is the diagnostic, and the esp_err naming table lives in the wifi
// package.
func errOf(code int32) error {
	if code == 0 {
		return nil
	}
	if code == errNotInstalled {
		return ErrNotInstalled
	}
	return esperr.Coded(code, "")
}

// Opts configures the sensor at Install time. Use DefaultOpts for the
// IDF-recommended starting point.
type Opts struct {
	// RangeMinC / RangeMaxC declare the temperature span the caller cares
	// about, in °C. The driver picks its internal range/dacdiv steps to
	// cover it; a span the chip cannot cover reports the driver's
	// ESP_ERR_INVALID_ARG (0x102) - the S31's steps top out at ±1 °C
	// accuracy either way. Default -10..80.
	RangeMinC int32
	RangeMaxC int32
	// AllowPD lets the driver back the sensor state up across sleep so
	// its power domain can be powered down; costs a little RAM. Default
	// false.
	AllowPD bool
}

// DefaultOpts returns the IDF-recommended configuration
// (TEMPERATURE_SENSOR_CONFIG_DEFAULT(-10, 80)): a -10..80 °C span, no sleep
// power-domain allowance.
func DefaultOpts() Opts {
	return Opts{RangeMinC: -10, RangeMaxC: 80}
}

// Install installs the temperature sensor driver with opts. The sensor
// stays inactive until Start. Installing twice reports the driver's
// ESP_ERR_INVALID_STATE (0x103) - Uninstall first to re-install. An
// uncovered range span reports the driver's ESP_ERR_INVALID_ARG (0x102).
func Install(opts Opts) error {
	pd := int32(0)
	if opts.AllowPD {
		pd = 1
	}
	return errOf(install(opts.RangeMinC, opts.RangeMaxC, pd))
}

// Start enables the sensor: the conversions begin and Read works. Starting
// an already-running sensor reports the driver's ESP_ERR_INVALID_STATE;
// before Install, the glue reports ErrNotInstalled (0x7201).
func Start() error {
	return errOf(start())
}

// Read returns the measured temperature in degrees Celsius. The sensor's
// accuracy is ±1 °C; a value outside the installed range span reports the
// driver's ESP_FAIL (-1). Before Install the glue reports ErrNotInstalled
// (0x7201), before Start the driver's ESP_ERR_INVALID_STATE (0x103).
func Read() (float64, error) {
	milli, err := ReadMilliC()
	if err != nil {
		return 0, err
	}
	return float64(milli) / 1000, nil
}

// ReadMilliC returns the same reading as Read in integer milli-degrees
// Celsius (degrees × 1000) - the form the C boundary actually carries, kept
// public for integer-only logging and image-size-sensitive callers.
func ReadMilliC() (int32, error) {
	var milli int32
	if e := readMilli(&milli); e != 0 {
		return 0, errOf(e)
	}
	return milli, nil
}

// Stop disables the sensor: conversions halt and Read reports
// ESP_ERR_INVALID_STATE until the next Start.
func Stop() error {
	return errOf(stop())
}

// Uninstall tears the driver down and frees the sensor. After a successful
// Uninstall the package is back at the pre-Install state and Install can run
// again. Before Install the glue reports ErrNotInstalled (0x7201).
func Uninstall() error {
	return errOf(uninstall())
}
