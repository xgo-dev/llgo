// Package opus binds llgo-compiled Go code to the ESP-IDF Opus codec
// (the managed component 78/esp-opus: vanilla upstream Opus compiled
// fixed-point, -DFIXED_POINT=1 -DDISABLE_FLOAT_API). It exposes the two
// codec directions Go wants: encode S16 PCM into Opus packets, and decode
// Opus packets back to S16 PCM (with packet-loss concealment on an empty
// frame), plus the encoder ctls that matter for a live stream: bitrate,
// complexity and frame duration.
//
// # Layout
//
// Every package under pkg/esp follows the same template (glue lives in a
// subdirectory because the go tool rejects C sources inside a Go package):
//
//	opus.go           public API, no build tag
//	opus_idf.go       //go:build llgo.idf - bodyless //go:linkname externs
//	opus_stub.go      //go:build !llgo.idf - host-build stubs (go vet/gopls)
//	glue/opus_glue.c  C glue: the codec handle table and the opus_* call
//	                  translations, ESP_PLATFORM-guarded, with honest
//	                  ESP_ERR_NOT_SUPPORTED stubs outside an IDF build
//
// The consuming firmware registers the component dependency (78/esp-opus)
// and copies glue/opus_glue.c into its main component next to the synced
// glues of the other pkg/esp packages (integration way 1 in
// pkg/esp/README.md). Unlike ble/bt there is no mailbox and no task here:
// Opus is a synchronous library, every call runs to completion on the
// calling goroutine's own stack, so there are no callbacks that could race
// the single-task llgo fiber scheduler.
//
// # Usage
//
//	enc, err := opus.NewEncoder(48000, 1, opus.AppAudio)
//	err = enc.SetBitrate(32000)
//	pcm := make([]int16, 960) // 48000 Hz * 20 ms * 1 channel
//	// ... fill one frame of samples ...
//	frame, err := enc.Encode(pcm)        // one Opus packet
//	dec, err := opus.NewDecoder(48000, 1)
//	out := make([]int16, 2880)           // room for the largest frame
//	n, err := dec.Decode(frame, out)     // n samples per channel
//	n, err = dec.Decode(nil, out[...] )  // empty frame: concealment
//	enc.Close()
//	dec.Close()
//
// # Frames and buffers
//
// One Encode call consumes exactly frameSize*channels samples -
// frameSize = sampleRate * frameMs / 1000, frameMs ∈ {5, 10, 20, 40, 60}
// (default 20; Opus also allows 2.5 ms, which this binding's integer
// millisecond interface does not spell). The returned frame aliases the
// encoder's internal buffer (MaxPacketBytes, the one-frame wire maximum)
// and is valid until the next call on the same encoder; copy it out to keep
// it. Decode sizes its output by the packet's own duration: pcmOut must
// hold len/ channels ≥ the largest duration the stream can produce, or the
// decoder reports ErrBufferTooSmall.
//
// # Fixed point
//
// The component is compiled fixed-point and with the float API disabled,
// so the PCM interface is opus_int16 in both directions and there is no
// float path. Quality is the fixed-point reference quality; the acceptance
// probe under _demo/esp-idf-opus pins it with an SNR gate (encode→decode a
// synthetic tone, SNR > 25 dB).
//
// # Memory
//
// The codec states are C objects (mono 48 kHz: encoder 24.7 KiB, decoder
// 18.1 KiB, see StateBytes), never copied into Go: the glue owns them in a
// small static handle table (two encoders, two decoders - more reports
// ESP_ERR_NO_MEM), and Go carries only the handle. The library allocates
// them through malloc. Whether that malloc rides the llgo tinygogc wrap
// (--wrap=malloc at the final link, states inside the GC heap) or lands on
// the firmware's own IDF heap depends on how the consuming project links;
// both are workable - the demo's sdkconfig prefers PSRAM for allocations of
// 16 KiB and up (CONFIG_SPIRAM_USE_MALLOC), so every state lands in PSRAM
// there - but the ledger differs, and pkg/esp/opus itself makes no promise
// beyond "C-side, never copied". All of it happens on the calling
// goroutine - never in a callback or an interrupt.
//
// # Stacks
//
// opus_encode/opus_decode are deep C calls that take alloca scratch in the
// tens of KiB. Goroutine fiber stacks on this target are small (2 KiB
// default; the main goroutine's fiber is 4x the -goroutine-stack-size
// value, floor 16 KiB), and the fiber guard only checks at cooperative
// swap points - a deep C call that overflows its fiber writes below the
// floor unchecked, which measurably corrupted the allocator globals before
// anything crashed. Size the stacks explicitly: build the program with
// llgo build -goroutine-stack-size 16384 (main fiber 64 KiB, measured to
// fit the 48 kHz paths) and keep the static GC window big enough to hold
// the main fiber (the acceptance demo pairs it with a 128 KiB window).
//
// # Error model
//
// Failures are the family's shared *esperr.Error, in separated code
// spaces (Code stays inspectable; errors.Is matches by code):
//
//	opus domain    the raw Opus error, kept negative (-1..-7) exactly as
//	               the library reports it and named OPUS_* - the same
//	               pass-through convention the tls package uses for
//	               negative mbedtls codes
//	esp domain     raw esp_err from the glue itself (no free handle slot:
//	               ESP_ERR_NO_MEM; a stale handle: ESP_ERR_INVALID_STATE)
//	Go domain      0x79xx - this package's own conditions, refused before
//	               anything is submitted (ErrBadValue; see the segment plan
//	               below)
//	stub tag       0x7F00 - a host build, or a firmware compiled without
//	               the codec component (ErrUnsupported)
//
// The 0x79xx segment: the family's internal segments are 0x76xx bt,
// 0x77xx A2DP, 0x78xx reserved for further classic-audio profiles, 0x7Bxx
// ble, 0x7Cxx the esperr meta segment, 0x7Dxx ws, 0x7Exx tls and 0x7Fxx
// the stub tag; 0x79xx is the lowest unclaimed segment.
package opus

