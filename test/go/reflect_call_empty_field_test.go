package gotest

import (
	"reflect"
	"testing"
)

type reflectCallEmptyField struct {
	f1, f2 *byte
	empty  struct{}
}

func reflectCallEmptyFieldFunc(e reflectCallEmptyField, s []string) {
	if len(s) != 1 || s[0] != "hi" {
		panic("bad slice")
	}
}

func TestReflectCallStructWithEmptyField(t *testing.T) {
	reflect.ValueOf(reflectCallEmptyFieldFunc).Call([]reflect.Value{
		reflect.ValueOf(reflectCallEmptyField{}),
		reflect.ValueOf([]string{"hi"}),
	})
}
