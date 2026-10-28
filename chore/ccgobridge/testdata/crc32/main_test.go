package main

import "testing"

func TestReferenceCRC(t *testing.T) {
	for _, tc := range []struct {
		poly, want uint32
	}{
		{0xedb88320, 0xcbf43926}, // IEEE check value.
		{0x82f63b78, 0xe3069283}, // CRC32C check value.
	} {
		if got := referenceCRC([]byte("123456789"), tc.poly); got != tc.want {
			t.Fatalf("poly=%08x: got=%08x want=%08x", tc.poly, got, tc.want)
		}
		if got := referenceCRC(nil, tc.poly); got != 0 {
			t.Fatalf("empty CRC=%08x", got)
		}
	}
}
