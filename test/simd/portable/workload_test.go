//go:build goexperiment.simd && (amd64 || arm64 || wasm)

package portable

import (
	"fmt"
	"math"
	"simd"
	"testing"
)

// dotSIMD scores equal-length vectors with ordinary (unfused) multiply/add.
// Keep the reduction outside the loop and include partial loads in the workload.
func dotSIMD(x, y []float32) float32 {
	var acc simd.Float32s
	i := 0
	for ; i+acc.Len() <= len(x); i += acc.Len() {
		a := simd.LoadFloat32s(x[i:])
		b := simd.LoadFloat32s(y[i:])
		acc = acc.Add(a.Mul(b))
	}
	if i < len(x) {
		a, _ := simd.LoadFloat32sPart(x[i:])
		b, _ := simd.LoadFloat32sPart(y[i:])
		acc = acc.Add(a.Mul(b))
	}
	lanes := make([]float32, acc.Len())
	acc.Store(lanes)
	var result float32
	for _, v := range lanes {
		result += v
	}
	return result
}

func dotScalar(x, y []float32) float32 {
	var result float32
	for i, v := range x {
		result += float32(v * y[i])
	}
	return result
}

func dotReference(x, y []float32) (sum, magnitude float64) {
	for i, v := range x {
		p := float64(v) * float64(y[i])
		sum += p
		magnitude += math.Abs(p)
	}
	return
}

func dotInputs(n, offset int) ([]float32, []float32) {
	x := make([]float32, n+offset)[offset:]
	y := make([]float32, n+offset+1)[offset+1:]
	for i := range x {
		// Small dyadic values keep every product and sum exact, regardless
		// of reduction order, even for the largest case below.
		x[i] = float32((i*17+3)%31-15) / 16
		y[i] = float32((i*13+7)%29-14) / 16
	}
	return x, y
}

func TestPortableDotWorkload(t *testing.T) {
	t.Logf("dot workload: width=%d emulated=%t", simd.VectorBitSize(), simd.Emulated())
	width := (simd.Float32s{}).Len()
	lengths := []int{127, 128, 129, 383, 384, 385, 767, 768, 769, 4099}
	for n := 0; n <= 2*width+1; n++ {
		lengths = append(lengths, n)
	}
	for offset := 0; offset < 4; offset++ {
		for _, n := range lengths {
			x, y := dotInputs(n, offset)
			want, _ := dotReference(x, y)
			if got := dotSIMD(x, y); float64(got) != want {
				t.Fatalf("offset=%d length=%d: dot=%g want %g", offset, n, got, want)
			}
		}
	}
}

func TestPortableDotBatch(t *testing.T) {
	// The scoring stage of vector search: one query against a small corpus,
	// with common embedding dimensions and independently checked scores.
	for _, n := range []int{128, 384, 768} {
		query, _ := dotInputs(n, 1)
		corpus := make([]float32, 32*n)
		for i := range corpus {
			corpus[i] = float32((i*37+i/n*11)%61-30) / 32
		}
		for row := 0; row < 32; row++ {
			candidate := corpus[row*n : (row+1)*n]
			want, _ := dotReference(query, candidate)
			if got := dotSIMD(query, candidate); float64(got) != want {
				t.Fatalf("dimension=%d row=%d: score=%g want %g", n, row, got, want)
			}
		}
	}
}

func TestPortableDotRounding(t *testing.T) {
	for _, n := range []int{3, 17, 129, 769} {
		x, y := dotInputs(n, 1)
		for i := range x {
			x[i] /= 3
			y[i] /= 7
		}
		want, magnitude := dotReference(x, y)
		// Bound accumulated float32 rounding by gamma(2*n+1)*sum(abs(x*y)),
		// permitting different reduction orders without a fixed decimal epsilon.
		eps := float64(2*n+1) * (1.0 / (1 << 24))
		tolerance := eps / (1 - eps) * magnitude
		got := float64(dotSIMD(x, y))
		if math.IsNaN(got) || math.Abs(got-want) > tolerance {
			t.Fatalf("length=%d: dot=%g want %g tolerance=%g", n, got, want, tolerance)
		}
	}
}

func BenchmarkSIMDDot(b *testing.B) {
	for _, n := range []int{128, 384, 768, 4096} {
		x, y := dotInputs(n, 1)
		want, _ := dotReference(x, y)
		for _, impl := range []struct {
			name string
			dot  func([]float32, []float32) float32
		}{{"scalar", dotScalar}, {"simd", dotSIMD}} {
			b.Run(fmt.Sprintf("%s/%d", impl.name, n), func(b *testing.B) {
				b.SetBytes(int64(8 * n))
				b.ReportAllocs()
				var got float32
				for b.Loop() {
					got = impl.dot(x, y)
				}
				if float64(got) != want {
					b.Fatalf("dot=%g want %g", got, want)
				}
			})
		}
	}
}
