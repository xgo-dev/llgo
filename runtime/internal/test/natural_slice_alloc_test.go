//go:build llgo

package test

import (
	"testing"
	"unsafe"

	rt "github.com/xgo-dev/llgo/runtime/internal/runtime"
)

func TestNaturalMakeSlice(t *testing.T) {
	for _, hasPointers := range []bool{false, true} {
		for _, tc := range []struct{ length, capacity, size int }{
			{0, 0, 1}, {0, 9, 1}, {7, 17, 1}, {3, 9, 8}, {7, 17, 0},
		} {
			for _, aligned := range []bool{false, true} {
				var result rt.Slice
				if aligned {
					result = rt.MakeSliceAligned(tc.length, tc.capacity, tc.size, 1, hasPointers)
				} else {
					result = rt.MakeSlice(tc.length, tc.capacity, tc.size, hasPointers)
				}
				view := *(*[]byte)(unsafe.Pointer(&result))
				data := unsafe.SliceData(view)
				if len(view) != tc.length || cap(view) != tc.capacity || data == nil {
					t.Fatalf("pointers=%v aligned=%v case=%+v: len=%d cap=%d data=%p", hasPointers, aligned, tc, len(view), cap(view), data)
				}
				for _, b := range unsafe.Slice(data, tc.capacity*tc.size) {
					if b != 0 {
						t.Fatalf("pointers=%v aligned=%v case=%+v: nonzero capacity", hasPointers, aligned, tc)
					}
				}
			}
		}
		maxInt := int(^uint(0) >> 1)
		for _, tc := range []struct {
			length, capacity, size int
			want                   string
		}{
			{-1, -1, 1, "runtime error: makeslice: len out of range"},
			{2, 1, 1, "runtime error: makeslice: cap out of range"},
			{0, -1, 1, "runtime error: makeslice: cap out of range"},
			{maxInt, maxInt, 2, "runtime error: makeslice: len out of range"},
			{1, maxInt, 2, "runtime error: makeslice: cap out of range"},
		} {
			for _, aligned := range []bool{false, true} {
				func() {
					defer func() {
						got := recover()
						err, ok := got.(error)
						if !ok || err.Error() != tc.want {
							t.Errorf("pointers=%v aligned=%v case=%+v: panic=%v", hasPointers, aligned, tc, got)
						}
					}()
					if aligned {
						rt.MakeSliceAligned(tc.length, tc.capacity, tc.size, 1, hasPointers)
					} else {
						rt.MakeSlice(tc.length, tc.capacity, tc.size, hasPointers)
					}
				}()
			}
		}
	}
}

var naturalMakeSliceBenchmarkSink rt.Slice

func BenchmarkNaturalMakeSlice(b *testing.B) {
	for _, tc := range []struct {
		name string
		size int
	}{
		{"small", 4}, {"medium", 64}, {"page", 4096},
	} {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				naturalMakeSliceBenchmarkSink = rt.MakeSlice(tc.size, tc.size, 1, false)
			}
		})
	}
}
