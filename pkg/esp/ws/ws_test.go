package ws

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/xgo-dev/llgo/pkg/esp/esperr"
	"github.com/xgo-dev/llgo/pkg/esp/net"
	"github.com/xgo-dev/llgo/pkg/esp/tls"
)

// The real transports must keep satisfying the ws interfaces structurally
// (checked hosted, where both transport packages compile against their
// stubs; the target build of this package itself stays zero-import).
var (
	_ Conn      = (*net.TCPConn)(nil)
	_ Deadliner = (*net.TCPConn)(nil)
	_ Conn      = (*tls.TLSConn)(nil)
	_ Deadliner = (*tls.TLSConn)(nil)
)

// The tests run hosted (go test): they exercise the framing state machine,
// the handshake parser and the closing handshake against a hand-rolled
// RFC 6455 server over an in-memory transport that mimics the pkg/esp
// contracts ((int, error) io-shaped reads/writes, ErrWouldBlock poll reads
// without a deadline, ErrTimeout with one, all-or-nothing writes).

// mempipe is an in-memory ws.Conn/Deadliner: everything written to one end
// becomes readable at the other. Without a deadline, a Read with no data
// returns (0, eagain) after a 2 ms spin (poll mode); with one, it returns
// (0, etimedout) once the deadline passes. Writes are all-or-nothing.
// chunkReads makes every Read deliver at most one byte, to stress the
// header/payload accumulation loops. maxWrite caps every Write at that
// many bytes (partial writes, never zero); zeroOnce makes the first Write
// accept nothing and report 0 - the transport stalled for a whole quantum
// (the writeFrame / handshake write loops' r == 0 branches).
type mempipe struct {
	mu         *sync.Mutex // one mutex for both ends: all state is pair-wide
	rbuf       []byte      // bytes written by the peer, not yet read here
	peer       *mempipe
	closed     bool
	deadlineMs int32
	chunkReads bool
	maxWrite   int
	zeroOnce   bool
}

func newPipePair(chunkReads bool) (*mempipe, *mempipe) {
	mu := &sync.Mutex{}
	a := &mempipe{mu: mu, chunkReads: chunkReads}
	b := &mempipe{mu: mu, chunkReads: chunkReads}
	a.peer, b.peer = b, a
	return a, b
}

func (m *mempipe) Read(p []byte) (int, error) {
	if m == nil || len(p) == 0 {
		return 0, nil
	}
	m.mu.Lock()
	dms := m.deadlineMs
	m.mu.Unlock()
	dl := time.Time{}
	if dms > 0 {
		dl = time.Now().Add(time.Duration(dms) * time.Millisecond)
	}
	spin := time.Now().Add(2 * time.Millisecond)
	for {
		m.mu.Lock()
		if len(m.rbuf) > 0 {
			n := len(p)
			if m.chunkReads && n > 1 {
				n = 1
			}
			if n > len(m.rbuf) {
				n = len(m.rbuf)
			}
			copy(p, m.rbuf[:n])
			m.rbuf = m.rbuf[n:]
			m.mu.Unlock()
			return n, nil
		}
		m.mu.Unlock()
		if m.closed && m.peer.closed {
			return 0, ErrEOF // orderly stream close
		}
		if dms > 0 {
			if !time.Now().Before(dl) {
				return 0, ErrTimeout
			}
			time.Sleep(200 * time.Microsecond)
			continue
		}
		if time.Now().After(spin) {
			return 0, ErrWouldBlock
		}
		time.Sleep(100 * time.Microsecond)
	}
}

func (m *mempipe) Write(p []byte) (int, error) {
	if m == nil {
		return 0, ErrClosed
	}
	if len(p) == 0 {
		return 0, nil
	}
	if m.zeroOnce {
		m.mu.Lock()
		m.zeroOnce = false
		m.mu.Unlock()
		return 0, nil // one quantum-long stall: nothing accepted
	}
	n := len(p)
	if m.maxWrite > 0 && n > m.maxWrite {
		n = m.maxWrite // partial write: only maxWrite bytes accepted
	}
	m.mu.Lock()
	m.peer.rbuf = append(m.peer.rbuf, p[:n]...)
	m.mu.Unlock()
	return n, nil
}

func (m *mempipe) Close() error {
	if m == nil {
		return ErrClosed
	}
	m.mu.Lock()
	already := m.closed
	m.closed = true
	m.mu.Unlock()
	if already {
		return ErrClosed
	}
	return nil
}

func (m *mempipe) SetDeadline(ms int32) error {
	if m == nil {
		return ErrClosed
	}
	if ms < 0 {
		return errOf(einval)
	}
	m.mu.Lock()
	m.deadlineMs = ms
	m.mu.Unlock()
	return nil
}

// ---- RFC 6455 server side (test peer) ----

