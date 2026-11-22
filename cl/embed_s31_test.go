package cl_test

import (
	"bytes"
	"cmp"
	"context"
	"crypto/sha256"
	"debug/elf"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/qiniu/x/test"
	"github.com/xgo-dev/llgo/cl/cltest"
	"github.com/xgo-dev/llgo/internal/build"
	"github.com/xgo-dev/llgo/internal/firmware"
)

// espEmuTarget describes one baremetal ESP target esp-emu boots from a
// ROM-loaded SRAM app image. The XIP flash targets boot through the ESP-IDF
// second-stage bootloader instead and keep their own helpers below.
type espEmuTarget struct {
	target     string      // llgo target name, e.g. "esp32c3-basic"
	emuArg     string      // esp-emu --chip value
	imageID    uint16      // chip id recorded in the ESP image header
	flashOff   int         // app image offset inside the merged flash dump
	ramWindows [][2]uint64 // physical RAM windows an allocated section may occupy
}

var (
	// The ESP32-C3 boots the app image from flash offset 0x0. Its 400 KiB
	// SRAM maps as two disjoint windows: the data bus sees the low 256 KiB
	// and the instruction bus maps the 256 KiB directly above it, below the
	// 16 KiB ICACHE (targets/esp32c3.memory.ld).
	esp32c3BasicTarget = espEmuTarget{
		target:  "esp32c3-basic",
		emuArg:  "esp32c3",
		imageID: 0x0005,
		ramWindows: [][2]uint64{
			{0x3fc80000, 0x3fca0000},
			{0x403a0000, 0x403e0000},
		},
	}
	// The ESP32-S31 baremetal target maps one 512 KiB DIRAM window for code
	// and data, and esp-emu boots the app image from flash offset 0x2000.
	esp32s31BasicTarget = espEmuTarget{
		target:     "esp32s31-basic",
		emuArg:     "esp32s31",
		imageID:    0x0020,
		flashOff:   0x2000,
		ramWindows: [][2]uint64{{0x2f000000, 0x2f080000}},
	}
)

// The flash (XIP) variant boots through the ESP-IDF second-stage bootloader.
const (
	s31XIPWindowBase    = 0x40000000
	s31XIPBootloaderOff = 0x2000
	s31XIPPartitionOff  = 0x8000
	s31XIPAppOff        = 0x10000
)

// TestS31XIPReflectSmoke runs the flash (XIP) target through the ESP-IDF
// second-stage bootloader under esp-emu. reflectmkfn exercises the libffi
// closure bridge, whose trampolines execute from SRAM allocated at runtime —
// exactly the path the SRAM-only layout cannot fit. The test needs the
// bootloader and partition table binaries of any ESP-IDF build for esp32s31
// (docker run espressif/idf:latest idf.py build); their paths come through
// LLGO_S31_XIP_BOOTLOADER and LLGO_S31_XIP_PARTTABLE.
func TestS31XIPReflectSmoke(t *testing.T) {
	requireEmbedTest(t)
	emu, ok := findEspEmu()
	if !ok {
		t.Skip("esp-emu is not available; set ESP_EMU to the emulator path")
	}
	bootloader := os.Getenv("LLGO_S31_XIP_BOOTLOADER")
	parttable := os.Getenv("LLGO_S31_XIP_PARTTABLE")
	if bootloader == "" || parttable == "" {
		t.Skip("flash (XIP) boot assets not configured; set LLGO_S31_XIP_BOOTLOADER and LLGO_S31_XIP_PARTTABLE")
	}

	elfPath := buildS31ELFWithTarget(t, "esp32s31-flash", "./_testgo/reflectmkfn", t.TempDir())
	assertS31CacheSyncWrap(t, elfPath)

	appImg := filepath.Join(t.TempDir(), "app.bin")
	if err := firmware.ConvertFormats("esp32s31", "", map[string]string{
		"out": elfPath,
		"bin": appImg,
	}); err != nil {
		t.Fatalf("app image generation failed: %v", err)
	}
	assertS31XIPImage(t, appImg)

	imgPath := filepath.Join(t.TempDir(), "flash.bin")
	mergeS31XIPFlash(t, bootloader, parttable, appImg, imgPath)
	runS31XIPImage(t, emu, imgPath)
}

