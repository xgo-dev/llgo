package crossinline_test

import (
	"runtime"
	"testing"

	"github.com/xgo-dev/llgo/benchmark/crossinline/value"
)

// Each iteration depends on the previous result. KeepAlive retains the final
// value even when LTO can remove an otherwise unread package-level sink.
// The ordinary function is a control for the existing cross-package importer.
// Keep calls direct: a shared function-value helper would change what is measured.
func BenchmarkFunction(b *testing.B) {
	n := uint64(1)
	for i := range b.N {
		n = value.Mix(n, uint64(i))
	}
	runtime.KeepAlive(n)
}

func BenchmarkValueMethod(b *testing.B) {
	n := value.Number(1)
	for i := range b.N {
		n = n.Mix(uint64(i))
	}
	runtime.KeepAlive(n)
}

func BenchmarkPointerMethod(b *testing.B) {
	s := value.State{Value: 1}
	for i := range b.N {
		s.Mix(uint64(i))
	}
	runtime.KeepAlive(s.Value)
}

func pairs(seed uint64) [1024]value.Pair {
	var values [1024]value.Pair
	for i := range values {
		values[i] = value.Pair{Left: seed ^ uint64(i), Right: value.Mix(seed, uint64(i))}
	}
	return values
}

// Field access exposes independent iterations to LLVM once the body is
// imported. The runtime seed prevents folding the data into constants.
func BenchmarkFunctionFields(b *testing.B) {
	values := pairs(uint64(b.N))
	b.ResetTimer()
	var sum uint64
	for i := range b.N {
		sum += value.Sum(values[i&(len(values)-1)])
	}
	runtime.KeepAlive(sum)
}

func BenchmarkValueMethodFields(b *testing.B) {
	values := pairs(uint64(b.N))
	b.ResetTimer()
	var sum uint64
	for i := range b.N {
		sum += values[i&(len(values)-1)].Sum()
	}
	runtime.KeepAlive(sum)
}

func BenchmarkPointerMethodFields(b *testing.B) {
	values := pairs(uint64(b.N))
	b.ResetTimer()
	var sum uint64
	for i := range b.N {
		sum += values[i&(len(values)-1)].PointerSum()
	}
	runtime.KeepAlive(sum)
}

func records(seed uint64) [256]value.Record {
	var values [256]value.Record
	for i := range values {
		for j := range values[i].Fields {
			values[i].Fields[j] = value.Mix(seed, uint64(i*len(values[i].Fields)+j))
		}
	}
	return values
}

// A larger value receiver also exposes the cost of passing unused fields.
func BenchmarkFunctionRecord(b *testing.B) {
	values := records(uint64(b.N))
	b.ResetTimer()
	var sum uint64
	for i := range b.N {
		sum += value.First(values[i&(len(values)-1)])
	}
	runtime.KeepAlive(sum)
}

func BenchmarkValueMethodRecord(b *testing.B) {
	values := records(uint64(b.N))
	b.ResetTimer()
	var sum uint64
	for i := range b.N {
		sum += values[i&(len(values)-1)].First()
	}
	runtime.KeepAlive(sum)
}

func TestMethodResults(t *testing.T) {
	n := value.Number(1)
	s := value.State{Value: 1}
	want := uint64(1)
	for x := range uint64(1000) {
		want = (want ^ x) * 0x9e3779b97f4a7c15
		n = n.Mix(x)
		s.Mix(x)
		if uint64(n) != want || s.Value != want {
			t.Fatalf("iteration %d: value=%d pointer=%d, want %d", x, n, s.Value, want)
		}
	}
	for _, p := range pairs(want) {
		if got := p.Left + p.Right; value.Sum(p) != got || p.Sum() != got || p.PointerSum() != got {
			t.Fatal("field access changed the result")
		}
	}
	for _, r := range records(want) {
		if value.First(r) != r.Fields[0] || r.First() != r.Fields[0] {
			t.Fatal("record access changed the result")
		}
	}
}
