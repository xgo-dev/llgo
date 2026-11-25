// ESP-NOW probe for "llgo as an ESP-IDF component" on ESP32-S31.
//
// Built with: llgo build -target esp32s31-idf -o ../idfproj/main/llgo_out.o ./goapp
// The relocatable object is linked into the IDF firmware by the main
// component; app_main calls the generated C entry, which hands control to
// the llgo fiber scheduler (runtime.RunMain).
//
// The probe drives the pkg/esp/espnow surface and nothing else, in the
// documented initialization order (WiFi first, ESP-NOW second - this
// package never starts the radio itself):
//
//	wifi.Setup -> SetModeSTA -> Start, then espnow.Init
//
// From there: the protocol version and the device's own MAC, the peer-list
// lifecycle (add / duplicate / count / known / delete with the driver's
// error codes), the send paths (broadcast with the asynchronous
// SendResult.Wait confirmation, a burst of three to exercise the per-MAC
// FIFO, the synchronous refusals - unknown peer, empty payload, oversize
// payload), the encryption entry points (PMK, per-peer LMK), SetChannel,
// and a full Deinit -> re-Init cycle. A receive handler is registered up
// front and prints any frame that arrives.
//
// Under esp-emu every control-plane call is real; there is no second
// ESP-NOW device to talk to, so no frame is ever RECEIVED there and true
// delivery semantics stay hardware items (the demo README records the
// measured emulator fidelity). Every wait is bounded; every unexpected
// result prints a FAIL line and returns, so a broken step never looks like
// a hang.
package main

import (
	"github.com/xgo-dev/llgo/pkg/esp/esperr"
	"github.com/xgo-dev/llgo/pkg/esp/espnow"
	"github.com/xgo-dev/llgo/pkg/esp/wifi"
)

// errText renders an esperr error as its pipeline form ("ESP_ERR_ESPNOW_ARG
// (12390)"); nil as "ok". The names come from espnow.ErrName, filled in by
// the package's errOf.
func errText(err error) string {
	if err == nil {
		return "ok"
	}
	return err.Error()
}

// want checks one probe step: on an unexpected code it prints a FAIL line
// and reports false (the caller returns), so the log reads as a checklist.
func want(stage string, err error, code int32) bool {
	if err == nil {
		if code == 0 {
			println("[go] espnow", stage, "ok")
			return true
		}
		println("[go] espnow", stage, "FAIL: ok, want code", int(code))
		return false
	}
	if e, ok := err.(*esperr.Error); ok && e.Code == code {
		println("[go] espnow", stage, "ok:", e.Name)
		return true
	}
	println("[go] espnow", stage, "FAIL:", errText(err), "- want code", int(code))
	return false
}

