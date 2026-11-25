// LITTEST
package main

import "unsafe"

type named32 uint32
type named64 uint64
type namedString string

type structKey struct {
	value uint64
}

var sink int

// CHECK-LABEL: define void @main.testChannel(ptr %0){{.*}} {
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.MapAssignFast64Ptr"
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.MapAccess1Fast64"
// CHECK: call { ptr, i1 } @"{{.*}}/runtime/internal/runtime.MapAccess2Fast64"
// CHECK: call void @"{{.*}}/runtime/internal/runtime.MapDeleteFast64"
// CHECK-LABEL: define void @main.testFallback(){{.*}} {
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.MapAccess1"
// CHECK: call { ptr, i1 } @"{{.*}}/runtime/internal/runtime.MapAccess2"
// CHECK: call void @"{{.*}}/runtime/internal/runtime.MapDelete"
// CHECK-LABEL: define void @main.testInt(){{.*}} {
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.MapAssignFast64"
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.MapAccess1Fast64"
// CHECK-LABEL: define void @main.testPointer(ptr %0){{.*}} {
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.MapAssignFast64Ptr"
// CHECK: ptrtoint ptr %0 to i64
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.MapAccess1Fast64"
// CHECK: call { ptr, i1 } @"{{.*}}/runtime/internal/runtime.MapAccess2Fast64"
// CHECK: call void @"{{.*}}/runtime/internal/runtime.MapDeleteFast64"
// CHECK-LABEL: define void @main.testString(){{.*}} {
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.MapAssignFastStr"
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.MapAccess1FastStr"
// CHECK: call { ptr, i1 } @"{{.*}}/runtime/internal/runtime.MapAccess2FastStr"
// CHECK: call void @"{{.*}}/runtime/internal/runtime.MapDeleteFastStr"
// CHECK-LABEL: define void @main.testStringFunc(){{.*}} {
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.MapAssignFastStr"
// CHECK-LABEL: define void @main.testUint32(){{.*}} {
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.MapAssignFast32"
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.MapAccess1Fast32"
// CHECK: call { ptr, i1 } @"{{.*}}/runtime/internal/runtime.MapAccess2Fast32"
// CHECK: call void @"{{.*}}/runtime/internal/runtime.MapDeleteFast32"
// CHECK-LABEL: define void @main.testUint64(){{.*}} {
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.MapAssignFast64"
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.MapAccess1Fast64"
// CHECK: call { ptr, i1 } @"{{.*}}/runtime/internal/runtime.MapAccess2Fast64"
// CHECK: call void @"{{.*}}/runtime/internal/runtime.MapDeleteFast64"
// CHECK-LABEL: define void @main.testUnsafePointer(ptr %0){{.*}} {
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.MapAssignFast64Ptr"
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.MapAccess1Fast64"
// CHECK: call { ptr, i1 } @"{{.*}}/runtime/internal/runtime.MapAccess2Fast64"
// CHECK: call void @"{{.*}}/runtime/internal/runtime.MapDeleteFast64"

func testUint32() {
	m := make(map[named32]int)
	m[7] = 11
	sink += m[7]
	if value, ok := m[7]; ok {
		sink += value
	}
	delete(m, 7)
}

func testUint64() {
	m := make(map[named64]int)
	m[1<<40] = 13
	sink += m[1<<40]
	if value, ok := m[1<<40]; ok {
		sink += value
	}
	delete(m, 1<<40)
}

func testInt() {
	m := make(map[int]int)
	m[37] = 37
	sink += m[37]
}

func testString() {
	m := make(map[namedString]int)
	m["fast"] = 17
	sink += m["fast"]
	if value, ok := m["fast"]; ok {
		sink += value
	}
	delete(m, "fast")
}

func addSink() {
	sink++
}

func testStringFunc() {
	m := make(map[string]func())
	m["call"] = addSink
	m["call"]()
}

func testPointer(key *int) {
	m := make(map[*int]int)
	m[key] = 19
	sink += m[key]
	if value, ok := m[key]; ok {
		sink += value
	}
	delete(m, key)
}

func testChannel(key chan int) {
	m := make(map[chan int]int)
	m[key] = 23
	sink += m[key]
	if value, ok := m[key]; ok {
		sink += value
	}
	delete(m, key)
}

func testUnsafePointer(key unsafe.Pointer) {
	m := make(map[unsafe.Pointer]int)
	m[key] = 29
	sink += m[key]
	if value, ok := m[key]; ok {
		sink += value
	}
	delete(m, key)
}

func testFallback() {
	key := structKey{31}
	m := make(map[structKey]int)
	m[key] = 31
	sink += m[key]
	if value, ok := m[key]; ok {
		sink += value
	}
	delete(m, key)
}

func main() {
	value := 1
	testUint32()
	testUint64()
	testInt()
	testString()
	testStringFunc()
	testPointer(&value)
	testChannel(make(chan int))
	testUnsafePointer(unsafe.Pointer(&value))
	testFallback()
	println(sink)
}
