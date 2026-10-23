//go:build go1.27

package main

import (
	"math"
	"runtime"
	"syscall"
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

type stdCallInfo struct {
	fn, n, args, r1, r2 uintptr
}

//go:linkname publicStdCall internal/runtime/syscall/windows.StdCall
func publicStdCall(*stdCallInfo) uint32

//go:linkname publicStdCallAddr internal/runtime/syscall/windows.AsmStdCallAddr
func publicStdCallAddr() uintptr

//go:linkname cSumVariable C.llgo_windows_sum_variable
func cSumVariable(count uintptr) uintptr

//go:linkname cNoArguments C.llgo_windows_no_arguments
func cNoArguments() uintptr

//go:linkname cCallStdCallAddress C.llgo_windows_call_stdcall_address
func cCallStdCallAddress(address uintptr, info *stdCallInfo) uint32

//go:linkname cUnwindSeen C.llgo_windows_unwind_seen
func cUnwindSeen() int32

//go:linkname cHighWord C.llgo_windows_high_word
func cHighWord() uint64

//go:linkname cStdcallSum12 stdcall.llgo_windows_sum12_stdcall
func cStdcallSum12(a1, a2, a3, a4, a5, a6, a7, a8, a9, a10, a11, a12 uintptr) uintptr

//go:linkname cThunkRegisterCanary C.llgo_windows_syscall_register_canary
func cThunkRegisterCanary(info unsafe.Pointer) int32

func testDynamicWindowsSyscalls() {
	const lastError = 1234
	if runtime.GOARCH == "arm64" {
		// Call the actual six-word native thunk directly: a C wrapper that
		// happened to save x20 itself could otherwise conceal the defect.
		block := struct {
			fn, n, args, r1, r2, err uintptr
		}{fn: uintptr(c.Func(cNoArguments))}
		if cThunkRegisterCanary(unsafe.Pointer(&block)) != 1 || block.r1 != 7 || block.err != lastError {
			panic("actual ARM64 thunk clobbered x19/x20/x29 or its native call result")
		}
	}
	for _, n := range []int{0, 8, 12, 42, 43, 1026} {
		arguments := make([]uintptr, n)
		fn, want := uintptr(c.Func(cNoArguments)), uintptr(7)
		if n != 0 {
			fn = uintptr(c.Func(cSumVariable))
			arguments[0] = uintptr(n - 1)
			want = uintptr(n*(n-1)) / 2
			for i := 1; i < n; i++ {
				arguments[i] = uintptr(i)
			}
		}
		r1, _, errno := syscall.SyscallN(fn, arguments...)
		if r1 != want || errno != lastError || cUnwindSeen() != 1 {
			panic("dynamic SyscallN lost arguments, last-error, or unwind chain")
		}
		for _, throughC := range []bool{false, true} {
			block := struct {
				before uintptr
				info   stdCallInfo
				after  uintptr
			}{before: 0x73928a51, after: 0x73928a51}
			block.info = stdCallInfo{fn: fn, n: uintptr(n), r2: 0xa5a5}
			if n != 0 {
				block.info.args = uintptr(unsafe.Pointer(&arguments[0]))
			}
			before := block.info
			var err uint32
			if throughC {
				err = cCallStdCallAddress(publicStdCallAddr(), &block.info)
			} else {
				err = publicStdCall(&block.info)
			}
			if err != lastError || block.info.r1 != want || block.info.fn != before.fn ||
				block.info.n != before.n || block.info.args != before.args ||
				block.before != 0x73928a51 || block.after != 0x73928a51 || cUnwindSeen() != 1 {
				panic("Go/C StdCall entry violated the actual five-word contract")
			}
			if runtime.GOARCH == "arm64" && block.info.r2 != before.r2 {
				panic("ARM64 StdCall changed its original untouched R2 field")
			}
		}
		runtime.KeepAlive(arguments)
	}
	if runtime.GOARCH == "386" {
		r1, r2, _ := syscall.SyscallN(uintptr(c.Func(cHighWord)))
		if uint32(r1) != 0x89abcdef || uint32(r2) != 0x81234567 {
			panic("386 native call lost its DX:AX high-word return")
		}
		r1, _, _ = syscall.SyscallN(uintptr(c.Func(cStdcallSum12)), 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12)
		if r1 != 78 {
			panic("386 dynamic frame did not restore stdcall callee-pop stack")
		}
	}
	testPublicStdCallReturns()
}

func testPublicStdCallReturns() {
	var info stdCallInfo
	var arguments [2]uintptr
	var wantR1, wantR2 uintptr
	switch runtime.GOARCH {
	case "386":
		info.fn = uintptr(c.Func(cHighWord))
		wantR1, wantR2 = 0x89abcdef, 0x81234567
	case "amd64":
		info.fn = uintptr(c.Func(cAddDouble))
		arguments = [2]uintptr{uintptr(math.Float64bits(1.25)), uintptr(math.Float64bits(2.5))}
		info.n, info.args = 2, uintptr(unsafe.Pointer(&arguments[0]))
		wantR2 = uintptr(math.Float64bits(3.75))
	default:
		return // ARM64 has no second result register in this SDK contract.
	}
	for _, throughC := range []bool{false, true} {
		info.r1, info.r2 = 0, 0
		if throughC {
			cCallStdCallAddress(publicStdCallAddr(), &info)
		} else {
			publicStdCall(&info)
		}
		if info.r2 != wantR2 || runtime.GOARCH == "386" && info.r1 != wantR1 {
			panic("public Go/C StdCall did not retain the real secondary return register")
		}
	}
	runtime.KeepAlive(arguments)
}
