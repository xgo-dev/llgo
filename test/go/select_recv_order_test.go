package gotest

import "testing"

var selectRecvOrderTarget int

func TestSingleCaseSelectReceiveAssignOrder(t *testing.T) {
	order := make([]int, 0, 2)
	c := make(chan int, 1)
	selectRecvOrderTarget = 0

	chanArg := func(ch chan int, marker int) chan int {
		order = append(order, marker)
		return ch
	}
	ptrArg := func(p *int, marker int) *int {
		order = append(order, marker)
		return p
	}

	c <- 42
	select {
	case *ptrArg(&selectRecvOrderTarget, 100) = <-chanArg(c, 1):
	}

	if selectRecvOrderTarget != 42 {
		t.Fatalf("select receive assignment stored %d, want 42", selectRecvOrderTarget)
	}
	if len(order) != 2 || order[0] != 1 || order[1] != 100 {
		t.Fatalf("single-case select receive assignment order = %v, want [1 100]", order)
	}
}
