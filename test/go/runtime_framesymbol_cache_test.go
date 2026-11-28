package gotest

import (
	"runtime"
	"strings"
	"testing"
)

func validSymbolFrame(pc uintptr) bool {
	f, _ := runtime.CallersFrames([]uintptr{pc}).Next()
	return f.Entry != 0 && f.Function != "" && !strings.HasPrefix(f.Function, "pc=0x")
}

func measureCallersFramesAlloc(pcs []uintptr, iters int) uint64 {
	for i := 0; i < 32; i++ {
		frames := runtime.CallersFrames(pcs)
		for {
			f, more := frames.Next()
			if f.Function == "" {
				panic("missing frame function")
			}
			if !more {
				break
			}
		}
	}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for i := 0; i < iters; i++ {
		frames := runtime.CallersFrames(pcs)
		for {
			f, more := frames.Next()
			if f.Function == "" {
				panic("missing frame function")
			}
			if !more {
				break
			}
		}
	}
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}

func makeSamePCs(pc uintptr, n int) []uintptr {
	out := make([]uintptr, n)
	for i := range out {
		out[i] = pc
	}
	return out
}

// TestCallersFramesStrideCacheNoThrash guards against direct-mapped cache
// thrashing when active stack frames share the same low PC bits modulo 16 KiB
// (4096 * 4-byte instructions on arm64) or land in the same 4-way set.
func TestCallersFramesStrideCacheNoThrash(t *testing.T) {
	var stack [8]uintptr
	n := runtime.Callers(1, stack[:])
	if n == 0 {
		t.Fatal("runtime.Callers returned no frames")
	}
	p0 := stack[0]
	if !validSymbolFrame(p0) {
		t.Fatalf("caller PC %#x did not symbolize", p0)
	}

	const stride = uintptr(4096 * 4) // 16 KiB: old (pc>>2)&4095 period
	var p1 uintptr
	for k := uintptr(1); k <= 64; k++ {
		if cand := p0 + k*stride; validSymbolFrame(cand) {
			p1 = cand
			break
		}
		if p0 > k*stride+4096 {
			if cand := p0 - k*stride; validSymbolFrame(cand) {
				p1 = cand
				break
			}
		}
	}
	if p1 == 0 {
		t.Skip("could not find a second text PC congruent modulo 16 KiB")
	}

	const iters = 2048
	baseline := measureCallersFramesAlloc([]uintptr{p0, p0}, iters)
	strided := measureCallersFramesAlloc([]uintptr{p0, p1}, iters)
	if strided > baseline+128<<10 {
		t.Fatalf("strided CallersFrames(%#x, %#x) allocated %d bytes vs baseline %d bytes over %d walks",
			p0, p1, strided, baseline, iters)
	}

	// Also exercise 4-way associativity within a single folded cache set.
	setHash := func(pc uintptr) uintptr {
		p := pc - 1
		return ((p >> 2) ^ (p >> 12)) & 1023
	}
	targetSet := setHash(p0)
	sameSet := []uintptr{p0}
	for off := uintptr(4); off <= 1<<18 && len(sameSet) < 4; off += 4 {
		cands := []uintptr{p0 + off}
		if p0 > off+4096 {
			cands = append(cands, p0-off)
		}
		for _, cand := range cands {
			if setHash(cand) == targetSet && validSymbolFrame(cand) {
				sameSet = append(sameSet, cand)
				if len(sameSet) == 4 {
					break
				}
			}
		}
	}
	if len(sameSet) >= 2 {
		sameBaseline := measureCallersFramesAlloc(makeSamePCs(p0, len(sameSet)), iters)
		sameAlloc := measureCallersFramesAlloc(sameSet, iters)
		if sameAlloc > sameBaseline+128<<10 {
			t.Fatalf("same-set CallersFrames(%#x) allocated %d bytes vs baseline %d bytes over %d walks",
				sameSet, sameAlloc, sameBaseline, iters)
		}
	}
}
