package gotest

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
)

type channelHandoffMessage struct {
	client, sequence int
}

type channelHandoffPhase struct {
	count int
	start chan struct{}
}

// Keep clients and workers alive between phases so neither thread startup nor
// channel construction is part of the measured handoff path (issue #3116).
type channelHandoffPool struct {
	jobs             chan channelHandoffMessage
	gates            []chan *channelHandoffPhase
	ready, finished  chan int
	workers, clients sync.WaitGroup
}

func newChannelHandoffPool(clients, capacity int) *channelHandoffPool {
	p := &channelHandoffPool{
		jobs:  make(chan channelHandoffMessage, capacity),
		gates: make([]chan *channelHandoffPhase, clients),
		ready: make(chan int, clients), finished: make(chan int, clients),
	}
	replies := make([]chan channelHandoffMessage, clients)
	for id := 0; id < clients; id++ {
		p.gates[id] = make(chan *channelHandoffPhase)
		replies[id] = make(chan channelHandoffMessage, 1)
	}
	for i := 0; i < 4; i++ {
		p.workers.Add(1)
		go func() {
			defer p.workers.Done()
			for msg := range p.jobs {
				replies[msg.client] <- msg
			}
		}()
	}
	for id := 0; id < clients; id++ {
		id := id
		p.clients.Add(1)
		go func() {
			defer p.clients.Done()
			for phase := range p.gates[id] {
				p.ready <- id
				<-phase.start
				count := phase.count / clients
				if id < phase.count%clients {
					count++
				}
				for sequence := 0; sequence < count; sequence++ {
					msg := channelHandoffMessage{id, sequence}
					p.jobs <- msg
					if got := <-replies[id]; got != msg {
						panic("channel handoff lost or misrouted a reply")
					}
				}
				p.finished <- id
			}
		}()
	}
	return p
}

func (p *channelHandoffPool) run(count int) {
	phase := &channelHandoffPhase{count: count, start: make(chan struct{})}
	for _, gate := range p.gates {
		gate <- phase
	}
	for range p.gates {
		<-p.ready
	}
	close(phase.start)
	for range p.gates {
		<-p.finished
	}
}

func (p *channelHandoffPool) close() {
	for _, gate := range p.gates {
		close(gate)
	}
	p.clients.Wait()
	close(p.jobs)
	p.workers.Wait()
}

func TestChannelContendedHandoff(t *testing.T) {
	for _, clients := range []int{1, 16, 64} {
		for _, capacity := range []int{0, 4} {
			t.Run(fmt.Sprintf("clients=%d/capacity=%d", clients, capacity), func(t *testing.T) {
				p := newChannelHandoffPool(clients, capacity)
				defer p.close()
				for _, count := range []int{0, clients * 16, clients*128 + 1, 0} {
					p.run(count)
					runtime.GC()
				}
			})
		}
	}
}

func BenchmarkChannelContendedHandoff(b *testing.B) {
	p := newChannelHandoffPool(16, 4)
	defer p.close()
	p.run(4096)
	runtime.GC()
	b.ResetTimer()
	p.run(b.N)
	b.StopTimer()
}
