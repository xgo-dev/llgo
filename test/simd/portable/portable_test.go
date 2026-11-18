//go:build goexperiment.simd && (amd64 || arm64 || wasm)

package portable

import (
	"os"
	"simd"
	"simd/archsimd"
	"testing"

	"github.com/xgo-dev/llgo/test/simd/portable/internal/vectorops"
)

func TestPortableSelection(t *testing.T) {
	if os.Getenv("GODEBUG") == "simd=0" && !simd.Emulated() {
		t.Fatal("explicit emulation request selected hardware vectors")
	}
}

func TestPortableFamilies(t *testing.T) {
	lanes := []int{(simd.Int8s{}).Len(), (simd.Uint8s{}).Len(), (simd.Int16s{}).Len(), (simd.Uint16s{}).Len(), (simd.Int32s{}).Len(), (simd.Uint32s{}).Len(), (simd.Int64s{}).Len(), (simd.Uint64s{}).Len(), (simd.Float32s{}).Len(), (simd.Float64s{}).Len()}
	bits := []int{8, 8, 16, 16, 32, 32, 64, 64, 32, 64}
	for i, n := range lanes {
		if n*bits[i] != simd.VectorBitSize() {
			t.Fatalf("family %d has %d lanes", i, n)
		}
	}
	lanes32 := (simd.Uint32s{}).Len()
	for n := 0; n <= lanes32; n++ {
		xs := make([]uint32, n)
		for i := range xs {
			xs[i] = uint32(7 + 2*i)
		}
		v, loaded := simd.LoadUint32sPart(xs)
		if loaded != n {
			t.Fatalf("loaded %d, want %d", loaded, n)
		}
		v = v.IfElse(v.Greater(simd.BroadcastUint32s(8)), simd.BroadcastUint32s(99))
		out := make([]uint32, n+1)
		out[n] = 0xfeed
		if stored := v.StorePart(out[:n]); stored != n {
			t.Fatalf("stored %d, want %d", stored, n)
		}
		for i := range xs {
			want := xs[i]
			if want <= 8 {
				want = 99
			}
			if out[i] != want {
				t.Fatalf("part[%d]=%d want %d", i, out[i], want)
			}
		}
		if out[n] != 0xfeed {
			t.Fatal("partial store overwrote guard")
		}
	}
	signed := make([]int8, (simd.Int8s{}).Len())
	simd.BroadcastInt8s(120).Add(simd.BroadcastInt8s(20)).Store(signed)
	for _, v := range signed {
		if v != -116 {
			t.Fatalf("signed wrap = %d", v)
		}
	}
	floats := make([]float32, (simd.Float32s{}).Len())
	simd.BroadcastFloat32s(1.5).Mul(simd.BroadcastFloat32s(2)).Add(simd.BroadcastFloat32s(0.25)).Store(floats)
	for _, v := range floats {
		if v != 3.25 {
			t.Fatalf("float = %v", v)
		}
	}
}

func TestPortableArchConversion(t *testing.T) {
	// Emulated ToArch retains the software representation. FromArch accepts
	// architecture vectors, so exercise that round trip in hardware mode.
	x := simd.BroadcastUint32s(21)
	arch := x.ToArch()
	if simd.Emulated() {
		if _, ok := arch.(simd.Uint32s); !ok {
			t.Fatalf("emulated ToArch lost its portable representation: %T", arch)
		}
		return
	}
	// The amd64-specific test covers the 256-bit architecture round trip.
	if simd.VectorBitSize() != 128 {
		return
	}
	y := simd.Uint32sFromArch(arch.(archsimd.Uint32x4))
	var out [4]uint32
	y.Store(out[:])
	for _, v := range out {
		if v != 21 {
			t.Fatalf("arch round trip = %d", v)
		}
	}
}

func TestPortableArithmetic(t *testing.T) {
	for n := 0; n < 97; n++ {
		xs := make([]uint32, n)
		var want uint32
		for i := range xs {
			xs[i] = uint32(i*i + 3)
			want += xs[i]
		}
		var reducer vectorops.Reducer
		if got := reducer.Reduce(xs); got != want {
			t.Fatalf("length %d: got %d, want %d", n, got, want)
		}
	}
	t.Logf("portable selection: width=%d emulated=%t", simd.VectorBitSize(), simd.Emulated())
}

func TestPortableCalls(t *testing.T) {
	b := vectorops.NewBlock(3)
	if got := b.Sum(); got != uint32(3*b.Vector.Len()) || b.Tag != 7 {
		t.Fatalf("cross-package block: %+v, sum=%d", b, got)
	}
	b.Vector = vectorops.AddN(b.Vector, 4)
	method := b.Sum
	if got := method(); got != uint32(7*b.Vector.Len()) {
		t.Fatalf("recursive/method-value sum=%d", got)
	}
	add := vectorops.Adder(b.Vector)
	value := add(simd.BroadcastUint32s(2))
	holder := vectorops.Hold(value)
	if got := vectorops.Length(holder.Value); got != b.Vector.Len() {
		t.Fatalf("generic length=%d", got)
	}
	if got := (vectorops.Block{Vector: holder.Value}).Sum(); got != uint32(9*value.Len()) {
		t.Fatalf("closure result=%d", got)
	}
	if sum, tag := vectorops.Tagged(5, 19); sum != uint32(5*value.Len()) || tag != 19 {
		t.Fatalf("generic boundary=(%d,%d)", sum, tag)
	}
}

func TestPortableCapturedInterface(t *testing.T) {
	x := simd.BroadcastUint32s(8)
	box := any(x)
	get := func() simd.Uint32s { return box.(simd.Uint32s) }
	if got := (vectorops.Block{Vector: get()}).Sum(); got != uint32(8*x.Len()) {
		t.Fatalf("captured interface=%d", got)
	}
	var reducer interface{ Reduce([]uint32) uint32 } = vectorops.Reducer{}
	if got := reducer.Reduce([]uint32{1, 2, 3, 4, 5}); got != 15 {
		t.Fatalf("method-body dependent receiver=%d", got)
	}
}
