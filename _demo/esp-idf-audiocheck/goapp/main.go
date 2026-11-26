package main

// Audiocheck: the pre-flash verification firmware for the ESP32-S31 audio
// board (Function-CoreBoard-1, ES8311 codec on I2C 0x18, MCLK GPIO52, amp
// GPIO57, mic on board). One firmware, three serial phases, so a single
// flash session answers every opus/wakenet bring-up question before the
// board leaves the desk:
//
//   - Phase A (automatic): opus self-check. A synthesized 1 kHz sine + noise
//     segment (3 s, 16 kHz mono S16) round-trips the codec at three frame
//     sizes (60/40/20 ms); SNR, correlation and max error are computed on
//     the lookahead-aligned stream, and each round gates itself.
//   - Phase B (automatic): one voice-loop cycle (the esp-idf-voice-loop
//     shape): energy VAD listens, captures 2 s of speech after it opens
//     (plus the pre-roll ring), opus-encodes, decodes and plays the
//     utterance back - EMU compares against the capture, HW streams to the
//     speaker through the amp pin.
//   - Phase C (loops forever on HW): wakenet on the pkg/esp/wakenet engine.
//     The engine runs on the glue's own core-1 session task (PIE lives on
//     core 1; detect never crosses into Go). EMU feeds the esp-sr test
//     utterance through the test source (the esp-idf-wakenet shape); HW
//     captures the real microphone (SourceI2S) and plays a short amp beep
//     on every detection - the beep rides the same I2S controller as the
//     capture channel: the engine is parked (Stop), the beep TX channel
//     opens on the freed TX slot, plays, closes, the engine resumes
//     (Start), so the codec's clock pins are never double-driven.
//
// Run modes, auto-detected by the ES8311's I2C answer at 0x18 (the
// esp-idf-voice-loop probe): HW = real codec + mic + speaker; EMU (esp-emu
// has no codec) = the synthetic scene for phase B and the injected test
// utterance for phase C, through the same state machines the board runs.
//
// Every line starts with "[go] ac"; golden/emu-audiocheck.txt pins the
// exact ones and floats the values that carry measurement noise. Deep C
// calls (opus encode/decode) need the raised fiber stacks (see
// host/artifact.conf: -goroutine-stack-size 16384 + 128 KiB GC window, the
// measured voice-loop configuration). Phase timings come from the
// esp_timer glue; under esp-emu they are not meaningful - the HW numbers
// are the ones the realtime budget is judged on.

import (
	_ "unsafe" // for the //go:linkname declarations below

	"github.com/xgo-dev/llgo/pkg/esp/gpio"
	"github.com/xgo-dev/llgo/pkg/esp/i2c"
	"github.com/xgo-dev/llgo/pkg/esp/i2s"
	"github.com/xgo-dev/llgo/pkg/esp/opus"
	"github.com/xgo-dev/llgo/pkg/esp/wakenet"
)

// ---------- glue ----------

// fixStreams is link-compatibility only since the clite mechanism fix (see
// host/llgo_syms.S); kept so the print path matches the family demos.
//
//go:linkname fixStreams C.llgoFixStreams
func fixStreams()

// heapLedger prints the two IDF pools at one demo stage (demo_glue.c).
//
//go:linkname heapLedger C.llgo_ac_heap
func heapLedger(tag *byte, n int32)

//go:linkname nowUs C.llgo_ac_now_us
func nowUs() int64

//go:linkname nowMs C.llgo_ac_now_ms
func nowMs() uint32

// The VAD's tunables travel from Go (the consts below) into the C engine;
// feed returns 0 = nothing, 1 = utterance opened, 2 = closed (phase B
// opens on 1 and uses a fixed window after it, so 2 never arrives here).
//
//go:linkname vadCfg C.llgo_ac_vad_cfg
func vadCfg(openNum, openDen, openFrames, hangoverFrames, floorMin, floorMax int32)

//go:linkname vadReset C.llgo_ac_vad_reset
func vadReset()

//go:linkname vadFeed C.llgo_ac_vad_feed
func vadFeed(pcm *int16, n int32, rms *int32, floor *int32) int32

// injectTestPcm hands the demo's hilexin utterance to the wakenet glue's
// test source (demo_glue.c) and returns the sample count (EMU phase C).
//
//go:linkname injectTestPcm C.llgo_ac_test_pcm
func injectTestPcm() int32

// setWnMclk tells the synced wakenet glue which pin carries MCLK for its
// I2S RX channel (post-sync.sh patches the template; the default -1 keeps
// the upstream unused-pin behavior, which is what EMU runs use).
//
//go:linkname setWnMclk C.llgo_ac_mclk
func setWnMclk(pin int32)

// gosched hands the fiber scheduler around while a phase waits (the
// wakenet event poller goroutine cannot run without it).
//
//go:linkname gosched github.com/xgo-dev/llgo/runtime/internal/runtime.Gosched
func gosched()

// Diagnostic glue (diag_glue.c): one TX reference probe and one full RX
// probe per call, each tearing its channel down before returning. The out
// arrays are Go heap slices (stable addresses) filled by the C side.
//
//go:linkname diagTxEdges C.llgo_i2s_diag_tx_edges
func diagTxEdges(out *int32, mclk, bclk, ws, dout, rate int32) int32

//go:linkname diagRxProbe C.llgo_i2s_diag_rx_probe
func diagRxProbe(out *int32, port, mclk, bclk, ws, din, bits, stereo, rate int32) int32

// chunkRmsGet reads and clears the phase C capture-energy window (the
// demo glue's s_ac_chunk_rms / s_ac_chunks; the wakenet template's capture
// loop calls llgo_ac_chunk_rms with every chunk that feeds detect, see
// post-sync.sh). rms = the latest chunk's RMS, chunks = how many chunks
// the engine consumed since the last read (~31/s at 512 samples, 16 kHz).
// EMU never populates it (the test source skips the I2S branch).
//
//go:linkname chunkRmsGet C.llgo_ac_chunk_rms_get
func chunkRmsGet(rms *int32, chunks *int32)

