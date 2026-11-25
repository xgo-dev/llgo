// Package ws is a pure-Go RFC 6455 WebSocket client for llgo programs,
// written for the ESP32-S31 ESP-IDF target (pkg/esp family). It performs
// the HTTP Upgrade handshake, frame encoding/decoding with the
// client-to-server masking the RFC requires, automatic ping answering,
// pong tracking and the closing handshake - over a transport the caller
// supplies.
//
// # Dependency inversion
//
// The package does NOT import pkg/esp/net or pkg/esp/tls (its only import
// is the family's shared error base pkg/esp/esperr, which carries no glue
// and no stdlib weight). The transport is the local interface
//
//	type Conn interface {
//	    Read(p []byte) (n int, err error)
//	    Write(p []byte) (n int, err error)
//	    Close() error
//	}
//
// whose method set is exactly what *net.TCPConn and *tls.TLSConn already
// expose (Go interfaces are structural: any type with those three methods
// satisfies ws.Conn). The caller dials first - net.Dial for ws://,
// tls.TLSDial for wss:// - and hands the connection to HandshakeURL. To
// enforce deadlines the transport additionally needs a SetDeadline(ms
// int32) error method (interface Deadliner below); both real transports
// have one.
//
// # Pure Go, no stdlib
//
// The package imports nothing - not even "unsafe". The handshake needs
// SHA-1 and base64 (hash.go); the package keeps its own implementations
// rather than pulling crypto/sha1 and encoding/base64 because the stdlib
// versions drag megabytes of reflect machinery into the image for a
// 20-byte digest (when this package was written they were not even
// linkable on esp32s31-idf - a libffi preprocessing gap since fixed; see
// hash.go for the full history and the image-size rationale). hash.go
// therefore carries its own RFC 3174 SHA-1 and RFC 4648 base64, and the
// handshake response is
// parsed by a hand-rolled buffered reader over Conn - there is no bufio.
// Being glue-free also means there is nothing to copy into the project's
// main component: a ws program links the net and/or tls glues it already
// needs and nothing else, and the package compiles unchanged on a host
// build (its unit tests run with go test).
//
// # Deadlines: delegated, never invented
//
// This package has no clock and starts no goroutines - every bounded wait
// is borrowed from the transport, which already implements the
// bounded-quantum model (20 ms poll quanta that feed the watchdog):
//
//   - HandshakeURL takes timeoutMs (> 0 required) and applies it to the
//     transport's SetDeadline for the whole handshake, clearing the
//     deadline again before it returns. A transport without SetDeadline
//     cannot bound the handshake, so it is rejected up front with
//     ErrNoDeadline.
//   - WSConn.SetDeadline forwards to the transport; with a deadline set,
//     ReadMessage reports ErrTimeout and WriteMessage (0, ErrTimeout) when
//     it passes - the net.Conn semantics of the transport. Without a
//     deadline, ReadMessage is a poll-style read: (0, ErrWouldBlock) after
//     one quantum with nothing arrived, and it can resume mid-frame on the
//     next call (the framing state machine keeps its progress).
//     WriteMessage without a deadline loops over the transport's short
//     writes; a quantum-long stall aborts the call with (0, ErrWouldBlock),
//     leaving a partial frame on the wire - the connection must then be
//     closed. Set a deadline for anything but a controlled loopback peer.
//
// # Scheduling
//
// Same architecture property as net and tls: every bounded wait is a
// C-level select on the transport side that gives the llgo scheduler no
// cooperative yield point, so goroutines do not interleave while one sits
// in a WebSocket read or write. Run one session on one goroutine, or hand
// control between fibers through channel operations at message boundaries.
//
// # Protocol scope
//
//   - RFC 6455 baseline only: no extensions (in particular no
//     permessage-deflate - RSV bits set by a peer are a protocol error),
//     no subprotocols (none are offered in the request).
//   - The client always masks (RFC requirement); server-to-client frames
//     must not be masked and a masked one is rejected as ErrProtocol.
//   - Text frames are not validated as UTF-8: the RFC 6455 section 8.1
//     client MUST is intentionally waived. Whoever passes OpText to
//     WriteMessage guarantees the bytes are valid UTF-8 (use OpBinary for
//     arbitrary bytes); a peer's text frame is delivered as received.
//   - The closing handshake is lenient on the receive side: a peer close
//     frame with a status code is echoed verbatim, even when the code is
//     outside the RFC's valid ranges (the strict answer would be a 1002
//     protocol error). A close frame with no status code is echoed as an
//     EMPTY close payload - a code below 1000 must never go on the wire
//     (RFC 6455 section 7.4.1). CloseCode reports what the peer sent.
//   - Control frames (ping/pong/close) are interleaved-safe: ping is
//     answered with an automatic pong inside the read path, pong is
//     reported to the caller (PongSeen) as a (0, 0) ReadMessage result,
//     and close completes the closing handshake.
//   - A fragmented data message arrives as one ReadMessage result per
//     fragment (the FIN bit is not reassembled into a single message);
//     LastOpcode reports each fragment's own opcode. Peers that fragment
//     appear to the caller as consecutive partial messages - fine for
//     streaming use, and the acceptance peer never fragments.
//   - One frame per ReadMessage call; a frame larger than the caller's
//     buffer is drained and reported as ErrTooBig, leaving the connection
//     usable for the next frame.
//   - The handshake response must fit 512 bytes (any real server's does);
//     headers must occur at most once each (duplicate Sec-WebSocket-Accept
//     is a handshake failure). Response bytes after the header are a
//     handshake failure: a compliant server sends no frames before the
//     client has spoken.
//
// # Randomness
//
// The client key (16 bytes, RFC 6455 section 4.1 asks for unpredictable)
// and the per-frame mask keys come from an in-package xorshift64* PRNG.
// Call Seed once at startup with the best entropy available (an uptime
// counter, a floating ADC pin, RSSI jitter - whatever the board offers).
// Unseeded, the sequence is deterministic: wire-valid and fine for lab
// acceptance runs, but predictable to an intermediary - seed it when the
// path leaves the bench.
//
// # Error model
//
// WS-layer failures are text-first sentinel error values of the shared
// esperr type: ErrHandshake, ErrNotWS, ErrAcceptKey, ErrTooBig,
// ErrProtocol, ErrPeerClosed, ErrConnDead and ErrNoDeadline, each rendering
// its condition ("ws: frame too large") and keeping its internal
// 0x7D00-segment code on Error.Code as the diagnostic annotation (outside
// the lwip errno range, the DNS h_errno range, the esp_err range and the
// tls 0x7E00 segment; ErrName names the codes). Three conventions on top,
// matching the transport packages:
//
//   - Transport failures pass through unchanged: ReadMessage returns the
//     transport's own error (an lwip-errno esperr.Error like
//     ErrTimeout/ErrWouldBlock, or the raw negative mbedtls code of a wss
//     record failure); WriteMessage returns (0, err). A transport EOF
//     without a close frame - the RFC's abnormal closure - becomes
//     ErrConnDead.
//   - Calls on a nil, closed or failed-out connection report ErrClosed
//     (code EBADF 9), the net/tls defensive value.
//   - Close reports the fate of the graceful part: nil when the peer
//     echoed the close frame, the wait's error (ErrWouldBlock/ErrTimeout)
//     when it did not answer within the connection deadline - the
//     underlying transport is ALWAYS released, whatever the report
//     (tls.Close's pattern).
//
// # Ownership
//
// HandshakeURL owns the transport from the moment it is called: on any
// failure the transport is closed before it returns (a failed handshake
// leaks nothing), and after success the WSConn owns it - WSConn.Close
// closes the underlying transport exactly once, and every later call on
// the WSConn is a defensive EBADF.
package ws

