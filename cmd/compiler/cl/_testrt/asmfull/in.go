// LITTEST darwin/arm64 linux/amd64 windows/386 windows/amd64 windows/arm64
package main

import _ "unsafe"

//go:linkname asmFull llgo.asm
func asmFull(instruction string, regs map[string]any) uintptr

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call void asm sideeffect "nop", ""()
// CHECK: call void asm sideeffect "# test value ${0}", "r"(i{{32|64}} 42)
// ARM64: [[ASM_RESULT:%[0-9]+]] = call i64 asm sideeffect "mov $0, ${1}", "=&r,r"(i64 42)
// AMD64: [[ASM_RESULT:%[0-9]+]] = call i64 asm sideeffect "movq ${1}, $0", "=&r,r"(i64 42)
// WINDOWS-386: [[ASM_RESULT:%[0-9]+]] = call i32 asm sideeffect "movl ${1}, $0", "=&r,r"(i32 42)
// WINDOWS-386: [[ASM_RESULT_EXT:%[0-9]+]] = zext i32 [[ASM_RESULT]] to i64
// WINDOWS-386-NEXT: call void @"{{.*}}.PrintUint"(i64 [[ASM_RESULT_EXT]])
// AMD64: call void @"{{.*}}.PrintUint"(i64 [[ASM_RESULT]])
// ARM64: call void @"{{.*}}.PrintUint"(i64 [[ASM_RESULT]])
// WINDOWS-386: [[ASM_UNUSED:%[0-9]+]] = call i32 asm sideeffect "# calc ${1} + ${2} -> $0", "=&r,r,r"(i32 25, i32 17)
// AMD64: [[ASM_UNUSED:%[0-9]+]] = call i64 asm sideeffect "# calc ${1} + ${2} -> $0", "=&r,r,r"(i64 25, i64 17)
// ARM64: [[ASM_UNUSED:%[0-9]+]] = call i64 asm sideeffect "# calc ${1} + ${2} -> $0", "=&r,r,r"(i64 25, i64 17)
// CHECK-NEXT: ret void
func main() {
	// no input,no return value
	asmFull("nop", nil)
	// input only,no return value
	asmFull("# test value {value}", map[string]any{"value": 42})
	// input with return value
	res1 := asmFull(moveInstruction, map[string]any{
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