func markLedger(stage string) {
	b := []byte(stage)
	heapLedger(&b[0], int32(len(b)))
}

// ---------- board wiring (as verified by esp-idf-all's audio self-test) ----------

const (
	rate    = 16000
	frameMs = 30
	frameS  = rate * frameMs / 1000 // 480 mono samples per VAD frame

	i2cSda     = i2c.Pin(51)
	i2cScl     = i2c.Pin(50)
	es8311Addr = 0x18
	i2sMCLK    = i2s.Pin(52) // codec MCLK (256 x fs)
	i2sBCLK    = i2s.Pin(53)
	i2sWS      = i2s.Pin(55)
	i2sDOUT    = i2s.Pin(56)  // playback (codec DAC)
	i2sDIN     = i2s.Pin(54)  // record (codec ASDOUT)
	paPin      = gpio.Pin(57) // NS4150B amp enable, high = on
)

// ---------- shared tunables ----------

const (
	encodeBitrate = 24000 // bps, voip speech
	opusFrameMs   = 20
	opusFrame     = rate * opusFrameMs / 1000 // 320 samples per packet
	// The pre-roll ring: ~1 s of frames kept while the VAD is deaf, so the
	// capture does not eat the first syllable when it opens.
	prerollFrames = rate / frameS          // 33 frames fit the ring
	prerollCap    = prerollFrames * frameS // 15840 samples
	// Phase B: fixed capture window after the VAD opens (the task's
	// "record 2 s"), on whole 30 ms frames.
	capSamples = 2 * rate
	capFrames  = (capSamples + frameS - 1) / frameS // 67 frames = 2010 ms

	// VAD tuning (identical to esp-idf-voice-loop): open at 2x the noise
	// floor for 3 consecutive frames (~90 ms). Phase B never closes - the
	// capture window is fixed - but the engine keeps its hysteresis shape.
	vadOpenNum    = 2
	vadOpenDen    = 1
	vadOpenFrames = 3
	vadHangover   = 17
	vadFloorMin   = 5  // HW noise floor sits at rms ~1; 60 was the emu-scene value
	vadFloorMax   = 4000

	// Quality gates (family values: esp-idf-opus's voip segment and
	// voice-loop's roundtrip segment both gate 10.0 dB / max 6000 - the
	// gate catches misalignment or a broken decode, which read < 5 dB
	// with max error at full amplitude; SILK's perceptual shaping
	// legitimately lands in the low teens of dB on a tone).
	gateTenths    = 100 // 10.0 dB
	maxErrGate    = 6000
	realtimeMaxPc = 80 // HW: measured opus 16k mono encode runs ~62% realtime on S31

	// Phase A: the self-check signal - 1 kHz sine + seeded noise, 3 s at
	// 16 kHz. 48000 samples divide every frame size the rounds use.
	aToneHz     = 1000
	aAmp        = 8000
	aNoiseAmp   = 1000
	aSeconds    = 3
	aSamples    = aSeconds * rate // 48000
	aMaxPackets = aSamples / opusFrame
	aGateTenths = 80 // 8.0 dB: under the measured voip-on-tone SNR (~13 dB)

	// Phase C (EMU): the acceptance window over the injected utterance,
	// esp-idf-wakenet's shape. HW has no window: it loops until power-off.
	cRunMs       = 12000
	cHeartbeatMs = 5000
	cPassHits    = 2   // the utterance replays every ~2 s: several hits fit
	cPassFrames  = 220 // ~7 s of chunks; the hit dances park ~0.3 s each

	// Phase C detection cue (HW): 240 ms of 990 Hz through the codec DAC
	// and the amp, stereo-interleaved S16 for the I2S TX channel. EMU never
	// plays it (the sink drops output; the dance parks and resumes the
	// engine to keep the code path identical).
	beepMs       = 240
	beepHz       = 990
	beepAmp      = 7000
	beepFrames   = rate * beepMs / 1000 // 3840 stereo frames
	beepBytes    = beepFrames * 4
	beepTailMs   = 120 // driver DMA ring drains after the last Write
	parkSettleMs = 150 // >= one chunk period: the parked capture disables
)

// aFrameMses is the phase A round list (largest frame first: the 60 ms
// round proves long-frame support, the 20 ms round matches phase B's shape).
var aFrameMses = [3]int{60, 40, 20}

// ---------- synthetic sources (EMU mode) ----------

// sineTable is one period, peak 8000 (the audited esp-idf-all table; the
// Q16 phase convention reproduces the frequency within 0.2%).
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

// The phase B scene is 128 VAD frames (~3.84 s): 13 frames of lead silence
// (floor settling + pre-roll fill), 85 frames of a continuous speech-like
// utterance (340 Hz + 680 Hz + seeded noise, RMS ~2700 - far above the 2x
// open threshold), 30 frames of tail silence. Deterministic: the golden's
// exact counts (preroll frames, capture samples, packets) all derive from
// this layout. The utterance is continuous - unlike voice-loop's syllabic
// one - because phase B's capture window is fixed: it records straight
// through the open frame and needs the source to stay "loud" for 70+ frames.
const (
	sceneLeadFrames = 13
	sceneUttFrames  = 85
	sceneTailFrames = 30
	sceneTotal      = sceneLeadFrames + sceneUttFrames + sceneTailFrames
	sceneSamples    = sceneTotal * frameS
)

var scene = buildScene()

