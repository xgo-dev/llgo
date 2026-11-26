package main

// Voice loop: microphone → energy VAD → Opus (16 kHz mono) → decode →
// speaker. The user talks; after they stop (hangover), the utterance is
// Opus-encoded, decoded again and played back - a single-duplex echo
// that exercises the whole audio path through pkg/esp/opus.
//
// One goroutine, no concurrency: the phases are strictly sequential
// (listen → record → encode → decode-play), and each I2S phase opens its
// own master channel that owns the codec-side clock pins - the exact
// shape esp-idf-all's audio self-test proved on this board (the record
// channel is closed before the playback channel opens; the pins are
// never double-driven). No scheduling jitter can bite a phase that never
// overlaps another.
//
// Two run modes, auto-detected by the ES8311's I2C answer at 0x18:
//
//   - HW: the real codec (audited register sequence), I2S RX capture
//     into a PSRAM buffer, playback through the amp pin. The VAD stays
//     deaf for the first ~450 ms of every cycle (codec power-up pop) and
//     for ~1.5 s after every playback (the speaker's own tail must not
//     re-trigger it), while the capture keeps draining so the DMA ring
//     and the noise floor stay sane.
//   - EMU (the emulator has no codec): a deterministic synthetic scene
//     flows through the SAME VAD → encode → decode → compare path, so
//     the state machine, the VAD verdicts, the encode budget and the
//     quality gates are all exercised and pinned by the golden - only
//     the analog pins are virtual.
//
// Every line starts with "[go] vloop"; golden/emu-voice-loop.txt pins
// the exact ones and floats the values that carry measurement noise.
//
// Deep C calls: opus encode/decode need the raised fiber stacks (see
// host/artifact.conf); the VAD feed is a trivial 480-sample RMS.

import (

	"github.com/xgo-dev/llgo/pkg/esp/gpio"
	"github.com/xgo-dev/llgo/pkg/esp/i2c"
	"github.com/xgo-dev/llgo/pkg/esp/i2s"
	"github.com/xgo-dev/llgo/pkg/esp/opus"
)

// ---------- glue ----------

func markLedger(stage string) {
	b := []byte(stage)
	heapLedger(&b[0], int32(len(b)))
}

// ---------- tunables ----------

const (
	rate    = 16000
	frameMs = 30
	frameS  = rate * frameMs / 1000 // 480 mono samples per VAD frame

	// VAD tuning: open at 2x the noise floor (6 dB) for 3 consecutive
	// frames (~90 ms), close after 17 quiet frames (~510 ms). The floor
	// clamp window is in amplitude domain: below 60 a dead-silent room
	// would let dust trigger speech; above 4000 a loud room would never
	// open. (All adjustable here - the C engine just executes them.)
	vadOpenNum    = 2
	vadOpenDen    = 1
	vadOpenFrames = 3
	vadHangover   = 17
	vadFloorMin   = 60
	vadFloorMax   = 4000

	maxUtteranceS = 10                   // PSRAM budget: 16000*2B*10 = 320 KiB
	maxUtterance  = maxUtteranceS * rate // mono int16 samples
	encodeBitrate = 24000                // bps, voip speech
	opusFrameMs   = 20
	opusFrame     = rate * opusFrameMs / 1000 // 320 samples per packet
	maxPackets    = maxUtterance/opusFrame + 2
	encodeArenaK  = 128 // packet arena (KiB), worst-case VBR headroom
	prerollCap    = rate / frameS * frameS

	cycles = 2  // utterances the golden exercises
	muteM  = 30 // frames the VAD stays deaf at each HW cycle start (~900 ms):
	// codec power-up pop, plus the previous playback's room tail

	// Board wiring (as verified by esp-idf-all's audio self-test).
	i2cSda     = i2c.Pin(51)
	i2cScl     = i2c.Pin(50)
	es8311Addr = 0x18
	i2sMCLK    = i2s.Pin(52) // codec MCLK (256 x fs)
	i2sBCLK    = i2s.Pin(53)
	i2sWS      = i2s.Pin(55)
	i2sDOUT    = i2s.Pin(56)  // playback (codec DAC)
	i2sDIN     = i2s.Pin(54)  // record (codec ASDOUT)
	paPin      = gpio.Pin(57) // NS4150B amp enable, high = on

	// Quality gates for the encode→decode segment (voip mode on a
	// speech-like synthetic tone; same reasoning as the opus probe's
	// 16 kHz segment - SILK's perceptual shaping legitimately lands in
	// the low teens of dB, the gate catches real failures: misalignment
	// or a broken decode reads < 5 dB with max error at full amplitude).
	gateTenths    = 100 // 10.0 dB
	maxErrGate    = 6000
	realtimeMaxPc = 50 // HW only: encoding must fit half the realtime budget
)

