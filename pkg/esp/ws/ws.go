// Package ws is a pure-Go RFC 6455 WebSocket client for llgo programs,
// written for the ESP32-S31 ESP-IDF target (pkg/esp family). It performs
// the HTTP Upgrade handshake, frame encoding/decoding with the
// client-to-server masking the RFC requires, automatic ping answering,
// pong tracking and the closing handshake - over a transport the caller
// supplies.
//
// # Dependency inversion
//
// The package does NOT import pkg/esp/net or pkg/esp/tls. The transport is
// the local interface
//
//	type Conn interface {
//	    Read(p []byte) (n int32, err int32)
//	    Write(p []byte) int32
//	    Close() int32
//	}
//
// whose method set is exactly what *net.TCPConn and *tls.TLSConn already
// expose (Go interfaces are structural: any type with those three methods
// satisfies ws.Conn). The caller dials first - net.Dial for ws://,
// tls.TLSDial for wss:// - and hands the connection to HandshakeURL. To
// enforce deadlines the transport additionally needs a SetDeadline(ms
// int32) int32 method (interface Deadliner below); both real transports
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
//     ReadMessage returns ETIMEDOUT (116) and WriteMessage a negative
//     -ETIMEDOUT when it passes - the net.Conn semantics of the transport.
//     Without a deadline, ReadMessage is a poll-style read: (0, eagain)
//     after one quantum with nothing arrived, and it can resume mid-frame
//     on the next call (the framing state machine keeps its progress).
//     WriteMessage without a deadline loops over the transport's short
//     writes; a quantum-long stall aborts the call with -eagain leaving a
//     partial frame on the wire - the connection must then be closed. Set
//     a deadline for anything but a controlled loopback peer.
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
// # Error codes
//
// WS-layer failures use the 0x7D00 segment, outside the lwip errno range,
// the DNS h_errno range, the esp_err range and the tls 0x7E00 segment
// (ErrName lists them). Three conventions on top, matching the transport
// packages:
//
//   - Transport failures pass through unchanged: ReadMessage returns the
//     raw code (lwip errno like ETIMEDOUT/EAGAIN, or the raw negative
//     mbedtls code of a wss record failure), WriteMessage returns the
//     negated code (its success value is the payload length, like
//     TCPConn.Write's, so failures are negative). A transport EOF without
//     a close frame - the RFC's abnormal closure - becomes ErrConnDead.
//   - Calls on a nil, closed or failed-out connection return EBADF (9),
//     the net/tls defensive value.
//   - Close reports the fate of the graceful part: 0 when the peer echoed
//     the close frame, the wait's error (eagain/etimedout) when it did not
//     answer within the connection deadline - the underlying transport is
//     ALWAYS released, whatever the report (tls.Close's pattern).
//
// # Ownership
//
// HandshakeURL owns the transport from the moment it is called: on any
// failure the transport is closed before it returns (a failed handshake
// leaks nothing), and after success the WSConn owns it - WSConn.Close
// closes the underlying transport exactly once, and every later call on
// the WSConn is a defensive EBADF.
package ws

// WebSocket opcodes (RFC 6455 section 5.2).
const (
	OpCont   = 0x0 // continuation frame of a fragmented message
	OpText   = 0x1 // UTF-8 data frame
	OpBinary = 0x2 // binary data frame
	OpClose  = 0x8 // closing handshake
	OpPing   = 0x9 // keepalive probe (answered automatically)
	OpPong   = 0xA // keepalive answer (reported via PongSeen)
)

