package cpp

import _ "unsafe"

const (
	LLGoFiles   = "$LLGO_EH_CFLAGS: _wrap/exception.cpp"
	LLGoPackage = "link: -sDEFAULT_TO_CXX -sDISABLE_EXCEPTION_CATCHING=0"
)

//go:linkname Catch C.llgo_eh_cpp_catch
func Catch() int32
