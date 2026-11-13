package wordmap

import (
	"math/rand"
	"runtime"
	"strconv"
	"testing"
	"unsafe"
)

func TestOperations(t *testing.T) {
	var table Map[uintptr, *int]
	want := make(map[uintptr]*int)
	rng := rand.New(rand.NewSource(42))
	for step := 0; step < 20000; step++ {
		key := uintptr(rng.Intn(1000) + 1)
		switch rng.Intn(3) {
		case 0:
			value := new(int)
			*value = step
			table.Set(key, value)
			want[key] = value
		case 1:
			table.Delete(key)
			delete(want, key)
		case 2:
			if got := table.Get(key); got != want[key] {
				t.Fatalf("step %d: key %d = %v, want %v", step, key, got, want[key])
			}
		}
		if table.count != len(want) {
			t.Fatalf("step %d: count = %d, want %d", step, table.count, len(want))
		}
	}
	seen := make(map[uintptr]bool)
	table.Range(func(k uintptr, v *int) {
		if seen[k] || v != want[k] {
			t.Errorf("unexpected range entry %d", k)
		}
		seen[k] = true
	})
	if len(seen) != len(want) {
		t.Fatal("range omitted entries")
	}
	for k := range want {
		table.Delete(k)
	}
	for _, e := range table.entries {
		if e.key != 0 || e.value != nil {
			t.Fatal("removed entries retain keys or values")
		}
	}
	table.Delete(0)
	if table.Get(0) != nil || table.Get(123) != nil {
		t.Fatal("empty lookup")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("zero key accepted")
		}
	}()
	table.Set(0, nil)
}

func TestWrappedCollisionDeletion(t *testing.T) {
	var keys []uintptr
	for k := uintptr(1); len(keys) < 6; k++ {
		if hash(k)&7 == 6 {
			keys = append(keys, k)
		}
	}
	for removed := range keys {
		var table Map[uintptr, int]
		for i, k := range keys {
			table.Set(k, i+1)
		}
		table.Delete(keys[removed])
		for i, k := range keys {
			want := i + 1
			if i == removed {
				want = 0
			}
			if got := table.Get(k); got != want {
				t.Fatalf("removed %d, key %d = %d, want %d", removed, i, got, want)
			}
		}
	}
}

func TestOverwriteAtGrowthThreshold(t *testing.T) {
	var table Map[uintptr, int]
	for k := uintptr(1); k <= 6; k++ {
		table.Set(k, int(k))
	}
	capacity := len(table.entries)
	for repeat := 0; repeat < 100; repeat++ {
		for k := uintptr(1); k <= 6; k++ {
			table.Set(k, repeat)
			if table.Get(k) != repeat {
				t.Fatal("overwrite lost its value")
			}
		}
	}
	if len(table.entries) != capacity || table.count != 6 {
		t.Fatal("overwriting existing keys grew the table")
	}
	table.Set(7, 7)
	if len(table.entries) <= capacity || table.Get(7) != 7 {
		t.Fatal("inserting a new key did not grow the table")
	}
}

func TestPointerRoots(t *testing.T) {
	var table Map[unsafe.Pointer, *[64]int]
	for i := 0; i < 256; i++ {
		key := new([64]int)
		key[0] = i
		value := new([64]int)
		value[0] = i + 256
		table.Set(unsafe.Pointer(key), value)
	}
	for i := 0; i < 4; i++ {
		runtime.GC()
	}
	seen := [256]bool{}
	table.Range(func(k unsafe.Pointer, v *[64]int) {
		i := (*[64]int)(k)[0]
		if i < 0 || i >= len(seen) || seen[i] || v[0] != i+256 {
			t.Fatal("pointer key or value lost across GC")
		}
		seen[i] = true
	})
	for _, ok := range seen {
		if !ok {
			t.Fatal("entry lost across GC")
		}
	}
	runtime.KeepAlive(&table)
}

var benchmarkValue uintptr

func BenchmarkRegistry(b *testing.B) {
	for _, size := range []int{1, 32, 1024, 4096} {
		for _, operation := range []string{"lookup", "replace", "churn"} {
			for _, useTable := range []bool{true, false} {
				name := "map"
				if useTable {
					name = "table"
				}
				b.Run(name+"/"+operation+"/"+strconv.Itoa(size), func(b *testing.B) {
					var table Map[uintptr, uintptr]
					baseline := make(map[uintptr]uintptr)
					for i := 1; i <= size; i++ {
						k := uintptr(i * 16)
						if useTable {
							table.Set(k, k)
						} else {
							baseline[k] = k
						}
					}
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						k := uintptr((i&(size-1) + 1) * 16)
						if useTable {
							switch operation {
							case "lookup":
								benchmarkValue = table.Get(k)
							case "churn":
								table.Delete(k)
								table.Set(k, k)
							case "replace":
								table.Set(k, k)
							}
						} else {
							switch operation {
							case "lookup":
								benchmarkValue = baseline[k]
							case "churn":
								delete(baseline, k)
								baseline[k] = k
							case "replace":
								baseline[k] = k
							}
						}
					}
				})
			}
		}
	}
}
