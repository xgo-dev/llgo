package main

import (
	"github.com/xgo-dev/llgo/pkg/esp/esperr"
	"github.com/xgo-dev/llgo/pkg/esp/wifi"
)

// psGuardProbe checks the station-only guard of the power-save surface while
// the driver still has no mode at all (right after Setup, before SetSTA/
// SetModeSTA picks one): both calls must be refused with ESP_ERR_WIFI_MODE
// (12293) before anything reaches the driver - the same refusal a program
// running AP-only gets. The driver itself would swallow the call silently
// here (esp_wifi_set_ps validates only the enum range and reports success),
// so the refusal coming back named is the proof the glue guard fired.
//
// A failure does not stop the run: the line says FAIL and main proceeds, so
// one log captures everything around a broken acceptance.
func psGuardProbe() {
	println("[go] wifi ps: guard start")
	err := wifi.SetPowerSave(wifi.PowerSaveMinModem)
	if esperr.Code(err) != 0x3005 {
		println("[go] wifi ps: FAIL guard set err =", errText(err))
		return
	}
	_, err = wifi.GetPowerSave()
	if esperr.Code(err) != 0x3005 {
		println("[go] wifi ps: FAIL guard get err =", errText(err))
		return
	}
	println("[go] wifi ps: mode guard refused", errText(err))
	println("[go] wifi ps: guard PASS")
}

// psProbe exercises the power-save surface against the associated station:
// the driver default after esp_wifi_init is MIN_MODEM (esp_wifi.h), then a
// None round-trip (the latency-critical, radio-always-on mode) and back to
// MinModem - leaving the driver in the state every earlier acceptance run
// saw, so the net/udp probes that follow this call run under the same
// power-save condition as before and double as the proof that the
// association still passes traffic after the mode flips.
func psProbe() {
	println("[go] wifi ps: start")
	ps, err := wifi.GetPowerSave()
	if err != nil {
		println("[go] wifi ps: FAIL read default err =", errText(err))
		return
	}
	if ps != wifi.PowerSaveMinModem {
		println("[go] wifi ps: FAIL default ps=" + ps.String() + " want MIN_MODEM")
		return
	}
	println("[go] wifi ps: default ps=" + ps.String())

	if err := wifi.SetPowerSave(wifi.PowerSaveNone); err != nil {
		println("[go] wifi ps: FAIL set none err =", errText(err))
		return
	}
	ps, err = wifi.GetPowerSave()
	if err != nil {
		println("[go] wifi ps: FAIL read none err =", errText(err))
		return
	}
	if ps != wifi.PowerSaveNone {
		println("[go] wifi ps: FAIL none readback ps=" + ps.String() + " want NONE")
		return
	}
	println("[go] wifi ps: none ok ps=" + ps.String())

	if err := wifi.SetPowerSave(wifi.PowerSaveMinModem); err != nil {
		println("[go] wifi ps: FAIL set min err =", errText(err))
		return
	}
	ps, err = wifi.GetPowerSave()
	if err != nil {
		println("[go] wifi ps: FAIL read min err =", errText(err))
		return
	}
	if ps != wifi.PowerSaveMinModem {
		println("[go] wifi ps: FAIL min readback ps=" + ps.String() + " want MIN_MODEM")
		return
	}
	println("[go] wifi ps: min ok ps=" + ps.String())
	println("[go] wifi ps: PASS")
}
