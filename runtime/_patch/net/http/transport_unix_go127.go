// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.
// See LICENSES/Go-BSD-3-Clause.txt at this module root for license terms.
//
// Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
// Licensed under the Apache License, Version 2.0 (see LICENSE).

//go:build (darwin || linux) && !baremetal && go1.27 && !go1.28

package http

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http/httptrace"
	"net/http/internal/ascii"
	"net/textproto"
	"net/url"
	"os"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

// Upstream provenance:
//   `getConn`, `dialConn`, `roundTrip` (async fallback), `closeLocked`, and
//   `proxyAuth` are derived from Go 1.27 `src/net/http/transport.go` and must
//   be resynced when the target Go toolchain major version advances (gated by
//   the `go1.27 && !go1.28` build constraint above).
//
// On LLGo's 1:1 OS-thread runtime, Go's standard HTTP/1.1 client Transport
// spawns two dedicated background goroutines (readLoop + writeLoop) per
// persistent connection, multiplying OS threads by 3x and adding 5 cross-thread
// channel handoffs to every keep-alive request.
//
// For pooled plaintext socket HTTP/1.1 connections (non-ClientConn, non-TLS,
// non-Expect-100-continue, no write-side httptrace hooks, and bodyless or
// known in-memory request bodies <= 64 KiB via Request.outgoingLength() that
// cannot block reading or deadlock against kernel TCP send buffers), keep
// persistConn in synchronous mode (pc.writech == nil):
//   - Write the request and read response headers directly on the caller's
//     goroutine with zero channel hops and zero extra OS threads.
//   - Attach request context cancellation via context.AfterFunc (zero goroutines
//     on Transport.roundTrip's stdlib cancelCtx) to close pc.conn promptly.
//   - Recycle idle sockets into a shared kqueue/epoll watcher after a brief
//     userspace pending window (~1-2ms, zero syscalls on hot keep-alive reuse)
//     so server-initiated idle closes/EOF/RST are detected promptly.
//   - Lazily promote to standard readLoop+writeLoop on demand for Expect: 100-
//     continue, streaming/unknown-length or >64 KiB request bodies, or requests
//     with write-side httptrace hooks (WroteHeaders/WroteRequest/100-continue).

// errSyncClosedEarly is local to the patch so it compiles across all Go 1.27.x
// point releases (upstream net/http added errClosedEarly in a 1.27 point release).
var errSyncClosedEarly = errors.New("net/http: response body closed early")

type nopPlainReader struct{}

func (nopPlainReader) Read([]byte) (int, error) { return 0, io.EOF }

var (
	nopCloserType         = reflect.TypeOf(io.NopCloser(nopPlainReader{}))
	nopCloserWriterToType = reflect.TypeOf(io.NopCloser((*bytes.Reader)(nil)))
)

// isInMemoryRequestBody reports whether body is empty or a standard in-memory
// reader (*bytes.Buffer, *bytes.Reader, *strings.Reader wrapped by io.NopCloser
// and/or readTrackingBody). Streaming readers with a known Content-Length stay
// on the async path so an early final response from the server is not blocked
// behind a stalled Request.Body.Read.
func isInMemoryRequestBody(body io.ReadCloser) bool {
	if body == nil || body == NoBody {
		return true
	}
	if rt, ok := body.(*readTrackingBody); ok {
		body = rt.ReadCloser
		if body == nil || body == NoBody {
			return true
		}
	}
	rv := reflect.ValueOf(body)
	if !rv.IsValid() {
		return true
	}
	if t := rv.Type(); t == nopCloserWriterToType || t == nopCloserType {
		if inner, ok := rv.Field(0).Interface().(io.Reader); ok {
			switch inner.(type) {
			case *bytes.Buffer, *bytes.Reader, *strings.Reader:
				return true
			}
		}
	}
	return false
}

const (
	clientIdleShardCount  = 16
	clientIdleTickNs      = 1_000_000 // 1ms promotion tick while entries are pending
	clientIdleMaxSyncBody = 64 << 10
	clientIdleUnused      = 0
	clientIdlePend        = 1
	clientIdlePoll        = 2
	clientIdleDisarm      = 3
)

type clientIdleEntry struct {
	pc       *persistConn
	rawConn  syscall.RawConn
	fd       int
	state    uint8
	seen     bool
	disarmed atomic.Bool
}

type clientIdleShard struct {
	mu      sync.Mutex
	entries map[*persistConn]*clientIdleEntry
	pending map[*persistConn]*clientIdleEntry
	_       [32]byte
}

var httpClientIdleMgr struct {
	once          sync.Once
	ready         atomic.Bool
	poller        *httpBgPoller
	shards        [clientIdleShardCount]clientIdleShard
	totalArmed    atomic.Int64
	pendingCount  atomic.Int64
	polledDisarms atomic.Int64
	sleeping      atomic.Int32
}

func clientIdleShardFor(pc *persistConn) *clientIdleShard {
	h := uintptr(unsafe.Pointer(pc))
	return &httpClientIdleMgr.shards[(h>>6)&(clientIdleShardCount-1)]
}

