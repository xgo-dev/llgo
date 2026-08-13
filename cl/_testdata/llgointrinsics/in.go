// LITTEST
package llgointrinsics

import (
	"unsafe"
)

// llgo.funcPCABI0 resolves declarations and non-capturing functions directly,
// and returns the code pointer (not the environment) for a closure.
// CHECK-LABEL: define i64 @"{{.*}}.UseBare"(){{.*}} {
// CHECK: ret i64 ptrtoint (ptr @bar to i64)
// CHECK-LABEL: define i64 @"{{.*}}.UseC"(){{.*}} {
// CHECK: ret i64 ptrtoint (ptr @write to i64)
// CHECK-LABEL: define i64 @"{{.*}}.UseCTrampoline"(){{.*}} {
// CHECK: ret i64 ptrtoint (ptr @write to i64)
// CHECK-LABEL: define i64 @"{{.*}}.UseClosure"(){{.*}} {
// CHECK: [[UC_X:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT: [[UC_ENV:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT: [[UC_X_SLOT:%[0-9]+]] = getelementptr inbounds { ptr }, ptr [[UC_ENV]], i32 0, i32 0
// CHECK-NEXT: store ptr [[UC_X]], ptr [[UC_X_SLOT]]
// CHECK-NEXT: [[UC_CLOSURE:%[0-9]+]] = insertvalue { ptr, ptr } { ptr @"{{.*}}.UseClosure$1", ptr undef }, ptr [[UC_ENV]], 1
// CHECK-NEXT: ret i64 ptrtoint (ptr @"{{.*}}.UseClosure$1" to i64)
// CHECK-LABEL: define void @"{{.*}}.UseClosure$1"(ptr {{(nest|swiftself)}}
// CHECK: [[UC_ENV_VALUE:%[0-9]+]] = load { ptr }, ptr %{{[0-9]+}}
// CHECK-NEXT: [[UC_X_PTR:%[0-9]+]] = extractvalue { ptr } [[UC_ENV_VALUE]], 0
// CHECK-NEXT: [[UC_OLD_X:%[0-9]+]] = load i64, ptr [[UC_X_PTR]]
// CHECK-NEXT: [[UC_NEW_X:%[0-9]+]] = add i64 [[UC_OLD_X]], 1
// CHECK-NEXT: [[UC_X_PTR_AGAIN:%[0-9]+]] = extractvalue { ptr } [[UC_ENV_VALUE]], 0
// CHECK-NEXT: store i64 [[UC_NEW_X]], ptr [[UC_X_PTR_AGAIN]]
// CHECK-NEXT: ret void
// CHECK-LABEL: define i64 @"{{.*}}.UseFunc"(){{.*}} {
// CHECK: ret i64 ptrtoint (ptr @"{{.*}}.UseFunc$1" to i64)
// CHECK-LABEL: define void @"{{.*}}.UseFunc$1"(){{.*}} {
// CHECK: ret void
// CHECK-LABEL: define i64 @"{{.*}}.UseLibc"(){{.*}} {
// CHECK: ret i64 ptrtoint (ptr @foo to i64)
// CHECK-LABEL: define void @"{{.*}}.UseSkip"(){{.*}} {
// CHECK-NOT: call {{.*}}skip
// CHECK: call void @"{{.*}}.PrintUint"(i64 0)
// CHECK-NEXT: call void @"{{.*}}.PrintUint"(i64 0)
// CHECK-NEXT: call void @"{{.*}}.PrintUint"(i64 0)
// CHECK-NEXT: ret void

//go:linkname funcPCABI0 llgo.funcPCABI0
func funcPCABI0(fn interface{}) uintptr

//go:linkname skip llgo.skip
func skip()

//go:linkname skipWithRet llgo.skip
func skipWithRet() uintptr

//go:linkname skipWithMultiRet llgo.skip
func skipWithMultiRet() (uintptr, uintptr)

//go:linkname libc_foo_trampoline C.foo
func libc_foo_trampoline()

//go:linkname bar_trampoline bar_trampoline
func bar_trampoline()

//go:linkname write C.write
func write(fd int, buf unsafe.Pointer, count int) int

//go:linkname write_trampoline C.write
func write_trampoline()

func UseBare() uintptr {
	return funcPCABI0(bar_trampoline)
}

func UseC() uintptr {
	return funcPCABI0(write)
}

func UseCTrampoline() uintptr {
	return funcPCABI0(write_trampoline)
}

func UseClosure() uintptr {
	var x int
	return funcPCABI0(func() {
		x++
	})
}

func UseFunc() uintptr {
	return funcPCABI0(func() {})
}

func UseLibc() uintptr {
	return funcPCABI0(libc_foo_trampoline)
}

func UseSkip() {
	skip()
	i := skipWithRet()
	print(i)
	a, b := skipWithMultiRet()
	print(a, b)
}
