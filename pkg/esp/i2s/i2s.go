// Package i2s drives ESP-IDF's new I2S driver in standard (Philips) mode
// from llgo-compiled Go code.
//
// It follows the pkg/esp package template (see the package wifi docs and
// pkg/esp/README.md). Scope: i2s_std.h only, 16/32-bit samples, mono or
// stereo. Every driver call here is synchronous and bounded by a transfer
// timeout, so there is no mailbox: each method maps to one glue call that
// returns the raw esp_err code (0 = ESP_OK).
//
// # Usage
//
//	ch, e := i2s.NewChannel(true)                      // TX; false = RX
//	e = ch.InitStd(-1, 4, 5, 6, -1, i2s.Bits16, true, 44100)
//	  // pins mclk,bclk,ws,dout,din (-1 = unused), 16-bit stereo, 44.1 kHz
//	e = ch.Enable()
//	n, e := ch.Write(samples)                          // len(samples) bytes
//	e = ch.Disable()
//	e = ch.Close()
//
// Buffers are byte slices; the sample format is the C driver's native
// little-endian, slot-interleaved for stereo. Handles are opaque C pointers
// carried in Go as uintptr integers, so the GC never scans them.
//
// # Glue integration
//
// Copy glue/i2s_glue.c into your project's main component (WHOLE_ARCHIVE
// required); see pkg/esp/README.md for the two integration recipes.
package i2s

// DefaultTimeoutMs bounds Write/Read transfers: the glue passes it to the
// driver, whose timeout_ms aborts with ESP_ERR_TIMEOUT instead of hanging
// the llgo scheduler forever.
const DefaultTimeoutMs = 1000

// ESP_ERR_INVALID_ARG: returned for Go-side misuse (nil or zero-value
// channel) before anything crosses into C.
const ErrInvalidArg = 0x102

// Bits selects the sample width (i2s_data_bit_width_t on the C side).
const (
	Bits16 uint32 = 16
	Bits32 uint32 = 32
)

// Pin identifies a GPIO by number (same numbering as the gpio package;
// each pkg/esp package is kept dependency-free, so this is a separate type).
type Pin int32

// Channel is one I2S TX or RX channel handle. Create it with NewChannel;
// the zero-value type is not usable.
type Channel struct {
	handle uintptr // opaque C i2s_chan_handle_t
}

// NewChannel allocates one channel on any free I2S port (DMA descriptor
// counts and the rest of the channel config stay in the C glue's
// I2S_CHANNEL_DEFAULT_CONFIG). tx = true allocates a transmit channel,
// false a receive channel. On error the esp_err code is returned and the
// Channel is nil.
func NewChannel(tx bool) (*Channel, int32) {
	var c Channel
	if e := newChannel(&c.handle, txBool(tx)); e != 0 {
		return nil, e
	}
	return &c, 0
}

func txBool(tx bool) int32 {
	if tx {
		return 1
	}
	return 0
}

// InitStd configures the channel for standard (Philips) format: the
// Philips slot defaults (I2S_STD_PHILIPS_SLOT_DEFAULT_CONFIG), the 256x
// MCLK multiple and the clock source stay in the C glue. Pins: mclk may be
// -1 when the DAC/codec needs no master clock; dout is TX-only, din
// RX-only - pass -1 for the one that is unused. bits is Bits16 or Bits32,
// stereo selects two interleaved slots over one (mono), sampleRateHz is
// the sample rate per slot.
func (c *Channel) InitStd(mclk, bclk, ws, dout, din Pin, bits uint32, stereo bool, sampleRateHz int32) int32 {
	if c == nil || c.handle == 0 {
		return ErrInvalidArg
	}
	mode := int32(0)
	if stereo {
		mode = 1
	}
	return initStd(c.handle, int32(mclk), int32(bclk), int32(ws),
		int32(dout), int32(din), int32(bits), mode, sampleRateHz)
}

// Enable starts the channel's DMA and clocks (i2s_channel_enable).
func (c *Channel) Enable() int32 {
	if c == nil || c.handle == 0 {
		return ErrInvalidArg
	}
	return enable(c.handle)
}

// Disable stops the channel (i2s_channel_disable).
func (c *Channel) Disable() int32 {
	if c == nil || c.handle == 0 {
		return ErrInvalidArg
	}
	return disable(c.handle)
}

// Write blocks until the samples are queued into the DMA ring or the
// DefaultTimeoutMs budget runs out; n reports how many bytes were accepted.
func (c *Channel) Write(p []byte) (int, int32) {
	if c == nil || c.handle == 0 {
		return 0, ErrInvalidArg
	}
	if len(p) == 0 {
		return 0, 0
	}
	var n int32
	if e := write(c.handle, &p[0], int32(len(p)), &n, DefaultTimeoutMs); e != 0 {
		return int(n), e
	}
	return int(n), 0
}

// Read blocks until len(p) bytes of samples arrived or the DefaultTimeoutMs
// budget runs out; n reports how many bytes were filled.
func (c *Channel) Read(p []byte) (int, int32) {
	if c == nil || c.handle == 0 {
		return 0, ErrInvalidArg
	}
	if len(p) == 0 {
		return 0, 0
	}
	var n int32
	if e := read(c.handle, &p[0], int32(len(p)), &n, DefaultTimeoutMs); e != 0 {
		return int(n), e
	}
	return int(n), 0
}

// Close frees the channel (i2s_del_channel). The channel must be disabled
// first, or the driver returns an error.
func (c *Channel) Close() int32 {
	if c == nil || c.handle == 0 {
		return ErrInvalidArg
	}
	e := del(c.handle)
	c.handle = 0
	return e
}
