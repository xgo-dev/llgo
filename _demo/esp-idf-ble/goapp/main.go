// BLE central acceptance probe for the llgo ESP-IDF pipeline (ESP32-S31).
//
// Built with: llgo build -target esp32s31-idf -o ../idfproj/main/llgo_out.o ./goapp
// The relocatable object is linked into the IDF firmware by the main component;
// app_main calls the generated C entry, which hands control to the llgo fiber
// scheduler (runtime.RunMain).
//
// The probe drives the whole pkg/esp/ble surface in order: bring-up and sync,
// one scan abandoned through Cancel, the scan that finds the Bumble peer, a
// connect, the MTU exchange, GATT discovery (16-bit and 128-bit services),
// read, write with read-back, a notification subscription with two deliveries,
// unsubscription, and the disconnect. Every step prints one `[go] ble ...` line;
// `[go] ble PASS` is the acceptance gate.
package main

import (
	_ "unsafe" // for go:linkname

	"github.com/xgo-dev/llgo/pkg/esp/ble"
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
	// peerName is the local name the host-side Bumble peer advertises
	// (idfproj/host/bumble_peer.py).
	peerName = "llgo-peer"

	// The peer's attribute table (bumble_peer.py): a 16-bit echo service with
	// an echo and a notify characteristic, and a 128-bit service with a second
	// echo characteristic. The UUIDs are parsed from their text forms, which is
	// what a program does; the handles come from the discovery.
	echoSvcText = "fff0"
	echoChrText = "fff1"
	notifyText  = "fff2"
	wideSvcText = "6c6c676f-1111-2222-3344-556677889900"
	wideChrText = "6c6c676f-1111-2222-3344-556677889901"

	// stepWaitMs is the bound of one event wait inside a retry loop, and
	// stepTries turns it into the phase's overall budget.
	stepWaitMs = 500

	scanTries       = 20 // 10 s to find the peer
	connectTries    = 24 // 12 s to connect
	mtuTries        = 10 // 5 s for the MTU exchange the glue starts on connect
	disconnectTries = 10 // 5 s to see the disconnection
	notifyTries     = 10 // 5 s per notification
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

// decI32 formats a signed value in decimal (the RSSI is negative in every real
// scan report, and printing it through uint32 would turn -50 into 206).
func decI32(v int32) string {
	if v < 0 {
		return "-" + dec32(uint32(-v))
	}
	return dec32(uint32(v))
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

// writeMode names the ATT write procedure the package picks for a
// characteristic: 1 = write request (acknowledged), 2 = write command. The rule
// is the package's (Write prefers the acknowledged form when the peer declares
// both); printing it makes the log say which procedure the write below used - a
// with-response write can only complete through the peer's ATT Write Response.
func writeMode(chr ble.Characteristic) uint32 {
	if chr.Properties&ble.PropWrite != 0 {
		return 1
	}
	if chr.Properties&ble.PropWriteNoRsp != 0 {
		return 2
	}
	return 0
}

// mustUUID parses a UUID constant of the peer's attribute table. A parse error
// is a probe bug, not a device condition, so it stops the run.
func mustUUID(text string) ble.UUID {
	u, err := ble.ParseUUID(text)
	if err != nil {
		println("[go] ble FAIL uuid " + text)
	}
	return u
}

func main() {
	fixStreams()
	println("[go] ble: start")
	println("[go] ble backend = " + ble.Backend())

	// Subscribe before Init: the sync event waits in the C mailbox until the
	// poller starts, but subscribing first removes the question entirely.
	sub := ble.Subscribe(ble.EvAdv, ble.EvScanDone, ble.EvConnect,
		ble.EvDisconnect, ble.EvError, ble.EvReset, ble.EvMtu)
	defer sub.Close()

	if err := ble.Init(); err != nil {
		println("[go] ble init err=" + errText(err))
		println("[go] ble FAIL init")
		return
	}
	println("[go] ble init ok")
	ble.PrintVersion()
	heapMark("init")

	// Phase 1: cancel. A scan is started and abandoned before any result;
	// the cancelled procedure reports nothing, so what this phase proves is
	// that the command is accepted and the driver is usable afterwards.
	op, err := ble.StartScan(ble.ScanOptions{IntervalMs: 50, WindowMs: 50})
	if err != nil {
		println("[go] ble scan start err=" + errText(err))
		println("[go] ble FAIL cancel")
		return
	}
	if err := ble.Cancel(op); err != nil {
		println("[go] ble cancel err=" + errText(err))
		println("[go] ble FAIL cancel")
		return
	}
	println("[go] ble scan cancel ok op=" + dec32(uint32(op)))

	// Phase 2: find the peer. Duplicate filtering is off on purpose: both the
	// advertising payload and the RSSI of every report then reach the probe,
	// which is what makes the printed numbers meaningful.
	op, err = ble.StartScan(ble.ScanOptions{FilterDuplicates: false})
	if err != nil {
		println("[go] ble scan start err=" + errText(err))
		println("[go] ble FAIL scan")
		return
	}
	println("[go] ble scan start op=" + dec32(uint32(op)))
	heapMark("scan start")

	var peer ble.Peer
	found := false
	for i := 0; i < scanTries && !found; i++ {
		ev, err := sub.Next(stepWaitMs)
		if err != nil {
			continue // nothing this half second
		}
		switch ev.Kind {
		case ble.EvAdv:
			name := ble.AdvName(ev.AdvData())
			if name != peerName {
				continue
			}
			peer = ev.Peer()
			println("[go] ble scan found name=" + name +
				" addr=" + ble.AddrString(ev.Addr) +
				" addrtype=" + dec32(uint32(ev.AddrType)) +
				" rssi=" + decI32(int32(ev.Rssi)) +
				" advlen=" + dec32(uint32(ev.AdvLen)))
			found = true
		case ble.EvError:
			println("[go] ble scan err=" + errText(ble.ErrStatus(ev.Status)))
		case ble.EvScanDone:
			println("[go] ble scan ended op=" + dec32(uint32(ev.OpID)) +
				" reason=" + dec32(uint32(ev.Status)))
		}
	}
	if !found {
		println("[go] ble FAIL scan")
		return
	}
	if err := ble.StopScan(); err != nil {
		println("[go] ble scan stop err=" + errText(err))
		println("[go] ble FAIL scan")
		return
	}
	println("[go] ble scan stop ok")

	// Phase 3: connect. The connect completes on the controller's own
	// schedule; the event carries both the operation and the connection id.
	op, err = ble.Connect(peer, ble.ConnectOptions{TimeoutMs: 8000})
	if err != nil {
		println("[go] ble connect err=" + errText(err))
		println("[go] ble FAIL connect")
		return
	}
	println("[go] ble connect start op=" + dec32(uint32(op)))
	var conn ble.ConnID
	connected := false
	for i := 0; i < connectTries && !connected; i++ {
		ev, err := sub.Next(stepWaitMs)
		if err != nil {
			continue
		}
		switch ev.Kind {
		case ble.EvConnect:
			if ev.OpID != op {
				continue
			}
			if ev.Status != 0 {
				println("[go] ble connect status=" + errText(ble.ErrStatus(ev.Status)))
			} else {
				conn = ev.ConnID
				println("[go] ble connect ok conn=" + dec32(uint32(conn)))
				connected = true
			}
		case ble.EvError:
			println("[go] ble connect err=" + errText(ble.ErrStatus(ev.Status)))
		case ble.EvScanDone:
			println("[go] ble scan ended op=" + dec32(uint32(ev.OpID)) +
				" reason=" + dec32(uint32(ev.Status)))
		}
	}
	if !connected {
		println("[go] ble FAIL connect")
		return
	}
	heapMark("connected")

	// Phase 4: the ATT MTU. The glue asks for the exchange as soon as the
	// connection comes up, so EvMtu reports the negotiated value; the MTU query
	// must agree with the event.
	mtu := int32(0)
	for i := 0; i < mtuTries && mtu == 0; i++ {
		ev, err := sub.Next(stepWaitMs)
		if err != nil {
			continue
		}
		if ev.Kind == ble.EvMtu && ev.ConnID == conn {
			mtu = ev.Status
		}
	}
	if mtu == 0 {
		println("[go] ble FAIL mtu")
		return
	}
	apiMtu, err := ble.MTU(conn)
	if err != nil {
		println("[go] ble mtu err=" + errText(err))
		println("[go] ble FAIL mtu")
		return
	}
	if int32(apiMtu) != mtu {
		println("[go] ble FAIL mtu event=" + dec32(uint32(mtu)) +
			" api=" + dec32(uint32(apiMtu)))
		return
	}
	println("[go] ble mtu=" + dec32(uint32(mtu)))

	// Phase 5: discovery. Services() reports the peer's primary services; the
	// probe looks its two up by UUID (one 16-bit, one 128-bit) and then asks
	// for the characteristics of each.
	echoSvcUUID := mustUUID(echoSvcText)
	wideSvcUUID := mustUUID(wideSvcText)
	echoChrUUID := mustUUID(echoChrText)
	notifyChrUUID := mustUUID(notifyText)
	wideChrUUID := mustUUID(wideChrText)

	svcs, err := ble.Services(conn)
	if err != nil {
		println("[go] ble services err=" + errText(err))
		println("[go] ble FAIL services")
		return
	}
	println("[go] ble services count=" + dec32(uint32(len(svcs))))

	var echoSvc, wideSvc ble.Service
	echoFound, wideFound := false, false
	for _, s := range svcs {
		switch s.UUID {
		case echoSvcUUID:
			echoSvc, echoFound = s, true
			println("[go] ble echo svc uuid=" + s.UUID.String() +
				" handle=" + dec32(uint32(s.Handle)) +
				" end=" + dec32(uint32(s.End)))
		case wideSvcUUID:
			wideSvc, wideFound = s, true
			println("[go] ble wide svc uuid=" + s.UUID.String() +
				" handle=" + dec32(uint32(s.Handle)) +
				" end=" + dec32(uint32(s.End)))
		}
	}
	if !echoFound || !wideFound {
		println("[go] ble FAIL services echo=" + dec32(boolTo32(echoFound)) +
			" wide=" + dec32(boolTo32(wideFound)))
		return
	}

	echoChars, err := ble.Characteristics(conn, echoSvc.UUID)
	if err != nil {
		println("[go] ble chars err=" + errText(err))
		println("[go] ble FAIL chars")
		return
	}
	println("[go] ble echo chrs=" + dec32(uint32(len(echoChars))))
	var echoChr, notifyChr ble.Characteristic
	echoChrFound, notifyChrFound := false, false
	for _, c := range echoChars {
		println("[go] ble echo chr uuid=" + c.UUID.String() +
			" handle=" + dec32(uint32(c.ValueHandle)) +
			" props=" + dec32(uint32(c.Properties)) +
			" read=" + dec32(boolTo32(c.CanRead())) +
			" write=" + dec32(boolTo32(c.CanWrite())) +
			" notify=" + dec32(boolTo32(c.CanNotify())))
		switch c.UUID {
		case echoChrUUID:
			echoChr, echoChrFound = c, true
		case notifyChrUUID:
			notifyChr, notifyChrFound = c, true
		}
	}
	if !echoChrFound || !notifyChrFound {
		println("[go] ble FAIL chars echo=" + dec32(boolTo32(echoChrFound)) +
			" notify=" + dec32(boolTo32(notifyChrFound)))
		return
	}
	heapMark("discovered")

	// Phase 6: read, then write and read back. The first read reports the
	// peer's initial value (empty in bumble_peer.py); the read after the write
	// has to return the probe's own bytes, which is the write's proof.
	before, err := ble.Read(conn, echoChr)
	if err != nil {
		println("[go] ble read err=" + errText(err))
		println("[go] ble FAIL read")
		return
	}
	println("[go] ble read len=" + dec32(uint32(len(before))))

	payload := "ping-00"
	if err := ble.Write(conn, echoChr, []byte(payload)); err != nil {
		println("[go] ble write err=" + errText(err))
		println("[go] ble FAIL write")
		return
	}
	println("[go] ble write mode=" + dec32(writeMode(echoChr)) +
		" len=" + dec32(uint32(len(payload))))
	after, err := ble.Read(conn, echoChr)
	if err != nil {
		println("[go] ble read back err=" + errText(err))
		println("[go] ble FAIL write")
		return
	}
	println("[go] ble read back len=" + dec32(uint32(len(after))) +
		" data=" + string(after))
	if string(after) != payload {
		println("[go] ble FAIL write read-back mismatch")
		return
	}

	// Phase 7: notifications. SubscribeChar writes the CCCD; every echo write
	// after that makes the peer push the same bytes back, and the subscription
	// delivers them through the mailbox.
	nsub, err := ble.SubscribeChar(conn, notifyChr)
	if err != nil {
		println("[go] ble subscribe err=" + errText(err))
		println("[go] ble FAIL subscribe")
		return
	}
	println("[go] ble subscribe ok handle=" + dec32(uint32(notifyChr.ValueHandle)))
	for i := 0; i < 2; i++ {
		payload := "ping-0" + dec32(uint32(i+1))
		if err := ble.Write(conn, echoChr, []byte(payload)); err != nil {
			println("[go] ble write err=" + errText(err))
			println("[go] ble FAIL notify")
			return
		}
		println("[go] ble write mode=" + dec32(writeMode(echoChr)) +
			" len=" + dec32(uint32(len(payload))))
		got := false
		for t := 0; t < notifyTries && !got; t++ {
			ev, err := nsub.Next(stepWaitMs)
			if err != nil {
				continue
			}
			if ev.Kind != ble.EvNotify {
				continue
			}
			println("[go] ble notify n=" + dec32(uint32(i+1)) +
				" handle=" + dec32(uint32(ev.AttHandle)) +
				" ind=" + dec32(uint32(ev.Indication)) +
				" len=" + dec32(uint32(ev.DataLen)) +
				" data=" + string(ev.DataBytes()))
			got = string(ev.DataBytes()) == payload
		}
		if !got {
			println("[go] ble FAIL notify " + dec32(uint32(i+1)))
			return
		}
	}
	heapMark("notified")
	if err := ble.Unsubscribe(nsub); err != nil {
		println("[go] ble unsubscribe err=" + errText(err))
		println("[go] ble FAIL unsubscribe")
		return
	}
	println("[go] ble unsubscribe ok")

	// Phase 8: the unacknowledged write procedure. WriteWithoutResponse sends
	// an ATT Write Command - no response, so the call reports the submission
	// only - and the read-back proves the peer applied it in order.
	nrsp := "norsp"
	if err := ble.WriteWithoutResponse(conn, echoChr, []byte(nrsp)); err != nil {
		println("[go] ble write nrsp err=" + errText(err))
		println("[go] ble FAIL nrsp")
		return
	}
	println("[go] ble write nrsp mode=2 len=" + dec32(uint32(len(nrsp))))
	nrspBack, err := ble.Read(conn, echoChr)
	if err != nil {
		println("[go] ble read back nrsp err=" + errText(err))
		println("[go] ble FAIL nrsp")
		return
	}
	println("[go] ble read back nrsp len=" + dec32(uint32(len(nrspBack))) +
		" data=" + string(nrspBack))
	if string(nrspBack) != nrsp {
		println("[go] ble FAIL nrsp read-back mismatch")
		return
	}

	// Phase 9: the 128-bit path. Same read/write/read-back against the
	// characteristic whose UUID is a vendor 128-bit value.
	wideChars, err := ble.Characteristics(conn, wideSvc.UUID)
	if err != nil {
		println("[go] ble chars128 err=" + errText(err))
		println("[go] ble FAIL chars128")
		return
	}
	println("[go] ble wide chrs=" + dec32(uint32(len(wideChars))))
	wideChrFound := false
	var wideChr ble.Characteristic
	for _, c := range wideChars {
		if c.UUID == wideChrUUID {
			wideChr, wideChrFound = c, true
			println("[go] ble wide chr uuid=" + c.UUID.String() +
				" handle=" + dec32(uint32(c.ValueHandle)) +
				" props=" + dec32(uint32(c.Properties)))
		}
	}
	if !wideChrFound {
		println("[go] ble FAIL chars128")
		return
	}
	if _, err := ble.Read(conn, wideChr); err != nil {
		println("[go] ble wide read err=" + errText(err))
		println("[go] ble FAIL wide")
		return
	}
	widePayload := "u128"
	if err := ble.Write(conn, wideChr, []byte(widePayload)); err != nil {
		println("[go] ble wide write err=" + errText(err))
		println("[go] ble FAIL wide")
		return
	}
	println("[go] ble wide write mode=" + dec32(writeMode(wideChr)) +
		" len=" + dec32(uint32(len(widePayload))))
	wideBack, err := ble.Read(conn, wideChr)
	if err != nil {
		println("[go] ble wide read back err=" + errText(err))
		println("[go] ble FAIL wide")
		return
	}
	println("[go] ble wide read back len=" + dec32(uint32(len(wideBack))) +
		" data=" + string(wideBack))
	if string(wideBack) != widePayload {
		println("[go] ble FAIL wide read-back mismatch")
		return
	}

	// Phase 10: disconnect. The event's Status is the HCI reason the controller
	// reports (NimBLE tags it with its HCI base).
	if err := ble.Disconnect(conn); err != nil {
		println("[go] ble disconnect err=" + errText(err))
		println("[go] ble FAIL disconnect")
		return
	}
	gone := false
	for i := 0; i < disconnectTries && !gone; i++ {
		ev, err := sub.Next(stepWaitMs)
		if err != nil {
			continue
		}
		switch ev.Kind {
		case ble.EvDisconnect:
			if ev.ConnID != conn {
				continue
			}
			println("[go] ble disconnect done reason=" + dec32(uint32(ev.Status)))
			gone = true
		case ble.EvError:
			println("[go] ble disconnect err=" + errText(ble.ErrStatus(ev.Status)))
		}
	}
	if !gone {
		println("[go] ble FAIL disconnect")
		return
	}
	heapMark("disconnected")
	println("[go] ble PASS")

	// Phase 11: shutdown. nimble_port_stop waits for the host task, so this is
	// the one blocking call after bring-up; it runs after the PASS gate so a
	// failure here cannot hide the central's result.
	if err := ble.Deinit(); err != nil {
		println("[go] ble deinit err=" + errText(err))
		return
	}
	println("[go] ble deinit ok")
	heapMark("deinit")
	println("[go] ble done")
}

// boolTo32 renders a flag as 0/1 for the log.
func boolTo32(v bool) uint32 {
	if v {
		return 1
	}
	return 0
}
