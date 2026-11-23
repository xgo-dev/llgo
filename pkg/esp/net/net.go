// Package net binds llgo-compiled Go code to the ESP-IDF lwip TCP socket
// stack (the same BSD-style sockets esp_netif hands out once the station
// has an IP).
//
// # Layout
//
// Every package under pkg/esp follows the same template (glue lives in a
// subdirectory because the go tool rejects C sources inside a Go package):
//
//	net.go          public API, no build tag
//	net_idf.go      //go:build llgo.idf - bodyless //go:linkname externs
//	net_stub.go     //go:build !llgo.idf - host-build stubs (go vet/gopls)
//	glue/net_glue.c ESP-IDF C glue template, compiled only under ESP_PLATFORM
//
// # Usage
//
// After the wifi package reports EvGotIP (the sockets need an IP):
//
//	c, e := net.Dial(net.IPv4(192, 168, 4, 1), 9000, 10000)
//	net.SetDeadline(c, 5000)
//	w := c.Write(msg)
//	n, e := c.Read(buf)
//	c.Close()
//
// # The bounded-quantum model
//
// llgo's fiber scheduler runs on one FreeRTOS task, so a socket call that
// blocks in C would freeze the whole Go scheduler and starve the watchdog.
// The glue therefore keeps every socket non-blocking (fcntl O_NONBLOCK) and
// this package wraps each operation in a bounded retry loop: every loop
// iteration attempts the syscall once (it fails immediately with EAGAIN
// when it would block), waits one poll quantum in lwip_select - a wait that
// yields the CPU to the idle tasks, so the watchdog stays fed - and retries
// until the operation's deadline passes. Timeouts are accurate to within
// one quantum (20 ms). There is no unbounded wait anywhere in the path
// except the documented Resolve freeze on a real DNS name (see Resolve).
//
// # Deadlines
//
// Two deadline forms coexist, mirroring what the underlying calls can
// bound:
//
//   - Dial and Accept take an explicit timeoutMs parameter (<= 0 is
//     rejected with EINVAL - the model requires every wait to be bounded).
//   - Read, Write and SetDeadline use the connection deadline set by
//     SetDeadline (relative ms; 0 = no deadline). Without a deadline a Read
//     gives up after one quantum with EAGAIN, and a Write returns a short
//     count as soon as the socket stops accepting within a quantum; with a
//     deadline both retry until it passes (Read then reports ETIMEDOUT, a
//     Write reports -ETIMEDOUT).
//
// # Error codes
//
// Errors are the raw lwip errno values as int32 (0 = success) - not
// esp_err codes - because that is what lwip's socket layer sets. Resolve
// returns the DNS API's h_errno-family codes (200-213). net.ErrName maps
// every code this pipeline can hit to its name. A negative value from
// Write is -errno. On a host build (no llgo.idf tag) everything returns
// errUnsupported = 0x7F00, a code outside both real ranges.
//
// # Ownership
//
// The glue keeps no state: every function takes or returns an fd integer
// and allocates nothing, so there is nothing to free on the C side. The Go
// caller owns each socket: Close releases the fd and must be called once
// per Conn or Listener (a second Close, or any call on a nil conn, is a
// defensive no-op returning EBADF). Close shuts both directions down
// first, so a stalled peer can never hold it open.
//
// # Scheduling
//
// A net call keeps the calling goroutine within a bounded window, but the
// quanta inside it are C-level waits: lwip_select parks the FreeRTOS task
// and gives the llgo scheduler no cooperative yield point. Other goroutines
// therefore do not run while one goroutine is inside a net wait - the same
// scheduler architecture property the wifi package's vTaskDelay pacing has.
// Measured under esp-emu with a dial goroutine plus an accept main, each
// bounded to a 3 s window: the accept main ran its whole window alone
// (3026 ms) and timed out; the dial goroutine got no CPU during it and only
// ran once the accept side reached a channel join, finishing its connect in
// 0 ms - the two windows ran strictly serially. The watchdog stays fed, but
// structure concurrent socket I/O accordingly: keep a session on one
// goroutine, or hand control between fibers through channel operations at
// chunk boundaries - a channel send/receive is a real scheduler yield.
//
// # Glue integration
//
// The C glue is a template: copy glue/net_glue.c into your project's main
// component (WHOLE_ARCHIVE required - see pkg/esp/README.md for both
// integration recipes and the boundary rules the glue must obey). The main
// component needs REQUIRES lwip esp_timer.
package net

