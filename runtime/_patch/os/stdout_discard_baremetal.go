//go:build llgo && linux && baremetal && llgo.baremetal.stdio && llgo.size.embedded

package os

import "unsafe"

//go:linkname llgoConsoleWrite C.write
func llgoConsoleWrite(fd int32, data unsafe.Pointer, count uintptr) int32

// Cached packages default to the normal writer. The compact entry disables
// this guard only when no program code can observe the result or stdio state.
var llgoFullStdout bool

func llgoWriteStdoutDiscard(s string) (int, error) {
	if llgoFullStdout {
		return llgoWriteStdoutDescriptor(s)
	}
	p := unsafe.Pointer(unsafe.StringData(s))
	left := uintptr(len(s))
	for {
		count := left
		if count > llgoStdoutMaxWrite {
			count = llgoStdoutMaxWrite
		}
		n := llgoConsoleWrite(1, p, count)
		if n <= 0 || uintptr(n) >= left || uintptr(n) > count {
			break
		}
		p = unsafe.Add(p, uintptr(n))
		left -= uintptr(n)
	}
	return 0, nil
}
