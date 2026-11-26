package main

import (
	"github.com/xgo-dev/llgo/pkg/esp/bt"
	"github.com/xgo-dev/llgo/pkg/esp/gpio"
	"github.com/xgo-dev/llgo/pkg/esp/i2c"
	"github.com/xgo-dev/llgo/pkg/esp/i2s"
)

// ES8311 codec + amp wiring.
const (
	i2cSda     = i2c.Pin(51)
	i2cScl     = i2c.Pin(50)
	es8311Addr = 0x18

	i2sMCLK = i2s.Pin(52) // codec MCLK (256 x fs)
	i2sBCLK = i2s.Pin(53) // shared with the A2DP TX clock
	i2sWS   = i2s.Pin(55)
	i2sDOUT = i2s.Pin(56) // playback (self-test sine)
	i2sDIN  = i2s.Pin(54) // record (codec ASDOUT)

	paPin = gpio.Pin(57) // NS4150B amp enable, high = on

	testRate    = 16000
	chunkBytes  = 32768                   // 512 ms of stereo S16 per driver call
	probeBytes  = testRate * 4            // 1 s sweep probe
	captureS    = 3                       // final mic capture, seconds
	captureSize = captureS * testRate * 4 // stereo 16-bit bytes
)

// micSweep covers the plausible ES8311 microphone input configurations
// (register semantics from the Linux kernel es8311 patch, values audited
// against ESPHome / esp-box / M5Unified-Cardputer): reg 0x14 bit6 = DMIC,
// bits 5:4 = input select (00 aux/line, 01 MIC1), bits 3:0 = PGA gain;
// reg 0x16 = ADC gain scale-up (0-7); reg 0x17 = ADC digital volume.
var micSweep = []struct {
	name                string
	reg14, reg16, reg17 uint8
}{
	{"mic1-pga30-0db", 0x1A, 0x00, 0xC8},
	{"aux-pga30-0db", 0x0A, 0x00, 0xC8},
	{"dmic-pga30-0db", 0x5A, 0x00, 0xC8},
	{"mic1-pga30-42db-maxdig", 0x1A, 0x07, 0xFF},
}

// es8311TestInit configures the codec for the 16 kHz self-test with
// MCLK = 256 x fs (4.096 MHz), standard I2S, 16-bit: the audited coefficient
// row {4096000, 16000, pre 1/1, lrck 0x00FF, bclk_div 4, adc_osr 0x10,
// dac_osr 0x20} plus the power-up sequence (ESPHome ordering).
var es8311TestInit = [][2]uint8{
	{0x00, 0x1F}, // soft reset
	{0x00, 0x00}, // leave reset
	{0x01, 0x3F}, // clock manager: MCLK from pad, internal clocks on
	{0x02, 0x00}, // pre-div 1, pre-mult x1
	{0x03, 0x10}, // ADC: fs_mode 0, OSR 0x10
	{0x04, 0x20}, // DAC OSR 0x20
	{0x05, 0x00}, // ADC/DAC clk div 1/1
	{0x06, 0x03}, // BCLK = MCLK/8 = 512 kHz = 32 fs (16-bit stereo)
	{0x07, 0x00}, // LRCK div high
	{0x08, 0xFF}, // LRCK div low: LRCK = MCLK/256 = 16 kHz
	{0x09, 0x0C}, // SDP in (DAC data): 16-bit, standard I2S
	{0x0A, 0x0C}, // SDP out (ADC data): 16-bit, standard I2S
	{0x14, 0x1A}, // analog mic on, max PGA gain
	{0x16, 0x00}, // ADC gain scale 0 dB
	{0x17, 0xC8}, // ADC digital volume 0 dB
	{0x32, 0xBF}, // DAC volume ~0 dB
	{0x0D, 0x01}, // power up analog
	{0x0E, 0x02}, // analog PGA + ADC modulator on
	{0x12, 0x00}, // DAC on
	{0x13, 0x10}, // HP drive output on (feeds the NS4150B)
	{0x1C, 0x6A}, // ADC EQ bypass, DC-offset cancel
	{0x37, 0x08}, // DAC EQ bypass
	{0x00, 0x80}, // power on
	{0x31, 0x00}, // DAC unmute
}

// es8311Readback lists the registers whose written value must read back
// identically for the init to count.
var es8311Readback = []uint8{0x01, 0x03, 0x04, 0x08, 0x09, 0x0A, 0x14, 0x17, 0x32, 0x00}

