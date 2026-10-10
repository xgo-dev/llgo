//go:build windows && 386

package runtime

// Win32 has no PE unwind table that can validate a guessed function boundary.
// The compiler therefore emits exact entry PCs for address-taken functions;
// accepting a post-prologue look-ahead here could misclassify an ordinary PC
// near the end of one function as the next function.
const runtimeFuncPCEntrySlack = 0

const runtimeFuncPCDedupBySiteEntry = false

func runtimeFuncPCSiteEntry(pc uintptr) uintptr {
	return pc
}

func runtimeFuncPCMayUseEntrySlack(uintptr) bool {
	return false
}
