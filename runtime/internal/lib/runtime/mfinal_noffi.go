//go:build llgo_noffi && !nogc

package runtime

// SetFinalizer is unavailable in the noffi runtime variant. Full-LTO builds
// use this variant only after proving that the program does not use FFI.
func SetFinalizer(obj any, finalizer any) {
	_, _ = obj, finalizer
}

func runFinalizers() {}
