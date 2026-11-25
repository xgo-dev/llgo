package main

import (
	"github.com/xgo-dev/llgo/pkg/esp/esperr"
	"github.com/xgo-dev/llgo/pkg/esp/net"
	"github.com/xgo-dev/llgo/pkg/esp/tls"
	"github.com/xgo-dev/llgo/pkg/esp/ws"
)

// wsRound exercises one live WebSocket session: send the 256-byte pattern
// as one binary frame, read the echo back (skipping control events),
// compare, ping and wait for the pong, then close with the RFC's normal
// closure code. Reports the outcome as [go] ws lines; returns whether the
// round verified. The host-side evidence for the close handshake is the
// wssrv log line "close handshake (close code 1000)".
func wsRound(w *ws.WSConn, tag string, seed uint32) bool {
	const patternLen = 256
	var msg [patternLen]byte
	s := seed
	for i := range msg {
		s = s*1664525 + 1013904223
		msg[i] = byte(s >> 24)
	}
	if err := w.SetDeadline(5000); err != nil {
		println("[go] ws", tag, "set_deadline err =", errText(err))
		return false
	}
	if r, werr := w.WriteMessage(msg[:], ws.OpBinary); werr != nil || r != patternLen {
		println("[go] ws", tag, "write r =", r, "want", patternLen)
		w.Close(1002)
		return false
	}
	var got [patternLen]byte
	total := 0
	for total < patternLen {
		n, rerr := w.ReadMessage(got[total:])
		if n > 0 {
			total += n
			continue
		}
		if rerr != nil && rerr != esperr.ErrEOF {
			println("[go] ws", tag, "read err =", errText(rerr), "at", total)
		} else {
			println("[go] ws", tag, "control/zero-length event during echo read")
		}
		break
	}
	mismatches := 0
	for i := 0; i < total; i++ {
		if got[i] != msg[i] {
			mismatches++
		}
	}
	ok := total == patternLen && mismatches == 0
	if ok {
		if w.LastOpcode() != ws.OpBinary {
			println("[go] ws", tag, "echo opcode =", int(w.LastOpcode()), "want binary")
			ok = false
		}
	}
	if ok {
		println("[go] ws", tag, "echo", total, "/", patternLen, "bytes verified")
	} else {
		println("[go] ws", tag, "echo got", total, "/", patternLen,
			"bytes, mismatches =", mismatches)
		w.Close(1002)
		return false
	}

	// Ping/pong: the pong is consumed by the read path and reported as a
	// (0, nil) event with PongSeen() true - pump until it shows up or the
	// deadline (2 s) cuts the wait.
	if err := w.Ping(); err != nil {
		println("[go] ws", tag, "ping err =", errText(err))
		w.Close(1002)
		return false
	}
	w.SetDeadline(2000)
	sawPong := false
	var scratch [8]byte
	for i := 0; i < 100 && !sawPong; i++ {
		if _, rerr := w.ReadMessage(scratch[:]); rerr != nil && rerr != ws.ErrTimeout {
			println("[go] ws", tag, "pong wait err =", errText(rerr))
			w.Close(1002)
			return false
		}
		sawPong = w.PongSeen()
	}
	if sawPong {
		println("[go] ws", tag, "pong ok")
	} else {
		println("[go] ws", tag, "FAIL pong")
		w.Close(1002)
		return false
	}

	// Closing handshake: the deadline bounds the wait for the peer's echo
	// (nil = graceful); wssrv logs the code as the host-side evidence.
	w.SetDeadline(2000)
	if err := w.Close(1000); err != nil {
		println("[go] ws", tag, "close report =", errText(err))
		return false
	}
	println("[go] ws", tag, "close ok (peer code", int(w.CloseCode()), ")")
	return true
}

// wsSession dials the transport, handshakes /ws, runs one round, and
// prints the live/post-close heap marks named ws-<tag>-live / ws-<tag>.
func wsSession(tag, hostport string, port uint16, caPEM []byte) bool {
	t0 := rdcycle()
	var c ws.Conn
	var derr error
	if caPEM == nil && port == 9000 {
		c, derr = net.Dial(net.IPv4(192, 168, 4, 1), port, 10000)
	} else {
		c, derr = tls.TLSDial(net.IPv4(192, 168, 4, 1), port, 15000, caPEM)
	}
	if derr != nil {
		println("[go] ws", tag, "dial err =", errText(derr))
		return false
	}
	w, err := ws.HandshakeURL(c, hostport, "/ws", 5000)
	if err != nil {
		println("[go] ws", tag, "handshake err =", errText(err))
		return false
	}
	println("[go] ws", tag, "handshake ok in", int((rdcycle()-t0)/240000), "ms")
	heapMark("ws-" + tag + "-live")
	ok := wsRound(w, tag, uint32(0x12345678)^(uint32(port)<<16))
	heapMark("ws-" + tag)
	return ok
}

