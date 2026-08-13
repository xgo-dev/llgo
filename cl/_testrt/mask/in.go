// LITTEST
package main

func main() {
	println(mask(1))
	println(mask_shl(127, 5))
	println(mask_shl8(127, 5))
	println(mask_shl8u(127, 5))
	println(mask_shl8(127, 16))
	println(mask_shl8u(127, 16))
	println(mask_shr(127, 5))
	println(mask_shr8(127, 5))
	println(mask_shr8u(127, 5))
	println(mask_shr8(127, 16))
}

// The sign bit must survive the two constant shifts, including the explicit
// overshift select inserted by the lowering.
// CHECK-LABEL: define i32 @main.mask(i8 %0){{.*}} {
// CHECK: [[MASK_EXT:%[0-9]+]] = sext i8 %0 to i32
// CHECK-NEXT: [[MASK_SHL:%[0-9]+]] = shl i32 [[MASK_EXT]], 31
// CHECK-NEXT: [[MASK_SAFE:%[0-9]+]] = select i1 false, i32 0, i32 [[MASK_SHL]]
// CHECK-NEXT: [[MASK_RESULT:%[0-9]+]] = ashr i32 [[MASK_SAFE]], 31
// CHECK-NEXT: ret i32 [[MASK_RESULT]]
func mask(x int8) int32 {
	return int32(x) << 31 >> 31
}

// CHECK-LABEL: define i64 @main.mask_shl(i64 %0, i64 %1){{.*}} {
// CHECK: [[SHL_NEG:%[0-9]+]] = icmp slt i64 %1, 0
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.AssertNegativeShift"(i1 [[SHL_NEG]])
// CHECK-NEXT: [[SHL_WIDE:%[0-9]+]] = icmp uge i64 %1, 64
// CHECK-NEXT: [[SHL_VALUE:%[0-9]+]] = shl i64 %0, %1
// CHECK-NEXT: [[SHL_RESULT:%[0-9]+]] = select i1 [[SHL_WIDE]], i64 0, i64 [[SHL_VALUE]]
// CHECK-NEXT: ret i64 [[SHL_RESULT]]
func mask_shl(x int, y int) int {
	return x << y
}

// CHECK-LABEL: define i8 @main.mask_shl8(i8 %0, i64 %1){{.*}} {
// CHECK: [[SHL8_NEG:%[0-9]+]] = icmp slt i64 %1, 0
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.AssertNegativeShift"(i1 [[SHL8_NEG]])
// CHECK-NEXT: [[SHL8_COUNT:%[0-9]+]] = trunc i64 %1 to i8
// CHECK-NEXT: [[SHL8_WIDE:%[0-9]+]] = icmp uge i8 [[SHL8_COUNT]], 8
// CHECK-NEXT: [[SHL8_VALUE:%[0-9]+]] = shl i8 %0, [[SHL8_COUNT]]
// CHECK-NEXT: [[SHL8_RESULT:%[0-9]+]] = select i1 [[SHL8_WIDE]], i8 0, i8 [[SHL8_VALUE]]
// CHECK-NEXT: ret i8 [[SHL8_RESULT]]
func mask_shl8(x int8, y int) int8 {
	return x << y
}

// CHECK-LABEL: define i8 @main.mask_shl8u(i8 %0, i64 %1){{.*}} {
// CHECK: [[SHL8U_NEG:%[0-9]+]] = icmp slt i64 %1, 0
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.AssertNegativeShift"(i1 [[SHL8U_NEG]])
// CHECK-NEXT: [[SHL8U_COUNT:%[0-9]+]] = trunc i64 %1 to i8
// CHECK-NEXT: [[SHL8U_WIDE:%[0-9]+]] = icmp uge i8 [[SHL8U_COUNT]], 8
// CHECK-NEXT: [[SHL8U_VALUE:%[0-9]+]] = shl i8 %0, [[SHL8U_COUNT]]
// CHECK-NEXT: [[SHL8U_RESULT:%[0-9]+]] = select i1 [[SHL8U_WIDE]], i8 0, i8 [[SHL8U_VALUE]]
// CHECK-NEXT: ret i8 [[SHL8U_RESULT]]
func mask_shl8u(x uint8, y int) uint8 {
	return x << y
}

// Signed right shifts clamp an oversized count to width-1, preserving sign.
// CHECK-LABEL: define i64 @main.mask_shr(i64 %0, i64 %1){{.*}} {
// CHECK: [[SHR_NEG:%[0-9]+]] = icmp slt i64 %1, 0
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.AssertNegativeShift"(i1 [[SHR_NEG]])
// CHECK-NEXT: [[SHR_WIDE:%[0-9]+]] = icmp uge i64 %1, 64
// CHECK-NEXT: [[SHR_COUNT:%[0-9]+]] = select i1 [[SHR_WIDE]], i64 63, i64 %1
// CHECK-NEXT: [[SHR_RESULT:%[0-9]+]] = ashr i64 %0, [[SHR_COUNT]]
// CHECK-NEXT: ret i64 [[SHR_RESULT]]
func mask_shr(x int, y int) int {
	return x >> y
}

// CHECK-LABEL: define i8 @main.mask_shr8(i8 %0, i64 %1){{.*}} {
// CHECK: [[SHR8_NEG:%[0-9]+]] = icmp slt i64 %1, 0
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.AssertNegativeShift"(i1 [[SHR8_NEG]])
// CHECK-NEXT: [[SHR8_TRUNC:%[0-9]+]] = trunc i64 %1 to i8
// CHECK-NEXT: [[SHR8_WIDE:%[0-9]+]] = icmp uge i8 [[SHR8_TRUNC]], 8
// CHECK-NEXT: [[SHR8_COUNT:%[0-9]+]] = select i1 [[SHR8_WIDE]], i8 7, i8 [[SHR8_TRUNC]]
// CHECK-NEXT: [[SHR8_RESULT:%[0-9]+]] = ashr i8 %0, [[SHR8_COUNT]]
// CHECK-NEXT: ret i8 [[SHR8_RESULT]]
func mask_shr8(x int8, y int) int8 {
	return x >> y
}

// Unsigned right shifts instead select zero for an oversized count.
// CHECK-LABEL: define i8 @main.mask_shr8u(i8 %0, i64 %1){{.*}} {
// CHECK: [[SHR8U_NEG:%[0-9]+]] = icmp slt i64 %1, 0
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.AssertNegativeShift"(i1 [[SHR8U_NEG]])
// CHECK-NEXT: [[SHR8U_COUNT:%[0-9]+]] = trunc i64 %1 to i8
// CHECK-NEXT: [[SHR8U_WIDE:%[0-9]+]] = icmp uge i8 [[SHR8U_COUNT]], 8
// CHECK-NEXT: [[SHR8U_VALUE:%[0-9]+]] = lshr i8 %0, [[SHR8U_COUNT]]
// CHECK-NEXT: [[SHR8U_RESULT:%[0-9]+]] = select i1 [[SHR8U_WIDE]], i8 0, i8 [[SHR8U_VALUE]]
// CHECK-NEXT: ret i8 [[SHR8U_RESULT]]
func mask_shr8u(x uint8, y int) uint8 {
	return x >> y
}
