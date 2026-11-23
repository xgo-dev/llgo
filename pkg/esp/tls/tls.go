// Package tls binds llgo-compiled Go code to the ESP-IDF mbedtls TLS stack
// (mbedtls 4.1.1 in IDF v6.3-dev), giving a connected TCP socket (see the
// net package) a record layer: TLSDial connects and handshakes, then the
// connection exchanges plaintext through Read and Write while mbedtls
// encrypts and authenticates every byte.
//
// # Layout
//
// Same template as pkg/esp/net (glue lives in a subdirectory because the go
// tool rejects C sources inside a Go package):
//
//	tls.go          public API, no build tag
//	tls_idf.go      //go:build llgo.idf - bodyless //go:linkname externs
//	tls_stub.go     //go:build !llgo.idf - host-build stubs (go vet/gopls)
//	glue/tls_glue.c ESP-IDF C glue template, compiled only under ESP_PLATFORM
//
// The package is deliberately dependency-free (only stdlib/unsafe): the TCP
// primitives it needs (socket, connect, poll, shutdown, close, uptime clock)
// are the C symbols net_glue.c already defines, declared here again by
// linkname - so the glue set a TLS program links is tls_glue.c PLUS
// net_glue.c, without the Go package importing pkg/esp/net.
//
// # Usage
//
// After the wifi package reports EvGotIP (and, on a host peer, that peer is
// listening):
//
//	c, e := tls.TLSDial(net.IPv4(192, 168, 4, 1), 9443, 10000, nil) // VERIFY_NONE
//	c, e := tls.TLSDial(ip, port, 10000, caPEM)                     // VERIFY_REQUIRED
//	w := c.Write(msg)
//	n, e := c.Read(buf)
//	e = c.Close()
//
// # The bounded-quantum model (inherited from net)
//
// llgo's fiber scheduler runs on one FreeRTOS task, so a call that blocks in
// C would freeze the whole Go scheduler and starve the watchdog. The glue
// keeps the TCP socket non-blocking and every operation is a Go-side retry
// loop: one non-blocking attempt (one handshake step, one record
// read/write, one close_notify), then a 20 ms poll quantum on the socket -
// a wait that parks the FreeRTOS task on lwip's select semaphore and feeds
// the idle tasks - then a deadline check. The mbedtls layer reports what it
// wants through MBEDTLS_ERR_SSL_WANT_READ/WANT_WRITE and this package maps
// those to the socket quanta, so TLS 1.2 and 1.3 handshakes advance at the
// same pace the raw TCP operations do. Timeouts are accurate to within one
// quantum. There is no unbounded wait anywhere in the path.
//
// # Deadlines
//
// TLSDial takes an explicit timeoutMs (<= 0 is rejected with EINVAL - the
// model requires bounded waits) that bounds BOTH the TCP connect and the
// handshake as one total budget. Read and Write honor the deadline set by
// SetDeadline (relative ms; 0 = no deadline), with net.Conn semantics:
// with no deadline a Read gives up after one quantum with EAGAIN and a
// Write returns a "nothing accepted" signal (-EAGAIN) as soon as one
// quantum passes without progress; with a deadline both retry until it
// passes (Read reports ETIMEDOUT, Write -ETIMEDOUT).
//
// One semantic difference from raw TCP, inherent to TLS: Write accepts a
// buffer all-or-nothing. mbedtls encrypts the whole plaintext into one
// record (it never takes a partial plaintext), so a Write that could not
// complete has consumed NOTHING and the caller retries the whole buffer -
// there are no short positive writes in this package.
//
// # Certificates and authmode
//
// TLSDial's caPEM parameter chooses the verification mode:
//
//   - caPEM == nil: MBEDTLS_SSL_VERIFY_NONE. The handshake runs, the
//     session key is still authenticated (an active MITM cannot read the
//     traffic), but the peer's identity is not checked. This is the
//     default for deterministic emulator acceptance runs.
//   - caPEM != nil: the PEM is parsed (one x509_crt_parse call on a
//     NUL-terminated copy - mbedtls's PEM path requires the terminator)
//     and MBEDTLS_SSL_VERIFY_REQUIRED is set. The server chain must verify
//     against that trust anchor, including validity dates; a failure ends
//     the dial with CertVerifyFailed. The PEM may hold one CA certificate
//     or a chain of concatenated PEM blocks (mbedtls parses them all; any
//     failed member rejects the dial with CaParseFailed) - a single-CA PEM
//     is the documented and tested path. Verification covers chain and
//     validity only: the glue calls
//     mbedtls_ssl_set_hostname(NULL) - the explicit opt-out the TLS layer
//     demands before it allows certificate verification on a client
//     without a hostname (MBEDTLS_ERR_SSL_CERTIFICATE_VERIFICATION_WITHOUT_
//     HOSTNAME) - so no CN/SAN matching happens; the caller's identity
//     check is its trust in the CA.
//
// The RNG behind the handshake is PSA Crypto random: mbedtls 4.1 wires the
// SSL layer to psa_generate_random and IDF v6.3-dev initializes PSA at
// startup (and feeds it from the hardware TRNG through esp_fill_random).
// The ctr_drbg of older mbedtls is a private header in 4.1.1, hence the
// PSA path. Under esp-emu the TRNG register reads back deterministic
// values, so emulated handshakes are reproducible (fine for acceptance;
// on hardware this is a real TRNG).
//
// # Error codes
//
// TLS-layer failures use a new segment at 0x7E00 (outside the lwip errno
// range, the DNS h_errno range and the esp_err range), listed by ErrName:
// HandshakeTimeout, CertVerifyFailed, CaParseFailed, WantPollExhausted,
// Handshake, PsaInitFailed. Three conventions on top:
//
//   - TCP-phase failures (socket, connect) surface as the raw lwip errno,
//     exactly as net.Dial returns them - ErrName names those values too.
//   - Dial failures inside mbedtls's own setup (config defaults, ssl_setup,
//     the rest of the glue's llgo_tls_new) pass through the raw negative
//     mbedtls code; the only 0x7E00 codes that phase produces are its own
//     CaParseFailed and PsaInitFailed. The Handshake code exists only on
//     the handshake phase, where the dial loop maps the codes its table
//     does not act on to it.
//   - Record-phase failures after the handshake surface as the raw
//     negative mbedtls code (e.g. -0x7780 fatal alert); mbedtls treats
//     such an error as fatal for the session, and so should the caller.
//     ErrName names the common ones.
//
// # Ownership
//
// The glue allocates one C-side context per connection (the mbedtls state:
// ssl context, ssl config, parsed CA) and hands it to Go as an opaque
// handle stored in a uintptr field - so the GC never scans it as a
// pointer. Close frees the context and the fd exactly once; a second
// Close, or any call on a nil or closed conn, is a defensive no-op
// returning EBADF. A failed TLSDial frees everything before returning, so
// a failed dial leaks nothing.
//
// # Scheduling
//
// Same architecture property as the net package: the quantum waits are
// C-level select waits that give the llgo scheduler no cooperative yield
// point, so other goroutines do not run while one goroutine sits inside a
// TLS wait. Keep a TLS session on one goroutine, or hand control between
// fibers through channel operations at chunk boundaries - a channel
// send/receive is a real scheduler yield.
//
// # Glue integration
//
// The C glue is a template: copy glue/tls_glue.c into your project's main
// component (WHOLE_ARCHIVE required - see pkg/esp/README.md for both
// integration recipes and the boundary rules). A TLS program links BOTH
// glues: tls_glue.c needs REQUIRES mbedtls; the shared socket/clock
// primitives come from net_glue.c (REQUIRES lwip esp_timer).
package tls