import "github.com/xgo-dev/llgo/pkg/esp/esperr"

// The Go-domain codes: conditions this package detects itself, in the
// family's unclaimed 0x79xx segment (see the error model above). The
// codec's own failures do NOT live here - they keep the library's negative
// codes, which cannot collide with any esp_err or family segment by sign.
const (
	codeBadValue    = 0x7901
	codeUnsupported = 0x7F00
)

// The seven Opus error codes, named as the library names them. They are the
// library's own negative values (opus_defines.h), kept as the sentinel
// codes so errors.Is matches a raw codec failure without any wrapping
// step - the same convention the tls package uses for negative mbedtls
// codes.
var (
	// ErrBadArg reports an argument the codec refused: a value outside the
	// legal range (bitrate, complexity), a frame size that is not a valid
	// Opus frame duration, a decoder reset on a live stream.
	ErrBadArg = &esperr.Error{Code: -1, Name: "OPUS_BAD_ARG"}

	// ErrBufferTooSmall reports an output buffer that cannot hold the result:
	// a Decode whose pcmOut is shorter than the packet's duration.
	ErrBufferTooSmall = &esperr.Error{Code: -2, Name: "OPUS_BUFFER_TOO_SMALL"}

	// ErrInternal reports an internal codec error. By the Opus
	// documentation's own wording this is always a bug, in the library or
	// in this binding.
	ErrInternal = &esperr.Error{Code: -3, Name: "OPUS_INTERNAL_ERROR"}

	// ErrInvalidPacket reports compressed data that is corrupted or not an
	// Opus packet.
	ErrInvalidPacket = &esperr.Error{Code: -4, Name: "OPUS_INVALID_PACKET"}

	// ErrUnimplemented reports a request the codec's build does not
	// implement (an API disabled at compile time, e.g. the float API of
	// this fixed-point build).
	ErrUnimplemented = &esperr.Error{Code: -5, Name: "OPUS_UNIMPLEMENTED"}

	// ErrInvalidState reports a codec asked to work on a state that is not
	// valid for the request.
	ErrInvalidState = &esperr.Error{Code: -6, Name: "OPUS_INVALID_STATE"}

	// ErrAllocFail reports the codec's own allocator refusing a request
	// (and, transitively, the firmware being out of GC-heap/PSRAM memory).
	ErrAllocFail = &esperr.Error{Code: -7, Name: "OPUS_ALLOC_FAIL"}
)

