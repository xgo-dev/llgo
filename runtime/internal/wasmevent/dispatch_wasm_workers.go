//go:build llgo && js && wasm && llgo.wasm_workers

package wasmevent

var wakeEvent func()

// InstallWake registers the scheduler wakeup used when another worker changes
// the earliest host-event deadline.
func InstallWake(wake func()) {
	wakeEvent = wake
}

func notifyWake() {
	if wakeEvent != nil {
		wakeEvent()
	}
}
