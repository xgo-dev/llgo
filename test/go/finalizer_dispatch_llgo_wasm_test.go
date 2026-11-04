//go:build llgo && wasm && llgo.wasm.gc.linear

package gotest

import (
	"runtime"
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
			functionType := face.Type.FuncType()
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

// Exercise registry-owned descriptors after real linear-GC collection, for
// every ordinary pointer/interface result dispatch. The original dispatcher
// cases above retain their narrow-result and large SRet coverage.
func TestRuntimeWasmFinalizerDescriptorScalarResults(t *testing.T) {
	tests := []struct {
		name     string
		callback func(chan<- int) any
	}{
		{"pointer/void", func(done chan<- int) any {
			return func(p *finalizerAssignableValue) { done <- p.value }
		}},
		{"pointer/i32", func(done chan<- int) any {
			return func(p *finalizerAssignableValue) uint32 {
				done <- p.value
				return 0xf1234567
			}
		}},
		{"pointer/i64", func(done chan<- int) any {
			return func(p *finalizerAssignableValue) uint64 {
				done <- p.value
				return 0xf123456789abcdef
			}
		}},
		{"pointer/f32", func(done chan<- int) any {
			return func(p *finalizerAssignableValue) float32 {
				done <- p.value
				return 3.25
			}
		}},
		{"pointer/f64", func(done chan<- int) any {
			return func(p *finalizerAssignableValue) float64 {
				done <- p.value
				return 9.75
			}
		}},
		{"interface/void", func(done chan<- int) any {
			return func(p any) { done <- p.(*finalizerAssignableValue).value }
		}},
		{"interface/i32", func(done chan<- int) any {
			return func(p any) uint32 {
				done <- p.(*finalizerAssignableValue).value
				return 0xf1234567
			}
		}},
		{"interface/i64", func(done chan<- int) any {
			return func(p any) uint64 {
				done <- p.(*finalizerAssignableValue).value
				return 0xf123456789abcdef
			}
		}},
		{"interface/f32", func(done chan<- int) any {
			return func(p any) float32 {
				done <- p.(*finalizerAssignableValue).value
				return 3.25
			}
		}},
		{"interface/f64", func(done chan<- int) any {
			return func(p any) float64 {
				done <- p.(*finalizerAssignableValue).value
				return 9.75
			}
		}},
	}
	for i, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			const registrations = 8
			done := make(chan int, registrations)
			want := 81234 + i
			registerFinalizerForTest(func() {
				for n := 0; n < registrations; n++ {
					object := &finalizerAssignableValue{value: want, keep: new(int)}
					// The factory erases the static callback type so registration
					// reaches SetFinalizer's actual metadata classifier.
					runtime.SetFinalizer(object, test.callback(done))
				}
			})
			waitForFinalizerValue(t, done, want)
		})
	}
}
