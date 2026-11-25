package main

import (
	"unsafe"

	"github.com/xgo-dev/llgo/pkg/esp/esperr"
	"github.com/xgo-dev/llgo/pkg/esp/net"
	"github.com/xgo-dev/llgo/pkg/esp/tls"
)

// compiledCAPEM reads the acceptance CA (or the negative-test CA) that
// idfproj/host/gencert.sh generated into main/tls_ca.h / tls_ca_wrong.h,
// which demo_glue.c exposes. An empty result means "not compiled in": the
// probe reports SKIP for the VERIFY_REQUIRED phase, exactly like the
// scan-only wifi run. The buffer lives on the Go heap (a 2 KiB stack buffer
// would violate the fiber-stack discipline).
//
//go:linkname demoTLSCA C.llgo_demo_tls_ca
func demoTLSCA(buf *byte, cap int32, outLen *int32)

//go:linkname demoTLSCAWrong C.llgo_demo_tls_ca_wrong
func demoTLSCAWrong(buf *byte, cap int32, outLen *int32)

func compiledCAPEM(buf *byte, outLen *int32, wrong bool) []byte {
	if wrong {
		demoTLSCAWrong(buf, int32(2048), outLen)
	} else {
		demoTLSCA(buf, int32(2048), outLen)
	}
	if *outLen <= 0 {
		return nil
	}
	return unsafe.Slice(buf, *outLen)
}

// tlsRound exercises one live TLS session: write the 256-byte pattern, read
// the echo back in short reads, compare, and close with a graceful
// close_notify (whose host-side evidence is the tlssrv log line "peer sent
// TLS close_notify"). Reports the outcome as a [go] tls line; returns
// whether the round verified.
func tlsRound(c *tls.TLSConn, tag string) bool {
	const patternLen = 256
	var msg [patternLen]byte
	s := uint32(0x12345678)
	for i := range msg {
		s = s*1664525 + 1013904223
		msg[i] = byte(s >> 24)
	}
	if err := c.SetDeadline(5000); err != nil {
		println("[go] tls", tag, "set_deadline err =", errText(err))
		return false
	}
	if w, werr := c.Write(msg[:]); werr != nil || w != patternLen {
		println("[go] tls", tag, "write n =", w, "want", patternLen)
		return false
	}
	var got [patternLen]byte
	total := 0
	for total < patternLen {
		n, rerr := c.Read(got[total:])
		if n > 0 {
			total += n
			continue
		}
		if rerr != nil && rerr != esperr.ErrEOF {
			println("[go] tls", tag, "read err =", errText(rerr), "at", total)
		} else {
			println("[go] tls", tag, "read peer closed at", total)
		}
		break
	}
	mismatches := 0
	for i := 0; i < total; i++ {
		if got[i] != msg[i] {
			mismatches++
		}
	}
	if err := c.Close(); err != nil {
		println("[go] tls", tag, "close report =", errText(err))
	}
	if total == patternLen && mismatches == 0 {
		println("[go] tls", tag, "echo", total, "/", patternLen, "bytes verified")
		return true
	}
	println("[go] tls", tag, "echo got", total, "/", patternLen,
		"bytes, mismatches =", mismatches)
	return false
}

