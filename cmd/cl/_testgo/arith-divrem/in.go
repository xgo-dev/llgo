// LITTEST
// Scope: common
package main

// Integer division must guard LLVM's undefined zero-divisor case and the
// signed minInt/-1 overflow while preserving Go's defined result.

// CHECK-LABEL: define i64 @main.signedDiv(
// CHECK: [[SD_ZERO:%[0-9]+]] = icmp eq i64 %1, 0
// CHECK: call void @"{{.*}}/runtime/internal/runtime.AssertDivideByZero"(i1 [[SD_ZERO]])
// CHECK: [[SD_ZERO_SAFE:%[0-9]+]] = select i1 [[SD_ZERO]], i64 1, i64 %1
// CHECK: [[SD_MIN:%[0-9]+]] = icmp eq i64 %0, -9223372036854775808
// CHECK: [[SD_NEG_ONE:%[0-9]+]] = icmp eq i64 %1, -1
// CHECK: [[SD_OVERFLOW:%[0-9]+]] = and i1 [[SD_MIN]], [[SD_NEG_ONE]]
// CHECK: [[SD_SAFE_X:%[0-9]+]] = select i1 [[SD_OVERFLOW]], i64 0, i64 %0
// CHECK: [[SD_SAFE_Y:%[0-9]+]] = select i1 [[SD_OVERFLOW]], i64 1, i64 [[SD_ZERO_SAFE]]
// CHECK: [[SD_VALUE:%[0-9]+]] = sdiv i64 [[SD_SAFE_X]], [[SD_SAFE_Y]]
// CHECK: select i1 [[SD_OVERFLOW]], i64 %0, i64 [[SD_VALUE]]
func signedDiv(x, y int64) int64 { return x / y }

// CHECK-LABEL: define i64 @main.signedRem(
// CHECK: [[SR_ZERO:%[0-9]+]] = icmp eq i64 %1, 0
// CHECK: call void @"{{.*}}/runtime/internal/runtime.AssertDivideByZero"(i1 [[SR_ZERO]])
// CHECK: [[SR_ZERO_SAFE:%[0-9]+]] = select i1 [[SR_ZERO]], i64 1, i64 %1
// CHECK: [[SR_MIN:%[0-9]+]] = icmp eq i64 %0, -9223372036854775808
// CHECK: [[SR_NEG_ONE:%[0-9]+]] = icmp eq i64 %1, -1
// CHECK: [[SR_OVERFLOW:%[0-9]+]] = and i1 [[SR_MIN]], [[SR_NEG_ONE]]
// CHECK: [[SR_SAFE_X:%[0-9]+]] = select i1 [[SR_OVERFLOW]], i64 0, i64 %0
// CHECK: [[SR_SAFE_Y:%[0-9]+]] = select i1 [[SR_OVERFLOW]], i64 1, i64 [[SR_ZERO_SAFE]]
// CHECK: [[SR_VALUE:%[0-9]+]] = srem i64 [[SR_SAFE_X]], [[SR_SAFE_Y]]
// CHECK: select i1 [[SR_OVERFLOW]], i64 0, i64 [[SR_VALUE]]
func signedRem(x, y int64) int64 { return x % y }

// CHECK-LABEL: define i64 @main.unsignedDiv(
// CHECK: [[UD_ZERO:%[0-9]+]] = icmp eq i64 %1, 0
// CHECK: call void @"{{.*}}/runtime/internal/runtime.AssertDivideByZero"(i1 [[UD_ZERO]])
// CHECK: [[UD_SAFE_Y:%[0-9]+]] = select i1 [[UD_ZERO]], i64 1, i64 %1
// CHECK: udiv i64 %0, [[UD_SAFE_Y]]
func unsignedDiv(x, y uint64) uint64 { return x / y }

// CHECK-LABEL: define i64 @main.unsignedRem(
// CHECK: [[UR_ZERO:%[0-9]+]] = icmp eq i64 %1, 0
// CHECK: call void @"{{.*}}/runtime/internal/runtime.AssertDivideByZero"(i1 [[UR_ZERO]])
// CHECK: [[UR_SAFE_Y:%[0-9]+]] = select i1 [[UR_ZERO]], i64 1, i64 %1
// CHECK: urem i64 %0, [[UR_SAFE_Y]]
func unsignedRem(x, y uint64) uint64 { return x % y }

func main() {
	println(signedDiv(17, 5), signedRem(-17, 5))
	println(unsignedDiv(17, 5), unsignedRem(17, 5))
	min := int64(-1 << 63)
	println(signedDiv(min, -1), signedRem(min, -1))
}
