package gotest

import (
	"reflect"
	"testing"
)

//llgo:cdecl
type deferredNativeFunc func(*[]int32, int32)

//go:noinline
func recordNativeLoopDefers(out *[]int32, count int) {
	first := deferredNativeFunc(func(out *[]int32, n int32) {
		*out = append(*out, 100+n)
	})
	second := deferredNativeFunc(func(out *[]int32, n int32) {
		*out = append(*out, 200+n)
	})
	for i := 0; i < count; i++ {
		fn := first
		if i%2 != 0 {
			fn = second
		}
		defer fn(out, int32(i))
	}
}

func TestNativeLoopDeferCapturesCallee(t *testing.T) {
	for _, tc := range []struct {
		count int
		want  []int32
	}{
		{0, nil},
		{1, []int32{100}},
		{4, []int32{203, 102, 201, 100}},
	} {
		var got []int32
		recordNativeLoopDefers(&got, tc.count)
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("%d loop defers: got %v, want %v", tc.count, got, tc.want)
		}
	}
}

//llgo:cdecl
type deferredNativeNoArgs func()

var nativeDeferOrder int

//go:noinline
func recordNativeNoArgsDefers(count int) {
	first := deferredNativeNoArgs(func() { nativeDeferOrder = nativeDeferOrder*10 + 1 })
	second := deferredNativeNoArgs(func() { nativeDeferOrder = nativeDeferOrder*10 + 2 })
	for i := 0; i < count; i++ {
		fn := first
		if i%2 != 0 {
			fn = second
		}
		defer fn()
	}
}

func TestNativeLoopDeferWithoutArguments(t *testing.T) {
	nativeDeferOrder = 0
	recordNativeNoArgsDefers(4)
	if nativeDeferOrder != 2121 {
		t.Fatalf("defer order = %d, want 2121", nativeDeferOrder)
	}
}
