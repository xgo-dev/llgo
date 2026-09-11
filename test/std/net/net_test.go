package net_test

import (
	"context"
	"io"
	"net"
	"runtime"
	"testing"
	"time"
)

func TestParseIP(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"192.168.1.1", true},
		{"::1", true},
		{"2001:db8::1", true},
		{"invalid", false},
	}
	for _, tt := range tests {
		ip := net.ParseIP(tt.input)
		if (ip != nil) != tt.want {
			t.Errorf("ParseIP(%q) valid = %v, want %v", tt.input, ip != nil, tt.want)
		}
	}
}

func TestIPv4(t *testing.T) {
	ip := net.IPv4(192, 168, 1, 1)
	if ip == nil {
		t.Fatal("IPv4 returned nil")
	}
	if ip.String() != "192.168.1.1" {
		t.Errorf("IPv4().String() = %q, want %q", ip.String(), "192.168.1.1")
	}
}

func TestIPString(t *testing.T) {
	ip := net.ParseIP("192.168.1.1")
	if ip.String() != "192.168.1.1" {
		t.Errorf("IP.String() = %q, want %q", ip.String(), "192.168.1.1")
	}
}

func TestIPEqual(t *testing.T) {
	ip1 := net.ParseIP("192.168.1.1")
	ip2 := net.ParseIP("192.168.1.1")
	ip3 := net.ParseIP("192.168.1.2")
	if !ip1.Equal(ip2) {
		t.Error("Equal IPs not equal")
	}
	if ip1.Equal(ip3) {
		t.Error("Different IPs equal")
	}
}

func TestIPTo4(t *testing.T) {
	ip := net.ParseIP("192.168.1.1")
	ip4 := ip.To4()
	if ip4 == nil {
		t.Error("To4() returned nil for IPv4 address")
	}
}

func TestIPTo16(t *testing.T) {
	ip := net.ParseIP("192.168.1.1")
	ip16 := ip.To16()
	if ip16 == nil {
		t.Error("To16() returned nil")
	}
}

func TestIPDefaultMask(t *testing.T) {
	ip := net.IPv4(192, 168, 1, 1)
	mask := ip.DefaultMask()
	if mask == nil {
		t.Error("DefaultMask() returned nil")
	}
}

func TestIPMask(t *testing.T) {
	ip := net.IPv4(192, 168, 1, 1)
	mask := net.CIDRMask(24, 32)
	masked := ip.Mask(mask)
	want := "192.168.1.0"
	if masked.String() != want {
		t.Errorf("Mask() = %q, want %q", masked.String(), want)
	}
}

func TestIPIsLoopback(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		{"127.0.0.1", true},
		{"::1", true},
		{"192.168.1.1", false},
	}
	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		if got := ip.IsLoopback(); got != tt.want {
			t.Errorf("IP(%q).IsLoopback() = %v, want %v", tt.ip, got, tt.want)
		}
	}
}

func TestIPIsMulticast(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		{"224.0.0.1", true},
		{"ff02::1", true},
		{"192.168.1.1", false},
	}
	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		if got := ip.IsMulticast(); got != tt.want {
			t.Errorf("IP(%q).IsMulticast() = %v, want %v", tt.ip, got, tt.want)
		}
	}
}

func TestIPIsPrivate(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		{"192.168.1.1", true},
		{"10.0.0.1", true},
		{"8.8.8.8", false},
	}
	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		if got := ip.IsPrivate(); got != tt.want {
			t.Errorf("IP(%q).IsPrivate() = %v, want %v", tt.ip, got, tt.want)
		}
	}
}

func TestIPIsUnspecified(t *testing.T) {
	ip := net.ParseIP("0.0.0.0")
	if !ip.IsUnspecified() {
		t.Error("0.0.0.0 IsUnspecified() = false")
	}
}

func TestIPIsGlobalUnicast(t *testing.T) {
	ip := net.ParseIP("8.8.8.8")
	if !ip.IsGlobalUnicast() {
		t.Error("8.8.8.8 IsGlobalUnicast() = false")
	}
}

func TestIPIsLinkLocalUnicast(t *testing.T) {
	ip := net.ParseIP("169.254.1.1")
	if !ip.IsLinkLocalUnicast() {
		t.Error("169.254.1.1 IsLinkLocalUnicast() = false")
	}
}

func TestIPIsLinkLocalMulticast(t *testing.T) {
	ip := net.ParseIP("224.0.0.1")
	if !ip.IsLinkLocalMulticast() {
		t.Error("224.0.0.1 IsLinkLocalMulticast() = false")
	}
}

func TestIPIsInterfaceLocalMulticast(t *testing.T) {
	ip := net.ParseIP("ff01::1")
	if !ip.IsInterfaceLocalMulticast() {
		t.Error("ff01::1 IsInterfaceLocalMulticast() = false")
	}
}

func TestIPMarshalText(t *testing.T) {
	ip := net.ParseIP("192.168.1.1")
	text, err := ip.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText() error = %v", err)
	}
	if string(text) != "192.168.1.1" {
		t.Errorf("MarshalText() = %q, want %q", string(text), "192.168.1.1")
	}
}

