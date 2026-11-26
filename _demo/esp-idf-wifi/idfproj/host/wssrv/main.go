// Command wssrv is the host-side acceptance peer for the esp-emu network,
// TLS and WebSocket probes (_demo/esp-idf-wifi). One process covers every
// peer the demo firmware talks to:
//
//	127.0.0.1:9000   plain TCP: raw echo OR ws:// (sniffed per connection)
//	127.0.0.1:9443   TLS (crypto/tls, gencert.sh certs): TLS echo OR wss://
//
// The emulator's user-mode NAT aliases its gateway address (192.168.4.1 in
// the default subnet) to the host and forwards guest connections to the
// same port on the host's loopback, so the firmware's dials land here:
// netProbe    -> 192.168.4.1:9000 raw echo
// tlsProbe    -> 192.168.4.1:9443 TLS echo (permissive + strict)
// wsProbe     -> ws://192.168.4.1:9000/ws and wss://192.168.4.1:9443/ws
//
// A connection's first four bytes decide the mode: "GET " begins the
// RFC 6455 opening handshake (handled by this file's hand-rolled server
// frame loop, no external dependencies); anything else is echoed raw, byte
// for byte, until the peer closes - which is what netProbe's and
// tlsProbe's 256-byte pattern echo need. Inside TLS the same sniff runs on
// the decrypted stream, after crypto/tls has answered the ClientHello, so
// one listener serves the TLS echo and wss:// alike. This keeps the
// acceptance run to a single host peer; the dedicated echosrv/tlssrv of
// the net/tls-only phases remain available unchanged.
//
// The WebSocket side is a baseline RFC 6455 server: it verifies the
// required request headers (Upgrade: websocket, Connection: Upgrade, a
// Sec-WebSocket-Key), answers with Sec-WebSocket-Accept =
// base64(sha1(key+GUID)), echoes data frames back unmasked with the same
// opcode, answers pings with pongs, and completes the closing handshake by
// echoing the client's close frame. Client frames MUST be masked (the RFC
// requirement); a violation aborts the session. No subprotocols, no
// permessage-deflate.
//
// Usage:
//
//	go run ./idfproj/host/wssrv [-ws 127.0.0.1:9000] [-wss 127.0.0.1:9443]
//
// Every connection is logged: the mode (raw/ws/wss), the handshake path,
// the echoed byte count, and how the session ended. For ws/wss the client's
// close frame code is logged - that line is the host-side evidence for the
// closing handshake (the analog of tlssrv's "peer sent TLS close_notify");
// for TLS echoes the trailing read reports the close_notify (io.EOF) vs an
// abrupt disconnect, same as tlssrv.
package main

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

func main() {
	wsAddr := flag.String("ws", "127.0.0.1:9000", "plain listener (raw echo or ws://)")
	wssAddr := flag.String("wss", "127.0.0.1:9443", "TLS listener (TLS echo or wss://)")
	flag.Parse()

	ln, err := net.Listen("tcp", *wsAddr)
	if err != nil {
		log.Fatalf("wssrv: listen %s: %v", *wsAddr, err)
	}
	go accept(ln, false)

	_, thisFile, _, _ := runtime.Caller(0)
	certDir := filepath.Join(filepath.Dir(thisFile), "..", "tlssrv", "certs")
	cert, err := tls.LoadX509KeyPair(
		filepath.Join(certDir, "tls.crt"), filepath.Join(certDir, "tls.key"))
	if err != nil {
		log.Fatalf("wssrv: load certificate: %v (run idfproj/host/gencert.sh first)", err)
	}
	tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
	tln, err := tls.Listen("tcp", *wssAddr, tlsCfg)
	if err != nil {
		log.Fatalf("wssrv: tls listen %s: %v", *wssAddr, err)
	}
	log.Printf("wssrv: listening on %s (raw echo | ws) and %s (tls echo | wss)",
		ln.Addr(), tln.Addr())
	accept(tln, true)
}

func accept(ln net.Listener, isTLS bool) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatalf("wssrv: accept: %v", err)
		}
		go serve(conn, isTLS)
	}
}

// serve sniffs the first four bytes and dispatches: "GET " is a WebSocket
// handshake, anything else is the raw echo the net/TLS probes expect.
func serve(conn net.Conn, isTLS bool) {
	defer conn.Close()
	tag := "raw"
	if isTLS {
		tag = "tls"
	}
	if err := conn.SetDeadline(time.Now().Add(30 * time.Second)); err != nil {
		log.Printf("wssrv: set deadline: %v", err)
		return
	}
	var sniff [4]byte
	if _, err := io.ReadFull(conn, sniff[:]); err != nil {
		log.Printf("wssrv: %s connection from %s: sniff failed: %v",
			tag, conn.RemoteAddr(), err)
		return
	}
	if string(sniff[:]) == "GET " {
		wsSession(conn, sniff[:], isTLS)
		return
	}
	// Raw echo: put the sniffed bytes back on the wire first, then copy
	// until the peer closes.
	if _, err := conn.Write(sniff[:]); err != nil {
		log.Printf("wssrv: %s connection from %s: echo start failed: %v",
			tag, conn.RemoteAddr(), err)
		return
	}
	n, err := io.Copy(conn, conn)
	how := "peer closed"
	if isTLS {
		// The trailing read is the close_notify evidence, tlssrv-style:
		// io.EOF means the peer drained with a graceful TLS shutdown.
		_, rerr := conn.Read(make([]byte, 1))
		switch {
		case rerr == io.EOF:
			how = "peer sent TLS close_notify"
		case rerr == nil:
			how = "peer still sending"
		default:
			how = "trailing read: " + rerr.Error()
		}
	}
	log.Printf("wssrv: %s connection from %s: echoed %d bytes (copy err=%v), %s",
		tag, conn.RemoteAddr(), n, err, how)
}