// ---------- synthetic scene (EMU mode) ----------

// sineTable is one period, peak 8000 (as audited in esp-idf-all's
// audio.go; the phase convention there - Q16 step of freq*65536/rate,
// index (phase>>10)&63 - reproduces the frequency within 0.2%).
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

// The scene is 133 VAD frames (~3.99 s): 13 frames of lead silence
// (floor settling + pre-roll fill), 80 frames of a syllabic "utterance"
// (5-on/3-off frame envelope, 340 Hz + 680 Hz + seeded noise, RMS ≈ 2700
// - far above the 2x-floor open threshold, far below clipping), 40
// frames of tail silence. Deterministic: the golden's exact counts
// (samples, packets, pre-roll frames) all derive from this layout.
const (
	sceneLeadFrames  = 13
	sceneUttFrames   = 80
	sceneTailFrames  = 40
	sceneFramesTotal = sceneLeadFrames + sceneUttFrames + sceneTailFrames
	sceneSamples     = sceneFramesTotal * frameS
)

var scene = buildScene()

func buildScene() []int16 {
	s := make([]int16, sceneSamples)
	seed := int32(20261120)
	p1, p2 := 0, 0
	step1 := 340 * 65536 / rate // 340 Hz, Q16 phase per sample
	step2 := 680 * 65536 / rate
	for fi := 0; fi < sceneUttFrames; fi++ {
		if fi%8 >= 5 { // syllable envelope: 5 frames on, 3 off
			continue
		}
		for si := 0; si < frameS; si++ {
			seed = seed*1103515245 + 12345
			v := (int32(sineTable[(p1>>10)&63])*3600 +
				int32(sineTable[(p2>>10)&63])*1500 +
				(seed>>16)%1600) >> 13
			s[(sceneLeadFrames+fi)*frameS+si] = int16(v)
			p1 += step1
			p2 += step2
		}
	}
	return s
}

// ---------- board audio (HW mode) ----------

