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

// Package wasmevent owns host-driven events for the single-worker WebAssembly
// scheduler. The queue is platform independent so timer semantics can be tested
// without a WebAssembly host.
package wasmevent

const maxInt64 = int64(^uint64(0) >> 1)

type Callback func(arg any, timer *Timer, scheduled, now int64)

type Timer struct {
	queue    *queue
	callback Callback
	arg      any
	when     int64
	period   int64
	sequence uint64
	index    int
	active   bool
}

func (t *Timer) Deadline() (int64, bool) {
	return t.when, t.active
}

type queue struct {
	timers       []*Timer
	nextSequence uint64
}

func (q *queue) len() int {
	return len(q.timers)
}

func (q *queue) deadline() (int64, bool) {
	if len(q.timers) == 0 {
		return 0, false
	}
	return q.timers[0].when, true
}

// Reset schedules timer and reports whether it was active before the reset.
func (q *queue) reset(timer *Timer, when, period int64, callback Callback, arg any) bool {
	if timer == nil {
		return false
	}
	wasActive := timer.active
	if timer.active {
		timer.queue.remove(timer.index)
	}
	timer.queue = q
	timer.callback = callback
	timer.arg = arg
	timer.when = when
	timer.period = period
	timer.sequence = q.nextSequence
	q.nextSequence++
	timer.index = len(q.timers)
	timer.active = true
	q.timers = append(q.timers, timer)
	q.siftUp(timer.index)
	return wasActive
}

// Stop removes timer and reports whether it was active.
func (q *queue) stop(timer *Timer) bool {
	if timer == nil || !timer.active || timer.queue != q {
		return false
	}
	q.remove(timer.index)
	timer.callback = nil
	timer.arg = nil
	return true
}

// RunDue runs each timer due at now once. Periodic timers skip missed periods
// and are reinserted before their callback, so a callback can Stop or Reset
// itself with the same semantics as any other active timer.
func (q *queue) runDue(now int64) int {
	ran := 0
	for len(q.timers) != 0 {
		timer := q.timers[0]
		if timer.when > now {
			break
		}

		scheduled := timer.when
		period := timer.period
		callback := timer.callback
		arg := timer.arg
		q.remove(0)

		if period > 0 {
			next := nextPeriodicDeadline(scheduled, period, now)
			q.reset(timer, next, period, callback, arg)
		}
		if callback != nil {
			callback(arg, timer, scheduled, now)
		}
		if !timer.active {
			timer.callback = nil
			timer.arg = nil
		}
		ran++
	}
	return ran
}

func waitForEvent(q *queue, now func() int64, wait func(uint64)) bool {
	for {
		current := now()
		if q.runDue(current) != 0 {
			return true
		}
		deadline, ok := q.deadline()
		if !ok {
			return false
		}
		if deadline > current {
			wait(uint64(deadline - current))
		}
	}
}

func nextPeriodicDeadline(when, period, now int64) int64 {
	if when > maxInt64-period {
		return maxInt64
	}
	next := when + period
	if next > now {
		return next
	}
	steps := (now-when)/period + 1
	if steps > (maxInt64-when)/period {
		return maxInt64
	}
	return when + steps*period
}

func (q *queue) less(i, j int) bool {
	left, right := q.timers[i], q.timers[j]
	if left.when != right.when {
		return left.when < right.when
	}
	return left.sequence < right.sequence
}

func (q *queue) swap(i, j int) {
	q.timers[i], q.timers[j] = q.timers[j], q.timers[i]
	q.timers[i].index = i
	q.timers[j].index = j
}

func (q *queue) siftUp(index int) {
	for index > 0 {
		parent := (index - 1) / 2
		if !q.less(index, parent) {
			return
		}
		q.swap(index, parent)
		index = parent
	}
}

func (q *queue) siftDown(index int) {
	for {
		left := index*2 + 1
		if left >= len(q.timers) {
			return
		}
		child := left
		if right := left + 1; right < len(q.timers) && q.less(right, left) {
			child = right
		}
		if !q.less(child, index) {
			return
		}
		q.swap(index, child)
		index = child
	}
}

func (q *queue) remove(index int) {
	timer := q.timers[index]
	last := len(q.timers) - 1
	if index != last {
		q.timers[index] = q.timers[last]
		q.timers[index].index = index
	}
	q.timers[last] = nil
	q.timers = q.timers[:last]
	if index < len(q.timers) {
		parent := (index - 1) / 2
		if index > 0 && q.less(index, parent) {
			q.siftUp(index)
		} else {
			q.siftDown(index)
		}
	}
	timer.queue = nil
	timer.index = -1
	timer.active = false
}
