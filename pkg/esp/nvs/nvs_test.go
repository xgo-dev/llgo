//go:build !llgo.idf

// Host-build tests. They exercise what a host build can exercise: the stub
// path (no NVS layer in this build), the argument validation the Go layer
// does before any binding runs, the closed-handle guards, the capacity
// contract and the error segments. Device behavior - the flash round-trip,
// the entry iterator, the recovery recipe - is verified under esp-emu by
// the _demo/esp-idf-nvs acceptance probe, not here.
package nvs

import (
	"errors"
	"testing"

	"github.com/xgo-dev/llgo/pkg/esp/esperr"
)

// TestErrorSegments pins the segment plan the esperr family agreed on: this
// package's own conditions in 0x75xx (the lowest unclaimed segment - 0x76/77
// bt+A2DP, 0x78 reserved classic-audio, 0x79 opus, 0x7A wakenet, 0x7B ble,
// 0x7C esperr meta, 0x7D/7E ws/tls, 0x7F stubs) and the raw esp_err
// pass-through carrying the NVS family and the generic codes unnamed-shape
// (named by ErrName, code stable).
func TestErrorSegments(t *testing.T) {
	for _, c := range []struct {
		err  error
		code int32
	}{
		// The binding's own capacity conditions, the 0x75xx segment.
		{ErrTooLong, 0x7501},
		{ErrTooManyKeys, 0x7502},
		// The stub tag sits above every real code space.
		{&esperr.Error{Code: errUnsupported}, 0x7F00},
	} {
		if got := esperr.Code(c.err); got != c.code {
			t.Errorf("esperr.Code(%v) = %#x, want %#x", c.err, got, c.code)
		}
	}
	// Raw esp_err passes through with its NVS-family code intact - the
	// code is the stable, inspectable identifier.
	if got := esperr.Code(errOf(0x1102)); got != 0x1102 {
		t.Errorf("esperr.Code(errOf(NVS_NOT_FOUND)) = %#x, want 0x1102", got)
	}
	if got := esperr.Code(errOf(ErrInvalidArg)); got != ErrInvalidArg {
		t.Errorf("esperr.Code(errOf(INVALID_ARG)) = %#x, want %#x", got, ErrInvalidArg)
	}
	// The family sentinels keep their meaning across the family.
	if !errors.Is(errOf(9), esperr.ErrClosed) {
		t.Error("errors.Is(errOf(9), esperr.ErrClosed) = false, want true")
	}
}

// TestStubHonesty pins the host-build contract: with no NVS layer behind
// it, every binding that reaches C reports the stub tag 0x7F00 - a code
// outside every real esp_err range, so a stub result can never be mistaken
// for device behavior. The Go-side guards (invalid names, closed handles)
// fire first and keep their own codes.
func TestStubHonesty(t *testing.T) {
	if e := Init(); esperr.Code(e) != errUnsupported {
		t.Errorf("Init() = %v, want the stub tag %#x", e, errUnsupported)
	}
	if e := Deinit(); esperr.Code(e) != errUnsupported {
		t.Errorf("Deinit() = %v, want the stub tag %#x", e, errUnsupported)
	}
	if _, e := Open("wifi"); esperr.Code(e) != errUnsupported {
		t.Errorf("Open() = %v, want the stub tag %#x", e, errUnsupported)
	}
	if _, e := ReadStats(); esperr.Code(e) != errUnsupported {
		t.Errorf("ReadStats() = %v, want the stub tag %#x", e, errUnsupported)
	}
	// A live-looking handle (one Open had succeeded) reaches the stub.
	ns := &Namespace{handle: 1, name: "wifi"}
	for _, tc := range []struct {
		name string
		e    error
	}{
		{"GetU8", err2(ns.GetU8("k"))},
		{"GetI32", err2(ns.GetI32("k"))},
		{"GetU64", err2(ns.GetU64("k"))},
		{"GetString", err2(ns.GetString("k"))},
		{"GetBlob", err2(ns.GetBlob("k"))},
		{"SetU8", ns.SetU8("k", 1)},
		{"SetI32", ns.SetI32("k", 1)},
		{"SetU64", ns.SetU64("k", 1)},
		{"SetString", ns.SetString("k", "v")},
		{"SetBlob", ns.SetBlob("k", []byte{1})},
		{"Delete", ns.Delete("k")},
		{"Commit", ns.Commit()},
		{"Keys", err2(ns.Keys())},
	} {
		if esperr.Code(tc.e) != errUnsupported {
			t.Errorf("%s on a live handle = %v, want the stub tag %#x", tc.name, tc.e, errUnsupported)
		}
	}
}

