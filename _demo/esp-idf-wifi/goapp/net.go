package main

import (
	"github.com/xgo-dev/llgo/pkg/esp/esperr"
	"github.com/xgo-dev/llgo/pkg/esp/net"
	"github.com/xgo-dev/llgo/pkg/esp/wifi"
)

// netProbe exercises pkg/esp/net end to end over the emulator's NAT. After
// the association above the station owns 192.168.4.2 and the emulator
// aliases its gateway address 192.168.4.1 to the host machine, forwarding
// each guest connection to the same port on the host's loopback - which is
// where idfproj/host/echosrv listens (go run ./idfproj/host/echosrv).
//
// The 256-byte pattern echo through that NAT is the PASS gate ([go] net
// PASS). Resolve is best-effort: a failed lookup reports SKIP. The loopback
// self-connect is best-effort up to the handshake - a failed Listen or Dial
// reports SKIP - but once Dial has succeeded into the listener's backlog, a
// failed Accept or a broken round-trip reports FAIL: with the loopback path
// already proven good, that would be a bug in the socket stack, not a
// missing environment feature. Every wait goes through the bounded-quantum
// model, so a missing peer surfaces as an errno, never a hang. Like the
// peripheral probes above, this one must terminate on its own; probeNet is
// the contingency switch.
func netProbe() {
	println("[go] net: start")
	heapMark("net-baseline")

	// 256 bytes of LCG pseudo-random pattern: an echo cannot pass by
	// accident, and the generator needs no imports.
	const patternLen = 256
	var msg [patternLen]byte
	s := uint32(0x9E3779B9)
	for i := range msg {
		s = s*1664525 + 1013904223
		msg[i] = byte(s >> 24)
	}

	t0 := rdcycle()
	// 192.168.4.1 is the esp-emu NAT gateway alias for the host (the
	// firmware itself is 192.168.4.2); on real hardware this would be the AP
	// side's address.
	c, err := net.Dial(net.IPv4(192, 168, 4, 1), 9000, 10000)
	if err != nil {
		println("[go] net dial err =", errText(err))
		println("[go] net FAIL dial")
		return
	}
	println("[go] net dial ok in", int((rdcycle()-t0)/240000), "ms")

	if err := c.SetDeadline(5000); err != nil {
		println("[go] net set_deadline err =", errText(err))
		println("[go] net FAIL deadline")
		c.Close()
		return
	}
	if w, werr := c.Write(msg[:]); werr != nil || w != patternLen {
		println("[go] net write n =", w, "want", patternLen)
		println("[go] net FAIL write")
		c.Close()
		return
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
			println("[go] net read err =", errText(rerr), "at", total)
		} else {
			println("[go] net read peer closed at", total)
		}
		break
	}
	mismatches := 0
	for i := 0; i < total; i++ {
		if got[i] != msg[i] {
			mismatches++
		}
	}
	c.Close()
	if total == patternLen && mismatches == 0 {
		println("[go] net echo", total, "/", patternLen, "bytes verified")
		println("[go] net PASS")
	} else {
		println("[go] net echo got", total, "/", patternLen,
			"bytes, mismatches =", mismatches)
		println("[go] net FAIL echo")
		return
	}
	heapMark("net-echo")

	// Best-effort: name resolution through the emulator's DNS proxy. The
	// dotted-quad path is instant; this checks the localhost lookup.
	t0 = rdcycle()
	ip, rerr := net.Resolve("localhost")
	if rerr != nil {
		println("[go] net resolve localhost err =", errText(rerr), "- SKIP")
	} else {
		println("[go] net resolve localhost =", wifi.IPString(ip),
			"in", int((rdcycle()-t0)/240000), "ms")
	}

	// Best-effort: loopback Listen/Accept self-connect (127.0.0.1 works in
	// the emulator because IDF enables LWIP_NETIF_LOOPBACK). Kept strictly
	// sequential on one goroutine: a quantum wait is a C-level select that
	// gives the llgo scheduler no yield point, so a concurrent shape cannot
	// interleave. Measured under esp-emu (dial goroutine + accept main,
	// 3 s windows each): the accept main ran its whole 3026 ms window alone
	// and timed out, and the dial goroutine - starved the whole time - only
	// ran once the accept side reached a channel join, completing its
	// connect in 0 ms. Sequential also matches the protocol: the handshake
	// completes into the listener's backlog before Accept, so the concurrent
	// form would add nothing but a guaranteed 3 s serialization stall.
	ln, lerr := net.Listen(9001)
	if lerr != nil {
		println("[go] net listen 9001 err =", errText(lerr), "- SKIP")
		return
	}
	cc, derr := net.Dial(net.IPv4(127, 0, 0, 1), 9001, 3000)
	if derr != nil {
		println("[go] net loopback SKIP: dial err =", errText(derr))
		ln.Close()
		return
	}
	ln.SetDeadline(3000)
	lc, aerr := ln.Accept()
	if aerr != nil {
		println("[go] net loopback FAIL: accept err =", errText(aerr))
		cc.Close()
		ln.Close()
		return
	}
	cc.SetDeadline(3000)
	lc.SetDeadline(3000)
	var one, atAccept, back [1]byte
	one[0] = 0x5A
	ok := false
	if w, werr := cc.Write(one[:]); werr == nil {
		ok = w == 1
	}
	if ok {
		n, rerr := lc.Read(atAccept[:])
		ok = n == 1 && rerr == nil && atAccept[0] == 0x5A
	}
	if ok {
		w, werr := lc.Write(atAccept[:])
		ok = werr == nil && w == 1
	}
	if ok {
		n, rerr := cc.Read(back[:])
		ok = n == 1 && rerr == nil && back[0] == 0x5A
	}
	cc.Close()
	lc.Close()
	ln.Close()
	if ok {
		println("[go] net loopback self-connect PASS")
	} else {
		println("[go] net loopback FAIL")
	}
	heapMark("net-loopback")
}

