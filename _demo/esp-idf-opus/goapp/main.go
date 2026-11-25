package main

// Opus codec acceptance probe: encode → decode a synthetic tone through the
// pkg/esp/opus binding and compare segment by segment (SNR / max error /
// RMS, integer math only - no floating point on this soft-float target).
// The codec's C states ride the glue's handle table and the tinygogc malloc
// wrap; the ledger lines below make that visible.
//
// Every line starts with "[go] opus"; golden/emu-opus-probe.txt pins the
// exact ones and floats the values that carry measurement noise.
//
// The probe runs entirely on the main goroutine: opus_encode/decode are
// deep C calls (alloca scratch in the tens of KiB), so the build raises the
// fiber stacks (-goroutine-stack-size in host/artifact.conf; the main fiber
// is 4x that value) - see the package docs for the stack rules and the
// measured failure the first attempt hit.

import (
	"errors"
	_ "unsafe" // for the //go:linkname declaration below

	"github.com/xgo-dev/llgo/pkg/esp/opus"
)

// heapLedger is the C side's view of both IDF pools (demo_glue.c) - the
// memory ledger's stage lines, interleaved with the probe's own [go] lines.
//
//go:linkname heapLedger C.llgo_opus_heap
func heapLedger(tag *byte, n int32)

// fixStreams repoints clite's stdout/stderr/stdin handles at the ESP-IDF
// picolibc streams (host/llgo_syms.S): on baremetal clite opens its own
// streams with Fopen("/dev/stdout"), which fails under ESP-IDF (no such VFS
// device) and would leave every Go println writing to a NULL FILE*. Must run
// after the Go package inits and before the first print.
//
// TODO(runtime): a mechanism-level fix is landing in clite's stdio selection
// (under llgo.idf, reference the C streams directly instead of Fopen); once
// it ships, this call and the llgo_syms.S shim are dead weight - remove them
// with that change.
//
//go:linkname fixStreams C.llgoFixStreams
func fixStreams()

func markLedger(stage string) {
	b := []byte(stage)
	heapLedger(&b[0], int32(len(b)))
}

// The probe's quality gates, in tenths of a dB, one per segment: 25 dB for
// the music-mode 48 kHz segment (a clean tone must survive nearly intact),
// 10 dB for the speech-mode segment - SILK's perceptual noise shaping is
// tuned for speech formants, and a pure tone legitimately loses ~13 dB there
// (measured, this codec); the gate's job is catching real failures
// (misalignment or a broken decode reads SNR < 5 dB with max error at the
// sine amplitude), not certifying hi-fi. maxErrHalfAmp is the second guard:
// a decoded sample may differ from the input by at most half the amplitude.
const (
	gate48kTenths  = 250 // 25.0 dB
	gateVoipTenths = 100 // 10.0 dB
	maxErrHalfAmp  = 6000
)

func main() {
	fixStreams()
	println("[go] opus start")
	markLedger("boot")

	// Validation the binding does before anything reaches the codec.
	if _, err := opus.NewEncoder(44100, 1, opus.AppAudio); !errors.Is(err, opus.ErrBadValue) {
		println("[go] opus FAIL rate-44100 err =", errText(err))
		return
	}
	if _, err := opus.NewEncoder(48000, 3, opus.AppAudio); !errors.Is(err, opus.ErrBadValue) {
		println("[go] opus FAIL channels-3 err =", errText(err))
		return
	}
	if _, err := opus.NewDecoder(48000, 1); err != nil {
		// On this firmware the codec component IS linked - a refusal here
		// would mean the glue did not land in the build.
		println("[go] opus FAIL decoder-refused err =", errText(err))
		return
	}
	println("[go] opus validation ok rate44100+ch3-refused")

	// State sizes (opus_*_get_size, arithmetic - no allocations): the
	// memory ledger's codec lines.
	println("[go] opus states enc1 =", opus.EncoderStateBytes(1),
		"dec1 =", opus.DecoderStateBytes(1),
		"enc2 =", opus.EncoderStateBytes(2),
		"dec2 =", opus.DecoderStateBytes(2))

	// Segment 1: 48 kHz mono, music application, 48 kbps - the broad
	// quality point, with the packet-loss concealment check at the end.
	ok1 := segTone("tone48k", 48000, 1, opus.AppAudio, 48000, 50, gate48kTenths, true)
	markLedger("seg1-done")

	// Segment 2: 16 kHz mono, voip application, 16 kbps - the speech point
	// (the voip application applies its 6 kHz highpass, so its error budget
	// is the application's own choice).
	ok2 := segTone("voip16k", 16000, 1, opus.AppVoip, 16000, 50, gateVoipTenths, false)
	markLedger("seg2-done")

	// Encoder ctls: complexity out of range must refuse with the codec's
	// own OPUS_BAD_ARG; a legal one must pass; the frame-duration table
	// must move Encode's consumption and refuse non-Opus durations.
	if !ctlChecks() {
		return
	}
	markLedger("ctls-done")

	if !ok1 || !ok2 {
		println("[go] opus FAIL segment-quality")
		return
	}
	println("[go] opus PASS")
	println("[go] opus done")
}

