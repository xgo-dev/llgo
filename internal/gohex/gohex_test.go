package gohex

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestConstructor(t *testing.T) {
	m := NewMemory()
	if a, ok := m.GetStartAddress(); ok != false || a != 0 {
		t.Error("incorrect initial start Address")
	}
	if len(m.GetDataSegments()) != 0 {
		t.Error("incorrect initial data segments")
	}
	if m.extendedAddress != 0 {
		t.Error("incorrect initial data segments")
	}
}

func parseIntelHex(m *Memory, str string) error {
	return m.ParseIntelHex(strings.NewReader(str))
}

func checkErrorType(t *testing.T, err error, et parseErrorType, msg string) {
	if err != nil {
		perr, ok := err.(*parseError)
		if ok == true {
			if perr.errorType != et {
				t.Error(perr.Error())
				t.Error(err)
			}
		} else {
			t.Error(err)
		}
	} else {
		t.Error(msg)
	}
}

func assertParseError(t *testing.T, m *Memory, input string, et parseErrorType, err string) {
	e := parseIntelHex(m, input)
	checkErrorType(t, e, et, err)
}

func TestSyntaxError(t *testing.T) {
	m := NewMemory()
	assertParseError(t, m, "00000001FF\n", _SYNTAX_ERROR, "no colon error")
	assertParseError(t, m, ":qw00000001FF\n", _SYNTAX_ERROR, "no ascii hex error")
	assertParseError(t, m, ":0000001FF\n", _SYNTAX_ERROR, "no odd/even hex error")
}

func TestDataError(t *testing.T) {
	m := NewMemory()
	assertParseError(t, m, ":000000FF\n", _DATA_ERROR, "no line length error")
	assertParseError(t, m, ":02000000FE\n", _DATA_ERROR, "no data length error")
	assertParseError(t, m, "\n", _DATA_ERROR, "no end of file line error")
	assertParseError(t, m, ":000000FF01\n", _DATA_ERROR, "no end of file line error")
	assertParseError(t, m, ":0400000501000000F6\n", _DATA_ERROR, "no end of file line error")
	assertParseError(t, m, ":0400000501000000F6\n:0400000502000000F5\n:00000001FF\n", _DATA_ERROR, "no multiple start Address lines error")
	assertParseError(t, m, ":048000000102030472\n:04800300050607085F\n:00000001FF\n", _DATA_ERROR, "no segments overlap error")
	assertParseError(t, m, ":048000000102030472\n:047FFD000506070866\n:00000001FF\n", _DATA_ERROR, "no segments overlap error")
}

func TestChecksumError(t *testing.T) {
	m := NewMemory()
	assertParseError(t, m, ":00000101FF\n", _CHECKSUM_ERROR, "no checksum error")
	assertParseError(t, m, ":00000001FE\n", _CHECKSUM_ERROR, "no checksum error")
	assertParseError(t, m, ":0000000001\n", _CHECKSUM_ERROR, "no checksum error")
	assertParseError(t, m, ":000000FF02\n", _CHECKSUM_ERROR, "no checksum error")
}

func TestRecordsError(t *testing.T) {
	m := NewMemory()
	assertParseError(t, m, ":00000101FE\n", _RECORD_ERROR, "no eof record error")
	assertParseError(t, m, ":00010001FE\n", _RECORD_ERROR, "no eof record error")
	assertParseError(t, m, ":0100000100FE\n", _RECORD_ERROR, "no eof record error")
	assertParseError(t, m, ":020001040101F7\n", _RECORD_ERROR, "no extended Address record error")
	assertParseError(t, m, ":020100040101F7\n", _RECORD_ERROR, "no extended Address record error")
	assertParseError(t, m, ":03000004010100F7\n", _RECORD_ERROR, "no extended Address record error")
	assertParseError(t, m, ":0400010501010101F2\n", _RECORD_ERROR, "no start Address record error")
	assertParseError(t, m, ":0401000501010101F2\n", _RECORD_ERROR, "no start Address record error")
	assertParseError(t, m, ":050000050101010100F2\n", _RECORD_ERROR, "no start Address record error")
}

