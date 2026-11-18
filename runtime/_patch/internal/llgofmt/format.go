//go:build llgo && wasip1 && wasm && llgo.fmt.value

// Package llgofmt formats values whose default formatting is determined by
// their static shape. The compiler checks the shape before calling this package.
package llgofmt

import (
	"internal/fmtsort"
	"reflect"
	"strconv"
)

func Value(v any) string {
	return string(appendValue(nil, reflect.ValueOf(v), true, nil))
}

func ValueWithMaps(v any) string {
	return string(appendValue(nil, reflect.ValueOf(v), true, appendMap))
}

type mapAppender func([]byte, reflect.Value) []byte

func appendValue(dst []byte, v reflect.Value, top bool, maps mapAppender) []byte {
	switch v.Kind() {
	case reflect.Bool:
		return strconv.AppendBool(dst, v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.AppendInt(dst, v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.AppendUint(dst, v.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		bits := 64
		if v.Kind() == reflect.Float32 {
			bits = 32
		}
		return strconv.AppendFloat(dst, v.Float(), 'g', -1, bits)
	case reflect.Complex64, reflect.Complex128:
		bits := 128
		if v.Kind() == reflect.Complex64 {
			bits = 64
		}
		return append(dst, strconv.FormatComplex(v.Complex(), 'g', -1, bits)...)
	case reflect.String:
		return append(dst, v.String()...)
	case reflect.Array, reflect.Slice:
		dst = append(dst, '[')
		for i := 0; i < v.Len(); i++ {
			if i != 0 {
				dst = append(dst, ' ')
			}
			dst = appendValue(dst, v.Index(i), false, maps)
		}
		return append(dst, ']')
	case reflect.Struct:
		dst = append(dst, '{')
		for i := 0; i < v.NumField(); i++ {
			if i != 0 {
				dst = append(dst, ' ')
			}
			dst = appendValue(dst, v.Field(i), false, maps)
		}
		return append(dst, '}')
	case reflect.Map:
		return maps(dst, v)
	case reflect.Pointer:
		if top && v.UnsafePointer() != nil {
			elem := v.Elem()
			switch elem.Kind() {
			case reflect.Array, reflect.Slice, reflect.Struct, reflect.Map:
				return appendValue(append(dst, '&'), elem, false, maps)
			}
		}
		fallthrough
	case reflect.UnsafePointer, reflect.Chan, reflect.Func:
		if p := v.UnsafePointer(); p != nil {
			return strconv.AppendUint(append(dst, "0x"...), uint64(uintptr(p)), 16)
		}
		return append(dst, "<nil>"...)
	default:
		panic("llgofmt: unsupported value")
	}
}

func appendMap(dst []byte, v reflect.Value) []byte {
	dst = append(dst, "map["...)
	for i, pair := range fmtsort.Sort(v) {
		if i != 0 {
			dst = append(dst, ' ')
		}
		dst = appendValue(dst, pair.Key, false, appendMap)
		dst = append(dst, ':')
		dst = appendValue(dst, pair.Value, false, appendMap)
	}
	return append(dst, ']')
}