func buildScene() []int16 {
	s := make([]int16, sceneSamples)
	seed := int32(20261120)
	p1, p2 := 0, 0
	step1 := 340 * 65536 / rate // 340 Hz, Q16 phase per sample
	step2 := 680 * 65536 / rate
	for fi := 0; fi < sceneUttFrames; fi++ {
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

var beepPcm = buildBeep()

func buildBeep() []byte {
	b := make([]byte, beepBytes)
	p := 0
	step := beepHz * 65536 / rate
	for i := 0; i < beepFrames; i++ {
		v := int32(sineTable[(p>>10)&63]) * beepAmp / 8000
		p += step
		b[i*4] = byte(v)
		b[i*4+1] = byte(v >> 8)
		b[i*4+2] = byte(v)
		b[i*4+3] = byte(v >> 8)
	}
	return b
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
	// 0x15 = ADC ramp is the one register the official esp_codec_dev
	// sequence adds that matters: without it the ADC data link stays silent
	// on hardware (proven by the diag probe: din edges but zero DMA). The
	// official analog-power block (0x0B/0x0C/0x10/0x11/0x44/0x45) silences
	// the path on this board - measured, not assumed - so it is not carried
	// over.
	//
	// Gain lives in 0x16, NOT 0x17: esp_codec_dev's es8311_set_mic_gain
	// writes 0x16 = 0..7, a 6 dB-per-step ADC digital gain (42 dB at 7;
	// both esp-bsp and esp_codec_dev spell the same enum), and never reads
	// it back - the register reads zero on silicon whatever you write
	// (write-only bits; this board measured the same, which is why the
	// readback probe cannot verify it - the phase C chunk-rms heartbeat
	// is the verdict). 0x17 is a pure digital ATTENUATOR: 0xC8 = 0 dB is
	// its ceiling (the esp-bsp driver writes exactly that as "ADC gain"),
	// 0.5 dB per step downward, and anything above 0xC8 is out of range -
	// measured: 0xF8 made speech rms DROP to 32.
	{0x14, 0x1A}, // analog mic on (MIC1), analog PGA max (the analog stage
	              // is already wide open; the digital gain below does the rest)
	{0x15, 0x40}, // ADC ramp: without it the ADC data link never runs
	{0x16, 0x04}, // ADC digital gain +24 dB per the official enum. Measured:
	              // phase C chunk-rms reads ~271 with this set (not the +24 dB
	              // arithmetic 78 -> 1250), so the codec digital gain delivers
	              // at most a fraction here; the glue's software x8 (+18 dB,
	              // llgo_ac_chunk_gain) is the level path that counts
	              // (speech ~270 -> ~2160 at the engine).
	{0x17, 0xC8}, // ADC digital volume 0 dB (the legal ceiling; see above)
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

// ---------- RX bring-up diagnostics (HW only) ----------

// codecBus / codecDev stay alive for the whole HW session: the RX
// diagnostics read (and, for the official-variant probe, write) the codec's
// registers after boot, and esp-idf-all's audio path needs the same
// long-lived device. The EMU probe closes them (device first, bus second -
// the driver refuses a bus with devices still attached).
var (
	codecBus *i2c.Bus
	codecDev *i2c.Device
)

// es8311Official carries the esp_codec_dev (v1.6.2, the driver the official
// GMF record pipelines run) open/start register writes the audited demo
// sequence never spelled: the analog reference enables (0x0B/0x0C), the
// L/R analog power bytes (0x10/0x11), the ADC gain scale at 0x24 (the demo
// writes 0x00), the ADC ramp/dmic-sense register (0x15 bit6) and the GPIO
// / GP control bytes (0x44/0x45). Volume (0x17) stays at the demo's 0 dB.
// The esp-bsp-era es8311.c agreed with the demo sequence; only this newer
// driver diverges - these eight writes are the codec-side delta.
var es8311Official = [][2]uint8{
	{0x0B, 0x00},
	{0x0C, 0x00},
	{0x10, 0x1F},
	{0x11, 0x7F},
	{0x16, 0x24},
	{0x15, 0x40},
	{0x44, 0x08},
	{0x45, 0x00},
}

// codecApplyInit re-runs the audited boot sequence against the live codec
// (the diagnostics mutate registers; the phases below expect the boot
// state, so the diagnostics re-apply it on the way out).
func codecApplyInit() {
	for _, w := range es8311Init {
		codecDev.Write([]byte{w[0], w[1]})
	}
}

var hexd = [16]byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'a', 'b', 'c', 'd', 'e', 'f'}

func hex8(v uint8) string {
	var buf [2]byte
	buf[0] = hexd[v>>4]
	buf[1] = hexd[v&15]
	return string(buf[:])
}

func hex32(v int32) string {
	var buf [8]byte
	u := uint32(v)
	for i := 7; i >= 0; i-- {
		buf[i] = hexd[u&15]
		u >>= 4
	}
	return string(buf[:])
}

// diagRegs dumps the codec's ADC/clock/power register bank in three lines
// (readback, not the written values - the chip's actual state).
func diagRegs(tag string) {
	var regs = [18]uint8{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09,
		0x0A, 0x0D, 0x0E, 0x12, 0x14, 0x15, 0x16, 0x17}
	var vals [18]uint8
	for i := 0; i < len(regs); i++ {
		var b [1]byte
		if codecDev != nil && codecDev.WriteRead([]byte{regs[i]}, b[:]) == nil {
			vals[i] = b[0]
		}
	}
	for base := 0; base < 18; base += 6 {
		println("[go] ac diag regs", tag,
			"r", hex8(regs[base]), "=", hex8(vals[base]),
			"r", hex8(regs[base+1]), "=", hex8(vals[base+1]),
			"r", hex8(regs[base+2]), "=", hex8(vals[base+2]),
			"r", hex8(regs[base+3]), "=", hex8(vals[base+3]),
			"r", hex8(regs[base+4]), "=", hex8(vals[base+4]),
			"r", hex8(regs[base+5]), "=", hex8(vals[base+5]))
	}
}

// diagProbe builds one RX channel on port, counts edges on the four codec
// pins, samples the controller's rx_done raw bit, attempts two short reads
// and tears the channel down - all from the C side. names/cfg: stock = the
// demo's exact configuration; official = the esp_codec_dev register set
// applied first; official-mono / official-32 = slot variants; i2s1 = the
// other controller.
func diagProbe(name string, port, bits int32, stereo bool, official bool) {
	if official && codecDev != nil {
		for _, w := range es8311Official {
			codecDev.Write([]byte{w[0], w[1]})
		}
	}
	st := int32(0)
	if stereo {
		st = 1
	}
	var out [14]int32
	diagRxProbe(&out[0], port, int32(i2sMCLK), int32(i2sBCLK), int32(i2sWS), int32(i2sDIN), bits, st, rate)
	println("[go] ac diag probe", name, "port =", port, "bits =", bits, "stereo =", stereo,
		"setup err =", out[0], out[1], out[2],
		"edges mclk =", out[3], "bclk =", out[4], "ws =", out[5], "din =", out[6],
		"rxint =", out[7],
		"reads =", out[8], "err", out[9], "/", out[10], "err", out[11],
		"rxconf =", hex32(out[12]), hex32(out[13]))
}

// diagRX splits the silent record path into observable facts. Runs between
// phase A and phase B (HW only - the EMU golden path never enters here);
// every probe builds and tears down its own channel, and the codec leaves
// with the audited boot state re-applied, so the phases below run exactly
// as before.
//
// Reading the log:
//   - edges 0 = dead line, >0 = signal present (heavily undersampled
//     count; the absolute value carries no meaning)
//   - rxint bit0 (rx_done) = the controller completed RX DMA buffers in
//     the sampled window; bit2 (rx_hung) = the DMA did not keep up; the
//     register sampled is I2S0's, so the field only means something on
//     the port-0 probes (the i2s1 probe shows I2S0 idle = 0)
//   - reads = n err code: n>0 with err 0 = data arrived; err 263 = the
//     driver's 1 s timeout (ESP_ERR_TIMEOUT 0x107); a second read of
//     -1 err -1 = not reached (the first read already failed)
//   - rxconf = the I2S0 rx_conf/rx_conf1 raw values (the i2s1 probe's
//     snapshot is likewise I2S0's)
//   - the tx-ref probe is the board-verified playback path: it sanity-
//     checks the counting method and the clock pins themselves
func diagRX() {
	println("[go] ac diag start (HW RX bring-up facts)")
	diagRegs("post-init")

	var tx [5]int32
	diagTxEdges(&tx[0], int32(i2sMCLK), int32(i2sBCLK), int32(i2sWS), int32(i2sDOUT), rate)
	println("[go] ac diag tx-ref err =", tx[0], "edges mclk =", tx[1], "bclk =", tx[2],
		"ws =", tx[3], "dout =", tx[4])

	diagProbe("stock", 0, 16, true, false)
	diagProbe("official", 0, 16, true, true)
	diagProbe("official-mono", 0, 16, false, true)
	diagProbe("official-32", 0, 32, true, true)
	diagProbe("official-i2s1", 1, 16, true, true)

	if codecDev != nil {
		codecApplyInit()
	}
	diagRegs("restored")
	println("[go] ac diag end (edges 0 = dead line; rxint bit0 = controller got DMA buffers; reads err 263 = timeout)")
}

// tryCodecInit probes the ES8311's I2C interface (chip ID 0x83 at 0x18);
// its failure is what flips the demo into EMU mode (the emulator has no
// codec). On success it applies the audited configuration with readback
// of the identity-bearing registers and keeps the bus/device globals.
func tryCodecInit() bool {
	bus, err := i2c.NewBus(i2cSda, i2cScl)
	if err != nil {
		return false
	}
	dev, err := bus.Device(es8311Addr, 100000)
	if err != nil {
		bus.Close()
		return false
	}
	alive := false
	defer func() {
		if !alive {
			dev.Close()
			bus.Close()
		}
	}()
	var id [1]byte
	if err := dev.WriteRead([]byte{0xFD}, id[:]); err != nil || id[0] != 0x83 {
		return false // no ACK at 0x18: emulator, or a wiring problem
	}
	for _, w := range es8311Init {
		if err := dev.Write([]byte{w[0], w[1]}); err != nil {
			println("[go] ac codec write reg =", w[0], "err =", errText(err))
			return false
		}
	}
	for _, reg := range []uint8{0x01, 0x08, 0x14, 0x17} {
		var got [1]byte
		if err := dev.WriteRead([]byte{reg}, got[:]); err != nil {
			println("[go] ac codec readback reg =", reg, "err =", errText(err))
			return false
		}
		want := uint8(0)
		for _, w := range es8311Init { // later writes win, like on hardware
			if w[0] == reg {
				want = w[1]
			}
		}
		if got[0] != want {
			println("[go] ac codec reg =", reg, "readback =", got[0], "want =", want)
			return false
		}
	}
	alive = true
	codecBus, codecDev = bus, dev
	return true
}

// newRX / newTX allocate one master channel owning the codec-side clock
// pins for its phase (record needs DIN + clocks, playback DOUT + clocks -
// never both at once in this single-duplex loop).
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
	hwMode   bool
	amp      gpio.Pin
	utter    = make([]int16, prerollCap+capFrames*frameS) // capture (heap, PSRAM-backed)
	arena    = make([]byte, encodeArenaK*1024)
	pktSizes = make([]int, aMaxPackets+2) // the largest consumer is phase A
	preroll  = make([]int16, prerollCap)  // 1 s pre-roll ring
	prerollN int                          // valid frames in the ring
	prerollP int                          // next write frame
	scenePos int
	rx       *i2s.Channel
)

// encodeArenaK is the packet arena in KiB: 3 s of VBR packets at 24 kbps
// is ~9 KB; 128 KiB is the voice-loop headroom shape (PSRAM-backed heap).
const encodeArenaK = 128

func main() {
	fixStreams()
	println("[go] ac start")

	hwMode = tryCodecInit()
	if hwMode {
		println("[go] ac mode HW (ES8311 answering at 0x18; mic + speaker + amp live)")
	} else {
		println("[go] ac mode EMU (no ES8311 at 0x18; synthetic sources, playback sink drops output)")
	}
	markLedger("boot")

	if hwMode {
		amp = paPin
		amp.SetDirection(gpio.Output, gpio.PullNone)
		amp.SetLevel(false)
	}

	if !phaseA() {
		println("[go] ac FAIL at A")
		return
	}
	if hwMode {
		diagRX() // HW only: the record-path bring-up facts, before phase B
	}
	if !phaseB() {
		println("[go] ac FAIL at B")
		return
	}
	markLedger("ab-done")
	if hwMode {
		// Phase C loops forever on the board: A+B already printed their
		// verdicts, and the console now carries heartbeats and HITs until
		// power-off (there is no "[go] ac done" line on HW).
		println("[go] ac A+B PASS (HW); phase C loops until power-off")
	}
	if !phaseC(hwMode) {
		return // its FAIL line already printed
	}
	markLedger("c-done")
	println("[go] ac PASS")
	println("[go] ac done")
}

// ---------- phase A: opus self-check ----------

// phaseA round-trips a 1 kHz sine + noise segment through the codec at
// three frame sizes and gates each round on the aligned compare (SNR,
// correlation, max error). The timings come from the esp_timer glue: under
// esp-emu they are NOT meaningful - the HW numbers are the realtime-budget
// evidence (that is why they are printed per round, not gated on).
func phaseA() bool {
	println("[go] ac A start tone =", aToneHz, "Hz + noise samples =", aSamples,
		"rate =", rate, "bitrate =", encodeBitrate,
		"(timings via esp_timer; emu values not meaningful, HW carries them)")

	// The test signal: one continuous segment, identical for every round
	// (the rounds differ only in frame size).
	src := make([]int16, aSamples)
	seed := int32(20261121)
	p := 0
	step := aToneHz * 65536 / rate
	for i := 0; i < aSamples; i++ {
		seed = seed*1103515245 + 12345
		v := (int32(sineTable[(p>>10)&63])*aAmp + (seed>>16)%(2*aNoiseAmp) - aNoiseAmp) >> 13
		src[i] = int16(v)
		p += step
	}

	for _, fms := range aFrameMses {
		if !aRound(fms, src) {
			return false
		}
	}
	println("[go] ac A PASS")
	return true
}

// aRound is one encode-decode round at one frame size. The compare aligns
// by the codec's lookahead (decoded frame i carries the input starting at
// sample i*frame-delay); correlation is the Pearson r of the aligned
// pairs, printed in hundredths, all integer math.
func aRound(fms int, src []int16) bool {
	fs := rate * fms / 1000
	packets := aSamples / fs
	enc, err := opus.NewEncoder(rate, 1, opus.AppVoip)
	if err != nil {
		println("[go] ac A FAIL encoder frame-ms =", fms, "err =", errText(err))
		return false
	}
	defer enc.Close()
	if err := enc.SetBitrate(encodeBitrate); err != nil {
		println("[go] ac A FAIL bitrate err =", errText(err))
		return false
	}
	if err := enc.SetFrameSizeMs(fms); err != nil {
		println("[go] ac A FAIL frame-ms err =", errText(err))
		return false
	}
	dec, err := opus.NewDecoder(rate, 1)
	if err != nil {
		println("[go] ac A FAIL decoder err =", errText(err))
		return false
	}
	defer dec.Close()
	delay := enc.Lookahead()

	var frame [960]int16 // holds the largest legal frame (60 ms @ 16 kHz)
	t0 := nowUs()
	encBytes := 0
	for i := 0; i < packets; i++ {
		copy(frame[:fs], src[i*fs:(i+1)*fs])
		pkt, err := enc.Encode(frame[:fs])
		if err != nil {
			println("[go] ac A FAIL encode frame-ms =", fms, "at", i, "err =", errText(err))
			return false
		}
		if encBytes+len(pkt) > len(arena) || i >= len(pktSizes) {
			println("[go] ac A FAIL arena full frame-ms =", fms)
			return false
		}
		copy(arena[encBytes:], pkt)
		pktSizes[i] = len(pkt)
		encBytes += len(pkt)
	}
	encMs := (nowUs() - t0) / 1000

	t0 = nowUs()
	var sigSq, decSq, errSq, num int64
	var maxErr int64
	decSamples, consumed := 0, 0
	off := 0
	var out [960]int16
	for i := 0; i < packets; i++ {
		plen := pktSizes[i]
		n, err := dec.Decode(arena[off:off+plen], out[:])
		if err != nil {
			println("[go] ac A FAIL decode frame-ms =", fms, "at", i, "err =", errText(err))
			return false
		}
		for j := 0; j < n; j++ {
			v := int64(out[j])
			srcIdx := consumed + j - delay
			if srcIdx >= 0 && srcIdx < aSamples {
				ref := int64(src[srcIdx])
				d := ref - v
				if d < 0 {
					d = -d
				}
				if d > maxErr {
					maxErr = d
				}
				errSq += d * d
				sigSq += ref * ref
				decSq += v * v
				num += ref * v
			}
		}
		consumed += n
		decSamples += n
		off += plen
	}
	decMs := (nowUs() - t0) / 1000

	snrTenths := 0
	if errSq > 0 {
		snrTenths = tenthsDB(sigSq / errSq)
	}
	println("[go] ac A frame-ms =", fms, "packets =", packets, "bytes =", encBytes,
		"enc-ms =", encMs, "dec-ms =", decMs, "snr =", snrDb(snrTenths), "dB",
		"corr =", corrText(num, sigSq, decSq), "max =", maxErr)
	if decSamples != aSamples {
		println("[go] ac A FAIL sample-count frame-ms =", fms, "dec =", decSamples,
			"want =", aSamples)
		return false
	}
	if snrTenths < aGateTenths || maxErr > maxErrGate {
		println("[go] ac A FAIL quality frame-ms =", fms, "snr =", snrDb(snrTenths),
			"dB max =", maxErr, "(gate", snrDb(aGateTenths), "dB max", maxErrGate, ")")
		return false
	}
	println("[go] ac A frame-ms =", fms, "PASS")
	return true
}

// ---------- phase B: voice-loop one cycle ----------

// phaseB is the esp-idf-voice-loop cycle with a fixed capture window: the
// VAD opens on speech (3 frames at 2x floor), the pre-roll ring is folded
// in, exactly capFrames more frames are captured (2 s), and the utterance
// is opus-encoded, decoded and played back. EMU compares the decode
// against the capture (fidelity gate); HW streams to the speaker.
func phaseB() bool {
	vadCfg(vadOpenNum, vadOpenDen, vadOpenFrames, vadHangover, vadFloorMin, vadFloorMax)
	println("[go] ac B start (voice-loop one cycle, capture", capSamples/rate,
		"s + preroll)")
	println("[go] ac B vad cfg open =", vadOpenNum, "x frames =", vadOpenFrames,
		"floor = [", vadFloorMin, ",", vadFloorMax, "] frame =", frameS, "@",
		frameMs, "ms rate =", rate)

	enc, err := opus.NewEncoder(rate, 1, opus.AppVoip)
	if err != nil {
		println("[go] ac B FAIL encoder err =", errText(err))
		return false
	}
	defer enc.Close()
	if err := enc.SetBitrate(encodeBitrate); err != nil {
		println("[go] ac B FAIL bitrate err =", errText(err))
		return false
	}
	if err := enc.SetFrameSizeMs(opusFrameMs); err != nil {
		println("[go] ac B FAIL frame-ms err =", errText(err))
		return false
	}
	dec, err := opus.NewDecoder(rate, 1)
	if err != nil {
		println("[go] ac B FAIL decoder err =", errText(err))
		return false
	}
	defer dec.Close()
	lookahead := enc.Lookahead()
	println("[go] ac B codec ok bitrate =", encodeBitrate, "frame-ms =", opusFrameMs,
		"lookahead =", lookahead)

	if hwMode {
		ch, err := newRX()
		if err != nil {
			println("[go] ac B FAIL rx setup err =", errText(err))
			return false
		}
		if err := ch.Enable(); err != nil {
			ch.Close()
			println("[go] ac B FAIL rx enable err =", errText(err))
			return false
		}
		rx = ch
	}

	println("[go] ac B listening")
	vadReset()
	scenePos = 0
	prerollN, prerollP = 0, 0

	// Mute window (HW only): the codec's power-up pop must not open the
	// VAD; the capture keeps draining so the DMA ring stays sane.
	mute := 0
	if hwMode {
		mute = 30
	}

	var frame [frameS]int16
	var rms, floor int32
	uttLen := 0
	preSamples := 0
	capCount := 0
	opened := false
	hwDbg := 0
	for {
		if !srcFrame(frame[:]) {
			println("[go] ac B FAIL source exhausted without speech")
			return false
		}
		if mute > 0 {
			mute--
			continue
		}
		if !opened {
			if vadFeed(&frame[0], frameS, &rms, &floor) == 1 {
				opened = true
				uttLen = takePreroll()
				preSamples = uttLen
				println("[go] ac B speech-open rms =", rms, "floor =", floor,
					"preroll =", uttLen/frameS)
			} else {
				if hwDbg++; hwDbg >= 33 { // ~1 s: expose the live rms/floor
					hwDbg = 0
					println("[go] ac B rms =", rms, "floor =", floor)
				}
				pushPreroll(frame[:])
				continue
			}
		}
		// Opened: fixed-window capture - the VAD verdict is ignored, the
		// window runs to its end no matter what the room does. The frame
		// that opened the utterance is the window's first frame.
		copy(utter[uttLen:uttLen+frameS], frame[:])
		uttLen += frameS
		capCount++
		if capCount >= capFrames {
			break
		}
	}
	if rx != nil { // release the clock pins before playback drives them
		rx.Disable()
		rx.Close()
		rx = nil
	}
	println("[go] ac B speech-cap ms =", capCount*frameMs, "samples =", uttLen,
		"preroll =", preSamples/frameS)
	return bPlay(enc, dec, lookahead, uttLen)
}

// ---------- phase B playback / compare ----------

// bPlay encodes the captured utterance, then decodes and either streams it
// to the speaker (HW: realtime playback, amp on) or compares it against
// the capture (EMU: the fidelity gate, aligned by the lookahead).
func bPlay(enc *opus.Encoder, dec *opus.Decoder, lookahead, uttLen int) bool {
	uttMs := uttLen * 1000 / rate

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
			println("[go] ac B FAIL encode at", off, "err =", errText(err))
			return false
		}
		if encBytes+len(pkt) > len(arena) || packets >= len(pktSizes) {
			println("[go] ac B FAIL packet arena full at", packets)
			return false
		}
		pktSizes[packets] = len(pkt)
		copy(arena[encBytes:], pkt)
		encBytes += len(pkt)
		packets++
	}
	encMs := (nowUs() - t0) / 1000
	ratio := 0
	if uttMs > 0 {
		ratio = int(encMs) * 100 / uttMs
	}
	println("[go] ac B encode ms =", encMs, "packets =", packets,
		"bytes =", encBytes, "ratio =", ratio, "%")
	if hwMode && ratio >= realtimeMaxPc {
		println("[go] ac B FAIL encode budget ratio =", ratio,
			"% (want <", realtimeMaxPc, "%)")
		return false
	}

	var out [opusFrame]int16
	playSamples := 0
	off := 0
	if hwMode {
		tx, err := newTX()
		if err != nil {
			println("[go] ac B FAIL tx setup err =", errText(err))
			return false
		}
		if err := tx.Enable(); err != nil {
			tx.Close()
			println("[go] ac B FAIL tx enable err =", errText(err))
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
			plen := pktSizes[p]
			n, err := dec.Decode(arena[off:off+plen], out[:])
			if err != nil {
				println("[go] ac B FAIL decode at", p, "err =", errText(err))
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
				println("[go] ac B FAIL play write at", p, "err =", errText(err))
				return false
			}
			playSamples += n
			off += plen
		}
		playMs := (nowUs() - t0) / 1000
		println("[go] ac B decode-play ms =", playMs, "packets =", packets,
			"samples =", playSamples, "(realtime playback, amp on)")
		return true
	}

	// EMU: sink = stats + the fidelity gate against the capture.
	t0 = nowUs()
	errSq, sigSq, maxErr := int64(0), int64(0), int64(0)
	for p := 0; p < packets; p++ {
		plen := pktSizes[p]
		n, err := dec.Decode(arena[off:off+plen], out[:])
		if err != nil {
			println("[go] ac B FAIL decode at", p, "err =", errText(err))
			return false
		}
		for i := 0; i < n; i++ {
			v := int64(out[i])
			src := playSamples + i - lookahead
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
		playSamples += n
		off += plen
	}
	playMs := (nowUs() - t0) / 1000
	println("[go] ac B decode-play ms =", playMs, "packets =", packets,
		"samples =", playSamples)
	snrTenths := 0
	if errSq > 0 {
		snrTenths = tenthsDB(sigSq / errSq)
	}
	println("[go] ac B snr =", snrDb(snrTenths), "dB max =", maxErr,
		"(gate", snrDb(gateTenths), "dB max", maxErrGate, ")")
	if snrTenths < gateTenths || maxErr > maxErrGate {
		println("[go] ac B FAIL quality snr =", snrDb(snrTenths), "dB max =", maxErr)
		return false
	}
	println("[go] ac B seg PASS (gate", snrDb(gateTenths), "dB)")
	println("[go] ac B PASS")
	return true
}

