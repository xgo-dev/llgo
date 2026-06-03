// LITTEST
package main

import "fmt"

type state struct {
	v int
}

// CHECK-LABEL: define void @"{{.*}}returnorder.main"(){{.*}} {
func main() {
	// CHECK: call { %"{{.*}}returnorder.state", i64 } @"{{.*}}returnorder.returnStateAndMut"()
	// CHECK: extractvalue { %"{{.*}}returnorder.state", i64 } %1, 0
	// CHECK: extractvalue { %"{{.*}}returnorder.state", i64 } %1, 1
	// CHECK: icmp ne i64 %5, 2
	// CHECK: call %"{{.*}}String" @fmt.Sprintf
	// CHECK: call void @"{{.*}}Panic"
	// CHECK: call void @"{{.*}}PrintString"(%"{{.*}}String" { ptr @3, i64 2 })
	// CHECK-NEXT: call void @"{{.*}}PrintByte"(i8 10)
	// CHECK-NEXT: ret void
	// CHECK: icmp ne i64 %3, 2
	a, b := returnStateAndMut()
	if a.v != 2 || b != 2 {
		panic(fmt.Sprintf("return order mismatch: got (%d,%d), want (2,2)", a.v, b))
	}
	println("ok")
}

// CHECK-LABEL: define { %"{{.*}}returnorder.state", i64 } @"{{.*}}returnorder.returnStateAndMut"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}AllocZ"(i64 8)
// CHECK-NEXT:   %1 = getelementptr inbounds %"{{.*}}returnorder.state", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i64 1, ptr %1, align 8
// CHECK-NEXT:   %2 = call i64 @"{{.*}}returnorder.(*state).mutate"(ptr %0, i64 2)
// CHECK-NEXT:   %3 = load %"{{.*}}returnorder.state", ptr %0, align 8
// CHECK-NEXT:   %4 = insertvalue { %"{{.*}}returnorder.state", i64 } undef, %"{{.*}}returnorder.state" %3, 0
// CHECK-NEXT:   %5 = insertvalue { %"{{.*}}returnorder.state", i64 } %4, i64 %2, 1
// CHECK-NEXT:   ret { %"{{.*}}returnorder.state", i64 } %5
func returnStateAndMut() (state, int) {
	x := state{v: 1}
	return x, x.mutate(2)
}

// CHECK-LABEL: define i64 @"{{.*}}returnorder.(*state).mutate"(ptr %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}returnorder.state", ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i64 %1, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds %"{{.*}}returnorder.state", ptr %0, i32 0, i32 0
// CHECK-NEXT:   %4 = load i64, ptr %3, align 8
// CHECK-NEXT:   ret i64 %4
func (s *state) mutate(next int) int {
	s.v = next
	return s.v
}