// ErrBadValue reports a caller-side argument this package refuses before
// submitting anything: a sample rate outside the Opus set
// (8000/12000/16000/24000/48000), a channel count outside 1..2, an unknown
// application, a frame duration outside {5,10,20,40,60} ms, an Encode
// whose pcm does not hold exactly one frame, a Decode whose pcmOut is too
// small to be sized.
var ErrBadValue = &esperr.Error{Code: codeBadValue, Name: ErrName(codeBadValue)}

// ErrUnsupported reports that this firmware (or build) has no Opus at all:
// a host build, or an ESP-IDF firmware that did not link the codec
// component. The glue reports ESP_ERR_NOT_SUPPORTED and the package
// renders it as this sentinel, the family's stub tag.
var ErrUnsupported = &esperr.Error{Code: codeUnsupported, Name: ErrName(codeUnsupported)}

// errOf wraps a raw pipeline code as this package's error value: 0 is nil,
// the negative Opus codes keep their sign (the tls package's mbedtls
// convention - the sign is what distinguishes an Opus error from an
// esp_err), positive codes are esp_err or the stub tag.
func errOf(code int32) error {
	if code == 0 {
		return nil
	}
	if code == codeUnsupported {
		return ErrUnsupported
	}
	return &esperr.Error{Code: code, Name: ErrName(code)}
}

// Application selects the encoder's signal model, the values Opus
// documents (opus_application in opus_defines.h).
const (
	// AppVoip biases the encoder for speech: gives best quality at 16 kHz
	// and applies a 6 kHz highpass.
	AppVoip = 2048 // OPUS_APPLICATION_VOIP
	// AppAudio biases the encoder for music and general audio in the full
	// 20 kHz band.
	AppAudio = 2049 // OPUS_APPLICATION_AUDIO
	// AppLowDelay configures the encoder for the least latency the format
	// allows, trading quality for it.
	AppLowDelay = 2051 // OPUS_APPLICATION_RESTRICTED_LOWDELAY
)

// Frame durations Encode accepts, in milliseconds (Opus' frame size set
// minus the 2.5 ms value this binding's integer interface does not spell;
// see the frames section above).
const (
	Frame5Ms  = 5
	Frame10Ms = 10
	Frame20Ms = 20
	Frame40Ms = 40
	Frame60Ms = 60

	// DefaultFrameMs is the frame duration NewEncoder uses - Opus' usual
	// default and the value with the best quality/latency trade-off.
	DefaultFrameMs = 20

	// MaxPacketBytes is the wire maximum of one Opus packet (1275 bytes)
	// plus the byte Encode's length reporting needs. Encode always returns
	// a slice of this length's buffer, so a caller can pre-size any copy
	// target by it.
	MaxPacketBytes = 1276
)

// MaxBitrate and MinBitrate are the bitrate bounds opus_encoder_ctl
// (OPUS_SET_BITRATE) accepts; the codec itself clamps values inside the
// range and refuses values outside it (ErrBadArg). BitrateAuto and
// BitrateMax are the two documented special values.
const (
	MinBitrate  = 500
	MaxBitrate  = 512000
	BitrateAuto = -1000 // OPUS_BITRATE_AUTO: the encoder picks from the rate and channels
	BitrateMax  = -1    // OPUS_BITRATE_MAX: the maximum the format allows
)

// maxFrameSamples is the largest samples-per-channel value any legal
// (rate, frameMs) pair can produce: 48000 * 60 ms. Decode pcmOut buffers
// are sized against it (per channel).
const maxFrameSamples = 2880

