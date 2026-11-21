package cl_test

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qiniu/x/test"
	"github.com/xgo-dev/llgo/cl/cltest"
)

// The fiber stack-overflow guard only exists on the RV32 baremetal fiber
// scheduler: hosted stacks grow instead of fataling, and the xtensa ESP32
// keeps the pthread scheduler, so the hosted and esp32 suites skip both
// fixtures. stacksafe joins the C3 emu auto-walk like every other fixture;
// stackover ends in the runtime's fatal error, which the shared firmware
// runner rejects, so it runs exclusively through the fatal-tolerant runner
// below — on S31 and C3 here, never through the auto-walk.

func TestFiberStackGuardEmu(t *testing.T) {
	requireEmbedTest(t)
	emu, ok := findEspEmu()
	if !ok {
		t.Skip("esp-emu is not available; set ESP_EMU to the emulator path")
	}
	for _, tgt := range []espEmuTarget{esp32s31BasicTarget, esp32c3BasicTarget} {
		t.Run(tgt.target, func(t *testing.T) {
			elfPath := buildESPEmuELF(t, tgt, "./_testgo/stackover", t.TempDir())
			assertESPRAMLayout(t, tgt, elfPath)
			raw := runEspEmuFirmwareExpectFatal(t, emu, tgt, elfPath,
				"fatal error: goroutine stack overflow")
			filtered := filterEspEmuOutput(cltest.FilterEmulatorOutput(raw))
			_, expected, err := cltest.ReadGolden("./_testgo/stackover/expect.txt")
			if err != nil {
				t.Fatal("ReadGolden failed:", err)
			}
			if test.Diff(t, "./_testgo/stackover/expect.txt.new",
				normalizeGoldenOutput(filtered), normalizeGoldenOutput(string(expected))) {
				t.Fatal("unexpected output")
			}
		})
	}
}

// TestS31FiberStackSafe pins the stacksafe golden on the S31 emu target: the
// C3 auto-walk already covers the fixture, but the S31 suite runs fixtures
// explicitly, so the no-false-positive side of the guard gets its golden here
// too. stackover for the same target lives in TestFiberStackGuardEmu.
func TestS31FiberStackSafe(t *testing.T) {
	requireEmbedTest(t)
	emu, ok := findEspEmu()
	if !ok {
		t.Skip("esp-emu is not available; set ESP_EMU to the emulator path")
	}
	elfPath := buildESPEmuELF(t, esp32s31BasicTarget, "./_testgo/stacksafe", t.TempDir())
	assertESPRAMLayout(t, esp32s31BasicTarget, elfPath)
	runESPEmuGolden(t, emu, esp32s31BasicTarget, elfPath, "./_testgo/stacksafe/expect.txt")
}

// runEspEmuFirmwareExpectFatal boots the ELF expecting the runtime to die
// through a fatal error. Unlike runEspEmuFirmware it neither rejects the
// fatal line nor requires a clean emulator exit: the semihosting exit after
// the runtime's exit(2) may surface as any emulator status, and a guest that
// never reaches the fatal is bounded by the emulator's --timeout. Only the
// expected fatal line and the golden comparison decide the outcome.
func runEspEmuFirmwareExpectFatal(t *testing.T, emu string, tgt espEmuTarget, elfPath, wantFatal string) string {
	t.Helper()
	imgPath := filepath.Join(t.TempDir(), "flash.bin")
	buildESPEmuFlashImage(t, tgt, elfPath, imgPath)

	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, emu, "--chip", tgt.emuArg, "--firmware", imgPath, "--timeout", "15s")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	_ = cmd.Run()
	out := buf.String()
	if !strings.Contains(out, wantFatal) {
		t.Fatalf("expected %q in emulator output\n%s", wantFatal, espOutputTail(out))
	}
	return out
}
