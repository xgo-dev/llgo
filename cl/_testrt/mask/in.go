// LITTEST
package main

// CHECK-LABEL: define void @"{{.*}}/cl/_testrt/mask.main"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = call i32 @"{{.*}}/cl/_testrt/mask.mask"(i8 1)
// CHECK-NEXT:   %1 = sext i32 %0 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %1)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %2 = call i64 @"{{.*}}/cl/_testrt/mask.mask_shl"(i64 127, i64 5)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %2)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %3 = call i8 @"{{.*}}/cl/_testrt/mask.mask_shl8"(i8 127, i64 5)
// CHECK-NEXT:   %4 = sext i8 %3 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %4)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %5 = call i8 @"{{.*}}/cl/_testrt/mask.mask_shl8u"(i8 127, i64 5)
// CHECK-NEXT:   %6 = zext i8 %5 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %6)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %7 = call i8 @"{{.*}}/cl/_testrt/mask.mask_shl8"(i8 127, i64 16)
// CHECK-NEXT:   %8 = sext i8 %7 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %8)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %9 = call i8 @"{{.*}}/cl/_testrt/mask.mask_shl8u"(i8 127, i64 16)
// CHECK-NEXT:   %10 = zext i8 %9 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %10)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %11 = call i64 @"{{.*}}/cl/_testrt/mask.mask_shr"(i64 127, i64 5)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %11)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %12 = call i8 @"{{.*}}/cl/_testrt/mask.mask_shr8"(i8 127, i64 5)
// CHECK-NEXT:   %13 = sext i8 %12 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %13)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %14 = call i8 @"{{.*}}/cl/_testrt/mask.mask_shr8u"(i8 127, i64 5)
// CHECK-NEXT:   %15 = zext i8 %14 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintUint"(i64 %15)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   %16 = call i8 @"{{.*}}/cl/_testrt/mask.mask_shr8"(i8 127, i64 16)
// CHECK-NEXT:   %17 = sext i8 %16 to i64
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintInt"(i64 %17)
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
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

// CHECK-LABEL: define i32 @"{{.*}}/cl/_testrt/mask.mask"(i8 %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = sext i8 %0 to i32
// CHECK-NEXT:   %2 = shl i32 %1, 31
// CHECK-NEXT:   %3 = select i1 false, i32 0, i32 %2
// CHECK-NEXT:   %4 = ashr i32 %3, 31
// CHECK-NEXT:   ret i32 %4
// CHECK-NEXT: }
func mask(x int8) int32 {
	return int32(x) << 31 >> 31
}

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testrt/mask.mask_shl"(i64 %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = icmp slt i64 %1, 0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNegativeShift"(i1 %2)
// CHECK-NEXT:   %3 = icmp uge i64 %1, 64
// CHECK-NEXT:   %4 = shl i64 %0, %1
// CHECK-NEXT:   %5 = select i1 %3, i64 0, i64 %4
// CHECK-NEXT:   ret i64 %5
// CHECK-NEXT: }
func mask_shl(x int, y int) int {
	return x << y
}

// CHECK-LABEL: define i8 @"{{.*}}/cl/_testrt/mask.mask_shl8"(i8 %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = icmp slt i64 %1, 0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNegativeShift"(i1 %2)
// CHECK-NEXT:   %3 = trunc i64 %1 to i8
// CHECK-NEXT:   %4 = icmp uge i8 %3, 8
// CHECK-NEXT:   %5 = shl i8 %0, %3
// CHECK-NEXT:   %6 = select i1 %4, i8 0, i8 %5
// CHECK-NEXT:   ret i8 %6
// CHECK-NEXT: }
func mask_shl8(x int8, y int) int8 {
	return x << y
}

// CHECK-LABEL: define i8 @"{{.*}}/cl/_testrt/mask.mask_shl8u"(i8 %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = icmp slt i64 %1, 0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNegativeShift"(i1 %2)
// CHECK-NEXT:   %3 = trunc i64 %1 to i8
// CHECK-NEXT:   %4 = icmp uge i8 %3, 8
// CHECK-NEXT:   %5 = shl i8 %0, %3
// CHECK-NEXT:   %6 = select i1 %4, i8 0, i8 %5
// CHECK-NEXT:   ret i8 %6
// CHECK-NEXT: }
func mask_shl8u(x uint8, y int) uint8 {
	return x << y
}

// CHECK-LABEL: define i64 @"{{.*}}/cl/_testrt/mask.mask_shr"(i64 %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = icmp slt i64 %1, 0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNegativeShift"(i1 %2)
// CHECK-NEXT:   %3 = icmp uge i64 %1, 64
// CHECK-NEXT:   %4 = select i1 %3, i64 63, i64 %1
// CHECK-NEXT:   %5 = ashr i64 %0, %4
// CHECK-NEXT:   ret i64 %5
// CHECK-NEXT: }
func mask_shr(x int, y int) int {
	return x >> y
}

// CHECK-LABEL: define i8 @"{{.*}}/cl/_testrt/mask.mask_shr8"(i8 %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = icmp slt i64 %1, 0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNegativeShift"(i1 %2)
// CHECK-NEXT:   %3 = trunc i64 %1 to i8
// CHECK-NEXT:   %4 = icmp uge i8 %3, 8
// CHECK-NEXT:   %5 = select i1 %4, i8 7, i8 %3
// CHECK-NEXT:   %6 = ashr i8 %0, %5
// CHECK-NEXT:   ret i8 %6
// CHECK-NEXT: }
func mask_shr8(x int8, y int) int8 {
	return x >> y
}

// CHECK-LABEL: define i8 @"{{.*}}/cl/_testrt/mask.mask_shr8u"(i8 %0, i64 %1){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %2 = icmp slt i64 %1, 0
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.AssertNegativeShift"(i1 %2)
// CHECK-NEXT:   %3 = trunc i64 %1 to i8
// CHECK-NEXT:   %4 = icmp uge i8 %3, 8
// CHECK-NEXT:   %5 = lshr i8 %0, %3
// CHECK-NEXT:   %6 = select i1 %4, i8 0, i8 %5
// CHECK-NEXT:   ret i8 %6
// CHECK-NEXT: }
func mask_shr8u(x uint8, y int) uint8 {
	return x >> y
}