// validRate reports whether r is one of the sample rates Opus accepts.
func validRate(r int) bool {
	switch r {
	case 8000, 12000, 16000, 24000, 48000:
		return true
	}
	return false
}

// frameSamples maps (sample rate, frame duration) to the samples-per-channel
// count Encode consumes; every legal pair divides exactly. ok is false for
// a duration outside the legal set (the caller renders ErrBadValue).
func frameSamples(rate, ms int) (n int, ok bool) {
	switch ms {
	case Frame5Ms, Frame10Ms, Frame20Ms, Frame40Ms, Frame60Ms:
		return rate * ms / 1000, true
	}
	return 0, false
}

// Encoder encodes S16 PCM into Opus packets. Its C state lives behind the
// glue's handle table; the zero value is not usable - build one with
// NewEncoder. The methods are not concurrent-safe with each other; drive
// one encoder from one goroutine at a time (the cooperative scheduler
// makes that the natural case on the device).
type Encoder struct {
	handle     int32
	sampleRate int
	channels   int
	frameMs    int
	frameSize  int // samples per channel Encode consumes
	out        []byte
}

// NewEncoder creates an encoder for sampleRate (8000/12000/16000/24000/
// 48000), channels (1 or 2) and application (AppVoip/AppAudio/AppLowDelay).
// The encoder starts at Opus' own defaults: 20 ms frames, VBR, the
// bitrate the rate and channel count imply. Everything else is a ctl call
// away (SetBitrate, SetComplexity, SetFrameSizeMs). On a firmware without
// the codec component, or a host build, this reports ErrUnsupported.
func NewEncoder(sampleRate, channels, application int) (*Encoder, error) {
	if !validRate(sampleRate) || channels < 1 || channels > 2 {
		return nil, ErrBadValue
	}
	switch application {
	case AppVoip, AppAudio, AppLowDelay:
	default:
		return nil, ErrBadValue
	}
	var h int32
	if code := encOpen(int32(sampleRate), int32(channels), int32(application), &h); code != 0 {
		return nil, errOf(code)
	}
	return &Encoder{
		handle:     h,
		sampleRate: sampleRate,
		channels:   channels,
		frameMs:    DefaultFrameMs,
		frameSize:  sampleRate * DefaultFrameMs / 1000,
		out:        make([]byte, MaxPacketBytes),
	}, nil
}

// Encode encodes exactly one frame: len(pcm) must be
// frameSize*channels samples (interleaved, S16), and the returned slice is
// the packet - aliased to the encoder's internal buffer, valid until the
// next call on this encoder. A length mismatch is refused with ErrBadValue
// before anything is submitted; after Close the encoder reports
// esperr.ErrClosed like the family's other closed objects.
func (e *Encoder) Encode(pcm []int16) ([]byte, error) {
	if e == nil || e.handle == 0 {
		return nil, esperr.ErrClosed
	}
	if len(pcm) != e.frameSize*e.channels {
		return nil, ErrBadValue
	}
	var p *int16
	if len(pcm) > 0 {
		p = &pcm[0]
	}
	var n int32
	if code := encEncode(e.handle, p, int32(e.frameSize), &e.out[0], int32(len(e.out)), &n); code != 0 {
		return nil, errOf(code)
	}
	return e.out[:n], nil
}

// SetBitrate sets the target bitrate in bits per second, clamped by the
// codec to [MinBitrate, MaxBitrate] (values outside it, except the two
// special values below, are ErrBadArg).
func (e *Encoder) SetBitrate(bps int) error {
	if e == nil || e.handle == 0 {
		return esperr.ErrClosed
	}
	return errOf(encSetBitrate(e.handle, int32(bps)))
}

// SetComplexity sets the computation allowance, 0 (cheapest) to 10
// (best quality per bitrate); fixed-point SILK uses the range 0..10 and
// the codec refuses values outside it (ErrBadArg).
func (e *Encoder) SetComplexity(c int) error {
	if e == nil || e.handle == 0 {
		return esperr.ErrClosed
	}
	return errOf(encSetComplexity(e.handle, int32(c)))
}