type testServer struct {
	t        *testing.T
	conn     *mempipe // the server's own end (reads what the client writes)
	client   *mempipe // the client's end: HandshakeURL runs on this
	key      string   // Sec-WebSocket-Key from the request
	path     string
	upgraded bool

	fmu          sync.Mutex // guards the flags the test polls
	sawClose     bool
	sawCloseCode uint16
	sawCloseLen  int  // payload length of the client's close frame
	sawMask      bool // client frames must carry the mask bit
	gotPong      bool
	gotAutoPong  []byte
}

// pongSeen snapshots the pong flags for the test goroutine.
func (s *testServer) pongSeen() (bool, []byte) {
	s.fmu.Lock()
	defer s.fmu.Unlock()
	return s.gotPong, s.gotAutoPong
}

func (s *testServer) closeCode() uint16 {
	s.fmu.Lock()
	defer s.fmu.Unlock()
	return s.sawCloseCode
}

// closeSeen snapshots the close handshake flags for the test goroutine
// (seen, the client's close code, its payload length).
func (s *testServer) closeSeen() (bool, uint16, int) {
	s.fmu.Lock()
	defer s.fmu.Unlock()
	return s.sawClose, s.sawCloseCode, s.sawCloseLen
}

// peerClosed reports whether the client end of the pipe pair is gone -
// readExact uses it to stop waiting for frames that will never come once
// the client side closed (the negative tests all end this way; without
// this check the server goroutine would spin to its 5 s deadline and call
// t.Errorf after the test completed).
func (s *testServer) peerClosed() bool {
	s.conn.mu.Lock()
	defer s.conn.mu.Unlock()
	return s.conn.peer.closed
}

func startServer(t *testing.T, clientEnd *mempipe, denyUpgrade, wrongAccept, maskReply bool) *testServer {
	s := &testServer{t: t, conn: clientEnd.peer, client: clientEnd}
	go s.run(denyUpgrade, wrongAccept, maskReply)
	return s
}

