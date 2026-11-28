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
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// On LLGo's 1:1 OS-thread runtime, keeping a dedicated goroutine blocked in
// bufio.Reader.Peek(4) for every idle HTTP/1.1 Keep-Alive connection holds one
// OS thread (and BDWGC STW stack scan) per idle socket.
//
// Offload idle HTTP/1.1 Keep-Alive sockets to a single shared kqueue/epoll
// watcher after a bounded hot-grace window (up to httpIdleMaxHot concurrent
// hot waiters so active keep-alive bursts incur minimal handoff overhead while
// connection storms offload immediately):
//   - When no explicit IdleTimeout/ReadTimeout is configured, hot waiters block
//     in bufio.Reader.Peek(4) with no per-request deadline timer and are evicted
//     lazily by the watcher after ~100..150ms idle (httpIdleHotGrace).
//   - When an explicit IdleTimeout or ReadTimeout is configured, hot waiters
//     use a short bounded deadline (httpIdleDeadlineGrace, 5ms) before parking.
//   - When a parked socket becomes readable or receives EOF/RST, the watcher
//     dispatches the connection to an elastic worker pool (which always spawns
//     a new goroutine if all cached workers are busy, so slow handlers or
//     self-call loopback requests can never starve or deadlock).
//   - IdleTimeout expiry, Server.closeIdleConns, Server.Close, and external
//     net.Conn closures are all observed promptly and run the exact same
//     connection cleanup and StateClosed hook path.
//   - TLS, net.Pipe, custom net.Conn mocks, and already-buffered pipelined
//     connections fall back transparently to standard bufio.Reader.Peek(4).
//
// Lock ordering invariant:
//   Server.mu -> httpIdleMgr.mu
//   internal/poll.FD.fdmu (rawConn.Control) -> httpIdleMgr.mu
// Never call rawConn.Control, c.rwc.Close, or user hooks under httpIdleMgr.mu.

const (
	httpIdleHotGrace      = 150 * time.Millisecond
	httpIdleDeadlineGrace = 5 * time.Millisecond
	httpIdleMaxHot        = 128
	httpIdleMinWorkers    = 16
	httpIdleProbeNs       = 500_000_000 // 500ms liveness check for external rwc.Close
	httpIdleWorkerTTL     = 500 * time.Millisecond
	httpIdleWarmWorkerTTL = 10 * time.Second
	httpHotSweepTickNs    = int64(httpIdleHotGrace / httpHotIdleTicks)
	httpHotIdleTicks      = 3 // evict hot conn after ~100..150ms idle
	httpHotEvictBit       = uint64(1) << 63
	httpHotClosedVal      = uint64(0)
	httpIdleStateParkng   = 1
	httpIdleStateParked   = 2
	httpIdleStateTaken    = 3
)

type httpHotEntry struct {
	c         *conn
	idleGen   atomic.Uint64 // odd: blocked in Peek(4); even: active/closed; bit63: sweeper evicted
	lastGen   uint64        // watcher-private
	idleTicks uint8         // watcher-private
}

type httpIdleEntry struct {
	c         *conn
	ctx       context.Context
	cancelCtx context.CancelFunc
	rawConn   syscall.RawConn
	fd        int
	deadline  time.Time
	state     uint8
	closed    bool
}

var httpIdleMgr struct {
	once          sync.Once
	ready         atomic.Bool
	poller        *httpBgPoller
	mu            sync.Mutex
	byConn        map[*conn]*httpIdleEntry
	byFD          map[int]*httpIdleEntry
	hotEntries    map[*httpHotEntry]struct{}
	deadlineCount int
	nextWakeNs    int64 // UnixNano of watcher's next wake deadline; 0 means sleeping indefinitely
	hotCount      atomic.Int64
	work          chan func()
	idleWorkers   atomic.Int32
	maxWorkers    int32
}

