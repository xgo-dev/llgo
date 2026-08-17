package main

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/clite/bdwgc"
	"github.com/xgo-dev/llgo/runtime/internal/clite/bdwgc/_test/testing"
)

// ------ Test malloc ------

func TestMalloc(t *testing.T) {
	pn := (*int)(bdwgc.Malloc(unsafe.Sizeof(int(0))))
	*pn = 1 << 30
	c.Printf(c.Str("value: %d, %x, %p, %p\n"), *pn, *pn, pn, &pn)

	pl := (*int64)(bdwgc.Realloc(c.Pointer(pn), unsafe.Sizeof(int64(0))))
	*pl = 1 << 60
	c.Printf(c.Str("value: %lld, %llx, %p, %p\n"), *pl, *pl, pl, &pl)

	bdwgc.Free(c.Pointer(pl))
}

// ------ Test finalizer ------

const (
	RETURN_VALUE_FREED = 1 << 31
)

var called uint = 0

func setReturnValueFreed(pobj c.Pointer, clientData c.Pointer) {
	called |= RETURN_VALUE_FREED
	c.Printf(c.Str("called: %x\n"), called)
}

func setLoopValueFreed(pobj c.Pointer, clientData c.Pointer) {
	pmask := (*uint)(clientData)
	called |= *pmask
	c.Printf(c.Str("called: %x\n"), called)
}

func isCalled(mask uint) bool {
	return called&mask != 0
}

func returnValue() *int {
	pn := bdwgc.Malloc(unsafe.Sizeof(int(0)))
	bdwgc.RegisterFinalizer(pn, setReturnValueFreed, nil, nil, nil)
	return (*int)(pn)
}

func callFunc() {
	pn := returnValue()
	*pn = 1 << 30
	c.Printf(c.Str("value: %d, %x, %p, %p\n"), *pn, *pn, pn, &pn)
	bdwgc.Gcollect()
	check(!isCalled(RETURN_VALUE_FREED), c.Str("finalizer should not be called"))
}

func loop() {
	for i := 0; i < 5; i++ {
		p := bdwgc.Malloc(unsafe.Sizeof(int(0)))
		pn := (*int)(p)
		*pn = i
		c.Printf(c.Str("value: %d, %x, %p, %p\n"), *pn, *pn, pn, &pn)
		pflag := (*uint)(c.Malloc(unsafe.Sizeof(uint(0))))
		*pflag = 1 << i
		bdwgc.RegisterFinalizer(p, setLoopValueFreed, c.Pointer(pflag), nil, nil)
		bdwgc.Gcollect()
		check(!isCalled(1<<i), c.Str("finalizer should not be called"))
		for j := 0; j < i; j++ {
			check(isCalled(1<<j), c.Str("finalizers of previous objects should be called"))
		}
	}
}

// Clear stack to avoid reference
func clearStack() {
	p := c.Alloca(128)
	c.Memset(p, 0, 128)
}

func check(b bool, msg *c.Char) {
	if !b {
		c.Printf(c.Str("check failed: %s\n"), msg)
		panic("check failed")
	}
}

func TestFinalizer(t *testing.T) {
	bdwgc.Init()

	callFunc()
	clearStack()
	bdwgc.Gcollect()
	check(isCalled(RETURN_VALUE_FREED), c.Str("finalizer should be called"))

	loop()
}