// The default poll quantum: how long every retry loop waits in lwip_select
// between syscall attempts. One wait is long enough to keep the watchdog
// fed and short enough that timeout accuracy stays well under a tenth of a
// second.
const quantumMs = 20

// lwip errno values this package acts on, verified against the picolibc
// errno.h the ESP-IDF build compiles with. The low values match Linux, and
// so do ECONNRESET (104) and ECONNREFUSED (111); the socket values above
// that diverge: EADDRINUSE 112 (98), ENETUNREACH 114 (101), ETIMEDOUT 116
// (110), EINPROGRESS 119 (115), EALREADY 120 (114), EISCONN 127 (106),
// ENOTCONN 128 (107), ENOBUFS 105 (132) - picolibc value first, Linux's in
// parentheses. The shim fidelity table in the esp README records the values
// as printed by the firmware itself.
const (
	ebadf        = 9   // operation on a closed (or nil) socket
	eagain       = 11  // EWOULDBLOCK: non-blocking call would block
	einval       = 22  // invalid argument
	enametoolong = 91  // name longer than a DNS name may be
	econnreset   = 104 // peer reset the connection
	enobufs      = 105 // lwip out of buffers
	eshutdown    = 110 // can't send after shutdown
	econnrefused = 111 // nothing listening at the far end
	eaddrinuse   = 112 // local address/port already in use
	econnaborted = 113 // connection aborted (e.g. accept target gone)
	enetunreach  = 114 // no route (typically: no IP yet)
	etimedout    = 116 // deadline (or TCP retransmit limit) passed
	ehostunreach = 118 // far host unreachable
	einprogress  = 119 // connect started, still in flight
	ealready     = 120 // connect already in flight
	eisconn      = 127 // socket is already connected
	enotconn     = 128 // socket is not connected
	shutRdwr     = 2   // SHUT_RDWR: both directions
)

// Conn is the interface shared by the socket types this package hands out
// (TCPConn today). Read returns short reads the caller must loop over;
// Write is whole-buffer or a negative errno.
type Conn interface {
	// Read reads up to len(p) bytes. It returns the byte count and an
	// errno (0 = success, including the orderly-close case: n == 0 with
	// err == 0 means the peer closed the stream).
	Read(p []byte) (n int32, err int32)
	// Write writes the whole buffer and returns len(p), or a negative
	// errno when it fails (see the TCPConn.Write doc for the exact
	// deadline semantics).
	Write(p []byte) int32
	// Close releases the socket.
	Close() int32
}

// TCPConn is a connected TCP socket. The fd travels as a plain int32 and
// the glue holds nothing behind it. The deadline is an absolute uptime-ms
// value plus an explicit active flag: the flag - not a 0 sentinel - says
// whether a deadline is set, so a deadline that lands exactly on the 32-bit
// counter wrap (0) is still honored.
type TCPConn struct {
	fd          int32
	deadlineMs  uint32 // absolute uptime-ms deadline, meaningful when hasDeadline
	hasDeadline bool
}

// Listener is a listening TCP socket on the wildcard address.
type Listener struct {
	fd int32
}

// IPv4 packs four octets into the address form this package and the wifi
// package exchange (the in-memory network-order form: on the little-endian
// RV32 the octets sit in the uint32 from the low byte up, exactly as
// wifi.GetIP, wifi.Event.IP and wifi.IPString handle them).
func IPv4(b0, b1, b2, b3 uint8) uint32 {
	return uint32(b0) | uint32(b1)<<8 | uint32(b2)<<16 | uint32(b3)<<24
}