func ensureHTTPClientIdleWatcher() bool {
	httpClientIdleMgr.once.Do(func() {
		p, err := newHTTPBgPoller()
		if err != nil {
			return
		}
		for i := range httpClientIdleMgr.shards {
			httpClientIdleMgr.shards[i].entries = make(map[*persistConn]*clientIdleEntry, 16)
			httpClientIdleMgr.shards[i].pending = make(map[*persistConn]*clientIdleEntry, 8)
		}
		httpClientIdleMgr.poller = p
		httpClientIdleMgr.ready.Store(true)
		go runHTTPClientIdleWatcher(p)
	})
	return httpClientIdleMgr.ready.Load()
}

func httpClientIdleArm(pc *persistConn) {
	if !httpClientIdleMgr.ready.Load() {
		return
	}
	e := &clientIdleEntry{
		pc:    pc,
		fd:    -1,
		state: clientIdlePend,
	}

	var replacedState uint8
	sh := clientIdleShardFor(pc)
	sh.mu.Lock()
	if prev, ok := sh.entries[pc]; ok {
		replacedState = prev.state
		prev.state = clientIdleDisarm
		prev.disarmed.Store(true)
	}
	sh.entries[pc] = e
	sh.pending[pc] = e
	sh.mu.Unlock()

	if replacedState == clientIdleUnused {
		httpClientIdleMgr.totalArmed.Add(1)
	}
	if replacedState != clientIdlePend {
		httpClientIdleMgr.pendingCount.Add(1)
	}
	if httpClientIdleMgr.sleeping.Load() != 0 {
		if httpClientIdleMgr.sleeping.CompareAndSwap(1, 0) {
			httpClientIdleMgr.poller.wake()
		}
	}
}

func httpClientIdleDisarm(pc *persistConn) {
	if !httpClientIdleMgr.ready.Load() {
		return
	}
	sh := clientIdleShardFor(pc)
	sh.mu.Lock()
	e, ok := sh.entries[pc]
	if !ok {
		sh.mu.Unlock()
		return
	}
	delete(sh.entries, pc)
	delete(sh.pending, pc)
	prevState := e.state
	rawConn := e.rawConn
	e.state = clientIdleDisarm
	e.disarmed.Store(true)
	sh.mu.Unlock()

	httpClientIdleMgr.totalArmed.Add(-1)
	if prevState == clientIdlePend {
		httpClientIdleMgr.pendingCount.Add(-1)
		return
	}
	if prevState == clientIdlePoll {
		if rawConn != nil {
			_ = rawConn.Control(func(u uintptr) {
				httpClientIdleMgr.poller.del(int(u))
			})
		}
		if httpClientIdleMgr.polledDisarms.Add(1) >= 64 && httpClientIdleMgr.sleeping.Load() != 0 {
			if httpClientIdleMgr.sleeping.CompareAndSwap(1, 0) {
				httpClientIdleMgr.poller.wake()
			}
		}
	}
}

func runHTTPClientIdleWatcher(p *httpBgPoller) {
	byFD := make(map[int]*clientIdleEntry, 64)
	pending := make([]*persistConn, 0, 64)
	readyFDs := make([]int, 0, 64)

	for {
		var timeoutNs int64 = clientIdleTickNs
		if httpClientIdleMgr.pendingCount.Load() == 0 && httpClientIdleMgr.polledDisarms.Load() == 0 {
			httpClientIdleMgr.sleeping.Store(1)
			if httpClientIdleMgr.pendingCount.Load() == 0 && httpClientIdleMgr.polledDisarms.Load() == 0 {
				timeoutNs = -1
			} else {
				httpClientIdleMgr.sleeping.Store(0)
			}
		}

		readyFDs = p.wait(timeoutNs, readyFDs[:0])
		httpClientIdleMgr.sleeping.Store(0)

		for _, fd := range readyFDs {
			e, ok := byFD[fd]
			if !ok {
				continue
			}
			delete(byFD, fd)
			handleReadyClientIdleEntry(p, e, byFD)
		}

		if httpClientIdleMgr.pendingCount.Load() > 0 {
			pending = pending[:0]
			for i := range httpClientIdleMgr.shards {
				sh := &httpClientIdleMgr.shards[i]
				sh.mu.Lock()
				for pc, e := range sh.pending {
					if e.state == clientIdlePend {
						if !e.seen {
							e.seen = true
						} else {
							pending = append(pending, pc)
						}
					} else {
						delete(sh.pending, pc)
					}
				}
				sh.mu.Unlock()
			}
			for _, pc := range pending {
				promotePendingClientIdleEntry(p, pc, byFD)
			}
		}

		if len(byFD) > 0 && (httpClientIdleMgr.polledDisarms.Swap(0) > 0 || httpClientIdleMgr.totalArmed.Load() == 0) {
			for fd, e := range byFD {
				if e.disarmed.Load() {
					delete(byFD, fd)
				}
			}
		}
	}
}

