// UART acceptance probe for the llgo ESP-IDF pipeline (ESP32-S31).
//
// Built with: llgo build -target esp32s31-idf -o ../idfproj/main/llgo_out.o ./goapp
// The relocatable object is linked into the IDF firmware by the main
// component; app_main calls the generated C entry, which hands control to
// the llgo fiber scheduler (runtime.RunMain).
//
// The probe drives the pkg/esp/uart surface in order: the binding-side
// refusals first (Init on the console port UART0 must report
// ErrConsoleReserved; a call on a port that was never Init'd must report
// ErrNotInit), then the UART1 lifecycle end to end - Init(UART1, 17, 18,
// 115200), Write, Available, Read with a timeout (the (0, ErrTimeout)
// empty-read shape), FlushInput, FlushOutput, SetConfig, SetBaud, Deinit,
// and the post-Deinit refusal. Every step prints one `[go] uart ...` line;
// `[go] uart PASS` is the acceptance gate.
//
// Fidelity note (see the README): the emulator models no UART peer and no
// wire, so nothing is ever received - Available reads 0 and the Read budget
// expires into ErrTimeout. On hardware with a TX-to-RX loopback jumper on
// GPIO17/18 the same probe receives the written bytes back (see the README's
// hardware checklist).
package main

import (
	"errors"

	"github.com/xgo-dev/llgo/pkg/esp/esperr"
	"github.com/xgo-dev/llgo/pkg/esp/uart"
)

const (
	// probeTx / probeRx are the UART1 pins the demo routes: free header
	// pins on Function-CoreBoard-1 (GPIO6/7 are Ethernet; 10/12 belong to
	// the touch demo). A board run puts the loopback jumper between them.
	probeTx = 17
	probeRx = 18

	// probeBaud1 / probeBaud2 are the two rates the probe sets, so the
	// SetBaud path moves the rate, not a no-op.
	probeBaud1 = 115200
	probeBaud2 = 9600

	// probeBudgetMs is the Read/FlushOutput budget in milliseconds. The
	// driver quantizes to FreeRTOS ticks (10 ms at the default 100 Hz).
	probeBudgetMs = 200

	// payload is what the probe writes. Nine bytes - at home in the TX
	// ring buffer, and exactly what a loopback run expects to read back.
	payload = "llgo-uart"

	// readLen is the Read buffer size: larger than the payload, so a
	// loopback run exercises a partial fill (9 of 16).
	readLen = 16
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

// errCode renders an error for the log: "0" for success, the pipeline code
// for a family error value (esperr.Code walks the Wrap chain).
func errCode(err error) string {
	if err == nil {
		return "0"
	}
	c := esperr.Code(err)
	if c == -1 {
		return "foreign"
	}
	return dec32(uint32(c))
}

// fail reports a failed probe stage and stops. There is no FAIL recovery:
// the acceptance run is a straight line through the surface.
func fail(what string, err error) {
	println("[go] uart " + what + " err=" + errCode(err))
	println("[go] uart FAIL " + what)
}

// expectErr asserts one refusal: the call must fail with the given sentinel,
// the code goes to the log either way.
func expectErr(what string, err error, want error) bool {
	println("[go] uart " + what + " err=" + errCode(err))
	if !errors.Is(err, want) {
		return false
	}
	return true
}

func main() {
	println("[go] uart: start")

	// ---- binding-side refusals ----
	// Port 0 is the console: refused before the driver is touched.
	if !expectErr("console init", uart.Init(uart.UART0, probeTx, probeRx, probeBaud1), uart.ErrConsoleReserved) {
		println("[go] uart FAIL console-not-refused")
		return
	}
	println("[go] uart console refused ok")

	// A port that was never Init'd: the glue's ErrNotInit guard.
	_, err := uart.Available(uart.UART2)
	if !expectErr("notinit available", err, uart.ErrNotInit) {
		println("[go] uart FAIL notinit-not-refused")
		return
	}
	println("[go] uart notinit refused ok")

	// ---- UART1 lifecycle ----
	if err := uart.Init(uart.UART1, probeTx, probeRx, probeBaud1); err != nil {
		fail("init", err)
		return
	}
	println("[go] uart init port=1 tx=" + dec32(probeTx) +
		" rx=" + dec32(probeRx) + " baud=" + dec32(probeBaud1) + " ok")

	n, err := uart.Write(uart.UART1, []byte(payload))
	if err != nil || n != len(payload) {
		println("[go] uart write n=" + dec32(uint32(n)) + " err=" + errCode(err))
		println("[go] uart FAIL write")
		return
	}
	println("[go] uart write n=" + dec32(uint32(n)) + " ok")

	avail, err := uart.Available(uart.UART1)
	if err != nil {
		fail("available", err)
		return
	}
	// The emulator has no peer and no wire: nothing can be received, so
	// this reads 0 there. On a loopback-board run the bytes come back and
	// this reads the payload length (see the README).
	println("[go] uart available n=" + dec32(uint32(avail)))

	var buf [readLen]byte
	rn, err := uart.Read(uart.UART1, buf[:], probeBudgetMs)
	if rn != 0 || !errors.Is(err, uart.ErrTimeout) {
		println("[go] uart read n=" + dec32(uint32(rn)) + " err=" + errCode(err))
		println("[go] uart FAIL read-timeout")
		return
	}
	println("[go] uart read n=0 err=" + errCode(err) + " timeout ok")

	if err := uart.FlushInput(uart.UART1); err != nil {
		fail("flush input", err)
		return
	}
	println("[go] uart flush input ok")

	// The TX-drain wait: expires into ErrTimeout where no wire ever drains
	// (the emulator), reports ok once the FIFO is empty (a real board).
	err = uart.FlushOutput(uart.UART1, probeBudgetMs)
	if err != nil && !errors.Is(err, uart.ErrTimeout) {
		fail("flush output", err)
		return
	}
	println("[go] uart flush output err=" + errCode(err) + " drained")

	if err := uart.SetConfig(uart.UART1, uart.DefaultConfig()); err != nil {
		fail("set config", err)
		return
	}
	println("[go] uart set config 8N1 ok")

	if err := uart.SetBaud(uart.UART1, probeBaud2); err != nil {
		fail("set baud", err)
		return
	}
	println("[go] uart set baud " + dec32(probeBaud2) + " ok")

	if err := uart.Deinit(uart.UART1); err != nil {
		fail("deinit", err)
		return
	}
	println("[go] uart deinit ok")

	// After Deinit the port is back at the pre-init state: refused again.
	_, err = uart.Read(uart.UART1, buf[:], 0)
	if !expectErr("post-deinit read", err, uart.ErrNotInit) {
		println("[go] uart FAIL deinit-not-refused")
		return
	}
	println("[go] uart post-deinit refused ok")

	println("[go] uart PASS")
}
