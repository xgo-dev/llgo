// LITTEST
package main

// CHECK-LABEL: define i64 @"{{.*}}allocinloop.Foo"(%"{{.*}}String" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = extractvalue %"{{.*}}String" %0, 1
// CHECK-NEXT:   ret i64 %1
func Foo(s string) int {
	return len(s)
}

// CHECK-LABEL: define void @"{{.*}}allocinloop.Test"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:
// CHECK-NEXT:   %0 = phi i64 [ 0, %_llgo_0 ], [ %4, %_llgo_2 ]
// CHECK-NEXT:   %1 = phi i64 [ 0, %_llgo_0 ], [ %5, %_llgo_2 ]
// CHECK-NEXT:   %2 = icmp slt i64 %1, 10000000
// CHECK-NEXT:   br i1 %2, label %_llgo_2, label %_llgo_3
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:
// CHECK-NEXT:   %3 = call i64 @"{{.*}}allocinloop.Foo"(%"{{.*}}String" { ptr @0, i64 5 })
// CHECK-NEXT:   %4 = add i64 %0, %3
// CHECK-NEXT:   %5 = add i64 %1, 1
// CHECK-NEXT:   br label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_3:
// CHECK-NEXT:   call void @"{{.*}}PrintInt"(i64 %0)
// CHECK-NEXT:   call void @"{{.*}}PrintByte"(i8 10)
// CHECK-NEXT:   ret void
func Test() {
	j := 0
	for i := 0; i < 10000000; i++ {
		j += Foo("hello")
	}
	println(j)
}

// CHECK-LABEL: define void @"{{.*}}allocinloop.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}allocinloop.Test"()
// CHECK-NEXT:   ret void
func main() {
	Test()
}
