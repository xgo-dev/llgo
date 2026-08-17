package main

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/runtime"
)

const (
	eltSize = int(unsafe.Sizeof(0))
)

func doChan(cap int) {
	c := runtime.NewChan(eltSize, cap)
	go func() {
		v := 100
		runtime.ChanSend(c, unsafe.Pointer(&v), eltSize)
	}()
	var ret int
	runtime.ChanRecv(c, unsafe.Pointer(&ret), eltSize)
	println(ret)
}

func main() {
	doChan(10)
	doChan(0)

	c := runtime.NewChan(eltSize, 3)

	v := 1
	runtime.ChanSend(c, unsafe.Pointer(&v), eltSize)
	v = 2
	runtime.ChanSend(c, unsafe.Pointer(&v), eltSize)
	v = 3
	runtime.ChanSend(c, unsafe.Pointer(&v), eltSize)
	runtime.ChanClose(c)

	v = 10

	if runtime.ChanTrySend(c, unsafe.Pointer(&v), eltSize) {
		println("error: chan send to closed chan")
	}

	for {
		if ok := runtime.ChanRecv(c, unsafe.Pointer(&v), eltSize); !ok {
			break
		}
		println(v)
	}
}
