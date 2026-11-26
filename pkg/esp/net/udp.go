// UDP datagram sockets over the same lwip socket layer and the same
// bounded-quantum model as the TCP types in net.go. The C-side inventory
// grows by four primitives (llgo_net_dgram_socket, llgo_net_bind,
// llgo_net_sendto, llgo_net_recvfrom); poll, close, sock_error, send, recv
// and the deadline clock are shared with the TCP path unchanged.
//
// # Datagram semantics
//
// A datagram is atomic: every send hands the whole buffer to lwip in one
// syscall (a datagram socket never reports a short positive count), and
// every receive drains one datagram. Datagrams are never joined and never
// split by this package, but recvfrom truncates at the caller's buffer: a
// datagram longer than len(p) is cut to len(p) bytes and the remainder is
// discarded by lwip (plain POSIX recvfrom semantics - there is no way to
// read the tail afterwards). Size receive buffers at or above the largest
// datagram the peer may send. A zero-length datagram is reported as
// (0, addr, nil) from ReadFrom and (0, nil) from Read - UDP has no EOF, so
// that shape is a real (empty) datagram here, never a close.
//
// # The connected form
//
// DialUDP installs a fixed peer (lwip udp_connect), which lets Read and
// Write drop the address bookkeeping - the same shapes TCPConn exposes.
// Read and Write on a socket that was not connected (ListenUDP without
// DialUDP) are rejected with ENOTCONN; use ReadFrom and WriteTo there.
// WriteTo and ReadFrom work on both forms (sendto/recvfrom with an explicit
// address override the fixed peer per call).
//
// One connected-form subtlety is asynchronous: if a datagram sent earlier
// draws an ICMP port-unreachable, lwip does not fail the send - the error
// surfaces on the next receive as ECONNREFUSED. The first such ECONNREFUSED
// after DialUDP is swallowed (treated like EAGAIN: one quantum wait, then
// the loop retries), because it reports a datagram that is already gone
// rather than something the current receive can act on; a second
// ECONNREFUSED is a real error. The same swallow would also absorb a
// refused error caused by WriteTo traffic on a socket that was connected
// later - the flag is per connection, not per direction.
//
// # Errors and deadlines
//
// Errors are Go error values of the shared esperr type carrying the same
// raw lwip errnos as the TCP side (named by ErrName). The transport
// sentinels apply unchanged: ErrWouldBlock for the poll-style read without
// a deadline, ErrTimeout on deadline expiry, ErrClosed for calls on a
// closed or nil socket - Close is idempotent. Deadlines are
// UDPConn.SetDeadline's relative milliseconds, identical in shape to
// TCPConn's: 0 clears, < 0 is EINVAL, and expiry reports ErrTimeout.
// Without a deadline a receive gives up after one quantum with
// ErrWouldBlock (a poll-style read) and a send reports (0, ErrWouldBlock)
// after one quantum of no progress - every wait stays bounded, the
// package-wide contract.
package net

// Addr is a UDP endpoint in the package's address convention: IP is the
// in-memory network-order word IPv4 produces (and wifi.GetIP exchanges),
// Port the host-order port number.
type Addr struct {
	IP   uint32
	Port uint16
}

// String renders the address as "1.2.3.4:9000" - octets in the order a
// person writes them, which for the in-memory form means low byte first
// (the same order wifi.IPString prints).
func (a Addr) String() string {
	const digits = "0123456789"
	var buf [21]byte // "255.255.255.255:65535" fits exactly
	pos := 0
	for i := 0; i < 4; i++ {
		if i > 0 {
			buf[pos] = '.'
			pos++
		}
		o := uint32(a.IP>>(8*uint(i))) & 0xff
		if o >= 100 {
			buf[pos] = digits[o/100]
			pos++
		}
		if o >= 10 {
			buf[pos] = digits[(o/10)%10]
			pos++
		}
		buf[pos] = digits[o%10]
		pos++
	}
	buf[pos] = ':'
	pos++
	p := uint32(a.Port)
	if p >= 10000 {
		buf[pos] = digits[p/10000]
		pos++
	}
	if p >= 1000 {
		buf[pos] = digits[(p/1000)%10]
		pos++
	}
	if p >= 100 {
		buf[pos] = digits[(p/100)%10]
		pos++
	}
	if p >= 10 {
		buf[pos] = digits[(p/10)%10]
		pos++
	}
	buf[pos] = digits[p%10]
	pos++
	return string(buf[:pos])
}

