package escape

import (
	"runtime"
	"testing"
)

//go:noinline
func saveBudgetDefer(out *[]int, data [30000]byte, id int) {
	runtime.GC()
	*out = append(*out, 100*id+int(data[0]))
}

func TestDeferStackBudgetPreservesOwnership(t *testing.T) {
	var got []int
	func() {
		var data [30000]byte
		data[0] = 1
		defer saveBudgetDefer(&got, data, 1)
		data[0] = 2
		defer saveBudgetDefer(&got, data, 2)
		data[0] = 3
		defer saveBudgetDefer(&got, data, 3)
		runtime.GC()
	}()
	if len(got) != 3 || got[0] != 303 || got[1] != 202 || got[2] != 101 {
		t.Fatalf("defer ownership/capture/order corrupted: %v", got)
	}
}
