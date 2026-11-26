// Behavior regression fixture: runs on hosted and on the RV32 baremetal
// scheduler; the golden output must stay identical on both.

package main

import "container/heap"

type intHeap []int

func (h intHeap) Len() int           { return len(h) }
func (h intHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h intHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *intHeap) Push(x any)        { *h = append(*h, x.(int)) }
func (h *intHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

func main() {
	h := &intHeap{5, 3, 8, 1}
	heap.Init(h)
	println("min", (*h)[0])
	heap.Push(h, 2)
	println("min2", (*h)[0])
	for h.Len() > 0 {
		println("pop", heap.Pop(h).(int))
	}
}
