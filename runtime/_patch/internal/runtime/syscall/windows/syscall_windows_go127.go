//go:build windows && go1.27

package windows

import _ "unsafe"

//llgo:skip asmstdcall

// These entries own the whole selected native syscall assembly file. The C
// adapter consumes the SDK's actual five-word block, not a gc register frame.

//go:linkname llgoStdCall C.llgo_windows_stdcall_go127
func llgoStdCall(*StdCallInfo) uint32

//go:linkname llgoStdCallAddr C.llgo_windows_stdcall_go127_addr
func llgoStdCallAddr() uintptr

func StdCall(fn *StdCallInfo) uint32 {
	return llgoStdCall(fn)
}

func AsmStdCallAddr() uintptr {
	return llgoStdCallAddr()
}
