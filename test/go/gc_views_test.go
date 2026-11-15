package gotest

import (
	"runtime"
	"testing"
	"unsafe"
)

type gcViewRecord struct {
	values [128]int
	child  *gcViewRecord
}

var gcViewGarbage *gcViewRecord

//go:noinline
func gcViewPressure() {
	runtime.GC()
	for range 64 {
		gcViewGarbage = new(gcViewRecord)
	}
	runtime.GC()
}

//go:noinline
func gcViewCheck(n int) bool {
	owner := new(gcViewRecord)
	owner.values[3], owner.values[5] = n, n+1
	part := owner.values[2:8][1:]
	array := (*[3]int)(part)
	converted := (*uint)(unsafe.Pointer(&array[2]))
	owner = nil
	gcViewPressure()
	return part[0] == n && *converted == uint(n+1)
}

//go:noinline
func gcViewLoaded() bool {
	holder := &gcViewRecord{child: new(gcViewRecord)}
	holder.child.values[17] = 42
	// A load must start a new owner: retaining holder after this store does
	// not retain the allocation containing the derived pointer.
	part := &holder.child.values[17]
	holder.child = nil
	gcViewPressure()
	return *part == 42
}

func TestGCViews(t *testing.T) {
	done := make(chan bool, 1)
	go func() {
		previous := new(gcViewRecord)
		previous.values[1] = -1
		p := &previous.values[1]
		for i := range 4 {
			next := new(gcViewRecord)
			next.values[1] = i
			q := &next.values[1]
			gcViewPressure()
			if *p != i-1 || *q != i || !gcViewCheck(i) || !gcViewLoaded() {
				done <- false
				return
			}
			p = q // A phi must retain the previous iteration's allocation.
		}
		done <- true
	}()
	if !<-done {
		t.Fatal("GC lost an allocation behind a derived view")
	}
}
