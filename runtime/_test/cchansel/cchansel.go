package main

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/runtime"
)

const (
	eltSize = int(unsafe.Sizeof(0))
)

func fibonacci(c, quit *runtime.Chan) {
	x, y := 0, 1
	for {
		isel, _ := runtime.Select(
			runtime.ChanOp{C: c, Send: true, Val: unsafe.Pointer(&x), Size: int32(eltSize)},
			runtime.ChanOp{C: quit},
		)
		if isel == 0 {
			x, y = y, x+y
		} else {
			println("quit")
			return
		}
	}
}

func main() {
	c := runtime.NewChan(eltSize, 0)
	quit := runtime.NewChan(eltSize, 0)
	go func() {
		for i := 0; i < 10; i++ {
			val := 0
			runtime.ChanRecv(c, unsafe.Pointer(&val), eltSize)
			println(val)
		}
		runtime.ChanClose(quit)
	}()
	fibonacci(c, quit)
}
