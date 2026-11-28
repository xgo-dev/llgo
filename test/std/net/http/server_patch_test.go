//go:build darwin || linux

package http_test

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"net/textproto"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

func TestServerBackgroundReadDisconnectPrompt(t *testing.T) {
	for _, delay := range []time.Duration{0, 10 * time.Millisecond} {
		entered := make(chan struct{})
		cancelled := make(chan struct{})
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			close(entered)
			<-r.Context().Done()
			close(cancelled)
		}))
		conn, err := net.Dial("tcp", srv.Listener.Addr().String())
		if err != nil {
			srv.Close()
			t.Fatalf("dial: %v", err)
		}
		if _, err := io.WriteString(conn, "GET /slow HTTP/1.1\r\nHost: localhost\r\n\r\n"); err != nil {
			conn.Close()
			srv.Close()
			t.Fatalf("write: %v", err)
		}
		<-entered
		if delay > 0 {
			time.Sleep(delay)
		}
		conn.Close()
		select {
		case <-cancelled:
		case <-time.After(1 * time.Second):
			srv.Close()
			t.Fatalf("delay=%v: handler did not observe client TCP close promptly", delay)
		}
		srv.Close()
	}
}

func TestServerBackgroundReadPipelinedRequests(t *testing.T) {
	var reqCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := reqCount.Add(1)
		if n == 1 {
			// Stay in handler past the background-read promotion tick so the
			// watcher peeks the pipelined second request without canceling ctx.
			time.Sleep(15 * time.Millisecond)
			if err := r.Context().Err(); err != nil {
				http.Error(w, "unexpected cancel: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}
		io.WriteString(w, "ok:"+r.URL.Path)
	}))
	defer srv.Close()

	conn, err := net.Dial("tcp", srv.Listener.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))

	raw := "GET /one HTTP/1.1\r\nHost: localhost\r\n\r\n" +
		"GET /two HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"
	if _, err := io.WriteString(conn, raw); err != nil {
		t.Fatalf("write pipelined: %v", err)
	}
	br := bufio.NewReader(conn)
	for _, want := range []string{"ok:/one", "ok:/two"} {
		resp, err := http.ReadResponse(br, nil)
		if err != nil {
			t.Fatalf("ReadResponse(%s): %v", want, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK || string(body) != want {
			t.Fatalf("got status=%d body=%q, want 200 %q", resp.StatusCode, string(body), want)
		}
	}
}

func TestServerBackgroundReadHijackClean(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "no hijack", http.StatusInternalServerError)
			return
		}
		c, rw, err := hj.Hijack()
		if err != nil {
			return
		}
		defer c.Close()
		io.WriteString(rw, "HTTP/1.1 101 Switching Protocols\r\n\r\nHIJACKED")
		rw.Flush()
	}))
	defer srv.Close()

	conn, err := net.Dial("tcp", srv.Listener.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	if _, err := io.WriteString(conn, "GET /hj HTTP/1.1\r\nHost: localhost\r\n\r\n"); err != nil {
		t.Fatalf("write: %v", err)
	}
	data, err := io.ReadAll(conn)
	if err != nil {
		t.Fatalf("read hijacked: %v", err)
	}
	if !strings.Contains(string(data), "HIJACKED") {
		t.Fatalf("unexpected hijacked response: %q", string(data))
	}
}

type pipeListener struct {
	ch     chan net.Conn
	closed chan struct{}
	once   sync.Once
}

func newPipeListener() *pipeListener {
	return &pipeListener{ch: make(chan net.Conn, 4), closed: make(chan struct{})}
}

func (l *pipeListener) Accept() (net.Conn, error) {
	select {
	case c := <-l.ch:
		return c, nil
	case <-l.closed:
		return nil, net.ErrClosed
	}
}

func (l *pipeListener) Close() error {
	l.once.Do(func() { close(l.closed) })
	return nil
}

func (l *pipeListener) Addr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1}
}

