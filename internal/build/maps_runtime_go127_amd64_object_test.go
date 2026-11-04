//go:build go1.27 && (amd64 || llgo_amd64_cross)

package build

import (
	"runtime"
	"testing"

	xtoolllvm "github.com/xgo-dev/llgo/internal/xtool/llvm"
)

// Native AMD64 CI owns this complete selected-SDK package. Other hosts require
// the explicit cross role and actual AMD64 tools.
func TestMapsRuntimeAMD64CompletePackage(t *testing.T) {
	t.Setenv("CCFLAGS", "--target="+xtoolllvm.GetTargetTriple(runtime.GOOS, "amd64"))
	mapsRuntimeCompletePackage(t, runtime.GOOS, "amd64")
}
