// BLE central acceptance probe for the llgo ESP-IDF pipeline (ESP32-S31).
//
// Built with: llgo build -target esp32s31-idf -o ../idfproj/main/llgo_out.o ./goapp
// The relocatable object is linked into the IDF firmware by the main component;
// app_main calls the generated C entry, which hands control to the llgo fiber
// scheduler (runtime.RunMain).
//
// The probe drives the whole P-A surface of pkg/esp/ble in order: bring-up and
// sync, one scan abandoned through Cancel, the scan that finds the Bumble peer,
// a connect, a disconnect, and the stack shutdown. Every step prints one
// `[go] ble ...` line; `[go] ble PASS` is the acceptance gate.
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

	// stepWaitMs is the bound of one event wait inside a retry loop, and
	// stepTries turns it into the phase's overall budget.
	stepWaitMs = 500

	scanTries       = 20 // 10 s to find the peer
	connectTries    = 24 // 12 s to connect
	disconnectTries = 10 // 5 s to see the disconnection
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

func main() {
	fixStreams()
	println("[go] ble: start")
	println("[go] ble backend = " + ble.Backend())

	// Subscribe before Init: the sync event waits in the C mailbox until the
	// poller starts, but subscribing first removes the question entirely.
	sub := ble.Subscribe(ble.EvAdv, ble.EvScanDone, ble.EvConnect,
		ble.EvDisconnect, ble.EvError, ble.EvReset)
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

	// Phase 4: disconnect. The event's Status is the HCI reason the controller
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

	// Phase 5: shutdown. nimble_port_stop waits for the host task, so this is
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
