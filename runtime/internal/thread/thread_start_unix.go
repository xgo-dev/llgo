//go:build !windows && !(baremetal && llgo.singlecontext)

package thread

import c "github.com/xgo-dev/llgo/runtime/internal/clite"

// CreateDetached starts a detached host thread using the native pthread
// backend. GC-enabled builds select the collector-aware create entry point.
func CreateDetached(stackSize uintptr, routine RoutineFunc, arg c.Pointer) c.Int {
	var attr nativeAttr
	if ret := attr.Init(); ret != 0 {
		return ret
	}
	if ret := attr.SetDetached(pthreadCreateDetached); ret != 0 {
		_ = attr.Destroy()
		return ret
	}
	if stackSize != 0 {
		if ret := attr.SetStackSize(stackSize); ret != 0 {
			_ = attr.Destroy()
			return ret
		}
	}
	var native nativeThread
	ret := create(&native, &attr, routine, arg)
	// Once create succeeds, the caller-owned context belongs to the detached
	// thread. An attribute destroy failure cannot be reported as create failure.
	_ = attr.Destroy()
	return ret
}
