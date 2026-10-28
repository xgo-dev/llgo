package main

import (
	"fmt"
	"runtime"
	"unsafe"

	"github.com/ebitengine/purego"
)

func main() {
	h, err := purego.Dlopen("libc.so.6", purego.RTLD_NOW|purego.RTLD_LOCAL)
	if err != nil {
		panic(err)
	}
	defer purego.Dlclose(h)
	var strlen func(string) uintptr
	purego.RegisterLibFunc(&strlen, h, "strlen")
	if n := strlen("purego ABI boundary"); n != 19 {
		panic(n)
	}
	fmt.Println("purego strlen verified")
	var qsort func(unsafe.Pointer, uintptr, uintptr, uintptr)
	purego.RegisterLibFunc(&qsort, h, "qsort")
	for _, direction := range []int32{1, -1} {
		callback := comparator(direction)
		runtime.GC()
		values := []int64{9, -5, 2, 2, 0}
		qsort(unsafe.Pointer(&values[0]), uintptr(len(values)), 8, callback)
		for i := 1; i < len(values); i++ {
			if (direction > 0 && values[i-1] > values[i]) || (direction < 0 && values[i-1] < values[i]) {
				panic(fmt.Sprintf("qsort: %v", values))
			}
		}
		runtime.KeepAlive(values)
	}
	fmt.Println("purego qsort closures, CDecl and GC verified")

	math, err := purego.Dlopen("libm.so.6", purego.RTLD_NOW|purego.RTLD_LOCAL)
	if err != nil {
		panic(err)
	}
	defer purego.Dlclose(math)
	var scale func(float64, int32) float64
	purego.RegisterLibFunc(&scale, math, "scalbn")
	if got := scale(1.5, 3); got != 12 {
		panic(got)
	}
	fmt.Println("purego mixed float call verified")
}

func comparator(direction int32) uintptr {
	return purego.NewCallback(func(_ purego.CDecl, left, right *int64) int32 {
		runtime.GC()
		if *left < *right {
			return -direction
		}
		if *left > *right {
			return direction
		}
		return 0
	})
}
