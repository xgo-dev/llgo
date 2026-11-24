package gotest

import (
	"fmt"
	"reflect"
	"testing"
)

//llgo:cdecl
type reflectGenericIntFormatter func(int) string

func reflectGenericFormatter[T any](v T) string {
	return fmt.Sprintf("value: %v", v)
}

func TestReflectGenericNamedFuncChannelSend(t *testing.T) {
	ch := make(chan reflectGenericIntFormatter, 1)
	var fn reflectGenericIntFormatter = reflectGenericFormatter
	reflect.ValueOf(ch).Send(reflect.ValueOf(fn))
	if got := (<-ch)(14); got != "value: 14" {
		t.Fatalf("received formatter result = %q, want %q", got, "value: 14")
	}
}

func TestReflectNativeFuncStructField(t *testing.T) {
	type record struct {
		F    reflectGenericIntFormatter
		Tail int
	}
	item := record{Tail: 99}
	var fn reflectGenericIntFormatter = reflectGenericFormatter
	field := reflect.ValueOf(&item).Elem().Field(0)
	field.Set(reflect.ValueOf(fn))
	if got := field.Addr().Elem().Interface().(reflectGenericIntFormatter)(14); got != "value: 14" {
		t.Fatalf("reflected callback returned %q", got)
	}
	if item.Tail != 99 {
		t.Fatalf("setting the callback changed the following field to %d", item.Tail)
	}
	dynamic := reflect.StructOf([]reflect.StructField{
		{Name: "F", Type: reflect.TypeOf(fn)},
		{Name: "Tail", Type: reflect.TypeOf(0)},
	})
	if dynamic.Size() != reflect.TypeOf(item).Size() || dynamic.Field(1).Offset != reflect.TypeOf(item).Field(1).Offset {
		t.Fatalf("StructOf changed the callback's storage layout: %v", dynamic)
	}
}