// ---------- phase C: wakenet ----------

// phaseC brings the wakenet engine up and listens. EMU: the injected
// hilexin utterance replays through the test source inside a bounded
// acceptance window, then the phase gates and closes. HW: the real mic
// (SourceI2S, the glue's RX channel - MCLK wired through the post-sync
// hook) and an infinite loop with 5 s heartbeats; every detection plays
// the amp beep via the park-beep-resume dance.
func phaseC(hw bool) bool {
	mode := "TEST"
	if hw {
		mode = "MIC"
	}
	println("[go] ac C start mode =", mode)
	if hw {
		setWnMclk(52) // the ES8311 cannot produce mic data without MCLK
	} else {
		n := injectTestPcm()
		println("[go] ac C test pcm samples =", n)
	}
	cfg := wakenet.ModelConfig{}
	if hw {
		cfg = wakenet.ModelConfig{
			Source: wakenet.SourceI2S,
			BCLK:   wakenet.Pin(53),
			WS:     wakenet.Pin(55),
			DIN:    wakenet.Pin(54),
			// Threshold comparison: wn9's per-model default is 0.60-0.61
			// (model info string); 0.50 is the experiment arm (valid range
			// 0.4..0.9999, the glue prints the engine's readback as
			// "[c] wn threshold"). EMU never sets it (zero value = the
			// model default, so the golden path is untouched).
			Threshold: 0.5,
		}
	}
	if err := wakenet.Init(cfg); err != nil {
		println("[go] ac C FAIL init err =", errText(err))
		return false
	}
	println("[go] ac C init ok model =", wakenet.ModelName(),
		"chunk =", wakenet.ChunkSamples(), "rate =", wakenet.SampleRate())
	markLedger("c-armed")
	if err := wakenet.Start(); err != nil {
		println("[go] ac C FAIL start err =", errText(err))
		return false
	}
	println("[go] ac C start ok")

	events := wakenet.Events()
	hits := 0
	framesAcc := int32(0) // Stats reset per Start (every hit dance), so the
	// total is accumulated across segments here
	t0 := int32(nowMs())
	lastBeat := t0
	lastHeap := t0
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				println("[go] ac C FAIL events closed")
				return false
			}
			switch ev.Kind {
			case wakenet.EvDetected:
				hits++
				println("[go] ac C HIT n =", hits, "ts =", ev.Timestamp)
				if f, _, _ := wakenet.Stats(); f > 0 {
					framesAcc += f // close the segment this hit ends
				}
				hitDance(hw)
			case wakenet.EvError:
				println("[go] ac C error code =", ev.Code, "ts =", ev.Timestamp)
			case wakenet.EvOverflow:
				println("[go] ac C overflow =", ev.Code)
			}
		default:
		}
		now := int32(nowMs())
		if now-lastBeat >= cHeartbeatMs {
			lastBeat = now
			f, _, _ := wakenet.Stats()
			if hw {
				// The engine-input energy window, BEFORE the glue's +18 dB
				// software gain (llgo_ac_chunk_gain, post-sync.sh): the raw
				// record path. Speech reads ~270 here (-42 dBFS, 15 dB under
				// the esp-sr reference that detects cleanly); the detect
				// input the heartbeat frames reflect is the x8 raised one.
				var crms, cch int32
				chunkRmsGet(&crms, &cch)
				println("[go] ac C listening... t =", (now-t0)/1000, "s frames =", framesAcc+f,
					"hits =", hits, "chunks =", cch, "chunk-rms =", crms)
			} else {
				println("[go] ac C listening... t =", (now-t0)/1000, "s frames =", framesAcc+f,
					"hits =", hits)
			}
		}
		if now-lastHeap >= 10000 {
			lastHeap = now
			markLedger("run")
		}
		if !hw && now-t0 >= cRunMs {
			break
		}
		gosched()
	}

	if err := wakenet.Stop(); err != nil {
		println("[go] ac C FAIL stop err =", errText(err))
		return false
	}
	f, avg, mx := wakenet.Stats()
	framesAcc += f
	println("[go] ac C final hits =", hits, "frames =", framesAcc, "avg_us =", avg,
		"max_us =", mx)
	if hits < cPassHits || framesAcc < cPassFrames {
		println("[go] ac C FAIL gate hits =", hits, "(want >=", cPassHits,
			") frames =", framesAcc, "(want >=", cPassFrames, ")")
		return false
	}
	if err := wakenet.Close(); err != nil {
		println("[go] ac C FAIL close err =", errText(err))
		return false
	}
	println("[go] ac C closed")
	return true
}

