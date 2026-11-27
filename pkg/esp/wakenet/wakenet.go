// Package wakenet binds llgo-compiled Go code to the ESP-SR WakeNet wake-word
// engine: the model partition is enumerated, a wakenet instance (wn9_hilexin
// by default) is loaded, and detections arrive as events - while the whole
// audio path runs on a FreeRTOS task of its own.
//
// # Why the audio path lives in C
//
// WakeNet detect is a synchronous recomputation that costs 3-6.4 ms per call
// (wn9 ~1.5 ms avg / 2.2 ms max measured on S31, the official wn10@3ch
// measurement 6.4 ms) and runs once per 32 ms audio chunk - roughly thirty
// heavy C calls per second. Worse, the engine's compute path executes the
// S31's PIE (SIMD coprocessor) instructions, and PIE exists on core 1 only:
// the FreeRTOS port migrates any task that executes one on core 0, and the
// migration path validates the task TCB's recorded stack pointer - which for
// the llgo scheduler task is a swapped-out fiber stack in the GC heap,
// outside its FreeRTOS stack bounds, and the kernel aborts. So the glue
// (glue/wakenet_glue.c) owns the whole pipeline on its own session task,
// pinned to core 1:
//
//	Init (scheduler task): config, model-partition mmap, optional I2S RX
//	  setup - no PIE anywhere - then the session task starts and the
//	  scheduler task waits (bounded) for its engine-ready sign
//	session task (core 1): engine create, then the loop - I2S RX 16 kHz
//	  16-bit mono (DMA descriptor ring) or a built-in test source replaying
//	  an injected PCM array -> one chunk -> detect (timed) -> detections
//	  and task errors into a bounded static event ring
//	Go polls the ring (microseconds per record) and reads counters
//
// detect never crosses from Go, and the scheduler task never executes a PIE
// instruction. The session task allocates nothing in its loop (the chunk
// buffer is one allocation at create, the ring is static). A pure-WakeNet
// instance costs ~21 KB internal RAM + the 12 KiB session-task stack, and
// ~320 KB PSRAM (weights, DET_MODE_95; the model itself is mmap'ed from
// flash) - measured on S31 under esp-emu; the numbers that make it fitable
// next to a WiFi+TLS stack.
//
// # Usage
//
//	err := wakenet.Init(wakenet.ModelConfig{})     // "" model = first wn*
//	fmt.Println(wakenet.ModelName(), wakenet.SampleRate())
//	err = wakenet.Start()                          // audio task begins
//	for ev := range wakenet.Events() {
//	    if ev.Kind == wakenet.EvDetected {
//	        // wake word heard
//	    }
//	}
//	err = wakenet.Stop()
//	err = wakenet.Close()
//
// A program without a microphone runs the engine on the test source instead:
// inject a PCM array (the esp-sr test apps embed one per wake word) through
// the demo hook (llgo_wakenet_set_test_pcm on the C side) and the task
// replays it in a loop at real-time pace - detections and per-call timings
// are then real, with no analog front end attached.
//
// # Event delivery
//
// Events() returns the package's single event channel: one consumer. The
// poller goroutine moves records from the C ring to the channel in bounded
// batches, handing the scheduler away between batches; a consumer that falls
// more than EventQueueLen events behind loses its own events silently. What
// is never silent is ring overflow on the C side: a dropped record is counted
// there and reported once as an EvOverflow event carrying the drop count.
// Close stops the poller; the channel closes when the poller's final pass
// drains (within one scheduler tick), and events already buffered stay
// readable until then.
//
// # Timing statistics
//
// The detect task measures every call. Stats reports the frames processed
// since Start and the average and worst detect time in microseconds - the
// numbers to compare against the 32 ms real-time budget (and the official
// benchmark) once the engine runs next to other work.
//
// # Init is a one-shot blocking call
//
// Init enumerates the model partition, resolves the model name, and waits
// (bounded) while the session task creates the engine - the create
// decompresses the model weights once (26 ms measured for wn9 on S31 under
// esp-emu; undocumented by Espressif) and runs on core 1, where the PIE
// coprocessor lives. The wait is the same class of one-shot lifecycle freeze
// as ble.Init and wifi.Setup. Detect never crosses from Go.
//
// # Error model
//
// Failures are the family's shared *esperr.Error, in explicitly separated
// code spaces (Code stays inspectable; errors.Is matches by code):
//
//	esp domain       raw esp_err, below 0x4000 - Init/Start/Close failures
//	                 reported by the IDF drivers (ESP_ERR_NO_MEM, ...), and
//	                 ESP_ERR_NOT_SUPPORTED (0x106) translated to
//	                 ErrUnsupported (no engine in this firmware/build)
//	Go domain 0x7Axx this package's own conditions: ErrNotReady (call before
//	                 Init), ErrInvalidState (wrong phase: double Init, Start
//	                 while running, Stop while stopped), ErrTimeout (Close's
//	                 bounded wait for the detect task saw no exit),
//	                 ErrBadValue (a ModelConfig this package refuses),
//	                 ErrNotFound (no such model, or no model partition)
//	stub tag 0x7F00  ErrUnsupported - a host build, where no C glue exists
//
// The 0x7Axx segment is this package's registry slot (0x79 is opus, 0x7B is
// ble, 0x7C the esperr meta segment, 0x7D/0x7E ws/tls, 0x7F stubs).
//
// # Scope
//
// One model instance at a time, one detection channel, mono 16-bit audio.
// DET_MODE_95 (weights in PSRAM) is fixed; multi-instance, MultiNet and the
// AFE audio front end (AEC/NS/VAD) are separate stages. The I2S setup is the
// plain Philips standard mode with three pins (BCLK/WS/DIN); a codec that
// needs MCLK or a shared full-duplex bus with a playback path is not wired
// yet - the pins are the whole configuration surface.
//
// The capture reads the bus as STEREO and keeps slot 0, because that is what
// a mono-slot configuration actually delivered on hardware: the codec drove
// two bytes per frame per slot regardless, so the mono-shaped read ran at
// twice the chunk rate and handed the engine interleaved channels - measured
// on S31 as 62.5 frames/s instead of 31.25 with zero detections, and
// 31.25 frames/s with the first detection right after the deinterleave fix.
package wakenet