// wsProbe exercises pkg/esp/ws end to end over the emulator's NAT against
// idfproj/host/wssrv (raw echo or WebSocket on 127.0.0.1:9000, TLS echo or
// wss on 127.0.0.1:9443, sniffed per connection). Phases, strictly
// sequential on this goroutine (a quantum wait is a C-level select that
// gives the scheduler no yield point):
//
//  0. crypto smoke: sha1("abc") (FIPS 180-1), the RFC 6455 section 1.3
//     accept-key example and RFC 4648 base64 vectors against the in-package
//     SHA-1/base64 (the package stays stdlib-free for image size - see the
//     ws package doc);
//  1. ws://192.168.4.1:9000/ws (plain TCP transport);
//  2. wss://192.168.4.1:9443/ws with TLSDial VERIFY_NONE;
//  3. wss://192.168.4.1:9443/ws with the compiled-in CA (VERIFY_REQUIRED);
//     SKIPped when no CA was compiled in (run idfproj/host/gencert.sh),
//     like the tls probe's required phase.
//
// Each of phases 1-3 runs handshake + binary echo + ping/pong + close.
// "[go] ws PASS" fires only when phase 1 AND 2 passed AND phase 3 ran and
// passed - that is the emu acceptance gate (--exit-on). Without a
// compiled-in CA phase 3 is SKIPped and the probe ends at "[go] ws done" -
// PASS is never printed, so a gate run in that shape cannot exit green by
// accident (it runs to the emu's 150 s timeout instead); use --exit-on
// "[go] ws done" for the run that reports the SKIP. A FAIL path ends the
// probe without the done marker, so it ends at the timeout too. Every wait
// is deadline-bounded (net/tls quanta), so a missing peer surfaces as an
// error code, never a hang; probeWS is the contingency switch.
func wsProbe() {
	println("[go] ws: start")
	heapMark("ws-baseline")

	// Phase 0: handshake-crypto known values.
	wantSum := [20]byte{
		0xa9, 0x99, 0x3e, 0x36, 0x47, 0x06, 0x81, 0x6a, 0xba, 0x3e,
		0x25, 0x71, 0x78, 0x50, 0xc2, 0x6c, 0x9c, 0xd0, 0xd8, 0x9d,
	}
	if got := ws.SHA1Sum([]byte("abc")); got != wantSum {
		println("[go] ws sha1 abc MISMATCH")
		println("[go] ws FAIL crypto-smoke")
		return
	}
	println("[go] ws sha1 abc ok")
	if ws.AcceptKey([]byte("dGhlIHNhbXBsZSBub25jZQ==")) != "s3pPLMBiTxaQ9kYGzzhZRbK+xOo=" {
		println("[go] ws accept-key MISMATCH (RFC 6455 example)")
		println("[go] ws FAIL crypto-smoke")
		return
	}
	println("[go] ws accept-key ok (RFC 6455 example)")
	enc := ws.Base64StdEncode([]byte("foobar"))
	dec, derr := ws.Base64StdDecode("Zm9vYg==")
	if enc != "Zm9vYmFy" || derr != nil || len(dec) != 4 || string(dec) != "foob" {
		println("[go] ws base64 MISMATCH", enc)
		println("[go] ws FAIL crypto-smoke")
		return
	}
	println("[go] ws base64 vectors ok (RFC 4648)")

	// Seed the handshake-key / mask-key PRNG with the cycle counter (the
	// best entropy a lab bench has; see the ws package doc's randomness
	// note for what a deployment would use).
	ws.Seed(uint64(rdcycle()) | 1)

	plainOK := wsSession("plain", "192.168.4.1:9000", 9000, nil)
	if !plainOK {
		println("[go] ws FAIL plain")
		return
	}
	noneOK := wsSession("wss-none", "192.168.4.1:9443", 9443, nil)
	if !noneOK {
		println("[go] ws FAIL wss-none")
		return
	}

	caBuf := make([]byte, 2048)
	var caLen int32
	ca := compiledCAPEM(&caBuf[0], &caLen, false)
	if ca == nil {
		println("[go] ws wss-required SKIP: no CA compiled in (run idfproj/host/gencert.sh)")
		println("[go] ws done")
		return
	}
	reqOK := wsSession("wss-required", "192.168.4.1:9443", 9443, ca)
	if !reqOK {
		println("[go] ws FAIL wss-required")
		return
	}
	println("[go] ws PASS")
	println("[go] ws done")
}
