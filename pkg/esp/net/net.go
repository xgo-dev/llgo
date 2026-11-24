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
//	c, err := net.Dial(net.IPv4(192, 168, 4, 1), 9000, 10000)
//	if err != nil { ... }
//	c.SetDeadline(5000)
//	n, err := c.Write(msg)
//	n, err = c.Read(buf)
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
//   - Dial takes an explicit timeoutMs parameter (<= 0 is rejected with
//     EINVAL - the model requires every wait to be bounded); Accept is
//     bounded by the listener's own SetDeadline.
//   - Read, Write, Accept and SetDeadline use the deadline set by
//     SetDeadline (relative ms; 0 = no deadline). Without a deadline a
//     Read gives up after one quantum with ErrWouldBlock, an Accept after
//     one quantum with ErrWouldBlock, and a Write returns a short count
//     with ErrWouldBlock as soon as the socket stops accepting within a
//     quantum; with a deadline both retry until it passes (Read then
//     reports ErrTimeout, a Write (n, ErrTimeout) with the accepted
//     prefix).
//
// Deadlines are relative milliseconds - NOT stdlib's time.Time. That is a
// deliberate deviation: time.Time has never been validated on the
// bare-metal target and the bounded-quantum model does not need wall-clock
// arithmetic; the relative-ms signature keeps every wait explicitly
// bounded. Everything else follows io.Reader/io.Writer and net.Conn
// conventions.
//
// # Error model
//
// Errors are Go error values of the shared pkg/esp/esperr type: Code is
// the raw lwip errno (0 = success never crosses back; 0 wraps to nil),
// Name its ErrName rendering. The transport sentinels are re-exported
// here: ErrClosed (a second Close, or any call on a closed or nil conn),
// ErrTimeout (deadline passed), ErrWouldBlock (poll-style give-up
// without a deadline) and ErrEOF (orderly peer close - the family's io.EOF
// counterpart, kept in-family because the stdlib io package drags the
// errors/reflect machinery into the bare-metal image). Other
// failures carry the pipeline code: ECONNRESET after a peer reset, the
// resolver's h_errno family from Resolve (200-213), and so on - all
// nameable through ErrName. errUnsupported paths (host builds) surface as
// *esperr.Error{Code: 0x7F00}.
//
// # Ownership
//
// The glue keeps no state: every function takes or returns an fd integer
// and allocates nothing, so there is nothing to free on the C side. The Go
// caller owns each socket: Close releases the fd. Close is idempotent - a
// second Close (or any call on a closed or nil conn) reports ErrClosed,
// not a raw errno. Close shuts both directions down first, so a stalled
// peer can never hold it open.
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

import "github.com/xgo-dev/llgo/pkg/esp/esperr"

// The transport error sentinels, re-exported from the shared pkg/esp/esperr
// base so callers can stay inside this package's namespace. See the
// error-model section above.
var (
	ErrClosed     = esperr.ErrClosed
	ErrTimeout    = esperr.ErrTimeout
	ErrWouldBlock = esperr.ErrWouldBlock
	ErrEOF        = esperr.ErrEOF
)

// errOf converts a pipeline code to the error the API returns (nil for 0).
// The three sentinel codes map to the shared esperr values; anything else
// becomes an *esperr.Error named through ErrName.
func errOf(code int32) error {
	switch code {
	case 0:
		return nil
	case ebadf:
		return ErrClosed
	case eagain:
		return ErrWouldBlock
	case etimedout:
		return ErrTimeout
	}
	return &esperr.Error{Code: code, Name: ErrName(code)}
}

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
	shutWr       = 1   // SHUT_WR: half-close, no more sends (CloseWrite)
	shutRdwr     = 2   // SHUT_RDWR: both directions
)

// Conn is the interface shared by the socket types this package hands out
// (TCPConn and UDPConn). Read returns short reads the caller must loop
// over; Write returns the accepted byte count with an error, the io.Writer
// convention. Deadlines are relative milliseconds (see the deadlines
// section above) - the one deliberate deviation from stdlib net.Conn.
type Conn interface {
	// Read reads up to len(p) bytes. It returns the byte count and an
	// error: io.EOF when the peer closed the stream in order, ErrTimeout
	// on deadline expiry, ErrWouldBlock for the poll-style read without
	// a deadline.
	Read(p []byte) (n int, err error)
	// Write writes p and returns how many bytes were accepted plus an
	// error (see TCPConn.Write for the exact deadline semantics).
	Write(p []byte) (n int, err error)
	// Close releases the socket. Idempotent: later calls report ErrClosed.
	Close() error
	// SetDeadline sets the I/O deadline (relative ms; 0 clears, < 0 is
	// EINVAL).
	SetDeadline(ms int32) error
}