// TestS31XIPFmtConsole runs the fmtall golden fixture on the flash (XIP)
// target. fmt reaches the USB-Serial/JTAG console through os.Stdout →
// syscall.Write → internal/runtime/syscall/linux.Syscall6, whose baremetal
// arm answer routes to the C library's write() — a path hexpkg and
// reflectmkfn never exercise (builtin println and no output respectively),
// which is why a broken fmt console went unnoticed until the first fmt XIP
// run. The runtime golden pins the routing together with fmt's formatting.
func TestS31XIPFmtConsole(t *testing.T) {
	requireEmbedTest(t)
	emu, ok := findEspEmu()
	if !ok {
		t.Skip("esp-emu is not available; set ESP_EMU to the emulator path")
	}
	bootloader := os.Getenv("LLGO_S31_XIP_BOOTLOADER")
	parttable := os.Getenv("LLGO_S31_XIP_PARTTABLE")
	if bootloader == "" || parttable == "" {
		t.Skip("flash (XIP) boot assets not configured; set LLGO_S31_XIP_BOOTLOADER and LLGO_S31_XIP_PARTTABLE")
	}

	elfPath := buildS31ELFWithTarget(t, "esp32s31-flash", "./_testgo/fmtall", t.TempDir())
	appImg := filepath.Join(t.TempDir(), "app.bin")
	if err := firmware.ConvertFormats("esp32s31", "", map[string]string{
		"out": elfPath,
		"bin": appImg,
	}); err != nil {
		t.Fatalf("app image generation failed: %v", err)
	}
	assertS31XIPImage(t, appImg)

	imgPath := filepath.Join(t.TempDir(), "flash.bin")
	mergeS31XIPFlash(t, bootloader, parttable, appImg, imgPath)
	raw := runS31XIPImage(t, emu, imgPath)

	_, expected, err := cltest.ReadGolden(filepath.Join("_testgo/fmtall", "expect.txt"))
	if err != nil {
		t.Fatal("ReadGolden failed:", err)
	}
	filtered := s31XIPAppOutput(t, raw)
	if test.Diff(t, filepath.Join("_testgo/fmtall", "expect.txt.new"),
		normalizeGoldenOutput(filtered), normalizeGoldenOutput(string(expected))) {
		t.Fatal("unexpected output")
	}
}

// s31XIPAppOutput returns the application output from a merged-flash run:
// everything after the bootloader's "Loaded app from partition" line with
// CRLF line endings and emulator log lines removed. Besides log filtering,
// cltest.FilterEmulatorOutput also carries the truncation semantics of
// cutting everything up to and including the ROM's "entry 0x…" boot line;
// in this XIP flow that line precedes the marker above, so the explicit
// marker scan drives the cut.
func s31XIPAppOutput(t *testing.T, raw string) string {
	t.Helper()
	lines := strings.Split(raw, "\n")
	for i, line := range lines {
		if strings.Contains(line, "Loaded app from partition") {
			return filterEspEmuOutput(cltest.FilterEmulatorOutput(strings.Join(lines[i+1:], "\n")))
		}
	}
	t.Fatalf("bootloader log marker not found\n%s", espOutputTail(raw))
	return ""
}

