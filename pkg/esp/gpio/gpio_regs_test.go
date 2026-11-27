//go:build esp32c3 || esp32s31

// Host-build tests for the register tables: the one part of the baremetal GPIO
// backend a host can check. They run with -tags esp32c3 or -tags esp32s31.
//
// Every expectation here is an absolute address written as a literal, taken
// from the SOC headers by hand, so the arithmetic in gpio_regs_<chip>.go is
// checked against the header rather than against itself: a wrong base, stride,
// first-pad offset, register offset or gap mask makes one of these fail. The
// header line for each value is next to the constant in gpio_regs_<chip>.go.
package gpio

import "testing"

// table is what the headers say about one chip.
type table struct {
	chip    string
	pinMax  int32   // highest usable pin
	gaps    []int32 // pads in the pad map that are not GPIOs
	ioMux0  uintptr // ioMuxAddr(0)
	ioMuxN  uintptr // ioMuxAddr(pinMax)
	out     uintptr // GPIO_OUT_REG
	outSet  uintptr // GPIO_OUT_W1TS_REG
	outClr  uintptr // GPIO_OUT_W1TC_REG
	enSet   uintptr // GPIO_ENABLE_W1TS_REG
	enClear uintptr // GPIO_ENABLE_W1TC_REG
	in      uintptr // GPIO_IN_REG
}

func wantTable(t *testing.T) table {
	t.Helper()
	switch gpioBaremetalChip {
	case "esp32c3":
		return table{
			chip:    "esp32c3",
			pinMax:  21,
			gaps:    nil,
			ioMux0:  0x60009004, // io_mux_reg.h:163 XTAL_32K_P_U + 0x04
			ioMuxN:  0x60009058, // io_mux_reg.h:253 GPIO21_U + 0x58
			out:     0x60004004, // gpio_reg.h:50
			outSet:  0x60004008, // gpio_reg.h:58
			outClr:  0x6000400C, // gpio_reg.h:66
			enSet:   0x60004024, // gpio_reg.h:90
			enClear: 0x60004028, // gpio_reg.h:98
			in:      0x6000403C, // gpio_reg.h:114
		}
	case "esp32s31":
		return table{
			chip:   "esp32s31",
			pinMax: 61,
			gaps:   []int32{29, 41},
			ioMux0: 0x20582000, // io_mux_reg.h:148 U_XTAL32K_HYS_P + 0x00
			// The pad map is dense: 4*pin holds for every pin, including
			// the two holes (U_GPIO29 + 0x74, U_GPIO41 + 0xa4), and the
			// slot past the last GPIO is U_GPIO62 + 0xf8 (io_mux_reg.h:503).
			ioMuxN:  0x205820F4, // io_mux_reg.h:499 U_GPIO61 + 0xf4
			out:     0x20583004, // gpio_reg.h:37
			outSet:  0x20583008, // gpio_reg.h:53
			outClr:  0x2058300C, // gpio_reg.h:69
			enSet:   0x20583038, // gpio_reg.h:201
			enClear: 0x2058303C, // gpio_reg.h:218
			in:      0x20583064, // gpio_reg.h:333
		}
	}
	t.Fatalf("no expected table for chip %q", gpioBaremetalChip)
	return table{}
}

// TestIOMuxAddressing pins both ends of the pad register map.
func TestIOMuxAddressing(t *testing.T) {
	w := wantTable(t)
	if got := ioMuxAddr(0); got != w.ioMux0 {
		t.Errorf("%s ioMuxAddr(0) = %#x, want %#x", w.chip, got, w.ioMux0)
	}
	if got := ioMuxAddr(w.pinMax); got != w.ioMuxN {
		t.Errorf("%s ioMuxAddr(%d) = %#x, want %#x", w.chip, w.pinMax, got, w.ioMuxN)
	}
}

// TestGPIORegisterOffsets pins the five register addresses the backend writes
// through, so a wrong offset cannot hide behind the base arithmetic.
func TestGPIORegisterOffsets(t *testing.T) {
	w := wantTable(t)
	for _, c := range []struct {
		name string
		got  uintptr
		want uintptr
	}{
		{"GPIO_OUT", gpioBase + gpioRegOut, w.out},
		{"GPIO_OUT_W1TS", gpioBase + gpioRegOutW1TS, w.outSet},
		{"GPIO_OUT_W1TC", gpioBase + gpioRegOutW1TC, w.outClr},
		{"GPIO_ENABLE_W1TS", gpioBase + gpioRegEnableW1S, w.enSet},
		{"GPIO_ENABLE_W1TC", gpioBase + gpioRegEnableW1C, w.enClear},
		{"GPIO_IN", gpioBase + gpioRegIn, w.in},
	} {
		if c.got != c.want {
			t.Errorf("%s %s = %#x, want %#x", w.chip, c.name, c.got, c.want)
		}
	}
}

// TestPinValidity pins the usable pin set: every pad below the count except
// the two holes on the S31 (soc_caps.h:242 SOC_GPIO_VALID_GPIO_MASK).
func TestPinValidity(t *testing.T) {
	w := wantTable(t)
	if gpioPinCount != w.pinMax+1 {
		t.Fatalf("%s gpioPinCount = %d, want %d", w.chip, gpioPinCount, w.pinMax+1)
	}
	for pin := int32(-1); pin <= w.pinMax+1; pin++ {
		want := pin >= 0 && pin <= w.pinMax
		for _, gap := range w.gaps {
			if pin == gap {
				want = false
			}
		}
		if got := pinOK(pin); got != want {
			t.Errorf("%s pinOK(%d) = %v, want %v", w.chip, pin, got, want)
		}
	}
}