func (s *testServer) run(denyUpgrade, wrongAccept, maskReply bool) {
	defer s.conn.Close()
	// Read the request header section (bounded, raw reads - no bufio, so
	// no second reader can swallow frame bytes after the upgrade).
	var req [1024]byte
	n, scanned := 0, 0
	for {
		m, rerr := s.conn.Read(req[n:])
		if m > 0 {
			n += m
			if idx := headerEndOf(req[:n], scanned); idx >= 0 {
				scanned = idx
				break
			}
			scanned = n - 3
			if scanned < 0 {
				scanned = 0
			}
			if n == len(req) {
				s.t.Error("server: request header too big")
				return
			}
			continue
		}
		if rerr != nil && rerr != ErrWouldBlock {
			s.t.Errorf("server: read request err %v", rerr)
			return
		}
		time.Sleep(200 * time.Microsecond)
	}
	for _, line := range splitLines(req[:scanned]) {
		if len(line) > 4 && string(line[:4]) == "GET " {
			rest := string(line[4:])
			for j := 0; j < len(rest); j++ {
				if rest[j] == ' ' {
					s.path = rest[:j]
					break
				}
			}
		}
		if len(line) > 18 && string(line[:18]) == "Sec-WebSocket-Key:" {
			v := line[18:]
			for len(v) > 0 && (v[0] == ' ' || v[0] == '\t') {
				v = v[1:]
			}
			s.key = string(v)
		}
	}
	if s.key == "" {
		s.t.Errorf("server: no Sec-WebSocket-Key in request %q", req[:scanned])
		return
	}
	if s.path != "/ws" {
		s.t.Errorf("server: request path %q, want /ws", s.path)
	}
	h := sha1.Sum([]byte(s.key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	accept := base64.StdEncoding.EncodeToString(h[:])
	if wrongAccept {
		accept = "AAAAAAAAAAAAAAAAAAAAAAAAAAA="
	}
	status := "101 Switching Protocols"
	if denyUpgrade {
		status = "200 OK"
	}
	resp := "HTTP/1.1 " + status + "\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + accept + "\r\n\r\n"
	if _, err := s.conn.Write([]byte(resp)); err != nil {
		s.t.Errorf("server write response: %v", err)
		return
	}
	if denyUpgrade || wrongAccept {
		// Not a usable upgrade response: the client will reject it (ErrNotWS
		// / ErrAcceptKey) and close. This server never enters the frame loop
		// - nothing else to serve.
		return
	}
	s.upgraded = true
	s.frameLoop(maskReply)
}

// headerEndOf mirrors the client's headerEnd scan: the index past CRLFCRLF.
func headerEndOf(p []byte, from int) int {
	for i := from; i+4 <= len(p); i++ {
		if p[i] == '\r' && p[i+1] == '\n' && p[i+2] == '\r' && p[i+3] == '\n' {
			return i + 4
		}
	}
	return -1
}

func splitLines(p []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i := 0; i+1 < len(p); i++ {
		if p[i] == '\r' && p[i+1] == '\n' {
			lines = append(lines, p[start:i])
			start = i + 2
			i++
		}
	}
	return lines
}

func (s *testServer) readExact(n int) []byte {
	buf := make([]byte, n)
	got := 0
	deadline := time.Now().Add(5 * time.Second)
	for got < n {
		if s.peerClosed() {
			return buf[:got] // client gone: no more frames are coming
		}
		if time.Now().After(deadline) {
			s.t.Errorf("server: timeout reading %d bytes (got %d)", n, got)
			return buf[:got]
		}
		nn, rerr := s.conn.Read(buf[got:])
		if nn > 0 {
			got += nn
			continue
		}
		if rerr != nil && rerr != ErrWouldBlock {
			s.t.Errorf("server: read err %v at %d/%d", rerr, got, n)
			return buf[:got]
		}
		time.Sleep(200 * time.Microsecond)
	}
	return buf
}

// readFrame reads one client frame, unmasks it, and checks the mask bit.
func (s *testServer) readFrame() (fin bool, op byte, payload []byte, ok bool) {
	h := s.readExact(2)
	if len(h) < 2 {
		return false, 0, nil, false
	}
	s.fmu.Lock()
	s.sawMask = h[1]&0x80 != 0
	s.fmu.Unlock()
	if !s.sawMask {
		s.t.Errorf("server: client frame without mask bit")
	}
	length := int(h[1] & 0x7f)
	switch length {
	case 126:
		ext := s.readExact(2)
		length = int(ext[0])<<8 | int(ext[1])
	case 127:
		ext := s.readExact(8)
		length = int(ext[7]) // small frames only in these tests
	}
	var mask [4]byte
	if s.sawMask {
		copy(mask[:], s.readExact(4))
	}
	payload = s.readExact(length)
	for i := range payload {
		payload[i] ^= mask[i&3]
	}
	return h[0]&0x80 != 0, h[0] & 0x0f, payload, true
}

func (s *testServer) writeFrame(op byte, payload []byte) {
	var frame []byte
	if len(payload) < 126 {
		frame = append([]byte{0x80 | op, byte(len(payload))}, payload...)
	} else {
		n := len(payload)
		frame = append([]byte{0x80 | op, 126, byte(n >> 8), byte(n)}, payload...)
	}
	if _, err := s.conn.Write(frame); err != nil {
		s.t.Errorf("server write frame: %v", err)
	}
}

// writeFrameMasked is the RFC-violating helper for the negative test: the
// server must never mask, this one does.
func (s *testServer) writeFrameMasked(op byte, payload []byte) {
	frame := []byte{0x80 | op, 0x80 | byte(len(payload)), 1, 2, 3, 4}
	for i := range payload {
		frame = append(frame, payload[i]^byte(1+(i%4)))
	}
	if _, err := s.conn.Write(frame); err != nil {
		s.t.Errorf("server write masked frame: %v", err)
	}
}

func (s *testServer) frameLoop(maskReply bool) {
	for {
		fin, op, payload, ok := s.readFrame()
		if !ok {
			return
		}
		switch op {
		case 0x1, 0x2:
			if maskReply {
				s.writeFrameMasked(op, payload) // protocol violation on purpose
				s.conn.Close()                  // done provoking: stop reading
				return
			}
			s.writeFrame(op, payload) // echo, unmasked, same opcode
			if !fin {
				s.t.Errorf("server: unexpected FIN=0 client frame")
			}
		case 0x9:
			s.writeFrame(0xA, payload) // ping -> pong
		case 0xA:
			s.fmu.Lock()
			s.gotPong = true
			s.gotAutoPong = payload
			s.fmu.Unlock()
		case 0x8:
			code := uint16(0)
			if len(payload) >= 2 {
				code = uint16(payload[0])<<8 | uint16(payload[1])
			}
			s.fmu.Lock()
			s.sawClose = true
			s.sawCloseCode = code
			s.sawCloseLen = len(payload)
			s.fmu.Unlock()
			s.writeFrame(0x8, payload) // echo the close
			return
		default:
			s.t.Errorf("server: unexpected opcode %#x", op)
			return
		}
	}
}

// ---- tests ----

func dial(t *testing.T, srv *testServer, timeoutMs int32) *WSConn {
	t.Helper()
	w, err := HandshakeURL(srv.client, "192.168.4.1:9000", "/ws", timeoutMs)
	if err != nil {
		t.Fatalf("HandshakeURL err %v", err)
	}
	if !srv.upgraded {
		t.Fatalf("server did not upgrade")
	}
	return w
}

func TestHandshakeAndEchoBinary(t *testing.T) {
	_, b := newPipePair(false)
	srv := startServer(t, b, false, false, false)
	w := dial(t, srv, 5000)

	msg := make([]byte, 300) // 126-length encoding on the wire
	for i := range msg {
		msg[i] = byte(i * 7)
	}
	if n, err := w.WriteMessage(msg, OpBinary); err != nil || n != len(msg) {
		t.Fatalf("WriteMessage n=%d err=%v want %d", n, err, len(msg))
	}
	got := make([]byte, len(msg))
	total := 0
	for total < len(msg) {
		n, rerr := w.ReadMessage(got[total:])
		if n > 0 {
			total += n
			continue
		}
		t.Fatalf("ReadMessage n=%d err=%v", n, rerr)
	}
	for i := range msg {
		if got[i] != msg[i] {
			t.Fatalf("echo mismatch at %d: %d != %d", i, got[i], msg[i])
		}
	}
	if w.LastOpcode() != OpBinary {
		t.Fatalf("LastOpcode=%d want OpBinary", w.LastOpcode())
	}
	w.SetDeadline(2000)
	if err := w.Close(1000); err != nil {
		t.Fatalf("Close err %v", err)
	}
	if w.CloseCode() != 1000 {
		t.Fatalf("CloseCode=%d want 1000", w.CloseCode())
	}
	if cc := srv.closeCode(); cc != 1000 {
		t.Fatalf("server saw close code %d", cc)
	}
	if err := w.Close(1000); err != ErrClosed {
		t.Fatalf("second Close = %v, want ErrClosed", err)
	}
}

func TestEchoText(t *testing.T) {
	_, b := newPipePair(false)
	srv := startServer(t, b, false, false, false)
	w := dial(t, srv, 5000)

	msg := []byte("hello over llgo-ws")
	if n, err := w.WriteMessage(msg, OpText); err != nil || n != len(msg) {
		t.Fatalf("WriteMessage n=%d err=%v", n, err)
	}
	got := make([]byte, len(msg))
	n, rerr := w.ReadMessage(got)
	if n != len(msg) || rerr != nil {
		t.Fatalf("ReadMessage n=%d err=%v", n, rerr)
	}
	if string(got) != string(msg) {
		t.Fatalf("echo mismatch: %q", got)
	}
	if w.LastOpcode() != OpText {
		t.Fatalf("LastOpcode=%d want OpText", w.LastOpcode())
	}
	w.SetDeadline(2000)
	if err := w.Close(1000); err != nil {
		t.Fatalf("Close err %v", err)
	}
}

func TestPingPong(t *testing.T) {
	_, b := newPipePair(false)
	srv := startServer(t, b, false, false, false)
	w := dial(t, srv, 5000)

	if err := w.Ping(); err != nil {
		t.Fatalf("Ping err %v", err)
	}
	w.SetDeadline(2000)
	sawPong := false
	var scratch [8]byte
	for i := 0; i < 100 && !sawPong; i++ {
		if _, rerr := w.ReadMessage(scratch[:]); rerr != nil && rerr != ErrTimeout {
			t.Fatalf("ReadMessage err %v", rerr)
		}
		sawPong = w.PongSeen()
	}
	if !sawPong {
		t.Fatalf("pong never reported")
	}
	w.SetDeadline(2000)
	if err := w.Close(1000); err != nil {
		t.Fatalf("Close err %v", err)
	}
}

// TestServerPingAutoAnswer: the server pings; the client must answer with
// an automatic pong and keep delivering data frames afterwards.
func TestServerPingAutoAnswer(t *testing.T) {
	_, b := newPipePair(false)
	srv := startServer(t, b, false, false, false)
	w := dial(t, srv, 5000)

	srv.writeFrame(0x9, []byte("probe")) // server ping
	msg := []byte("after-ping")
	srv.writeFrame(0x2, msg)

	got := make([]byte, len(msg))
	total := 0
	for total < len(msg) {
		n, rerr := w.ReadMessage(got[total:])
		if n > 0 {
			total += n
			continue
		}
		t.Fatalf("ReadMessage n=%d err=%v", n, rerr)
	}
	if string(got) != string(msg) {
		t.Fatalf("post-ping echo mismatch: %q", got)
	}
	deadline := time.Now().Add(2 * time.Second)
	var sawPong bool
	var autoPong []byte
	for {
		sawPong, autoPong = srv.pongSeen()
		if sawPong || !time.Now().Before(deadline) {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if !sawPong {
		t.Fatalf("server never saw the automatic pong")
	}
	if string(autoPong) != "probe" {
		t.Fatalf("auto-pong payload %q, want echoed ping payload", autoPong)
	}
	w.SetDeadline(2000)
	if err := w.Close(1000); err != nil {
		t.Fatalf("Close err %v", err)
	}
}

func TestHandshakeNotWS(t *testing.T) {
	_, b := newPipePair(false)
	srv := startServer(t, b, true, false, false) // replies "200 OK"
	if _, err := HandshakeURL(srv.client, "h", "/ws", 5000); !errors.Is(err, ErrNotWS) {
		t.Fatalf("err=%v, want ErrNotWS", err)
	}
}

func TestHandshakeBadAccept(t *testing.T) {
	_, b := newPipePair(false)
	srv := startServer(t, b, false, true, false) // wrong Sec-WebSocket-Accept
	if _, err := HandshakeURL(srv.client, "h", "/ws", 5000); !errors.Is(err, ErrAcceptKey) {
		t.Fatalf("err=%v, want ErrAcceptKey", err)
	}
}

func TestMaskingRequired(t *testing.T) {
	_, b := newPipePair(false)
	srv := startServer(t, b, false, false, true) // server masks its reply
	w := dial(t, srv, 5000)
	if n, err := w.WriteMessage([]byte("trigger"), OpText); err != nil || n != 7 {
		t.Fatalf("WriteMessage n=%d err=%v", n, err)
	}
	var buf [8]byte
	if _, rerr := w.ReadMessage(buf[:]); !errors.Is(rerr, ErrProtocol) {
		t.Fatalf("err=%v, want ErrProtocol", rerr)
	}
	// The protocol violation killed the session on the spot (framing is
	// misaligned - there is no recovery): the transport is already
	// released and every later call reports ErrClosed.
	if err := w.Close(1002); err != ErrClosed {
		t.Fatalf("Close after protocol error: %v, want ErrClosed", err)
	}
}

func TestFragmentedMessage(t *testing.T) {
	_, b := newPipePair(false)
	srv := startServer(t, b, false, false, false)
	w := dial(t, srv, 5000)

	// A fragmented text message, written raw: two fragments delivered as
	// two reads (per-frame delivery, FIN not reassembled - the documented
	// behavior).
	if _, err := srv.conn.Write([]byte{0x01, 0x03, 'f', 'o', 'o'}); err != nil {
		t.Fatalf("write fragment1: %v", err)
	}
	if _, err := srv.conn.Write([]byte{0x80, 0x03, 'b', 'a', 'r'}); err != nil {
		t.Fatalf("write fragment2: %v", err)
	}
	var part [3]byte
	n, rerr := w.ReadMessage(part[:])
	if n != 3 || rerr != nil || string(part[:]) != "foo" {
		t.Fatalf("fragment1 n=%d err=%v %q", n, rerr, part[:])
	}
	if w.LastOpcode() != OpText {
		t.Fatalf("fragment1 opcode %d", w.LastOpcode())
	}
	n, rerr = w.ReadMessage(part[:])
	if n != 3 || rerr != nil || string(part[:]) != "bar" {
		t.Fatalf("fragment2 n=%d err=%v %q", n, rerr, part[:])
	}
	if w.LastOpcode() != OpCont {
		t.Fatalf("fragment2 opcode %d, want OpCont", w.LastOpcode())
	}
	w.SetDeadline(2000)
	if err := w.Close(1000); err != nil {
		t.Fatalf("Close err %v", err)
	}
}

func TestFrameTooBigDrains(t *testing.T) {
	_, b := newPipePair(false)
	srv := startServer(t, b, false, false, false)
	w := dial(t, srv, 5000)

	big := make([]byte, 400)
	for i := range big {
		big[i] = byte(i + 1)
	}
	srv.writeFrame(0x2, big) // 400-byte frame vs a 64-byte buffer
	small := []byte("tiny")
	srv.writeFrame(0x2, small)

	buf := make([]byte, 64)
	if _, rerr := w.ReadMessage(buf); !errors.Is(rerr, ErrTooBig) {
		t.Fatalf("err=%v, want ErrTooBig", rerr)
	}
	n, rerr := w.ReadMessage(buf)
	if rerr != nil || string(buf[:n]) != "tiny" {
		t.Fatalf("post-drain read n=%d err=%v %q", n, rerr, buf[:n])
	}
	w.SetDeadline(2000)
	if err := w.Close(1000); err != nil {
		t.Fatalf("Close err %v", err)
	}
}

func TestPollModeEAGAIN(t *testing.T) {
	_, b := newPipePair(false)
	srv := startServer(t, b, false, false, false)
	w := dial(t, srv, 5000)

	// No deadline: a read with nothing queued must come back as
	// ErrWouldBlock promptly, and a later read must still frame correctly.
	var buf [8]byte
	if _, rerr := w.ReadMessage(buf[:]); !errors.Is(rerr, ErrWouldBlock) {
		t.Fatalf("poll read err=%v, want ErrWouldBlock", rerr)
	}
	srv.writeFrame(0x1, []byte("lazy"))
	n, rerr := w.ReadMessage(buf[:])
	if rerr != nil || string(buf[:n]) != "lazy" {
		t.Fatalf("post-poll read n=%d err=%v %q", n, rerr, buf[:n])
	}
	w.SetDeadline(2000)
	if err := w.Close(1000); err != nil {
		t.Fatalf("Close err %v", err)
	}
}

func TestByteAtATimeTransport(t *testing.T) {
	_, b := newPipePair(true) // every transport Read delivers 1 byte
	srv := startServer(t, b, false, false, false)
	w := dial(t, srv, 5000)

	msg := make([]byte, 200)
	for i := range msg {
		msg[i] = byte(i ^ 0x5A)
	}
	if n, err := w.WriteMessage(msg, OpBinary); err != nil || n != len(msg) {
		t.Fatalf("WriteMessage n=%d err=%v", n, err)
	}
	got := make([]byte, len(msg))
	total := 0
	for total < len(msg) {
		n, rerr := w.ReadMessage(got[total:])
		if n > 0 {
			total += n
			continue
		}
		t.Fatalf("ReadMessage n=%d err=%v at %d", n, rerr, total)
	}
	for i := range msg {
		if got[i] != msg[i] {
			t.Fatalf("echo mismatch at %d", i)
		}
	}
	w.SetDeadline(5000)
	if err := w.Close(1000); err != nil {
		t.Fatalf("Close err %v", err)
	}
}

func TestHandshakeArgumentChecks(t *testing.T) {
	a, _ := newPipePair(false)
	if _, err := HandshakeURL(a, "h", "/ws", 0); esperr.Code(err) != einval {
		t.Fatalf("timeoutMs=0: %v, want EINVAL", err)
	}
	if _, err := HandshakeURL(nil, "h", "/ws", 1000); err != ErrClosed {
		t.Fatalf("nil conn: %v, want ErrClosed", err)
	}
	// A transport with only Read/Write/Close cannot bound the handshake.
	type connOnly struct {
		Conn
	}
	if _, err := HandshakeURL(connOnly{a}, "h", "/ws", 1000); !errors.Is(err, ErrNoDeadline) {
		t.Fatalf("no-deadline conn: %v, want ErrNoDeadline", err)
	}
}

func TestPeerInitiatedClose(t *testing.T) {
	_, b := newPipePair(false)
	srv := startServer(t, b, false, false, false)
	w := dial(t, srv, 5000)

	// The server initiates the close handshake; ReadMessage must report
	// ErrPeerClosed with the code, and the transport must be released.
	srv.writeFrame(0x8, []byte{0x03, 0xE9}) // 1001 going away
	var buf [8]byte
	if _, rerr := w.ReadMessage(buf[:]); !errors.Is(rerr, ErrPeerClosed) {
		t.Fatalf("err=%v, want ErrPeerClosed", rerr)
	}
	if w.CloseCode() != 1001 {
		t.Fatalf("CloseCode=%d, want 1001", w.CloseCode())
	}
	if err := w.Close(1000); err != ErrClosed {
		t.Fatalf("Close after peer close = %v, want ErrClosed", err)
	}
}

// TestPeerCloseNoCode: a close frame without a status code must be echoed
// as an EMPTY close payload - a 0 code on the wire is invalid (RFC 6455
// section 7.4.1 reserves 0-999).
func TestPeerCloseNoCode(t *testing.T) {
	_, b := newPipePair(false)
	srv := startServer(t, b, false, false, false)
	w := dial(t, srv, 5000)

	srv.writeFrame(0x8, nil)
	var buf [8]byte
	if _, rerr := w.ReadMessage(buf[:]); !errors.Is(rerr, ErrPeerClosed) {
		t.Fatalf("err=%v, want ErrPeerClosed", rerr)
	}
	if w.CloseCode() != 0 {
		t.Fatalf("CloseCode=%d, want 0 (the peer sent none)", w.CloseCode())
	}
	var seen bool
	var code uint16
	var n int
	deadline := time.Now().Add(2 * time.Second)
	for !seen && time.Now().Before(deadline) {
		seen, code, n = srv.closeSeen()
		if !seen {
			time.Sleep(time.Millisecond)
		}
	}
	if !seen {
		t.Fatalf("server never saw the close echo")
	}
	if code != 0 || n != 0 {
		t.Fatalf("server saw close code %d len %d, want an empty echo", code, n)
	}
}

// TestShortWriteTransport: the client's end accepts at most 5 bytes per
// Write, so every write loop - the handshake request and every frame,
// including the close frame - must loop over partial writes (never r == 0).
func TestShortWriteTransport(t *testing.T) {
	_, b := newPipePair(false)
	srv := startServer(t, b, false, false, false)
	srv.client.maxWrite = 5
	w := dial(t, srv, 5000)

	msg := make([]byte, 200)
	for i := range msg {
		msg[i] = byte(i ^ 0xA5)
	}
	if n, err := w.WriteMessage(msg, OpBinary); err != nil || n != len(msg) {
		t.Fatalf("WriteMessage n=%d err=%v want %d", n, err, len(msg))
	}
	got := make([]byte, len(msg))
	total := 0
	for total < len(msg) {
		n, rerr := w.ReadMessage(got[total:])
		if n > 0 {
			total += n
			continue
		}
		t.Fatalf("ReadMessage n=%d err=%v at %d", n, rerr, total)
	}
	for i := range msg {
		if got[i] != msg[i] {
			t.Fatalf("echo mismatch at %d", i)
		}
	}
	w.SetDeadline(5000)
	if err := w.Close(1000); err != nil {
		t.Fatalf("Close err %v", err)
	}
}

// TestWriteStall: the transport stalls for a whole quantum (Write returns
// 0) - WriteMessage must report -EAGAIN (a partial frame may be on the
// wire, the connection must then be closed) and the close handshake must
// still complete the release cleanly.
func TestWriteStall(t *testing.T) {
	_, b := newPipePair(false)
	srv := startServer(t, b, false, false, false)
	w := dial(t, srv, 5000)

	srv.client.zeroOnce = true
	if _, err := w.WriteMessage([]byte("stall"), OpText); !errors.Is(err, ErrWouldBlock) {
		t.Fatalf("WriteMessage = %v, want ErrWouldBlock", err)
	}
	// Nothing of the stalled frame went out (the stall was the frame's
	// first write), so the close frame arrives on a clean stream and the
	// closing handshake completes; the release is verified by the second
	// Close reporting ErrClosed.
	w.SetDeadline(5000)
	if err := w.Close(1000); err != nil {
		t.Fatalf("Close after stall = %v, want nil", err)
	}
	if err := w.Close(1000); err != ErrClosed {
		t.Fatalf("second Close = %v, want ErrClosed", err)
	}
}

// TestHandshakeWriteStall: the same stall during the handshake request's
// write loop - HandshakeURL must fail with -EAGAIN and release the
// transport itself (failed handshakes leak nothing).
func TestHandshakeWriteStall(t *testing.T) {
	a, _ := newPipePair(false)
	a.zeroOnce = true
	if _, err := HandshakeURL(a, "h", "/ws", 5000); !errors.Is(err, ErrWouldBlock) {
		t.Fatalf("HandshakeURL = %v, want ErrWouldBlock", err)
	}
	if err := a.Close(); err != ErrClosed {
		t.Fatalf("Close after failed handshake = %v, want ErrClosed", err)
	}
	if err := a.peer.Close(); err != nil { // the peer end was never ws's to close
		t.Fatalf("peer end Close = %v, want nil", err)
	}
}

func TestSha1AndBase64Vectors(t *testing.T) {
	// FIPS 180-1 / RFC 3174 known values.
	wantABC := [20]byte{
		0xa9, 0x99, 0x3e, 0x36, 0x47, 0x06, 0x81, 0x6a, 0xba, 0x3e,
		0x25, 0x71, 0x78, 0x50, 0xc2, 0x6c, 0x9c, 0xd0, 0xd8, 0x9d,
	}
	if got := SHA1Sum([]byte("abc")); got != wantABC {
		t.Fatalf("sha1(abc) = %x", got)
	}
	// SHA-1 of a 56-byte message exercises the two-block tail path (the
	// 0x80 and padding leave no room for the length). Its value is pinned
	// against the stdlib in TestAcceptKeyMatchesStdlib (size 56 in the
	// boundary list).
	// RFC 6455 section 1.3 example: the accept key for the sample key.
	if got := AcceptKey([]byte("dGhlIHNhbXBsZSBub25jZQ==")); got != "s3pPLMBiTxaQ9kYGzzhZRbK+xOo=" {
		t.Fatalf("AcceptKey(sample) = %q", got)
	}
	// RFC 4648 vectors.
	cases := []struct {
		in, out string
	}{
		{"", ""},
		{"f", "Zg=="},
		{"fo", "Zm8="},
		{"foo", "Zm9v"},
		{"foob", "Zm9vYg=="},
		{"fooba", "Zm9vYmE="},
		{"foobar", "Zm9vYmFy"},
	}
	for _, c := range cases {
		if got := Base64StdEncode([]byte(c.in)); got != c.out {
			t.Fatalf("encode(%q) = %q, want %q", c.in, got, c.out)
		}
		dec, derr := Base64StdDecode(c.out)
		if derr != nil || string(dec) != c.in {
			t.Fatalf("decode(%q) = %q, %v; want %q", c.out, dec, derr, c.in)
		}
	}
	if _, err := Base64StdDecode("Zm9vYg"); esperr.Code(err) != einval { // length % 4 != 0
		t.Fatalf("bad length: %v", err)
	}
	if _, err := Base64StdDecode("Zm9vY*=="); esperr.Code(err) != einval { // bad character
		t.Fatalf("bad char: %v", err)
	}
	if _, err := Base64StdDecode("Z=g="); esperr.Code(err) != einval { // misplaced padding
		t.Fatalf("misplaced pad: %v", err)
	}
}

func TestAcceptKeyMatchesStdlib(t *testing.T) {
	// Cross-check the in-package primitives against the real stdlib on
	// random keys: the handshake verification must agree with what a real
	// server computes (this also exercises SHA-1 over ~60-byte inputs, the
	// exact handshake length, plus multi-block inputs via the random
	// pipeline below).
	for i := 0; i < 32; i++ {
		key := make([]byte, 16)
		if _, err := rand.Read(key); err != nil {
			t.Fatal(err)
		}
		enc := base64.StdEncoding.EncodeToString(key)
		h := sha1.Sum([]byte(enc + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
		want := base64.StdEncoding.EncodeToString(h[:])
		if got := AcceptKey([]byte(enc)); got != want {
			t.Fatalf("AcceptKey(%q) = %q, want %q", enc, got, want)
		}
	}
	// Larger inputs: the 56-byte two-block tail edge, 64-byte aligned,
	// the 55/56/119/120 padding boundaries, and a long message.
	for _, size := range []int{55, 56, 57, 64, 119, 120, 1000} {
		msg := make([]byte, size)
		if _, err := rand.Read(msg); err != nil {
			t.Fatal(err)
		}
		want := sha1.Sum(msg)
		if got := SHA1Sum(msg); got != want {
			t.Fatalf("sha1(%d bytes) mismatch", size)
		}
	}
}

func TestErrName(t *testing.T) {
	cases := []struct {
		code int32
		want string
	}{
		{0, "OK"},
		{wsErrNotWS, "WS_ERR_NOT_WEBSOCKET"},
		{wsErrPeerClosed, "WS_ERR_PEER_CLOSED"},
		{eagain, "EAGAIN"},
		{etimedout, "ETIMEDOUT"},
		{-eagain, "-EAGAIN"},
		{-etimedout, "-ETIMEDOUT"},
		{-0x7780, "-errno 30592"}, // mbedtls codes fall through to the value form
	}
	for _, c := range cases {
		if got := ErrName(c.code); got != c.want {
			t.Errorf("ErrName(%d) = %q, want %q", c.code, got, c.want)
		}
	}
}

// TestSentinelErrors pins the esperr error-model contract: the sentinels
// match by pipeline code through errors.Is, and esperr.Code extracts the
// code a value carries (including from a sibling package's sentinel).
func TestSentinelErrors(t *testing.T) {
	// A plain esperr.Error built from the raw errno satisfies errors.Is
	// against the sentinel that carries the same code.
	if !errors.Is(&esperr.Error{Code: eagain, Name: "EAGAIN"}, ErrWouldBlock) {
		t.Errorf("errors.Is(raw EAGAIN, ErrWouldBlock) = false")
	}
	if !errors.Is(ErrTimeout, esperr.ErrTimeout) {
		t.Errorf("errors.Is(ErrTimeout, esperr.ErrTimeout) = false")
	}
	if errors.Is(ErrWouldBlock, ErrTimeout) {
		t.Errorf("ErrWouldBlock matched ErrTimeout")
	}
	if esperr.Code(ErrClosed) != ebadf {
		t.Errorf("esperr.Code(ErrClosed) = %d, want EBADF", esperr.Code(ErrClosed))
	}
	// The tls package's handshake sentinel is a plain esperr.Error: errors.Is
	// works across packages without any wiring, and its code is the internal
	// 0x7E02 segment value.
	if !errors.Is(tls.ErrCertVerifyFailed, &esperr.Error{Code: tls.ErrCertVerifyFailed.Code}) {
		t.Errorf("errors.Is(tls.ErrCertVerifyFailed, same-code Error) = false")
	}
	if tls.ErrCertVerifyFailed.Error() != "tls: certificate verify failed (code 32258)" {
		t.Errorf("tls.ErrCertVerifyFailed.Error() = %q", tls.ErrCertVerifyFailed.Error())
	}
	// ErrName still names the sentinel's code (the diagnostic path).
	if got := ErrName(wsErrProtocol); got != "WS_ERR_PROTOCOL" {
		t.Errorf("ErrName(wsErrProtocol) = %q", got)
	}
}

func TestSeedStableAndSettable(t *testing.T) {
	var a, b, c, d, e [4]byte
	Seed(0x1234)
	prngFill(a[:]) // draw 1 of seed 0x1234
	Seed(0)        // must be ignored: the state stays mid-sequence
	prngFill(b[:]) // draw 2
	Seed(0x1234)   // a real reseed restarts the sequence
	prngFill(c[:])
	prngFill(d[:])
	if c != a || d != b {
		t.Fatalf("reseed did not restart the sequence")
	}
	// Seed(0) between two draws must not restart anything either.
	Seed(0x1234)
	prngFill(a[:])
	Seed(0)
	prngFill(e[:])
	if e != b {
		t.Fatalf("Seed(0) changed the state")
	}
}
