//go:build !llgo || (!arm64 && !amd64 && !386)

package runtime

// Procyield is a portable fallback for targets without an inline spin hint.
func Procyield(cycles uint32) {}
