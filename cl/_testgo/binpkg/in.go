// Behavior regression fixture: runs on hosted and on the RV32 baremetal
// scheduler; the golden output must stay identical on both.

package main

import "encoding/binary"

func main() {
	var b [8]byte
	binary.LittleEndian.PutUint32(b[:4], 0xdeadbeef)
	println("le32", b[0], b[1], b[2], b[3])
	binary.BigEndian.PutUint16(b[4:6], 0x1234)
	println("be16", b[4], b[5])
	binary.LittleEndian.PutUint64(b[:], 0x0102030405060708)
	println("le64", b[0], b[7])
	println("roundtrip", binary.LittleEndian.Uint32(b[:4]) == 0x01020304)
	buf := make([]byte, 10)
	n := binary.PutUvarint(buf, 300)
	println("uvarint", n, buf[0], buf[1])
	u, m := binary.Uvarint(buf[:n])
	println("uvarintrd", u, m)
	n2 := binary.PutVarint(buf, -100)
	vi, m2 := binary.Varint(buf[:n2])
	println("varint", vi, m2)
}
