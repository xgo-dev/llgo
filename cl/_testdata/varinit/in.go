// LITTEST
package main

var a = 100

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[OLD_A:%[0-9]+]] = load i64, ptr @main.a
// CHECK-NEXT: [[NEW_A:%[0-9]+]] = add i64 [[OLD_A]], 1
// CHECK-NEXT: store i64 [[NEW_A]], ptr @main.a
func main() {
	a++
	_ = a
}