// hitDance is the detection cue, plus the engine park/resume the beep
// needs: the capture channel (the glue's own, on the codec's clock pins)
// is disabled by the park, the beep TX channel opens on the freed TX slot
// of the SAME I2S controller (the RX channel object still exists, so
// I2S_NUM_AUTO lands there), plays, closes - leaving the pins routed to
// that controller - and Start re-enables the capture. EMU skips the beep
// (no codec, no TX channel) but keeps the dance, so the park/resume path
// the HW depends on is exercised and pinned by the golden either way.
func hitDance(hw bool) {
	if err := wakenet.Stop(); err != nil {
		println("[go] ac C FAIL park err =", errText(err))
		return
	}
	pauseMs(parkSettleMs) // >= one chunk period: the parked loop disables RX
	if hw {
		println("[go] ac C beep on")
		beepPlay()
	} else {
		println("[go] ac C beep skip (emu)")
		pauseMs(parkSettleMs)
	}
	if err := wakenet.Start(); err != nil {
		println("[go] ac C FAIL resume err =", errText(err))
	}
}

// beepPlay plays the detection cue on HW: one TX channel on the freed slot
// of the capture's controller, MCLK/BCLK/WS re-pointed at the same pins
// (identical clock numbers: std mode is always 2 slots, so the mono
// capture and the stereo playback share the 4.096 MHz MCLK / 512 kHz
// BCLK), amp on, the whole cue written (Write paces realtime), tail
// drained, amp off. The channel is closed after; the pins stay routed to
// this controller, which is where the resumed capture channel lives.
func beepPlay() {
	ch, err := newTX()
	if err != nil {
		println("[go] ac C FAIL beep tx err =", errText(err))
		return
	}
	if err := ch.Enable(); err != nil {
		ch.Close()
		println("[go] ac C FAIL beep enable err =", errText(err))
		return
	}
	amp.SetLevel(true)
	// The cue is best-effort: the detection is what the gate is about. The
	// TX channel shares the controller with the parked capture channel, and
	// on hardware the write can time out (263) when the TX side does not
	// pick up the clock - the HIT line and the resumed capture stay valid
	// either way.
	if _, err := ch.Write(beepPcm); err != nil {
		println("[go] ac C beep write err =", errText(err), "(best-effort, continuing)")
	}
	pauseMs(beepTailMs)
	amp.SetLevel(false)
	ch.Disable()
	ch.Close()
}