func promotePendingClientIdleEntry(p *httpBgPoller, pc *persistConn, byFD map[int]*clientIdleEntry) {
	sh := clientIdleShardFor(pc)
	sh.mu.Lock()
	e, ok := sh.entries[pc]
	if !ok || e.state != clientIdlePend {
		delete(sh.pending, pc)
		sh.mu.Unlock()
		return
	}
	sh.mu.Unlock()

	if pc.isBroken() {
		sh.mu.Lock()
		if sh.entries[pc] == e && e.state == clientIdlePend {
			delete(sh.entries, pc)
			delete(sh.pending, pc)
			e.state = clientIdleDisarm
			e.disarmed.Store(true)
			sh.mu.Unlock()
			httpClientIdleMgr.pendingCount.Add(-1)
			httpClientIdleMgr.totalArmed.Add(-1)
			return
		}
		sh.mu.Unlock()
		return
	}

	var rawConn syscall.RawConn
	if sc, ok := pc.conn.(syscall.Conn); ok {
		if rc, err := sc.SyscallConn(); err == nil {
			rawConn = rc
		}
	}
	if rawConn == nil {
		sh.mu.Lock()
		if sh.entries[pc] == e && e.state == clientIdlePend {
			delete(sh.entries, pc)
			delete(sh.pending, pc)
			e.state = clientIdleDisarm
			e.disarmed.Store(true)
			sh.mu.Unlock()
			httpClientIdleMgr.pendingCount.Add(-1)
			httpClientIdleMgr.totalArmed.Add(-1)
			return
		}
		sh.mu.Unlock()
		return
	}

	fd, hasData, closed, armed, _ := peekAndArmSocket(rawConn, p)
	if closed || hasData {
		// An idle HTTP/1.1 client connection receiving EOF, error, or
		// unsolicited bytes (such as 408 Request Timeout) is dead.
		sh.mu.Lock()
		if sh.entries[pc] == e && e.state == clientIdlePend {
			delete(sh.entries, pc)
			delete(sh.pending, pc)
			e.state = clientIdleDisarm
			e.disarmed.Store(true)
			sh.mu.Unlock()
			httpClientIdleMgr.pendingCount.Add(-1)
			httpClientIdleMgr.totalArmed.Add(-1)
			pc.close(errServerClosedIdle)
			pc.t.removeIdleConn(pc)
			return
		}
		sh.mu.Unlock()
		return
	}
	if armed && fd >= 0 {
		sh.mu.Lock()
		if sh.entries[pc] == e && e.state == clientIdlePend {
			delete(sh.pending, pc)
			e.rawConn = rawConn
			e.fd = fd
			e.state = clientIdlePoll
			byFD[fd] = e
			sh.mu.Unlock()
			httpClientIdleMgr.pendingCount.Add(-1)
			return
		}
		sh.mu.Unlock()
		_ = rawConn.Control(func(u uintptr) {
			p.del(int(u))
		})
		return
	}

	sh.mu.Lock()
	if sh.entries[pc] == e && e.state == clientIdlePend {
		delete(sh.entries, pc)
		delete(sh.pending, pc)
		e.state = clientIdleDisarm
		e.disarmed.Store(true)
		sh.mu.Unlock()
		httpClientIdleMgr.pendingCount.Add(-1)
		httpClientIdleMgr.totalArmed.Add(-1)
		return
	}
	sh.mu.Unlock()
}

func handleReadyClientIdleEntry(p *httpBgPoller, e *clientIdleEntry, byFD map[int]*clientIdleEntry) {
	if e.disarmed.Load() {
		return
	}
	pc := e.pc
	if pc == nil {
		return
	}
	sh := clientIdleShardFor(pc)
	sh.mu.Lock()
	if sh.entries[pc] != e || e.state != clientIdlePoll {
		sh.mu.Unlock()
		return
	}
	rawConn := e.rawConn
	sh.mu.Unlock()

	fd, hasData, closed, armed, _ := peekAndArmSocket(rawConn, p)
	if armed && fd >= 0 {
		byFD[fd] = e
		return
	}
	sh.mu.Lock()
	if sh.entries[pc] == e && e.state == clientIdlePoll {
		delete(sh.entries, pc)
		e.state = clientIdleDisarm
		e.disarmed.Store(true)
		sh.mu.Unlock()
		httpClientIdleMgr.totalArmed.Add(-1)
	} else {
		sh.mu.Unlock()
		return
	}
	if closed || hasData {
		pc.close(errServerClosedIdle)
		pc.t.removeIdleConn(pc)
	}
}

func canUseSyncPersistConn(pconn *persistConn) bool {
	if pconn.isClientConn || pconn.tlsState != nil {
		return false
	}
	if _, isTLS := pconn.conn.(*tls.Conn); isTLS {
		return false
	}
	sc, ok := pconn.conn.(syscall.Conn)
	if !ok {
		return false
	}
	if _, err := sc.SyscallConn(); err != nil {
		return false
	}
	return ensureHTTPClientIdleWatcher()
}

func (pc *persistConn) promoteToAsync() {
	pc.reqch = make(chan requestAndChan, 1)
	pc.writech = make(chan writeRequest, 1)
	pc.writeErrCh = make(chan error, 1)
	pc.writeLoopDone = make(chan struct{})
	go pc.readLoop()
	go pc.writeLoop()
}

