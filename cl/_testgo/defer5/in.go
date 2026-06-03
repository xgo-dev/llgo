// LITTEST
package main

// CHECK-LABEL: define void @"{{.*}}.main"(){{.*}} {
func main() {
	// CHECK: GetThreadDefer
	// CHECK: blockaddress(@"{{.*}}.main", %_llgo_2)
	// CHECK: blockaddress(@"{{.*}}.main", %_llgo_8)
	// CHECK: call void @"{{.*}}.main$1"()
	// CHECK: call void @"{{.*}}.main$2"()
	// CHECK: FreeDeferNode
	// CHECK: FreeDeferNode
	defer println("A")
	defer func() {
		if e := recover(); e != nil {
			println("in defer 1")
			panic("panic in defer 1")
		}
	}()
	defer func() {
		println("in defer 2")
		panic("panic in defer 2")
	}()
	defer println("B")
	panic("panic in main")
}

// CHECK-LABEL: define void @"{{.*}}.main$1"(){{.*}} {
// CHECK: Recover
// CHECK: PrintString
// CHECK: PrintByte
// CHECK-LABEL: define void @"{{.*}}.main$2"(){{.*}} {
// CHECK: PrintString
// CHECK: PrintByte