// tlsProbe exercises pkg/esp/tls end to end over the emulator's NAT against
// idfproj/host/tlssrv (TLS echo on 127.0.0.1:9443, forwarded as
// 192.168.4.1:9443). Phases:
//
//  1. VERIFY_NONE dial + 256-byte echo + close_notify - the deterministic
//     core ("[go] tls none ...");
//  2. VERIFY_REQUIRED dial with the compiled-in CA, same exchange
//     ("[go] tls required ..."); SKIPped when no CA was compiled in (run
//     idfproj/host/gencert.sh), like the scan-only wifi run;
//  3. negative test: a dial with a CA that signed nothing must be rejected
//     with CertVerifyFailed ("[go] tls bad-ca ...");
//  4. the Config+Dial probes the deprecated TLSDial never covered: correct
//     ServerName + CA passes ("[go] tls sni ok"), wrong ServerName and
//     wrong CA are rejected ("[go] tls sni reject-host / reject-ca"), and
//     InsecureSkipVerify still completes ("[go] tls skip ok") - the four
//     SNI acceptance lines; each passing dial is proven usable by a quiet
//     echo round plus the exported ConnectionState.
//
// "[go] tls PASS" fires only when phase 1 passed AND phase 2 ran and passed
// AND phase 4's four probes went the right direction
// - that is the emu acceptance gate (--exit-on). Without a compiled-in CA
// phase 2 is SKIPped and the probe ends at "[go] tls done" - PASS is never
// printed, so a gate run in that shape cannot exit green by accident (it
// runs to the emu's 150 s timeout instead); use --exit-on "[go] tls done"
// for the run that reports the SKIP, same convention as the scan-only wifi
// run. A FAIL path ends the probe without the done marker, so it ends at
// the timeout too. Every wait
// is bounded (20 ms quanta under deadlines), so a missing peer surfaces as
// an error code, never a hang; probeTLS is the contingency switch.
func tlsProbe() {
	println("[go] tls: start")
	heapMark("tls-baseline")

	t0 := rdcycle()
	c, err := tls.TLSDial(net.IPv4(192, 168, 4, 1), 9443, 15000, nil)
	if err != nil {
		println("[go] tls none dial err =", errText(err))
		println("[go] tls FAIL dial-none")
		return
	}
	println("[go] tls none dial ok in", int((rdcycle()-t0)/240000), "ms")
	// Live-connection mark: the buffers + handshake state are held right
	// now (this is the "does the default MFL hit the internal-heap red
	// line" number). tlsRound closes before the next mark, so the free
	// column between marks is post-close - only the live marks show the
	// held footprint.
	heapMark("tls-none-live")
	if !tlsRound(c, "none") {
		println("[go] tls FAIL none")
		return
	}
	heapMark("tls-none")

	caBuf := make([]byte, 2048)
	wrongBuf := make([]byte, 2048)
	var caLen, wrongLen int32
	ca := compiledCAPEM(&caBuf[0], &caLen, false)
	caWrong := compiledCAPEM(&wrongBuf[0], &wrongLen, true)
	if ca == nil {
		println("[go] tls required SKIP: no CA compiled in (run idfproj/host/gencert.sh)")
		println("[go] tls done")
		return
	}
	t0 = rdcycle()
	c, err = tls.TLSDial(net.IPv4(192, 168, 4, 1), 9443, 15000, ca)
	if err != nil {
		println("[go] tls required dial err =", errText(err))
		println("[go] tls FAIL dial-required")
		return
	}
	println("[go] tls required dial ok in", int((rdcycle()-t0)/240000), "ms")
	heapMark("tls-required-live")
	if !tlsRound(c, "required") {
		println("[go] tls FAIL required")
		return
	}
	heapMark("tls-required")

	// Negative test: a CA that signed nothing must not authenticate the
	// peer. The dial fails inside the handshake with CertVerifyFailed;
	// any other outcome (success, or a different error) is a FAIL.
	if caWrong == nil {
		println("[go] tls bad-ca SKIP: no wrong CA compiled in")
	} else {
		c, err = tls.TLSDial(net.IPv4(192, 168, 4, 1), 9443, 15000, caWrong)
		if err == tls.ErrCertVerifyFailed {
			println("[go] tls bad-ca correctly rejected (", errText(err), ")")
		} else {
			println("[go] tls bad-ca dial err =", errText(err))
			println("[go] tls FAIL bad-ca (want CertVerifyFailed)")
			return
		}
		heapMark("tls-badca")
	}

	// SNI/identity probes through the Config+Dial path - the checks the
	// deprecated TLSDial never did. The TCP destination stays the NAT alias
	// the whole run dials; the certificate gencert.sh made names the
	// service behind it (SAN DNS:localhost, IP:127.0.0.1), so the verified
	// name travels in Config.ServerName, not in the address. Four probes,
	// one line each: the right name passes, the wrong name and the wrong CA
	// are rejected inside the handshake, and the explicit skip shape still
	// completes. Any other outcome prints a FAIL line and stops before
	// PASS, so the emu acceptance gate covers these four directions too.
	natPeer := "192.168.4.1:9443"

	// Correct name + correct CA + a version floor: a fully verified
	// session. ConnectionState is read while the session is live (State on
	// a closed conn legitimately reports nothing), then the quiet echo
	// round proves the session usable and closes it.
	c, err = tls.Dial(natPeer, 15000, &tls.Config{
		ServerName: "localhost",
		RootCAs:    ca,
		MinVersion: tls.VersionTLS12,
	})
	if err != nil {
		println("[go] tls sni dial err =", errText(err))
		println("[go] tls FAIL sni-ok")
		return
	}
	if !tlsStateVerified(c) {
		c.Close()
		println("[go] tls FAIL sni-ok (ConnectionState)")
		return
	}
	if !tlsRoundQuiet(c) {
		println("[go] tls FAIL sni-ok (echo)")
		return
	}
	println("[go] tls sni ok")

	// Wrong name, right CA: the handshake must refuse the peer - this is
	// the MITM shape TLSDial would have accepted (any cert of this CA).
	c, err = tls.Dial(natPeer, 15000, &tls.Config{ServerName: "nope.llgo.test", RootCAs: ca})
	if err == tls.ErrCertVerifyFailed {
		println("[go] tls sni reject-host (", errText(err), ")")
	} else {
		println("[go] tls sni dial err =", errText(err))
		println("[go] tls FAIL reject-host (want CertVerifyFailed)")
		return
	}

	// Right name, wrong CA: the chain cannot be built to the anchor that
	// signed nothing.
	c, err = tls.Dial(natPeer, 15000, &tls.Config{ServerName: "localhost", RootCAs: caWrong})
	if err == tls.ErrCertVerifyFailed {
		println("[go] tls sni reject-ca (", errText(err), ")")
	} else {
		println("[go] tls sni dial err =", errText(err))
		println("[go] tls FAIL reject-ca (want CertVerifyFailed)")
		return
	}

	// Explicit skip: no identity checking, session still fully usable
	// (same live-State + quiet-round order as the sni-ok probe).
	c, err = tls.Dial(natPeer, 15000, &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		println("[go] tls skip dial err =", errText(err))
		println("[go] tls FAIL skip-ok")
		return
	}
	if !tlsStateVerified(c) {
		c.Close()
		println("[go] tls FAIL skip-ok (ConnectionState)")
		return
	}
	if !tlsRoundQuiet(c) {
		println("[go] tls FAIL skip-ok (echo)")
		return
	}
	println("[go] tls skip ok")

	println("[go] tls PASS")
	println("[go] tls done")
}