func TestAddress(t *testing.T) {
	m := NewMemory()
	err := parseIntelHex(m, ":020000041234B4\n:0400000501020304ED\n:00000001FF\n")
	if err != nil {
		t.Error("unexpected error: ", err.Error())
	}
	if m.lineNum != 3 {
		t.Error("incorrect lines number")
	}
	if m.extendedAddress != 0x12340000 {
		t.Errorf("incorrect extended Address: %08X", m.extendedAddress)
	}
	if a, ok := m.GetStartAddress(); a != 0x01020304 && ok == true {
		t.Errorf("incorrect start Address: %08X", m.startAddress)
	}
	if len(m.GetDataSegments()) != 0 {
		t.Error("incorrect data segments")
	}
	if m.eofFlag != true {
		t.Error("incorrect eof flag state")
	}
	if m.startFlag != true {
		t.Error("incorrect start flag state")
	}
	err = parseIntelHex(m, ":020000049ABCA4\n:0400000591929394AD\n:00000001FF\n")
	if err != nil {
		t.Error("unexpected error: ", err.Error())
	}
	if m.extendedAddress != 0x9ABC0000 {
		t.Errorf("incorrect extended Address: %08X", m.extendedAddress)
	}
	if a, ok := m.GetStartAddress(); a != 0x91929394 && ok == true {
		t.Errorf("incorrect start Address: %08X", m.startAddress)
	}

	err = parseIntelHex(m, ":020000041234B4\n:02000004234592\n:00000001FF\n")
	if err != nil {
		t.Error("unexpected error: ", err.Error())
	}
	if m.extendedAddress != 0x23450000 {
		t.Errorf("incorrect extended Address: %08X", m.extendedAddress)
	}
}