func (pc *persistConn) syncPutIdleConn(treq *transportRequest) {
	// Never return a connection with unread/unsolicited bytes already buffered
	// in pc.br (matches upstream readLoopPeekFailLocked response-desync guard).
	if pc.br.Buffered() != 0 {
		pc.close(errServerClosedIdle)
		return
	}
	trace := treq.trace
	httpClientIdleArm(pc)
	if err := pc.t.tryPutIdleConn(pc); err != nil {
		httpClientIdleDisarm(pc)
		if trace != nil && trace.PutIdleConn != nil && err != errKeepAlivesDisabled {
			trace.PutIdleConn(err)
		}
		pc.close(err)
		return
	}
	if trace != nil && trace.PutIdleConn != nil {
		trace.PutIdleConn(nil)
	}
}

func (pc *persistConn) mapSyncRoundTripError(req *transportRequest, startBytesWritten int64, err error) error {
	if err == nil {
		return nil
	}
	if cerr := pc.canceled(); cerr != nil {
		return cerr
	}
	req.mu.Lock()
	reqErr := req.err
	req.mu.Unlock()
	if reqErr != nil {
		return reqErr
	}
	if err == errServerClosedIdle {
		if pc.nwrite == startBytesWritten {
			return nothingWrittenError{err}
		}
		return err
	}
	if _, ok := err.(transportReadFromServerError); ok {
		if pc.nwrite == startBytesWritten {
			return nothingWrittenError{err}
		}
		return err
	}
	if pc.isBroken() {
		if pc.nwrite == startBytesWritten {
			return nothingWrittenError{err}
		}
		return fmt.Errorf("net/http: HTTP/1.x transport connection broken: %w", err)
	}
	return err
}

func (pc *persistConn) readResponseSync(treq *transportRequest, trace *httptrace.ClientTrace) (resp *Response, err error) {
	for {
		resp, err = ReadResponse(pc.br, treq.Request)
		if err != nil {
			return nil, err
		}
		resCode := resp.StatusCode
		is1xx := 100 <= resCode && resCode <= 199
		is1xxNonTerminal := is1xx && resCode != StatusSwitchingProtocols
		if is1xxNonTerminal {
			if trace != nil && trace.Got1xxResponse != nil {
				if err := trace.Got1xxResponse(resCode, textproto.MIMEHeader(resp.Header)); err != nil {
					return nil, err
				}
				pc.readLimit = pc.maxHeaderResponseSize()
			}
			continue
		}
		break
	}
	if resp.isProtocolSwitch() {
		resp.Body = newReadWriteCloserBody(pc.br, pc.conn)
	}
	resp.TLS = pc.tlsState
	return resp, nil
}

func (pc *persistConn) roundTripSync(req *transportRequest) (*Response, error) {
	testHookEnterRoundTrip()

	pc.mu.Lock()
	if pc.closed != nil {
		closedErr := pc.closed
		pc.mu.Unlock()
		return nil, pc.mapSyncRoundTripError(req, pc.nwrite, closedErr)
	}
	headerFn := pc.mutateHeaderFunc
	pc.mu.Unlock()

	if headerFn != nil {
		headerFn(req.extraHeaders())
	}

	requestedGzip := false
	if !pc.t.DisableCompression &&
		req.Header.Get("Accept-Encoding") == "" &&
		req.Header.Get("Range") == "" &&
		req.Method != "HEAD" {
		requestedGzip = true
		req.extraHeaders().Set("Accept-Encoding", "gzip")
	}

	if pc.t.DisableKeepAlives &&
		!req.wantsClose() &&
		!isProtocolSwitchHeader(req.Header) {
		req.extraHeaders().Set("Connection", "close")
	}

	if err := req.ctx.Err(); err != nil {
		cause := context.Cause(req.ctx)
		pc.cancelRequest(cause)
		return nil, cause
	}
	stopCancel := context.AfterFunc(req.ctx, func() {
		pc.cancelRequest(context.Cause(req.ctx))
	})

	startBytesWritten := pc.nwrite
	err := req.Request.write(pc.bw, pc.isProxy, req.extra, nil)
	if bre, ok := err.(requestBodyReadError); ok {
		err = bre.error
		req.setError(err)
	}
	if err == nil {
		err = pc.bw.Flush()
	}
	if err != nil {
		stopCancel()
		if pc.nwrite == startBytesWritten {
			err = nothingWrittenError{err}
		}
		pc.close(fmt.Errorf("write error: %w", err))
		return nil, pc.mapSyncRoundTripError(req, startBytesWritten, err)
	}

	pc.readLimit = pc.maxHeaderResponseSize()
	respHeaderTO := pc.t.ResponseHeaderTimeout
	if respHeaderTO > 0 {
		pc.conn.SetReadDeadline(time.Now().Add(respHeaderTO))
	}
	// Match upstream readLoop: only a failure on the initial 1-byte peek is
	// classified as transportReadFromServerError (retryable on a reused conn).
	// Subsequent ReadResponse / Got1xxResponse errors stay terminal.
	var resp *Response
	_, err = pc.br.Peek(1)
	if err == nil {
		if req.trace != nil && req.trace.GotFirstResponseByte != nil {
			req.trace.GotFirstResponseByte()
		}
		resp, err = pc.readResponseSync(req, req.trace)
	} else if !(respHeaderTO > 0 && os.IsTimeout(err) && pc.canceled() == nil) {
		err = transportReadFromServerError{err}
	}
	if respHeaderTO > 0 {
		pc.conn.SetReadDeadline(time.Time{})
	}
	if err != nil {
		stopCancel()
		if respHeaderTO > 0 && os.IsTimeout(err) && pc.canceled() == nil {
			pc.close(errTimeout)
			return nil, errTimeout
		}
		if pc.readLimit <= 0 {
			err = fmt.Errorf("net/http: server response headers exceeded %d bytes; aborted", pc.maxHeaderResponseSize())
		}
		pc.close(err)
		return nil, pc.mapSyncRoundTripError(req, startBytesWritten, err)
	}
	pc.readLimit = maxInt64

	bodyWritable := resp.bodyIsWritable()
	isConnect := req.Request.Method == "CONNECT"
	hasBody := req.Request.Method != "HEAD" && resp.ContentLength != 0
	alive := !(resp.Close || req.Request.Close || resp.StatusCode <= 199 || bodyWritable || isConnect)

	if !hasBody || bodyWritable {
		stopCancel()
		if alive && !pc.sawEOF {
			pc.syncPutIdleConn(req)
		} else if bodyWritable {
			pc.close(errCallerOwnsConn)
		} else {
			pc.close(errReadLoopExiting)
		}
		req.cancel(errRequestDone)
		return resp, nil
	}

	origBody := resp.Body
	body := &bodyEOFSignal{
		body: origBody,
		earlyCloseFn: func() error {
			stopCancel()
			req.cancel(errRequestDone)
			if alive && resp.ContentLength <= maxPostCloseReadBytes {
				// Return immediately to the caller of Body.Close (matching
				// upstream readLoop, which unblocks eofc before maybeDrainBody)
				// and finish draining or closing the connection asynchronously.
				go func() {
					if maybeDrainBody(origBody) && !pc.sawEOF {
						pc.syncPutIdleConn(req)
					} else {
						pc.close(errSyncClosedEarly)
					}
				}()
			} else {
				pc.close(errSyncClosedEarly)
			}
			return nil
		},
		fn: func(err error) error {
			stopCancel()
			req.cancel(errRequestDone)
			if err == io.EOF {
				if alive && !pc.sawEOF {
					pc.syncPutIdleConn(req)
				} else {
					pc.close(errReadLoopExiting)
				}
			} else if err != nil {
				cerr := pc.canceled()
				pc.close(errReadLoopExiting)
				if cerr != nil {
					return cerr
				}
			} else {
				pc.close(errReadLoopExiting)
			}
			return err
		},
	}

	resp.Body = body
	if requestedGzip && ascii.EqualFold(resp.Header.Get("Content-Encoding"), "gzip") {
		resp.Body = &gzipReader{body: body}
		resp.Header.Del("Content-Encoding")
		resp.Header.Del("Content-Length")
		resp.ContentLength = -1
		resp.Uncompressed = true
	}
	return resp, nil
}