// assertS31XIPImage checks the app image the bootloader will parse: the
// writable segment first with the app descriptor at image offset 0x20,
// exactly two mapped segments, and every mapped segment holding its data at
// the file offset its virtual address maps to (offset in the image file must
// equal VMA-0x40000000, modulo the MMU page size, since the app partition is
// 64 KiB aligned).
func assertS31XIPImage(t *testing.T, imgPath string) {
	t.Helper()
	img, err := os.ReadFile(imgPath)
	if err != nil {
		t.Fatal(err)
	}
	if img[0] != 0xE9 {
		t.Fatalf("not an ESP image: magic %#x", img[0])
	}
	const appDescMagic = 0xABCD5432
	if got := binary.LittleEndian.Uint32(img[0x20:]); got != appDescMagic {
		t.Fatalf("app descriptor magic at image offset 0x20: got %#x", got)
	}
	off := 24
	maps := 0
	for seg := 0; seg < int(img[1]); seg++ {
		if off+8 > len(img) {
			t.Fatalf("segment %d header past end of image", seg)
		}
		addr := binary.LittleEndian.Uint32(img[off:])
		length := binary.LittleEndian.Uint32(img[off+4:])
		off += 8
		if off+int(length) > len(img) {
			t.Fatalf("segment %d data past end of image", seg)
		}
		if addr >= s31XIPWindowBase {
			maps++
			if want := addr - s31XIPWindowBase; uint32(off) != want {
				t.Fatalf("mapped segment %d data at file offset %#x, virtual address %#x maps %#x", seg, off, addr, want)
			}
		}
		off += int(length)
	}
	if maps != 2 {
		t.Fatalf("bootloader expects exactly 2 mapped segments, image has %d", maps)
	}
}

// mergeS31XIPFlash assembles the 4 MiB flash dump: bootloader, partition
// table and the application at their ESP-IDF default offsets.
func mergeS31XIPFlash(t *testing.T, bootloader, parttable, appImg, imgPath string) {
	t.Helper()
	flash := make([]byte, 4*1024*1024)
	for i := range flash {
		flash[i] = 0xFF
	}
	for _, place := range []struct {
		off int
		src string
	}{
		{s31XIPBootloaderOff, bootloader},
		{s31XIPPartitionOff, parttable},
		{s31XIPAppOff, appImg},
	} {
		data, err := os.ReadFile(place.src)
		if err != nil {
			t.Fatalf("read %s: %v", place.src, err)
		}
		if place.off+len(data) > len(flash) {
			t.Fatalf("%s overflows the 4 MiB flash image", place.src)
		}
		copy(flash[place.off:], data)
	}
	if err := os.WriteFile(imgPath, flash, 0o644); err != nil {
		t.Fatal(err)
	}
}

// runS31XIPImage boots the merged flash under esp-emu. The XIP application
// drives USB-Serial/JTAG outside the ROM console selector, so the console
// must be pinned to usb-jtag. A clean baremetal exit shows up as the
// semihost EBREAK with the PC inside the flash window; a crash would print
// a Guru Meditation dump instead. Returns the raw emulator output.
func runS31XIPImage(t *testing.T, emu, imgPath string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, emu,
		"--chip", "esp32s31", "--firmware", imgPath,
		"--timeout", "15s", "--console", "usb-jtag")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		t.Fatalf("esp-emu failed: %v\n%s", err, espOutputTail(buf.String()))
	}
	out := buf.String()
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "fatal error" || strings.HasPrefix(line, "fatal error:") || strings.HasPrefix(line, "panic:") {
			t.Fatalf("emulator runtime failure: %s\n%s", line, espOutputTail(out))
		}
	}
	if !strings.Contains(out, "Loaded app from partition") {
		t.Fatalf("bootloader did not load the app\n%s", espOutputTail(out))
	}
	if !strings.Contains(out, "EBREAK at PC=0x4") {
		t.Fatalf("application did not exit cleanly through the semihost EBREAK\n%s", espOutputTail(out))
	}
	return out
}

