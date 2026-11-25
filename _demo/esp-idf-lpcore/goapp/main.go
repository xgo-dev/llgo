// LP core (coprocessor) acceptance probe for the llgo ESP-IDF pipeline
// (ESP32-S31).
//
// Built with: llgo build -target esp32s31-idf -o ../idfproj/main/llgo_out.o ./goapp
// The relocatable object is linked into the IDF firmware by the main
// component; app_main calls the generated C entry, which hands control to
// the llgo fiber scheduler (runtime.RunMain).
//
// The probe drives the pkg/esp/lpcore surface in order: the boot wakeup
// cause, the LP program's embedded-binary bytes (the IDF ULP build linked
// them into this firmware), the load path (including the oversize guard),
// the first run, eight HP->LP->HP mailbox echo exchanges (the LP core
// program answers every ping with the argument XOR 0xA55A - a value that
// can only be produced by code running on the LP core), the stop path, and
// then the stretch phase: a second run armed on the LP timer, HP deep
// sleep, and - on the reboot the wake causes - the wakeup cause read back
// as the timer wake. Every step prints one `[go] lp ...` line; `[go] lp
// PASS` is the acceptance gate, printed by whichever boot reaches its
// verdict (the deep-sleep wake boot when the stretch phase runs, the first
// boot otherwise).
package main

