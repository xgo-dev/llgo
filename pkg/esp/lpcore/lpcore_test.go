//go:build !llgo.idf

// Host-build tests for the LP core surface. A host build has no LP core and
// no mailbox, so nothing here claims a coprocessor exists: the tests pin
// what the pure-Go layer owns - the program size guards and the error codes
// they report, the wakeup-source bit values, the (cmd, arg) message packing,
// the ErrName table, and the esperr error-value contract - and that every
// driver binding reports errUnsupported instead of faking success.
package lpcore

import (
	"errors"
	"testing"

	"github.com/xgo-dev/llgo/pkg/esp/esperr"
)

// TestWakeupSourceValues pins the constants to the IDF
// ULP_LP_CORE_WAKEUP_SOURCE_* bit numbers the glue casts straight into
// ulp_lp_core_cfg_t.wakeup_source: a value that drifts here would silently
// arm a different wake event than the name says.
func TestWakeupSourceValues(t *testing.T) {
	tests := map[uint32]uint32{
		WakeHP:      1 << 0, // ULP_LP_CORE_WAKEUP_SOURCE_HP_CPU
		WakeLPUart:  1 << 1, // ULP_LP_CORE_WAKEUP_SOURCE_LP_UART
		WakeLPIO:    1 << 2, // ULP_LP_CORE_WAKEUP_SOURCE_LP_IO
		WakeETM:     1 << 3, // ULP_LP_CORE_WAKEUP_SOURCE_ETM
		WakeLPTimer: 1 << 4, // ULP_LP_CORE_WAKEUP_SOURCE_LP_TIMER
		WakeVAD:     1 << 5, // ULP_LP_CORE_WAKEUP_SOURCE_LP_VAD
	}
	seen := map[uint32]bool{}
	for w, want := range tests {
		if w != want {
			t.Fatalf("wakeup source %d, want the IDF BIT value %d", w, want)
		}
		if seen[w] {
			t.Fatalf("wakeup source bit %d used twice", w)
		}
		seen[w] = true
	}
}

// TestLoadValidation pins the client-side guards: an empty program and an
// oversize program are rejected before anything crosses into C, with the
// esp_err codes the driver itself uses; a size inside the LP RAM ceiling
// reaches the binding, which on a host reports errUnsupported - never a
// fake success.
func TestLoadValidation(t *testing.T) {
	if err := Load(nil); esperr.Code(err) != 0x102 {
		t.Fatalf("Load(nil) = %v, want ESP_ERR_INVALID_ARG (0x102)", err)
	}
	if err := Load([]byte{}); esperr.Code(err) != 0x102 {
		t.Fatalf("Load(empty) = %v, want ESP_ERR_INVALID_ARG (0x102)", err)
	}
	if err := Load(make([]byte, MaxProgramSize+1)); esperr.Code(err) != 0x104 {
		t.Fatalf("Load(MaxProgramSize+1) = %v, want ESP_ERR_INVALID_SIZE (0x104)", err)
	}
	err := Load(make([]byte, 256))
	if !errors.Is(err, &esperr.Error{Code: errUnsupported}) {
		t.Fatalf("Load(256B) on a host build = %v, want errUnsupported from the stub", err)
	}
	if named, ok := err.(*esperr.Error); !ok || named.Name != "esp_err 0x007F00" {
		t.Fatalf("stub error = %v, want the hex fallback name (the stub code has no esp_err name)", err)
	}
}

// TestMaxProgramSize pins the ceiling to the S31's LP RAM: 32 KiB, above
// the Kconfig reservation cap (32744) the IDF load path enforces per
// project, so the Go-side guard can only ever reject programs the driver
// would reject too.
func TestMaxProgramSize(t *testing.T) {
	if MaxProgramSize != 32768 {
		t.Fatalf("MaxProgramSize = %d, want the 32 KiB LP RAM ceiling", MaxProgramSize)
	}
}

// TestRunValidation pins the wakeup-source guard: the driver rejects a
// config with no wakeup bit with ESP_ERR_INVALID_ARG (ulp_lp_core_run's own
// check), and the Go side reports the same shape before crossing into C.
// A valid config reaches the stub, which stays honest.
func TestRunValidation(t *testing.T) {
	if err := Run(Config{}); esperr.Code(err) != 0x102 {
		t.Fatalf("Run(zero cfg) = %v, want ESP_ERR_INVALID_ARG (0x102)", err)
	}
	err := Run(Config{WakeupSource: WakeHP})
	if !errors.Is(err, &esperr.Error{Code: errUnsupported}) {
		t.Fatalf("Run(WakeHP) on a host build = %v, want errUnsupported from the stub", err)
	}
	err = Run(Config{WakeupSource: WakeLPTimer, TimerSleepUs: 1000000})
	if !errors.Is(err, &esperr.Error{Code: errUnsupported}) {
		t.Fatalf("Run(WakeLPTimer) on a host build = %v, want errUnsupported from the stub", err)
	}
}

