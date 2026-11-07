//go:build llgo

package structs

import (
	"reflect"
	"testing"
	"unsafe"
)

type markerAlias = HostLayout
type markerDefined HostLayout

type layoutProbe struct {
	_     markerDefined
	Value uint64
	Tail  struct{}
}
type aliasProbe struct {
	_     markerAlias
	Value uint64
	Tail  struct{}
}

func TestHostLayoutCompatibility(t *testing.T) {
	const size = unsafe.Sizeof(layoutProbe{})
	// Keep the trailing zero-sized field inside the object.
	const alignment = unsafe.Alignof(layoutProbe{})
	const want = (8 + 1 + alignment - 1) &^ (alignment - 1)
	if size != want {
		t.Fatalf("source size=%d, want %d", size, want)
	}
	if got := reflect.TypeOf(layoutProbe{}).Size(); got != size {
		t.Fatalf("reflected size=%d, source=%d", got, size)
	}
	if got := unsafe.Sizeof([2]layoutProbe{}); got != 2*size {
		t.Fatalf("array size=%d, want %d", got, 2*size)
	}
	if got := unsafe.Offsetof(layoutProbe{}.Tail); got != 8 {
		t.Fatalf("tail offset=%d, want 8", got)
	}
	if got := unsafe.Sizeof(aliasProbe{}); got != size {
		t.Fatalf("alias size=%d, want %d", got, size)
	}
}