// TCPConn is a connected TCP socket. The fd travels as a plain int32 and
// the glue holds nothing behind it. The deadline bookkeeping is the shared
// socketDeadline form.
type TCPConn struct {
	fd int32
	socketDeadline
}

// socketDeadline is the deadline bookkeeping shared by TCPConn, UDPConn
// and Listener: an absolute uptime-ms value plus an explicit active flag -
// the flag, not a 0 sentinel, says whether a deadline is set, so a deadline
// that lands exactly on the 32-bit counter wrap (0) is still honored.
type socketDeadline struct {
	deadlineMs  uint32 // absolute uptime-ms deadline, meaningful when hasDeadline
	hasDeadline bool
}

// pace waits one poll quantum on fd (a select that yields the CPU) and
// reports how the bounded wait ended as a pipeline code: 0 = retry the
// syscall now, eagain = no deadline is set and nothing became ready within
// the quantum, etimedout = the deadline passed. A select failure surfaces
// as its errno - the caller's next syscall would report the same dead
// socket anyway. Shared by TCPConn and UDPConn.
func (d *socketDeadline) pace(fd int32, wantWrite bool) int32 {
	w := int32(0)
	if wantWrite {
		w = 1
	}
	var ready int32
	if e := netPoll(fd, w, quantumMs, &ready); e != 0 {
		return e
	}
	if d.hasDeadline && past(d.deadlineMs) {
		return etimedout
	}
	if ready == 0 && !d.hasDeadline {
		return eagain
	}
	return 0
}

