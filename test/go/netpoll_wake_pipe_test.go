//go:build (darwin || linux) && !baremetal

package gotest

import (
	"errors"
	"net"
	"os"
	"sync"
	"testing"
	"time"
)

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

	// 1. Repeated read waits + deadline cancellations reuse the cached wake pipe.
	for i := 0; i < 16; i++ {
		if err := client.SetReadDeadline(time.Time{}); err != nil {
			t.Fatalf("reset deadline %d: %v", i, err)
		}
		errCh := make(chan error, 1)
		go func() {
			var b [1]byte
			_, rerr := client.Read(b[:])
			errCh <- rerr
		}()
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

	// 2. Concurrent Read + Write waiters on the same fd must each receive their
	// own deadline/close wake without stealing the other waiter's notification.
	if err := client.SetDeadline(time.Time{}); err != nil {
		t.Fatal(err)
	}
	if tc, ok := client.(*net.TCPConn); ok {
		_ = tc.SetWriteBuffer(4096)
	}
	if tc, ok := server.(*net.TCPConn); ok {
		_ = tc.SetReadBuffer(4096)
	}

	readErrCh := make(chan error, 1)
	go func() {
		var b [1]byte
		_, rerr := client.Read(b[:])
		readErrCh <- rerr
	}()

	// Fill the TCP send buffer so a subsequent Write blocks in pollWait('w')
	// concurrently with the Read blocked in pollWait('r').
	writeErrCh := make(chan error, 1)
	go func() {
		chunk := make([]byte, 64*1024)
		for {
			if _, werr := client.Write(chunk); werr != nil {
				writeErrCh <- werr
				return
			}
		}
	}()

	time.Sleep(25 * time.Millisecond)

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
		wg.Add(1)
		go func(conn net.Conn) {
			defer wg.Done()
			var b [1]byte
			_, _ = conn.Read(b[:])
		}(c)
		time.Sleep(time.Millisecond)
		c.Close()
		s.Close()
	}
	wg.Wait()
}
