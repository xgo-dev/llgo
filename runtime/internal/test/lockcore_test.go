//go:build llgo && !baremetal

package test

import (
	"testing"

	rt "github.com/xgo-dev/llgo/runtime/internal/runtime"
)

// TestLockCoreHostedUnsupported pins the hosted contract of runtime.LockCore:
// the API exists everywhere, but a platform without a pin gets an explicit
// error, never a silent pretend-pin. The target-tagged variants are exercised
// end to end by the emulator suites (cl/_testgo/lockcore for the
// always-locked RV32 baremetal targets, _demo/esp-idf-lockcore for the
// esp32s31-idf FreeRTOS pin).
func TestLockCoreHostedUnsupported(t *testing.T) {
	for _, id := range []int{0, 1, 9, -1} {
		if err := rt.LockCore(id); err == nil {
			t.Fatalf("LockCore(%d) returned nil on hosted; want an explicit error", id)
		}
	}
}
