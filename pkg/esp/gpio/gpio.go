// Package gpio drives ESP-IDF GPIO pins from llgo-compiled Go code.
//
// It follows the pkg/esp package template (see the package wifi docs and
// pkg/esp/README.md): wifi-style three-file Go split plus a thin C glue
// template. Most IDF GPIO calls are synchronous register writes, so there
// is no mailbox here - every function maps to one glue call that returns
// the raw esp_err code (0 = ESP_OK).
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
func (p Pin) SetDirection(m Mode, pull Pull) int32 {
	return setDirection(int32(p), int32(m), int32(pull))
}

// SetLevel drives an output pin high or low. Returns the esp_err code.
func (p Pin) SetLevel(high bool) int32 {
	lvl := int32(0)
	if high {
		lvl = 1
	}
	return setLevel(int32(p), lvl)
}

// GetLevel reads the pin's input path: true = high, false = low (the
// gpio_get_level 0/1, mapped to bool the way the IDF semantic reads).
// gpio_get_level cannot fail, so there is no error result.
func (p Pin) GetLevel() bool {
	return getLevel(int32(p)) != 0
}

// Toggle inverts an output pin's level. Returns the esp_err code.
func (p Pin) Toggle() int32 {
	return toggle(int32(p))
}
