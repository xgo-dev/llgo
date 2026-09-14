// LITTEST
// Scope: common
package main

import "github.com/xgo-dev/llgo/runtime/internal/runtime/math"

// CHECK-LABEL: define {{.*}} @main.runtimeMul(
// CHECK: call { i[[WIDTH:32|64]], i1 } @llvm.umul.with.overflow.i[[WIDTH]](
// CHECK-NOT: AssertDivideByZero
// CHECK-NOT: @"{{.*}}math.MulUintptr"
// CHECK: ret
func runtimeMul(a, b uintptr) (uintptr, bool) {
	return math.MulUintptr(a, b)
}

var multiplyValue = math.MulUintptr

func main() {
	max := ^uintptr(0)
	edges := []uintptr{0, 1, 2, 3, max / 2, max/2 + 1, max - 1, max}
	for p := uintptr(1); p != 0; p <<= 1 {
		edges = append(edges, p-1, p, p+1)
	}
	for _, a := range edges {
		for _, b := range edges {
			product, overflow := runtimeMul(a, b)
			want := a != 0 && b > max/a
			if product != a*b || overflow != want {
				panic("runtime MulUintptr mismatch")
			}
			product, overflow = multiplyValue(a, b)
			if product != a*b || overflow != want {
				panic("runtime MulUintptr function value mismatch")
			}
		}
	}
	println("runtime multiply overflow ok")
}
