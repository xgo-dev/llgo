package gotest

import (
	"runtime"
	"sync"
	"testing"
)

func TestChannelWaiterReuseAcrossChannels(t *testing.T) {
	type message struct {
		value int
		data  []byte
	}
	var workers sync.WaitGroup
	for worker := 0; worker < 4; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for capacity := 0; capacity <= 1; capacity++ {
				jobs := make(chan *message, capacity)
				replies := make(chan *message, capacity)
				done := make(chan struct{})
				go func() {
					defer close(done)
					for m := range jobs {
						replies <- m
					}
					close(replies)
				}()
				for i := 0; i < 256; i++ {
					m := &message{value: i, data: []byte{byte(i), byte(i >> 8)}}
					jobs <- m
					if i%64 == 0 {
						runtime.GC()
					}
					if got := <-replies; got != m || got.value != i || got.data[0] != byte(i) {
						t.Errorf("channel lost pointer-bearing value %d", i)
					}
				}
				close(jobs)
				if got, ok := <-replies; got != nil || ok {
					t.Error("closed channel did not return its zero value")
				}
				<-done
			}
		}()
	}
	workers.Wait()
}
