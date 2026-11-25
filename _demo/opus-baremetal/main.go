// The baremetal Opus acceptance probe: pkg/esp/opus against its vendored
// sources (csrc/, LLGoFiles), no ESP-IDF anywhere. Encode a synthetic tone
// into Opus packets, decode them back, compare aligned by the codec's own
// lookahead, exercise packet-loss concealment and the encoder ctls, and gate
// on the firmware's own PASS verdict - the same quality gates the ESP-IDF
// acceptance run (_demo/esp-idf-opus) pins for the same codec, so the two
// build paths stay honest against each other. The probe runs entirely on
// the main goroutine: opus_encode/opus_decode are deep C calls, so the
// build raises the fiber stacks (-goroutine-stack-size 16384; the main
// fiber is 4x that value) - see the package docs for the stack rules.
//
// Build and run (from this directory):
//
//	llgo build -o app.elf -target esp32s31 -goroutine-stack-size 16384 .
//	python3 build_flash.py app.elf flash.bin
//	esp-emu --chip esp32s31 --firmware flash.bin --timeout 120s
package main

import (
	"errors"

	"github.com/xgo-dev/llgo/pkg/esp/esperr"
	"github.com/xgo-dev/llgo/pkg/esp/opus"
)

// The quality gates of the ESP-IDF acceptance run, in tenths of a dB: a
// clean tone must survive nearly intact at the 48 kHz music point (measured
// ~39.3 dB there); the gate's job is catching real failures (misalignment
// or a broken decode reads SNR < 5 dB with max error at the sine amplitude).
const (
	gate48kTenths = 250 // 25.0 dB
	maxErrHalfAmp = 6000
)

func main() {
	println("[go] opusbm start")

	// State sizes are the arithmetic opus_*_get_size - a zero here would
	// mean the vendored codec did not land in the build.
	println("[go] opusbm states enc1 =", opus.EncoderStateBytes(1),
		"dec1 =", opus.DecoderStateBytes(1),
		"enc2 =", opus.EncoderStateBytes(2),
		"dec2 =", opus.DecoderStateBytes(2))

	// Validation the binding does before anything reaches the codec.
	if _, err := opus.NewEncoder(44100, 1, opus.AppAudio); !errors.Is(err, opus.ErrBadValue) {
		println("[go] opusbm FAIL rate-44100 err =", errText(err))
		return
	}
	if _, err := opus.NewDecoder(48000, 1); err != nil {
		println("[go] opusbm FAIL decoder-refused err =", errText(err))
		return
	}
	println("[go] opusbm validation ok rate44100-refused")

	ok := segTone("tone48k", 48000, 1, opus.AppAudio, 48000, 50, gate48kTenths, true)
	if !ok || !ctlChecks() {
		println("[go] opusbm FAIL")
		return
	}
	println("[go] opusbm PASS")
	println("[go] opusbm done")
}

