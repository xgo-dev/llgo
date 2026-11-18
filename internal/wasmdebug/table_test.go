package wasmdebug

import (
	"reflect"
	"testing"
)

func TestFunctionTable(t *testing.T) {
	// Two active segments; the second explicitly selects table zero and
	// overwrites slot 2. Function indices are unrelated to table indices.
	payload := []byte{2, 0, 0x41, 1, 0x0b, 2, 7, 9, 2, 0, 0x41, 2, 0x0b, 0, 1, 11}
	module := appendSection(append([]byte(nil), wasmHeader...), 9, payload)
	got, err := FunctionTable(module)
	if err != nil || !reflect.DeepEqual(got, map[uint32]uint32{1: 7, 2: 11}) {
		t.Fatalf("table = %v, %v", got, err)
	}
	if got, err := FunctionTable(appendCustomSection(wasmHeader, "name", nil)); err != nil || got != nil {
		t.Fatalf("absent table = %v, %v", got, err)
	}
	for i := 0; i < len(payload); i++ {
		if _, err := FunctionTable(appendSection(wasmHeader, 9, payload[:i])); err == nil {
			t.Fatalf("accepted truncated elements at %d", i)
		}
	}
	if _, err := FunctionTable(appendSection(module, 9, []byte{0})); err == nil {
		t.Fatal("accepted duplicate element section")
	}
	if _, err := FunctionTable([]byte("invalid")); err == nil {
		t.Fatal("accepted invalid module")
	}
	for _, raw := range [][]byte{
		{1, 0, 0x42, 0x80, 0x80, 0x80, 0x80, 0x10, 0x0b, 1, 9}, // table64 offset outside supported index range
		{1, 1},                         // passive segment: no initial function identity
		{1, 4},                         // expression elements require a different reader
		{1, 2, 1},                      // another table
		{1, 0, 0x23, 0, 0x0b, 1, 9},    // imported global offset
		{1, 0, 0x41, 0x7f, 0x0b, 1, 9}, // negative offset
		{1, 0, 0x41, 0xff, 0xff, 0xff, 0xff, 8, 0x0b, 1, 9}, // signed overflow
		{1, 0, 0x41, 1, 0x01, 1, 9},                         // not an end opcode
		{1, 2, 0, 0x41, 1, 0x0b, 1, 1, 9},                   // not funcref
		{1, 0, 0x41, 1, 0x0b, 2, 0x80, 0x80},                // truncated function index
		{0, 1},                                              // trailing bytes
	} {
		if _, err := FunctionTable(appendSection(wasmHeader, 9, raw)); err == nil {
			t.Errorf("accepted unsupported/invalid elements %x", raw)
		}
	}
	// Signed LEB encodes positive 64 as c0 00, not 40 (-64).
	got, err = FunctionTable(appendSection(wasmHeader, 9, []byte{1, 0, 0x41, 0xc0, 0, 0x0b, 1, 9}))
	if err != nil || got[64] != 9 {
		t.Fatalf("multibyte offset = %v, %v", got, err)
	}
	got, err = FunctionTable(appendSection(wasmHeader, 9, []byte{1, 0, 0x42, 0xc0, 0, 0x0b, 1, 9}))
	if err != nil || got[64] != 9 {
		t.Fatalf("table64 offset = %v, %v", got, err)
	}
}
