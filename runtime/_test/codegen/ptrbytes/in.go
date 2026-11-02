// LITTEST linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64 windows/386
package main

import "reflect"

// The common type header begins with Size_ and PtrBytes. Scalar suffixes must
// not clear the prefix ending at the last pointer. Check both pointer widths,
// including recursion through an inline struct and an array of structs.
// AMD64-DAG: @_llgo_main.pointerTail = {{.*}}{ i64 16, i64 8,
// ARM64-DAG: @_llgo_main.pointerTail = {{.*}}{ i64 16, i64 8,
// WINDOWS-386-DAG: @_llgo_main.pointerTail = {{.*}}{ i32 8, i32 4,
// AMD64-DAG: @_llgo_main.offsetPointerTail = {{.*}}{ i64 24, i64 16,
// ARM64-DAG: @_llgo_main.offsetPointerTail = {{.*}}{ i64 24, i64 16,
// WINDOWS-386-DAG: @_llgo_main.offsetPointerTail = {{.*}}{ i32 12, i32 8,
// AMD64-DAG: @_llgo_main.nestedTail = {{.*}}{ i64 32, i64 16,
// ARM64-DAG: @_llgo_main.nestedTail = {{.*}}{ i64 32, i64 16,
// WINDOWS-386-DAG: @_llgo_main.nestedTail = {{.*}}{ i32 16, i32 8,
// AMD64-DAG: @_llgo_main.zeroArrayTail = {{.*}}{ i64 16, i64 8,
// ARM64-DAG: @_llgo_main.zeroArrayTail = {{.*}}{ i64 16, i64 8,
// WINDOWS-386-DAG: @_llgo_main.zeroArrayTail = {{.*}}{ i32 8, i32 4,
// AMD64-DAG: @_llgo_main.lastPointer = {{.*}}{ i64 16, i64 16,
// ARM64-DAG: @_llgo_main.lastPointer = {{.*}}{ i64 16, i64 16,
// WINDOWS-386-DAG: @_llgo_main.lastPointer = {{.*}}{ i32 8, i32 8,
// AMD64-DAG: @_llgo_main.pointerFree = {{.*}}{ i64 16, i64 0,
// ARM64-DAG: @_llgo_main.pointerFree = {{.*}}{ i64 16, i64 0,
// WINDOWS-386-DAG: @_llgo_main.pointerFree = {{.*}}{ i32 8, i32 0,
// AMD64-DAG: @_llgo_main.pointerPairs = {{.*}}{ i64 32, i64 24,
// ARM64-DAG: @_llgo_main.pointerPairs = {{.*}}{ i64 32, i64 24,
// WINDOWS-386-DAG: @_llgo_main.pointerPairs = {{.*}}{ i32 16, i32 12,
// AMD64-DAG: @_llgo_main.emptyPointers = {{.*}}{ i64 0, i64 0,
// ARM64-DAG: @_llgo_main.emptyPointers = {{.*}}{ i64 0, i64 0,
// WINDOWS-386-DAG: @_llgo_main.emptyPointers = {{.*}}{ i32 0, i32 0,
// CHECK-LABEL: define void @main.main()

type pointerTail struct {
	P *byte
	N uintptr
}

type offsetPointerTail struct {
	Head uintptr
	P    *byte
	Tail uintptr
}

type nestedTail struct {
	Head  byte
	Inner pointerTail
	Tail  uintptr
}

type zeroArrayTail struct {
	P     *byte
	Empty [0]*byte
	Tail  uintptr
}

type lastPointer struct {
	Head uintptr
	P    *byte
}

type pointerFree struct {
	A, B uintptr
}

type pointerPairs [2]pointerTail
type emptyPointers [0]*byte

func main() {
	// Keeping these types in interfaces retains their descriptors for the IR
	// checks. Name output also verifies that each retained type is exercised.
	values := []any{
		pointerTail{}, offsetPointerTail{}, nestedTail{}, zeroArrayTail{},
		lastPointer{}, pointerFree{}, pointerPairs{}, emptyPointers{},
	}
	for _, value := range values {
		println(reflect.TypeOf(value).String())
	}
}
