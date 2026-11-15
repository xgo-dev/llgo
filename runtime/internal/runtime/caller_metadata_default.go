//go:build !llgo || !wasm || !llgo_pclntab_external || llgo.wasm.debugger

package runtime

type callerLocationMetadata struct{}

func resolveCallerFrame(frame CallerFrame) CallerFrame         { return frame }
func callerLocationHash(CallerFrame) uintptr                   { return 0 }
func sameCallerLocationMetadata(CallerFrame, CallerFrame) bool { return true }

func hasCallerLocation(CallerFrame) bool { return false }
