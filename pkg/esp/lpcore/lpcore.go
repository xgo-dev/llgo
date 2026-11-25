// Package lpcore drives the ESP32-S31's LP (low-power) coprocessor core
// from llgo-compiled Go code: load an LP program into LP RAM, run it, and
// exchange single-word messages with it over the hardware mailbox.
//
// The LP core is an independent small RISC-V (rv32imac_zicsr) that keeps
// running while the HP core sleeps. Its RAM is 32 KiB; the load path
// rejects anything larger (ESP_ERR_INVALID_SIZE), and the firmware itself
// additionally enforces the project's own LP RAM reservation
// (CONFIG_ULP_COPROC_RESERVE_MEM, which can be smaller than the physical
// ceiling - the IDF build reserves a slice of LP RAM for the program).
//
// It follows the pkg/esp package template (see the package wifi docs and
// pkg/esp/README.md): wifi-style three-file Go split plus a thin C glue
// template. IDF v6.3-dev API surface: ulp_lp_core_load_binary /
// ulp_lp_core_run / ulp_lp_core_stop (ulp component) and the handle-based
// hardware mailbox lp_core_mailbox_init / send / receive.
//
// # Usage
//
//	if e := lpcore.Load(prog); e != nil { ... }          // prog = the LP .bin
//	if e := lpcore.Run(lpcore.Config{WakeupSource: lpcore.WakeHP}); e != nil { ... }
//	if e := lpcore.MailboxSend(lpcore.Pack(1, 42)); e != nil { ... } // retry on ErrTimeout
//	msg, err := lpcore.MailboxRecv(2000)                 // bounded poll
//	cmd, arg := msg.Cmd(), msg.Arg()
//	lpcore.Stop()
//
// # Message words
//
// The mailbox carries one 32-bit word per message (the C lp_message_t).
// Pack/Cmd/Arg layer a (cmd, arg) convention over that word - cmd in the
// high half, arg in the low half (both 16-bit) - so the demo-grade
// command protocols stay one message each way. A program that owns its
// own wire format can send raw words (Msg(x)) and unpack them itself.
//
// # Sleeping and scheduling
//
// The LP core itself never sleeps in this model: Run starts it, and it
// runs (or waits in its own loops) until Stop. The LP_TIMER wakeup source
// arms the LP timer to wake the LP core out of its own sleep - a program
// that wants HP deep sleep binds esp_deep_sleep_start in its own glue;
// this package intentionally stays out of the PMU surface.
//
// # Glue integration
//
// Copy glue/lpcore_glue.c into your project's main component
// (WHOLE_ARCHIVE required; REQUIRES ulp); see pkg/esp/README.md for the
// two integration recipes and the Go<->C boundary rules.
package lpcore

import "github.com/xgo-dev/llgo/pkg/esp/esperr"

// errOf wraps a raw esp_err code as the error value this package returns
// (nil for ESP_OK), named through ErrName so a printed error reads like
// the IDF's own name for the code.
func errOf(code int32) error {
	if code == 0 {
		return nil
	}
	return &esperr.Error{Code: code, Name: ErrName(code)}
}

// ErrTimeout reports a bounded mailbox wait that found nothing:
// ESP_ERR_TIMEOUT (0x107), the esp_err family's own timeout code (unlike
// the transports' ErrTimeout, which carries the lwip ETIMEDOUT value).
var ErrTimeout = &esperr.Error{Code: 0x107, Name: ErrName(0x107)}

// ErrName names the esp_err codes this package can surface with their IDF
// names; anything else renders as the bare hex form, which keeps the code
// itself the stable identifier.
func ErrName(code int32) string {
	switch code {
	case 0:
		return "ESP_OK"
	case -1:
		return "ESP_FAIL"
	case 0x101:
		return "ESP_ERR_NO_MEM"
	case 0x102:
		return "ESP_ERR_INVALID_ARG"
	case 0x103:
		return "ESP_ERR_INVALID_STATE"
	case 0x104:
		return "ESP_ERR_INVALID_SIZE"
	case 0x105:
		return "ESP_ERR_NOT_FOUND"
	case 0x107:
		return "ESP_ERR_TIMEOUT"
	}
	return "esp_err 0x" + hex6(uint32(code))
}

