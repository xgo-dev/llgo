// LITTEST: POST-ABI darwin/arm64 linux/amd64 linux/arm64 windows/amd64
// Scope: arch (arm64/amd64 float-to-integer ABI)
package main

func f32ToI32(v float32) int32       { return int32(v) }
func f32ToU32(v float32) uint32      { return uint32(v) }
func f64ToUintptr(v float64) uintptr { return uintptr(v) }

const threshold = 100

var (
	scalar  float64 = 1
	counter         = 100
)

func zConstantsAndGlobals() {
	if threshold > 100 {
		scalar = 0
	}
	counter++
}

func zControlInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func zControlFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func zIncrementU32(v uint32) uint32 { return v + 1 }

// Preserve the mixed untyped float/int regression exactly: both reciprocal
// expressions are constant-folded to ten before the typed additions.
func zMixedUntypedFloat(i64 int64, u64 uint64) (uint64, int64, int64, uint64) {
	c := i64 + (1.0 / (1.0 / 10))
	d := u64 + (1.0 / (1.0 / 10))
	return u64, i64, c, d
}

// Saturating targets use intrinsics while amd64 keeps its legacy CVTT
// behavior. uint32 must still convert through signed i64 before truncation.
// CHECK-LABEL: define i32 @main.f32ToI32(
// ARM64: call i32 @llvm.fptosi.sat.i32.f32(float
// AMD64: fcmp olt float {{.*}}, 0xC1E0000000000000
// AMD64: fcmp oge float {{.*}}, 0x41E0000000000000
// AMD64: fcmp uno float
// AMD64: fptosi float {{.*}} to i32
// AMD64: select i1 {{.*}}, i32 -2147483648, i32
// CHECK: ret i32

// CHECK-LABEL: define i32 @main.f32ToU32(
// ARM64: call i64 @llvm.fptosi.sat.i64.f32(float
// AMD64: fcmp olt float {{.*}}, 0xC3E0000000000000
// AMD64: fcmp oge float {{.*}}, 0x43E0000000000000
// AMD64: fptosi float {{.*}} to i64
// AMD64: select i1 {{.*}}, i64 -9223372036854775808, i64
// CHECK: trunc i64 {{.*}} to i32
// CHECK: ret i32

// CHECK-LABEL: define i64 @main.f64ToUintptr(
// ARM64: call i64 @llvm.fptoui.sat.i64.f64(double
// AMD64: fcmp oge double {{.*}}, 0x43E0000000000000
// AMD64: fsub double {{.*}}, 0x43E0000000000000
// AMD64: fptosi double {{.*}} to i64
// AMD64: or i64
// CHECK: ret i64

// The former fncall/untyped/uint packages only added these scalar paths.
// Keep them beside the target-sensitive numeric conversions without another
// package compile.
// CHECK-LABEL: define void @main.zConstantsAndGlobals(){{.*}} {
// CHECK: store double 0.000000e+00, ptr @main.scalar
// CHECK: [[OLD_COUNTER:%[0-9]+]] = load i64, ptr @main.counter
// CHECK-NEXT: [[NEW_COUNTER:%[0-9]+]] = add i64 [[OLD_COUNTER]], 1
// CHECK-NEXT: store i64 [[NEW_COUNTER]], ptr @main.counter

// CHECK-LABEL: define double @main.zControlFloat(double %0, double %1){{.*}} {
// CHECK: [[FLOAT_GREATER:%[0-9]+]] = fcmp ogt double %0, %1
// CHECK-NEXT: br i1 [[FLOAT_GREATER]], label %{{[^,]+}}, label %{{[^ ]+}}
// CHECK: ret double %0
// CHECK: ret double %1

// CHECK-LABEL: define i64 @main.zControlInt(i64 %0, i64 %1){{.*}} {
// CHECK: [[INT_GREATER:%[0-9]+]] = icmp sgt i64 %0, %1
// CHECK-NEXT: br i1 [[INT_GREATER]], label %{{[^,]+}}, label %{{[^ ]+}}
// CHECK: ret i64 %0
// CHECK: ret i64 %1

// CHECK-LABEL: define i32 @main.zIncrementU32(i32 %0){{.*}} {
// CHECK: [[INCREMENTED:%[0-9]+]] = add i32 %0, 1
// CHECK-NEXT: ret i32 [[INCREMENTED]]

// CHECK-LABEL: define void @main.zMixedUntypedFloat(
// CHECK-SAME: ptr sret({ i64, i64, i64, i64 }) %[[MIXED_RESULT:[0-9]+]], i64 %[[MIXED_I64:[0-9]+]], i64 %[[MIXED_U64:[0-9]+]]){{.*}} {
// CHECK: [[MIXED_SIGNED:%[0-9]+]] = add i64 %[[MIXED_I64]], 10
// CHECK-NEXT: [[MIXED_UNSIGNED:%[0-9]+]] = add i64 %[[MIXED_U64]], 10
// CHECK: insertvalue { i64, i64, i64, i64 } {{.*}}, i64 [[MIXED_SIGNED]], 2
// CHECK: [[MIXED_VALUE:%[0-9]+]] = insertvalue { i64, i64, i64, i64 } {{.*}}, i64 [[MIXED_UNSIGNED]], 3
// CHECK-NEXT: store { i64, i64, i64, i64 } [[MIXED_VALUE]], ptr %[[MIXED_RESULT]]
