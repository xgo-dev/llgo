package gotest

import (
	"reflect"
	"testing"
)

//llgo:stdcall
type deferredStdcallFunc func(*[]int32, int32)

//llgo:stdcall
type deferredStdcallNoArgs func()

//go:noinline
func recordStdcallLoopDefers(out *[]int32, count int) {
	first := deferredStdcallFunc(func(out *[]int32, n int32) { *out = append(*out, 100+n) })
	second := deferredStdcallFunc(func(out *[]int32, n int32) { *out = append(*out, 200+n) })
	for i := 0; i < count; i++ {
		fn := first
		if i%2 != 0 {
			fn = second
		}
		defer fn(out, int32(i))
	}
}

//go:noinline
func recordStdcallNoArgsDefers(count int) {
	first := deferredStdcallNoArgs(func() { nativeDeferOrder = nativeDeferOrder*10 + 1 })
	second := deferredStdcallNoArgs(func() { nativeDeferOrder = nativeDeferOrder*10 + 2 })
	for i := 0; i < count; i++ {
		fn := first
		if i%2 != 0 {
			fn = second
		}
		defer fn()
	}
}

func TestStdcallLoopDeferCapturesCallee(t *testing.T) {
	for _, tc := range []struct {
		count int
		want  []int32
	}{
		{0, nil}, {1, []int32{100}}, {4, []int32{203, 102, 201, 100}},
	} {
		var got []int32
		recordStdcallLoopDefers(&got, tc.count)
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("%d stdcall loop defers: got %v, want %v", tc.count, got, tc.want)
		}
	}
	nativeDeferOrder = 0
	recordStdcallNoArgsDefers(4)
	if nativeDeferOrder != 2121 {
		t.Fatalf("stdcall defer order = %d, want 2121", nativeDeferOrder)
	}
}