// UDPConn is a UDP datagram socket, bound to a local address by ListenUDP
// and optionally fixed to a remote peer by DialUDP. Internals mirror
// TCPConn (an fd plus the shared socketDeadline bookkeeping) plus the
// one-shot ECONNREFUSED swallow flag the connected form needs.
type UDPConn struct {
	fd int32
	socketDeadline

	// swallowRefused is set by DialUDP: the first ECONNREFUSED a receive
	// sees on the connected socket is an asynchronous leftover from an
	// earlier send, not a current-read error, and is absorbed once.
	swallowRefused bool
}

// ListenUDP opens an unconnected UDP socket bound to laddr. A nil laddr
// binds the wildcard address with an ephemeral port (like the TCP side,
// the wildcard covers every local IP: the station address and, with
// LWIP_NETIF_LOOPBACK, 127.0.0.1). The socket is non-blocking from birth
// and its deadline starts unset.
func ListenUDP(laddr *Addr) (*UDPConn, error) {
	var fd int32
	if e := netDgramSocket(&fd); e != 0 {
		return nil, errOf(e)
	}
	ip, port := uint32(0), int32(0) // 0/0: wildcard address, ephemeral port
	if laddr != nil {
		ip, port = laddr.IP, int32(laddr.Port)
	}
	if e := netBind(fd, ip, port); e != 0 {
		netClose(fd)
		return nil, errOf(e)
	}
	return &UDPConn{fd: fd}, nil
}

// DialUDP opens a connected UDP socket: bound locally to laddr (nil = the
// wildcard with an ephemeral port) and fixed to raddr, which must not be
// nil (EINVAL - a connected form without a peer has no meaning). UDP
// connect is a local PCB operation that completes immediately - no
// in-flight state, no timeout parameter - but it is not validated against
// the network: a peer that is down only shows up when a datagram's ICMP
// refusal comes back (as the swallowed ECONNREFUSED, see the package doc).
// The deadline starts unset, like every socket this package hands out.
func DialUDP(laddr, raddr *Addr) (*UDPConn, error) {
	if raddr == nil {
		return nil, errOf(einval)
	}
	c, e := ListenUDP(laddr)
	if e != nil {
		return nil, e
	}
	if e := netConnect(c.fd, raddr.IP, int32(raddr.Port)); e != 0 {
		c.Close()
		return nil, errOf(e)
	}
	c.swallowRefused = true
	return c, nil
}

// ReadFrom receives one datagram into p and reports the sender's address.
// The datagram is truncated at len(p) (the discarded tail is gone - size p
// for the peer's largest datagram, see the package doc). The outcomes:
//
//	n > 0, sender, nil  a datagram of n bytes arrived
//	0, sender, nil      a zero-length datagram arrived (UDP has no EOF)
//	0, Addr{}, ErrWouldBlock
//	                    no deadline set and nothing arrived within one
//	                    quantum - a poll-style read
//	0, Addr{}, ErrTimeout
//	                    a deadline was set and passed with no datagram
//
// A zero-length p is a no-op success (a zero-size receive cannot store
// even the smallest datagram, and recvfrom with a NULL buffer is not a
// call this package makes).
func (c *UDPConn) ReadFrom(p []byte) (int, Addr, error) {
	if c == nil || c.fd < 0 {
		return 0, Addr{}, ErrClosed
	}
	if len(p) == 0 {
		return 0, Addr{}, nil
	}
	for {
		var n int32
		var ip uint32
		var port int32
		e := netRecvFrom(c.fd, &p[0], int32(len(p)), &ip, &port, &n)
		if e == 0 {
			return int(n), Addr{IP: ip, Port: uint16(port)}, nil
		}
		if e != eagain {
			return 0, Addr{}, errOf(e)
		}
		if pe := c.pace(false); pe != 0 {
			return 0, Addr{}, errOf(pe)
		}
	}
}

