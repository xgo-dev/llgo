// Baremetal peripheral smoke test: GPIO and hardware I2C through pkg/esp on a
// plain `llgo build -target ...` binary, with no ESP-IDF and no FreeRTOS in
// the link.
//
// It is the runnable artefact behind the non-IDF backend: the ESP-IDF path is
// covered by the _demo/esp-idf-* projects, which need idf.py and a whole
// component tree, while this one is the same two API calls a baremetal program
// makes and nothing else.
//
// Three checks, each printing one fixed line so the output can be compared
// against golden/:
//
//	gpio loopback ok|mismatch   drive the pin high and low, read it back
//	i2c scan found=N addr=0x..  probe every reserved-free 7-bit address
//	i2c register read ...       only when the scan found a device
//	i2c bulk write/read/verify  a payload wider than the FIFO, both directions
//
// What the emulator can and cannot prove is in README.md; the short version is
// that esp-emu executes the register sequences, mirrors GPIO_OUT into GPIO_IN
// unconditionally (regardless of the direction register), and models the I2C
// ACK bit and a register file at 0x50 - so a
// "mismatch", a "found=0" or a different register value here is the model's
// answer, not a driver bug, and the answer on hardware is a separate question.
package main

import (
	"github.com/xgo-dev/llgo/pkg/esp/gpio"
	"github.com/xgo-dev/llgo/pkg/esp/i2c"
)

const (
	// loopbackPin carries the GPIO check. GPIO3 is a plain pad on both
	// supported SoCs: inside the valid-GPIO mask of each chip and not one of
	// the S31's two pad holes, so no peripheral function claims it and the
	// check only touches the GPIO block.
	loopbackPin = 3
	// sdaPin / sclPin are the ESP32-C3 devkit defaults and the pair the
	// ESP-IDF examples use, so a board wired for those works here too.
	sdaPin = 8
	sclPin = 9
)

func main() {
	println("esp-baremetal-periph: gpio + hardware i2c, no esp-idf")
	checkGPIO()
	checkI2C()
	println("esp-baremetal-periph: done")
}

// checkGPIO drives a pin high and low and reads it back. The pin's input
// buffer stays on (that is what Output means in pkg/esp), so a chip whose pad
// loopback works reports its own driven level.
func checkGPIO() {
	p := gpio.Pin(loopbackPin)
	if err := p.SetDirection(gpio.Output, gpio.PullNone); err != nil {
		println("gpio loopback error:", err.Error())
		return
	}
	if err := p.SetLevel(true); err != nil {
		println("gpio loopback error:", err.Error())
		return
	}
	high := p.GetLevel()
	if err := p.SetLevel(false); err != nil {
		println("gpio loopback error:", err.Error())
		return
	}
	low := p.GetLevel()
	if high && !low {
		println("gpio loopback ok")
		return
	}
	println("gpio loopback mismatch high=", high, "low=", low)
}

// checkI2C installs a master bus, scans the usable address range, and reads one
// register from the first device it finds.
func checkI2C() {
	bus, err := i2c.NewBus(sdaPin, sclPin)
	if err != nil {
		println("i2c scan error:", err.Error())
		return
	}
	found, err := bus.Scan(0x08, 0x77)
	if err != nil {
		println("i2c scan error:", err.Error())
		return
	}
	println("i2c scan found=" + dec(len(found)))
	for _, addr := range found {
		println("i2c scan addr=0x" + hex2(addr))
	}
	if len(found) == 0 {
		_ = bus.Close()
		return
	}
	readRegister(bus, found[0])
	bulkTransfer(bus, found[0])
	_ = bus.Close()
}

// readRegister reads one register from the first device found. A device that
// does not answer at register level is reported, not retried: the point of the
// check is that the transfer path runs, and an unlabelled I2C part has no
// register map to assume.
func readRegister(bus *i2c.Bus, addr uint8) {
	dev, err := bus.Device(addr, i2c.SpeedStandard)
	if err != nil {
		println("i2c register error:", err.Error())
		return
	}
	defer dev.Close()
	var buf [1]byte
	if err := dev.ReadReg(0x00, buf[:]); err != nil {
		println("i2c register read failed:", err.Error())
		return
	}
	println("i2c register read reg=0x00 val=0x" + hex2(buf[0]))
}

// bulkTransfer pushes one payload wider than the controller's FIFO through the
// write path and pulls one wider than the FIFO back through the read path, so
// the command-list chunking is exercised on both directions. The bytes written
// at registers 0x00..0x27 are read back and compared, which is what pins the
// chunk boundaries down: a lost or duplicated chunk shows up as a mismatch, a
// hang, or an error - not as a silent success.
//
// The check writes device memory. On the emulator's register file at 0x50 that
// is harmless; on real hardware the first scanned device may be an EEPROM or
// anything else, so a "bulk verify mismatch" line here is the device's page
// semantics showing through, not a driver bug (see README.md).
func bulkTransfer(bus *i2c.Bus, addr uint8) {
	dev, err := bus.Device(addr, i2c.SpeedStandard)
	if err != nil {
		println("i2c bulk error:", err.Error())
		return
	}
	defer dev.Close()

	// 41 bytes on the wire: register pointer + 40 data bytes, three past the
	// 32-byte FIFO. The pattern steps by 7 so a shifted chunk cannot look
	// like the original data.
	var wr [41]byte
	wr[0] = 0x00 // register pointer
	for i := 1; i < len(wr); i++ {
		wr[i] = uint8(0x10 + i*7)
	}
	if err := dev.Write(wr[:]); err != nil {
		println("i2c bulk write failed:", err.Error())
		return
	}
	println("i2c bulk write len=41 ok")

	var rd [40]byte
	if err := dev.ReadReg(0x00, rd[:]); err != nil {
		println("i2c bulk read failed:", err.Error())
		return
	}
	println("i2c bulk read len=40 ok")

	bad := 0
	for i := 0; i < len(rd); i++ {
		if rd[i] != wr[i+1] {
			bad++
		}
	}
	if bad == 0 {
		println("i2c bulk verify ok")
		return
	}
	println("i2c bulk verify mismatch=" + dec(bad))
}

// hex2 formats one byte as two lowercase hex digits (println cannot print a
// hex number, and fmt would pull the whole formatting machinery into the
// image).
func hex2(v uint8) string {
	const digits = "0123456789abcdef"
	return string([]byte{digits[v>>4], digits[v&0x0F]})
}

// dec formats a small non-negative number in decimal, for the same reason as
// hex2: the golden output must not depend on how println spaces its operands.
func dec(v int) string {
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
