package main

import "runtime"

type object struct {
	next  *object
	value int
}

var global *object

//go:noinline
func makeRoot() *object {
	return &object{next: &object{value: 41}, value: 7}
}

//go:noinline
func snapshot() int {
	local := makeRoot()
	for i := 0; i < 40; i++ {
		temporary := make([]*object, 1300+(i&31))
		temporary[i] = &object{value: i}
		runtime.GC()
		if temporary[i].value != i {
			panic("allocation in slice was lost")
		}
		runtime.KeepAlive(temporary)
	}
	return local.value + local.next.value + global.next.value
}

//go:noinline
func recoverRoot() (value int) {
	local := makeRoot()
	defer func() {
		if recover() != "restore compiler root frame" {
			panic("panic value lost")
		}
		runtime.GC()
		value = local.value + local.next.value
	}()
	runtime.GC()
	panic("restore compiler root frame")
}

func main() {
	global = makeRoot()
	if recoverRoot() != 48 {
		panic("root lost after recovery")
	}
	println("live stack and globals", snapshot())
}