func ensureHTTPIdleWatcher() bool {
	httpIdleMgr.once.Do(func() {
		p, err := newHTTPBgPoller()
		if err != nil {
			return
		}
		httpIdleMgr.byConn = make(map[*conn]*httpIdleEntry, 64)
		httpIdleMgr.byFD = make(map[int]*httpIdleEntry, 64)
		httpIdleMgr.hotEntries = make(map[*httpHotEntry]struct{}, 32)
		httpIdleMgr.work = make(chan func())
		maxW := int32(runtime.GOMAXPROCS(0) * 2)
		if maxW < 4 {
			maxW = 4
		}
		if maxW > 32 {
			maxW = 32
		}
		httpIdleMgr.maxWorkers = maxW
		httpIdleMgr.poller = p
		httpIdleMgr.ready.Store(true)
		go runHTTPIdleWatcher(p)
	})
	return httpIdleMgr.ready.Load()
}

func httpHotRegister(c *conn) *httpHotEntry {
	if !ensureHTTPIdleWatcher() {
		return nil
	}
	he := &httpHotEntry{c: c}
	var wake bool
	wantWakeBy := time.Now().UnixNano() + httpHotSweepTickNs
	httpIdleMgr.mu.Lock()
	httpIdleMgr.hotEntries[he] = struct{}{}
	if httpIdleMgr.nextWakeNs == 0 || httpIdleMgr.nextWakeNs > wantWakeBy {
		wake = true
	}
	httpIdleMgr.mu.Unlock()
	if wake {
		httpIdleMgr.poller.wake()
	}
	return he
}

func httpHotUnregister(he *httpHotEntry) {
	if he == nil {
		return
	}
	he.idleGen.Store(httpHotClosedVal)
	httpIdleMgr.mu.Lock()
	delete(httpIdleMgr.hotEntries, he)
	httpIdleMgr.mu.Unlock()
}

// httpIdleDispatch runs fn on an already-idle pooled worker if one is waiting,
// otherwise spawns a new goroutine immediately. It never blocks the caller and
// cannot deadlock even if all pooled workers block on slow I/O or loopback calls.
func httpIdleDispatch(fn func()) {
	select {
	case httpIdleMgr.work <- fn:
		return
	default:
	}
	go httpIdleWorkerRun(fn)
}

func httpIdleWorkerRun(fn func()) {
	var timer *time.Timer
	for {
		fn()
		select {
		case next := <-httpIdleMgr.work:
			fn = next
			continue
		default:
		}
		idle := httpIdleMgr.idleWorkers.Add(1)
		if idle > httpIdleMgr.maxWorkers {
			httpIdleMgr.idleWorkers.Add(-1)
			if timer != nil {
				timer.Stop()
			}
			return
		}
		// Keep a small warm core of idle workers longer than burst workers so
		// active dispatch bursts avoid frequent thread churn, while still
		// expiring cleanly when the server becomes idle.
		ttl := httpIdleWorkerTTL
		if idle <= httpIdleMinWorkers {
			ttl = httpIdleWarmWorkerTTL
		}
		if timer == nil {
			timer = time.NewTimer(ttl)
		} else {
			timer.Reset(ttl)
		}
		select {
		case next := <-httpIdleMgr.work:
			httpIdleMgr.idleWorkers.Add(-1)
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			fn = next
		case <-timer.C:
			httpIdleMgr.idleWorkers.Add(-1)
			return
		}
	}
}

func (e *httpIdleEntry) closeConnection() {
	e.cancelCtx()
	if !e.c.hijacked() {
		e.c.close()
		e.c.setState(e.c.rwc, StateClosed, runHooks)
	}
}

