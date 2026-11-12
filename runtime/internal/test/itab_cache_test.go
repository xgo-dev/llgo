//go:build llgo

package test

import (
	"runtime"
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
	fun   [1]uintptr
}

func TestItabTableCollisionAndGrowth(t *testing.T) {
	if got, want := unsafe.Sizeof(itabPrefix{}), unsafe.Sizeof(rt.Itab{}); got != want {
		t.Fatalf("itab mirror size = %d, want %d", got, want)
	}
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
	if got := rt.NewItab(a, typ); got != firstA {
		t.Fatal("growth lost the original a itab")
	}
	if got := rt.NewItab(b, typ); got != firstB {
		t.Fatal("growth lost the original b itab")
	}
}

func TestItabTableConcurrentGrowth(t *testing.T) {
	typ := abi.TypeFor[itabCacheValue]()
	base := abi.TypeFor[itabCacheInterface]().InterfaceType()
	stable := new(abi.InterfaceType)
	*stable = *base
	stableItab := rt.NewItab(stable, typ)
	if stableItab == nil {
		t.Fatal("NewItab returned nil for the stable pair")
	}

	// Read an existing pair while workers insert the same fresh pairs. The
	// shared hash exercises collision chains across growth, while concurrent
	// conversions must all return the same canonical itab for each pair.
	const n, workers = 1024, 4
	interfaces := make([]*abi.InterfaceType, n)
	for i := range interfaces {
		interfaces[i] = new(abi.InterfaceType)
		*interfaces[i] = *base
	}
	results := make([][]*rt.Itab, workers)
	for i := range results {
		results[i] = make([]*rt.Itab, n)
	}
	start := make(chan struct{})
	done := make(chan struct{}, workers)
	for worker := 0; worker < workers; worker++ {
		go func(worker int) {
			defer func() { done <- struct{}{} }()
			<-start
			for i, inter := range interfaces {
				if got := rt.NewItab(stable, typ); got != stableItab {
					t.Errorf("worker %d, interface %d: stable itab changed during growth", worker, i)
					return
				}
				got := rt.NewItab(inter, typ)
				results[worker][i] = got
				if got == nil {
					t.Errorf("worker %d, interface %d: NewItab returned nil", worker, i)
					return
				}
				view := (*itabPrefix)(unsafe.Pointer(got))
				if view.inter != inter || view.typ != typ || view.hash != typ.Hash || view.fun[0] == 0 {
					t.Errorf("worker %d, interface %d: observed an incomplete or incorrect itab", worker, i)
					return
				}
				if i%16 == 0 {
					runtime.Gosched()
				}
			}
		}(worker)
	}
	close(start)
	for worker := 0; worker < workers; worker++ {
		<-done
	}
	if t.Failed() {
		return
	}
	for i, inter := range interfaces {
		want := rt.NewItab(inter, typ)
		for worker := 0; worker < workers; worker++ {
			if got := results[worker][i]; got != want {
				t.Fatalf("worker %d, interface %d: NewItab returned a noncanonical itab", worker, i)
			}
		}
	}
	if got := rt.NewItab(stable, typ); got != stableItab {
		t.Fatal("growth lost the stable itab")
	}
}