// TestS31ReflectSmoke runs the reflect fixtures the S31 baremetal target must
// execute under esp-emu, then asserts the linked image keeps the SRAM-only
// layout legal and the fence.i cache-sync wrapper actually linked. The C3
// embed suite covers the remaining embedded targets.
func TestS31ReflectSmoke(t *testing.T) {
	requireEmbedTest(t)
	emu, ok := findEspEmu()
	if !ok {
		t.Skip("esp-emu is not available; set ESP_EMU to the emulator path")
	}

	reflectElf := buildESPEmuELF(t, esp32s31BasicTarget, "./_testgo/reflect", t.TempDir())
	assertESPRAMLayout(t, esp32s31BasicTarget, reflectElf)
	assertS31CacheSyncWrap(t, reflectElf)
	runESPEmuImage(t, emu, esp32s31BasicTarget, reflectElf, []string{
		"call.slice 9 6", "call.func", "101", "call.closure", "201", "call.method", "202",
	})

	for _, tc := range []struct {
		pkg    string
		want   []string
		golden bool
	}{
		{pkg: "./_testgo/reflectmkfn"},
		{pkg: "./_testrt/reflectclosureenv", want: []string{"ok"}},
		{pkg: "./_testrt/ptrtothislazy", want: []string{"ok"}},
		// The time and starvation fixtures exercise the cycle-clock paths end
		// to end: scheduler parking, deadline wake, goroutine round trips and
		// the handoff yield. They compare the full golden like
		// TestS31XIPFmtConsole, which turns wake and release order into a
		// hard assertion instead of a marker scan. The timer fixtures add the
		// callback timer list on top of the sleeper list: deadline-ordered
		// delivery, refire, reset and the goroutine-context callback.
		{pkg: "./_testgo/timesleep", golden: true},
		{pkg: "./_testgo/sleepgoroutine", golden: true},
		{pkg: "./_testgo/timerafter", golden: true},
		{pkg: "./_testgo/timertick", golden: true},
		{pkg: "./_testgo/timerfunc", golden: true},
		// The time_* fixtures pin the user-visible timer semantics: deadline
		// order with select's default arm (time_after), ticker stop (time_tick),
		// NewTimer Stop/Reset return values and delivery (time_stop), and the
		// collector's reachability of objects held only through the pending
		// timer record (time_gckeep).
		{pkg: "./_testgo/time_after", golden: true},
		{pkg: "./_testgo/time_tick", golden: true},
		{pkg: "./_testgo/time_stop", golden: true},
		{pkg: "./_testgo/time_gckeep", golden: true},
		{pkg: "./_testgo/mutexstarve", golden: true},
		// Too close to the C3 IRAM ceiling to keep running there once the
		// vendored memmove/division shims landed; the 512 KiB S31 SRAM has
		// room, so the Value.Convert coverage stays on a baremetal target.
		// The fixture is assert-based (panics on any mismatch) and prints
		// nothing, so the plain run entry's clean-exit check is the assert.
		{pkg: "./_testgo/reflectconv"},
	} {
		t.Run(filepath.Base(tc.pkg), func(t *testing.T) {
			elfPath := buildESPEmuELF(t, esp32s31BasicTarget, tc.pkg, t.TempDir())
			if tc.golden {
				runESPEmuGolden(t, emu, esp32s31BasicTarget, elfPath, filepath.Join(tc.pkg, "expect.txt"))
				return
			}
			runESPEmuImage(t, emu, esp32s31BasicTarget, elfPath, tc.want)
		})
	}
}

// runESPEmuGolden boots the ELF under esp-emu and compares the filtered app
// output with the named expect.txt, the full-golden flow of
// TestS31XIPFmtConsole applied to the SRAM target.
func runESPEmuGolden(t *testing.T, emu string, tgt espEmuTarget, elfPath, golden string) {
	t.Helper()
	raw := runEspEmuFirmware(t, emu, tgt, elfPath)
	_, expected, err := cltest.ReadGolden(golden)
	if err != nil {
		t.Fatal("ReadGolden failed:", err)
	}
	filtered := filterEspEmuOutput(cltest.FilterEmulatorOutput(raw))
	if test.Diff(t, filepath.Join(filepath.Dir(golden), "expect.txt.new"),
		normalizeGoldenOutput(filtered), normalizeGoldenOutput(string(expected))) {
		t.Fatal("unexpected output")
	}
}

