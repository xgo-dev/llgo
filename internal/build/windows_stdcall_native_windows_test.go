//go:build go1.27 && windows

package build

import (
	"runtime"
	"testing"
)

func TestWindowsStdCallCompleteNativePackage(t *testing.T) {
	windowsStdCallCompletePackage(t, runtime.GOARCH)
}
