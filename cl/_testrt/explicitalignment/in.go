// LITTEST
// Scope: common
package main

import (
	"unsafe"

	"github.com/xgo-dev/llgo/cl/_testrt/explicitalignment/local"
)

var records [2]local.Record

func check(items []local.Record) {
	items[0] = local.Record{Tag: 1, Value: 17, Tail: 91}
	items[1] = local.Record{Tag: 2, Value: 29, Tail: 93}
	if local.Check(&items[0], unsafe.Sizeof(items[0]), unsafe.Alignof(items[0]), unsafe.Offsetof(items[0].Value), unsafe.Offsetof(items[0].Tail)) != 0 {
		panic("C pointer layout or alignment")
	}
	if items[0].Value != 17 || items[0].Tail != 97 || items[1].Value != 41 || items[1].Tail != 93 {
		panic("C array write or stride")
	}
}

// CHECK-LABEL: define void @main.main(){{.*}} {
func main() {
	check(records[:])
	var stack [2]local.Record
	check(stack[:])
	items := make([]local.Record, 0, 1)
	for i := 0; i < 17; i++ {
		items = append(items, local.Record{})
	}
	check(items[:2])
	println("explicit alignment ok")
}
