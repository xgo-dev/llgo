// SPP server acceptance probe for the llgo ESP-IDF pipeline (ESP32-S31).
//
// Built with: llgo build -target esp32s31-idf -o ../idfproj/main/llgo_out.o ./goapp
// The relocatable object is linked into the IDF firmware by the main component;
// app_main calls the generated C entry, which hands control to the llgo fiber
// scheduler (runtime.RunMain).
//
// The probe drives the classic Bluetooth surface in order: controller +
// Bluedroid bring-up, the device name and discoverability, the SPP server
// (SDP record + RFCOMM listener), the wait for the peer (idfproj/host peer
// run of bumble_classic_peer.py spp), the echo loop over the received stream
// (every read chunk is written straight back), the byte-pattern verification,
// the peer's close, and the stack shutdown. Every step prints one `[go] bt ...`
// line; `[go] bt PASS` is the acceptance gate.
package main

import (
	_ "unsafe" // for go:linkname

	"github.com/xgo-dev/llgo/pkg/esp/bt"
	"github.com/xgo-dev/llgo/pkg/esp/esperr"
)

// llgoFixStreams is the assembly helper merged into llgo_out.o
// (host/llgo_syms.S): it repoints clite's stdout/stderr/stdin handles at the
// ESP-IDF picolibc streams, because clite's baremetal init fopen fails under
// IDF's VFS. Must run after package init and before the first print.
//
//go:linkname fixStreams C.llgoFixStreams
func fixStreams()

// heapPrint prints the IDF heap pools from the C side (main/demo_glue.c).
//
//go:linkname heapPrint C.llgo_demo_heap
func heapPrint(tag *byte, n int32)

const (
	// acceptWaitMs bounds the wait for the peer: the host-side bumble peer
	// process defaults to waiting 10 s before it pages, so the probe's
	// budget covers a slow start plus the connection attempt itself.
	acceptWaitMs = 60000

	// ioWaitMs bounds one Read/Write inside the echo loop: the peer sends
	// one 512-byte record every 200 ms, so five seconds covers dozens of
	// records plus the link's own latency.
	ioWaitMs = 5000

	// stepWaitMs is the bound of one event wait inside the A2DP phase's
	// polling loops (the SPP phase waits through SPPAccept/Read's own
	// bounds instead).
	stepWaitMs = 500

	// totalBytes is the payload the peer sends: 25 records of 512 bytes
	// (bumble_classic_peer.py run_spp).
	totalBytes = 25 * 512
)

// heapMark prints the IDF heap watermarks at one probe stage. The label is a
// short literal, copied into a stack buffer so the C side sees plain bytes.
func heapMark(tag string) {
	var buf [24]byte
	n := copy(buf[:], tag)
	heapPrint(&buf[0], int32(n))
}

// errText renders an error for the log: "0" for success, "CODE NAME" for a
// family error value.
func errText(err error) string {
	if err == nil {
		return "0"
	}
	if e, ok := err.(*esperr.Error); ok {
		if e.Name != "" {
			return dec32(uint32(e.Code)) + " " + e.Name
		}
		return dec32(uint32(e.Code))
	}
	return "nonzero"
}

// isEOF reports whether err is the family's end-of-stream condition (the
// peer's close) - a plain code comparison, the same test esperr.Is applies.
func isEOF(err error) bool {
	e, ok := err.(*esperr.Error)
	return ok && e.Code == esperr.ErrEOF.Code
}

