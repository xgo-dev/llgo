//go:build llgo

package json

import "strconv"

//llgo:template llgoMarshalBool llgoMarshalInt llgoMarshalUint llgoMarshalString
func Marshal(v any) ([]byte, error)

// Exact built-in types keep named types and their methods on the generic path.
type llgoSigned interface {
	int | int8 | int16 | int32 | int64
}

type llgoUnsigned interface {
	uint | uint8 | uint16 | uint32 | uint64 | uintptr
}

func llgoMarshalBool(v bool) ([]byte, error) {
	return strconv.AppendBool(nil, v), nil
}

func llgoMarshalInt[T llgoSigned](v T) ([]byte, error) {
	return strconv.AppendInt(nil, int64(v), 10), nil
}

func llgoMarshalUint[T llgoUnsigned](v T) ([]byte, error) {
	return strconv.AppendUint(nil, uint64(v), 10), nil
}
