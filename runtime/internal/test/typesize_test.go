package test

import (
	"runtime"
	"testing"
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/clite/os"
	"github.com/xgo-dev/llgo/runtime/internal/clite/pthread/sync"
	"github.com/xgo-dev/llgo/runtime/internal/clite/setjmp"
	"github.com/xgo-dev/llgo/runtime/internal/clite/time"
)

func TestOSTypes(t *testing.T) {
	if unsafe.Sizeof(mode) != unsafe.Sizeof(os.ModeT(0)) {
		t.Fatalf("mode size mismatch: %d != %d", unsafe.Sizeof(mode), unsafe.Sizeof(os.ModeT(0)))
	}
	if unsafe.Sizeof(uid) != unsafe.Sizeof(os.UidT(0)) {
		t.Fatalf("uid size mismatch: %d != %d", unsafe.Sizeof(uid), unsafe.Sizeof(os.UidT(0)))
	}
	if unsafe.Sizeof(gid) != unsafe.Sizeof(os.GidT(0)) {
		t.Fatalf("gid size mismatch: %d != %d", unsafe.Sizeof(gid), unsafe.Sizeof(os.GidT(0)))
	}
	if unsafe.Sizeof(off) != unsafe.Sizeof(os.OffT(0)) {
		t.Fatalf("off size mismatch: %d != %d", unsafe.Sizeof(off), unsafe.Sizeof(os.OffT(0)))
	}
	if unsafe.Sizeof(dev) != unsafe.Sizeof(os.DevT(0)) {
		t.Fatalf("dev size mismatch: %d != %d", unsafe.Sizeof(dev), unsafe.Sizeof(os.DevT(0)))
	}
}

