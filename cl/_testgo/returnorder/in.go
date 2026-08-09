// LITTEST
package main

import "fmt"

type state struct {
	v int
}

// CHECK-LABEL: define void @main.main(){{.*}} {
func main() {
	// CHECK: call { %main.state, i64 } @main.returnStateAndMut()
	// CHECK: extractvalue { %main.state, i64 } %1, 0
	// CHECK: extractvalue { %main.state, i64 } %1, 1
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

// CHECK-LABEL: define { %main.state, i64 } @main.returnStateAndMut(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call ptr @"{{.*}}AllocZ"(i64 8)
// CHECK-NEXT:   %1 = getelementptr inbounds %main.state, ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i64 1, ptr %1, align 8
// CHECK-NEXT:   %2 = call i64 @"main.(*state).mutate"(ptr %0, i64 2)
// CHECK-NEXT:   %3 = load %main.state, ptr %0, align 8
// CHECK-NEXT:   %4 = insertvalue { %main.state, i64 } undef, %main.state %3, 0
// CHECK-NEXT:   %5 = insertvalue { %main.state, i64 } %4, i64 %2, 1
// CHECK-NEXT:   ret { %main.state, i64 } %5
// ESCAPE-LABEL: define { %main.state, i64 } @main.returnStateAndMut(){{.*}} {
// ESCAPE-NEXT: _llgo_0:
// ESCAPE-NEXT:   %.stack = alloca i8, i64 8, align 8
// ESCAPE-NEXT:   call void @llvm.memset.p0.i64(ptr %.stack, i8 0, i64 8, i1 false)
// ESCAPE-NEXT:   %0 = getelementptr inbounds %main.state, ptr %.stack, i32 0, i32 0
// ESCAPE-NEXT:   store i64 1, ptr %0, align 8
// ESCAPE-NEXT:   %1 = call i64 @"main.(*state).mutate"(ptr %.stack, i64 2)
// ESCAPE-NEXT:   %2 = load %main.state, ptr %.stack, align 8
// ESCAPE-NEXT:   %3 = insertvalue { %main.state, i64 } undef, %main.state %2, 0
// ESCAPE-NEXT:   %4 = insertvalue { %main.state, i64 } %3, i64 %1, 1
// ESCAPE-NEXT:   ret { %main.state, i64 } %4
// ESCAPE-NEXT: }
func returnStateAndMut() (state, int) {
	x := state{v: 1}
	return x, x.mutate(2)
}

// CHECK-LABEL: define i64 @"main.(*state).mutate"(ptr %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = getelementptr inbounds %main.state, ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i64 %1, ptr %2, align 8
// CHECK-NEXT:   %3 = getelementptr inbounds %main.state, ptr %0, i32 0, i32 0
// CHECK-NEXT:   %4 = load i64, ptr %3, align 8
// CHECK-NEXT:   ret i64 %4
func (s *state) mutate(next int) int {
	s.v = next
	return s.v
}
