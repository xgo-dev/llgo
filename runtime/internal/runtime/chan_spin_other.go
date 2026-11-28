//go:build !llgo || (!arm64 && !amd64 && !386)

package runtime

const chanSignalSpinLimit = 0

func chanSignalSpinHint() {}
