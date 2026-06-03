// LITTEST
package llgointrinsics

import (
	"unsafe"
)

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

// CHECK-LABEL: define i64 @"{{.*}}.UseBare"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret i64 ptrtoint (ptr @bar to i64)
// CHECK-NEXT: }
func UseBare() uintptr {
	return funcPCABI0(bar_trampoline)
}

// CHECK-LABEL: define i64 @"{{.*}}.UseC"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret i64 ptrtoint (ptr @write to i64)
// CHECK-NEXT: }
func UseC() uintptr {
	return funcPCABI0(write)
}

// CHECK-LABEL: define i64 @"{{.*}}.UseCTrampoline"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret i64 ptrtoint (ptr @write to i64)
// CHECK-NEXT: }
func UseCTrampoline() uintptr {
	return funcPCABI0(write_trampoline)
}

// CHECK-LABEL: define i64 @"{{.*}}.UseClosure"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}.AllocZ"(i64 8)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}.AllocU"(i64 8)
// CHECK:   ret i64 ptrtoint (ptr @"{{.*}}.UseClosure$1" to i64)
// CHECK-LABEL: define void @"{{.*}}.UseClosure$1"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK:   %4 = add i64 %3, 1
// CHECK:   ret void
// CHECK-NEXT: }
func UseClosure() uintptr {
	var x int
	return funcPCABI0(func() {
		x++
	})
}

// CHECK-LABEL: define i64 @"{{.*}}.UseFunc"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret i64 ptrtoint (ptr @"{{.*}}.UseFunc$1" to i64)
// CHECK-NEXT: }
// CHECK-LABEL: define void @"{{.*}}.UseFunc$1"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func UseFunc() uintptr {
	return funcPCABI0(func() {})
}

// CHECK-LABEL: define i64 @"{{.*}}.UseLibc"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret i64 ptrtoint (ptr @foo to i64)
// CHECK-NEXT: }
func UseLibc() uintptr {
	return funcPCABI0(libc_foo_trampoline)
}

// CHECK-LABEL: define void @"{{.*}}.UseSkip"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}.PrintUint"(i64 0)
// CHECK-NEXT:   call void @"{{.*}}.PrintUint"(i64 0)
// CHECK-NEXT:   call void @"{{.*}}.PrintUint"(i64 0)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func UseSkip() {
	skip()
	i := skipWithRet()
	print(i)
	a, b := skipWithMultiRet()
	print(a, b)
}
