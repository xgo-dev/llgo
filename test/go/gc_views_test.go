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

// Keep the tested records on the heap even when their callers only return bool.
//
//go:noinline
func gcViewNew() *gcViewRecord { return new(gcViewRecord) }

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
	owner := gcViewNew()
	owner.values[3], owner.values[5] = n, n+1
	// Plan the owner at this safepoint before forming the views. Ordinary
	// builds can then drop the views' separate roots at the next collection.
	gcViewPressure()
	part := owner.values[2:8][1:]
	array := (*[3]int)(part)
	converted := (*uint)(unsafe.Pointer(&array[2]))
	owner = nil
	gcViewPressure()
	return part[0] == n && *converted == uint(n+1)
}

//go:noinline
func gcViewLoaded() bool {
	holder := gcViewNew()
	holder.child = gcViewNew()
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
		previous := gcViewNew()
		previous.values[1] = -1
		p := &previous.values[1]
		for i := range 4 {
			next := gcViewNew()
			next.values[1] = i
			gcViewPressure() // Keep next planned while p retains the old object.
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
