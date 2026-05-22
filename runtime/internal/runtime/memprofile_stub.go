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

func MemProfileSyntheticFrame(pc uintptr) (function string, line int, ok bool) {
	return "", 0, false
}

func MemProfileSnapshot() []MemProfileRecord {
	return nil
}