// Dial connects a TCP socket to ip:port within timeoutMs milliseconds.
// ip is the in-memory network-order form (see IPv4); port is the host-order
// port number; timeoutMs must be positive - the bounded-quantum model
// rejects waits without an explicit deadline (EINVAL).
//
// Under the hood: the glue creates the socket with O_NONBLOCK already set,
// connect either completes immediately (an emulator or loopback peer is
// that fast) or reports EINPROGRESS, and the quantum loop selects for
// writability until the deadline, then reads SO_ERROR: 0 means connected,
// anything else (ECONNREFUSED, ETIMEDOUT, ...) is the real connect error.
// On every failure path the socket is closed before Dial returns, so a
// failed Dial leaks nothing.
func Dial(ip uint32, port uint16, timeoutMs int32) (*TCPConn, int32) {
	if timeoutMs <= 0 {
		return nil, einval
	}
	var fd int32
	if e := netSocket(&fd); e != 0 {
		return nil, e
	}
	e := netConnect(fd, ip, int32(port))
	if e == 0 {
		return &TCPConn{fd: fd}, 0
	}
	if e != einprogress && e != ealready {
		netClose(fd)
		return nil, e
	}
	deadline := nowMs() + uint32(timeoutMs)
	for {
		var ready int32
		if pe := netPoll(fd, 1, quantumMs, &ready); pe != 0 {
			netClose(fd)
			return nil, pe
		}
		if ready == 1 {
			var soErr int32
			if se := netSockError(fd, &soErr); se != 0 {
				netClose(fd)
				return nil, se
			}
			if soErr == 0 {
				return &TCPConn{fd: fd}, 0
			}
			netClose(fd)
			return nil, soErr
		}
		if past(deadline) {
			netClose(fd)
			return nil, etimedout
		}
	}
}

// Resolve looks up a host name and returns its IPv4 address in the same
// in-memory network-order form Dial takes. It wraps lwip's gethostbyname:
//
//   - A dotted-quad string ("192.168.4.1") is parsed by lwip directly and
//     never touches DNS - it returns in microseconds.
//   - A real name goes through the lwip DNS client, which blocks the
//     calling task on a driver semaphore (it yields the CPU, so the
//     watchdog is safe) but freezes the llgo scheduler until lwip's own DNS
//     timers finish - typically well under a second, with a multi-second
//     worst case when the resolver is unreachable. Prefer Dial by numeric
//     IP on time-critical paths.
//   - A name longer than 255 bytes is rejected up front (ENAMETOOLONG).
//
// Failures return the DNS API's h_errno-family codes (HOST_NOT_FOUND,
// TRY_AGAIN, ... - see ErrName), not socket errnos.
func Resolve(host string) (uint32, int32) {
	if host == "" {
		return 0, einval
	}
	if len(host) > 255 {
		return 0, enametoolong
	}
	buf := make([]uint8, len(host))
	copy(buf, host)
	var ip uint32
	if e := netResolve(&buf[0], int32(len(buf)), &ip); e != 0 {
		return 0, e
	}
	return ip, 0
}

// Listen opens a listening socket on the wildcard address (all local IPs,
// so both the station address and loopback ring it) bound to port. The
// backlog is fixed at 4 pending connections - plenty for a device probe,
// and lwip keeps small queues cheap.
func Listen(port uint16) (*Listener, int32) {
	var fd int32
	if e := netSocket(&fd); e != 0 {
		return nil, e
	}
	if e := netBindListen(fd, int32(port), 4); e != 0 {
		netClose(fd)
		return nil, e
	}
	return &Listener{fd: fd}, 0
}

// Accept waits up to timeoutMs milliseconds (must be positive - EINVAL
// otherwise) for an incoming connection and returns it as a fresh TCPConn.
// The accepted socket is non-blocking from birth, like every socket this
// package creates; its deadline starts unset, so SetDeadline it before
// reading.
func (l *Listener) Accept(timeoutMs int32) (*TCPConn, int32) {
	if l == nil || l.fd < 0 {
		return nil, ebadf
	}
	if timeoutMs <= 0 {
		return nil, einval
	}
	deadline := nowMs() + uint32(timeoutMs)
	for {
		var fd int32
		e := netAccept(l.fd, &fd)
		if e == 0 {
			return &TCPConn{fd: fd}, 0
		}
		if e != eagain {
			return nil, e
		}
		var ready int32
		if pe := netPoll(l.fd, 0, quantumMs, &ready); pe != 0 {
			return nil, pe
		}
		if past(deadline) {
			return nil, etimedout
		}
	}
}

