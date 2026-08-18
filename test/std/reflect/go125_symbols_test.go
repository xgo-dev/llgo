//go:build go1.25

package reflect_test

import (
	"reflect"
	"testing"
)

func TestTypeAssert(t *testing.T) {
	value := reflect.ValueOf(42)
	if got, ok := reflect.TypeAssert[int](value); !ok || got != 42 {
		t.Fatalf("TypeAssert[int] = %d, %v; want 42, true", got, ok)
	}
	if got, ok := reflect.TypeAssert[string](value); ok || got != "" {
		t.Fatalf("TypeAssert[string] = %q, %v; want empty, false", got, ok)
	}
}
