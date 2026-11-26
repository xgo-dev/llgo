// Touch sensor + temperature sensor acceptance probe for the llgo ESP-IDF
// pipeline (ESP32-S31).
//
// Built with: llgo build -target esp32s31-idf -o ../idfproj/main/llgo_out.o ./goapp
// The relocatable object is linked into the IDF firmware by the main
// component; app_main calls the generated C entry, which hands control to
// the llgo fiber scheduler (runtime.RunMain).
//
// The probe drives the pkg/esp/temp and pkg/esp/touch surfaces in order:
// install/start/read the temperature sensor (five reads, integer
// milli-degrees over the integer-only logging path), then bring the touch
// controller up (Init -> Config -> Start), scan two channels for ten rounds
// printing data/baseline/delta (the calibration-flow quantities), drain the
// active/inactive event ring each round, demonstrate Recalibrate, then Stop
// and Deinit. Every step prints one `[go] tt ...` line; `[go] tt PASS` is
// the acceptance gate: every API call returned esp_err 0.
//
// Fidelity note (see the README): under esp-emu neither the touch
// controller nor the temperature sensor is modeled by hardware - the
// register reads land on unmapped/zero state, so the values this probe
// prints there are whatever the (real) driver code computes from that
// state. The PASS gate is about the binding+driver path returning esp_err 0
// end to end; reading real capacitance counts needs the board.
package main

import (
	_ "unsafe" // for go:linkname

	"github.com/xgo-dev/llgo/pkg/esp/esperr"
	"github.com/xgo-dev/llgo/pkg/esp/nvs"
	"github.com/xgo-dev/llgo/pkg/esp/temp"
	"github.com/xgo-dev/llgo/pkg/esp/touch"
)

// vTaskDelay yields the CPU for ticks: the pacing between the temperature
// reads and the touch scan rounds. On a host build the stub makes it a
// no-op.
//
//go:linkname vTaskDelay C.vTaskDelay
func vTaskDelay(ticks uint32)

const (
	// probeChans are the two channels the demo scans: Chan4 = GPIO10 and
	// Chan6 = GPIO12. The S31's Function-CoreBoard-1 claims GPIO6/7
	// (Chan0/Chan1) for Ethernet, so the demo avoids them; these two sit
	// on header pins a finger can reach.
	probeChans = 2
	// thresh is the demo's active threshold in channel-data units. The
	// right value depends on the pad and the sample config's gain - tune
	// on hardware (README's calibration flow); this only needs to be a
	// valid configuration for the path to exercise.
	thresh = 200

	// tempReads is the temperature sample count, one per 200 ms.
	tempReads = 5
	// touchRounds is the touch scan-round count, one per 500 ms.
	touchRounds = 10
	// touchTick is the per-round pacing in FreeRTOS ticks (10 ms each).
	touchTick = 50
	// tempTick is the per-read pacing in FreeRTOS ticks.
	tempTick = 20
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

// decSigned formats a signed value in decimal (for the touch delta).
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
	println("[go] tt " + what + " err=" + errText(err))
	println("[go] tt FAIL " + what)
}

func main() {
	println("[go] tt: start")

	// ---- NVS boot counter (power-cycle persistence) ----
	// Every boot reads the stored count, prints it, then stores count+1
	// with an explicit Commit. Watching this line climb across hard
	// power-off cycles is the persistence proof - flash outlives reset.
	if err := nvs.Init(); err != nil {
		fail("nvs init", err)
		return
	}
	ns, err := nvs.Open("boottt")
	if err != nil {
		fail("nvs open", err)
		return
	}
	boots, err := ns.GetU8("boot")
	if err != nil && esperr.Code(err) != 0x1102 { // ESP_ERR_NVS_NOT_FOUND: first ever boot
		fail("nvs get", err)
		return
	}
	println("[go] tt boot count = " + dec32(uint32(boots)) +
		" (increments every power cycle)")
	if err := ns.SetU8("boot", boots+1); err != nil {
		fail("nvs set", err)
		return
	}
	if err := ns.Commit(); err != nil {
		fail("nvs commit", err)
		return
	}
	ns.Close()
	println("[go] tt nvs store ok")

	// ---- temperature sensor ----
	if err := temp.Install(temp.DefaultOpts()); err != nil {
		fail("temp install", err)
		return
	}
	println("[go] tt temp install ok")

	if err := temp.Start(); err != nil {
		fail("temp start", err)
		return
	}
	println("[go] tt temp start ok")

	for i := 0; i < tempReads; i++ {
		milli, err := temp.ReadMilliC()
		if err != nil {
			fail("temp read", err)
			return
		}
		println("[go] tt temp read i=" + dec32(uint32(i)) +
			" milli=" + decSigned(milli))
		vTaskDelay(tempTick)
	}

	if err := temp.Stop(); err != nil {
		fail("temp stop", err)
		return
	}
	println("[go] tt temp stop ok")

	if err := temp.Uninstall(); err != nil {
		fail("temp uninstall", err)
		return
	}
	println("[go] tt temp uninstall ok")

	// ---- touch sensor ----
	if err := touch.Init(touch.DefaultOpts()); err != nil {
		fail("touch init", err)
		return
	}
	println("[go] tt touch init ok")

	chans := [probeChans]touch.ChannelID{touch.Chan4, touch.Chan6}
	for _, ch := range chans {
		if err := touch.Config(ch, touch.ChanOpts{ActiveThreshold: thresh}); err != nil {
			fail("touch config", err)
			return
		}
		// The GPIO the channel pads onto is a build fact (the fixed S31
		// channel->GPIO map): print it so the golden pins the map too.
		println("[go] tt touch config ch=" + dec32(uint32(ch)) +
			" gpio=" + dec32(uint32(touch.GPIO(ch))) +
			" thresh=" + dec32(uint32(thresh)) + " ok")
	}

	if err := touch.Start(); err != nil {
		fail("touch start", err)
		return
	}
	println("[go] tt touch start ok")

	for r := 0; r < touchRounds; r++ {
		vTaskDelay(touchTick)

		for _, ch := range chans {
			data, err := touch.Read(ch)
			if err != nil {
				fail("touch read", err)
				return
			}
			base, err := touch.Baseline(ch)
			if err != nil {
				fail("touch baseline", err)
				return
			}
			delta, err := touch.Delta(ch)
			if err != nil {
				fail("touch delta", err)
				return
			}
			println("[go] tt touch round=" + dec32(uint32(r)) +
				" ch=" + dec32(uint32(ch)) +
				" data=" + dec32(data) +
				" base=" + dec32(base) +
				" delta=" + decSigned(delta))
		}

		// Drain the active/inactive event ring: under the emulator no
		// event can fire (no capacitance to move the data), so the
		// expected line here is the empty drain; on hardware a real
		// touch prints its transitions.
		for {
			ev, ok := touch.PollEvent()
			if !ok {
				break
			}
			state := "inactive"
			if ev.Active {
				state = "active"
			}
			println("[go] tt touch event ch=" + dec32(uint32(ev.Channel)) +
				" " + state)
		}
	}

	// Mid-run recalibration on one channel (pad untouched): the
	// benchmark-reset step of the calibration flow.
	if err := touch.Recalibrate(touch.Chan4); err != nil {
		fail("touch recalibrate", err)
		return
	}
	println("[go] tt touch recalibrate ch=4 ok")

	if err := touch.Stop(); err != nil {
		fail("touch stop", err)
		return
	}
	println("[go] tt touch stop ok")

	if err := touch.Deinit(); err != nil {
		fail("touch deinit", err)
		return
	}
	println("[go] tt touch deinit ok")

	println("[go] tt PASS")
}