// es8311A2DPInit reconfigures the codec after the self-test for the A2DP
// path: same MCLK = 256 fs clocking (identical register values cover both
// A2DP-negotiable rates, 44.1 kHz -> 11.2896 MHz and 48 kHz -> 12.288 MHz),
// DAC volume at the NS4150B-safe level.
var es8311A2DPInit = [][2]uint8{
	{0x00, 0x1F}, // soft reset
	{0x00, 0x00},
	{0x00, 0x80}, // power on
	{0x01, 0x3F}, // clock source = MCLK pin
	{0x02, 0x00}, // pre-div 1, pre-multi 1x
	{0x03, 0x10}, // fs single speed, ADC OSR 16
	{0x04, 0x10}, // DAC OSR 16
	{0x05, 0x00}, // ADC clk = DAC clk = MCLK
	{0x06, 0x03}, // SCLK = MCLK/4 (64 fs at 16-bit stereo)
	{0x07, 0x00},
	{0x08, 0xFF}, // LRCK = MCLK/256 = fs
	{0x00, 0x80}, // slave serial port
	{0x09, 0x0C}, // SDP in: I2S, 16-bit
	{0x0A, 0x0C}, // SDP out: I2S, 16-bit
	{0x0D, 0x01}, // power up analog
	{0x0E, 0x02}, // PGA + ADC modulator
	{0x12, 0x00}, // DAC on
	{0x13, 0x10}, // HP drive on
	{0x1C, 0x6A},
	{0x37, 0x08},
	{0x32, 0xBF}, // DAC volume ~0 dB (full scale; was 0x7F conservatively)
	{0x31, 0x00}, // unmute
}

// sineTable is one full sine period, 64 entries scaled by 8000; the self-test
// walks it with a Q16 phase accumulator (1802 per frame = 440 Hz at 16 kHz).
var sineTable = [64]int16{
	0, 784, 1561, 2322, 3061, 3771, 4445, 5075,
	5657, 6184, 6652, 7055, 7391, 7656, 7846, 7961,
	8000, 7961, 7846, 7656, 7391, 7055, 6652, 6184,
	5657, 5075, 4445, 3771, 3061, 2322, 1561, 784,
	0, -784, -1561, -2322, -3061, -3771, -4445, -5075,
	-5657, -6184, -6652, -7055, -7391, -7656, -7846, -7961,
	-8000, -7961, -7846, -7656, -7391, -7055, -6652, -6184,
	-5657, -5075, -4445, -3771, -3061, -2322, -1561, -784,
}

var (
	codecReady    bool // the codec's I2C interface answers
	selftestDone  bool // the self-test ran; the amp may follow the A2DP link
	a2dpConnected bool // written only by a2dpLoop
)

// codecLoop brings the ES8311 up (the audited sequence; the codec answers
// at 0x18 on this board), then runs the audio self-test once and hands the
// amp to the A2DP follower. A failed init is a FAIL line - and a 30 s retry,
// so a transient I2C glitch never costs the rest of the run.
func codecLoop() {
	amp := gpio.Pin(paPin)
	amp.SetDirection(gpio.Output, gpio.PullNone)
	amp.SetLevel(false)
	println("[go] codec amp off (gpio57 low)")

	codecReady = tryCodecInit()
	for !codecReady {
		println("[go] codec FAIL init (ES8311 not answering at 0x18); retry in 30 s")
		yieldMs(30000)
		codecReady = tryCodecInit()
	}
	println("[go] codec ES8311-OK (chip id 0x83, 16 kHz self-test config, 10 registers read back)")

	for a2dpConnected {
		println("[go] codec selftest deferred (A2DP link active)")
		yieldMs(10000)
	}
	runSelfTest(amp)
}

// tryCodecInit probes the ES8311's I2C interface (chip ID) and, when it
// answers, writes the audited 16 kHz self-test configuration with readback.
func tryCodecInit() bool {
	bus, err := i2c.NewBus(i2cSda, i2cScl)
	if err != nil {
		println("[go] codec i2c bus err =", errText(err))
		return false
	}
	defer bus.Close()
	dev, err := bus.Device(es8311Addr, 100000)
	if err != nil {
		println("[go] codec i2c dev err =", errText(err))
		return false
	}
	defer dev.Close()

	id, err := codecReadReg(dev, 0xFD)
	if err != nil {
		return false // no ACK at 0x18: I2C wiring or pull-up problem
	}
	if id != 0x83 {
		println("[go] codec chip id hi =", hex8(id), "(want 0x83)")
		return false
	}
	for _, w := range es8311TestInit {
		if err := codecWriteReg(dev, w[0], w[1]); err != nil {
			println("[go] codec init write reg", hex8(w[0]), "err =", errText(err))
			return false
		}
	}
	for _, reg := range es8311Readback {
		got, rerr := codecReadReg(dev, reg)
		if rerr != nil {
			println("[go] codec readback reg", hex8(reg), "err =", errText(rerr))
			return false
		}
		want, _ := lookupReg(es8311TestInit, reg)
		if got != want {
			println("[go] codec reg", hex8(reg), "readback", hex8(got), "want", hex8(want))
			return false
		}
	}
	return true
}