// findEspEmu locates the emulator binary; ESP_EMU overrides the PATH lookup.
func findEspEmu() (string, bool) {
	if p := os.Getenv("ESP_EMU"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
	}
	p, err := exec.LookPath("esp-emu")
	return p, err == nil
}

// buildESPEmuELF builds pkg for an esp-emu target. Target builds default to
// no embedded function-info tables, which keeps the SRAM-only images inside
// the S31 budget.
func buildESPEmuELF(t *testing.T, tgt espEmuTarget, pkg, dir string) string {
	return buildTargetELF(t, tgt.target, pkg, dir)
}

// buildS31ELFWithTarget builds pkg for the named esp32s31 target.
func buildS31ELFWithTarget(t *testing.T, target, pkg, dir string) string {
	return buildTargetELF(t, target, pkg, dir)
}

func buildTargetELF(t *testing.T, target, pkg, dir string) string {
	t.Helper()
	elfPath := filepath.Join(dir, "app.elf")
	conf := build.NewDefaultConf(build.ModeBuild)
	conf.Target = target
	conf.OutFile = elfPath
	if _, err := build.Do([]string{pkg}, conf); err != nil {
		t.Fatalf("build %s failed: %v", pkg, err)
	}
	return elfPath
}

type espSegment struct {
	addr uint32
	data []byte
}

// parseESPImage returns the entry point, the flashable segments and every
// allocated section range of the linked ELF.
func parseESPImage(t *testing.T, elfPath string) (uint32, []espSegment, [][2]uint64) {
	t.Helper()
	f, err := elf.Open(elfPath)
	if err != nil {
		t.Fatalf("open %s: %v", elfPath, err)
	}
	defer f.Close()
	var segs []espSegment
	var ranges [][2]uint64
	for _, s := range f.Sections {
		if s.Size == 0 || s.Flags&elf.SHF_ALLOC == 0 {
			continue
		}
		ranges = append(ranges, [2]uint64{s.Addr, s.Addr + s.Size})
		switch s.Type {
		case elf.SHT_PROGBITS, elf.SHT_INIT_ARRAY, elf.SHT_PREINIT_ARRAY, elf.SHT_FINI_ARRAY:
		default:
			// NOBITS sections occupy RAM but are not flashed.
			continue
		}
		data, err := s.Data()
		if err != nil {
			t.Fatalf("read section %s: %v", s.Name, err)
		}
		for len(data)%4 != 0 {
			data = append(data, 0)
		}
		segs = append(segs, espSegment{addr: uint32(s.Addr), data: data})
	}
	slices.SortFunc(segs, func(a, b espSegment) int { return cmp.Compare(a.addr, b.addr) })
	return uint32(f.Entry), segs, ranges
}

