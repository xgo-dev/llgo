//go:build linux || darwin || windows

package gotest

import (
	"os"
	"os/exec"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"testing"
	"unsafe"
)

//go:noinline
func tracebackFault(depth int, p *byte) byte {
	if depth == 0 {
		return *p
	}
	return 1 + tracebackFault(depth-1, p)
}

func TestRuntimeCallersDeepFault(t *testing.T) {
	// Match the other Windows fault tests: isolate the upstream Go exception
	// recovery issue (golang/go#81238) from unrelated tests in this process.
	if runtime.GOOS == "windows" && os.Getenv("LLGO_DEEP_FAULT_CHILD") != "1" {
		cmd := exec.Command(os.Args[0], "-test.run=^TestRuntimeCallersDeepFault$")
		cmd.Env = append(os.Environ(), "LLGO_DEEP_FAULT_CHILD=1")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("deep fault child: %v\n%s", err, out)
		}
		return
	}
	oldGC := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(oldGC)

	// A protected page exercises an actual hardware exception even when the
	// compiler emits explicit checks for a nil Go pointer.
	page, _ := protectedMemory(t, 1, 0, 1)
	var workers sync.WaitGroup
	for i := 0; i < 4; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			old := debug.SetPanicOnFault(true)
			defer debug.SetPanicOnFault(old)
			ensureWindowsExceptionStackHeadroom()
			for repeat := 0; repeat < 3; repeat++ {
				func() {
					defer func() {
						if recover() == nil {
							t.Error("protected memory did not panic")
							return
						}
						pcs := make([]uintptr, 1024)
						n := runtime.Callers(0, pcs)
						frames := runtime.CallersFrames(pcs[:n])
						found := 0
						for {
							f, more := frames.Next()
							if strings.HasSuffix(f.Function, ".tracebackFault") {
								found++
							}
							if !more {
								break
							}
						}
						if found != 251 {
							t.Errorf("fault Callers returned %d recursive frames, want 251", found)
						}
					}()
					tracebackFault(250, (*byte)(unsafe.Pointer(&page[0])))
				}()
				if stack := string(debug.Stack()); strings.Contains(stack, ".tracebackFault(") {
					t.Errorf("stale fault snapshot after recovery:\n%s", stack)
				}
			}
		}()
	}
	workers.Wait()
}