import "github.com/xgo-dev/llgo/pkg/esp/esperr"

// The Go-domain codes: conditions this package detects itself, in the 0x7Axx
// segment (see the error model above).
const (
	codeNotReady     = 0x7A01
	codeInvalidState = 0x7A02
	codeTimeout      = 0x7A03
	codeBadValue     = 0x7A04
	codeNotFound     = 0x7A05
	codeUnsupported  = 0x7F00
)

// ErrNotReady reports a call made before Init completed (or after Close):
// there is no engine to start, stop, or query. It is the package's semantic
// sentinel - match it with errors.Is - and it keeps the 0x7A01 segment code
// on Error.Code as the diagnostic annotation, so errors.As and esperr.Code
// still expose the number the Go guard or the glue reported.
var ErrNotReady = newSentinel("esp/wakenet: not initialized", codeNotReady)

// ErrInvalidState reports a call in the wrong phase: a second Init, a Start
// while the engine runs, a Stop while it does not, or a Close of an engine
// that was never opened. It is the package's semantic sentinel - match it
// with errors.Is - and it keeps the 0x7A02 segment code on Error.Code as
// the diagnostic annotation, so errors.As and esperr.Code still expose the
// number the Go guard or the glue reported.
var ErrInvalidState = newSentinel("esp/wakenet: invalid state", codeInvalidState)

// ErrTimeout reports Close's bounded wait for the detect task's exit
// acknowledgment passing: the task did not leave within three seconds, and
// Close freed nothing (the engine stays loaded; a retry after diagnosing the
// wedged task is safe). It is the package's semantic sentinel - match it
// with errors.Is - and it keeps the 0x7A03 segment code on Error.Code as
// the diagnostic annotation, so errors.As and esperr.Code still expose the
// number the Go guard reported.
var ErrTimeout = newSentinel("esp/wakenet: timeout", codeTimeout)

// ErrBadValue reports a ModelConfig this package refuses before anything
// crosses into C: a model name longer than MaxModelName, a threshold outside
// the 0.4..0.9999 range the engine accepts, or SourceI2S without the three
// pins a master RX bus needs. It is the package's semantic sentinel - match
// it with errors.Is - and it keeps the 0x7A04 segment code on Error.Code as
// the diagnostic annotation, so errors.As and esperr.Code still expose the
// number the Go guard reported.
var ErrBadValue = newSentinel("esp/wakenet: invalid argument", codeBadValue)

// ErrNotFound reports a model resolution failure: the requested Model is not
// in the "model" partition, the partition carries no wn* model at all, or the
// partition itself is missing (esp_srmodel_init found nothing to map). It is
// the package's semantic sentinel - match it with errors.Is - and it keeps
// the 0x7A05 segment code on Error.Code as the diagnostic annotation, so
// errors.As and esperr.Code still expose the number the glue reported.
var ErrNotFound = newSentinel("esp/wakenet: model not found", codeNotFound)

