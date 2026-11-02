package runtime

import "sync/atomic"

// Initialize before application package initialization. Reading the environment
// lazily on the first query would incorrectly observe an os.Setenv in init.
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
	if n := parseGOMAXPROCS(gomaxprocsEnv()); n > 0 {
		return n
	}
	return gomaxprocsDefault()
}

func gomaxprocsDefault() int {
	if n := int(c_maxprocs()); n > 0 {
		return n
	}
	return 1
}

// Match the positive int32 startup environment accepted by the Go runtime.
// An invalid, zero, negative or overflowing value selects the default.
func parseGOMAXPROCS(value string) int {
	n := 0
	for i := 0; i < len(value); i++ {
		digit := int(value[i]) - '0'
		if digit < 0 || digit > 9 || n > (1<<31-1-digit)/10 {
			return 0
		}
		n = n*10 + digit
	}
	return n
}