// tlsRoundQuiet is tlsRound without the per-step prints (the sni probes
// below report exactly one line each): the 256-byte pattern out, the echo
// back in short reads, byte-compare, graceful close. Closes the conn on
// every path; returns whether the round verified.
func tlsRoundQuiet(c *tls.TLSConn) bool {
	const patternLen = 256
	var msg [patternLen]byte
	s := uint32(0x9e3779b9)
	for i := range msg {
		s = s*1664525 + 1013904223
		msg[i] = byte(s >> 24)
	}
	defer c.Close()
	if err := c.SetDeadline(5000); err != nil {
		return false
	}
	if w, werr := c.Write(msg[:]); werr != nil || w != patternLen {
		return false
	}
	var got [patternLen]byte
	total := 0
	for total < patternLen {
		n, _ := c.Read(got[total:])
		if n > 0 {
			total += n
			continue
		}
		return false // peer closed early (0, io.EOF) or a real error
	}
	for i := 0; i < total; i++ {
		if got[i] != msg[i] {
			return false
		}
	}
	return true
}

// tlsStateVerified checks ConnectionState's shape on a live session: the
// negotiated version and cipher suite strings are present and the peer
// certificate exported with content. No prints - the sni probes keep one
// line per probe; a FAIL line only fires when the probe overall fails.
func tlsStateVerified(c *tls.TLSConn) bool {
	st := c.State()
	return st.Version != "" && st.CipherSuite != "" &&
		len(st.PeerCerts) == 1 && len(st.PeerCerts[0]) > 0
}
