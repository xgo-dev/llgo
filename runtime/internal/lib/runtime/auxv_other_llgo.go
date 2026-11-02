//go:build !linux || baremetal

package runtime

import _ "unsafe"

// The Go runtime exports this hook on every platform, including platforms
// without a process auxiliary vector. x/sys/cpu may reference it unconditionally.
//
//go:linkname getAuxv runtime.getAuxv
func getAuxv() []uintptr { return nil }
