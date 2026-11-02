//go:build llgo && goexperiment.simd && amd64

package portable

import (
	"simd"
	"simd/archsimd"
	"testing"

	"github.com/xgo-dev/llgo/test/simd/portable/internal/vectorops"
)

var portableFMVInitCalls int
var portableFMVInitial = portableFMVInitialize()

func portableFMVInitialize() vectorops.FMVBlock {
	portableFMVInitCalls++
	b := vectorops.FMVBlock{Values: [2]simd.Uint32s{simd.BroadcastUint32s(10), simd.BroadcastUint32s(3)}}
	b, _ = vectorops.FMVGuarded(b, func() {})
	return b
}

func TestPortableFMVInit(t *testing.T) {
	if portableFMVInitCalls != 1 {
		t.Fatalf("initializer ran %d times", portableFMVInitCalls)
	}
	want := uint32(7)
	if archsimd.X86.AVX2() {
		want = 13
	}
	v := portableFMVInitial.Values[0]
	xs := make([]uint32, v.Len())
	v.Store(xs)
	for i, x := range xs {
		if x != want {
			t.Fatalf("initialized lane %d=%d want %d", i, x, want)
		}
	}
}
