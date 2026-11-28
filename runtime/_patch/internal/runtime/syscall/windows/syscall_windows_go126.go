//go:build windows && go1.26 && !go1.27

package windows

import _ "unsafe"

//llgo:skip asmstdcall

// These entries own the whole selected native syscall assembly file. The C
// adapter consumes the SDK's actual six-word block directly.

//go:linkname llgoStdCall C.llgo_windows_stdcall_go126
func llgoStdCall(*StdCallInfo)

//go:linkname llgoStdCallAddr C.llgo_windows_stdcall_go126_addr
func llgoStdCallAddr() uintptr

func StdCall(fn *StdCallInfo) {
	llgoStdCall(fn)
}

func AsmStdCallAddr() uintptr {
	return llgoStdCallAddr()
}