import "github.com/xgo-dev/llgo/pkg/esp/esperr"

// WebSocket opcodes (RFC 6455 section 5.2).
const (
	OpCont   = 0x0 // continuation frame of a fragmented message
	OpText   = 0x1 // UTF-8 data frame
	OpBinary = 0x2 // binary data frame
	OpClose  = 0x8 // closing handshake
	OpPing   = 0x9 // keepalive probe (answered automatically)
	OpPong   = 0xA // keepalive answer (reported via PongSeen)
)

// WS-layer error codes, the internal 0x7D00 segment. Callers match on the
// sentinel errors below; the values themselves stay inside this package
// (ErrName's table and the sentinels).
const (
	wsErrHandshake  = 0x7D01
	wsErrNotWS      = 0x7D02
	wsErrAcceptKey  = 0x7D03
	wsErrTooBig     = 0x7D04
	wsErrProtocol   = 0x7D05
	wsErrPeerClosed = 0x7D06
	wsErrConnDead   = 0x7D07
	wsErrNoDeadline = 0x7D08
)

// WS-layer failure sentinels, text-first: errors.Is tells the classes
// apart, and each keeps its historical 0x7D00-segment code on Error.Code
// and its ErrName entry on Error.Name as the diagnostic annotation (the
// package has no C-side producer for these codes, but code-era matching -
// errors.Is against a raw 0x7Dxx Error - and the field-level diagnostics
// keep working). See the package doc's error-model section.
var (
	// ErrHandshake: the peer's response was not a usable HTTP/1.x upgrade
	// response - malformed status line, a header bigger than the 512-byte
	// response window, garbage after the header, or a missing required
	// header. (A 101 with wrong Upgrade/Connection headers is ErrNotWS.)
	ErrHandshake = newSentinel("ws: malformed handshake response", wsErrHandshake)
	// ErrNotWS: the response is HTTP but not a WebSocket upgrade - the
	// status is not "101", or Upgrade/Connection do not carry the required
	// tokens (websocket / upgrade). The classic shape when the far end is
	// not a WebSocket server (a plain echo, a proxy, an error page).
	ErrNotWS = newSentinel("ws: not a websocket upgrade", wsErrNotWS)
	// ErrAcceptKey: Sec-WebSocket-Accept did not equal
	// base64(SHA-1(key + GUID)) for the key we sent - either the peer is
	// broken or something in between answered.
	ErrAcceptKey = newSentinel("ws: accept key mismatch", wsErrAcceptKey)
	// ErrTooBig: a data frame did not fit the caller's ReadMessage buffer.
	// The frame was drained, so the connection stays usable; the message
	// content is lost.
	ErrTooBig = newSentinel("ws: frame too large", wsErrTooBig)
	// ErrProtocol: the peer violated the framing rules - a masked
	// server-to-client frame, RSV bits set (an extension we did not
	// negotiate), a fragmented or oversize control frame, an unknown
	// opcode. A violation means the byte stream is misaligned, so the
	// session is dead on the spot: the transport is released before the
	// error is reported (every later call reports ErrClosed). No close
	// frame is sent on the way out - a deliberate deviation from the
	// RFC 6455 section 7.1.7 SHOULD to answer a protocol error with a 1002
	// close: a misaligned stream cannot promise a well-formed closing
	// exchange.
	ErrProtocol = newSentinel("ws: protocol error", wsErrProtocol)
	// ErrPeerClosed: the orderly end of a session, not a failure - the
	// peer sent a close frame, we echoed it, the transport is closed.
	// CloseCode carries the peer's code (1000 normal, 1001 going away,
	// ...). Close() reports nil for this same event when WE initiated.
	ErrPeerClosed = newSentinel("ws: peer closed", wsErrPeerClosed)
	// ErrConnDead: the underlying transport closed (or failed) without a
	// WebSocket close frame - the RFC's abnormal closure. The transport is
	// already gone; just stop using the WSConn.
	ErrConnDead = newSentinel("ws: connection dead", wsErrConnDead)
	// ErrNoDeadline: the transport has no SetDeadline method, so the
	// bounded-wait model cannot be honored - HandshakeURL refuses to run
	// unbounded (its timeoutMs cannot be enforced) and WSConn.SetDeadline
	// has nothing to forward to.
	ErrNoDeadline = newSentinel("ws: no deadline support", wsErrNoDeadline)
)

