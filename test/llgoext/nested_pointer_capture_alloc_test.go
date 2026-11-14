//go:build llgo && !nogc

package llgoext

import "testing"

var nestedAllocValue = uint64(7)
var nestedAllocOther = uint64(23)
var nestedAllocSink func() uint64
var nestedAllocRead uint64

//go:noinline
func nestedAllocRestore(p *uint64) func() uint64 {
	return func() uint64 {
		previous := *p
		defer func() { *p = previous }()
		*p++
		return *p
	}
}

//go:noinline
func nestedAllocReadPointer(p *uint64) func() uint64 {
	return func() uint64 {
		defer func() { nestedAllocRead = *p }()
		return *p
	}
}

//go:noinline
func nestedAllocTwo(p, q *uint64) func() uint64 {
	return func() uint64 {
		previous := *p
		defer func() { *p = previous; *q += 0 }()
		*p++
		return *p
	}
}

//go:noinline
func nestedAllocShared(p *uint64) func() uint64 {
	return func() uint64 {
		defer func() { p = &nestedAllocOther }()
		return *p
	}
}

func TestNestedPointerCaptureAllocations(t *testing.T) {
	for _, tc := range []struct {
		name  string
		build func() func() uint64
		want  uint64
		alloc float64
	}{
		{"read", func() func() uint64 { return nestedAllocReadPointer(&nestedAllocValue) }, 7, 1},
		{"restore", func() func() uint64 { return nestedAllocRestore(&nestedAllocValue) }, 8, 1},
		{"two pointers", func() func() uint64 { return nestedAllocTwo(&nestedAllocValue, &nestedAllocValue) }, 8, 1},
		{"shared binding", func() func() uint64 { return nestedAllocShared(&nestedAllocValue) }, 7, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			nestedAllocValue = 7
			got := testing.AllocsPerRun(1000, func() {
				nestedAllocSink = tc.build()
				if v := nestedAllocSink(); v != tc.want {
					t.Fatalf("callback result=%d, want %d", v, tc.want)
				}
			})
			if got != tc.alloc || nestedAllocValue != 7 {
				t.Fatalf("allocations/call=%v, pointee=%d; want %v, 7", got, nestedAllocValue, tc.alloc)
			}
			if tc.name == "read" && nestedAllocRead != 7 {
				t.Fatalf("deferred read=%d, want 7", nestedAllocRead)
			}
		})
	}
}
