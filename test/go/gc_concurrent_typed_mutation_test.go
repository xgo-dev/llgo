//go:build !baremetal && !nogc

package gotest

import (
	"runtime"
	"runtime/debug"
	"strconv"
	"sync"
	"testing"
)

type gcMutationStruct struct {
	boolean bool
	u8      uint8
	u16     uint16
	u32     uint32
	u64     uint64
	i8      int8
	i16     int16
	i32     int32
	i64     int64
	value   *int
	f32     float32
	f64     float64
	c64     complex64
	c128    complex128
	text    string
	b       byte
	r       rune
	u       uint
	number  int
	uip     uintptr
}

// TestGCConcurrentTypedMutation covers the typed stores in GOROOT/test/gcgort.go
// without spawning a native thread for every allocation or channel operation.
func TestGCConcurrentTypedMutation(t *testing.T) {
	oldPercent := debug.SetGCPercent(1)
	defer debug.SetGCPercent(oldPercent)
	collect, gcDone := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(gcDone)
		for range collect {
			runtime.GC()
		}
	}()
	defer func() {
		close(collect)
		<-gcDone
	}()

	checkGCConcurrentMutation(t, "bool", collect, func(i int) bool { return i%2 == 0 })
	checkGCConcurrentMutation(t, "uint8", collect, func(i int) uint8 { return uint8(i) })
	checkGCConcurrentMutation(t, "uint16", collect, func(i int) uint16 { return uint16(i) })
	checkGCConcurrentMutation(t, "uint32", collect, func(i int) uint32 { return uint32(i) })
	checkGCConcurrentMutation(t, "uint64", collect, func(i int) uint64 { return uint64(i) })
	checkGCConcurrentMutation(t, "int8", collect, func(i int) int8 { return int8(i) })
	checkGCConcurrentMutation(t, "int16", collect, func(i int) int16 { return int16(i) })
	checkGCConcurrentMutation(t, "int32", collect, func(i int) int32 { return int32(i) })
	checkGCConcurrentMutation(t, "int64", collect, func(i int) int64 { return int64(i) })
	checkGCConcurrentMutation(t, "float32", collect, func(i int) float32 { return float32(i) + 0.5 })
	checkGCConcurrentMutation(t, "float64", collect, func(i int) float64 { return float64(i) + 0.5 })
	checkGCConcurrentMutation(t, "complex64", collect, func(i int) complex64 { return complex(float32(i), 1) })
	checkGCConcurrentMutation(t, "complex128", collect, func(i int) complex128 { return complex(float64(i), 1) })
	checkGCConcurrentMutation(t, "byte", collect, func(i int) byte { return byte(i) })
	checkGCConcurrentMutation(t, "rune", collect, func(i int) rune { return rune(i) })
	checkGCConcurrentMutation(t, "uint", collect, func(i int) uint { return uint(i) })
	checkGCConcurrentMutation(t, "int", collect, func(i int) int { return i })
	checkGCConcurrentMutation(t, "uintptr", collect, func(i int) uintptr { return uintptr(i) })
	checkGCConcurrentMutation(t, "string", collect, func(i int) string { return strconv.Itoa(i + 1000000) })
	checkGCConcurrentMutation(t, "struct", collect, func(i int) gcMutationStruct {
		value := new(int)
		*value = i
		return gcMutationStruct{
			boolean: i%2 == 0,
			u8:      uint8(i),
			u16:     uint16(i),
			u32:     uint32(i),
			u64:     uint64(i),
			i8:      int8(i),
			i16:     int16(i),
			i32:     int32(i),
			i64:     int64(i),
			value:   value,
			f32:     float32(i) + 0.5,
			f64:     float64(i) + 0.5,
			c64:     complex(float32(i), 1),
			c128:    complex(float64(i), 1),
			text:    strconv.Itoa(i + 1000000),
			b:       byte(i),
			r:       rune(i),
			u:       uint(i),
			number:  i,
			uip:     uintptr(i),
		}
	})
}

func checkGCConcurrentMutation[T comparable](t *testing.T, name string, collect chan<- struct{}, makeValue func(int) T) {
	t.Run(name, func(t *testing.T) {
		// Subtests run sequentially: eight workers and eight senders at a time,
		// plus the shared collector. No allocation or operation creates a goroutine.
		const workers, allocs, mods, length = 8, 8, 8, 9
		var done sync.WaitGroup
		for worker := 0; worker < workers; worker++ {
			done.Add(1)
			go func(worker int) {
				defer done.Done()
				toSend, channel := make(chan T), make(chan T)
				senderDone := make(chan struct{})
				go func() {
					defer close(senderDone)
					for value := range toSend {
						channel <- value
					}
				}()
				defer func() {
					close(toSend)
					<-senderDone
				}()
				for allocation := 0; allocation < allocs; allocation++ {
					var value T
					pointer := new(T)
					var array [length]T
					slice := make([]T, length)
					mapping := make(map[int]T)
					pointerKeys := make(map[*T]T)
					var boxed any
					for modification := 0; modification < mods; modification++ {
						index := modification % length
						want := makeValue((worker*allocs+allocation)*mods + modification)
						value, *pointer, array[index], slice[index] = want, want, want, want
						mapping[index], pointerKeys[pointer] = want, want
						toSend <- want
						boxed = want
						if allocation == 0 && modification == 0 {
							// One request per worker bounds explicit collection to
							// 160 calls, with typed stores and a channel send live.
							collect <- struct{}{}
						}
						runtime.Gosched()
						received := <-channel
						for _, stored := range []struct {
							name  string
							value T
						}{
							{"value", value}, {"pointer", *pointer}, {"array", array[index]}, {"slice", slice[index]},
							{"map", mapping[index]}, {"pointer-key map", pointerKeys[pointer]},
							{"channel", received}, {"interface", boxed.(T)},
						} {
							if stored.value != want {
								t.Errorf("worker %d allocation %d modification %d %s: got %v, want %v",
									worker, allocation, modification, stored.name, stored.value, want)
								return
							}
						}
						if stored, ok := boxed.(gcMutationStruct); ok &&
							(stored.value == nil || *stored.value != stored.number || stored.text != strconv.Itoa(stored.number+1000000)) {
							t.Errorf("worker %d allocation %d modification %d: GC changed data referenced by struct %+v",
								worker, allocation, modification, stored)
							return
						}
					}
				}
			}(worker)
		}
		done.Wait()
	})
}
