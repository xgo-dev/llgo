//go:build llgo && !wasm

package cgo

import "testing"

func TestNarrowCConstantArguments(t *testing.T) {
	for _, call := range []struct {
		name string
		fn   func() (int32, uint32, int32, uint32)
	}{
		{"direct", directNarrowConstants},
		{"indirect", indirectNarrowConstants},
	} {
		t.Run(call.name, func(t *testing.T) {
			a, b, c, d := call.fn()
			if a != -8 || b != 250 || c != -300 || d != 60000 {
				t.Fatalf("got (%d, %d, %d, %d), want (-8, 250, -300, 60000)", a, b, c, d)
			}
		})
	}
}

func TestNarrowCArguments(t *testing.T) {
	for _, call := range []struct {
		name string
		fn   func(int8, uint8, int16, uint16) (int32, uint32, int32, uint32)
	}{
		{"direct", directNarrow},
		{"indirect", indirectNarrow},
	} {
		t.Run(call.name, func(t *testing.T) {
			for _, values := range []struct {
				a int8
				b uint8
				c int16
				d uint16
			}{
				{-8, 250, -300, 60000},
				{-128, 255, -32768, 65535},
				{-1, 128, -1, 32768},
				{127, 0, 32767, 0},
			} {
				a, b, c, d := call.fn(values.a, values.b, values.c, values.d)
				if a != int32(values.a) || b != uint32(values.b) || c != int32(values.c) || d != uint32(values.d) {
					t.Errorf("(%d, %d, %d, %d): got (%d, %d, %d, %d)", values.a, values.b, values.c, values.d, a, b, c, d)
				}
			}
		})
	}
}

func TestNarrowCStackArguments(t *testing.T) {
	for _, flag := range []bool{false, true} {
		want := int32(-8 + 10*250 + 100*(-300) + 1000*60000)
		if !flag {
			want = -want
		}
		if got := stackedNarrow(-8, 250, -300, 60000, flag); got != want {
			t.Errorf("flag=%t: got %d, want %d", flag, got, want)
		}
	}
}
