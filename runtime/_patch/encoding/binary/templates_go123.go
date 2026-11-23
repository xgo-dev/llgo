//go:build llgo && llgo.size.binary && go1.23

package binary

//llgo:template llgoEncode
func Encode(buf []byte, order ByteOrder, data any) (int, error)

//llgo:template llgoDecode
func Decode(buf []byte, order ByteOrder, data any) (int, error)

func llgoEncode[O ByteOrder, T any](buf []byte, order O, v T) (int, error) {
	return Encode(buf, order, v)
}

func llgoDecode[O ByteOrder, T any](buf []byte, order O, v T) (int, error) {
	return Decode(buf, order, v)
}

// LLGoBinaryShortBuffer keeps the error object used by Encode and Decode.
func LLGoBinaryShortBuffer() error { return errBufferTooSmall }
