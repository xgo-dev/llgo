// LITTEST
package main

var a = 100

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i64, ptr @main.a, align 8
// CHECK-NEXT:   %1 = add i64 %0, 1
// CHECK-NEXT:   store i64 %1, ptr @main.a, align 8
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	a++
	_ = a
}