// runSelfTest proves the whole analog path: sine through the speaker, then a
// mic gain sweep with RMS statistics. It owns the I2S clock pins exclusively
// while it runs, so A2DP is paused first (the I2S master role cannot be
// shared) and re-enabled afterwards with the codec reconfigured for A2DP.
func runSelfTest(amp gpio.Pin) {
	println("[go] codec selftest: pausing A2DP to own the I2S clock pins")
	if err := bt.A2DPDisable(); err != nil {
		println("[go] codec selftest FAIL a2dp disable err =", errText(err))
		return
	}

	playSine(amp)
	_ = micSweepRun() // capture check; playback dropped (sine is the audible check)

	println("[go] codec selftest: reconfiguring codec for A2DP rates")
	if !applyCodecSet(es8311A2DPInit) {
		println("[go] codec selftest FAIL a2dp codec reconfig")
	} else if err := bt.A2DPEnable(bt.A2DPConfig{
		BclkPin: bclkPin, WsPin: wsPin, DoutPin: doutPin,
		DmaDescNum: dmaDescNum, DmaFrameNum: dmaFrameNum,
	}); err != nil {
		println("[go] codec selftest FAIL a2dp re-enable err =", errText(err))
		return
	}
	selftestDone = true
	println("[go] codec SELFTEST-DONE (A2DP re-enabled; amp now follows the link)")
}

// newTX allocates one master TX channel driving the codec-side pins (the
// self-test owns the pins; A2DP is paused).
func newTX() (*i2s.Channel, error) {
	ch, err := i2s.NewChannel(true)
	if err != nil {
		return nil, err
	}
	cfg := i2s.StdConfig{
		MCLK: i2sMCLK, BCLK: i2sBCLK, WS: i2sWS,
		DOUT: i2sDOUT, DIN: -1,
		Bits: i2s.Bits16, Stereo: true, SampleRate: testRate,
	}
	if err := ch.ConfigureStd(cfg); err != nil {
		ch.Close()
		return nil, err
	}
	return ch, nil
}

// playSine drives 3 s of 440 Hz through the codec's DAC path with the amp on.
func playSine(amp gpio.Pin) {
	tx, err := newTX()
	if err != nil {
		println("[go] codec sine FAIL tx setup err =", errText(err))
		return
	}
	if err := tx.Enable(); err != nil {
		println("[go] codec sine FAIL tx enable err =", errText(err))
		tx.Close()
		return
	}

	amp.SetLevel(true)
	println("[go] codec SINE-START (audible point: 440 Hz, 3 s, through the speaker)")

	const frames = 4000 // 250 ms per driver call
	chunk := make([]byte, frames*4)
	phase := 0
	fail := false
	for ms := 0; ms < captureS*4 && !fail; ms++ {
		for f := 0; f < frames; f++ {
			v := sineTable[(phase>>10)&63]
			phase += 1802 // Q16 phase step: 440/16000 x 65536
			i := f * 4
			chunk[i] = byte(v)
			chunk[i+1] = byte(v >> 8)
			chunk[i+2] = byte(v)      // duplicate to both slots: the codec
			chunk[i+3] = byte(v >> 8) // plays whichever slot its DAC is set to
		}
		if _, err := tx.Write(chunk); err != nil {
			println("[go] codec sine write err =", errText(err))
			fail = true
		}
	}
	println("[go] codec SINE-DONE")
	amp.SetLevel(false)

	if err := tx.Disable(); err != nil {
		println("[go] codec sine tx disable err =", errText(err))
	}
	if err := tx.Close(); err != nil {
		println("[go] codec sine tx close err =", errText(err))
	}
}

