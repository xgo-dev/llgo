package main

// NVS acceptance probe: drive the pkg/esp/nvs binding through its whole
// surface and prove flash persistence across soft restarts under esp-emu.
//
// The probe is restart-driven and stage-keyed: the stage counter itself
// lives in NVS (namespace "eternal", key "stage"), so each boot reads where
// the previous one left off:
//
//	stage 0 (fresh)   write phase: every Set*/Get* shape round-tripped,
//	                  the error paths (wrong type, missing key, over-long
//	                  key, NUL value, over-long read, delete, empty and
//	                  never-written namespaces), the wifi credentials
//	                  staged and committed - then stage = 1, Commit,
//	                  esp_restart.
//	stage 1           verify phase: everything read back off flash and
//	                  compared, the deleted key still absent - then
//	                  stage = 2, Commit, esp_restart.
//	stage 2           final phase: re-verify after a second restart and
//	                  gate the run on PASS.
//
// If the flash did not persist, stage would come back missing on every
// boot and the probe would loop in stage 0 - the emulator timeout makes
// that visible instead of hiding it.
//
// Every line starts with "[go] nvs"; golden/emu-nvs-probe.txt pins the
// exact ones and floats the ledger numbers that legitimately vary.

import (
	"errors"

	"github.com/xgo-dev/llgo/pkg/esp/esperr"
	"github.com/xgo-dev/llgo/pkg/esp/nvs"
)

import _ "unsafe" // for the //go:linkname declarations below

// restart is the IDF esp_restart: a soft reset. On the target the flash
// keeps its contents; the probe's whole persistence claim rests on that.
//
//go:linkname restart C.esp_restart
func restart()

// heapPrint is the demo glue's ledger line: both IDF pools at one probe
// stage ([c] nvs heap ...), for the report's memory note.
//
//go:linkname heapPrint C.llgo_demo_heap
func heapPrint(tag *byte, n int32)

func markHeap(stage string) {
	b := []byte(stage)
	heapPrint(&b[0], int32(len(b)))
}

// codeHex renders an error's pipeline code as 0x.., the shape the golden
// pins (no strconv on this target).
func codeHex(e error) string {
	v := uint32(esperr.Code(e))
	const digits = "0123456789ABCDEF"
	buf := [6]byte{'0', 'x', 0, 0, 0, 0}
	for i := 0; i < 4; i++ {
		buf[5-i] = digits[(v>>(uint(i)*4))&0xF]
	}
	return string(buf[:])
}

// codeOf renders "0x.. (NAME)" or "nil" - one compact, greppable token per
// error-path check.
func codeOf(e error) string {
	if e == nil {
		return "nil"
	}
	if ee, ok := e.(*esperr.Error); ok {
		return codeHex(ee) + " " + ee.Name
	}
	return e.Error()
}

// expect fails the phase loudly when an error path did not return the code
// the IDF contract promises.
func expect(name string, got error, want int32) bool {
	if esperr.Code(got) != want {
		println("[go] nvs FAIL", name, "err =", codeOf(got))
		return false
	}
	println("[go] nvs", name, "err =", codeOf(got))
	return true
}

// the write-phase payload: a wifi credential pair plus one value of every
// numeric/string/blob shape, under namespaces a real layout would use.
const (
	wantSSID = "llgo-home"
	wantPass = "hunter2-secret"
	wantU8   = 200
	wantI32  = -123456
	wantU64  = uint64(1099511628263) // 1<<40 + 7
	wantStr  = "llgo-nvs-value"
)

// lcgNext is a deterministic byte stream for the blob (and the over-long
// probe string): the same sequence on every run, so the golden stays exact.
func lcgNext(state *uint32) byte {
	*state = *state*1664525 + 1013904223
	return byte(*state >> 24)
}