// SetFrameSizeMs changes the frame duration the next Encode calls consume,
// one of Frame5Ms..Frame60Ms (20 ms default). The sample count per call
// moves with it: sampleRate * ms / 1000 per channel.
func (e *Encoder) SetFrameSizeMs(ms int) error {
	if e == nil || e.handle == 0 {
		return esperr.ErrClosed
	}
	n, ok := frameSamples(e.sampleRate, ms)
	if !ok {
		return ErrBadValue
	}
	e.frameMs, e.frameSize = ms, n
	return nil
}

// Lookahead returns the encoder's algorithmic delay in samples per channel
// (OPUS_GET_LOOKAHEAD): the decoded stream is shifted by this many samples
// relative to the encode input, so a roundtrip comparison must align by it
// (decoded frame i starts at input sample i*frameSize - Lookahead). A closed
// or nil encoder reports 0.
func (e *Encoder) Lookahead() int {
	if e == nil || e.handle == 0 {
		return 0
	}
	return int(encLookahead(e.handle))
}

// Close frees the encoder's C state. Idempotent by the family convention:
// the first call releases the handle, a second reports esperr.ErrClosed.
func (e *Encoder) Close() error {
	if e == nil || e.handle == 0 {
		return esperr.ErrClosed
	}
	code := encClose(e.handle)
	e.handle = 0
	return errOf(code)
}

// Decoder decodes Opus packets back to S16 PCM. Same state discipline as
// Encoder: the C state lives behind the glue's handle table, build one
// with NewDecoder, drive it from one goroutine at a time.
type Decoder struct {
	handle     int32
	sampleRate int
	channels   int
}

// NewDecoder creates a decoder for sampleRate (8000/12000/16000/24000/
// 48000) and channels (1 or 2). The decoder must match the encoder's rate
// and channel count - a mismatched packet still decodes (the packet
// carries its own configuration) but the rate the samples represent is
// the decoder's. ErrUnsupported on a build without the codec.
func NewDecoder(sampleRate, channels int) (*Decoder, error) {
	if !validRate(sampleRate) || channels < 1 || channels > 2 {
		return nil, ErrBadValue
	}
	var h int32
	if code := decOpen(int32(sampleRate), int32(channels), &h); code != 0 {
		return nil, errOf(code)
	}
	return &Decoder{handle: h, sampleRate: sampleRate, channels: channels}, nil
}

// Decode decodes one packet into pcmOut (interleaved S16) and returns the
// samples produced per channel - the packet carries its own duration, so
// the value is read off the data, not configured. pcmOut is sized by the
// caller: len(pcmOut)/channels is the largest duration it can hold, and a
// packet longer than that is ErrBufferTooSmall (never truncated). A nil or
// empty frame asks the decoder for packet-loss concealment: it invents the
// continuation of the stream, the standard Opus behavior for a lost
// packet. After Close the decoder reports esperr.ErrClosed.
func (d *Decoder) Decode(frame []byte, pcmOut []int16) (int, error) {
	if d == nil || d.handle == 0 {
		return 0, esperr.ErrClosed
	}
	if len(pcmOut)/d.channels < 1 {
		return 0, ErrBadValue
	}
	var data *uint8
	if len(frame) > 0 {
		data = &frame[0]
	}
	var p *int16
	if len(pcmOut) > 0 {
		p = &pcmOut[0]
	}
	var n int32
	if code := decDecode(d.handle, data, int32(len(frame)), p,
		int32(len(pcmOut)/d.channels), &n); code != 0 {
		return 0, errOf(code)
	}
	return int(n), nil
}

// Close frees the decoder's C state (same idempotence as Encoder.Close).
func (d *Decoder) Close() error {
	if d == nil || d.handle == 0 {
		return esperr.ErrClosed
	}
	code := decClose(d.handle)
	d.handle = 0
	return errOf(code)
}

