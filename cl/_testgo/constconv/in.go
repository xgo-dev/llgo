// LITTEST
package main

// CHECK-LABEL: define void @"{{.*}}constconv.main"(){{.*}} {
func main() {
	var i64 int64 = 1
	var u64 uint64 = 1
	var c int64 = i64 + (1.0 / (1.0 / 10))
	var d uint64 = u64 + (1.0 / (1.0 / 10))
	// CHECK: PrintUint"(i64 1)
	// CHECK-NEXT: PrintByte"(i8 32)
	// CHECK-NEXT: PrintInt"(i64 1)
	// CHECK-NEXT: PrintByte"(i8 32)
	// CHECK-NEXT: PrintInt"(i64 11)
	// CHECK-NEXT: PrintByte"(i8 32)
	// CHECK-NEXT: PrintUint"(i64 11)
	// CHECK-NEXT: PrintByte"(i8 10)
	// CHECK-NEXT: ret void
	println(u64, i64, c, d)
}
