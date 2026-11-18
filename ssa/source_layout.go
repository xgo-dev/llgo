package ssa

import "math"

// sourceLayout describes source sizes without constructing LLVM types.
type sourceLayout struct{ size, align int64 }

func arraySourceLayout(elem sourceLayout, n int64) sourceLayout {
	size := int64(0)
	if n > 0 {
		size = -1
		if elem.size >= 0 && (elem.size == 0 || n <= math.MaxInt64/elem.size) {
			size = elem.size * n
		}
	}
	return sourceLayout{size, elem.align}
}

// aggregateSourceLayout shares offset, overflow, and tail-padding arithmetic.
// Callers choose field layouts; the indexed callback avoids an extra scratch slice.
func aggregateSourceLayout(numFields int, fieldLayout func(int) sourceLayout, padTail bool) (sourceLayout, []int64) {
	result := sourceLayout{align: 1}
	offsets := make([]int64, numFields)
	for i := 0; i < numFields; i++ {
		field := fieldLayout(i)
		result.align = max(result.align, field.align)
		result.size = sourceLayoutAlign(result.size, field.align)
		offsets[i] = result.size
		if result.size < 0 || field.size < 0 || field.size > math.MaxInt64-result.size {
			result.size = -1
		} else {
			result.size += field.size
			if padTail && i == numFields-1 && field.size == 0 && result.size > 0 {
				if result.size == math.MaxInt64 {
					result.size = -1
				} else {
					result.size++ // Keep a final zero-sized Go field inside the object.
				}
			}
		}
	}
	result.size = sourceLayoutAlign(result.size, result.align)
	return result, offsets
}

func sourceLayoutAlign(size, alignment int64) int64 {
	if size < 0 || size > math.MaxInt64-(alignment-1) {
		return -1
	}
	return align(size, alignment)
}
