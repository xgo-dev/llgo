package wasmdebug

import (
	"bytes"
	"testing"
)

func TestReplaceDataSlot(t *testing.T) {
	slot := PCLNIdentityPlaceholder()
	value := bytes.Repeat([]byte{0x42}, len(slot))
	for _, flags := range []byte{0, 1, 2} {
		payload := []byte{1, flags}
		if flags == 2 {
			payload = append(payload, 0)
		}
		if flags != 1 {
			payload = append(payload, 0x41, 0x00, 0x0b)
		}
		payload = appendULEB32(payload, uint32(len(slot)+2))
		payload = append(payload, 0x12)
		payload = append(payload, slot[:]...)
		payload = append(payload, 0x34)
		module := append([]byte(nil), wasmHeader...)
		module = append(module, 11)
		module = appendULEB32(module, uint32(len(payload)))
		module = append(module, payload...)
		before := bytes.Clone(module)
		got, err := ReplaceData(module, "identity", slot[:], value)
		if err != nil {
			t.Fatal(err)
		}
		want := bytes.Replace(before, slot[:], value, 1)
		if !bytes.Equal(got, want) || !bytes.Equal(module, before) {
			t.Fatal("changed bytes outside the reserved slot or modified input")
		}
		if _, err := ReplaceData(got, "identity", slot[:], value); err == nil {
			t.Fatal("rewrote a slot without its expected content")
		}
		duplicate := append([]byte(nil), module...)
		duplicate = append(duplicate, module[len(wasmHeader):]...)
		if _, err := ReplaceData(duplicate, "identity", slot[:], value); err == nil {
			t.Fatal("accepted ambiguous slots")
		}
		for n := 0; n < len(module); n++ {
			if _, err := ReplaceData(module[:n], "identity", slot[:], value); err == nil {
				t.Fatalf("accepted truncated module at %d", n)
			}
		}
	}
}
