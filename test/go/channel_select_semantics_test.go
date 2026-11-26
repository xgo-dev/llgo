package gotest

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestNilChannelSendRecvBlock(t *testing.T) {
	done := make(chan string, 2)
	var ch chan int

	go func() {
		ch <- 1
		done <- "send"
	}()
	go func() {
		<-ch
		done <- "recv"
	}()

	time.Sleep(20 * time.Millisecond)
	select {
	case op := <-done:
		t.Fatalf("nil channel %s completed, want block", op)
	default:
	}
}

func TestNilChannelSelectCasesUseDefault(t *testing.T) {
	var ch chan int

	select {
	case ch <- 1:
		t.Fatal("nil channel send case selected")
	case <-ch:
		t.Fatal("nil channel receive case selected")
	default:
	}
}

func TestClosedChannelSendPanics(t *testing.T) {
	ch := make(chan int)
	close(ch)

	expectChannelPanicContaining(t, "send on closed channel", func() {
		ch <- 1
	})
}

func TestClosedChannelSelectSendPanics(t *testing.T) {
	ch := make(chan int)
	close(ch)

	expectChannelPanicContaining(t, "send on closed channel", func() {
		select {
		case ch <- 1:
		default:
			t.Fatal("closed channel send selected default")
		}
	})
}

func TestClosedChannelSelectRecvReturnsZero(t *testing.T) {
	ch := make(chan int)
	close(ch)

	select {
	case v, ok := <-ch:
		if v != 0 || ok {
			t.Fatalf("closed channel receive = %d, %v; want 0, false", v, ok)
		}
	default:
		t.Fatal("closed channel receive did not select")
	}
}

func TestHighFrequencySelectRecvHandshake(t *testing.T) {
	const n = 100000
	ch := make(chan int)
	dummy := make(chan int)
	done := make(chan struct{})

	go func() {
		for i := 0; i < n; i++ {
			ch <- i
		}
		close(done)
	}()

	for i := 0; i < n; i++ {
		select {
		case v := <-ch:
			if v != i {
				t.Fatalf("receive %d = %d", i, v)
			}
		case <-dummy:
			t.Fatal("dummy channel selected")
		}
	}

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("sender did not finish")
	}
}

func TestSelectMoreThanInlineCases(t *testing.T) {
	// Exercise blocking select with more than the 8 inline cached waiters,
	// covering both inline and heap-allocated waiters plus close/panic reuse.
	ch := make([]chan int, 12)
	for i := range ch {
		ch[i] = make(chan int)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for round := 0; round < 20; round++ {
			idx := round % len(ch)
			ch[idx] <- round + 1
		}
		close(ch[11])
	}()

	for round := 0; round < 20; round++ {
		var got int
		select {
		case got = <-ch[0]:
		case got = <-ch[1]:
		case got = <-ch[2]:
		case got = <-ch[3]:
		case got = <-ch[4]:
		case got = <-ch[5]:
		case got = <-ch[6]:
		case got = <-ch[7]:
		case got = <-ch[8]:
		case got = <-ch[9]:
		case got = <-ch[10]:
		case got = <-ch[11]:
		}
		if got != round+1 {
			t.Fatalf("round %d: got %d, want %d", round, got, round+1)
		}
	}

	// Verify heap-allocated waiter slot (index 11) observes close cleanly.
	select {
	case v, ok := <-ch[11]:
		if v != 0 || ok {
			t.Fatalf("closed ch[11] = (%d, %v), want (0, false)", v, ok)
		}
	case <-ch[0]:
		t.Fatal("unexpected recv on ch[0]")
	}
	<-done

	// Verify panic-on-send-closed releases the cached select state cleanly for reuse.
	expectChannelPanicContaining(t, "send on closed channel", func() {
		select {
		case ch[11] <- 1:
		case <-ch[0]:
		}
	})
	go func() { ch[10] <- 99 }()
	select {
	case v := <-ch[10]:
		if v != 99 {
			t.Fatalf("post-panic select got %d, want 99", v)
		}
	case <-ch[0]:
		t.Fatal("unexpected recv on ch[0]")
	}
}

func TestChannelCloseWakesManyWaiters(t *testing.T) {
	const waiters = 24
	for round := 0; round < 10; round++ {
		recvCh := make(chan int)
		sendCh := make(chan int)
		idle := make(chan int)
		ready := make(chan struct{}, waiters*3)
		done := make(chan error, waiters*3)

		for i := 0; i < waiters; i++ {
			go func() {
				ready <- struct{}{}
				v, ok := <-recvCh
				if v != 0 || ok {
					done <- fmt.Errorf("recv got (%d, %v), want (0, false)", v, ok)
					return
				}
				done <- nil
			}()
			go func() {
				ready <- struct{}{}
				select {
				case v, ok := <-recvCh:
					if v != 0 || ok {
						done <- fmt.Errorf("select recv got (%d, %v), want (0, false)", v, ok)
						return
					}
					done <- nil
				case <-idle:
					done <- fmt.Errorf("unexpected idle recv")
				}
			}()
			go func() {
				ready <- struct{}{}
				defer func() {
					if r := recover(); r == nil || !strings.Contains(fmt.Sprint(r), "send on closed channel") {
						done <- fmt.Errorf("send panic = %v, want send on closed channel", r)
						return
					}
					done <- nil
				}()
				sendCh <- 1
			}()
		}

		for i := 0; i < waiters*3; i++ {
			<-ready
		}
		time.Sleep(time.Millisecond)
		close(recvCh)
		close(sendCh)
		for i := 0; i < waiters*3; i++ {
			select {
			case err := <-done:
				if err != nil {
					t.Fatalf("round %d: %v", round, err)
				}
			case <-time.After(10 * time.Second):
				t.Fatalf("round %d: timed out waiting for waiter %d", round, i)
			}
		}
	}
}

func expectChannelPanicContaining(t *testing.T, want string, f func()) {
	t.Helper()
	defer func() {
		err := recover()
		if err == nil {
			t.Fatalf("expected panic containing %q", want)
		}
		if got := fmt.Sprint(err); !strings.Contains(got, want) {
			t.Fatalf("panic = %q, want contains %q", got, want)
		}
	}()
	f()
}
