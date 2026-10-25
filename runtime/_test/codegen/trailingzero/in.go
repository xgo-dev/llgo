// LITTEST linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64
package main

import (
	"reflect"
	"unsafe"
)

// Issue #2797: physical storage must agree with reflection's element sizes
// and field offsets. Check the type bodies without freezing instruction order.
// AMD64-DAG: %main.tail = type { i32, [0 x i32], i8 }
// ARM64-DAG: %main.tail = type { i32, [0 x i32], i8 }
// AMD64-DAG: %main.emptyTail = type { i64, {}, i8 }
// ARM64-DAG: %main.emptyTail = type { i64, {}, i8 }
// AMD64-DAG: %main.naturalTail = type { double, float, {} }
// ARM64-DAG: %main.naturalTail = type { double, float, {} }
// CHECK-DAG: %main.outer = type { %main.tail, i32, i32 }
// CHECK-DAG: %main.nativeTail = type { i32, [0 x i32] }
// CHECK-DAG: %main.nativeFloatTail = type { double, {} }
// CHECK-LABEL: define void @main.main()

type tail struct {
	Count int32
	End   [0]int32
}

type emptyTail struct {
	Value int64
	End   struct{}
}

type naturalTail struct {
	D   float64
	F   float32
	End struct{}
}

type zeroFields struct {
	Empty struct{}
	Array [0]int64
}

type outer struct {
	Head  tail
	Next  int32
	Guard int32
}

//llgo:cdecl
type nativeTail struct {
	Count int32
	End   [0]int32
}

//llgo:cdecl
type nativeFloatTail struct {
	Value float64
	End   struct{}
}

var nativeItems = [2]nativeTail{{Count: 1}, {Count: 2}}

func main() {
	items := [4]tail{{Count: 10}, {Count: 20}, {Count: 30}, {Count: 40}}
	emptyItems := [2]emptyTail{}
	println("layout", reflect.TypeOf(tail{}).Size(), uintptr(unsafe.Pointer(&items[1]))-uintptr(unsafe.Pointer(&items[0])))
	// The Go size is 12 on 386 and 16 on amd64/arm64; both need matching stride.
	println("empty tail", reflect.TypeOf(emptyTail{}).Size() == uintptr(unsafe.Pointer(&emptyItems[1]))-uintptr(unsafe.Pointer(&emptyItems[0])))
	zeros := [2]zeroFields{}
	println("zero", reflect.TypeOf(struct{}{}).Size(), reflect.TypeOf(zeroFields{}).Size(),
		uintptr(unsafe.Pointer(&zeros[1]))-uintptr(unsafe.Pointer(&zeros[0])))

	value := reflect.ValueOf(&items).Elem().Index(1).Field(0)
	println("array read", items[1].Count, value.Int())
	value.SetInt(77)
	println("array write", items[0].Count, items[1].Count, items[2].Count, items[3].Count)

	src := [8]tail{{Count: 10}, {Count: 20}, {Count: 30}, {Count: 40}, {Count: 50}, {Count: 60}, {Count: 70}, {Count: 80}}
	dst := [8]tail{{}, {}, {}, {}, {Count: 101}, {Count: 102}, {Count: 103}, {Count: 104}}
	// Keep guards inside the backing array so a wrong copy size is observable
	// without writing beyond the allocation.
	println("copied", reflect.Copy(reflect.ValueOf(dst[:4]), reflect.ValueOf(src[:4])))
	println("values", dst[0].Count, dst[1].Count, dst[2].Count, dst[3].Count)
	println("guards", dst[4].Count, dst[5].Count, dst[6].Count, dst[7].Count)

	nested := outer{Head: tail{Count: 10}, Next: 20, Guard: 30}
	field := reflect.ValueOf(&nested).Elem().Field(1)
	println("nested read", nested.Next, field.Int())
	field.SetInt(77)
	println("nested write", nested.Head.Count, nested.Next, nested.Guard)

	// Exercise reflect.Call's description of newly added tail storage.
	type callTail struct {
		A, B *byte
		End  struct{}
	}
	call := func(value callTail, text []string) { println("call", len(text), text[0]) }
	reflect.ValueOf(call).Call([]reflect.Value{reflect.ValueOf(callTail{}), reflect.ValueOf([]string{"hi"})})

	// Existing natural padding must not become an extra integer FFI field.
	roundTrip := func(value naturalTail, marker int32) naturalTail {
		if marker != 37 {
			panic("incorrect argument after padded struct")
		}
		return value
	}
	want := naturalTail{D: 1.25, F: 2.5}
	out := reflect.ValueOf(roundTrip).Call([]reflect.Value{reflect.ValueOf(want), reflect.ValueOf(int32(37))})
	got := out[0].Interface().(naturalTail)
	println("natural call", got.D == want.D, got.F == want.F)
	println("native value", nativeItems[1].Count)

	// The native descriptor must not introduce Go tail storage into the FFI
	// argument description and shift the scalar argument that follows it.
	nativeCall := func(value nativeFloatTail, marker uint64) float64 {
		if marker != 37 {
			panic("incorrect argument after native struct")
		}
		return value.Value
	}
	native := nativeFloatTail{Value: 1.25}
	result := reflect.ValueOf(nativeCall).Call([]reflect.Value{reflect.ValueOf(native), reflect.ValueOf(uint64(37))})
	println("native call", result[0].Float() == native.Value)
}
