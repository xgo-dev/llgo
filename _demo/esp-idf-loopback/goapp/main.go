// Loopback acceptance probe for the SPI / UART / ADC bindings on the
// ESP32-S31-Function-EvalBoard, one firmware for all three data paths.
//
// Built with: llgo build -target esp32s31-idf -o ../idfproj/main/llgo_out.o ./goapp
//
// The wiring (four jumper wires, header J2):
//
//	20 (GPIO42, SPI MOSI)  -> 17 (GPIO43, SPI MISO)   SPI loopback
//	15 (GPIO45, UART1 TX)  -> 16 (GPIO46, UART1 RX)   UART loopback
//	13 (GPIO47, ADC1 ch5)  -> 35 (3V3) or  1 (GND)    ADC known voltage
//
// The probe runs the SPI and UART loops once (fixed patterns compared
// byte-exact, at two SPI clock rates), then samples ADC on GPIO47 in a
// loop while printing instructions: touch the wire to 3V3, read ~3300 mV,
// touch it to GND, read ~0 mV. Every verdict prints one `[go] lb ...`
// line; the emulation cannot model the wires, so this firmware is a
// hardware item by construction (each binding's own demo pins the control
// plane under esp-emu).
package main

import (
	_ "unsafe" // for go:linkname

	"github.com/xgo-dev/llgo/pkg/esp/adc"
	"github.com/xgo-dev/llgo/pkg/esp/spi"
	"github.com/xgo-dev/llgo/pkg/esp/uart"
)

//go:linkname vTaskDelay C.vTaskDelay
func vTaskDelay(ticks uint32)

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

func decSigned32(v int32) string {
	if v < 0 {
		return "-" + dec32(uint32(-int64(v)))
	}
	return dec32(uint32(v))
}

func hexByte(b byte) string {
	const hex = "0123456789abcdef"
	return string([]byte{hex[b>>4], hex[b&0xf]})
}

func hexBytes(b []byte, n int) string {
	if len(b) < n {
		n = len(b)
	}
	s := ""
	for i := 0; i < n; i++ {
		s += hexByte(b[i])
	}
	return s
}

func errText(err error) string {
	if err == nil {
		return "ok"
	}
	return err.Error()
}

func passfail(ok bool) string {
	if ok {
		return "PASS"
	}
	return "FAIL"
}

// spiLoop drives the SPI master in NoCS mode with MOSI wired to MISO:
// every Transfer's rx must be byte-identical to its tx. Two clock rates
// both run, so the DMA path and the SetFrequency path are both exercised.
func spiLoop() bool {
	println("[go] lb spi: init mosi=42 miso=43 sclk=44 nocls freq=10MHz")
	if err := spi.Init(42, 43, 44, spi.NoCS, spi.Freq10MHz); err != nil {
		println("[go] lb spi init err =", errText(err))
		return false
	}

	ok := true
	for _, hz := range []uint32{spi.Freq10MHz, spi.Freq20MHz} {
		if hz != spi.Freq10MHz {
			if err := spi.SetFrequency(hz); err != nil {
				println("[go] lb spi setfreq err =", errText(err))
				ok = false
				break
			}
		}
		// 260 bytes puts the transfer tail off the 4-byte word boundary,
		// exercising the binding's RX staging path on every transfer.
		tx := make([]byte, 260)
		for i := range tx {
			tx[i] = byte(i*7) + byte(hz)
		}
		rx := make([]byte, len(tx))
		if _, err := spi.Transfer(tx, rx); err != nil {
			println("[go] lb spi transfer hz=" + dec32(hz) + " err =", errText(err))
			ok = false
			break
		}
		mismatch := -1
		for i := range tx {
			if tx[i] != rx[i] {
				mismatch = i
				break
			}
		}
		if mismatch >= 0 {
			println("[go] lb spi MISMATCH hz=" + dec32(hz) +
				" at=" + dec32(uint32(mismatch)) +
				" want=" + hexByte(tx[mismatch]) +
				" got=" + hexByte(rx[mismatch]))
			println("[go] lb spi FAIL hz=" + dec32(hz))
			ok = false
			break
		}
		println("[go] lb spi loopback hz=" + dec32(hz) + " n=260 exact")
	}

	// Odd-length transfer: the staging edge.
	if ok {
		tx := []byte{0xde, 0xad, 0xbe}
		rx := make([]byte, 3)
		if _, err := spi.Transfer(tx, rx); err != nil {
			println("[go] lb spi transfer len3 err =", errText(err))
			ok = false
		} else if rx[0] != 0xde || rx[1] != 0xad || rx[2] != 0xbe {
			println("[go] lb spi MISMATCH len3 got=" + hexBytes(rx, 3))
			println("[go] lb spi FAIL len3")
			ok = false
		} else {
			println("[go] lb spi loopback len=3 exact")
		}
	}

	if err := spi.Deinit(); err != nil {
		println("[go] lb spi deinit err =", errText(err))
	}
	if ok {
		println("[go] lb SPI PASS")
	}
	return ok
}

// uartLoop drives UART1 with TX wired to RX: write a 64-byte pattern, read
// it back byte-exact, then exercise SetBaud.
func uartLoop() bool {
	println("[go] lb uart: init port=1 tx=45 rx=46 baud=115200")
	if err := uart.Init(uart.UART1, 45, 46, 115200); err != nil {
		println("[go] lb uart init err =", errText(err))
		return false
	}

	tx := make([]byte, 64)
	for i := range tx {
		tx[i] = byte(i*3 + 1)
	}
	n, err := uart.Write(uart.UART1, tx)
	if err != nil || n != len(tx) {
		println("[go] lb uart write n=" + dec32(uint32(n)) + " err =", errText(err))
		println("[go] lb uart FAIL write")
		return false
	}
	rx := make([]byte, len(tx)+8) // read with headroom; want exactly len(tx)
	got := 0
	for tries := 0; tries < 40 && got < len(tx); tries++ {
		vTaskDelay(2)
		n, err = uart.Read(uart.UART1, rx[got:], 100)
		if err != nil || n <= 0 {
			continue
		}
		got += n
	}
	if got != len(tx) {
		println("[go] lb uart read got=" + dec32(uint32(got)) + " want=64")
		println("[go] lb uart FAIL read")
		return false
	}
	mismatch := -1
	for i := range tx {
		if tx[i] != rx[i] {
			mismatch = i
			break
		}
	}
	if mismatch >= 0 {
		println("[go] lb uart MISMATCH at=" + dec32(uint32(mismatch)) +
			" want=" + hexByte(tx[mismatch]) + " got=" + hexByte(rx[mismatch]))
		println("[go] lb uart FAIL loopback")
		return false
	}
	println("[go] lb uart loopback n=64 exact")

	if err := uart.SetBaud(uart.UART1, 9600); err != nil {
		println("[go] lb uart setbaud err =", errText(err))
		println("[go] lb uart FAIL baud")
		return false
	}
	println("[go] lb uart setbaud 9600 ok")
	if err := uart.SetBaud(uart.UART1, 115200); err != nil {
		println("[go] lb uart setbaud back err =", errText(err))
		println("[go] lb uart FAIL baud")
		return false
	}
	if err := uart.Deinit(uart.UART1); err != nil {
		println("[go] lb uart deinit err =", errText(err))
	}
	println("[go] lb UART PASS")
	return true
}

// adcLoop samples GPIO47 (ADC1) so a jumper to 3V3 reads full scale and a
// jumper to GND reads zero. The loop prints instructions and keeps reading
// every second - the operator moves one wire, the console shows the level
// follow it.
func adcLoop() {
	println("[go] lb adc: init unit1 bitwidth17 atten0")
	if err := adc.Init(adc.Unit1, adc.DefaultOpts()); err != nil {
		println("[go] lb adc init err =", errText(err))
		println("[go] lb adc FAIL init")
		return
	}
	if err := adc.Config(47, adc.Atten0dB); err != nil {
		println("[go] lb adc config err =", errText(err))
		println("[go] lb adc FAIL config")
		return
	}
	ver, verr := adc.CaliEfuseVersion()
	println("[go] lb adc caliefuse =", decSigned32(ver), "err =", errText(verr))

	println("[go] lb: move the GPIO47 wire between J2-35 (3V3) and J2-1 (GND);")
	println("[go] lb: expected ~3300 mV on 3V3, ~0 mV on GND. Reading every second:")

	for round := 0; round < 30; round++ {
		raw, err := adc.Read(47)
		if err != nil {
			println("[go] lb adc round=" + dec32(uint32(round)) + " read err =", errText(err))
		} else {
			mv, merr := adc.ReadMillivolts(47)
			println("[go] lb adc round=" + dec32(uint32(round)) +
				" raw=" + dec32(raw) +
				" mv=" + decSigned32(mv) +
				" " + errText(merr))
		}
		vTaskDelay(100)
	}
	if err := adc.Deinit(adc.Unit1); err != nil {
		println("[go] lb adc deinit err =", errText(err))
	}
	println("[go] lb ADC sampling window done (verify levels follow the wire)")
}

func main() {
	println("[go] lb: SPI/UART/ADC loopback acceptance (Function-EvalBoard J2)")
	println("[go] lb wiring: J2-20 GPIO42 -> J2-17 GPIO43 (SPI), J2-15 GPIO45 -> J2-16 GPIO46 (UART), J2-13 GPIO47 -> 3V3/GND (ADC)")

	spiOK := spiLoop()
	uartOK := uartLoop()
	adcLoop()

	println("[go] lb verdict spi =", passfail(spiOK), " uart =", passfail(uartOK),
		" adc = see levels above")
	if spiOK && uartOK {
		println("[go] lb PASS")
	} else {
		println("[go] lb FAIL")
	}
}
