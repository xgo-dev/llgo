//go:build go1.21

package runtime_test

import (
	"runtime"
	"testing"
	"time"
	_ "unsafe"
)

//go:linkname runtimeAuxv runtime.getAuxv
func runtimeAuxv() []uintptr

//go:linkname runtimeNanotime runtime.nanotime
func runtimeNanotime() int64

//go:linkname runtimeFastrand64 runtime.fastrand64
func runtimeFastrand64() uint64

func TestLegacyFastrand64Hook(t *testing.T) {
	first := runtimeFastrand64()
	for i := 0; i < 16; i++ {
		if runtimeFastrand64() != first {
			return
		}
	}
	t.Fatal("legacy random hook returned the same value repeatedly")
}

func TestNanotimeHook(t *testing.T) {
	before := runtimeNanotime()
	time.Sleep(5 * time.Millisecond)
	if after := runtimeNanotime(); after <= before {
		t.Fatalf("monotonic clock did not advance: %d -> %d", before, after)
	}
}

func TestAuxvHook(t *testing.T) {
	auxv := runtimeAuxv()
	if len(auxv)%2 != 0 {
		t.Fatalf("auxv is not a list of tag/value pairs: %v", auxv)
	}
	switch runtime.GOOS {
	case "darwin", "ios", "windows", "js", "wasip1":
		if len(auxv) != 0 {
			t.Fatalf("unexpected auxiliary vector on %s: %v", runtime.GOOS, auxv)
		}
	}
	for i := 0; i < len(auxv); i += 2 {
		if auxv[i] == 0 {
			t.Fatalf("auxv includes its AT_NULL terminator: %v", auxv)
		}
	}
}
