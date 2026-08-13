// LITTEST
package main

const c = 100

var a float64 = 1

// CHECK: @main.a = global double 1.000000e+00

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: store double 0.000000e+00, ptr @main.a
func main() {
	if c > 100 {
		a = 0
	}
}
