// LITTEST
// Scope: common
package main

import "github.com/xgo-dev/llgo/cmd/cl/_testlibc/cppabi/local"

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call i64 @llgo_test_cpp_add(i64 20, i64 22)
func main() {
	if local.Add(20, 22) != 42 {
		panic("C++ ABI bridge returned the wrong value")
	}
	println("cpp abi ok")
}