func main() {
	// Bring the NVS layer up. On a fresh flash this formats the partition;
	// on the second and third boots the layer is already initialized and
	// Init is an honest no-op (the idempotency contract).
	if e := nvs.Init(); e != nil {
		println("[go] nvs FAIL init err =", codeOf(e))
		return
	}

	// The stage key decides the phase. NVS_ERR_NOT_FOUND - the fresh-flash
	// shape, the namespace never written - means stage 0. Anything else is
	// a broken layer, reported loudly.
	eternal, e := nvs.Open("eternal")
	if e != nil {
		println("[go] nvs FAIL open-eternal err =", codeOf(e))
		return
	}
	defer eternal.Close()
	stage, e := eternal.GetU8("stage")
	if e != nil {
		// The fresh-flash shape - namespace never written, key absent - is
		// stage 0; the code check is the pin. Anything else is a broken
		// layer, reported loudly.
		if !expect("stage0-fresh", e, 0x1102) {
			return
		}
		stage = 0
	}

	switch stage {
	case 0:
		if !phase0(eternal) {
			return
		}
		// Arm the next boot BEFORE restarting: the stage key itself is the
		// persistence proof - boot 2 only reaches the verify phase if this
		// Commit survived the reset.
		if e := eternal.SetU8("stage", 1); e != nil || eternal.Commit() != nil {
			println("[go] nvs FAIL stage1")
			return
		}
		markHeap("stage0-done")
		println("[go] nvs stage0 done")
		restart()
	case 1:
		if !phase1(eternal) {
			return
		}
		if e := eternal.SetU8("stage", 2); e != nil || eternal.Commit() != nil {
			println("[go] nvs FAIL stage2")
			return
		}
		println("[go] nvs stage1 done")
		restart()
	case 2:
		if !phase2(eternal) {
			return
		}
		markHeap("done")
		println("[go] nvs PASS")
		println("[go] nvs done")
	default:
		println("[go] nvs FAIL stuck stage =", int32(stage))
	}
}