func TestDataSegments(t *testing.T) {
	m := NewMemory()
	err := parseIntelHex(m, ":048000000102030472\n:04800400050607085E\n:00000001FF\n")
	if err != nil {
		t.Error("unexpected error: ", err.Error())
	}
	if len(m.GetDataSegments()) != 1 {
		t.Errorf("incorrect number of data segments: %v", len(m.GetDataSegments()))
	}
	seg := m.GetDataSegments()[0]
	p := DataSegment{Address: 0x8000, Data: []byte{1, 2, 3, 4, 5, 6, 7, 8}}
	if reflect.DeepEqual(seg, p) == false {
		t.Errorf("incorrect segment: %v != %v", seg, p)
	}

	err = parseIntelHex(m, ":048000000102030472\n:047FFC000506070867\n:00000001FF\n")
	if err != nil {
		t.Error("unexpected error: ", err.Error())
	}
	if len(m.GetDataSegments()) != 1 {
		t.Errorf("incorrect number of data segments: %v", len(m.GetDataSegments()))
	}
	seg = m.GetDataSegments()[0]
	p = DataSegment{Address: 0x7FFC, Data: []byte{5, 6, 7, 8, 1, 2, 3, 4}}
	if reflect.DeepEqual(seg, p) == false {
		t.Errorf("incorrect segment: %v != %v", seg, p)
	}

	err = parseIntelHex(m, ":048000000102030472\n:04800800050607085A\n:00000001FF\n")
	if err != nil {
		t.Error("unexpected error: ", err.Error())
	}
	if len(m.GetDataSegments()) != 2 {
		t.Errorf("incorrect number of data segments: %v", len(m.GetDataSegments()))
	}
	seg = m.GetDataSegments()[0]
	p = DataSegment{Address: 0x8000, Data: []byte{1, 2, 3, 4}}
	if reflect.DeepEqual(seg, p) == false {
		t.Errorf("incorrect segment: %v != %v", seg, p)
	}
	seg = m.GetDataSegments()[1]
	p = DataSegment{Address: 0x8008, Data: []byte{5, 6, 7, 8}}
	if reflect.DeepEqual(seg, p) == false {
		t.Errorf("incorrect segment: %v != %v", seg, p)
	}

	err = parseIntelHex(m, ":04800800050607085A\n:048000000102030472\n\n:00000001FF\n")
	if err != nil {
		t.Error("unexpected error: ", err.Error())
	}
	if len(m.GetDataSegments()) != 2 {
		t.Errorf("incorrect number of data segments: %v", len(m.GetDataSegments()))
	}
	seg = m.GetDataSegments()[0]
	p = DataSegment{Address: 0x8000, Data: []byte{1, 2, 3, 4}}
	if reflect.DeepEqual(seg, p) == false {
		t.Errorf("incorrect segment: %v != %v", seg, p)
	}
	seg = m.GetDataSegments()[1]
	p = DataSegment{Address: 0x8008, Data: []byte{5, 6, 7, 8}}
	if reflect.DeepEqual(seg, p) == false {
		t.Errorf("incorrect segment: %v != %v", seg, p)
	}

	err = parseIntelHex(m, ":020000041000EA\n:048000000102030472\n:04800800050607085A\n:00000001FF\n")
	if err != nil {
		t.Error("unexpected error: ", err.Error())
	}
	if len(m.GetDataSegments()) != 2 {
		t.Errorf("incorrect number of data segments: %v", len(m.GetDataSegments()))
	}
	seg = m.GetDataSegments()[0]
	p = DataSegment{Address: 0x10008000, Data: []byte{1, 2, 3, 4}}
	if reflect.DeepEqual(seg, p) == false {
		t.Errorf("incorrect segment: %v != %v", seg, p)
	}
	seg = m.GetDataSegments()[1]
	p = DataSegment{Address: 0x10008008, Data: []byte{5, 6, 7, 8}}
	if reflect.DeepEqual(seg, p) == false {
		t.Errorf("incorrect segment: %v != %v", seg, p)
	}

	err = parseIntelHex(m, ":020000042000DA\n:048000000506070862\n:020000041000EA\n:048000000102030472\n:00000001FF\n")
	if err != nil {
		t.Error("unexpected error: ", err.Error())
	}
	if len(m.GetDataSegments()) != 2 {
		t.Errorf("incorrect number of data segments: %v", len(m.GetDataSegments()))
	}
	seg = m.GetDataSegments()[0]
	p = DataSegment{Address: 0x10008000, Data: []byte{1, 2, 3, 4}}
	if reflect.DeepEqual(seg, p) == false {
		t.Errorf("incorrect segment: %v != %v", seg, p)
	}
	seg = m.GetDataSegments()[1]
	p = DataSegment{Address: 0x20008000, Data: []byte{5, 6, 7, 8}}
	if reflect.DeepEqual(seg, p) == false {
		t.Errorf("incorrect segment: %v != %v", seg, p)
	}
}

func TestClear(t *testing.T) {
	m := NewMemory()
	err := parseIntelHex(m, ":020000049ABCA4\n:0400000591929394AD\n:048000000102030472\n:00000001FF\n")
	if err != nil {
		t.Error("unexpected error: ", err.Error())
	}
	m.Clear()
	if m.lineNum != 0 {
		t.Error("incorrect lines number")
	}
	if len(m.GetDataSegments()) != 0 {
		t.Error("incorrect data segments")
	}
	if m.extendedAddress != 0 {
		t.Errorf("incorrect extended Address: %08X", m.extendedAddress)
	}
	if a, _ := m.GetStartAddress(); a != 0 {
		t.Errorf("incorrect start Address: %08X", m.extendedAddress)
	}
	if m.eofFlag != false {
		t.Error("incorrect eof flag state")
	}
	if m.startFlag != false {
		t.Error("incorrect start flag state")
	}
}

