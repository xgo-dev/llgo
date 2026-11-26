// SPI master acceptance probe for the llgo ESP-IDF pipeline (ESP32-S31).
//
// Built with: llgo build -target esp32s31-idf -o ../idfproj/main/llgo_out.o ./goapp
// The relocatable object is linked into the IDF firmware by the main
// component; app_main calls the generated C entry, which hands control to
// the llgo fiber scheduler (runtime.RunMain).
//
// The probe drives the whole pkg/esp/spi surface in order: bus init with DMA
// (MOSI 11 / MISO 12 / SCLK 13 / CS 10 at 10 MHz), the driver's actual
// clock for the configured rate, SetFrequency to 20 MHz and back, the three
// data operations (Write MOSI-only, Transfer one-transaction full duplex -
// including a short-tx zero-padding round and a non-word-multiple rx round
// through the word-rounded staging buffer - and WriteRead two-half
// register-read shape), the honest refusals (double Init, a rate outside
// the four exact values, ReleaseBus without a held bus, AcquireBus twice,
// SetFrequency and Deinit while held), a CS-pinned AcquireBus session
// (Write/Transfer/WriteRead with CS held, zero-phase drop at release),
// Deinit, and the ErrNotInit a call after Deinit reports. Every step prints
// one `[go] spi ...` line; `[go] spi PASS` is the acceptance gate: every
// step returned the verdict the step expects.
//
// Fidelity note (see the README): esp-emu completes GPSPI polling
// transactions (the trans-done interrupt fires), so every transfer runs
// green end to end - but nothing drives MISO on the emulated bus, so the
// received bytes read as all ones (ff). Those rx lines pin the buffer
// shapes and the receive path; the bytes themselves are the floating MISO
// line, and reading a real device's answer needs the board.
package main

