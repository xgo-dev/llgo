package main

import (
	"os"
	"runtime"
	"runtime/debug"
	"time"
)

//go:noinline
func panicOrigin() { panic("traceback-config") }

//go:noinline
func parkedWorker(ready chan<- struct{}, stop <-chan struct{}) {
	close(ready)
	<-stop
}

func main() {
	if level, ok := os.LookupEnv("LLGO_SET_TRACEBACK"); ok {
		debug.SetTraceback(level)
	}
	if level, ok := os.LookupEnv("LLGO_SET_TRACEBACK_AGAIN"); ok {
		debug.SetTraceback(level)
	}
	if os.Getenv("LLGO_WER_PROBE") != "" {
		probeWER()
		return
	}
	ready, stop := make(chan struct{}), make(chan struct{})
	go parkedWorker(ready, stop)
	<-ready
	time.Sleep(time.Millisecond)
	if mode := os.Getenv("LLGO_STACK_ONLY"); mode != "" {
		buf := make([]byte, 32768)
		os.Stdout.Write(buf[:runtime.Stack(buf, mode == "all")])
		close(stop)
		return
	}
	panicOrigin()
}