func TestAddBinary(t *testing.T) {
	m := NewMemory()
	err := m.AddBinary(0x20000, []byte{1, 2, 3, 4})
	err = m.AddBinary(0x20004, []byte{5, 6, 7, 8})
	if err != nil {
		t.Error("unexpected error: ", err.Error())
	}
	if len(m.GetDataSegments()) != 1 {
		t.Errorf("incorrect number of data segments: %v", len(m.GetDataSegments()))
	}
	seg := m.GetDataSegments()[0]
	p := DataSegment{Address: 0x20000, Data: []byte{1, 2, 3, 4, 5, 6, 7, 8}}
	if reflect.DeepEqual(seg, p) == false {
		t.Errorf("incorrect segment: %v != %v", seg, p)
	}

	err = m.AddBinary(0x10000, []byte{1, 2, 3, 4})
	err = m.AddBinary(0xFFFC, []byte{5, 6, 7, 8})
	if err != nil {
		t.Error("unexpected error: ", err.Error())
	}
	if len(m.GetDataSegments()) != 2 {
		t.Errorf("incorrect number of data segments: %v", len(m.GetDataSegments()))
	}
	seg = m.GetDataSegments()[0]
	p = DataSegment{Address: 0xFFFC, Data: []byte{5, 6, 7, 8, 1, 2, 3, 4}}
	if reflect.DeepEqual(seg, p) == false {
		t.Errorf("incorrect segment: %v != %v", seg, p)
	}
	seg = m.GetDataSegments()[1]
	p = DataSegment{Address: 0x20000, Data: []byte{1, 2, 3, 4, 5, 6, 7, 8}}
	if reflect.DeepEqual(seg, p) == false {
		t.Errorf("incorrect segment: %v != %v", seg, p)
	}

	err = m.AddBinary(0x15000, []byte{1, 2, 3, 4})
	err = m.AddBinary(0x14FF8, []byte{5, 6, 7, 8, 9, 10, 11, 12})
	if err != nil {
		t.Error("unexpected error: ", err.Error())
	}
	if len(m.GetDataSegments()) != 3 {
		t.Errorf("incorrect number of data segments: %v", len(m.GetDataSegments()))
	}
	seg = m.GetDataSegments()[0]
	p = DataSegment{Address: 0xFFFC, Data: []byte{5, 6, 7, 8, 1, 2, 3, 4}}
	if reflect.DeepEqual(seg, p) == false {
		t.Errorf("incorrect segment: %v != %v", seg, p)
	}
	seg = m.GetDataSegments()[1]
	p = DataSegment{Address: 0x14FF8, Data: []byte{5, 6, 7, 8, 9, 10, 11, 12, 1, 2, 3, 4}}
	if reflect.DeepEqual(seg, p) == false {
		t.Errorf("incorrect segment: %v != %v", seg, p)
	}
	seg = m.GetDataSegments()[2]
	p = DataSegment{Address: 0x20000, Data: []byte{1, 2, 3, 4, 5, 6, 7, 8}}
	if reflect.DeepEqual(seg, p) == false {
		t.Errorf("incorrect segment: %v != %v", seg, p)
	}
	err = m.AddBinary(0x15000, []byte{1, 2, 3, 4})

	m.Clear()

	err = m.AddBinary(0x0008, []byte{9, 10, 11, 12})
	err = m.AddBinary(0x0000, []byte{1, 2, 3, 4})
	err = m.AddBinary(0x0004, []byte{5, 6, 7, 8})
	if err != nil {
		t.Error("unexpected error: ", err.Error())
	}
	if len(m.GetDataSegments()) != 1 {
		t.Errorf("incorrect number of data segments: %v", len(m.GetDataSegments()))
	}

	seg = m.GetDataSegments()[0]
	p = DataSegment{Address: 0x0000, Data: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}}
	if reflect.DeepEqual(seg, p) == false {
		t.Errorf("incorrect segment: %v != %v", seg, p)
	}

}

