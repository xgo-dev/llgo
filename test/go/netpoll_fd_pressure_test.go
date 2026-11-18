//go:build (darwin || linux) && !baremetal

package gotest

import (
	"context"
	"errors"
	"net"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestNetworkIOUnderFDPressure(t *testing.T) {
	const childEnv = "LLGO_TEST_NETPOLL_FD_PRESSURE_CHILD"
	if os.Getenv(childEnv) == "" {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestNetworkIOUnderFDPressure$")
		cmd.Env = append(os.Environ(), childEnv+"=1")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("fd-pressure child: %v\n%s", err, output)
		}
		return
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	peer, err := ln.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer peer.Close()

	// Confine the process-wide limit change to this child. Existing sockets
	// stay usable, but creating even one wake pipe must fail with EMFILE.
	var saved syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &saved); err != nil {
		t.Fatal(err)
	}
	limited := saved
	limited.Cur = 0
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &limited); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &saved); err != nil {
			t.Error(err)
		}
	}()
	if f, err := os.Open(os.DevNull); !errors.Is(err, syscall.EMFILE) {
		if f != nil {
			f.Close()
		}
		t.Fatalf("open under fd pressure: %v, want EMFILE", err)
	}

	results := make(chan error, 1)
	read := func() {
		go func() {
			var b [1]byte
			_, err := conn.Read(b[:])
			results <- err
		}()
	}
	wait := func(want error) {
		t.Helper()
		select {
		case err := <-results:
			if !errors.Is(err, want) {
				t.Fatalf("Read: %v, want %v", err, want)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("Read did not wake under fd pressure")
		}
	}
	read()
	time.Sleep(30 * time.Millisecond)
	if _, err := peer.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	wait(nil)

	if err := conn.SetReadDeadline(time.Now().Add(40 * time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	read()
	wait(os.ErrDeadlineExceeded)
	if err := conn.SetReadDeadline(time.Time{}); err != nil {
		t.Fatal(err)
	}
	read()
	time.Sleep(30 * time.Millisecond)
	if err := conn.SetReadDeadline(time.Now().Add(-time.Second)); err != nil {
		t.Fatal(err)
	}
	wait(os.ErrDeadlineExceeded)
	if err := conn.SetReadDeadline(time.Time{}); err != nil {
		t.Fatal(err)
	}
	read()
	time.Sleep(30 * time.Millisecond)
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	wait(net.ErrClosed)
}
