// LITTEST
package main

import "unsafe"

// The loop carries the accumulated string and keeps its binary concat path.
// CHECK-LABEL: define {{.*}} @main.concat(
// CHECK: phi %"{{.*}}/runtime/internal/runtime.String"
// CHECK: getelementptr inbounds %"{{.*}}/runtime/internal/runtime.String"
// CHECK: call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringCat"
func concat(args ...string) (ret string) {
	for _, v := range args {
		ret += v
	}
	return
}

// The source conversion may use a local buffer, but the fused result must
// survive this function and later allocations. In particular, neither empty
// operands nor short strings justify returning borrowed input storage.
//
//go:noinline
func fromStack(seed byte) string {
	var bytes [8]byte
	for i := range bytes {
		bytes[i] = seed
	}
	part := string(bytes[:])
	return part + "/" + part
}

//go:noinline
func grouped(seed byte) string {
	part := string([]byte{seed, ':'})
	return part + "0" + part + "1" + part + "2" + part + "3" + part + "4" +
		part + "5" + part + "6" + part + "7" + part + "8"
}

// Removing the empty literal leaves a binary concatenation.
// CHECK-LABEL: define {{.*}} @main.info(
// CHECK-NOT: runtime/internal/runtime.StringCat
// CHECK-NOT: alloca [{{[0-9]+}} x %"{{.*}}/runtime/internal/runtime.String"]
// CHECK: call %"{{.*}}/runtime/internal/runtime.String" @"{{.*}}/runtime/internal/runtime.StringCat"
// CHECK-NOT: runtime/internal/runtime.StringCat
// CHECK: ret %"{{.*}}/runtime/internal/runtime.String"
func info(s string) string {
	return "" + s + "..."
}

type name string

//go:noinline
func copyName(s name) name { return "" + s + "" }

func main() {
	result := concat("Hello", " ", "World")
	if info(result) != "Hello World..." {
		panic("fused concat result")
	}
	copied := copyName(name(result))
	if string(copied) != result || unsafe.StringData(string(copied)) == unsafe.StringData(result) {
		panic("single remaining operand must be copied")
	}
	if grouped('x') != "x:0x:1x:2x:3x:4x:5x:6x:7x:8" {
		panic("grouped concat order")
	}
	saved := make([]string, 64)
	for i := range saved {
		saved[i] = fromStack(byte('A' + i%26))
	}
	for i, value := range saved {
		seed := byte('A' + i%26)
		if len(value) != 17 || value[8] != '/' {
			panic("fused concat lifetime")
		}
		for j := 0; j < 8; j++ {
			if value[j] != seed || value[j+9] != seed {
				panic("fused concat input lifetime")
			}
		}
	}
	println(result)
}