// newSentinel declares a semantic sentinel that keeps its historical
// 0x7D00 segment code on Error.Code and its ErrName entry on Error.Name as
// the diagnostic annotation (esperr.New is text-only; the number stays
// inspectable, and code-era matching and the code-field diagnostics keep
// working).
func newSentinel(text string, code int32) *esperr.Error {
	e, _ := esperr.New(text).(*esperr.Error)
	e.Code = code
	e.Name = ErrName(code)
	return e
}

// The transport error sentinels, re-exported from the shared esperr base
// so callers can stay inside this package's namespace.
var (
	ErrClosed     = esperr.ErrClosed
	ErrTimeout    = esperr.ErrTimeout
	ErrWouldBlock = esperr.ErrWouldBlock
	ErrEOF        = esperr.ErrEOF
)

// errOf converts a pipeline code to the error the API returns (nil for 0).
// The sentinel codes (the 0x7D00 segment and the transport errnos) map to
// their sentinel values; anything else (an lwip errno passed through from
// a transport-shaped code path) forwards as a bare esperr.Coded value
// named through ErrName - a forwarded code is its own diagnostic, and the
// bare *esperr.Error keeps the field-level code/name diagnostics
// byte-identical.
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
	case wsErrHandshake:
		return ErrHandshake
	case wsErrNotWS:
		return ErrNotWS
	case wsErrAcceptKey:
		return ErrAcceptKey
	case wsErrTooBig:
		return ErrTooBig
	case wsErrProtocol:
		return ErrProtocol
	case wsErrPeerClosed:
		return ErrPeerClosed
	case wsErrConnDead:
		return ErrConnDead
	case wsErrNoDeadline:
		return ErrNoDeadline
	}
	return esperr.Coded(code, ErrName(code))
}

// lwip errno values this package acts on, same values the net package pins
// (picolibc numbering; net.go lists the Linux divergences). They are
// redeclared because the ws package imports no sibling transport package -
// not even pkg/esp/net.
const (
	ebadf     = 9   // call on a nil, closed or failed-out connection
	eagain    = 11  // poll-style read: nothing within one quantum
	einval    = 22  // invalid argument
	etimedout = 116 // deadline passed
)

// wsGUID is the RFC 6455 section 1.3 magic GUID of the accept key.
const wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// handshakeBufSize bounds the handshake response: the status line plus the
// required headers of any real server fit easily.
const handshakeBufSize = 512

// Conn is the transport interface: the method set *net.TCPConn and
// *tls.TLSConn already expose (structural match, no import). Read follows
// the net.Conn contract of the pkg/esp family: n bytes read with a nil
// error, (0, io.EOF) meaning the stream closed cleanly, (0,
// ErrWouldBlock) meaning "nothing in one quantum" when no deadline is set,
// (0, ErrTimeout) when a deadline passed. Write writes p and returns how
// many bytes were accepted plus an error - the io.Writer convention.
type Conn interface {
	Read(p []byte) (n int, err error)
	Write(p []byte) (n int, err error)
	Close() error
}

// Deadliner is the optional transport deadline interface: SetDeadline(ms
// int32) error with net's semantics (relative ms from now, 0 clears, < 0
// EINVAL). HandshakeURL requires it (the handshake budget must be
// enforceable); *net.TCPConn and *tls.TLSConn both satisfy it.
type Deadliner interface {
	SetDeadline(ms int32) error
}