// Listener is a listening TCP socket on the wildcard address. Its Accept
// deadline is set with SetDeadline; without one, Accept is a poll-style
// call that gives up after one quantum with ErrWouldBlock.
type Listener struct {
	fd int32
	socketDeadline
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
func Dial(ip uint32, port uint16, timeoutMs int32) (*TCPConn, error) {
	if timeoutMs <= 0 {
		return nil, errOf(einval)
	}
	var fd int32
	if e := netSocket(&fd); e != 0 {
		return nil, errOf(e)
	}
	e := netConnect(fd, ip, int32(port))
	if e == 0 {
		return &TCPConn{fd: fd}, nil
	}
	if e != einprogress && e != ealready {
		netClose(fd)
		return nil, errOf(e)
	}
	deadline := nowMs() + uint32(timeoutMs)
	for {
		var ready int32
		if pe := netPoll(fd, 1, quantumMs, &ready); pe != 0 {
			netClose(fd)
			return nil, errOf(pe)
		}
		if ready == 1 {
			var soErr int32
			if se := netSockError(fd, &soErr); se != 0 {
				netClose(fd)
				return nil, errOf(se)
			}
			if soErr == 0 {
				return &TCPConn{fd: fd}, nil
			}
			netClose(fd)
			return nil, errOf(soErr)
		}
		if past(deadline) {
			netClose(fd)
			return nil, ErrTimeout
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
// Failures return the DNS API's h_errno-family errors (HOST_NOT_FOUND,
// TRY_AGAIN, ... - named by ErrName), not socket errnos.
func Resolve(host string) (uint32, error) {
	if host == "" {
		return 0, errOf(einval)
	}
	if len(host) > 255 {
		return 0, errOf(enametoolong)
	}
	buf := make([]uint8, len(host))
	copy(buf, host)
	var ip uint32
	if e := netResolve(&buf[0], int32(len(buf)), &ip); e != 0 {
		return 0, errOf(e)
	}
	return ip, nil
}

// Listen opens a listening socket on the wildcard address (all local IPs,
// so both the station address and loopback ring it) bound to port. The
// backlog is fixed at 4 pending connections - plenty for a device probe,
// and lwip keeps small queues cheap. The listener's Accept deadline starts
// unset: SetDeadline it for a bounded Accept, or take the one-quantum
// poll-style Accept.
func Listen(port uint16) (*Listener, error) {
	var fd int32
	if e := netSocket(&fd); e != 0 {
		return nil, errOf(e)
	}
	if e := netBindListen(fd, int32(port), 4); e != 0 {
		netClose(fd)
		return nil, errOf(e)
	}
	return &Listener{fd: fd}, nil
}

// Accept returns the next incoming connection as a fresh TCPConn. With a
// listener deadline set (SetDeadline) it retries until the deadline passes
// (then ErrTimeout); without one it is a poll-style call: one quantum of
// waiting, then ErrWouldBlock when nobody connected - the bounded model's
// shape, a deliberate deviation from stdlib's blocking Accept. The
// accepted socket is non-blocking from birth, like every socket this
// package creates; its deadline starts unset, so SetDeadline it before
// reading.
func (l *Listener) Accept() (*TCPConn, error) {
	if l == nil || l.fd < 0 {
		return nil, ErrClosed
	}
	c, e := l.accept(l.deadlineMs, l.hasDeadline)
	return c, errOf(e)
}

// AcceptTimeout is the pre-deadline-model accept form, kept as a
// deprecated alias: it waits up to timeoutMs milliseconds (must be
// positive - EINVAL otherwise) for an incoming connection. New code should
// use SetDeadline + Accept.
//
// Deprecated: use SetDeadline and Accept.
func (l *Listener) AcceptTimeout(timeoutMs int32) (*TCPConn, error) {
	if l == nil || l.fd < 0 {
		return nil, ErrClosed
	}
	if timeoutMs <= 0 {
		return nil, errOf(einval)
	}
	c, e := l.accept(nowMs()+uint32(timeoutMs), true)
	return c, errOf(e)
}

// accept is the shared bounded accept loop: one non-blocking attempt, one
// poll quantum, then either the poll-style give-up (no deadline) or the
// deadline check. The internal-code return keeps both callers'
// errOf mapping in one place.
func (l *Listener) accept(deadline uint32, hasDeadline bool) (*TCPConn, int32) {
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
		if !hasDeadline {
			return nil, eagain
		}
		if past(deadline) {
			return nil, etimedout
		}
	}
}

// SetDeadline sets the listener's Accept deadline to ms milliseconds from
// now, or clears it with 0. ms < 0 is EINVAL.
func (l *Listener) SetDeadline(ms int32) error {
	if l == nil || l.fd < 0 {
		return ErrClosed
	}
	if ms < 0 {
		return errOf(einval)
	}
	if ms == 0 {
		l.hasDeadline = false
		return nil
	}
	l.hasDeadline = true
	l.deadlineMs = nowMs() + uint32(ms)
	return nil
}

// Close releases the listening socket. Connections already accepted are
// unaffected; close them individually. Idempotent: a second Close reports
// ErrClosed.
func (l *Listener) Close() error {
	if l == nil || l.fd < 0 {
		return ErrClosed
	}
	fd := l.fd
	l.fd = -1
	return errOf(netClose(fd))
}

// Read reads up to len(p) bytes from the connection and returns the count
// plus an error. Reads are short: the caller loops until it has what it
// needs. The outcomes:
//
//	n > 0, nil          n bytes arrived
//	0, ErrEOF           the peer closed the stream in order (recv
//	                    returned 0)
//	0, ErrWouldBlock    no deadline set and nothing arrived within one
//	                    quantum (EAGAIN) - a poll-style read
//	0, ErrTimeout       a deadline was set and passed with no data
//
// A zero-length p is a no-op success (passing it to recv would make lwip
// return 0, which Read could not tell from a peer close).
func (c *TCPConn) Read(p []byte) (int, error) {
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
			if n == 0 {
				return 0, ErrEOF // orderly peer close
			}
			return int(n), nil
		}
		if e != eagain {
			return 0, errOf(e)
		}
		if pe := c.pace(false); pe != 0 {
			return 0, errOf(pe)
		}
	}
}

// Write writes p to the connection, looping internally, and returns how
// many bytes were accepted plus an error - the io.Writer convention,
// including its "n < len(p) implies err != nil" contract:
//
//	len(p), nil         the whole buffer was handed to lwip
//	n, ErrTimeout       the deadline passed with n bytes already
//	                    accepted; retry with p[n:] or consider the
//	                    connection's state unknown (an unknown prefix of
//	                    p may still be in flight)
//	n, ErrWouldBlock    no deadline set and the socket stopped accepting
//	                    after n bytes within one quantum; retry with
//	                    p[n:] later
//	n, err              a wait that itself failed (dead socket, e.g.
//	                    ECONNRESET) after n accepted bytes - treat as
//	                    fatal for the connection
//
// The deadline-expiry case is the fix over the pre-error model, which
// returned only -ETIMEDOUT and threw away the accepted count.
func (c *TCPConn) Write(p []byte) (int, error) {
	if c == nil || c.fd < 0 {
		return 0, ErrClosed
	}
	if len(p) == 0 {
		return 0, nil
	}
	total := int32(0)
	for {
		var n int32
		e := netSend(c.fd, &p[total], int32(len(p))-total, &n)
		if e == 0 && n > 0 {
			total += n
			if total == int32(len(p)) {
				return int(total), nil
			}
			continue
		}
		if e != 0 && e != eagain {
			return int(total), errOf(e)
		}
		// EAGAIN (or the defensive send-returned-zero case): wait one
		// quantum for writability and honor the deadline.
		if pe := c.pace(true); pe != 0 {
			if pe == etimedout {
				return int(total), ErrTimeout
			}
			// EAGAIN (would-block give-up) or a select failure (the
			// next syscall would report the same dead socket).
			return int(total), errOf(pe)
		}
	}
}

// CloseWrite half-closes the connection: no more data will be sent (the
// TCP FIN goes out), but reads keep working - the shape for a request that
// signals its end by EOF and still waits for the response. Idempotent like
// Close; ErrClosed after the connection is gone.
func (c *TCPConn) CloseWrite() error {
	if c == nil || c.fd < 0 {
		return ErrClosed
	}
	return errOf(netShutdown(c.fd, shutWr))
}

// Close shuts the connection down in both directions and releases the fd.
// The shutdown-first order means Close never waits on a draining peer: any
// unsent data in lwip's buffers is dropped instead. Close is idempotent: a
// second Close on the same conn (or any call on a closed or nil conn)
// reports ErrClosed.
func (c *TCPConn) Close() error {
	if c == nil || c.fd < 0 {
		return ErrClosed
	}
	fd := c.fd
	c.fd = -1
	netShutdown(fd, shutRdwr)
	return errOf(netClose(fd))
}

// SetDeadline sets the connection's I/O deadline to ms milliseconds from
// now, or clears it with 0. Both Read and Write honor it until SetDeadline
// changes it. ms < 0 is EINVAL.
//
// This is the method form: *TCPConn carries SetDeadline as a method (like
// tls.TLSConn) so a structural interface such as pkg/esp/ws's Deadliner can
// recognize it without importing this package. The package-level
// SetDeadline below is the same operation kept for existing callers.
func (c *TCPConn) SetDeadline(ms int32) error {
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

// SetDeadline sets the connection's I/O deadline (relative ms; 0 clears,
// < 0 is EINVAL). It is the package-level form of TCPConn.SetDeadline,
// kept for existing callers; the method form exists so structural
// interfaces (pkg/esp/ws.Deadliner) can match *TCPConn without importing
// this package.
//
// Deprecated: use the TCPConn.SetDeadline method.
func SetDeadline(c *TCPConn, ms int32) error {
	return c.SetDeadline(ms)
}

// pace is TCPConn's view of the shared quantum wait; see socketDeadline.
func (c *TCPConn) pace(wantWrite bool) int32 {
	return c.socketDeadline.pace(c.fd, wantWrite)
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
// h_errno-family codes (200-213). Mind the picolibc numbering: several
// socket values above 100 differ from Linux (ETIMEDOUT 116 vs 110,
// EINPROGRESS 119 vs 115, ENOTCONN 128 vs 107) while others coincide
// (ECONNRESET 104, ECONNREFUSED 111) - the shim fidelity table in
// pkg/esp/README.md records the values as printed by the firmware itself.
// Errors returned by this package carry their name already (esperr.Error
// Name field); ErrName stays for callers that hold a bare esperr code. A
// negative input is resolved through its absolute value (the pre-error
// Write convention's -errno shape).
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