// assertESPRAMLayout verifies that the whole allocated footprint stays
// inside the target's physical RAM windows without section overlaps and that
// the heap markers are consistent with the linker script.
func assertESPRAMLayout(t *testing.T, tgt espEmuTarget, elfPath string) {
	t.Helper()
	_, _, ranges := parseESPImage(t, elfPath)
	slices.SortFunc(ranges, func(a, b [2]uint64) int { return cmp.Compare(a[0], b[0]) })
	inWindow := func(addr uint64) ([2]uint64, bool) {
		for _, w := range tgt.ramWindows {
			if addr >= w[0] && addr < w[1] {
				return w, true
			}
		}
		return [2]uint64{}, false
	}
	for i, r := range ranges {
		_, startOK := inWindow(r[0])
		_, endOK := inWindow(r[1] - 1)
		if !startOK || !endOK {
			t.Fatalf("section [%#x, %#x) escapes the %s RAM windows", r[0], r[1], tgt.target)
		}
		if i > 0 && r[0] < ranges[i-1][1] {
			t.Fatalf("section [%#x, %#x) overlaps a section ending at %#x", r[0], r[1], ranges[i-1][1])
		}
	}

	f, err := elf.Open(elfPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	symsyms, err := f.Symbols()
	if err != nil {
		t.Fatal(err)
	}
	syms := make(map[string]uint64)
	for _, s := range symsyms {
		syms[s.Name] = s.Value
	}
	heapStart, okStart := syms["_heapStart"]
	heapEnd, okEnd := syms["_heapEnd"]
	if !okStart || !okEnd || heapStart >= heapEnd {
		t.Fatalf("heap markers inconsistent: _heapStart=%#x(%v) _heapEnd=%#x(%v)",
			heapStart, okStart, heapEnd, okEnd)
	}
	heapWindow, ok := inWindow(heapStart)
	if !ok || heapEnd > heapWindow[1] {
		t.Fatalf("heap [%#x, %#x) escapes the %s RAM windows", heapStart, heapEnd, tgt.target)
	}
	if stack, ok := syms["__stack"]; ok {
		if _, inside := inWindow(stack); !inside {
			t.Fatalf("__stack=%#x escapes the %s RAM windows", stack, tgt.target)
		}
	}
}

// assertS31CacheSyncWrap verifies the linked __wrap___clear_cache is the
// fence.i stub rather than the compiler-rt baremetal abort, so dynamically
// generated code stays instruction-cache coherent.
func assertS31CacheSyncWrap(t *testing.T, elfPath string) {
	t.Helper()
	f, err := elf.Open(elfPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	syms, err := f.Symbols()
	if err != nil {
		t.Fatal(err)
	}
	var wrapAddr uint64
	for _, s := range syms {
		if s.Name == "__wrap___clear_cache" {
			wrapAddr = s.Value
		}
	}
	if wrapAddr == 0 {
		t.Fatal("__wrap___clear_cache is not linked; the cache-sync wrapper is missing")
	}
	for _, s := range f.Sections {
		if s.Type != elf.SHT_PROGBITS || wrapAddr < s.Addr || wrapAddr >= s.Addr+s.Size {
			continue
		}
		data, err := s.Data()
		if err != nil {
			t.Fatal(err)
		}
		code := data[wrapAddr-s.Addr:]
		if !bytes.HasPrefix(code, []byte{0x0f, 0x10, 0x00, 0x00}) {
			t.Fatalf("__wrap___clear_cache does not start with fence.i: % x", code[:min(8, len(code))])
		}
		return
	}
	t.Fatal("__wrap___clear_cache is outside any PROGBITS section")
}

// buildESPEmuFlashImage writes the ESP image format esp-emu loads: a v1
// image with the target's chip id and an appended SHA-256, merged into a
// 4 MiB flash dump at the target's boot offset.
func buildESPEmuFlashImage(t *testing.T, tgt espEmuTarget, elfPath, imgPath string) {
	t.Helper()
	entry, segs, _ := parseESPImage(t, elfPath)
	csum := byte(0xEF)
	for _, sg := range segs {
		for _, b := range sg.data {
			csum ^= b
		}
	}
	var img bytes.Buffer
	hdr := []byte{0xE9, byte(len(segs)), 2, 0x1F}
	hdr = binary.LittleEndian.AppendUint32(hdr, entry)
	hdr = append(hdr, 0xEE, 0, 0, 0)
	hdr = binary.LittleEndian.AppendUint16(hdr, tgt.imageID)
	hdr = append(hdr, 0)
	hdr = append(hdr, make([]byte, 8)...)
	hdr = append(hdr, 1)
	img.Write(hdr)
	for _, sg := range segs {
		img.Write(binary.LittleEndian.AppendUint32(nil, sg.addr))
		img.Write(binary.LittleEndian.AppendUint32(nil, uint32(len(sg.data))))
		img.Write(sg.data)
	}
	for img.Len()%16 != 15 {
		img.WriteByte(0)
	}
	img.WriteByte(csum)
	sum := sha256.Sum256(img.Bytes())
	img.Write(sum[:])

	flash := make([]byte, 4*1024*1024)
	copy(flash[tgt.flashOff:], img.Bytes())
	if err := os.WriteFile(imgPath, flash, 0o644); err != nil {
		t.Fatal(err)
	}
}

// runEspEmuFirmware images the ELF and boots it under esp-emu. It rejects
// the baremetal runtime's fatal/panic output in the raw stream and returns
// the unfiltered output for the caller's assertions.
func runEspEmuFirmware(t *testing.T, emu string, tgt espEmuTarget, elfPath string) string {
	t.Helper()
	imgPath := filepath.Join(t.TempDir(), "flash.bin")
	buildESPEmuFlashImage(t, tgt, elfPath, imgPath)

	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, emu, "--chip", tgt.emuArg, "--firmware", imgPath, "--timeout", "15s")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		t.Fatalf("esp-emu failed: %v\n%s", err, espOutputTail(buf.String()))
	}
	// Emulators can exit successfully after a runtime trap, so the unfiltered
	// output is checked before any filter can discard diagnostics. Unlike the
	// shared harness, which scans only run-only goldens, this scans every
	// runnable fixture on purpose: esp-emu reports no distinguishable process
	// exit for guest traps, and a fixture whose expected lines happen to
	// precede a fatal error would otherwise compare clean.
	for line := range strings.SplitSeq(buf.String(), "\n") {
		line = strings.TrimSpace(line)
		if line == "fatal error" || strings.HasPrefix(line, "fatal error:") || strings.HasPrefix(line, "panic:") {
			t.Fatalf("emulator runtime failure: %s\n%s", line, espOutputTail(buf.String()))
		}
	}
	return buf.String()
}

