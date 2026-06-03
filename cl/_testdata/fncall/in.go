// LITTEST
package main

// CHECK-LABEL: define void @"{{.*}}.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i64 @"{{.*}}.max"(i64 1, i64 2)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	_ = max(1, 2)
}

// CHECK-LABEL: define i64 @"{{.*}}.max"(i64 %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = icmp sgt i64 %0, %1
// CHECK-NEXT:   br i1 %2, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:{{.*}}
// CHECK-NEXT:   ret i64 %0
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:{{.*}}
// CHECK-NEXT:   ret i64 %1
// CHECK-NEXT: }
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
