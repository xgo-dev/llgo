package gotest

import (
	"reflect"
	"testing"
)

type reflectConvertNamedFunction func(int) int
type reflectConvertOtherFunction func(int) int
type reflectConvertVariadicFunction func(...int) int

func TestReflectFunctionConversionRejectsDifferentShapes(t *testing.T) {
	for _, test := range []struct {
		name   string
		source any
		target any
	}{
		{"parameter-type", (func(int) int)(nil), (func(string) int)(nil)},
		{"parameter-arity", (func(int) int)(nil), (func(int, int) int)(nil)},
		{"parameter-empty", (func(int) int)(nil), (func() int)(nil)},
		{"result-type", (func(int) int)(nil), (func(int) string)(nil)},
		{"result-arity", (func(int) int)(nil), (func(int) (int, int))(nil)},
		{"result-empty", (func(int) int)(nil), (func(int))(nil)},
		{"slice-to-variadic", (func([]int) int)(nil), (func(...int) int)(nil)},
		{"variadic-to-slice", (func(...int) int)(nil), (func([]int) int)(nil)},
	} {
		t.Run(test.name, func(t *testing.T) {
			value := reflect.ValueOf(test.source)
			target := reflect.TypeOf(test.target)
			if value.Type().ConvertibleTo(target) {
				t.Errorf("%v reports conversion to incompatible %v", value.Type(), target)
			}
			if value.CanConvert(target) {
				t.Errorf("Value.CanConvert accepts %v to incompatible %v", value.Type(), target)
			}
			if !reflectFunctionConversionPanics(value, target) {
				t.Errorf("Value.Convert did not reject %v to incompatible %v", value.Type(), target)
			}
		})
	}
}

func reflectFunctionConversionPanics(value reflect.Value, target reflect.Type) (panicked bool) {
	defer func() { panicked = recover() != nil }()
	_ = value.Convert(target)
	return false
}

func TestReflectFunctionConversionPreservesSameShape(t *testing.T) {
	twice := func(value int) int { return value * 2 }
	sum := func(values ...int) int {
		result := 0
		for _, value := range values {
			result += value
		}
		return result
	}
	for _, test := range []struct {
		name   string
		source any
		target any
		args   []reflect.Value
	}{
		{"named-to-unnamed", reflectConvertNamedFunction(twice), (func(int) int)(nil), []reflect.Value{reflect.ValueOf(21)}},
		{"unnamed-to-named", twice, reflectConvertNamedFunction(nil), []reflect.Value{reflect.ValueOf(21)}},
		{"named-to-named", reflectConvertNamedFunction(twice), reflectConvertOtherFunction(nil), []reflect.Value{reflect.ValueOf(21)}},
		{"named-variadic-to-unnamed", reflectConvertVariadicFunction(sum), (func(...int) int)(nil), []reflect.Value{reflect.ValueOf(10), reflect.ValueOf(32)}},
		{"unnamed-variadic-to-named", sum, reflectConvertVariadicFunction(nil), []reflect.Value{reflect.ValueOf(10), reflect.ValueOf(32)}},
	} {
		t.Run(test.name, func(t *testing.T) {
			value := reflect.ValueOf(test.source)
			target := reflect.TypeOf(test.target)
			if !value.Type().ConvertibleTo(target) || !value.CanConvert(target) {
				t.Fatalf("same-shape function conversion %v to %v was rejected", value.Type(), target)
			}
			converted := value.Convert(target)
			if converted.Type() != target {
				t.Fatalf("converted type = %v, want %v", converted.Type(), target)
			}
			result := converted.Call(test.args)
			if len(result) != 1 || result[0].Int() != 42 {
				t.Fatalf("converted function returned %v, want one result of 42", result)
			}
		})
	}
}
