//go:build llgo

package llgoext

import (
	"fmt"
	"os"
	"runtime"
	_ "unsafe"
)

const (
	mainGoexitLifecycleChild    = "LLGO_TEST_MAIN_GOEXIT_LIFECYCLE"
	mainGoexitLifecycleChildArg = "-llgo.main-goexit-lifecycle-child"
)

//go:linkname runtimeGStateForTesting github.com/xgo-dev/llgo/runtime/internal/runtime.GStateForTesting
func runtimeGStateForTesting() (count uint64, mainExited bool)

func init() {
	child := os.Getenv(mainGoexitLifecycleChild) == "1"
	for _, arg := range os.Args[1:] {
		child = child || arg == mainGoexitLifecycleChildArg
	}
	if !child {
		return
	}
	done := make(chan int, 1)
	defer func() { done <- 0 }()
	go func() {
		<-done
		for {
			count, mainExited := runtimeGStateForTesting()
			if count == 1 && mainExited {
				break
			}
		}
		fmt.Println("WORKER_RETURNING")
	}()
	runtime.Goexit()
}