// runESPEmuImage boots the ELF under esp-emu and requires the wanted
// markers in the filtered output.
func runESPEmuImage(t *testing.T, emu string, tgt espEmuTarget, elfPath string, want []string) {
	t.Helper()
	filtered := cltest.FilterEmulatorOutput(runEspEmuFirmware(t, emu, tgt, elfPath))
	for _, w := range want {
		if !strings.Contains(filtered, w) {
			t.Fatalf("missing output marker %q\n%s", w, espOutputTail(filtered))
		}
	}
}

// espEmuLogLine matches the emulator's own log lines. esp-emu reports the
// semihost exit after the guest output while QEMU stays silent, so suite
// comparisons must not see them.
var espEmuLogLine = regexp.MustCompile(`^\[(INFO|WARN|ERROR|DEBUG|TRACE)(\s|])`)

// filterEspEmuOutput strips esp-emu's log lines from filtered app output.
func filterEspEmuOutput(output string) string {
	var kept []string
	for line := range strings.SplitSeq(output, "\n") {
		if !espEmuLogLine.MatchString(line) {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
}

func TestFilterEspEmuOutput(t *testing.T) {
	input := strings.Join([]string{
		"[INFO  cli::run] Starting emulation (chip=esp32s31)",
		"[WARN] no host console attached",
		"[ERROR machine] simulated bus fault",
		"hello from app",
		"ESP-ROM:esp32s31-20251218",
		"",
	}, "\n")
	want := strings.Join([]string{
		"hello from app",
		"ESP-ROM:esp32s31-20251218",
		"",
	}, "\n")
	if got := filterEspEmuOutput(input); got != want {
		t.Fatalf("filterEspEmuOutput() = %q, want %q", got, want)
	}
}

func espOutputTail(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > 20 {
		lines = lines[len(lines)-20:]
	}
	return strings.Join(lines, "\n")
}
