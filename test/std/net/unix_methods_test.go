package net_test

import (
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestUnixConnMethodCoverage(t *testing.T) {
	// Keep socket addresses below the platform sockaddr path limit. In
	// particular, Windows supports AF_UNIX streams but has a short path limit.
	dir, err := os.MkdirTemp(".", ".llgo-unix-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	streamPath := filepath.Join(dir, "stream.sock")
	streamAddr := &net.UnixAddr{Name: streamPath, Net: "unix"}
	ln, err := net.ListenUnix("unix", streamAddr)
	if err != nil {
		t.Fatalf("ListenUnix error: %v", err)
	}
	ln.SetUnlinkOnClose(true)
	t.Cleanup(func() {
		if err := ln.Close(); err != nil {
			t.Errorf("ln.Close: %v", err)
		}
	})
	if ln.Addr() == nil {
		t.Error("UnixListener Addr returned nil")
	}

	if err := ln.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Errorf("UnixListener.SetDeadline: %v", err)
	}
	if lf, err := ln.File(); err == nil {
		defer lf.Close()
		if l2, err := net.FileListener(lf); err == nil {
			l2.Close()
		} else {
			t.Logf("Unix FileListener: %v", err)
		}
	} else {
		t.Logf("UnixListener.File unavailable: %v", err)
	}
	if raw, err := ln.SyscallConn(); err == nil {
		if err := raw.Control(func(uintptr) {}); err != nil {
			t.Logf("UnixListener.SyscallConn: %v", err)
		}
	} else {
		t.Logf("UnixListener.SyscallConn unavailable: %v", err)
	}

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		c, err := ln.Accept()
		if err != nil {
			t.Errorf("AcceptUnix error: %v", err)
			return
		}
		conn, ok := c.(*net.UnixConn)
		if !ok {
			t.Errorf("UnixListener.Accept returned %T", c)
			c.Close()
			return
		}
		defer conn.Close()

		if conn.LocalAddr() == nil || conn.RemoteAddr() == nil {
			t.Error("UnixConn stream addresses nil")
		}
		checkSocketOption(t, "SetReadBuffer", conn.SetReadBuffer(2048))
		checkSocketOption(t, "SetWriteBuffer", conn.SetWriteBuffer(2048))
		if err := conn.SetDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
			t.Errorf("SetDeadline: %v", err)
		}
		if err := conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
			t.Errorf("SetReadDeadline: %v", err)
		}
		if err := conn.SetWriteDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
			t.Errorf("SetWriteDeadline: %v", err)
		}

		buf := make([]byte, 5)
		if _, err := io.ReadFull(conn, buf); err != nil {
			t.Errorf("Unix stream Read: %v", err)
		}
		if _, err := conn.Write([]byte("pong")); err != nil {
			t.Errorf("Unix stream Write: %v", err)
		}

		conn.CloseRead()
		conn.CloseWrite()

		if raw, err := conn.SyscallConn(); err == nil {
			if err := raw.Control(func(uintptr) {}); err != nil {
				t.Logf("Unix stream SyscallConn: %v", err)
			}
		} else {
			t.Logf("Unix stream SyscallConn unavailable: %v", err)
		}
		if f, err := conn.File(); err == nil {
			f.Close()
		} else {
			t.Logf("Unix stream File unavailable: %v", err)
		}
	}()

	client, err := net.DialUnix("unix", nil, streamAddr)
	if err != nil {
		t.Fatalf("DialUnix error: %v", err)
	}
	defer client.Close()
	if err := client.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Errorf("client SetDeadline: %v", err)
	}
	if err := client.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Errorf("client SetReadDeadline: %v", err)
	}
	if err := client.SetWriteDeadline(time.Now().Add(time.Second)); err != nil {
		t.Errorf("client SetWriteDeadline: %v", err)
	}
	if client.LocalAddr() == nil || client.RemoteAddr() == nil {
		t.Fatal("client unix stream addresses nil")
	}
	if raw, err := client.SyscallConn(); err == nil {
		if err := raw.Control(func(uintptr) {}); err != nil {
			t.Logf("client stream SyscallConn: %v", err)
		}
	} else {
		t.Logf("client stream SyscallConn unavailable: %v", err)
	}
	if f, err := client.File(); err == nil {
		defer f.Close()
		if c2, err := net.FileConn(f); err == nil {
			c2.Close()
		} else {
			t.Logf("client stream FileConn: %v", err)
		}
	} else {
		t.Logf("client stream File unavailable: %v", err)
	}

	if _, err := client.Write([]byte("hello")); err != nil {
		t.Fatalf("client stream Write: %v", err)
	}
	reply := make([]byte, 4)
	if _, err := io.ReadFull(client, reply); err != nil {
		t.Fatalf("client stream Read: %v", err)
	}

	<-serverDone

	additionalDone := make(chan struct{})
	go func() {
		defer close(additionalDone)
		spare, err := ln.AcceptUnix()
		if err != nil {
			t.Logf("AcceptUnix fallback: %v", err)
			return
		}
		spare.Close()
	}()
	if extraClient, err := net.DialUnix("unix", nil, streamAddr); err == nil {
		extraClient.Close()
	}
	<-additionalDone

	// Datagram coverage
	gramPath := filepath.Join(dir, "gram.sock")
	gramAddr := &net.UnixAddr{Name: gramPath, Net: "unixgram"}
	gram, err := net.ListenUnixgram("unixgram", gramAddr)
	if runtime.GOOS == "windows" {
		// Go exposes the Unix datagram APIs on Windows, but Winsock does not
		// implement AF_UNIX SOCK_DGRAM. Verify the documented platform behavior
		// instead of hiding all Unix-socket coverage behind a Windows skip.
		if err == nil {
			gram.Close()
			t.Fatal("ListenUnixgram unexpectedly succeeded on Windows")
		}
		return
	}
	if err != nil {
		t.Fatalf("ListenUnixgram error: %v", err)
	}
	defer gram.Close()
	if err := gram.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Errorf("unixgram SetDeadline: %v", err)
	}
	checkSocketOption(t, "unixgram SetReadBuffer", gram.SetReadBuffer(1024))
	checkSocketOption(t, "unixgram SetWriteBuffer", gram.SetWriteBuffer(1024))

	clientGramPath := filepath.Join(dir, "client.sock")
	clientGramAddr := &net.UnixAddr{Name: clientGramPath, Net: "unixgram"}
	clientGram, err := net.DialUnix("unixgram", clientGramAddr, gramAddr)
	if err != nil {
		t.Fatalf("DialUnix unixgram error: %v", err)
	}
	defer clientGram.Close()
	t.Cleanup(func() {
		if err := os.Remove(clientGramPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Errorf("Remove(%q): %v", clientGramPath, err)
		}
	})
	if err := clientGram.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Errorf("unixgram client SetDeadline: %v", err)
	}

	buf := make([]byte, 64)
	if _, err := clientGram.Write([]byte("gram1")); err != nil {
		t.Fatalf("unixgram client Write: %v", err)
	}
	if _, _, err := gram.ReadFrom(buf); err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if _, err := clientGram.Write([]byte("gram2")); err != nil {
		t.Fatalf("unixgram client Write second: %v", err)
	}
	clientPeer := new(net.UnixAddr)
	if n, peer, err := gram.ReadFromUnix(buf); err != nil {
		t.Fatalf("ReadFromUnix: %v", err)
	} else {
		if n == 0 {
			t.Fatal("ReadFromUnix returned no data")
		}
		if peer != nil {
			*clientPeer = *peer
		}
	}

	oob := make([]byte, 128)
	if _, _, err := clientGram.WriteMsgUnix([]byte("msg"), nil, nil); err != nil {
		t.Logf("client WriteMsgUnix: %v", err)
	} else {
		if n, oobn, flags, addr, err := gram.ReadMsgUnix(buf, oob); err != nil {
			t.Fatalf("ReadMsgUnix: %v", err)
		} else if n == 0 || addr == nil || oobn < 0 || flags < 0 {
			t.Error("ReadMsgUnix returned unexpected values")
		}
	}

	if _, err := gram.WriteTo([]byte("reply-generic"), clientPeer); err != nil {
		t.Fatalf("gram WriteTo: %v", err)
	}
	if _, err := gram.WriteToUnix([]byte("reply"), clientPeer); err != nil {
		t.Fatalf("gram WriteToUnix reply: %v", err)
	}
	if _, _, err := gram.WriteMsgUnix([]byte("reply-msg"), nil, clientPeer); err != nil {
		t.Fatalf("gram WriteMsgUnix: %v", err)
	}

	buf = buf[:cap(buf)]
	clientGram.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	for i := 0; i < 2; i++ {
		if _, err := clientGram.Read(buf); err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				break
			}
			t.Fatalf("unixgram client Read: %v", err)
		}
	}

	if err := clientGram.CloseRead(); err != nil {
		t.Logf("unixgram CloseRead: %v", err)
	}
	if err := clientGram.CloseWrite(); err != nil {
		t.Logf("unixgram CloseWrite: %v", err)
	}
}