func (pc *persistConn) roundTrip(req *transportRequest) (resp *Response, err error) {
	if pc.writech == nil {
		httpClientIdleDisarm(pc)
		outLen := req.Request.outgoingLength()
		needAsync := (req.ProtoAtLeast(1, 1) && req.Body != nil && req.expectsContinue()) ||
			outLen < 0 || outLen > clientIdleMaxSyncBody ||
			!isInMemoryRequestBody(req.Body) ||
			(req.trace != nil && (req.trace.WroteHeaders != nil || req.trace.WroteRequest != nil || req.trace.Wait100Continue != nil || req.trace.Got100Continue != nil))
		if !needAsync {
			return pc.roundTripSync(req)
		}
		pc.promoteToAsync()
	}

	testHookEnterRoundTrip()

	pc.mu.Lock()
	if pc.isClientConn {
		if !pc.reserved {
			pc.mu.Unlock()
			if err := pc.waitForAvailability(req.ctx); err != nil {
				return nil, err
			}
			pc.mu.Lock()
		}
		pc.reserved = false
		pc.inFlight = true
	}
	pc.numExpectedResponses++
	headerFn := pc.mutateHeaderFunc
	pc.mu.Unlock()

	if headerFn != nil {
		headerFn(req.extraHeaders())
	}

	requestedGzip := false
	if !pc.t.DisableCompression &&
		req.Header.Get("Accept-Encoding") == "" &&
		req.Header.Get("Range") == "" &&
		req.Method != "HEAD" {
		requestedGzip = true
		req.extraHeaders().Set("Accept-Encoding", "gzip")
	}

	var continueCh chan struct{}
	if req.ProtoAtLeast(1, 1) && req.Body != nil && req.expectsContinue() {
		continueCh = make(chan struct{}, 1)
	}

	if pc.t.DisableKeepAlives &&
		!req.wantsClose() &&
		!isProtocolSwitchHeader(req.Header) {
		req.extraHeaders().Set("Connection", "close")
	}

	gone := make(chan struct{})
	defer close(gone)

	startBytesWritten := pc.nwrite
	writeErrCh := make(chan error, 1)
	pc.writech <- writeRequest{req, writeErrCh, continueCh}

	resc := make(chan responseAndError)
	pc.reqch <- requestAndChan{
		treq:       req,
		ch:         resc,
		addedGzip:  requestedGzip,
		continueCh: continueCh,
		callerGone: gone,
	}

	handleResponse := func(re responseAndError) (*Response, error) {
		if (re.res == nil) == (re.err == nil) {
			panic(fmt.Sprintf("internal error: exactly one of res or err should be set; nil=%v", re.res == nil))
		}
		if re.err != nil {
			return nil, pc.mapRoundTripError(req, startBytesWritten, re.err)
		}
		return re.res, nil
	}

	var respHeaderTimer <-chan time.Time
	ctxDoneChan := req.ctx.Done()
	pcClosed := pc.closech
	for {
		testHookWaitResLoop()
		select {
		case err := <-writeErrCh:
			if err != nil {
				pc.close(fmt.Errorf("write error: %w", err))
				return nil, pc.mapRoundTripError(req, startBytesWritten, err)
			}
			if d := pc.t.ResponseHeaderTimeout; d > 0 {
				timer := time.NewTimer(d)
				defer timer.Stop()
				respHeaderTimer = timer.C
			}
		case <-pcClosed:
			select {
			case re := <-resc:
				return handleResponse(re)
			default:
			}
			return nil, pc.mapRoundTripError(req, startBytesWritten, pc.closed)
		case <-respHeaderTimer:
			pc.close(errTimeout)
			return nil, errTimeout
		case re := <-resc:
			return handleResponse(re)
		case <-ctxDoneChan:
			select {
			case re := <-resc:
				return handleResponse(re)
			default:
			}
			pc.cancelRequest(context.Cause(req.ctx))
		}
	}
}