func TestServerBackgroundReadTLSAndPipeFallback(t *testing.T) {
	// 1. TLS fallback (non-raw-socket background read)
	entered, cancelled := make(chan struct{}), make(chan struct{})
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-r.Context().Done()
		close(cancelled)
	}))
	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL, nil)
	go func() {
		resp, err := ts.Client().Do(req)
		if err == nil {
			resp.Body.Close()
		}
	}()
	<-entered
	time.Sleep(10 * time.Millisecond)
	cancel()
	select {
	case <-cancelled:
	case <-time.After(1 * time.Second):
		ts.Close()
		t.Fatal("TLS handler did not observe client disconnect promptly")
	}
	ts.Close()

	// 2. net.Pipe fallback (non-syscall.Conn)
	ln := newPipeListener()
	pEntered, pCancelled := make(chan struct{}), make(chan struct{})
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(pEntered)
		<-r.Context().Done()
		close(pCancelled)
	})}
	go srv.Serve(ln)
	defer srv.Close()

	cSide, sSide := net.Pipe()
	ln.ch <- sSide
	go func() {
		_, _ = io.WriteString(cSide, "GET /pipe HTTP/1.1\r\nHost: localhost\r\n\r\n")
	}()
	<-pEntered
	time.Sleep(10 * time.Millisecond)
	cSide.Close()
	select {
	case <-pCancelled:
	case <-time.After(1 * time.Second):
		t.Fatal("net.Pipe handler did not observe client close promptly")
	}
}

