// LITTEST
package main

func end() {
	if recovered := recover(); recovered != nil {
		defer panic(recovered)
		println("will panic in defer")
	}
	println("end")
}

func main() {
	defer end()
	panic("panic in main")
}

// A noreturn panic must terminate its LLVM block. In particular, the defer
// cleanup emitted for end must not append an instruction after unreachable.
// CHECK-LABEL: define void @main.end(){{.*}} {
// CHECK: call void @"{{.*}}/runtime/internal/runtime.Panic"
// CHECK-NEXT: unreachable
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_{{[A-Za-z0-9_]+}}:
