package net_test

import (
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

func TestTCPConnMethodCoverage(t *testing.T) {
	ln, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatalf("ListenTCP error: %v", err)
	}
	t.Cleanup(func() {
		if err := ln.Close(); err != nil {
			t.Errorf("ln.Close: %v", err)
		}
	})

	if lf, err := ln.File(); err == nil {
		defer lf.Close()
		if fl, err := net.FileListener(lf); err == nil {
			fl.Close()
		} else {
			t.Logf("FileListener: %v", err)
		}
	} else {
		t.Logf("TCPListener.File unavailable: %v", err)
	}

	if raw, err := ln.SyscallConn(); err == nil {
		done := make(chan struct{})
		if err := raw.Control(func(uintptr) {}); err != nil {
			t.Logf("TCPListener.SyscallConn control: %v", err)
		}
		close(done)
	} else {
		t.Logf("TCPListener.SyscallConn unavailable: %v", err)
	}

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := ln.Accept()
		if err != nil {
			t.Errorf("Accept error: %v", err)
			return
		}
		tcpConn, ok := conn.(*net.TCPConn)
		if !ok {
			t.Errorf("connection type %T, want *net.TCPConn", conn)
			conn.Close()
			return
		}
		defer tcpConn.Close()

		checkSocketOption(t, "SetReadBuffer", (*net.TCPConn).SetReadBuffer(tcpConn, 4096))
		checkSocketOption(t, "SetWriteBuffer", (*net.TCPConn).SetWriteBuffer(tcpConn, 4096))
		if err := (*net.TCPConn).SetDeadline(tcpConn, time.Now().Add(500*time.Millisecond)); err != nil {
			t.Errorf("SetDeadline: %v", err)
		}
		if err := (*net.TCPConn).SetReadDeadline(tcpConn, time.Now().Add(500*time.Millisecond)); err != nil {
			t.Errorf("SetReadDeadline: %v", err)
		}
		if err := (*net.TCPConn).SetWriteDeadline(tcpConn, time.Now().Add(500*time.Millisecond)); err != nil {
			t.Errorf("SetWriteDeadline: %v", err)
		}

		if (*net.TCPConn).LocalAddr(tcpConn) == nil {
			t.Error("LocalAddr returned nil")
		}
		if (*net.TCPConn).RemoteAddr(tcpConn) == nil {
			t.Error("RemoteAddr returned nil")
		}

		buf := make([]byte, 5)
		tcpConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		if _, err := (*net.TCPConn).Read(tcpConn, buf); err != nil {
			t.Errorf("server ReadFull: %v", err)
		}
		if _, err := (*net.TCPConn).Write(tcpConn, []byte("pong")); err != nil {
			t.Errorf("server Write: %v", err)
		}

		if _, err := tcpConn.ReadFrom(strings.NewReader("from-server")); err != nil {
			t.Errorf("ReadFrom: %v", err)
		}

		if raw, err := tcpConn.SyscallConn(); err == nil {
			if err := raw.Control(func(uintptr) {}); err != nil {
				t.Logf("server SyscallConn control: %v", err)
			}
		} else {
			t.Logf("server SyscallConn unavailable: %v", err)
		}

		if f, err := tcpConn.File(); err == nil {
			f.Close()
		} else {
			t.Logf("server File unavailable: %v", err)
		}

		writeToDone := make(chan struct{})
		go func() {
			defer close(writeToDone)
			tcpConn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
			if _, err := tcpConn.WriteTo(io.Discard); err != nil && !errors.Is(err, os.ErrDeadlineExceeded) {
				t.Logf("WriteTo: %v", err)
			}
		}()
		<-writeToDone
	}()

	client, err := net.DialTCP("tcp", nil, ln.Addr().(*net.TCPAddr))
	if err != nil {
		t.Fatalf("DialTCP error: %v", err)
	}
	t.Cleanup(func() {
		if err := (*net.TCPConn).Close(client); err != nil {
			t.Errorf("client.Close: %v", err)
		}
	})

	checkSocketOption(t, "client SetReadBuffer", (*net.TCPConn).SetReadBuffer(client, 4096))
	checkSocketOption(t, "client SetWriteBuffer", (*net.TCPConn).SetWriteBuffer(client, 4096))
	if err := (*net.TCPConn).SetDeadline(client, time.Now().Add(time.Second)); err != nil {
		t.Errorf("client SetDeadline: %v", err)
	}
	if err := (*net.TCPConn).SetReadDeadline(client, time.Now().Add(time.Second)); err != nil {
		t.Errorf("client SetReadDeadline: %v", err)
	}
	if err := (*net.TCPConn).SetWriteDeadline(client, time.Now().Add(time.Second)); err != nil {
		t.Errorf("client SetWriteDeadline: %v", err)
	}

	if (*net.TCPConn).LocalAddr(client) == nil {
		t.Fatal("client LocalAddr returned nil")
	}
	if (*net.TCPConn).RemoteAddr(client) == nil {
		t.Fatal("client RemoteAddr returned nil")
	}

	if raw, err := client.SyscallConn(); err == nil {
		if err := raw.Control(func(uintptr) {}); err != nil {
			t.Logf("client SyscallConn control: %v", err)
		}
	} else {
		t.Logf("client SyscallConn unavailable: %v", err)
	}

	if f, err := client.File(); err == nil {
		defer f.Close()
		if dup, err := net.FileConn(f); err == nil {
			dup.Close()
		} else {
			t.Logf("FileConn: %v", err)
		}
	} else {
		t.Logf("client File unavailable: %v", err)
	}

	if _, err := (*net.TCPConn).Write(client, []byte("hello")); err != nil {
		t.Fatalf("client Write: %v", err)
	}

	reply := make([]byte, 4)
	if _, err := (*net.TCPConn).Read(client, reply); err != nil {
		t.Fatalf("client Read pong: %v", err)
	}

	extra := make([]byte, len("from-server"))
	if _, err := (*net.TCPConn).Read(client, extra); err != nil {
		t.Fatalf("client Read from ReadFrom: %v", err)
	}

	if _, err := client.ReadFrom(strings.NewReader("from-client")); err != nil {
		t.Fatalf("client ReadFrom: %v", err)
	}

	client.CloseWrite()
	client.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	if _, err := client.WriteTo(io.Discard); err != nil && !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Logf("client WriteTo: %v", err)
	}

	<-serverDone
}
