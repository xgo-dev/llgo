package cpp

import _ "unsafe"

const (
	LLGoFiles   = "$LLGO_EH_CFLAGS: _wrap/exception.cpp"
	LLGoPackage = "link: -sDEFAULT_TO_CXX -fwasm-exceptions"
)

//go:linkname Catch C.llgo_eh_cpp_catch
func Catch() int32

//go:linkname CatchAndSuspend C.llgo_eh_cpp_catch_and_suspend
func CatchAndSuspend(indirect int32)

//go:linkname Mode C.llgo_eh_test_mode
func Mode() int32