// The default poll quantum: how long every retry loop waits for the socket
// between attempts. Same value the net package uses; one wait is long
// enough to keep the watchdog fed and short enough that timeout accuracy
// stays well under a tenth of a second.
const quantumMs = 20

// TLSConn is a connected TLS session. The C-side mbedtls context travels
// as an opaque uintptr handle (the GC must never scan it as a pointer),
// and fd is the underlying non-blocking TCP socket the quantum polls use.
// The deadline is an absolute uptime-ms value plus an explicit active flag:
// the flag - not a 0 sentinel - says whether a deadline is set, so a
// deadline that lands exactly on the 32-bit counter wrap (0) is still
// honored.
type TLSConn struct {
	h           uintptr
	fd          int32
	deadlineMs  uint32 // absolute uptime-ms deadline, meaningful when hasDeadline
	hasDeadline bool
}

// mbedtls error codes this package acts on by value. Values pinned from the
// mbedtls 4.1.1 headers IDF v6.3-dev ships (mbedtls/ssl.h, mbedtls/x509.h);
// the WANT pair changed sign value from mbedtls 3.x (-0x6880/-0x6890 there,
// -0x6900/-0x6880 here - WANT_WRITE sorts before WANT_READ).
const (
	mbErrWantRead         = -0x6900 // record layer wants more input
	mbErrWantWrite        = -0x6880 // record layer wants the socket writable
	mbErrPeerCloseNotify  = -0x7880 // peer sent the TLS close notification
	mbErrConnEOF          = -0x7280 // TCP closed cleanly without close_notify
	mbErrCertVerifyFailed = -0x2700 // server chain failed x509 verification
)

