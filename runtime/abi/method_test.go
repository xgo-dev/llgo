package abi

import (
	"testing"
	"unsafe"
)

func TestUncommonMethod(t *testing.T) {
	var table struct {
		uncommon UncommonType
		padding  [13]byte
		methods  [2]Method
	}
	table.uncommon.Mcount = 2
	table.uncommon.Moff = uint32(unsafe.Offsetof(table.methods))
	table.methods[0].Name_ = "Alpha"
	table.methods[1].Name_ = "example.com/private.beta"
	for i := range table.methods {
		got := table.uncommon.Method(i)
		if got != &table.methods[i] || got.Name_ != table.methods[i].Name_ {
			t.Fatalf("method %d did not reference the original metadata", i)
		}
	}
}

func TestUncommonMethodBounds(t *testing.T) {
	for _, tc := range []struct {
		count uint16
		index int
	}{{0, 0}, {2, -1}, {2, 2}, {2, 3}} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("Method(%d) with %d methods did not panic", tc.index, tc.count)
				}
			}()
			(&UncommonType{Mcount: tc.count}).Method(tc.index)
		}()
	}
}