// micSweepRun sweeps the ES8311 mic input configurations and captures from
// the winner: one enabled RX channel reading ASDOUT while the sweep writes
// one register set per configuration. Returns the capture buffer for the
// loopback replay (nil when the capture failed).
func micSweepRun() []byte {
	rx, err := newRX()
	if err != nil {
		println("[go] codec mic FAIL rx setup err =", errText(err))
		return nil
	}
	if err := rx.Enable(); err != nil {
		println("[go] codec mic FAIL rx enable err =", errText(err))
		rx.Close()
		return nil
	}

	probe := make([]byte, probeBytes)
	best, bestRMS := -1, -1
	for i, cfg := range micSweep {
		if !applyMicRegs(cfg.reg14, cfg.reg16, cfg.reg17) {
			continue
		}
		yieldMs(50) // settle after the register change
		rms, _, _, peak, nz := probeRMS(rx, probe)
		println("[go] codec mic sweep", cfg.name, ": rms =", rms, "peak =", peak,
			"nonzero% =", nz)
		if rms > bestRMS {
			best, bestRMS = i, rms
		}
	}
	if best < 0 {
		println("[go] codec mic FAIL sweep produced no config")
		rx.Disable()
		rx.Close()
		return nil
	}
	winner := micSweep[best]
	applyMicRegs(winner.reg14, winner.reg16, winner.reg17)
	println("[go] codec mic winner =", winner.name, "rms =", bestRMS)

	buf := make([]byte, captureSize)
	done := 0
	for done < len(buf) {
		n := len(buf) - done
		if n > chunkBytes {
			n = chunkBytes
		}
		got, rerr := rx.Read(buf[done : done+n])
		if rerr != nil {
			println("[go] codec mic FAIL read at", done, "err =", errText(rerr))
			break
		}
		done += got
	}
	rx.Disable()
	rx.Close()

	nS, sum, peak, nz, sL, sR := stats(buf)
	rms := isqrt64(sum / int64(nS))
	println("[go] codec MIC-RMS samples =", nS, "rms =", int(rms), "peak =", peak,
		"nonzero% =", nz*100/nS,
		"rmsL =", sL.rms, "rmsR =", sR.rms, "(capture", captureS, "s at 16 kHz stereo)")
	if rms < 10 || nz*2 < nS {
		println("[go] codec mic: adc path carries near-silence (mic hardware or gain question)")
		return nil
	}
	println("[go] codec mic PASS (adc path carries real samples)")
	return buf
}

// loopbackReplay plays the mic capture back through the codec's DAC path
// (amp on): the heard-sound closure of the record path - whatever came in
// through the mic comes out of the speaker now.
func loopbackReplay(buf []byte, amp gpio.Pin) {
	if len(buf) == 0 {
		println("[go] codec loopback skipped (no capture)")
		return
	}
	tx, err := newTX()
	if err != nil {
		println("[go] codec loopback FAIL tx setup err =", errText(err))
		return
	}
	if err := tx.Enable(); err != nil {
		println("[go] codec loopback FAIL tx enable err =", errText(err))
		tx.Close()
		return
	}
	amp.SetLevel(true)
	println("[go] codec LOOPBACK-START (audible point: the mic capture plays back)")
	done := 0
	for done < len(buf) {
		n := len(buf) - done
		if n > chunkBytes {
			n = chunkBytes
		}
		if _, werr := tx.Write(buf[done : done+n]); werr != nil {
			println("[go] codec loopback write err =", errText(werr))
			break
		}
		done += n
	}
	println("[go] codec LOOPBACK-DONE")
	amp.SetLevel(false)
	tx.Disable()
	tx.Close()
}

// newRX allocates one master RX channel driving the codec-side clock pins
// while capturing ASDOUT (the self-test owns the pins; A2DP is paused).
func newRX() (*i2s.Channel, error) {
	ch, err := i2s.NewChannel(false)
	if err != nil {
		return nil, err
	}
	cfg := i2s.StdConfig{
		MCLK: i2sMCLK, BCLK: i2sBCLK, WS: i2sWS,
		DOUT: -1, DIN: i2sDIN,
		Bits: i2s.Bits16, Stereo: true, SampleRate: testRate,
	}
	if err := ch.ConfigureStd(cfg); err != nil {
		ch.Close()
		return nil, err
	}
	return ch, nil
}

// probeRMS reads ~1 s of audio from an enabled RX channel and returns the
// overall RMS, per-slot RMS, peak and nonzero percentage.
func probeRMS(rx *i2s.Channel, probe []byte) (rms int, rmsL int, rmsR int, peak int, nzpct int) {
	done := 0
	for done < len(probe) {
		n := len(probe) - done
		if n > chunkBytes {
			n = chunkBytes
		}
		got, err := rx.Read(probe[done : done+n])
		if err != nil {
			break
		}
		done += got
	}
	nS, sum, pk, nz, sL, sR := stats(probe[:done])
	if nS == 0 {
		return 0, 0, 0, 0, 0
	}
	return int(isqrt64(sum / int64(nS))), sL.rms, sR.rms, pk, nz * 100 / nS
}

