//go:build (darwin || linux) && !baremetal && go1.27 && !go1.28

/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package http

import (
	"crypto/tls"
	"io"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

// On LLGo's 1:1 OS-thread runtime, spawning a goroutine per HTTP request in
// connReader.startBackgroundRead and aborting it via SetReadDeadline + cond.Wait
// adds significant thread-handoff, lock, and syscall overhead to fast handlers.
//
// Replace per-request backgroundRead goroutines with a sharded fast path + a
// single shared kqueue/epoll watcher:
//   - Fast requests that complete before the watcher tick (~500us) disarm purely
//     in userspace with 0 goroutines, 0 syscalls, and 0 cond waits.
//   - Slow TCP/Unix requests are registered on the single shared kqueue/epoll
//     watcher and inspected via non-blocking MSG_PEEK inside rawConn.Control
//     (preventing fd reuse races even under concurrent Close), canceling
//     r.Context() promptly on peer EOF/RST while leaving pipelined bytes untouched.
//   - Non-socket connections (*tls.Conn, net.Pipe, custom net.Conn mocks) fall
//     back to standard go cr.backgroundRead() after the short pending grace window.
//
// Lock ordering invariant: ALWAYS acquire cr.mu BEFORE bgShard.mu.
// The watcher copies candidate entries out of bgShard.mu and releases shard.mu
// before acquiring cr.mu.

const (
	bgShardCount  = 16
	bgPollTickNs  = 500_000 // 500us poll interval while requests are in flight
	bgStateUnused = 0
	bgStatePend   = 1
	bgStatePoll   = 2
	bgStateFallbk = 3
	bgStateDisarm = 4
)

type bgWatchEntry struct {
	cr       *connReader
	rawConn  syscall.RawConn
	fd       int
	state    uint8
	seen     bool
	disarmed atomic.Bool
}

type bgWatchShard struct {
	mu      sync.Mutex
	entries map[*connReader]*bgWatchEntry
	_       [40]byte
}

var httpBgMgr struct {
	once          sync.Once
	ready         atomic.Bool
	poller        *httpBgPoller
	shards        [bgShardCount]bgWatchShard
	totalArmed    atomic.Int64
	pendingCount  atomic.Int64
	polledDisarms atomic.Int64
	sleeping      atomic.Int32
}

var bgEntryPool = sync.Pool{
	New: func() any { return new(bgWatchEntry) },
}

func bgShardFor(cr *connReader) *bgWatchShard {
	h := uintptr(unsafe.Pointer(cr))
	return &httpBgMgr.shards[(h>>6)&(bgShardCount-1)]
}

func isNonPeekableErrno(err error) bool {
	return err == syscall.ENOTSOCK || err == syscall.EOPNOTSUPP || err == syscall.EINVAL
}

func ensureHTTPBgWatcher() bool {
	httpBgMgr.once.Do(func() {
		p, err := newHTTPBgPoller()
		if err != nil {
			return
		}
		for i := range httpBgMgr.shards {
			httpBgMgr.shards[i].entries = make(map[*connReader]*bgWatchEntry, 16)
		}
		httpBgMgr.poller = p
		httpBgMgr.ready.Store(true)
		go runHTTPBgWatcher(p)
	})
	return httpBgMgr.ready.Load()
}

// arm registers cr while the caller holds cr.mu.
func httpBgWatchArm(cr *connReader) bool {
	if !ensureHTTPBgWatcher() {
		return false
	}
	e := bgEntryPool.Get().(*bgWatchEntry)
	e.cr = cr
	e.rawConn = nil
	e.fd = -1
	e.state = bgStatePend
	e.seen = false
	e.disarmed.Store(false)

	var replacedState uint8
	sh := bgShardFor(cr)
	sh.mu.Lock()
	if prev, ok := sh.entries[cr]; ok {
		replacedState = prev.state
		prev.state = bgStateDisarm
		prev.disarmed.Store(true)
	}
	sh.entries[cr] = e
	sh.mu.Unlock()

	if replacedState == bgStateUnused {
		httpBgMgr.totalArmed.Add(1)
	}
	if replacedState != bgStatePend {
		httpBgMgr.pendingCount.Add(1)
	}
	if httpBgMgr.sleeping.Load() != 0 {
		if httpBgMgr.sleeping.CompareAndSwap(1, 0) {
			httpBgMgr.poller.wake()
		}
	}
	return true
}

// disarm removes cr while the caller holds cr.mu. It returns true if the
// background read was completely cancelled without a fallback goroutine running.
func httpBgWatchDisarm(cr *connReader) bool {
	if !httpBgMgr.ready.Load() {
		return false
	}
	sh := bgShardFor(cr)
	sh.mu.Lock()
	e, ok := sh.entries[cr]
	if !ok {
		sh.mu.Unlock()
		return false
	}
	delete(sh.entries, cr)
	prevState := e.state
	rawConn := e.rawConn
	e.state = bgStateDisarm
	e.disarmed.Store(true)
	sh.mu.Unlock()

	httpBgMgr.totalArmed.Add(-1)
	if prevState == bgStatePend {
		httpBgMgr.pendingCount.Add(-1)
		e.cr = nil
		e.rawConn = nil
		e.fd = -1
		e.seen = false
		bgEntryPool.Put(e)
		return true
	}
	if prevState == bgStateFallbk {
		return false
	}
	if prevState == bgStatePoll {
		if rawConn != nil {
			_ = rawConn.Control(func(u uintptr) {
				httpBgMgr.poller.del(int(u))
			})
		}
		if httpBgMgr.polledDisarms.Add(1) >= 64 && httpBgMgr.sleeping.Load() != 0 {
			if httpBgMgr.sleeping.CompareAndSwap(1, 0) {
				httpBgMgr.poller.wake()
			}
		}
		// Do not recycle e here: runHTTPBgWatcher's byFD map may still hold e
		// until the next prune pass.
		return true
	}
	return true
}

func httpBgReleaseFallback(cr *connReader, e *bgWatchEntry) {
	sh := bgShardFor(cr)
	sh.mu.Lock()
	if sh.entries[cr] == e {
		delete(sh.entries, cr)
		e.state = bgStateDisarm
		e.disarmed.Store(true)
		sh.mu.Unlock()
		httpBgMgr.totalArmed.Add(-1)
		return
	}
	sh.mu.Unlock()
}

func httpBgStartFallback(cr *connReader, e *bgWatchEntry) {
	cr.rwc.SetReadDeadline(time.Time{})
	go func() {
		cr.backgroundRead()
		httpBgReleaseFallback(cr, e)
	}()
}

// peekAndArmSocket checks whether rawConn has unread data or has reached
// EOF/error without consuming any pipelined request byte, and if EAGAIN,
// registers the fd on p while still holding internal/poll's fdmu inside
// rawConn.Control. This prevents fd-close/reuse races even under concurrent
// Server.Close(). Non-stream descriptors (ENOTSOCK/EOPNOTSUPP/EINVAL) return
// closed=false, armed=false so the caller falls back to cr.backgroundRead().
func peekAndArmSocket(rawConn syscall.RawConn, p *httpBgPoller) (fd int, hasData bool, closed bool, armed bool, err error) {
	fd = -1
	var buf [1]byte
	ctrlErr := rawConn.Control(func(u uintptr) {
		fd = int(u)
		for {
			n, _, rerr := syscall.Recvfrom(fd, buf[:], syscall.MSG_PEEK|syscall.MSG_DONTWAIT)
			if rerr == syscall.EINTR {
				continue
			}
			if rerr == syscall.EAGAIN || rerr == syscall.EWOULDBLOCK {
				if p.add(fd) == nil {
					armed = true
				}
				return
			}
			if rerr != nil {
				if isNonPeekableErrno(rerr) {
					return
				}
				closed = true
				err = rerr
				return
			}
			if n == 0 {
				closed = true
				err = io.EOF
				return
			}
			hasData = true
			return
		}
	})
	if ctrlErr != nil {
		return fd, false, true, false, ctrlErr
	}
	return fd, hasData, closed, armed, err
}

func runHTTPBgWatcher(p *httpBgPoller) {
	byFD := make(map[int]*bgWatchEntry, 64)
	pending := make([]*connReader, 0, 64)
	readyFDs := make([]int, 0, 64)

	for {
		// Only run the 500us promotion tick while at least one entry is still
		// pending. Once slow handlers have been promoted to kqueue/epoll, block
		// in p.wait(-1) with zero timer wakeups or shard scans.
		var timeoutNs int64 = bgPollTickNs
		if httpBgMgr.pendingCount.Load() == 0 && httpBgMgr.polledDisarms.Load() == 0 {
			httpBgMgr.sleeping.Store(1)
			// Re-check after publishing sleeping=1 to prevent a missed wake.
			if httpBgMgr.pendingCount.Load() == 0 && httpBgMgr.polledDisarms.Load() == 0 {
				timeoutNs = -1
			} else {
				httpBgMgr.sleeping.Store(0)
			}
		}

		readyFDs = p.wait(timeoutNs, readyFDs[:0])
		httpBgMgr.sleeping.Store(0)

		// 1. Handle ready fds from kqueue/epoll.
		for _, fd := range readyFDs {
			e, ok := byFD[fd]
			if !ok {
				continue
			}
			delete(byFD, fd)
			handleReadyBgEntry(p, e, byFD)
		}

		// 2. Promote entries that survived at least one full tick (~500us) to
		// kqueue/epoll or fallback goroutine.
		if httpBgMgr.pendingCount.Load() > 0 {
			pending = pending[:0]
			for i := range httpBgMgr.shards {
				sh := &httpBgMgr.shards[i]
				sh.mu.Lock()
				for cr, e := range sh.entries {
					if e.state == bgStatePend {
						if !e.seen {
							e.seen = true
						} else {
							pending = append(pending, cr)
						}
					}
				}
				sh.mu.Unlock()
			}
			for _, cr := range pending {
				promotePendingBgEntry(p, cr, byFD)
			}
		}

		// 3. Prune any disarmed entries left in byFD.
		if len(byFD) > 0 && (httpBgMgr.polledDisarms.Swap(0) > 0 || httpBgMgr.totalArmed.Load() == 0) {
			for fd, e := range byFD {
				if e.disarmed.Load() {
					delete(byFD, fd)
				}
			}
		}
	}
}

func promotePendingBgEntry(p *httpBgPoller, cr *connReader, byFD map[int]*bgWatchEntry) {
	cr.lock()
	defer cr.unlock()

	sh := bgShardFor(cr)
	sh.mu.Lock()
	e, ok := sh.entries[cr]
	if !ok || e.state != bgStatePend || !cr.inRead {
		sh.mu.Unlock()
		return
	}
	sh.mu.Unlock()

	// Try raw socket non-blocking peek + kqueue/epoll watch for non-TLS sockets.
	var rawConn syscall.RawConn
	if _, isTLS := cr.rwc.(*tls.Conn); !isTLS {
		if sc, ok := cr.rwc.(syscall.Conn); ok {
			if rc, err := sc.SyscallConn(); err == nil {
				rawConn = rc
			}
		}
	}

	if rawConn != nil {
		fd, hasData, closed, armed, rerr := peekAndArmSocket(rawConn, p)
		if closed || hasData {
			sh.mu.Lock()
			if sh.entries[cr] == e && e.state == bgStatePend {
				delete(sh.entries, cr)
				e.state = bgStateDisarm
				e.disarmed.Store(true)
				sh.mu.Unlock()
				httpBgMgr.pendingCount.Add(-1)
				httpBgMgr.totalArmed.Add(-1)
				e.cr = nil
				e.rawConn = nil
				e.fd = -1
				e.seen = false
				bgEntryPool.Put(e)
			} else {
				sh.mu.Unlock()
				return
			}
			if closed {
				if rerr == nil {
					rerr = io.EOF
				}
				cr.handleReadErrorLocked(rerr)
			}
			// For pipelined data (hasData), leave bytes in kernel buffer for next readRequest.
			cr.aborted = false
			cr.inRead = false
			cr.cond.Broadcast()
			return
		}
		if armed && fd >= 0 {
			sh.mu.Lock()
			if sh.entries[cr] == e && e.state == bgStatePend {
				e.rawConn = rawConn
				e.fd = fd
				e.state = bgStatePoll
				byFD[fd] = e
				sh.mu.Unlock()
				httpBgMgr.pendingCount.Add(-1)
				return
			}
			sh.mu.Unlock()
			_ = rawConn.Control(func(u uintptr) {
				p.del(int(u))
			})
			return
		}
	}

	// Fallback for TLS, net.Pipe, non-stream syscall.Conn, or poller add error.
	sh.mu.Lock()
	if sh.entries[cr] != e || e.state != bgStatePend {
		sh.mu.Unlock()
		return
	}
	e.state = bgStateFallbk
	sh.mu.Unlock()
	httpBgMgr.pendingCount.Add(-1)

	httpBgStartFallback(cr, e)
}

func handleReadyBgEntry(p *httpBgPoller, e *bgWatchEntry, byFD map[int]*bgWatchEntry) {
	if e.disarmed.Load() {
		return
	}
	cr := e.cr
	if cr == nil {
		return
	}
	cr.lock()
	defer cr.unlock()

	sh := bgShardFor(cr)
	sh.mu.Lock()
	if sh.entries[cr] != e || e.state != bgStatePoll || !cr.inRead {
		sh.mu.Unlock()
		return
	}
	rawConn := e.rawConn
	sh.mu.Unlock()

	fd, hasData, closed, armed, rerr := peekAndArmSocket(rawConn, p)
	if armed && fd >= 0 {
		// Spurious readiness (EAGAIN); re-armed oneshot watch inside Control.
		byFD[fd] = e
		return
	}
	if !closed && !hasData {
		// Re-arm failed or fd is non-peekable; fall back to standard backgroundRead.
		sh.mu.Lock()
		if sh.entries[cr] == e && e.state == bgStatePoll {
			e.state = bgStateFallbk
			sh.mu.Unlock()
			httpBgStartFallback(cr, e)
			return
		}
		sh.mu.Unlock()
		return
	}

	sh.mu.Lock()
	if sh.entries[cr] == e && e.state == bgStatePoll {
		delete(sh.entries, cr)
		e.state = bgStateDisarm
		e.disarmed.Store(true)
		sh.mu.Unlock()
		httpBgMgr.totalArmed.Add(-1)
	} else {
		sh.mu.Unlock()
		return
	}

	if closed {
		if rerr == nil {
			rerr = io.EOF
		}
		cr.handleReadErrorLocked(rerr)
	}
	cr.aborted = false
	cr.inRead = false
	cr.cond.Broadcast()
}

func (cr *connReader) startBackgroundRead() {
	cr.lock()
	defer cr.unlock()
	if cr.inRead {
		panic("invalid concurrent Body.Read call")
	}
	if cr.hasByte {
		return
	}
	cr.inRead = true
	if cr.conn != nil && (cr.conn.server.ReadTimeout != 0 || cr.conn.server.ReadHeaderTimeout != 0) {
		cr.rwc.SetReadDeadline(time.Time{})
	}
	if !httpBgWatchArm(cr) {
		cr.rwc.SetReadDeadline(time.Time{})
		go cr.backgroundRead()
	}
}

func (cr *connReader) abortPendingRead() {
	cr.lock()
	defer cr.unlock()
	if !cr.inRead {
		return
	}
	if httpBgWatchDisarm(cr) {
		cr.aborted = false
		cr.inRead = false
		return
	}
	cr.aborted = true
	cr.rwc.SetReadDeadline(aLongTimeAgo)
	for cr.inRead {
		cr.cond.Wait()
	}
	cr.rwc.SetReadDeadline(time.Time{})
}
