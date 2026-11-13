package base

import "os"

type hiddenIface interface {
	hidden() int
}

type Exported struct{}

//go:noinline
func (Exported) hidden() int {
	// Keep the required method observable to the symbol check even when
	// caller-owned interface receiver storage lets LLVM eliminate a pure
	// constant-return call. The fixture runs without additional arguments.
	return 16 + len(os.Args)
}

func Call(v hiddenIface) int {
	return v.hidden()
}
