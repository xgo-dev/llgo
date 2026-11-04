//go:build darwin || linux

package net_test

import (
	"errors"
	"net"
	"os"
	"testing"
	"time"
)

// Each operation must observe its own notification even when another blocked
// Accept or Read consumes a notification at the same time.
func TestPollWakeAllDescriptors(t *testing.T) {
	const count = 32
	listeners := make([]net.Listener, count)
	done := make(chan error, count)
	for i := range listeners {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		listeners[i] = ln
		t.Cleanup(func() { ln.Close() })
		go func() {
			conn, err := ln.Accept()
			if conn != nil {
				conn.Close()
			}
			done <- err
		}()
	}
	time.Sleep(30 * time.Millisecond)
	start := time.Now()
	for _, ln := range listeners {
		go ln.Close()
	}
	timer := time.NewTimer(750 * time.Millisecond)
	defer timer.Stop()
	for i := 0; i < count; i++ {
		select {
		case err := <-done:
			if !errors.Is(err, net.ErrClosed) {
				t.Fatalf("Accept: %v, want net.ErrClosed", err)
			}
		case <-timer.C:
			t.Fatalf("only %d/%d Accept calls woke after Close (%v)", i, count, time.Since(start))
		}
	}
}

func TestPollWakeReadWriteDeadline(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	accepted := make(chan net.Conn, 1)
	go func() {
		c, _ := ln.Accept()
		accepted <- c
	}()
	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	peer := <-accepted
	if peer == nil {
		t.Fatal("Accept failed")
	}
	defer peer.Close()
	if err := conn.(*net.TCPConn).SetWriteBuffer(1024); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 2)
	go func() {
		var b [1]byte
		_, err := conn.Read(b[:])
		done <- err
	}()
	go func() {
		buf := make([]byte, 64*1024)
		for {
			if _, err := conn.Write(buf); err != nil {
				done <- err
				return
			}
		}
	}()
	time.Sleep(30 * time.Millisecond)
	if err := conn.SetDeadline(time.Now().Add(-time.Second)); err != nil {
		t.Fatal(err)
	}
	timer := time.NewTimer(750 * time.Millisecond)
	defer timer.Stop()
	for i := 0; i < 2; i++ {
		select {
		case err := <-done:
			if !errors.Is(err, os.ErrDeadlineExceeded) {
				t.Fatalf("I/O: %v, want deadline exceeded", err)
			}
		case <-timer.C:
			t.Fatalf("only %d/2 read/write waiters observed the new deadline", i)
		}
	}
	// Clearing an expired deadline must make the descriptor usable again.
	if err := conn.SetDeadline(time.Time{}); err != nil {
		t.Fatal(err)
	}
	if _, err := peer.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	var b [1]byte
	if _, err := conn.Read(b[:]); err != nil || b[0] != 'x' {
		t.Fatalf("Read after clearing deadline: %q, %v", b, err)
	}
}