// segTone encodes and decodes frames frames of a 440 Hz sine (amplitude
// 12000, S16) at the given rate/channels/application/bitrate, and prints
// the segment's ledger line: encoded bytes total, decoded samples per
// channel total, SNR in dB, max absolute error, and the error RMS. All
// comparisons stay in integer math.
func segTone(name string, rate, ch, app, bitrate, frames, gateTenths int, plc bool) bool {
	enc, err := opus.NewEncoder(rate, ch, app)
	if err != nil {
		println("[go] opus FAIL", name, "encoder err =", errText(err))
		return false
	}
	defer enc.Close()
	if err := enc.SetBitrate(bitrate); err != nil {
		println("[go] opus FAIL", name, "bitrate err =", errText(err))
		return false
	}
	if err := enc.SetComplexity(5); err != nil {
		println("[go] opus FAIL", name, "complexity err =", errText(err))
		return false
	}
	dec, err := opus.NewDecoder(rate, ch)
	if err != nil {
		println("[go] opus FAIL", name, "decoder err =", errText(err))
		return false
	}
	defer dec.Close()

	frame := rate * 20 / 1000 // 20 ms frames, samples per channel
	step := 440 * 32768 / rate
	delay := enc.Lookahead() // codec's algorithmic delay, samples per channel
	pcm := make([]int16, frame*ch)
	out := make([]int16, 2880*ch)            // room for the largest legal frame
	stream := make([]int16, frames*frame*ch) // the whole encode input, for the aligned compare
	phase := 0
	var encBytes int
	var decSamples int
	var sigSq, errSq int64
	var maxErr int64
	for i := 0; i < frames; i++ {
		fillSine(pcm, ch, step, &phase)
		copy(stream[i*frame*ch:(i+1)*frame*ch], pcm)
		pkt, err := enc.Encode(pcm)
		if err != nil {
			println("[go] opus FAIL", name, "frame", i, "encode err =", errText(err))
			return false
		}
		encBytes += len(pkt)
		n, err := dec.Decode(pkt, out)
		if err != nil {
			println("[go] opus FAIL", name, "frame", i, "decode err =", errText(err))
			return false
		}
		decSamples += n
		// Decoded frame i carries the input starting at sample i*frame-delay
		// (the codec's pre-skip/lookahead shift). Frames whose aligned window
		// would reach before the stream start are skipped.
		off := i*frame - delay
		if off < 0 {
			continue
		}
		for s := 0; s < n*ch; s++ {
			d := int64(out[s]) - int64(stream[off*ch+s])
			if d < 0 {
				d = -d
			}
			if d > maxErr {
				maxErr = d
			}
			errSq += d * d
			sigSq += int64(stream[off*ch+s]) * int64(stream[off*ch+s])
		}
	}

	snrTenths := 0
	if errSq > 0 {
		snrTenths = tenthsDB(sigSq / errSq)
	}
	println("[go] opus seg", name, "frames =", frames, "enc =", encBytes,
		"dec =", decSamples, "snr =", snrDb(snrTenths), "dB max =", int(maxErr),
		"rms =", int(isqrt64(errSq/int64((frames-(delay+frame-1)/frame)*frame*ch))),
		"delay =", delay)

	if decSamples != frames*frame {
		println("[go] opus FAIL", name, "sample-count dec =", decSamples,
			"want =", frames*frame)
		return false
	}
	if snrTenths < gateTenths || maxErr > maxErrHalfAmp {
		println("[go] opus FAIL", name, "quality snr =",
			snrDb(snrTenths), "dB gate =", snrDb(gateTenths), "dB max =",
			int(maxErr), "gate =", maxErrHalfAmp)
		return false
	}

	if plc {
		// Packet-loss concealment: an empty frame makes the decoder invent
		// the continuation. Concealment fills exactly the requested
		// capacity (which must be a multiple of 2.5 ms - the codec's own
		// rule), so the probe asks for exactly one frame. The samples are
		// synthetic - the check is that the decoder answers with a full
		// frame and then resyncs to real data cleanly.
		n, err := dec.Decode(nil, out[:frame*ch])
		if err != nil || n != frame {
			println("[go] opus FAIL plc decode n =", n, "err =", errText(err))
			return false
		}
		println("[go] opus plc conceal =", n, "of", frame)
		pkt, err := enc.Encode(pcm) // the encoder did not move; reuse the frame
		if err != nil {
			println("[go] opus FAIL plc reencode err =", errText(err))
			return false
		}
		n, err = dec.Decode(pkt, out[:frame*ch])
		if err != nil || n != frame {
			println("[go] opus FAIL plc resync n =", n, "err =", errText(err))
			return false
		}
		println("[go] opus plc ok resync =", n)
	}
	println("[go] opus seg", name, "PASS")
	return true
}

