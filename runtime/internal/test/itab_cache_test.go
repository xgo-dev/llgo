//go:build llgo

package test

import (
	"testing"
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
	rt "github.com/xgo-dev/llgo/runtime/internal/runtime"
)

type itabCacheInterface interface {
	Probe() int
}

type itabCacheValue struct{}

func (itabCacheValue) Probe() int { return 42 }

// itabPrefix mirrors the ABI-visible prefix of runtime.Itab. It lets this
// package verify cache collision fallback without exposing runtime internals.
type itabPrefix struct {
	inter *abi.InterfaceType
	typ   *abi.Type
	hash  uint32
	_     uint32
	fun   [1]uintptr
}

func TestItabCacheCollisionFallback(t *testing.T) {
	typ := abi.TypeFor[itabCacheValue]()
	base := abi.TypeFor[itabCacheInterface]().InterfaceType()
	// Distinct descriptors with the same hash force a direct-mapped slot
	// collision while keeping all metadata referenced by heap objects.
	a := new(abi.InterfaceType)
	*a = *base
	b := new(abi.InterfaceType)
	*b = *base

	firstA := rt.NewItab(a, typ)
	firstB := rt.NewItab(b, typ)
	if firstA == nil || firstB == nil {
		t.Fatal("NewItab returned nil for an implemented interface")
	}
	for n := 0; n < 1000; n++ {
		for _, tc := range []struct {
			name string
			want *abi.InterfaceType
			got  *rt.Itab
		}{
			{"a", a, rt.NewItab(a, typ)},
			{"b", b, rt.NewItab(b, typ)},
		} {
			view := (*itabPrefix)(unsafe.Pointer(tc.got))
			if view.inter != tc.want || view.typ != typ || view.hash != typ.Hash || view.fun[0] == 0 {
				t.Fatalf("%s: corrupt itab after cache collision: inter=%p typ=%p hash=%x fun=%x", tc.name, view.inter, view.typ, view.hash, view.fun[0])
			}
		}
	}
}
