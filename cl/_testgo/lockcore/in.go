// Behavior regression fixture for runtime.LockCore on the RV32 baremetal
// fiber scheduler (esp32c3-basic, esp32s31-basic: single-hart scheduler, no
// second scheduler core). Every requested core id is trivially satisfied,
// including ids no such target could ever have - the runtime documents that
// as "always locked" rather than rejecting ids it cannot compare against.
// The golden output is identical on every backend that runs this fixture;
// hosted does not run it (imports "runtime", the darwin/arm64 bytealg
// limitation, like gosched) - there LockCore reports an explicit
// not-implemented error instead, pinned by
// runtime/internal/test/lockcore_test.go.
package main

import "runtime"

func main() {
	if err := runtime.LockCore(0); err != nil {
		println("lockcore 0 FAIL")
	} else {
		println("lockcore 0 ok")
	}
	if err := runtime.LockCore(1); err != nil {
		println("lockcore 1 FAIL")
	} else {
		println("lockcore 1 ok")
	}
	if err := runtime.LockCore(9); err != nil {
		println("lockcore 9 FAIL")
	} else {
		println("lockcore 9 ok")
	}
}