func TestDataOverlaps(t *testing.T) {
	m := NewMemory()

	err := m.AddBinary(0x0004, []byte{1, 2, 3, 4})
	if err != nil {
		t.Error("unexpected error: ", err.Error())
	}

	err = m.AddBinary(0x0000, []byte{5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	checkErrorType(t, err, _DATA_ERROR, "no data segments overlaps error")
	err = m.AddBinary(0x0005, []byte{5, 6})
	checkErrorType(t, err, _DATA_ERROR, "no data segments overlaps error")
	err = m.AddBinary(0x0002, []byte{1, 2, 3, 4})
	checkErrorType(t, err, _DATA_ERROR, "no data segments overlaps error")
	err = m.AddBinary(0x0006, []byte{1, 2, 3, 4})
	checkErrorType(t, err, _DATA_ERROR, "no data segments overlaps error")

	err = m.AddBinary(0x0008, []byte{5})
	if err != nil {
		t.Error("unexpected error: ", err.Error())
	}

	if len(m.GetDataSegments()) != 1 {
		t.Errorf("incorrect number of data segments: %v", len(m.GetDataSegments()))
	}
	seg := m.GetDataSegments()[0]
	p := DataSegment{Address: 0x0004, Data: []byte{1, 2, 3, 4, 5}}
	if reflect.DeepEqual(seg, p) == false {
		t.Errorf("incorrect segment: %v != %v", seg, p)
	}
}

func TestSetStartMemory(t *testing.T) {
	m := NewMemory()
	m.SetStartAddress(0x12345678)

	if a, ok := m.GetStartAddress(); a != 0x12345678 || ok != true {
		t.Errorf("wrong start address: %v", a)
	}

	err := parseIntelHex(m, ":020000049ABCA4\n:048000000102030472\n:00000001FF\n")
	if err != nil {
		t.Error("unexpected error: ", err.Error())
	}

	if a, ok := m.GetStartAddress(); a != 0 || ok != false {
		t.Errorf("wrong start address: %v", a)
	}

	err = parseIntelHex(m, ":020000049ABCA4\n:0400000591929394AD\n:048000000102030472\n:00000001FF\n")
	if err != nil {
		t.Error("unexpected error: ", err.Error())
	}

	if a, ok := m.GetStartAddress(); a != 0x91929394 || ok != true {
		t.Errorf("wrong start address: %v", a)
	}

	m.SetStartAddress(0x23456789)

	if a, ok := m.GetStartAddress(); a != 0x23456789 || ok != true {
		t.Errorf("wrong start address: %v", a)
	}
}

func TestMultiSegmentsParse(t *testing.T) {
	m := NewMemory()

	err := parseIntelHex(m,
		":020000025000AC\n"+
			":10000000A5A9AEFC5FAAB488B8A8860F8BC79C943C\n"+
			":0200000260009C\n"+
			":10000000F384980CA450DC26572ECE667CAF34DFE8\n"+
			":0200000460009A\n"+
			":10000000F384980CA450DC26572ECE667CAF34DEE9\n"+
			":00000001FF\n")
	if err != nil {
		t.Error("unexpected error: ", err.Error())
	}

	if len(m.GetDataSegments()) != 3 {
		t.Errorf("incorrect number of data segments: %v", len(m.GetDataSegments()))
	}
	seg := m.GetDataSegments()[0]
	p := DataSegment{Address: 0x00050000, Data: []byte{0xA5, 0xA9, 0xAE, 0xFC, 0x5F, 0xAA, 0xB4, 0x88, 0xB8, 0xA8, 0x86, 0x0F, 0x8B, 0xC7, 0x9C, 0x94}}
	if reflect.DeepEqual(seg, p) == false {
		t.Errorf("incorrect segment: %v != %v", seg, p)
	}
	seg = m.GetDataSegments()[1]
	p = DataSegment{Address: 0x00060000, Data: []byte{0xF3, 0x84, 0x98, 0x0C, 0xA4, 0x50, 0xDC, 0x26, 0x57, 0x2E, 0xCE, 0x66, 0x7C, 0xAF, 0x34, 0xDF}}
	if reflect.DeepEqual(seg, p) == false {
		t.Errorf("incorrect segment: %v != %v", seg, p)
	}
	seg = m.GetDataSegments()[2]
	p = DataSegment{Address: 0x60000000, Data: []byte{0xF3, 0x84, 0x98, 0x0C, 0xA4, 0x50, 0xDC, 0x26, 0x57, 0x2E, 0xCE, 0x66, 0x7C, 0xAF, 0x34, 0xDE}}
	if reflect.DeepEqual(seg, p) == false {
		t.Errorf("incorrect segment: %v != %v", seg, p)
	}
}

func TestExtendedAddressIntelHex(t *testing.T) {

	m := NewMemory()
	oks := ":020000020000FC\n" +
		":0C0000000102030405060708090A0B0CA6\n" +
		":020000023000CC\n" +
		":048000000102030472\n" +
		":00000001FF\n"

	err := parseIntelHex(m, oks)
	if err != nil {
		t.Error("unexpected error: ", err.Error())
	}
	if len(m.GetDataSegments()) != 2 {
		t.Errorf("incorrect number of data segments: %v", len(m.GetDataSegments()))
	}

	seg := m.GetDataSegments()[0]
	p := DataSegment{Address: 0x0000, Data: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}}
	if reflect.DeepEqual(seg, p) == false {
		t.Errorf("incorrect segment: %v != %v", seg, p)
	}
	seg = m.GetDataSegments()[1]
	p = DataSegment{Address: 0x38000, Data: []byte{1, 2, 3, 4}}
	if reflect.DeepEqual(seg, p) == false {
		t.Errorf("incorrect segment: %v != %v", seg, p)
	}
}

func TestDumpIntelHex(t *testing.T) {
	m := NewMemory()
	m.SetStartAddress(0x12345678)
	m.AddBinary(0x18000, []byte{1, 2, 3, 4})
	m.AddBinary(0x20000000, []byte{11, 12, 13, 14})
	m.AddBinary(0x8, []byte{9, 10, 11, 12})
	m.AddBinary(0x0, []byte{1, 2, 3, 4})
	m.AddBinary(0x4, []byte{5, 6, 7, 8})
	buf := bytes.Buffer{}
	m.DumpIntelHex(&buf, 16)
	dump := buf.String()
	oks := ":0400000512345678E3\n" +
		":020000040000FA\n" +
		":0C0000000102030405060708090A0B0CA6\n" +
		":020000040001F9\n" +
		":048000000102030472\n" +
		":020000042000DA\n" +
		":040000000B0C0D0ECA\n" +
		":00000001FF\n"

	if buf.String() != oks {
		t.Errorf("wrong hex dump:\n%v", dump)
	}

	m.Clear()

	err := parseIntelHex(m, buf.String())
	if err != nil {
		t.Error("unexpected error: ", err.Error())
	}
	if len(m.GetDataSegments()) != 3 {
		t.Errorf("incorrect number of data segments: %v", len(m.GetDataSegments()))
	}
	seg := m.GetDataSegments()[0]
	p := DataSegment{Address: 0x0000, Data: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}}
	if reflect.DeepEqual(seg, p) == false {
		t.Errorf("incorrect segment: %v != %v", seg, p)
	}
	seg = m.GetDataSegments()[1]
	p = DataSegment{Address: 0x18000, Data: []byte{1, 2, 3, 4}}
	if reflect.DeepEqual(seg, p) == false {
		t.Errorf("incorrect segment: %v != %v", seg, p)
	}
	seg = m.GetDataSegments()[2]
	p = DataSegment{Address: 0x20000000, Data: []byte{11, 12, 13, 14}}
	if reflect.DeepEqual(seg, p) == false {
		t.Errorf("incorrect segment: %v != %v", seg, p)
	}

	m.Clear()
	m.AddBinary(0xFFFC, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24})
	m.AddBinary(0x10040, []byte{1, 2, 3, 4})
	buf = bytes.Buffer{}
	m.DumpIntelHex(&buf, 16)
	dump = buf.String()
	oks = ":020000040000FA\n" +
		":04FFFC0001020304F7\n" +
		":020000040001F9\n" +
		":1000000005060708090A0B0C0D0E0F101112131428\n" +
		":040010001516171892\n" +
		":0400400001020304B2\n" +
		":00000001FF\n"
	if buf.String() != oks {
		t.Errorf("wrong hex dump:\n%v", dump)
	}

	m.Clear()

	err = parseIntelHex(m, buf.String())
	if err != nil {
		t.Error("unexpected error: ", err.Error())
	}
	if len(m.GetDataSegments()) != 2 {
		t.Errorf("incorrect number of data segments: %v", len(m.GetDataSegments()))
	}
	seg = m.GetDataSegments()[0]
	p = DataSegment{Address: 0xFFFC, Data: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24}}
	if reflect.DeepEqual(seg, p) == false {
		t.Errorf("incorrect segment: %v != %v", seg, p)
	}
	seg = m.GetDataSegments()[1]
	p = DataSegment{Address: 0x10040, Data: []byte{1, 2, 3, 4}}
	if reflect.DeepEqual(seg, p) == false {
		t.Errorf("incorrect segment: %v != %v", seg, p)
	}

	m.Clear()
	d := make([]byte, 512)
	m.AddBinary(0x2FF20, d)
	buf = bytes.Buffer{}
	m.DumpIntelHex(&buf, 64)
	dump = buf.String()
	oks = ":020000040002F8\n" +
		":40FF200000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000A1\n" +
		":40FF60000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000061\n" +
		":40FFA0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000021\n" +
		":20FFE000000000000000000000000000000000000000000000000000000000000000000001\n" +
		":020000040003F7\n" +
		":4000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000C0\n" +
		":400040000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000080\n" +
		":400080000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000040\n" +
		":4000C0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000\n" +
		":200100000000000000000000000000000000000000000000000000000000000000000000DF\n" +
		":00000001FF\n"

	if buf.String() != oks {
		t.Errorf("wrong hex dump:\n%v", dump)
	}

	m.Clear()

	err = parseIntelHex(m, buf.String())
	if err != nil {
		t.Error("unexpected error: ", err.Error())
	}
	if len(m.GetDataSegments()) != 1 {
		t.Errorf("incorrect number of data segments: %v", len(m.GetDataSegments()))
	}
	seg = m.GetDataSegments()[0]
	p = DataSegment{Address: 0x2FF20, Data: make([]byte, 512)}
	if reflect.DeepEqual(seg, p) == false {
		t.Errorf("incorrect segment: %v != %v", seg, p)
	}
}