// ErrUnsupported reports that this firmware (or build) has no WakeNet engine
// at all: a host build, or the ESP_ERR_NOT_SUPPORTED the glue reports when
// the firmware carries no engine. It is the package's semantic sentinel -
// match it with errors.Is - and it keeps the 0x7F00 stub-tag code on
// Error.Code as the diagnostic annotation, so errors.As and esperr.Code
// still expose the number the pipeline reported.
var ErrUnsupported = newSentinel("esp/wakenet: unsupported", codeUnsupported)

// newSentinel declares a semantic sentinel that keeps its historical
// segment code on Error.Code as the annotation (esperr.New is text-only;
// the glue still reports the code, and the error-model doc above pins the
// segment).
func newSentinel(text string, code int32) *esperr.Error {
	e, _ := esperr.New(text).(*esperr.Error)
	e.Code = code
	return e
}

// espErr wraps a raw esp_err code (0 = nil) - the esp domain of this
// package. Everything but the no-engine translation forwards as a bare
// esperr.Coded value - the driver's code and name are the diagnostic.
func espErr(code int32) error {
	if code == 0 {
		return nil
	}
	if code == 0x106 { // ESP_ERR_NOT_SUPPORTED: no engine in the firmware
		return ErrUnsupported
	}
	return esperr.Coded(code, ErrName(code))
}

// Source selects where the detect task's audio comes from.
type Source int32

const (
	// SourceTest feeds the engine from the test source: an injected PCM
	// array replayed in a loop at real-time pace, or silence when nothing
	// was injected. The default - a program can exercise the whole binding
	// with no analog front end attached.
	SourceTest Source = 0

	// SourceI2S captures from an I2S RX channel the glue configures in
	// standard (Philips) mode at the model's sample rate, 16-bit mono.
	// BCLK, WS and DIN must name real pins.
	SourceI2S Source = 1
)

// Pin identifies a GPIO by number (same numbering as the gpio package; each
// pkg/esp package is kept dependency-free, so this is a separate type).
type Pin int32

// MaxModelName is the longest model name Init accepts ("wn9_hilexin" is 11).
const MaxModelName = 31

// ModelConfig configures Init. The zero value is a working configuration:
// the first wn* model in the partition, the test source, the model's own
// detection threshold.
type ModelConfig struct {
	// Model is the model name to load ("wn9_hilexin"); empty selects the
	// first wn* model the partition carries. Names come from the esp-sr
	// model directory packaged into the firmware.
	Model string

	// Source selects the audio source (SourceTest by default).
	Source Source

	// BCLK, WS, DIN are the I2S pins for SourceI2S (all three required;
	// ignored for SourceTest).
	BCLK, WS, DIN Pin

	// Threshold overrides the model's detection threshold (0.4..0.9999;
	// higher = fewer false wakes). Zero keeps the model's own value. The
	// engine carries per-model defaults (the model info string's threshold
	// fields), which is the right starting point.
	Threshold float32
}

// EventKind identifies one event kind; it aliases int32 so the Ev* constants
// read directly out of Event.Kind.
type EventKind = int32

// Event kinds delivered on the Events channel.
const (
	// EvNone is the zero-value sentinel, never delivered.
	EvNone EventKind = 0

	// EvDetected reports one wake-word detection. Code carries the engine's
	// wakenet_state (1 = WAKENET_DETECTED). The engine re-arms itself within
	// a few frames (measured: back to no-detect without any caller action;
	// the glue never calls clean - it crashes on this esp-sr build), so the
	// next utterance triggers a fresh event.
	EvDetected EventKind = 1

	// EvError reports a failure inside the audio path: Code carries the
	// esp_err (an I2S read failure, most commonly). The task backs off
	// briefly and keeps running - a flaky bus produces a stream of these,
	// not a dead engine.
	EvError EventKind = 2

	// EvOverflow reports C-ring loss: Code is the number of records dropped
	// since the previous overflow report. Detection events are rare, so
	// reaching this means a program stopped consuming events for a long
	// time.
	EvOverflow EventKind = -1
)

