//go:build llgo && !baremetal && !nogc

package gotest

import (
	"runtime"
	"testing"
)

type nativeFinalizerObject struct {
	value     int
	finalized chan<- int
}

func (p *nativeFinalizerObject) finalize() { p.finalized <- p.value }

type nativeFinalizerInterface interface{ finalize() }

//llgo:cdecl
type nativePointerFinalizer func(*nativeFinalizerObject)

//llgo:cdecl
type nativeInterfaceFinalizer func(nativeFinalizerInterface)

//llgo:cdecl
type nativeResultFinalizer func(*nativeFinalizerObject) int64

//llgo:cdecl
type nativeAggregateFinalizer func(*nativeFinalizerObject) [64]int

//go:noinline
func registerBoxedFinalizer(obj, fn any) { runtime.SetFinalizer(obj, fn) }

func TestRuntimeSetFinalizerNative(t *testing.T) {
	for i, tc := range []struct {
		name string
		fn   any
	}{
		{"pointer", nativePointerFinalizer(func(p *nativeFinalizerObject) { p.finalize() })},
		{"interface", nativeInterfaceFinalizer(func(p nativeFinalizerInterface) { p.finalize() })},
		{"scalar result", nativeResultFinalizer(func(p *nativeFinalizerObject) int64 { p.finalize(); return 123 })},
		{"aggregate result", nativeAggregateFinalizer(func(p *nativeFinalizerObject) (r [64]int) { p.finalize(); r[63] = 123; return })},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Conservative stack roots may keep an individual allocation alive.
			// Exercise each native signature with independently collectible objects,
			// as in TestRuntimeSetFinalizerAssignableArgumentTypes.
			const registrations = 8
			want := 42 + i
			finalized := make(chan int, registrations)
			registerFinalizerForTest(func() {
				for range registrations {
					p := &nativeFinalizerObject{value: want, finalized: finalized}
					registerBoxedFinalizer(p, tc.fn)
				}
			})
			waitForFinalizerValue(t, finalized, want)
		})
	}
}

func TestRuntimeSetFinalizerNativeCancel(t *testing.T) {
	finalized := make(chan int, 1)
	registerFinalizerForTest(func() {
		p := &nativeFinalizerObject{value: 42, finalized: finalized}
		registerBoxedFinalizer(p, nativePointerFinalizer(func(p *nativeFinalizerObject) { p.finalize() }))
		runtime.SetFinalizer(p, nil)
		runtime.KeepAlive(p)
	})
	for range 3 {
		runGCWithTimeout(t)
	}
	select {
	case <-finalized:
		t.Fatal("canceled native finalizer ran")
	default:
	}
}

func TestRuntimeSetFinalizerDirectErrorResult(t *testing.T) {
	finalized := make(chan int, 1)
	registerFinalizerForTest(func() {
		p := &nativeFinalizerObject{value: 40, finalized: finalized}
		registerBoxedFinalizer(p, nativePointerFinalizer(func(p *nativeFinalizerObject) { p.finalize() }))
		runtime.SetFinalizer(p, func(p *nativeFinalizerObject) error {
			finalized <- p.value + 2
			return nil
		})
		runtime.KeepAlive(p)
	})
	waitForFinalizerValue(t, finalized, 42)
}
