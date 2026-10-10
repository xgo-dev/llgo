//go:build !windows || (!386 && !amd64 && !arm64)

package runtime

// LLVM entry-site anchors can land after the backend-generated function
// prologue. Targets without a specialized Windows entry lookup retain a small
// look-ahead window so function-value PCs still resolve to their metadata.
const runtimeFuncPCEntrySlack = 64

const runtimeFuncPCDedupBySiteEntry = false

func runtimeFuncPCSiteEntry(pc uintptr) uintptr {
	return pc
}

func runtimeFuncPCMayUseEntrySlack(uintptr) bool {
	return true
}
