package llvmintrinsic

import (
	"fmt"

	"github.com/xgo-dev/llvm"
)

func vectorShape(t llvm.Type) (count uint64, scalable bool, ok bool) {
	n, scalable := t.VectorElementCount()
	return uint64(n), scalable, n != 0
}

// PointerIndexBits returns the index width for a pointer or pointer vector in a
// target data layout. It is independent of the Go word size and pointer size.
func PointerIndexBits(t llvm.Type, layout string) (uint64, error) {
	if t.IsNil() {
		return 0, fmt.Errorf("nil pointer type")
	}
	td, err := llvm.ParseTargetData(layout)
	if err != nil {
		return 0, fmt.Errorf("invalid data layout: %w", err)
	}
	defer td.Dispose()
	bits := td.IndexTypeSizeInBits(t)
	if bits == 0 {
		return 0, fmt.Errorf("non-pointer type %s", t)
	}
	return uint64(bits), nil
}
