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

func TestQueueRemovePreservesFIFO(t *testing.T) {
	for remove := 0; remove < 3; remove++ {
		nodes := [3]*testNode{{value: 0}, {value: 1}, {value: 2}}
		var q Queue[*testNode]
		for _, node := range nodes {
			q.Push(node)
		}
		if q.Remove(nodes[remove]) != nodes[remove] || q.Len() != 2 {
			t.Fatalf("Remove(%d) did not detach the node", remove)
		}
		if q.Remove(nodes[remove]) != nil || !q.Push(nodes[remove]) {
			t.Fatalf("removed node %d could not be reused", remove)
		}
		for i, node := range nodes {
			if i != remove && q.Pop() != node {
				t.Fatalf("Remove(%d) changed FIFO order", remove)
			}
		}
		if q.Pop() != nodes[remove] || q.Pop() != nil || q.Len() != 0 {
			t.Fatalf("Remove(%d) broke the tail or length", remove)
		}
	}
	var q, other Queue[*testNode]
	node := &testNode{}
	other.Push(node)
	if q.Remove(nil) != nil || q.Remove(node) != nil || other.Pop() != node {
		t.Fatal("Remove accepted a node outside the queue")
	}
}

func TestQueueRemoveAfterCursor(t *testing.T) {
	first, second, third := &testNode{}, &testNode{}, &testNode{}
	var q Queue[*testNode]
	q.Push(first)
	q.Push(second)
	q.Push(third)
	if q.RemoveAfter(third) != nil || q.Len() != 3 {
		t.Fatal("removal after the tail changed the queue")
	}
	if q.RemoveAfter(first) != second || !q.Push(second) || q.RemoveAfter(first) != third {
		t.Fatal("cursor removal broke node reuse or FIFO order")
	}
	if q.RemoveAfter(nil) != first || q.RemoveAfter(nil) != second || q.RemoveAfter(nil) != nil || q.Len() != 0 {
		t.Fatal("front removal broke the head, tail or length")
	}
}