func TestServerIdleKeepAliveOffloadAndShutdown(t *testing.T) {
	const numIdle = 120
	var closedCount atomic.Int32
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, "pong:"+r.URL.Path)
		}),
		ConnState: func(_ net.Conn, st http.ConnState) {
			if st == http.StateClosed {
				closedCount.Add(1)
			}
		},
	}
	go srv.Serve(ln)

	baseG := runtime.NumGoroutine()
	conns := make([]net.Conn, 0, numIdle)
	readers := make([]*bufio.Reader, 0, numIdle)
	defer func() {
		for _, c := range conns {
			c.Close()
		}
	}()

	for i := 0; i < numIdle; i++ {
		c, err := net.Dial("tcp", ln.Addr().String())
		if err != nil {
			t.Fatalf("dial %d: %v", i, err)
		}
		_ = c.SetDeadline(time.Now().Add(5 * time.Second))
		if _, err := io.WriteString(c, "GET /first HTTP/1.1\r\nHost: localhost\r\n\r\n"); err != nil {
			t.Fatalf("write %d: %v", i, err)
		}
		br := bufio.NewReader(c)
		resp, err := http.ReadResponse(br, nil)
		if err != nil {
			t.Fatalf("read first %d: %v", i, err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		conns = append(conns, c)
		readers = append(readers, br)
	}

	// On LLGo, idle keep-alive connections offload to the shared kqueue/epoll
	// watcher after the hot-grace window (~150ms) instead of pinning one OS
	// thread per idle socket.
	if runtime.Compiler == "llgo" {
		deadline := time.Now().Add(1500 * time.Millisecond)
		var afterG int
		for time.Now().Before(deadline) {
			afterG = runtime.NumGoroutine()
			if afterG < baseG+40 {
				break
			}
			time.Sleep(15 * time.Millisecond)
		}
		if afterG >= baseG+40 {
			t.Fatalf("expected idle keep-alive offload: goroutines=%d (baseline=%d, idleConns=%d)", afterG, baseG, numIdle)
		}
	}

	// Wake 10 parked connections and verify they cleanly serve another request.
	for i := 0; i < 10; i++ {
		idx := i * 11
		if _, err := io.WriteString(conns[idx], "GET /second HTTP/1.1\r\nHost: localhost\r\n\r\n"); err != nil {
			t.Fatalf("write second %d: %v", idx, err)
		}
		resp, err := http.ReadResponse(readers[idx], nil)
		if err != nil {
			t.Fatalf("read second %d: %v", idx, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if string(body) != "pong:/second" {
			t.Fatalf("second %d body=%q, want pong:/second", idx, string(body))
		}
	}

	// Graceful shutdown must close all idle parked connections promptly and fire StateClosed.
	shutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	waitClose := time.Now().Add(1 * time.Second)
	for closedCount.Load() < numIdle && time.Now().Before(waitClose) {
		time.Sleep(5 * time.Millisecond)
	}
	if got := closedCount.Load(); got != numIdle {
		t.Fatalf("StateClosed count = %d, want %d", got, numIdle)
	}
}

func TestServerIdleTimeoutExpiresParkedConns(t *testing.T) {
	const numIdle = 20
	var closedCount atomic.Int32
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := &http.Server{
		IdleTimeout: 40 * time.Millisecond,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, "ok")
		}),
		ConnState: func(_ net.Conn, st http.ConnState) {
			if st == http.StateClosed {
				closedCount.Add(1)
			}
		},
	}
	go srv.Serve(ln)
	defer srv.Close()

	conns := make([]net.Conn, 0, numIdle)
	defer func() {
		for _, c := range conns {
			c.Close()
		}
	}()
	for i := 0; i < numIdle; i++ {
		c, err := net.Dial("tcp", ln.Addr().String())
		if err != nil {
			t.Fatalf("dial %d: %v", i, err)
		}
		if _, err := io.WriteString(c, "GET / HTTP/1.1\r\nHost: localhost\r\n\r\n"); err != nil {
			t.Fatalf("write %d: %v", i, err)
		}
		br := bufio.NewReader(c)
		resp, err := http.ReadResponse(br, nil)
		if err != nil {
			t.Fatalf("read %d: %v", i, err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		conns = append(conns, c)
	}

	deadline := time.Now().Add(1 * time.Second)
	for closedCount.Load() < numIdle && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if got := closedCount.Load(); got != numIdle {
		t.Fatalf("IdleTimeout closed %d/%d parked conns", got, numIdle)
	}
}

func TestServerElasticWorkerPoolNoSelfCallDeadlock(t *testing.T) {
	const concurrency = 40
	var outerReady atomic.Int32
	allReady := make(chan struct{})
	var baseURL string
	client := &http.Client{
		Timeout: 3 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        concurrency * 2,
			MaxIdleConnsPerHost: concurrency * 2,
		},
	}
	defer client.CloseIdleConnections()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	baseURL = "http://" + ln.Addr().String()
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/inner" {
				io.WriteString(w, "inner-ok")
				return
			}
			if r.URL.Path == "/warm" {
				io.WriteString(w, "warm-ok")
				return
			}
			// Wait until all 40 resumed idle connections are concurrently inside
			// the handler, then perform a loopback HTTP request back to this server.
			if outerReady.Add(1) == concurrency {
				close(allReady)
			}
			select {
			case <-allReady:
			case <-time.After(3 * time.Second):
				http.Error(w, "barrier timeout", http.StatusGatewayTimeout)
				return
			}
			resp, err := client.Get(baseURL + "/inner")
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			w.Write(body)
		}),
	}
	go srv.Serve(ln)
	defer srv.Close()

	conns := make([]net.Conn, concurrency)
	readers := make([]*bufio.Reader, concurrency)
	for i := 0; i < concurrency; i++ {
		c, err := net.Dial("tcp", ln.Addr().String())
		if err != nil {
			t.Fatalf("dial %d: %v", i, err)
		}
		defer c.Close()
		_ = c.SetDeadline(time.Now().Add(5 * time.Second))
		if _, err := io.WriteString(c, "GET /warm HTTP/1.1\r\nHost: localhost\r\n\r\n"); err != nil {
			t.Fatalf("warm write %d: %v", i, err)
		}
		br := bufio.NewReader(c)
		resp, err := http.ReadResponse(br, nil)
		if err != nil {
			t.Fatalf("warm read %d: %v", i, err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		conns[i] = c
		readers[i] = br
	}

	// Allow the 150ms hot grace and an extra sweep interval before resuming
	// the connections through the shared idle watcher.
	time.Sleep(300 * time.Millisecond)

	var wg sync.WaitGroup
	errs := make(chan error, concurrency)
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			if _, err := io.WriteString(conns[idx], "GET /outer HTTP/1.1\r\nHost: localhost\r\n\r\n"); err != nil {
				errs <- err
				return
			}
			resp, err := http.ReadResponse(readers[idx], nil)
			if err != nil {
				errs <- err
				return
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK || string(body) != "inner-ok" {
				errs <- io.ErrUnexpectedEOF
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent self-call failed: %v", err)
	}
}

func TestServerIdleKeepAlivePreservesRequestContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.Context().Err(); err != nil {
			http.Error(w, "ctx err: "+err.Error(), http.StatusInternalServerError)
			return
		}
		io.WriteString(w, "ok:"+r.URL.Path)
	}))
	defer srv.Close()

	conn, err := net.Dial("tcp", srv.Listener.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	br := bufio.NewReader(conn)

	for _, path := range []string{"/r1", "/r2", "/r3"} {
		if _, err := io.WriteString(conn, "GET "+path+" HTTP/1.1\r\nHost: localhost\r\n\r\n"); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
		resp, err := http.ReadResponse(br, nil)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK || string(body) != "ok:"+path {
			t.Fatalf("%s: got status=%d body=%q, want 200 ok:%s", path, resp.StatusCode, string(body), path)
		}
		// Pause past the 150ms lazy hot-grace window so the connection parks on
		// the shared idle watcher and resumes for the next request with live ctx.
		time.Sleep(220 * time.Millisecond)
	}
}

