package runqueue

import "testing"

type testNode struct {
	value  int
	queued bool
	next   *testNode
}

func (node *testNode) RunqueueNext() *testNode {
	return node.next
}

func (node *testNode) SetRunqueueNext(next *testNode) {
	node.next = next
}

func (node *testNode) RunqueueQueued() bool {
	return node.queued
}

func (node *testNode) SetRunqueueQueued(queued bool) {
	node.queued = queued
}

func TestQueueFIFOAndReuse(t *testing.T) {
	first := &testNode{value: 1}
	second := &testNode{value: 2}
	var q Queue[*testNode]

	if !q.Push(first) || !q.Push(second) {
		t.Fatal("Push rejected initialized nodes")
	}
	if q.Push(first) {
		t.Fatal("Push accepted a queued node")
	}
	if got := q.Len(); got != 2 {
		t.Fatalf("Len = %d, want 2", got)
	}
	if got := q.Pop(); got != first || got.value != 1 {
		t.Fatalf("first Pop = %p, want %p", got, first)
	}
	if got := q.Pop(); got != second || got.value != 2 {
		t.Fatalf("second Pop = %p, want %p", got, second)
	}
	if got := q.Pop(); got != nil {
		t.Fatalf("empty Pop = %p, want nil", got)
	}
	if !q.Push(first) || q.Pop() != first {
		t.Fatal("queue did not accept a reused node")
	}
}

func TestQueueRejectsInvalidNodes(t *testing.T) {
	var q Queue[*testNode]
	if q.Push(nil) {
		t.Fatal("Push accepted nil")
	}
}