// segTone encodes and decodes frames frames of a 440 Hz sine (amplitude
// 12000, S16) and prints the segment's ledger line: encoded bytes total,
// decoded samples per channel total, SNR in dB, max absolute error, and the
// error RMS. All comparisons stay in integer math.
func segTone(name string, rate, ch, app, bitrate, frames, gateTenths int, plc bool) bool {
	enc, err := opus.NewEncoder(rate, ch, app)
	if err != nil {
		println("[go] opusbm FAIL", name, "encoder err =", errText(err))
		return false
	}
	defer enc.Close()
	if err := enc.SetBitrate(bitrate); err != nil {
		println("[go] opusbm FAIL", name, "bitrate err =", errText(err))
		return false
	}
	if err := enc.SetComplexity(5); err != nil {
		println("[go] opusbm FAIL", name, "complexity err =", errText(err))
		return false
	}
	dec, err := opus.NewDecoder(rate, ch)
	if err != nil {
		println("[go] opusbm FAIL", name, "decoder err =", errText(err))
		return false
	}
	defer dec.Close()

	frame := rate * 20 / 1000 // 20 ms frames, samples per channel
	step := 440 * 32768 / rate
	delay := enc.Lookahead() // codec's algorithmic delay, samples per channel
	pcm := make([]int16, frame*ch)
	out := make([]int16, 2880*ch)
	// The last two encoded frames, for the lookahead-aligned compare: the
	// decoded window of frame i starts at i*frame-delay, which reaches one
	// frame back. (The IDF demo keeps the whole input in PSRAM; here the
	// SRAM heap is the budget, so the probe holds two frames.)
	ring := make([]int16, 2*frame*ch)
	phase := 0
	var encBytes int
	var decSamples int
	var sigSq, errSq int64
	var maxErr int64
	var cmpSamples int64
	for i := 0; i < frames; i++ {
		fillSine(pcm, ch, step, &phase)
		copy(ring[(i%2)*frame*ch:(i%2+1)*frame*ch], pcm)
		pkt, err := enc.Encode(pcm)
		if err != nil {
			println("[go] opusbm FAIL", name, "frame", i, "encode err =", errText(err))
			return false
		}
		encBytes += len(pkt)
		n, err := dec.Decode(pkt, out)
		if err != nil {
			println("[go] opusbm FAIL", name, "frame", i, "decode err =", errText(err))
			return false
		}
		decSamples += n
		// Decoded frame i carries the input starting at sample i*frame-delay
		// (the codec's pre-skip/lookahead shift). Frames whose aligned window
		// would reach before the stream start are skipped; the rest read
		// their reference samples from the two-frame ring (the window of
		// frame i lies within frames i-1..i).
		off := i*frame - delay
		if off < 0 {
			continue
		}
		for s := 0; s < n*ch; s++ {
			g := off*ch + s // global interleaved index of the reference sample
			ref := ring[g%(2*frame*ch)]
			d := int64(out[s]) - int64(ref)
			if d < 0 {
				d = -d
			}
			if d > maxErr {
				maxErr = d
			}
			errSq += d * d
			sigSq += int64(ref) * int64(ref)
		}
		cmpSamples += int64(n * ch)
	}

	snrTenths := 0
	if errSq > 0 {
		snrTenths = tenthsDB(sigSq / errSq)
	}
	println("[go] opusbm seg", name, "frames =", frames, "enc =", encBytes,
		"dec =", decSamples, "snr =", snrDb(snrTenths), "dB max =", int(maxErr),
		"rms =", int(isqrt64(errSq/cmpSamples)), "delay =", delay)

	if decSamples != frames*frame {
		println("[go] opusbm FAIL", name, "sample-count dec =", decSamples,
			"want =", frames*frame)
		return false
	}
	if snrTenths < gateTenths || maxErr > maxErrHalfAmp {
		println("[go] opusbm FAIL", name, "quality snr =",
			snrDb(snrTenths), "dB gate =", snrDb(gateTenths), "dB max =",
			int(maxErr), "gate =", maxErrHalfAmp)
		return false
	}

	if plc {
		// Packet-loss concealment: an empty frame makes the decoder invent
		// the continuation; the capacity must be a multiple of 2.5 ms (the
		// codec's own rule), so the probe asks for exactly one frame.
		n, err := dec.Decode(nil, out[:frame*ch])
		if err != nil || n != frame {
			println("[go] opusbm FAIL plc decode n =", n, "err =", errText(err))
			return false
		}
		println("[go] opusbm plc conceal =", n, "of", frame)
		pkt, err := enc.Encode(pcm) // the encoder did not move; reuse the frame
		if err != nil {
			println("[go] opusbm FAIL plc reencode err =", errText(err))
			return false
		}
		n, err = dec.Decode(pkt, out[:frame*ch])
		if err != nil || n != frame {
			println("[go] opusbm FAIL plc resync n =", n, "err =", errText(err))
			return false
		}
		println("[go] opusbm plc ok resync =", n)
	}
	println("[go] opusbm seg", name, "PASS")
	return true
}

// ctlChecks pins the ctl surface's contract on this build: an out-of-range
// complexity refuses with the codec's OPUS_BAD_ARG, a legal one passes, and
// the frame-duration table moves Encode's consumption and refuses non-Opus
// durations and short frames.
func ctlChecks() bool {
	enc, err := opus.NewEncoder(48000, 1, opus.AppAudio)
	if err != nil {
		println("[go] opusbm FAIL ctl encoder err =", errText(err))
		return false
	}
	defer enc.Close()
	if err := enc.SetComplexity(11); !errors.Is(err, opus.ErrBadArg) {
		println("[go] opusbm FAIL ctl complexity-11 err =", errText(err))
		return false
	}
	if err := enc.SetComplexity(8); err != nil {
		println("[go] opusbm FAIL ctl complexity-8 err =", errText(err))
		return false
	}
	if err := enc.SetFrameSizeMs(30); !errors.Is(err, opus.ErrBadValue) {
		println("[go] opusbm FAIL ctl frame-30ms err =", errText(err))
		return false
	}
	if err := enc.SetFrameSizeMs(40); err != nil {
		println("[go] opusbm FAIL ctl frame-40ms err =", errText(err))
		return false
	}
	pcm := make([]int16, 48000*40/1000) // one 40 ms frame at 48 kHz
	if _, err := enc.Encode(pcm[:100]); !errors.Is(err, opus.ErrBadValue) {
		println("[go] opusbm FAIL ctl short-frame err =", errText(err))
		return false
	}
	if _, err := enc.Encode(pcm); err != nil {
		println("[go] opusbm FAIL ctl frame-40ms-encode err =", errText(err))
		return false
	}
	println("[go] opusbm ctl ok bad-arg frame-table length-guard")
	return true
}

