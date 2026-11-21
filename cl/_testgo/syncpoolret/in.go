// Behavior regression fixture: sync.Pool keeps a Put object for the next Get
// instead of falling back to New — the baremetal pool backend must retain
// pool locals the way the hosted one does. The golden output is
// scheduling-order independent on every backend.
package main

import "sync"

type payload struct {
	tag int
}

func main() {
	var pool sync.Pool
	pool.New = func() any { return &payload{tag: -1} }

	kept := &payload{tag: 42}
	pool.Put(kept)
	got := pool.Get().(*payload)
	if got == kept {
		println("pool kept", got.tag)
		return
	}
	println("pool substituted", got.tag)
}
