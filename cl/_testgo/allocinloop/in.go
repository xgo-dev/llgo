// LITTEST
package main

// The string literal passed in the loop must not introduce a per-iteration
// allocation.
// CHECK-LABEL: define void @main.Test(){{.*}} {
// CHECK-NOT: Alloc
// CHECK: [[TOTAL:%[0-9]+]] = phi i64 [ 0, %{{.*}} ], [ [[NEXT_TOTAL:%[0-9]+]], %{{.*}} ]
// CHECK-NEXT: [[ITER:%[0-9]+]] = phi i64 [ 0, %{{.*}} ], [ [[NEXT_ITER:%[0-9]+]], %{{.*}} ]
// CHECK: [[VALUE:%[0-9]+]] = call i64 @main.Foo(%"{{.*}}String" { ptr @{{[0-9]+}}, i64 5 })
// CHECK-NEXT: [[NEXT_TOTAL]] = add i64 [[TOTAL]], [[VALUE]]
// CHECK-NEXT: [[NEXT_ITER]] = add i64 [[ITER]], 1
// CHECK-NOT: Alloc
// CHECK: call void @"{{.*}}PrintInt"(i64 [[TOTAL]])
// CHECK: ret void

func Foo(s string) int {
	return len(s)
}

func Test() {
	j := 0
	for i := 0; i < 10000000; i++ {
		j += Foo("hello")
	}
	println(j)
}

func main() {
	Test()
}
