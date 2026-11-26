// BLE peripheral acceptance probe for the llgo ESP-IDF pipeline (ESP32-S31).
//
// Built with: llgo build -target esp32s31-idf -o ../idfproj/main/llgo_out.o ./goapp
// The relocatable object is linked into the IDF firmware by the main component;
// app_main calls the generated C entry, which hands control to the llgo fiber
// scheduler (runtime.RunMain).
//
// The probe drives the peripheral half of pkg/esp/ble in order: bring-up and
// sync, the pairing posture, the local GATT server registration (one 16-bit
// service and one 128-bit one, committed once through CommitServices), the
// initial characteristic value, the advertisement named "llgo-periph", the
// central's connection, its subscription, its writes echoed back as
// notifications, and the disconnection. Every step prints one
// `[go] blep ...` line; `[go] blep PASS` is the acceptance gate.
//
// The host-side counterpart is idfproj/host/bumble_central.py, a Bumble
// central that scans for this device, discovers the same table, reads,
// subscribes, writes and disconnects - its own log is the cross-evidence
// that the connection was real on both ends.
package main

import (
	_ "unsafe" // for go:linkname

	"github.com/xgo-dev/llgo/pkg/esp/ble"
	"github.com/xgo-dev/llgo/pkg/esp/esperr"
)

// heapPrint prints the IDF heap pools from the C side (main/demo_glue.c).
//
//go:linkname heapPrint C.llgo_demo_heap
func heapPrint(tag *byte, n int32)

const (
	// advName is the local name the probe advertises; the central scans for
	// exactly this string.
	advName = "llgo-periph"

	// The local attribute table, mirrored by the central script's discovery:
	// a 16-bit service with an echo and a notify characteristic, and a
	// 128-bit service with a second echo characteristic. The UUIDs are
	// parsed from their text forms, which is what a program does; the
	// handles come from LocalHandle after the commit.
	echoSvcText = "fff0"
	echoChrText = "fff1"
	notifyText  = "fff2"
	wideSvcText = "6c6c676f-2222-3333-4444-556677889900"
	wideChrText = "6c6c676f-2222-3333-4444-556677889901"

	// stepWaitMs is the bound of one event wait inside a retry loop, and
	// stepTries turns it into the phase's overall budget.
	stepWaitMs = 500

	connectTries    = 240 // 120 s for the central to find us and connect
	subscribeTries  = 240 // 120 s for the CCCD write to arrive (manual phone ops)
	subWatchTries   = 30  // 15 s per subscription/unsubscription notice
	disconnectTries = 240 // 120 s to see the disconnection (manual phone ops)
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

// mustUUID parses a UUID constant of the local table. A parse error is a
// probe bug, not a device condition, so it stops the run.
func mustUUID(text string) ble.UUID {
	u, err := ble.ParseUUID(text)
	if err != nil {
		println("[go] blep FAIL uuid " + text)
	}
	return u
}

func main() {
	println("[go] blep: start")
	println("[go] blep backend = " + ble.Backend())

	// Subscribe before Init: the sync event waits in the C mailbox until the
	// poller starts, but subscribing first removes the question entirely.
	sub := ble.Subscribe(ble.EvPeriphConnect, ble.EvPeriphDisconnect,
		ble.EvCharWrite, ble.EvSubscribed, ble.EvAdvDone,
		ble.EvServicesDone, ble.EvError, ble.EvReset)
	defer sub.Close()

	if err := ble.Init(); err != nil {
		println("[go] blep init err=" + errText(err))
		println("[go] blep FAIL init")
		return
	}
	println("[go] blep init ok")
	ble.PrintVersion()
	heapMark("init")

	// The pairing posture (SetSecurityParams): bond + just-works. The central
	// in this acceptance run does not pair, so this exercises the call and
	// records the posture - a provisioning stage would rely on it.
	if err := ble.SetSecurityParams(ble.SecurityParams{
		IoCap: ble.IoCapNoInputNoOutput,
		Bond:  true,
	}); err != nil {
		println("[go] blep security err=" + errText(err))
		println("[go] blep FAIL security")
		return
	}
	println("[go] blep security ok")

	// Phase 1: the local GATT server's registry, then the commit. The 16-bit
	// service carries the echo (read | write | write-without-response) and a
	// notify characteristic; the 128-bit service a second echo. Handle
	// budgets: 1 + 2 + 3 = 6 and 1 + 2 = 3, with headroom declared.
	echoSvc := mustUUID(echoSvcText)
	echoChr := mustUUID(echoChrText)
	notifyChrUUID := mustUUID(notifyText)
	wideSvc := mustUUID(wideSvcText)
	wideChrUUID := mustUUID(wideChrText)

	if err := ble.AddService(echoSvc, 8); err != nil {
		println("[go] blep addsvc err=" + errText(err))
		println("[go] blep FAIL addsvc")
		return
	}
	if err := ble.AddCharacteristic(echoSvc, echoChr,
		ble.PropRead|ble.PropWrite|ble.PropWriteNoRsp); err != nil {
		println("[go] blep addchr err=" + errText(err))
		println("[go] blep FAIL addchr")
		return
	}
	if err := ble.AddCharacteristic(echoSvc, notifyChrUUID, ble.PropNotify); err != nil {
		println("[go] blep addchr err=" + errText(err))
		println("[go] blep FAIL addchr")
		return
	}
	if err := ble.AddService(wideSvc, 4); err != nil {
		println("[go] blep addsvc err=" + errText(err))
		println("[go] blep FAIL addsvc")
		return
	}
	if err := ble.AddCharacteristic(wideSvc, wideChrUUID,
		ble.PropRead|ble.PropWrite); err != nil {
		println("[go] blep addchr err=" + errText(err))
		println("[go] blep FAIL addchr")
		return
	}
	println("[go] blep registry svcs=2 chrs=3")
	if err := ble.CommitServices(); err != nil {
		println("[go] blep commit err=" + errText(err))
		println("[go] blep FAIL commit")
		return
	}
	echoHandle, err := ble.LocalHandle(echoChr)
	if err != nil {
		println("[go] blep handle err=" + errText(err))
		println("[go] blep FAIL handle")
		return
	}
	notifyHandle, err := ble.LocalHandle(notifyChrUUID)
	if err != nil {
		println("[go] blep handle err=" + errText(err))
		println("[go] blep FAIL handle")
		return
	}
	wideHandle, err := ble.LocalHandle(wideChrUUID)
	if err != nil {
		println("[go] blep handle err=" + errText(err))
		println("[go] blep FAIL handle")
		return
	}
	println("[go] blep commit ok echo=" + dec32(uint32(echoHandle)) +
		" notify=" + dec32(uint32(notifyHandle)) +
		" wide=" + dec32(uint32(wideHandle)))
	heapMark("committed")

	// The initial value of the echo characteristic: what the central's first
	// read answers before anything is written.
	if err := ble.WriteCharValue(echoHandle, []byte("hello")); err != nil {
		println("[go] blep setvalue err=" + errText(err))
		println("[go] blep FAIL setvalue")
		return
	}
	println("[go] blep setvalue ok len=5")

	// Phase 2: advertise. The composition puts the name in the payload; the
	// outcome arrives as EvAdvDone.
	op, err := ble.StartAdvertising(ble.AdvParams{Name: advName, IntervalMs: 50})
	if err != nil {
		println("[go] blep adv err=" + errText(err))
		println("[go] blep FAIL adv")
		return
	}
	println("[go] blep adv start op=" + dec32(uint32(op)))
	advUp := false
	for i := 0; i < subWatchTries && !advUp; i++ {
		ev, err := sub.Next(stepWaitMs)
		if err != nil {
			continue
		}
		if ev.Kind == ble.EvAdvDone && ev.OpID == op {
			if ev.Status != 0 {
				println("[go] blep adv status=" + errText(ble.ErrStatus(ev.Status)))
			} else {
				advUp = true
			}
		}
	}
	if !advUp {
		println("[go] blep FAIL adv")
		return
	}
	println("[go] blep adv ok")
	heapMark("advertising")

	// Phase 3: the central connects. Its address arrives with the event.
	var conn ble.ConnID
	connected := false
	for i := 0; i < connectTries && !connected; i++ {
		ev, err := sub.Next(stepWaitMs)
		if err != nil {
			continue
		}
		switch ev.Kind {
		case ble.EvPeriphConnect:
			conn = ev.ConnID
			println("[go] blep peer connected conn=" + dec32(uint32(conn)) +
				" addr=" + ble.AddrString(ev.Addr) +
				" addrtype=" + dec32(uint32(ev.AddrType)))
			connected = true
		case ble.EvError:
			println("[go] blep err=" + errText(ble.ErrStatus(ev.Status)))
		}
	}
	if !connected {
		println("[go] blep FAIL connect")
		return
	}
	heapMark("connected")

	// Phase 4: the central subscribes to the notify characteristic (its CCCD
	// write arrives as EvSubscribed).
	subscribed := false
	for i := 0; i < subscribeTries && !subscribed; i++ {
		ev, err := sub.Next(stepWaitMs)
		if err != nil {
			continue
		}
		if ev.Kind == ble.EvSubscribed && ev.ConnID == conn && ev.AttHandle == notifyHandle {
			if ev.Status == 1 {
				println("[go] blep subscribed handle=" + dec32(uint32(notifyHandle)) +
					" cccd=1")
				subscribed = true
			} else {
				println("[go] blep subscribed cccd=" + dec32(uint32(ev.Status)))
			}
		}
	}
	if !subscribed {
		println("[go] blep FAIL subscribe")
		return
	}

	// Phase 5: the echo loop. Every write the central makes is reported as
	// EvCharWrite (the glue has already cached the value) and pushed back as
	// a notification on the notify characteristic. Two payloads: one
	// acknowledged write, one write command.
	echoed := 0
	idle := 0
	for echoed < 2 && idle < 30 { // 15 s budget for both writes
		ev, err := sub.Next(stepWaitMs)
		if err != nil {
			idle++
			continue
		}
		if ev.Kind == ble.EvCharWrite && ev.ConnID == conn && ev.AttHandle == echoHandle {
			echoed++
			idle = 0
			payload := string(ev.DataBytes())
			println("[go] blep write n=" + dec32(uint32(echoed)) +
				" handle=" + dec32(uint32(ev.AttHandle)) +
				" len=" + dec32(uint32(ev.DataLen)) +
				" data=" + payload)
			if err := ble.Notify(conn, notifyHandle, ev.DataBytes()); err != nil {
				println("[go] blep notify err=" + errText(err))
				println("[go] blep FAIL notify")
				return
			}
			println("[go] blep notify n=" + dec32(uint32(echoed)) +
				" handle=" + dec32(uint32(notifyHandle)) +
				" len=" + dec32(uint32(len(payload))))
			continue
		}
		if ev.Kind == ble.EvPeriphDisconnect {
			break // the central gave up early; the disconnect phase reports it
		}
	}
	if echoed < 2 {
		println("[go] blep FAIL echo")
		return
	}
	heapMark("echoed")

	// Phase 6: the central unsubscribes (CCCD 0x0000) and disconnects.
	unsubscribed := false
	gone := false
	for i := 0; i < subWatchTries+disconnectTries && !(unsubscribed && gone); i++ {
		ev, err := sub.Next(stepWaitMs)
		if err != nil {
			continue
		}
		switch ev.Kind {
		case ble.EvSubscribed:
			if ev.ConnID == conn && ev.AttHandle == notifyHandle && ev.Status == 0 {
				println("[go] blep unsubscribed cccd=0")
				unsubscribed = true
			}
		case ble.EvPeriphDisconnect:
			if ev.ConnID == conn {
				println("[go] blep disconnected conn=" + dec32(uint32(conn)) +
					" reason=" + ble.DisconnectReasonName(ev.Status))
				gone = true
			}
		}
	}
	if !unsubscribed || !gone {
		println("[go] blep FAIL teardown unsub=" + dec32(boolTo32(unsubscribed)) +
			" gone=" + dec32(boolTo32(gone)))
		return
	}
	heapMark("disconnected")
	println("[go] blep PASS")

	// Phase 7: shutdown. Deinit releases the stack; it runs after the PASS
	// gate so a failure here cannot hide the peripheral's result.
	if err := ble.Deinit(); err != nil {
		println("[go] blep deinit err=" + errText(err))
		return
	}
	println("[go] blep deinit ok")
	heapMark("deinit")
	println("[go] blep done")
}

// boolTo32 renders a flag as 0/1 for the log.
func boolTo32(v bool) uint32 {
	if v {
		return 1
	}
	return 0
}
