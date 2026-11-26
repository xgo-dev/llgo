// Behavior regression fixture: runs on hosted and on the RV32 baremetal
// scheduler; the golden output must stay identical on both.

package main

import "container/list"

func main() {
	l := list.New()
	l.PushBack(1)
	l.PushBack(2)
	l.PushFront(0)
	for e := l.Front(); e != nil; e = e.Next() {
		println("elem", e.Value.(int))
	}
	println("len", l.Len())
	l.Remove(l.Front())
	println("after", l.Front().Value.(int), l.Len())
	l.Init()
	println("init", l.Len())
}