// httpIdlePark attempts to transfer ownership of an idle HTTP/1.1 connection to
// the shared kqueue/epoll watcher.
//   - parked=true: ownership transferred; caller MUST return without closing c.
//   - hasData=true: next request bytes are already readable on the socket.
//   - closed=true: socket is closed/errored or server began closing idle conns.
//   - all false: watcher unavailable; caller should fall back to bufr.Peek(4).
func httpIdlePark(c *conn, ctx context.Context, cancelCtx context.CancelFunc, rawConn syscall.RawConn, deadline time.Time) (parked, hasData, closed bool) {
	if !ensureHTTPIdleWatcher() {
		return false, false, false
	}
	if c.server.shuttingDown() || !c.server.doKeepAlives() {
		return false, false, true
	}

	entry := &httpIdleEntry{
		c:         c,
		ctx:       ctx,
		cancelCtx: cancelCtx,
		rawConn:   rawConn,
		fd:        -1,
		deadline:  deadline,
		state:     httpIdleStateParkng,
	}

	httpIdleMgr.mu.Lock()
	httpIdleMgr.byConn[c] = entry
	httpIdleMgr.mu.Unlock()

	var buf [1]byte
	var wakeWatcher bool
	ctrlErr := rawConn.Control(func(u uintptr) {
		fd := int(u)
		for {
			n, _, rerr := syscall.Recvfrom(fd, buf[:], syscall.MSG_PEEK|syscall.MSG_DONTWAIT)
			if rerr == syscall.EINTR {
				continue
			}
			if rerr == syscall.EAGAIN || rerr == syscall.EWOULDBLOCK {
				httpIdleMgr.mu.Lock()
				if entry.closed || c.server.shuttingDown() || !c.server.doKeepAlives() {
					if httpIdleMgr.byConn[c] == entry {
						delete(httpIdleMgr.byConn, c)
					}
					httpIdleMgr.mu.Unlock()
					closed = true
					return
				}
				entry.fd = fd
				entry.state = httpIdleStateParked
				httpIdleMgr.byFD[fd] = entry
				if err := httpIdleMgr.poller.add(fd); err != nil {
					delete(httpIdleMgr.byFD, fd)
					if httpIdleMgr.byConn[c] == entry {
						delete(httpIdleMgr.byConn, c)
					}
					entry.state = httpIdleStateTaken
					httpIdleMgr.mu.Unlock()
					return
				}
				if !deadline.IsZero() {
					httpIdleMgr.deadlineCount++
				}
				// Wake watcher if it was sleeping indefinitely or its next wake is
				// later than this entry's idle deadline.
				if httpIdleMgr.nextWakeNs == 0 {
					wakeWatcher = true
				} else if !deadline.IsZero() && deadline.UnixNano() < httpIdleMgr.nextWakeNs {
					wakeWatcher = true
				}
				parked = true
				httpIdleMgr.mu.Unlock()
				return
			}
			if rerr != nil {
				if isNonPeekableErrno(rerr) {
					return
				}
				closed = true
				return
			}
			if n == 0 {
				closed = true
				return
			}
			hasData = true
			return
		}
	})

	if parked {
		if wakeWatcher {
			httpIdleMgr.poller.wake()
		}
		return true, false, false
	}

	httpIdleMgr.mu.Lock()
	wasClosed := entry.closed
	if httpIdleMgr.byConn[c] == entry {
		delete(httpIdleMgr.byConn, c)
	}
	httpIdleMgr.mu.Unlock()

	if ctrlErr != nil || wasClosed {
		return false, false, true
	}
	return false, hasData, closed
}

// httpIdleNotifyClosed informs the idle watcher that Server.Close or
// Server.closeIdleConns has closed c.rwc. Safe to call while holding Server.mu.
func httpIdleNotifyClosed(c *conn) {
	if !httpIdleMgr.ready.Load() {
		return
	}
	httpIdleMgr.mu.Lock()
	entry, ok := httpIdleMgr.byConn[c]
	if !ok {
		httpIdleMgr.mu.Unlock()
		return
	}
	if entry.state == httpIdleStateParkng {
		entry.closed = true
		httpIdleMgr.mu.Unlock()
		return
	}
	if entry.state != httpIdleStateParked {
		httpIdleMgr.mu.Unlock()
		return
	}
	delete(httpIdleMgr.byConn, c)
	if httpIdleMgr.byFD[entry.fd] == entry {
		delete(httpIdleMgr.byFD, entry.fd)
	}
	if !entry.deadline.IsZero() {
		httpIdleMgr.deadlineCount--
	}
	entry.state = httpIdleStateTaken
	httpIdleMgr.mu.Unlock()

	httpIdleDispatch(entry.closeConnection)
}