func main() {
	defer wifi.Finish() // flush C stdio on every exit path
	println("[go] hello from llgo on FreeRTOS/ESP32-S31 (ESP-NOW probe)")

	// Scheduler sanity: a channel roundtrip across two goroutines before
	// anything else, so a scheduler break never masquerades as a radio
	// break.
	roundtrip := make(chan int, 1)
	go func() { roundtrip <- 42 }()
	if got := <-roundtrip; got == 42 {
		println("[go] goroutine channel roundtrip ok")
	} else {
		println("[go] espnow FAIL scheduler")
		return
	}

	// ---- radio bring-up: ESP-NOW rides a started WiFi driver ----
	println("[go] wifi: setup ...")
	if !want("wifi setup", wifi.Setup(), 0) {
		println("[go] espnow FAIL wifi setup")
		return
	}
	if !want("wifi events", wifi.EventsRegister(), 0) {
		println("[go] espnow FAIL wifi events")
		return
	}
	if !want("wifi mode", wifi.SetModeSTA(), 0) {
		println("[go] espnow FAIL wifi mode")
		return
	}
	if !want("wifi start", wifi.Start(), 0) {
		println("[go] espnow FAIL wifi start")
		return
	}
	staStart := wifi.Subscribe(wifi.EvStaStart)
	if _, err := staStart.Next(3000); err != nil {
		println("[go] espnow FAIL: no EvStaStart within 3s:", errText(err))
		return
	}
	staStart.Close()
	println("[go] espnow wifi ready (station started, unassociated)")

	// ---- the receive handler goes in before any traffic can flow ----
	espnow.SetRecvHandler(func(from [6]uint8, data []byte) {
		n := len(data)
		if n > 32 {
			n = 32
		}
		println("[go] espnow recv from", espnow.MACString(from), "len", len(data), "payload", string(data[:n]))
	})

	// ---- bring-up and identity ----
	if !want("init", espnow.Init(), 0) {
		println("[go] espnow FAIL init")
		return
	}
	if v, err := espnow.Version(); err == nil {
		println("[go] espnow protocol version =", int(v))
	} else {
		println("[go] espnow version query:", errText(err))
	}
	println("[go] espnow own sta mac =", espnow.MACString(espnow.OwnMAC(espnow.IfSTA)))

	// ---- peer-list lifecycle ----
	var stranger = [6]uint8{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc}
	if !want("add broadcast peer", espnow.AddPeer(espnow.PeerConfig{MAC: espnow.BroadcastMAC}), 0) {
		println("[go] espnow FAIL add broadcast peer")
		return
	}
	want("add broadcast peer again", espnow.AddPeer(espnow.PeerConfig{MAC: espnow.BroadcastMAC}), 0x306B) // ESP_ERR_ESPNOW_EXIST
	if !espnow.PeerIsKnown(espnow.BroadcastMAC) {
		println("[go] espnow FAIL: broadcast peer not known after add")
		return
	}
	if espnow.PeerIsKnown(stranger) {
		println("[go] espnow FAIL: stranger reported known")
		return
	}
	println("[go] espnow peer is known: bcast yes, stranger no")
	if n, err := espnow.PeerCount(); err != nil || n.Total != 1 || n.Encrypted != 0 {
		println("[go] espnow FAIL: peer count =", n.Total, n.Encrypted, errText(err))
		return
	}
	println("[go] espnow peer count = 1 total, 0 encrypted")

	// Encryption entry points. An encrypted peer without a key is refused
	// client-side (the driver would silently send plaintext); with a key it
	// consumes one encrypted slot.
	want("add peer, encrypt without key", espnow.AddPeer(espnow.PeerConfig{MAC: stranger, Encrypt: true}), 0x3066) // ESP_ERR_ESPNOW_ARG
	var lmk [16]uint8
	for i := range lmk {
		lmk[i] = uint8(i + 1)
	}
	if !want("add encrypted peer", espnow.AddPeer(espnow.PeerConfig{MAC: stranger, Encrypt: true, LMK: lmk}), 0) {
		println("[go] espnow FAIL add encrypted peer")
		return
	}
	if !want("set pmk", espnow.SetPMK(lmk), 0) {
		println("[go] espnow FAIL set pmk")
		return
	}
	if !want("rotate peer lmk", espnow.SetLMK(stranger, lmk), 0) {
		println("[go] espnow FAIL rotate peer lmk")
		return
	}
	if n, err := espnow.PeerCount(); err != nil || n.Total != 2 || n.Encrypted != 1 {
		println("[go] espnow FAIL: encrypted peer not counted:", n.Total, n.Encrypted, errText(err))
		return
	}
	println("[go] espnow peer count = 2 total, 1 encrypted")
	want("delete stranger once", espnow.DeletePeer(stranger), 0)
	want("delete stranger twice", espnow.DeletePeer(stranger), 0x3069) // ESP_ERR_ESPNOW_NOT_FOUND

	// ---- send paths: the synchronous refusals first ----
	want("send to unknown peer", must(espnow.Send(stranger, []byte("x"))), 0x3069) // ESP_ERR_ESPNOW_NOT_FOUND
	if _, err := espnow.Send(espnow.BroadcastMAC, nil); err == nil || espCode(err) != 0x102 {
		println("[go] espnow FAIL: empty payload accepted:", errText(err))
		return
	}
	println("[go] espnow send empty payload: ESP_ERR_INVALID_ARG")
	if _, err := espnow.Send(espnow.BroadcastMAC, make([]byte, espnow.MaxDataLen+1)); err == nil || espCode(err) != 0x102 {
		println("[go] espnow FAIL: oversize payload accepted:", errText(err))
		return
	}
	println("[go] espnow send oversize payload: ESP_ERR_INVALID_ARG")

	// ---- broadcast ping with the asynchronous confirmation ----
	res, err := espnow.Broadcast([]byte("llgo-espnow-ping"))
	if err != nil {
		println("[go] espnow FAIL broadcast:", errText(err))
		return
	}
	status, err := res.Wait(1000)
	if err != nil {
		println("[go] espnow FAIL broadcast confirm:", errText(err))
		return
	}
	if !status.OK() {
		println("[go] espnow FAIL broadcast confirm:", status.String())
		return
	}
	println("[go] espnow broadcast confirm: success")

	// ---- a burst: three sends to one MAC resolve oldest-first through the
	// pump's per-MAC FIFO ----
	for i := 0; i < 3; i++ {
		r, err := espnow.Broadcast([]byte("burst"))
		if err != nil {
			println("[go] espnow FAIL burst send:", errText(err))
			return
		}
		s, err := r.Wait(1000)
		if err != nil {
			println("[go] espnow FAIL burst confirm:", errText(err))
			return
		}
		println("[go] espnow burst", i, "confirm:", s.String())
	}
	println("[go] espnow dropped =", int(espnow.Dropped()))

	// ---- SetChannel: honest probe, no PASS/FAIL (the answer depends on the
	// mode/state the driver is in; two nodes just call it with the same
	// channel while unconnected) ----
	println("[go] espnow set channel 6:", errText(espnow.SetChannel(6)))

	// ---- full lifecycle: deinit, the NOT_INIT refusals, re-init ----
	if !want("deinit", espnow.Deinit(), 0) {
		println("[go] espnow FAIL deinit")
		return
	}
	want("send after deinit", must(espnow.Send(espnow.BroadcastMAC, []byte("x"))), 0x3065) // ESP_ERR_ESPNOW_NOT_INIT
	if !want("re-init", espnow.Init(), 0) {
		println("[go] espnow FAIL re-init")
		return
	}
	if !want("re-add broadcast peer", espnow.AddPeer(espnow.PeerConfig{MAC: espnow.BroadcastMAC}), 0) {
		println("[go] espnow FAIL re-add broadcast peer")
		return
	}
	res, err = espnow.Broadcast([]byte("after-reinit"))
	if err != nil {
		println("[go] espnow FAIL broadcast after re-init:", errText(err))
		return
	}
	status, err = res.Wait(1000)
	if err != nil {
		println("[go] espnow FAIL confirm after re-init:", errText(err))
		return
	}
	println("[go] espnow confirm after re-init:", status.String())

	println("[go] espnow PASS")
	println("[go] espnow done")
}

// must flattens Send's (result, error) into the error alone, for the refusal
// probes (a refusal leaves no result to wait on).
func must(_ *espnow.SendResult, err error) error { return err }

// espCode extracts the pipeline code of an esperr error (0 for nil).
func espCode(err error) int32 {
	if e, ok := err.(*esperr.Error); ok {
		return e.Code
	}
	return 0
}
