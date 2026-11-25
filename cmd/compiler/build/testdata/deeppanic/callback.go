package deeppanic

/*
void call_go(void);
void call_go_on_thread(void);
void call_go_through_library(void);
*/
import "C"

import "runtime/debug"

var callbackDepth int
var callbackFault bool

//export panic_callback
func panic_callback() {
	if callbackFault {
		debug.SetPanicOnFault(true)
		deepFault(callbackDepth)
		return
	}
	deepCall(callbackDepth)
}

//go:noinline
func callCGoCallback() {
	C.call_go()
}

func callForeignThreadCallback() { C.call_go_on_thread() }

//go:noinline
func callDynamicLibraryCallback() { C.call_go_through_library() }
