// LITTEST
package main

import _ "unsafe"

// CHECK: {{^}}@0 = private unnamed_addr constant [44 x i8] c"{{.*}}/cl/_testdata/method.T", align 1{{$}}
// CHECK: {{^}}@1 = private unnamed_addr constant [3 x i8] c"Add", align 1{{$}}

type T int

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testdata/method.T.Add"(i64 %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = add i64 %0, %1
// CHECK-NEXT:   ret i64 %2
// CHECK-NEXT: }

func (a T) Add(b T) T {
	return a + b
}

//go:linkname printf C.printf
func printf(format *int8, __llgo_va_list ...any)

var format = [...]int8{'H', 'e', 'l', 'l', 'o', ' ', '%', 'd', '\n', 0}

func main() {
	a := T(1)
	printf(&format[0], a.Add(2))
}

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testdata/method.(*T).Add"(ptr %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %2, %"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 44 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @1, i64 3 })
// CHECK-NEXT:   %3 = load i64, ptr %0, align 8
// CHECK-NEXT:   %4 = call i64 @"{{.*}}/cl/_testdata/method.T.Add"(i64 %3, i64 %1)
// CHECK-NEXT:   ret i64 %4
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/method.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testdata/method.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testdata/method.init$guard", align 1
// CHECK-NEXT:   store i8 72, ptr @"{{.*}}/cl/_testdata/method.format", align 1
// CHECK-NEXT:   store i8 101, ptr getelementptr inbounds (i8, ptr @"{{.*}}/cl/_testdata/method.format", i64 1), align 1
// CHECK-NEXT:   store i8 108, ptr getelementptr inbounds (i8, ptr @"{{.*}}/cl/_testdata/method.format", i64 2), align 1
// CHECK-NEXT:   store i8 108, ptr getelementptr inbounds (i8, ptr @"{{.*}}/cl/_testdata/method.format", i64 3), align 1
// CHECK-NEXT:   store i8 111, ptr getelementptr inbounds (i8, ptr @"{{.*}}/cl/_testdata/method.format", i64 4), align 1
// CHECK-NEXT:   store i8 32, ptr getelementptr inbounds (i8, ptr @"{{.*}}/cl/_testdata/method.format", i64 5), align 1
// CHECK-NEXT:   store i8 37, ptr getelementptr inbounds (i8, ptr @"{{.*}}/cl/_testdata/method.format", i64 6), align 1
// CHECK-NEXT:   store i8 100, ptr getelementptr inbounds (i8, ptr @"{{.*}}/cl/_testdata/method.format", i64 7), align 1
// CHECK-NEXT:   store i8 10, ptr getelementptr inbounds (i8, ptr @"{{.*}}/cl/_testdata/method.format", i64 8), align 1
// CHECK-NEXT:   store i8 0, ptr getelementptr inbounds (i8, ptr @"{{.*}}/cl/_testdata/method.format", i64 9), align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/method.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i64 @"{{.*}}/cl/_testdata/method.T.Add"(i64 1, i64 2)
// CHECK-NEXT:   call void (ptr, ...) @printf(ptr @"{{.*}}/cl/_testdata/method.format", i64 %0)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
