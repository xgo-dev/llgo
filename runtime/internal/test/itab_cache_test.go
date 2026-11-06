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
// package inspect table results without exposing runtime internals.
type itabPrefix struct {
	inter *abi.InterfaceType
	typ   *abi.Type
	hash  uint32
	_     uint32
	fun   [1]uintptr
}

func TestItabTableCollisionAndGrowth(t *testing.T) {
	typ := abi.TypeFor[itabCacheValue]()
	base := abi.TypeFor[itabCacheInterface]().InterfaceType()
	// Distinct descriptors with the same hash exercise open-addressing
	// collision handling while keeping all metadata referenced by heap objects.
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
				t.Fatalf("%s: corrupt itab after hash collision: inter=%p typ=%p hash=%x fun=%x", tc.name, view.inter, view.typ, view.hash, view.fun[0])
			}
		}
	}

	// Force the table past its initial 75% load factor. All descriptors share
	// the same hash, so this also exercises rehashing under a long collision
	// chain rather than only the usual distributed-hash case.
	const n = 700
	interfaces := make([]*abi.InterfaceType, n)
	first := make([]*rt.Itab, n)
	for i := range interfaces {
		interfaces[i] = new(abi.InterfaceType)
		*interfaces[i] = *base
		first[i] = rt.NewItab(interfaces[i], typ)
		if first[i] == nil {
			t.Fatalf("interface %d: NewItab returned nil", i)
		}
	}
	for i, inter := range interfaces {
		if got := rt.NewItab(inter, typ); got != first[i] {
			t.Fatalf("interface %d: NewItab did not return canonical itab", i)
		}
	}
}
