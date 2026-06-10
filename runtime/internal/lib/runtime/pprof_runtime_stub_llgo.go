//go:build darwin || linux

package runtime

import llrt "github.com/goplus/llgo/runtime/internal/runtime"

// StackRecord is a minimal placeholder for runtime/pprof.
type StackRecord struct {
	Stack0 [32]uintptr
}

func (r *StackRecord) Stack() []uintptr {
	for i, pc := range r.Stack0 {
		if pc == 0 {
			return r.Stack0[:i]
		}
	}
	return r.Stack0[:]
}

// MemProfileRecord is a minimal placeholder for runtime/pprof.
type MemProfileRecord struct {
	AllocBytes, FreeBytes     int64
	AllocObjects, FreeObjects int64
	Stack0                    [32]uintptr
}

func (r *MemProfileRecord) InUseBytes() int64 {
	return r.AllocBytes - r.FreeBytes
}

func (r *MemProfileRecord) InUseObjects() int64 {
	return r.AllocObjects - r.FreeObjects
}

func (r *MemProfileRecord) Stack() []uintptr {
	for i, pc := range r.Stack0 {
		if pc == 0 {
			return r.Stack0[:i]
		}
	}
	return r.Stack0[:]
}

// BlockProfileRecord is a minimal placeholder for runtime/pprof.
type BlockProfileRecord struct {
	Count  int64
	Cycles int64
	StackRecord
}

func MemProfile(p []MemProfileRecord, inuseZero bool) (n int, ok bool) {
	llrt.SetMemProfileRate(MemProfileRate)
	records := llrt.MemProfileSnapshot()
	n = len(records)
	if len(p) < n {
		return n, false
	}
	for i, r := range records {
		allocObjects, allocBytes := scaleMemProfileSample(r.AllocObjects, r.AllocBytes, MemProfileRate)
		p[i] = MemProfileRecord{
			AllocBytes:   allocBytes,
			AllocObjects: allocObjects,
			Stack0:       r.Stack0,
		}
	}
	return n, true
}

func scaleMemProfileSample(objects, bytes int64, rate int) (int64, int64) {
	if objects <= 0 || bytes <= 0 {
		return 0, 0
	}
	if rate <= 1 {
		return objects, bytes
	}
	avgSize := float64(bytes) / float64(objects)
	prob := 1 - expNeg(avgSize/float64(rate))
	if prob <= 0 {
		return 1, bytes / objects
	}
	sampledObjects := int64(float64(objects)*prob + 0.5)
	if sampledObjects < 1 {
		sampledObjects = 1
	}
	return sampledObjects, sampledObjects * (bytes / objects)
}

func expNeg(x float64) float64 {
	if x <= 0 {
		return 1
	}
	if x > 0.5 {
		y := expNeg(x / 2)
		return y * y
	}
	term := 1.0
	sum := 1.0
	for i := 1; i <= 12; i++ {
		term *= -x / float64(i)
		sum += term
	}
	if sum < 0 {
		return 0
	}
	if sum > 1 {
		return 1
	}
	return sum
}

func BlockProfile(p []BlockProfileRecord) (n int, ok bool) {
	return 0, false
}

func MutexProfile(p []BlockProfileRecord) (n int, ok bool) {
	return 0, false
}

func ThreadCreateProfile(p []StackRecord) (n int, ok bool) {
	return 0, false
}

func NumGoroutine() int {
	return 1
}

func SetCPUProfileRate(hz int) {}

func FuncForPC(pc uintptr) *Func {
	return nil
}