// TestHandleGuards pins the i2c idiom: methods on a nil or zero-value
// Namespace (never opened, or after Close) refuse with ESP_ERR_INVALID_ARG
// before anything crosses into C, while Close itself reports the family's
// ErrClosed and stays idempotent.
func TestHandleGuards(t *testing.T) {
	var zero *Namespace
	if e := zero.SetU8("k", 1); esperr.Code(e) != ErrInvalidArg {
		t.Errorf("nil SetU8 = %v, want ESP_ERR_INVALID_ARG", e)
	}
	if e := zero.Commit(); esperr.Code(e) != ErrInvalidArg {
		t.Errorf("nil Commit = %v, want ESP_ERR_INVALID_ARG", e)
	}
	if _, e := zero.GetU8("k"); esperr.Code(e) != ErrInvalidArg {
		t.Errorf("nil GetU8 = %v, want ESP_ERR_INVALID_ARG", e)
	}
	if _, e := zero.Keys(); esperr.Code(e) != ErrInvalidArg {
		t.Errorf("nil Keys = %v, want ESP_ERR_INVALID_ARG", e)
	}
	if e := zero.Close(); !errors.Is(e, esperr.ErrClosed) {
		t.Errorf("nil Close = %v, want esperr.ErrClosed", e)
	}
	ns := &Namespace{handle: 1, name: "wifi"}
	if e := ns.Close(); e != nil {
		t.Errorf("first Close = %v, want nil", e)
	}
	if ns.handle != 0 {
		t.Error("Close left the handle set")
	}
	if e := ns.Close(); !errors.Is(e, esperr.ErrClosed) {
		t.Errorf("second Close = %v, want esperr.ErrClosed", e)
	}
	if e := ns.SetU8("k", 1); esperr.Code(e) != ErrInvalidArg {
		t.Errorf("closed SetU8 = %v, want ESP_ERR_INVALID_ARG", e)
	}
}

// TestNameValidation pins the NVS name rules the Go layer enforces before
// anything crosses into C: 1..MaxKeyLen bytes, no NUL byte, keys and
// namespaces alike. Every refusal carries ESP_ERR_INVALID_ARG.
func TestNameValidation(t *testing.T) {
	for _, s := range []string{"", "k", "fifteen-char-ok", "sixteen-char-key", "nul\x00key"} {
		want := len(s) >= 1 && len(s) <= MaxKeyLen && !containsNul(s)
		if got := validName(s); got != want {
			t.Errorf("validName(%q) = %v, want %v", s, got, want)
		}
	}
	ns := &Namespace{handle: 1, name: "wifi"}
	if _, e := ns.GetU8("sixteen-char-key"); esperr.Code(e) != ErrInvalidArg {
		t.Errorf("long key GetU8 = %v, want ESP_ERR_INVALID_ARG", e)
	}
	if _, e := ns.GetU8(""); esperr.Code(e) != ErrInvalidArg {
		t.Errorf("empty key GetU8 = %v, want ESP_ERR_INVALID_ARG", e)
	}
	if e := ns.SetString("k", "va\x00lue"); esperr.Code(e) != ErrInvalidArg {
		t.Errorf("NUL value SetString = %v, want ESP_ERR_INVALID_ARG", e)
	}
	if _, e := Open("sixteen-char-key"); esperr.Code(e) != ErrInvalidArg {
		t.Errorf("long namespace Open = %v, want ESP_ERR_INVALID_ARG", e)
	}
	if _, e := Open(""); esperr.Code(e) != ErrInvalidArg {
		t.Errorf("empty namespace Open = %v, want ESP_ERR_INVALID_ARG", e)
	}
}

// TestCapacityContract pins the constants that mirror the glue's static
// storage (LG_NVS_VALUE_CAP, LG_NVS_MAX_KEYS in nvs_glue.c): the glue is
// the storage, these are the documented contract, and a drift between the
// two files would turn "binding capacity" errors into lies.
func TestCapacityContract(t *testing.T) {
	if MaxValue != 512 || MaxKeys != 32 || MaxKeyLen != 15 {
		t.Errorf("capacity constants drifted: MaxValue=%d MaxKeys=%d MaxKeyLen=%d",
			MaxValue, MaxKeys, MaxKeyLen)
	}
}

// TestErrNameTable checks the ErrName vocabulary: the NVS family as IDF
// v6 spells it (ESP_ERR_NVS_BASE 0x1100), the generic codes the glue can
// return, this package's own segment, and the family's hex fallback.
func TestErrNameTable(t *testing.T) {
	for _, c := range []struct {
		code int32
		want string
	}{
		{0, "ESP_OK"},
		{0x102, "ESP_ERR_INVALID_ARG"},
		{0x1101, "ESP_ERR_NVS_NOT_INITIALIZED"},
		{0x1102, "ESP_ERR_NVS_NOT_FOUND"},
		{0x1103, "ESP_ERR_NVS_TYPE_MISMATCH"},
		{0x110D, "ESP_ERR_NVS_NO_FREE_PAGES"},
		{0x1110, "ESP_ERR_NVS_NEW_VERSION_FOUND"},
		{0x7501, "NVS_ERR_TOO_LONG"},
		{0x7502, "NVS_ERR_TOO_MANY_KEYS"},
		{0x7F00, "NVS_ERR_UNSUPPORTED"},
		{0x555555, "esp_err 0x555555"},
	} {
		if got := ErrName(c.code); got != c.want {
			t.Errorf("ErrName(%#x) = %q, want %q", c.code, got, c.want)
		}
	}
}

func err2(_ any, e error) error { return e }

func containsNul(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == 0 {
			return true
		}
	}
	return false
}
