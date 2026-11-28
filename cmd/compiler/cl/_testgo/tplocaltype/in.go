// LITTEST
package main

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call i64 @main.use1()
// CHECK-NEXT: call i64 @main.use2()
func main() {
	_ = use1()
	_ = use2()
}

// CHECK-LABEL: define i64 @main.use1(){{.*}} {
// CHECK: [[USE1:%[0-9]+]] = call i64 @"main.id[{{.*}}T.1.0]"(i64 1)
// CHECK-NEXT: ret i64 [[USE1]]
func use1() int {
	type T int
	return int(id[T](1))
}

// CHECK-LABEL: define i64 @main.use2(){{.*}} {
// CHECK: [[USE2:%[0-9]+]] = call i64 @"main.id[{{.*}}T.2.0]"(i64 2)
// CHECK-NEXT: ret i64 [[USE2]]
func use2() int {
	type T int
	return int(id[T](2))
}

func id[T ~int](v T) T {
	return v
}

// CHECK-LABEL: define linkonce i64 @"main.id[{{.*}}T.1.0]"(i64 %0){{.*}} {
// CHECK: ret i64 %0

// CHECK-LABEL: define linkonce i64 @"main.id[{{.*}}T.2.0]"(i64 %0){{.*}} {
// CHECK: ret i64 %0
