//go:build darwin || linux

package http_test

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
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
	// watcher after the 5ms hot-grace window instead of pinning one OS thread
	// per idle socket.
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

	// Let all 40 idle connections park into the shared idle watcher.
	time.Sleep(25 * time.Millisecond)

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