// hex6 formats the low 24 bits as fixed-width hex (esp_err codes fit).
func hex6(v uint32) string {
	const digits = "0123456789ABCDEF"
	var buf [6]byte
	for i := 0; i < 6; i++ {
		buf[5-i] = digits[(v>>(uint(i)*4))&0xF]
	}
	return string(buf[:])
}

// MaxProgramSize is the S31 LP RAM ceiling in bytes: a program larger
// than this cannot fit the coprocessor's memory at all. Load rejects
// oversize programs with ESP_ERR_INVALID_SIZE before crossing into C;
// the IDF's load path then enforces the project's own LP RAM reservation
// (CONFIG_ULP_COPROC_RESERVE_MEM, Kconfig-capped at 32744 on the S31),
// which can be smaller than this physical bound.
const MaxProgramSize = 32 * 1024

// Load copies an LP core program (the .bin the IDF ULP build emits) into
// LP RAM. An oversize program is rejected client-side with
// ESP_ERR_INVALID_SIZE - the same code the driver uses - and an empty
// program with ESP_ERR_INVALID_ARG (the driver would happily memcpy zero
// bytes and boot garbage; there is no honest success to report).
// Loading also stops a running LP core and clears the LP RAM, so a
// re-Load always starts the program from its reset state.
func Load(program []byte) error {
	if len(program) == 0 {
		return errOf(0x102) // ESP_ERR_INVALID_ARG: nothing to boot
	}
	if len(program) > MaxProgramSize {
		return errOf(0x104) // ESP_ERR_INVALID_SIZE: the driver's own code
	}
	return errOf(load(&program[0], int32(len(program))))
}

// Wakeup source flags for Config.WakeupSource. They are the IDF's
// ULP_LP_CORE_WAKEUP_SOURCE_* bits, combinable with |. WakeHP starts the
// LP core right away (a single wake from the HP CPU); the others arm
// hardware events that wake it later. On the S31 the LP core has no ROM,
// so Run writes the boot address into LP RAM before the core starts.
const (
	WakeHP      uint32 = 1 << 0 // started by the HP core (one single wakeup)
	WakeLPUart  uint32 = 1 << 1 // wake on LP UART RX pulses
	WakeLPIO    uint32 = 1 << 2 // wake on LP IO interrupt
	WakeETM     uint32 = 1 << 3 // wake on an ETM event
	WakeLPTimer uint32 = 1 << 4 // wake on the LP timer (see Config.TimerSleepUs)
	WakeVAD     uint32 = 1 << 5 // wake on the LP VAD
)

// Config describes one LP core run.
type Config struct {
	// WakeupSource is a Wake* bitmask; zero is rejected with
	// ESP_ERR_INVALID_ARG (the driver's own guard, mirrored client-side).
	WakeupSource uint32
	// TimerSleepUs is the LP timer period when WakeupSource includes
	// WakeLPTimer, in microseconds; ignored otherwise.
	TimerSleepUs uint32
}

// Run configures the LP core and boots the program a previous Load put in
// LP RAM. It returns ESP_ERR_INVALID_ARG for a Config with no wakeup
// source bit set (the driver's own guard, mirrored client-side so a host
// build reports the same shape) and otherwise the driver's esp_err.
func Run(cfg Config) error {
	if cfg.WakeupSource == 0 {
		return errOf(0x102) // ESP_ERR_INVALID_ARG: the driver's own guard
	}
	return errOf(run(int32(cfg.WakeupSource), int32(cfg.TimerSleepUs)))
}

