package test

import (
	"testing"
	"time"
)

func TestSelectRecvWakesForBlockedUnbufferedSend(t *testing.T) {
	for i := 0; i < 50; i++ {
		res := make(chan struct{})
		done := make(chan struct{})
		started := make(chan struct{})
		received := make(chan struct{})
		sendDone := make(chan struct{})

		go func() {
			close(started)
			select {
			case <-res:
				close(received)
			case <-done:
			}
		}()

		<-started
		// Give the goroutine a brief chance to reach the blocking select path so
		// this regression exercises select wakeups rather than a direct recv fast path.
		time.Sleep(time.Millisecond)

		go func() {
			select {
			case res <- struct{}{}:
				close(sendDone)
			case <-done:
			}
		}()

		select {
		case <-sendDone:
		case <-time.After(200 * time.Millisecond):
			close(done)
			t.Fatalf("iteration %d: unbuffered send did not wake select receiver", i)
		}

		select {
		case <-received:
		case <-time.After(200 * time.Millisecond):
			close(done)
			t.Fatalf("iteration %d: select receiver did not receive sent value", i)
		}

		close(done)
	}
}

func TestSelectMixedUnbufferedPeersMakeProgress(t *testing.T) {
	for i := 0; i < 50; i++ {
		a := make(chan struct{})
		b := make(chan struct{})
		done := make(chan struct{}, 2)

		go func() {
			select {
			case <-a:
			case b <- struct{}{}:
			}
			done <- struct{}{}
		}()
		go func() {
			select {
			case <-b:
			case a <- struct{}{}:
			}
			done <- struct{}{}
		}()

		for j := 0; j < 2; j++ {
			select {
			case <-done:
			case <-time.After(200 * time.Millisecond):
				t.Fatalf("iteration %d: mixed unbuffered select peers did not make progress", i)
			}
		}
	}
}