func (pc *persistConn) closeLocked(err error) {
	if err == nil {
		panic("nil error")
	}
	if pc.closed == nil {
		pc.closed = err
		pc.t.decConnsPerHost(pc.cacheKey)
		if pc.alt == nil {
			if pc.writech == nil {
				httpClientIdleDisarm(pc)
			}
			if err != errCallerOwnsConn {
				pc.conn.Close()
			}
			close(pc.closech)
		} else {
			if cc, ok := pc.alt.(io.Closer); ok {
				cc.Close()
			}
		}
	}
	pc.mutateHeaderFunc = nil
}

func (t *Transport) tryPopIdleConnFast(key connectMethodKey) (*persistConn, time.Time, bool) {
	t.idleMu.Lock()
	t.closeIdle = false
	list, ok := t.idleConn[key]
	if !ok || len(list) == 0 {
		t.idleMu.Unlock()
		return nil, time.Time{}, false
	}
	var oldTime time.Time
	if t.IdleConnTimeout > 0 {
		oldTime = time.Now().Add(-t.IdleConnTimeout)
	}
	var got *persistConn
	var idleAt time.Time
	for len(list) > 0 {
		pconn := list[len(list)-1]
		tooOld := !oldTime.IsZero() && pconn.idleAt.Round(0).Before(oldTime)
		if tooOld {
			go pconn.closeConnIfStillIdle()
		}
		if pconn.isBroken() || tooOld {
			list = list[:len(list)-1]
			continue
		}
		got = pconn
		idleAt = pconn.idleAt
		if pconn.alt == nil {
			t.idleLRU.remove(pconn)
			list = list[:len(list)-1]
		}
		break
	}
	if len(list) > 0 {
		t.idleConn[key] = list
	} else {
		delete(t.idleConn, key)
	}
	t.idleMu.Unlock()
	if got == nil {
		return nil, time.Time{}, false
	}
	return got, idleAt, true
}

func (t *Transport) getConn(treq *transportRequest, cm connectMethod) (_ *persistConn, err error) {
	req := treq.Request
	trace := treq.trace
	ctx := req.Context()
	if trace != nil && trace.GetConn != nil {
		trace.GetConn(cm.addr())
	}

	if !t.DisableKeepAlives {
		select {
		case <-treq.ctx.Done():
			err := context.Cause(treq.ctx)
			if err == errRequestCanceled {
				err = errRequestCanceledConn
			}
			return nil, err
		default:
		}
		if pc, idleAt, ok := t.tryPopIdleConnFast(cm.key()); ok {
			if pc.alt == nil && trace != nil && trace.GotConn != nil {
				info := httptrace.GotConnInfo{
					Conn:   pc.conn,
					Reused: pc.isReused(),
				}
				if !idleAt.IsZero() {
					info.WasIdle = true
					info.IdleTime = time.Since(idleAt)
				}
				trace.GotConn(info)
			}
			return pc, nil
		}
	}

	dialCtx, dialCancel := context.WithCancel(context.WithoutCancel(ctx))

	w := &wantConn{
		cm:         cm,
		key:        cm.key(),
		ctx:        dialCtx,
		cancelCtx:  dialCancel,
		result:     make(chan connOrError, 1),
		beforeDial: testHookPrePendingDial,
		afterDial:  testHookPostPendingDial,
	}
	defer func() {
		if err != nil {
			w.cancel(t)
		}
	}()

	if delivered := t.queueForIdleConn(w); !delivered {
		t.queueForDial(w)
	}

	select {
	case r := <-w.result:
		if r.pc != nil && r.pc.alt == nil && trace != nil && trace.GotConn != nil {
			info := httptrace.GotConnInfo{
				Conn:   r.pc.conn,
				Reused: r.pc.isReused(),
			}
			if !r.idleAt.IsZero() {
				info.WasIdle = true
				info.IdleTime = time.Since(r.idleAt)
			}
			trace.GotConn(info)
		}
		if r.err != nil {
			select {
			case <-treq.ctx.Done():
				err := context.Cause(treq.ctx)
				if err == errRequestCanceled {
					err = errRequestCanceledConn
				}
				return nil, err
			default:
			}
		}
		return r.pc, r.err
	case <-treq.ctx.Done():
		err := context.Cause(treq.ctx)
		if err == errRequestCanceled {
			err = errRequestCanceledConn
		}
		return nil, err
	}
}

