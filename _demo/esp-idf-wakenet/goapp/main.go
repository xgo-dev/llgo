// WakeNet acceptance probe for the llgo ESP-IDF pipeline (ESP32-S31).
//
// Built with: host/make-artifact.sh (llgo build -target esp32s31-idf) and
// then idf.py build in idfproj/. The relocatable object is linked into the
// IDF firmware by the main component; app_main calls the generated C entry,
// which hands control to the llgo fiber scheduler (runtime.RunMain).
//
// The probe drives the pkg/esp/wakenet surface on the test source - no
// microphone needed: the demo glue injects the esp-sr test app's "Hi, Lexin"
// utterance as the engine's input PCM - and answers the questions P0/P1 exist
// for: does the model load, does the detect task loop in real time next to
// the Go scheduler, and what does one detect call cost. Every step prints one
// `[go] wakenet ...` line; `[go] wakenet PASS` is the acceptance gate (model
// loaded, task ran ~10 s of audio through detect). Whether the emulator
// executes the model's own SIMD-heavy compute is a separate question the run
// log answers: with detections, end to end; without, the frames/avg/max
// numbers still prove the loop.
package main

import (

	"github.com/xgo-dev/llgo/pkg/esp/esperr"
	"github.com/xgo-dev/llgo/pkg/esp/wakenet"
)

const (
	// The probe runs ~12 s of audio through the engine (a 3.77 s utterance
	// replayed in a loop) and gates on the detect task having kept up.
	runMs         = 12000
	passFrames    = 300 // ~9.6 s of chunks at 32 ms - the loop ran, continuously
	statsEvery    = 1000
	heapMarkEvery = 4000
)

func main() {
	fixStreams()
	println("[go] wakenet: start")
	heapMark("boot")

	n := injectTestPcm()
	println("[go] wakenet test pcm samples = " + dec32(uint32(n)))

	err := wakenet.Init(wakenet.ModelConfig{})
	if err != nil {
		println("[go] wakenet FAIL init err=" + errText(err))
		return
	}
	println("[go] wakenet init ok model=" + wakenet.ModelName() +
		" chunk=" + dec32(uint32(wakenet.ChunkSamples())) +
		" rate=" + dec32(uint32(wakenet.SampleRate())))
	heapMark("init")

	if err := wakenet.Start(); err != nil {
		println("[go] wakenet FAIL start err=" + errText(err))
		return
	}
	println("[go] wakenet start ok")

	events := wakenet.Events()
	detections := 0
	errors := 0
	overflows := 0
	started := int32(nowMs())
	lastStats := started
	lastHeap := started

	for now := int32(nowMs()); now-started < runMs; now = int32(nowMs()) {
		select {
		case ev, ok := <-events:
			if !ok {
				println("[go] wakenet FAIL events closed")
				return
			}
			switch ev.Kind {
			case wakenet.EvDetected:
				detections++
				println("[go] wakenet DETECTED #" + dec32(uint32(detections)) +
					" state=" + dec32(uint32(ev.Code)) + " ts=" + dec32(uint32(ev.Timestamp)))
			case wakenet.EvError:
				errors++
				println("[go] wakenet error " + dec32(uint32(ev.Code)) + " ts=" +
					dec32(uint32(ev.Timestamp)))
			case wakenet.EvOverflow:
				overflows += int(ev.Code)
				println("[go] wakenet overflow " + dec32(uint32(ev.Code)))
			}
		default:
		}
		if now-lastStats >= statsEvery {
			lastStats = now
			f, avg, mx := wakenet.Stats()
			println("[go] wakenet frames=" + dec32(uint32(f)) + " avg_us=" +
				dec32(uint32(avg)) + " max_us=" + dec32(uint32(mx)) +
				" detections=" + dec32(uint32(detections)))
		}
		if now-lastHeap >= heapMarkEvery {
			lastHeap = now
			heapMark("run")
		}
		gosched()
	}

	if err := wakenet.Stop(); err != nil {
		println("[go] wakenet FAIL stop err=" + errText(err))
		return
	}
	println("[go] wakenet stop ok")

	f, avg, mx := wakenet.Stats()
	det := wakenet.Detected()
	println("[go] wakenet final frames=" + dec32(uint32(f)) + " avg_us=" +
		dec32(uint32(avg)) + " max_us=" + dec32(uint32(mx)) +
		" detections=" + dec32(uint32(detections)) + " errors=" +
		dec32(uint32(errors)) + " overflows=" + dec32(uint32(overflows)) +
		" detected_flag=" + boolText(det))

	if err := wakenet.Close(); err != nil {
		println("[go] wakenet FAIL close err=" + errText(err))
		return
	}
	heapMark("closed")

	if f >= passFrames {
		println("[go] wakenet PASS frames=" + dec32(uint32(f)) +
			" detections=" + dec32(uint32(detections)) +
			" avg_us=" + dec32(uint32(avg)) + " max_us=" + dec32(uint32(mx)))
	} else {
		println("[go] wakenet FAIL frames=" + dec32(uint32(f)) + " < " +
			dec32(uint32(passFrames)))
	}
}

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

// boolText renders a Go bool for the log.
func boolText(b bool) string {
	if b {
		return "1"
	}
	return "0"
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