// ---------- helpers ----------

// srcFrame fills frame[0:frameS] with the next mono frame from the active
// source (scene or mic). HW reads block in the driver, so capture is
// realtime by construction.
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
			println("[go] ac B mic read err =", errText(err))
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
	prerollP = (prerollP + 1) % prerollFrames
	if prerollN < prerollFrames {
		prerollN++
	}
}

// takePreroll copies the ring in chronological order onto the utterance
// buffer and empties it, returning the sample count.
func takePreroll() int {
	n := prerollN
	start := (prerollP - n + prerollFrames) % prerollFrames
	for i := 0; i < n; i++ {
		f := (start + i) % prerollFrames
		copy(utter[i*frameS:(i+1)*frameS], preroll[f*frameS:(f+1)*frameS])
	}
	prerollN = 0
	return n * frameS
}

// pauseMs busy-waits with the scheduler handed around (the event poller
// goroutine must keep running; the demos avoid time.Sleep on this target).
func pauseMs(ms int32) {
	dead := int32(nowMs()) + ms
	for int32(nowMs()) < dead {
		gosched()
	}
}

// corrText renders the Pearson correlation of the aligned pairs as "0.NN":
// r = sum(ref*dec) / (sqrt(sum ref^2) * sqrt(sum dec^2)), in hundredths,
// integer math (each norm is square-rooted before the multiply, so the
// intermediates stay inside int64).
func corrText(num, sigSq, decSq int64) string {
	if num < 0 || sigSq <= 0 || decSq <= 0 {
		return "0.00"
	}
	t := num * 100 / (isqrt64(sigSq) * isqrt64(decSq))
	if t > 99 {
		t = 99
	}
	return "0." + itoa(int(t))
}

// tenthsDB estimates 10*log10(ratio) in tenths of a dB for a ratio >= 1
// (integer log2 + polynomial on the mantissa; good to ~0.2 dB).
func tenthsDB(ratio int64) int {
	if ratio < 1 {
		return 0
	}
	// Normalize to m = ratio * 2^-e with m in [2^16, 2^17).
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
	for {
		nr := (r + x/r) >> 1
		if nr >= r {
			break
		}
		r = nr
	}
	return r
}
