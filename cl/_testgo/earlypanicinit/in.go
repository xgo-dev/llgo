// Early-panic diagnostic fixture for the ESP baremetal targets: a nil
// dereference during package init must still produce the runtime's early
// USB-Serial/JTAG banner ("llgo panic: ..." plus the scanned pc chain)
// even though stdio has not been set up yet, and the ordinary stdio panic
// line must still follow it (exit semantics unchanged).
//
// No expect.txt: the banner carries build-specific PCs, so the assertion
// lives in cl/embed_earlypanic_test.go, which boots this fixture under
// esp-emu and scans for the banner markers. Hosted and the emu auto-walk
// treat a fixture without expect.txt as run-less (IR checks only).

package main

func init() {
	var p *int
	_ = *p
}

func main() {
	println("unreachable")
}