func TestIPUnmarshalText(t *testing.T) {
	var ip net.IP
	err := ip.UnmarshalText([]byte("192.168.1.1"))
	if err != nil {
		t.Fatalf("UnmarshalText() error = %v", err)
	}
	if ip.String() != "192.168.1.1" {
		t.Errorf("UnmarshalText() ip = %q, want %q", ip.String(), "192.168.1.1")
	}
}

func TestIPAppendText(t *testing.T) {
	ip := net.ParseIP("192.168.1.1")
	b := []byte("prefix:")
	result, err := ip.AppendText(b)
	if err != nil {
		t.Fatalf("AppendText() error = %v", err)
	}
	if len(result) <= len(b) {
		t.Error("AppendText() did not append data")
	}
}

func TestParseCIDR(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"192.168.1.0/24", false},
		{"2001:db8::/32", false},
		{"invalid", true},
	}
	for _, tt := range tests {
		ip, ipnet, err := net.ParseCIDR(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseCIDR(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && (ip == nil || ipnet == nil) {
			t.Errorf("ParseCIDR(%q) returned nil", tt.input)
		}
	}
}

func TestIPNetContains(t *testing.T) {
	_, ipnet, err := net.ParseCIDR("192.168.1.0/24")
	if err != nil {
		t.Fatalf("ParseCIDR error: %v", err)
	}
	ip1 := net.ParseIP("192.168.1.1")
	ip2 := net.ParseIP("192.168.2.1")
	if !ipnet.Contains(ip1) {
		t.Error("Contains() = false for IP in network")
	}
	if ipnet.Contains(ip2) {
		t.Error("Contains() = true for IP outside network")
	}
}

func TestIPNetString(t *testing.T) {
	_, ipnet, err := net.ParseCIDR("192.168.1.0/24")
	if err != nil {
		t.Fatalf("ParseCIDR error: %v", err)
	}
	got := ipnet.String()
	want := "192.168.1.0/24"
	if got != want {
		t.Errorf("IPNet.String() = %q, want %q", got, want)
	}
}

func TestIPNetNetwork(t *testing.T) {
	_, ipnet, err := net.ParseCIDR("192.168.1.0/24")
	if err != nil {
		t.Fatalf("ParseCIDR error: %v", err)
	}
	network := ipnet.Network()
	if network == "" {
		t.Error("IPNet.Network() returned empty string")
	}
}

func TestParseMAC(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"01:23:45:67:89:ab", false},
		{"01-23-45-67-89-ab", false},
		{"0123.4567.89ab", false},
		{"invalid", true},
	}
	for _, tt := range tests {
		mac, err := net.ParseMAC(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseMAC(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && mac == nil {
			t.Errorf("ParseMAC(%q) returned nil", tt.input)
		}
	}
}

func TestHardwareAddrString(t *testing.T) {
	mac, err := net.ParseMAC("01:23:45:67:89:ab")
	if err != nil {
		t.Fatalf("ParseMAC error: %v", err)
	}
	got := mac.String()
	want := "01:23:45:67:89:ab"
	if got != want {
		t.Errorf("HardwareAddr.String() = %q, want %q", got, want)
	}
}

func TestJoinHostPort(t *testing.T) {
	tests := []struct {
		host string
		port string
		want string
	}{
		{"example.com", "80", "example.com:80"},
		{"192.168.1.1", "8080", "192.168.1.1:8080"},
		{"::1", "80", "[::1]:80"},
	}
	for _, tt := range tests {
		got := net.JoinHostPort(tt.host, tt.port)
		if got != tt.want {
			t.Errorf("JoinHostPort(%q, %q) = %q, want %q", tt.host, tt.port, got, tt.want)
		}
	}
}

func TestSplitHostPort(t *testing.T) {
	tests := []struct {
		hostport string
		wantHost string
		wantPort string
		wantErr  bool
	}{
		{"example.com:80", "example.com", "80", false},
		{"192.168.1.1:8080", "192.168.1.1", "8080", false},
		{"[::1]:80", "::1", "80", false},
		{"invalid", "", "", true},
	}
	for _, tt := range tests {
		host, port, err := net.SplitHostPort(tt.hostport)
		if (err != nil) != tt.wantErr {
			t.Errorf("SplitHostPort(%q) error = %v, wantErr %v", tt.hostport, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && (host != tt.wantHost || port != tt.wantPort) {
			t.Errorf("SplitHostPort(%q) = (%q, %q), want (%q, %q)", tt.hostport, host, port, tt.wantHost, tt.wantPort)
		}
	}
}

func TestResolveTCPAddr(t *testing.T) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:8080")
	if err != nil {
		t.Fatalf("ResolveTCPAddr error: %v", err)
	}
	if addr == nil {
		t.Error("ResolveTCPAddr returned nil")
	}
}

func TestTCPAddrString(t *testing.T) {
	addr := &net.TCPAddr{
		IP:   net.ParseIP("192.168.1.1"),
		Port: 8080,
	}
	got := addr.String()
	if got != "192.168.1.1:8080" {
		t.Errorf("TCPAddr.String() = %q, want %q", got, "192.168.1.1:8080")
	}
}

func TestTCPAddrNetwork(t *testing.T) {
	addr := &net.TCPAddr{
		IP:   net.ParseIP("192.168.1.1"),
		Port: 8080,
	}
	network := addr.Network()
	if network != "tcp" {
		t.Errorf("TCPAddr.Network() = %q, want %q", network, "tcp")
	}
}

func TestResolveUDPAddr(t *testing.T) {
	addr, err := net.ResolveUDPAddr("udp", "localhost:8080")
	if err != nil {
		t.Fatalf("ResolveUDPAddr error: %v", err)
	}
	if addr == nil {
		t.Error("ResolveUDPAddr returned nil")
	}
}

func TestUDPAddrString(t *testing.T) {
	addr := &net.UDPAddr{
		IP:   net.ParseIP("192.168.1.1"),
		Port: 8080,
	}
	got := addr.String()
	if got != "192.168.1.1:8080" {
		t.Errorf("UDPAddr.String() = %q, want %q", got, "192.168.1.1:8080")
	}
}

func TestUDPAddrNetwork(t *testing.T) {
	addr := &net.UDPAddr{
		IP:   net.ParseIP("192.168.1.1"),
		Port: 8080,
	}
	network := addr.Network()
	if network != "udp" {
		t.Errorf("UDPAddr.Network() = %q, want %q", network, "udp")
	}
}

func TestResolveIPAddr(t *testing.T) {
	addr, err := net.ResolveIPAddr("ip", "localhost")
	if err != nil {
		t.Fatalf("ResolveIPAddr error: %v", err)
	}
	if addr == nil {
		t.Error("ResolveIPAddr returned nil")
	}
}

func TestIPAddrString(t *testing.T) {
	addr := &net.IPAddr{
		IP: net.ParseIP("192.168.1.1"),
	}
	got := addr.String()
	if got != "192.168.1.1" {
		t.Errorf("IPAddr.String() = %q, want %q", got, "192.168.1.1")
	}
}

func TestIPAddrNetwork(t *testing.T) {
	addr := &net.IPAddr{
		IP: net.ParseIP("192.168.1.1"),
	}
	network := addr.Network()
	if network != "ip" {
		t.Errorf("IPAddr.Network() = %q, want %q", network, "ip")
	}
}

func TestResolveUnixAddr(t *testing.T) {
	addr, err := net.ResolveUnixAddr("unix", "/tmp/test.sock")
	if err != nil {
		t.Fatalf("ResolveUnixAddr error: %v", err)
	}
	if addr == nil {
		t.Error("ResolveUnixAddr returned nil")
	}
}

func TestUnixAddrString(t *testing.T) {
	addr := &net.UnixAddr{
		Name: "/tmp/test.sock",
		Net:  "unix",
	}
	got := addr.String()
	if got != "/tmp/test.sock" {
		t.Errorf("UnixAddr.String() = %q, want %q", got, "/tmp/test.sock")
	}
}

func TestUnixAddrNetwork(t *testing.T) {
	addr := &net.UnixAddr{
		Name: "/tmp/test.sock",
		Net:  "unix",
	}
	network := addr.Network()
	if network != "unix" {
		t.Errorf("UnixAddr.Network() = %q, want %q", network, "unix")
	}
}

func TestPipe(t *testing.T) {
	c1, c2 := net.Pipe()
	if c1 == nil || c2 == nil {
		t.Fatal("Pipe returned nil")
	}
	defer c1.Close()
	defer c2.Close()

	go func() {
		c1.Write([]byte("hello"))
	}()

	buf := make([]byte, 5)
	n, err := c2.Read(buf)
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	if n != 5 || string(buf) != "hello" {
		t.Errorf("Read = %q, want %q", string(buf), "hello")
	}
}

func TestListenTCP(t *testing.T) {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("ResolveTCPAddr error: %v", err)
	}
	ln, err := net.ListenTCP("tcp", addr)
	if err != nil {
		t.Fatalf("ListenTCP error: %v", err)
	}
	defer ln.Close()
	if ln.Addr() == nil {
		t.Error("ListenTCP returned nil address")
	}
}

func TestTCPListenerAccept(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer ln.Close()

	go func() {
		net.Dial("tcp", ln.Addr().String())
	}()

	conn, err := ln.Accept()
	if err != nil {
		t.Fatalf("Accept error: %v", err)
	}
	defer conn.Close()
}

func TestTCPListenerClose(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	err = ln.Close()
	if err != nil {
		t.Errorf("Close error: %v", err)
	}
}

func TestTCPListenerAddr(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer ln.Close()
	addr := ln.Addr()
	if addr == nil {
		t.Error("Addr() returned nil")
	}
}

func TestDialTCP(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer ln.Close()

	go func() {
		ln.Accept()
	}()

	addr := ln.Addr().(*net.TCPAddr)
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		t.Fatalf("DialTCP error: %v", err)
	}
	defer conn.Close()
}

