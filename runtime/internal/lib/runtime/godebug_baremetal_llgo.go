//go:build baremetal

package runtime

import _ "unsafe"

var (
	godebugDefault string
	godebugEnv     string
	godebugUpdate  func(string, string)
)

func godebugGetenv() string {
	return ""
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
	godebugEnv = godebugGetenv()
	godebugNotify()
}

//go:linkname godebug_setNewIncNonDefault internal/godebug.setNewIncNonDefault
func godebug_setNewIncNonDefault(newIncNonDefault func(string) func()) {}

//go:linkname godebug_registerMetric internal/godebug.registerMetric
func godebug_registerMetric(name string, read func() uint64) {}