// udpProbe exercises pkg/esp/net's UDP datagram sockets (UDPConn) end to
// end over the same NAT path netProbe just proved, plus the guest loopback.
// Three deadline-bounded round-trips, strictly sequential on this
// goroutine (a quantum wait is a C-level select that gives the llgo
// scheduler no yield point):
//
//  1. unconnected form against the host peer: WriteTo + ReadFrom to
//     192.168.4.1:9002 - forwarded by the NAT to idfproj/host/udpsrv,
//     which echoes each datagram back to its source;
//  2. connected form against the same peer: DialUDP + Write + Read (one
//     Read per datagram: a datagram socket has no partial reads, so the
//     echo arrives as a single piece);
//  3. loopback self-conversation: ListenUDP on 127.0.0.1:9003 (127.0.0.1
//     works in the emulator because IDF enables LWIP_NETIF_LOOPBACK), a
//     connected DialUDP into it, and the bound socket's ReadFrom/WriteTo
//     answering - the UDP analog of netProbe's loopback self-connect.
//
// "[go] net udp PASS" is the probe's gate; every failure prints a
// "[go] net udp FAIL ..." line and returns without PASS. Every wait is
// deadline-bounded (20 ms quanta under explicit deadlines), so a missing
// peer surfaces as an errno, never a hang; probeUDP is the contingency
// switch, same convention as the other probes.
func udpProbe() {
	println("[go] net udp: start")
	const (
		hostLen     = 128 // phase 1: unconnected WriteTo/ReadFrom datagram
		peerLen     = 256 // phase 2: connected Write/Read datagram
		lbLen       = 64  // phase 3: loopback datagram
		udpPort     = 9002
		lbUdpPort   = 9003
		udpDeadline = 5000
	)
	// lcg fills n bytes with the same pseudo-random generator netProbe
	// uses; an echo cannot pass by accident, and it needs no imports.
	lcg := func(seed uint32, n int32) []byte {
		buf := make([]byte, n)
		s := seed
		for i := range buf {
			s = s*1664525 + 1013904223
			buf[i] = byte(s >> 24)
		}
		return buf
	}
	mism := func(a, b []byte) int {
		m := 0
		for i := range a {
			if a[i] != b[i] {
				m++
			}
		}
		return m
	}
	deadline := func(c *net.UDPConn) bool {
		if err := c.SetDeadline(udpDeadline); err != nil {
			println("[go] net udp set_deadline err =", errText(err))
			return false
		}
		return true
	}

	// Phase 1: unconnected WriteTo/ReadFrom to the host peer. The NAT
	// rewrites the echo's source back to 192.168.4.1:9002, so ReadFrom's
	// sender address is deterministic and gets verified against it.
	host := net.Addr{IP: net.IPv4(192, 168, 4, 1), Port: udpPort}
	c, err := net.ListenUDP(nil)
	if err != nil {
		println("[go] net udp listen err =", errText(err))
		println("[go] net udp FAIL listen")
		return
	}
	if !deadline(c) {
		println("[go] net udp FAIL listen-deadline")
		c.Close()
		return
	}
	msg := lcg(0x51ED270B, hostLen)
	if w, werr := c.WriteTo(msg, host); werr != nil || w != hostLen {
		println("[go] net udp write_to n =", w, "err =", errText(werr))
		println("[go] net udp FAIL write-to")
		c.Close()
		return
	}
	var got [hostLen]byte
	n, from, rerr := c.ReadFrom(got[:])
	if rerr != nil || n != hostLen || mism(msg, got[:]) != 0 {
		println("[go] net udp read_from n =", n, "err =", errText(rerr),
			"peer =", from.String(), "mismatches =", mism(msg, got[:]))
		println("[go] net udp FAIL read-from")
		c.Close()
		return
	}
	if from != host {
		println("[go] net udp read_from peer =", from.String(), "want", host.String())
		println("[go] net udp FAIL read-from-peer")
		c.Close()
		return
	}
	c.Close()
	println("[go] net udp peer " + from.String())
	println("[go] net udp round-trip ok n=" + dec32(n))

	// Phase 2: connected DialUDP Write/Read to the same peer.
	cc, derr := net.DialUDP(nil, &host)
	if derr != nil {
		println("[go] net udp dial err =", errText(derr))
		println("[go] net udp FAIL dial")
		return
	}
	if !deadline(cc) {
		println("[go] net udp FAIL dial-deadline")
		cc.Close()
		return
	}
	msg2 := lcg(0x2545F491, peerLen)
	if w, werr := cc.Write(msg2); werr != nil || w != peerLen {
		println("[go] net udp write n =", w, "want", peerLen)
		println("[go] net udp FAIL write")
		cc.Close()
		return
	}
	var got2 [peerLen]byte
	n2, rerr2 := cc.Read(got2[:])
	if rerr2 != nil || n2 != peerLen || mism(msg2, got2[:]) != 0 {
		println("[go] net udp read n =", n2, "err =", errText(rerr2),
			"mismatches =", mism(msg2, got2[:]))
		println("[go] net udp FAIL read")
		cc.Close()
		return
	}
	cc.Close()
	println("[go] net udp connected round-trip ok n=" + dec32(n2))

	// Phase 3: loopback self-conversation on 127.0.0.1. UDP has no
	// accept: the bound socket ReadFrom's the datagram, sees the client's
	// ephemeral port in the sender address, and WriteTo's the echo back.
	lb := net.Addr{IP: net.IPv4(127, 0, 0, 1), Port: lbUdpPort}
	srv, serr := net.ListenUDP(&lb)
	if serr != nil {
		println("[go] net udp listen loopback err =", errText(serr))
		println("[go] net udp FAIL listen-loopback")
		return
	}
	cli, cerr := net.DialUDP(nil, &lb)
	if cerr != nil {
		println("[go] net udp dial loopback err =", errText(cerr))
		println("[go] net udp FAIL dial-loopback")
		srv.Close()
		return
	}
	if !deadline(srv) || !deadline(cli) {
		println("[go] net udp FAIL loopback-deadline")
		cli.Close()
		srv.Close()
		return
	}
	msg3 := lcg(0x0B6675F1, lbLen)
	if w, werr := cli.Write(msg3); werr != nil || w != lbLen {
		println("[go] net udp loopback write n =", w, "want", lbLen)
		println("[go] net udp FAIL loopback-write")
		cli.Close()
		srv.Close()
		return
	}
	var atSrv, back [lbLen]byte
	n3, from3, rerr3 := srv.ReadFrom(atSrv[:])
	if rerr3 != nil || n3 != lbLen || mism(msg3, atSrv[:]) != 0 {
		println("[go] net udp loopback read_from n =", n3, "err =", errText(rerr3),
			"mismatches =", mism(msg3, atSrv[:]))
		println("[go] net udp FAIL loopback-read-from")
		cli.Close()
		srv.Close()
		return
	}
	if w, werr := srv.WriteTo(atSrv[:n3], from3); werr != nil || w != n3 {
		println("[go] net udp loopback write_to n =", w, "err =", errText(werr))
		println("[go] net udp FAIL loopback-echo")
		cli.Close()
		srv.Close()
		return
	}
	n4, rerr4 := cli.Read(back[:])
	cli.Close()
	srv.Close()
	if rerr4 != nil || n4 != lbLen || mism(msg3, back[:]) != 0 {
		println("[go] net udp loopback read n =", n4, "err =", errText(rerr4),
			"mismatches =", mism(msg3, back[:]))
		println("[go] net udp FAIL loopback-read")
		return
	}
	println("[go] net udp loopback round-trip ok n=" + dec32(n4))
	println("[go] net udp PASS")
}

