//go:build !llgo

// Host build (go build / go vet / gopls): the codec exists only in firmware
// builds - an ESP-IDF link resolves the bindings against the project-side
// glue (opus_idf.go), a baremetal llgo build against the vendored sources
// (opus_baremetal.go) - so every binding here reports the family's
// unsupported sentinel. Nothing here talks to a codec; the Go layer turns
// the sentinel into ErrUnsupported exactly as it does for a firmware built
// without the codec component, and the closed-handle guards fire before
// any binding would run.
package opus

const errUnsupported = 0x7F00

func encOpen(sampleRate, channels, application int32, out *int32) int32 {
	_, _, _, _ = sampleRate, channels, application, out
	return errUnsupported
}

func encEncode(h int32, pcm *int16, frameSize int32, out *uint8, cap int32, outLen *int32) int32 {
	_, _, _, _, _, _ = h, pcm, frameSize, out, cap, outLen
	return errUnsupported
}

func encSetBitrate(h, bps int32) int32      { _, _ = h, bps; return errUnsupported }
func encSetComplexity(h, c int32) int32     { _, _ = h, c; return errUnsupported }
func encLookahead(h int32) int32            { _ = h; return 0 }
func encClose(h int32) int32                { _ = h; return errUnsupported }
func encSize(channels int32) int32          { _ = channels; return 0 }

func decOpen(sampleRate, channels int32, out *int32) int32 {
	_, _, _ = sampleRate, channels, out
	return errUnsupported
}

func decDecode(h int32, data *uint8, n int32, pcm *int16, frameCap int32, outSamples *int32) int32 {
	_, _, _, _, _, _ = h, data, n, pcm, frameCap, outSamples
	return errUnsupported
}

func decClose(h int32) int32       { _ = h; return errUnsupported }
func decSize(channels int32) int32 { _ = channels; return 0 }
