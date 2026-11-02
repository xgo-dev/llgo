package gotest

import (
	"reflect"
	"runtime"
	"strconv"
	"testing"
)

func TestChannelPollReadiness(t *testing.T) {
	for _, capacity := range []int{0, 1, 4} {
		ch := make(chan int, capacity)
		poll := func(want int, wantOK, wantSelected bool) {
			t.Helper()
			select {
			case got, ok := <-ch:
				if !wantSelected || got != want || ok != wantOK {
					t.Fatalf("capacity %d: receive = %d, %v", capacity, got, ok)
				}
			default:
				if wantSelected {
					t.Fatalf("capacity %d: unexpected default", capacity)
				}
			}
		}
		poll(0, false, false)
		for round := 0; round < 3; round++ {
			for i := 1; i <= capacity; i++ {
				ch <- i
			}
			for i := 1; i <= capacity; i++ {
				poll(i, true, true)
			}
			poll(0, false, false)
		}
		for i := 1; i <= capacity; i++ {
			ch <- i
		}
		close(ch)
		for i := 1; i <= capacity; i++ {
			poll(i, true, true)
		}
		poll(0, false, true)
		poll(0, false, true)
	}
}

func TestChannelPollConcurrentSendClose(t *testing.T) {
	// Both ordinary and select waiters must publish readiness. The final
	// receive also observes a write before close without another sync edge.
	for _, useSelect := range []bool{false, true} {
		for _, capacity := range []int{0, 1, 4} {
			ch := make(chan int, capacity)
			var published int
			go func() {
				for i := 1; i <= 1000; i++ {
					if useSelect {
						select {
						case ch <- i:
						case <-make(chan struct{}):
						}
					} else {
						ch <- i
					}
				}
				published = 42
				close(ch)
			}()
			next := 1
		loop:
			for {
				select {
				case v, ok := <-ch:
					if !ok {
						break loop
					}
					if v != next {
						t.Fatalf("got %d, want %d", v, next)
					}
					next++
				default:
					runtime.Gosched()
				}
			}
			if next != 1001 || published != 42 {
				t.Fatal("lost send or close synchronization")
			}
		}
	}
}

func TestChannelReflectTryRecvReadiness(t *testing.T) {
	ch := make(chan int, 1)
	v := reflect.ValueOf(ch)
	if got, ok := v.TryRecv(); got.IsValid() || ok {
		t.Fatal("empty receive selected")
	}
	ch <- 42
	if got, ok := v.TryRecv(); !ok || got.Int() != 42 {
		t.Fatal("buffered receive missed")
	}
	if got, ok := v.TryRecv(); got.IsValid() || ok {
		t.Fatal("drained receive selected")
	}
	close(ch)
	if got, ok := v.TryRecv(); !got.IsValid() || ok || got.Int() != 0 {
		t.Fatal("closed receive missed")
	}
}

func TestChannelPollAfterCloseWake(t *testing.T) {
	// A receiver may wake before close finishes notifying the other waiters.
	// Having observed close, it must never select default on the same channel.
	for round := 0; round < 8; round++ {
		ch := make(chan int)
		done := make(chan bool, 16)
		for i := 0; i < 16; i++ {
			go func() {
				<-ch
				select {
				case _, ok := <-ch:
					done <- !ok
				default:
					done <- false
				}
			}()
		}
		runtime.Gosched()
		close(ch)
		for i := 0; i < 16; i++ {
			if !<-done {
				t.Fatal("close was observed then missed")
			}
		}
	}
}

func BenchmarkChannelPollEmpty(b *testing.B) {
	ch := make(chan struct{})
	for i := 0; i < b.N; i++ {
		select {
		case <-ch:
			b.Fatal("unexpected receive")
		default:
		}
	}
}

func BenchmarkChannelBufferedRoundTrip(b *testing.B) {
	for _, capacity := range []int{1, 64, 1024} {
		b.Run(strconv.Itoa(capacity), func(b *testing.B) {
			ch := make(chan int, capacity)
			for i := 0; i < b.N; i++ {
				ch <- i
				if got := <-ch; got != i {
					b.Fatal("incorrect value", got)
				}
			}
		})
	}
}

func BenchmarkChannelSPSC(b *testing.B) {
	for _, capacity := range []int{1, 64, 1024} {
		b.Run(strconv.Itoa(capacity), func(b *testing.B) {
			ch := make(chan int, capacity)
			done := make(chan struct{})
			go func() {
				for i := 0; i < b.N; i++ {
					ch <- i
				}
				close(done)
			}()
			for i := 0; i < b.N; i++ {
				if got := <-ch; got != i {
					b.Fatal("incorrect value", got)
				}
			}
			<-done
		})
	}
}