// reconnProbe exercises pkg/esp/wifi's auto-reconnect state machine end to
// end. An emulator cannot make its soft AP drop the station on its own, so
// the probe drives every transition itself; the one IDF signal it cannot
// fake (a router vanishing while the program idles) is reproduced by a
// failed association - connecting to an SSID nobody serves delivers the
// same unexpected EvStaDisconnected a lost router would, and from there the
// machine's behavior is identical. Phases, strictly sequential on this
// goroutine:
//
//  1. SetAutoReconnect(true) + a user Disconnect: the DISCONNECTED arrives
//     (plain IDF event) but the machine must stay idle - nothing reconnects
//     in the quiet window that follows;
//  2. a manual Connect still associates while the machine is enabled, and
//     its GOT_IP resets the backoff;
//  3. a failed association (llgo-nosuchap SSID): the station is switched
//     over through an intentional Disconnect first (the driver refuses a
//     new target while associated - "disconnect before connecting to new
//     ap"), whose event the machine consumes without scheduling; then the
//     association fails and the machine - nobody else, this code makes no
//     Connect call in the window - drives the retries; a second
//     DISCONNECTED inside the window is its fingerprint;
//  4. restoring the real credentials: the machine's next attempt associates
//     and its GOT_IP ends the episode (backoff reset);
//  5. SetAutoReconnect(false): a user disconnect now stays down, proving
//     the off switch;
//  6. a final manual Connect restores the association for the probes that
//     follow.
//
// "[go] wifi reconn: PASS" fires only when every phase held. Every wait is
// a bounded Next window, so a machine that never acts surfaces as a FAIL
// line, never a hang. On real hardware the router-power-cycle variant of
// phase 3 (AP gone mid-association) remains a pending verification item -
// the emulator cannot inject it.
func reconnProbe(ssid, pass string) {
	println("[go] wifi reconn: start")
	ev := wifi.Subscribe(wifi.EvStaDisconnected, wifi.EvStaConnected, wifi.EvGotIP)
	defer ev.Close()

	// waitKind consumes events until one of kind arrives or timeoutMs
	// passes; intermediate events of other kinds are dropped.
	waitKind := func(kind int32, timeoutMs int32) (wifi.Event, bool) {
		for ms := int32(0); ms < timeoutMs; ms += 100 {
			if e, err := ev.Next(100); err == nil {
				if e.Kind == kind {
					return e, true
				}
			}
		}
		return wifi.Event{}, false
	}
	// quiet reports whether no event of interest arrives within ms.
	quiet := func(ms int32) bool {
		for left := int32(0); left < ms; left += 100 {
			if _, err := ev.Next(100); err == nil {
				return false
			}
		}
		return true
	}

	// Phase 1: the machine is enabled, then the program disconnects on
	// purpose - the disconnect event must not schedule a retry.
	wifi.SetAutoReconnect(true)
	println("[go] wifi reconn: enable phase=" + wifi.ReconnectState().String())
	if wifi.Disconnect() != nil {
		println("[go] wifi reconn: FAIL user-disconnect")
		return
	}
	if _, ok := waitKind(wifi.EvStaDisconnected, 3000); !ok {
		println("[go] wifi reconn: FAIL user-disconnect-event")
		return
	}
	if !quiet(5000) || wifi.ReconnectState() != wifi.ReconnIdle {
		println("[go] wifi reconn: FAIL suppress")
		return
	}
	println("[go] wifi reconn: user disconnect suppressed")

	// Phase 2: a manual reconnect still works with the machine enabled.
	if wifi.Connect() != nil {
		println("[go] wifi reconn: FAIL manual-connect")
		return
	}
	if _, ok := waitKind(wifi.EvGotIP, 15000); !ok || wifi.ReconnectState() != wifi.ReconnIdle {
		println("[go] wifi reconn: FAIL manual-reconnect")
		return
	}
	println("[go] wifi reconn: manual reconnect ok")

	// Phase 3: unexpected disconnection via a failed association. The
	// driver rejects a new target while associated ("disconnect before
	// connecting to new ap"), so the switch goes through an intentional
	// Disconnect first - its event arms the flag and the machine consumes
	// it without scheduling. From there the machine owns esp_wifi_connect.
	if wifi.Disconnect() != nil {
		println("[go] wifi reconn: FAIL phase3-disconnect")
		return
	}
	if _, ok := waitKind(wifi.EvStaDisconnected, 3000); !ok {
		println("[go] wifi reconn: FAIL phase3-disconnect-event")
		return
	}
	if wifi.SetSTA("llgo-nosuchap-xyz", "nosuchpassword") != nil {
		println("[go] wifi reconn: FAIL set-sta-bad")
		return
	}
	if wifi.Connect() != nil {
		println("[go] wifi reconn: FAIL bad-connect")
		return
	}
	if _, ok := waitKind(wifi.EvStaDisconnected, 10000); !ok {
		println("[go] wifi reconn: FAIL unexpected-disconnect")
		return
	}
	// No Connect call from this code until phase 4: the next
	// DISCONNECTED can only come from a machine-driven retry.
	if _, ok := waitKind(wifi.EvStaDisconnected, 12000); !ok {
		println("[go] wifi reconn: FAIL no-auto-retry")
		return
	}
	println("[go] wifi reconn: auto retries observed")

	// Phase 4: restore the real credentials; the machine's next attempt
	// (still on its own backoff schedule) recovers and ends the episode.
	if wifi.SetSTA(ssid, pass) != nil {
		println("[go] wifi reconn: FAIL set-sta-good")
		return
	}
	got, ok := waitKind(wifi.EvGotIP, 20000)
	if !ok || got.IP == 0 {
		println("[go] wifi reconn: FAIL auto-recover")
		return
	}
	println("[go] wifi reconn: auto recovered ip=" + wifi.IPString(got.IP))

	// Phase 5: switch the machine off; a user disconnect stays down.
	wifi.SetAutoReconnect(false)
	if wifi.Disconnect() != nil {
		println("[go] wifi reconn: FAIL disable-disconnect")
		return
	}
	if _, ok := waitKind(wifi.EvStaDisconnected, 3000); !ok {
		println("[go] wifi reconn: FAIL disable-disconnect-event")
		return
	}
	if !quiet(4000) || wifi.ReconnectState() != wifi.ReconnIdle {
		println("[go] wifi reconn: FAIL disable-suppress")
		return
	}
	println("[go] wifi reconn: disable stays down")

	// Phase 6: restore the association the net/tls/ws probes expect.
	if wifi.Connect() != nil {
		println("[go] wifi reconn: FAIL restore-connect")
		return
	}
	got, ok = waitKind(wifi.EvGotIP, 15000)
	if !ok || got.IP == 0 {
		println("[go] wifi reconn: FAIL restore-ip")
		return
	}
	println("[go] wifi reconn: restored ip=" + wifi.IPString(got.IP))
	println("[go] wifi reconn: PASS")
}
