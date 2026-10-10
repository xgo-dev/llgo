//go:build llgo

package llgoext

import "testing"

func TestLinknameNarrowCArguments(t *testing.T) {
	want := narrowValues{-8, 250, -300, 60000}
	if got := linkedNarrow(-8, 250, -300, 60000); got != want {
		t.Errorf("direct: got %+v, want %+v", got, want)
	}
	for _, call := range []struct {
		name string
		fn   narrowFunction
	}{
		{"converted-linkname", narrowFunction(linkedNarrow)},
		{"returned-C-pointer", linkedNarrowAddress()},
	} {
		t.Run(call.name, func(t *testing.T) {
			if got := call.fn(-8, 250, -300, 60000); got != want {
				t.Fatalf("got %+v, want %+v", got, want)
			}
			want := narrowValues{-128, 255, -32768, 65535}
			if got := call.fn(-128, 255, -32768, 65535); got != want {
				t.Fatalf("limits: got %+v, want %+v", got, want)
			}
		})
	}
}

func TestLinknameNarrowCStackArguments(t *testing.T) {
	for _, flag := range []bool{false, true} {
		want := narrowValues{-8, 250, -300, 60000}
		if !flag {
			want.s8 = 8
		}
		if got := linkedNarrowStack(1, 2, 3, 4, 5, 6, 7, 8, -8, 250, -300, 60000, flag); got != want {
			t.Errorf("direct, flag=%t: got %+v, want %+v", flag, got, want)
		}
		for _, call := range []struct {
			name string
			fn   narrowStackFunction
		}{
			{"converted-linkname", narrowStackFunction(linkedNarrowStack)},
			{"returned-C-pointer", linkedNarrowStackAddress()},
		} {
			if got := call.fn(1, 2, 3, 4, 5, 6, 7, 8, -8, 250, -300, 60000, flag); got != want {
				t.Errorf("%s, flag=%t: got %+v, want %+v", call.name, flag, got, want)
			}
		}
	}
}

func goNarrowArguments(a int8, b uint8, c int16, d uint16, flag bool) int32 {
	if a != -8 || b != 250 || c != -300 || d != 60000 {
		return -1
	}
	if flag {
		return 42
	}
	return 43
}

func TestLinknameNarrowCCallbackArguments(t *testing.T) {
	for _, flag := range []bool{false, true} {
		want := int32(43)
		if flag {
			want = 42
		}
		if got := linkedNarrowInvoke(goNarrowArguments, flag); got != want {
			t.Errorf("flag=%t: got %d, want %d", flag, got, want)
		}
	}
}

func TestLinknameNarrowCVariadicArguments(t *testing.T) {
	// Supply the C-promoted tail types; this regression checks the narrow fixed
	// parameters of the ellipsis prototype, not automatic tail conversion.
	if got := linkedNarrowVariadic(-8, -300, true, 17, int32(-128), float64(1.5)); got != 42 {
		t.Errorf("direct: got %d, want 42", got)
	}
	for _, call := range []struct {
		name string
		fn   narrowVariadicFunction
	}{
		{"converted-linkname", narrowVariadicFunction(linkedNarrowVariadic)},
		{"returned-C-pointer", linkedNarrowVariadicAddress()},
	} {
		if got := call.fn(-8, -300, true, 17, int32(-128), float64(1.5)); got != 42 {
			t.Errorf("%s: got %d, want 42", call.name, got)
		}
	}
}

func goNarrowS8(x int32) int8    { return int8(x) }
func goNarrowU8(x int32) uint8   { return uint8(x) }
func goNarrowS16(x int32) int16  { return int16(x) }
func goNarrowU16(x int32) uint16 { return uint16(x) }

func TestLinknameNarrowCReturns(t *testing.T) {
	if a, b, c, d := int32(linkedNarrowReturnS8(248)), uint32(linkedNarrowReturnU8(506)),
		int32(linkedNarrowReturnS16(65236)), uint32(linkedNarrowReturnU16(125536)); a != -8 || b != 250 || c != -300 || d != 60000 {
		t.Errorf("direct: got (%d, %d, %d, %d), want (-8, 250, -300, 60000)", a, b, c, d)
	}
	s8 := narrowS8Callback(linkedNarrowReturnS8)
	u8 := narrowU8Callback(linkedNarrowReturnU8)
	s16 := narrowS16Callback(linkedNarrowReturnS16)
	u16 := narrowU16Callback(linkedNarrowReturnU16)
	if a, b, c, d := int32(s8(248)), uint32(u8(506)), int32(s16(65236)), uint32(u16(125536)); a != -8 || b != 250 || c != -300 || d != 60000 {
		t.Errorf("indirect: got (%d, %d, %d, %d), want (-8, 250, -300, 60000)", a, b, c, d)
	}
}

func roundtripGoNarrowS8(fn func(int32) int8) narrowS8Callback {
	return linkedNarrowS8Roundtrip(narrowS8Callback(fn))
}

func TestLinknameNarrowCCallbacks(t *testing.T) {
	// A Go entry converted to a C function pointer still uses the Go return
	// convention. Returning it through C must not make the caller assume that
	// the entry has already sign- or zero-extended a narrow result.
	s8 := linkedNarrowS8Roundtrip(goNarrowS8)
	u8 := linkedNarrowU8Roundtrip(goNarrowU8)
	s16 := linkedNarrowS16Roundtrip(goNarrowS16)
	u16 := linkedNarrowU16Roundtrip(goNarrowU16)
	if got := int32(s8(248)); got != -8 {
		t.Errorf("int8 callback: got %d, want -8", got)
	}
	if got := uint32(u8(506)); got != 250 {
		t.Errorf("uint8 callback: got %d, want 250", got)
	}
	if got := int32(s16(65236)); got != -300 {
		t.Errorf("int16 callback: got %d, want -300", got)
	}
	if got := uint32(u16(125536)); got != 60000 {
		t.Errorf("uint16 callback: got %d, want 60000", got)
	}
	if got := int32(roundtripGoNarrowS8(goNarrowS8)(248)); got != -8 {
		t.Errorf("Go func value callback: got %d, want -8", got)
	}
}