// TLS-layer error codes, the 0x7E00 segment. See ErrName for the names.
const (
	// HandshakeTimeout: the handshake was still only making WANT progress
	// when TLSDial's total budget (timeoutMs) ran out. The TCP layer was
	// fine; the peer was too slow or the network too lossy.
	HandshakeTimeout = 0x7E01
	// CertVerifyFailed: VERIFY_REQUIRED (caPEM given) and the server chain
	// failed x509 verification against that CA.
	CertVerifyFailed = 0x7E02
	// CaParseFailed: caPEM did not yield a usable certificate (hard parse
	// error, or mbedtls reported a failed member in the PEM - any failed
	// member rejects the whole parameter).
	CaParseFailed = 0x7E03
	// WantPollExhausted: Close's bounded close_notify wait ended before the
	// peer exchange completed. The fd and the context are released
	// regardless - Close never fails to clean up; the code only reports
	// that the graceful part did not finish.
	WantPollExhausted = 0x7E04
	// Handshake: any other fatal mbedtls error during the handshake
	// (protocol failure, alert from the peer, out of memory) - the dial
	// loop maps codes its table does not act on to this one. The setup
	// phase never produces it: there the raw negative mbedtls code is
	// passed through (see the error-codes section above).
	Handshake = 0x7E05
	// PsaInitFailed: the glue's defensive psa_crypto_init call failed -
	// the handshake has no random source. IDF initializes PSA at boot, so
	// reaching this means the crypto subsystem itself is broken.
	PsaInitFailed = 0x7E06
)

// lwip errno values the TCP phase can surface, same values the net package
// pins (picolibc numbering - see net.go for the Linux divergences). They
// are redeclared here because the packages stay mutually dependency-free.
const (
	ebadf        = 9   // operation on a closed (or nil) connection
	eagain       = 11  // EWOULDBLOCK: no deadline set, nothing in one quantum
	einval       = 22  // invalid argument
	enametoolong = 91  // name longer than a DNS name may be
	econnreset   = 104 // peer reset the connection
	econnrefused = 111 // nothing listening at the far end
	eaddrinuse   = 112 // local address/port already in use
	enetunreach  = 114 // no route (typically: no IP yet)
	etimedout    = 116 // deadline (or TCP retransmit limit) passed
	ehostunreach = 118 // far host unreachable
	einprogress  = 119 // connect started, still in flight
	ealready     = 120 // connect already in flight
	shutRdwr     = 2   // SHUT_RDWR: both directions
)

// closeBudgetMs bounds Close's close_notify exchange: long enough for a
// live peer to answer within one quantum turn, short enough that Close
// stays a bounded call even against a vanished peer.
const closeBudgetMs = 500