// Event is one record from the glue's event ring.
type Event struct {
	// Kind is one of the Ev* constants.
	Kind EventKind

	// Code is the payload: the wakenet_state for EvDetected, the esp_err for
	// EvError, the dropped-record count for EvOverflow, 0 otherwise.
	Code int32

	// Timestamp is the esp_timer millisecond count when the glue posted the
	// record. It wraps after ~49.7 days (a u32 crossing the 32-bit boundary
	// unchanged); a program that cares computes deltas, not absolute dates.
	Timestamp uint32
}

// Limits and defaults of this package.
const (
	// EventQueueLen is the event channel's buffer. The poller never blocks
	// the C ring on a full channel: a consumer that has not drained its
	// events loses them (and the C ring's own overflow reporting keeps an
	// honest count of what did not fit there).
	EventQueueLen = 8

	// maxPollBatch bounds how many records the poller moves per pass, so a
	// burst cannot hold the scheduler: the batch ends with a Gosched and one
	// tick of rest.
	maxPollBatch = 32
)

// evRecord mirrors the glue's llgo_wakenet_ev_t field for field (all
// four-byte scalars; the C side pins the layout with _Static_assert).
type evRecord struct {
	Kind      int32
	Code      int32
	Timestamp uint32
}

// The session state. The llgo scheduler is cooperative, so plain variables
// are race-free: a caller only observes them between its own scheduling
// points, and the poller touches them only from goroutine switches.
var (
	session   bool // Init done, Close not yet
	running   bool // Start done, Stop not yet
	shutdown  bool // Close done; retires the poller
	opened    bool // Init reached the glue (legalizes Close on a failed Init)
	modelName string
	chunkN    int32
	rateHz    int32
	eventsCh  chan Event
)

// Init resolves and loads the wake-word model. It enumerates the "model"
// partition (mmap), resolves the model name, brings up the optional I2S RX
// channel (not enabled until Start), then starts the glue's session task and
// waits (bounded) while it creates the engine - the create runs on the
// session task, never on the scheduler task, because wn9's compute path
// executes PIE instructions and PIE exists on core 1 only (see the package
// docs). The blocking here is that bounded wait, a one-shot lifecycle freeze
// like ble.Init (26 ms measured for wn9 on S31). The resolved name and the
// engine's chunk size and sample rate stay queryable through
// ModelName/ChunkSamples/SampleRate.
//
// A failed Init leaves only Close legal: the glue holds the session task and
// the model mapping until Close releases them.
//
// Errors: ErrBadValue for a config this package refuses before crossing into
// C, ErrNotFound for a model (or partition) that is not there, ErrNotFound
// wrapped around ESP_ERR_NOT_FOUND, the raw esp_err for driver failures,
// ErrInvalidState for a second Init without a Close.
func Init(cfg ModelConfig) error {
	if session {
		return esperr.Wrap("wakenet: init", ErrInvalidState)
	}
	if len(cfg.Model) > MaxModelName {
		return esperr.Wrap("wakenet: init", ErrBadValue)
	}
	if cfg.Threshold != 0 && (cfg.Threshold < 0.4 || cfg.Threshold > 0.9999) {
		return esperr.Wrap("wakenet: init", ErrBadValue)
	}
	if cfg.Source != SourceTest && cfg.Source != SourceI2S {
		return esperr.Wrap("wakenet: init", ErrBadValue)
	}
	if cfg.Source == SourceI2S && (cfg.BCLK < 0 || cfg.WS < 0 || cfg.DIN < 0) {
		return esperr.Wrap("wakenet: init", ErrBadValue)
	}
	// The threshold crosses as thousandths: floats never cross the llgo
	// soft-float / IDF hard-float boundary.
	milli := int32(0)
	if cfg.Threshold != 0 {
		milli = int32(cfg.Threshold*1000 + 0.5)
	}
	var namePtr *uint8
	var nameLen int32
	if cfg.Model != "" {
		b := []byte(cfg.Model)
		namePtr = &b[0]
		nameLen = int32(len(b))
	}
	src := int32(cfg.Source)
	e := espErr(wnInit(namePtr, nameLen, src, int32(cfg.BCLK), int32(cfg.WS),
		int32(cfg.DIN), milli))
	// The glue holds state after any real call (the session task, the model
	// mapping) - only Close may release it. The stub tag 0x7F00 means the
	// call never reached a glue at all, so there is nothing to close.
	if esperr.Code(e) != codeUnsupported {
		opened = true
	}
	if e != nil {
		return e
	}
	var nameBuf [MaxModelName + 1]byte
	modelName = string(nameBuf[:wnModelName(&nameBuf[0], int32(len(nameBuf)))])
	wnChunk(&chunkN, &rateHz)
	eventsCh = make(chan Event, EventQueueLen)
	session = true
	shutdown = false
	go pollLoop(eventsCh)
	return nil
}