func TestTypeSizes(t *testing.T) {
	// Test platform-specific sizes based on OS and architecture
	switch {
	case runtime.GOOS == "linux" && runtime.GOARCH == "amd64":
		// Linux amd64 specific tests
		if pthreadOnceSizeLinuxAmd64 != pthreadOnceSizeReal {
			t.Fatalf("pthreadOnceSizeLinuxAmd64 mismatch: %d != %d", pthreadOnceSizeLinuxAmd64, pthreadOnceSizeReal)
		}
		if pthreadMutexSizeLinuxAmd64 != pthreadMutexSizeReal {
			t.Fatalf("pthreadMutexSizeLinuxAmd64 mismatch: %d != %d", pthreadMutexSizeLinuxAmd64, pthreadMutexSizeReal)
		}
		if pthreadMutexAttrSizeLinuxAmd64 != pthreadMutexAttrSizeReal {
			t.Fatalf("pthreadMutexAttrSizeLinuxAmd64 mismatch: %d != %d", pthreadMutexAttrSizeLinuxAmd64, pthreadMutexAttrSizeReal)
		}
		if pthreadCondSizeLinuxAmd64 != pthreadCondSizeReal {
			t.Fatalf("pthreadCondSizeLinuxAmd64 mismatch: %d != %d", pthreadCondSizeLinuxAmd64, pthreadCondSizeReal)
		}
		if pthreadCondAttrSizeLinuxAmd64 != pthreadCondAttrSizeReal {
			t.Fatalf("pthreadCondAttrSizeLinuxAmd64 mismatch: %d != %d", pthreadCondAttrSizeLinuxAmd64, pthreadCondAttrSizeReal)
		}
		if pthreadRWLockSizeLinuxAmd64 != pthreadRWLockSizeReal {
			t.Fatalf("pthreadRWLockSizeLinuxAmd64 mismatch: %d != %d", pthreadRWLockSizeLinuxAmd64, pthreadRWLockSizeReal)
		}
		if pthreadRWLockAttrSizeLinuxAmd64 != pthreadRWLockAttrSizeReal {
			t.Fatalf("pthreadRWLockAttrSizeLinuxAmd64 mismatch: %d != %d", pthreadRWLockAttrSizeLinuxAmd64, pthreadRWLockAttrSizeReal)
		}
		if clockSizeLinuxAmd64 != clockSizeReal {
			t.Fatalf("clockSizeLinuxAmd64 mismatch: %d != %d", clockSizeLinuxAmd64, clockSizeReal)
		}
		if sigjmpBufSizeLinuxAmd64 != sigjmpBufSizeReal {
			t.Fatalf("sigjmpBufSizeLinuxAmd64 mismatch: %d != %d", sigjmpBufSizeLinuxAmd64, sigjmpBufSizeReal)
		}
		if jmpBufSizeLinuxAmd64 != jmpBufSizeReal {
			t.Fatalf("jmpBufSizeLinuxAmd64 mismatch: %d != %d", jmpBufSizeLinuxAmd64, jmpBufSizeReal)
		}
	case runtime.GOOS == "linux" && runtime.GOARCH == "arm64":
		// Linux arm64 specific tests
		if pthreadOnceSizeLinuxArm64 != pthreadOnceSizeReal {
			t.Fatalf("pthreadOnceSizeLinuxArm64 mismatch: %d != %d", pthreadOnceSizeLinuxArm64, pthreadOnceSizeReal)
		}
		if pthreadMutexSizeLinuxArm64 != pthreadMutexSizeReal {
			t.Fatalf("pthreadMutexSizeLinuxArm64 mismatch: %d != %d", pthreadMutexSizeLinuxArm64, pthreadMutexSizeReal)
		}
		if pthreadMutexAttrSizeLinuxArm64 != pthreadMutexAttrSizeReal {
			t.Fatalf("pthreadMutexAttrSizeLinuxArm64 mismatch: %d != %d", pthreadMutexAttrSizeLinuxArm64, pthreadMutexAttrSizeReal)
		}
		if pthreadCondSizeLinuxArm64 != pthreadCondSizeReal {
			t.Fatalf("pthreadCondSizeLinuxArm64 mismatch: %d != %d", pthreadCondSizeLinuxArm64, pthreadCondSizeReal)
		}
		if pthreadCondAttrSizeLinuxArm64 != pthreadCondAttrSizeReal {
			t.Fatalf("pthreadCondAttrSizeLinuxArm64 mismatch: %d != %d", pthreadCondAttrSizeLinuxArm64, pthreadCondAttrSizeReal)
		}
		if pthreadRWLockSizeLinuxArm64 != pthreadRWLockSizeReal {
			t.Fatalf("pthreadRWLockSizeLinuxArm64 mismatch: %d != %d", pthreadRWLockSizeLinuxArm64, pthreadRWLockSizeReal)
		}
		if pthreadRWLockAttrSizeLinuxArm64 != pthreadRWLockAttrSizeReal {
			t.Fatalf("pthreadRWLockAttrSizeLinuxArm64 mismatch: %d != %d", pthreadRWLockAttrSizeLinuxArm64, pthreadRWLockAttrSizeReal)
		}
		if clockSizeLinuxArm64 != clockSizeReal {
			t.Fatalf("clockSizeLinuxArm64 mismatch: %d != %d", clockSizeLinuxArm64, clockSizeReal)
		}
		if sigjmpBufSizeLinuxArm64 != sigjmpBufSizeReal {
			t.Fatalf("sigjmpBufSizeLinuxArm64 mismatch: %d != %d", sigjmpBufSizeLinuxArm64, sigjmpBufSizeReal)
		}
		if jmpBufSizeLinuxArm64 != jmpBufSizeReal {
			t.Fatalf("jmpBufSizeLinuxArm64 mismatch: %d != %d", jmpBufSizeLinuxArm64, jmpBufSizeReal)
		}
	case runtime.GOOS == "darwin" && runtime.GOARCH == "amd64":
		// macOS amd64 specific tests
		if pthreadOnceSizeDarwinAmd64 != pthreadOnceSizeReal {
			t.Fatalf("pthreadOnceSizeDarwinAmd64 mismatch: %d != %d", pthreadOnceSizeDarwinAmd64, pthreadOnceSizeReal)
		}
		if pthreadMutexSizeDarwinAmd64 != pthreadMutexSizeReal {
			t.Fatalf("pthreadMutexSizeDarwinAmd64 mismatch: %d != %d", pthreadMutexSizeDarwinAmd64, pthreadMutexSizeReal)
		}
		if pthreadMutexAttrSizeDarwinAmd64 != pthreadMutexAttrSizeReal {
			t.Fatalf("pthreadMutexAttrSizeDarwinAmd64 mismatch: %d != %d", pthreadMutexAttrSizeDarwinAmd64, pthreadMutexAttrSizeReal)
		}
		if pthreadCondSizeDarwinAmd64 != pthreadCondSizeReal {
			t.Fatalf("pthreadCondSizeDarwinAmd64 mismatch: %d != %d", pthreadCondSizeDarwinAmd64, pthreadCondSizeReal)
		}
		if pthreadCondAttrSizeDarwinAmd64 != pthreadCondAttrSizeReal {
			t.Fatalf("pthreadCondAttrSizeDarwinAmd64 mismatch: %d != %d", pthreadCondAttrSizeDarwinAmd64, pthreadCondAttrSizeReal)
		}
		if pthreadRWLockSizeDarwinAmd64 != pthreadRWLockSizeReal {
			t.Fatalf("pthreadRWLockSizeDarwinAmd64 mismatch: %d != %d", pthreadRWLockSizeDarwinAmd64, pthreadRWLockSizeReal)
		}
		if pthreadRWLockAttrSizeDarwinAmd64 != pthreadRWLockAttrSizeReal {
			t.Fatalf("pthreadRWLockAttrSizeDarwinAmd64 mismatch: %d != %d", pthreadRWLockAttrSizeDarwinAmd64, pthreadRWLockAttrSizeReal)
		}
		if clockSizeDarwinAmd64 != clockSizeReal {
			t.Fatalf("clockSizeDarwinAmd64 mismatch: %d != %d", clockSizeDarwinAmd64, clockSizeReal)
		}
		if sigjmpBufSizeDarwinAmd64 != sigjmpBufSizeReal {
			t.Fatalf("sigjmpBufSizeDarwinAmd64 mismatch: %d != %d", sigjmpBufSizeDarwinAmd64, sigjmpBufSizeReal)
		}
		if jmpBufSizeDarwinAmd64 != jmpBufSizeReal {
			t.Fatalf("jmpBufSizeDarwinAmd64 mismatch: %d != %d", jmpBufSizeDarwinAmd64, jmpBufSizeReal)
		}
	case runtime.GOOS == "darwin" && runtime.GOARCH == "arm64":
		// macOS arm64 specific tests
		if pthreadOnceSizeDarwinArm64 != pthreadOnceSizeReal {
			t.Fatalf("pthreadOnceSizeDarwinArm64 mismatch: %d != %d", pthreadOnceSizeDarwinArm64, pthreadOnceSizeReal)
		}
		if pthreadMutexSizeDarwinArm64 != pthreadMutexSizeReal {
			t.Fatalf("pthreadMutexSizeDarwinArm64 mismatch: %d != %d", pthreadMutexSizeDarwinArm64, pthreadMutexSizeReal)
		}
		if pthreadMutexAttrSizeDarwinArm64 != pthreadMutexAttrSizeReal {
			t.Fatalf("pthreadMutexAttrSizeDarwinArm64 mismatch: %d != %d", pthreadMutexAttrSizeDarwinArm64, pthreadMutexAttrSizeReal)
		}
		if pthreadCondSizeDarwinArm64 != pthreadCondSizeReal {
			t.Fatalf("pthreadCondSizeDarwinArm64 mismatch: %d != %d", pthreadCondSizeDarwinArm64, pthreadCondSizeReal)
		}
		if pthreadCondAttrSizeDarwinArm64 != pthreadCondAttrSizeReal {
			t.Fatalf("pthreadCondAttrSizeDarwinArm64 mismatch: %d != %d", pthreadCondAttrSizeDarwinArm64, pthreadCondAttrSizeReal)
		}
		if pthreadRWLockSizeDarwinArm64 != pthreadRWLockSizeReal {
			t.Fatalf("pthreadRWLockSizeDarwinArm64 mismatch: %d != %d", pthreadRWLockSizeDarwinArm64, pthreadRWLockSizeReal)
		}
		if pthreadRWLockAttrSizeDarwinArm64 != pthreadRWLockAttrSizeReal {
			t.Fatalf("pthreadRWLockAttrSizeDarwinArm64 mismatch: %d != %d", pthreadRWLockAttrSizeDarwinArm64, pthreadRWLockAttrSizeReal)
		}
		if clockSizeDarwinArm64 != clockSizeReal {
			t.Fatalf("clockSizeDarwinArm64 mismatch: %d != %d", clockSizeDarwinArm64, clockSizeReal)
		}
		if sigjmpBufSizeDarwinArm64 != sigjmpBufSizeReal {
			t.Fatalf("sigjmpBufSizeDarwinArm64 mismatch: %d != %d", sigjmpBufSizeDarwinArm64, sigjmpBufSizeReal)
		}
		if jmpBufSizeDarwinArm64 != jmpBufSizeReal {
			t.Fatalf("jmpBufSizeDarwinArm64 mismatch: %d != %d", jmpBufSizeDarwinArm64, jmpBufSizeReal)
		}
	default:
		t.Fatalf("Unsupported platform: %s, %s", runtime.GOOS, runtime.GOARCH)
	}

	// Sync package size checks
	onceSize, mutexSize, mutexAttrSize, condSize, condAttrSize, rwlockSize, rwlockAttrSize := getPlatformPthreadSizes()
	clockSize := getPlatformClockSizes()
	sigjmpBufSize, jmpBufSize := getPlatformJmpBufSizes()

	if sync.PthreadOnceSize != onceSize {
		t.Fatalf("PthreadOnceSize mismatch: %d != %d", sync.PthreadOnceSize, onceSize)
	}
	if sync.PthreadMutexSize != mutexSize {
		t.Fatalf("PthreadMutexSize mismatch: %d != %d", sync.PthreadMutexSize, mutexSize)
	}
	if sync.PthreadMutexAttrSize != mutexAttrSize {
		t.Fatalf("PthreadMutexAttrSize mismatch: %d != %d", sync.PthreadMutexAttrSize, mutexAttrSize)
	}
	if sync.PthreadCondSize != condSize {
		t.Fatalf("PthreadCondSize mismatch: %d != %d", sync.PthreadCondSize, condSize)
	}
	if sync.PthreadCondAttrSize != condAttrSize {
		t.Fatalf("PthreadCondAttrSize mismatch: %d != %d", sync.PthreadCondAttrSize, condAttrSize)
	}
	if sync.PthreadRWLockSize != rwlockSize {
		t.Fatalf("PthreadRWLockSize mismatch: %d != %d", sync.PthreadRWLockSize, rwlockSize)
	}
	if sync.PthreadRWLockAttrSize != rwlockAttrSize {
		t.Fatalf("PthreadRWLockAttrSize mismatch: %d != %d", sync.PthreadRWLockAttrSize, rwlockAttrSize)
	}
	if time.ClockTSize != clockSize {
		t.Fatalf("ClockSize mismatch: %d != %d", time.ClockTSize, clockSize)
	}
	if setjmp.SigjmpBufSize != sigjmpBufSize {
		t.Fatalf("SigjmpBufSize mismatch: %d != %d", setjmp.SigjmpBufSize, sigjmpBufSize)
	}
	if setjmp.JmpBufSize != jmpBufSize {
		t.Fatalf("JmpBufSize mismatch: %d != %d", setjmp.JmpBufSize, jmpBufSize)
	}
}
