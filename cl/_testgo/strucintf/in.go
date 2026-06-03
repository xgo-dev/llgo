// LITTEST
package main

import "github.com/goplus/llgo/cl/_testdata/foo"

// CHECK-LABEL: define %"{{.*}}eface" @"{{.*}}strucintf.Foo"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = alloca { i64 }, align 8
// CHECK-NEXT:   call void @llvm.memset(ptr %0, i8 0, i64 8, i1 false)
// CHECK-NEXT:   %1 = getelementptr inbounds { i64 }, ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i64 1, ptr %1, align 8
// CHECK-NEXT:   %2 = load { i64 }, ptr %0, align 8
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store { i64 } %2, ptr %3, align 8
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}eface" { ptr @"{{.*}}strucintf.struct{{.*}}", ptr undef }, ptr %3, 1
// CHECK-NEXT:   ret %"{{.*}}eface" %4
func Foo() any {
	return struct{ v int }{1}
}

// CHECK-LABEL: define void @"{{.*}}strucintf.main"(){{.*}} {
func main() {
	// CHECK: call %"{{.*}}eface" @"{{.*}}strucintf.Foo"()
	// CHECK: icmp eq ptr %{{[0-9]+}}, @"{{.*}}strucintf.struct{{.*}}"
	v := Foo()

	if x, ok := v.(struct{ v int }); ok {
		// CHECK: call void @"{{.*}}PrintInt"(i64 %{{[0-9]+}})
		// CHECK-NEXT: call void @"{{.*}}PrintByte"(i8 10)
		println(x.v)
	} else {
		println("Foo: not ok")
	}

	// CHECK: call %"{{.*}}eface" @"{{.*}}foo.Bar"()
	// CHECK: icmp eq ptr %{{[0-9]+}}, @"{{.*}}struct{{.*}}"
	bar := foo.Bar()

	if x, ok := bar.(struct{ V int }); ok {
		// CHECK: call void @"{{.*}}PrintInt"(i64 %{{[0-9]+}})
		// CHECK-NEXT: call void @"{{.*}}PrintByte"(i8 10)
		println(x.V)
	} else {
		println("Bar: not ok")
	}

	// CHECK: call %"{{.*}}eface" @"{{.*}}foo.F"()
	// CHECK: icmp eq ptr %{{[0-9]+}}, @"{{.*}}strucintf.struct{{.*}}"
	if x, ok := foo.F().(struct{ v int }); ok {
		// CHECK: call void @"{{.*}}PrintInt"(i64 %{{[0-9]+}})
		// CHECK-NEXT: call void @"{{.*}}PrintByte"(i8 10)
		println(x.v)
	} else {
		// CHECK: call void @"{{.*}}PrintString"(%"{{.*}}String" { ptr @{{[0-9]+}}, i64 9 })
		// CHECK-NEXT: call void @"{{.*}}PrintByte"(i8 10)
		println("F: not ok")
	}
}
