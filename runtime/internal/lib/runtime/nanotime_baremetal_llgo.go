//go:build baremetal

package runtime

func runtimeNano() int64 { return 0 }

// The standard library also links directly to runtime.nanotime.
func nanotime() int64 { return runtimeNano() }