// WS-layer error codes, the 0x7D00 segment. See ErrName and the package
// doc's error-codes section.
const (
	// ErrHandshake: the peer's response was not a usable HTTP/1.x upgrade
	// response - malformed status line, a header bigger than the 512-byte
	// response window, garbage after the header, or a missing required
	// header. (A 101 with wrong Upgrade/Connection headers is ErrNotWS.)
	ErrHandshake = 0x7D01
	// ErrNotWS: the response is HTTP but not a WebSocket upgrade - the
	// status is not "101", or Upgrade/Connection do not carry the required
	// tokens (websocket / upgrade). The classic shape when the far end is
	// not a WebSocket server (a plain echo, a proxy, an error page).
	ErrNotWS = 0x7D02
	// ErrAcceptKey: Sec-WebSocket-Accept did not equal
	// base64(SHA-1(key + GUID)) for the key we sent - either the peer is
	// broken or something in between answered.
	ErrAcceptKey = 0x7D03
	// ErrTooBig: a data frame did not fit the caller's ReadMessage buffer.
	// The frame was drained, so the connection stays usable; the message
	// content is lost.
	ErrTooBig = 0x7D04
	// ErrProtocol: the peer violated the framing rules - a masked
	// server-to-client frame, RSV bits set (an extension we did not
	// negotiate), a fragmented or oversize control frame, an unknown
	// opcode. A violation means the byte stream is misaligned, so the
	// session is dead on the spot: the transport is released before the
	// code is reported (every later call is EBADF). No close frame is sent
	// on the way out - a deliberate deviation from the RFC 6455 section
	// 7.1.7 SHOULD to answer a protocol error with a 1002 close: a
	// misaligned stream cannot promise a well-formed closing exchange.
	ErrProtocol = 0x7D05
	// ErrPeerClosed: the orderly end of a session, not a failure - the
	// peer sent a close frame, we echoed it, the transport is closed.
	// CloseCode carries the peer's code (1000 normal, 1001 going away,
	// ...). Close() reports 0 for this same event when WE initiated.
	ErrPeerClosed = 0x7D06
	// ErrConnDead: the underlying transport closed (or failed) without a
	// WebSocket close frame - the RFC's abnormal closure. The transport is
	// already gone; just stop using the WSConn.
	ErrConnDead = 0x7D07
	// ErrNoDeadline: the transport has no SetDeadline method, so the
	// bounded-wait model cannot be honored - HandshakeURL refuses to run
	// unbounded (its timeoutMs cannot be enforced) and WSConn.SetDeadline
	// has nothing to forward to.
	ErrNoDeadline = 0x7D08
)

// lwip errno values this package acts on, same values the net package pins
// (picolibc numbering; net.go lists the Linux divergences). They are
// redeclared because the ws package imports nothing - not even pkg/esp/net.
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
// the net.Conn contract of the pkg/esp family: n bytes read with err == 0,
// n == 0 with err == 0 meaning the stream closed cleanly, n == 0 with
// err == eagain meaning "nothing in one quantum" when no deadline is set,
// n == 0 with err == etimedout when a deadline passed. Write writes the
// whole buffer and returns its length, or a negative code.
type Conn interface {
	Read(p []byte) (n int32, err int32)
	Write(p []byte) int32
	Close() int32
}

