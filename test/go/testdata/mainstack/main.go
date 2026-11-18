package main

import "runtime"

type node struct {
	previous *node
	payload  *uint64
	values   [64]uint64
}

//go:noinline
func walk(previous *node, depth uint64) uint64 {
	if depth == 0 {
		runtime.GC()
		var total uint64
		for p := previous; p != nil; p = p.previous {
			total += *p.payload
			for _, value := range p.values {
				total += value
			}
		}
		return total
	}
	var current node
	current.previous = previous
	current.payload = new(uint64)
	*current.payload = depth
	for i := range current.values {
		current.values[i] = depth + uint64(i)
	}
	return walk(&current, depth-1)
}

func main() {
	const depth = 4096
	const valueCount = uint64(len(node{}.values))
	// Each frame at depth d contributes (valueCount+1)*d + sum(0..valueCount-1).
	const want = (valueCount+1)*depth*(depth+1)/2 + depth*valueCount*(valueCount-1)/2
	if got := walk(nil, depth); got != want {
		panic("stack chain or heap payload lost across GC")
	}
	println("main-stack-ok")
}
