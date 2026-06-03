// LITTEST
package main

// CHECK-LABEL: define void @"{{.*}}tplocaltype.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i64 @"{{.*}}tplocaltype.use1"()
// CHECK-NEXT:   %1 = call i64 @"{{.*}}tplocaltype.use2"()
// CHECK-NEXT:   ret void
func main() {
	_ = use1()
	_ = use2()
}

// CHECK-LABEL: define i64 @"{{.*}}tplocaltype.use1"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i64 @"{{.*}}tplocaltype.id[{{.*}}T.1.0]"(i64 1)
// CHECK-NEXT:   ret i64 %0
func use1() int {
	type T int
	return int(id[T](1))
}

// CHECK-LABEL: define i64 @"{{.*}}tplocaltype.use2"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i64 @"{{.*}}tplocaltype.id[{{.*}}T.2.0]"(i64 2)
// CHECK-NEXT:   ret i64 %0
func use2() int {
	type T int
	return int(id[T](2))
}

func id[T ~int](v T) T {
	return v
}

// CHECK-LABEL: define linkonce i64 @"{{.*}}tplocaltype.id[{{.*}}T.1.0]"(i64 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret i64 %0

// CHECK-LABEL: define linkonce i64 @"{{.*}}tplocaltype.id[{{.*}}T.2.0]"(i64 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret i64 %0
