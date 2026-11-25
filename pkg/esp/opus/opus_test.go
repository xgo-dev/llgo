//go:build !llgo

// Host-build tests. They exercise what a host build can exercise: the stub
// path (no codec in this build), the argument validation the Go layer does
// before any binding runs, the closed-handle guards, the frame-size
// parameter table and the error spaces. Device behavior - the codec's
// actual quality and the C glue's handle table - is verified under esp-emu
// by the _demo/esp-idf-opus acceptance probe, not here.
package opus

import (
	"errors"
	"testing"

	"github.com/xgo-dev/llgo/pkg/esp/esperr"
)

// TestErrorSegments pins the segment plan the esperr family agreed on: this
// package's own conditions in 0x79xx (the lowest unclaimed segment - 0x76/77
// bt+A2DP, 0x78 reserved classic-audio, 0x7B ble, 0x7C esperr meta, 0x7D/7E
// ws/tls, 0x7F stubs), the stub tag above everything, and the negative Opus
// codes staying negative.
func TestErrorSegments(t *testing.T) {
	for _, c := range []struct {
		err  error
		code int32
	}{
		// The seven Opus failures keep the library's own negative codes -
		// the sentinel and a raw codec failure are one code space, so
		// errors.Is matches without wrapping.
		{ErrBadArg, -1},
		{ErrBufferTooSmall, -2},
		{ErrInternal, -3},
		{ErrInvalidPacket, -4},
		{ErrUnimplemented, -5},
		{ErrInvalidState, -6},
		{ErrAllocFail, -7},
		// The binding's own refusals live in the 0x79xx segment; the stub
		// tag sits above every real code space.
		{ErrBadValue, 0x7901},
		{ErrUnsupported, 0x7F00},
	} {
		if got := esperr.Code(c.err); got != c.code {
			t.Errorf("esperr.Code(%v) = %#x, want %#x", c.err, got, c.code)
		}
	}
	// A raw Opus error passes through with its sign preserved - that sign
	// is what separates an Opus failure from an esp_err of the same
	// magnitude (the tls package's negative-mbedtls convention).
	if got := esperr.Code(errOf(-4)); got != -4 {
		t.Errorf("esperr.Code(errOf(OPUS_INVALID_PACKET)) = %d, want -4", got)
	}
}

// TestErrNameTable checks the ErrName vocabulary: the negative Opus codes
// with their library names, the esp_err codes the glue can return, this
// package's own segment, and the family's hex fallback for anything else.
func TestErrNameTable(t *testing.T) {
	for _, c := range []struct {
		code int32
		want string
	}{
		{0, "OPUS_OK"},
		{-1, "OPUS_BAD_ARG"},
		{-2, "OPUS_BUFFER_TOO_SMALL"},
		{-3, "OPUS_INTERNAL_ERROR"},
		{-4, "OPUS_INVALID_PACKET"},
		{-5, "OPUS_UNIMPLEMENTED"},
		{-6, "OPUS_INVALID_STATE"},
		{-7, "OPUS_ALLOC_FAIL"},
		{-99, "opus error -99"},
		{0x101, "ESP_ERR_NO_MEM"},
		{0x103, "ESP_ERR_INVALID_STATE"},
		{0x7901, "OPUS_ERR_BAD_VALUE"},
		{0x7F00, "OPUS_ERR_UNSUPPORTED"},
		{0x555555, "esp error 0x555555"},
	} {
		if got := ErrName(c.code); got != c.want {
			t.Errorf("ErrName(%d) = %q, want %q", c.code, got, c.want)
		}
	}
}

