package gotest

import (
	"runtime"
	"testing"
)

type escapeRootBlock [1024]byte
type escapeRootHolder struct {
	pointer *escapeRootBlock
	other   int
}

//go:noinline
func escapeRootReplace(holder *escapeRootHolder, value byte) {
	holder.pointer = new(escapeRootBlock)
	holder.pointer[0] = value
	runtime.GC()
	runtime.GC()
}

//go:noinline
func escapeRootSliceReplace(values []*escapeRootBlock) {
	for i := range values {
		values[i] = new(escapeRootBlock)
		values[i][0] = byte(i + 1)
	}
	runtime.GC()
	runtime.GC()
}

//go:noinline
func escapeRootMapReplace(values map[int]*escapeRootBlock) {
	for i := 0; i < 16; i++ {
		values[i] = new(escapeRootBlock)
		values[i][0] = byte(i + 3)
	}
	runtime.GC()
	runtime.GC()
}

func TestEscapeTypedStackFields(t *testing.T) {
	var holder escapeRootHolder
	for i := 0; i < 12; i++ {
		escapeRootReplace(&holder, byte(i+1))
		runtime.GC()
		if holder.pointer[0] != byte(i+1) {
			t.Fatal("mutated stack field lost", i, holder.pointer[0])
		}
	}
	values := make([]*escapeRootBlock, 4)
	escapeRootSliceReplace(values)
	runtime.GC()
	for i, value := range values {
		if value[0] != byte(i+1) {
			t.Fatal("slice field", i, value[0])
		}
	}
	entries := make(map[int]*escapeRootBlock)
	escapeRootMapReplace(entries)
	runtime.GC()
	for i := 0; i < 16; i++ {
		if entries[i][0] != byte(i+3) {
			t.Fatal("map field", i, entries[i][0])
		}
	}
}

func TestEscapeTypedStackPanicAndDefer(t *testing.T) {
	for i := 0; i < 12; i++ {
		func() {
			var holder escapeRootHolder
			escapeRootReplace(&holder, byte(i+1))
			defer func() {
				// Collection before recover must trace the panic payload even
				// after the panicking activation has been unwound.
				runtime.GC()
				if got := recover(); got != i {
					t.Errorf("panic=%v want %d", got, i)
				}
				if holder.pointer[0] != byte(i+1) {
					t.Error("deferred stack field lost")
				}
			}()
			defer runtime.GC()
			panic(i)
		}()
	}
}

func TestEscapeTypedStackGoroutineExit(t *testing.T) {
	done := make(chan byte, 1)
	go func() {
		var holder escapeRootHolder
		escapeRootReplace(&holder, 37)
		defer func() { runtime.GC(); done <- holder.pointer[0] }()
		runtime.Goexit()
	}()
	if got := <-done; got != 37 {
		t.Fatal("Goexit lost stack fields", got)
	}
	runtime.GC()
}
