// LITTEST
package main

import _ "unsafe"

//go:linkname asmFull llgo.asm
func asmFull(instruction string, regs map[string]any) uintptr

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/asmfull.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void asm sideeffect "nop", ""()
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.MakeMap"(ptr @"map[_llgo_string]_llgo_any", i64 1)
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store i64 42, ptr {{%[0-9]+}}, align 8
// CHECK: insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }
// CHECK: call void asm sideeffect "# test value ${0}", "r"(i64 42)
// CHECK: call i64 asm sideeffect "mov $0, ${1}", "=&r,r"(i64 42)
// CHECK: call i64 asm sideeffect "# calc ${1} + ${2} -> $0", "=&r,r,r"(i64 25, i64 17)
// CHECK: ret void
// CHECK-NEXT: }
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
