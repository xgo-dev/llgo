package main

import (
	"time"
	_ "unsafe"
)

//go:linkname useAddressSema runtime.darwinUseAddressSema
func useAddressSema() bool

//go:linkname hasTimedAddressWait C.llgo_darwin_has_wait_uint32_timeout
func hasTimedAddressWait() int32

//go:linkname semacquire sync.runtime_Semacquire
func semacquire(*uint32)

//go:linkname semrelease sync.runtime_Semrelease
func semrelease(*uint32, bool, int)

func main() {
	if useAddressSema() || hasTimedAddressWait() != 0 {
		panic("address wait symbols still available")
	}
	var sema uint32
	started := make(chan struct{})
	acquired := make(chan struct{})
	go func() {
		close(started)
		semacquire(&sema)
		close(acquired)
	}()
	<-started
	const delay = 20 * time.Millisecond
	start := time.Now()
	time.Sleep(delay)
	if time.Since(start) < delay {
		panic("fallback timer fired early")
	}
	select {
	case <-acquired:
		panic("fallback semaphore acquired before release")
	default:
	}
	semrelease(&sema, false, 0)
	select {
	case <-acquired:
	case <-time.After(time.Second):
		panic("fallback semaphore did not wake")
	}
}
