package gotest

import "testing"

func TestRuntimeSharedStorePanicLines(t *testing.T) {
	checkRecoveredStorePanicLine(t)
	for _, branch := range []bool{false, true} {
		checkRecoveredPanicLine(t, "runtimeStatementSharedStore", 901, func() {
			runtimeStatementSharedStore(runtimeStatementNilPointer, branch)
		})
	}
}

//go:noinline
func runtimeStatementSharedStore(pointer *int, branch bool) {
	if branch {
//line shared_store.go:901
		*pointer = 1
	} else {
//line shared_store.go:901
		*pointer = 2
	}
}
