package cl_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"debug/elf"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/xgo-dev/llgo/cl/cltest"
	"github.com/xgo-dev/llgo/internal/build"
)

// The ESP32-S31 baremetal target maps one 512 KiB DIRAM window for code and
// data, and esp-emu boots the app image from flash offset 0x2000.
const (
	s31SRAMStart = 0x2f000000
	s31SRAMEnd   = 0x2f080000
	s31FlashOff  = 0x2000
	s31ChipID    = 0x0020
)

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

	reflectElf := buildS31ELF(t, "./_testgo/reflect", t.TempDir())
	assertS31SRAMLayout(t, reflectElf)
	assertS31CacheSyncWrap(t, reflectElf)
	runS31Image(t, emu, reflectElf, []string{
		"call.slice 9 6", "call.func", "101", "call.closure", "201", "call.method", "202",
	})

	for _, tc := range []struct {
		pkg  string
		want []string
	}{
		{pkg: "./_testgo/reflectmkfn"},
		{pkg: "./_testrt/reflectclosureenv", want: []string{"ok"}},
		{pkg: "./_testrt/ptrtothislazy", want: []string{"ok"}},
	} {
		t.Run(filepath.Base(tc.pkg), func(t *testing.T) {
			elfPath := buildS31ELF(t, tc.pkg, t.TempDir())
			runS31Image(t, emu, elfPath, tc.want)
		})
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

// buildS31ELF builds pkg for the esp32s31-basic target with function-info
// metadata disabled; that metadata alone pushes the reflect image past the
// SRAM budget.
func buildS31ELF(t *testing.T, pkg, dir string) string {
	t.Helper()
	elfPath := filepath.Join(dir, "s31.elf")
	old, had := os.LookupEnv("LLGO_FUNCINFO")
	if err := os.Setenv("LLGO_FUNCINFO", "0"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if had {
			_ = os.Setenv("LLGO_FUNCINFO", old)
		} else {
			_ = os.Unsetenv("LLGO_FUNCINFO")
		}
	}()
	conf := build.NewDefaultConf(build.ModeBuild)
	conf.Target = "esp32s31-basic"
	conf.OutFile = elfPath
	if _, err := build.Do([]string{pkg}, conf); err != nil {
		t.Fatalf("build %s failed: %v", pkg, err)
	}
	return elfPath
}

type s31Segment struct {
	addr uint32
	data []byte
}

// parseS31Image returns the entry point, the flashable segments and every
// allocated section range of the linked ELF.
func parseS31Image(t *testing.T, elfPath string) (uint32, []s31Segment, [][2]uint64) {
	t.Helper()
	f, err := elf.Open(elfPath)
	if err != nil {
		t.Fatalf("open %s: %v", elfPath, err)
	}
	defer f.Close()
	var segs []s31Segment
	var ranges [][2]uint64
	for _, s := range f.Sections {
		if s.Size == 0 || s.Flags&elf.SHF_ALLOC == 0 {
			continue
		}
		ranges = append(ranges, [2]uint64{s.Addr, s.Addr + s.Size})
		switch s.Type {
		case elf.SHT_PROGBITS, elf.SHT_INIT_ARRAY, elf.SHT_PREINIT_ARRAY, elf.SHT_FINI_ARRAY:
		default:
			// NOBITS sections occupy SRAM but are not flashed.
			continue
		}
		data, err := s.Data()
		if err != nil {
			t.Fatalf("read section %s: %v", s.Name, err)
		}
		for len(data)%4 != 0 {
			data = append(data, 0)
		}
		segs = append(segs, s31Segment{addr: uint32(s.Addr), data: data})
	}
	sort.Slice(segs, func(i, j int) bool { return segs[i].addr < segs[j].addr })
	return uint32(f.Entry), segs, ranges
}

// assertS31SRAMLayout verifies that the whole allocated footprint stays inside
// the single DIRAM window without section overlaps and that the heap markers
// are consistent with the linker script.
func assertS31SRAMLayout(t *testing.T, elfPath string) {
	t.Helper()
	_, _, ranges := parseS31Image(t, elfPath)
	sort.Slice(ranges, func(i, j int) bool { return ranges[i][0] < ranges[j][0] })
	for i, r := range ranges {
		if r[0] < s31SRAMStart || r[1] > s31SRAMEnd {
			t.Fatalf("section [%#x, %#x) escapes the S31 SRAM window", r[0], r[1])
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
	if !okStart || !okEnd || heapStart >= heapEnd || heapEnd > s31SRAMEnd {
		t.Fatalf("heap markers inconsistent: _heapStart=%#x(%v) _heapEnd=%#x(%v)",
			heapStart, okStart, heapEnd, okEnd)
	}
	if stack, ok := syms["__stack"]; ok && (stack <= s31SRAMStart || stack > s31SRAMEnd) {
		t.Fatalf("__stack=%#x escapes the S31 SRAM window", stack)
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

// buildS31FlashImage writes the ESP image format esp-emu loads: a v1 image
// with chip id 0x0020 and an appended SHA-256, merged into a 4 MiB flash dump
// at the S31 boot offset.
func buildS31FlashImage(t *testing.T, elfPath, imgPath string) {
	t.Helper()
	entry, segs, _ := parseS31Image(t, elfPath)
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
	hdr = binary.LittleEndian.AppendUint16(hdr, s31ChipID)
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
	copy(flash[s31FlashOff:], img.Bytes())
	if err := os.WriteFile(imgPath, flash, 0o644); err != nil {
		t.Fatal(err)
	}
}

// runS31Image boots the ELF under esp-emu, rejects the baremetal runtime's
// fatal/panic output in the raw stream and requires the wanted markers in the
// filtered output.
func runS31Image(t *testing.T, emu, elfPath string, want []string) {
	t.Helper()
	imgPath := filepath.Join(t.TempDir(), "flash.bin")
	buildS31FlashImage(t, elfPath, imgPath)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, emu, "--chip", "esp32s31", "--firmware", imgPath, "--timeout", "15s")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		t.Fatalf("esp-emu failed: %v\n%s", err, s31OutputTail(buf.String()))
	}
	// The raw scan mirrors the bare-metal guard in the shared embed suite:
	// emulators can exit successfully after a runtime trap, so the unfiltered
	// output is checked before any filter can discard diagnostics.
	for _, line := range strings.Split(buf.String(), "\n") {
		line = strings.TrimSpace(line)
		if line == "fatal error" || strings.HasPrefix(line, "fatal error:") || strings.HasPrefix(line, "panic:") {
			t.Fatalf("emulator runtime failure: %s\n%s", line, s31OutputTail(buf.String()))
		}
	}
	filtered := cltest.FilterEmulatorOutput(buf.String())
	for _, w := range want {
		if !strings.Contains(filtered, w) {
			t.Fatalf("missing output marker %q\n%s", w, s31OutputTail(filtered))
		}
	}
}

func s31OutputTail(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > 20 {
		lines = lines[len(lines)-20:]
	}
	return strings.Join(lines, "\n")
}