type nonSocketRawConn struct {
	fd uintptr
}

func (r nonSocketRawConn) Control(f func(fd uintptr)) error {
	f(r.fd)
	return nil
}

func (r nonSocketRawConn) Read(f func(fd uintptr) (done bool)) error {
	f(r.fd)
	return nil
}

func (r nonSocketRawConn) Write(f func(fd uintptr) (done bool)) error {
	f(r.fd)
	return nil
}

type nonSocketSyscallConn struct {
	net.Conn
	fd uintptr
}

func (c *nonSocketSyscallConn) SyscallConn() (syscall.RawConn, error) {
	return nonSocketRawConn{fd: c.fd}, nil
}

func TestServerBackgroundReadNonSocketSyscallConnFallback(t *testing.T) {
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer pr.Close()
	defer pw.Close()

	ln := newPipeListener()
	entered := make(chan struct{})
	cancelled := make(chan struct{})
	var ctxErrDuringFirst atomic.Value
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/first" {
			// Survive past backgroundRead promotion; ENOTSOCK from recvfrom
			// must fall back to standard backgroundRead instead of canceling ctx.
			time.Sleep(15 * time.Millisecond)
			if err := r.Context().Err(); err != nil {
				ctxErrDuringFirst.Store(err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			io.WriteString(w, "first-ok")
			return
		}
		close(entered)
		<-r.Context().Done()
		close(cancelled)
	})}
	go srv.Serve(ln)
	defer srv.Close()

	cSide, sSide := net.Pipe()
	defer cSide.Close()
	ln.ch <- &nonSocketSyscallConn{Conn: sSide, fd: pr.Fd()}

	br := bufio.NewReader(cSide)
	if _, err := io.WriteString(cSide, "GET /first HTTP/1.1\r\nHost: localhost\r\n\r\n"); err != nil {
		t.Fatalf("write first: %v", err)
	}
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatalf("read first: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || string(body) != "first-ok" {
		t.Fatalf("first: status=%d body=%q ctxErr=%v", resp.StatusCode, string(body), ctxErrDuringFirst.Load())
	}

	// Second request on same keep-alive non-socket conn: closing peer must cancel ctx via fallback.
	go func() {
		_, _ = io.WriteString(cSide, "GET /second HTTP/1.1\r\nHost: localhost\r\n\r\n")
	}()
	<-entered
	time.Sleep(10 * time.Millisecond)
	cSide.Close()
	select {
	case <-cancelled:
	case <-time.After(1 * time.Second):
		t.Fatal("non-socket syscall.Conn fallback did not observe client close promptly")
	}
}