// fillSine writes one frame of a 440 Hz S16 sine (amplitude 12000) into
// buf, advancing phase across calls. phase is a Q15 circle (0..32767 = one
// period) advanced by step per sample.
func fillSine(buf []int16, ch, step int, phase *int) {
	for s := 0; s < len(buf); s += ch {
		*phase = (*phase + step) & (1<<15 - 1)
		v := sineAt(*phase)
		for c := 0; c < ch; c++ {
			buf[s+c] = v
		}
	}
}

// sineAt renders a Q15 phase (0..32767 = one period) as an S16 sample from
// a 128-entry quarter-wave table (linear interpolation between entries
// keeps the interpolation error at ~6 LSB, an order under the codec's own
// quantization).
func sineAt(ph int) int16 {
	ph &= 1<<15 - 1
	quad := ph >> 13 // 0..3
	off := ph & 8191 // within the quadrant
	v := 0
	if quad&1 == 0 { // rising quadrant: 0 -> peak
		i := off >> 6 // 0..127
		v = quarterWave[i] + (quarterWave[i+1]-quarterWave[i])*(off&63)/64
	} else { // falling quadrant: peak -> 0
		v = quarterWave[(8192-off)>>6]
	}
	if quad >= 2 {
		return -int16(v)
	}
	return int16(v)
}

// quarterWave: sin(pi/2 * n/128) * 12000, n = 0..128, rounded - a
// build-time constant array, so the probe needs no float math.
var quarterWave = [129]int{
	0, 147, 294, 442, 589, 736, 883, 1030, 1176, 1323, 1469, 1615, 1761,
	1906, 2052, 2196, 2341, 2485, 2629, 2773, 2916, 3058, 3201, 3342, 3483, 3624,
	3764, 3904, 4043, 4181, 4319, 4456, 4592, 4728, 4863, 4997, 5131, 5263, 5395,
	5526, 5657, 5786, 5915, 6042, 6169, 6295, 6420, 6544, 6667, 6789, 6910, 7030,
	7148, 7266, 7383, 7498, 7613, 7726, 7838, 7949, 8059, 8167, 8274, 8381, 8485,
	8589, 8691, 8792, 8891, 8990, 9087, 9182, 9276, 9369, 9460, 9550, 9638, 9725,
	9811, 9895, 9978, 10059, 10138, 10216, 10293, 10368, 10441, 10513, 10583, 10652, 10719,
	10784, 10848, 10910, 10971, 11029, 11087, 11142, 11196, 11248, 11299, 11347, 11394, 11440,
	11483, 11525, 11565, 11604, 11640, 11675, 11708, 11740, 11769, 11797, 11823, 11848, 11870,
	11891, 11910, 11927, 11942, 11956, 11967, 11977, 11986, 11992, 11996, 11999, 12000,
}

// tenthsDB estimates 10*log10(ratio) in tenths of a dB for a ratio >= 1,
// with an integer log2 (binary exponent + a polynomial on the mantissa;
// good to ~0.2 dB over the whole range).
func tenthsDB(ratio int64) int {
	if ratio < 1 {
		return 0
	}
	// Normalize to m = ratio * 2^-e with m in [2^16, 2^17), i.e. log2(ratio)
	// = e + 16 + log2(m/2^16).
	e := 0
	m := ratio
	for m >= 1<<17 {
		m >>= 1
		e++
	}
	for m < 1<<16 {
		m <<= 1
		e--
	}
	// log2(1+x) ≈ ln2^-1 * x * (1 - x/2 + x^2/3), x = f/2^16, in 16.16.
	const ln2inv = 96132 // 1.44269 << 16
	f := int(m) - 1<<16
	x := int64(f)
	log2m := ln2inv * x * (1<<16 - x/2 + x*x/(3*1<<16)) >> 16 >> 16 // 16.16
	// log2(ratio) in 16.16; tenths of a dB = log2 * 10*log10(2) = log2 * 3.0103.
	log2x := (int64(e)+16)*(1<<16) + log2m
	tenths := log2x * 301030 >> 16 / 10000
	if tenths < 0 {
		return 0
	}
	return int(tenths)
}

// snrDb renders tenths of a dB as "NN.N" without float math.
func snrDb(tenths int) string {
	s := ""
	neg := tenths < 0
	if neg {
		tenths = -tenths
		s = "-"
	}
	return s + itoa(tenths/10) + "." + itoa(tenths%10)
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}

// isqrt64 is the integer square root the family's audio probes use
// (Newton, per esp-idf-all's audio.go).
func isqrt64(x int64) int64 {
	if x <= 0 {
		return 0
	}
	r := x
	for {
		nr := (r + x/r) >> 1
		if nr >= r {
			break
		}
		r = nr
	}
	return r
}

func errText(err error) string {
	e, ok := err.(*esperr.Error)
	if !ok {
		return err.Error()
	}
	return e.Error() + ": " + opus.ErrText(e.Code)
}