// dec32 formats v in decimal (no strconv: the stdlib drags reflect machinery
// into a bare-metal image, the family rule every pkg/esp probe follows).
func dec32(v uint32) string {
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

func main() {
	fixStreams()
	println("[go] bt: start")
	println("[go] bt backend = " + bt.Backend())

	// fail prints the phase's FAIL verdict and tears the stack down, so a
	// failed run ends at the done anchor within seconds instead of idling
	// until the emulator's timeout. Best-effort: on the earliest failures
	// there may be nothing up yet, and Deinit's answer is diagnostic noise.
	fail := func(msg string) {
		println("[go] bt FAIL " + msg)
		if err := bt.Deinit(); err != nil {
			println("[go] bt deinit err=" + errText(err))
		}
	}

	// Subscribe before Init: every lifecycle event of the run then has a
	// reader from the first moment, and the profiles' own confirmations
	// (which SPPListen waits through the same mailbox) are never lost. The
	// probe takes the catch-all: it watches every kind of both profiles.
	sub := bt.Subscribe()
	defer sub.Close()

	if err := bt.Init(); err != nil {
		println("[go] bt init err=" + errText(err))
		fail("init")
		return
	}
	println("[go] bt init ok")
	heapMark("init")

	if err := bt.SetName("llgo-bt"); err != nil {
		println("[go] bt name err=" + errText(err))
		fail("name")
		return
	}
	println("[go] bt name ok")

	// Discoverable + connectable: the S31 answers inquiry and page, which is
	// what lets the peer's page land.
	if err := bt.SetDiscoverable(true); err != nil {
		println("[go] bt discoverable err=" + errText(err))
		fail("discoverable")
		return
	}
	println("[go] bt discoverable ok")

	// The SPP server: profile init (CB mode), then the SDP record + RFCOMM
	// listener on SCN 1 (the SPPOptions zero value). SPPListen returns after
	// the profile's own START confirmation, so the record bumble discovers
	// is already live.
	if err := bt.SPPListen(bt.SPPOptions{}); err != nil {
		println("[go] bt spp listen err=" + errText(err))
		fail("listen")
		return
	}
	println("[go] bt spp listen scn=" + dec32(uint32(bt.DefaultSCN)))
	heapMark("listen")

	println("[go] bt accept wait")
	conn, err := bt.SPPAccept(acceptWaitMs)
	if err != nil {
		println("[go] bt accept err=" + errText(err))
		fail("accept")
		return
	}
	println("[go] bt accept ok addr=" + conn.AddrString())
	heapMark("connected")

	// Echo loop: every received record goes straight back to the peer. The
	// pattern check is stream-positional (byte i of the peer's payload is
	// 0x30 + ((i/512) % 10)), so it stays true no matter how RFCOMM frames
	// the records. The loop ends on the peer's close (esperr.ErrEOF).
	var buf [1024]byte
	rx, tx, pos := 0, 0, 0
	patOK := true
	eof := false
	for !eof {
		n, rerr := conn.Read(buf[:], ioWaitMs)
		if rerr != nil {
			if isEOF(rerr) {
				eof = true
				break
			}
			println("[go] bt read err=" + errText(rerr))
			fail("read")
			return
		}
		for j := 0; j < n; j++ {
			if buf[j] != byte(0x30+((pos+j)/512)%10) {
				patOK = false
			}
		}
		pos += n
		rx += n
		m, werr := conn.Write(buf[:n], ioWaitMs)
		if werr != nil {
			println("[go] bt write err=" + errText(werr))
			fail("write")
			return
		}
		tx += m
	}
	if rx != totalBytes || !patOK {
		fail("echo rx=" + dec32(uint32(rx)) +
			" pat=" + dec32(boolTo32(patOK)))
		return
	}
	println("[go] bt echo rx=" + dec32(uint32(rx)) +
		" tx=" + dec32(uint32(tx)) + " pat=ok")
	println("[go] bt peer closed eof")
	heapMark("echo")

	// ---- A2DP sink phase ----
	//
	// The same boot serves the audio profile next: the SPP peer has left
	// (its ACL drop is in the mailbox), so the A2DP peer pages into a quiet
	// link. Enable wires the decoded-PCM path to I2S pins 4/5/6; a phone
	// then connects, negotiates SBC and streams - the audio itself never
	// enters Go, the probe only watches the control-plane events and the
	// byte counters until the peer drops the link.
	if err := bt.A2DPEnable(bt.A2DPConfig{BclkPin: 4, WsPin: 5, DoutPin: 6,
		DmaDescNum: 12, DmaFrameNum: 511}); err != nil {
		println("[go] bt a2dp enable err=" + errText(err))
		fail("a2dp enable")
		return
	}
	println("[go] bt a2dp enable ok")
	heapMark("a2dp on")

	connected := false
	cfg := false
	rate, chans := int32(0), int32(0)
	for i := 0; i < 120 && (!connected || !cfg); i++ { // 60 s to page + stream up
		ev, err := sub.Next(stepWaitMs)
		if err != nil {
			continue
		}
		switch ev.Kind {
		case int32(bt.EvA2DPConnected):
			connected = true
			println("[go] bt a2dp connected addr=" + ev.AddrString())
		case int32(bt.EvA2DPAudioCfg):
			cfg = true
			rate, chans = ev.Aux, ev.Status
			println("[go] bt a2dp cfg rate=" + dec32(uint32(rate)) +
				" ch=" + dec32(uint32(chans)))
		}
	}
	if !connected || !cfg {
		fail("a2dp stream up")
		return
	}
	heapMark("a2dp cfg")

	// Stream until the peer drops the link (its script pumps the whole file
	// and disconnects). The counters are the audio path's telemetry: the
	// source's media clock feeds total, the local audio clock consumes
	// written, and dropped is what a full ring swallowed.
	gone := false
	for i := 0; i < 600 && !gone; i++ { // 300 s of streaming budget
		ev, err := sub.Next(stepWaitMs)
		if err != nil {
			continue
		}
		if ev.Kind == int32(bt.EvA2DPDisconnected) {
			gone = true
			println("[go] bt a2dp disconnected rsn=" + dec32(uint32(ev.Status)))
		}
	}
	if !gone {
		fail("a2dp stream end")
		return
	}
	total, written, dropped := bt.A2DPStats()
	println("[go] bt a2dp stats total=" + dec32(total) +
		" written=" + dec32(written) + " dropped=" + dec32(dropped))
	// The gate is a ratio, not a non-zero check: the source's full-speed
	// pump still leaves the I2S path with well over half the PCM in the emu
	// (the fixed audio clock consumes it as fast as it is produced), so a
	// wedged pump (a few bytes written) or an all-drop run fails here.
	if total == 0 || written < total/2 {
		fail("a2dp starved")
		return
	}
	println("[go] bt a2dp stream ok")
	heapMark("a2dp eof")

	println("[go] bt PASS")

	// Shutdown, after the gate so a teardown failure cannot hide the
	// acceptance result: the profiles deinit closes what is left of the
	// connections, the stack powers down, and the rings reset.
	if err := bt.A2DPDisable(); err != nil {
		println("[go] bt a2dp disable err=" + errText(err))
		return
	}
	println("[go] bt a2dp disable ok")
	if err := bt.Deinit(); err != nil {
		println("[go] bt deinit err=" + errText(err))
		return
	}
	println("[go] bt deinit ok")
	heapMark("deinit")
	println("[go] bt done")
}

// boolTo32 renders a flag as 0/1 for the log.
func boolTo32(v bool) uint32 {
	if v {
		return 1
	}
	return 0
}