// WriteTo sends p as one datagram to addr, retrying past EAGAIN the same
// way TCPConn.Write retries: with a deadline set, until the deadline passes
// (then (0, ErrTimeout)); without one, one quantum, then (0,
// ErrWouldBlock). The datagram is atomic - it either goes out whole or not
// at all, so success returns (len(p), nil) and there is no short positive
// send to resume; any other err carries the errno (e.g. ENOBUFS 105 when
// lwip has run out of buffers, ENETUNREACH 114 before the station has an
// IP). A zero-length p is a no-op success like on the TCP side.
func (c *UDPConn) WriteTo(p []byte, addr Addr) (int, error) {
	if c == nil || c.fd < 0 {
		return 0, ErrClosed
	}
	if len(p) == 0 {
		return 0, nil
	}
	for {
		var n int32
		e := netSendTo(c.fd, &p[0], int32(len(p)), addr.IP, int32(addr.Port), &n)
		if e == 0 {
			return int(n), nil
		}
		if e != eagain {
			return 0, errOf(e)
		}
		if pe := c.pace(true); pe != 0 {
			return 0, errOf(pe)
		}
	}
}

// Read receives one datagram from the connected peer and returns the count
// plus an error - the TCPConn.Read shapes, except there is no stream-end
// sentinel (UDP has no EOF; (0, nil) is a zero-length datagram). Connected
// form only: an unconnected socket reports ENOTCONN (128). The first
// asynchronous ECONNREFUSED (111) is swallowed - see the package doc - so
// a datagram the peer refused earlier cannot mask a datagram that is
// waiting now.
func (c *UDPConn) Read(p []byte) (int, error) {
	if c == nil || c.fd < 0 {
		return 0, ErrClosed
	}
	if len(p) == 0 {
		return 0, nil
	}
	for {
		var n int32
		e := netRecv(c.fd, &p[0], int32(len(p)), &n)
		if e == 0 {
			return int(n), nil
		}
		if e == econnrefused && c.swallowRefused {
			c.swallowRefused = false
			e = eagain // asynchronous leftover: fall into the wait path
		}
		if e != eagain {
			return 0, errOf(e)
		}
		if pe := c.pace(false); pe != 0 {
			return 0, errOf(pe)
		}
	}
}

// Write sends p as one datagram to the connected peer and reports
// (len(p), nil) on success - the TCPConn.Write convention (with a
// deadline: (0, ErrTimeout) when it passes; without: (0, ErrWouldBlock)
// after one quantum of no progress). Datagrams are atomic, so like TCP
// there is no short positive send to resume. Connected form only (ENOTCONN
// otherwise; use WriteTo).
func (c *UDPConn) Write(p []byte) (int, error) {
	if c == nil || c.fd < 0 {
		return 0, ErrClosed
	}
	if len(p) == 0 {
		return 0, nil
	}
	for {
		var n int32
		e := netSend(c.fd, &p[0], int32(len(p)), &n)
		if e == 0 {
			return int(n), nil
		}
		if e != eagain {
			return 0, errOf(e)
		}
		if pe := c.pace(true); pe != 0 {
			return 0, errOf(pe)
		}
	}
}

// Close releases the socket. UDP has no directions, so unlike TCPConn no
// shutdown is issued - lwip frees the PCB on the spot and can never be
// held open by the peer. Close is idempotent: a second Close, or any call
// on a nil conn, reports ErrClosed exactly like the TCP side.
func (c *UDPConn) Close() error {
	if c == nil || c.fd < 0 {
		return ErrClosed
	}
	fd := c.fd
	c.fd = -1
	return errOf(netClose(fd))
}

// SetDeadline sets the connection's I/O deadline to ms milliseconds from
// now, or clears it with 0; ms < 0 is EINVAL. Both the receive and the
// send paths honor it until SetDeadline changes it. Method form with the
// same semantics as TCPConn.SetDeadline (relative ms; the wrap-safe
// absolute bookkeeping lives in the shared socketDeadline fields).
func (c *UDPConn) SetDeadline(ms int32) error {
	if c == nil || c.fd < 0 {
		return ErrClosed
	}
	if ms < 0 {
		return errOf(einval)
	}
	if ms == 0 {
		c.hasDeadline = false
		return nil
	}
	c.hasDeadline = true
	c.deadlineMs = nowMs() + uint32(ms)
	return nil
}

// pace is UDPConn's view of the shared quantum wait; see socketDeadline.
func (c *UDPConn) pace(wantWrite bool) int32 {
	return c.socketDeadline.pace(c.fd, wantWrite)
}

// Compile-time checks: *UDPConn carries the same core method set as
// *TCPConn, so it satisfies Conn and can travel through interfaces built
// on it (pkg/esp/ws's structural transport is the precedent).
var _ Conn = (*UDPConn)(nil)
