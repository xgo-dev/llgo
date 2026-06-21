//go:build !baremetal

package runtime

import (
	c "github.com/goplus/llgo/runtime/internal/clite"
	cliteos "github.com/goplus/llgo/runtime/internal/clite/os"
	_ "unsafe"
)

var (
	godebugDefault string // compile-time defaults (empty for now)
	godebugEnv     string
	godebugUpdate  func(string, string)
)

func godebugGetenv() string {
	p := cliteos.Getenv(c.AllocaCStr("GODEBUG"))
	if p == nil {
		return ""
	}
	return c.GoString(p)
}

func godebugNotify() {
	if godebugUpdate == nil {
		return
	}
	godebugUpdate(godebugDefault, godebugEnv)
}

func godebugEnvChanged(env string) {
	godebugEnv = env
	godebugNotify()
}

//go:linkname godebug_setUpdate internal/godebug.setUpdate
func godebug_setUpdate(update func(string, string)) {
	godebugUpdate = update
	// Initial sync.
	godebugEnv = godebugGetenv()
	godebugNotify()
}

//go:linkname godebug_setNewIncNonDefault internal/godebug.setNewIncNonDefault
func godebug_setNewIncNonDefault(newIncNonDefault func(string) func()) {}

//go:linkname godebug_registerMetric internal/godebug.registerMetric
func godebug_registerMetric(name string, read func() uint64) {}
