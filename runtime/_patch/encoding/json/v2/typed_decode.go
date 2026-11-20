//go:build llgo && baremetal && llgo.baremetal.stdio && llgo.size.embedded && goexperiment.jsonv2 && go1.26

package json

import (
	"encoding/json/internal/jsonwire"
	"encoding/json/jsontext"
	"errors"
	"io"
	"reflect"
	"strconv"
	"unsafe"
)

func LlgoReadTyped[T any](data []byte, dst *T, decode func(*jsontext.Decoder, *T) error, options Options) error {
	dec := export.GetBufferedDecoder(data, options)
	defer export.PutBufferedDecoder(dec)
	state := export.Decoder(dec)
	// Validate before mutating the destination, including trailing input.
	if err := state.CheckNextValue(true); err != nil {
		if err == io.EOF {
			return &jsontext.SyntacticError{ByteOffset: int64(len(data)), Err: io.ErrUnexpectedEOF}
		}
		return err
	}
	if err := decode(dec, dst); err != nil {
		return err
	}
	return state.CheckEOF()
}

func LlgoReadBool[T ~bool](dec *jsontext.Decoder, dst *T) error {
	tok, err := dec.ReadToken()
	if err != nil {
		return err
	}
	switch tok.Kind() {
	case 'n':
		return nil
	case 't', 'f':
		*dst = T(tok.Bool())
		return nil
	}
	return LlgoBadToken(dec, dst)
}

func LlgoReadString[T ~string](dec *jsontext.Decoder, dst *T) error {
	var flags jsonwire.ValueFlags
	v, err := export.Decoder(dec).ReadValue(&flags)
	if err != nil {
		return err
	}
	switch v.Kind() {
	case 'n':
		return nil
	case '"':
		*dst = T(jsonwire.UnquoteMayCopy(v, flags.IsVerbatim()))
		return nil
	}
	return newUnmarshalErrorAfter(dec, reflect.TypeFor[T](), nil)
}

func LlgoReadInt[T ~int | ~int8 | ~int16 | ~int32 | ~int64](dec *jsontext.Decoder, dst *T) error {
	v, err := dec.ReadValue()
	if err != nil {
		return err
	}
	switch v.Kind() {
	case 'n':
		return nil
	case '0':
		n, err := strconv.ParseInt(string(v), 10, int(8*unsafe.Sizeof(*dst)))
		if err != nil {
			return newUnmarshalErrorAfterWithValue(dec, reflect.TypeFor[T](), errors.Unwrap(err))
		}
		*dst = T(n)
		return nil
	}
	return newUnmarshalErrorAfter(dec, reflect.TypeFor[T](), nil)
}

func LlgoReadUint[T ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr](dec *jsontext.Decoder, dst *T) error {
	v, err := dec.ReadValue()
	if err != nil {
		return err
	}
	switch v.Kind() {
	case 'n':
		return nil
	case '0':
		n, err := strconv.ParseUint(string(v), 10, int(8*unsafe.Sizeof(*dst)))
		if err != nil {
			return newUnmarshalErrorAfterWithValue(dec, reflect.TypeFor[T](), errors.Unwrap(err))
		}
		*dst = T(n)
		return nil
	}
	return newUnmarshalErrorAfter(dec, reflect.TypeFor[T](), nil)
}

func LlgoBadToken[T any](dec *jsontext.Decoder, dst *T) error {
	return newUnmarshalErrorAfterWithSkipping(dec, reflect.TypeFor[T](), nil)
}

func LlgoRecoverable(err error) bool {
	_, ok := err.(*SemanticError)
	return ok
}
