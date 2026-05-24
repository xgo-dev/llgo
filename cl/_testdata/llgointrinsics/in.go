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

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testdata/llgointrinsics.UseBare"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret i64 ptrtoint (ptr @bar to i64)
// CHECK-NEXT: }

func UseBare() uintptr {
	return funcPCABI0(bar_trampoline)
}

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testdata/llgointrinsics.UseC"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret i64 ptrtoint (ptr @write to i64)
// CHECK-NEXT: }

func UseC() uintptr {
	return funcPCABI0(write)
}

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testdata/llgointrinsics.UseCTrampoline"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret i64 ptrtoint (ptr @write to i64)
// CHECK-NEXT: }

func UseCTrampoline() uintptr {
	return funcPCABI0(write_trampoline)
}

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testdata/llgointrinsics.UseClosure"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   %2 = getelementptr inbounds { ptr }, ptr %1, i32 0, i32 0
// CHECK-NEXT:   store ptr %0, ptr %2, align 8
// CHECK-NEXT:   %3 = insertvalue { ptr, ptr } { ptr @"{{.*}}/cl/_testdata/llgointrinsics.UseClosure$1", ptr undef }, ptr %1, 1
// CHECK-NEXT:   ret i64 ptrtoint (ptr @"{{.*}}/cl/_testdata/llgointrinsics.UseClosure$1" to i64)
// CHECK-NEXT: }

func UseClosure() uintptr {
	var x int

	// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/llgointrinsics.UseClosure$1"(ptr %0){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   %1 = load { ptr }, ptr %0, align 8
	// CHECK-NEXT:   %2 = extractvalue { ptr } %1, 0
	// CHECK-NEXT:   %3 = load i64, ptr %2, align 8
	// CHECK-NEXT:   %4 = add i64 %3, 1
	// CHECK-NEXT:   %5 = extractvalue { ptr } %1, 0
	// CHECK-NEXT:   store i64 %4, ptr %5, align 8
	// CHECK-NEXT:   ret void
	// CHECK-NEXT: }

	return funcPCABI0(func() {
		x++
	})
}

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testdata/llgointrinsics.UseFunc"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret i64 ptrtoint (ptr @"{{.*}}/cl/_testdata/llgointrinsics.UseFunc$1" to i64)
// CHECK-NEXT: }

func UseFunc() uintptr {

	// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/llgointrinsics.UseFunc$1"(){{.*}} {
	// CHECK-NEXT: _llgo_0:
	// CHECK-NEXT:   ret void
	// CHECK-NEXT: }

	return funcPCABI0(func() {})
}

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testdata/llgointrinsics.UseLibc"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret i64 ptrtoint (ptr @foo to i64)
// CHECK-NEXT: }

func UseLibc() uintptr {
	return funcPCABI0(libc_foo_trampoline)
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/llgointrinsics.UseSkip"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 0)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 0)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 0)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

func UseSkip() {
	skip()
	i := skipWithRet()
	print(i)
	a, b := skipWithMultiRet()
	print(a, b)
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/llgointrinsics.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testdata/llgointrinsics.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testdata/llgointrinsics.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
