package wasmevent

import (
	"reflect"
	"testing"
)

func appendEvent(arg any, _ *Timer, scheduled, now int64) {
	events := arg.(*[]int64)
	*events = append(*events, scheduled, now)
}

type orderedEvent struct {
	id     int64
	events *[]int64
}

func appendOrderedEvent(arg any, _ *Timer, _, _ int64) {
	event := arg.(orderedEvent)
	*event.events = append(*event.events, event.id)
}

func TestQueueOrdersDeadlinesAndTies(t *testing.T) {
	var q queue
	var events []int64
	var first, second, third Timer
	q.reset(&first, 30, 0, appendOrderedEvent, orderedEvent{1, &events})
	q.reset(&second, 10, 0, appendOrderedEvent, orderedEvent{2, &events})
	q.reset(&third, 10, 0, appendOrderedEvent, orderedEvent{3, &events})

	if deadline, ok := q.deadline(); !ok || deadline != 10 {
		t.Fatalf("Deadline = %d, %v; want 10, true", deadline, ok)
	}
	if got := q.runDue(9); got != 0 {
		t.Fatalf("RunDue(9) = %d, want 0", got)
	}
	if got := q.runDue(10); got != 2 {
		t.Fatalf("RunDue(10) = %d, want 2", got)
	}
	if want := []int64{2, 3}; !reflect.DeepEqual(events, want) {
		t.Fatalf("events = %v, want %v", events, want)
	}
	if got := q.runDue(30); got != 1 || q.len() != 0 {
		t.Fatalf("runDue(30) = %d, Len = %d; want 1, 0", got, q.len())
	}
}

func TestQueueResetAndStop(t *testing.T) {
	var q queue
	var timer Timer
	if active := q.reset(&timer, 30, 0, nil, nil); active {
		t.Fatal("initial Reset reported an active timer")
	}
	if active := q.reset(&timer, 20, 0, nil, nil); !active {
		t.Fatal("second Reset reported an inactive timer")
	}
	if deadline, ok := timer.Deadline(); !ok || deadline != 20 {
		t.Fatalf("timer deadline = %d, %v; want 20, true", deadline, ok)
	}
	if !q.stop(&timer) {
		t.Fatal("Stop reported an inactive timer")
	}
	if q.stop(&timer) {
		t.Fatal("second Stop reported an active timer")
	}
	if _, ok := q.deadline(); ok {
		t.Fatal("stopped timer remained in the queue")
	}
}

func TestQueuePeriodicTimerSkipsMissedPeriods(t *testing.T) {
	var q queue
	var timer Timer
	var events []int64
	q.reset(&timer, 10, 10, appendEvent, &events)

	if got := q.runDue(35); got != 1 {
		t.Fatalf("RunDue(35) = %d, want 1", got)
	}
	if want := []int64{10, 35}; !reflect.DeepEqual(events, want) {
		t.Fatalf("events = %v, want %v", events, want)
	}
	if deadline, ok := timer.Deadline(); !ok || deadline != 40 {
		t.Fatalf("timer deadline = %d, %v; want 40, true", deadline, ok)
	}
}

func TestQueueCallbackCanResetAndStop(t *testing.T) {
	var q queue
	var resetTimer, stoppedTimer Timer
	resetCount := 0
	q.reset(&resetTimer, 5, 0, func(_ any, timer *Timer, _, _ int64) {
		resetCount++
		q.reset(timer, 20, 0, func(any, *Timer, int64, int64) {
			resetCount++
		}, nil)
	}, nil)
	q.reset(&stoppedTimer, 5, 5, func(_ any, timer *Timer, _, _ int64) {
		q.stop(timer)
	}, nil)

	if got := q.runDue(5); got != 2 {
		t.Fatalf("RunDue(5) = %d, want 2", got)
	}
	if q.len() != 1 {
		t.Fatalf("Len = %d, want 1", q.len())
	}
	if got := q.runDue(20); got != 1 || resetCount != 2 {
		t.Fatalf("RunDue(20) = %d, reset callbacks = %d; want 1, 2", got, resetCount)
	}
}

func TestWaitForEventRechecksAfterEarlyWake(t *testing.T) {
	var q queue
	var timer Timer
	now := int64(0)
	waits := 0
	q.reset(&timer, 10, 0, func(any, *Timer, int64, int64) {}, nil)

	if !waitForEvent(&q, func() int64 { return now }, func(delay uint64) {
		waits++
		if delay != uint64(10-now) {
			t.Fatalf("wait delay = %d, want %d", delay, 10-now)
		}
		now += 5
	}) {
		t.Fatal("waitForEvent reported no event")
	}
	if waits != 2 {
		t.Fatalf("host waits = %d, want 2", waits)
	}
	if waitForEvent(&q, func() int64 { return now }, func(uint64) {}) {
		t.Fatal("empty queue reported an event")
	}
}

func BenchmarkQueueResetAndRunDue(b *testing.B) {
	var q queue
	var timer Timer
	callback := func(any, *Timer, int64, int64) {}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		q.reset(&timer, int64(i), 0, callback, nil)
		q.runDue(int64(i))
	}
}