// es8311Init is the audited 16 kHz configuration from esp-idf-all's audio
// self-test (MCLK = 256 fs = 4.096 MHz, standard I2S, 16-bit), including
// the mic path: MIC1 input at max PGA gain, ADC digital volume 0 dB.
var es8311Init = [][2]uint8{
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
	{0x14, 0x1A}, // analog mic on (MIC1), max PGA gain
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

// tryCodecInit probes the ES8311's I2C interface (chip ID 0x83 at 0x18);
// its failure is what flips the demo into EMU mode (the emulator has no
// codec). On success it applies the audited configuration with readback
// of the identity-bearing registers.
func tryCodecInit() bool {
	bus, err := i2c.NewBus(i2cSda, i2cScl)
	if err != nil {
		return false
	}
	defer bus.Close()
	dev, err := bus.Device(es8311Addr, 100000)
	if err != nil {
		return false
	}
	var id [1]byte
	if err := dev.WriteRead([]byte{0xFD}, id[:]); err != nil || id[0] != 0x83 {
		return false // no ACK at 0x18: emulator, or a wiring problem
	}
	for _, w := range es8311Init {
		if err := dev.Write([]byte{w[0], w[1]}); err != nil {
			println("[go] vloop codec write reg =", w[0], "err =", errText(err))
			return false
		}
	}
	for _, reg := range []uint8{0x01, 0x08, 0x14, 0x17} {
		var got [1]byte
		if err := dev.WriteRead([]byte{reg}, got[:]); err != nil {
			println("[go] vloop codec readback reg =", reg, "err =", errText(err))
			return false
		}
		want := uint8(0)
		for _, w := range es8311Init { // later writes win, like on hardware
			if w[0] == reg {
				want = w[1]
			}
		}
		if got[0] != want {
			println("[go] vloop codec reg =", reg, "readback =", got[0], "want =", want)
			return false
		}
	}
	return true
}

// newRX / newTX allocate one master channel owning the codec-side clock
// pins for its phase (record needs DIN + clocks, playback DOUT + clocks
// - never both at once in this single-duplex loop).
func newRX() (*i2s.Channel, error) {
	ch, err := i2s.NewChannel(false)
	if err != nil {
		return nil, err
	}
	cfg := i2s.StdConfig{
		MCLK: i2sMCLK, BCLK: i2sBCLK, WS: i2sWS,
		DOUT: -1, DIN: i2sDIN,
		Bits: i2s.Bits16, Stereo: true, SampleRate: rate,
	}
	if err := ch.ConfigureStd(cfg); err != nil {
		ch.Close()
		return nil, err
	}
	return ch, nil
}

func newTX() (*i2s.Channel, error) {
	ch, err := i2s.NewChannel(true)
	if err != nil {
		return nil, err
	}
	cfg := i2s.StdConfig{
		MCLK: i2sMCLK, BCLK: i2sBCLK, WS: i2sWS,
		DOUT: i2sDOUT, DIN: -1,
		Bits: i2s.Bits16, Stereo: true, SampleRate: rate,
	}
	if err := ch.ConfigureStd(cfg); err != nil {
		ch.Close()
		return nil, err
	}
	return ch, nil
}

// ---------- state ----------

var (
	hwMode      bool
	utter       = make([]int16, maxUtterance) // PSRAM (320 KiB): the capture
	arena       = make([]byte, encodeArenaK*1024)
	packetSizes [maxPackets]int
	preroll     = make([]int16, prerollCap) // 1 s pre-roll ring
	prerollN    int                         // valid frames in the ring
	prerollP    int                         // next write frame
	scenePos    int
	rx          *i2s.Channel
)

func main() {
	fixStreams()
	println("[go] vloop start")

	hwMode = tryCodecInit()
	if hwMode {
		println("[go] vloop mode HW (ES8311 answering at 0x18, audited 16 kHz config)")
	} else {
		println("[go] vloop mode EMU (no ES8311 at 0x18; synthetic source, playback sink drops output)")
	}
	markLedger("boot")

	vadCfg(vadOpenNum, vadOpenDen, vadOpenFrames, vadHangover, vadFloorMin, vadFloorMax)
	println("[go] vloop vad cfg open =", vadOpenNum, "x frames =", vadOpenFrames,
		"hangover =", vadHangover, "floor = [", vadFloorMin, ",", vadFloorMax,
		"] frame =", frameS, "@", frameMs, "ms rate =", rate)

	enc, err := opus.NewEncoder(rate, 1, opus.AppVoip)
	if err != nil {
		println("[go] vloop FAIL encoder err =", errText(err))
		return
	}
	defer enc.Close()
	if err := enc.SetBitrate(encodeBitrate); err != nil {
		println("[go] vloop FAIL bitrate err =", errText(err))
		return
	}
	if err := enc.SetFrameSizeMs(opusFrameMs); err != nil {
		println("[go] vloop FAIL frame-ms err =", errText(err))
		return
	}
	dec, err := opus.NewDecoder(rate, 1)
	if err != nil {
		println("[go] vloop FAIL decoder err =", errText(err))
		return
	}
	defer dec.Close()
	lookahead := enc.Lookahead()
	println("[go] vloop codec ok bitrate =", encodeBitrate, "frame-ms =", opusFrameMs,
		"lookahead =", lookahead)

	markLedger("armed")

	var amp gpio.Pin
	if hwMode {
		amp = gpio.Pin(paPin)
		amp.SetDirection(gpio.Output, gpio.PullNone)
		amp.SetLevel(false)
	}

	for c := 1; c <= cycles; c++ {
		if !runCycle(c, enc, dec, lookahead, amp) {
			println("[go] vloop FAIL at cycle", c)
			return
		}
	}
	println("[go] vloop PASS")
	println("[go] vloop done")
}

// srcFrame fills frame[0:frameS] with the next mono frame from the active
// source (scene or mic), returning false when the source ran dry. HW reads
// block in the driver, so capture is realtime by construction.
func srcFrame(frame []int16) bool {
	if !hwMode {
		if scenePos+frameS > len(scene) {
			return false
		}
		copy(frame, scene[scenePos:scenePos+frameS])
		scenePos += frameS
		return true
	}
	var raw [frameS * 4]byte // stereo 16-bit: 1920 bytes per frame
	got := 0
	for got < len(raw) {
		n, err := rx.Read(raw[got:])
		if err != nil {
			println("[go] vloop mic read err =", errText(err))
			return false
		}
		got += n
	}
	for i := 0; i < frameS; i++ { // slot 0 = the codec's ADC
		b := i * 4
		frame[i] = int16(uint16(raw[b]) | uint16(raw[b+1])<<8)
	}
	return true
}

func pushPreroll(frame []int16) {
	copy(preroll[prerollP*frameS:(prerollP+1)*frameS], frame)
	prerollP = (prerollP + 1) % (prerollCap / frameS)
	if prerollN < prerollCap/frameS {
		prerollN++
	}
}

// takePreroll copies the ring in chronological order onto the utterance
// buffer and empties it, returning the sample count.
func takePreroll() int {
	n := prerollN
	start := (prerollP - n + prerollCap/frameS) % (prerollCap / frameS)
	for i := 0; i < n; i++ {
		f := (start + i) % (prerollCap / frameS)
		copy(utter[i*frameS:(i+1)*frameS], preroll[f*frameS:(f+1)*frameS])
	}
	prerollN = 0
	return n * frameS
}

// runCycle is one listen → record → encode → decode-play pass. The mute
// windows (HW only) keep the codec's power-up pop and the speaker's own
// echo tail from triggering the VAD, while the capture keeps draining so
// the DMA ring and the noise floor stay sane.
func runCycle(c int, enc *opus.Encoder, dec *opus.Decoder, lookahead int, amp gpio.Pin) bool {
	vadReset()
	scenePos = 0
	prerollN, prerollP = 0, 0
	mute := 0
	if hwMode {
		mute = muteM
		ch, err := newRX()
		if err != nil {
			println("[go] vloop FAIL rx setup err =", errText(err))
			return false
		}
		if err := ch.Enable(); err != nil {
			ch.Close()
			println("[go] vloop FAIL rx enable err =", errText(err))
			return false
		}
		rx = ch
	}
	println("[go] vloop cycle =", c, "listening")

	var frame [frameS]int16
	var rms, floor int32
	uttLen := 0
	truncated := false
	open := false
	for {
		if !srcFrame(frame[:]) {
			println("[go] vloop cycle =", c, "FAIL source exhausted without speech")
			return false
		}
		if mute > 0 { // drain only: pop and echo must not open the VAD
			mute--
			continue
		}
		switch vadFeed(&frame[0], frameS, &rms, &floor) {
		case 1:
			open = true
			uttLen = takePreroll()
			preFrames := uttLen / frameS
			copy(utter[uttLen:uttLen+frameS], frame[:])
			uttLen += frameS
			println("[go] vloop cycle =", c, "speech-open rms =", rms,
				"floor =", floor, "preroll =", preFrames)
		case 2:
			println("[go] vloop cycle =", c, "speech-close ms =", uttLen*frameMs/frameS,
				"samples =", uttLen, "quiet =", vadHangover)
			if hwMode { // release the clock pins before playback drives them
				rx.Disable()
				rx.Close()
				rx = nil
			}
			if !encodePlay(c, enc, dec, lookahead, uttLen, amp) {
				return false
			}
			return true // cycle complete; the next one mutes its own head
		default:
			if !open {
				pushPreroll(frame[:])
			} else if uttLen+frameS <= maxUtterance {
				copy(utter[uttLen:uttLen+frameS], frame[:])
				uttLen += frameS
			} else if !truncated {
				truncated = true
				println("[go] vloop cycle =", c, "utterance truncated at", maxUtteranceS, "s cap")
			}
		}
	}
}

// encodePlay runs the Opus roundtrip over the captured utterance and
// streams the decode into the sink (speaker on HW, stats + fidelity gate
// in EMU). The SNR compares the decode (aligned by the codec's lookahead)
// against the capture itself - on EMU that is the synthetic scene, on HW
// the mic bytes, so it always measures codec fidelity, never the room.
func encodePlay(c int, enc *opus.Encoder, dec *opus.Decoder, lookahead, uttLen int, amp gpio.Pin) bool {
	uttMs := uttLen * 1000 / rate

	// --- encode (the whole utterance; the budget line is the point) ---
	t0 := nowUs()
	packets, encBytes := 0, 0
	var pad [opusFrame]int16
	for off := 0; off < uttLen; off += opusFrame {
		n := uttLen - off
		if n > opusFrame {
			n = opusFrame
		}
		copy(pad[:], utter[off:off+n])
		for i := n; i < opusFrame; i++ {
			pad[i] = 0
		}
		pkt, err := enc.Encode(pad[:])
		if err != nil {
			println("[go] vloop cycle =", c, "FAIL encode at", off, "err =", errText(err))
			return false
		}
		if encBytes+len(pkt) > len(arena) || packets >= maxPackets {
			println("[go] vloop cycle =", c, "FAIL packet arena full at", packets)
			return false
		}
		packetSizes[packets] = len(pkt)
		copy(arena[encBytes:], pkt)
		encBytes += len(pkt)
		packets++
	}
	encMs := (nowUs() - t0) / 1000
	ratio := 0
	if uttMs > 0 {
		ratio = int(encMs) * 100 / uttMs
	}
	println("[go] vloop cycle =", c, "encode ms =", encMs, "packets =", packets,
		"bytes =", encBytes, "ratio =", ratio, "%")
	if hwMode && ratio >= realtimeMaxPc {
		println("[go] vloop cycle =", c, "FAIL encode budget ratio =", ratio,
			"% (want <", realtimeMaxPc, "%)")
		return false
	}

	// --- decode + play (streamed: one packet in, one chunk out) ---
	var out [opusFrame]int16
	playSamples := 0
	off := 0
	if hwMode {
		tx, err := newTX()
		if err != nil {
			println("[go] vloop cycle =", c, "FAIL tx setup err =", errText(err))
			return false
		}
		if err := tx.Enable(); err != nil {
			tx.Close()
			println("[go] vloop cycle =", c, "FAIL tx enable err =", errText(err))
			return false
		}
		amp.SetLevel(true)
		defer func() {
			amp.SetLevel(false)
			tx.Disable()
			tx.Close()
		}()

		t0 = nowUs()
		var stereo [opusFrame * 4]byte
		for p := 0; p < packets; p++ {
			plen := packetSizes[p]
			n, err := dec.Decode(arena[off:off+plen], out[:])
			if err != nil {
				println("[go] vloop cycle =", c, "FAIL decode at", p, "err =", errText(err))
				return false
			}
			for i := 0; i < n; i++ { // duplicate mono to both slots
				b := i * 4
				v := out[i]
				stereo[b] = byte(v)
				stereo[b+1] = byte(v >> 8)
				stereo[b+2] = byte(v)
				stereo[b+3] = byte(v >> 8)
			}
			if _, err := tx.Write(stereo[:n*4]); err != nil {
				println("[go] vloop cycle =", c, "FAIL play write at", p, "err =", errText(err))
				return false
			}
			playSamples += n
			off += plen
		}
		playMs := (nowUs() - t0) / 1000
		println("[go] vloop cycle =", c, "decode-play ms =", playMs, "packets =", packets,
			"samples =", playSamples, "(realtime playback, amp on)")
		return true
	}

	// EMU: sink = stats + the fidelity gate against the capture.
	t0 = nowUs()
	errSq, sigSq, maxErr, sinkSq := int64(0), int64(0), int64(0), int64(0)
	consumed := 0
	for p := 0; p < packets; p++ {
		plen := packetSizes[p]
		n, err := dec.Decode(arena[off:off+plen], out[:])
		if err != nil {
			println("[go] vloop cycle =", c, "FAIL decode at", p, "err =", errText(err))
			return false
		}
		for i := 0; i < n; i++ {
			v := int64(out[i])
			a := v
			if a < 0 {
				a = -a
			}
			sinkSq += a * a
			src := consumed + i - lookahead
			if src >= 0 && src < uttLen {
				ref := int64(utter[src])
				d := ref - v
				if d < 0 {
					d = -d
				}
				if d > maxErr {
					maxErr = d
				}
				sigSq += ref * ref
				errSq += d * d
			}
		}
		consumed += n
		playSamples += n
		off += plen
	}
	playMs := (nowUs() - t0) / 1000
	outRMS := isqrt64(sinkSq / int64(playSamples))
	println("[go] vloop cycle =", c, "decode-play ms =", playMs, "packets =", packets,
		"samples =", playSamples)
	snrTenths := 0
	if errSq > 0 {
		snrTenths = tenthsDB(sigSq / errSq)
	}
	println("[go] vloop cycle =", c, "snr =", snrDb(snrTenths), "dB max =", maxErr,
		"rms =", outRMS, "(gate", snrDb(gateTenths), "dB max", maxErrGate, ")")
	if snrTenths < gateTenths || maxErr > maxErrGate {
		println("[go] vloop cycle =", c, "FAIL quality snr =", snrDb(snrTenths),
			"dB max =", maxErr)
		return false
	}
	println("[go] vloop cycle =", c, "seg PASS (gate", snrDb(gateTenths), "dB)")
	println("[go] vloop cycle =", c, "played samples =", playSamples, "rmsout =", outRMS)
	return true
}

// ---------- helpers (integer math only; soft-float target) ----------

func tenthsDB(ratio int64) int {
	if ratio < 1 {
		return 0
	}
	// Normalize to m = ratio * 2^-e with m in [2^16, 2^17), i.e.
	// log2(ratio) = e + 16 + log2(m/2^16).
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
	log2m := ln2inv * x * (1<<16 - x/2 + x*x/(3*1<<16)) >> 16 >> 16
	log2x := (int64(e)+16)*(1<<16) + log2m
	tenths := log2x * 301030 >> 16 / 10000
	if tenths < 0 {
		return 0
	}
	return int(tenths)
}

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
