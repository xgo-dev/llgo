package main

import "encoding/binary"

type packet struct{ A uint16 }

// Keep a second reference to the generic formatter through a function value.
var encode = binary.Encode

func main() {
	var buf [4]byte
	n, err := binary.Encode(buf[:2], binary.BigEndian, packet{0x1234})
	if n != 2 || err != nil || buf[0] != 0x12 || buf[1] != 0x34 {
		panic("wrong static encoding")
	}
	n, err = encode(buf[2:], binary.LittleEndian, packet{0x5678})
	if n != 2 || err != nil || buf[2] != 0x78 || buf[3] != 0x56 {
		panic("wrong dynamic encoding")
	}
	println("generic binary codec ok")
}