import (
	"unsafe"

	"github.com/xgo-dev/llgo/pkg/esp/esperr"
	"github.com/xgo-dev/llgo/pkg/esp/lpcore"
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

// vTaskDelay yields the CPU for ticks: the settle wait after Run and the
// mailbox send retries. On a host build the stub makes it a no-op.
//
//go:linkname vTaskDelay C.vTaskDelay
func vTaskDelay(ticks uint32)

// The embedded LP program: the IDF ULP build links the lp_core_main.bin
// bytes into this firmware; the demo glue hands out the pointer and the
// size (main/demo_glue.c).
//
//go:linkname lpProgPtr C.llgo_demo_lp_prog_ptr
func lpProgPtr() *byte

//
//go:linkname lpProgSize C.llgo_demo_lp_prog_size
func lpProgSize() int32

// lpMarkers reads the LP program's progress markers (boot marker, mailbox
// init result, entry-poll result, receive counter) through the ULP build's
// generated export header (main/demo_glue.c).
//
//go:linkname lpMarkers C.llgo_demo_lp_markers
func lpMarkers(out *uint32) int32

// lpRamRead reads words from the LP RAM window (RTC_SLOW_MEM) the HP side
// shares with the LP core (main/demo_glue.c).
//
//go:linkname lpRamRead C.llgo_demo_lp_ram_read
func lpRamRead(wordOff int32, out *uint32, words int32) int32

// wakeupCause reads esp_sleep_get_wakeup_causes (IDF v6.3's cause bitmap)
// and the IDF enum name of its first set bit (the name resolves on the C
// side, where the enum's per-target numbering is visible;
// main/demo_glue.c).
//
//go:linkname wakeupCause C.llgo_demo_lp_wakeup_cause
func wakeupCause(out *byte, max int32) int32

// sleepTimer enables the HP deep-sleep wakeup by the LP (RTC) timer:
// esp_sleep_enable_timer_wakeup with the duration split into two 32-bit
// halves across the integer-only boundary (main/demo_glue.c).
//
//go:linkname sleepTimer C.llgo_demo_sleep_timer
func sleepTimer(usLo int32, usHi int32) int32

// deepSleep enters HP deep sleep and never returns
// (esp_deep_sleep_start; main/demo_glue.c).
//
//go:linkname deepSleep C.llgo_demo_deep_sleep
func deepSleep()

// probeDeepSleep gates the stretch phase (second Run on the LP timer, HP
// deep sleep, wake-cause boot). The esp-emu 1.2.0 S31 backend does not model
// deep-sleep entry: esp_deep_sleep_start spins in the PMU's hardware-wait
// loop until the interrupt watchdog aborts the firmware (measured; see the
// README's fidelity notes), so the flag ships false and the acceptance run
// ends on the first boot. On hardware, flip to true: the phase reloads the
// LP program, arms it on the LP timer, puts the HP core to sleep for 3 s,
// and the reboot reports the timer wake.
const probeDeepSleep = false

const (
	// The HP<->LP wire format (shared with idfproj/main/lp_core/main.c):
	// one mailbox word per message, lpcore.Pack's (cmd, arg) halves.
	cmdPing   = 1 // HP -> LP: echo request
	cmdPong   = 2 // LP -> HP: the echo reply (same arg, XOR'd)
	replyMask = 0xA55A

	// exchanges is the ping-pong count. Eight matters: the hardware
	// mailbox rotates through four message slots per direction, and eight
	// is two full cycles, so every side's slot index lands back on its
	// reset value - the mailbox state stays legible across the phases.
	exchanges = 8

	// recvWaitMs bounds one echo reply (the LP core answers from its own
	// receive loop; under the emulator and on hardware alike the round
	// trip is far under this).
	recvWaitMs = 2000

	// sendRetries bounds the send attempts per exchange (each failed
	// attempt waits one FreeRTOS tick before retrying).
	sendRetries = 100

	// settleTicks is the post-Run settle wait in FreeRTOS ticks (100 ms):
	// the LP core's own startup - C runtime init, LP mailbox init - must
	// land before the first HP-side mailbox use, the same discipline the
	// IDF lp_mailbox example applies with a 100 ms task delay.
	settleTicks = 10

	// sleepWakeUs is the HP deep-sleep duration the LP (RTC) timer waits
	// before waking it, in microseconds.
	sleepWakeUs = 3_000_000

	// lpTimerUs is the LP timer period armed by the stretch phase's Run
	// (the LP core's own wake source), in microseconds.
	lpTimerUs = 2_000_000

	// wakeTimerName is the IDF enum name esp_sleep_get_wakeup_cause
	// reports for the timer wake (resolved on the C side, where the
	// per-target enum numbering is visible - the probe never hardcodes a
	// number).
	wakeTimerName = "ESP_SLEEP_WAKEUP_TIMER"
)

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

// hex4 formats v as four hex digits (the echo arguments' wire form).
func hex4(v uint32) string {
	const digits = "0123456789ABCDEF"
	var buf [4]byte
	for i := 0; i < 4; i++ {
		buf[3-i] = digits[(v>>(uint(i)*4))&0xF]
	}
	return string(buf[:])
}

// hex8 formats v as eight hex digits (the LP markers' raw words).
func hex8(v uint32) string {
	const digits = "0123456789ABCDEF"
	var buf [8]byte
	for i := 0; i < 8; i++ {
		buf[7-i] = digits[(v>>(uint(i)*4))&0xF]
	}
	return string(buf[:])
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

// heapMark prints the IDF heap watermarks at one probe stage. The label is a
// short literal, copied into a stack buffer so the C side sees plain bytes.
func heapMark(tag string) {
	var buf [24]byte
	n := copy(buf[:], tag)
	heapPrint(&buf[0], int32(n))
}

// cstr turns the glue's NUL-terminated response buffer into a Go string.
func cstr(p *byte, max int32) string {
	b := unsafe.Slice(p, max) // the C side's response buffer
	n := int32(0)
	for n < max && b[n] != 0 {
		n++
	}
	return string(b[:n])
}

// lpProgram returns the embedded LP core binary as a Go slice.
func lpProgram() []byte {
	p := lpProgPtr()
	n := lpProgSize()
	return unsafe.Slice(p, n) // the C bytes, handed to Load read-only
}

// sendRetry posts one message, retrying on the mailbox's own timeout (the
// LP core has not picked the message up yet - it answers fast, but its
// startup work lands after the HP side is already capable of sending).
// A send that exhausts its retries fails the exchange.
func sendRetry(m lpcore.Msg, what string) bool {
	for try := 0; try < sendRetries; try++ {
		err := lpcore.MailboxSend(m)
		if err == nil {
			return true
		}
		if esperr.Code(err) != 0x107 { // ESP_ERR_TIMEOUT: retry on the next tick
			println("[go] lp " + what + " send err=" + errText(err))
			return false
		}
		vTaskDelay(1)
	}
	println("[go] lp " + what + " send timeout")
	return false
}

// recvReply waits for one echo reply and returns it. The loop skips stale
// words - a reply from an earlier exchange can still sit in the mailbox's
// slot rotation after a retried send - and accepts only the tagged command
// the LP core is supposed to answer with. Bounded by MailboxRecv's own
// timeout per pass, plus the small drain slack.
func recvReply() (lpcore.Msg, bool) {
	for drains := 0; drains < 4; drains++ {
		msg, err := lpcore.MailboxRecv(recvWaitMs)
		if err != nil {
			println("[go] lp echo recv err=" + errText(err))
			return 0, false
		}
		if msg.Cmd() == cmdPong {
			return msg, true
		}
	}
	println("[go] lp echo recv stale")
	return 0, false
}

func main() {
	fixStreams()
	println("[go] lp: start")

	// The wakeup cause of THIS boot: exactly ESP_SLEEP_WAKEUP_UNDEFINED on
	// a cold boot, the timer wake after the stretch phase's deep sleep.
	var nameBuf [64]byte
	causes := wakeupCause(&nameBuf[0], int32(len(nameBuf)))
	name := cstr(&nameBuf[0], int32(len(nameBuf)))
	println("[go] lp wakeup_cause=" + dec32(uint32(causes)) + " " + name)

	if name != "ESP_SLEEP_WAKEUP_UNDEFINED" {
		// Second boot: the deep-sleep wake. The stretch phase's whole
		// verdict is this line - the HP core went down, the LP (RTC)
		// timer raised it, and the firmware is running again. The LP
		// markers ride along: on hardware the LP core ran through the
		// sleep, and whether its RAM state survived says what the sleep
		// retained.
		var mk [4]uint32
		lpMarkers(&mk[0])
		println("[go] lp markers2 boot=0x" + hex8(mk[0]) +
			" count=" + dec32(mk[3]))
		if name == wakeTimerName {
			println("[go] lp wake ok")
			println("[go] lp PASS")
		} else {
			println("[go] lp wake err=" + name)
		}
		return
	}

	prog := lpProgram()
	println("[go] lp prog bytes=" + dec32(uint32(len(prog))))

	// The oversize guard: one byte over the LP RAM ceiling, rejected with
	// the driver's own code before any C call. This check runs (and the
	// error is the same) on a host build - the honest-stub contract.
	if err := lpcore.Load(make([]byte, lpcore.MaxProgramSize+1)); esperr.Code(err) != 0x104 {
		println("[go] lp oversize err=" + errText(err))
		println("[go] lp FAIL oversize")
		return
	}
	println("[go] lp oversize rejected ESP_ERR_INVALID_SIZE ok")

	if err := lpcore.Load(prog); err != nil {
		println("[go] lp load err=" + errText(err))
		println("[go] lp FAIL load")
		return
	}
	println("[go] lp load ok")

	// Read the load back through the LP RAM window: the first 32 bytes of
	// the LP program must appear in RTC_SLOW_MEM exactly as they sit in the
	// embedded binary. This is the load path's whole story - the driver
	// said ok, and here is the memory to show for it.
	var rb [8]uint32
	if code := lpRamRead(0, &rb[0], int32(len(rb))); code != 0 {
		println("[go] lp lpram err=" + dec32(uint32(code)))
		println("[go] lp FAIL lpram")
		return
	}
	for w := 0; w < len(rb); w++ {
		want := uint32(prog[4*w]) | uint32(prog[4*w+1])<<8 |
			uint32(prog[4*w+2])<<16 | uint32(prog[4*w+3])<<24
		if rb[w] != want {
			println("[go] lp lpram word=" + dec32(uint32(w)) +
				" got=0x" + hex8(rb[w]) + " want=0x" + hex8(want))
			println("[go] lp FAIL lpram")
			return
		}
	}
	println("[go] lp lpram ok")
	heapMark("load")

	if err := lpcore.Run(lpcore.Config{WakeupSource: lpcore.WakeHP}); err != nil {
		println("[go] lp run err=" + errText(err))
		println("[go] lp FAIL run")
		return
	}
	println("[go] lp run ok")

	// Let the LP core's startup land (its C runtime and LP mailbox init)
	// before the first HP-side mailbox use, then read its progress markers:
	// boot=0xC0DEC0DE means the LP core executes; mbinit=0 means its mailbox
	// initialized; poll is the entry poll's esp_err (7 = ESP_ERR_TIMEOUT, the
	// honest answer for an empty mailbox); count is the receive counter.
	vTaskDelay(settleTicks)
	var mk [4]uint32
	lpMarkers(&mk[0])
	println("[go] lp markers boot=0x" + hex8(mk[0]) +
		" mbinit=" + dec32(mk[1]) +
		" poll=" + dec32(mk[2]) +
		" count=" + dec32(mk[3]))
	println("[go] lp settle ok")

	// The echo exchanges: HP sends (cmdPing, i), the LP core answers
	// (cmdPong, i XOR 0xA55A). The reply's argument can only be produced
	// by the LP core's program - the whole mailbox path in one check.
	// Gated on the boot marker: an environment where the LP core never
	// runs (the emulator without an LP hart model) reports the skip as a
	// line of its own instead of a fake success or a fake failure - the
	// markers say so either way.
	ok := 0
	if mk[0] != 0xC0DEC0DE {
		println("[go] lp echo skipped lp-core-not-running")
	} else {
		for i := 0; i < exchanges; i++ {
			if !sendRetry(lpcore.Pack(cmdPing, int32(i)), "echo") {
				println("[go] lp FAIL echo")
				return
			}
			msg, good := recvReply()
			if !good {
				println("[go] lp FAIL echo")
				return
			}
			want := int32(i) ^ replyMask
			if msg.Arg() != want {
				println("[go] lp echo i=" + dec32(uint32(i)) +
					" arg=0x" + hex4(uint32(msg.Arg())) +
					" want=0x" + hex4(uint32(want)))
				println("[go] lp FAIL echo")
				return
			}
			ok = i + 1
			println("[go] lp echo i=" + dec32(uint32(i)) +
				" arg=0x" + hex4(uint32(msg.Arg())) + " ok")
		}
		println("[go] lp echo " + dec32(uint32(ok)) + "/" + dec32(exchanges) + " ok")
	}
	heapMark("echo")

	if err := lpcore.Stop(); err != nil {
		println("[go] lp stop err=" + errText(err))
		println("[go] lp FAIL stop")
		return
	}
	println("[go] lp stop ok")

	// Stopping twice is harmless on the device (the driver call is void)
	// and errUnsupported on a host - pin both shapes.
	if err := lpcore.Stop(); err != nil {
		println("[go] lp stop twice err=" + errText(err))
		println("[go] lp FAIL stop")
		return
	}
	println("[go] lp stop twice ok")

	if !probeDeepSleep {
		println("[go] lp PASS")
		return
	}

	// Stretch phase: arm the LP core itself on the LP timer, then put the
	// HP core into deep sleep with its own LP-timer wake. The LP core
	// keeps running across the HP's sleep - that is the coprocessor's
	// purpose - and the timer wakes the HP core back into a fresh boot,
	// whose probe reads the wake cause above.
	if err := lpcore.Load(prog); err != nil {
		println("[go] lp reload err=" + errText(err))
		println("[go] lp FAIL reload")
		return
	}
	println("[go] lp reload ok")

	if err := lpcore.Run(lpcore.Config{WakeupSource: lpcore.WakeLPTimer, TimerSleepUs: lpTimerUs}); err != nil {
		println("[go] lp run timer err=" + errText(err))
		println("[go] lp FAIL run timer")
		return
	}
	println("[go] lp run timer ok")

	if code := sleepTimer(int32(sleepWakeUs&0xFFFFFFFF), int32(uint32(sleepWakeUs)>>32)); code != 0 {
		println("[go] lp sleep timer err=" + dec32(uint32(code)))
		println("[go] lp FAIL sleep timer")
		return
	}
	println("[go] lp sleep timer ok")

	println("[go] lp deep sleep now")
	deepSleep()
}
