// LITTEST
package main

const c = 100

var a float64 = 1

// CHECK: @"{{.*}}/untyped.a" = global double 0.000000e+00, align 8

// CHECK-LABEL: define void @"{{.*}}.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   br i1 false, label %_llgo_1, label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:{{.*}}
// CHECK-NEXT:   store double 0.000000e+00, ptr @"{{.*}}.a", align 8
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:{{.*}}
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func main() {
	if c > 100 {
		a = 0
	}
}