func TestDial(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer ln.Close()

	go func() {
		ln.Accept()
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	defer conn.Close()
}

func TestDialTimeout(t *testing.T) {
	conn, err := net.DialTimeout("tcp", "192.0.2.1:80", 100*time.Millisecond)
	if err == nil {
		conn.Close()
	}
}

func TestTCPConnClose(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer ln.Close()

	go func() {
		ln.Accept()
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	err = conn.Close()
	if err != nil {
		t.Errorf("Close error: %v", err)
	}
}

func TestTCPConnRead(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			t.Errorf("Accept error: %v", err)
			return
		}
		defer conn.Close()
		if _, err := conn.Write([]byte("hello")); err != nil {
			t.Errorf("server Write error: %v", err)
		}
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	defer conn.Close()

	buf := make([]byte, 5)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	if n != 5 || string(buf) != "hello" {
		t.Errorf("Read = %q, want %q", string(buf), "hello")
	}
}

func TestTCPConnWrite(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer ln.Close()

	done := make(chan error, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			done <- err
			return
		}
		defer conn.Close()
		buf := make([]byte, 5)
		_, err = io.ReadFull(conn, buf)
		done <- err
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	defer conn.Close()

	n, err := conn.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	if n != 5 {
		t.Errorf("Write = %d, want 5", n)
	}

	if err := <-done; err != nil {
		t.Fatalf("server error: %v", err)
	}
}

func TestTCPConnLocalAddr(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer ln.Close()

	go func() {
		ln.Accept()
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	defer conn.Close()

	addr := conn.LocalAddr()
	if addr == nil {
		t.Error("LocalAddr() returned nil")
	}
}

func TestTCPConnRemoteAddr(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer ln.Close()

	go func() {
		ln.Accept()
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	defer conn.Close()

	addr := conn.RemoteAddr()
	if addr == nil {
		t.Error("RemoteAddr() returned nil")
	}
}

func TestTCPConnSetDeadline(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer ln.Close()

	go func() {
		ln.Accept()
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	defer conn.Close()

	err = conn.SetDeadline(time.Now().Add(time.Second))
	if err != nil {
		t.Errorf("SetDeadline error: %v", err)
	}
}

func TestTCPConnSetReadDeadline(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer ln.Close()

	go func() {
		ln.Accept()
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	defer conn.Close()

	err = conn.SetReadDeadline(time.Now().Add(time.Second))
	if err != nil {
		t.Errorf("SetReadDeadline error: %v", err)
	}
}

func TestTCPConnSetWriteDeadline(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer ln.Close()

	go func() {
		ln.Accept()
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	defer conn.Close()

	err = conn.SetWriteDeadline(time.Now().Add(time.Second))
	if err != nil {
		t.Errorf("SetWriteDeadline error: %v", err)
	}
}

func TestListenUDP(t *testing.T) {
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("ResolveUDPAddr error: %v", err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatalf("ListenUDP error: %v", err)
	}
	defer conn.Close()
	if conn.LocalAddr() == nil {
		t.Error("ListenUDP returned nil address")
	}
}

func TestDialUDP(t *testing.T) {
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("ResolveUDPAddr error: %v", err)
	}
	server, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatalf("ListenUDP error: %v", err)
	}
	defer server.Close()
	serverAddr, ok := server.LocalAddr().(*net.UDPAddr)
	if !ok {
		t.Fatalf("LocalAddr type = %T, want *net.UDPAddr", server.LocalAddr())
	}

	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		t.Fatalf("DialUDP error: %v", err)
	}
	defer conn.Close()
}

func TestUDPConnClose(t *testing.T) {
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("ResolveUDPAddr error: %v", err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatalf("ListenUDP error: %v", err)
	}
	err = conn.Close()
	if err != nil {
		t.Errorf("Close error: %v", err)
	}
}

func TestUDPConnLocalAddr(t *testing.T) {
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("ResolveUDPAddr error: %v", err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatalf("ListenUDP error: %v", err)
	}
	defer conn.Close()
	if conn.LocalAddr() == nil {
		t.Error("LocalAddr() returned nil")
	}
}

func TestListen(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer ln.Close()
	if ln.Addr() == nil {
		t.Error("Listen returned nil address")
	}
}

func TestListenPacket(t *testing.T) {
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("ListenPacket error: %v", err)
	}
	defer pc.Close()
	if pc.LocalAddr() == nil {
		t.Error("ListenPacket returned nil address")
	}
}

func TestInterfaces(t *testing.T) {
	ifaces, err := net.Interfaces()
	if checkWasmInterfaces(t, ifaces, err) {
		return
	}
	if err != nil {
		t.Fatalf("Interfaces error: %v", err)
	}
	if len(ifaces) == 0 {
		t.Skip("No interfaces found")
	}
}

func TestInterfaceByName(t *testing.T) {
	ifaces, err := net.Interfaces()
	if checkWasmInterfaces(t, ifaces, err) {
		iface, err := net.InterfaceByName("llgo-no-such-interface")
		checkMissingWasmInterface(t, iface, err)
		return
	}
	if err != nil || len(ifaces) == 0 {
		t.Skip("No interfaces found")
	}
	iface, err := net.InterfaceByName(ifaces[0].Name)
	if err != nil {
		t.Fatalf("InterfaceByName error: %v", err)
	}
	if iface == nil {
		t.Error("InterfaceByName returned nil")
	}
}

func TestInterfaceByIndex(t *testing.T) {
	ifaces, err := net.Interfaces()
	if checkWasmInterfaces(t, ifaces, err) {
		iface, err := net.InterfaceByIndex(1)
		checkMissingWasmInterface(t, iface, err)
		return
	}
	if err != nil || len(ifaces) == 0 {
		t.Skip("No interfaces found")
	}
	iface, err := net.InterfaceByIndex(ifaces[0].Index)
	if err != nil {
		t.Fatalf("InterfaceByIndex error: %v", err)
	}
	if iface == nil {
		t.Error("InterfaceByIndex returned nil")
	}
}

func TestInterfaceAddrs(t *testing.T) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		t.Fatalf("InterfaceAddrs error: %v", err)
	}
	if runtime.GOARCH == "wasm" {
		if len(addrs) != 0 {
			t.Fatalf("wasm InterfaceAddrs = %v, want empty list", addrs)
		}
		return
	}
	if len(addrs) == 0 {
		t.Skip("No interface addresses found")
	}
}