func (t *Transport) dialConn(ctx context.Context, cm connectMethod, isClientConn bool, internalStateHook func()) (pconn *persistConn, err error) {
	if p := t.protocols(); p.http3() {
		if p.HTTP1() || p.HTTP2() || p.UnencryptedHTTP2() {
			return nil, errors.New("http: when using HTTP3, Transport.Protocols must contain only HTTP3")
		}
		if t.h3Transport == nil {
			return nil, errors.New("http: Transport.Protocols contains HTTP3, but Transport does not support HTTP/3")
		}
		rt, err := t.h3Transport.DialClientConn(ctx, cm.addr(), cm.proxyURL, internalStateHook)
		if err != nil {
			return nil, err
		}
		return &persistConn{
			t:        t,
			cacheKey: cm.key(),
			alt:      rt,
		}, nil
	}

	pconn = &persistConn{
		t:                 t,
		cacheKey:          cm.key(),
		closech:           make(chan struct{}),
		isClientConn:      isClientConn,
		internalStateHook: internalStateHook,
	}
	trace := httptrace.ContextClientTrace(ctx)
	wrapErr := func(err error) error {
		if cm.proxyURL != nil {
			return &net.OpError{Op: "proxyconnect", Net: "tcp", Err: err}
		}
		return err
	}

	if rt, err := t.http2ExternalDial(ctx, cm); err != errors.ErrUnsupported {
		if err != nil {
			return nil, err
		}
		return &persistConn{t: t, cacheKey: pconn.cacheKey, alt: rt}, nil
	}

	if cm.scheme() == "https" && t.hasCustomTLSDialer() {
		var err error
		pconn.conn, err = t.customDialTLS(ctx, "tcp", cm.addr())
		if err != nil {
			return nil, wrapErr(err)
		}
		type connectionStater interface {
			ConnectionState() tls.ConnectionState
		}
		type handshaker interface {
			HandshakeContext(context.Context) error
		}
		if cstater, ok := pconn.conn.(connectionStater); ok {
			if trace != nil && trace.TLSHandshakeStart != nil {
				trace.TLSHandshakeStart()
			}
			if handshaker, ok := cstater.(handshaker); ok {
				if err := handshaker.HandshakeContext(ctx); err != nil {
					go pconn.conn.Close()
					if trace != nil && trace.TLSHandshakeDone != nil {
						trace.TLSHandshakeDone(tls.ConnectionState{}, err)
					}
					return nil, err
				}
			}
			cs := cstater.ConnectionState()
			if trace != nil && trace.TLSHandshakeDone != nil {
				trace.TLSHandshakeDone(cs, nil)
			}
			pconn.tlsState = &cs
		}
	} else {
		conn, err := t.dial(ctx, "tcp", cm.addr())
		if err != nil {
			return nil, wrapErr(err)
		}
		pconn.conn = conn
		if cm.scheme() == "https" {
			var firstTLSHost string
			if firstTLSHost, _, err = net.SplitHostPort(cm.addr()); err != nil {
				return nil, wrapErr(err)
			}
			if err = pconn.addTLS(ctx, firstTLSHost, trace); err != nil {
				return nil, wrapErr(err)
			}
		}
	}

	switch {
	case cm.proxyURL == nil:
	case cm.proxyURL.Scheme == "socks5" || cm.proxyURL.Scheme == "socks5h":
		conn := pconn.conn
		d := socksNewDialer("tcp", conn.RemoteAddr().String())
		if u := cm.proxyURL.User; u != nil {
			auth := &socksUsernamePassword{
				Username: u.Username(),
			}
			auth.Password, _ = u.Password()
			d.AuthMethods = []socksAuthMethod{
				socksAuthMethodNotRequired,
				socksAuthMethodUsernamePassword,
			}
			d.Authenticate = auth.Authenticate
		}
		if _, err := d.DialWithConn(ctx, conn, "tcp", cm.targetAddr); err != nil {
			conn.Close()
			return nil, err
		}
	case cm.targetScheme == "http":
		pconn.isProxy = true
		if pa := cm.proxyAuth(); pa != "" {
			pconn.mutateHeaderFunc = func(h Header) {
				h.Set("Proxy-Authorization", pa)
			}
		}
	case cm.targetScheme == "https":
		conn := pconn.conn
		var hdr Header
		if t.GetProxyConnectHeader != nil {
			var err error
			hdr, err = t.GetProxyConnectHeader(ctx, cm.proxyURL, cm.targetAddr)
			if err != nil {
				conn.Close()
				return nil, err
			}
		} else {
			hdr = t.ProxyConnectHeader
		}
		if hdr == nil {
			hdr = make(Header)
		}
		if pa := cm.proxyAuth(); pa != "" {
			hdr = hdr.Clone()
			hdr.Set("Proxy-Authorization", pa)
		}
		connectReq := &Request{
			Method: "CONNECT",
			URL:    &url.URL{Opaque: cm.targetAddr},
			Host:   cm.targetAddr,
			Header: hdr,
		}

		connectCtx, cancel := testHookProxyConnectTimeout(ctx, 1*time.Minute)
		defer cancel()

		didReadResponse := make(chan struct{})
		var (
			resp *Response
			err  error
		)
		go func() {
			defer close(didReadResponse)
			err = connectReq.Write(conn)
			if err != nil {
				return
			}
			br := bufio.NewReader(&io.LimitedReader{R: conn, N: t.maxHeaderResponseSize()})
			resp, err = ReadResponse(br, connectReq)
		}()
		select {
		case <-connectCtx.Done():
			conn.Close()
			<-didReadResponse
			return nil, connectCtx.Err()
		case <-didReadResponse:
		}
		if err != nil {
			conn.Close()
			return nil, err
		}

		if t.OnProxyConnectResponse != nil {
			err = t.OnProxyConnectResponse(ctx, cm.proxyURL, connectReq, resp)
			if err != nil {
				conn.Close()
				return nil, err
			}
		}

		if resp.StatusCode != 200 {
			_, text, ok := strings.Cut(resp.Status, " ")
			conn.Close()
			if !ok {
				return nil, errors.New("unknown status code")
			}
			return nil, errors.New(text)
		}
	}

	if cm.proxyURL != nil && cm.targetScheme == "https" {
		if err := pconn.addTLS(ctx, cm.tlsHost(), trace); err != nil {
			return nil, err
		}
	}

	unencryptedHTTP2 := pconn.tlsState == nil &&
		t.Protocols != nil &&
		t.Protocols.UnencryptedHTTP2() &&
		!t.Protocols.HTTP1()

	http2 := unencryptedHTTP2 ||
		(pconn.tlsState != nil && pconn.tlsState.NegotiatedProtocol == "h2")

	if http2 && t.h2Transport != nil {
		if isClientConn {
			cc, err := t.http2NewClientConn(pconn.conn, internalStateHook)
			if err == nil {
				return &persistConn{t: t, cacheKey: pconn.cacheKey, alt: cc, isClientConn: true}, nil
			}
			if err != errors.ErrUnsupported {
				return nil, err
			}
		} else {
			rt, err := t.http2AddConn(cm.targetScheme, cm.targetAddr, pconn.conn)
			if err == nil {
				return &persistConn{t: t, cacheKey: pconn.cacheKey, alt: rt}, nil
			}
			if err != errors.ErrUnsupported {
				return nil, err
			}
		}
	}

	if isClientConn && (unencryptedHTTP2 || (pconn.tlsState != nil && pconn.tlsState.NegotiatedProtocol == "h2")) {
		altProto, _ := t.altProto.Load().(map[string]RoundTripper)
		h2, ok := altProto["https"].(newClientConner)
		if !ok {
			return nil, errors.New("http: HTTP/2 implementation does not support NewClientConn (update golang.org/x/net?)")
		}
		alt, err := h2.NewClientConn(pconn.conn, internalStateHook)
		if err != nil {
			pconn.conn.Close()
			return nil, err
		}
		return &persistConn{t: t, cacheKey: pconn.cacheKey, alt: alt, isClientConn: true}, nil
	}

	if unencryptedHTTP2 {
		next, ok := t.TLSNextProto[nextProtoUnencryptedHTTP2]
		if !ok {
			return nil, errors.New("http: Transport does not support unencrypted HTTP/2")
		}
		alt := next(cm.targetAddr, unencryptedTLSConn(pconn.conn))
		if e, ok := alt.(erringRoundTripper); ok {
			return nil, e.RoundTripErr()
		}
		return &persistConn{t: t, cacheKey: pconn.cacheKey, alt: alt}, nil
	}

	if s := pconn.tlsState; s != nil && s.NegotiatedProtocolIsMutual && s.NegotiatedProtocol != "" {
		tlsConn, tlsConnOK := pconn.conn.(*tls.Conn)
		if next, ok := t.TLSNextProto[s.NegotiatedProtocol]; tlsConnOK && ok {
			alt := next(cm.targetAddr, tlsConn)
			if e, ok := alt.(erringRoundTripper); ok {
				return nil, e.RoundTripErr()
			}
			return &persistConn{t: t, cacheKey: pconn.cacheKey, alt: alt}, nil
		}
	}

	pconn.br = bufio.NewReaderSize(pconn, t.readBufferSize())
	pconn.bw = bufio.NewWriterSize(persistConnWriter{pconn}, t.writeBufferSize())

	if canUseSyncPersistConn(pconn) {
		return pconn, nil
	}

	pconn.reqch = make(chan requestAndChan, 1)
	pconn.writech = make(chan writeRequest, 1)
	pconn.writeErrCh = make(chan error, 1)
	pconn.writeLoopDone = make(chan struct{})
	go pconn.readLoop()
	go pconn.writeLoop()
	return pconn, nil
}
