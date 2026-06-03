package gotest

import "testing"

func TestRangeUnbufferedChanReceivesFinalValueBeforeClose(t *testing.T) {
	const n = 16
	for round := 0; round < 20; round++ {
		c := make(chan int)
		go func() {
			for i := 0; i < n; i++ {
				c <- i
			}
			close(c)
		}()

		want := 0
		for got := range c {
			if got != want {
				t.Fatalf("round %d: range value %d = %d, want %d", round, want, got, want)
			}
			want++
		}
		if want != n {
			t.Fatalf("round %d: range received %d values, want %d", round, want, n)
		}
	}
}

func TestUnbufferedChanReceiveOKAfterSenderCloses(t *testing.T) {
	for round := 0; round < 20; round++ {
		c := make(chan int)
		go func() {
			c <- 99
			close(c)
		}()

		got, ok := <-c
		if !ok || got != 99 {
			t.Fatalf("round %d: receive = %d, %v; want 99, true", round, got, ok)
		}
		got, ok = <-c
		if ok || got != 0 {
			t.Fatalf("round %d: closed receive = %d, %v; want 0, false", round, got, ok)
		}
	}
}