func TestInterfaceAddrsMethod(t *testing.T) {
	ifaces, err := net.Interfaces()
	if checkWasmInterfaces(t, ifaces, err) {
		iface := &net.Interface{Index: 1}
		addrs, err := iface.Addrs()
		if err != nil || len(addrs) != 0 {
			t.Fatalf("wasm Interface.Addrs = %v, %v; want empty list, nil", addrs, err)
		}
		return
	}
	if err != nil || len(ifaces) == 0 {
		t.Skip("No interfaces found")
	}
	addrs, err := ifaces[0].Addrs()
	if err != nil {
		t.Fatalf("Interface.Addrs error: %v", err)
	}
	_ = addrs
}

func TestInterfaceMulticastAddrs(t *testing.T) {
	ifaces, err := net.Interfaces()
	if checkWasmInterfaces(t, ifaces, err) {
		iface := &net.Interface{Index: 1}
		addrs, err := iface.MulticastAddrs()
		if err != nil || len(addrs) != 0 {
			t.Fatalf("wasm Interface.MulticastAddrs = %v, %v; want empty list, nil", addrs, err)
		}
		return
	}
	if err != nil || len(ifaces) == 0 {
		t.Skip("No interfaces found")
	}
	addrs, err := ifaces[0].MulticastAddrs()
	if err != nil {
		t.Fatalf("Interface.MulticastAddrs error: %v", err)
	}
	_ = addrs
}

