//go:build go1.23

package runtime

import _ "unsafe"

// The build parser fills this file with GOROOT runtime.setCrashFD.
// LLGo owns crashFD, panicking and fatal-output delivery; GOROOT owns the
// concurrent descriptor exchange. See doc/runtime-source-reuse.md.

//go:linkname setCrashFD
func setCrashFD(fd uintptr) uintptr