func runHTTPIdleWatcher(p *httpBgPoller) {
	readyFDs := make([]int, 0, 64)
	expired := make([]*httpIdleEntry, 0, 32)
	probeList := make([]*httpIdleEntry, 0, 64)
	hotList := make([]*httpHotEntry, 0, 64)
	lastProbe := time.Now()
	nextHotSweep := time.Now().Add(time.Duration(httpHotSweepTickNs))

	for {
		now := time.Now()
		expired = expired[:0]
		var timeoutNs int64 = -1

		httpIdleMgr.mu.Lock()
		hasHot := len(httpIdleMgr.hotEntries) > 0
		if len(httpIdleMgr.byFD) > 0 {
			timeoutNs = httpIdleProbeNs
			if httpIdleMgr.deadlineCount > 0 {
				for fd, entry := range httpIdleMgr.byFD {
					if entry.state != httpIdleStateParked {
						delete(httpIdleMgr.byFD, fd)
						continue
					}
					if !entry.deadline.IsZero() {
						rem := entry.deadline.Sub(now)
						if rem <= 0 {
							delete(httpIdleMgr.byFD, fd)
							if httpIdleMgr.byConn[entry.c] == entry {
								delete(httpIdleMgr.byConn, entry.c)
							}
							httpIdleMgr.deadlineCount--
							entry.state = httpIdleStateTaken
							expired = append(expired, entry)
							continue
						}
						if int64(rem) < timeoutNs {
							timeoutNs = int64(rem)
						}
					}
				}
				if len(httpIdleMgr.byFD) == 0 {
					timeoutNs = -1
				}
			}
		}
		if hasHot {
			hotRem := int64(nextHotSweep.Sub(now))
			if hotRem < 1_000_000 {
				hotRem = 1_000_000
			}
			if timeoutNs < 0 || hotRem < timeoutNs {
				timeoutNs = hotRem
			}
		}
		if timeoutNs < 0 {
			httpIdleMgr.nextWakeNs = 0
		} else {
			if timeoutNs < 1_000_000 {
				timeoutNs = 1_000_000
			}
			httpIdleMgr.nextWakeNs = now.UnixNano() + timeoutNs
		}
		httpIdleMgr.mu.Unlock()

		for _, entry := range expired {
			e := entry
			httpIdleDispatch(e.closeConnection)
		}

		readyFDs = p.wait(timeoutNs, readyFDs[:0])

		for _, fd := range readyFDs {
			httpIdleMgr.mu.Lock()
			entry, ok := httpIdleMgr.byFD[fd]
			if !ok || entry.state != httpIdleStateParked {
				httpIdleMgr.mu.Unlock()
				continue
			}
			delete(httpIdleMgr.byFD, fd)
			if httpIdleMgr.byConn[entry.c] == entry {
				delete(httpIdleMgr.byConn, entry.c)
			}
			if !entry.deadline.IsZero() {
				httpIdleMgr.deadlineCount--
			}
			entry.state = httpIdleStateTaken
			httpIdleMgr.mu.Unlock()

			e := entry
			httpIdleDispatch(func() {
				e.c.serveHTTP1(e.ctx, e.cancelCtx, true, e.deadline)
			})
		}

		// Lazily evict hot-grace keep-alive connections that have been idle in
		// Peek(4) across httpHotIdleTicks consecutive sweeps (~100..150ms) so
		// active keep-alive loops avoid per-request deadline timers.
		if nowSweep := time.Now(); !nowSweep.Before(nextHotSweep) {
			nextHotSweep = nowSweep.Add(time.Duration(httpHotSweepTickNs))
			hotList = hotList[:0]
			httpIdleMgr.mu.Lock()
			for he := range httpIdleMgr.hotEntries {
				hotList = append(hotList, he)
			}
			httpIdleMgr.mu.Unlock()

			for _, he := range hotList {
				g := he.idleGen.Load()
				if g&1 == 1 && g&httpHotEvictBit == 0 {
					if g == he.lastGen {
						he.idleTicks++
					} else {
						he.lastGen = g
						he.idleTicks = 1
					}
					if he.idleTicks >= httpHotIdleTicks {
						he.c.r.lock()
						if he.idleGen.CompareAndSwap(g, g|httpHotEvictBit) {
							he.c.r.aborted = true
							he.c.rwc.SetReadDeadline(aLongTimeAgo)
						}
						he.c.r.unlock()
						he.lastGen = 0
						he.idleTicks = 0
					}
				} else {
					he.lastGen = 0
					he.idleTicks = 0
				}
			}
		}

		// Periodically probe parked rawConns outside httpIdleMgr.mu so even an
		// external c.rwc.Close() (outside Server.Close/closeIdleConns) is caught.
		if time.Since(lastProbe) >= time.Duration(httpIdleProbeNs) {
			lastProbe = time.Now()
			probeList = probeList[:0]
			httpIdleMgr.mu.Lock()
			for _, entry := range httpIdleMgr.byFD {
				if entry.state == httpIdleStateParked {
					probeList = append(probeList, entry)
				}
			}
			httpIdleMgr.mu.Unlock()

			for _, entry := range probeList {
				if err := entry.rawConn.Control(func(uintptr) {}); err != nil {
					httpIdleMgr.mu.Lock()
					if entry.state == httpIdleStateParked {
						if httpIdleMgr.byFD[entry.fd] == entry {
							delete(httpIdleMgr.byFD, entry.fd)
						}
						if httpIdleMgr.byConn[entry.c] == entry {
							delete(httpIdleMgr.byConn, entry.c)
						}
						if !entry.deadline.IsZero() {
							httpIdleMgr.deadlineCount--
						}
						entry.state = httpIdleStateTaken
						httpIdleMgr.mu.Unlock()
						e := entry
						httpIdleDispatch(e.closeConnection)
						continue
					}
					httpIdleMgr.mu.Unlock()
				}
			}
		}
	}
}

