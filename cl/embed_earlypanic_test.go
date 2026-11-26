package cl_test

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The early panic diagnostics (runtime/internal/runtime/earlypanic_baremetal.go
// plus _wrap/earlypanic_riscv32.c) print through a polled UART0 FIFO writer
// that works before stdio, the allocator or the trap setup exists. Every fatal path on the ESP baremetal targets must produce:
//
//   - a Go panic: "llgo panic: <value>", "at sp=0x...", the goroutine line
//     and a "#N pc=0x..." chain, followed by the ordinary stdio panic line
//     (exit semantics unchanged),
//   - a C-level fault: "llgo fault: mcause=... mepc=... mtval=... sp=..."
//     with the same chain, chained onward to the device trap so the
//     semihosting exit keeps working.
//
// The reports carry build-specific PCs, so the fixtures carry no expect.txt
// and these tests scan the emulator output for the markers instead. The
// runner ignores the emulator's exit status: a guest that reported a panic
// or a fault may terminate through the semihosting exit, through the chained
// ROM handler or through the emulator timeout, and every one of those is a
// valid ending for these fixtures. The --timeout bounds a guest that never
// reports at all.

func runEspEmuAnyExit(t *testing.T, emu string, tgt espEmuTarget, imgPath string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, emu, "--chip", tgt.emuArg, "--firmware", imgPath, "--timeout", "15s")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	_ = cmd.Run()
	return buf.String()
}

func runEarlyPanicFixture(t *testing.T, emu string, tgt espEmuTarget, pkg string, want []string) string {
	t.Helper()
	elfPath := buildESPEmuELF(t, tgt, pkg, t.TempDir())
	assertESPRAMLayout(t, tgt, elfPath)

	imgPath := filepath.Join(t.TempDir(), "flash.bin")
	buildESPEmuFlashImage(t, tgt, elfPath, imgPath)
	raw := runEspEmuAnyExit(t, emu, tgt, imgPath)

	for _, w := range want {
		if !strings.Contains(raw, w) {
			t.Fatalf("early panic output missing %q\n%s", w, espOutputTail(raw))
		}
	}
	return raw
}

// assertPanicChainShape checks the scanned pc chain lines: at least
// wantFrames frames, each "pc=0x" with lowercase hex digits.
func assertPanicChainShape(t *testing.T, raw string, wantFrames int) {
	t.Helper()
	frames := 0
	for line := range strings.SplitSeq(raw, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, " pc=0x")
		if idx < 0 {
			t.Fatalf("malformed chain line %q\n%s", line, espOutputTail(raw))
		}
		pc := line[idx+len(" pc=0x"):]
		if pc == "" {
			t.Fatalf("empty pc in chain line %q\n%s", line, espOutputTail(raw))
		}
		for _, r := range pc {
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
				t.Fatalf("non-hex pc %q in chain line %q\n%s", pc, line, espOutputTail(raw))
			}
		}
		frames++
	}
	if frames < wantFrames {
		t.Fatalf("expected at least %d chain frames, got %d\n%s", wantFrames, frames, espOutputTail(raw))
	}
}

func TestEarlyPanicInitBanner(t *testing.T) {
	requireEmbedTest(t)
	emu, ok := findEspEmu()
	if !ok {
		t.Skip("esp-emu is not available; set ESP_EMU to the emulator path")
	}
	for _, tgt := range []espEmuTarget{esp32s31BasicTarget, esp32c3BasicTarget} {
		t.Run(tgt.target, func(t *testing.T) {
			raw := runEarlyPanicFixture(t, emu, tgt, "./_testgo/earlypanicinit", []string{
				// The early banner, printed before stdio exists.
				"llgo panic: invalid memory address or nil pointer dereference",
				"at sp=0x",
				"goroutine ",
				// The scanned, symbolizable chain.
				"#0 pc=0x",
				// The ordinary stdio panic line still follows: exit
				// semantics unchanged.
				"panic: runtime error: invalid memory address or nil pointer dereference",
			})
			assertPanicChainShape(t, raw, 3)
		})
	}
}

func TestEarlyPanicGoroutineDump(t *testing.T) {
	requireEmbedTest(t)
	emu, ok := findEspEmu()
	if !ok {
		t.Skip("esp-emu is not available; set ESP_EMU to the emulator path")
	}
	for _, tgt := range []espEmuTarget{esp32s31BasicTarget, esp32c3BasicTarget} {
		t.Run(tgt.target, func(t *testing.T) {
			raw := runEarlyPanicFixture(t, emu, tgt, "./_testgo/earlypanicgs", []string{
				// The current G reports running, with its chain.
				"llgo panic: invalid memory address or nil pointer dereference",
				"[running]:\n#0 pc=0x",
				// The parked G reports waiting, with the chain preserved on
				// its fiber swap frame.
				"[waiting]:\n#0 pc=0x",
			})
			running := strings.Count(raw, "[running]:")
			waiting := strings.Count(raw, "[waiting]:")
			if running != 1 || waiting != 1 {
				t.Fatalf("expected 1 running and 1 waiting goroutine, got %d and %d\n%s", running, waiting, espOutputTail(raw))
			}
		})
	}
}

func TestEarlyPanicFaultReport(t *testing.T) {
	requireEmbedTest(t)
	emu, ok := findEspEmu()
	if !ok {
		t.Skip("esp-emu is not available; set ESP_EMU to the emulator path")
	}
	// s31 only: the esp32c3 emulator accepts unmapped stores silently and
	// turns the fault fixture into an emulator-specific heap-aliasing loop
	// that never reaches the trap; the same fixture under a cli-built image
	// faults and reports correctly on c3, so that is an emulator interaction
	// to chase separately, not a runtime gap.
	t.Run(esp32s31BasicTarget.target, func(t *testing.T) {
		tgt := esp32s31BasicTarget
		raw := runEarlyPanicFixture(t, emu, tgt, "./_testgo/earlypanicfault", []string{
			// The fault report from the chained trap handler. The markers are
			// short fragments because the emulator's own log lines interleave
			// into the console stream mid-line.
			"llgo fault:",
			"mcause=0x",
			"mepc=0x",
			"mtval=0xe0000004",
			"sp=0x",
			// The scanned, symbolizable chain.
			"#0 pc=0x",
		})
		assertPanicChainShape(t, raw, 1)
	})
}
