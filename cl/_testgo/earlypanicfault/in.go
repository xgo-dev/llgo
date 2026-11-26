// Early-fault diagnostic fixture for the ESP baremetal targets: a wild
// pointer store from C glue raises a store access fault, which the runtime's
// chained trap handler must report ("llgo fault: mcause=... mepc=... "
// "mtval=... sp=..." plus the scanned pc chain) before chaining to the
// device file's trap, keeping the semihosting exit path intact.
//
// No expect.txt: the report carries build-specific PCs, so the assertion
// lives in cl/embed_earlypanic_test.go, which boots this fixture under
// esp-emu and scans for the report markers. Hosted and the emu auto-walk
// treat a fixture without expect.txt as run-less (IR checks only).

package main

import _ "unsafe"

const LLGoFiles = "glue.c"

//go:linkname triggerFault C.llgo_trigger_fault
func triggerFault()

func main() {
	println("boot ok")
	triggerFault()
	println("unreachable")
}
