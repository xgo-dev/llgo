//go:build goexperiment.simd && amd64

package portable

import (
	"os"
	"simd"
	"simd/archsimd"
	"strings"
	"testing"
)

func TestPortableRequestedWidth(t *testing.T) {
	// With no eligible hardware upstream selects emulation before processing
	// an explicit width. Otherwise every accepted request must select exactly it.
	if !archsimd.X86.AVX() && !archsimd.X86.AVX2() {
		return
	}
	request := ""
	for _, setting := range strings.Split(os.Getenv("GODEBUG"), ",") {
		if v, ok := strings.CutPrefix(setting, "simd="); ok {
			request = strings.TrimPrefix(v, "+")
		}
	}
	want := 0
	switch request {
	case "128":
		want = 128
	case "256":
		want = 256
	default:
		return
	}
	if simd.Emulated() || simd.VectorBitSize() != want || (simd.Uint32s{}).Len() != want/32 {
		t.Fatalf("requested %s: width=%d emulated=%t lanes=%d", request, simd.VectorBitSize(), simd.Emulated(), (simd.Uint32s{}).Len())
	}
}

func TestPortableArch256Conversion(t *testing.T) {
	if simd.Emulated() || simd.VectorBitSize() != 256 {
		return
	}
	input := [8]uint32{1, 3, 5, 7, 11, 13, 17, 19}
	v := simd.LoadUint32s(input[:])
	arch, ok := v.ToArch().(archsimd.Uint32x8)
	if !ok {
		t.Fatal("256-bit ToArch did not return Uint32x8")
	}
	result := simd.Uint32sFromArch(arch)
	var out [8]uint32
	result.Store(out[:])
	if out != input {
		t.Fatalf("architecture round trip = %v, want %v", out, input)
	}
}