// phase0 is the fresh-flash write phase: the whole API surface, the error
// paths, then the wifi credentials - everything a later boot will verify.
func phase0(eternal *nvs.Namespace) bool {
	// --- the wifi credential pair, the layout a real provisioning app
	// would use: one namespace, two strings, one commit.
	wifi, e := nvs.Open("wifi")
	if e != nil {
		println("[go] nvs FAIL open-wifi err =", codeOf(e))
		return false
	}
	defer wifi.Close()
	if e := wifi.SetString("ssid", wantSSID); e != nil {
		println("[go] nvs FAIL set-ssid err =", codeOf(e))
		return false
	}
	if e := wifi.SetString("pass", wantPass); e != nil {
		println("[go] nvs FAIL set-pass err =", codeOf(e))
		return false
	}
	if e := wifi.Commit(); e != nil {
		println("[go] nvs FAIL commit-wifi err =", codeOf(e))
		return false
	}
	if v, e := wifi.GetString("ssid"); e != nil || v != wantSSID {
		println("[go] nvs FAIL readback-ssid v =", v, "err =", codeOf(e))
		return false
	}
	println("[go] nvs wifi ok ssid pass committed")

	// --- one value of every shape, round-tripped immediately.
	app, e := nvs.Open("app")
	if e != nil {
		println("[go] nvs FAIL open-app err =", codeOf(e))
		return false
	}
	defer app.Close()

	if e := app.SetU8("u8", wantU8); e != nil {
		println("[go] nvs FAIL set-u8 err =", codeOf(e))
		return false
	}
	if v, e := app.GetU8("u8"); e != nil || v != wantU8 {
		println("[go] nvs FAIL rt-u8 v =", int32(v), "err =", codeOf(e))
		return false
	}
	if e := app.SetI32("i32", wantI32); e != nil {
		println("[go] nvs FAIL set-i32 err =", codeOf(e))
		return false
	}
	if v, e := app.GetI32("i32"); e != nil || v != wantI32 {
		println("[go] nvs FAIL rt-i32 err =", codeOf(e))
		return false
	}
	if e := app.SetU64("u64", wantU64); e != nil {
		println("[go] nvs FAIL set-u64 err =", codeOf(e))
		return false
	}
	if v, e := app.GetU64("u64"); e != nil || v != wantU64 {
		println("[go] nvs FAIL rt-u64 err =", codeOf(e))
		return false
	}
	if e := app.SetString("str", wantStr); e != nil {
		println("[go] nvs FAIL set-str err =", codeOf(e))
		return false
	}
	if v, e := app.GetString("str"); e != nil || v != wantStr {
		println("[go] nvs FAIL rt-str v =", v, "err =", codeOf(e))
		return false
	}
	state := uint32(0x12345678)
	blob := make([]byte, 64)
	for i := range blob {
		blob[i] = lcgNext(&state)
	}
	if e := app.SetBlob("blob", blob); e != nil {
		println("[go] nvs FAIL set-blob err =", codeOf(e))
		return false
	}
	rb, e := app.GetBlob("blob")
	if e != nil || len(rb) != len(blob) {
		println("[go] nvs FAIL rt-blob n =", int32(len(rb)), "err =", codeOf(e))
		return false
	}
	for i := range blob {
		if rb[i] != blob[i] {
			println("[go] nvs FAIL rt-blob at =", int32(i))
			return false
		}
	}
	println("[go] nvs roundtrip ok u8 i32 u64 str blob64")

	// --- the error paths, each pinned to the code this IDF actually
	// returns. The primitive "wrong type" check pins measured behavior,
	// not the doc ideal: nvs_page.cpp's hash-list prefilter indexes items
	// by (namespace, type, key), so an I32 read of a U8-stored key misses
	// the hash and reports ESP_ERR_NVS_NOT_FOUND - the page scan that
	// would return ESP_ERR_NVS_TYPE_MISMATCH never runs (measured, IDF
	// v6.3-dev, emu run 1 of this probe).
	if e := app.SetU8("t8", 7); e != nil {
		println("[go] nvs FAIL set-t8 err =", codeOf(e))
		return false
	}
	if !expect("wrong-type", firstErr(app.GetI32("t8")), 0x1102) {
		return false
	}
	if !expect("missing-key", firstErr(app.GetU8("absent")), 0x1102) {
		return false
	}
	if !expect("key-too-long", app.SetU8("sixteen-char-key", 1), 0x102) {
		return false
	}
	if !expect("key-empty", app.SetU8("", 1), 0x102) {
		return false
	}
	if !expect("value-nul", app.SetString("bad", "a\x00b"), 0x102) {
		return false
	}

	// A write-side no-cap check: 600 printable bytes stage fine, then read
	// back over the binding's read cap (MaxValue 512) - refused as the
	// binding's own ErrTooLong, the value left intact, then cleaned up.
	// (Printable only: SetString honestly refuses NUL-bearing values.)
	big := make([]byte, 600)
	for i := range big {
		big[i] = 'a' + lcgNext(&state)%26
	}
	if e := app.SetString("big", string(big)); e != nil {
		println("[go] nvs FAIL set-big err =", codeOf(e))
		return false
	}
	_, e = app.GetString("big")
	if !errors.Is(e, nvs.ErrTooLong) {
		println("[go] nvs FAIL big-too-long err =", codeOf(e))
		return false
	}
	println("[go] nvs read-cap ok 600-in 512-max err =", codeOf(e))
	if e := app.Delete("big"); e != nil {
		println("[go] nvs FAIL del-big err =", codeOf(e))
		return false
	}

	// Delete: staged, committed, and verified absent right away; deleting
	// the same key again reports NOT_FOUND.
	if e := app.SetU8("gone", 1); e != nil {
		println("[go] nvs FAIL set-gone err =", codeOf(e))
		return false
	}
	if e := app.Delete("gone"); e != nil || app.Commit() != nil {
		println("[go] nvs FAIL del-gone err =", codeOf(e))
		return false
	}
	if !expect("deleted-key", firstErr(app.GetU8("gone")), 0x1102) {
		return false
	}
	if !expect("delete-absent", app.Delete("absent"), 0x1102) {
		return false
	}
	if e := app.Commit(); e != nil {
		println("[go] nvs FAIL commit-app err =", codeOf(e))
		return false
	}

	// The never-written namespace: Open succeeds (read-write creates on
	// first write), but the entry iterator finds nothing - the
	// "namespace does not exist" shape, ESP_ERR_NVS_NOT_FOUND.
	ghost, e := nvs.Open("ghost")
	if e != nil {
		println("[go] nvs FAIL open-ghost err =", codeOf(e))
		return false
	}
	if _, e := ghost.Keys(); !expect("namespace-void", e, 0x1102) {
		ghost.Close()
		return false
	}
	ghost.Close()

	// Keys(): the two written namespaces, sorted so the golden is order-
	// independent (the iterator reports physical slot order).
	keys, e := app.Keys()
	if e != nil {
		println("[go] nvs FAIL keys-app err =", codeOf(e))
		return false
	}
	sortStrings(keys)
	println("[go] nvs keys app n =", int32(len(keys)), join(keys))
	wkeys, e := wifi.Keys()
	if e != nil {
		println("[go] nvs FAIL keys-wifi err =", codeOf(e))
		return false
	}
	sortStrings(wkeys)
	println("[go] nvs keys wifi n =", int32(len(wkeys)), join(wkeys))

	// The flash-wide ledger (observation only - the golden floats it).
	s, e := nvs.ReadStats()
	if e != nil {
		println("[go] nvs FAIL stats err =", codeOf(e))
		return false
	}
	println("[go] nvs stats used =", s.Used, "free =", s.Free,
		"avail =", s.Available, "total =", s.Total, "ns =", s.Namespaces)
	return true
}