func TestFlagsString(t *testing.T) {
	flags := net.FlagUp | net.FlagLoopback
	s := flags.String()
	if s == "" {
		t.Error("Flags.String() returned empty string")
	}
}

func TestDialer(t *testing.T) {
	d := &net.Dialer{
		Timeout: time.Second,
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer ln.Close()

	go func() {
		ln.Accept()
	}()

	conn, err := d.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("Dialer.Dial error: %v", err)
	}
	defer conn.Close()
}

func TestDialerDialContext(t *testing.T) {
	d := &net.Dialer{}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer ln.Close()

	go func() {
		ln.Accept()
	}()

	ctx := context.Background()
	conn, err := d.DialContext(ctx, "tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("Dialer.DialContext error: %v", err)
	}
	defer conn.Close()
}

func TestDialerMultipathTCP(t *testing.T) {
	d := &net.Dialer{}
	d.SetMultipathTCP(true)
	if !d.MultipathTCP() {
		t.Error("Dialer.MultipathTCP() = false after SetMultipathTCP(true)")
	}
}

func TestBuffersRead(t *testing.T) {
	buffers := net.Buffers{
		[]byte("hello"),
		[]byte("world"),
	}
	buf := make([]byte, 10)
	n, err := buffers.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("Buffers.Read error: %v", err)
	}
	if n > 10 {
		t.Errorf("Read too many bytes: %d", n)
	}
}

func TestBuffersWriteTo(t *testing.T) {
	buffers := net.Buffers{
		[]byte("hello"),
		[]byte("world"),
	}
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	go func() {
		buffers.WriteTo(c1)
	}()

	buf := make([]byte, 10)
	n, _ := c2.Read(buf)
	if n == 0 {
		t.Error("WriteTo wrote no data")
	}
}

func TestAddrError(t *testing.T) {
	e := &net.AddrError{
		Err:  "test error",
		Addr: "127.0.0.1",
	}
	got := e.Error()
	if got == "" {
		t.Error("AddrError.Error() returned empty string")
	}
	if e.Timeout() {
		t.Error("AddrError.Timeout() = true")
	}
	if e.Temporary() {
		t.Error("AddrError.Temporary() = true")
	}
}

func TestDNSError(t *testing.T) {
	e := &net.DNSError{
		Err:  "test error",
		Name: "example.com",
	}
	got := e.Error()
	if got == "" {
		t.Error("DNSError.Error() returned empty string")
	}
}

func TestDNSConfigError(t *testing.T) {
	e := &net.DNSConfigError{
		Err: &net.DNSError{Err: "test"},
	}
	got := e.Error()
	if got == "" {
		t.Error("DNSConfigError.Error() returned empty string")
	}
}