// ModelName returns the model Init resolved ("" before Init).
func ModelName() string { return modelName }

// ChunkSamples returns the number of int16 samples one detect call consumes
// (512 for wn9 = 32 ms at 16 kHz; 0 before Init).
func ChunkSamples() int32 { return chunkN }

// LastRMS is the root-mean-square level of the audio chunk the engine saw
// most recently (int16 scale). It separates "the engine hears nothing" from
// "the engine hears silence": a stopped frame counter means a dead feed, a
// stuck-near-zero RMS means a dead microphone path (codec register state or
// a pin another owner re-routed).
func LastRMS() int32 {
	if !session {
		return 0
	}
	return wnLastRMS()
}

// PeakRMS is the loudest chunk since the last Start - sampled once per beat,
// LastRMS misses a short wake word, this cannot.
func PeakRMS() int32 {
	if !session {
		return 0
	}
	return wnPeakRMS()
}

// Hits counts the chunks where the engine's detect call returned a positive
// state since the last Start. Hits without detections means the event path
// dropped something; no hits at all means the engine itself stopped
// recognising (the audio feed is alive - check LastRMS/PeakRMS).
func Hits() int32 {
	if !session {
		return 0
	}
	return wnHits()
}

// SampleRate returns the engine's sample rate in Hz (16000; 0 before Init).
func SampleRate() int32 { return rateHz }

// Start resumes the session task's audio loop: it captures chunks, runs
// detect on each, posts detections and errors to the event ring, and
// accumulates the Stats counters (the task resets them on this transition).
// Start returns as soon as the loop is marked running - the loop's own
// failures arrive as EvError events, not as a return code. Errors:
// ErrNotReady before Init, ErrInvalidState while already running,
// esperr.ErrClosed after Close.
func Start() error {
	if shutdown {
		return esperr.Wrap("wakenet: start", esperr.ErrClosed)
	}
	if !session {
		return esperr.Wrap("wakenet: start", ErrNotReady)
	}
	if running {
		return esperr.Wrap("wakenet: start", ErrInvalidState)
	}
	drain() // stale records from a previous run never reach the new session
	if e := espErr(wnStart()); e != nil {
		return e
	}
	running = true
	return nil
}

// Stop parks the session task's audio loop. Cooperative: the loop notices
// at its top and stops capturing within one chunk period, never mid-C-call,
// so the engine's state stays consistent for a re-Start. Stop also hands the
// I2S pins back: the capture channel is deleted (the board's I2S pins are
// single-owner, so a conversation round can open its own channels while the
// engine is parked) and Start rebuilds it. Stop blocks up to 2 s waiting for
// that teardown. The engine itself stays alive - model mapping and working
// buffers included - which is what makes a Stop/Start cycle cheap and free
// of the re-create failures a Close/Init cycle hits on a fragmented heap.
// Events already in flight stay readable. Errors: ErrNotReady before Init,
// ErrInvalidState when the engine is not running, ErrTimeout when the bus
// did not come free, esperr.ErrClosed after Close.
func Stop() error {
	if shutdown {
		return esperr.Wrap("wakenet: stop", esperr.ErrClosed)
	}
	if !session {
		return esperr.Wrap("wakenet: stop", ErrNotReady)
	}
	if !running {
		return esperr.Wrap("wakenet: stop", ErrInvalidState)
	}
	e := espErr(wnStop())
	running = false
	return e
}

// Events returns the package's single event channel: one consumer, created
// at Init. Before Init it is nil; after Close it closes (within one
// scheduler tick, after the poller's final pass - events already delivered
// stay readable until then).
func Events() <-chan Event { return eventsCh }

// Detected reports whether a detection fired since the last Start (sticky
// until the next Start resets it). It is the polling alternative to the
// Events channel for programs that just want "did it fire".
func Detected() bool { return session && !shutdown && wnDetected() == 1 }

// Stats reports the detect task's counters since the last Start: frames
// processed, average and worst detect call in microseconds. All zeros before
// the first Start. The three fields come from single-writer u32 counters the
// task updates without locks; they can be one frame apart from each other,
// which a diagnostic does not mind. The number to judge them by: one chunk
// is 32000 µs of audio, so an average well under that leaves the task - and
// the rest of the firmware - real time to spare.
func Stats() (frames int32, avgDetectUs int32, maxDetectUs int32) {
	if !session || shutdown {
		return 0, 0, 0
	}
	frames, avgDetectUs, maxDetectUs = 0, 0, 0
	wnStats(&frames, &avgDetectUs, &maxDetectUs)
	return frames, avgDetectUs, maxDetectUs
}

