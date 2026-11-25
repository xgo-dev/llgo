//go:build !llgo || !baremetal

package runtime

// LockCore is unimplemented on hosted and Wasm targets: the OS scheduler
// there offers no single "the" core for a pin, so the honest answer is an
// explicit error, never a silent pretend-pin.
var errLockCoreUnsupported = errorString("runtime.LockCore: not implemented on this platform")

func LockCore(id int) error {
	return errLockCoreUnsupported
}
