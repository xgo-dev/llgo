// LITTEST
package main

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call i64 @main.max(i64 1, i64 2)
// CHECK-NEXT: ret void
func main() {
	_ = max(1, 2)
}

// CHECK-LABEL: define i64 @main.max(i64 %0, i64 %1){{.*}} {
// CHECK: [[GREATER:%[0-9]+]] = icmp sgt i64 %0, %1
// CHECK-NEXT: br i1 [[GREATER]], label %{{[^,]+}}, label %{{[^ ]+}}
// CHECK: ret i64 %0
// CHECK: ret i64 %1
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