// ctlChecks pins the ctl surface's contract on this build: an out-of-range
// complexity refuses with the codec's OPUS_BAD_ARG, a legal one passes,
// and the frame-duration table moves Encode's consumption and refuses
// non-Opus durations and short frames.
func ctlChecks() bool {
	enc, err := opus.NewEncoder(48000, 1, opus.AppAudio)
	if err != nil {
		println("[go] opus FAIL ctl encoder err =", errText(err))
		return false
	}
	defer enc.Close()
	if err := enc.SetComplexity(11); !errors.Is(err, opus.ErrBadArg) {
		println("[go] opus FAIL ctl complexity-11 err =", errText(err))
		return false
	}
	if err := enc.SetComplexity(8); err != nil {
		println("[go] opus FAIL ctl complexity-8 err =", errText(err))
		return false
	}
	if err := enc.SetFrameSizeMs(30); !errors.Is(err, opus.ErrBadValue) {
		println("[go] opus FAIL ctl frame-30ms err =", errText(err))
		return false
	}
	if err := enc.SetFrameSizeMs(40); err != nil {
		println("[go] opus FAIL ctl frame-40ms err =", errText(err))
		return false
	}
	pcm := make([]int16, 48000*40/1000) // one 40 ms frame at 48 kHz
	if _, err := enc.Encode(pcm[:100]); !errors.Is(err, opus.ErrBadValue) {
		println("[go] opus FAIL ctl short-frame err =", errText(err))
		return false
	}
	if _, err := enc.Encode(pcm); err != nil {
		println("[go] opus FAIL ctl frame-40ms-encode err =", errText(err))
		return false
	}
	println("[go] opus ctl ok bad-arg frame-table length-guard")
	return true
}

// fillSine writes one frame of a 440 Hz S16 sine (amplitude 12000) into
// buf, advancing phase across calls. phase is a Q15 circle (0..32767 = one
// period) advanced by step per sample; both encoder and decoder see the
// identical samples, so the step's truncation (440 Hz lands at 440.0 Hz at
// 48 kHz, 439.9 Hz at 16 kHz) never shows up in the error metrics.
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
		v = qw(i) + (qw(i+1)-qw(i))*(off&63)/64
	} else { // falling quadrant: peak -> 0
		v = qw((8192 - off) >> 6)
	}
	if quad >= 2 {
		return -int16(v)
	}
	return int16(v)
}

// qw indexes the quarter-wave table: sin(pi/2 * n/128) * 12000, n = 0..128,
// rounded - a build-time constant array, so the probe needs no float math.
func qw(n int) int { return quarterWave[n] }

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