func TestClientTransportKeepAliveRetryTimeoutAndLargePost(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/slow-header":
			time.Sleep(120 * time.Millisecond)
			io.WriteString(w, "slow")
		case "/echo":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Write(body)
		default:
			io.WriteString(w, "ok")
		}
	}))
	defer srv.Close()

	tr := &http.Transport{
		MaxIdleConns:          32,
		MaxIdleConnsPerHost:   32,
		ResponseHeaderTimeout: 50 * time.Millisecond,
	}
	defer tr.CloseIdleConnections()
	client := &http.Client{Transport: tr}

	// 1. Exercise keep-alive GETs across concurrent connections.
	const conns = 16
	var wg sync.WaitGroup
	errs := make(chan error, conns)
	for i := 0; i < conns; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				resp, err := client.Get(srv.URL + "/ok")
				if err != nil {
					errs <- err
					return
				}
				data, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil || string(data) != "ok" {
					errs <- io.ErrUnexpectedEOF
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("keep-alive GET failed: %v", err)
	}

	// 2. ResponseHeaderTimeout fires while waiting for the response.
	if resp, err := client.Get(srv.URL + "/slow-header"); err == nil {
		resp.Body.Close()
		t.Fatal("expected ResponseHeaderTimeout error, got nil")
	}

	// 3. A large POST echoes the full body.
	payload := strings.Repeat("0123456789abcdef", 8192) // 128 KiB
	trLarge := &http.Transport{MaxIdleConnsPerHost: 4}
	defer trLarge.CloseIdleConnections()
	resp, err := (&http.Client{Transport: trLarge}).Post(srv.URL+"/echo", "text/plain", strings.NewReader(payload))
	if err != nil {
		t.Fatalf("large POST failed: %v", err)
	}
	got, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil || string(got) != payload {
		t.Fatalf("large POST echo mismatch: len=%d want=%d err=%v", len(got), len(payload), err)
	}

	// 4. Server closing an idle keep-alive conn is retried transparently on GET.
	trRetry := &http.Transport{MaxIdleConnsPerHost: 4}
	defer trRetry.CloseIdleConnections()
	cRetry := &http.Client{Transport: trRetry}
	r1, err := cRetry.Get(srv.URL + "/ok")
	if err != nil {
		t.Fatalf("first GET: %v", err)
	}
	io.Copy(io.Discard, r1.Body)
	r1.Body.Close()
	srv.CloseClientConnections()
	time.Sleep(15 * time.Millisecond)
	r2, err := cRetry.Get(srv.URL + "/ok")
	if err != nil {
		t.Fatalf("GET after server closed idle conn failed: %v", err)
	}
	io.Copy(io.Discard, r2.Body)
	r2.Body.Close()

	// 5. An unknown-length POST uses chunked encoding and echoes cleanly.
	chunkedReq, err := http.NewRequest("POST", srv.URL+"/echo", io.NopCloser(strings.NewReader("chunked-body")))
	if err != nil {
		t.Fatalf("new chunked req: %v", err)
	}
	chunkedReq.ContentLength = 0
	rChunk, err := cRetry.Do(chunkedReq)
	if err != nil {
		t.Fatalf("chunked POST failed: %v", err)
	}
	gotChunk, err := io.ReadAll(rChunk.Body)
	rChunk.Body.Close()
	if err != nil || string(gotChunk) != "chunked-body" {
		t.Fatalf("chunked POST mismatch: got=%q err=%v", string(gotChunk), err)
	}
}