// TLSDial connects a TCP socket to ip:port, then runs the TLS client
// handshake, all within timeoutMs milliseconds (<= 0 is rejected with
// EINVAL - the bounded-quantum model rejects waits without an explicit
// deadline). ip is the in-memory network-order form net.IPv4 produces;
// port is the host-order port number.
//
// caPEM selects the verification mode, see the package doc: nil means
// VERIFY_NONE, otherwise the PEM must parse into a usable CA and
// VERIFY_REQUIRED applies (a failed parse returns CaParseFailed before any
// bytes hit the network, a failed verification CertVerifyFailed during the
// handshake). Either way the returned TLSConn carries plaintext on Read and
// Write.
//
// On every failure path the fd is closed and the C context freed before
// TLSDial returns, so a failed dial leaks nothing. Errors: lwip errno from
// the TCP phase (same as net.Dial), a 0x7E00 TLS code from the phases
// above, or Handshake for unmapped mbedtls failures.
func TLSDial(ip uint32, port uint16, timeoutMs int32, caPEM []byte) (*TLSConn, int32) {
	if timeoutMs <= 0 {
		return nil, einval
	}
	var fd int32
	if e := netSocket(&fd); e != 0 {
		return nil, e
	}
	e := netConnect(fd, ip, int32(port))
	if e != 0 && e != einprogress && e != ealready {
		netClose(fd)
		return nil, e
	}
	deadline := nowMs() + uint32(timeoutMs)
	if e != 0 { // connect in flight: select for writability, then SO_ERROR
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
					break
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

	// TLS setup: config, CA parse, authmode, BIO, ssl_setup. caPEM == nil
	// travels as a zero length, not a nil pointer - the boundary passes
	// integer and pointer pairs, and a Go slice with no elements would
	// pass a nil pointer either way.
	var caPtr *byte
	caLen := int32(0)
	if len(caPEM) > 0 {
		caPtr = &caPEM[0]
		caLen = int32(len(caPEM))
	}
	var h uintptr
	if e := tlsNew(fd, caPtr, caLen, &h); e != 0 {
		netClose(fd)
		return nil, e // CaParseFailed or a raw negative mbedtls code (setup)
	}

	// Handshake by quanta: one handshake_step per loop, a poll quantum for
	// the WANT direction, the same total deadline as the TCP phase.
	for {
		r := tlsHandstep(h)
		if r == 0 {
			return &TLSConn{h: h, fd: fd}, 0
		}
		if r == mbErrWantRead || r == mbErrWantWrite {
			var ready int32
			w := int32(0)
			if r == mbErrWantWrite {
				w = 1
			}
			if pe := netPoll(fd, w, quantumMs, &ready); pe != 0 {
				tlsFree(h)
				netClose(fd)
				return nil, pe
			}
			if past(deadline) {
				tlsFree(h)
				netClose(fd)
				return nil, HandshakeTimeout
			}
			continue
		}
		if r == mbErrCertVerifyFailed {
			tlsFree(h)
			netClose(fd)
			return nil, CertVerifyFailed
		}
		tlsFree(h)
		netClose(fd)
		return nil, Handshake
	}
}

// Read reads up to len(p) bytes of decrypted plaintext and returns the
// count plus an error code. Reads are short: one call returns what fits
// the caller's buffer from the record mbedtls holds; the caller loops until
// it has what it needs. The sentinel combinations:
//
//	n > 0, err == 0     n bytes of plaintext arrived
//	n == 0, err == 0    the peer closed the session in order: TLS
//	                    close_notify arrived (or the TCP stream closed
//	                    cleanly without one)
//	n == 0, err == 11   no deadline set and nothing arrived within one
//	                    quantum (EAGAIN) - a poll-style read
//	n == 0, err == 116  a deadline was set and passed with no data
//	                    (ETIMEDOUT)
//
// Any other error is the raw negative mbedtls code (a fatal session error;
// see ErrName for the common names). A zero-length p is a no-op success
// (it would otherwise be indistinguishable from a peer close).
func (c *TLSConn) Read(p []byte) (int32, int32) {
	if c == nil || c.fd < 0 {
		return 0, ebadf
	}
	if len(p) == 0 {
		return 0, 0
	}
	for {
		var n int32
		e := tlsRead(c.h, &p[0], int32(len(p)), &n)
		if e == 0 {
			return n, 0
		}
		if e == mbErrPeerCloseNotify || e == mbErrConnEOF {
			return 0, 0
		}
		if e == mbErrWantRead || e == mbErrWantWrite {
			if pe := c.pace(e == mbErrWantWrite); pe != 0 {
				return 0, pe
			}
			continue
		}
		return 0, e
	}
}

// Write submits the whole buffer for encryption and returns len(p) on
// success. mbedtls accepts plaintext all-or-nothing (one record, no
// partial plaintext), so the failure contract is simpler than TCP's:
//
//	>= len(p)          the buffer is encrypted and handed to the socket
//	                   (mbedtls may still be retrying the record's bytes
//	                   inside its own BIO loop on later calls)
//	-11 (-EAGAIN)      no deadline set and one quantum passed without
//	                   progress; NOTHING was consumed, retry the whole
//	                   buffer later
//	-116 (-ETIMEDOUT)  a deadline was set and passed; nothing was consumed
//	other negative     the poll reported a socket error (negated lwip
//	                   errno) or a raw fatal mbedtls code - the session
//	                   is unusable either way
func (c *TLSConn) Write(p []byte) int32 {
	if c == nil || c.fd < 0 {
		return -ebadf
	}
	if len(p) == 0 {
		return 0
	}
	for {
		var n int32
		e := tlsWrite(c.h, &p[0], int32(len(p)), &n)
		if e == 0 {
			return n // the whole buffer, or nothing on the error paths below
		}
		if e == mbErrWantRead || e == mbErrWantWrite {
			if pe := c.pace(e == mbErrWantWrite); pe != 0 {
				return -pe // nothing accepted: caller retries all of p
			}
			continue
		}
		return e
	}
}

// Close ends the session: it tries to send the TLS close notification
// within a bounded 500 ms wait (WantPollExhausted if the wait ends first),
// then frees the C context and shuts the TCP socket down in both
// directions - the fd is ALWAYS released, whatever the report. A second
// Close on the same conn is a no-op returning EBADF (9), as is any call on
// a closed or nil conn. mbedtls treats a failed close_notify as
// recoverable (the session is over either way); the TCP shutdown means a
// stalled peer can never hold the close open.
func (c *TLSConn) Close() int32 {
	if c == nil || c.fd < 0 {
		return ebadf
	}
	fd := c.fd
	h := c.h
	c.fd = -1
	c.h = 0
	report := int32(0)
	deadline := nowMs() + closeBudgetMs
	for {
		e := tlsCloseNotify(h)
		if e == 0 {
			break
		}
		if e == mbErrWantRead || e == mbErrWantWrite {
			var ready int32
			w := int32(0)
			if e == mbErrWantWrite {
				w = 1
			}
			if pe := netPoll(fd, w, quantumMs, &ready); pe != 0 {
				report = pe
				break
			}
			if past(deadline) {
				report = WantPollExhausted
				break
			}
			continue
		}
		report = e // fatal mbedtls code: record it, tear down anyway
		break
	}
	tlsFree(h)
	netShutdown(fd, shutRdwr)
	if e := netClose(fd); e != 0 && report == 0 {
		return e
	}
	return report
}

// SetDeadline sets the session's I/O deadline to ms milliseconds from now,
// or clears it with 0. Both Read and Write honor it until SetDeadline
// changes it. ms < 0 is EINVAL. Returns 0 on success.
func (c *TLSConn) SetDeadline(ms int32) int32 {
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
// how the bounded wait ended: 0 = retry the TLS attempt now, eagain = no
// deadline is set and nothing became ready within the quantum, etimedout =
// the session deadline passed. A select failure surfaces as its errno -
// the next TLS attempt would report the same dead socket anyway.
func (c *TLSConn) pace(wantWrite bool) int32 {
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
// "no deadline" is carried by TLSConn.hasDeadline instead of a 0-value
// sentinel.
func past(deadline uint32) bool {
	return uint32(nowMs()-deadline) < 1<<31
}

// ErrName maps the codes this pipeline can hit to their names. Three
// families share one namespace: the 0x7E00 TLS codes this package defines,
// the negative mbedtls codes the setup and record phases pass through, and
// the lwip errno values (picolibc numbering - see net.ErrName for the
// Linux divergences) the TCP phase can surface. A negative value is tried
// against the mbedtls table first, then resolved through the errno table by
// its absolute value (net.ErrName's pattern) - so Write's -EAGAIN contract
// reports "EAGAIN" instead of a hex code. A code from no family prints as
// its value.
func ErrName(code int32) string {
	if code < 0 {
		switch code {
		case mbErrWantRead:
			return "MBEDTLS_ERR_SSL_WANT_READ"
		case mbErrWantWrite:
			return "MBEDTLS_ERR_SSL_WANT_WRITE"
		case mbErrPeerCloseNotify:
			return "MBEDTLS_ERR_SSL_PEER_CLOSE_NOTIFY"
		case mbErrConnEOF:
			return "MBEDTLS_ERR_SSL_CONN_EOF"
		case mbErrCertVerifyFailed:
			return "MBEDTLS_ERR_X509_CERT_VERIFY_FAILED"
		case -0x7780:
			return "MBEDTLS_ERR_SSL_FATAL_ALERT_MESSAGE"
		case -0x7700:
			return "MBEDTLS_ERR_SSL_UNEXPECTED_MESSAGE"
		case -0x7300:
			return "MBEDTLS_ERR_SSL_DECODE_ERROR"
		case -0x7200:
			return "MBEDTLS_ERR_SSL_INVALID_RECORD"
		case -0x7180:
			return "MBEDTLS_ERR_SSL_INVALID_MAC"
		case -0x7080:
			return "MBEDTLS_ERR_SSL_FEATURE_UNAVAILABLE"
		case -0x7000:
			return "MBEDTLS_ERR_SSL_CRYPTO_IN_PROGRESS"
		case -0x7A00:
			return "MBEDTLS_ERR_SSL_BAD_CERTIFICATE"
		case -0x7500:
			return "MBEDTLS_ERR_SSL_UNSUPPORTED_EXTENSION"
		case -0x7800:
			return "MBEDTLS_ERR_SSL_UNRECOGNIZED_NAME"
		case -0x7480:
			return "MBEDTLS_ERR_SSL_NO_CLIENT_CERTIFICATE"
		case -0x004C:
			return "MBEDTLS_ERR_NET_RECV_FAILED"
		case -0x004E:
			return "MBEDTLS_ERR_NET_SEND_FAILED"
		case -0x004F:
			return "MBEDTLS_ERR_NET_CONN_RESET"
		}
		// Not a named mbedtls code. Values in the errno range are the
		// negated lwip errnos the Write contract and the BIO pass through
		// (abs pattern above, 134 = the largest errno named below);
		// anything larger is a raw mbedtls code, printed in its
		// conventional hex form.
		if v := -code; v <= 134 {
			code = v
		} else {
			return "mbedtls -0x" + hex(uint32(v))
		}
	}
	switch code {
	case 0:
		return "OK"
	case HandshakeTimeout:
		return "TLS_ERR_HANDSHAKE_TIMEOUT"
	case CertVerifyFailed:
		return "TLS_ERR_CERT_VERIFY"
	case CaParseFailed:
		return "TLS_ERR_CA_PARSE"
	case WantPollExhausted:
		return "TLS_ERR_WANT_POLL_EXHAUSTED"
	case Handshake:
		return "TLS_ERR_HANDSHAKE"
	case PsaInitFailed:
		return "TLS_ERR_PSA_INIT"
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
	case 134:
		return "ENOTSUP"
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

// hex formats v in lowercase hex (mbedtls error codes are conventionally
// printed in hex; mbedtls -0x6900 names the code unambiguously).
func hex(v uint32) string {
	if v == 0 {
		return "0"
	}
	var buf [8]byte
	i := len(buf)
	for v > 0 {
		i--
		d := v & 15
		if d < 10 {
			buf[i] = byte('0' + d)
		} else {
			buf[i] = byte('a' + d - 10)
		}
		v >>= 4
	}
	return string(buf[i:])
}
