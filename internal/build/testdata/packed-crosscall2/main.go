package main

/*
#cgo CFLAGS: -pthread
#cgo LDFLAGS: -pthread
void *callback_entry_address(void);
void *foreign_entry_address(void);
*/
import "C"

import (
	"os"
	"runtime"
	"unsafe"
)

//go:linkname runtimeCgocall runtime.cgocall
func runtimeCgocall(fn, arg unsafe.Pointer) int32

type callbackFrame struct {
	code                 unsafe.Pointer
	input, depth, result uint64
	callbacks            uint64
}

var callbackEntry unsafe.Pointer
var captured func(uint64) uint64

func makeCaptured() func(uint64) uint64 {
	state := new([64]uint64)
	state[0], state[63] = 0x12345678, 0x100000001
	return func(input uint64) uint64 {
		return input + state[0] + state[63]
	}
}

// This is a noncapturing wrapper. The managed function it dispatches to has
// captures; they are not smuggled into the raw code pointer's calling ABI.
func callbackDispatch(raw unsafe.Pointer) {
	frame := (*callbackFrame)(raw)
	runtime.GC()
	frame.result = captured(frame.input)
	frame.callbacks = 1
	if frame.depth != 0 {
		nested := callbackFrame{code: frame.code, input: frame.input + 1}
		if runtimeCgocall(callbackEntry, unsafe.Pointer(&nested)) != -19 {
			panic("nested packed call status")
		}
		frame.result += nested.result
		frame.callbacks += nested.callbacks
		runtime.KeepAlive(&nested)
	}
}

func main() {
	mode := os.Args[1]
	callbackEntry = C.callback_entry_address()
	entry := callbackEntry
	depth := uint64(0)
	switch mode {
	case "same-thread":
	case "nested":
		depth = 1
	case "foreign-thread":
		entry = C.foreign_entry_address()
	default:
		panic("unknown callback mode")
	}

	captured = makeCaptured()
	wrapper := callbackDispatch
	descriptor := *(*unsafe.Pointer)(unsafe.Pointer(&wrapper))
	code := *(*unsafe.Pointer)(descriptor)
	for i := uint64(0); i < 128; i++ {
		frame := callbackFrame{code: code, input: i, depth: depth}
		status := runtimeCgocall(entry, unsafe.Pointer(&frame))
		want := i + 0x12345678 + 0x100000001
		if depth != 0 {
			want += i + 1 + 0x12345678 + 0x100000001
		}
		if status != -19 || frame.result != want || frame.callbacks != depth+1 {
			panic("callback lost frame, captures, nesting, or return")
		}
		runtime.KeepAlive(&frame)
	}
	runtime.KeepAlive(wrapper)
	println(mode, "verified 128")
}
