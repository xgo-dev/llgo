// Package gpio drives ESP GPIO pins from llgo-compiled Go code.
//
// It follows the pkg/esp package template (see the package wifi docs and
// pkg/esp/README.md): wifi-style three-file Go split plus a thin C glue
// template. Most IDF GPIO calls are synchronous register writes, so there
// is no mailbox here - every function maps to one glue call whose esp_err
// result is wrapped as the family's shared error value (nil = ESP_OK, the
// code stays inspectable on the error).
//
// # Backends
//
// Which driver sits behind the API is a build-tag decision, and the API is
// identical on all three:
//
//   - llgo.idf (an ESP-IDF project): glue/gpio_glue.c, which calls the IDF
//     gpio driver (gpio_idf.go).
//   - llgo build -target esp32c3|esp32s31 with no IDF: the register-level
//     baremetal backend (gpio_baremetal.go) writes the GPIO and I/O MUX
//     registers directly, so a plain llgo binary drives pins with no IDF and
//     no FreeRTOS in the link.
//   - any other build (host, Wasm, a baremetal chip without a register
//     table): the honest stub, which reports errUnsupported instead of
//     touching hardware (gpio_stub.go).
//
// # Usage
//
//	p := gpio.Pin(4)
//	if e := p.SetDirection(gpio.Output, gpio.PullNone); e != 0 { ... }
//	p.SetLevel(true)
//	high := p.GetLevel() // bool: true = high, false = low
//	p.Toggle()
//
// Output pins are configured as INPUT_OUTPUT so GetLevel reads back what
// SetLevel wrote (a plain output mode leaves the input buffer disabled and
// a read-back would not round-trip on hardware).
//
// # Glue integration
//
// Copy glue/gpio_glue.c into your project's main component (WHOLE_ARCHIVE
// required); see pkg/esp/README.md for the two integration recipes.
package gpio

import "github.com/xgo-dev/llgo/pkg/esp/esperr"

// errOf turns one glue result into the error this package returns (nil for
// ESP_OK): every code forwards as a bare esperr.Coded value - the driver's
// esp_err is the diagnostic, and the esp_err naming table lives in the wifi
// package. The package has no Go-side refusal of its own (the glue owns the
// pin-range guard), so it declares no sentinel.
func errOf(code int32) error {
	if code == 0 {
		return nil
	}
	return esperr.Coded(code, "")
}

// Mode selects a pin's direction.
type Mode uint32

const (
	Input  Mode = 0 // input buffer on, output off
	Output Mode = 1 // output on; the glue also enables the input buffer
	// so Set->Get round-trips
)

// Pull selects the internal pull resistors (valid in input modes; IDF
// recommends external pull-ups for buses).
type Pull uint32

const (
	PullNone Pull = 0
	PullUp   Pull = 1
	PullDown Pull = 2
)

// Pin identifies a GPIO by number. The glue validates the range before it
// touches the pin: anything outside [0, 64) returns ESP_ERR_INVALID_ARG
// (0x102), because the pin-bit-mask shift would otherwise be undefined
// behavior. The IDF driver further rejects the chip's absent slots (29 and
// 41 on ESP32-S31) the same way.
type Pin int32

// SetDirection configures the pin: direction mode plus pull resistors.
// It is the pinMode equivalent; the glue runs gpio_config followed by
// gpio_set_direction.
func (p Pin) SetDirection(m Mode, pull Pull) error {
	return errOf(setDirection(int32(p), int32(m), int32(pull)))
}

// SetLevel drives an output pin high or low.
func (p Pin) SetLevel(high bool) error {
	lvl := int32(0)
	if high {
		lvl = 1
	}
	return errOf(setLevel(int32(p), lvl))
}

// GetLevel reads the pin's input path: true = high, false = low (the
// gpio_get_level 0/1, mapped to bool the way the IDF semantic reads).
// gpio_get_level cannot fail, so there is no error result.
func (p Pin) GetLevel() bool {
	return getLevel(int32(p)) != 0
}

// Toggle inverts an output pin's level.
func (p Pin) Toggle() error {
	return errOf(toggle(int32(p)))
}
