// LITTEST
package main

/*
#include "in.h"
*/
import "C"
import "fmt"

// CHECK-LABEL: define i32 @"{{.*}}/cl/_testgo/cgocfiles._Cfunc_test_structs"(ptr %0, ptr %1, ptr %2, ptr %3, ptr %4){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %6 = load ptr, ptr @"{{.*}}/cl/_testgo/cgocfiles._cgo_{{.*}}_Cfunc_test_structs", align 8
// CHECK-NEXT:   %7 = load ptr, ptr %6, align 8
// CHECK-NEXT:   %8 = call i32 %7(ptr %0, ptr %1, ptr %2, ptr %3, ptr %4)
// CHECK-NEXT:   ret i32 %8
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/cgocfiles.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 4)
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}/cl/_testgo/cgocfiles._Ctype_struct___3", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i32 1, ptr %1, align 4
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}/cl/_testgo/cgocfiles._Ctype_struct___4", ptr %2, i32 0, i32 0
// CHECK-NEXT:   %4 = getelementptr inbounds %"{{.*}}/cl/_testgo/cgocfiles._Ctype_struct___4", ptr %2, i32 0, i32 1
// CHECK-NEXT:   store i32 1, ptr %3, align 4
// CHECK-NEXT:   store i32 2, ptr %4, align 4
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 12)
// CHECK-NEXT:   %6 = getelementptr inbounds %"{{.*}}/cl/_testgo/cgocfiles._Ctype_struct___0", ptr %5, i32 0, i32 0
// CHECK-NEXT:   %7 = getelementptr inbounds %"{{.*}}/cl/_testgo/cgocfiles._Ctype_struct___0", ptr %5, i32 0, i32 1
// CHECK-NEXT:   %8 = getelementptr inbounds %"{{.*}}/cl/_testgo/cgocfiles._Ctype_struct___0", ptr %5, i32 0, i32 2
// CHECK-NEXT:   store i32 1, ptr %6, align 4
// CHECK-NEXT:   store i32 2, ptr %7, align 4
// CHECK-NEXT:   store i32 3, ptr %8, align 4
// CHECK-NEXT:   %9 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 16)
// CHECK-NEXT:   %10 = getelementptr inbounds %"{{.*}}/cl/_testgo/cgocfiles._Ctype_struct___1", ptr %9, i32 0, i32 0
// CHECK-NEXT:   %11 = getelementptr inbounds %"{{.*}}/cl/_testgo/cgocfiles._Ctype_struct___1", ptr %9, i32 0, i32 1
// CHECK-NEXT:   %12 = getelementptr inbounds %"{{.*}}/cl/_testgo/cgocfiles._Ctype_struct___1", ptr %9, i32 0, i32 2
// CHECK-NEXT:   %13 = getelementptr inbounds %"{{.*}}/cl/_testgo/cgocfiles._Ctype_struct___1", ptr %9, i32 0, i32 3
// CHECK-NEXT:   store i32 1, ptr %10, align 4
// CHECK-NEXT:   store i32 2, ptr %11, align 4
// CHECK-NEXT:   store i32 3, ptr %12, align 4
// CHECK-NEXT:   store i32 4, ptr %13, align 4
// CHECK-NEXT:   %14 = call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 20)
// CHECK-NEXT:   %15 = getelementptr inbounds %"{{.*}}/cl/_testgo/cgocfiles._Ctype_struct___2", ptr %14, i32 0, i32 0
// CHECK-NEXT:   %16 = getelementptr inbounds %"{{.*}}/cl/_testgo/cgocfiles._Ctype_struct___2", ptr %14, i32 0, i32 1
// CHECK-NEXT:   %17 = getelementptr inbounds %"{{.*}}/cl/_testgo/cgocfiles._Ctype_struct___2", ptr %14, i32 0, i32 2
// CHECK-NEXT:   %18 = getelementptr inbounds %"{{.*}}/cl/_testgo/cgocfiles._Ctype_struct___2", ptr %14, i32 0, i32 3
// CHECK-NEXT:   %19 = getelementptr inbounds %"{{.*}}/cl/_testgo/cgocfiles._Ctype_struct___2", ptr %14, i32 0, i32 4
// CHECK-NEXT:   store i32 1, ptr %15, align 4
// CHECK-NEXT:   store i32 2, ptr %16, align 4
// CHECK-NEXT:   store i32 3, ptr %17, align 4
// CHECK-NEXT:   store i32 4, ptr %18, align 4
// CHECK-NEXT:   store i32 5, ptr %19, align 4
// CHECK-NEXT:   %20 = call i32 @"{{.*}}/cl/_testgo/cgocfiles._Cfunc_test_structs"(ptr %0, ptr %2, ptr %5, ptr %9, ptr %14)
func main() {
	r := C.test_structs(&C.s4{a: 1}, &C.s8{a: 1, b: 2}, &C.s12{a: 1, b: 2, c: 3}, &C.s16{a: 1, b: 2, c: 3, d: 4}, &C.s20{a: 1, b: 2, c: 3, d: 4, e: 5})
	fmt.Println(r)
	if r != 35 {
		panic("test_structs failed")
	}
}
