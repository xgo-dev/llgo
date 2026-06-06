//go:build !darwin && !linux

package runtime

type MemProfileRecord struct {
	AllocBytes   int64
	AllocObjects int64
	Stack0       [32]uintptr
}

func recordMemProfileAlloc(size uintptr) {
}

func SetMemProfileRate(rate int) {
}

func SetMemProfileRatePtr(rate *int) {
}

func MemProfileSuppress() bool {
	return false
}

func MemProfileRestoreSuppressed(old bool) {
}

func MemProfileEnter(function string) {
}

func MemProfileExit() {
}

func MemProfileSyntheticFrame(pc uintptr) (function string, line int, ok bool) {
	return "", 0, false
}

func MemProfileSnapshot() []MemProfileRecord {
	return nil
}