// TestErrText checks the plain-language rendering: every negative Opus code
// and every Go-domain code renders a sentence, and the renderings differ
// (a table entry must not silently collapse into a neighbor's text).
func TestErrText(t *testing.T) {
	seen := make(map[string]int32)
	for code := int32(-7); code <= -1; code++ {
		text := ErrText(code)
		if text == "" || text == ErrName(code) && code >= -7 {
			t.Errorf("ErrText(%d) = %q, want a sentence", code, text)
		}
		if other, dup := seen[text]; dup {
			t.Errorf("ErrText(%d) and ErrText(%d) render the same text %q", other, code, text)
		}
		seen[text] = code
	}
	for _, code := range []int32{codeBadValue, codeUnsupported, 0x101, 0x103} {
		if text := ErrText(code); text == "" || text == ErrName(code) {
			t.Errorf("ErrText(%#x) = %q, want a sentence", code, text)
		}
	}
}

// TestOpusErrPassthrough pins errOf: 0 is nil, the negative Opus codes
// match their sentinels by code (errors.Is needs no wrapping step), and the
// stub tag renders ErrUnsupported rather than a raw-value error.
func TestOpusErrPassthrough(t *testing.T) {
	if err := errOf(0); err != nil {
		t.Fatalf("errOf(0) = %v, want nil", err)
	}
	for _, c := range []struct {
		code int32
		want error
	}{
		{-1, ErrBadArg},
		{-2, ErrBufferTooSmall},
		{-3, ErrInternal},
		{-4, ErrInvalidPacket},
		{-5, ErrUnimplemented},
		{-6, ErrInvalidState},
		{-7, ErrAllocFail},
	} {
		err := errOf(c.code)
		if !errors.Is(err, c.want) {
			t.Errorf("errOf(%d) does not match %v", c.code, c.want)
		}
		e, ok := err.(*esperr.Error)
		if !ok || e.Name != ErrName(c.code) {
			t.Errorf("errOf(%d) = %v, want a %q-named esperr", c.code, err, ErrName(c.code))
		}
	}
	if err := errOf(0x7F00); !errors.Is(err, ErrUnsupported) {
		t.Errorf("errOf(stub tag) = %v, want ErrUnsupported", err)
	}
}

// TestStubReportsUnsupported exercises the host stub's honesty: the
// constructors refuse, and the state-size reporters answer 0 (no glue, no
// invented numbers).
func TestStubReportsUnsupported(t *testing.T) {
	if _, err := NewEncoder(48000, 1, AppAudio); !errors.Is(err, ErrUnsupported) {
		t.Errorf("NewEncoder = %v, want ErrUnsupported", err)
	}
	if _, err := NewDecoder(48000, 1); !errors.Is(err, ErrUnsupported) {
		t.Errorf("NewDecoder = %v, want ErrUnsupported", err)
	}
	if got := EncoderStateBytes(1); got != 0 {
		t.Errorf("EncoderStateBytes = %d on a host build, want 0", got)
	}
	if got := DecoderStateBytes(2); got != 0 {
		t.Errorf("DecoderStateBytes = %d on a host build, want 0", got)
	}
}

// TestValidation pins the argument validation the constructors do before
// any binding would run: the Opus sample-rate set, the channel range, the
// application values.
func TestValidation(t *testing.T) {
	for _, r := range []int{8000, 12000, 16000, 24000, 48000} {
		if !validRate(r) {
			t.Errorf("validRate(%d) = false, want true", r)
		}
	}
	for _, r := range []int{0, 4000, 11025, 22050, 32000, 44100, 96000, -48000} {
		if validRate(r) {
			t.Errorf("validRate(%d) = true, want false", r)
		}
	}
	// A refused argument must fail before the stub is even reached - the
	// error is ErrBadValue, never the stub's ErrUnsupported.
	if _, err := NewEncoder(44100, 1, AppAudio); !errors.Is(err, ErrBadValue) {
		t.Errorf("NewEncoder(44100 Hz) = %v, want ErrBadValue", err)
	}
	if _, err := NewEncoder(48000, 3, AppAudio); !errors.Is(err, ErrBadValue) {
		t.Errorf("NewEncoder(3 ch) = %v, want ErrBadValue", err)
	}
	if _, err := NewEncoder(48000, 0, AppAudio); !errors.Is(err, ErrBadValue) {
		t.Errorf("NewEncoder(0 ch) = %v, want ErrBadValue", err)
	}
	if _, err := NewEncoder(48000, 1, 2049+1); !errors.Is(err, ErrBadValue) {
		t.Errorf("NewEncoder(bad application) = %v, want ErrBadValue", err)
	}
	if _, err := NewDecoder(44100, 1); !errors.Is(err, ErrBadValue) {
		t.Errorf("NewDecoder(44100 Hz) = %v, want ErrBadValue", err)
	}
}

