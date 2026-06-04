// LITTEST
package main

import _ "unsafe"

//
//go:linkname asmFull llgo.asm
// CHECK-LINE: @18 = private unnamed_addr constant [5 x i8] c"value", align 1
// CHECK-LINE: @19 = private unnamed_addr constant [7 x i8] c"Result:", align 1
// CHECK-LINE: @20 = private unnamed_addr constant [1 x i8] c"x", align 1
// CHECK-LINE: @21 = private unnamed_addr constant [1 x i8] c"y", align 1

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
// CHECK-NEXT:   call void asm sideeffect "nop", ""()
// CHECK-NEXT:   %0 = call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map[_llgo_string]_llgo_any", i64 1)
// CHECK-NEXT:   %1 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 42, ptr %1, align 8
// CHECK-NEXT:   %2 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %1, 1
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @18, i64 5 }, ptr %3, align 8
// CHECK-NEXT:   %4 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_string]_llgo_any", ptr %0, ptr %3)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %2, ptr %4, align 8
// CHECK-NEXT:   call void asm sideeffect "# test value ${0}", "r"(i64 42)
// CHECK-NEXT:   %5 = call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map[_llgo_string]_llgo_any", i64 1)
// CHECK-NEXT:   %6 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 42, ptr %6, align 8
// CHECK-NEXT:   %7 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %6, 1
// CHECK-NEXT:   %8 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @18, i64 5 }, ptr %8, align 8
// CHECK-NEXT:   %9 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_string]_llgo_any", ptr %5, ptr %8)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %7, ptr %9, align 8
// CHECK-NEXT:   %10 = call i64 asm sideeffect "mov $0, ${1}", "=&r,r"(i64 42)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @19, i64 7 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 32)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %10)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %11 = call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map[_llgo_string]_llgo_any", i64 2)
// CHECK-NEXT:   %12 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 25, ptr %12, align 8
// CHECK-NEXT:   %13 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %12, 1
// CHECK-NEXT:   %14 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @20, i64 1 }, ptr %14, align 8
// CHECK-NEXT:   %15 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_string]_llgo_any", ptr %11, ptr %14)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %13, ptr %15, align 8
// CHECK-NEXT:   %16 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 17, ptr %16, align 8
// CHECK-NEXT:   %17 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %16, 1
// CHECK-NEXT:   %18 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 16)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.String" { ptr @21, i64 1 }, ptr %18, align 8
// CHECK-NEXT:   %19 = call ptr @"{{.*}}/runtime/internal/runtime.MapAssign"(ptr @"map[_llgo_string]_llgo_any", ptr %11, ptr %18)
// CHECK-NEXT:   store %"{{.*}}/runtime/internal/runtime.eface" %17, ptr %19, align 8
// CHECK-NEXT:   %20 = call i64 asm sideeffect "# calc ${1} + ${2} -> $0", "=&r,r,r"(i64 25, i64 17)
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
