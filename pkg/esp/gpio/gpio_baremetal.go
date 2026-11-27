//go:build llgo && baremetal && !llgo.idf && (esp32c3 || esp32s31)

// The register-level GPIO backend: no ESP-IDF, no FreeRTOS, no glue template.
// It keeps the package's public API byte for byte (gpio.go) and implements it
// by writing the chip's GPIO and I/O MUX registers directly, so a plain
// `llgo build -target esp32s31` program can drive a pin.
//
// The backend is chosen by build tags, not by a runtime switch: llgo.idf
// builds keep the ESP-IDF driver path (gpio_idf.go), and every other build
// keeps the honest stub (gpio_stub.go) - except a baremetal llgo build of a
// chip that has a register table (gpio_regs_<chip>.go), which is this file.
// That is why `baremetal` alone is not enough for the tag below: the tag names
// the chips too, so a Cortex-M or AVR target keeps the stub instead of
// compiling a backend with no table behind it.
//
// Byte-level access goes through pkg/esp/internal/mmio, whose C shim is the
// only place a `volatile` load or store is expressible; a Go-level
// `*(*uint32)(unsafe.Pointer(addr))` would be merged or dropped by the
// optimizer.
package gpio

import (
	"github.com/xgo-dev/llgo/pkg/esp/internal/mmio"
)

// errInvalidPin is the ESP_ERR_INVALID_ARG value (0x102) the ESP-IDF glue
// returns for a pin outside [0, 64) (glue/gpio_glue.c). The baremetal backend
// returns the same code for a pin outside the chip's own range, so callers see
// one behaviour across both backends.
const errInvalidPin = 0x102

// I/O MUX bit fields. The two supported SoCs share the layout, so it lives
// here rather than in the per-chip tables:
//
//	components/soc/esp32c3/register/soc/io_mux_reg.h:31-65
//	components/soc/esp32s31/register/soc/io_mux_reg.h:42-65
//
// FUN_PD bit 7 (pulldown), FUN_PU bit 8 (pullup), FUN_IE bit 9 (input buffer),
// FUN_DRV bits [11:10] (drive strength), MCU_SEL bits [14:12] (pad function).
const (
	ioMuxFunPD    uint32 = 1 << 7
	ioMuxFunPU    uint32 = 1 << 8
	ioMuxFunIE    uint32 = 1 << 9
	ioMuxMcuSel          = 0x7 << 12
	ioMuxMcuSelSh        = 12
	// pinFuncGPIO is PIN_FUNC_GPIO: the MCU_SEL value that hands the pad to
	// the GPIO block (io_mux_reg.h:120 on the C3, :140 on the S31). Both
	// chips use 1, and every peripheral function on them is either 0 or a
	// value above 1.
	pinFuncGPIO uint32 = 1
)

// unsupported reports the shared "invalid pin" code for a pin this SoC does
// not have. There is no "no register table" case left: the build tag above
// selects this file only for the chips whose table exists.
func unsupported(pin int32) int32 {
	if !pinOK(pin) {
		return errInvalidPin
	}
	return 0
}

// setDirection hands the pad to the GPIO block and programs direction and
// pulls. It mirrors what the ESP-IDF glue asks gpio_config for: Output also
// turns the input buffer on, because gpio.go promises that a SetLevel->GetLevel
// round trip reads back the driven level.
func setDirection(pin int32, mode int32, pull int32) int32 {
	if e := unsupported(pin); e != 0 {
		return e
	}
	mux := ioMuxAddr(pin)
	bit := uint32(1) << uint(pin)

	// MCU_SEL = PIN_FUNC_GPIO routes the pad through the GPIO block; the
	// drive strength and the sleep-mode bits keep their reset values.
	mmio.Clear32(mux, ioMuxMcuSel)
	mmio.Set32(mux, pinFuncGPIO<<ioMuxMcuSelSh)

	// Always enable the input buffer: an input needs it to read, an output
	// needs it for GetLevel to see the driven level.
	mmio.Set32(mux, ioMuxFunIE)

	switch Pull(pull) {
	case PullUp:
		mmio.Clear32(mux, ioMuxFunPD)
		mmio.Set32(mux, ioMuxFunPU)
	case PullDown:
		mmio.Clear32(mux, ioMuxFunPU)
		mmio.Set32(mux, ioMuxFunPD)
	default:
		mmio.Clear32(mux, ioMuxFunPU|ioMuxFunPD)
	}

	// Write-1-to-set / write-1-to-clear registers: one store of the mask,
	// never a read-modify-write. They are write-only, so a read returns a
	// value with no defined meaning, and `read | mask` / `read &^ mask`
	// written back would either do nothing (read as 0) or touch every other
	// pin's enable bit (a 1 in the mask clears that pin).
	if Mode(mode) == Output {
		mmio.Write32(gpioBase+gpioRegEnableW1S, bit)
	} else {
		mmio.Write32(gpioBase+gpioRegEnableW1C, bit)
	}
	return 0
}

// setLevel drives an output pin through the write-1-to-set/clear registers: a
// single store of one bit, so no other pin's level can be disturbed and no
// read of the write-only register is involved.
func setLevel(pin int32, level int32) int32 {
	if e := unsupported(pin); e != 0 {
		return e
	}
	bit := uint32(1) << uint(pin)
	if level != 0 {
		mmio.Write32(gpioBase+gpioRegOutW1TS, bit)
	} else {
		mmio.Write32(gpioBase+gpioRegOutW1TC, bit)
	}
	return 0
}

// getLevel reads the pin's input path.
func getLevel(pin int32) int32 {
	if unsupported(pin) != 0 {
		return 0
	}
	return int32(mmio.Read32(gpioBase+gpioRegIn) >> uint(pin) & 1)
}

// toggle inverts an output pin. The driver output register carries every pin's
// driven level; only the one bit is written back through the W1T registers.
func toggle(pin int32) int32 {
	if e := unsupported(pin); e != 0 {
		return e
	}
	bit := uint32(1) << uint(pin)
	if mmio.Read32(gpioBase+gpioRegOut)&bit != 0 {
		return setLevel(pin, 0)
	}
	return setLevel(pin, 1)
}
