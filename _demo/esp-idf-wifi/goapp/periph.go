package main

import (
	"github.com/xgo-dev/llgo/pkg/esp/gpio"
	"github.com/xgo-dev/llgo/pkg/esp/i2c"
	"github.com/xgo-dev/llgo/pkg/esp/i2s"
)

// ---- Peripheral probes (pkg/esp/gpio, pkg/esp/i2c, pkg/esp/i2s).
//
// Each probe prints one [go] periph line per call and must terminate on its
// own: every driver path used here has a driver-side or emulator-side
// timeout, so a missing peripheral model (esp-emu fidelity is only proven
// for WiFi and PSRAM) surfaces as an esp_err code, not a hang. If a probe
// ever does hang the emulator or the board, flip its switch below to false
// - the remaining probes stay independent.
const (
	probeGPIO   = true
	probeI2C    = true
	probeI2S    = true
	probeNet    = true
	probeUDP    = true
	probeTLS    = true
	probeWS     = true
	probeReconn = true
)

// level01 maps a GetLevel bool back to IDF's 0/1 for the probe's golden
// output line ("high = 1 low = 0 toggled = 1").
func level01(b bool) int {
	if b {
		return 1
	}
	return 0
}

func gpioProbe() {
	p := gpio.Pin(4)
	if err := p.SetDirection(gpio.Output, gpio.PullNone); err != nil {
		println("[go] periph gpio set_direction err =", errCode(err))
		return
	}
	if err := p.SetLevel(true); err != nil {
		println("[go] periph gpio set_level(true) err =", errCode(err))
		return
	}
	high := p.GetLevel()
	if err := p.SetLevel(false); err != nil {
		println("[go] periph gpio set_level(false) err =", errCode(err))
		return
	}
	low := p.GetLevel()
	if err := p.Toggle(); err != nil {
		println("[go] periph gpio toggle err =", errCode(err))
		return
	}
	toggled := p.GetLevel()
	println("[go] periph gpio roundtrip high =", level01(high),
		"low =", level01(low), "toggled =", level01(toggled))
	if high && !low && toggled {
		println("[go] periph gpio PASS")
	} else {
		println("[go] periph gpio FAIL (roundtrip mismatch)")
	}
}

func i2cProbe() {
	bus, err := i2c.NewBus(8, 9)
	if err != nil {
		println("[go] periph i2c new_bus err =", errCode(err))
		return
	}
	println("[go] periph i2c bus install ok")
	// Address 0x50 with nothing attached. On hardware the transfer fails
	// with an explicit error (NACK/timeout) within the 1 s driver timeout -
	// never a hang and never a crash. The emulator does not model ACK/NACK,
	// so there the same read reports err = 0 and a 0xFF byte; real transfers
	// need real hardware.
	dev, err := bus.Device(0x50, 100000)
	if err != nil {
		println("[go] periph i2c add_device err =", errCode(err))
		bus.Close()
		return
	}
	var reg [1]byte
	err = dev.Read(reg[:])
	println("[go] periph i2c read err =", errCode(err), "byte =", int(reg[0]))
	if err := dev.Close(); err != nil {
		println("[go] periph i2c rm_device err =", errCode(err))
	}
	if err := bus.Close(); err != nil {
		println("[go] periph i2c del_bus err =", errCode(err))
		return
	}
	println("[go] periph i2c done")
}

func i2sProbe() {
	ch, err := i2s.NewChannel(true)
	if err != nil {
		println("[go] periph i2s alloc err =", errCode(err))
		return
	}
	println("[go] periph i2s channel alloc ok")
	// TX channel: bclk GPIO4, ws GPIO5, dout GPIO6, no mclk and no din.
	if err := ch.ConfigureStd(i2s.StdConfig{
		MCLK: -1, BCLK: 4, WS: 5, DOUT: 6, DIN: -1,
		Bits: i2s.Bits16, Stereo: true, SampleRate: 44100,
	}); err != nil {
		println("[go] periph i2s init_std err =", errCode(err))
		ch.Close()
		return
	}
	println("[go] periph i2s std init ok")
	if err := ch.Enable(); err != nil {
		println("[go] periph i2s enable err =", errCode(err))
		ch.Close()
		return
	}
	// One DMA-scale buffer of 16-bit stereo silence (256 frames = 1024 bytes).
	silence := make([]byte, 1024)
	n, werr := ch.Write(silence)
	println("[go] periph i2s write n =", n, "err =", errCode(werr))
	if err := ch.Disable(); err != nil {
		println("[go] periph i2s disable err =", errCode(err))
	}
	if err := ch.Close(); err != nil {
		println("[go] periph i2s del err =", errCode(err))
		return
	}
	println("[go] periph i2s done")
}

func periphProbe() {
	println("[go] periph: start")
	if probeGPIO {
		gpioProbe()
	} else {
		println("[go] periph gpio skipped (known to hang this target)")
	}
	if probeI2C {
		i2cProbe()
	} else {
		println("[go] periph i2c skipped (known to hang this target)")
	}
	if probeI2S {
		i2sProbe()
	} else {
		println("[go] periph i2s skipped (known to hang this target)")
	}
}