// phase1 is the first verify phase: everything phase0 committed is read
// back off flash after one soft restart.
func phase1(eternal *nvs.Namespace) bool {
	return verify(eternal, 1)
}

// phase2 re-verifies after the second restart and closes the run.
func phase2(eternal *nvs.Namespace) bool {
	return verify(eternal, 2)
}

// verify reads the committed state back and compares it; boots counts the
// restarts the state has survived.
func verify(eternal *nvs.Namespace, boots int32) bool {
	wifi, e := nvs.Open("wifi")
	if e != nil {
		println("[go] nvs FAIL open-wifi err =", codeOf(e))
		return false
	}
	defer wifi.Close()
	ssid, e := wifi.GetString("ssid")
	if e != nil || ssid != wantSSID {
		println("[go] nvs FAIL verify-ssid v =", ssid, "err =", codeOf(e))
		return false
	}
	pass, e := wifi.GetString("pass")
	if e != nil || pass != wantPass {
		println("[go] nvs FAIL verify-pass err =", codeOf(e))
		return false
	}

	app, e := nvs.Open("app")
	if e != nil {
		println("[go] nvs FAIL open-app err =", codeOf(e))
		return false
	}
	defer app.Close()
	if v, e := app.GetU8("u8"); e != nil || v != wantU8 {
		println("[go] nvs FAIL verify-u8 err =", codeOf(e))
		return false
	}
	if v, e := app.GetI32("i32"); e != nil || v != wantI32 {
		println("[go] nvs FAIL verify-i32 err =", codeOf(e))
		return false
	}
	if v, e := app.GetU64("u64"); e != nil || v != wantU64 {
		println("[go] nvs FAIL verify-u64 err =", codeOf(e))
		return false
	}
	if v, e := app.GetString("str"); e != nil || v != wantStr {
		println("[go] nvs FAIL verify-str v =", v, "err =", codeOf(e))
		return false
	}
	state := uint32(0x12345678)
	blob := make([]byte, 64)
	for i := range blob {
		blob[i] = lcgNext(&state)
	}
	rb, e := app.GetBlob("blob")
	if e != nil || len(rb) != len(blob) {
		println("[go] nvs FAIL verify-blob err =", codeOf(e))
		return false
	}
	for i := range blob {
		if rb[i] != blob[i] {
			println("[go] nvs FAIL verify-blob at =", int32(i))
			return false
		}
	}
	if !expect("gone-still-gone", firstErr(app.GetU8("gone")), 0x1102) {
		return false
	}
	keys, e := app.Keys()
	if e != nil {
		println("[go] nvs FAIL keys-app err =", codeOf(e))
		return false
	}
	sortStrings(keys)
	println("[go] nvs keys app n =", int32(len(keys)), join(keys))
	s, e := nvs.ReadStats()
	if e != nil {
		println("[go] nvs FAIL stats err =", codeOf(e))
		return false
	}
	println("[go] nvs stats used =", s.Used, "free =", s.Free,
		"avail =", s.Available, "total =", s.Total, "ns =", s.Namespaces)
	println("[go] nvs verify ok boots =", boots)
	return true
}

// sortStrings orders the key list (insertion sort; no sort package on this
// image) so the golden does not depend on NVS slot order.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

func join(s []string) string {
	out := ""
	for i, v := range s {
		if i > 0 {
			out += " "
		}
		out += v
	}
	return out
}

func firstErr(_ any, e error) error { return e }