func TestToBinary(t *testing.T) {
	m := NewMemory()
	m.AddBinary(0x20000000, []byte{11, 12, 13, 14})
	m.AddBinary(0xA, []byte{9, 10, 11, 12})
	m.AddBinary(0x4, []byte{5, 6, 7, 8})

	data := m.ToBinary(0, 16, 0xFF)
	org := []byte{0xFF, 0xFF, 0xFF, 0xFF, 5, 6, 7, 8, 0xFF, 0xFF, 9, 10, 11, 12, 0xFF, 0xFF}
	if reflect.DeepEqual(data, org) == false {
		t.Errorf("incorrect binary data: %v", data)
	}
	data = m.ToBinary(0x1FFFFFFF, 8, 0)
	org = []byte{0, 11, 12, 13, 14, 0, 0, 0}
	if reflect.DeepEqual(data, org) == false {
		t.Errorf("incorrect binary data: %v", data)
	}
}

func TestSetBinary(t *testing.T) {
	m := NewMemory()
	m.AddBinary(0x00, []byte{0, 1, 2, 3})
	m.AddBinary(0x08, []byte{8, 9, 10, 11})
	m.AddBinary(0x10, []byte{16, 17, 18, 19})

	m.SetBinary(0x02, []byte{102, 103, 4, 5, 6, 7, 108, 109, 110, 111, 12, 13})

	data := m.ToBinary(0, 20, 0xFF)
	org := []byte{0, 1, 102, 103, 4, 5, 6, 7, 108, 109, 110, 111, 12, 13, 0xFF, 0xFF, 16, 17, 18, 19}
	if reflect.DeepEqual(data, org) == false {
		t.Errorf("incorrect binary data: %v", data)
	}
	if len(m.GetDataSegments()) != 2 {
		t.Errorf("incorrect number of data segments: %v", len(m.GetDataSegments()))
	}
}

