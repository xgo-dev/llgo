//go:build llgo.idf

// llgo.idf build: bodyless externs resolved by the ESP-IDF link against
// main/opus_glue.c - the same pattern as runtime/internal/runtime/
// idf_freertos.go (VTaskDelay). The symbols stay undefined in llgo_out.o by
// design; the project's main component provides the definitions at the
// final link. The glue, not this file, is where the opus.h ABI lives: every
// binding below exchanges only integers, handles and pointers.
package opus

import _ "unsafe" // for go:linkname

//go:linkname encOpen C.llgo_opus_enc_open
func encOpen(sampleRate, channels, application int32, out *int32) int32

//go:linkname encEncode C.llgo_opus_enc_encode
func encEncode(h int32, pcm *int16, frameSize int32, out *uint8, cap int32, outLen *int32) int32

//go:linkname encSetBitrate C.llgo_opus_enc_set_bitrate
func encSetBitrate(h, bps int32) int32

//go:linkname encSetComplexity C.llgo_opus_enc_set_complexity
func encSetComplexity(h, c int32) int32

//go:linkname encLookahead C.llgo_opus_enc_lookahead
func encLookahead(h int32) int32

//go:linkname encClose C.llgo_opus_enc_close
func encClose(h int32) int32

//go:linkname encSize C.llgo_opus_enc_size
func encSize(channels int32) int32

//go:linkname decOpen C.llgo_opus_dec_open
func decOpen(sampleRate, channels int32, out *int32) int32

//go:linkname decDecode C.llgo_opus_dec_decode
func decDecode(h int32, data *uint8, n int32, pcm *int16, frameCap int32, outSamples *int32) int32

//go:linkname decClose C.llgo_opus_dec_close
func decClose(h int32) int32

//go:linkname decSize C.llgo_opus_dec_size
func decSize(channels int32) int32