// TestStubReportsUnsupported pins the honest-stub contract for the rest of
// the surface: nothing pretends a coprocessor exists, and MailboxRecv
// reports errUnsupported rather than a fake timeout - there is no mailbox
// here to time out on.
func TestStubReportsUnsupported(t *testing.T) {
	unsupported := &esperr.Error{Code: errUnsupported}

	for name, err := range map[string]error{
		"Stop":        Stop(),
		"MailboxSend": MailboxSend(Pack(1, 42)),
	} {
		if !errors.Is(err, unsupported) {
			t.Fatalf("%s on a host build = %v, want errUnsupported (0x%X)", name, err, errUnsupported)
		}
	}
	if msg, err := MailboxRecv(0); msg != 0 || !errors.Is(err, unsupported) {
		t.Fatalf("MailboxRecv(0) on a host build = (%d, %v), want (0, errUnsupported)", msg, err)
	}
	if msg, err := MailboxRecv(50); !errors.Is(err, unsupported) {
		t.Fatalf("MailboxRecv(50) on a host build = (%d, %v), want errUnsupported (not a timeout)", msg, err)
	}
	// A negative bound keeps its documented shape: one poll, then the same
	// report - never a skipped poll.
	if msg, err := MailboxRecv(-1); msg != 0 || !errors.Is(err, unsupported) {
		t.Fatalf("MailboxRecv(-1) on a host build = (%d, %v), want (0, errUnsupported)", msg, err)
	}
}

// TestPackUnpack pins the (cmd, arg) message convention: cmd in the high
// half, arg in the low half, both truncated to their 16 bits, and the raw
// word round-trips. The negative-word case matters because a cmd with bit
// 15 set makes the whole word negative as an int32.
func TestPackUnpack(t *testing.T) {
	tests := []struct {
		cmd, arg int32
		word     int32
	}{
		{0x1234, 0xABCD, 0x1234ABCD},
		{0, 0, 0},
		{1, 42, (1 << 16) | 42},
		{0xFFFF, 0xFFFF, -1},
		{-1, -1, -1},             // both halves truncate: cmd 0xFFFF, arg 0xFFFF
		{0x12345, 7, 0x23450007}, // cmd truncates to 16 bits
		{2, 0x12345, 0x00022345}, // arg truncates to 16 bits
	}
	for _, tc := range tests {
		word := int32(Pack(tc.cmd, tc.arg))
		if word != tc.word {
			t.Fatalf("Pack(0x%X, 0x%X) = 0x%08X, want 0x%08X", tc.cmd, tc.arg, word, tc.word)
		}
		m := Msg(tc.word)
		if m.Cmd() != int32(tc.word>>16) || m.Arg() != int32(tc.word&0xFFFF) {
			t.Fatalf("Msg(0x%08X) = (%d, %d), want the word's own halves", tc.word, m.Cmd(), m.Arg())
		}
	}
}

// TestErrName covers the table this package prints through: the esp_err
// values the LP core pipeline can return (including the size and timeout
// codes the guards produce) and the hex fallback for codes the table does
// not name.
func TestErrName(t *testing.T) {
	tests := map[int32]string{
		0:      "ESP_OK",
		-1:     "ESP_FAIL",
		0x101:  "ESP_ERR_NO_MEM",
		0x102:  "ESP_ERR_INVALID_ARG",
		0x103:  "ESP_ERR_INVALID_STATE",
		0x104:  "ESP_ERR_INVALID_SIZE",
		0x105:  "ESP_ERR_NOT_FOUND",
		0x107:  "ESP_ERR_TIMEOUT",
		0x7F00: "esp_err 0x007F00",
		0x1234: "esp_err 0x001234",
	}
	for code, want := range tests {
		if got := ErrName(code); got != want {
			t.Fatalf("ErrName(0x%X) = %q, want %q", code, got, want)
		}
	}
}

// TestErrOf pins the error-value contract the family shares: ESP_OK is nil,
// anything else carries both the code and its IDF name, and errors.Is
// matches by code through esperr.
func TestErrOf(t *testing.T) {
	if err := errOf(0); err != nil {
		t.Fatalf("errOf(0) = %v, want nil", err)
	}
	err := errOf(0x104)
	if esperr.Code(err) != 0x104 {
		t.Fatalf("errOf(0x104) code = 0x%X", esperr.Code(err))
	}
	if err.Error() != "ESP_ERR_INVALID_SIZE (260)" {
		t.Fatalf("errOf(0x104) = %q", err.Error())
	}
	if !errors.Is(err, &esperr.Error{Code: 0x104}) {
		t.Fatal("errors.Is did not match by code")
	}
	if errors.Is(err, esperr.ErrTimeout) {
		t.Fatal("errors.Is matched an unrelated sentinel")
	}
	if !errors.Is(ErrTimeout, &esperr.Error{Code: 0x107}) {
		t.Fatal("ErrTimeout does not satisfy errors.Is against ESP_ERR_TIMEOUT")
	}
	if ErrTimeout.Error() != "ESP_ERR_TIMEOUT (263)" {
		t.Fatalf("ErrTimeout.Error() = %q", ErrTimeout.Error())
	}
}

// TestQuantumDivisibility pins the invariant MailboxRecv relies on: the
// poll quantum is a whole FreeRTOS tick. If tickMs ever grew past
// pollQuantumMs, vTaskDelay(0) in the loop would spin instead of park, and
// a timeout bound would burn CPU instead of measuring time.
func TestQuantumDivisibility(t *testing.T) {
	if pollQuantumMs%tickMs != 0 || pollQuantumMs/tickMs < 1 {
		t.Fatalf("pollQuantumMs %d is not a positive multiple of tickMs %d",
			pollQuantumMs, tickMs)
	}
}
