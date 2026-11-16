package main

/*
#cgo LDFLAGS: -pthread
void llgo_teardown_arm(void);
void llgo_teardown_await_wait(void);
void llgo_teardown_check_retained(void);
void llgo_teardown_finish_main(void);
int llgo_teardown_foreign_thread(void);
*/
import "C"

import (
	"os"
	"runtime"
)

var values = make(chan int)

//export GoChannelWait
func GoChannelWait() C.int {
	C.llgo_teardown_arm()
	value := <-values
	C.llgo_teardown_check_retained()
	return C.int(value)
}

func main() {
	if len(os.Args) != 2 {
		panic("missing teardown scenario")
	}
	go func() {
		C.llgo_teardown_await_wait()
		values <- 42
		if os.Args[1] == "main-goexit" {
			C.llgo_teardown_finish_main()
		}
	}()
	switch os.Args[1] {
	case "main-goexit":
		C.llgo_teardown_arm()
		if value := <-values; value != 42 {
			panic("wrong channel value")
		}
		// The main G owns a host-TLS lifecycle, so Goexit must delegate
		// cached signal teardown to destroyG through setg(nil).
		C.llgo_teardown_check_retained()
		runtime.Goexit()
	case "foreign-thread":
		// The callback's G survives its C export and is destroyed by the
		// host thread's TLS destructor, before pthread_join returns.
		if C.llgo_teardown_foreign_thread() != 42 {
			panic("wrong foreign callback value")
		}
		println("ok")
	default:
		panic("unknown teardown scenario")
	}
}