// WSConn is one WebSocket session over a caller-supplied transport. All
// methods may be called on a nil receiver and return EBADF after Close or
// a protocol failure, like the pkg/esp transports. One session, one
// goroutine (see the package doc's scheduling note).
type WSConn struct {
	c  Conn
	dl Deadliner

	closed    bool   // framing is over and the transport is released
	closeSent bool   // we already sent our close frame
	peerCode  uint16 // code the peer put in its close frame
	pongSeen  bool   // a pong arrived since the last manual check
	lastOp    uint8  // opcode of the last delivered frame

	// Read state machine. rbuf stages the frame header (2 + 8 bytes is the
	// RFC maximum for a server-to-client frame, which never carries a
	// mask); rhave/rneed survive poll-mode EAGAIN returns so a retried
	// ReadMessage resumes mid-header, and fgot does the same mid-payload.
	rbuf  [10]byte
	rhave int32
	rneed int32
	inHdr bool
	ffin  bool
	fop   uint8
	flen  int32
	fgot  int32

	// ctrl stages control-frame payloads (the RFC caps them at 125 bytes)
	// and doubles as the drain buffer for oversize data frames.
	ctrl [125]byte
}

// prngState backs the client key and the per-frame mask keys; Seed sets it.
var prngState uint64 = 0x2545F4914F6CDD1D

// Seed seeds the handshake-key / mask-key PRNG (xorshift64*) once at
// startup. seed == 0 is ignored (a zero state would lock up). Unseeded the
// sequence is deterministic - fine on the bench, predictable through an
// intermediary (see the package doc's randomness section).
func Seed(seed uint64) {
	if seed != 0 {
		prngState = seed
	}
}

// prngFill refills p from the PRNG.
func prngFill(p []byte) {
	for i := range p {
		x := prngState
		x ^= x >> 12
		x ^= x << 25
		x ^= x >> 27
		prngState = x
		p[i] = byte((x * 0x2545F4914F6CDD1D) >> 56)
	}
}

// AcceptKey computes the RFC 6455 section 4.2.2 Sec-WebSocket-Accept value
// for a client key: base64(SHA-1(key + GUID)). HandshakeURL verifies the
// server's header against this; exporting it also lets a program check a
// key/value pair without running a session, and gives pkg/esp programs
// their only reachable SHA-1/base64 combination.
func AcceptKey(clientKey []byte) string {
	buf := make([]byte, 0, len(clientKey)+len(wsGUID))
	buf = append(buf, clientKey...)
	buf = append(buf, wsGUID...)
	sum := SHA1Sum(buf)
	return Base64StdEncode(sum[:])
}

// HandshakeURL runs the RFC 6455 opening handshake over the already
// connected transport c (net.Dial it for ws://, tls.TLSDial it for wss://)
// and returns the session. host is the Host header value ("host:port",
// e.g. "192.168.4.1:9000"), path the request target (e.g. "/ws"),
// timeoutMs the total budget for writing the request and reading the
// response (must be > 0; the bounded-wait model refuses unbounded waits).
//
// The request carries exactly the headers RFC 6455 requires - GET, Host,
// Upgrade: websocket, Connection: Upgrade, a base64 16-byte
// Sec-WebSocket-Key and Sec-WebSocket-Version: 13 - and nothing else: no
// subprotocols, no extensions. The response must be a 101 with the
// required tokens and a Sec-WebSocket-Accept equal to AcceptKey(our key);
// anything else is one of ErrHandshake / ErrNotWS / ErrAcceptKey.
//
// From the call onward the transport belongs to HandshakeURL: on any
// failure it is closed before returning; on success the returned WSConn
// owns it (and the handshake deadline is cleared again, so the caller's
// SetDeadline decides the session's pacing).
func HandshakeURL(c Conn, host, path string, timeoutMs int32) (*WSConn, error) {
	if c == nil {
		return nil, ErrClosed
	}
	if timeoutMs <= 0 {
		return nil, errOf(einval)
	}
	dl, ok := c.(Deadliner)
	if !ok || dl == nil {
		return nil, ErrNoDeadline
	}
	if err := dl.SetDeadline(timeoutMs); err != nil {
		return nil, err
	}
	fail := func(err error) (*WSConn, error) {
		dl.SetDeadline(0)
		c.Close()
		return nil, err
	}

	// Request line and required headers. The key is 16 PRNG bytes,
	// base64-encoded (24 chars).
	var key [16]byte
	prngFill(key[:])
	req := "GET " + path + " HTTP/1.1\r\n" +
		"Host: " + host + "\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: " + Base64StdEncode(key[:]) + "\r\n" +
		"Sec-WebSocket-Version: 13\r\n\r\n"
	rb := []byte(req)
	total := 0
	for total < len(rb) {
		r, werr := c.Write(rb[total:])
		if werr != nil {
			return fail(werr)
		}
		if r == 0 {
			return fail(ErrWouldBlock) // a whole-quantum stall
		}
		total += r
	}

	// Read the response header section: accumulate up to handshakeBufSize
	// bytes until the CRLFCRLF terminator. With the deadline set, every
	// read blocks until data or ETIMEDOUT - no poll-mode resume needed
	// here, the budget bounds the whole phase.
	var resp [handshakeBufSize]byte
	n, scanned := int32(0), int32(0)
	for {
		m, rerr := c.Read(resp[n:])
		if m > 0 {
			n += int32(m)
			if headerEnd(resp[:n], &scanned) {
				break
			}
			if n == int32(len(resp)) {
				return fail(ErrHandshake)
			}
			continue
		}
		if rerr == nil || rerr == ErrEOF {
			return fail(ErrConnDead) // transport closed without a response
		}
		return fail(rerr)
	}
	if n != scanned { // bytes after the header: server spoke first
		return fail(ErrHandshake)
	}
	if err := parseResponse(resp[:scanned], Base64StdEncode(key[:])); err != nil {
		return fail(err)
	}
	dl.SetDeadline(0)
	return &WSConn{c: c, dl: dl, inHdr: true, rneed: 2}, nil
}