// EncoderStateBytes reports the C state size in bytes an encoder for
// channels needs (opus_encoder_get_size) - the memory ledger's encoder
// line. 0 on a host build (no glue).
func EncoderStateBytes(channels int) int {
	return int(encSize(int32(channels)))
}

// DecoderStateBytes reports the C state size in bytes a decoder for
// channels needs (opus_decoder_get_size) - the memory ledger's decoder
// line. 0 on a host build (no glue).
func DecoderStateBytes(channels int) int {
	return int(decSize(int32(channels)))
}

// ErrName names the codes this pipeline can report: the negative Opus
// errors with their library names, the esp_err codes the glue itself can
// return, this package's 0x79xx segment, and the stub tag. The negative
// Opus codes are the raw library values (OPUS_BAD_ARG .. OPUS_ALLOC_FAIL),
// rendered here; ErrText gives them as full sentences.
func ErrName(code int32) string {
	if code < 0 {
		switch code {
		case -1:
			return "OPUS_BAD_ARG"
		case -2:
			return "OPUS_BUFFER_TOO_SMALL"
		case -3:
			return "OPUS_INTERNAL_ERROR"
		case -4:
			return "OPUS_INVALID_PACKET"
		case -5:
			return "OPUS_UNIMPLEMENTED"
		case -6:
			return "OPUS_INVALID_STATE"
		case -7:
			return "OPUS_ALLOC_FAIL"
		}
		return "opus error " + itoa(int(code))
	}
	switch code {
	case 0:
		return "OPUS_OK"
	case 0x101:
		return "ESP_ERR_NO_MEM"
	case 0x102:
		return "ESP_ERR_INVALID_ARG"
	case 0x103:
		return "ESP_ERR_INVALID_STATE"
	case 0x106:
		return "ESP_ERR_NOT_SUPPORTED"
	case 0x107:
		return "ESP_ERR_TIMEOUT"
	case codeBadValue:
		return "OPUS_ERR_BAD_VALUE"
	case codeUnsupported:
		return "OPUS_ERR_UNSUPPORTED"
	}
	return "esp error 0x" + hex6(uint32(code))
}

// ErrText renders an error code as the plain-language sentence the Opus
// manual uses for it - what a log line wants next to the raw code. Known
// Go-domain and esp_err codes render with their own meaning; anything
// else falls back to the ErrName token.
func ErrText(code int32) string {
	if code < 0 {
		switch code {
		case -1:
			return "invalid argument: a value is out of range or the frame size is not a valid Opus duration"
		case -2:
			return "not enough room in the output buffer for the result"
		case -3:
			return "internal codec error (a bug in the library or the binding)"
		case -4:
			return "corrupted or non-Opus compressed data"
		case -5:
			return "not implemented in this build of the codec"
		case -6:
			return "the codec state is invalid for this request"
		case -7:
			return "the codec could not allocate memory for the request"
		}
		return ErrName(code)
	}
	switch code {
	case codeBadValue:
		return "argument refused by the binding: wrong sample rate, channels, application, frame duration or frame length"
	case 0x103:
		return "no such codec state: the handle is stale or the codec is closed"
	case 0x101:
		return "out of memory (GC heap window and PSRAM exhausted for this allocation)"
	case 0x106, codeUnsupported:
		return "this build has no Opus codec (host build, or the firmware did not link the component)"
	}
	return ErrName(code)
}

// itoa formats v in decimal (negatives with a leading minus). No strconv:
// the family keeps pkg/esp import-free besides esperr.
func itoa(v int) string {
	neg := v < 0
	x := int32(v)
	if neg {
		x = -x
	}
	if x == 0 {
		return "0"
	}
	var buf [10]byte
	i := len(buf)
	for x > 0 {
		i--
		buf[i] = byte('0' + x%10)
		x /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// hex6 renders a code as six hex digits, the family's fallback format.
func hex6(v uint32) string {
	const hex = "0123456789abcdef"
	var out [6]byte
	for i := 5; i >= 0; i-- {
		out[i] = hex[v&0xF]
		v >>= 4
	}
	return string(out[:])
}