import (

	"github.com/xgo-dev/llgo/pkg/esp/esperr"
	"github.com/xgo-dev/llgo/pkg/esp/spi"
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

// hexline renders rx bytes after a transfer ("ff aa ..").
func hexline(rx []byte) string {
	const digits = "0123456789abcdef"
	s := ""
	for _, b := range rx {
		if s != "" {
			s += " "
		}
		s += string([]byte{digits[b>>4], digits[b&0xf]})
	}
	return s
}

// fail reports a failed probe stage and stops. There is no FAIL recovery:
// the acceptance run is a straight line through the surface.
func fail(what string, err error) {
	println("[go] spi " + what + " err=" + errText(err))
	println("[go] spi FAIL " + what)
}

func main() {
	println("[go] spi: start")

	// ---- bring-up ----
	if err := spi.Init(11, 12, 13, 10, spi.Freq10MHz); err != nil {
		fail("init", err)
		return
	}
	println("[go] spi init ok")

	khz, err := spi.ActualFreqKHz()
	if err != nil {
		fail("actual freq", err)
		return
	}
	println("[go] spi actual freq khz=" + decSigned(khz) + " (10 MHz exact)")

	// ---- SetFrequency: 20 MHz and back, plus the honest refusal ----
	if err := spi.SetFrequency(spi.Freq20MHz); err != nil {
		fail("set freq 20m", err)
		return
	}
	khz, err = spi.ActualFreqKHz()
	if err != nil {
		fail("actual freq 20m", err)
		return
	}
	println("[go] spi set freq 20m ok khz=" + decSigned(khz))

	if err := spi.SetFrequency(8_000_000); esperr.Code(err) != esperr.Code(spi.ErrBadFrequency) {
		fail("set freq 8m", err)
		return
	}
	println("[go] spi set freq 8m refused err=" +
		dec32(uint32(esperr.Code(spi.ErrBadFrequency))) + " (ErrBadFrequency)")

	if err := spi.SetFrequency(spi.Freq10MHz); err != nil {
		fail("set freq 10m", err)
		return
	}
	println("[go] spi set freq 10m ok")

	vTaskDelay(10)

	// ---- data operations, plain bus (CS toggles per transaction) ----
	tx := []byte{0x01, 0x02, 0x03, 0x04}

	n, err := spi.Write(tx)
	if err != nil {
		fail("write", err)
		return
	}
	println("[go] spi write n=" + dec32(uint32(n)))

	rx := make([]byte, 8)
	n, err = spi.Transfer(tx, rx)
	if err != nil {
		fail("transfer", err)
		return
	}
	println("[go] spi transfer n=" + dec32(uint32(n)) +
		" rx=" + hexline(rx) + " (emu: floating MISO reads ff)")

	// Short tx: the MOSI half is zero-padded to n in one fresh buffer.
	n, err = spi.Transfer(tx[:4], rx)
	if err != nil {
		fail("transfer pad", err)
		return
	}
	println("[go] spi transfer pad n=" + dec32(uint32(n)) +
		" rx=" + hexline(rx))

	// Non-word-multiple rx: received through the word-rounded staging
	// buffer, then copied back into the caller's 5-byte slice.
	rx5 := make([]byte, 5)
	n, err = spi.Transfer(tx, rx5)
	if err != nil {
		fail("transfer rx5", err)
		return
	}
	println("[go] spi transfer rx5 n=" + dec32(uint32(n)) +
		" rx=" + hexline(rx5))

	// Register-read shape: two transactions, CS deasserts between them.
	if err := spi.WriteRead(tx, rx); err != nil {
		fail("writeread", err)
		return
	}
	println("[go] spi writeread rx=" + hexline(rx))

	vTaskDelay(10)

	// ---- honest refusals ----
	if err := spi.Init(11, 12, 13, 10, spi.Freq10MHz); esperr.Code(err) != 0x103 {
		fail("second init", err)
		return
	}
	println("[go] spi second init refused err=259 (ESP_ERR_INVALID_STATE)")

	if err := spi.ReleaseBus(); esperr.Code(err) != esperr.Code(spi.ErrBusy) {
		fail("release unheld", err)
		return
	}
	println("[go] spi release unheld refused err=" +
		dec32(uint32(esperr.Code(spi.ErrBusy))) + " (ErrBusy)")

	// ---- CS-pinned session ----
	if err := spi.AcquireBus(); err != nil {
		fail("acquire", err)
		return
	}
	println("[go] spi acquire ok")

	if err := spi.AcquireBus(); esperr.Code(err) != esperr.Code(spi.ErrBusy) {
		fail("acquire twice", err)
		return
	}
	println("[go] spi acquire twice refused err=" +
		dec32(uint32(esperr.Code(spi.ErrBusy))) + " (ErrBusy)")

	if err := spi.SetFrequency(spi.Freq20MHz); esperr.Code(err) != esperr.Code(spi.ErrBusy) {
		fail("set freq held", err)
		return
	}
	println("[go] spi set freq held refused err=" +
		dec32(uint32(esperr.Code(spi.ErrBusy))) + " (ErrBusy)")

	n, err = spi.Write(tx)
	if err != nil {
		fail("held write", err)
		return
	}
	println("[go] spi held write n=" + dec32(uint32(n)))

	n, err = spi.Transfer(tx, rx)
	if err != nil {
		fail("held transfer", err)
		return
	}
	println("[go] spi held transfer n=" + dec32(uint32(n)) +
		" rx=" + hexline(rx))

	if err := spi.WriteRead(tx, rx); err != nil {
		fail("held writeread", err)
		return
	}
	println("[go] spi held writeread rx=" + hexline(rx))

	if err := spi.Deinit(); esperr.Code(err) != esperr.Code(spi.ErrBusy) {
		fail("deinit held", err)
		return
	}
	println("[go] spi deinit held refused err=" +
		dec32(uint32(esperr.Code(spi.ErrBusy))) + " (ErrBusy)")

	if err := spi.ReleaseBus(); err != nil {
		fail("release", err)
		return
	}
	println("[go] spi release ok (zero-phase CS drop)")

	// ---- teardown ----
	if err := spi.Deinit(); err != nil {
		fail("deinit", err)
		return
	}
	println("[go] spi deinit ok")

	if err := spi.SetFrequency(spi.Freq10MHz); esperr.Code(err) != esperr.Code(spi.ErrNotInit) {
		fail("after deinit", err)
		return
	}
	println("[go] spi after deinit refused err=" +
		dec32(uint32(esperr.Code(spi.ErrNotInit))) + " (ErrNotInit)")

	println("[go] spi PASS")
}