func TestDefaultResolver(t *testing.T) {
	if net.DefaultResolver == nil {
		t.Error("DefaultResolver is nil")
	}
}

func TestResolverLookupHost(t *testing.T) {
	r := net.DefaultResolver
	addrs, err := r.LookupHost(context.Background(), "localhost")
	if err != nil {
		t.Fatalf("LookupHost error: %v", err)
	}
	if len(addrs) == 0 {
		t.Error("LookupHost returned no addresses")
	}
}

func TestResolverLookupAddr(t *testing.T) {
	r := net.DefaultResolver
	names, err := r.LookupAddr(context.Background(), "127.0.0.1")
	if err != nil {
		t.Skip("LookupAddr failed (may not have reverse DNS)")
	}
	_ = names
}

func TestResolverLookupCNAME(t *testing.T) {
	r := net.DefaultResolver
	cname, err := r.LookupCNAME(context.Background(), "localhost")
	if err != nil {
		t.Skipf("LookupCNAME error (DNS configuration issue): %v", err)
	}
	if cname == "" {
		t.Error("LookupCNAME returned empty string")
	}
}

func TestLookupHost(t *testing.T) {
	addrs, err := net.LookupHost("localhost")
	if err != nil {
		t.Fatalf("LookupHost error: %v", err)
	}
	if len(addrs) == 0 {
		t.Error("LookupHost returned no addresses")
	}
}

func TestLookupIP(t *testing.T) {
	ips, err := net.LookupIP("localhost")
	if err != nil {
		t.Fatalf("LookupIP error: %v", err)
	}
	if len(ips) == 0 {
		t.Error("LookupIP returned no IPs")
	}
}

func TestLookupAddr(t *testing.T) {
	names, err := net.LookupAddr("127.0.0.1")
	if err != nil {
		t.Skip("LookupAddr failed (may not have reverse DNS)")
	}
	_ = names
}

func TestLookupCNAME(t *testing.T) {
	cname, err := net.LookupCNAME("localhost")
	if err != nil {
		t.Skipf("LookupCNAME error (DNS configuration issue): %v", err)
	}
	if cname == "" {
		t.Error("LookupCNAME returned empty string")
	}
}

func TestLookupPort(t *testing.T) {
	port, err := net.LookupPort("tcp", "http")
	if err != nil {
		t.Fatalf("LookupPort error: %v", err)
	}
	if port != 80 {
		t.Errorf("LookupPort() = %d, want 80", port)
	}
}

func TestLookupTXT(t *testing.T) {
	records, err := net.LookupTXT("localhost")
	if err != nil {
		t.Skip("LookupTXT failed")
	}
	_ = records
}

func TestLookupMX(t *testing.T) {
	records, err := net.LookupMX("localhost")
	if err != nil {
		t.Skip("LookupMX failed")
	}
	_ = records
}

func TestLookupNS(t *testing.T) {
	records, err := net.LookupNS("localhost")
	if err != nil {
		t.Skip("LookupNS failed")
	}
	_ = records
}

func TestLookupSRV(t *testing.T) {
	cname, records, err := net.LookupSRV("xmpp-server", "tcp", "localhost")
	if err != nil {
		t.Skip("LookupSRV failed")
	}
	_, _ = cname, records
}

func TestIPMaskSize(t *testing.T) {
	mask := net.CIDRMask(24, 32)
	ones, bits := mask.Size()
	if ones != 24 || bits != 32 {
		t.Errorf("Size() = (%d, %d), want (24, 32)", ones, bits)
	}
}

func TestIPMaskString(t *testing.T) {
	mask := net.CIDRMask(24, 32)
	s := mask.String()
	if s == "" {
		t.Error("IPMask.String() returned empty string")
	}
}

func TestCIDRMask(t *testing.T) {
	mask := net.CIDRMask(24, 32)
	if mask == nil {
		t.Error("CIDRMask returned nil")
	}
}

func TestIPv4Mask(t *testing.T) {
	mask := net.IPv4Mask(255, 255, 255, 0)
	if mask == nil {
		t.Error("IPv4Mask returned nil")
	}
}

func TestErrClosed(t *testing.T) {
	if net.ErrClosed == nil {
		t.Error("ErrClosed is nil")
	}
}

func TestIPv4Constants(t *testing.T) {
	if net.IPv4len != 4 {
		t.Errorf("IPv4len = %d, want 4", net.IPv4len)
	}
	if net.IPv6len != 16 {
		t.Errorf("IPv6len = %d, want 16", net.IPv6len)
	}
	if net.IPv4zero == nil {
		t.Error("IPv4zero is nil")
	}
	if net.IPv4bcast == nil {
		t.Error("IPv4bcast is nil")
	}
	if net.IPv4allsys == nil {
		t.Error("IPv4allsys is nil")
	}
	if net.IPv4allrouter == nil {
		t.Error("IPv4allrouter is nil")
	}
	if net.IPv6zero == nil {
		t.Error("IPv6zero is nil")
	}
	if net.IPv6unspecified == nil {
		t.Error("IPv6unspecified is nil")
	}
	if net.IPv6loopback == nil {
		t.Error("IPv6loopback is nil")
	}
	if net.IPv6interfacelocalallnodes == nil {
		t.Error("IPv6interfacelocalallnodes is nil")
	}
	if net.IPv6linklocalallnodes == nil {
		t.Error("IPv6linklocalallnodes is nil")
	}
	if net.IPv6linklocalallrouters == nil {
		t.Error("IPv6linklocalallrouters is nil")
	}
}

