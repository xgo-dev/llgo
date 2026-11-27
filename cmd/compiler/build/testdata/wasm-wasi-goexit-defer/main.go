package main

import (
	"os"
	"runtime"
	_ "unsafe"
)

const LLGoFiles = "_wrap/longjmp.c"

//go:linkname cLongjmp C.llgo_wasi_thread_longjmp
func cLongjmp() int32

func init() {
	if len(os.Args) > 1 && os.Args[1] == "init" {
		defer println("wasi init defer ok")
		runtime.Goexit()
	}
}

//go:noinline
func raise(depth int) {
	if depth == 0 {
		panic(42)
	}
	defer func() {}()
	raise(depth - 1)
}

func catch() {
	defer func() {
		if recover() != 42 {
			panic("wrong recovered value")
		}
	}()
	raise(3)
}

//go:noinline
func exit() { runtime.Goexit() }

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "main":
			defer println("wasi main defer ok")
			runtime.Goexit()
		case "uncaught":
			go func() { panic("wasi uncaught sentinel") }()
			select {}
		}
	}
	const workers = 4
	done := make(chan int, workers)
	for range workers {
		go func() {
			for range 20 {
				catch()
				if cLongjmp() != 7 {
					panic("C longjmp did not return to its own thread")
				}
			}
			defer func() {
				// A branch after catching the longjmp must not observe a
				// transient cluster-wide termination flag.
				sum := 0
				for i := range 8 {
					sum += i
				}
				done <- sum
			}()
			exit()
		}()
	}
	for range workers {
		if <-done != 28 {
			panic("defer did not run")
		}
	}
	runtime.GC()
	println("wasi worker defer ok")
}
