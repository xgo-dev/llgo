// LITTEST
package main

// CHECK-LABEL: define void @"{{.*}}print.main"(){{.*}} {
func main() {
	// CHECK: PrintInt"(i64 46)
	// CHECK-NEXT: PrintByte"(i8 32)
	// CHECK-NEXT: PrintUint"(i64 46)
	// CHECK-NEXT: PrintByte"(i8 10)
	// CHECK-NOT: PrintString"
	// CHECK: ret void
	println('.', byte('.'))
}
