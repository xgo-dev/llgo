package main

import (
	"runtime"
	"unsafe"
)

//go:linkname currentGForTesting github.com/goplus/llgo/runtime/internal/runtime.CurrentGForTesting
func currentGForTesting() unsafe.Pointer

//go:linkname parkForTesting github.com/goplus/llgo/runtime/internal/runtime.ParkForTesting
func parkForTesting()

//go:linkname readyForTesting github.com/goplus/llgo/runtime/internal/runtime.ReadyForTesting
func readyForTesting(unsafe.Pointer)

//go:linkname schedulerStateForTesting github.com/goplus/llgo/runtime/internal/runtime.SchedulerStateForTesting
func schedulerStateForTesting() (runq uintptr, mid int64, pid int32)

//go:linkname gmpForTesting github.com/goplus/llgo/runtime/internal/runtime.GMPForTesting
func gmpForTesting() (goid, parentGoid uint64, mid int64, pid int32, gstatus, pstatus uint32, linked bool)

var (
	parked     unsafe.Pointer
	mainMID    int64
	mainPID    int32
	mainGID    uint64
	seenG      [4]uint64
	seenGCount int
	eventLog   [8]int
	eventCount int
	done       int
)

func event(value int) {
	eventLog[eventCount] = value
	eventCount++
}

func checkCurrentG() {
	goid, parent, mid, pid, gstatus, pstatus, linked := gmpForTesting()
	if goid == 0 || goid == mainGID || parent != mainGID {
		panic("invalid goroutine identity")
	}
	if mid != mainMID || pid != mainPID {
		panic("goroutine did not reuse the single worker M/P")
	}
	if gstatus != 2 || pstatus != 1 || !linked {
		panic("invalid running G/M/P state")
	}
	for i := 0; i < seenGCount; i++ {
		if seenG[i] == goid {
			panic("duplicate goroutine identity")
		}
	}
	seenG[seenGCount] = goid
	seenGCount++
}

func main() {
	checkWasmModel()
	var (
		gstatus uint32
		pstatus uint32
		linked  bool
	)
	mainGID, _, mainMID, mainPID, gstatus, pstatus, linked = gmpForTesting()
	if mainGID == 0 || mainMID == 0 || mainPID < 0 || gstatus != 2 || pstatus != 1 || !linked {
		panic("invalid main G/M/P state")
	}

	go func() {
		checkCurrentG()
		event(1)
		parked = currentGForTesting()
		parkForTesting()
		event(8)
		done++
	}()

	go func() {
		checkCurrentG()
		event(2)
		runtime.Gosched()
		event(6)
		readyForTesting(parked)
		event(7)
		done++
	}()

	go func() {
		checkCurrentG()
		defer func() {
			if recover() != "expected panic" {
				panic("unexpected recover value")
			}
			event(3)
			done++
		}()
		panic("expected panic")
	}()

	go func() {
		checkCurrentG()
		defer func() {
			event(4)
			done++
		}()
		runtime.Goexit()
		panic("Goexit returned")
	}()

	if runq, mid, pid := schedulerStateForTesting(); runq != 4 || mid != mainMID || pid != mainPID {
		panic("invalid initial scheduler state")
	}
	event(0)
	for done != 4 {
		runtime.Gosched()
	}

	want := [...]int{0, 1, 2, 3, 4, 6, 7, 8}
	if eventCount != len(want) {
		panic("unexpected event count")
	}
	for i, value := range want {
		if eventLog[i] != value {
			panic("unexpected scheduler order")
		}
	}
	if seenGCount != len(seenG) {
		panic("not all goroutines ran")
	}
	println("wasm scheduler ok")
}
