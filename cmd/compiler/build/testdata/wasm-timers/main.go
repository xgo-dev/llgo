package main

import (
	"os"
	"time"
)

func main() {
	testProcessState()
	testClock()
	testSleep()
	testTimerStopAndReset()
	testTicker()
	testAfterFunc()
	testTimeoutSelect()
	println("wasm timers ok")
}

func testProcessState() {
	if len(os.Args) == 0 || os.Args[0] == "" {
		panic("missing process arguments")
	}
	if got := os.Getenv("LLGO_WASM_TEST_ENV"); got != "wasm-env-ok" {
		panic("missing initial process environment")
	}
	if err := os.Setenv("LLGO_WASM_MUTATED_ENV", "set"); err != nil || os.Getenv("LLGO_WASM_MUTATED_ENV") != "set" {
		panic("Setenv failed")
	}
	if err := os.Unsetenv("LLGO_WASM_MUTATED_ENV"); err != nil || os.Getenv("LLGO_WASM_MUTATED_ENV") != "" {
		panic("Unsetenv failed")
	}
}

func testClock() {
	name, offset := time.Now().Zone()
	if name == "" || offset < -24*60*60 || offset > 24*60*60 {
		panic("invalid local time zone")
	}
}

func testSleep() {
	start := time.Now()
	time.Sleep(5 * time.Millisecond)
	if time.Since(start) < 4*time.Millisecond {
		panic("Sleep returned early")
	}
}

func testTimerStopAndReset() {
	timer := time.NewTimer(100 * time.Millisecond)
	if !timer.Stop() {
		panic("active timer Stop returned false")
	}
	if timer.Stop() {
		panic("stopped timer Stop returned true")
	}
	if timer.Reset(5 * time.Millisecond) {
		panic("stopped timer Reset returned true")
	}
	<-timer.C
	if timer.Stop() {
		panic("expired timer Stop returned true")
	}

	timer = time.NewTimer(100 * time.Millisecond)
	if !timer.Reset(5 * time.Millisecond) {
		panic("active timer Reset returned false")
	}
	<-timer.C
	if timer.Stop() {
		panic("reset timer Stop returned true after expiry")
	}
}

func testTicker() {
	ticker := time.NewTicker(3 * time.Millisecond)
	for i := 0; i < 3; i++ {
		<-ticker.C
	}
	ticker.Stop()
}

func testAfterFunc() {
	done := make(chan int)
	time.AfterFunc(5*time.Millisecond, func() {
		done <- 42
	})
	if value := <-done; value != 42 {
		panic("AfterFunc result mismatch")
	}
}

func testTimeoutSelect() {
	blocked := make(chan struct{})
	select {
	case <-blocked:
		panic("blocked channel became ready")
	case <-time.After(5 * time.Millisecond):
	}
}
