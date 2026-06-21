package gotest

import (
	"strconv"
	"testing"
)

func TestClosedChannelReceiveYieldsZeroAndOKFalse(t *testing.T) {
	ch := make(chan int, 1)
	ch <- 7
	close(ch)

	if got, ok := <-ch; got != 7 || !ok {
		t.Fatalf("buffered receive from closed channel = %d, %v, want 7, true", got, ok)
	}
	if got, ok := <-ch; got != 0 || ok {
		t.Fatalf("empty receive from closed channel = %d, %v, want 0, false", got, ok)
	}

	got := 99
	select {
	case got = <-ch:
	default:
		t.Fatal("receive from closed channel did not select")
	}
	if got != 0 {
		t.Fatalf("select receive from closed channel stored %d, want 0", got)
	}
}

func TestSendOnClosedChannelPanics(t *testing.T) {
	ch := make(chan int)
	close(ch)

	expectChannelPanicContaining(t, "send on closed channel", func() {
		ch <- 1
	})
}

func TestSelectSendOnClosedChannelPanics(t *testing.T) {
	ch := make(chan int)
	close(ch)

	expectChannelPanicContaining(t, "send on closed channel", func() {
		select {
		case ch <- 1:
			t.Fatal("send on closed channel selected without panic")
		default:
			t.Fatal("default selected for send on closed channel")
		}
	})

	expectChannelPanicContaining(t, "send on closed channel", func() {
		var never chan int
		select {
		case ch <- 1:
			t.Fatal("send on closed channel selected without panic")
		case <-never:
			t.Fatal("nil channel receive selected")
		}
	})
}

func TestCloseClosedChannelPanics(t *testing.T) {
	expectChannelPanicContaining(t, "close of nil channel", func() {
		var ch chan int
		close(ch)
	})

	ch := make(chan int)
	close(ch)
	expectChannelPanicContaining(t, "close of closed channel", func() {
		close(ch)
	})
}

func TestMakeChannelCapacityOutOfRangePanics(t *testing.T) {
	type intChan chan int

	n := -1
	expectChannelPanicContaining(t, "makechan: size out of range", func() {
		_ = make(intChan, n)
	})
	expectChannelPanicContaining(t, "makechan: size out of range", func() {
		_ = make(intChan, int64(n))
	})

	if strconv.IntSize == 64 {
		var n2 int64 = 1 << 59
		expectChannelPanicContaining(t, "makechan: size out of range", func() {
			_ = make(intChan, int(n2))
		})
		n2 = 1<<63 - 1
		expectChannelPanicContaining(t, "makechan: size out of range", func() {
			_ = make(intChan, int(n2))
		})
	} else {
		n = 1<<31 - 1
		expectChannelPanicContaining(t, "makechan: size out of range", func() {
			_ = make(intChan, n)
		})
		expectChannelPanicContaining(t, "makechan: size out of range", func() {
			_ = make(intChan, int64(n))
		})
	}
}

func TestUnbufferedRangeReceivesLastValueBeforeClose(t *testing.T) {
	for i := 0; i < 100; i++ {
		ch := make(chan int)
		go func(v int) {
			ch <- v
			close(ch)
		}(i)

		var got []int
		for v := range ch {
			got = append(got, v)
		}
		if len(got) != 1 || got[0] != i {
			t.Fatalf("iteration %d: range over send-then-close channel = %v, want [%d]", i, got, i)
		}
	}
}
