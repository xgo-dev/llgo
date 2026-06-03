// LITTEST
package main

import (
	"math/bits"
)

// CHECK-LABEL: define void @"{{.*}}/cl/_testlibgo/mathbits.main"(){{.*}} {
func main() {
	// CHECK: %0 = call i64 @"math/bits.Len8"(i8 20)
	// CHECK: %1 = call i64 @"math/bits.OnesCount"(i64 20)
	println(bits.Len8(20))
	println(bits.OnesCount(20))
}