// Serve a new connection.
func (c *conn) serve(ctx context.Context) {
	if ra := c.rwc.RemoteAddr(); ra != nil {
		c.remoteAddr = ra.String()
	}
	ctx = context.WithValue(ctx, LocalAddrContextKey, c.rwc.LocalAddr())
	handedToHTTP1 := false
	defer func() {
		if handedToHTTP1 {
			return
		}
		if err := recover(); err != nil && err != ErrAbortHandler {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			c.server.logf("http: panic serving %v: %v\n%s", c.remoteAddr, err, buf)
		}
		if !c.hijacked() {
			c.close()
			c.setState(c.rwc, StateClosed, runHooks)
		}
	}()

	type connectionStater interface {
		ConnectionState() tls.ConnectionState
	}
	type handshakeContexter interface {
		HandshakeContext(ctx context.Context) error
	}
	if connStater, ok := c.rwc.(connectionStater); ok {
		tlsTO := c.server.tlsHandshakeTimeout()
		if tlsTO > 0 {
			dl := time.Now().Add(tlsTO)
			c.rwc.SetReadDeadline(dl)
			c.rwc.SetWriteDeadline(dl)
		}
		var err error
		if handshaker, ok := c.rwc.(handshakeContexter); ok {
			err = handshaker.HandshakeContext(ctx)
		}
		if err != nil {
			var reason string
			if re, ok := err.(tls.RecordHeaderError); ok && re.Conn != nil && tlsRecordHeaderLooksLikeHTTP(re.RecordHeader) {
				io.WriteString(re.Conn, "HTTP/1.0 400 Bad Request\r\n\r\nClient sent an HTTP request to an HTTPS server.\n")
				re.Conn.Close()
				reason = "client sent an HTTP request to an HTTPS server"
			} else {
				reason = err.Error()
			}
			c.server.logf("http: TLS handshake error from %s: %v", c.rwc.RemoteAddr(), reason)
			return
		}
		if tlsTO > 0 {
			c.rwc.SetReadDeadline(time.Time{})
			c.rwc.SetWriteDeadline(time.Time{})
		}
		c.tlsState = new(tls.ConnectionState)
		*c.tlsState = connStater.ConnectionState()
		proto := c.tlsState.NegotiatedProtocol
		if proto == "h2" && c.server.h2 != nil {
			c.setState(c.rwc, StateActive, skipHooks)
			const sawClientPreface = false
			c.server.serveHTTP2Conn(ctx, c.rwc, serverHandler{c.server}, sawClientPreface, nil, nil)
			return
		}
		tlsConn, tlsConnOK := c.rwc.(*tls.Conn)
		if validNextProto(proto) && tlsConnOK {
			if fn := c.server.TLSNextProto[proto]; fn != nil {
				h := initALPNRequest{ctx, tlsConn, serverHandler{c.server}}
				c.setState(c.rwc, StateActive, skipHooks)
				fn(c.server, tlsConn, h)
			}
			return
		}
	}

	if c.tlsState == nil {
		if tc, ok := c.rwc.(connectionStater); ok {
			c.tlsState = new(tls.ConnectionState)
			*c.tlsState = tc.ConnectionState()
		}
	}

	ctx, cancelCtx := context.WithCancel(ctx)
	c.cancelCtx = cancelCtx

	c.r = &connReader{conn: c, rwc: c.rwc}
	c.bufr = newBufioReader(c.r)
	c.bufw = newBufioWriterSize(checkConnErrorWriter{c}, 4<<10)

	if d := c.server.readHeaderTimeout(); d > 0 {
		c.rwc.SetReadDeadline(time.Now().Add(d))
	}

	protos := c.server.protocols()
	if c.tlsState == nil && protos.UnencryptedHTTP2() {
		if c.maybeServeUnencryptedHTTP2(ctx) {
			cancelCtx()
			return
		}
	}
	if !protos.HTTP1() {
		cancelCtx()
		return
	}

	handedToHTTP1 = true
	c.serveHTTP1(ctx, cancelCtx, false, time.Time{})
}

