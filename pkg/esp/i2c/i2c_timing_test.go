// Host-build tests for the bus-timing arithmetic. This is the only part of the
// baremetal I2C backend a host can check: the values below are what
// i2c_ll_master_cal_bus_clk produces for a 40 MHz XTAL source, computed with
// the ESP-IDF formula spelled out in the test so a transcription slip in
// busTimingFor fails here instead of on a board.
package i2c

import "testing"

func TestBusTiming(t *testing.T) {
	cases := []struct {
		speed uint32
		want  busTiming
	}{
		// 100 kHz: div = 40e6/(100e3*1024)+1 = 1, sclk = 40 MHz,
		// half = 200 -> low 200, waitHigh 98, high 102, hold 50, sample
		// 100, setup/hold 200.
		{speed: 100_000, want: busTiming{
			clkDiv: 1, low: 200, high: 102, waitHigh: 98,
			sdaHold: 50, sdaSampl: 100, setup: 200, hold: 200,
		}},
		// 400 kHz: half = 50 -> low 50, waitHigh 23, high 27, hold 12,
		// sample 25, setup/hold 50.
		{speed: 400_000, want: busTiming{
			clkDiv: 1, low: 50, high: 27, waitHigh: 23,
			sdaHold: 12, sdaSampl: 25, setup: 50, hold: 50,
		}},
	}
	for _, c := range cases {
		got := busTimingFor(i2cSourceHzTest, c.speed)
		if got != c.want {
			t.Errorf("busTimingFor(40MHz, %d) = %+v, want %+v", c.speed, got, c.want)
		}
	}
}

// i2cSourceHzTest is the crystal both supported SoCs run their I2C block from.
const i2cSourceHzTest uint32 = 40_000_000

// TestBusTimingRegistersFit pins the field widths the timing values are written
// through: the period registers are 9 bits wide and scl_wait_high is 7 bits, so
// a rate that would overflow them has to be refused up front instead of
// silently running at the wrong speed.
func TestBusTimingRegistersFit(t *testing.T) {
	for _, speed := range []uint32{minSpeedHz, 50_000, 80_000, SpeedStandard, SpeedFast, maxSpeedHz} {
		tm := busTimingFor(i2cSourceHzTest, speed)
		if tm.low > 0x1FF || tm.high > 0x1FF || tm.waitHigh > 0x7F ||
			tm.sdaHold > 0x1FF || tm.sdaSampl > 0x1FF || tm.setup > 0x1FF || tm.hold > 0x1FF {
			t.Errorf("speed %d produces a timing that overflows a register: %+v", speed, tm)
		}
		if tm.low == 0 || tm.high == 0 || tm.sdaHold == 0 || tm.sdaSampl == 0 || tm.setup == 0 || tm.hold == 0 {
			t.Errorf("speed %d produces a zero timing field: %+v", speed, tm)
		}
	}
}

// TestValidSpeed pins the accepted range: the endpoints are inclusive and
// anything outside them is refused before a transfer programs the bus.
func TestValidSpeed(t *testing.T) {
	cases := []struct {
		speed uint32
		want  bool
	}{
		{0, false},
		{minSpeedHz - 1, false},
		{minSpeedHz, true},
		{SpeedStandard, true},
		{SpeedFast, true},
		{maxSpeedHz, true},
		{maxSpeedHz + 1, false},
	}
	for _, c := range cases {
		if got := validSpeed(c.speed); got != c.want {
			t.Errorf("validSpeed(%d) = %v, want %v", c.speed, got, c.want)
		}
	}
}

// TestSCLTimeoutVal pins the TO register transcription. IDF encodes the SCL
// timeout as an exponent n, hardware gives up after 2^n source ticks, and
// i2c_ll_cal_scl_timeout computes 32 - __builtin_clz(ticks), which is
// bits.Len32. 40 MHz * 2000us = 80000 ticks -> 17 (2^17 = 131072 >= 80000,
// 2^16 = 65536 < 80000); the same for 2500us on the S31 default. The wrong
// formula 32 - bits.Len32 returns 15 here, which no arithmetic mistake can
// bridge, so a regression flips this test.
func TestSCLTimeoutVal(t *testing.T) {
	cases := []struct {
		sourceHz uint32
		waitUS   uint32
		want     uint32
	}{
		{i2cSourceHzTest, 2000, 17}, // C3 default (I2C_LL_SCL_WAIT_US_VAL_DEFAULT)
		{i2cSourceHzTest, 2500, 17}, // S31 default
		{i2cSourceHzTest, 1000, 16}, // 40000 ticks: 2^16 >= 40000 > 2^15
		{i2cSourceHzTest, 100, 12},  // 4000 ticks: 2^12 >= 4000 > 2^11
	}
	for _, c := range cases {
		if got := sclTimeoutVal(c.sourceHz, c.waitUS); got != c.want {
			t.Errorf("sclTimeoutVal(%d, %d) = %d, want %d", c.sourceHz, c.waitUS, got, c.want)
		}
	}
}
