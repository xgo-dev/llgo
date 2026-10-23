//go:build llgo && wasm && llgo.wasm.gc.linear

package gotest

import (
	"testing"
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
)

// These are the existing production classifier and pointer dispatcher, not a
// test implementation of the finalizer ABI. This test deliberately bypasses
// collection and the ready queue; the original SetFinalizer tests still own
// reachability, registration, and automatic dispatch coverage.
//
//go:linkname classifyFinalizerResultForTest runtime.classifyWasmFinalizerResult
func classifyFinalizerResultForTest([]*abi.Type) (uint8, uintptr)

//go:linkname callPointerFinalizerForTest runtime.callWasmPointerFinalizer
func callPointerFinalizerForTest(function, object unsafe.Pointer, result uint8, size uintptr)

func TestRuntimeWasmFinalizerResultDispatcher(t *testing.T) {
	const (
		i32Result  = uint8(1)
		sretResult = uint8(5)
	)
	observed := make(chan int, 2)
	tests := []struct {
		name     string
		callback any
		result   uint8
		size     uintptr
	}{
		{
			name: "bool",
			callback: func(p *int) bool {
				observed <- *p
				return true
			},
			result: i32Result,
		},
		{
			name: "int8",
			callback: func(p *int) int8 {
				observed <- *p
				return -128
			},
			result: i32Result,
		},
		{
			name: "uint8",
			callback: func(p *int) uint8 {
				observed <- *p
				return 255
			},
			result: i32Result,
		},
		{
			name: "int16",
			callback: func(p *int) int16 {
				observed <- *p
				return -32768
			},
			result: i32Result,
		},
		{
			name: "uint16",
			callback: func(p *int) uint16 {
				observed <- *p
				return 65535
			},
			result: i32Result,
		},
		{
			name: "int32",
			callback: func(p *int) int32 {
				observed <- *p
				return -2147483648
			},
			result: i32Result,
		},
		{
			name: "uint32",
			callback: func(p *int) uint32 {
				observed <- *p
				return 4294967295
			},
			result: i32Result,
		},
		{
			name: "large_array",
			callback: func(p *int) (unused [250]int) {
				observed <- *p
				return
			},
			result: sretResult,
			size:   unsafe.Sizeof([250]int{}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Every callback above preserves the failing fixture's *int
			// parameter and captures only this channel. Read the actual closure
			// type and data using the same ABI representation as SetFinalizer.
			face := (*abi.EmptyInterface)(unsafe.Pointer(&tt.callback))
			if face.Type == nil || !face.Type.IsClosure() || face.Data == nil {
				t.Fatal("callback has no concrete closure type or data")
			}
			closureType := face.Type.StructType()
			if closureType == nil || len(closureType.Fields) == 0 {
				t.Fatal("callback closure has no function field")
			}
			functionType := closureType.Fields[0].Typ.FuncType()
			if functionType == nil || len(functionType.Out) != 1 {
				t.Fatal("callback does not have one result")
			}
			result, size := classifyFinalizerResultForTest(functionType.Out)
			if result != tt.result || size != tt.size {
				t.Fatalf("result ABI = (%d, %d), want (%d, %d)", result, size, tt.result, tt.size)
			}

			for _, want := range []int{42, -17} {
				object := new(int)
				*object = want
				callPointerFinalizerForTest(face.Data, unsafe.Pointer(object), result, size)
				// This is synchronous dispatch, so no collection, sleep, retry,
				// or deadline is needed to observe the completed callback.
				select {
				case got := <-observed:
					if got != want {
						t.Fatalf("callback got %d, want %d", got, want)
					}
				default:
					t.Fatal("dispatcher returned without invoking the callback")
				}
				select {
				case got := <-observed:
					t.Fatalf("dispatcher invoked the callback more than once: %d", got)
				default:
				}
				if *object != want {
					t.Fatalf("dispatcher modified its argument: got %d, want %d", *object, want)
				}
			}
		})
	}
}