// headerEnd reports whether resp[:n] now contains the CRLFCRLF that ends
// an HTTP header section; *scanned tracks how far the search has gone so
// successive reads never rescan the whole buffer (only the last three
// bytes are retried, in case the terminator spans two reads). *scanned
// receives the index just past the terminator.
func headerEnd(resp []byte, scanned *int32) bool {
	n := int32(len(resp))
	lo := *scanned
	if hi := n - 3; lo > hi {
		lo = hi
	}
	if lo < 0 {
		lo = 0
	}
	for i := lo; i+4 <= n; i++ {
		if resp[i] == '\r' && resp[i+1] == '\n' && resp[i+2] == '\r' && resp[i+3] == '\n' {
			*scanned = i + 4
			return true
		}
	}
	*scanned = n - 3
	if *scanned < 0 {
		*scanned = 0
	}
	return false
}

// parseResponse validates the handshake response accumulated in resp (up
// to but excluding the trailing CRLFCRLF): status line "HTTP/1.1 101"
// (HTTP/1.1 only),
// Upgrade carrying the websocket token, Connection carrying upgrade, and
// Sec-WebSocket-Accept equal to the expected accept value. A duplicate
// required header is a failure (the response window is 512 bytes; any real
// server's response fits easily).
func parseResponse(resp []byte, keyB64 string) error {
	lineEnd := indexCRLF(resp, 0)
	if lineEnd < 0 {
		return ErrHandshake
	}
	line := resp[:lineEnd]
	// "HTTP/1.1 101 anything" - HTTP/1.1 only: the literal "HTTP/1.1"
	// prefix (HTTP/1.0 and every other version is rejected below), one
	// space, then the 3-digit code.
	if len(line) < 12 || !prefixBytes(line, "HTTP/1.") || line[7] != '1' ||
		line[8] != ' ' || line[9] != '1' || line[10] != '0' || line[11] != '1' {
		return ErrNotWS
	}

	upgrade, connection, accept := false, false, false
	pos := lineEnd + 2
	for pos < int32(len(resp)) {
		if pos+1 < int32(len(resp)) && resp[pos] == '\r' && resp[pos+1] == '\n' {
			break // empty line: end of headers (defensive; headerEnd checked)
		}
		lineEnd = indexCRLF(resp, pos)
		if lineEnd < 0 {
			return ErrHandshake
		}
		line = resp[pos:lineEnd]
		pos = lineEnd + 2
		colon := int32(-1)
		for i, b := range line {
			if b == ':' {
				colon = int32(i)
				break
			}
		}
		if colon <= 0 {
			return ErrHandshake
		}
		name := line[:colon]
		value := trimOWS(line[colon+1:])
		switch {
		case eqFoldASCII(name, "upgrade"):
			if upgrade {
				return ErrHandshake
			}
			upgrade = hasTokenFold(value, "websocket")
		case eqFoldASCII(name, "connection"):
			if connection {
				return ErrHandshake
			}
			connection = hasTokenFold(value, "upgrade")
		case eqFoldASCII(name, "sec-websocket-accept"):
			if accept {
				return ErrHandshake
			}
			accept = bytesEqual(value, []byte(AcceptKey([]byte(keyB64))))
		}
	}
	if !upgrade || !connection {
		return ErrNotWS
	}
	if !accept {
		return ErrAcceptKey
	}
	return nil
}

// indexCRLF finds the next "\r\n" in p at or after from, or -1.
func indexCRLF(p []byte, from int32) int32 {
	for i := from; i+1 < int32(len(p)); i++ {
		if p[i] == '\r' && p[i+1] == '\n' {
			return i
		}
	}
	return -1
}

// prefixBytes reports whether p starts with s.
func prefixBytes(p []byte, s string) bool {
	if len(p) < len(s) {
		return false
	}
	for i := 0; i < len(s); i++ {
		if p[i] != s[i] {
			return false
		}
	}
	return true
}

// trimOWS trims optional whitespace (RFC 7230 OWS: space/tab) both sides.
func trimOWS(p []byte) []byte {
	for len(p) > 0 && (p[0] == ' ' || p[0] == '\t') {
		p = p[1:]
	}
	for len(p) > 0 && (p[len(p)-1] == ' ' || p[len(p)-1] == '\t') {
		p = p[:len(p)-1]
	}
	return p
}