func TestServerIdleClearsHandlerReadDeadlineAndHandlesPartialPrefix(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/set-deadline" {
			rc := http.NewResponseController(w)
			if err := rc.SetReadDeadline(time.Now().Add(40 * time.Millisecond)); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		io.WriteString(w, "ok:"+r.URL.Path)
	}))
	defer srv.Close()

	addr := strings.TrimPrefix(srv.URL, "http://")
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	br := bufio.NewReader(conn)

	// 1. Handler sets a 40ms read deadline via ResponseController; keeping the
	// connection idle for 80ms must not kill it (deadline must be cleared).
	if _, err := io.WriteString(conn, "GET /set-deadline HTTP/1.1\r\nHost: localhost\r\n\r\n"); err != nil {
		t.Fatalf("write 1: %v", err)
	}
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatalf("read 1: %v", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	time.Sleep(80 * time.Millisecond)

	// 2. Send a 2-byte prefix before grace expiry, pause past the 150ms grace
	// window, then send the rest of the request; server must not stall in park.
	if _, err := io.WriteString(conn, "GE"); err != nil {
		t.Fatalf("write partial prefix: %v", err)
	}
	time.Sleep(220 * time.Millisecond)
	if _, err := io.WriteString(conn, "T /after HTTP/1.1\r\nHost: localhost\r\n\r\n"); err != nil {
		t.Fatalf("write remainder: %v", err)
	}
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	resp2, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatalf("read 2 after handler deadline + partial prefix: %v", err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK || string(body2) != "ok:/after" {
		t.Fatalf("resp2: status=%d body=%q", resp2.StatusCode, string(body2))
	}
}

func TestClientTransportRejectsUnsolicitedBufferedBytes(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	var accepted atomic.Int32
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			n := accepted.Add(1)
			go func(conn net.Conn, id int32) {
				defer conn.Close()
				br := bufio.NewReader(conn)
				req, err := http.ReadRequest(br)
				if err != nil {
					return
				}
				io.Copy(io.Discard, req.Body)
				req.Body.Close()
				if id == 1 {
					// Append a fabricated unsolicited second response in the same
					// write so pc.br buffers it alongside the first response.
					_, _ = io.WriteString(conn,
						"HTTP/1.1 200 OK\r\nContent-Length: 5\r\n\r\nfirst"+
							"HTTP/1.1 200 OK\r\nContent-Length: 6\r\n\r\npoison")
				} else {
					_, _ = io.WriteString(conn, "HTTP/1.1 200 OK\r\nContent-Length: 6\r\n\r\nsecond")
				}
			}(c, n)
		}
	}()

	tr := &http.Transport{MaxIdleConnsPerHost: 4}
	defer tr.CloseIdleConnections()
	client := &http.Client{Transport: tr}
	url := "http://" + ln.Addr().String() + "/"

	r1, err := client.Get(url)
	if err != nil {
		t.Fatalf("first GET: %v", err)
	}
	b1, err := io.ReadAll(r1.Body)
	r1.Body.Close()
	if err != nil || string(b1) != "first" {
		t.Fatalf("first body=%q err=%v", string(b1), err)
	}

	r2, err := client.Get(url)
	if err != nil {
		t.Fatalf("second GET: %v", err)
	}
	b2, err := io.ReadAll(r2.Body)
	r2.Body.Close()
	if err != nil || string(b2) != "second" {
		t.Fatalf("second body=%q (want \"second\", not poisoned) err=%v", string(b2), err)
	}
	if got := accepted.Load(); got < 2 {
		t.Fatalf("expected second request to open a new connection, accepted=%d", got)
	}
}

type blockingBodyReader struct {
	closed chan struct{}
	once   sync.Once
}

func (b *blockingBodyReader) Read(p []byte) (int, error) {
	<-b.closed
	return 0, errors.New("body closed")
}

func (b *blockingBodyReader) Close() error {
	b.once.Do(func() { close(b.closed) })
	return nil
}