// TestFrameSizeTable pins the samples-per-channel table Encode consumes -
// every legal (rate, ms) pair divides exactly, and the mapping matches the
// Opus frame-duration set the package spells. This is the table a caller
// sizes its pcm buffers by.
func TestFrameSizeTable(t *testing.T) {
	for _, c := range []struct {
		rate, ms, want int
	}{
		{8000, 5, 40}, {8000, 60, 480},
		{12000, 10, 120}, {12000, 20, 240},
		{16000, 5, 80}, {16000, 20, 320}, {16000, 60, 960},
		{24000, 10, 240}, {24000, 40, 960},
		{48000, 5, 240}, {48000, 10, 480}, {48000, 20, 960},
		{48000, 40, 1920}, {48000, 60, 2880},
	} {
		n, ok := frameSamples(c.rate, c.ms)
		if !ok || n != c.want {
			t.Errorf("frameSamples(%d, %d) = %d,%v; want %d,true", c.rate, c.ms, n, ok, c.want)
		}
	}
	// 2.5 ms is a legal Opus duration but not spellable through the integer
	// millisecond interface; anything not in the set is refused.
	for _, ms := range []int{0, 1, 2, 3, 4, 15, 25, 80, -20} {
		if _, ok := frameSamples(48000, ms); ok {
			t.Errorf("frameSamples(48000, %d) accepted, want refusal", ms)
		}
	}
	if maxFrameSamples != 2880 { // 48000 * 60 ms
		t.Errorf("maxFrameSamples = %d, want 2880", maxFrameSamples)
	}
	if MaxPacketBytes != 1276 { // the one-frame wire maximum, 1275, +1
		t.Errorf("MaxPacketBytes = %d, want 1276", MaxPacketBytes)
	}
}

// TestClosedGuards exercises the closed-handle discipline on a host build:
// a zero-value codec reports the family's closed sentinel for every
// operation (the guards run before the stub bindings), Close is idempotent,
// and the constructor-shaped fields are set only through NewEncoder - the
// zero value never masquerades as a live codec.
func TestClosedGuards(t *testing.T) {
	var enc *Encoder
	if _, err := enc.Encode(make([]int16, 960)); !errors.Is(err, esperr.ErrClosed) {
		t.Errorf("nil encoder Encode = %v, want esperr.ErrClosed", err)
	}
	if err := enc.SetBitrate(32000); !errors.Is(err, esperr.ErrClosed) {
		t.Errorf("nil encoder SetBitrate = %v, want esperr.ErrClosed", err)
	}
	if err := enc.Close(); !errors.Is(err, esperr.ErrClosed) {
		t.Errorf("nil encoder Close = %v, want esperr.ErrClosed", err)
	}

	e := &Encoder{handle: 0, frameSize: 960, channels: 1}
	if _, err := e.Encode(make([]int16, 960)); !errors.Is(err, esperr.ErrClosed) {
		t.Errorf("zero-handle Encode = %v, want esperr.ErrClosed", err)
	}
	if err := e.Close(); !errors.Is(err, esperr.ErrClosed) {
		t.Errorf("zero-handle Close = %v, want esperr.ErrClosed", err)
	}

	var dec *Decoder
	if _, err := dec.Decode(nil, make([]int16, 960)); !errors.Is(err, esperr.ErrClosed) {
		t.Errorf("nil decoder Decode = %v, want esperr.ErrClosed", err)
	}

	d := &Decoder{handle: 0, channels: 1}
	if _, err := d.Decode(nil, make([]int16, 960)); !errors.Is(err, esperr.ErrClosed) {
		t.Errorf("zero-handle Decode = %v, want esperr.ErrClosed", err)
	}
}

