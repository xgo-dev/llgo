package pollbudget

import "testing"

func TestBudget(t *testing.T) {
	budget := New(3)
	if budget.Poll() {
		t.Fatal("first poll reached the slow path")
	}
	if budget.Poll() {
		t.Fatal("second poll reached the slow path")
	}
	if !budget.Poll() {
		t.Fatal("third poll did not reach the slow path")
	}
	if budget.Poll() {
		t.Fatal("budget did not reset")
	}
}

func TestZeroQuantum(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("New(0) did not panic")
		}
	}()
	New(0)
}

func BenchmarkBudgetPoll(b *testing.B) {
	budget := New(1024)
	for b.Loop() {
		budget.Poll()
	}
}
