//go:build llgo && baremetal && llgo.baremetal.stdio && llgo.size.embedded && goexperiment.jsonv2 && go1.26

package json

import (
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"reflect"
)

// LlgoDecodeTyped runs a compiler-generated decoder with the same legacy
// options and error conversion as Unmarshal.
func LlgoDecodeTyped[T any](data []byte, dst *T, decode func(*jsontext.Decoder, *T) error) error {
	if dst == nil {
		return &InvalidUnmarshalError{reflect.TypeOf(dst)}
	}
	err := jsonv2.LlgoReadTyped(data, dst, decode, DefaultOptionsV1())
	if err != nil {
		return transformUnmarshalError(dst, err)
	}
	return nil
}
