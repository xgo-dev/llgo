//go:build llgo

package gotest

import (
	"reflect"
	"testing"
)

//llgo:cdecl
type reflectFuncvalNativeEntry func(int) int

//llgo:cdecl
type reflectFuncvalNativeAlias func(int) int

type reflectFuncvalGoEntry func(int) int

func reflectFuncvalRoleTwice(n int) int { return n * 2 }

func TestReflectFunctionValueRoleConversion(t *testing.T) {
	var native reflectFuncvalNativeEntry = reflectFuncvalRoleTwice
	var goValue reflectFuncvalGoEntry = reflectFuncvalRoleTwice
	plain := reflect.TypeOf((func(int) int)(nil))
	for _, test := range []struct {
		name   string
		source reflect.Value
		target reflect.Type
	}{
		{"native-to-Go", reflect.ValueOf(native), plain},
		{"Go-to-native", reflect.ValueOf(goValue), reflect.TypeOf(native)},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.source.Type().ConvertibleTo(test.target) {
				t.Error("native entry and Go descriptor have different function-value roles")
			}
			if test.source.CanConvert(test.target) {
				t.Error("Value.CanConvert accepted an entry/descriptor reinterpretation")
			}
			defer func() {
				if recover() == nil {
					t.Error("Value.Convert must reject an entry/descriptor reinterpretation")
				}
			}()
			_ = test.source.Convert(test.target)
		})
	}

	t.Run("named-Go-to-unnamed-Go", func(t *testing.T) {
		converted := reflect.ValueOf(goValue).Convert(plain).Interface().(func(int) int)
		if got := converted(21); got != 42 {
			t.Fatalf("named Go conversion returned %d, want 42", got)
		}
	})
	t.Run("unnamed-Go-to-named-Go", func(t *testing.T) {
		converted := reflect.ValueOf(reflectFuncvalRoleTwice).Convert(reflect.TypeOf(goValue)).Interface().(reflectFuncvalGoEntry)
		if got := converted(21); got != 42 {
			t.Fatalf("unnamed Go conversion returned %d, want 42", got)
		}
	})
	t.Run("native-to-native", func(t *testing.T) {
		target := reflect.TypeOf(reflectFuncvalNativeAlias(nil))
		converted := reflect.ValueOf(native).Convert(target).Interface().(reflectFuncvalNativeAlias)
		if got := converted(21); got != 42 {
			t.Fatalf("native alias conversion returned %d, want 42", got)
		}
	})
}
