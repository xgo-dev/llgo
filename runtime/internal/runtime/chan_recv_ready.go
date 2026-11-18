//go:build !baremetal

package runtime

import "github.com/xgo-dev/llgo/runtime/internal/sync/atomic"

func (p *Chan) recvMayBeReady() bool {
	return atomic.Load(&p.recvReady) != 0
}

func (p *Chan) publishRecvReady() {
	var ready uint32
	if p.closed || p.qcount != 0 || p.sendq.first != nil {
		ready = 1
	}
	// The mutex serializes publishers. Avoid a write when readiness has not
	// changed; receivers lock before accessing values or closed state.
	if atomic.Load(&p.recvReady) != ready {
		atomic.Store(&p.recvReady, ready)
	}
}

func (p *Chan) publishRecvClosed() {
	atomic.Store(&p.recvReady, 1)
}