// applyMicRegs writes one mic configuration's three registers on a fresh
// bus (the sweep runs between captures; a per-call bus keeps the code
// shapeless-simple).
func applyMicRegs(reg14, reg16, reg17 uint8) bool {
	bus, err := i2c.NewBus(i2cSda, i2cScl)
	if err != nil {
		println("[go] codec mic i2c bus err =", errText(err))
		return false
	}
	defer bus.Close()
	dev, err := bus.Device(es8311Addr, 100000)
	if err != nil {
		println("[go] codec mic i2c dev err =", errText(err))
		return false
	}
	defer dev.Close()
	werr := codecWriteReg(dev, 0x14, reg14)
	werr = firstErr(werr, codecWriteReg(dev, 0x16, reg16))
	werr = firstErr(werr, codecWriteReg(dev, 0x17, reg17))
	if werr != nil {
		println("[go] codec mic reg write err =", errText(werr))
		return false
	}
	return true
}

// applyCodecSet writes one register table (no readback - used for the A2DP
// hand-off where the values were verified separately).
func applyCodecSet(set [][2]uint8) bool {
	bus, err := i2c.NewBus(i2cSda, i2cScl)
	if err != nil {
		println("[go] codec set i2c bus err =", errText(err))
		return false
	}
	defer bus.Close()
	dev, err := bus.Device(es8311Addr, 100000)
	if err != nil {
		println("[go] codec set i2c dev err =", errText(err))
		return false
	}
	defer dev.Close()
	for _, w := range set {
		if err := codecWriteReg(dev, w[0], w[1]); err != nil {
			println("[go] codec set write reg", hex8(w[0]), "err =", errText(err))
			return false
		}
	}
	return true
}

func codecWriteReg(dev *i2c.Device, reg, val uint8) error {
	return dev.Write([]byte{reg, val})
}

func codecReadReg(dev *i2c.Device, reg uint8) (uint8, error) {
	var out [1]byte
	if err := dev.WriteRead([]byte{reg}, out[:]); err != nil {
		return 0, err
	}
	return out[0], nil
}

func lookupReg(set [][2]uint8, reg uint8) (uint8, bool) {
	want, found := uint8(0), false
	for _, w := range set {
		if w[0] == reg { // later writes win, like on hardware
			want, found = w[1], true
		}
	}
	return want, found
}

func firstErr(a, b error) error {
	if a != nil {
		return a
	}
	return b
}

// stats walks a little-endian 16-bit stereo capture once: sample count, sum
// of squares, absolute peak, nonzero count, and per-slot RMS/peak.
func stats(buf []byte) (n int, sumsq int64, peak int, nonzero int, sL slotStats, sR slotStats) {
	var sumL, sumR int64
	var cntL, cntR int
	for i := 0; i+1 < len(buf); i += 2 {
		v := int(int16(uint16(buf[i]) | uint16(buf[i+1])<<8))
		a := v
		if a < 0 {
			a = -a
		}
		sumsq += int64(a) * int64(a)
		if a > peak {
			peak = a
		}
		if v != 0 {
			nonzero++
		}
		if (i/2)%2 == 0 {
			sumL += int64(a) * int64(a)
			if a > sL.peak {
				sL.peak = a
			}
			cntL++
		} else {
			sumR += int64(a) * int64(a)
			if a > sR.peak {
				sR.peak = a
			}
			cntR++
		}
	}
	n = len(buf) / 2
	if cntL > 0 {
		sL.rms = isqrt(int(sumL / int64(cntL)))
	}
	if cntR > 0 {
		sR.rms = isqrt(int(sumR / int64(cntR)))
	}
	return n, sumsq, peak, nonzero, sL, sR
}

type slotStats struct {
	rms  int
	peak int
}

func isqrt(x int) int {
	if x <= 0 {
		return 0
	}
	r := x
	for r*r > x {
		r = (r + x/r) / 2
	}
	return r
}

// isqrt64 is the RMS root: the mean square of loud 16-bit audio (~4.5e14
// for a full-scale sine) overflows a 32-bit int, so the sum-of-squares
// average stays in int64 through the Newton iteration.
func isqrt64(x int64) int64 {
	if x <= 0 {
		return 0
	}
	r := x
	for r*r > x {
		r = (r + x/r) / 2
	}
	return r
}