func TestClientTransportEarlyRejectSmallStreamingPost(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		br := bufio.NewReader(conn)
		// Read request headers only, then immediately reject with 413 + Connection: close.
		for {
			line, err := br.ReadString('\n')
			if err != nil || line == "\r\n" {
				break
			}
		}
		_, _ = io.WriteString(conn, "HTTP/1.1 413 Payload Too Large\r\nConnection: close\r\nContent-Length: 0\r\n\r\n")
	}()

	tr := &http.Transport{MaxIdleConnsPerHost: 4}
	defer tr.CloseIdleConnections()
	client := &http.Client{Transport: tr, Timeout: 2 * time.Second}

	body := &blockingBodyReader{closed: make(chan struct{})}
	defer body.Close()
	req, err := http.NewRequest("POST", "http://"+ln.Addr().String()+"/upload", body)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.ContentLength = 8

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("streaming POST early 413 failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", resp.StatusCode)
	}
}

func TestClientTransportGot1xxErrorTerminalOnReusedConn(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	var requests atomic.Int32
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				br := bufio.NewReader(c)
				for {
					req, err := http.ReadRequest(br)
					if err != nil {
						return
					}
					io.Copy(io.Discard, req.Body)
					req.Body.Close()
					n := requests.Add(1)
					switch n {
					case 1:
						// Warm the keep-alive connection.
						_, _ = io.WriteString(c, "HTTP/1.1 200 OK\r\nContent-Length: 4\r\n\r\nwarm")
					case 2:
						// Send 103 Early Hints followed by 200 on the reused conn.
						_, _ = io.WriteString(c, "HTTP/1.1 103 Early Hints\r\nLink: </style.css>; rel=preload\r\n\r\nHTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nok")
						return
					default:
						// If the client erroneously retried on a new connection, return 200.
						_, _ = io.WriteString(c, "HTTP/1.1 200 OK\r\nContent-Length: 7\r\n\r\nretried")
						return
					}
				}
			}(conn)
		}
	}()

	tr := &http.Transport{MaxIdleConnsPerHost: 4}
	defer tr.CloseIdleConnections()
	client := &http.Client{Transport: tr}
	url := "http://" + ln.Addr().String() + "/"

	r1, err := client.Get(url)
	if err != nil {
		t.Fatalf("warm GET: %v", err)
	}
	io.Copy(io.Discard, r1.Body)
	r1.Body.Close()

	sentinel := errors.New("stop on 103")
	ctx := httptrace.WithClientTrace(context.Background(), &httptrace.ClientTrace{
		Got1xxResponse: func(code int, header textproto.MIMEHeader) error {
			if code == 103 {
				return sentinel
			}
			return nil
		},
	})
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext: %v", err)
	}
	r2, err := client.Do(req)
	if err == nil {
		r2.Body.Close()
		t.Fatalf("expected sentinel error from Got1xxResponse, got status %d (requests=%d)", r2.StatusCode, requests.Load())
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want errors.Is(_, sentinel)", err)
	}
	if got := requests.Load(); got != 2 {
		t.Fatalf("requests = %d, want 2 (no retry after Got1xxResponse error)", got)
	}
}

func TestClientTransportEarlyBodyCloseReturnsPromptly(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	releaseBody := make(chan struct{})
	defer close(releaseBody)

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		br := bufio.NewReader(conn)
		if _, err := http.ReadRequest(br); err != nil {
			return
		}
		// Flush response headers with Content-Length: 8 but withhold body bytes
		// until after the client closes Response.Body.
		_, _ = io.WriteString(conn, "HTTP/1.1 200 OK\r\nContent-Length: 8\r\n\r\n")
		<-releaseBody
	}()

	tr := &http.Transport{MaxIdleConnsPerHost: 4}
	defer tr.CloseIdleConnections()
	client := &http.Client{Transport: tr}

	resp, err := client.Get("http://" + ln.Addr().String() + "/delayed-body")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	start := time.Now()
	if err := resp.Body.Close(); err != nil {
		t.Fatalf("Body.Close: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 20*time.Millisecond {
		t.Fatalf("early Response.Body.Close blocked for %v (want < 20ms)", elapsed)
	}
}

