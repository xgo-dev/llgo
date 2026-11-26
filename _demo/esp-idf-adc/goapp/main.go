// ADC oneshot acceptance probe for the llgo ESP-IDF pipeline (ESP32-S31).
//
// Built with: llgo build -target esp32s31-idf -o ../idfproj/main/llgo_out.o ./goapp
// The relocatable object is linked into the IDF firmware by the main
// component; app_main calls the generated C entry, which hands control to
// the llgo fiber scheduler (runtime.RunMain).
//
// The probe drives the pkg/esp/adc oneshot surface in order: the chip-level
// calibration eFuse version (which regime ReadMillivolts runs in), Unit1
// bring-up (Init with DefaultOpts = the chip's 17-bit width), the honest
// double-Init refusal (the driver's ESP_ERR_NOT_FOUND), the pad->channel
// configs (GPIO42 = ADC1 ch0, the N side; GPIO43 = ADC1 ch1, the P side -
// channel-aware calibration keeps separate characteristics for the two
// sides), the honest refusals (the glue's double-Init ESP_ERR_INVALID_STATE,
// the family precedent; an atten the chip does not have - it has exactly
// ADC_ATTEN_DB_0 - and GPIO3, a header pin that is not an analog pad at all
// on this chip: the S31's ADC lives on GPIO42..57), five raw
// reads and five millivolt reads on GPIO42, one round of each on GPIO43,
// a read of a configured-lifecycle unit on a never-configured pad (GPIO44,
// ESP_ERR_INVALID_STATE), the ErrNotInit refusals around Deinit (Unit2 was
// never brought up; the after-Deinit read), and the teardown. Every step
// prints one `[go] adc ...` line; `[go] adc PASS` is the acceptance gate:
// every step returned the verdict the step expects.
//
// Fidelity note (see the README): the raw= and mv= values in the golden are
// whatever the emulator's analog model produces on its floating inputs.
// They pin the read path (the conversion completes, the calibration
// conversion runs, the values are stable across the five rounds) - the
// actual analog accuracy is a board property: verify the millivolts against
// a known voltage on hardware.
package main

import (

	"github.com/xgo-dev/llgo/pkg/esp/adc"
	"github.com/xgo-dev/llgo/pkg/esp/esperr"
)

// dec32 formats v in decimal (no strconv: the stdlib drags reflect machinery
// into a bare-metal image, the family rule every pkg/esp probe follows).
func dec32(v uint32) string {
	if v == 0 {
		return "0"
	}
	var buf [10]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}

// decSigned formats a signed value in decimal.
func decSigned(v int32) string {
	if v < 0 {
		return "-" + dec32(uint32(-int64(v)))
	}
	return dec32(uint32(v))
}

// errText renders an error for the log: "0" for success, "CODE" for a family
// error value.
func errText(err error) string {
	if err == nil {
		return "0"
	}
	if e, ok := err.(*esperr.Error); ok {
		return dec32(uint32(e.Code))
	}
	return "nonzero"
}

// fail reports a failed probe stage and stops. There is no FAIL recovery:
// the acceptance run is a straight line through the surface.
func fail(what string, err error) {
	println("[go] adc " + what + " err=" + errText(err))
	println("[go] adc FAIL " + what)
}