func TestTCPConnMethods(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer ln.Close()

	go func() {
		conn, _ := ln.Accept()
		if conn != nil {
			conn.Close()
		}
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	tcpConn := conn.(*net.TCPConn)
	defer tcpConn.Close()

	if err := tcpConn.SetKeepAlive(true); err != nil {
		t.Errorf("SetKeepAlive error: %v", err)
	}

	if err := tcpConn.SetKeepAlivePeriod(time.Second); err != nil {
		t.Errorf("SetKeepAlivePeriod error: %v", err)
	}

	if err := tcpConn.SetLinger(0); err != nil {
		t.Errorf("SetLinger error: %v", err)
	}

	if err := tcpConn.SetNoDelay(true); err != nil {
		t.Errorf("SetNoDelay error: %v", err)
	}

	if err := tcpConn.SetReadBuffer(4096); err != nil {
		t.Errorf("SetReadBuffer error: %v", err)
	}

	if err := tcpConn.SetWriteBuffer(4096); err != nil {
		t.Errorf("SetWriteBuffer error: %v", err)
	}

	// CloseRead/CloseWrite may fail on some platforms if connection is not established
	if err := tcpConn.CloseRead(); err != nil {
		t.Logf("CloseRead error (may be platform-specific): %v", err)
	}

	if err := tcpConn.CloseWrite(); err != nil {
		t.Logf("CloseWrite error (may be platform-specific): %v", err)
	}
}

func TestTCPAddrAddrPort(t *testing.T) {
	addr := &net.TCPAddr{
		IP:   net.ParseIP("192.168.1.1"),
		Port: 8080,
	}
	ap := addr.AddrPort()
	if !ap.IsValid() {
		t.Error("TCPAddr.AddrPort() returned invalid AddrPort")
	}
}

func TestUDPAddrAddrPort(t *testing.T) {
	addr := &net.UDPAddr{
		IP:   net.ParseIP("192.168.1.1"),
		Port: 8080,
	}
	ap := addr.AddrPort()
	if !ap.IsValid() {
		t.Error("UDPAddr.AddrPort() returned invalid AddrPort")
	}
}

func TestUDPConnMethods(t *testing.T) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatalf("ListenUDP error: %v", err)
	}
	defer conn.Close()

	if err := conn.SetReadBuffer(4096); err != nil {
		t.Errorf("SetReadBuffer error: %v", err)
	}

	if err := conn.SetWriteBuffer(4096); err != nil {
		t.Errorf("SetWriteBuffer error: %v", err)
	}
}

func TestTCPListenerMethods(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer ln.Close()

	tcpLn := ln.(*net.TCPListener)

	if err := tcpLn.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Errorf("SetDeadline error: %v", err)
	}
}

func TestMXType(t *testing.T) {
	mx := &net.MX{
		Host: "mail.example.com",
		Pref: 10,
	}
	if mx.Host != "mail.example.com" {
		t.Error("MX.Host not set correctly")
	}
	if mx.Pref != 10 {
		t.Error("MX.Pref not set correctly")
	}
}

func TestNSType(t *testing.T) {
	ns := &net.NS{
		Host: "ns1.example.com",
	}
	if ns.Host != "ns1.example.com" {
		t.Error("NS.Host not set correctly")
	}
}

func TestSRVType(t *testing.T) {
	srv := &net.SRV{
		Target:   "target.example.com",
		Port:     8080,
		Priority: 10,
		Weight:   20,
	}
	if srv.Target != "target.example.com" {
		t.Error("SRV.Target not set correctly")
	}
}

func TestResolverLookupMethods(t *testing.T) {
	r := net.DefaultResolver
	ctx := context.Background()

	_, err := r.LookupIPAddr(ctx, "localhost")
	if err != nil {
		t.Errorf("LookupIPAddr error: %v", err)
	}

	_, err = r.LookupNetIP(ctx, "ip", "localhost")
	if err != nil {
		t.Errorf("LookupNetIP error: %v", err)
	}

	_, err = r.LookupPort(ctx, "tcp", "http")
	if err != nil {
		t.Errorf("LookupPort error: %v", err)
	}

	_, err = r.LookupMX(ctx, "localhost")
	if err != nil {
		t.Skip("LookupMX failed")
	}

	_, err = r.LookupNS(ctx, "localhost")
	if err != nil {
		t.Skip("LookupNS failed")
	}

	_, err = r.LookupTXT(ctx, "localhost")
	if err != nil {
		t.Skip("LookupTXT failed")
	}

	_, _, err = r.LookupSRV(ctx, "xmpp-server", "tcp", "localhost")
	if err != nil {
		t.Skip("LookupSRV failed")
	}

	_, err = r.LookupIP(ctx, "ip4", "localhost")
	if err != nil {
		t.Errorf("LookupIP error: %v", err)
	}
}