// Deadliner is the optional transport deadline interface: SetDeadline(ms
// int32) int32 with net's semantics (relative ms from now, 0 clears, < 0
// einval). HandshakeURL requires it (the handshake budget must be
// enforceable); *net.TCPConn and *tls.TLSConn both satisfy it.
type Deadliner interface {
	SetDeadline(ms int32) int32
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
func HandshakeURL(c Conn, host, path string, timeoutMs int32) (*WSConn, int32) {
	if c == nil {
		return nil, ebadf
	}
	if timeoutMs <= 0 {
		return nil, einval
	}
	dl, ok := c.(Deadliner)
	if !ok || dl == nil {
		return nil, ErrNoDeadline
	}
	if e := dl.SetDeadline(timeoutMs); e != 0 {
		return nil, e
	}
	fail := func(code int32) (*WSConn, int32) {
		dl.SetDeadline(0)
		c.Close()
		return nil, code
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
	total := int32(0)
	for total < int32(len(rb)) {
		r := c.Write(rb[total:])
		if r < 0 {
			return fail(r)
		}
		if r == 0 {
			return fail(-eagain)
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
		m, e := c.Read(resp[n:])
		if m > 0 {
			n += m
			if headerEnd(resp[:n], &scanned) {
				break
			}
			if n == int32(len(resp)) {
				return fail(ErrHandshake)
			}
			continue
		}
		if e == 0 {
			return fail(ErrConnDead)
		}
		return fail(e)
	}
	if n != scanned { // bytes after the header: server spoke first
		return fail(ErrHandshake)
	}
	if e := parseResponse(resp[:scanned], Base64StdEncode(key[:])); e != 0 {
		return fail(e)
	}
	dl.SetDeadline(0)
	return &WSConn{c: c, dl: dl, inHdr: true, rneed: 2}, 0
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
func parseResponse(resp []byte, keyB64 string) int32 {
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
	return 0
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
//	n > 0, err == 0            n payload bytes of a data frame (a frame
//	                           bigger than len(p) is drained and reported
//	                           as ErrTooBig - the connection stays usable)
//	n == 0, err == 0           a pong control frame was consumed
//	                           (PongSeen is now true) or a zero-length
//	                           data frame arrived - call again
//	n == 0, err == ErrPeerClosed
//	                           the peer completed the closing handshake;
//	                           the transport is closed, CloseCode has the
//	                           peer's code. The orderly session end, not
//	                           a failure.
//	n == 0, err == eagain      poll mode (no deadline): nothing within one
//	                           quantum; retry with the SAME buffer - the
//	                           call resumes mid-header or mid-payload
//	n == 0, err == etimedout   the connection deadline passed
//	otherwise                  ErrTooBig/ErrProtocol (see the constants),
//	                           ErrConnDead (transport closed without a
//	                           close frame), or the raw transport error
//	                           (lwip errno, or the negative mbedtls code
//	                           of a wss record failure)
//
// Pings are answered automatically inside the call; a pong is never
// delivered as data. ErrProtocol leaves the session closed (the framing is
// misaligned; there is no recovery).
func (w *WSConn) ReadMessage(p []byte) (int32, int32) {
	if w == nil || w.closed {
		return 0, ebadf
	}
	for {
		if w.inHdr {
			if e := w.readHeader(); e != 0 {
				return 0, e
			}
			switch w.fop {
			case OpText, OpBinary, OpCont:
				w.lastOp = w.fop
				if w.flen == 0 { // zero-length data frame
					w.resetHeader()
					return 0, 0
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
				if e := w.controlEvent(); e != 0 {
					return 0, e
				}
				if w.fop == OpPong {
					return 0, 0 // the pong event the caller pumps for
				}
				continue // ping answered automatically; keep reading
			}
			// data frame complete
			n, drained := w.flen, w.flen > int32(len(p))
			w.resetHeader()
			if drained {
				return 0, ErrTooBig // frame drained; the connection stays usable
			}
			return n, 0
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
		n, e := w.c.Read(dst)
		if n > 0 {
			w.fgot += n
			// Short reads are the transport's normal shape (TCP delivers
			// what the segment carried); keep accumulating the frame.
			continue
		}
		if e == 0 {
			return 0, ErrConnDead // transport closed mid-frame
		}
		return 0, e // eagain / etimedout / transport error, progress kept
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
func (w *WSConn) controlEvent() int32 {
	switch w.fop {
	case OpPing:
		payload := w.ctrl[:w.flen]
		w.resetHeader()
		if e := w.writeFrame(OpPong, payload); e < 0 {
			return e
		}
	case OpPong:
		w.pongSeen = true
		w.resetHeader()
		return 0 // the one control event reported to the caller
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
	return 0
}

// readHeader accumulates and decodes the next frame header across poll-mode
// retries. Server-to-client frames must not be masked (a masked one is
// ErrProtocol), RSV bits must be clear (no extensions negotiated), control
// frames must be unfragmented and at most 125 bytes.
func (w *WSConn) readHeader() int32 {
	for w.rhave < w.rneed {
		n, e := w.c.Read(w.rbuf[w.rhave:w.rneed])
		if n > 0 {
			w.rhave += n
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
		if e == 0 {
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
	return 0
}

// fatal marks the session dead and releases the transport: a framing
// violation (or an impossible length) means the byte stream is misaligned
// and no further frame can be decoded - there is no recovery but Close.
// No close frame is sent (the RFC 6455 section 7.1.7 SHOULD to answer
// with 1002 is deliberately not followed): a misaligned stream cannot
// promise a well-formed closing exchange.
func (w *WSConn) fatal(code int32) int32 {
	w.closed = true
	w.c.Close()
	return code
}

// WriteMessage sends one data frame with the given opcode (OpText or
// OpBinary; anything else is -einval - control frames have their own
// methods). The frame is built masked (the RFC requires client masking)
// and pushed whole; like the transports, it returns len(p) on success and
// a negative code on failure:
//
//	-len(p) shaped values      never: with a deadline set the transport
//	                           hands over the whole frame or fails, and
//	                           without one a quantum-long stall fails with
//	                           -eagain leaving a PARTIAL frame on the wire
//	-eagain                    no deadline set and the transport stalled
//	                           for one quantum: the connection must be
//	                           closed (a partial frame is already sent)
//	-116 (-ETIMEDOUT)          the deadline passed; nothing went out
//	other negative             transport failure (negated lwip errno, or
//	                           the raw negative mbedtls code over wss)
func (w *WSConn) WriteMessage(p []byte, opcode uint8) int32 {
	if w == nil || w.closed {
		return -ebadf
	}
	if opcode != OpText && opcode != OpBinary {
		return -einval
	}
	return w.writeFrame(opcode, p)
}

// writeFrame builds and sends one masked client frame (any opcode). It
// returns the payload length on success - 0 for empty payloads, which is
// what Ping() and the close handshake surface - or a negative code.
func (w *WSConn) writeFrame(opcode uint8, payload []byte) int32 {
	n := int32(len(payload))
	hdr := int32(2)
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
		buf[int(hdr)+4+i] ^= mask[i&3]
	}
	total := int32(0)
	for total < int32(len(buf)) {
		r := w.c.Write(buf[total:])
		if r < 0 {
			return r
		}
		if r == 0 {
			return -eagain
		}
		total += r
	}
	return n
}

// Ping sends an empty ping frame and returns 0, or a negative code on
// failure (same contract as WriteMessage). The peer's pong is reported by
// the next ReadMessage as a (0, 0) event with PongSeen() true - pump
// ReadMessage until PongSeen() to wait for it.
func (w *WSConn) Ping() int32 {
	if w == nil || w.closed {
		return -ebadf
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
//	0            graceful: the peer echoed the close frame (CloseCode now
//	             carries it)
//	eagain       no deadline and the peer stayed silent for one quantum
//	etimedout    the deadline passed without the peer's close frame
//	negative     the close frame itself could not be sent
//
// A second Close, or any call on a closed WSConn, is EBADF (9).
func (w *WSConn) Close(code uint16) int32 {
	if w == nil || w.closed {
		return ebadf
	}
	var pl [2]byte
	pl[0] = byte(code >> 8)
	pl[1] = byte(code)
	e := w.writeFrame(OpClose, pl[:])
	w.closeSent = true
	if e < 0 { // the close frame never went out: just release the transport
		w.closed = true
		w.c.Close()
		return e
	}
	// Wait for the peer's close frame. Data frames that arrive on the way
	// are drained (ReadMessage(nil) always takes the drain path); pongs
	// and pings are consumed as usual.
	var report int32 = eagain
	for {
		_, re := w.ReadMessage(nil)
		if re == ErrPeerClosed {
			return 0
		}
		if re != 0 {
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
func (w *WSConn) SetDeadline(ms int32) int32 {
	if w == nil || w.closed {
		return ebadf
	}
	if ms < 0 {
		return einval
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
	case ErrHandshake:
		name = "WS_ERR_HANDSHAKE"
	case ErrNotWS:
		name = "WS_ERR_NOT_WEBSOCKET"
	case ErrAcceptKey:
		name = "WS_ERR_ACCEPT_KEY"
	case ErrTooBig:
		name = "WS_ERR_FRAME_TOO_BIG"
	case ErrProtocol:
		name = "WS_ERR_PROTOCOL"
	case ErrPeerClosed:
		name = "WS_ERR_PEER_CLOSED"
	case ErrConnDead:
		name = "WS_ERR_CONN_DEAD"
	case ErrNoDeadline:
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