func main() {
	println("[go] adc: start")

	// ---- chip-level calibration facts ----
	ver, err := adc.CaliEfuseVersion()
	if err != nil {
		fail("efuse ver", err)
		return
	}
	println("[go] adc efuse ver=" + decSigned(ver) +
		" (0=nominal scale, >=1 burnt characteristics)")

	// ---- bring-up ----
	if err := adc.Init(adc.Unit1, adc.DefaultOpts()); err != nil {
		fail("init unit1", err)
		return
	}
	println("[go] adc init unit1 ok")

	if err := adc.Init(adc.Unit1, adc.DefaultOpts()); esperr.Code(err) != 0x103 {
		fail("second init", err)
		return
	}
	println("[go] adc second init refused err=259 (ESP_ERR_INVALID_STATE)")

	// ---- pad->channel configs (the S31's analog pads: GPIO42..57) ----
	// GPIO42 = ADC1 channel 0 (the N side of the converter's first
	// differential pair), GPIO43 = ADC1 channel 1 (the P side).
	if err := adc.Config(42, adc.Atten0dB); err != nil {
		fail("config pin42", err)
		return
	}
	println("[go] adc config pin42 atten0 ok")

	if err := adc.Config(43, adc.Atten0dB); err != nil {
		fail("config pin43", err)
		return
	}
	println("[go] adc config pin43 atten0 ok")

	// The chip has exactly one attenuation; asking for the classic
	// ESP32-generation 11 dB is a named refusal.
	if err := adc.Config(42, adc.Atten(11)); esperr.Code(err) != esperr.Code(adc.ErrBadAtten) {
		fail("config atten11", err)
		return
	}
	println("[go] adc config atten11 refused err=" +
		dec32(uint32(esperr.Code(adc.ErrBadAtten))) + " (ErrBadAtten)")

	// GPIO3 sits on the board header but not on the converter: the S31's
	// ADC pads are GPIO42..57.
	if err := adc.Config(3, adc.Atten0dB); esperr.Code(err) != esperr.Code(adc.ErrBadPin) {
		fail("config pin3", err)
		return
	}
	println("[go] adc config pin3 refused err=" +
		dec32(uint32(esperr.Code(adc.ErrBadPin))) + " (ErrBadPin: not an adc pad)")

	vTaskDelay(10)

	// ---- raw reads: five rounds on pin42, one on pin43 ----
	for i := 1; i <= 5; i++ {
		raw, err := adc.Read(42)
		if err != nil {
			fail("read pin42", err)
			return
		}
		println("[go] adc read pin42 " + dec32(uint32(i)) + " raw=" + dec32(raw))
	}
	raw, err := adc.Read(43)
	if err != nil {
		fail("read pin43", err)
		return
	}
	println("[go] adc read pin43 raw=" + dec32(raw))

	// A configured-lifecycle unit on a pad nobody configured: the glue's
	// honest ESP_ERR_INVALID_STATE.
	if _, err := adc.Read(44); esperr.Code(err) != 0x103 {
		fail("read unconfigured", err)
		return
	}
	println("[go] adc read pin44 unconfigured refused err=259 (ESP_ERR_INVALID_STATE)")

	vTaskDelay(10)

	// ---- millivolt reads: five rounds on pin42, one on pin43 ----
	for i := 1; i <= 5; i++ {
		mv, err := adc.ReadMillivolts(42)
		if err != nil {
			fail("millivolts pin42", err)
			return
		}
		println("[go] adc millivolts pin42 " + dec32(uint32(i)) + " mv=" + decSigned(mv))
	}
	mv, err := adc.ReadMillivolts(43)
	if err != nil {
		fail("millivolts pin43", err)
		return
	}
	println("[go] adc millivolts pin43 mv=" + decSigned(mv))

	vTaskDelay(10)

	// ---- teardown and the refusals around it ----
	if err := adc.Deinit(adc.Unit2); esperr.Code(err) != esperr.Code(adc.ErrNotInit) {
		fail("deinit unit2", err)
		return
	}
	println("[go] adc deinit unit2 refused err=" +
		dec32(uint32(esperr.Code(adc.ErrNotInit))) + " (ErrNotInit)")

	if err := adc.Deinit(adc.Unit1); err != nil {
		fail("deinit unit1", err)
		return
	}
	println("[go] adc deinit unit1 ok")

	if _, err := adc.Read(42); esperr.Code(err) != esperr.Code(adc.ErrNotInit) {
		fail("read after deinit", err)
		return
	}
	println("[go] adc read after deinit refused err=" +
		dec32(uint32(esperr.Code(adc.ErrNotInit))) + " (ErrNotInit)")

	println("[go] adc PASS")
}