func TestServerIdleTimeoutExpiresPartialPrefix(t *testing.T) {
	idle := make(chan struct{})
	prefixSent := make(chan struct{})
	defer close(prefixSent)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{
		IdleTimeout: 100 * time.Millisecond,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
		ConnState: func(_ net.Conn, state http.ConnState) {
			if state == http.StateIdle {
				close(idle)
				<-prefixSent
			}
		},
	}
	go srv.Serve(ln)
	defer srv.Close()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(conn, "GET / HTTP/1.1\r\nHost: localhost\r\n\r\n"); err != nil {
		t.Fatal(err)
	}
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	select {
	case <-idle:
	case <-time.After(3 * time.Second):
		t.Fatal("server never entered the idle state")
	}
	if _, err := io.WriteString(conn, "GE"); err != nil {
		t.Fatal(err)
	}
	// Send the partial prefix before the server begins its keep-alive wait.
	prefixSent <- struct{}{}
	if _, err := br.ReadByte(); !errors.Is(err, io.EOF) {
		t.Fatalf("connection with a partial request prefix should close at IdleTimeout, got %v", err)
	}
}

func TestClientTransportEarlyRejectWhileWritingHeaders(t *testing.T) {
	ln, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	release := make(chan struct{})
	defer close(release)
	serverErr := make(chan error, 1)
	go func() {
		conn, err := ln.AcceptTCP()
		if err != nil {
			serverErr <- err
			return
		}
		defer conn.Close()
		if err := conn.SetReadBuffer(1024); err != nil {
			serverErr <- err
			return
		}
		var firstByte [1]byte
		if _, err := io.ReadFull(conn, firstByte[:]); err != nil {
			serverErr <- err
			return
		}
		// Leave the connection open while rejecting the request, without
		// draining the remaining headers from the receive buffer.
		_, err = io.WriteString(conn, "HTTP/1.1 413 Content Too Large\r\nConnection: close\r\nContent-Length: 0\r\n\r\n")
		serverErr <- err
		<-release
	}()

	tr := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := (&net.Dialer{}).DialContext(ctx, network, addr)
			if err != nil {
				return nil, err
			}
			if err := conn.(*net.TCPConn).SetWriteBuffer(1024); err != nil {
				conn.Close()
				return nil, err
			}
			return conn, nil
		},
	}
	defer tr.CloseIdleConnections()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", "http://"+ln.Addr().String()+"/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Large", strings.Repeat("a", 1<<20))
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatalf("early response while writing request headers: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", resp.StatusCode)
	}
	if err := <-serverErr; err != nil {
		t.Fatalf("server: %v", err)
	}
}

func TestClientTransportCancelCompletedRequestAndReuseConn(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/no-body" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		io.WriteString(w, "ok")
	}))
	defer srv.Close()

	for _, path := range []string{"/body", "/no-body"} {
		t.Run(path[1:], func(t *testing.T) {
			tr := &http.Transport{MaxConnsPerHost: 1}
			defer tr.CloseIdleConnections()
			client := &http.Client{Transport: tr, Timeout: 3 * time.Second}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			first, err := http.NewRequestWithContext(ctx, "GET", srv.URL+path, nil)
			if err != nil {
				t.Fatal(err)
			}
			resp, err := client.Do(first)
			if err != nil {
				t.Fatal(err)
			}
			_, err = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if err != nil {
				t.Fatal(err)
			}

			var reused bool
			trace := &httptrace.ClientTrace{GotConn: func(info httptrace.GotConnInfo) { reused = info.Reused }}
			next, err := http.NewRequestWithContext(httptrace.WithClientTrace(context.Background(), trace), "GET", srv.URL+"/body", nil)
			if err != nil {
				t.Fatal(err)
			}
			cancel()
			resp, err = client.Do(next)
			if err != nil {
				t.Fatalf("request after canceling the completed request: %v", err)
			}
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil || string(body) != "ok" || !reused {
				t.Fatalf("next request: body=%q err=%v reused=%t", body, err, reused)
			}
		})
	}
}
