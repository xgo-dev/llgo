//go:build (darwin || linux) && !baremetal

package gotest

import (
	"errors"
	"net"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func countOpenFDs(t *testing.T) int {
	t.Helper()
	dir := "/proc/self/fd"
	if runtime.GOOS == "darwin" {
		dir = "/dev/fd"
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	return len(entries)
}

func TestNetpollCachedWakePipeAndConcurrentWaiters(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	client, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	server, err := ln.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	// 1. Repeated read waits + deadline cancellations on the same connection
	// wake promptly and keep the open descriptor count bounded.
	fdsBefore := countOpenFDs(t)
	for i := 0; i < 16; i++ {
		if err := client.SetReadDeadline(time.Time{}); err != nil {
			t.Fatalf("reset deadline %d: %v", i, err)
		}
		entered := make(chan struct{})
		errCh := make(chan error, 1)
		go func() {
			var b [1]byte
			close(entered)
			_, rerr := client.Read(b[:])
			errCh <- rerr
		}()
		<-entered
		time.Sleep(2 * time.Millisecond)
		if err := client.SetReadDeadline(time.Now().Add(-time.Second)); err != nil {
			t.Fatalf("cancel deadline %d: %v", i, err)
		}
		select {
		case rerr := <-errCh:
			if !errors.Is(rerr, os.ErrDeadlineExceeded) {
				t.Fatalf("iter %d: Read err = %v, want timeout", i, rerr)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("iter %d: Read did not wake on deadline cancel", i)
		}
	}
	if fdsAfter := countOpenFDs(t); fdsAfter-fdsBefore > 4 {
		t.Fatalf("repeated deadline resets grew open fds from %d to %d", fdsBefore, fdsAfter)
	}

	// 2. Concurrent Read + Write waiters on the same fd must each receive their
	// own deadline/close wake without stealing the other waiter's notification.
	if err := client.SetDeadline(time.Time{}); err != nil {
		t.Fatal(err)
	}
	if tc, ok := client.(*net.TCPConn); ok {
		if err := tc.SetWriteBuffer(4096); err != nil {
			t.Fatalf("SetWriteBuffer: %v", err)
		}
	}
	if tc, ok := server.(*net.TCPConn); ok {
		if err := tc.SetReadBuffer(4096); err != nil {
			t.Fatalf("SetReadBuffer: %v", err)
		}
	}

	readStarted := make(chan struct{})
	readErrCh := make(chan error, 1)
	go func() {
		var b [1]byte
		close(readStarted)
		_, rerr := client.Read(b[:])
		readErrCh <- rerr
	}()
	<-readStarted

	// Fill the TCP send buffer so a subsequent Write blocks in pollWait('w')
	// concurrently with the Read blocked in pollWait('r').
	writeEntered := make(chan struct{})
	var writeAttempts atomic.Int64
	writeErrCh := make(chan error, 1)
	go func() {
		chunk := make([]byte, 64*1024)
		close(writeEntered)
		for {
			writeAttempts.Add(1)
			if _, werr := client.Write(chunk); werr != nil {
				writeErrCh <- werr
				return
			}
		}
	}()
	<-writeEntered

	// Wait until writeAttempts stops advancing, proving the writer is blocked
	// inside client.Write / pollWait('w') rather than draining into the buffer.
	waitDeadline := time.Now().Add(2 * time.Second)
	blocked := false
	for time.Now().Before(waitDeadline) {
		before := writeAttempts.Load()
		if before > 0 {
			time.Sleep(20 * time.Millisecond)
			if writeAttempts.Load() == before {
				select {
				case werr := <-writeErrCh:
					t.Fatalf("Write exited unexpectedly before deadline cancel: %v", werr)
				default:
					blocked = true
				}
				break
			}
			continue
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !blocked {
		t.Fatal("writer did not block in send buffer before deadline cancel")
	}

	// Cancel only the write deadline first; Read must remain blocked until we
	// cancel the read deadline (or close).
	if err := client.SetWriteDeadline(time.Now().Add(-time.Second)); err != nil {
		t.Fatal(err)
	}
	select {
	case werr := <-writeErrCh:
		if !errors.Is(werr, os.ErrDeadlineExceeded) {
			t.Fatalf("Write err = %v, want timeout", werr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Write did not wake on write deadline cancel")
	}

	select {
	case rerr := <-readErrCh:
		t.Fatalf("Read woke unexpectedly on write deadline cancel: %v", rerr)
	case <-time.After(20 * time.Millisecond):
	}

	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case rerr := <-readErrCh:
		if !errors.Is(rerr, net.ErrClosed) {
			t.Fatalf("Read err after Close = %v, want net.ErrClosed", rerr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Read did not wake on Close")
	}
}

func TestNetpollConcurrentCloseReleasesCachedWakePipe(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	beforeFDs := countOpenFDs(t)
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		c, err := net.Dial("tcp", ln.Addr().String())
		if err != nil {
			t.Fatalf("dial %d: %v", i, err)
		}
		s, err := ln.Accept()
		if err != nil {
			c.Close()
			t.Fatalf("accept %d: %v", i, err)
		}
		entered := make(chan struct{})
		wg.Add(1)
		go func(conn net.Conn) {
			defer wg.Done()
			var b [1]byte
			close(entered)
			_, _ = conn.Read(b[:])
		}(c)
		<-entered
		time.Sleep(time.Millisecond)
		c.Close()
		s.Close()
	}
	wg.Wait()

	afterFDs := countOpenFDs(t)
	if afterFDs-beforeFDs > 8 {
		t.Fatalf("open fds leaked after 64 wait/close cycles: before=%d after=%d", beforeFDs, afterFDs)
	}
}