// Close tears the engine down: the session task exits (waited for, bounded),
// then the I2S channel, the engine, the model partition mapping and the
// buffers are released. The Events channel closes when the poller's final
// pass drains; the package returns to the un-init state, and a fresh Init
// may follow. Close is legal even after a failed Init - the glue holds its
// session task until Close. Errors: esperr.ErrClosed for a second Close,
// ErrInvalidState for a Close of an engine never opened (Init never reached
// the glue), ErrTimeout when the session task ignored its exit for three
// seconds (nothing was freed; see ErrTimeout), the raw esp_err for driver
// failures.
func Close() error {
	if shutdown {
		return esperr.Wrap("wakenet: close", esperr.ErrClosed)
	}
	// opened, not session: a failed Init still left state in the glue (the
	// session task, at the least) that only Close may release.
	if !opened {
		return esperr.Wrap("wakenet: close", ErrInvalidState)
	}
	running = false
	e := espErr(wnClose())
	shutdown = true
	session = false
	opened = false
	modelName = ""
	chunkN = 0
	rateHz = 0
	return e
}

// drain empties the C ring and the event channel: consumer-owned indexes,
// run between scheduling points, so a stale detection from a previous run
// can never be mistaken for one of the new session.
func drain() {
	var rec evRecord
	for wnPoll(&rec) == 1 {
	}
	for {
		select {
		case <-eventsCh:
		default:
			return
		}
	}
}

// pollLoop is the package's single C-ring consumer: drain up to maxPollBatch
// records into the channel (never blocking - a consumer behind EventQueueLen
// loses its own events), report any C-side drop count, hand the rest of the
// scheduling slice to other goroutines, then rest the FreeRTOS task for one
// real tick - which also feeds the idle tasks and the watchdog. On shutdown
// it closes the channel; nothing else ever does.
func pollLoop(ch chan Event) {
	for {
		if shutdown {
			close(ch)
			return
		}
		for n := 0; n < maxPollBatch; n++ {
			var rec evRecord
			if wnPoll(&rec) != 1 {
				break
			}
			ev := Event{Kind: rec.Kind, Code: rec.Code, Timestamp: rec.Timestamp}
			select {
			case ch <- ev:
			default:
			}
		}
		if d := wnOverflowDelta(); d > 0 {
			select {
			case ch <- Event{Kind: EvOverflow, Code: d}:
			default:
			}
		}
		gosched()
		vTaskDelay(1) // one FreeRTOS tick: 10 ms at the IDF default 100 Hz
	}
}

// ErrName maps a code this package can report to its name: the Go-domain
// codes, the common esp_err codes the lifecycle calls can return, and a
// generic form for everything else.
func ErrName(code int32) string {
	switch code {
	case 0:
		return "ESP_OK"
	case -1:
		return "ESP_FAIL"
	case 0x101:
		return "ESP_ERR_NO_MEM"
	case 0x102:
		return "ESP_ERR_INVALID_ARG"
	case 0x103:
		return "ESP_ERR_INVALID_STATE"
	case 0x104:
		return "ESP_ERR_INVALID_SIZE"
	case 0x105:
		return "ESP_ERR_NOT_FOUND"
	case 0x106:
		return "ESP_ERR_NOT_SUPPORTED"
	case 0x107:
		return "ESP_ERR_TIMEOUT"
	case codeNotReady:
		return "WAKENET_ERR_NOT_READY"
	case codeInvalidState:
		return "WAKENET_ERR_INVALID_STATE"
	case codeTimeout:
		return "WAKENET_ERR_TIMEOUT"
	case codeBadValue:
		return "WAKENET_ERR_BAD_VALUE"
	case codeNotFound:
		return "WAKENET_ERR_NOT_FOUND"
	case codeUnsupported:
		return "WAKENET_ERR_UNSUPPORTED"
	}
	const digits = "0123456789ABCDEF"
	var buf [8]byte
	buf[0] = '0'
	buf[1] = 'x'
	for i := 0; i < 6; i++ {
		v := byte(code >> (uint(5-i) * 4))
		buf[2+i] = digits[v&0xF]
	}
	start := 2
	for start < 6 && buf[start] == '0' { // "0x1f" reads better than "0x00001f"
		start++
	}
	return "esp_err " + string(buf[0:2]) + string(buf[start:8])
}