func TestRemoveBinary(t *testing.T) {
	m := NewMemory()

	m.AddBinary(0x00, []byte{0, 1, 2, 3, 4, 5, 6, 7})
	m.AddBinary(0x0A, []byte{0, 1, 2, 3})
	m.AddBinary(0x10, []byte{8, 9, 10, 11})

	m.RemoveBinary(0x02, 4)
	m.RemoveBinary(0x0C, 6)

	data := m.ToBinary(0, 20, 0xFF)
	org := []byte{0, 1, 0xFF, 0xFF, 0xFF, 0xFF, 6, 7, 0xFF, 0xFF, 0, 1, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 10, 11}
	if reflect.DeepEqual(data, org) == false {
		t.Errorf("incorrect binary data: %v", data)
	}
	if len(m.GetDataSegments()) != 4 {
		t.Errorf("incorrect number of data segments: %v", len(m.GetDataSegments()))
	}

	m.Clear()

	m.AddBinary(0x00, []byte{0, 1, 2, 3, 4, 5, 6, 7})
	m.AddBinary(0x0A, []byte{0, 1, 2, 3})
	m.AddBinary(0x10, []byte{8, 9, 10, 11})

	m.RemoveBinary(0x00, 4)
	m.RemoveBinary(0x0A, 4)

	data = m.ToBinary(0, 20, 0xFF)
	org = []byte{0xFF, 0xFF, 0xFF, 0xFF, 4, 5, 6, 7, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 8, 9, 10, 11}
	if reflect.DeepEqual(data, org) == false {
		t.Errorf("incorrect binary data: %v", data)
	}
	if len(m.GetDataSegments()) != 2 {
		t.Errorf("incorrect number of data segments: %v", len(m.GetDataSegments()))
	}

	m.Clear()

	m.AddBinary(0x00, []byte{0, 1})

	m.RemoveBinary(0x00, 2)

	data = m.ToBinary(0, 4, 0xFF)
	org = []byte{0xFF, 0xFF, 0xFF, 0xFF}
	if reflect.DeepEqual(data, org) == false {
		t.Errorf("incorrect binary data: %v", data)
	}
	if len(m.GetDataSegments()) != 0 {
		t.Errorf("incorrect number of data segments: %v", len(m.GetDataSegments()))
	}

	m.Clear()

	m.AddBinary(0x00, []byte{0, 1, 2, 3, 4, 5, 6, 7})
	m.AddBinary(0x0A, []byte{0, 1, 2, 3})
	m.AddBinary(0x10, []byte{8, 9, 10, 11})

	m.RemoveBinary(0x0A, 4)

	data = m.ToBinary(0, 20, 0xFF)
	org = []byte{0, 1, 2, 3, 4, 5, 6, 7, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 8, 9, 10, 11}
	if reflect.DeepEqual(data, org) == false {
		t.Errorf("incorrect binary data: %v", data)
	}
	if len(m.GetDataSegments()) != 2 {
		t.Errorf("incorrect number of data segments: %v", len(m.GetDataSegments()))
	}

	m.AddBinary(0x0A, []byte{0, 1, 2, 3})

	data = m.ToBinary(0, 20, 0xFF)
	org = []byte{0, 1, 2, 3, 4, 5, 6, 7, 0xFF, 0xFF, 0, 1, 2, 3, 0xFF, 0xFF, 8, 9, 10, 11}
	if reflect.DeepEqual(data, org) == false {
		t.Errorf("incorrect binary data: %v", data)
	}
	if len(m.GetDataSegments()) != 3 {
		t.Errorf("incorrect number of data segments: %v", len(m.GetDataSegments()))
	}

	m.Clear()

	m.AddBinary(0x00, []byte{0, 1})

	m.RemoveBinary(0x02, 2)

	data = m.ToBinary(0, 4, 0xFF)
	org = []byte{0, 1, 0xFF, 0xFF}
	if reflect.DeepEqual(data, org) == false {
		t.Errorf("incorrect binary data: %v", data)
	}
	if len(m.GetDataSegments()) != 1 {
		t.Errorf("incorrect number of data segments: %v", len(m.GetDataSegments()))
	}
}