func requestBodyLimitReached(w *response) bool {
	// Go 1.27.4 changed requestBodyLimitHit from bool to atomic.Bool.
	if flag, ok := any(&w.requestBodyLimitHit).(*bool); ok {
		return *flag
	}
	return any(&w.requestBodyLimitHit).(*atomic.Bool).Load()
}

func (c *conn) serveHTTP1(ctx context.Context, cancelCtx context.CancelFunc, resumed bool, resumedIdleDeadline time.Time) {
	var inFlightResponse *response
	handedOff := false
	// Claim a hot-grace slot once for this worker activation instead of
	// bouncing a global atomic counter on every keep-alive request.
	hotSlot := httpIdleMgr.hotCount.Add(1) <= httpIdleMaxHot
	if !hotSlot {
		httpIdleMgr.hotCount.Add(-1)
	}
	// Cache syscall.RawConn once per activation for eligible plaintext sockets.
	var cachedRawConn syscall.RawConn
	if c.tlsState == nil {
		if _, isTLS := c.rwc.(*tls.Conn); !isTLS {
			if sc, ok := c.rwc.(syscall.Conn); ok {
				if rc, err := sc.SyscallConn(); err == nil {
					cachedRawConn = rc
				}
			}
		}
	}
	var hotEntry *httpHotEntry
	if hotSlot && cachedRawConn != nil {
		hotEntry = httpHotRegister(c)
	}
	var hotWaitSeq uint64 = 1
	defer func() {
		if hotEntry != nil {
			httpHotUnregister(hotEntry)
		}
		if hotSlot {
			httpIdleMgr.hotCount.Add(-1)
		}
		if handedOff {
			return
		}
		if err := recover(); err != nil && err != ErrAbortHandler {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			c.server.logf("http: panic serving %v: %v\n%s", c.remoteAddr, err, buf)
		}
		if inFlightResponse != nil {
			inFlightResponse.cancelCtx()
			inFlightResponse.disableWriteContinue(true)
		}
		cancelCtx()
		if !c.hijacked() {
			if inFlightResponse != nil {
				inFlightResponse.conn.r.abortPendingRead()
				inFlightResponse.reqBody.Close()
			}
			c.close()
			c.setState(c.rwc, StateClosed, runHooks)
		}
	}()

	if resumed {
		if !c.server.doKeepAlives() {
			return
		}
		if !resumedIdleDeadline.IsZero() {
			if !time.Now().Before(resumedIdleDeadline) {
				return
			}
			c.rwc.SetReadDeadline(resumedIdleDeadline)
		} else {
			c.rwc.SetReadDeadline(time.Time{})
		}
		if _, err := c.bufr.Peek(4); err != nil {
			return
		}
		if d := c.server.readHeaderTimeout(); d > 0 {
			c.rwc.SetReadDeadline(time.Now().Add(d))
		} else {
			c.rwc.SetReadDeadline(time.Time{})
		}
	}

	for {
		w, err := c.readRequest(ctx)
		if c.r.remain != c.server.initialReadLimitSize() {
			c.setState(c.rwc, StateActive, runHooks)
		}
		if c.server.shuttingDown() {
			return
		}
		if err != nil {
			const errorHeaders = "\r\nContent-Type: text/plain; charset=utf-8\r\nConnection: close\r\n\r\n"

			switch {
			case err == errTooLarge:
				const publicErr = "431 Request Header Fields Too Large"
				fmt.Fprintf(c.rwc, "HTTP/1.1 "+publicErr+errorHeaders+publicErr)
				c.closeWriteAndWait()
				return

			case isUnsupportedTEError(err):
				code := StatusNotImplemented
				fmt.Fprintf(c.rwc, "HTTP/1.1 %d %s%sUnsupported transfer encoding", code, StatusText(code), errorHeaders)
				return

			case isCommonNetReadError(err):
				return

			default:
				if v, ok := err.(statusError); ok {
					fmt.Fprintf(c.rwc, "HTTP/1.1 %d %s: %s%s%d %s: %s", v.code, StatusText(v.code), v.text, errorHeaders, v.code, StatusText(v.code), v.text)
					return
				}
				const publicErr = "400 Bad Request"
				fmt.Fprintf(c.rwc, "HTTP/1.1 "+publicErr+errorHeaders+publicErr)
				return
			}
		}

		req := w.req
		if req.expectsContinue() {
			if req.ProtoAtLeast(1, 1) && req.ContentLength != 0 {
				w.ecReader = &expectContinueReader{readCloser: req.Body, resp: w}
				w.canWriteContinue.Store(true)
				req.Body = w.ecReader
			}
		} else if req.Header.get("Expect") != "" {
			w.sendExpectationFailed()
			return
		}

		c.curReq.Store(w)

		if w.reqBody.bodyRemains() {
			w.reqBody.registerOnHitEOF(w.conn.r.startBackgroundRead)
		} else {
			w.conn.r.startBackgroundRead()
		}

		inFlightResponse = w
		serverHandler{c.server}.ServeHTTP(w, w.req)
		inFlightResponse = nil
		w.cancelCtx()
		if c.hijacked() {
			c.r.releaseConn()
			return
		}
		w.finishRequest()
		c.rwc.SetWriteDeadline(time.Time{})
		if !w.shouldReuseConnection() {
			if requestBodyLimitReached(w) || w.closedRequestBodyEarly() {
				c.closeWriteAndWait()
			}
			return
		}
		c.setState(c.rwc, StateIdle, runHooks)
		c.curReq.Store(nil)

		if !w.conn.server.doKeepAlives() {
			return
		}

		var idleDeadline time.Time
		if d := c.server.idleTimeout(); d > 0 {
			idleDeadline = time.Now().Add(d)
		}
		readHeaderTO := c.server.readHeaderTimeout()

		if cachedRawConn != nil && c.bufr.Buffered() == 0 && !c.r.hasByte {
			needPark := false
			if hotSlot {
				if hotEntry != nil && idleDeadline.IsZero() {
					// Lazy watcher-driven hot-grace wait when no IdleTimeout/ReadTimeout
					// is set: clear any read deadline a handler may have installed via
					// ResponseController.SetReadDeadline, then block in Peek(4) with no
					// grace timer. The idle watcher evicts only if hotEntry.idleGen
					// remains unchanged across httpHotIdleTicks (~100..150ms).
					c.rwc.SetReadDeadline(time.Time{})
					seq := hotWaitSeq
					hotWaitSeq += 2
					if hotWaitSeq&httpHotEvictBit != 0 {
						hotWaitSeq = 1
					}
					hotEntry.idleGen.Store(seq)
					_, peekErr := c.bufr.Peek(4)
					prev := hotEntry.idleGen.Swap(seq + 1)
					if prev&httpHotEvictBit != 0 {
						c.r.lock()
						c.r.aborted = false
						c.rwc.SetReadDeadline(time.Time{})
						c.r.unlock()
						if peekErr == nil || c.bufr.Buffered() > 0 {
							if readHeaderTO > 0 {
								c.rwc.SetReadDeadline(time.Now().Add(readHeaderTO))
							}
							continue
						}
						if os.IsTimeout(peekErr) {
							needPark = true
						} else {
							return
						}
					} else {
						if peekErr == nil || c.bufr.Buffered() > 0 {
							if readHeaderTO > 0 {
								c.rwc.SetReadDeadline(time.Now().Add(readHeaderTO))
							}
							continue
						}
						return
					}
				} else {
					graceDeadline := time.Now().Add(httpIdleDeadlineGrace)
					if !idleDeadline.IsZero() && idleDeadline.Before(graceDeadline) {
						graceDeadline = idleDeadline
					}
					// Explicit IdleTimeout/ReadTimeout (or watcher init fallback): use
					// a short bounded grace deadline on c.bufr.Peek(4).
					c.rwc.SetReadDeadline(graceDeadline)
					c.r.lock()
					c.r.aborted = true
					c.r.unlock()
					_, peekErr := c.bufr.Peek(4)
					c.r.lock()
					c.r.aborted = false
					c.r.unlock()
					if peekErr == nil {
						if readHeaderTO > 0 {
							c.rwc.SetReadDeadline(time.Now().Add(readHeaderTO))
						} else {
							c.rwc.SetReadDeadline(time.Time{})
						}
						continue
					}
					if os.IsTimeout(peekErr) {
						if !idleDeadline.IsZero() && !time.Now().Before(idleDeadline) {
							return
						}
						// An incomplete prefix in bufr must finish under the original
						// idle deadline. The poller only sees bytes in the socket.
						needPark = c.bufr.Buffered() == 0
					} else {
						return
					}
				}
			} else {
				needPark = true
			}

			if needPark {
				c.rwc.SetReadDeadline(time.Time{})
				parked, hasData, closed := httpIdlePark(c, ctx, cancelCtx, cachedRawConn, idleDeadline)
				if parked {
					handedOff = true
					return
				}
				if closed {
					return
				}
				if hasData {
					if !idleDeadline.IsZero() {
						c.rwc.SetReadDeadline(idleDeadline)
					}
					if _, err := c.bufr.Peek(4); err != nil {
						return
					}
					if readHeaderTO > 0 {
						c.rwc.SetReadDeadline(time.Now().Add(readHeaderTO))
					} else if !idleDeadline.IsZero() {
						c.rwc.SetReadDeadline(time.Time{})
					}
					continue
				}
			}
		}

		// Wait for a complete prefix before starting the next request's timeouts.
		if !idleDeadline.IsZero() {
			c.rwc.SetReadDeadline(idleDeadline)
		} else {
			c.rwc.SetReadDeadline(time.Time{})
		}
		if _, err := c.bufr.Peek(4); err != nil {
			return
		}
		if readHeaderTO > 0 {
			c.rwc.SetReadDeadline(time.Now().Add(readHeaderTO))
		} else if !idleDeadline.IsZero() {
			c.rwc.SetReadDeadline(time.Time{})
		}
	}
}

func (s *Server) Close() error {
	s.inShutdown.Store(true)
	s.mu.Lock()
	defer s.mu.Unlock()
	err := s.closeListenersLocked()

	s.mu.Unlock()
	s.listenerGroup.Wait()
	s.mu.Lock()

	for c := range s.activeConn {
		c.rwc.Close()
		httpIdleNotifyClosed(c)
		delete(s.activeConn, c)
	}
	return err
}

func (s *Server) closeIdleConns() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	quiescent := true
	for c := range s.activeConn {
		st, unixSec := c.getState()
		if st == StateNew && unixSec < time.Now().Unix()-5 {
			st = StateIdle
		}
		if st != StateIdle || unixSec == 0 {
			quiescent = false
			continue
		}
		c.rwc.Close()
		httpIdleNotifyClosed(c)
		delete(s.activeConn, c)
	}
	return quiescent
}
