// LITTEST
// Scope: common
package main

import (
	"unsafe"

	"github.com/xgo-dev/llgo/cmd/cl/_testlibc/callbackmethod/local"
)

func visit(node local.Node, data unsafe.Pointer) int32 {
	*(*int32)(data) += node.Kind
	return node.Data * 2
}

func chooseVisitor(useFirst bool) func(local.Node, unsafe.Pointer) int32 {
	if useFirst {
		return visit
	}
	return func(node local.Node, data unsafe.Pointer) int32 {
		return visit(node, data)
	}
}

// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: call i32 @llgo_visit_node(
func main() {
	var seen int32
	node := local.Node{Kind: 21, Data: 21}
	data := unsafe.Pointer(&seen)
	var visitor local.VisitorAlias = visit
	for _, got := range []int32{
		local.VisitNode(node, visitor, data),
		node.Visit(visitor, data),
		local.Node.Visit(node, visitor, data),
		node.Visit(chooseVisitor(node.Kind != 0), data),
	} {
		if got != 43 {
			panic("C method callback returned incorrect values")
		}
	}
	bound := node.Visit
	if got := bound(visitor, data); got != 43 || seen != 105 {
		panic("bound C method callback returned incorrect values")
	}
	rejectCaptured(node, data, 7)
	if seen != 105 {
		panic("capturing callback reached C")
	}
	println("callback method ok")
}