// TestEncodeLengthGuard pins the one-frame-exactly contract before any
// binding would run: a live-looking encoder with a wrong pcm length is
// refused with ErrBadValue, and the check uses the configured frame size
// and channel count (mono 960 vs stereo 1920 at 48 kHz / 20 ms).
func TestEncodeLengthGuard(t *testing.T) {
	// handle must be nonzero to pass the closed guard; the stub never runs
	// because the length check fires first.
	e := &Encoder{handle: 1, sampleRate: 48000, channels: 1, frameSize: 960,
		out: make([]byte, MaxPacketBytes)}
	if _, err := e.Encode(make([]int16, 959)); !errors.Is(err, ErrBadValue) {
		t.Errorf("Encode(959 samples) = %v, want ErrBadValue", err)
	}
	if _, err := e.Encode(make([]int16, 1920)); !errors.Is(err, ErrBadValue) {
		t.Errorf("Encode(1920 samples, mono) = %v, want ErrBadValue", err)
	}
	st := &Encoder{handle: 1, sampleRate: 48000, channels: 2, frameSize: 960,
		out: make([]byte, MaxPacketBytes)}
	if _, err := st.Encode(make([]int16, 1920)); !errors.Is(err, ErrUnsupported) {
		t.Errorf("Encode(stereo, correct length) = %v, want the stub's ErrUnsupported", err)
	}
}

// TestDecodeSizingGuard pins the pcmOut sizing contract: a buffer that
// cannot hold one sample per channel is refused with ErrBadValue before
// the binding runs; a buffer that can hold at least one sample per channel
// reaches the stub (ErrUnsupported on a host), because only the packet
// knows its true duration.
func TestDecodeSizingGuard(t *testing.T) {
	d := &Decoder{handle: 1, sampleRate: 48000, channels: 2}
	if _, err := d.Decode([]byte{1, 2, 3}, make([]int16, 1)); !errors.Is(err, ErrBadValue) {
		t.Errorf("Decode(1 int16, stereo) = %v, want ErrBadValue", err)
	}
	if _, err := d.Decode([]byte{1, 2, 3}, make([]int16, 2)); !errors.Is(err, ErrUnsupported) {
		t.Errorf("Decode(2 int16, stereo) = %v, want the stub's ErrUnsupported", err)
	}
	// An empty pcmOut slice is refused (len/channels == 0), nil frame with a
	// real buffer reaches the stub as the concealment request.
	if _, err := d.Decode(nil, nil); !errors.Is(err, ErrBadValue) {
		t.Errorf("Decode(nil pcmOut) = %v, want ErrBadValue", err)
	}
}

// TestApplicationValues pins the wire constants against the opus_defines.h
// values the glue passes through: the Go layer must not renumber them.
func TestApplicationValues(t *testing.T) {
	if AppVoip != 2048 || AppAudio != 2049 || AppLowDelay != 2051 {
		t.Errorf("application constants drifted: %d/%d/%d, want 2048/2049/2051",
			AppVoip, AppAudio, AppLowDelay)
	}
	if MinBitrate != 500 || MaxBitrate != 512000 {
		t.Errorf("bitrate bounds drifted: %d..%d, want 500..512000", MinBitrate, MaxBitrate)
	}
	if BitrateAuto != -1000 || BitrateMax != -1 {
		t.Errorf("bitrate special values drifted: %d/%d, want -1000/-1",
			BitrateAuto, BitrateMax)
	}
}
