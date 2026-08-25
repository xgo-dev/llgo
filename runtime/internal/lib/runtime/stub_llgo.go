package runtime

import (
	"unsafe"

	clite "github.com/goplus/llgo/runtime/internal/clite"
)

func memmove(to, from unsafe.Pointer, n uintptr) {
	clite.Memmove(to, from, n)
}

// auxv is populated on relevant platforms but defined here for all platforms
// so x/sys/cpu and x/sys/unix can assume the getAuxv symbol exists without
// keeping its list of auxv-using GOOS build tags in sync.
//
// It contains an even number of elements, (tag, value) pairs.
var auxv []uintptr

// golang.org/x/sys/cpu and golang.org/x/sys/unix use getAuxv via linkname.
// Do not remove or change the type signature.
// See go.dev/issue/57336 and go.dev/issue/67401.
//
//go:linkname getAuxv
func getAuxv() []uintptr { return auxv }