// Close releases the listening socket. Connections already accepted are
// unaffected; close them individually.
func (l *Listener) Close() int32 {
	if l == nil || l.fd < 0 {
		return ebadf
	}
	fd := l.fd
	l.fd = -1
	return netClose(fd)
}

// Read reads up to len(p) bytes from the connection and returns the count
// plus an errno. Reads are short: the caller loops until it has what it
// needs. The sentinel combinations:
//
//	n > 0, err == 0     n bytes arrived
//	n == 0, err == 0    the peer closed the stream in order (recv returned 0)
//	n == 0, err == 11   no deadline set and nothing arrived within one
//	                    quantum (EAGAIN) - a poll-style read
//	n == 0, err == 116  a deadline was set and passed with no data
//	                    (ETIMEDOUT)
//
// A zero-length p is a no-op success (passing it to recv would make lwip
// return 0, which Read could not tell from a peer close).
func (c *TCPConn) Read(p []byte) (int32, int32) {
	if c == nil || c.fd < 0 {
		return 0, ebadf
	}
	if len(p) == 0 {
		return 0, 0
	}
	for {
		var n int32
		e := netRecv(c.fd, &p[0], int32(len(p)), &n)
		if e == 0 {
			return n, 0
		}
		if e != eagain {
			return 0, e
		}
		if pe := c.pace(false); pe != 0 {
			return 0, pe
		}
	}
}

// Write writes the whole buffer, looping internally until the socket has
// accepted every byte or the bounded wait ends. It returns len(p) on
// success and a negative errno otherwise:
//
//   - With a deadline: retries past EAGAIN until the deadline, then
//     returns -ETIMEDOUT (116). Deadline expiry never returns a short
//     positive count; the one short positive return possible while a
//     deadline is set is a wait that itself fails (dead socket) after some
//     bytes were already accepted. An unknown prefix of p may be in
//     flight, so after any non-full return the connection must be treated
//     as unusable.
//   - Without a deadline: returns a short non-negative count as soon as
//     one quantum passes without progress (or -EAGAIN for count 0); the
//     caller retries with p[n:] later.
//
// Any other negative return is -errno (e.g. -104 for ECONNRESET after a
// peer reset); treat it as fatal for the connection.
func (c *TCPConn) Write(p []byte) int32 {
	if c == nil || c.fd < 0 {
		return -ebadf
	}
	if len(p) == 0 {
		return 0
	}
	total := int32(0)
	for {
		var n int32
		e := netSend(c.fd, &p[total], int32(len(p))-total, &n)
		if e == 0 && n > 0 {
			total += n
			if total == int32(len(p)) {
				return total
			}
			continue
		}
		if e != 0 && e != eagain {
			return -e
		}
		// EAGAIN (or the defensive send-returned-zero case): wait one
		// quantum for writability and honor the deadline.
		if pe := c.pace(true); pe != 0 {
			if pe == etimedout {
				return -etimedout
			}
			if total == 0 {
				return -pe // -EAGAIN: nothing accepted, retry later
			}
			return total // short write: caller retries with p[total:]
		}
	}
}

// Close shuts the connection down in both directions and releases the fd.
// The shutdown-first order means Close never waits on a draining peer: any
// unsent data in lwip's buffers is dropped instead. A second Close on the
// same conn is a no-op returning EBADF (9), as is any call on a closed or
// nil conn.
func (c *TCPConn) Close() int32 {
	if c == nil || c.fd < 0 {
		return ebadf
	}
	fd := c.fd
	c.fd = -1
	netShutdown(fd, shutRdwr)
	return netClose(fd)
}

// SetDeadline sets the connection's I/O deadline to ms milliseconds from
// now, or clears it with 0. Both Read and Write honor it until SetDeadline
// changes it. ms < 0 is EINVAL. Returns 0 on success.
func SetDeadline(c *TCPConn, ms int32) int32 {
	if c == nil || c.fd < 0 {
		return ebadf
	}
	if ms < 0 {
		return einval
	}
	if ms == 0 {
		c.hasDeadline = false
		return 0
	}
	c.hasDeadline = true
	c.deadlineMs = nowMs() + uint32(ms)
	return 0
}

