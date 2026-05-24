//go:build darwin || linux

package runtime

// StackRecord is a minimal placeholder for runtime/pprof.
type StackRecord struct {
	Stack []uintptr
}

// MemProfileRecord is a minimal placeholder for runtime/pprof.
type MemProfileRecord struct {
	AllocBytes   int64
	FreeBytes    int64
	AllocObjects int64
	FreeObjects  int64
	Stack        []uintptr
}

func (r *MemProfileRecord) InUseBytes() int64 {
	return r.AllocBytes - r.FreeBytes
}

func (r *MemProfileRecord) InUseObjects() int64 {
	return r.AllocObjects - r.FreeObjects
}

// BlockProfileRecord is a minimal placeholder for runtime/pprof.
type BlockProfileRecord struct {
	Count  int64
	Cycles int64
	Stack  []uintptr
}

func MemProfile(p []MemProfileRecord, inuseZero bool) (n int, ok bool) {
	return 0, false
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
