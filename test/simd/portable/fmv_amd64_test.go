//go:build goexperiment.simd && amd64

package portable

import (
	"os"
	"simd"
	"simd/archsimd"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/test/simd/portable/internal/vectorops"
)

func TestPortableFMVSelection(t *testing.T) {
	debug := "," + os.Getenv("GODEBUG") + ","
	for _, feature := range []struct {
		name    string
		enabled bool
	}{
		{"avx2", archsimd.X86.AVX2()}, {"avx", archsimd.X86.AVX()}, {"fma", archsimd.X86.FMA()},
	} {
		if (strings.Contains(debug, ",cpu."+feature.name+"=off,") || strings.Contains(debug, ",cpu.all=off,")) && feature.enabled {
			t.Errorf("disabled feature %s is enabled", feature.name)
		}
	}
	if strings.Contains(debug, ",simd=0,") && !simd.Emulated() {
		t.Fatal("simd=0 selected hardware")
	}
	t.Logf("width=%d emulated=%t AVX2=%t AVX=%t FMA=%t", simd.VectorBitSize(), simd.Emulated(), archsimd.X86.AVX2(), archsimd.X86.AVX(), archsimd.X86.FMA())
}

func TestPortableFMVComposition(t *testing.T) {
	want, count := uint32(7), 5
	if archsimd.X86.AVX2() {
		want, count = 13, 106
	}
	if archsimd.X86.AVX() {
		count += 10
	}
	if archsimd.X86.FMA() {
		count++
	}
	makeBlock := func() vectorops.FMVBlock {
		return vectorops.FMVBlock{Values: [2]simd.Uint32s{simd.BroadcastUint32s(10), simd.BroadcastUint32s(3)}, Count: 5, Label: "preserved"}
	}
	check := func(name string, b vectorops.FMVBlock, n int) {
		t.Helper()
		if n != count || b.Count != count || b.Label != "preserved" {
			t.Fatalf("%s: count=%d field=%d label=%q, want count=%d", name, n, b.Count, b.Label, count)
		}
		for i, v := range b.Values {
			xs := make([]uint32, v.Len())
			v.Store(xs)
			expected := want
			if i == 1 {
				expected = 3
			}
			for lane, x := range xs {
				if x != expected {
					t.Fatalf("%s vector=%d lane=%d got=%d want=%d", name, i, lane, x, expected)
				}
			}
		}
	}
	b := makeBlock()
	// Mutating the original while it is passed by value must not change the copy;
	// assigning the return to the same variable also exercises sret/byval aliasing.
	b, n := vectorops.FMVGuarded(b, func() { b.Values[0] = simd.BroadcastUint32s(999); b.Count = -1; b.Label = "mutated" })
	check("direct mutation and return alias", b, n)
	indirect := vectorops.FMVGuarded
	b, n = indirect(makeBlock(), func() {})
	check("function value", b, n)
	b = makeBlock()
	method := b.Apply
	b, n = method()
	check("method value", b, n)
	closure := vectorops.FMVClosure(makeBlock())
	b, n = closure()
	check("closure", b, n)
	var iface interface {
		Apply() (vectorops.FMVBlock, int)
	} = makeBlock()
	b, n = iface.Apply()
	check("interface", b, n)
	b, n = vectorops.FMVGeneric(makeBlock(), 0)
	check("generic", b, n)
	b = makeBlock()
	b.Values = vectorops.FMVArray(b.Values)
	b.Count = count
	check("array", b, count)
}
