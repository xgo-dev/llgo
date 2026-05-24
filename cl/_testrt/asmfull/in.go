// LITTEST
package main

import _ "unsafe"

//go:linkname asmFull llgo.asm

// CHECK-LINE: @0 = private unnamed_addr constant [3 x i8] c"nop", align 1
// CHECK-LINE: @19 = private unnamed_addr constant [5 x i8] c"value", align 1
// CHECK-LINE: @20 = private unnamed_addr constant [20 x i8] c"# test value {value}", align 1
// CHECK-LINE: @21 = private unnamed_addr constant [15 x i8] c"mov {}, {value}", align 1
// CHECK-LINE: @22 = private unnamed_addr constant [7 x i8] c"Result:", align 1
// CHECK-LINE: @23 = private unnamed_addr constant [1 x i8] c"x", align 1
// CHECK-LINE: @24 = private unnamed_addr constant [1 x i8] c"y", align 1
// CHECK-LINE: @25 = private unnamed_addr constant [22 x i8] c"# calc {x} + {y} -> {}", align 1

func asmFull(instruction string, regs map[string]any) uintptr

func main() {
	// no input,no return value
	asmFull("nop", nil)
	// input only,no return value
	asmFull("# test value {value}", map[string]any{"value": 42})
	// input with return value
	res1 := asmFull("mov {}, {value}", map[string]any{
		"value": 42,
	})
	println("Result:", res1)
	// note(zzy): multiple inputs with return value
	// only for test register & constraint,not have actual meaning
	// the ir compare cannot crossplatform currently
	// so just use a comment to test it
	res2 := asmFull("# calc {x} + {y} -> {}", map[string]any{
		"x": 25,
		"y": 17,
	})
	// the result of asmFull on a comment is undefined, just make sure it can be compiled successfully.
	_ = res2
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/asmfull.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testrt/asmfull.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testrt/asmfull.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/asmfull.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i64 @"{{.*}}/cl/_testrt/asmfull.asmFull"(%"{{.*}}/runtime/internal/runtime.String" { ptr @0, i64 3 }, ptr null)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map[_llgo_string]_llgo_any", i64 1)
// CHECK-NEXT:   %2 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 42, ptr %2, align 8
// CHECK-NEXT:   %3 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %2, 1
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @19, i64 5 }, ptr %4, align 8
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_string]_llgo_any", ptr %1, ptr %4)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %3, ptr %5, align 8
// CHECK-NEXT:   %6 = call i64 @"{{.*}}/cl/_testrt/asmfull.asmFull"(%"{{.*}}/runtime/internal/runtime.String" { ptr @20, i64 20 }, ptr %1)
// CHECK-NEXT:   %7 = call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map[_llgo_string]_llgo_any", i64 1)
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 42, ptr %8, align 8
// CHECK-NEXT:   %9 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %8, 1
// CHECK-NEXT:   %10 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @19, i64 5 }, ptr %10, align 8
// CHECK-NEXT:   %11 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_string]_llgo_any", ptr %7, ptr %10)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %9, ptr %11, align 8
// CHECK-NEXT:   %12 = call i64 @"{{.*}}/cl/_testrt/asmfull.asmFull"(%"{{.*}}/runtime/internal/runtime.String" { ptr @21, i64 15 }, ptr %7)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @22, i64 7 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %12)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %13 = call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map[_llgo_string]_llgo_any", i64 2)
// CHECK-NEXT:   %14 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 25, ptr %14, align 8
// CHECK-NEXT:   %15 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %14, 1
// CHECK-NEXT:   %16 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @23, i64 1 }, ptr %16, align 8
// CHECK-NEXT:   %17 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_string]_llgo_any", ptr %13, ptr %16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %15, ptr %17, align 8
// CHECK-NEXT:   %18 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 17, ptr %18, align 8
// CHECK-NEXT:   %19 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %18, 1
// CHECK-NEXT:   %20 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @24, i64 1 }, ptr %20, align 8
// CHECK-NEXT:   %21 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_string]_llgo_any", ptr %13, ptr %20)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %19, ptr %21, align 8
// CHECK-NEXT:   %22 = call i64 @"{{.*}}/cl/_testrt/asmfull.asmFull"(%"{{.*}}/runtime/internal/runtime.String" { ptr @25, i64 22 }, ptr %13)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.nilinterequal"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.nilinterequal"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal8"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal8"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal64"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal64"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }
