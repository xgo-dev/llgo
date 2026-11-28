package crc16

import (
	"fmt"
	"testing"
)

var testData = []byte("123456789")

func testSelectedCRC(params Params, t *testing.T) {
	table := MakeTable(params)
	if table == nil {
		t.Errorf("Failed to create %q computer", params.Name)
	}

	crc := Checksum(testData, table)

	if crc != table.params.Check {
		t.Errorf("Invalid %q sample calculation, expected: %X, actual: %X", table.params.Name, table.params.Check, crc)
	}
}

func TestCRC16_ARC(t *testing.T) {
	testSelectedCRC(CRC16_ARC, t)
}

func TestCRC16_AUG_CCIT(t *testing.T) {
	testSelectedCRC(CRC16_AUG_CCITT, t)
}

func TestCRC16_BUYPASS(t *testing.T) {
	testSelectedCRC(CRC16_BUYPASS, t)
}

func TestCRC16_CCITT_FALSE(t *testing.T) {
	testSelectedCRC(CRC16_CCITT_FALSE, t)
}

func TestCRC16_CDMA2000(t *testing.T) {
	testSelectedCRC(CRC16_CDMA2000, t)
}

func TestCRC16_DDS_110(t *testing.T) {
	testSelectedCRC(CRC16_DDS_110, t)
}

func TestCRC16_DECT_R(t *testing.T) {
	testSelectedCRC(CRC16_DECT_R, t)
}

func TestCRC16_DECT_X(t *testing.T) {
	testSelectedCRC(CRC16_DECT_X, t)
}

func TestCRC16_DNP(t *testing.T) {
	testSelectedCRC(CRC16_DNP, t)
}

func TestCRC16_EN_13757(t *testing.T) {
	testSelectedCRC(CRC16_EN_13757, t)
}

func TestCRC16_GENIBUS(t *testing.T) {
	testSelectedCRC(CRC16_GENIBUS, t)
}

func TestCRC16_MAXIM(t *testing.T) {
	testSelectedCRC(CRC16_MAXIM, t)
}

func TestCRC16_MCRF4XX(t *testing.T) {
	testSelectedCRC(CRC16_MCRF4XX, t)
}

func TestCRC16_RIELLO(t *testing.T) {
	testSelectedCRC(CRC16_RIELLO, t)
}

func TestCRC16_T10_DIF(t *testing.T) {
	testSelectedCRC(CRC16_T10_DIF, t)
}

func TestCRC16_TELEDISK(t *testing.T) {
	testSelectedCRC(CRC16_TELEDISK, t)
}

func TestCRC16_TMS37157(t *testing.T) {
	testSelectedCRC(CRC16_TMS37157, t)
}

func TestCRC16_USB(t *testing.T) {
	testSelectedCRC(CRC16_USB, t)
}

func TestCRC16_CRC_A(t *testing.T) {
	testSelectedCRC(CRC16_CRC_A, t)
}

func TestCRC16_KERMIT(t *testing.T) {
	testSelectedCRC(CRC16_KERMIT, t)
}

func TestCRC16_MODBUS(t *testing.T) {
	testSelectedCRC(CRC16_MODBUS, t)
}

func TestCRC16_X_25(t *testing.T) {
	testSelectedCRC(CRC16_X_25, t)
}

func TestCRC16_XMODEM(t *testing.T) {
	testSelectedCRC(CRC16_XMODEM, t)
}

func TestHash(t *testing.T) {
	tbl := MakeTable(CRC16_XMODEM)
	h := New(tbl)

	fmt.Fprint(h, "standard")
	fmt.Fprint(h, " library hash interface")
	sum1 := h.Sum16()
	h.Reset()
	fmt.Fprint(h, "standard library hash interface")
	sum2 := h.Sum16()

	if sum1 != sum2 {
		t.Errorf("CRC16 checksums for chunked input %x should be equal %x", sum1, sum2)
	}

	var crc uint16 = 0xe698
	if sum1 != crc {
		t.Errorf("CRC16 for input should equal %x but was %x", crc, sum1)
	}

	if h.Size() != 2 {
		t.Errorf("CRC16 checksum should have a size of 2 byte. But was %d", h.Size())
	}
	buf := make([]byte, 0, 10)
	buf = h.Sum(buf)
	expected := []byte{0xe6, 0x98}
	if len(buf) != 2 || buf[0] != expected[0] || buf[1] != expected[1] {
		t.Errorf("CRC16 checksum should append %v to slice, but was %v", expected, buf)
	}

	// for the 100% test coverage
	if h.BlockSize() != 1 {
		t.Errorf("Block size should return 1")
	}
}
