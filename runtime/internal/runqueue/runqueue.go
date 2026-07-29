/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package runqueue provides an allocation-free intrusive FIFO for scheduler
// backends with one queue owner.
package runqueue

// Node is the intrusive link contract implemented by scheduler-owned values.
type Node[T comparable] interface {
	RunqueueNext() T
	SetRunqueueNext(T)
	RunqueueQueued() bool
	SetRunqueueQueued(bool)
}

type Queue[T interface {
	comparable
	Node[T]
}] struct {
	head T
	tail T
	size uintptr
}

// Push appends node and reports whether it was non-zero and not queued.
func (q *Queue[T]) Push(node T) bool {
	var zero T
	if node == zero || node.RunqueueQueued() {
		return false
	}
	node.SetRunqueueNext(zero)
	node.SetRunqueueQueued(true)
	if q.tail == zero {
		q.head = node
	} else {
		q.tail.SetRunqueueNext(node)
	}
	q.tail = node
	q.size++
	return true
}

// Pop removes and returns the oldest node, or its zero value when empty.
func (q *Queue[T]) Pop() T {
	var zero T
	node := q.head
	if node == zero {
		return zero
	}
	q.head = node.RunqueueNext()
	if q.head == zero {
		q.tail = zero
	}
	node.SetRunqueueNext(zero)
	node.SetRunqueueQueued(false)
	q.size--
	return node
}

func (q *Queue[T]) Len() uintptr {
	return q.size
}
