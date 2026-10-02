//go:build !baremetal

package runtime

import (
	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	_ "unsafe"
)

//go:linkname cpuGetenv C.getenv
func cpuGetenv(*c.Char) *c.Char

// CPUEnvironment returns the process-start CPU overrides. The internal/cpu
// initialization hook applies the official feature and GODEBUG policy once.
func CPUEnvironment() string {
	if env := cpuGetenv(c.Str("GODEBUG")); env != nil {
		return c.GoString(env)
	}
	return ""
}