// ---- RFC 6455 server side ----

func wsSession(conn net.Conn, sniff []byte, isTLS bool) {
	tag := "ws"
	if isTLS {
		tag = "wss"
	}
	// http.ReadRequest must see the sniffed bytes followed by the live
	// stream - a reader over the sniff alone would report a truncated
	// request.
	br := bufio.NewReader(io.MultiReader(bytes.NewReader(sniff), conn))
	req, err := http.ReadRequest(br)
	if err != nil {
		log.Printf("wssrv: %s connection from %s: not an HTTP request: %v",
			tag, conn.RemoteAddr(), err)
		return
	}
	key := strings.TrimSpace(req.Header.Get("Sec-WebSocket-Key"))
	if !strings.EqualFold(req.Header.Get("Upgrade"), "websocket") ||
		!tokenListHas(req.Header.Get("Connection"), "Upgrade") || key == "" {
		log.Printf("wssrv: %s connection from %s: %s %s: missing upgrade headers",
			tag, conn.RemoteAddr(), req.Method, req.URL.Path)
		return
	}
	h := sha1.Sum([]byte(key + wsGUID))
	resp := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + base64.StdEncoding.EncodeToString(h[:]) + "\r\n\r\n"
	if _, err := conn.Write([]byte(resp)); err != nil {
		log.Printf("wssrv: %s connection from %s: handshake write failed: %v",
			tag, conn.RemoteAddr(), err)
		return
	}
	log.Printf("wssrv: %s connection from %s: upgraded %s", tag, conn.RemoteAddr(), req.URL.Path)

	echoed := 0
	closeCode := uint16(0)
	closedBy := "abrupt"
frameLoop:
	for {
		fin, op, payload, ok := readFrame(br, conn)
		if !ok {
			break
		}
		switch op & 0x0f {
		case 0x1, 0x2, 0x0: // data (this server never fragments its replies)
			echoed += len(payload)
			writeFrame(conn, op&0x0f, payload)
			_ = fin
		case 0x9: // ping -> pong
			writeFrame(conn, 0xA, payload)
		case 0xA: // pong (we never ping; ignore)
		case 0x8: // close: echo the code and end
			if len(payload) >= 2 {
				closeCode = uint16(payload[0])<<8 | uint16(payload[1])
			}
			writeFrame(conn, 0x8, payload)
			closedBy = "close handshake"
			break frameLoop
		default:
			log.Printf("wssrv: %s connection from %s: reserved opcode %#x, closing",
				tag, conn.RemoteAddr(), op&0x0f)
			break frameLoop
		}
	}
	log.Printf("wssrv: %s connection from %s: echoed %d bytes, %s (close code %d)",
		tag, conn.RemoteAddr(), echoed, closedBy, closeCode)
}

// readFrame reads one client frame: mask bit required (RFC 6455: a client
// must mask), payload unmasked in place. Extended lengths 126/127 honored;
// anything above 1 MiB is rejected (the probes never get close).
func readFrame(br *bufio.Reader, conn net.Conn) (fin bool, op byte, payload []byte, ok bool) {
	var h [2]byte
	if _, err := io.ReadFull(br, h[:]); err != nil {
		return false, 0, nil, false
	}
	if h[1]&0x80 == 0 {
		log.Printf("wssrv: unmasked client frame (RFC 6455 5.1 violation)")
		return false, 0, nil, false
	}
	length := int64(h[1] & 0x7f)
	switch h[1] & 0x7f {
	case 126:
		var ext [2]byte
		if _, err := io.ReadFull(br, ext[:]); err != nil {
			return false, 0, nil, false
		}
		length = int64(ext[0])<<8 | int64(ext[1])
	case 127:
		var ext [8]byte
		if _, err := io.ReadFull(br, ext[:]); err != nil {
			return false, 0, nil, false
		}
		length = 0
		for _, b := range ext {
			length = length<<8 | int64(b)
		}
	}
	if length > 1<<20 {
		log.Printf("wssrv: frame too big (%d bytes)", length)
		return false, 0, nil, false
	}
	var mask [4]byte
	if _, err := io.ReadFull(br, mask[:]); err != nil {
		return false, 0, nil, false
	}
	payload = make([]byte, length)
	if _, err := io.ReadFull(br, payload); err != nil {
		return false, 0, nil, false
	}
	for i := range payload {
		payload[i] ^= mask[i&3]
	}
	return h[0]&0x80 != 0, h[0] & 0x0f, payload, true
}

// writeFrame writes one unmasked server frame (extended lengths included).
func writeFrame(conn net.Conn, op byte, payload []byte) {
	var frame []byte
	switch n := len(payload); {
	case n < 126:
		frame = append([]byte{0x80 | op, byte(n)}, payload...)
	case n <= 0xFFFF:
		frame = append([]byte{0x80 | op, 126, byte(n >> 8), byte(n)}, payload...)
	default:
		frame = []byte{0x80 | op, 127, 0, 0, 0, 0,
			byte(n >> 24), byte(n >> 16), byte(n >> 8), byte(n)}
		frame = append(frame, payload...)
	}
	if _, err := conn.Write(frame); err != nil {
		log.Printf("wssrv: frame write failed: %v", err)
	}
}

// tokenListHas reports whether a comma-separated header value contains the
// token ("keep-alive, Upgrade" contains "Upgrade").
func tokenListHas(value, tok string) bool {
	for _, part := range strings.Split(value, ",") {
		if strings.EqualFold(strings.TrimSpace(part), tok) {
			return true
		}
	}
	return false
}