// Stop puts the LP core to sleep and disarms every wakeup source. The
// driver call is void and cannot fail, so Stop is nil on the device for
// any state (stopping an already-stopped core is harmless); on a host
// build it is errUnsupported like every other binding. Run restarts the
// core afterwards.
func Stop() error {
	return errOf(stop())
}

// Msg is one mailbox message word: the exact 32 bits that travel between
// the HP and LP cores (the C lp_message_t). Pack/Cmd/Arg are the
// (cmd, arg) convention this package documents; a raw word converts with
// Msg(x).
type Msg int32

// Pack combines a 16-bit command and a 16-bit argument into one message
// word: (cmd << 16) | (arg & 0xFFFF). Both halves truncate to their 16
// bits - a caller with wider values must split them across two messages.
func Pack(cmd, arg int32) Msg {
	return Msg((cmd << 16) | (arg & 0xFFFF))
}

// Cmd returns the message's command half (bits 31..16, as sent by Pack).
// Note the Go-side value is signed: a cmd of 0x8000 reads back as -32768,
// while the LP side decodes the same bits unsigned - protocols should
// keep cmd values below 0x8000 to stay sign-clean.
func (m Msg) Cmd() int32 { return int32(m) >> 16 }

// Arg returns the message's argument half (bits 15..0, as sent by Pack).
func (m Msg) Arg() int32 { return int32(m) & 0xFFFF }

// MailboxSend posts one message word to the LP core. The send is one
// bounded blocking attempt: the driver's synchronous send waits for the LP
// core to receive (and acknowledge) the message through the mailbox
// interrupt, at most 500 ms, and this call reports ESP_ERR_TIMEOUT when
// that bound passed without an acknowledgement. The bound means the C
// call freezes the llgo fiber scheduler for up to 500 ms - call it from a
// goroutine session that owns the exchange, never from code that must
// keep other goroutines running (see the concurrency note in the package
// comment).
//
// The 500 ms wait is not just patience: every send advances the driver's
// TX slot cursor, so a caller-level retry storm wraps the four slots and
// overwrites messages the LP core has not consumed yet (measured on
// hardware). On ESP_ERR_TIMEOUT the word stays in its slot and may still
// be delivered later; pair every send with a tagged receive (the demo
// probe's (cmd, arg) convention) so a late or duplicated reply is
// recognizable. On a host build it is errUnsupported.
func MailboxSend(m Msg) error {
	return errOf(mailboxSend(int32(m)))
}

// MailboxRecv waits up to timeoutMs for one message word from the LP core
// and returns (msg, nil), or (zero, ErrTimeout) when the bound passed
// with nothing. A non-positive bound makes a single non-blocking poll
// (reported the same way on a miss). Any other esp_err surfaces as its
// named error.
//
// The bound is wall clock, one 10 ms tick at a time: every idle iteration
// parks the FreeRTOS task in vTaskDelay (which also feeds the idle task
// and the watchdog) instead of spinning, and the C-level receive always
// runs with a zero tick timeout - no C call is held across a scheduling
// point, so other goroutines keep running between quanta (Gosched). The
// bound can overshoot by up to one tick; a message that arrives during a
// sleep is seen on the next pass. On a host build it is errUnsupported.
func MailboxRecv(timeoutMs int32) (Msg, error) {
	for ms := int32(0); ; ms += pollQuantumMs {
		var raw int32
		code := mailboxRecv(&raw)
		if code == 0 {
			return Msg(raw), nil
		}
		if code != 0x107 { // ESP_ERR_TIMEOUT = empty this pass, keep polling
			return 0, errOf(code)
		}
		if ms >= timeoutMs {
			return 0, ErrTimeout
		}
		gosched()
		vTaskDelay(pollQuantumMs / tickMs)
	}
}

// The pacing constants mirror the wifi package's model (pkg/esp/README.md,
// "Async events go through a mailbox"): configTICK_RATE_HZ is 100, so one
// FreeRTOS tick is 10 ms and the recv loop rests exactly one tick per
// pass.
const (
	pollQuantumMs = 10
	tickMs        = 10
)
