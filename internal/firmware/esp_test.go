package firmware

import (
	"bytes"
	"testing"
)

func TestESP32C3FlashSectionPlacement(t *testing.T) {
	code, ok := esp32c3FlashOffset(0x42000004)
	if !ok {
		t.Fatal("code address was not mapped to flash")
	}
	data, ok := esp32c3FlashOffset(0x3c000007)
	if !ok {
		t.Fatal("read-only data address was not mapped to flash")
	}
	out := bytes.NewBuffer([]byte{0xe9, 1})
	err := appendESP32C3FlashSections(out, []*espImageSegment{
		{addr: data, data: []byte{11, 12}},
		{addr: code, data: []byte{1, 2, 3}},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{0xe9, 1, 0xff, 0xff, 1, 2, 3, 11, 12}
	if !bytes.Equal(out.Bytes(), want) {
		t.Fatalf("image = %x, want %x", out.Bytes(), want)
	}
}

func TestESP32C3FlashSectionBounds(t *testing.T) {
	for _, offset := range []uint32{0, esp32c3FlashSize - 1} {
		out := bytes.NewBuffer([]byte{0xe9})
		if err := appendESP32C3FlashSections(out, []*espImageSegment{{addr: offset, data: []byte{1, 2}}}); err == nil {
			t.Errorf("accepted two bytes at offset %#x with a boot image present", offset)
		}
	}
}
