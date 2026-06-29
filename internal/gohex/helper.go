package gohex

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
)

func calcSum(bytes []byte) byte {
	sum := 0
	for _, b := range bytes {
		sum += int(b)
	}
	sum %= 256
	sum = 256 - sum
	return byte(sum)
}

func checkSum(bytes []byte) error {
	sum := calcSum(bytes[:len(bytes)-1])
	last := bytes[len(bytes)-1]
	if sum != last {
		return errors.New("incorrect checksum (sum = " + fmt.Sprintf("%02X != %02X", sum, last) + ")")
	}
	return nil
}

func checkRecordSize(bytes []byte) error {
	if (int(bytes[0]) + 5) != len(bytes) {
		return errors.New("incorrect data length")
	}
	return nil
}

func checkEOF(bytes []byte) error {
	if bytes[0] != 0 {
		return errors.New("incorrect data length field in eof line")
	}
	if binary.BigEndian.Uint16(bytes[1:3]) != 0 {
		return errors.New("incorrect address field in eof line")
	}
	return nil
}

func getExtendedAddress(bytes []byte) (adr uint32, err error) {
	if bytes[0] != 2 {
		return 0, errors.New("incorrect data length field in extended linear address line")
	}
	if binary.BigEndian.Uint16(bytes[1:3]) != 0 {
		return 0, errors.New("incorrect address field in extended linear address line")
	}
	adr = uint32(binary.BigEndian.Uint16(bytes[4:6])) << (1 << bytes[3])
	return adr, nil
}

func getDataLine(bytes []byte) (adr uint16, data []byte) {
	size := bytes[0]
	adr = binary.BigEndian.Uint16(bytes[1:3])
	data = bytes[4 : size+4]
	return adr, data
}

func getStartAddress(bytes []byte) (adr uint32, err error) {
	if bytes[0] != 4 {
		return 0, errors.New("incorrect data length field in start address line")
	}
	if binary.BigEndian.Uint16(bytes[1:3]) != 0 {
		return 0, errors.New("incorrect address field in start address line")
	}
	adr = binary.BigEndian.Uint32(bytes[4:8])
	return adr, nil
}

func makeDataLine(adr uint16, recordType byte, data []byte) []byte {
	line := make([]byte, 5+len(data))
	line[0] = byte(len(data))
	binary.BigEndian.PutUint16(line[1:3], adr)
	line[3] = recordType
	copy(line[4:], data)
	line[len(line)-1] = calcSum(line[:len(line)-1])
	return line
}

func writeDataLine(writer io.Writer, lineAdr *uint32, byteAdr uint32, lineData *[]byte) error {
	s := strings.ToUpper(hex.EncodeToString(makeDataLine(uint16(*lineAdr&0x0000FFFF), _DATA_RECORD, *lineData)))
	_, err := fmt.Fprintf(writer, ":%s\n", s)
	*lineAdr = byteAdr
	*lineData = []byte{}
	return err
}

func writeStartAddressLine(writer io.Writer, startAdr uint32) error {
	a := make([]byte, 4)
	binary.BigEndian.PutUint32(a, startAdr)
	s := strings.ToUpper(hex.EncodeToString(makeDataLine(0, _START_RECORD, a)))
	_, err := fmt.Fprintf(writer, ":%s\n", s)
	return err
}

func writeExtendedAddressLine(writer io.Writer, extAdr uint32) {
	a := make([]byte, 2)
	binary.BigEndian.PutUint16(a, uint16(extAdr>>16))
	s := strings.ToUpper(hex.EncodeToString(makeDataLine(0, _ADR_32_RECORD, a)))
	fmt.Fprintf(writer, ":%s\n", s)
}

func writeEofLine(writer io.Writer) error {
	s := strings.ToUpper(hex.EncodeToString(makeDataLine(0, _EOF_RECORD, []byte{})))
	_, err := fmt.Fprintf(writer, ":%s\n", s)
	return err
}