func TestInvalidAddrError(t *testing.T) {
	var e net.InvalidAddrError = "invalid"
	if e.Error() == "" {
		t.Error("InvalidAddrError.Error() returned empty string")
	}
	if e.Timeout() {
		t.Error("InvalidAddrError.Timeout() = true")
	}
	if e.Temporary() {
		t.Error("InvalidAddrError.Temporary() = true")
	}
}

func TestUnknownNetworkError(t *testing.T) {
	var e net.UnknownNetworkError = "unknown"
	if e.Error() == "" {
		t.Error("UnknownNetworkError.Error() returned empty string")
	}
	if e.Timeout() {
		t.Error("UnknownNetworkError.Timeout() = true")
	}
	if e.Temporary() {
		t.Error("UnknownNetworkError.Temporary() = true")
	}
}

func TestParseError(t *testing.T) {
	e := &net.ParseError{
		Type: "IP address",
		Text: "invalid",
	}
	if e.Error() == "" {
		t.Error("ParseError.Error() returned empty string")
	}
	if e.Timeout() {
		t.Error("ParseError.Timeout() = true")
	}
	if e.Temporary() {
		t.Error("ParseError.Temporary() = true")
	}
}

func TestOpError(t *testing.T) {
	e := &net.OpError{
		Op:   "dial",
		Net:  "tcp",
		Addr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8080},
		Err:  io.EOF,
	}
	if e.Error() == "" {
		t.Error("OpError.Error() returned empty string")
	}
	if e.Timeout() {
		t.Error("OpError.Timeout() = true for non-timeout error")
	}
	if e.Temporary() {
		t.Error("OpError.Temporary() = true for non-temporary error")
	}
	if e.Unwrap() == nil {
		t.Error("OpError.Unwrap() returned nil")
	}
}

func TestDNSErrorMethods(t *testing.T) {
	e := &net.DNSError{
		Err:  "test error",
		Name: "example.com",
	}
	if e.Timeout() {
		t.Error("DNSError.Timeout() = true for non-timeout error")
	}
	if e.Temporary() {
		t.Error("DNSError.Temporary() = true for non-temporary error")
	}
	if e.Unwrap() != nil {
		t.Error("DNSError.Unwrap() should be nil for plain DNSError")
	}
}

func TestDNSConfigErrorMethods(t *testing.T) {
	e := &net.DNSConfigError{
		Err: &net.DNSError{Err: "test"},
	}
	if e.Timeout() {
		t.Error("DNSConfigError.Timeout() = true for non-timeout error")
	}
	if e.Temporary() {
		t.Error("DNSConfigError.Temporary() = true for non-temporary error")
	}
	if e.Unwrap() == nil {
		t.Error("DNSConfigError.Unwrap() returned nil")
	}
}

func TestListenConfig(t *testing.T) {
	lc := &net.ListenConfig{}
	lc.SetMultipathTCP(true)
	if !lc.MultipathTCP() {
		t.Error("MultipathTCP() = false after SetMultipathTCP(true)")
	}

	ln, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("ListenConfig.Listen error: %v", err)
	}
	defer ln.Close()

	pc, err := lc.ListenPacket(context.Background(), "udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("ListenConfig.ListenPacket error: %v", err)
	}
	defer pc.Close()
}

func TestErrWriteToConnected(t *testing.T) {
	if net.ErrWriteToConnected == nil {
		t.Error("ErrWriteToConnected is nil")
	}
}

func TestKeepAliveConfig(t *testing.T) {
	kac := net.KeepAliveConfig{
		Enable:   true,
		Idle:     time.Second,
		Interval: time.Second,
		Count:    3,
	}
	if !kac.Enable {
		t.Error("KeepAliveConfig.Enable = false")
	}
}

func TestTCPConnKeepAliveConfig(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer ln.Close()

	go func() {
		conn, _ := ln.Accept()
		if conn != nil {
			conn.Close()
		}
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	tcpConn := conn.(*net.TCPConn)
	defer tcpConn.Close()

	kac := net.KeepAliveConfig{
		Enable: true,
		Idle:   time.Second,
	}
	if err := tcpConn.SetKeepAliveConfig(kac); err != nil {
		t.Errorf("SetKeepAliveConfig error: %v", err)
	}
}

func TestTCPConnMultipathTCP(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer ln.Close()

	go func() {
		conn, _ := ln.Accept()
		if conn != nil {
			conn.Close()
		}
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("Dial error: %v", err)
	}
	tcpConn := conn.(*net.TCPConn)
	defer tcpConn.Close()

	if _, err := tcpConn.MultipathTCP(); err != nil {
		t.Logf("TCPConn.MultipathTCP() error: %v", err)
	}
}

func TestTCPListenerAcceptTCP(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer ln.Close()

	tcpLn := ln.(*net.TCPListener)

	go func() {
		net.Dial("tcp", ln.Addr().String())
	}()

	conn, err := tcpLn.AcceptTCP()
	if err != nil {
		t.Fatalf("AcceptTCP error: %v", err)
	}
	defer conn.Close()
}
