package runtime

import (
	"sync/atomic"

	llruntime "github.com/xgo-dev/llgo/runtime/internal/runtime"
)

// Initialize before application package initialization. Reading the environment
// lazily on the first query would incorrectly observe an os.Setenv in init.
// Use a machine word to preserve every positive int accepted by the setter;
// only the startup environment parser has an int32 bound.
var gomaxprocsSetting atomic.Uintptr

func init() {
	gomaxprocsSetting.Store(uintptr(initialGOMAXPROCS()))
}

// GOMAXPROCS records the requested setting and returns its previous value.
// Nonpositive requests only query it. LLGo's scheduler backends do not currently
// enforce this setting as an execution-parallelism limit.
func GOMAXPROCS(n int) int {
	if n <= 0 {
		return int(gomaxprocsSetting.Load())
	}
	return int(gomaxprocsSetting.Swap(uintptr(n)))
}

func initialGOMAXPROCS() int {
	if n := llruntime.ParseGOMAXPROCS(gomaxprocsEnv()); n > 0 {
		return n
	}
	return gomaxprocsDefault()
}

func gomaxprocsDefault() int {
	return llruntime.CPUCount()
}
