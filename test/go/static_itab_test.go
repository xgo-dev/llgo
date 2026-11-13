package gotest

import "testing"

type staticItabReader interface{ Value() uint64 }
type staticItabItem struct{ value uint64 }

//go:noinline
func (p *staticItabItem) Value() uint64 { return p.value }

//go:noinline
func staticItabConvert(p *staticItabItem) staticItabReader { return p }

//go:noinline
func staticItabRetain(r staticItabReader) staticItabReader { return r }

func TestStaticPointerInterfaceConversion(t *testing.T) {
	objects := [8]staticItabItem{{1}, {2}, {3}, {4}, {5}, {6}, {7}, {8}}
	var total uint64
	for i := 0; i < 10000; i++ {
		p := &objects[i&7]
		r := staticItabConvert(p)
		if r != staticItabRetain(p) || r.(*staticItabItem) != p {
			t.Fatal("conversion changed interface identity or its data pointer")
		}
		total += r.Value()
	}
	if total != 45000 {
		t.Fatalf("conversion checksum = %d, want 45000", total)
	}
	var p *staticItabItem
	if r := staticItabConvert(p); r == nil || r.(*staticItabItem) != nil {
		t.Fatal("conversion lost the concrete type of a nil pointer")
	}
}

func BenchmarkStaticPointerToInterface(b *testing.B) {
	objects := [8]staticItabItem{{1}, {2}, {3}, {4}, {5}, {6}, {7}, {8}}
	var sum uint64
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sum += staticItabConvert(&objects[i&7]).Value()
	}
	b.StopTimer()
	checkStaticItabBenchmark(b, sum)
}

func BenchmarkExistingStaticInterface(b *testing.B) {
	objects := [8]staticItabItem{{1}, {2}, {3}, {4}, {5}, {6}, {7}, {8}}
	var readers [8]staticItabReader
	for i := range objects {
		readers[i] = &objects[i]
	}
	var sum uint64
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sum += staticItabRetain(readers[i&7]).Value()
	}
	b.StopTimer()
	checkStaticItabBenchmark(b, sum)
}

func checkStaticItabBenchmark(b *testing.B, got uint64) {
	b.Helper()
	remainder := uint64(b.N % 8)
	want := uint64(b.N/8)*36 + remainder*(remainder+1)/2
	if got != want {
		b.Fatalf("checksum = %d, want %d", got, want)
	}
}
