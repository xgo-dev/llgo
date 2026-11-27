// The bus-timing arithmetic the baremetal backend programs the controller
// with. It is chip-independent on purpose: it takes the source-clock
// frequency as an argument, so it compiles and is unit-tested on a host, which
// is the only place the numbers can be checked without a scope on a board.
package i2c

import "math/bits"

// busTiming is one SCL frequency expressed in I2C source-clock ticks. It
// mirrors i2c_ll_master_cal_bus_clk and i2c_ll_master_set_bus_timing
// (components/esp_hal_i2c/<chip>/include/hal/i2c_ll.h, identical on both
// supported chips), so the baremetal backend produces the waveforms the
// ESP-IDF driver would.
type busTiming struct {
	clkDiv   uint32 // source clock divider: SCLK = source / clkDiv
	low      uint32 // SCL low period
	high     uint32 // SCL high period
	waitHigh uint32 // portion of the high period spent waiting for the bus
	sdaHold  uint32 // SDA hold time after the SCL falling edge
	sdaSampl uint32 // SDA sample point inside the high period
	setup    uint32 // START/STOP setup time
	hold     uint32 // START/STOP hold time
}

// The two bus rates the package documents. The register math is valid outside
// them; these are the rates real buses (and the ESP-IDF driver's own tests)
// run at.
const (
	// SpeedStandard is the 100 kHz standard-mode bus rate, the speed a
	// scan and an unlabelled device are safe at.
	SpeedStandard = 100_000
	// SpeedFast is the 400 kHz fast-mode bus rate.
	SpeedFast = 400_000
)

// validSpeed bounds. Below the minimum the divider cannot express the
// half-cycle; above the maximum the 9-bit period field overflows and the
// hardware would run slower than asked.
const (
	minSpeedHz uint32 = 10_000
	maxSpeedHz uint32 = 1_000_000
)

// busTimingFor converts a bus frequency into source-clock tick counts.
//
// The transcription follows the ESP-IDF HAL exactly, including the two choices
// that are not obvious from the register descriptions: scl_high and
// scl_wait_high are written without the -1 the TRM asks for (IDF measured that
// subtracting one makes SCL faster than requested), and the 80 kHz boundary
// switches scl_wait_high from a quarter period to half the period minus two
// ticks.
func busTimingFor(sourceHz, speedHz uint32) busTiming {
	clkDiv := sourceHz/(speedHz*1024) + 1
	sclkHz := sourceHz / clkDiv
	half := sclkHz / speedHz / 2

	t := busTiming{clkDiv: clkDiv, low: half}
	if speedHz >= 80_000 {
		t.waitHigh = half/2 - 2
	} else {
		t.waitHigh = half / 4
	}
	t.high = half - t.waitHigh
	t.sdaHold = half / 4
	t.sdaSampl = half / 2
	t.setup = half
	t.hold = half
	return t
}

// sclTimeoutVal converts a "wait this long for a stretched SCL" budget into
// the TO register value, the way the ESP-IDF master driver does it
// (i2c_ll_calculate_timeout_us_to_reg_val: 32 - clz(clk_per_us * timeout_us),
// esp_hal_i2c/<chip>/include/hal/i2c_ll.h).
//
// It is deliberately not derived from the bus timing: the hardware timeout
// bounds how long a slave may hold SCL low, which has nothing to do with the
// SCL period. IDF sets the register from the bus clock first
// (i2c_ll_master_set_bus_timing) and then overwrites it from
// I2C_LL_SCL_WAIT_US_VAL_DEFAULT (i2c_master.c: i2c_hal_master_set_scl_timeout_val
// right after i2c_hal_set_bus_timing), so the default is the value to match.
func sclTimeoutVal(sourceHz, waitUS uint32) uint32 {
	ticks := sourceHz / 1_000_000 * waitUS
	// IDF encodes the timeout as an exponent: the register holds n and the
	// hardware gives up after 2^n source ticks. i2c_ll_cal_scl_timeout
	// computes 32 - __builtin_clz(clk_cycle_num_per_us * timeout_us)
	// (esp_hal_i2c/esp32c3/include/hal/i2c_ll.h:980, esp32s31 :988), which is
	// exactly bits.Len32.
	return uint32(bits.Len32(ticks))
}

// validSpeed reports whether the timing math can express speedHz.
func validSpeed(speedHz uint32) bool {
	return speedHz >= minSpeedHz && speedHz <= maxSpeedHz
}
