package net_test

import (
	"errors"
	"net"
	"net/netip"
	"os"
	"testing"
	"time"
)

func TestUDPConnMethodCoverage(t *testing.T) {
	server, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatalf("ListenUDP error: %v", err)
	}
	t.Cleanup(func() {
		if err := server.Close(); err != nil {
			t.Errorf("server.Close: %v", err)
		}
	})

	checkSocketOption(t, "server SetReadBuffer", server.SetReadBuffer(4096))
	checkSocketOption(t, "server SetWriteBuffer", server.SetWriteBuffer(4096))
	if err := server.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Errorf("server SetDeadline: %v", err)
	}
	if err := server.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Errorf("server SetReadDeadline: %v", err)
	}
	if err := server.SetWriteDeadline(time.Now().Add(time.Second)); err != nil {
		t.Errorf("server SetWriteDeadline: %v", err)
	}
	if server.LocalAddr() == nil {
		t.Fatal("server LocalAddr returned nil")
	}
	if raw, err := server.SyscallConn(); err == nil {
		if err := raw.Control(func(uintptr) {}); err != nil {
			t.Logf("server SyscallConn control: %v", err)
		}
	} else {
		t.Logf("server SyscallConn unavailable: %v", err)
	}
	if f, err := server.File(); err == nil {
		defer f.Close()
		if pc, err := net.FilePacketConn(f); err == nil {
			pc.Close()
		} else {
			t.Logf("FilePacketConn: %v", err)
		}
	} else {
		t.Logf("server File unavailable: %v", err)
	}

	serverAddr := server.LocalAddr().(*net.UDPAddr)
	client, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		t.Fatalf("DialUDP error: %v", err)
	}
	defer client.Close()

	checkSocketOption(t, "client SetReadBuffer", client.SetReadBuffer(4096))
	checkSocketOption(t, "client SetWriteBuffer", client.SetWriteBuffer(4096))
	if err := client.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Errorf("client SetDeadline: %v", err)
	}
	if err := client.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Errorf("client SetReadDeadline: %v", err)
	}
	if err := client.SetWriteDeadline(time.Now().Add(time.Second)); err != nil {
		t.Errorf("client SetWriteDeadline: %v", err)
	}
	if client.LocalAddr() == nil {
		t.Fatal("client LocalAddr returned nil")
	}
	if client.RemoteAddr() == nil {
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
		if pc, err := net.FilePacketConn(f); err == nil {
			pc.Close()
		} else {
			t.Logf("client FilePacketConn: %v", err)
		}
	} else {
		t.Logf("client File unavailable: %v", err)
	}

	if _, err := client.Write([]byte("hello")); err != nil {
		t.Fatalf("client Write: %v", err)
	}

	buf := make([]byte, 64)
	n, addr, err := server.ReadFrom(buf)
	if err != nil {
		t.Fatalf("server ReadFrom: %v", err)
	}
	if n == 0 || addr.String() == "" {
		t.Error("server ReadFrom returned no data")
	}

	if _, err := client.Write([]byte("world")); err != nil {
		t.Fatalf("client Write second: %v", err)
	}
	buf = buf[:cap(buf)]
	n, udpAddr, err := server.ReadFromUDP(buf)
	if err != nil {
		t.Fatalf("server ReadFromUDP: %v", err)
	}
	if n == 0 || udpAddr == nil {
		t.Error("server ReadFromUDP missing data")
	}

	if _, err := client.Write([]byte("addrport")); err != nil {
		t.Fatalf("client Write third: %v", err)
	}
	buf = buf[:cap(buf)]
	n, ap, err := server.ReadFromUDPAddrPort(buf)
	if err != nil {
		t.Fatalf("server ReadFromUDPAddrPort: %v", err)
	}
	if n == 0 || !ap.IsValid() {
		t.Error("server ReadFromUDPAddrPort missing data")
	}

	oob := make([]byte, 128)
	if _, err := client.Write([]byte("msg-readmsg")); err != nil {
		t.Fatalf("client Write for ReadMsgUDP: %v", err)
	}
	buf = buf[:cap(buf)]
	if n, _, _, addr, err := server.ReadMsgUDP(buf, oob); err != nil {
		t.Fatalf("ReadMsgUDP: %v", err)
	} else if n == 0 || addr == nil {
		t.Error("ReadMsgUDP missing data")
	}

	if _, _, err := server.WriteMsgUDP([]byte("srv-msg"), nil, client.LocalAddr().(*net.UDPAddr)); err != nil {
		t.Fatalf("WriteMsgUDP: %v", err)
	}
	buf = buf[:cap(buf)]
	if n, _, _, addr, err := client.ReadMsgUDP(buf, oob); err != nil {
		t.Fatalf("client ReadMsgUDP: %v", err)
	} else if n == 0 || addr == nil {
		t.Error("client ReadMsgUDP missing data")
	}

	clientAddr := client.LocalAddr().(*net.UDPAddr)
	if _, err := server.WriteTo([]byte("reply"), clientAddr); err != nil {
		t.Fatalf("server WriteTo: %v", err)
	}
	if _, err := server.WriteToUDP([]byte("reply2"), clientAddr); err != nil {
		t.Fatalf("server WriteToUDP: %v", err)
	}

	if clientAP, ok := netip.AddrFromSlice(clientAddr.IP); ok {
		target := netip.AddrPortFrom(clientAP, uint16(clientAddr.Port))
		if _, err := server.WriteToUDPAddrPort([]byte("reply3"), target); err != nil {
			t.Fatalf("server WriteToUDPAddrPort: %v", err)
		}
		if _, _, err := server.WriteMsgUDPAddrPort([]byte("srv-msg-ap"), nil, target); err != nil {
			t.Fatalf("WriteMsgUDPAddrPort: %v", err)
		}
		buf = buf[:cap(buf)]
		if n, _, _, from, err := client.ReadMsgUDPAddrPort(buf, oob); err != nil {
			t.Fatalf("ReadMsgUDPAddrPort: %v", err)
		} else if n == 0 || !from.IsValid() {
			t.Error("ReadMsgUDPAddrPort missing data")
		}
	}

	buf = buf[:cap(buf)]
	client.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	for i := 0; i < 3; i++ {
		if _, err := client.Read(buf); err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				break
			}
			t.Fatalf("client Read: %v", err)
		}
	}
}
