package gotest

import (
	"fmt"
	"reflect"
	"testing"
)

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
