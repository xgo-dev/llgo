package main

import (
	"github.com/xgo-dev/llgo/pkg/esp/bt"
	"github.com/xgo-dev/llgo/pkg/esp/gpio"
)

// A2DP sink: the board's codec wiring (ESP32-S31-Function-CoreBoard-1,
// schematic-verified). MCLK (GPIO52) goes through the A2DPConfig.MCLKPin
// field - the ES8311 needs it; zero/negative would leave the codec's PLL
// without its system clock.
const (
	a2dpName = "llgo-audio"
	bclkPin  = 53 // I2S SCLK -> ES8311 SCLK
	wsPin    = 55 // I2S LRCK -> ES8311 LRCK
	doutPin  = 56 // I2S DSDIN -> ES8311 DSDIN (DAC data)
	mclkPin  = 52 // I2S MCLK -> ES8311 MCLK (256 x fs)

	dmaDescNum  = 12 // ~139 ms of jitter absorber at 44.1 kHz
	dmaFrameNum = 511
)

func btInit() {
	sub := bt.Subscribe()
	defer sub.Close()
	if err := bt.Init(); err != nil {
		println("[go] bt FAIL init err =", errText(err))
		return
	}
	if err := bt.SetName(a2dpName); err != nil {
		println("[go] bt FAIL name err =", errText(err))
		return
	}
	if err := bt.SetDiscoverable(true); err != nil {
		println("[go] bt FAIL discoverable err =", errText(err))
		return
	}
	println("[go] bt init ok backend =", bt.Backend(), "name =", a2dpName)
	heapLedger("bt-init")
}

func a2dpInit() {
	if !bt.Ready() {
		println("[go] a2dp FAIL no bt stack")
		return
	}
	if err := bt.A2DPEnable(bt.A2DPConfig{
		BclkPin: bclkPin, WsPin: wsPin, DoutPin: doutPin, MCLKPin: mclkPin,
		DmaDescNum: dmaDescNum, DmaFrameNum: dmaFrameNum,
	}); err != nil {
		println("[go] a2dp FAIL enable err =", errText(err))
		return
	}
	println("[go] a2dp enable ok bclk =", bclkPin, "ws =", wsPin, "dout =", doutPin,
		"mclk =", mclkPin, "dma =", dmaDescNum, "x", dmaFrameNum)
	heapLedger("a2dp-init")
}

// a2dpLoop logs the profile's events, prints the byte counters' deltas every
// 2 s (the data-plane verdict: at 44.1 kHz stereo the written delta tracks
// the total delta at ~352800 B / 2 s), and after the codec self-test has run
// it follows the link with the amp pin.
func a2dpLoop() {
	amp := gpio.Pin(paPin)
	amp.SetDirection(gpio.Output, gpio.PullNone)
	amp.SetLevel(false)

	sub := bt.Subscribe()
	defer sub.Close()
	var lt, lw, ld uint32
	secs := 0
	for {
		ev, err := sub.Next(2000)
		if err != nil {
			secs += 2
			tot, wri, drop := bt.A2DPStats()
			if tot < lt || wri < lw { // counters reset (self-test re-enable)
				println("[go] a2dp counters reset (self-test re-enable)")
			}
			// Quiet by default: print on a change of the byte counters and
			// a one-line heartbeat every 10 s, so an idle link does not
			// flood the console at the 2 s poll rate.
			if tot != lt || wri != lw || drop != ld || secs%10 == 0 {
				println("[go] a2dp t =", secs, "s tot+ =", tot-lt, "w+ =", wri-lw,
					"drop+ =", drop-ld, "| tot =", tot, "w =", wri, "d =", drop)
			}
			lt, lw, ld = tot, wri, drop
			continue
		}
		switch ev.Kind {
		case int32(bt.EvA2DPConnected):
			a2dpConnected = true
			println("[go] a2dp connected addr =", ev.AddrString())
			if selftestDone {
				amp.SetLevel(true)
				println("[go] a2dp amp on (codec ready, link up)")
			} else {
				println("[go] a2dp amp stays off (codec not ready yet)")
			}
		case int32(bt.EvA2DPAudioCfg):
			println("[go] a2dp AUDIO-CFG rate =", int(ev.Aux), "ch =", int(ev.Status))
		case int32(bt.EvA2DPVolume):
			println("[go] a2dp volume =", int(ev.Aux), "(peer)")
		case int32(bt.EvA2DPDisconnected):
			a2dpConnected = false
			println("[go] a2dp disconnected rsn =", int(ev.Status), "addr =", ev.AddrString())
			if selftestDone {
				amp.SetLevel(false)
			}
		case int32(bt.EvOverflow):
			println("[go] a2dp overflow ctrl =", int(ev.Status), "data =", int(ev.Aux))
		}
	}
}
