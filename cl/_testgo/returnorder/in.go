// LITTEST
package main

// Evaluate the mutating second result before loading the first result value.
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: [[MAIN_RESULT:%[0-9]+]] = call { %main.state, i64 } @main.returnStateAndMut()
// CHECK-NEXT: [[MAIN_STATE:%[0-9]+]] = extractvalue { %main.state, i64 } [[MAIN_RESULT]], 0
// CHECK: [[MAIN_MUTATED:%[0-9]+]] = extractvalue { %main.state, i64 } [[MAIN_RESULT]], 1
// CHECK: [[MAIN_VALUE:%[0-9]+]] = load i64, ptr %{{[0-9]+}}
// CHECK-NEXT: [[MAIN_BAD_STATE:%[0-9]+]] = icmp ne i64 [[MAIN_VALUE]], 2
// CHECK-NEXT: br i1 [[MAIN_BAD_STATE]], label %{{.*}}, label %{{.*}}
// CHECK: [[MAIN_BAD_RESULT:%[0-9]+]] = icmp ne i64 [[MAIN_MUTATED]], 2
// CHECK-NEXT: br i1 [[MAIN_BAD_RESULT]], label %{{.*}}, label %{{.*}}
// CHECK-LABEL: define { %main.state, i64 } @main.returnStateAndMut(){{.*}} {
// CHECK: [[RETURN_MUTATED:%[0-9]+]] = call i64 @"main.(*state).mutate"(ptr [[RETURN_STATE:%[0-9]+]], i64 2)
// CHECK-NEXT: [[RETURN_VALUE:%[0-9]+]] = load %main.state, ptr [[RETURN_STATE]]
// CHECK-NEXT: [[RETURN_PAIR0:%[0-9]+]] = insertvalue { %main.state, i64 } undef, %main.state [[RETURN_VALUE]], 0
// CHECK-NEXT: [[RETURN_PAIR:%[0-9]+]] = insertvalue { %main.state, i64 } [[RETURN_PAIR0]], i64 [[RETURN_MUTATED]], 1
// CHECK-NEXT: ret { %main.state, i64 } [[RETURN_PAIR]]
// CHECK-LABEL: define i64 @"main.(*state).mutate"(ptr %0, i64 %1){{.*}} {
// CHECK: [[MUTATE_FIELD:%[0-9]+]] = getelementptr inbounds %main.state, ptr %0, i32 0, i32 0
// CHECK-NEXT: store i64 %1, ptr [[MUTATE_FIELD]]
// CHECK: [[MUTATE_RESULT_FIELD:%[0-9]+]] = getelementptr inbounds %main.state, ptr %0, i32 0, i32 0
// CHECK-NEXT: [[MUTATE_VALUE:%[0-9]+]] = load i64, ptr [[MUTATE_RESULT_FIELD]]
// CHECK-NEXT: ret i64 [[MUTATE_VALUE]]

import "fmt"

type state struct {
	v int
}

func main() {
	a, b := returnStateAndMut()
	if a.v != 2 || b != 2 {
		panic(fmt.Sprintf("return order mismatch: got (%d,%d), want (2,2)", a.v, b))
	}
	println("ok")
}

func returnStateAndMut() (state, int) {
	x := state{v: 1}
	return x, x.mutate(2)
}

func (s *state) mutate(next int) int {
	s.v = next
	return s.v
}
