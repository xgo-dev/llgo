//go:build esp32c3 || esp32s31

// The pin arithmetic the register tables feed. It sits between the per-chip
// tables (gpio_regs_<chip>.go, selected by the chip build tag) and the
// backend (gpio_baremetal.go, selected by llgo && baremetal): keeping it out
// of both means `go test -tags esp32c3` can check the addressing on the host
// against the header values, which no baremetal binary can be asked to do.
package gpio

// pinOK reports whether pin is a usable GPIO on this chip.
func pinOK(pin int32) bool {
	if pin < 0 || pin >= gpioPinCount {
		return false
	}
	return gpioPinGapMask&(1<<uint(pin)) == 0
}

// ioMuxAddr is the I/O MUX configuration register of one pad.
func ioMuxAddr(pin int32) uintptr {
	return ioMuxBase + ioMuxFirstOffset + uintptr(pin)*ioMuxStride
}