// eqFoldASCII compares a byte slice against a string case-insensitively
// (ASCII).
func eqFoldASCII(a []byte, s string) bool {
	if len(a) != len(s) {
		return false
	}
	for i := 0; i < len(s); i++ {
		ca, cs := a[i], s[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cs >= 'A' && cs <= 'Z' {
			cs += 'a' - 'A'
		}
		if ca != cs {
			return false
		}
	}
	return true
}

// bytesEqual reports whether a and b are identical.
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// hasTokenFold reports whether the comma-separated token list value
// contains tok, case-insensitively ("keep-alive, Upgrade" contains
// "upgrade").
func hasTokenFold(value []byte, tok string) bool {
	start := 0
	for start <= len(value) {
		end := start
		for end < len(value) && value[end] != ',' {
			end++
		}
		t := trimOWS(value[start:end])
		if len(t) == len(tok) {
			ok := true
			for i := 0; i < len(tok); i++ {
				cv, ct := t[i], tok[i]
				if cv >= 'A' && cv <= 'Z' {
					cv += 'a' - 'A'
				}
				if cv != ct {
					ok = false
					break
				}
			}
			if ok {
				return true
			}
		}
		if end == len(value) {
			break
		}
		start = end + 1
	}
	return false
}

// ReadMessage receives the next frame. One call returns one event:
//
//	n > 0, nil          n payload bytes of a data frame (a frame bigger
//	                    than len(p) is drained and reported as ErrTooBig -
//	                    the connection stays usable)
//	0, nil              a pong control frame was consumed (PongSeen is now
//	                    true) or a zero-length data frame arrived - call
//	                    again
//	0, ErrPeerClosed    the peer completed the closing handshake; the
//	                    transport is closed, CloseCode has the peer's
//	                    code. The orderly session end, not a failure.
//	0, ErrWouldBlock    poll mode (no deadline): nothing within one
//	                    quantum; retry with the SAME buffer - the call
//	                    resumes mid-header or mid-payload
//	0, ErrTimeout       the connection deadline passed
//	otherwise           ErrTooBig/ErrProtocol (see the sentinels),
//	                    ErrConnDead (transport closed without a close
//	                    frame), or the raw transport error (an lwip-errno
//	                    esperr.Error, or the negative mbedtls code of a
//	                    wss record failure)
//
// Pings are answered automatically inside the call; a pong is never
// delivered as data. ErrProtocol leaves the session closed (the framing is
// misaligned; there is no recovery).
func (w *WSConn) ReadMessage(p []byte) (int, error) {
	if w == nil || w.closed {
		return 0, ErrClosed
	}
	for {
		if w.inHdr {
			if e := w.readHeader(); e != nil {
				return 0, e
			}
			switch w.fop {
			case OpText, OpBinary, OpCont:
				w.lastOp = w.fop
				if w.flen == 0 { // zero-length data frame
					w.resetHeader()
					return 0, nil
				}
			case OpPing, OpPong, OpClose:
				// control: payload staged into w.ctrl below
			default:
				return 0, ErrProtocol // reserved opcode (0x3-0x7, 0xB-0xF)
			}
		}
		if w.fgot == w.flen { // this frame's payload is complete
			switch w.fop {
			case OpPing, OpPong, OpClose:
				if e := w.controlEvent(); e != nil {
					return 0, e
				}
				if w.fop == OpPong {
					return 0, nil // the pong event the caller pumps for
				}
				continue // ping answered automatically; keep reading
			}
			// data frame complete
			n, drained := w.flen, w.flen > int32(len(p))
			w.resetHeader()
			if drained {
				return 0, ErrTooBig // frame drained; the connection stays usable
			}
			return int(n), nil
		}
		// One read attempt into the payload's destination: the caller's
		// buffer when the frame fits, the control/drain staging otherwise
		// (a chunk at a time for a frame larger than the staging buffer).
		chunk := w.flen - w.fgot
		if chunk > int32(len(w.ctrl)) {
			chunk = int32(len(w.ctrl))
		}
		dst := w.ctrl[:chunk]
		if w.fop != OpPing && w.fop != OpPong && w.fop != OpClose && w.flen <= int32(len(p)) {
			dst = p[w.fgot:w.flen]
		}
		n, rerr := w.c.Read(dst)
		if n > 0 {
			w.fgot += int32(n)
			// Short reads are the transport's normal shape (TCP delivers
			// what the segment carried); keep accumulating the frame.
			continue
		}
		if rerr == nil || rerr == ErrEOF {
			return 0, ErrConnDead // transport closed mid-frame
		}
		return 0, rerr // would-block / timeout / transport error, progress kept
	}
}

// resetHeader re-arms the read state machine for the next frame header.
func (w *WSConn) resetHeader() {
	w.inHdr = true
	w.rhave = 0
	w.rneed = 2
	w.fgot = 0
	w.flen = 0
}

// controlEvent dispatches a completed control frame (payload in
// w.ctrl[:flen]) and re-arms the header state. Data frames never come
// here.
func (w *WSConn) controlEvent() error {
	switch w.fop {
	case OpPing:
		payload := w.ctrl[:w.flen]
		w.resetHeader()
		if err := w.writeFrame(OpPong, payload); err != nil {
			return err
		}
	case OpPong:
		w.pongSeen = true
		w.resetHeader()
		return nil // the one control event reported to the caller
	case OpClose:
		code := uint16(0)
		if w.flen >= 2 {
			code = uint16(w.ctrl[0])<<8 | uint16(w.ctrl[1])
		}
		w.peerCode = code
		w.closed = true
		if !w.closeSent { // echo the close frame to complete the handshake.
			// A status code is echoed verbatim, even an invalid one (the
			// lenient choice - see the package doc). With no status code the
			// echo carries an EMPTY payload: a 0 code must not go on the
			// wire (RFC 6455 section 7.4.1 reserves 0-999).
			if w.flen >= 2 {
				var pl [2]byte
				pl[0] = byte(code >> 8)
				pl[1] = byte(code)
				w.writeFrame(OpClose, pl[:])
			} else {
				w.writeFrame(OpClose, nil)
			}
		}
		w.c.Close()
		return ErrPeerClosed
	}
	return nil
}

// readHeader accumulates and decodes the next frame header across poll-mode
// retries. Server-to-client frames must not be masked (a masked one is
// ErrProtocol), RSV bits must be clear (no extensions negotiated), control
// frames must be unfragmented and at most 125 bytes.
func (w *WSConn) readHeader() error {
	for w.rhave < w.rneed {
		n, e := w.c.Read(w.rbuf[w.rhave:w.rneed])
		if n > 0 {
			w.rhave += int32(n)
			if w.rhave == 2 && w.rneed == 2 {
				if w.rbuf[1]&0x80 != 0 {
					return w.fatal(ErrProtocol) // server frames carry no mask
				}
				switch w.rbuf[1] & 0x7f {
				case 126:
					w.rneed = 4
				case 127:
					w.rneed = 10
				}
			}
			continue
		}
		if e == nil || e == ErrEOF {
			return ErrConnDead
		}
		return e
	}
	b0, b1 := w.rbuf[0], w.rbuf[1]
	if b0&0x70 != 0 {
		return w.fatal(ErrProtocol) // RSV bits: an extension we did not negotiate
	}
	if b1&0x80 != 0 {
		// Authoritative mask check (the incremental one above only fires
		// when this call read the first two bytes; a resumed decode of an
		// aborted frame's staged header must not pass without it).
		return w.fatal(ErrProtocol)
	}
	w.ffin = b0&0x80 != 0
	w.fop = b0 & 0x0f
	length := int32(b1 & 0x7f)
	switch b1 & 0x7f {
	case 126:
		length = int32(uint32(w.rbuf[2])<<8 | uint32(w.rbuf[3]))
	case 127:
		var v uint64
		for _, b := range w.rbuf[2:10] {
			v = v<<8 | uint64(b)
		}
		if v>>31 != 0 {
			return w.fatal(ErrTooBig) // no int32 frame can be this long
		}
		length = int32(v)
	}
	if w.fop >= 0x8 { // control frames: FIN required, payload ≤ 125
		if !w.ffin || length > int32(len(w.ctrl)) {
			return w.fatal(ErrProtocol)
		}
	}
	w.flen = length
	w.inHdr = false
	w.fgot = 0
	return nil
}

// fatal marks the session dead and releases the transport: a framing
// violation (or an impossible length) means the byte stream is misaligned
// and no further frame can be decoded - there is no recovery but Close.
// No close frame is sent (the RFC 6455 section 7.1.7 SHOULD to answer
// with 1002 is deliberately not followed): a misaligned stream cannot
// promise a well-formed closing exchange.
func (w *WSConn) fatal(err error) error {
	w.closed = true
	w.c.Close()
	return err
}

// WriteMessage sends one data frame with the given opcode (OpText or
// OpBinary; anything else is an EINVAL error - control frames have their
// own methods). The frame is built masked (the RFC requires client
// masking) and pushed whole; it returns (len(p), nil) on success and
// (0, err) on failure - the frame is all-or-nothing up to the transport:
//
//	0, ErrWouldBlock   no deadline set and the transport stalled for one
//	                   quantum: the connection must be closed (a partial
//	                   frame is already on the wire)
//	0, ErrTimeout      the deadline passed; nothing went out
//	0, err             transport failure (an lwip-errno esperr.Error, or
//	                   the raw negative mbedtls code over wss)
func (w *WSConn) WriteMessage(p []byte, opcode uint8) (int, error) {
	if w == nil || w.closed {
		return 0, ErrClosed
	}
	if opcode != OpText && opcode != OpBinary {
		return 0, errOf(einval)
	}
	if err := w.writeFrame(opcode, p); err != nil {
		return 0, err
	}
	return len(p), nil
}

// writeFrame builds and sends one masked client frame (any opcode). It
// returns nil once the whole frame is out - 0-length payloads included,
// which is what Ping() and the close handshake surface - or the transport
// error / ErrWouldBlock (the quantum stall; see WriteMessage).
func (w *WSConn) writeFrame(opcode uint8, payload []byte) error {
	n := len(payload)
	hdr := 2
	switch {
	case n < 126:
	case n <= 0xFFFF:
		hdr = 4
	default:
		hdr = 10
	}
	buf := make([]byte, hdr+4+n)
	buf[0] = 0x80 | opcode
	switch hdr {
	case 2:
		buf[1] = 0x80 | byte(n)
	case 4:
		buf[1] = 0x80 | 126
		buf[2] = byte(n >> 8)
		buf[3] = byte(n)
	default:
		buf[1] = 0x80 | 127
		for k := 0; k < 8; k++ {
			buf[2+k] = byte(uint64(n) >> (8 * (7 - k)))
		}
	}
	var mask [4]byte
	prngFill(mask[:])
	copy(buf[hdr:], mask[:])
	copy(buf[hdr+4:], payload)
	for i := range payload {
		buf[hdr+4+i] ^= mask[i&3]
	}
	total := 0
	for total < len(buf) {
		r, err := w.c.Write(buf[total:])
		if err != nil {
			return err
		}
		if r == 0 {
			return ErrWouldBlock
		}
		total += r
	}
	return nil
}

// Ping sends an empty ping frame; the error contract is WriteMessage's.
// The peer's pong is reported by the next ReadMessage as a (0, nil) event
// with PongSeen() true - pump ReadMessage until PongSeen() to wait for it.
func (w *WSConn) Ping() error {
	if w == nil || w.closed {
		return ErrClosed
	}
	return w.writeFrame(OpPing, nil)
}

// Close performs the closing handshake with the given code (1000 normal
// closure, 1001 going away, 3000-4999 application ranges) and releases the
// underlying transport. It sends the close frame, then pumps the read path
// until the peer's close frame arrives (answering pings on the way) or the
// wait ends: the connection deadline bounds the wait - SetDeadline first
// for a bounded Close; without a deadline one quantum of silence is
// enough. The transport is closed exactly once, whatever the report:
//
//	nil              graceful: the peer echoed the close frame (CloseCode
//	                 now carries it)
//	ErrWouldBlock    no deadline and the peer stayed silent for one quantum
//	ErrTimeout       the deadline passed without the peer's close frame
//	other err        the close frame itself could not be sent
//
// A second Close, or any call on a closed WSConn, reports ErrClosed.
func (w *WSConn) Close(code uint16) error {
	if w == nil || w.closed {
		return ErrClosed
	}
	var pl [2]byte
	pl[0] = byte(code >> 8)
	pl[1] = byte(code)
	err := w.writeFrame(OpClose, pl[:])
	w.closeSent = true
	if err != nil { // the close frame never went out: just release the transport
		w.closed = true
		w.c.Close()
		return err
	}
	// Wait for the peer's close frame. Data frames that arrive on the way
	// are drained (ReadMessage(nil) always takes the drain path); pongs
	// and pings are consumed as usual.
	var report error = ErrWouldBlock
	for {
		_, re := w.ReadMessage(nil)
		if re == ErrPeerClosed {
			return nil
		}
		if re != nil {
			report = re
			break
		}
	}
	w.closed = true
	w.c.Close()
	return report
}

// SetDeadline sets the session's I/O deadline to ms milliseconds from now
// (0 clears it) by forwarding to the transport, so ReadMessage and
// WriteMessage honor it exactly like the transport's own Read/Write. ms <
// 0 is EINVAL. A transport without SetDeadline reports ErrNoDeadline.
func (w *WSConn) SetDeadline(ms int32) error {
	if w == nil || w.closed {
		return ErrClosed
	}
	if ms < 0 {
		return errOf(einval)
	}
	if w.dl == nil {
		return ErrNoDeadline
	}
	return w.dl.SetDeadline(ms)
}

// PongSeen reports whether a pong frame arrived since the last call. It
// never resets implicitly: poll it after Ping to wait for the answer
// (ReadMessage reports the pong as a (0, 0) event).
func (w *WSConn) PongSeen() bool {
	if w == nil {
		return false
	}
	seen := w.pongSeen
	w.pongSeen = false
	return seen
}

// CloseCode returns the close code the peer sent in its close frame (0 if
// none arrived yet). Meaningful after Close returned 0 or ReadMessage
// returned ErrPeerClosed.
func (w *WSConn) CloseCode() uint16 {
	if w == nil {
		return 0
	}
	return w.peerCode
}

// LastOpcode returns the opcode of the last delivered data frame (OpText,
// OpBinary or OpCont for a fragment).
func (w *WSConn) LastOpcode() uint8 {
	if w == nil {
		return 0
	}
	return w.lastOp
}

// ErrName maps a code from this package's pipeline to its name: the 0x7D00
// WS segment, the lwip errno values a TCP transport surfaces (picolibc
// numbering; a negative value is resolved through its absolute value, the
// net.ErrName pattern - WriteMessage's -eagain contract reads as "EAGAIN"),
// and anything else as its value; a large negative value is a raw mbedtls
// code from a wss record failure, printed in its conventional hex form
// (tls.ErrName names the common ones precisely).
func ErrName(code int32) string {
	neg := false
	if code < 0 {
		neg = true
		code = -code
	}
	name := ""
	switch code {
	case wsErrHandshake:
		name = "WS_ERR_HANDSHAKE"
	case wsErrNotWS:
		name = "WS_ERR_NOT_WEBSOCKET"
	case wsErrAcceptKey:
		name = "WS_ERR_ACCEPT_KEY"
	case wsErrTooBig:
		name = "WS_ERR_FRAME_TOO_BIG"
	case wsErrProtocol:
		name = "WS_ERR_PROTOCOL"
	case wsErrPeerClosed:
		name = "WS_ERR_PEER_CLOSED"
	case wsErrConnDead:
		name = "WS_ERR_CONN_DEAD"
	case wsErrNoDeadline:
		name = "WS_ERR_NO_DEADLINE"
	default:
		name = errnoName(code)
	}
	if neg {
		return "-" + name
	}
	return name
}

// errnoName names the lwip errno values (picolibc numbering) that a TCP
// transport can surface; unknown values print as "errno N".
func errnoName(code int32) string {
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
	case 116:
		return "ETIMEDOUT"
	case 118:
		return "EHOSTUNREACH"
	case 119:
		return "EINPROGRESS"
	case 120:
		return "EALREADY"
	case 128:
		return "ENOTCONN"
	case 134:
		return "ENOTSUP"
	}
	return "errno " + dec(uint32(code))
}

// dec formats v in decimal (no fmt/strconv in this package).
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