// pace waits one poll quantum (a select that yields the CPU) and reports
// how the bounded wait ended: 0 = retry the syscall now, eagain = no
// deadline is set and nothing became ready within the quantum, etimedout =
// the connection deadline passed. A select failure surfaces as its errno -
// the caller's next syscall would report the same dead socket anyway.
func (c *TCPConn) pace(wantWrite bool) int32 {
	w := int32(0)
	if wantWrite {
		w = 1
	}
	var ready int32
	if e := netPoll(c.fd, w, quantumMs, &ready); e != 0 {
		return e
	}
	if c.hasDeadline && past(c.deadlineMs) {
		return etimedout
	}
	if ready == 0 && !c.hasDeadline {
		return eagain
	}
	return 0
}

// past reports whether an absolute uptime-ms deadline has passed. The
// unsigned subtraction compares safely across the 32-bit millisecond
// counter wrap - including a deadline that wraps through 0, which is why
// "no deadline" is carried by TCPConn.hasDeadline instead of a 0-value
// sentinel.
func past(deadline uint32) bool {
	return uint32(nowMs()-deadline) < 1<<31
}

// ErrName maps the codes this pipeline can hit to their lwip names. Socket
// operations return errno values; Resolve returns the DNS API's
// h_errno-family codes (200-213). A negative Write result is handled too
// (the sign is stripped first). Mind the picolibc numbering: several socket
// values above 100 differ from Linux (ETIMEDOUT 116 vs 110, EINPROGRESS
// 119 vs 115, ENOTCONN 128 vs 107) while others coincide (ECONNRESET 104,
// ECONNREFUSED 111) - the shim fidelity table in pkg/esp/README.md records
// the values as printed by the firmware itself.
func ErrName(code int32) string {
	if code < 0 {
		code = -code
	}
	switch code {
	case 0:
		return "OK"
	case 4:
		return "EINTR"
	case 5:
		return "EIO"
	case 9:
		return "EBADF"
	case 11:
		return "EAGAIN"
	case 12:
		return "ENOMEM"
	case 13:
		return "EACCES"
	case 22:
		return "EINVAL"
	case 32:
		return "EPIPE"
	case 91:
		return "ENAMETOOLONG"
	case 95:
		return "EOPNOTSUPP"
	case 104:
		return "ECONNRESET"
	case 105:
		return "ENOBUFS"
	case 106:
		return "EAFNOSUPPORT"
	case 107:
		return "EPROTOTYPE"
	case 108:
		return "ENOTSOCK"
	case 109:
		return "ENOPROTOOPT"
	case 110:
		return "ESHUTDOWN"
	case 111:
		return "ECONNREFUSED"
	case 112:
		return "EADDRINUSE"
	case 113:
		return "ECONNABORTED"
	case 114:
		return "ENETUNREACH"
	case 115:
		return "ENETDOWN"
	case 116:
		return "ETIMEDOUT"
	case 117:
		return "EHOSTDOWN"
	case 118:
		return "EHOSTUNREACH"
	case 119:
		return "EINPROGRESS"
	case 120:
		return "EALREADY"
	case 121:
		return "EDESTADDRREQ"
	case 122:
		return "EMSGSIZE"
	case 123:
		return "EPROTONOSUPPORT"
	case 125:
		return "EADDRNOTAVAIL"
	case 126:
		return "ENETRESET"
	case 127:
		return "EISCONN"
	case 128:
		return "ENOTCONN"
	case 129:
		return "ETOOMANYREFS"
	case 134:
		return "ENOTSUP"
	case 200:
		return "EAI_NONAME"
	case 201:
		return "EAI_SERVICE"
	case 202:
		return "EAI_FAIL"
	case 203:
		return "EAI_MEMORY"
	case 204:
		return "EAI_FAMILY"
	case 210:
		return "HOST_NOT_FOUND"
	case 211:
		return "NO_DATA"
	case 212:
		return "NO_RECOVERY"
	case 213:
		return "TRY_AGAIN"
	}
	return "errno " + dec(uint32(code))
}

// dec formats v in decimal (fmt is not imported in pkg/esp packages).
func dec(v uint32) string {
	if v == 0 {
		return "0"
	}
	var buf [10]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}

// Compile-time check that *TCPConn implements the Conn contract.
var _ Conn = (*TCPConn)(nil)
