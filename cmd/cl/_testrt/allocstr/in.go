// LITTEST
// Scope: common
package main

import "unsafe"

// Keep these thin declarations local: this fixture owns LLGo's C-string and
// stack-allocation intrinsics, not github.com/goplus/lib's wrappers.

//go:linkname alloca llgo.alloca
func alloca(size uintptr) unsafe.Pointer

//go:linkname allocaCStr llgo.allocaCStr
func allocaCStr(s string) *int8

//go:linkname allocaCStrs llgo.allocaCStrs
func allocaCStrs(strs []string, endWithNil bool) **int8

//go:linkname cstr llgo.cstr
func cstr(s string) *int8

//go:linkname memcpy C.memcpy
func memcpy(dst, src unsafe.Pointer, n uintptr) unsafe.Pointer

//go:linkname strlen C.strlen
func strlen(str *int8) uintptr

//go:linkname printf C.printf
func printf(format *int8, __llgo_va_list ...any) int32

var strlenInput = [...]int8{'a', 'b', 'c', 0}

// CHECK: [[HELLO_TEXT:@[0-9]+]] = private unnamed_addr constant [12 x i8] c"Hello world\0A"
// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.String" @main.hello(){{.*}} {
// CHECK: ret %"{{.*}}/runtime/internal/runtime.String" { ptr [[HELLO_TEXT]], i64 12 }
func hello() string {
	return "Hello world\n"
}

// One bounded case owns the four related intrinsic shapes: dynamic C-string
// copy, constant C string, raw stack allocation, and a null-terminated C-string
// vector. The checks intentionally follow only the defining data flow.
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[TEXT:%[0-9]+]] = call %"{{.*}}String" @main.hello()
// CHECK-NEXT: [[TEXT_LEN:%[0-9]+]] = extractvalue %"{{.*}}String" [[TEXT]], 1
// CHECK-NEXT: [[CSTR_LEN:%[0-9]+]] = add i64 [[TEXT_LEN]], 1
// CHECK-NEXT: [[CSTR_BUF:%[0-9]+]] = alloca i8, i64 [[CSTR_LEN]]
// CHECK-NEXT: [[DYNAMIC_CSTR:%[0-9]+]] = call ptr @"{{.*}}CStrCopy"(ptr [[CSTR_BUF]], %"{{.*}}String" [[TEXT]])
// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr [[DYNAMIC_CSTR]])
// CHECK: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}})
// CHECK: [[RAW_BUF:%[0-9]+]] = alloca i8, i64 4
// CHECK-NEXT: call ptr @memcpy(ptr [[RAW_BUF]], ptr @{{[0-9]+}}, i64 4)
// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, ptr [[RAW_BUF]])
// CHECK: [[SLICE_LEN:%[0-9]+]] = extractvalue %{{.*}}Slice{{.*}}, 1
// CHECK-NEXT: [[VECTOR_LEN:%[0-9]+]] = add i64 [[SLICE_LEN]], 1
// CHECK-NEXT: [[VECTOR:%[0-9]+]] = alloca ptr, i64 [[VECTOR_LEN]]
// CHECK: [[STRLEN:%[0-9]+]] = call i64 @strlen(ptr @main.strlenInput)
// CHECK-NEXT: [[BAD_LENGTH:%[0-9]+]] = icmp ne i64 [[STRLEN]], 3
// CHECK-NEXT: br i1 [[BAD_LENGTH]], label %{{[^, ]+}}, label %{{[^, ]+}}
// CHECK: store ptr null, ptr %{{[0-9]+}}
// CHECK: call ptr @"{{.*}}CStrCopy"(ptr %{{[0-9]+}}, %"{{.*}}String" %{{[0-9]+}})
func main() {
	printf(allocaCStr(hello()))
	printf(cstr("Hello, world\n"))

	buf := alloca(4)
	memcpy(buf, unsafe.Pointer(cstr("Hi\n")), 4)
	printf(cstr("%s"), buf)

	values := allocaCStrs([]string{"a", "b", "c"}, true)
	items := unsafe.Slice(values, 4)
	for _, item := range items {
		if item == nil {
			break
		}
		printf(cstr("%s\n"), item)
	}

	if strlen(&strlenInput[0]) != 3 {
		panic("unexpected C string length")
	}
	// ESP-IDF's compact printf does not implement the C99 %zu modifier.
	printf(cstr("Length 3\n"))
}
